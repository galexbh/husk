package report

import (
	"fmt"
	"io"

	"github.com/galexbh/husk/internal/model"
)

func capacityHeader(r *model.CapacityReport) string {
	metricsNote := "consumo histórico: no disponible (Thanos Querier no accesible)"
	if r.HasMetrics {
		metricsNote = "consumo histórico: incluido"
	}
	return fmt.Sprintf("Capacity report — umbral de headroom %.0f%%, %s\n\n", r.HeadroomThresholdPercent, metricsNote)
}

// RenderCapacityTable escribe el reporte de capacity como tablas ASCII.
func RenderCapacityTable(w io.Writer, r *model.CapacityReport) error {
	if _, err := io.WriteString(w, capacityHeader(r)); err != nil {
		return err
	}
	for _, t := range capacityTables(r) {
		if _, err := io.WriteString(w, t.Render()+"\n\n"); err != nil {
			return err
		}
	}
	return nil
}

// RenderCapacityMarkdown escribe el reporte de capacity como tablas
// Markdown.
func RenderCapacityMarkdown(w io.Writer, r *model.CapacityReport) error {
	if _, err := io.WriteString(w, capacityHeader(r)); err != nil {
		return err
	}
	for _, t := range capacityTables(r) {
		if _, err := io.WriteString(w, t.RenderMarkdown()+"\n\n"); err != nil {
			return err
		}
	}
	return nil
}

// RenderCapacityJSON escribe el reporte de capacity completo como JSON
// indentado.
func RenderCapacityJSON(w io.Writer, r *model.CapacityReport) error {
	return writeJSON(w, r)
}

// RenderCapacity escribe el reporte de capacity en el formato de texto
// pedido.
func RenderCapacity(w io.Writer, r *model.CapacityReport, format string) error {
	switch format {
	case "", "table":
		return RenderCapacityTable(w, r)
	case "markdown":
		return RenderCapacityMarkdown(w, r)
	case "json":
		return RenderCapacityJSON(w, r)
	default:
		return unsupportedFormat(format)
	}
}
