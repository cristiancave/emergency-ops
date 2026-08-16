package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"emergencyops/triage/internal/domain"
)

// testLogger descarta la salida: en tests solo importa el comportamiento HTTP.
func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// mockService satisface TriageServiceUseCase sin necesidad de importar
// el paquete service. Este mock vive SOLO en tests del handler.
type mockService struct {
	ClassifyFn  func(ctx context.Context, report *domain.EmergencyReport) (*domain.TriageResult, error)
	GetResultFn func(ctx context.Context, reportID string) (*domain.TriageResult, error)
}

// Verificación en tiempo de compilación.
var _ TriageServiceUseCase = (*mockService)(nil)

func (m *mockService) ClassifyEmergency(ctx context.Context, report *domain.EmergencyReport) (*domain.TriageResult, error) {
	if m.ClassifyFn != nil {
		return m.ClassifyFn(ctx, report)
	}
	return nil, errors.New("ClassifyFn not defined in mock")
}

func (m *mockService) GetResult(ctx context.Context, reportID string) (*domain.TriageResult, error) {
	if m.GetResultFn != nil {
		return m.GetResultFn(ctx, reportID)
	}
	return nil, errors.New("GetResultFn not defined in mock")
}

// setupHandler crea un handler con el mock inyectado y un mux listo para usar.
func setupHandler(mock *mockService) http.Handler {
	h := NewHTTPHandler(mock, testLogger())
	mux := http.NewServeMux()
	h.Register(mux)
	return mux
}

// ==============================================================
// Tests: POST /triage
// ==============================================================

func TestClassifyEmergency_Success(t *testing.T) {
	mock := &mockService{
		ClassifyFn: func(ctx context.Context, report *domain.EmergencyReport) (*domain.TriageResult, error) {
			return &domain.TriageResult{
				ReportID:     report.ID,
				Priority:     domain.PriorityRed,
				Reason:       "síntoma crítico detectado",
				ClassifiedAt: time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC),
			}, nil
		},
	}
	handler := setupHandler(mock)

	body := `{
		"report_id": "REP-001",
		"patient_age": 45,
		"symptoms": ["dolor torácico"],
		"description": "test"
	}`

	req := httptest.NewRequest(http.MethodPost, "/triage", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	// Verificar status
	if rec.Code != http.StatusCreated {
		t.Errorf("status = %d, esperaba %d", rec.Code, http.StatusCreated)
	}

	// Verificar Content-Type
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, esperaba application/json", ct)
	}

	// Verificar el body de la respuesta
	var resp triageResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("no se pudo decodificar respuesta: %v", err)
	}

	if resp.Priority != "RED" {
		t.Errorf("priority = %q, esperaba RED", resp.Priority)
	}
	if resp.ReportID != "REP-001" {
		t.Errorf("report_id = %q, esperaba REP-001", resp.ReportID)
	}
}

func TestClassifyEmergency_InvalidJSON(t *testing.T) {
	handler := setupHandler(&mockService{})

	// JSON malformado (falta cerrar llave)
	body := `{"report_id": "REP-001"`

	req := httptest.NewRequest(http.MethodPost, "/triage", strings.NewReader(body))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, esperaba 400", rec.Code)
	}

	var resp errorResponse
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp.Code != "INVALID_JSON" {
		t.Errorf("code = %q, esperaba INVALID_JSON", resp.Code)
	}
}

func TestClassifyEmergency_DuplicateID(t *testing.T) {
	mock := &mockService{
		ClassifyFn: func(ctx context.Context, report *domain.EmergencyReport) (*domain.TriageResult, error) {
			return nil, domain.ErrDuplicateID
		},
	}
	handler := setupHandler(mock)

	body := `{"report_id":"REP-001","patient_age":30,"symptoms":["fiebre"]}`
	req := httptest.NewRequest(http.MethodPost, "/triage", strings.NewReader(body))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusConflict {
		t.Errorf("status = %d, esperaba 409 Conflict", rec.Code)
	}
}

func TestClassifyEmergency_ValidationError(t *testing.T) {
	mock := &mockService{
		ClassifyFn: func(ctx context.Context, report *domain.EmergencyReport) (*domain.TriageResult, error) {
			return nil, domain.ErrInvalidReport("edad debe estar entre 0 y 130")
		},
	}
	handler := setupHandler(mock)

	body := `{"report_id":"REP-001","patient_age":-5,"symptoms":["fiebre"]}`
	req := httptest.NewRequest(http.MethodPost, "/triage", strings.NewReader(body))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, esperaba 400", rec.Code)
	}

	var resp errorResponse
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp.Code != "INVALID_REPORT" {
		t.Errorf("code = %q, esperaba INVALID_REPORT", resp.Code)
	}
}

func TestClassifyEmergency_InternalError(t *testing.T) {
	mock := &mockService{
		ClassifyFn: func(ctx context.Context, report *domain.EmergencyReport) (*domain.TriageResult, error) {
			return nil, errors.New("some unexpected internal error")
		},
	}
	handler := setupHandler(mock)

	body := `{"report_id":"REP-001","patient_age":30,"symptoms":["fiebre"]}`
	req := httptest.NewRequest(http.MethodPost, "/triage", strings.NewReader(body))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, esperaba 500", rec.Code)
	}

	// Verificar que el mensaje al cliente NO expone detalles internos.
	var resp errorResponse
	json.NewDecoder(rec.Body).Decode(&resp)
	if strings.Contains(resp.Error, "unexpected internal error") {
		t.Errorf("mensaje interno filtrado al cliente: %q", resp.Error)
	}
}

// ==============================================================
// Tests: GET /triage/{reportID}
// ==============================================================

func TestGetResult_Success(t *testing.T) {
	mock := &mockService{
		GetResultFn: func(ctx context.Context, reportID string) (*domain.TriageResult, error) {
			if reportID != "REP-042" {
				t.Errorf("reportID recibido = %q, esperaba REP-042", reportID)
			}
			return &domain.TriageResult{
				ReportID:     reportID,
				Priority:     domain.PriorityYellow,
				Reason:       "fractura visible",
				ClassifiedAt: time.Now(),
			}, nil
		},
	}
	handler := setupHandler(mock)

	req := httptest.NewRequest(http.MethodGet, "/triage/REP-042", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, esperaba 200", rec.Code)
	}

	var resp triageResponse
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp.Priority != "YELLOW" {
		t.Errorf("priority = %q, esperaba YELLOW", resp.Priority)
	}
}

func TestGetResult_NotFound(t *testing.T) {
	mock := &mockService{
		GetResultFn: func(ctx context.Context, reportID string) (*domain.TriageResult, error) {
			return nil, domain.ErrReportNotFound
		},
	}
	handler := setupHandler(mock)

	req := httptest.NewRequest(http.MethodGet, "/triage/REP-NO-EXISTE", nil)
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

	if resp["status"] != "ok" {
		t.Errorf("status = %q, esperaba ok", resp["status"])
	}
	if resp["service"] != "triage" {
		t.Errorf("service = %q, esperaba triage", resp["service"])
	}
}

// ==============================================================
// Test de integración: verificar que el ContentType se maneja bien
// aunque el cliente mande body binario.
// ==============================================================

func TestClassifyEmergency_EmptyBody(t *testing.T) {
	handler := setupHandler(&mockService{})

	req := httptest.NewRequest(http.MethodPost, "/triage", bytes.NewReader(nil))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("body vacío debería devolver 400, recibí %d", rec.Code)
	}
}