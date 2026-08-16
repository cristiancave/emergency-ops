package domain

import (
	"math"
	"testing"
)

func TestAmbulanceType_CanHandle(t *testing.T) {
	tests := []struct {
		name        string
		ambType     AmbulanceType
		priority    Priority
		wantCanDo   bool
	}{
		{"ADVANCED puede RED", AmbulanceTypeAdvanced, PriorityRed, true},
		{"ADVANCED puede YELLOW", AmbulanceTypeAdvanced, PriorityYellow, true},
		{"ADVANCED puede GREEN", AmbulanceTypeAdvanced, PriorityGreen, true},
		{"BASIC NO puede RED", AmbulanceTypeBasic, PriorityRed, false},
		{"BASIC puede YELLOW", AmbulanceTypeBasic, PriorityYellow, true},
		{"BASIC puede GREEN", AmbulanceTypeBasic, PriorityGreen, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.ambType.CanHandle(tt.priority)
			if got != tt.wantCanDo {
				t.Errorf("CanHandle(%q) = %v, esperaba %v", tt.priority, got, tt.wantCanDo)
			}
		})
	}
}

func TestAmbulance_Validate(t *testing.T) {
	validAmbulance := func() *Ambulance {
		return &Ambulance{
			ID:       "AMB-001",
			Type:     AmbulanceTypeAdvanced,
			Status:   AmbulanceStatusAvailable,
			Location: Location{Latitude: 4.7110, Longitude: -74.0721},
			CrewSize: 3,
		}
	}

	tests := []struct {
		name    string
		modify  func(*Ambulance)
		wantErr bool
	}{
		{"ambulancia válida", func(a *Ambulance) {}, false},
		{"ID vacío", func(a *Ambulance) { a.ID = "" }, true},
		{"tipo inválido", func(a *Ambulance) { a.Type = "SUPER" }, true},
		{"status inválido", func(a *Ambulance) { a.Status = "OFFLINE" }, true},
		{"latitud inválida", func(a *Ambulance) { a.Location.Latitude = 200 }, true},
		{"crew size cero", func(a *Ambulance) { a.CrewSize = 0 }, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			amb := validAmbulance()
			tt.modify(amb)

			err := amb.Validate()
			if tt.wantErr && err == nil {
				t.Error("esperaba error, no lo recibí")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("no esperaba error, recibí: %v", err)
			}
		})
	}
}

func TestLocation_DistanceKmTo(t *testing.T) {
	// Bogotá (Plaza de Bolívar) → Medellín (Parque Berrío)
	// Distancia real aproximada: ~240 km en línea recta
	bogota := Location{Latitude: 4.5981, Longitude: -74.0758}
	medellin := Location{Latitude: 6.2518, Longitude: -75.5636}

	distance := bogota.DistanceKmTo(medellin)

	// Aceptamos un margen del 5% (Haversine no es exacta al 100%)
	expected := 240.0
	tolerance := expected * 0.05
	if math.Abs(distance-expected) > tolerance {
		t.Errorf("distancia = %.2f km, esperaba %.2f ± %.2f", distance, expected, tolerance)
	}
}

func TestLocation_DistanceKmTo_SamePoint(t *testing.T) {
	loc := Location{Latitude: 4.7110, Longitude: -74.0721}
	distance := loc.DistanceKmTo(loc)

	if distance != 0 {
		t.Errorf("distancia de un punto a sí mismo debe ser 0, recibí %f", distance)
	}
}

func TestLocation_EstimatedArrivalMinutes(t *testing.T) {
	from := Location{Latitude: 4.7110, Longitude: -74.0721}
	to := Location{Latitude: 4.7500, Longitude: -74.0500}

	// A 50 km/h, ~5 km debería tomar unos 6 minutos
	eta := from.EstimatedArrivalMinutes(to, 50.0)

	if eta < 3 || eta > 10 {
		t.Errorf("ETA = %d minutos, esperaba entre 3 y 10", eta)
	}
}

func TestPriority_IsValid(t *testing.T) {
	if !PriorityRed.IsValid() {
		t.Error("PriorityRed debe ser válida")
	}
	if Priority("URGENTE").IsValid() {
		t.Error("prioridad arbitraria no debe ser válida")
	}
}