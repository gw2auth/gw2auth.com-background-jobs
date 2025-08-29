module github.com/gw2auth/gw2auth.com-background-jobs

go 1.25

require (
	github.com/agoda-com/opentelemetry-go/otelslog v0.3.0
	github.com/agoda-com/opentelemetry-logs-go v0.6.0
	github.com/aws/aws-lambda-go v1.49.0
	github.com/aws/aws-sdk-go-v2 v1.38.2
	github.com/aws/aws-sdk-go-v2/config v1.31.5
	github.com/aws/aws-sdk-go-v2/service/ssm v1.64.1
	github.com/cockroachdb/cockroach-go/v2 v2.4.1
	github.com/exaring/otelpgx v0.9.3
	github.com/gofrs/uuid/v5 v5.0.0
	github.com/jackc/pgx-gofrs-uuid v0.0.0-20230224015001-1d428863c2e2
	github.com/jackc/pgx/v5 v5.7.5
	go.opentelemetry.io/contrib/detectors/aws/lambda v0.62.0
	go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-lambda-go/otellambda v0.62.0
	go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-sdk-go-v2/otelaws v0.62.0
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.62.0
	go.opentelemetry.io/contrib/propagators/aws v1.37.0
	go.opentelemetry.io/otel v1.37.0
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.37.0
	go.opentelemetry.io/otel/exporters/stdout/stdouttrace v1.37.0
	go.opentelemetry.io/otel/sdk v1.37.0
	go.opentelemetry.io/otel/trace v1.37.0
	golang.org/x/time v0.12.0
)

require (
	github.com/aws/aws-sdk-go-v2/credentials v1.18.9 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.18.5 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.4.5 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.7.5 // indirect
	github.com/aws/aws-sdk-go-v2/internal/ini v1.8.3 // indirect
	github.com/aws/aws-sdk-go-v2/service/dynamodb v1.43.4 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.13.1 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/endpoint-discovery v1.10.17 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.13.5 // indirect
	github.com/aws/aws-sdk-go-v2/service/sns v1.34.7 // indirect
	github.com/aws/aws-sdk-go-v2/service/sqs v1.38.8 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.29.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.34.1 // indirect
	github.com/aws/aws-sdk-go-v2/service/sts v1.38.1 // indirect
	github.com/aws/smithy-go v1.23.0 // indirect
	github.com/cenkalti/backoff/v4 v4.3.0 // indirect
	github.com/cenkalti/backoff/v5 v5.0.2 // indirect
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.27.1 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	go.opentelemetry.io/auto/sdk v1.1.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.37.0 // indirect
	go.opentelemetry.io/otel/metric v1.37.0 // indirect
	go.opentelemetry.io/proto/otlp v1.7.0 // indirect
	golang.org/x/crypto v0.39.0 // indirect
	golang.org/x/net v0.41.0 // indirect
	golang.org/x/sync v0.15.0 // indirect
	golang.org/x/sys v0.33.0 // indirect
	golang.org/x/text v0.26.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20250603155806-513f23925822 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250603155806-513f23925822 // indirect
	google.golang.org/grpc v1.73.0 // indirect
	google.golang.org/protobuf v1.36.6 // indirect
)
