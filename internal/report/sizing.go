package report

import (
	"fmt"
	"io"

	"github.com/galexbh/husk/internal/model"
)

func sizingHeader(r *model.SizingReport) string {
	return fmt.Sprintf(
		"Sizing report — lookback %s, CPU P%.0f, memoria P%.0f\n\n",
		r.Lookback, r.CPUPercentile*100, r.MemoryPercentile*100,
	)
}

// RenderSizingTable escribe el reporte de sizing como tablas ASCII, una
// sección por workload.
func RenderSizingTable(w io.Writer, r *model.SizingReport) error {
	if _, err := io.WriteString(w, sizingHeader(r)); err != nil {
		return err
	}
	for _, t := range sizingTables(r) {
		if _, err := io.WriteString(w, t.Render()+"\n\n"); err != nil {
			return err
		}
	}
	return nil
}

// RenderSizingMarkdown escribe el reporte de sizing como tablas Markdown.
func RenderSizingMarkdown(w io.Writer, r *model.SizingReport) error {
	if _, err := io.WriteString(w, sizingHeader(r)); err != nil {
		return err
	}
	for _, t := range sizingTables(r) {
		if _, err := io.WriteString(w, t.RenderMarkdown()+"\n\n"); err != nil {
			return err
		}
	}
	return nil
}

// RenderSizingJSON escribe el reporte de sizing completo como JSON
// indentado.
func RenderSizingJSON(w io.Writer, r *model.SizingReport) error {
	return writeJSON(w, r)
}

// RenderSizing escribe el reporte de sizing en el formato de texto pedido.
func RenderSizing(w io.Writer, r *model.SizingReport, format string) error {
	switch format {
	case "", "table":
		return RenderSizingTable(w, r)
	case "markdown":
		return RenderSizingMarkdown(w, r)
	case "json":
		return RenderSizingJSON(w, r)
	default:
		return unsupportedFormat(format)
	}
}

// RenderDryRunPatches escribe los patches sugeridos por --dry-run. Es
// puramente texto: husk nunca los aplica al cluster.
func RenderDryRunPatches(w io.Writer, patches []model.DryRunPatch) error {
	if len(patches) == 0 {
		_, err := io.WriteString(w, "--dry-run: no hay cambios sugeridos.\n")
		return err
	}
	if _, err := io.WriteString(w, "--dry-run: patches sugeridos (solo texto local; husk nunca los aplica al cluster)\n\n"); err != nil {
		return err
	}
	for _, p := range patches {
		if _, err := io.WriteString(w, p.YAML+"\n"); err != nil {
			return err
		}
	}
	return nil
}
