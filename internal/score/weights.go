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
	drMissingBackupDeduction    = 10.0
	drMissingBackupCap          = 40.0
	drStaleBackupDeduction      = 5.0
	drStaleBackupCap            = 20.0
	drEtcdUnverifiableDeduction = 10.0
	drEtcdStaleDeduction        = 15.0
	drCSIUnsupportedDeduction   = 5.0
	drCSIUnsupportedCap         = 20.0

	capacitySaturatedNodeDeduction = 15.0
	capacitySaturatedNodeCap       = 60.0
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
