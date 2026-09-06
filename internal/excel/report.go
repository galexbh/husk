package excel

import (
	"fmt"

	"github.com/galexbh/husk/internal/model"
)

// BuildReportWorkbook construye el workbook de `husk report generate
// --output excel`: hojas de Findings y Recommendations siempre, más las
// hojas de detalle de inventario cuando el reporte las incluye (mismos
// builders que BuildInventoryWorkbook), y una hoja "Summary" con
// hipervínculos a cada una.
func BuildReportWorkbook(r *model.Report, freezeColumns int, highlightRisks bool) (*Workbook, error) {
	wb, err := New(freezeColumns, highlightRisks)
	if err != nil {
		return nil, err
	}

	entries := make([]SummaryEntry, 0, 16)
	addSheet := func(label string, sheet Sheet) error {
		if len(sheet.Rows) == 0 {
			entries = append(entries, SummaryEntry{Label: label, Count: 0})
			return nil
		}
		if err := wb.AddSheet(sheet); err != nil {
			return fmt.Errorf("hoja %s: %w", sheet.Name, err)
		}
		entries = append(entries, SummaryEntry{Label: label, Count: len(sheet.Rows), SheetLink: sheet.Name})
		return nil
	}

	entries = append(entries, SummaryEntry{Label: fmt.Sprintf("Score de resiliencia: %.1f/100", r.ExecutiveSummary.ScoreTotal), Count: int(r.ExecutiveSummary.ScoreTotal)})

	if err := addSheet("Findings", findingsSheet(r.PrioritizedFindings)); err != nil {
		return nil, err
	}
	if err := addSheet("Recommendations", recommendationsSheet(r.Recommendations)); err != nil {
		return nil, err
	}

	if inv := r.Detail.Inventory; inv != nil {
		if err := addSheet("Deployments", workloadSheet("Deployments", inv.Deployments)); err != nil {
			return nil, err
		}
		if err := addSheet("StatefulSets", workloadSheet("StatefulSets", inv.StatefulSets)); err != nil {
			return nil, err
		}
		if err := addSheet("DaemonSets", workloadSheet("DaemonSets", inv.DaemonSets)); err != nil {
			return nil, err
		}
		if err := addSheet("Nodes", nodesSheet(inv.Nodes)); err != nil {
			return nil, err
		}
	}

	if err := wb.FinalizeSummary(fmt.Sprintf("Reporte consolidado — %s", r.Meta.ClusterName), entries); err != nil {
		return nil, err
	}

	return wb, nil
}

func findingsSheet(findings []model.Finding) Sheet {
	rows := make([][]string, 0, len(findings))
	risks := make([]model.RiskLevel, 0, len(findings))
	for _, f := range findings {
		rows = append(rows, []string{string(f.Severity), f.Category, f.Namespace, f.Resource, f.Message})
		risks = append(risks, f.Severity)
	}
	return Sheet{
		Name:     "Findings",
		Headers:  []string{"Severidad", "Categoría", "Namespace", "Recurso", "Mensaje"},
		Rows:     rows,
		RowRisks: risks,
	}
}

func recommendationsSheet(recs []model.Recommendation) Sheet {
	rows := make([][]string, 0, len(recs))
	risks := make([]model.RiskLevel, 0, len(recs))
	for _, r := range recs {
		rows = append(rows, []string{string(r.Severity), r.Category, r.Action})
		risks = append(risks, r.Severity)
	}
	return Sheet{
		Name:     "Recommendations",
		Headers:  []string{"Severidad", "Categoría", "Acción"},
		Rows:     rows,
		RowRisks: risks,
	}
}
