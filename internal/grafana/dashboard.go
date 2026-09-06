package grafana

// Dashboard es el "Dashboard JSON Model" exportable de Grafana: incluye
// __inputs/__requires para que, al importarlo, Grafana pida elegir el
// datasource de Prometheus/Thanos en vez de requerir un UID fijo — así el
// JSON es portable entre clusters/instalaciones de Grafana.
type Dashboard struct {
	Title         string     `json:"title"`
	UID           string     `json:"uid"`
	SchemaVersion int        `json:"schemaVersion"`
	Version       int        `json:"version"`
	Timezone      string     `json:"timezone"`
	Time          TimeRange  `json:"time"`
	Panels        []Panel    `json:"panels"`
	Inputs        []Input    `json:"__inputs"`
	Requires      []Requires `json:"__requires"`
}

// TimeRange es el rango de tiempo por defecto del dashboard.
type TimeRange struct {
	From string `json:"from"`
	To   string `json:"to"`
}

// Input declara una variable de entrada exportable del dashboard (ver
// datasourceVar en panels.go): Grafana la resuelve al importar.
type Input struct {
	Name        string `json:"name"`
	Label       string `json:"label"`
	Description string `json:"description"`
	Type        string `json:"type"`
	PluginID    string `json:"pluginId"`
	PluginName  string `json:"pluginName"`
}

// Requires declara las dependencias mínimas (versión de Grafana y del
// datasource) que necesita este dashboard.
type Requires struct {
	Type    string `json:"type"`
	ID      string `json:"id"`
	Name    string `json:"name"`
	Version string `json:"version"`
}

// BuildDashboard ensambla el dashboard "husk — Sizing & Capacity
// Overview": las mismas familias de métricas que usan internal/sizing e
// internal/capacity (container_cpu_usage_seconds_total,
// container_memory_working_set_bytes con id="/" para el nodo completo),
// generalizadas por namespace/nodo en vez de por workload específico, más
// un panel de alertas activas (la métrica meta ALERTS que expone el propio
// Prometheus para sus reglas de alerta).
func BuildDashboard() *Dashboard {
	const (
		colWidth = 12
		rowH     = 8
	)

	panels := []Panel{
		newTimeSeriesPanel(1, "CPU por namespace",
			`sum(rate(container_cpu_usage_seconds_total{container!="", container!="POD"}[5m])) by (namespace)`,
			"{{namespace}}", "short", 0, 0, colWidth, rowH),
		newTimeSeriesPanel(2, "Memoria working-set por namespace",
			`sum(container_memory_working_set_bytes{container!="", container!="POD"}) by (namespace)`,
			"{{namespace}}", "bytes", colWidth, 0, colWidth, rowH),
		newTimeSeriesPanel(3, "CPU por nodo (máquina completa)",
			`sum(rate(container_cpu_usage_seconds_total{id="/"}[5m])) by (node)`,
			"{{node}}", "short", 0, rowH, colWidth, rowH),
		newTimeSeriesPanel(4, "Memoria por nodo (máquina completa)",
			`sum(container_memory_working_set_bytes{id="/"}) by (node)`,
			"{{node}}", "bytes", colWidth, rowH, colWidth, rowH),
		newTablePanel(5, "Alertas activas", `ALERTS{alertstate="firing"}`, 0, 2*rowH, 2*colWidth, rowH),
	}

	return &Dashboard{
		Title:         "husk — Sizing & Capacity Overview",
		UID:           "husk-sizing-capacity-overview",
		SchemaVersion: 39,
		Version:       1,
		Timezone:      "browser",
		Time:          TimeRange{From: "now-24h", To: "now"},
		Panels:        panels,
		Inputs: []Input{{
			Name: "DS_PROMETHEUS", Label: "Prometheus", Description: "Thanos Querier o Prometheus del cluster",
			Type: "datasource", PluginID: "prometheus", PluginName: "Prometheus",
		}},
		Requires: []Requires{
			{Type: "grafana", ID: "grafana", Name: "Grafana", Version: "9.0.0"},
			{Type: "datasource", ID: "prometheus", Name: "Prometheus", Version: "1.0.0"},
		},
	}
}
