// Package grafana genera un dashboard de Grafana (JSON) con las mismas
// familias de métricas que husk ya usa para sizing y capacity, para que un
// operador pueda visualizarlas continuamente sin depender de correr husk.
// No se conecta a Grafana ni a Prometheus: solo produce el JSON.
package grafana

// datasourceVar es la variable de entrada estándar de Grafana para
// dashboards exportables ("Dashboard JSON Model" con __inputs): al
// importar, Grafana pide elegir el datasource de Prometheus/Thanos y
// sustituye esta referencia automáticamente. Así el dashboard no queda
// atado a un UID de datasource específico de un cluster.
const datasourceVar = "${DS_PROMETHEUS}"

// Panel es un panel de Grafana (subconjunto del esquema completo,
// suficiente para paneles de tipo timeseries/table/stat con targets de
// Prometheus).
type Panel struct {
	ID          int                    `json:"id"`
	Title       string                 `json:"title"`
	Type        string                 `json:"type"`
	Datasource  Datasource             `json:"datasource"`
	GridPos     GridPos                `json:"gridPos"`
	Targets     []Target               `json:"targets"`
	FieldConfig *FieldConfig           `json:"fieldConfig,omitempty"`
	Options     map[string]interface{} `json:"options,omitempty"`
}

// Datasource referencia el datasource de Prometheus/Thanos vía la
// variable de entrada del dashboard.
type Datasource struct {
	Type string `json:"type"`
	UID  string `json:"uid"`
}

// GridPos posiciona un panel en la grilla de 24 columnas de Grafana.
type GridPos struct {
	H int `json:"h"`
	W int `json:"w"`
	X int `json:"x"`
	Y int `json:"y"`
}

// Target es una query de Prometheus dentro de un panel.
type Target struct {
	Expr         string `json:"expr"`
	LegendFormat string `json:"legendFormat,omitempty"`
	RefID        string `json:"refId"`
}

// FieldConfig controla el formato de los valores del panel (por ejemplo,
// bytes o porcentaje).
type FieldConfig struct {
	Defaults FieldDefaults `json:"defaults"`
}

// FieldDefaults son los valores por defecto de formato de un campo (por
// ejemplo, la unidad "bytes").
type FieldDefaults struct {
	Unit string `json:"unit,omitempty"`
}

func promDatasource() Datasource {
	return Datasource{Type: "prometheus", UID: datasourceVar}
}

// newTimeSeriesPanel construye un panel de líneas de tiempo con una sola
// query.
func newTimeSeriesPanel(id int, title, expr, legend, unit string, x, y, w, h int) Panel {
	var fc *FieldConfig
	if unit != "" {
		fc = &FieldConfig{Defaults: FieldDefaults{Unit: unit}}
	}
	return Panel{
		ID:          id,
		Title:       title,
		Type:        "timeseries",
		Datasource:  promDatasource(),
		GridPos:     GridPos{H: h, W: w, X: x, Y: y},
		Targets:     []Target{{Expr: expr, LegendFormat: legend, RefID: "A"}},
		FieldConfig: fc,
	}
}

// newTablePanel construye un panel de tabla con una sola query en modo
// "instant" (adecuado para listados como alertas activas).
func newTablePanel(id int, title, expr string, x, y, w, h int) Panel {
	return Panel{
		ID:         id,
		Title:      title,
		Type:       "table",
		Datasource: promDatasource(),
		GridPos:    GridPos{H: h, W: w, X: x, Y: y},
		Targets:    []Target{{Expr: expr, RefID: "A"}},
	}
}
