package alertmanager

import (
	"regexp"

	"github.com/galexbh/husk/internal/model"
)

// relevantAlertPattern reconoce alertas de OOM, throttling y saturación de
// recursos — las únicas relevantes para correlacionar con hallazgos de
// sizing/capacity (otras alertas, como las de red o certificados, quedan
// fuera).
var relevantAlertPattern = regexp.MustCompile(`(?i)oom|throttl|memory|cpu|capacity|saturat|disk`)

// sizingOrCapacityCategories son las categorías de Finding que puede
// producir un incidente real de OOM/throttling/saturación (ver
// internal/score y internal/dr para el resto de categorías, que no
// aplican aquí).
var sizingOrCapacityCategories = map[string]bool{
	"sizing-no-limits":         true,
	"sizing-under-provisioned": true,
	"sizing-over-provisioned":  true,
	"node-saturated":           true,
	"node-headroom-warning":    true,
	"concentration-risk":       true,
}

// Correlate cruza hallazgos ya calculados con alertas activas, y devuelve
// los Finding.ID que tienen al menos una alerta activa relevante en su
// mismo namespace — evidencia de que el riesgo ya se manifestó como un
// incidente, no solo un riesgo preventivo.
//
// Es una correlación por namespace, no por recurso exacto: las labels de
// Alertmanager no siempre incluyen un identificador que coincida con
// Finding.Resource, así que namespace + tipo de alerta es la señal más
// confiable disponible sin acoplarse a un exporter específico.
func Correlate(findings []model.Finding, alerts []model.Alert) []string {
	namespacesWithActiveAlert := make(map[string]bool)
	for _, a := range alerts {
		if a.State != "active" {
			continue
		}
		if !relevantAlertPattern.MatchString(a.Name) {
			continue
		}
		if a.Namespace != "" {
			namespacesWithActiveAlert[a.Namespace] = true
		}
	}

	var correlated []string
	for _, f := range findings {
		if !sizingOrCapacityCategories[f.Category] {
			continue
		}
		if f.Namespace != "" && namespacesWithActiveAlert[f.Namespace] {
			correlated = append(correlated, f.ID)
		}
	}
	return correlated
}
