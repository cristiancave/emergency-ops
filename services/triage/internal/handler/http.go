package handler

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	"emergencyops/triage/internal/domain"
)

// TriageServiceUseCase define lo que el handler necesita del service.
// Como el handler define esta interfaz aquí, puede recibir cualquier
// implementación que la satisfaga — incluso un mock en tests, sin
// depender del paquete service.
//
// Este es el Interface Segregation Principle: el handler NO conoce
// toda la API del service, solo los métodos que usa.
type TriageServiceUseCase interface {
	ClassifyEmergency(ctx context.Context, report *domain.EmergencyReport) (*domain.TriageResult, error)
	GetResult(ctx context.Context, reportID string) (*domain.TriageResult, error)
}

// HTTPHandler expone el TriageService por HTTP.
type HTTPHandler struct {
	svc TriageServiceUseCase
}

// NewHTTPHandler construye el handler con la dependencia inyectada.
func NewHTTPHandler(svc TriageServiceUseCase) *HTTPHandler {
	return &HTTPHandler{svc: svc}
}

// Register registra las rutas del handler en el mux que se le pase.
// Delegamos el mux desde afuera para no forzar el uso del DefaultServeMux.
func (h *HTTPHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /triage", h.classifyEmergency)
	mux.HandleFunc("GET /triage/{reportID}", h.getResult)
	mux.HandleFunc("GET /health", h.health)
}

// ==============================================================
// DTOs (Data Transfer Objects): estructuras SOLO para HTTP/JSON.
// Están separadas de las entidades del domain para que los cambios
// de contrato HTTP no obliguen a cambiar el domain.
// ==============================================================

// classifyRequest es el JSON que recibimos en POST /triage.
type classifyRequest struct {
	ReportID    string   `json:"report_id"`
	PatientAge  int      `json:"patient_age"`
	Symptoms    []string `json:"symptoms"`
	Description string   `json:"description"`
}

// triageResponse es lo que devolvemos en la respuesta.
type triageResponse struct {
	ReportID     string    `json:"report_id"`
	Priority     string    `json:"priority"`
	Reason       string    `json:"reason"`
	ClassifiedAt time.Time `json:"classified_at"`
}

// errorResponse estandariza los errores devueltos por la API.
type errorResponse struct {
	Error string `json:"error"`
	Code  string `json:"code,omitempty"`
}

// ==============================================================
// Handlers
// ==============================================================

// classifyEmergency: POST /triage
func (h *HTTPHandler) classifyEmergency(w http.ResponseWriter, r *http.Request) {
	// 1. Parsear el JSON de entrada
	var req classifyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error(), "INVALID_JSON")
		return
	}
	defer r.Body.Close()

	// 2. Mapear DTO → entidad del domain
	report := &domain.EmergencyReport{
		ID:          req.ReportID,
		PatientAge:  req.PatientAge,
		Symptoms:    req.Symptoms,
		Description: req.Description,
		ReportedAt:  time.Now(),
	}

	// 3. Llamar al service
	result, err := h.svc.ClassifyEmergency(r.Context(), report)
	if err != nil {
		writeErrorFromDomain(w, err)
		return
	}

	// 4. Mapear entidad → DTO de respuesta y escribir JSON
	resp := triageResponse{
		ReportID:     result.ReportID,
		Priority:     result.Priority.String(),
		Reason:       result.Reason,
		ClassifiedAt: result.ClassifiedAt,
	}
	writeJSON(w, http.StatusCreated, resp)
}

// getResult: GET /triage/{reportID}
func (h *HTTPHandler) getResult(w http.ResponseWriter, r *http.Request) {
	reportID := r.PathValue("reportID")

	result, err := h.svc.GetResult(r.Context(), reportID)
	if err != nil {
		writeErrorFromDomain(w, err)
		return
	}

	resp := triageResponse{
		ReportID:     result.ReportID,
		Priority:     result.Priority.String(),
		Reason:       result.Reason,
		ClassifiedAt: result.ClassifiedAt,
	}
	writeJSON(w, http.StatusOK, resp)
}

// health: GET /health — indica que el servicio está vivo.
func (h *HTTPHandler) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"status":  "ok",
		"service": "triage",
	})
}

// ==============================================================
// Helpers privados del handler
// ==============================================================

// writeJSON serializa v como JSON y lo escribe con el status dado.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		// Ya escribimos el status, poco podemos hacer. Loggeamos y seguimos.
		log.Printf("error encoding JSON response: %v", err)
	}
}

// writeError escribe un error estandarizado con status y código específicos.
func writeError(w http.ResponseWriter, status int, msg, code string) {
	writeJSON(w, status, errorResponse{Error: msg, Code: code})
}

// writeErrorFromDomain traduce errores del dominio a códigos HTTP.
// Esta función encapsula la única traducción domain → HTTP en todo el sistema.
func writeErrorFromDomain(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrReportNotFound):
		writeError(w, http.StatusNotFound, err.Error(), "REPORT_NOT_FOUND")

	case errors.Is(err, domain.ErrDuplicateID):
		writeError(w, http.StatusConflict, err.Error(), "DUPLICATE_ID")

	case strings.Contains(err.Error(), "invalid emergency report"):
		// Los errores construidos con ErrInvalidReport no son centinelas,
		// así que los identificamos por prefijo. En producción podríamos
		// crear un tipo de error específico para hacerlo más robusto.
		writeError(w, http.StatusBadRequest, err.Error(), "INVALID_REPORT")

	case err.Error() == "reportID is required":
		writeError(w, http.StatusBadRequest, err.Error(), "MISSING_ID")

	default:
		// Error inesperado: loggeamos y devolvemos 500 sin filtrar detalles internos.
		log.Printf("internal error: %v", err)
		writeError(w, http.StatusInternalServerError, "internal server error", "INTERNAL_ERROR")
	}
}