package model

import "time"

// ExecutiveSummary es el resumen ejecutivo de `report generate`: las
// métricas agregadas que un lector sin tiempo necesita ver primero.
type ExecutiveSummary struct {
	ScoreTotal       float64          `json:"scoreTotal"`
	ScoreBreakdown   []DimensionScore `json:"scoreBreakdown"`
	Inventory        InventorySummary `json:"inventory"`
	CriticalFindings int              `json:"criticalFindings"` // severidad red
	WarningFindings  int              `json:"warningFindings"`  // severidad yellow
	TotalFindings    int              `json:"totalFindings"`
}

// Recommendation es una acción sugerida derivada de un hallazgo.
type Recommendation struct {
	FindingID string    `json:"findingId,omitempty"`
	Severity  RiskLevel `json:"severity"`
	Category  string    `json:"category"`
	Action    string    `json:"action"`
}

// ReportDetail agrupa el detalle técnico completo: los mismos reportes que
// producen `inventory`, `sizing report`, `capacity nodes` y `dr assess` por
// separado, aquí consolidados en un solo lugar.
type ReportDetail struct {
	Inventory *Inventory      `json:"inventory,omitempty"`
	Sizing    *SizingReport   `json:"sizing,omitempty"`
	Capacity  *CapacityReport `json:"capacity,omitempty"`
	DR        *DRReadiness    `json:"dr,omitempty"`
}

// Report es el resultado de `husk report generate`: portada/metadatos,
// resumen ejecutivo, hallazgos priorizados por severidad, detalle técnico
// y recomendaciones accionables. Appendix es el ClusterSnapshot completo en
// JSON, incluido solo cuando se pide con --appendix.
type Report struct {
	SchemaVersion string `json:"schemaVersion"`
	Meta          Meta   `json:"meta"`

	ExecutiveSummary    ExecutiveSummary `json:"executiveSummary"`
	PrioritizedFindings []Finding        `json:"prioritizedFindings"`
	Recommendations     []Recommendation `json:"recommendations"`
	Detail              ReportDetail     `json:"detail"`

	// AlertCorrelation es un enriquecimiento en tiempo real (no se guarda
	// en el historial ni participa en report diff: las alertas activas de
	// ahora no tienen sentido comparadas contra un snapshot pasado). Nil
	// cuando no se intentó correlacionar.
	AlertCorrelation *AlertCorrelation `json:"alertCorrelation,omitempty"`

	Appendix *ClusterSnapshot `json:"appendix,omitempty"`

	GeneratedAt time.Time `json:"generatedAt"`
}
