package db

import (
	"context"

	"github.com/exaring/otelpgx"
	pgxuuid "github.com/jackc/pgx-gofrs-uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func NewPgx(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, err
	}

	config.ConnConfig.Tracer = otelpgx.NewTracer()
	config.ConnConfig.RuntimeParams["application_name"] = "GW2Auth Background Jobs"
	config.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		pgxuuid.Register(conn.TypeMap())
		return nil
	}

	return pgxpool.NewWithConfig(ctx, config)
}
