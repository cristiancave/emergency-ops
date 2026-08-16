package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"emergencyops/dispatch/internal/domain"
)

// mockService satisface DispatchServiceUseCase.
type mockService struct {
	CreateDispatchFn func(ctx context.Context, reportID string, patientAge int, symptoms []string, description string, incidentLocation domain.Location) (*domain.Dispatch, error)
	GetDispatchFn    func(ctx context.Context, dispatchID string) (*domain.Dispatch, error)
}

var _ DispatchServiceUseCase = (*mockService)(nil)

func (m *mockService) CreateDispatch(ctx context.Context, reportID string, patientAge int, symptoms []string, description string, incidentLocation domain.Location) (*domain.Dispatch, error) {
	if m.CreateDispatchFn != nil {
		return m.CreateDispatchFn(ctx, reportID, patientAge, symptoms, description, incidentLocation)
	}
	return nil, errors.New("CreateDispatchFn not defined")
}

func (m *mockService) GetDispatch(ctx context.Context, dispatchID string) (*domain.Dispatch, error) {
	if m.GetDispatchFn != nil {
		return m.GetDispatchFn(ctx, dispatchID)
	}
	return nil, errors.New("GetDispatchFn not defined")
}

func setupHandler(mock *mockService) http.Handler {
	h := NewHTTPHandler(mock)
	mux := http.NewServeMux()
	h.Register(mux)
	return mux
}

// ==============================================================
// Tests: POST /dispatch
// ==============================================================

func TestCreateDispatch_Success(t *testing.T) {
	mock := &mockService{
		CreateDispatchFn: func(ctx context.Context, reportID string, patientAge int, symptoms []string, description string, incidentLocation domain.Location) (*domain.Dispatch, error) {
			return &domain.Dispatch{
				ID:                      "DSP-001",
				ReportID:                reportID,
				AmbulanceID:             "AMB-001",
				Priority:                domain.PriorityRed,
				IncidentLocation:        incidentLocation,
				AmbulanceLocation:       domain.Location{Latitude: 4.71, Longitude: -74.07},
				EstimatedArrivalMinutes: 8,
				CreatedAt:               time.Now(),
			}, nil
		},
	}
	handler := setupHandler(mock)

	body := `{
		"report_id": "REP-001",
		"patient_age": 45,
		"symptoms": ["dolor torácico"],
		"description": "test",
		"incident_latitude": 4.70,
		"incident_longitude": -74.07
	}`

	req := httptest.NewRequest(http.MethodPost, "/dispatch", bytes.NewReader([]byte(body)))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("status = %d, esperaba 201", rec.Code)
	}

	var resp dispatchResponse
	json.NewDecoder(rec.Body).Decode(&resp)

	if resp.DispatchID != "DSP-001" {
		t.Errorf("dispatch_id = %q, esperaba DSP-001", resp.DispatchID)
	}
	if resp.Priority != "RED" {
		t.Errorf("priority = %q, esperaba RED", resp.Priority)
	}
	if resp.EstimatedArrivalMinutes != 8 {
		t.Errorf("eta = %d, esperaba 8", resp.EstimatedArrivalMinutes)
	}
}

func TestCreateDispatch_MissingReportID(t *testing.T) {
	handler := setupHandler(&mockService{})

	body := `{
		"patient_age": 45,
		"symptoms": ["fiebre"],
		"incident_latitude": 4.70,
		"incident_longitude": -74.07
	}`

	req := httptest.NewRequest(http.MethodPost, "/dispatch", bytes.NewReader([]byte(body)))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, esperaba 400", rec.Code)
	}

	var resp errorResponse
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp.Code != "MISSING_REPORT_ID" {
		t.Errorf("code = %q, esperaba MISSING_REPORT_ID", resp.Code)
	}
}

func TestCreateDispatch_InvalidAge(t *testing.T) {
	handler := setupHandler(&mockService{})

	body := `{
		"report_id": "REP-001",
		"patient_age": -5,
		"symptoms": ["fiebre"],
		"incident_latitude": 4.70,
		"incident_longitude": -74.07
	}`

	req := httptest.NewRequest(http.MethodPost, "/dispatch", bytes.NewReader([]byte(body)))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, esperaba 400", rec.Code)
	}
}

