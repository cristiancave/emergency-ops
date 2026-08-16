// Package logger provee logging estructurado en JSON que se correlaciona
// con trazas: cada línea de log dentro de un span lleva trace_id/span_id,
// que es lo que permite pivotear entre logs y trazas en Grafana/Jaeger.
package logger

import (
	"context"
	"log/slog"
	"os"

	"go.opentelemetry.io/otel/trace"
)

// New crea un logger JSON hacia stdout, con el nombre del servicio como campo fijo.
// stdout (no un archivo) porque en ECS/Fargate los logs se recolectan del stream
// del contenedor.
func New(serviceName string) *slog.Logger {
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})
	return slog.New(handler).With("service", serviceName)
}

// Ctx devuelve un logger enriquecido con trace_id/span_id extraídos de ctx,
// si hay un span activo. Úsalo en cada punto de log dentro de un handler o
// de lógica de negocio instrumentada:
//
//	logger.Ctx(ctx, log).Info("dispatch created", "dispatch_id", d.ID)
func Ctx(ctx context.Context, l *slog.Logger) *slog.Logger {
	sc := trace.SpanContextFromContext(ctx)
	if !sc.IsValid() {
		return l
	}
	return l.With(
		"trace_id", sc.TraceID().String(),
		"span_id", sc.SpanID().String(),
	)
}
