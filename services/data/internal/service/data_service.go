package service

import (
	"context"
	"fmt"
	"time"

	"emergencyops/data/internal/domain"
	"emergencyops/data/internal/repository"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type DataService interface {
	ProcessTelemetry(ctx context.Context, req domain.TelemetryRequest) (*domain.TelemetryResponse, error)
	HealthCheck(ctx context.Context) error
}

type dataServiceImpl struct {
	repo   repository.Repository
	tracer trace.Tracer
}

func NewDataService(repo repository.Repository) DataService {
	return &dataServiceImpl{
		repo:   repo,
		tracer: otel.Tracer("emergencyops/data/service"),
	}
}

func (s *dataServiceImpl) ProcessTelemetry(ctx context.Context, req domain.TelemetryRequest) (*domain.TelemetryResponse, error) {
	ctx, span := s.tracer.Start(ctx, "data.ProcessTelemetry AWS Layer",
		trace.WithAttributes(
			attribute.String("aws.service", "data-service"),
			attribute.String("aws.rds.cluster", "emergency-ops-db-cluster"),
			attribute.String("report_id", req.ReportID),
		),
	)
	defer span.End()

	traceID := span.SpanContext().TraceID().String()
	spanID := span.SpanContext().SpanID().String()

	rec := domain.DataRecord{
		ReportID:  req.ReportID,
		TraceID:   traceID,
		SpanID:    spanID,
		Source:    req.Source,
		Payload:   req.Payload,
		CreatedAt: time.Now().UTC(),
	}

	saved, err := s.repo.SaveTelemetryRecord(ctx, rec)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("telemetry processing error: %w", err)
	}

	return &domain.TelemetryResponse{
		ID:        saved.ID,
		ReportID:  saved.ReportID,
		Status:    "PERSISTED_AWS_RDS",
		DBSystem:  "postgresql-rds",
		CreatedAt: saved.CreatedAt,
	}, nil
}

func (s *dataServiceImpl) HealthCheck(ctx context.Context) error {
	return s.repo.Ping(ctx)
}
