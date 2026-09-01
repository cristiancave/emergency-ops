package domain

import "time"

type DataRecord struct {
	ID        string    `json:"id"`
	ReportID  string    `json:"report_id"`
	TraceID   string    `json:"trace_id"`
	SpanID    string    `json:"span_id"`
	Source    string    `json:"source"`    // e.g. aws-rds-postgresql
	Payload   string    `json:"payload"`
	CreatedAt time.Time `json:"created_at"`
}

type TelemetryRequest struct {
	ReportID string `json:"report_id"`
	Source   string `json:"source"`
	Payload  string `json:"payload"`
}

type TelemetryResponse struct {
	ID        string    `json:"id"`
	ReportID  string    `json:"report_id"`
	Status    string    `json:"status"`
	DBSystem  string    `json:"db_system"`
	CreatedAt time.Time `json:"created_at"`
}
