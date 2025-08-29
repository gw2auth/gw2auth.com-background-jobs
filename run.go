package main

import (
	"context"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
)

func run(ctx context.Context, pool *pgxpool.Pool, httpClient *http.Client) error {
	return pool.Ping(ctx)
}
