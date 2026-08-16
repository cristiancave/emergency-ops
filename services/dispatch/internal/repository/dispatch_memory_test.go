package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	"emergencyops/dispatch/internal/domain"
)

func makeDispatch(id string) *domain.Dispatch {
	return &domain.Dispatch{
		ID:                      id,
		ReportID:                "REP-001",
		AmbulanceID:             "AMB-001",
		Priority:                domain.PriorityRed,
		IncidentLocation:        domain.Location{Latitude: 4.7, Longitude: -74.0},
		AmbulanceLocation:       domain.Location{Latitude: 4.71, Longitude: -74.07},
		EstimatedArrivalMinutes: 8,
		CreatedAt:               time.Now(),
	}
}

func TestMemoryDispatchRepository_SaveAndFind(t *testing.T) {
	ctx := context.Background()
	repo := NewMemoryDispatchRepository()

	d := makeDispatch("DSP-001")

	if err := repo.Save(ctx, d); err != nil {
		t.Fatalf("Save falló: %v", err)
	}

	found, err := repo.FindByID(ctx, "DSP-001")
	if err != nil {
		t.Fatalf("FindByID falló: %v", err)
	}

	if found.AmbulanceID != d.AmbulanceID {
		t.Errorf("AmbulanceID = %q, esperaba %q", found.AmbulanceID, d.AmbulanceID)
	}
}

func TestMemoryDispatchRepository_Save_Duplicate(t *testing.T) {
	ctx := context.Background()
	repo := NewMemoryDispatchRepository()

	d := makeDispatch("DSP-DUP")
	_ = repo.Save(ctx, d)

	err := repo.Save(ctx, d)
	if !errors.Is(err, domain.ErrDuplicateDispatchID) {
		t.Errorf("esperaba ErrDuplicateDispatchID, recibí: %v", err)
	}
}

func TestMemoryDispatchRepository_FindByID_NotFound(t *testing.T) {
	ctx := context.Background()
	repo := NewMemoryDispatchRepository()

	_, err := repo.FindByID(ctx, "DSP-999")
	if !errors.Is(err, domain.ErrDispatchNotFound) {
		t.Errorf("esperaba ErrDispatchNotFound, recibí: %v", err)
	}
}