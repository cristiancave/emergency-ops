package service

import (
	"context"
	"fmt"
	"sort"
	"time"

	"emergencyops/dispatch/internal/client"
	"emergencyops/dispatch/internal/domain"
)

// TriageClassifier clasifica emergencias llamando a triage-service.
// Satisfecha por *client.TriageClient en producción y por un mock en tests.
type TriageClassifier interface {
	Classify(ctx context.Context, req client.TriageRequest) (*client.TriageResponse, error)
}

// DispatchService orquesta el despacho de ambulancias.
// Depende de:
//  - TriageClassifier: para clasificar emergencias
//  - AmbulanceRepository: para acceder a la flota
//  - DispatchRepository: para persistir despachos
type DispatchService struct {
	triageClient      TriageClassifier
	ambulanceRepo     domain.AmbulanceRepository
	dispatchRepo      domain.DispatchRepository
	now               func() time.Time
	avgSpeedKmh       float64 // velocidad promedio para calcular ETA
}

// NewDispatchService construye el service con todas sus dependencias.
func NewDispatchService(
	triageClient TriageClassifier,
	ambulanceRepo domain.AmbulanceRepository,
	dispatchRepo domain.DispatchRepository,
) *DispatchService {
	return &DispatchService{
		triageClient:  triageClient,
		ambulanceRepo: ambulanceRepo,
		dispatchRepo:  dispatchRepo,
		now:           time.Now,
		avgSpeedKmh:   50.0, // velocidad urbana promedio
	}
}

// CreateDispatch realiza el flujo completo de despacho:
// 1. Clasifica la emergencia en triage-service
// 2. Busca una ambulancia disponible del tipo apropiado
// 3. Calcula ETA
// 4. Marca la ambulancia como ocupada
// 5. Persiste el despacho
// 6. Retorna el despacho
func (s *DispatchService) CreateDispatch(
	ctx context.Context,
	reportID string,
	patientAge int,
	symptoms []string,
	description string,
	incidentLocation domain.Location,
) (*domain.Dispatch, error) {
	// Validar entrada
	if reportID == "" {
		return nil, fmt.Errorf("reportID es obligatorio")
	}
	if err := incidentLocation.Validate(); err != nil {
		return nil, fmt.Errorf("ubicación inválida: %w", err)
	}

	// 1. Llamar a triage para clasificar
	triageReq := client.TriageRequest{
		ReportID:    reportID,
		PatientAge:  patientAge,
		Symptoms:    symptoms,
		Description: description,
	}
	triageResp, err := s.triageClient.Classify(ctx, triageReq)
	if err != nil {
		return nil, fmt.Errorf("failed to classify at triage: %w", err)
	}

	// Convertir la prioridad del formato triage al format dispatch
	// (ambos usan "RED", "YELLOW", "GREEN", así que es trivial)
	priority := domain.Priority(triageResp.Priority)
	if !priority.IsValid() {
		return nil, fmt.Errorf("invalid priority from triage: %q", priority)
	}

	// 2. Buscar una ambulancia disponible
	ambulance, err := s.findBestAmbulance(ctx, priority, incidentLocation)
	if err != nil {
		return nil, err
	}

	// 3. Calcular ETA
	eta := ambulance.Location.EstimatedArrivalMinutes(incidentLocation, s.avgSpeedKmh)

	// 4. Marcar la ambulancia como ocupada
	if err := s.ambulanceRepo.UpdateStatus(ctx, ambulance.ID, domain.AmbulanceStatusBusy); err != nil {
		return nil, fmt.Errorf("failed to mark ambulance as busy: %w", err)
	}

	// 5. Crear y persistir el despacho
	dispatch := &domain.Dispatch{
		ID:                      fmt.Sprintf("DSP-%d", s.now().Unix()),
		ReportID:                reportID,
		AmbulanceID:             ambulance.ID,
		Priority:                priority,
		IncidentLocation:        incidentLocation,
		AmbulanceLocation:       ambulance.Location,
		EstimatedArrivalMinutes: eta,
		CreatedAt:               s.now(),
	}

	if err := dispatch.Validate(); err != nil {
		return nil, fmt.Errorf("invalid dispatch: %w", err)
	}

	if err := s.dispatchRepo.Save(ctx, dispatch); err != nil {
		// Si falla al persistir, intentamos liberar la ambulancia
		// para que no quede marcada como ocupada eternamente.
		_ = s.ambulanceRepo.UpdateStatus(ctx, ambulance.ID, domain.AmbulanceStatusAvailable)
		return nil, fmt.Errorf("failed to save dispatch: %w", err)
	}

	return dispatch, nil
}

// GetDispatch obtiene un despacho previamente creado.
func (s *DispatchService) GetDispatch(ctx context.Context, dispatchID string) (*domain.Dispatch, error) {
	if dispatchID == "" {
		return nil, fmt.Errorf("dispatchID es obligatorio")
	}
	return s.dispatchRepo.FindByID(ctx, dispatchID)
}

// findBestAmbulance busca la ambulancia más cercana que:
// 1. Esté disponible
// 2. Pueda atender la prioridad dada (según su tipo)
func (s *DispatchService) findBestAmbulance(
	ctx context.Context,
	priority domain.Priority,
	incidentLocation domain.Location,
) (*domain.Ambulance, error) {
	// Obtener todas las ambulancias disponibles
	available, err := s.ambulanceRepo.FindAvailable(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch available ambulances: %w", err)
	}

	// Filtrar: solo las que pueden atender esta prioridad
	suitable := make([]*domain.Ambulance, 0)
	for _, amb := range available {
		if amb.Type.CanHandle(priority) {
			suitable = append(suitable, amb)
		}
	}

	if len(suitable) == 0 {
		return nil, domain.ErrNoAvailableAmbulance
	}

	// Calcular distancia a cada ambulancia y retornar la más cercana
	sort.Slice(suitable, func(i, j int) bool {
		distI := suitable[i].Location.DistanceKmTo(incidentLocation)
		distJ := suitable[j].Location.DistanceKmTo(incidentLocation)
		return distI < distJ
	})

	return suitable[0], nil
}