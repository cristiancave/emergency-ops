package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	"emergencyops/triage/internal/domain"
)

// Verificación en tiempo de compilación: MemoryRepository DEBE satisfacer
// la interfaz TriageRepository. Si en el futuro cambiamos la interfaz sin
// actualizar la implementación, este archivo NO COMPILA.
var _ domain.TriageRepository = (*MemoryRepository)(nil)

// helper: crea un reporte válido para reutilizar en los tests.
func makeReport(id string) *domain.EmergencyReport {
	return &domain.EmergencyReport{
		ID:          id,
		PatientAge:  40,
		Symptoms:    []string{"dolor torácico"},
		Description: "test",
		ReportedAt:  time.Now(),
	}
}

func TestMemoryRepository_SaveAndFindReport(t *testing.T) {
	ctx := context.Background()
	repo := NewMemoryRepository()

	report := makeReport("REP-001")

	// Guardar debe funcionar la primera vez.
	if err := repo.SaveReport(ctx, report); err != nil {
		t.Fatalf("SaveReport falló: %v", err)
	}

	// Recuperar el reporte guardado.
	found, err := repo.FindReportByID(ctx, "REP-001")
	if err != nil {
		t.Fatalf("FindReportByID falló: %v", err)
	}

	if found.ID != report.ID {
		t.Errorf("esperaba ID %q, recibí %q", report.ID, found.ID)
	}
	if found.PatientAge != report.PatientAge {
		t.Errorf("esperaba edad %d, recibí %d", report.PatientAge, found.PatientAge)
	}
	if len(found.Symptoms) != len(report.Symptoms) {
		t.Errorf("esperaba %d síntomas, recibí %d", len(report.Symptoms), len(found.Symptoms))
	}
}

func TestMemoryRepository_SaveReport_DuplicateID(t *testing.T) {
	ctx := context.Background()
	repo := NewMemoryRepository()

	report := makeReport("REP-DUP")

	// Primera vez: OK.
	if err := repo.SaveReport(ctx, report); err != nil {
		t.Fatalf("primer SaveReport falló: %v", err)
	}

	// Segunda vez con el mismo ID: debe fallar con ErrDuplicateID.
	err := repo.SaveReport(ctx, report)
	if err == nil {
		t.Fatal("esperaba error al guardar ID duplicado, recibí nil")
	}
	if !errors.Is(err, domain.ErrDuplicateID) {
		t.Errorf("esperaba ErrDuplicateID, recibí: %v", err)
	}
}

func TestMemoryRepository_FindReport_NotFound(t *testing.T) {
	ctx := context.Background()
	repo := NewMemoryRepository()

	_, err := repo.FindReportByID(ctx, "REP-NO-EXISTE")
	if err == nil {
		t.Fatal("esperaba error al buscar ID inexistente, recibí nil")
	}
	if !errors.Is(err, domain.ErrReportNotFound) {
		t.Errorf("esperaba ErrReportNotFound, recibí: %v", err)
	}
}

func TestMemoryRepository_SaveAndFindResult(t *testing.T) {
	ctx := context.Background()
	repo := NewMemoryRepository()

	result := &domain.TriageResult{
		ReportID:     "REP-001",
		Priority:     domain.PriorityRed,
		Reason:       "dolor torácico + edad > 60",
		ClassifiedAt: time.Now(),
	}

	if err := repo.SaveResult(ctx, result); err != nil {
		t.Fatalf("SaveResult falló: %v", err)
	}

	found, err := repo.FindResultByReportID(ctx, "REP-001")
	if err != nil {
		t.Fatalf("FindResultByReportID falló: %v", err)
	}

	if found.Priority != domain.PriorityRed {
		t.Errorf("esperaba prioridad %q, recibí %q", domain.PriorityRed, found.Priority)
	}
}

func TestMemoryRepository_FindResult_NotFound(t *testing.T) {
	ctx := context.Background()
	repo := NewMemoryRepository()

	_, err := repo.FindResultByReportID(ctx, "REP-NO-EXISTE")
	if !errors.Is(err, domain.ErrReportNotFound) {
		t.Errorf("esperaba ErrReportNotFound, recibí: %v", err)
	}
}

// Test que verifica que el repositorio devuelve COPIAS, no referencias
// al estado interno. Si mutamos el resultado, el estado interno no debe cambiar.
func TestMemoryRepository_ReturnsDefensiveCopy(t *testing.T) {
	ctx := context.Background()
	repo := NewMemoryRepository()

	report := makeReport("REP-COPY")
	if err := repo.SaveReport(ctx, report); err != nil {
		t.Fatalf("SaveReport falló: %v", err)
	}

	// Recuperar y mutar la copia recibida.
	found, _ := repo.FindReportByID(ctx, "REP-COPY")
	found.PatientAge = 999
	found.Description = "MUTATED"

	// Volver a recuperar: debe seguir teniendo los valores originales.
	fresh, _ := repo.FindReportByID(ctx, "REP-COPY")
	if fresh.PatientAge == 999 {
		t.Error("el repositorio devuelve referencias mutables, no copias defensivas")
	}
	if fresh.Description == "MUTATED" {
		t.Error("el repositorio devuelve referencias mutables, no copias defensivas")
	}
}