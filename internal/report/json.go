package report

import (
	"encoding/json"
	"io"

	"github.com/galexbh/husk/internal/model"
)

// RenderInventoryJSON escribe el inventario completo como JSON indentado.
func RenderInventoryJSON(w io.Writer, inv *model.Inventory) error {
	return writeJSON(w, inv)
}

// RenderInventorySummaryJSON escribe el resumen ejecutivo como JSON
// indentado.
func RenderInventorySummaryJSON(w io.Writer, s *model.InventorySummary) error {
	return writeJSON(w, s)
}

func writeJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}
