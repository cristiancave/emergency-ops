package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"emergencyops/triage/internal/domain"
)

// tracer para spans custom de lógica de negocio (no HTTP/DB, eso lo cubre
// la auto-instrumentación). Un tracer por paquete es la convención de OTel.
var tracer = otel.Tracer("emergencyops/triage/service")

// Síntomas críticos que disparan prioridad ROJA por sí solos.
var criticalSymptoms = map[string]bool{
	"pérdida de consciencia":          true,
	"dificultad respiratoria severa":  true,
	"dolor torácico":                  true,
	"sangrado abundante":              true,
}

// Síntomas que en pacientes mayores de 70 elevan a prioridad ROJA.
var elderlyRedFlagSymptoms = map[string]bool{
	"dolor torácico":                 true,
	"dificultad respiratoria severa": true,
	"sangrado abundante":             true,
	"confusión súbita":               true,
	"caída con golpe en la cabeza":   true,
}

// Síntomas que disparan prioridad AMARILLA.
var urgentSymptoms = map[string]bool{
	"fractura visible": true,
	"dolor severo":     true,
	"fiebre alta":      true,
}

// TriageService encapsula la lógica de clasificación de emergencias.
// Depende de TriageRepository (una interface), no de una implementación
// concreta — esto es Dependency Inversion Principle en acción.
type TriageService struct {
	repo domain.TriageRepository
	now  func() time.Time // función inyectable para poder testear con tiempo fijo
}

// NewTriageService construye un TriageService con las dependencias inyectadas.
func NewTriageService(repo domain.TriageRepository) *TriageService {
	return &TriageService{
		repo: repo,
		now:  time.Now, // por defecto usa el reloj real
	}
}

// ClassifyEmergency procesa un reporte de emergencia:
//  1. Valida el reporte.
//  2. Lo guarda en el repositorio.
//  3. Aplica el algoritmo de triage.
//  4. Guarda el resultado.
//  5. Retorna el resultado.
func (s *TriageService) ClassifyEmergency(ctx context.Context, report *domain.EmergencyReport) (*domain.TriageResult, error) {
	// 1. Validación del reporte (regla del domain)
	if err := report.Validate(); err != nil {
		return nil, fmt.Errorf("validating report: %w", err)
	}

	// 2. Persistir el reporte
	if err := s.repo.SaveReport(ctx, report); err != nil {
		return nil, fmt.Errorf("saving report: %w", err)
	}

	// 3. Aplicar el algoritmo de triage (span custom: es la lógica de
	// negocio crítica del servicio, a diferencia de HTTP/DB que ya vienen
	// instrumentados automáticamente).
	_, span := tracer.Start(ctx, "calculatePriority",
		trace.WithAttributes(
			attribute.Int("triage.patient_age", report.PatientAge),
			attribute.Int("triage.symptom_count", len(report.Symptoms)),
		),
	)
	priority, reason := calculatePriority(report)
	span.SetAttributes(
		attribute.String("triage.priority", string(priority)),
		attribute.String("triage.reason", reason),
	)
	span.End()

	result := &domain.TriageResult{
		ReportID:     report.ID,
		Priority:     priority,
		Reason:       reason,
		ClassifiedAt: s.now(),
	}

	// 4. Persistir el resultado
	if err := s.repo.SaveResult(ctx, result); err != nil {
		return nil, fmt.Errorf("saving triage result: %w", err)
	}

	// 5. Devolver el resultado
	return result, nil
}

// GetResult recupera el resultado de triage de un reporte previamente clasificado.
func (s *TriageService) GetResult(ctx context.Context, reportID string) (*domain.TriageResult, error) {
	if reportID == "" {
		return nil, fmt.Errorf("reportID is required")
	}

	result, err := s.repo.FindResultByReportID(ctx, reportID)
	if err != nil {
		return nil, fmt.Errorf("finding result: %w", err)
	}
	return result, nil
}

// calculatePriority aplica el algoritmo de triage sobre un reporte.
// Retorna la prioridad asignada y la razón textual (para trazabilidad).
//
// Esta función es privada (minúscula) porque es un detalle interno del service.
// Es pura: no tiene efectos secundarios, no depende del reloj ni de la BD.
// Por eso podemos testearla directamente sin mocks.
func calculatePriority(report *domain.EmergencyReport) (domain.Priority, string) {
	// Normalizamos síntomas a minúsculas para comparación.
	symptoms := normalizeSymptoms(report.Symptoms)

	// Regla 1: síntomas críticos → ROJO
	for _, s := range symptoms {
		if criticalSymptoms[s] {
			return domain.PriorityRed, fmt.Sprintf("síntoma crítico detectado: %s", s)
		}
	}

	// Regla 2: paciente mayor con síntomas de alarma → ROJO
	if report.PatientAge > 70 {
		for _, s := range symptoms {
			if elderlyRedFlagSymptoms[s] {
				return domain.PriorityRed, fmt.Sprintf("paciente >70 años con síntoma de alarma: %s", s)
			}
		}
	}

	// Regla 3: síntomas urgentes → AMARILLO
	for _, s := range symptoms {
		if urgentSymptoms[s] {
			return domain.PriorityYellow, fmt.Sprintf("síntoma urgente: %s", s)
		}
	}

	// Regla 4: default → VERDE
	return domain.PriorityGreen, "sin síntomas de alarma detectados"
}

// normalizeSymptoms convierte los síntomas a minúsculas y quita espacios
// para que la comparación sea case-insensitive.
func normalizeSymptoms(symptoms []string) []string {
	normalized := make([]string, len(symptoms))
	for i, s := range symptoms {
		normalized[i] = strings.ToLower(strings.TrimSpace(s))
	}
	return normalized
}