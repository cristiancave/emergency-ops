package domain

import (
	"errors"
	"fmt"
)

// Errores centinela: errores predefinidos que otros paquetes pueden comparar.
// Se usan cuando el error tiene un significado único e identificable.
var (
	ErrReportNotFound = errors.New("emergency report not found")
	ErrDuplicateID    = errors.New("report with this ID already exists")
)

// ErrInvalidReport crea un error de validación con un mensaje específico.
// Se usa para errores que tienen contexto variable (qué campo falló).
func ErrInvalidReport(reason string) error {
	return fmt.Errorf("invalid emergency report: %s", reason)
}