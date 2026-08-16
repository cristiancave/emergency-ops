package handler

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	"emergencyops/dispatch/internal/domain"
)

// DispatchServiceUseCase define la interfaz que el handler necesita del service.
type DispatchServiceUseCase interface {
	CreateDispatch(
		ctx context.Context,
		reportID string,
		patientAge int,
		symptoms []string,
		description string,
		incidentLocation domain.Location,
	) (*domain.Dispatch, error)

	GetDispatch(ctx context.Context, dispatchID string) (*domain.Dispatch, error)
}

// HTTPHandler expone DispatchService por HTTP.
type HTTPHandler struct {
	svc DispatchServiceUseCase
}

// NewHTTPHandler construye el handler con la dependencia inyectada.
func NewHTTPHandler(svc DispatchServiceUseCase) *HTTPHandler {
	return &HTTPHandler{svc: svc}
}

// Register registra las rutas en el mux.
func (h *HTTPHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /dispatch", h.createDispatch)
	mux.HandleFunc("GET /dispatch/{dispatchID}", h.getDispatch)
	mux.HandleFunc("GET /health", h.health)
}

// ==============================================================
// DTOs
// ==============================================================

// createDispatchRequest es el JSON que recibimos en POST /dispatch.
type createDispatchRequest struct {
	ReportID          string   `json:"report_id"`
	PatientAge        int      `json:"patient_age"`
	Symptoms          []string `json:"symptoms"`
	Description       string   `json:"description"`
	IncidentLatitude  float64  `json:"incident_latitude"`
	IncidentLongitude float64  `json:"incident_longitude"`
}

// dispatchResponse es lo que devolvemos.
type dispatchResponse struct {
	DispatchID              string    `json:"dispatch_id"`
	ReportID                string    `json:"report_id"`
	AmbulanceID             string    `json:"ambulance_id"`
	Priority                string    `json:"priority"`
	IncidentLatitude        float64   `json:"incident_latitude"`
	IncidentLongitude       float64   `json:"incident_longitude"`
	AmbulanceLatitude       float64   `json:"ambulance_latitude"`
	AmbulanceLongitude      float64   `json:"ambulance_longitude"`
	EstimatedArrivalMinutes int       `json:"estimated_arrival_minutes"`
	CreatedAt               time.Time `json:"created_at"`
}

// errorResponse estandariza errores HTTP.
type errorResponse struct {
	Error string `json:"error"`
	Code  string `json:"code,omitempty"`
}

// ==============================================================
// Handlers
// ==============================================================

// createDispatch: POST /dispatch
// Recibe un reporte de emergencia, llama al service para clasificar y despachar,
// y retorna el despacho asignado.
func (h *HTTPHandler) createDispatch(w http.ResponseWriter, r *http.Request) {
	// 1. Parsear JSON
	var req createDispatchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error(), "INVALID_JSON")
		return
	}
	defer r.Body.Close()

	// 2. Validar entrada básica
	if req.ReportID == "" {
		writeError(w, http.StatusBadRequest, "report_id es obligatorio", "MISSING_REPORT_ID")
		return
	}

	if req.PatientAge < 0 || req.PatientAge > 130 {
		writeError(w, http.StatusBadRequest, "patient_age debe estar entre 0 y 130", "INVALID_AGE")
		return
	}

	if len(req.Symptoms) == 0 {
		writeError(w, http.StatusBadRequest, "al menos un síntoma es obligatorio", "MISSING_SYMPTOMS")
		return
	}

	// 3. Construir Location y validar
	incidentLocation := domain.Location{
		Latitude:  req.IncidentLatitude,
		Longitude: req.IncidentLongitude,
	}
	if err := incidentLocation.Validate(); err != nil {
		writeError(w, http.StatusBadRequest, "coordenadas inválidas: "+err.Error(), "INVALID_LOCATION")
		return
	}

	// 4. Llamar al service
	dispatch, err := h.svc.CreateDispatch(
		r.Context(),
		req.ReportID,
		req.PatientAge,
		req.Symptoms,
		req.Description,
		incidentLocation,
	)
	if err != nil {
		writeErrorFromDomain(w, err)
		return
	}

	// 5. Mapear respuesta y escribir
	resp := dispatchResponse{
		DispatchID:              dispatch.ID,
		ReportID:                dispatch.ReportID,
		AmbulanceID:             dispatch.AmbulanceID,
		Priority:                string(dispatch.Priority),
		IncidentLatitude:        dispatch.IncidentLocation.Latitude,
		IncidentLongitude:       dispatch.IncidentLocation.Longitude,
		AmbulanceLatitude:       dispatch.AmbulanceLocation.Latitude,
		AmbulanceLongitude:      dispatch.AmbulanceLocation.Longitude,
		EstimatedArrivalMinutes: dispatch.EstimatedArrivalMinutes,
		CreatedAt:               dispatch.CreatedAt,
	}
	writeJSON(w, http.StatusCreated, resp)
}

// getDispatch: GET /dispatch/{dispatchID}
func (h *HTTPHandler) getDispatch(w http.ResponseWriter, r *http.Request) {
	dispatchID := r.PathValue("dispatchID")

	dispatch, err := h.svc.GetDispatch(r.Context(), dispatchID)
	if err != nil {
		writeErrorFromDomain(w, err)
		return
	}

	resp := dispatchResponse{
		DispatchID:              dispatch.ID,
		ReportID:                dispatch.ReportID,
		AmbulanceID:             dispatch.AmbulanceID,
		Priority:                string(dispatch.Priority),
		IncidentLatitude:        dispatch.IncidentLocation.Latitude,
		IncidentLongitude:       dispatch.IncidentLocation.Longitude,
		AmbulanceLatitude:       dispatch.AmbulanceLocation.Latitude,
		AmbulanceLongitude:      dispatch.AmbulanceLocation.Longitude,
		EstimatedArrivalMinutes: dispatch.EstimatedArrivalMinutes,
		CreatedAt:               dispatch.CreatedAt,
	}
	writeJSON(w, http.StatusOK, resp)
}

// health: GET /health
func (h *HTTPHandler) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"status":  "ok",
		"service": "dispatch",
	})
}

// ==============================================================
// Helpers privados
// ==============================================================

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("error encoding JSON response: %v", err)
	}
}

func writeError(w http.ResponseWriter, status int, msg, code string) {
	writeJSON(w, status, errorResponse{Error: msg, Code: code})
}

// writeErrorFromDomain traduce errores del dominio a códigos HTTP.
func writeErrorFromDomain(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrNoAvailableAmbulance):
		writeError(w, http.StatusServiceUnavailable, err.Error(), "NO_AVAILABLE_AMBULANCE")

	case errors.Is(err, domain.ErrDispatchNotFound):
		writeError(w, http.StatusNotFound, err.Error(), "DISPATCH_NOT_FOUND")

	case strings.Contains(err.Error(), "failed to classify at triage"):
		// Error downstream (triage-service falló)
		writeError(w, http.StatusBadGateway, "triage service error", "TRIAGE_ERROR")

	case strings.Contains(err.Error(), "coordenadas inválidas"):
		writeError(w, http.StatusBadRequest, err.Error(), "INVALID_LOCATION")

	case strings.Contains(err.Error(), "es obligatorio") || strings.Contains(err.Error(), "inválid"):
		writeError(w, http.StatusBadRequest, err.Error(), "INVALID_REQUEST")

	default:
		log.Printf("internal error: %v", err)
		writeError(w, http.StatusInternalServerError, "internal server error", "INTERNAL_ERROR")
	}
}