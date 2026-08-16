package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"emergencyops/dispatch/internal/client"
	"emergencyops/dispatch/internal/handler"
	"emergencyops/dispatch/internal/repository"
	"emergencyops/dispatch/internal/service"
)

type config struct {
	Port               string
	TriageServiceURL   string
	TriageClientTimeout time.Duration
	ShutdownTimeout    time.Duration
	ReadTimeout        time.Duration
	WriteTimeout       time.Duration
	IdleTimeout        time.Duration
}

func loadConfig() config {
	return config{
		Port:               getEnv("DISPATCH_PORT", "8080"),
		TriageServiceURL:   getEnv("TRIAGE_SERVICE_URL", "http://localhost:8081"),
		TriageClientTimeout: getEnvDuration("TRIAGE_CLIENT_TIMEOUT", 5*time.Second),
		ShutdownTimeout:    getEnvDuration("DISPATCH_SHUTDOWN_TIMEOUT", 10*time.Second),
		ReadTimeout:        getEnvDuration("DISPATCH_READ_TIMEOUT", 5*time.Second),
		WriteTimeout:       getEnvDuration("DISPATCH_WRITE_TIMEOUT", 10*time.Second),
		IdleTimeout:        getEnvDuration("DISPATCH_IDLE_TIMEOUT", 60*time.Second),
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
		log.Printf("warning: %s=%q no es una duración válida, usando default %v", key, v, defaultVal)
	}
	return defaultVal
}

func main() {
	// ==========================================================
	// 1. CONFIGURACIÓN
	// ==========================================================
	cfg := loadConfig()
	log.Printf("dispatch-service starting with config: port=%s, triage_url=%s", cfg.Port, cfg.TriageServiceURL)

	// ==========================================================
	// 2. WIRING (composition root)
	// ==========================================================

	// Cliente HTTP a triage-service
	triageClient := client.NewTriageClient(cfg.TriageServiceURL, cfg.TriageClientTimeout)

	// Repositorios: ambulancias con seed, despachos vacío
	ambulanceRepo := repository.NewMemoryAmbulanceRepositoryWithSeed()
	dispatchRepo := repository.NewMemoryDispatchRepository()

	// Service que orquesta todo
	dispatchSvc := service.NewDispatchService(triageClient, ambulanceRepo, dispatchRepo)

	// Handler HTTP
	httpHandler := handler.NewHTTPHandler(dispatchSvc)

	// Mux con las rutas
	mux := http.NewServeMux()
	httpHandler.Register(mux)

	// ==========================================================
	// 3. RUNTIME: servidor HTTP con timeouts y graceful shutdown
	// ==========================================================
	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      mux,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		IdleTimeout:  cfg.IdleTimeout,
	}

	serverErrors := make(chan error, 1)

	// Arrancar el servidor en una goroutine
	go func() {
		log.Printf("dispatch-service listening on http://localhost:%s", cfg.Port)
		log.Printf("dispatch-service -> triage-service at %s", cfg.TriageServiceURL)
		serverErrors <- srv.ListenAndServe()
	}()

	// Escuchar señales de shutdown
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)

	// Bloqueamos hasta error o shutdown
	select {
	case err := <-serverErrors:
		if !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server error: %v", err)
		}

	case sig := <-shutdown:
		log.Printf("shutdown signal received: %v", sig)

		ctx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
		defer cancel()

		if err := srv.Shutdown(ctx); err != nil {
			log.Printf("graceful shutdown failed: %v, forcing close", err)
			srv.Close()
		}

		log.Println("dispatch-service stopped cleanly")
	}
}