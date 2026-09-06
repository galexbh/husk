package report

import (
	"io"

	"github.com/galexbh/husk/internal/model"
)

// RenderInventoryMarkdown escribe el inventario como tablas Markdown, una
// sección por tipo de recurso; las secciones vacías se omiten.
func RenderInventoryMarkdown(w io.Writer, inv *model.Inventory) error {
	if len(inv.ConfigMaps) > 0 || len(inv.Secrets) > 0 {
		if _, err := io.WriteString(w, "> **Nota:** ConfigMaps y Secrets muestran solo metadatos (nombre, tipo, cantidad de claves); nunca su contenido.\n\n"); err != nil {
			return err
		}
	}
	for _, t := range inventoryTables(inv) {
		if _, err := io.WriteString(w, t.RenderMarkdown()+"\n\n"); err != nil {
			return err
		}
	}
	return nil
}

// RenderInventorySummaryMarkdown escribe el resumen ejecutivo y sus
// hallazgos como tablas Markdown.
func RenderInventorySummaryMarkdown(w io.Writer, s *model.InventorySummary) error {
	if _, err := io.WriteString(w, summaryTable(s).RenderMarkdown()+"\n\n"); err != nil {
		return err
	}
	if len(s.Findings) == 0 {
		_, err := io.WriteString(w, "Sin hallazgos de riesgo.\n")
		return err
	}
	_, err := io.WriteString(w, findingsTable(s.Findings).RenderMarkdown()+"\n")
	return err
}
