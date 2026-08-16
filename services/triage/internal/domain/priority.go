package domain

// Priority representa el nivel de urgencia de una emergencia médica,
// inspirado en el Manchester Triage System.
type Priority string

const (
	PriorityRed    Priority = "RED"    // Crítico, atención inmediata
	PriorityYellow Priority = "YELLOW" // Urgente, atención en ≤ 30 min
	PriorityGreen  Priority = "GREEN"  // No urgente, atención en ≤ 2h
)

// IsValid retorna true si la prioridad es uno de los valores permitidos.
func (p Priority) IsValid() bool {
	switch p {
	case PriorityRed, PriorityYellow, PriorityGreen:
		return true
	default:
		return false
	}
}

// String satisface la interfaz fmt.Stringer para imprimir la prioridad legible.
func (p Priority) String() string {
	return string(p)
}