package repository

import (
	"context"
	"database/sql"
	"fmt"
)

// schema define las tablas de triage-service. Se aplica con CREATE TABLE IF NOT EXISTS,
// así que correrla en cada arranque del servicio es seguro (idempotente). Para un
// proyecto de este tamaño no se justifica una herramienta de migraciones aparte.
const schema = `
CREATE TABLE IF NOT EXISTS emergency_reports (
    id           TEXT PRIMARY KEY,
    patient_age  INTEGER NOT NULL,
    symptoms     TEXT NOT NULL,
    description  TEXT NOT NULL,
    reported_at  TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS triage_results (
    report_id     TEXT PRIMARY KEY REFERENCES emergency_reports(id),
    priority      TEXT NOT NULL,
    reason        TEXT NOT NULL,
    classified_at TIMESTAMPTZ NOT NULL
);
`

// Migrate aplica el schema. Se llama una vez al arrancar el servicio.
func Migrate(ctx context.Context, db *sql.DB) error {
	if _, err := db.ExecContext(ctx, schema); err != nil {
		return fmt.Errorf("apply schema: %w", err)
	}
	return nil
}
