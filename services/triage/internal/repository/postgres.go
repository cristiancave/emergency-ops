package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"emergencyops/triage/internal/domain"
)

// PostgresRepository es la implementación en Postgres de domain.TriageRepository.
type PostgresRepository struct {
	db *sql.DB
}

// NewPostgresRepository construye el repositorio a partir de una conexión ya abierta.
// El caller es responsable de cerrar db.
func NewPostgresRepository(db *sql.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

// SaveReport persiste un reporte. Retorna ErrDuplicateID si el ID ya existe.
func (r *PostgresRepository) SaveReport(ctx context.Context, report *domain.EmergencyReport) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO emergency_reports (id, patient_age, symptoms, description, reported_at)
		VALUES ($1, $2, $3, $4, $5)
	`, report.ID, report.PatientAge, strings.Join(report.Symptoms, ","), report.Description, report.ReportedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return domain.ErrDuplicateID
		}
		return fmt.Errorf("insert emergency_report: %w", err)
	}
	return nil
}

// FindReportByID busca un reporte por su ID.
func (r *PostgresRepository) FindReportByID(ctx context.Context, id string) (*domain.EmergencyReport, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, patient_age, symptoms, description, reported_at
		FROM emergency_reports
		WHERE id = $1
	`, id)

	var report domain.EmergencyReport
	var symptoms string
	if err := row.Scan(&report.ID, &report.PatientAge, &symptoms, &report.Description, &report.ReportedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrReportNotFound
		}
		return nil, fmt.Errorf("select emergency_report: %w", err)
	}
	report.Symptoms = strings.Split(symptoms, ",")

	return &report, nil
}

// SaveResult persiste el resultado del triage (upsert: un resultado por reporte).
func (r *PostgresRepository) SaveResult(ctx context.Context, result *domain.TriageResult) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO triage_results (report_id, priority, reason, classified_at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (report_id) DO UPDATE
		SET priority = EXCLUDED.priority, reason = EXCLUDED.reason, classified_at = EXCLUDED.classified_at
	`, result.ReportID, string(result.Priority), result.Reason, result.ClassifiedAt)
	if err != nil {
		return fmt.Errorf("upsert triage_result: %w", err)
	}
	return nil
}

// FindResultByReportID busca el resultado de triage de un reporte.
func (r *PostgresRepository) FindResultByReportID(ctx context.Context, reportID string) (*domain.TriageResult, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT report_id, priority, reason, classified_at
		FROM triage_results
		WHERE report_id = $1
	`, reportID)

	var result domain.TriageResult
	var priority string
	if err := row.Scan(&result.ReportID, &priority, &result.Reason, &result.ClassifiedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrReportNotFound
		}
		return nil, fmt.Errorf("select triage_result: %w", err)
	}
	result.Priority = domain.Priority(priority)

	return &result, nil
}

// isUniqueViolation detecta violaciones de constraint UNIQUE/PK en Postgres (SQLSTATE 23505)
// sin acoplar este archivo al tipo de error concreto del driver pgx.
func isUniqueViolation(err error) bool {
	return strings.Contains(err.Error(), "23505")
}

// PingWithRetry espera a que la base de datos esté lista, reintentando con backoff fijo.
// Útil en el arranque del servicio si Postgres tarda en aceptar conexiones (p.ej. justo
// después de crear la instancia RDS o al levantar el contenedor local).
func PingWithRetry(ctx context.Context, db *sql.DB, attempts int, delay time.Duration) error {
	var lastErr error
	for i := 0; i < attempts; i++ {
		if lastErr = db.PingContext(ctx); lastErr == nil {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
		}
	}
	return fmt.Errorf("db not ready after %d attempts: %w", attempts, lastErr)
}
