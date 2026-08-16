package repository

import (
	"context"
	"sync"

	"emergencyops/dispatch/internal/domain"
)

// MemoryAmbulanceRepository es la implementación en memoria de AmbulanceRepository.
// Thread-safe gracias al RWMutex.
type MemoryAmbulanceRepository struct {
	mu         sync.RWMutex
	ambulances map[string]*domain.Ambulance
}

// Verificación en tiempo de compilación.
var _ domain.AmbulanceRepository = (*MemoryAmbulanceRepository)(nil)

// NewMemoryAmbulanceRepository crea un repositorio vacío.
func NewMemoryAmbulanceRepository() *MemoryAmbulanceRepository {
	return &MemoryAmbulanceRepository{
		ambulances: make(map[string]*domain.Ambulance),
	}
}

// NewMemoryAmbulanceRepositoryWithSeed crea un repositorio pre-poblado
// con una flota de ambulancias en distintas ubicaciones de Bogotá.
// Útil para desarrollo local y demos sin necesidad de setup previo.
func NewMemoryAmbulanceRepositoryWithSeed() *MemoryAmbulanceRepository {
	repo := NewMemoryAmbulanceRepository()

	seed := []*domain.Ambulance{
		{
			ID:       "AMB-001",
			Type:     domain.AmbulanceTypeAdvanced,
			Status:   domain.AmbulanceStatusAvailable,
			Location: domain.Location{Latitude: 4.7110, Longitude: -74.0721}, // Centro
			CrewSize: 3,
		},
		{
			ID:       "AMB-002",
			Type:     domain.AmbulanceTypeAdvanced,
			Status:   domain.AmbulanceStatusAvailable,
			Location: domain.Location{Latitude: 4.6768, Longitude: -74.0483}, // Chapinero
			CrewSize: 3,
		},
		{
			ID:       "AMB-003",
			Type:     domain.AmbulanceTypeBasic,
			Status:   domain.AmbulanceStatusAvailable,
			Location: domain.Location{Latitude: 4.7570, Longitude: -74.0450}, // Usaquén
			CrewSize: 2,
		},
		{
			ID:       "AMB-004",
			Type:     domain.AmbulanceTypeBasic,
			Status:   domain.AmbulanceStatusAvailable,
			Location: domain.Location{Latitude: 4.6280, Longitude: -74.0655}, // Teusaquillo
			CrewSize: 2,
		},
		{
			ID:       "AMB-005",
			Type:     domain.AmbulanceTypeAdvanced,
			Status:   domain.AmbulanceStatusBusy, // ocupada al inicio
			Location: domain.Location{Latitude: 4.6097, Longitude: -74.0817}, // La Candelaria
			CrewSize: 3,
		},
		{
			ID:       "AMB-006",
			Type:     domain.AmbulanceTypeBasic,
			Status:   domain.AmbulanceStatusAvailable,
			Location: domain.Location{Latitude: 4.7290, Longitude: -74.0940}, // Suba
			CrewSize: 2,
		},
	}

	// Los guardamos usando el mismo método público. Ignoramos errores
	// porque los datos seed están controlados por nosotros.
	ctx := context.Background()
	for _, amb := range seed {
		_ = repo.save(ctx, amb)
	}

	return repo
}

// save es privado: úsalo internamente cuando necesites crear/actualizar.
// La interfaz pública no permite crear ambulancias arbitrariamente,
// porque el flujo real es que la flota se gestiona por administración.
func (r *MemoryAmbulanceRepository) save(ctx context.Context, amb *domain.Ambulance) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	copy := *amb
	r.ambulances[amb.ID] = &copy
	return nil
}

// FindAvailable retorna todas las ambulancias con status AVAILABLE.
func (r *MemoryAmbulanceRepository) FindAvailable(ctx context.Context) ([]*domain.Ambulance, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]*domain.Ambulance, 0)
	for _, amb := range r.ambulances {
		if amb.IsAvailable() {
			copy := *amb
			result = append(result, &copy)
		}
	}

	return result, nil
}

// UpdateStatus cambia el estado de una ambulancia. Retorna ErrAmbulanceNotFound
// si no existe.
func (r *MemoryAmbulanceRepository) UpdateStatus(ctx context.Context, ambulanceID string, status domain.AmbulanceStatus) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	amb, exists := r.ambulances[ambulanceID]
	if !exists {
		return domain.ErrAmbulanceNotFound
	}

	amb.Status = status
	return nil
}

// FindByID busca una ambulancia por su ID.
func (r *MemoryAmbulanceRepository) FindByID(ctx context.Context, id string) (*domain.Ambulance, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	amb, exists := r.ambulances[id]
	if !exists {
		return nil, domain.ErrAmbulanceNotFound
	}

	copy := *amb
	return &copy, nil
}