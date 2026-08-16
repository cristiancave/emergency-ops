package domain

import (
	"fmt"
	"time"
)

// Priority es la prioridad de una emergencia. Se replica aquí para que
// dispatch no tenga que importar el módulo triage.
//
// En una arquitectura de microservicios, cada servicio define sus propios
// tipos. Los contratos entre servicios (JSON) hacen la traducción.
type Priority string

const (
	PriorityRed    Priority = "RED"
	PriorityYellow Priority = "YELLOW"
	PriorityGreen  Priority = "GREEN"
)

func (p Priority) IsValid() bool {
	return p == PriorityRed || p == PriorityYellow || p == PriorityGreen
}

// Dispatch representa la asignación de una ambulancia a una emergencia.
type Dispatch struct {
	ID                       string    // ID único del despacho
	ReportID                 string    // ID del reporte de emergencia original
	AmbulanceID              string    // ID de la ambulancia asignada
	Priority                 Priority  // prioridad clasificada por triage
	IncidentLocation         Location  // dónde ocurre la emergencia
	AmbulanceLocation        Location  // dónde está la ambulancia al momento del despacho
	EstimatedArrivalMinutes  int       // ETA en minutos
	CreatedAt                time.Time
}

// Validate verifica las invariantes del despacho.
func (d *Dispatch) Validate() error {
	if d.ID == "" {
		return fmt.Errorf("dispatch ID es obligatorio")
	}
	if d.ReportID == "" {
		return fmt.Errorf("report ID es obligatorio")
	}
	if d.AmbulanceID == "" {
		return fmt.Errorf("ambulance ID es obligatorio")
	}
	if !d.Priority.IsValid() {
		return fmt.Errorf("prioridad inválida: %q", d.Priority)
	}
	if err := d.IncidentLocation.Validate(); err != nil {
		return fmt.Errorf("ubicación del incidente inválida: %w", err)
	}
	if err := d.AmbulanceLocation.Validate(); err != nil {
		return fmt.Errorf("ubicación de la ambulancia inválida: %w", err)
	}
	if d.CreatedAt.IsZero() {
		return fmt.Errorf("CreatedAt es obligatorio")
	}
	return nil
}