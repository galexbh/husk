// Package diff compara dos ClusterSnapshot y resalta regresiones y
// mejoras: cambios en el score de resiliencia y hallazgos resueltos o
// introducidos entre ambos. Es una función pura sobre modelos ya
// poblados — no recolecta datos ni depende de internal/report.
package diff

import "github.com/galexbh/husk/internal/model"

// Compute compara from (snapshot anterior) contra to (snapshot posterior).
// Usa model.Score.Findings como la lista consolidada de hallazgos de cada
// snapshot (la misma que produce `husk score`), identificados por su ID
// estable (model.Finding.ID).
func Compute(from, to *model.ClusterSnapshot) *model.DiffResult {
	result := &model.DiffResult{
		FromGeneratedAt: from.Meta.GeneratedAt,
		ToGeneratedAt:   to.Meta.GeneratedAt,
	}

	if from.Score != nil && to.Score != nil {
		result.ScoreAvailable = true
		result.ScoreFrom = from.Score.Total
		result.ScoreTo = to.Score.Total
		result.ScoreDelta = to.Score.Total - from.Score.Total
		result.DimensionDeltas = dimensionDeltas(from.Score.Breakdown, to.Score.Breakdown)
	}

	fromFindings := findingsOf(from)
	toFindings := findingsOf(to)

	fromByID := indexByID(fromFindings)
	toByID := indexByID(toFindings)

	for id, f := range fromByID {
		if _, stillPresent := toByID[id]; !stillPresent {
			result.FindingsResolved = append(result.FindingsResolved, f)
		} else {
			result.FindingsPersisting++
		}
	}
	for id, f := range toByID {
		if _, existedBefore := fromByID[id]; !existedBefore {
			result.FindingsIntroduced = append(result.FindingsIntroduced, f)
		}
	}

	result.NoChanges = len(result.FindingsResolved) == 0 &&
		len(result.FindingsIntroduced) == 0 &&
		(!result.ScoreAvailable || result.ScoreDelta == 0)

	return result
}

// findingsOf devuelve la lista consolidada de hallazgos de un snapshot:
// los de Score si están disponibles (ya agregan sizing+dr+capacity), o los
// del InventorySummary como fallback cuando el snapshot no tiene Score
// (por ejemplo, un snapshot guardado por una fase anterior de husk).
func findingsOf(s *model.ClusterSnapshot) []model.Finding {
	if s.Score != nil {
		return s.Score.Findings
	}
	if s.InventorySummary != nil {
		return s.InventorySummary.Findings
	}
	return nil
}

func indexByID(findings []model.Finding) map[string]model.Finding {
	out := make(map[string]model.Finding, len(findings))
	for _, f := range findings {
		if f.ID == "" {
			continue
		}
		out[f.ID] = f
	}
	return out
}

func dimensionDeltas(from, to []model.DimensionScore) []model.DimensionDelta {
	fromByName := make(map[string]model.DimensionScore, len(from))
	for _, d := range from {
		fromByName[d.Name] = d
	}

	deltas := make([]model.DimensionDelta, 0, len(to))
	for _, t := range to {
		f, ok := fromByName[t.Name]
		if !ok || !f.Available || !t.Available {
			continue
		}
		deltas = append(deltas, model.DimensionDelta{
			Name: t.Name, From: f.Score, To: t.Score, Delta: t.Score - f.Score,
		})
	}
	return deltas
}
