package report

import (
	"io"

	"github.com/galexbh/husk/internal/model"
)

// RenderDRTable escribe el reporte de DR readiness como tablas ASCII.
func RenderDRTable(w io.Writer, dr *model.DRReadiness) error {
	for _, t := range drTables(dr) {
		if _, err := io.WriteString(w, t.Render()+"\n\n"); err != nil {
			return err
		}
	}
	return nil
}

// RenderDRMarkdown escribe el reporte de DR readiness como tablas
// Markdown.
func RenderDRMarkdown(w io.Writer, dr *model.DRReadiness) error {
	for _, t := range drTables(dr) {
		if _, err := io.WriteString(w, t.RenderMarkdown()+"\n\n"); err != nil {
			return err
		}
	}
	return nil
}

// RenderDRJSON escribe el DRReadiness completo como JSON indentado.
func RenderDRJSON(w io.Writer, dr *model.DRReadiness) error {
	return writeJSON(w, dr)
}

// RenderDR escribe el reporte de DR readiness en el formato de texto
// pedido.
func RenderDR(w io.Writer, dr *model.DRReadiness, format string) error {
	switch format {
	case "", "table":
		return RenderDRTable(w, dr)
	case "markdown":
		return RenderDRMarkdown(w, dr)
	case "json":
		return RenderDRJSON(w, dr)
	default:
		return unsupportedFormat(format)
	}
}
