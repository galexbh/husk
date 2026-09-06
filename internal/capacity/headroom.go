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

// nodeRisk determina el riesgo de un nodo y qué eje(s) lo causan. RiskRed:
// nodo no Ready/unschedulable, o CPU y memoria simultáneamente bajo
// threshold (saturación en ambos ejes). RiskYellow: un único eje bajo
// threshold con el otro sano — nivel intermedio para no tratar igual un
// nodo con headroom ajustado en un solo eje que uno realmente saturado
// (antes cualquier eje bajo el umbral marcaba RiskRed). RiskGreen: ningún
// eje bajo threshold. El segundo valor de retorno indica qué eje(s)
// dispararon el nivel devuelto ("cpu", "memory", ambos, o nil cuando el
// riesgo viene de not-ready/unschedulable o no hay riesgo).
func nodeRisk(ready, unschedulable bool, cpuHeadroom, memHeadroom, thresholdPercent float64) (model.RiskLevel, []string) {
	if !ready || unschedulable {
		return model.RiskRed, nil
	}
	cpuAtRisk := cpuHeadroom < thresholdPercent
	memAtRisk := memHeadroom < thresholdPercent
	switch {
	case cpuAtRisk && memAtRisk:
		return model.RiskRed, []string{"cpu", "memory"}
	case cpuAtRisk:
		return model.RiskYellow, []string{"cpu"}
	case memAtRisk:
		return model.RiskYellow, []string{"memory"}
	default:
		return model.RiskGreen, nil
	}
}
