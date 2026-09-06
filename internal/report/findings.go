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

// recommendationActions traduce cada categoría de hallazgo conocida en una
// acción sugerida concreta. Categorías sin entrada aquí caen al mensaje
// genérico del propio hallazgo.
var recommendationActions = map[string]string{
	"single-replica":             "Aumenta replicas a 2 o más para tolerar la pérdida de un pod, o documenta por qué el workload es de instancia única.",
	"missing-limits":             "Define resources.limits.cpu y resources.limits.memory; usa `husk sizing report --dry-run` para una recomendación basada en consumo real.",
	"missing-resourcequota":      "Crea un ResourceQuota para el namespace, para evitar que un workload sin límites agote la capacidad del cluster.",
	"sizing-no-limits":           "Define resources.limits; usa `husk sizing report --dry-run` para el patch sugerido.",
	"sizing-under-provisioned":   "Aumenta requests/limits: el consumo observado supera lo declarado (riesgo de throttling/OOM). Ver `husk sizing report --dry-run`.",
	"sizing-over-provisioned":    "Reduce requests/limits al consumo real observado para liberar headroom del cluster. Ver `husk sizing report --dry-run`.",
	"oadp-not-installed":         "Instala y configura el operador OADP en openshift-adp para habilitar backups gestionados con Velero.",
	"oadp-unhealthy":             "Revisa los logs del operador OADP y la DataProtectionApplication: la reconciliación está fallando.",
	"missing-backup":             "Crea un Schedule/Backup de Velero que incluya este namespace.",
	"stale-backup":               "Verifica que el Schedule de backup esté corriendo; el último backup completado supera la antigüedad máxima configurada.",
	"etcd-snapshot-unverifiable": "Configura y documenta un CronJob de backup de etcd, o confirma manualmente la antigüedad del último snapshot.",
	"etcd-snapshot-stale":        "Ejecuta un nuevo snapshot de etcd; el más reciente supera la antigüedad máxima configurada.",
	"csi-snapshot-unsupported":   "Usa una StorageClass cuyo provisioner tenga un VolumeSnapshotClass asociado, o crea uno para el provisioner actual.",
	"missing-pdb":                "Define un PodDisruptionBudget que cubra este workload, para protegerlo durante drenados de nodos.",
	"missing-topology-spread":    "Agrega topologySpreadConstraints al pod template para distribuir las réplicas entre nodos/zonas.",
	"node-saturated":             "Añade capacidad al cluster o redistribuye carga: el nodo tiene poco headroom libre.",
	"concentration-risk":         "Agrega topologySpreadConstraints o antiafinidad para distribuir las réplicas en más nodos/zonas.",
}

// BuildRecommendations deriva una acción sugerida por cada hallazgo, en el
// mismo orden en que aparecen (se espera una lista ya priorizada).
func BuildRecommendations(findings []model.Finding) []model.Recommendation {
	recs := make([]model.Recommendation, 0, len(findings))
	for _, f := range findings {
		action, ok := recommendationActions[f.Category]
		if !ok {
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
