package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// TriageResponse es lo que devuelve triage-service en POST /triage.
type TriageResponse struct {
	ReportID     string    `json:"report_id"`
	Priority     string    `json:"priority"`
	Reason       string    `json:"reason"`
	ClassifiedAt time.Time `json:"classified_at"`
}

// TriageRequest es lo que mandamos a triage-service.
type TriageRequest struct {
	ReportID    string   `json:"report_id"`
	PatientAge  int      `json:"patient_age"`
	Symptoms    []string `json:"symptoms"`
	Description string   `json:"description"`
}

// TriageClient realiza llamadas HTTP a triage-service.
// Encapsula timeouts, manejo de errores, y propagación de contexto.
type TriageClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewTriageClient construye un cliente con timeout explícito.
// baseURL debe ser "http://localhost:8081" (sin trailing slash).
func NewTriageClient(baseURL string, timeout time.Duration) *TriageClient {
	return &TriageClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

// Classify envía un reporte a triage-service y obtiene la clasificación.
//
// Propaga ctx: si se cancela o excede su deadline, la petición se aborta.
// Esto es crítico en microservicios: si triage-service cae o es lento,
// no queremos colgar indefinidamente. El contexto cancela la operación.
//
// Retorna un error si:
//  - ctx se cancela/timeout
//  - Error de red
//  - triage-service retorna 4xx o 5xx
func (c *TriageClient) Classify(ctx context.Context, req TriageRequest) (*TriageResponse, error) {
	// 1. Serializar el request a JSON en un buffer.
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	// 2. Construir la request HTTP con contexto propagado.
	httpReq, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.baseURL+"/triage",
		bytes.NewReader(body),
	)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	// 3. Setear headers.
	httpReq.Header.Set("Content-Type", "application/json")

	// 4. Ejecutar la request.
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		// Si el error es por contexto (timeout, cancelación),
		// queremos que sea obvio.
		if err == context.Canceled {
			return nil, fmt.Errorf("request cancelled: %w", err)
		}
		if err == context.DeadlineExceeded {
			return nil, fmt.Errorf("request timeout: %w", err)
		}
		return nil, fmt.Errorf("http request failed: %w", err)
	}
	defer resp.Body.Close()

	// 5. Leer el body de la respuesta.
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	// 6. Verificar status code.
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("triage service returned %d: %s", resp.StatusCode, string(respBody))
	}

	// 7. Deserializar la respuesta.
	var triageResp TriageResponse
	if err := json.Unmarshal(respBody, &triageResp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	return &triageResp, nil
}