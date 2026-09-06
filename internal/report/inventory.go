package report

import (
	"io"

	"github.com/galexbh/husk/internal/huskerr"
	"github.com/galexbh/husk/internal/model"
)

// RenderInventory escribe el inventario en el formato de texto pedido:
// "table", "markdown" o "json". El formato "excel" no lo maneja este
// paquete — internal/excel construye el workbook y lo guarda directamente
// en un archivo, ya que no es un formato de texto.
func RenderInventory(w io.Writer, inv *model.Inventory, format string) error {
	switch format {
	case "", "table":
		return RenderInventoryTable(w, inv)
	case "markdown":
		return RenderInventoryMarkdown(w, inv)
	case "json":
		return RenderInventoryJSON(w, inv)
	default:
		return unsupportedFormat(format)
	}
}

// RenderInventorySummary escribe el resumen ejecutivo del inventario en el
// formato de texto pedido.
func RenderInventorySummary(w io.Writer, s *model.InventorySummary, format string) error {
	switch format {
	case "", "table":
		return RenderInventorySummaryTable(w, s)
	case "markdown":
		return RenderInventorySummaryMarkdown(w, s)
	case "json":
		return RenderInventorySummaryJSON(w, s)
	default:
		return unsupportedFormat(format)
	}
}

func unsupportedFormat(format string) error {
	return huskerr.New(
		"formato de salida no soportado: "+format,
		"usa --output table|markdown|json (para Excel, ver --output excel en el comando inventory)",
		nil,
	)
}
