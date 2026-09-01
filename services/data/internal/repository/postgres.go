package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"emergencyops/data/internal/domain"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type Repository interface {
	SaveTelemetryRecord(ctx context.Context, record domain.DataRecord) (*domain.DataRecord, error)
	Ping(ctx context.Context) error
}

type postgresRepo struct {
	db     *sql.DB
	tracer trace.Tracer
}

type memoryRepo struct {
	records map[string]domain.DataRecord
	tracer  trace.Tracer
}

func NewPostgresRepository(db *sql.DB) Repository {
	return &postgresRepo{
		db:     db,
		tracer: otel.Tracer("emergencyops/data/repository/postgres"),
	}
}

func NewMemoryRepository() Repository {
	return &memoryRepo{
		records: make(map[string]domain.DataRecord),
		tracer:  otel.Tracer("emergencyops/data/repository/memory"),
	}
}

func (r *postgresRepo) SaveTelemetryRecord(ctx context.Context, rec domain.DataRecord) (*domain.DataRecord, error) {
	ctx, span := r.tracer.Start(ctx, "SaveTelemetryRecord AWS RDS SQL",
		trace.WithAttributes(
			attribute.String("db.system", "postgresql"),
			attribute.String("db.name", "emergency_db"),
			attribute.String("db.user", "aws_rds_admin"),
			attribute.String("db.operation", "INSERT"),
			attribute.String("db.sql.table", "data_telemetry_records"),
			attribute.String("db.statement", "INSERT INTO data_telemetry_records (id, report_id, trace_id, span_id, source, payload, created_at) VALUES ($1, $2, $3, $4, $5, $6, $7)"),
		),
	)
	defer span.End()

	if rec.ID == "" {
		rec.ID = "DAT-" + uuid.New().String()[:8]
	}
	if rec.CreatedAt.IsZero() {
		rec.CreatedAt = time.Now().UTC()
	}

	query := `INSERT INTO data_telemetry_records (id, report_id, trace_id, span_id, source, payload, created_at)
	          VALUES ($1, $2, $3, $4, $5, $6, $7)`
	
	_, err := r.db.ExecContext(ctx, query, rec.ID, rec.ReportID, rec.TraceID, rec.SpanID, rec.Source, rec.Payload, rec.CreatedAt)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("aws rds db insert failed: %w", err)
	}

	return &rec, nil
}

func (r *postgresRepo) Ping(ctx context.Context) error {
	return r.db.PingContext(ctx)
}

func (r *memoryRepo) SaveTelemetryRecord(ctx context.Context, rec domain.DataRecord) (*domain.DataRecord, error) {
	ctx, span := r.tracer.Start(ctx, "SaveTelemetryRecord Memory Fallback",
		trace.WithAttributes(
			attribute.String("db.system", "in_memory"),
			attribute.String("db.name", "telemetry_cache"),
		),
	)
	defer span.End()

	if rec.ID == "" {
		rec.ID = "DAT-MEM-" + uuid.New().String()[:8]
	}
	if rec.CreatedAt.IsZero() {
		rec.CreatedAt = time.Now().UTC()
	}

	r.records[rec.ID] = rec
	return &rec, nil
}

func (r *memoryRepo) Ping(ctx context.Context) error {
	return nil
}
