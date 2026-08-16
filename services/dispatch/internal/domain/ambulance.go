package domain

import "fmt"

// AmbulanceType define la capacidad médica de la ambulancia.
type AmbulanceType string

const (
	AmbulanceTypeAdvanced AmbulanceType = "ADVANCED" // con médico a bordo
	AmbulanceTypeBasic    AmbulanceType = "BASIC"    // solo paramédicos
)

func (t AmbulanceType) IsValid() bool {
	return t == AmbulanceTypeAdvanced || t == AmbulanceTypeBasic
}

// CanHandle indica si este tipo de ambulancia puede atender
// una emergencia con la prioridad dada.
//
// Regla: solo ADVANCED puede atender RED. BASIC puede atender YELLOW y GREEN.
func (t AmbulanceType) CanHandle(p Priority) bool {
	if p == PriorityRed {
		return t == AmbulanceTypeAdvanced
	}
	return true // ambos tipos atienden YELLOW y GREEN
}

// AmbulanceStatus indica si la ambulancia está libre o en misión.
type AmbulanceStatus string

const (
	AmbulanceStatusAvailable AmbulanceStatus = "AVAILABLE"
	AmbulanceStatusBusy      AmbulanceStatus = "BUSY"
)

func (s AmbulanceStatus) IsValid() bool {
	return s == AmbulanceStatusAvailable || s == AmbulanceStatusBusy
}

// Ambulance representa una ambulancia de la flota.
type Ambulance struct {
	ID       string
	Type     AmbulanceType
	Status   AmbulanceStatus
	Location Location
	CrewSize int
}

// Validate verifica las reglas invariantes de una ambulancia.
func (a *Ambulance) Validate() error {
	if a.ID == "" {
		return fmt.Errorf("ambulance ID es obligatorio")
	}
	if !a.Type.IsValid() {
		return fmt.Errorf("tipo de ambulancia inválido: %q", a.Type)
	}
	if !a.Status.IsValid() {
		return fmt.Errorf("estado de ambulancia inválido: %q", a.Status)
	}
	if err := a.Location.Validate(); err != nil {
		return fmt.Errorf("ubicación inválida: %w", err)
	}
	if a.CrewSize < 1 {
		return fmt.Errorf("crew size debe ser al menos 1, recibí %d", a.CrewSize)
	}
	return nil
}

// IsAvailable retorna true si la ambulancia está libre para asignarse.
func (a *Ambulance) IsAvailable() bool {
	return a.Status == AmbulanceStatusAvailable
}