package service

import (
	"context"
	"sync"

	"emergencyops/triage/internal/domain"
)

// mockRepository es una implementación falsa de TriageRepository para tests.
// Nos permite:
//  1. Inspeccionar qué se llamó (para verificar comportamiento).
//  2. Inyectar respuestas específicas (incluyendo errores).
//  3. Testear el service SIN necesidad de BD ni memoria real.
type mockRepository struct {
	mu sync.Mutex

	// Hooks: si están definidos, se llaman en vez de la lógica default.
	// Permiten simular errores o respuestas específicas por test.
	SaveReportFn           func(ctx context.Context, report *domain.EmergencyReport) error
	FindReportByIDFn       func(ctx context.Context, id string) (*domain.EmergencyReport, error)
	SaveResultFn           func(ctx context.Context, result *domain.TriageResult) error
	FindResultByReportIDFn func(ctx context.Context, reportID string) (*domain.TriageResult, error)

	// Registros de llamadas: qué se pasó y cuántas veces.
	SavedReports  []*domain.EmergencyReport
	SavedResults  []*domain.TriageResult
	SaveReportCalls int
	SaveResultCalls int
}

// Verificación en tiempo de compilación.
var _ domain.TriageRepository = (*mockRepository)(nil)

func (m *mockRepository) SaveReport(ctx context.Context, report *domain.EmergencyReport) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.SaveReportCalls++
	m.SavedReports = append(m.SavedReports, report)

	if m.SaveReportFn != nil {
		return m.SaveReportFn(ctx, report)
	}
	return nil
}

func (m *mockRepository) FindReportByID(ctx context.Context, id string) (*domain.EmergencyReport, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.FindReportByIDFn != nil {
		return m.FindReportByIDFn(ctx, id)
	}
	return nil, domain.ErrReportNotFound
}

func (m *mockRepository) SaveResult(ctx context.Context, result *domain.TriageResult) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.SaveResultCalls++
	m.SavedResults = append(m.SavedResults, result)

	if m.SaveResultFn != nil {
		return m.SaveResultFn(ctx, result)
	}
	return nil
}

func (m *mockRepository) FindResultByReportID(ctx context.Context, reportID string) (*domain.TriageResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.FindResultByReportIDFn != nil {
		return m.FindResultByReportIDFn(ctx, reportID)
	}
	return nil, domain.ErrReportNotFound
}