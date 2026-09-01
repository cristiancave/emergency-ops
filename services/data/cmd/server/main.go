package main

import (
	"context"
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

	"emergencyops/data/internal/handler"
	"emergencyops/data/internal/repository"
	"emergencyops/data/internal/service"
	"emergencyops/pkg/logger"
	"emergencyops/pkg/telemetry"
)

const serviceName = "data-service"
const serviceVersion = "1.0.0"

type config struct {
	Port            string
	ShutdownTimeout time.Duration
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	IdleTimeout     time.Duration
	DatabaseURL     string
	Environment     string
	OTLPEndpoint    string
}

func loadConfig() config {
	return config{
		Port:            getEnv("DATA_PORT", "8083"),
		ShutdownTimeout: getEnvDuration("DATA_SHUTDOWN_TIMEOUT", 10*time.Second),
		ReadTimeout:     getEnvDuration("DATA_READ_TIMEOUT", 5*time.Second),
		WriteTimeout:    getEnvDuration("DATA_WRITE_TIMEOUT", 10*time.Second),
		IdleTimeout:     getEnvDuration("DATA_IDLE_TIMEOUT", 60*time.Second),
		DatabaseURL:     getEnv("DATABASE_URL", ""),
		Environment:     getEnv("ENVIRONMENT", "aws-sandbox"),
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
	cfg := loadConfig()
	log := logger.New(serviceName)
	ctx := context.Background()

	log.Info("Initializing telemetry for data-service (AWS RDS)",
		"service", serviceName,
		"version", serviceVersion,
		"env", cfg.Environment,
		"otlp_endpoint", cfg.OTLPEndpoint,
	)

	shutdownOTel, metricsHandler, err := telemetry.Init(ctx, telemetry.Config{
		ServiceName:    serviceName,
		ServiceVersion: serviceVersion,
		Environment:    cfg.Environment,
		OTLPEndpoint:   cfg.OTLPEndpoint,
		OTLPInsecure:   true,
	})
	if err != nil {
		log.Error("Failed to initialize OpenTelemetry", "error", err.Error())
		os.Exit(1)
	}
	defer shutdownOTel(context.Background())

	var repo repository.Repository

	if cfg.DatabaseURL != "" {
		log.Info("Connecting to PostgreSQL (AWS RDS)", "db_url_masked", "postgres://*****")
		db, err := otelsql.Open("pgx", cfg.DatabaseURL,
			otelsql.WithAttributes(
				attribute.String("db.system", "postgresql"),
				attribute.String("db.name", "emergency_db"),
				attribute.String("aws.region", "us-east-1"),
			),
		)
		if err != nil {
			log.Error("Failed to open otelsql connection to AWS RDS", "error", err.Error())
			os.Exit(1)
		}
		defer db.Close()

		if _, err := otelsql.RegisterDBStatsMetrics(db, otelsql.WithAttributes(attribute.String("db.name", "emergency_db"))); err != nil {
			log.Warn("Failed to register DB stats metrics", "error", err.Error())
		}

		db.SetMaxOpenConns(25)
		db.SetMaxIdleConns(5)
		db.SetConnMaxLifetime(5 * time.Minute)

		repo = repository.NewPostgresRepository(db)
	} else {
		log.Warn("DATABASE_URL not set. Falling back to memory repository for data-service")
		repo = repository.NewMemoryRepository()
	}

	dataSvc := service.NewDataService(repo)
	h := handler.NewHTTPHandler(dataSvc)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /data/telemetry", h.ProcessTelemetry)
	mux.HandleFunc("GET /health", h.HealthCheck)
	mux.Handle("GET /metrics", metricsHandler)

	otelHandler := otelhttp.NewHandler(mux, "http.server.data",
		otelhttp.WithMessageEvents(otelhttp.ReadEvents, otelhttp.WriteEvents),
	)

	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      otelHandler,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		IdleTimeout:  cfg.IdleTimeout,
	}

	go func() {
		log.Info("Starting data-service HTTP server", "port", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("HTTP server failed", "error", err.Error())
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("Shutting down data-service gracefully...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error("Forced server shutdown", "error", err.Error())
	}
	log.Info("data-service stopped cleanly")
}
