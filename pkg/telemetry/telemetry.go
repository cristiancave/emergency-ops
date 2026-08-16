// Package telemetry centraliza el arranque de OpenTelemetry (trazas + métricas)
// para que dispatch-service y triage-service lo configuren de forma idéntica.
package telemetry

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

// Config define cómo se identifica y a dónde exporta este servicio.
type Config struct {
	ServiceName    string
	ServiceVersion string
	Environment    string // "dev", "staging", "prod"

	// OTLPEndpoint es el host:puerto del OTel Collector (receiver OTLP/gRPC).
	// Si está vacío, se usa el default "localhost:4317".
	OTLPEndpoint string

	// OTLPInsecure desactiva TLS en la conexión gRPC al Collector.
	// true por default: el Collector vive dentro de la misma red privada.
	OTLPInsecure bool
}

// Shutdown libera los recursos de telemetría (flush de spans pendientes, etc).
// Debe llamarse antes de que el proceso termine.
type Shutdown func(context.Context) error

// Init configura TracerProvider (OTLP -> Collector) y MeterProvider (Prometheus pull),
// los registra como globales de OTel, y configura la propagación W3C Trace Context
// para que dispatch -> triage propague trace_id/span_id automáticamente vía HTTP.
//
// metricsHandler es el handler HTTP que debe montarse en GET /metrics.
func Init(ctx context.Context, cfg Config) (shutdown Shutdown, metricsHandler http.Handler, err error) {
	res, err := buildResource(ctx, cfg)
	if err != nil {
		return nil, nil, fmt.Errorf("build resource: %w", err)
	}

	tp, err := buildTracerProvider(ctx, cfg, res)
	if err != nil {
		return nil, nil, fmt.Errorf("build tracer provider: %w", err)
	}
	otel.SetTracerProvider(tp)

	mp, promExporter, err := buildMeterProvider(res)
	if err != nil {
		return nil, nil, fmt.Errorf("build meter provider: %w", err)
	}
	otel.SetMeterProvider(mp)

	// W3C Trace Context + Baggage: esto es lo que hace que otelhttp propague
	// trace_id/span_id entre dispatch y triage a través del header "traceparent".
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	_ = promExporter // el exporter se registra solo; no necesitamos su referencia directa

	handler := promhttp.Handler()

	shutdown = func(ctx context.Context) error {
		if err := tp.Shutdown(ctx); err != nil {
			return fmt.Errorf("shutdown tracer provider: %w", err)
		}
		if err := mp.Shutdown(ctx); err != nil {
			return fmt.Errorf("shutdown meter provider: %w", err)
		}
		return nil
	}

	return shutdown, handler, nil
}

func buildResource(ctx context.Context, cfg Config) (*resource.Resource, error) {
	return resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(cfg.ServiceName),
			semconv.ServiceVersion(cfg.ServiceVersion),
			semconv.DeploymentEnvironment(cfg.Environment),
		),
		resource.WithHost(),
		resource.WithProcessPID(),
	)
}

func buildTracerProvider(ctx context.Context, cfg Config, res *resource.Resource) (*sdktrace.TracerProvider, error) {
	endpoint := cfg.OTLPEndpoint
	if endpoint == "" {
		endpoint = "localhost:4317"
	}

	opts := []otlptracegrpc.Option{
		otlptracegrpc.WithEndpoint(endpoint),
		otlptracegrpc.WithTimeout(5 * time.Second),
	}
	if cfg.OTLPInsecure {
		opts = append(opts, otlptracegrpc.WithInsecure())
	}

	exporter, err := otlptracegrpc.New(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("create OTLP trace exporter: %w", err)
	}

	return sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	), nil
}

func buildMeterProvider(res *resource.Resource) (*sdkmetric.MeterProvider, *prometheus.Exporter, error) {
	exporter, err := prometheus.New()
	if err != nil {
		return nil, nil, fmt.Errorf("create Prometheus exporter: %w", err)
	}

	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(exporter),
		sdkmetric.WithResource(res),
	)

	return mp, exporter, nil
}

// EnvOr lee una variable de entorno con default, atajo usado por main.go de cada servicio.
func EnvOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
