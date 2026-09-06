package promclient

import "fmt"

// Las métricas de cAdvisor con id="/" representan el cgroup raíz de la
// máquina completa (todo el nodo), con una etiqueta "node" que coincide
// directamente con el nombre del Node de Kubernetes — a diferencia de
// node-exporter, cuya etiqueta "instance" (host:puerto) no siempre coincide
// con ese nombre. Por eso se reutiliza la misma familia de métricas que
// internal/sizing en vez de las de node-exporter.

// NodeCPUAvgQuery calcula el uso promedio de CPU (núcleos) del nodo a lo
// largo de lookback.
func NodeCPUAvgQuery(node, lookback string) string {
	return fmt.Sprintf(`avg_over_time(rate(container_cpu_usage_seconds_total{id="/", node=%q}[5m])[%s:5m])`, node, lookback)
}

// NodeCPUPeakQuery calcula el pico de uso de CPU (núcleos) del nodo a lo
// largo de lookback.
func NodeCPUPeakQuery(node, lookback string) string {
	return fmt.Sprintf(`max_over_time(rate(container_cpu_usage_seconds_total{id="/", node=%q}[5m])[%s:5m])`, node, lookback)
}

// NodeMemoryAvgQuery calcula el uso promedio de memoria working-set
// (bytes) del nodo a lo largo de lookback.
func NodeMemoryAvgQuery(node, lookback string) string {
	return fmt.Sprintf(`avg_over_time(container_memory_working_set_bytes{id="/", node=%q}[%s])`, node, lookback)
}

// NodeMemoryPeakQuery calcula el pico de memoria working-set (bytes) del
// nodo a lo largo de lookback.
func NodeMemoryPeakQuery(node, lookback string) string {
	return fmt.Sprintf(`max_over_time(container_memory_working_set_bytes{id="/", node=%q}[%s])`, node, lookback)
}
