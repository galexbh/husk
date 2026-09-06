// Package capacity compara lo allocatable de cada nodo contra la suma de
// requests de los pods efectivamente programados en él, enriquecido con el
// consumo histórico real de internal/promclient cuando está disponible.
// Consume k8sclient/promclient directamente para recolectar (Nodes, Pods,
// Deployments/StatefulSets para concentración) pero no genera salida
// visual — eso vive en internal/report.
package capacity

import (
	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/galexbh/husk/internal/model"
)

// headroomPercent calcula, en porcentaje, cuánto de allocatable queda
// libre tras restar requested. Devuelve 100 cuando allocatable es cero o
// no se puede interpretar (nada que asignar, sin riesgo que reportar).
func headroomPercent(allocatable, requested resource.Quantity) float64 {
	allocF := allocatable.AsApproximateFloat64()
	if allocF <= 0 {
		return 100
	}
	reqF := requested.AsApproximateFloat64()

	free := allocF - reqF
	pct := (free / allocF) * 100
	switch {
	case pct < 0:
		return 0
	case pct > 100:
		return 100
	default:
		return pct
	}
}

// nodeRisk determina el riesgo de un nodo a partir de su estado y headroom.
func nodeRisk(ready, unschedulable bool, cpuHeadroom, memHeadroom, thresholdPercent float64) model.RiskLevel {
	if !ready || unschedulable {
		return model.RiskRed
	}
	if cpuHeadroom < thresholdPercent || memHeadroom < thresholdPercent {
		return model.RiskRed
	}
	return model.RiskGreen
}
