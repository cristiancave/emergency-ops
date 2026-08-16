package repository

import (
	"context"
	"errors"
	"testing"

	"emergencyops/dispatch/internal/domain"
)

func TestMemoryAmbulanceRepository_SeedData(t *testing.T) {
	ctx := context.Background()
	repo := NewMemoryAmbulanceRepositoryWithSeed()

	// Debe haber al menos 6 ambulancias (según nuestro seed).
	// Las que están AVAILABLE deben ser 5 (una está BUSY).
	available, err := repo.FindAvailable(ctx)
	if err != nil {
		t.Fatalf("FindAvailable falló: %v", err)
	}

	if len(available) != 5 {
		t.Errorf("esperaba 5 ambulancias disponibles, recibí %d", len(available))
	}
}

func TestMemoryAmbulanceRepository_FindByID(t *testing.T) {
	ctx := context.Background()
	repo := NewMemoryAmbulanceRepositoryWithSeed()

	amb, err := repo.FindByID(ctx, "AMB-001")
	if err != nil {
		t.Fatalf("FindByID falló: %v", err)
	}
	if amb.Type != domain.AmbulanceTypeAdvanced {
		t.Errorf("esperaba tipo ADVANCED, recibí %q", amb.Type)
	}
}

func TestMemoryAmbulanceRepository_FindByID_NotFound(t *testing.T) {
	ctx := context.Background()
	repo := NewMemoryAmbulanceRepositoryWithSeed()

	_, err := repo.FindByID(ctx, "AMB-999")
	if !errors.Is(err, domain.ErrAmbulanceNotFound) {
		t.Errorf("esperaba ErrAmbulanceNotFound, recibí: %v", err)
	}
}

func TestMemoryAmbulanceRepository_UpdateStatus(t *testing.T) {
	ctx := context.Background()
	repo := NewMemoryAmbulanceRepositoryWithSeed()

	// AMB-001 arranca AVAILABLE. La ponemos en BUSY.
	err := repo.UpdateStatus(ctx, "AMB-001", domain.AmbulanceStatusBusy)
	if err != nil {
		t.Fatalf("UpdateStatus falló: %v", err)
	}

	// Verificar que se actualizó.
	amb, _ := repo.FindByID(ctx, "AMB-001")
	if amb.Status != domain.AmbulanceStatusBusy {
		t.Errorf("esperaba status BUSY, recibí %q", amb.Status)
	}

	// Ahora debe haber solo 4 disponibles (originalmente 5, una menos).
	available, _ := repo.FindAvailable(ctx)
	if len(available) != 4 {
		t.Errorf("esperaba 4 disponibles después de update, recibí %d", len(available))
	}
}

func TestMemoryAmbulanceRepository_UpdateStatus_NotFound(t *testing.T) {
	ctx := context.Background()
	repo := NewMemoryAmbulanceRepositoryWithSeed()

	err := repo.UpdateStatus(ctx, "AMB-999", domain.AmbulanceStatusBusy)
	if !errors.Is(err, domain.ErrAmbulanceNotFound) {
		t.Errorf("esperaba ErrAmbulanceNotFound, recibí: %v", err)
	}
}

func TestMemoryAmbulanceRepository_FindAvailable_ReturnsCopies(t *testing.T) {
	ctx := context.Background()
	repo := NewMemoryAmbulanceRepositoryWithSeed()

	// Recuperar ambulancias y mutar la copia.
	available, _ := repo.FindAvailable(ctx)
	if len(available) == 0 {
		t.Fatal("esperaba al menos una ambulancia disponible")
	}

	originalStatus := available[0].Status
	available[0].Status = "MUTATED"

	// Volver a consultar: el estado interno NO debe haberse afectado.
	fresh, _ := repo.FindByID(ctx, available[0].ID)
	if fresh.Status != originalStatus {
		t.Error("FindAvailable retorna referencias mutables al estado interno")
	}
}