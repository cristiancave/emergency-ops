package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"emergencyops/triage/internal/domain"
)

// helper: crea un reporte base para reutilizar.
func makeReport() *domain.EmergencyReport {
	return &domain.EmergencyReport{
		ID:          "REP-001",
		PatientAge:  40,
		Symptoms:    []string{"malestar general"},
		Description: "test",
		ReportedAt:  time.Now(),
	}
}

// ============================================================
// Tests del algoritmo de triage (función pura)
// No requieren mock ni service — testeamos la función directamente.
// ============================================================

func TestCalculatePriority(t *testing.T) {
	tests := []struct {
		name         string
		age          int
		symptoms     []string
		wantPriority domain.Priority
	}{
		{
			name:         "síntoma crítico dolor torácico → ROJO",
			age:          40,
			symptoms:     []string{"dolor torácico"},
			wantPriority: domain.PriorityRed,
		},
		{
			name:         "pérdida de consciencia → ROJO",
			age:          30,
			symptoms:     []string{"pérdida de consciencia"},
			wantPriority: domain.PriorityRed,
		},
		{
			name:         "case insensitive: DOLOR TORÁCICO → ROJO",
			age:          40,
			symptoms:     []string{"DOLOR TORÁCICO"},
			wantPriority: domain.PriorityRed,
		},
		{
			name:         "paciente >70 con confusión súbita → ROJO",
			age:          75,
			symptoms:     []string{"confusión súbita"},
			wantPriority: domain.PriorityRed,
		},
		{
			name:         "confusión súbita en joven → VERDE",
			age:          25,
			symptoms:     []string{"confusión súbita"},
			wantPriority: domain.PriorityGreen,
		},
		{
			name:         "fractura visible → AMARILLO",
			age:          30,
			symptoms:     []string{"fractura visible"},
			wantPriority: domain.PriorityYellow,
		},
		{
			name:         "fiebre alta → AMARILLO",
			age:          30,
			symptoms:     []string{"fiebre alta"},
			wantPriority: domain.PriorityYellow,
		},
		{
			name:         "síntoma leve → VERDE",
			age:          30,
			symptoms:     []string{"tos leve"},
			wantPriority: domain.PriorityGreen,
		},
		{
			name:         "múltiples síntomas: gana el crítico",
			age:          30,
			symptoms:     []string{"tos leve", "dolor torácico", "malestar"},
			wantPriority: domain.PriorityRed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			report := &domain.EmergencyReport{
				ID:         "TEST",
				PatientAge: tt.age,
				Symptoms:   tt.symptoms,
				ReportedAt: time.Now(),
			}

			gotPriority, gotReason := calculatePriority(report)

			if gotPriority != tt.wantPriority {
				t.Errorf("prioridad = %q, esperaba %q", gotPriority, tt.wantPriority)
			}
			if gotReason == "" {
				t.Errorf("razón no debería estar vacía")
			}
		})
	}
}

// ============================================================
// Tests del service completo usando el mock
// Aquí verificamos ORQUESTACIÓN: que el service llame a las cosas correctas
// en el orden correcto y maneje los errores bien.
// ============================================================

func TestTriageService_ClassifyEmergency_HappyPath(t *testing.T) {
	ctx := context.Background()
	mock := &mockRepository{}
	svc := NewTriageService(mock)

	report := makeReport()
	report.Symptoms = []string{"dolor torácico"}

	result, err := svc.ClassifyEmergency(ctx, report)

	if err != nil {
		t.Fatalf("no esperaba error: %v", err)
	}

	// Verificar el resultado retornado.
	if result.Priority != domain.PriorityRed {
		t.Errorf("prioridad = %q, esperaba %q", result.Priority, domain.PriorityRed)
	}
	if result.ReportID != report.ID {
		t.Errorf("ReportID = %q, esperaba %q", result.ReportID, report.ID)
	}

	// Verificar interacciones con el repositorio.
	if mock.SaveReportCalls != 1 {
		t.Errorf("SaveReport llamado %d veces, esperaba 1", mock.SaveReportCalls)
	}
	if mock.SaveResultCalls != 1 {
		t.Errorf("SaveResult llamado %d veces, esperaba 1", mock.SaveResultCalls)
	}
}

