package main

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/XSAM/otelsql"
	_ "github.com/jackc/pgx/v5/stdlib"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/attribute"

	"emergencyops/pkg/logger"
	"emergencyops/pkg/telemetry"
	"emergencyops/triage/internal/domain"
	"emergencyops/triage/internal/handler"
	"emergencyops/triage/internal/repository"
	"emergencyops/triage/internal/service"
)

const serviceName = "triage-service"
const serviceVersion = "1.0.0"

// config agrupa toda la configuración del servicio.
// Los valores vienen de variables de entorno con defaults sensatos.
type config struct {
	Port            string
	ShutdownTimeout time.Duration
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	IdleTimeout     time.Duration
	DatabaseURL     string // si está vacío, se usa el repositorio en memoria
	Environment     string
	OTLPEndpoint    string
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
		DatabaseURL:     getEnv("DATABASE_URL", ""),
		Environment:     getEnv("ENVIRONMENT", "dev"),
		OTLPEndpoint:    getEnv("OTEL_EXPORTER_OTLP_ENDPOINT", "localhost:4317"),
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
	log.Info("starting", "port", cfg.Port, "environment", cfg.Environment)

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
	// Este es EL único lugar donde se sabe qué implementación
	// concreta se usa: si hay DATABASE_URL, Postgres; si no,
	// memoria (útil para desarrollo local y tests rápidos).

	var repo domain.TriageRepository
	var db *sql.DB

	if cfg.DatabaseURL != "" {
		var err error
		// otelsql.Open envuelve el driver pgx: cada query sale como un span
		// hijo del handler HTTP que la disparó, con el SQL como atributo.
		db, err = otelsql.Open("pgx", cfg.DatabaseURL,
			otelsql.WithAttributes(attribute.String("db.system", "postgresql")),
		)
		if err != nil {
			log.Error("failed to open database", "error", err)
			os.Exit(1)
		}

		if _, err := otelsql.RegisterDBStatsMetrics(db, otelsql.WithAttributes(
			attribute.String("db.system", "postgresql"),
		)); err != nil {
			log.Warn("failed to register DB stats metrics", "error", err)
		}

		pingCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		if err := repository.PingWithRetry(pingCtx, db, 10, 3*time.Second); err != nil {
			cancel()
			log.Error("database not reachable", "error", err)
			os.Exit(1)
		}
		cancel()

		if err := repository.Migrate(context.Background(), db); err != nil {
			log.Error("failed to apply migrations", "error", err)
			os.Exit(1)
		}

		log.Info("using PostgreSQL repository")
		repo = repository.NewPostgresRepository(db)
		defer db.Close()
	} else {
		log.Info("using in-memory repository (DATABASE_URL not set)")
		repo = repository.NewMemoryRepository()
	}

	svc := service.NewTriageService(repo)
	h := handler.NewHTTPHandler(svc, log)

	appMux := http.NewServeMux()
	h.Register(appMux)

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

	// Canal para recibir errores del servidor.
	serverErrors := make(chan error, 1)

	// Arrancamos el servidor en una goroutine para poder
	// escuchar señales del sistema operativo en paralelo.
	go func() {
		log.Info("listening", "addr", "http://localhost:"+cfg.Port)
		serverErrors <- srv.ListenAndServe()
	}()

	// Canal para capturar señales de terminación (Ctrl+C, kill, etc.).
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)

	// Bloqueamos hasta que ocurra un error o una señal.
	select {
	case err := <-serverErrors:
		if !errors.Is(err, http.ErrServerClosed) {
			log.Error("server error", "error", err)
			os.Exit(1)
		}

	case sig := <-shutdown:
		log.Info("shutdown signal received", "signal", sig.String())

		// Contexto con timeout para el shutdown: si no termina en X segundos,
		// forzamos la salida.
		shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
		defer cancel()

		// Shutdown le pide al server que:
		// - No acepte nuevas conexiones
		// - Espere a que las conexiones activas terminen
		// - Retorne cuando todas hayan terminado (o el ctx expire)
		if err := srv.Shutdown(shutdownCtx); err != nil {
			log.Warn("graceful shutdown failed, forcing close", "error", err)
			srv.Close()
		}

		log.Info("stopped cleanly")
	}
}
