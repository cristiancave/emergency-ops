// Package httpclient provee un *http.Client instrumentado con OTel para
// llamadas HTTP salientes entre servicios (p.ej. dispatch -> triage).
package httpclient

import (
	"net/http"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// New crea un cliente HTTP que abre un span por cada request saliente y
// propaga el trace context (header "traceparent") al servicio destino.
// Es lo que hace que una traza cruce de dispatch-service a triage-service
// como una sola traza, no dos desconectadas.
func New(timeout time.Duration) *http.Client {
	return &http.Client{
		Timeout:   timeout,
		Transport: otelhttp.NewTransport(http.DefaultTransport),
	}
}
