package diff

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/jedib0t/go-pretty/v6/table"

	"github.com/galexbh/husk/internal/huskerr"
	"github.com/galexbh/husk/internal/model"
)

// Render escribe un DiffResult en el formato de texto pedido: "table",
// "markdown" o "json".
func Render(w io.Writer, d *model.DiffResult, format string) error {
	switch format {
	case "", "table":
		return renderTable(w, d, false)
	case "markdown":
		return renderTable(w, d, true)
	case "json":
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(d)
	default:
		return huskerr.New("formato de salida no soportado: "+format, "usa --output table|markdown|json", nil)
	}
}

func renderTable(w io.Writer, d *model.DiffResult, markdown bool) error {
	write := func(s string) error {
		_, err := io.WriteString(w, s)
		return err
	}

	if d.NoChanges {
		return write(fmt.Sprintf(
			"Sin cambios entre %s y %s.\n",
			d.FromGeneratedAt.Format("2006-01-02 15:04"), d.ToGeneratedAt.Format("2006-01-02 15:04"),
		))
	}

	if err := write(fmt.Sprintf(
		"Comparando %s → %s\n\n",
		d.FromGeneratedAt.Format("2006-01-02 15:04"), d.ToGeneratedAt.Format("2006-01-02 15:04"),
	)); err != nil {
		return err
	}

	if d.ScoreAvailable {
		t := table.NewWriter()
		t.SetTitle(fmt.Sprintf("Score: %.1f → %.1f (%+.1f)", d.ScoreFrom, d.ScoreTo, d.ScoreDelta))
		t.SetStyle(table.StyleLight)
		t.AppendHeader(table.Row{"Dimensión", "Antes", "Después", "Delta"})
		for _, dd := range d.DimensionDeltas {
			t.AppendRow(table.Row{dd.Name, fmt.Sprintf("%.1f", dd.From), fmt.Sprintf("%.1f", dd.To), fmt.Sprintf("%+.1f", dd.Delta)})
		}
		if err := renderOne(write, t, markdown); err != nil {
			return err
		}
	}

	if len(d.FindingsResolved) > 0 {
		if err := renderOne(write, findingsTable("Hallazgos resueltos (mejoras)", d.FindingsResolved), markdown); err != nil {
			return err
		}
	}
	if len(d.FindingsIntroduced) > 0 {
		if err := renderOne(write, findingsTable("Hallazgos nuevos (regresiones)", d.FindingsIntroduced), markdown); err != nil {
			return err
		}
	}

	return write(fmt.Sprintf("Hallazgos que persisten sin cambios: %d\n", d.FindingsPersisting))
}

func findingsTable(title string, findings []model.Finding) table.Writer {
	t := table.NewWriter()
	t.SetTitle(fmt.Sprintf("%s (%d)", title, len(findings)))
	t.SetStyle(table.StyleLight)
	t.AppendHeader(table.Row{"Severidad", "Categoría", "Namespace", "Recurso", "Mensaje"})
	for _, f := range findings {
		t.AppendRow(table.Row{f.Severity, f.Category, f.Namespace, f.Resource, f.Message})
	}
	return t
}

func renderOne(write func(string) error, t table.Writer, markdown bool) error {
	if markdown {
		return write(t.RenderMarkdown() + "\n\n")
	}
	return write(t.Render() + "\n\n")
}
