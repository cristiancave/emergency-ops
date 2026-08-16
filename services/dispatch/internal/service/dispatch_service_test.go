package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"emergencyops/dispatch/internal/client"
	"emergencyops/dispatch/internal/domain"
)

func TestDispatchService_CreateDispatch_Success(t *testing.T) {
	ctx := context.Background()

	// Setup mocks
	triageClient := &mockTriageClient{
		ClassifyFn: func(ctx context.Context, req client.TriageRequest) (*client.TriageResponse, error) {
			return &client.TriageResponse{
				ReportID:     req.ReportID,
				Priority:     "RED",
				Reason:       "dolor torácico",
				ClassifiedAt: time.Now(),
			}, nil
		},
	}

	ambulanceRepo := &mockAmbulanceRepository{
		ambulances: map[string]*domain.Ambulance{
			"AMB-001": makeTestAmbulance("AMB-001", domain.AmbulanceTypeAdvanced,
				domain.Location{Latitude: 4.71, Longitude: -74.07}),
		},
	}

	dispatchRepo := &mockDispatchRepository{}

	svc := NewDispatchService(triageClient, ambulanceRepo, dispatchRepo)
	svc.now = func() time.Time { return time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC) }

	// Ejecutar
	dispatch, err := svc.CreateDispatch(
		ctx,
		"REP-001",
		45,
		[]string{"dolor torácico"},
		"test",
		domain.Location{Latitude: 4.70, Longitude: -74.07},
	)

	if err != nil {
		t.Fatalf("CreateDispatch falló: %v", err)
	}

	// Verificaciones
	if dispatch.ReportID != "REP-001" {
		t.Errorf("ReportID = %q, esperaba REP-001", dispatch.ReportID)
	}
	if dispatch.Priority != domain.PriorityRed {
		t.Errorf("Priority = %q, esperaba RED", dispatch.Priority)
	}
	if dispatch.AmbulanceID != "AMB-001" {
		t.Errorf("AmbulanceID = %q, esperaba AMB-001", dispatch.AmbulanceID)
	}

	// Verificar que la ambulancia fue marcada como ocupada
	amb, _ := ambulanceRepo.FindByID(ctx, "AMB-001")
	if amb.Status != domain.AmbulanceStatusBusy {
		t.Errorf("ambulancia no fue marcada como ocupada")
	}

	// Verificar que el despacho fue persistido
	if len(dispatchRepo.Saved) != 1 {
		t.Errorf("esperaba 1 despacho guardado, recibí %d", len(dispatchRepo.Saved))
	}
}

func TestDispatchService_CreateDispatch_TriageServiceDown(t *testing.T) {
	ctx := context.Background()

	triageClient := &mockTriageClient{
		ClassifyFn: func(ctx context.Context, req client.TriageRequest) (*client.TriageResponse, error) {
			return nil, errors.New("triage-service is down")
		},
	}

	ambulanceRepo := &mockAmbulanceRepository{ambulances: map[string]*domain.Ambulance{}}
	dispatchRepo := &mockDispatchRepository{}

	svc := NewDispatchService(triageClient, ambulanceRepo, dispatchRepo)

	_, err := svc.CreateDispatch(ctx, "REP-001", 30, []string{"fiebre"}, "test",
		domain.Location{Latitude: 4.7, Longitude: -74.0})

	if err == nil {
		t.Fatal("esperaba error cuando triage falla")
	}
}

func TestDispatchService_CreateDispatch_NoAvailableAmbulance(t *testing.T) {
	ctx := context.Background()

	triageClient := &mockTriageClient{
		ClassifyFn: func(ctx context.Context, req client.TriageRequest) (*client.TriageResponse, error) {
			return &client.TriageResponse{
				ReportID:     req.ReportID,
				Priority:     "RED",
				Reason:       "crítico",
				ClassifiedAt: time.Now(),
			}, nil
		},
	}

	// No hay ambulancias ADVANCED disponibles (RED requiere ADVANCED)
	ambulanceRepo := &mockAmbulanceRepository{
		ambulances: map[string]*domain.Ambulance{
			"AMB-001": makeTestAmbulance("AMB-001", domain.AmbulanceTypeBasic,
				domain.Location{Latitude: 4.71, Longitude: -74.07}),
		},
	}

	dispatchRepo := &mockDispatchRepository{}
	svc := NewDispatchService(triageClient, ambulanceRepo, dispatchRepo)

	_, err := svc.CreateDispatch(ctx, "REP-001", 30, []string{"dolor torácico"}, "test",
		domain.Location{Latitude: 4.7, Longitude: -74.0})

	if !errors.Is(err, domain.ErrNoAvailableAmbulance) {
		t.Errorf("esperaba ErrNoAvailableAmbulance, recibí: %v", err)
	}
}