func TestTriageService_ClassifyEmergency_InvalidReport(t *testing.T) {
	ctx := context.Background()
	mock := &mockRepository{}
	svc := NewTriageService(mock)

	report := makeReport()
	report.PatientAge = -5 // inválido

	_, err := svc.ClassifyEmergency(ctx, report)

	if err == nil {
		t.Fatal("esperaba error de validación, recibí nil")
	}

	// Verificar que NO se llamó al repositorio (falló antes).
	if mock.SaveReportCalls != 0 {
		t.Errorf("SaveReport no debería haberse llamado, se llamó %d veces", mock.SaveReportCalls)
	}
}

func TestTriageService_ClassifyEmergency_SaveReportFails(t *testing.T) {
	ctx := context.Background()
	mock := &mockRepository{
		SaveReportFn: func(ctx context.Context, r *domain.EmergencyReport) error {
			return domain.ErrDuplicateID
		},
	}
	svc := NewTriageService(mock)

	_, err := svc.ClassifyEmergency(ctx, makeReport())

	if err == nil {
		t.Fatal("esperaba error, recibí nil")
	}

	// El error debe envolver al original.
	if !errors.Is(err, domain.ErrDuplicateID) {
		t.Errorf("esperaba envolver ErrDuplicateID, recibí: %v", err)
	}

	// El mensaje debe tener contexto.
	if !strings.Contains(err.Error(), "saving report") {
		t.Errorf("esperaba contexto 'saving report' en: %q", err.Error())
	}

	// NO debería haberse llamado a SaveResult (falló antes).
	if mock.SaveResultCalls != 0 {
		t.Errorf("SaveResult no debería haberse llamado")
	}
}

func TestTriageService_ClassifyEmergency_UsesInjectedClock(t *testing.T) {
	ctx := context.Background()
	mock := &mockRepository{}
	svc := NewTriageService(mock)

	// Inyectar un reloj fijo.
	fixedTime := time.Date(2026, 1, 15, 10, 30, 0, 0, time.UTC)
	svc.now = func() time.Time { return fixedTime }

	result, err := svc.ClassifyEmergency(ctx, makeReport())
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}

	if !result.ClassifiedAt.Equal(fixedTime) {
		t.Errorf("ClassifiedAt = %v, esperaba %v", result.ClassifiedAt, fixedTime)
	}
}

func TestTriageService_GetResult_Success(t *testing.T) {
	ctx := context.Background()

	expected := &domain.TriageResult{
		ReportID: "REP-001",
		Priority: domain.PriorityYellow,
	}

	mock := &mockRepository{
		FindResultByReportIDFn: func(ctx context.Context, id string) (*domain.TriageResult, error) {
			if id != "REP-001" {
				t.Errorf("se llamó con ID %q, esperaba REP-001", id)
			}
			return expected, nil
		},
	}
	svc := NewTriageService(mock)

	got, err := svc.GetResult(ctx, "REP-001")
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if got.Priority != expected.Priority {
		t.Errorf("prioridad = %q, esperaba %q", got.Priority, expected.Priority)
	}
}

func TestTriageService_GetResult_EmptyID(t *testing.T) {
	ctx := context.Background()
	svc := NewTriageService(&mockRepository{})

	_, err := svc.GetResult(ctx, "")
	if err == nil {
		t.Fatal("esperaba error para ID vacío")
	}
}

func TestTriageService_GetResult_NotFound(t *testing.T) {
	ctx := context.Background()
	mock := &mockRepository{} // default retorna ErrReportNotFound
	svc := NewTriageService(mock)

	_, err := svc.GetResult(ctx, "NO-EXISTE")
	if !errors.Is(err, domain.ErrReportNotFound) {
		t.Errorf("esperaba ErrReportNotFound, recibí: %v", err)
	}
}