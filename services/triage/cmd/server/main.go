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

	"emergencyops/triage/internal/handler"
	"emergencyops/triage/internal/repository"
	"emergencyops/triage/internal/service"
)

// config agrupa toda la configuración del servicio.
// Los valores vienen de variables de entorno con defaults sensatos.
type config struct {
	Port            string
	ShutdownTimeout time.Duration
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	IdleTimeout     time.Duration
}

// loadConfig lee la configuración del entorno.
// Cada variable tiene un default para poder correr localmente sin setup.
func loadConfig() config {
	return config{
		Port:            getEnv("TRIAGE_PORT", "8081"),
		ShutdownTimeout: getEnvDuration("TRIAGE_SHUTDOWN_TIMEOUT", 10*time.Second),
		ReadTimeout:     getEnvDuration("TRIAGE_READ_TIMEOUT", 5*time.Second),
		WriteTimeout:    getEnvDuration("TRIAGE_WRITE_TIMEOUT", 10*time.Second),
		IdleTimeout:     getEnvDuration("TRIAGE_IDLE_TIMEOUT", 60*time.Second),
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
	log.Printf("triage-service starting with config: port=%s", cfg.Port)

	// ==========================================================
	// 2. WIRING (composition root)
	// ==========================================================
	// Este es EL único lugar donde se sabe qué implementación
	// concreta se usa. Cambiar aquí de MemoryRepository a
	// PostgresRepository sería una línea.

	repo := repository.NewMemoryRepository()
	svc := service.NewTriageService(repo)
	h := handler.NewHTTPHandler(svc)

	mux := http.NewServeMux()
	h.Register(mux)

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

	// Canal para recibir errores del servidor.
	serverErrors := make(chan error, 1)

	// Arrancamos el servidor en una goroutine para poder
	// escuchar señales del sistema operativo en paralelo.
	go func() {
		log.Printf("triage-service listening on http://localhost:%s", cfg.Port)
		serverErrors <- srv.ListenAndServe()
	}()

	// Canal para capturar señales de terminación (Ctrl+C, kill, etc.).
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)

	// Bloqueamos hasta que ocurra un error o una señal.
	select {
	case err := <-serverErrors:
		if !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server error: %v", err)
		}

	case sig := <-shutdown:
		log.Printf("shutdown signal received: %v", sig)

		// Contexto con timeout para el shutdown: si no termina en X segundos,
		// forzamos la salida.
		ctx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
		defer cancel()

		// Shutdown le pide al server que:
		// - No acepte nuevas conexiones
		// - Espere a que las conexiones activas terminen
		// - Retorne cuando todas hayan terminado (o el ctx expire)
		if err := srv.Shutdown(ctx); err != nil {
			log.Printf("graceful shutdown failed: %v, forcing close", err)
			srv.Close()
		}

		log.Println("triage-service stopped cleanly")
	}
}