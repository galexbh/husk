// Package score calcula el score de resiliencia (0-100): un promedio
// ponderado de cinco dimensiones (sizing, DR, capacity, PDB, topology
// spread), a partir de los reportes ya producidos por internal/sizing,
// internal/capacity e internal/dr. No recolecta datos de Kubernetes ni
// Prometheus por sí mismo — solo combina modelos ya poblados.
package score

// Deducciones fijas por hallazgo, por dimensión. CLAUDE.md solo expone los
// pesos entre dimensiones (score.weights.*) como configurables por YAML;
// la severidad relativa entre distintos tipos de hallazgo dentro de una
// misma dimensión es una decisión de producto de husk, documentada aquí.
const (
	sizingNoLimitsDeduction         = 15.0
	sizingUnderProvisionedDeduction = 12.0
	sizingOverProvisionedDeduction  = 5.0

	drOADPNotInstalledDeduction = 40.0
	drOADPUnhealthyDeduction    = 25.0
	// drMissingBackupMaxDeduction es la deducción cuando el 100% de los
	// namespaces de aplicación evaluados no tiene backup; escala
	// proporcionalmente con la fracción de namespaces sin backup, en vez de
	// un monto fijo por namespace con tope (antes: 10.0 por namespace, tope
	// 40.0) — así el mismo hallazgo absoluto no pesa distinto según el
	// tamaño del cluster.
	drMissingBackupMaxDeduction = 40.0
	drStaleBackupDeduction      = 5.0
	drStaleBackupCap            = 20.0
	drEtcdUnverifiableDeduction = 10.0
	drEtcdStaleDeduction        = 15.0
	drCSIUnsupportedDeduction   = 5.0
	drCSIUnsupportedCap         = 20.0

	// capacitySaturatedMaxDeduction es la deducción cuando todos los nodos
	// están en riesgo ALTO; escala proporcionalmente (antes: 15.0 por nodo,
	// tope 60.0), igual criterio que drMissingBackupMaxDeduction.
	capacitySaturatedMaxDeduction = 60.0
	// capacityHeadroomWarningWeight pondera un nodo en riesgo MEDIO (un solo
	// eje de headroom bajo el umbral) como la mitad de uno en riesgo ALTO
	// (ambos ejes, o nodo no-Ready/unschedulable) al proporcionalizar la
	// deducción de capacity.
	capacityHeadroomWarningWeight  = 0.5
	capacityConcentrationDeduction = 10.0
	capacityConcentrationCap       = 40.0
)

// clampScore acota un score al rango [0, 100].
func clampScore(v float64) float64 {
	switch {
	case v < 0:
		return 0
	case v > 100:
		return 100
	default:
		return v
	}
}

// capDeduction limita una deducción acumulada a un máximo, para que muchos
// hallazgos del mismo tipo no puedan por sí solos llevar la dimensión a 0
// de forma desproporcionada.
func capDeduction(total, limit float64) float64 {
	if total > limit {
		return limit
	}
	return total
}
