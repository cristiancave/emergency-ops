package domain

import (
	"strings"
	"testing"
	"time"
)

func TestEmergencyReport_Validate(t *testing.T) {
	// Un reporte válido de referencia que iremos modificando en cada caso.
	validReport := func() *EmergencyReport {
		return &EmergencyReport{
			ID:          "REP-001",
			PatientAge:  35,
			Symptoms:    []string{"dolor torácico"},
			Description: "paciente masculino con dolor en el pecho",
			ReportedAt:  time.Now(),
		}
	}

	// Table-driven tests: patrón idiomático de Go.
	// Cada caso tiene un nombre, una función que modifica el reporte,
	// y lo que esperamos (error o no).
	tests := []struct {
		name        string
		modify      func(*EmergencyReport)
		wantErr     bool
		wantErrMsg  string
	}{
		{
			name:    "reporte válido no retorna error",
			modify:  func(r *EmergencyReport) {},
			wantErr: false,
		},
		{
			name:       "ID vacío es inválido",
			modify:     func(r *EmergencyReport) { r.ID = "" },
			wantErr:    true,
			wantErrMsg: "ID es obligatorio",
		},
		{
			name:       "edad negativa es inválida",
			modify:     func(r *EmergencyReport) { r.PatientAge = -5 },
			wantErr:    true,
			wantErrMsg: "edad debe estar entre 0 y 130",
		},
		{
			name:       "edad mayor a 130 es inválida",
			modify:     func(r *EmergencyReport) { r.PatientAge = 150 },
			wantErr:    true,
			wantErrMsg: "edad debe estar entre 0 y 130",
		},
		{
			name:       "sin síntomas es inválido",
			modify:     func(r *EmergencyReport) { r.Symptoms = []string{} },
			wantErr:    true,
			wantErrMsg: "al menos un síntoma",
		},
		{
			name:       "fecha cero es inválida",
			modify:     func(r *EmergencyReport) { r.ReportedAt = time.Time{} },
			wantErr:    true,
			wantErrMsg: "fecha de reporte es obligatoria",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			report := validReport()
			tt.modify(report)

			err := report.Validate()

			if tt.wantErr {
				if err == nil {
					t.Errorf("esperaba error pero recibí nil")
					return
				}
				if !strings.Contains(err.Error(), tt.wantErrMsg) {
					t.Errorf("mensaje de error inesperado.\n  esperaba contener: %q\n  recibí: %q", tt.wantErrMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("no esperaba error pero recibí: %v", err)
				}
			}
		})
	}
}

func TestPriority_IsValid(t *testing.T) {
	tests := []struct {
		priority Priority
		want     bool
	}{
		{PriorityRed, true},
		{PriorityYellow, true},
		{PriorityGreen, true},
		{Priority("URGENTE"), false},
		{Priority(""), false},
		{Priority("red"), false}, // case-sensitive
	}

	for _, tt := range tests {
		t.Run(string(tt.priority), func(t *testing.T) {
			if got := tt.priority.IsValid(); got != tt.want {
				t.Errorf("Priority(%q).IsValid() = %v, want %v", tt.priority, got, tt.want)
			}
		})
	}
}