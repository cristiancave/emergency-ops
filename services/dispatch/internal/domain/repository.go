package domain

import "context"

// AmbulanceRepository gestiona el acceso a la flota de ambulancias.
type AmbulanceRepository interface {
	// FindAvailable retorna todas las ambulancias disponibles (Status = AVAILABLE).
	// El service se encarga de filtrarlas por tipo/distancia.
	FindAvailable(ctx context.Context) ([]*Ambulance, error)

	// UpdateStatus cambia el estado de una ambulancia.
	// Se usa cuando se despacha (AVAILABLE → BUSY) o se libera (BUSY → AVAILABLE).
	UpdateStatus(ctx context.Context, ambulanceID string, status AmbulanceStatus) error

	// FindByID busca una ambulancia específica.
	FindByID(ctx context.Context, id string) (*Ambulance, error)
}

// DispatchRepository gestiona la persistencia de despachos.
type DispatchRepository interface {
	Save(ctx context.Context, dispatch *Dispatch) error
	FindByID(ctx context.Context, id string) (*Dispatch, error)
}