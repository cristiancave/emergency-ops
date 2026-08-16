package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// TestTriageClient_Success: triage-service responde correctamente.
func TestTriageClient_Success(t *testing.T) {
	// Crear un servidor HTTP fake que simula triage-service.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/triage" {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Devolver una respuesta válida.
		resp := TriageResponse{
			ReportID: "REP-001",
			Priority: "RED",
			Reason:   "síntoma crítico",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// Crear el cliente apuntando al servidor fake.
	client := NewTriageClient(server.URL, 5*time.Second)
	ctx := context.Background()

	req := TriageRequest{
		ReportID:   "REP-001",
		PatientAge: 45,
		Symptoms:   []string{"dolor torácico"},
	}

	resp, err := client.Classify(ctx, req)
	if err != nil {
		t.Fatalf("Classify falló: %v", err)
	}

	if resp.Priority != "RED" {
		t.Errorf("priority = %q, esperaba RED", resp.Priority)
	}
}

// TestTriageClient_NetworkError: simular que triage-service no está disponible.
func TestTriageClient_NetworkError(t *testing.T) {
	// Apuntar a un puerto que no existe.
	client := NewTriageClient("http://localhost:9999", 1*time.Second)
	ctx := context.Background()

	req := TriageRequest{
		ReportID:   "REP-001",
		PatientAge: 30,
		Symptoms:   []string{"fiebre"},
	}

	_, err := client.Classify(ctx, req)
	if err == nil {
		t.Fatal("esperaba error, no lo recibí")
	}

	// Verificar que el mensaje menciona que falló.
	if err.Error() == "" {
		t.Error("error vacío")
	}
}

// TestTriageClient_TimeoutContext: cuando el contexto expira, la petición se cancela.
func TestTriageClient_TimeoutContext(t *testing.T) {
	// Servidor que deliberadamente demora mucho.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second) // espera 2 segundos
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewTriageClient(server.URL, 5*time.Second)

	// Contexto que expira en 100ms.
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	req := TriageRequest{ReportID: "REP-001"}

	_, err := client.Classify(ctx, req)
	if err == nil {
		t.Fatal("esperaba timeout error")
	}

	// El error debe ser por deadline excedido (el contexto expiró).
	if err.Error() == "" {
		t.Error("error vacío")
	}
}

// TestTriageClient_BadResponse: triage-service retorna status 4xx/5xx.
func TestTriageClient_BadResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "invalid report", http.StatusBadRequest)
	}))
	defer server.Close()

	client := NewTriageClient(server.URL, 5*time.Second)
	ctx := context.Background()

	req := TriageRequest{ReportID: "REP-001"}

	_, err := client.Classify(ctx, req)
	if err == nil {
		t.Fatal("esperaba error para status 400")
	}

	// El error debe mencionar el status code.
	if err.Error() == "" {
		t.Error("error vacío")
	}
}

// TestTriageClient_InvalidJSON: el servidor devuelve JSON malformado.
func TestTriageClient_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("not valid json {"))
	}))
	defer server.Close()

	client := NewTriageClient(server.URL, 5*time.Second)
	ctx := context.Background()

	req := TriageRequest{ReportID: "REP-001"}

	_, err := client.Classify(ctx, req)
	if err == nil {
		t.Fatal("esperaba error de JSON inválido")
	}
}

// TestTriageClient_ContextCancellation: cancelar el contexto aborta la petición.
func TestTriageClient_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Servidor lento.
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewTriageClient(server.URL, 10*time.Second) // timeout largo para no interferir

	// Contexto que cancelaremos manualmente.
	ctx, cancel := context.WithCancel(context.Background())

	// Cancelar después de 100ms.
	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()

	req := TriageRequest{ReportID: "REP-001"}

	_, err := client.Classify(ctx, req)
	if err == nil {
		t.Fatal("esperaba error de cancelación")
	}
}