package domain

import "context"

// TriageRepository define el contrato para persistir y recuperar
// reportes de emergencia y sus resultados de triage.
//
// El domain NO sabe cómo se implementa (memoria, PostgreSQL, Redis, etc.).
// Solo define QUÉ operaciones existen. Las implementaciones viven en
// el paquete repository/.
type TriageRepository interface {
	// SaveReport persiste un nuevo reporte de emergencia.
	// Retorna ErrDuplicateID si ya existe un reporte con ese ID.
	SaveReport(ctx context.Context, report *EmergencyReport) error

	// FindReportByID busca un reporte por su ID.
	// Retorna ErrReportNotFound si no existe.
	FindReportByID(ctx context.Context, id string) (*EmergencyReport, error)

	// SaveResult persiste el resultado del triage para un reporte.
	SaveResult(ctx context.Context, result *TriageResult) error

	// FindResultByReportID busca el resultado de triage de un reporte.
	// Retorna ErrReportNotFound si no existe.
	FindResultByReportID(ctx context.Context, reportID string) (*TriageResult, error)
}