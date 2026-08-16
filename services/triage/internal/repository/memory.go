package repository

import (
	"context"
	"sync"
	"emergencyops/triage/internal/domain"
)

// MemoryRepository es una implementación en memoria de TriageRepository.
// Es útil para tests, prototipos y desarrollo local sin BD.
//
// Es thread-safe: múltiples goroutines pueden leer/escribir concurrentemente
// gracias al RWMutex.
type MemoryRepository struct {
	mu      sync.RWMutex
	reports map[string]*domain.EmergencyReport // key: report ID
	results map[string]*domain.TriageResult    // key: report ID
}

// NewMemoryRepository crea una nueva instancia vacía del repositorio en memoria.
// Este es el "constructor" idiomático de Go.
func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		reports: make(map[string]*domain.EmergencyReport),
		results: make(map[string]*domain.TriageResult),
	}
}

// SaveReport persiste un reporte. Retorna ErrDuplicateID si el ID ya existe.
func (r *MemoryRepository) SaveReport(ctx context.Context, report *domain.EmergencyReport) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.reports[report.ID]; exists {
		return domain.ErrDuplicateID
	}

	// Guardamos una copia para evitar que el caller mute nuestro estado interno.
	copy := *report
	r.reports[report.ID] = &copy

	return nil
}

// FindReportByID busca un reporte por su ID.
func (r *MemoryRepository) FindReportByID(ctx context.Context, id string) (*domain.EmergencyReport, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	report, exists := r.reports[id]
	if !exists {
		return nil, domain.ErrReportNotFound
	}

	// Retornamos una copia para que el caller no pueda mutar nuestro estado interno.
	copy := *report
	return &copy, nil
}

// SaveResult persiste el resultado del triage.
func (r *MemoryRepository) SaveResult(ctx context.Context, result *domain.TriageResult) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	copy := *result
	r.results[result.ReportID] = &copy

	return nil
}

// FindResultByReportID busca el resultado de triage de un reporte.
func (r *MemoryRepository) FindResultByReportID(ctx context.Context, reportID string) (*domain.TriageResult, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result, exists := r.results[reportID]
	if !exists {
		return nil, domain.ErrReportNotFound
	}

	copy := *result
	return &copy, nil
}