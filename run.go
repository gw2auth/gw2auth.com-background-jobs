package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	crdbpgx "github.com/cockroachdb/cockroach-go/v2/crdb/crdbpgxv5"
	"github.com/gofrs/uuid/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	validCheckTimeout  = time.Hour * 24 * 7
	validCheckInterval = time.Hour * 24
	nameCheckInterval  = validCheckInterval
)

type tokenCheck struct {
	AccountId      uuid.UUID
	Gw2AccountId   uuid.UUID
	Gw2AccountName string
	Gw2ApiToken    string
}

func run(ctx context.Context, pool *pgxpool.Pool, httpClient *http.Client) error {
	now := time.Now()
	minLastValidTime := now.Add(-validCheckTimeout)
	maxValidCheckTime := now.Add(-validCheckInterval)
	maxNameCheckTime := now.Add(-nameCheckInterval)

	tokensToCheck := make([]tokenCheck, 0)
	err := crdbpgx.ExecuteTx(ctx, pool, pgx.TxOptions{AccessMode: pgx.ReadOnly}, func(tx pgx.Tx) error {
		rows, err := tx.Query(
			ctx,
			`
SELECT tk.account_id, tk.gw2_account_id, acc.gw2_account_name, tk.gw2_api_token
FROM gw2_account_api_tokens tk
INNER JOIN gw2_accounts acc
ON tk.account_id = acc.account_id AND tk.gw2_account_id = acc.gw2_account_id
WHERE tk.last_valid_time >= $1
AND (tk.last_valid_check_time <= $2 OR acc.last_name_check_time <= $3)
ORDER BY LEAST(tk.last_valid_check_time, acc.last_name_check_time) ASC
LIMIT 100
`,
			minLastValidTime,
			maxValidCheckTime,
			maxNameCheckTime,
		)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var t tokenCheck
			if err := rows.Scan(&t.AccountId, &t.Gw2AccountId, &t.Gw2AccountName, &t.Gw2ApiToken); err != nil {
				return err
			}

			tokensToCheck = append(tokensToCheck, t)
		}

		return rows.Err()
	})
	if err != nil {
		return err
	}

	slog.InfoContext(ctx, "checking validity and names", slog.Int("token.count", len(tokensToCheck)))

	for _, tk := range tokensToCheck {
		newGw2AccountName, err := checkToken(ctx, httpClient, tk)
		if err != nil {
			slog.InfoContext(
				ctx,
				"token check failed",
				slog.String("account.id", tk.AccountId.String()),
				slog.String("gw2account.id", tk.Gw2AccountId.String()),
				slog.String("error", err.Error()),
			)

			if err := crdbpgx.ExecuteTx(ctx, pool, pgx.TxOptions{}, func(tx pgx.Tx) error {
				_, err := pool.Exec(
					ctx,
					`
UPDATE gw2_account_api_tokens
SET last_valid_check_time = $1
WHERE account_id = $2
AND gw2_account_id = $3
`,
					now,
					tk.AccountId,
					tk.Gw2AccountId,
				)
				return err
			}); err != nil {
				return err
			}
		} else {
			slog.InfoContext(
				ctx,
				"token check succeeded",
				slog.String("account.id", tk.AccountId.String()),
				slog.String("gw2account.id", tk.Gw2AccountId.String()),
				slog.Bool("gw2account.name.changed", newGw2AccountName != tk.Gw2AccountName),
			)

			if err := crdbpgx.ExecuteTx(ctx, pool, pgx.TxOptions{}, func(tx pgx.Tx) error {
				_, err := pool.Exec(
					ctx,
					`
UPDATE gw2_account_api_tokens
SET last_valid_check_time = $1, last_valid_time = $1
WHERE account_id = $2
AND gw2_account_id = $3
`,
					now,
					tk.AccountId,
					tk.Gw2AccountId,
				)
				if err != nil {
					return err
				}

				_, err = pool.Exec(
					ctx,
					`
UPDATE gw2_accounts
SET gw2_account_name = $1, last_name_check_time = $2
WHERE account_id = $3
AND gw2_account_id = $4
`,
					newGw2AccountName,
					now,
					tk.AccountId,
					tk.Gw2AccountId,
				)
				return err
			}); err != nil {
				return err
			}
		}
	}

	return nil
}

func checkToken(ctx context.Context, httpClient *http.Client, tk tokenCheck) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.guildwars2.com/v2/account", nil)
	if err != nil {
		return "", err
	}

	q := req.URL.Query()
	q.Set("v", "2025-08-29T00:00:00.000Z")
	q.Set("access_token", tk.Gw2ApiToken)
	req.URL.RawQuery = q.Encode()

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var body struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return "", err
	}

	return body.Name, nil
}