func TestCreateDispatch_InvalidCoordinates(t *testing.T) {
	handler := setupHandler(&mockService{})

	body := `{
		"report_id": "REP-001",
		"patient_age": 45,
		"symptoms": ["fiebre"],
		"incident_latitude": 200,
		"incident_longitude": -74.07
	}`

	req := httptest.NewRequest(http.MethodPost, "/dispatch", bytes.NewReader([]byte(body)))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, esperaba 400", rec.Code)
	}

	var resp errorResponse
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp.Code != "INVALID_LOCATION" {
		t.Errorf("code = %q, esperaba INVALID_LOCATION", resp.Code)
	}
}

func TestCreateDispatch_NoAvailableAmbulance(t *testing.T) {
	mock := &mockService{
		CreateDispatchFn: func(ctx context.Context, reportID string, patientAge int, symptoms []string, description string, incidentLocation domain.Location) (*domain.Dispatch, error) {
			return nil, domain.ErrNoAvailableAmbulance
		},
	}
	handler := setupHandler(mock)

	body := `{
		"report_id": "REP-001",
		"patient_age": 45,
		"symptoms": ["dolor torácico"],
		"incident_latitude": 4.70,
		"incident_longitude": -74.07
	}`

	req := httptest.NewRequest(http.MethodPost, "/dispatch", bytes.NewReader([]byte(body)))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, esperaba 503", rec.Code)
	}
}

func TestCreateDispatch_TriageError(t *testing.T) {
	mock := &mockService{
		CreateDispatchFn: func(ctx context.Context, reportID string, patientAge int, symptoms []string, description string, incidentLocation domain.Location) (*domain.Dispatch, error) {
			return nil, errors.New("failed to classify at triage: connection refused")
		},
	}
	handler := setupHandler(mock)

	body := `{
		"report_id": "REP-001",
		"patient_age": 45,
		"symptoms": ["fiebre"],
		"incident_latitude": 4.70,
		"incident_longitude": -74.07
	}`

	req := httptest.NewRequest(http.MethodPost, "/dispatch", bytes.NewReader([]byte(body)))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadGateway {
		t.Errorf("status = %d, esperaba 502", rec.Code)
	}
}

// ==============================================================
// Tests: GET /dispatch/{dispatchID}
// ==============================================================

func TestGetDispatch_Success(t *testing.T) {
	mock := &mockService{
		GetDispatchFn: func(ctx context.Context, dispatchID string) (*domain.Dispatch, error) {
			return &domain.Dispatch{
				ID:                      dispatchID,
				ReportID:                "REP-001",
				AmbulanceID:             "AMB-001",
				Priority:                domain.PriorityYellow,
				IncidentLocation:        domain.Location{Latitude: 4.70, Longitude: -74.07},
				AmbulanceLocation:       domain.Location{Latitude: 4.71, Longitude: -74.07},
				EstimatedArrivalMinutes: 5,
				CreatedAt:               time.Now(),
			}, nil
		},
	}
	handler := setupHandler(mock)

	req := httptest.NewRequest(http.MethodGet, "/dispatch/DSP-001", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, esperaba 200", rec.Code)
	}

	var resp dispatchResponse
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp.DispatchID != "DSP-001" {
		t.Errorf("dispatch_id = %q, esperaba DSP-001", resp.DispatchID)
	}
}

func TestGetDispatch_NotFound(t *testing.T) {
	mock := &mockService{
		GetDispatchFn: func(ctx context.Context, dispatchID string) (*domain.Dispatch, error) {
			return nil, domain.ErrDispatchNotFound
		},
	}
	handler := setupHandler(mock)

	req := httptest.NewRequest(http.MethodGet, "/dispatch/DSP-999", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, esperaba 404", rec.Code)
	}
}

// ==============================================================
// Tests: GET /health
// ==============================================================

func TestHealth(t *testing.T) {
	handler := setupHandler(&mockService{})

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, esperaba 200", rec.Code)
	}

	var resp map[string]string
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp["service"] != "dispatch" {
		t.Errorf("service = %q, esperaba dispatch", resp["service"])
	}
}