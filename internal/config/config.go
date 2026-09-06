// Package config define la configuración de husk: los defaults hardcodeados
// (config.Default) y su combinación con un archivo YAML opcional
// (~/.husk/config.yaml o --config), vía config.Load. Ningún otro paquete
// debe leer YAML de configuración directamente.
package config

// Config es la configuración completa de husk.
type Config struct {
	Namespaces NamespacesConfig `yaml:"namespaces" mapstructure:"namespaces"`
	Sizing     SizingConfig     `yaml:"sizing" mapstructure:"sizing"`
	Capacity   CapacityConfig   `yaml:"capacity" mapstructure:"capacity"`
	DR         DRConfig         `yaml:"dr" mapstructure:"dr"`
	Score      ScoreConfig      `yaml:"score" mapstructure:"score"`
	Inventory  InventoryConfig  `yaml:"inventory" mapstructure:"inventory"`
	History    HistoryConfig    `yaml:"history" mapstructure:"history"`
}

// NamespacesConfig controla el filtro transversal de namespaces (ver
// internal/nsfilter).
type NamespacesConfig struct {
	ExcludePatterns []string `yaml:"exclude_patterns" mapstructure:"exclude_patterns"`
}

// SizingConfig controla los umbrales y ventana histórica del análisis de
// sizing (Fase 2).
type SizingConfig struct {
	CPUPercentile       float64 `yaml:"cpu_percentile" mapstructure:"cpu_percentile"`
	MemoryPercentile    float64 `yaml:"memory_percentile" mapstructure:"memory_percentile"`
	Lookback            string  `yaml:"lookback" mapstructure:"lookback"`
	CPULimitMultiplier  float64 `yaml:"cpu_limit_multiplier" mapstructure:"cpu_limit_multiplier"`
	MemoryBufferPercent float64 `yaml:"memory_buffer_percent" mapstructure:"memory_buffer_percent"`
	// OverProvisionFactor: si el request declarado de un contenedor supera
	// a su consumo observado por este factor, se marca "sobreaprovisionado".
	OverProvisionFactor float64 `yaml:"over_provision_factor" mapstructure:"over_provision_factor"`
	// SidecarContainerNames: fragmentos de nombre (comparación
	// case-insensitive, substring) que identifican contenedores sidecar
	// conocidos (service mesh, agentes de observabilidad/APM); esos
	// contenedores se marcan "sidecar-ignorado" en vez de aplicarles el
	// veredicto de sizing de la aplicación.
	SidecarContainerNames []string `yaml:"sidecar_container_names" mapstructure:"sidecar_container_names"`
}

// CapacityConfig controla los umbrales y ventana histórica del análisis de
// capacity planning (Fase 3).
type CapacityConfig struct {
	// HeadroomThresholdPercent: por debajo de este porcentaje de headroom
	// libre (CPU o memoria), un nodo se marca en riesgo de saturación.
	// Coincide con el criterio de la dimensión Capacity del score de
	// resiliencia ("100 si hay headroom suficiente, > 30% libre").
	HeadroomThresholdPercent float64 `yaml:"headroom_threshold_percent" mapstructure:"headroom_threshold_percent"`
	// Lookback es la ventana histórica usada para el consumo observado por
	// nodo (CPU/memoria promedio y pico).
	Lookback string `yaml:"lookback" mapstructure:"lookback"`
}

// DRConfig controla los umbrales de antigüedad usados por DR readiness
// (Fase 4).
type DRConfig struct {
	BackupMaxAge       string `yaml:"backup_max_age" mapstructure:"backup_max_age"`
	EtcdSnapshotMaxAge string `yaml:"etcd_snapshot_max_age" mapstructure:"etcd_snapshot_max_age"`
}

// ScoreConfig controla los pesos del score de resiliencia (Fase 4).
type ScoreConfig struct {
	Weights ScoreWeights `yaml:"weights" mapstructure:"weights"`
}

// ScoreWeights son los pesos por dimensión del score de resiliencia; deben
// sumar 1.0.
type ScoreWeights struct {
	Sizing   float64 `yaml:"sizing" mapstructure:"sizing"`
	DR       float64 `yaml:"dr" mapstructure:"dr"`
	Capacity float64 `yaml:"capacity" mapstructure:"capacity"`
	PDB      float64 `yaml:"pdb" mapstructure:"pdb"`
	Topology float64 `yaml:"topology" mapstructure:"topology"`
}

// InventoryConfig controla el comportamiento de `husk inventory` (Fase 1).
type InventoryConfig struct {
	IncludeExtended bool                 `yaml:"include_extended" mapstructure:"include_extended"`
	Excel           InventoryExcelConfig `yaml:"excel" mapstructure:"excel"`
}

// InventoryExcelConfig controla el formato del workbook Excel generado por
// `husk inventory --output excel`.
type InventoryExcelConfig struct {
	HighlightRisks bool `yaml:"highlight_risks" mapstructure:"highlight_risks"`
	FreezeColumns  int  `yaml:"freeze_columns" mapstructure:"freeze_columns"`
}

// HistoryConfig controla el historial local de snapshots
// (~/.husk/history/), usado por `report generate` y `report diff` (Fase 5).
type HistoryConfig struct {
	// RetainCount es cuántos snapshots recientes se conservan por cluster;
	// los más antiguos se podan automáticamente en cada `report generate`.
	RetainCount int `yaml:"retain_count" mapstructure:"retain_count"`
}
