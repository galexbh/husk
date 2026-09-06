package report

import (
	"io"

	"github.com/galexbh/husk/internal/model"
)

// RenderInventoryTable escribe el inventario como tablas ASCII, una sección
// por tipo de recurso; las secciones vacías se omiten.
func RenderInventoryTable(w io.Writer, inv *model.Inventory) error {
	if len(inv.ConfigMaps) > 0 || len(inv.Secrets) > 0 {
		if _, err := io.WriteString(w, "Nota: ConfigMaps y Secrets muestran solo metadatos (nombre, tipo, cantidad de claves); nunca su contenido.\n\n"); err != nil {
			return err
		}
	}
	for _, t := range inventoryTables(inv) {
		if _, err := io.WriteString(w, t.Render()+"\n\n"); err != nil {
			return err
		}
	}
	return nil
}

// RenderInventorySummaryTable escribe el resumen ejecutivo y sus hallazgos
// como tablas ASCII.
func RenderInventorySummaryTable(w io.Writer, s *model.InventorySummary) error {
	if _, err := io.WriteString(w, summaryTable(s).Render()+"\n\n"); err != nil {
		return err
	}
	if len(s.Findings) == 0 {
		_, err := io.WriteString(w, "Sin hallazgos de riesgo.\n")
		return err
	}
	_, err := io.WriteString(w, findingsTable(s.Findings).Render()+"\n")
	return err
}
