package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

	"emergencyops/dispatch/internal/client"
	"emergencyops/dispatch/internal/handler"
	"emergencyops/dispatch/internal/repository"
	"emergencyops/dispatch/internal/service"
	"emergencyops/pkg/logger"
	"emergencyops/pkg/telemetry"
)

const serviceName = "dispatch-service"
const serviceVersion = "1.0.0"

type config struct {
	Port                string
	TriageServiceURL    string
	TriageClientTimeout time.Duration
	ShutdownTimeout     time.Duration
	ReadTimeout         time.Duration
	WriteTimeout        time.Duration
	IdleTimeout         time.Duration
	Environment         string
	OTLPEndpoint        string
}

func loadConfig() config {
	return config{
		Port:                getEnv("DISPATCH_PORT", "8080"),
		TriageServiceURL:    getEnv("TRIAGE_SERVICE_URL", "http://localhost:8081"),
		TriageClientTimeout: getEnvDuration("TRIAGE_CLIENT_TIMEOUT", 5*time.Second),
		ShutdownTimeout:     getEnvDuration("DISPATCH_SHUTDOWN_TIMEOUT", 10*time.Second),
		ReadTimeout:         getEnvDuration("DISPATCH_READ_TIMEOUT", 5*time.Second),
		WriteTimeout:        getEnvDuration("DISPATCH_WRITE_TIMEOUT", 10*time.Second),
		IdleTimeout:         getEnvDuration("DISPATCH_IDLE_TIMEOUT", 60*time.Second),
		Environment:         getEnv("ENVIRONMENT", "dev"),
		OTLPEndpoint:        getEnv("OTEL_EXPORTER_OTLP_ENDPOINT", "localhost:4317"),
	}
}

func getEnv(key, defaultVal string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	return defaultVal
}

func getEnvDuration(key string, defaultVal time.Duration) time.Duration {
	if v, ok := os.LookupEnv(key); ok {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return defaultVal
}

func main() {
	// ==========================================================
	// 1. CONFIGURACIÓN
	// ==========================================================
	cfg := loadConfig()
	log := logger.New(serviceName)
	log.Info("starting", "port", cfg.Port, "triage_url", cfg.TriageServiceURL, "environment", cfg.Environment)

	// ==========================================================
	// 2. TELEMETRÍA: trazas OTLP -> Collector, métricas Prometheus
	// ==========================================================
	ctx := context.Background()
	shutdownTelemetry, metricsHandler, err := telemetry.Init(ctx, telemetry.Config{
		ServiceName:    serviceName,
		ServiceVersion: serviceVersion,
		Environment:    cfg.Environment,
		OTLPEndpoint:   cfg.OTLPEndpoint,
		OTLPInsecure:   true,
	})
	if err != nil {
		log.Error("failed to initialize telemetry", "error", err)
		os.Exit(1)
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := shutdownTelemetry(shutdownCtx); err != nil {
			log.Error("telemetry shutdown failed", "error", err)
		}
	}()

	// ==========================================================
	// 3. WIRING (composition root)
	// ==========================================================

	// Cliente HTTP a triage-service (instrumentado: propaga trace context)
	triageClient := client.NewTriageClient(cfg.TriageServiceURL, cfg.TriageClientTimeout)

	// Repositorios: ambulancias con seed, despachos vacío
	ambulanceRepo := repository.NewMemoryAmbulanceRepositoryWithSeed()
	dispatchRepo := repository.NewMemoryDispatchRepository()

	// Service que orquesta todo
	dispatchSvc := service.NewDispatchService(triageClient, ambulanceRepo, dispatchRepo)

	// Handler HTTP
	httpHandler := handler.NewHTTPHandler(dispatchSvc, log)

	// Mux con las rutas de negocio
	appMux := http.NewServeMux()
	httpHandler.Register(appMux)

	// Mux raíz: /metrics sin instrumentar (evita spans del scrape de Prometheus),
	// el resto envuelto con otelhttp para spans de servidor + propagación de contexto.
	rootMux := http.NewServeMux()
	rootMux.Handle("/metrics", metricsHandler)
	rootMux.Handle("/", otelhttp.NewHandler(appMux, serviceName))

	// ==========================================================
	// 4. RUNTIME: servidor HTTP con timeouts y graceful shutdown
	// ==========================================================
	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      rootMux,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		IdleTimeout:  cfg.IdleTimeout,
	}

	serverErrors := make(chan error, 1)

	// Arrancar el servidor en una goroutine
	go func() {
		log.Info("listening", "addr", "http://localhost:"+cfg.Port)
		serverErrors <- srv.ListenAndServe()
	}()

	// Escuchar señales de shutdown
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)

	// Bloqueamos hasta error o shutdown
	select {
	case err := <-serverErrors:
		if !errors.Is(err, http.ErrServerClosed) {
			log.Error("server error", "error", err)
			os.Exit(1)
		}

	case sig := <-shutdown:
		log.Info("shutdown signal received", "signal", sig.String())

		shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
		defer cancel()

		if err := srv.Shutdown(shutdownCtx); err != nil {
			log.Warn("graceful shutdown failed, forcing close", "error", err)
			srv.Close()
		}

		log.Info("stopped cleanly")
	}
}
