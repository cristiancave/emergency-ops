package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"emergencyops/data/internal/domain"
	"emergencyops/data/internal/service"
	"emergencyops/pkg/logger"
	"go.opentelemetry.io/otel/trace"
)

type HTTPHandler struct {
	svc service.DataService
	log *slog.Logger
}

func NewHTTPHandler(svc service.DataService) *HTTPHandler {
	return &HTTPHandler{
		svc: svc,
		log: logger.New("data-service"),
	}
}

func (h *HTTPHandler) ProcessTelemetry(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	span := trace.SpanFromContext(ctx)
	traceID := span.SpanContext().TraceID().String()
	spanID := span.SpanContext().SpanID().String()

	var req domain.TelemetryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Ctx(ctx, h.log).Error("Failed to decode telemetry JSON",
			"error", err.Error(),
			"trace_id", traceID,
			"span_id", spanID,
		)
		http.Error(w, `{"error":"invalid_payload"}`, http.StatusBadRequest)
		return
	}

	if req.ReportID == "" {
		req.ReportID = "RPT-AWS-DATA-GEN"
	}
	if req.Source == "" {
		req.Source = "aws-rds-postgresql"
	}

	resp, err := h.svc.ProcessTelemetry(ctx, req)
	if err != nil {
		logger.Ctx(ctx, h.log).Error("Failed to persist telemetry record",
			"error", err.Error(),
			"trace_id", traceID,
			"span_id", spanID,
			"report_id", req.ReportID,
		)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"error":    err.Error(),
			"trace_id": traceID,
			"status":   "DB_PERSISTENCE_FAILURE",
		})
		return
	}

	logger.Ctx(ctx, h.log).Info("Data telemetry record persisted to AWS RDS",
		"record_id", resp.ID,
		"report_id", resp.ReportID,
		"status", resp.Status,
		"trace_id", traceID,
		"span_id", spanID,
	)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}

func (h *HTTPHandler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if err := h.svc.HealthCheck(ctx); err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]string{"status": "UNHEALTHY", "error": err.Error()})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    "UP",
		"service":   "data-service",
		"provider":  "AWS",
		"timestamp": time.Now().UTC(),
	})
}
