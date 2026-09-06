// Package logging centraliza la configuración de slog para husk.
package logging

import (
	"log/slog"
	"os"
)

// New construye el logger estructurado del proceso. verbose habilita el
// nivel debug (incluyendo, en fases posteriores, las queries PromQL
// ejecutadas y sus tiempos de respuesta); de lo contrario el nivel es info.
func New(verbose bool) *slog.Logger {
	level := slog.LevelInfo
	if verbose {
		level = slog.LevelDebug
	}
	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level})
	return slog.New(handler)
}