func TestDispatchService_CreateDispatch_ChooseClosestAmbulance(t *testing.T) {
	ctx := context.Background()

	triageClient := &mockTriageClient{
		ClassifyFn: func(ctx context.Context, req client.TriageRequest) (*client.TriageResponse, error) {
			return &client.TriageResponse{
				ReportID: req.ReportID,
				Priority: "YELLOW",
			}, nil
		},
	}

	// Dos ambulancias, una más cercana que la otra
	ambulanceRepo := &mockAmbulanceRepository{
		ambulances: map[string]*domain.Ambulance{
			"AMB-001": makeTestAmbulance("AMB-001", domain.AmbulanceTypeBasic,
				domain.Location{Latitude: 4.71, Longitude: -74.07}), // cercana
			"AMB-002": makeTestAmbulance("AMB-002", domain.AmbulanceTypeBasic,
				domain.Location{Latitude: 4.50, Longitude: -74.00}), // lejana
		},
	}

	dispatchRepo := &mockDispatchRepository{}
	svc := NewDispatchService(triageClient, ambulanceRepo, dispatchRepo)

	dispatch, _ := svc.CreateDispatch(ctx, "REP-001", 30, []string{"fiebre"}, "test",
		domain.Location{Latitude: 4.70, Longitude: -74.07})

	if dispatch.AmbulanceID != "AMB-001" {
		t.Errorf("esperaba ambulancia cercana AMB-001, recibí %q", dispatch.AmbulanceID)
	}
}

func TestDispatchService_CreateDispatch_RevertAmbulanceOnSaveFailure(t *testing.T) {
	ctx := context.Background()

	triageClient := &mockTriageClient{
		ClassifyFn: func(ctx context.Context, req client.TriageRequest) (*client.TriageResponse, error) {
			return &client.TriageResponse{
				ReportID: req.ReportID,
				Priority: "YELLOW",
			}, nil
		},
	}

	ambulanceRepo := &mockAmbulanceRepository{
		ambulances: map[string]*domain.Ambulance{
			"AMB-001": makeTestAmbulance("AMB-001", domain.AmbulanceTypeBasic,
				domain.Location{Latitude: 4.71, Longitude: -74.07}),
		},
	}

	// Dispatch repo que falla al guardar
	dispatchRepo := &mockDispatchRepository{
		SaveFn: func(ctx context.Context, d *domain.Dispatch) error {
			return errors.New("database error")
		},
	}

	svc := NewDispatchService(triageClient, ambulanceRepo, dispatchRepo)

	_, err := svc.CreateDispatch(ctx, "REP-001", 30, []string{"fiebre"}, "test",
		domain.Location{Latitude: 4.70, Longitude: -74.07})

	if err == nil {
		t.Fatal("esperaba error al fallar dispatch save")
	}

	// Verificar que la ambulancia volvió a AVAILABLE (revertida)
	amb, _ := ambulanceRepo.FindByID(ctx, "AMB-001")
	if amb.Status != domain.AmbulanceStatusAvailable {
		t.Errorf("ambulancia no fue revertida a AVAILABLE en caso de error")
	}
}

func TestDispatchService_GetDispatch(t *testing.T) {
	ctx := context.Background()

	dispatchRepo := &mockDispatchRepository{
		Saved: []*domain.Dispatch{
			{
				ID:          "DSP-001",
				ReportID:    "REP-001",
				AmbulanceID: "AMB-001",
				Priority:    domain.PriorityYellow,
			},
		},
	}

	svc := NewDispatchService(&mockTriageClient{}, &mockAmbulanceRepository{}, dispatchRepo)

	dispatch, err := svc.GetDispatch(ctx, "DSP-001")
	if err != nil {
		t.Fatalf("GetDispatch falló: %v", err)
	}

	if dispatch.AmbulanceID != "AMB-001" {
		t.Errorf("AmbulanceID = %q, esperaba AMB-001", dispatch.AmbulanceID)
	}
}

func TestDispatchService_GetDispatch_NotFound(t *testing.T) {
	ctx := context.Background()
	dispatchRepo := &mockDispatchRepository{}
	svc := NewDispatchService(&mockTriageClient{}, &mockAmbulanceRepository{}, dispatchRepo)

	_, err := svc.GetDispatch(ctx, "DSP-999")
	if !errors.Is(err, domain.ErrDispatchNotFound) {
		t.Errorf("esperaba ErrDispatchNotFound, recibí: %v", err)
	}
}