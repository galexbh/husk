package report

import (
	"encoding/json"
	"io"
	"time"

	"github.com/galexbh/husk/internal/model"
)

// BuildReport ensambla el reporte consolidado de `report generate` a
// partir de los reportes ya producidos por internal/inventory,
// internal/sizing, internal/capacity, internal/dr e internal/score.
// invSummary y las secciones de detail pueden ser nil cuando no se
// recolectaron (por ejemplo, sizing sin Prometheus disponible).
func BuildReport(meta model.Meta, invSummary *model.InventorySummary, detail model.ReportDetail, s *model.Score, appendix *model.ClusterSnapshot) *model.Report {
	sources := make([][]model.Finding, 0, 2)
	if invSummary != nil {
		sources = append(sources, invSummary.Findings)
	}
	if s != nil {
		sources = append(sources, s.Findings)
	}
	findings := PrioritizeFindings(sources...)

	var scoreForSummary model.Score
	if s != nil {
		scoreForSummary = *s
	}

	return &model.Report{
		SchemaVersion:       model.SchemaVersion,
		Meta:                meta,
		ExecutiveSummary:    BuildExecutiveSummary(invSummary, &scoreForSummary, findings),
		PrioritizedFindings: findings,
		Recommendations:     BuildRecommendations(findings),
		Detail:              detail,
		Appendix:            appendix,
		GeneratedAt:         time.Now(),
	}
}

// RenderReportTable escribe el reporte consolidado como tablas ASCII.
func RenderReportTable(w io.Writer, r *model.Report) error {
	for _, t := range reportTables(r) {
		if _, err := io.WriteString(w, t.Render()+"\n\n"); err != nil {
			return err
		}
	}
	return appendixJSON(w, r)
}

// RenderReportMarkdown escribe el reporte consolidado como tablas
// Markdown.
func RenderReportMarkdown(w io.Writer, r *model.Report) error {
	for _, t := range reportTables(r) {
		if _, err := io.WriteString(w, t.RenderMarkdown()+"\n\n"); err != nil {
			return err
		}
	}
	return appendixJSON(w, r)
}

// RenderReportJSON escribe el reporte consolidado (incluido el apéndice,
// si está presente) como JSON indentado.
func RenderReportJSON(w io.Writer, r *model.Report) error {
	return writeJSON(w, r)
}

// RenderReport escribe el reporte consolidado en el formato de texto
// pedido.
func RenderReport(w io.Writer, r *model.Report, format string) error {
	switch format {
	case "", "table":
		return RenderReportTable(w, r)
	case "markdown":
		return RenderReportMarkdown(w, r)
	case "json":
		return RenderReportJSON(w, r)
	default:
		return unsupportedFormat(format)
	}
}

// appendixJSON escribe el apéndice JSON al final de la salida table/markdown,
// solo cuando --appendix lo solicitó (r.Appendix != nil).
func appendixJSON(w io.Writer, r *model.Report) error {
	if r.Appendix == nil {
		return nil
	}
	if _, err := io.WriteString(w, "Apéndice JSON (snapshot completo)\n==================================\n"); err != nil {
		return err
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(r.Appendix)
}
