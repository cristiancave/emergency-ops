package domain

import (
	"time"
)

// EmergencyReport es un reporte de emergencia entrante, antes de ser clasificado.
type EmergencyReport struct {
	ID          string    // identificador único (UUID)
	PatientAge  int       // edad del paciente en años
	Symptoms    []string  // lista de síntomas detectados
	Description string    // descripción libre del operador
	ReportedAt  time.Time // momento en que se recibió el reporte
}

// Validate verifica que el reporte cumpla las reglas invariantes del dominio.
// Devuelve un error si alguna regla se incumple.
func (r *EmergencyReport) Validate() error {
	if r.ID == "" {
		return ErrInvalidReport("ID es obligatorio")
	}
	if r.PatientAge < 0 || r.PatientAge > 130 {
		return ErrInvalidReport("edad debe estar entre 0 y 130 años")
	}
	if len(r.Symptoms) == 0 {
		return ErrInvalidReport("debe reportarse al menos un síntoma")
	}
	if r.ReportedAt.IsZero() {
		return ErrInvalidReport("fecha de reporte es obligatoria")
	}
	return nil
}

// TriageResult es el resultado de clasificar una emergencia.
type TriageResult struct {
	ReportID     string
	Priority     Priority
	Reason       string    // explicación de por qué se asignó esa prioridad
	ClassifiedAt time.Time
}