package domain

import (
	"fmt"
	"math"
)

// Location representa un punto geográfico con latitud y longitud.
type Location struct {
	Latitude  float64
	Longitude float64
}

// Validate verifica que las coordenadas estén dentro de rangos válidos.
func (l Location) Validate() error {
	if l.Latitude < -90 || l.Latitude > 90 {
		return fmt.Errorf("latitud debe estar entre -90 y 90, recibí %f", l.Latitude)
	}
	if l.Longitude < -180 || l.Longitude > 180 {
		return fmt.Errorf("longitud debe estar entre -180 y 180, recibí %f", l.Longitude)
	}
	return nil
}

// DistanceKmTo calcula la distancia en kilómetros entre este punto y otro,
// usando la fórmula de Haversine (distancia sobre la esfera terrestre).
//
// Es una aproximación: asume que la Tierra es una esfera perfecta.
// Para fines de despacho urbano es más que suficiente.
func (l Location) DistanceKmTo(other Location) float64 {
	const earthRadiusKm = 6371.0

	lat1 := toRadians(l.Latitude)
	lat2 := toRadians(other.Latitude)
	dLat := toRadians(other.Latitude - l.Latitude)
	dLon := toRadians(other.Longitude - l.Longitude)

	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1)*math.Cos(lat2)*
			math.Sin(dLon/2)*math.Sin(dLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return earthRadiusKm * c
}

// EstimatedArrivalMinutes calcula el tiempo estimado de llegada
// asumiendo una velocidad promedio urbana.
func (l Location) EstimatedArrivalMinutes(other Location, avgSpeedKmh float64) int {
	if avgSpeedKmh <= 0 {
		return 0
	}
	distanceKm := l.DistanceKmTo(other)
	hours := distanceKm / avgSpeedKmh
	minutes := hours * 60
	// Redondeamos hacia arriba: mejor sobreestimar que subestimar.
	return int(math.Ceil(minutes))
}

func toRadians(degrees float64) float64 {
	return degrees * math.Pi / 180
}