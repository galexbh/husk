package report

import (
	"sort"

	"github.com/galexbh/husk/internal/model"
)

// severityRank ordena los hallazgos por severidad: los más graves primero.
func severityRank(r model.RiskLevel) int {
	switch r {
	case model.RiskRed:
		return 0
	case model.RiskYellow:
		return 1
	case model.RiskGreen:
		return 2
	default:
		return 3
	}
}

// PrioritizeFindings combina varias listas de hallazgos (de inventory
// summary, sizing/dr/capacity vía score), quita duplicados por ID y los
// ordena por severidad (rojo primero) y, dentro de la misma severidad, por
// categoría y namespace para un orden estable y reproducible.
func PrioritizeFindings(sources ...[]model.Finding) []model.Finding {
	seen := make(map[string]bool)
	merged := make([]model.Finding, 0)

	for _, list := range sources {
		for _, f := range list {
			key := f.ID
			if key == "" {
				key = f.Category + "|" + f.Namespace + "|" + f.Resource + "|" + f.Message
			}
			if seen[key] {
				continue
			}
			seen[key] = true
			merged = append(merged, f)
		}
	}

	sort.SliceStable(merged, func(i, j int) bool {
		if severityRank(merged[i].Severity) != severityRank(merged[j].Severity) {
			return severityRank(merged[i].Severity) < severityRank(merged[j].Severity)
		}
		if merged[i].Category != merged[j].Category {
			return merged[i].Category < merged[j].Category
		}
		return merged[i].Namespace < merged[j].Namespace
	})

	return merged
}

// BuildRecommendations deriva una acción sugerida por cada hallazgo, en el
// mismo orden en que aparecen (se espera una lista ya priorizada). La
// acción viene de model.Finding.Recommendation (poblado por
// model.NewFinding desde internal/model/finding_catalog.go): un único
// catálogo, sin duplicar aquí el texto por categoría.
func BuildRecommendations(findings []model.Finding) []model.Recommendation {
	recs := make([]model.Recommendation, 0, len(findings))
	for _, f := range findings {
		action := f.Recommendation
		if action == "" {
			action = f.Message
		}
		recs = append(recs, model.Recommendation{
			FindingID: f.ID,
			Severity:  f.Severity,
			Category:  f.Category,
			Action:    action,
		})
	}
	return recs
}
