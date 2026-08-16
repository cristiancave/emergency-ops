package service

import (
	"context"
	"sync"

	"emergencyops/dispatch/internal/client"
	"emergencyops/dispatch/internal/domain"
)

// mockTriageClient simula triage-service sin tocar la red.
type mockTriageClient struct {
	mu        sync.Mutex
	ClassifyFn func(ctx context.Context, req client.TriageRequest) (*client.TriageResponse, error)
	Calls      int
}

func (m *mockTriageClient) Classify(ctx context.Context, req client.TriageRequest) (*client.TriageResponse, error) {
	m.mu.Lock()
	m.Calls++
	m.mu.Unlock()

	if m.ClassifyFn != nil {
		return m.ClassifyFn(ctx, req)
	}
	return &client.TriageResponse{
		ReportID:     req.ReportID,
		Priority:     "GREEN",
		Reason:       "mock default",
	}, nil
}

// mockAmbulanceRepository simula el repositorio de ambulancias.
type mockAmbulanceRepository struct {
	mu                    sync.Mutex
	ambulances            map[string]*domain.Ambulance
	FindAvailableFn       func(ctx context.Context) ([]*domain.Ambulance, error)
	UpdateStatusFn        func(ctx context.Context, ambulanceID string, status domain.AmbulanceStatus) error
	FindByIDFn            func(ctx context.Context, id string) (*domain.Ambulance, error)
}

func (m *mockAmbulanceRepository) FindAvailable(ctx context.Context) ([]*domain.Ambulance, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.FindAvailableFn != nil {
		return m.FindAvailableFn(ctx)
	}

	result := make([]*domain.Ambulance, 0)
	for _, amb := range m.ambulances {
		if amb.IsAvailable() {
			result = append(result, amb)
		}
	}
	return result, nil
}

func (m *mockAmbulanceRepository) UpdateStatus(ctx context.Context, ambulanceID string, status domain.AmbulanceStatus) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.UpdateStatusFn != nil {
		return m.UpdateStatusFn(ctx, ambulanceID, status)
	}

	if amb, exists := m.ambulances[ambulanceID]; exists {
		amb.Status = status
		return nil
	}
	return domain.ErrAmbulanceNotFound
}

func (m *mockAmbulanceRepository) FindByID(ctx context.Context, id string) (*domain.Ambulance, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.FindByIDFn != nil {
		return m.FindByIDFn(ctx, id)
	}

	if amb, exists := m.ambulances[id]; exists {
		copy := *amb
		return &copy, nil
	}
	return nil, domain.ErrAmbulanceNotFound
}

// mockDispatchRepository simula el repositorio de despachos.
type mockDispatchRepository struct {
	mu       sync.Mutex
	SaveFn   func(ctx context.Context, d *domain.Dispatch) error
	FindByIDFn func(ctx context.Context, id string) (*domain.Dispatch, error)
	Saved    []*domain.Dispatch
}

func (m *mockDispatchRepository) Save(ctx context.Context, d *domain.Dispatch) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.SaveFn != nil {
		return m.SaveFn(ctx, d)
	}

	m.Saved = append(m.Saved, d)
	return nil
}

func (m *mockDispatchRepository) FindByID(ctx context.Context, id string) (*domain.Dispatch, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.FindByIDFn != nil {
		return m.FindByIDFn(ctx, id)
	}

	for _, d := range m.Saved {
		if d.ID == id {
			copy := *d
			return &copy, nil
		}
	}
	return nil, domain.ErrDispatchNotFound
}

// helper: crea una ambulancia de prueba.
func makeTestAmbulance(id string, t domain.AmbulanceType, loc domain.Location) *domain.Ambulance {
	return &domain.Ambulance{
		ID:       id,
		Type:     t,
		Status:   domain.AmbulanceStatusAvailable,
		Location: loc,
		CrewSize: 2,
	}
}