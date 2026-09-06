package promclient

import "fmt"

// SizingQueryParams identifica el workload y contenedor a analizar, y los
// parámetros de internal/config.SizingConfig que gobiernan la consulta.
type SizingQueryParams struct {
	Namespace string
	OwnerKind string // Deployment|StatefulSet|DaemonSet
	OwnerName string
	Container string

	Lookback      string  // ej. "7d"
	CPUPercentile float64 // ej. 0.95
	MemPercentile float64 // ej. 0.99
}

// ownerJoin selecciona, vía kube-state-metrics (bundled en
// openshift-monitoring), los pods que pertenecen al workload dado. Si
// kube-state-metrics no está disponible, la query simplemente no devuelve
// series y el análisis lo reporta como "sin datos" en vez de fallar — ver
// docs/metrics.md.
//
// kube_pod_owner refleja el owner *directo* del pod. Para un Deployment eso
// nunca es el Deployment mismo, sino el ReplicaSet intermedio
// (Deployment → ReplicaSet → Pod) — matchear owner_kind="Deployment" ahí
// nunca encuentra nada. StatefulSet y DaemonSet sí controlan sus pods
// directamente, así que el match simple es correcto para esos dos. Para
// Deployment hace falta el salto adicional vía kube_replicaset_owner (el
// mismo patrón de kubernetes-mixin/Grafana para este join).
//
// El resultado siempre va envuelto en paréntesis: los call sites lo
// concatenan detrás de un `* on(pod) group_left()` propio, y sin agrupar
// explícitamente el join de dos saltos, PromQL lo evalúa como una cadena
// plana de `*` de izquierda a derecha — el primer `group_left()` (sin
// lista de labels) no copia la label `replicaset` que el segundo salto
// necesita para matchear, y la query completa vuelve a devolver vector
// vacío en vez de fallar visiblemente.
func ownerJoin(p SizingQueryParams) string {
	if p.OwnerKind == "Deployment" {
		return fmt.Sprintf(
			`(label_replace(kube_pod_owner{namespace=%q, owner_kind="ReplicaSet"}, "replicaset", "$1", "owner_name", "(.*)") * on(replicaset, namespace) group_left() kube_replicaset_owner{namespace=%q, owner_kind="Deployment", owner_name=%q})`,
			p.Namespace, p.Namespace, p.OwnerName,
		)
	}
	return fmt.Sprintf(`kube_pod_owner{namespace=%q, owner_kind=%q, owner_name=%q}`, p.Namespace, p.OwnerKind, p.OwnerName)
}

// CPUPercentileQuery calcula el percentil (CPUPercentile) del consumo total
// de CPU (núcleos) del workload —sumado entre sus réplicas— a lo largo de
// Lookback.
func CPUPercentileQuery(p SizingQueryParams) string {
	return fmt.Sprintf(
		`quantile_over_time(%g, (sum(rate(container_cpu_usage_seconds_total{namespace=%q, container=%q, container!="", container!="POD"}[5m]) * on(pod) group_left() %s))[%s:5m])`,
		p.CPUPercentile, p.Namespace, p.Container, ownerJoin(p), p.Lookback,
	)
}

// MemoryPercentileQuery calcula el percentil (MemPercentile) de memoria
// working-set total del workload a lo largo de Lookback.
func MemoryPercentileQuery(p SizingQueryParams) string {
	return fmt.Sprintf(
		`quantile_over_time(%g, (sum(container_memory_working_set_bytes{namespace=%q, container=%q, container!="", container!="POD"} * on(pod) group_left() %s))[%s:5m])`,
		p.MemPercentile, p.Namespace, p.Container, ownerJoin(p), p.Lookback,
	)
}

// MemoryPeakQuery calcula el pico de memoria working-set total del workload
// a lo largo de Lookback.
func MemoryPeakQuery(p SizingQueryParams) string {
	return fmt.Sprintf(
		`max_over_time((sum(container_memory_working_set_bytes{namespace=%q, container=%q, container!="", container!="POD"} * on(pod) group_left() %s))[%s:5m])`,
		p.Namespace, p.Container, ownerJoin(p), p.Lookback,
	)
}
