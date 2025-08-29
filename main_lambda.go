//go:build lambda

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/agoda-com/opentelemetry-go/otelslog"
	"github.com/agoda-com/opentelemetry-logs-go/exporters/otlp/otlplogs"
	"github.com/agoda-com/opentelemetry-logs-go/exporters/otlp/otlplogs/otlplogsgrpc"
	sdklogs "github.com/agoda-com/opentelemetry-logs-go/sdk/logs"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/gw2auth/gw2auth.com-background-jobs/db"
	"github.com/jackc/pgx/v5/pgxpool"
	lambdadetector "go.opentelemetry.io/contrib/detectors/aws/lambda"
	"go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-lambda-go/otellambda"
	"go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-sdk-go-v2/otelaws"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/contrib/propagators/aws/xray"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	sdkresource "go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
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

	defer tp.Shutdown(context.Background())
	defer lp.Shutdown(context.Background())

	pool, err := pgxPool(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "failed to connect to database", slog.String("error", err.Error()))
		os.Exit(1)
		return
	}
	defer pool.Close()

	lambda.StartWithOptions(
		otellambda.InstrumentHandler(lambdaHandler(pool), otellambda.WithTracerProvider(tp), otellambda.WithFlusher(tp)),
		lambda.WithContext(ctx),
	)
}

func telemetry(ctx context.Context) (*sdktrace.TracerProvider, *sdklogs.LoggerProvider, error) {
	resource, err := sdkresource.New(
		ctx,
		sdkresource.WithDetectors(lambdadetector.NewResourceDetector()),
		sdkresource.WithAttributes(semconv.ServiceName("GW2AuthBackgroundJobs")),
	)
	if err != nil {
		return nil, nil, err
	}

	// see: xrayconfig.NewTracerProvider(ctx)
	traceExp, err := otlptracegrpc.New(ctx, otlptracegrpc.WithInsecure())
	if err != nil {
		return nil, nil, err
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithBatcher(traceExp),
		sdktrace.WithIDGenerator(xray.NewIDGenerator()),
		sdktrace.WithResource(resource),
	)

	client := otlplogsgrpc.NewClient(otlplogsgrpc.WithInsecure())
	logExp, err := otlplogs.NewExporter(ctx, otlplogs.WithClient(client))
	if err != nil {
		return nil, nil, err
	}

	lp := sdklogs.NewLoggerProvider(
		sdklogs.WithBatcher(logExp),
		sdklogs.WithResource(resource),
	)

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(xray.Propagator{})

	log := slog.New(otelslog.NewOtelHandler(lp, &otelslog.HandlerOptions{}))
	slog.SetDefault(log)

	return tp, lp, nil
}

func pgxPool(ctx context.Context) (*pgxpool.Pool, error) {
	cfg, err := awsconfig.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, err
	}

	otelaws.AppendMiddlewares(&cfg.APIOptions)
	ssmc := ssm.NewFromConfig(cfg)
	resp, err := ssmc.GetParameter(ctx, &ssm.GetParameterInput{
		Name:           aws.String(os.Getenv("GW2AUTH_SSM_DATABASE_URL")),
		WithDecryption: aws.Bool(true),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve SSM parameter for database URL: %w", err)
	}

	return db.NewPgx(ctx, *resp.Parameter.Value)
}

func lambdaHandler(pool *pgxpool.Pool) func(ctx context.Context, request json.RawMessage) (json.RawMessage, error) {
	httpClient := &http.Client{
		Transport: otelhttp.NewTransport(http.DefaultTransport),
	}

	return func(ctx context.Context, request json.RawMessage) (json.RawMessage, error) {
		return request, run(ctx, pool, httpClient)
	}
}
