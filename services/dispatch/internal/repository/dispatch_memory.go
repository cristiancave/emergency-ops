package repository

import (
	"context"
	"sync"

	"emergencyops/dispatch/internal/domain"
)

// MemoryDispatchRepository implementa DispatchRepository en memoria.
type MemoryDispatchRepository struct {
	mu        sync.RWMutex
	dispatches map[string]*domain.Dispatch
}

var _ domain.DispatchRepository = (*MemoryDispatchRepository)(nil)

// NewMemoryDispatchRepository crea un repositorio vacío.
func NewMemoryDispatchRepository() *MemoryDispatchRepository {
	return &MemoryDispatchRepository{
		dispatches: make(map[string]*domain.Dispatch),
	}
}

// Save persiste un despacho. Retorna ErrDuplicateDispatchID si ya existe.
func (r *MemoryDispatchRepository) Save(ctx context.Context, d *domain.Dispatch) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.dispatches[d.ID]; exists {
		return domain.ErrDuplicateDispatchID
	}

	copy := *d
	r.dispatches[d.ID] = &copy
	return nil
}

// FindByID busca un despacho por su ID.
func (r *MemoryDispatchRepository) FindByID(ctx context.Context, id string) (*domain.Dispatch, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	d, exists := r.dispatches[id]
	if !exists {
		return nil, domain.ErrDispatchNotFound
	}

	copy := *d
	return &copy, nil
}