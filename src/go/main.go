package main

import (
	"context"
	"errors"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/jackc/pgx/v5"
	"time"
)

const version = "2022-11-05"

type JobExecutionRequest struct {
	Name string `json:"name"`
}

type ScheduledExecutionEvent struct {
	Version string                `json:"version"`
	Jobs    []JobExecutionRequest `json:"jobs"`
}

func main() {
	lambda.Start(func(ctx context.Context, event ScheduledExecutionEvent) ([]byte, error) {
		if event.Version > version {
			return nil, errors.New("unsupported version")
		}

		conn, err := pgx.Connect(ctx, "postgres://postgres:postgres@localhost:5433/postgres")
		if err != nil {
			return nil, err
		}
		defer conn.Close(context.Background())

		for _, job := range event.Jobs {
			go executeJob(ctx, conn, job)
		}

		return nil, nil
	})
}

func executeJob(ctx context.Context, conn *pgx.Conn, job JobExecutionRequest) error {
	var err error

	switch job.Name {
	case "DELETE_EXPIRED_SESSIONS":
		_, err = conn.Query(ctx, "DELETE FROM account_federation_sessions WHERE expiration_time <= $1", time.Now())
	case "DELETE_EXPIRED_AUTHORIZATIONS":
		_, err = conn.Query(ctx, "DELETE FROM client_authorizations WHERE COALESCE(GREATEST(authorization_code_expires_at, access_token_expires_at, refresh_token_expires_at), (last_update_time + INTERVAL '1 DAY')) <= $1", time.Now())
	default:
		err = errors.New("unknown job: " + job.Name)
	}

	return err
}
