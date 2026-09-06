package report

import "github.com/galexbh/husk/internal/model"

// BuildExecutiveSummary calcula el resumen ejecutivo de `report generate` a
// partir del inventario resumido, el score y la lista de hallazgos ya
// priorizada (para que los conteos de severidad coincidan exactamente con
// lo que se muestra en PrioritizedFindings, sin duplicados).
func BuildExecutiveSummary(invSummary *model.InventorySummary, s *model.Score, prioritized []model.Finding) model.ExecutiveSummary {
	summary := model.ExecutiveSummary{
		ScoreTotal:     s.Total,
		ScoreBreakdown: s.Breakdown,
		TotalFindings:  len(prioritized),
	}
	if invSummary != nil {
		summary.Inventory = *invSummary
	}

	for _, f := range prioritized {
		switch f.Severity {
		case model.RiskRed:
			summary.CriticalFindings++
		case model.RiskYellow:
			summary.WarningFindings++
		}
	}

	return summary
}
