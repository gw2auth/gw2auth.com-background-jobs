//go:build !lambda

package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/agoda-com/opentelemetry-go/otelslog"
	"github.com/agoda-com/opentelemetry-logs-go/exporters/stdout/stdoutlogs"
	sdklogs "github.com/agoda-com/opentelemetry-logs-go/sdk/logs"
	"github.com/gw2auth/gw2auth.com-background-jobs/db"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/contrib/propagators/aws/xray"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	sdkresource "go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.38.0"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	tp, lp, err := telemetry(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "failed to connect to database", slog.String("error", err.Error()))
		os.Exit(1)
		return
	}

	defer func() {
		ctx := context.Background()
		_ = tp.ForceFlush(ctx)
		_ = lp.ForceFlush(ctx)
		_ = tp.Shutdown(ctx)
		_ = lp.Shutdown(ctx)
	}()

	pool, err := pgxPool(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "failed to connect to database", slog.String("error", err.Error()))
		return
	}
	defer pool.Close()

	tc := NewTokenChecker(
		pool,
		&http.Client{
			Transport: otelhttp.NewTransport(http.DefaultTransport),
		},
	)

	{
		ctx, cancel := context.WithTimeout(ctx, 15*time.Minute)
		defer cancel()

		if err := tc.Run(ctx); err != nil {
			slog.ErrorContext(ctx, "failed to run", slog.String("error", err.Error()))
			return
		}
	}
}

func telemetry(ctx context.Context) (*sdktrace.TracerProvider, *sdklogs.LoggerProvider, error) {
	resource, err := sdkresource.New(
		ctx,
		sdkresource.WithAttributes(semconv.ServiceName("GW2AuthBackgroundJobs")),
	)
	if err != nil {
		return nil, nil, err
	}

	// see: xrayconfig.NewTracerProvider(ctx)
	traceExp, err := stdouttrace.New(stdouttrace.WithPrettyPrint(), stdouttrace.WithWriter(os.Stdout))
	if err != nil {
		return nil, nil, err
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithBatcher(traceExp),
		sdktrace.WithResource(resource),
	)

	logExp, err := stdoutlogs.NewExporter(stdoutlogs.WithWriter(os.Stdout))
	if err != nil {
		return nil, nil, err
	}

	lp := sdklogs.NewLoggerProvider(
		sdklogs.WithBatcher(logExp),
		sdklogs.WithResource(resource),
	)

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(xray.Propagator{})

	slog.SetDefault(slog.New(otelslog.NewOtelHandler(lp, &otelslog.HandlerOptions{})))

	return tp, lp, nil
}

func pgxPool(ctx context.Context) (*pgxpool.Pool, error) {
	return db.NewPgx(ctx, "postgres://gw2auth_app:@localhost:26257/defaultdb")
}
