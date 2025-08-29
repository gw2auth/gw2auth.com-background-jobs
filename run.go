package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	crdbpgx "github.com/cockroachdb/cockroach-go/v2/crdb/crdbpgxv5"
	"github.com/gofrs/uuid/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

const (
	validCheckTimeout  = time.Hour * 24 * 7
	validCheckInterval = time.Hour * 24
	nameCheckInterval  = validCheckInterval
)

type token struct {
	AccountId      uuid.UUID
	Gw2AccountId   uuid.UUID
	Gw2AccountName string
	Gw2ApiToken    string
}

type TokenChecker struct {
	pool       *pgxpool.Pool
	httpClient *http.Client
	tracer     trace.Tracer
}

func NewTokenChecker(pool *pgxpool.Pool, httpClient *http.Client) *TokenChecker {
	tracer := otel.Tracer("github.com/gw2auth/gw2auth.com-background-jobs::TokenChecker", trace.WithInstrumentationVersion("v0.0.1"))
	return &TokenChecker{
		pool:       pool,
		httpClient: httpClient,
		tracer:     tracer,
	}
}

func (tc *TokenChecker) Run(ctx context.Context) error {
	const timePerToken = time.Second * 10

	deadline, ok := ctx.Deadline()
	if !ok {
		deadline = time.Date(2100, time.January, 1, 0, 0, 0, 0, time.UTC)
	}

	for {
		remainingTime := time.Until(deadline)
		batchSize := min(int(remainingTime/timePerToken), 100)
		tokensToCheck, err := tc.loadBatch(ctx, batchSize)
		if err != nil {
			return fmt.Errorf("failed to load token batch: %w", err)
		}

		if len(tokensToCheck) < 1 {
			break
		}

		if err := tc.checkBatch(ctx, tokensToCheck); err != nil {
			return fmt.Errorf("failed to check token batch: %w", err)
		}
	}

	return nil
}

func (tc *TokenChecker) loadBatch(ctx context.Context, batchSize int) ([]token, error) {
	now := time.Now()
	minLastValidTime := now.Add(-validCheckTimeout)
	maxValidCheckTime := now.Add(-validCheckInterval)
	maxNameCheckTime := now.Add(-nameCheckInterval)

	tokensToCheck := make([]token, 0, batchSize)
	err := crdbpgx.ExecuteTx(ctx, tc.pool, pgx.TxOptions{AccessMode: pgx.ReadOnly}, func(tx pgx.Tx) error {
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
LIMIT $4
`,
			minLastValidTime,
			maxValidCheckTime,
			maxNameCheckTime,
			batchSize,
		)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var t token
			if err := rows.Scan(&t.AccountId, &t.Gw2AccountId, &t.Gw2AccountName, &t.Gw2ApiToken); err != nil {
				return err
			}

			tokensToCheck = append(tokensToCheck, t)
		}

		return rows.Err()
	})
	if err != nil {
		return nil, err
	}

	return tokensToCheck, nil
}

func (tc *TokenChecker) checkBatch(ctx context.Context, tokensToCheck []token) error {
	slog.InfoContext(ctx, "checking batch", slog.Int("batch.size", len(tokensToCheck)))

	for _, t := range tokensToCheck {
		if err := tc.checkSingle(ctx, t); err != nil {
			return fmt.Errorf("token check failed: %w", err)
		}
	}

	return nil
}

func (tc *TokenChecker) checkSingle(ctx context.Context, tk token) error {
	ctx, span := tc.tracer.Start(
		ctx,
		"CheckSingle",
		trace.WithSpanKind(trace.SpanKindServer),
		trace.WithAttributes(
			attribute.String("account.id", tk.AccountId.String()),
			attribute.String("gw2account.id", tk.Gw2AccountId.String()),
		),
	)
	defer span.End()

	if err := tc.checkSingleInternal(ctx, tk); err != nil {
		span.RecordError(err)
		return err
	}

	return nil
}

func (tc *TokenChecker) checkSingleInternal(ctx context.Context, tk token) error {
	var newGw2AccountName string
	var isValid bool
	{
		var err error
		newGw2AccountName, err = tc.getGw2AccountName(ctx, tk.Gw2ApiToken)
		if err != nil {
			slog.InfoContext(
				ctx,
				"could not get gw2 account name",
				slog.String("account.id", tk.AccountId.String()),
				slog.String("gw2account.id", tk.Gw2AccountId.String()),
				slog.String("error", err.Error()),
			)
		} else {
			slog.InfoContext(
				ctx,
				"token check succeeded",
				slog.String("account.id", tk.AccountId.String()),
				slog.String("gw2account.id", tk.Gw2AccountId.String()),
				slog.String("gw2account.name.new", newGw2AccountName),
				slog.String("gw2account.name.old", tk.Gw2AccountName),
				slog.Bool("gw2account.name.changed", newGw2AccountName != tk.Gw2AccountName),
			)
		}

		isValid = err == nil
	}

	now := time.Now()
	return crdbpgx.ExecuteTx(ctx, tc.pool, pgx.TxOptions{}, func(tx pgx.Tx) error {
		var err error
		if isValid {
			_, err1 := tc.pool.Exec(
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

			_, err2 := tc.pool.Exec(
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

			err = errors.Join(err1, err2)
		} else {
			_, err1 := tc.pool.Exec(
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

			_, err2 := tc.pool.Exec(
				ctx,
				`
UPDATE gw2_accounts
SET last_name_check_time = $1
WHERE account_id = $2
AND gw2_account_id = $3
`,
				now,
				tk.AccountId,
				tk.Gw2AccountId,
			)

			err = errors.Join(err1, err2)
		}

		return err
	})
}

func (tc *TokenChecker) getGw2AccountName(ctx context.Context, gw2ApiToken string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.guildwars2.com/v2/account", nil)
	if err != nil {
		return "", err
	}

	q := req.URL.Query()
	q.Set("v", "2025-08-29T00:00:00.000Z")
	q.Set("access_token", gw2ApiToken)
	req.URL.RawQuery = q.Encode()

	resp, err := tc.httpClient.Do(req)
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
