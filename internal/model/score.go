package model

import "time"

// DimensionScore es el resultado de una dimensión del score de resiliencia.
type DimensionScore struct {
	Name string `json:"name"`
	// Available indica si la dimensión pudo calcularse (por ejemplo,
	// Sizing requiere Prometheus/Thanos Querier). Las dimensiones no
	// disponibles no penalizan el score total: su peso se redistribuye
	// entre las dimensiones disponibles.
	Available bool `json:"available"`
	// Reason explica por qué la dimensión no está disponible, cuando
	// Available es false.
	Reason string `json:"reason,omitempty"`

	Score           float64  `json:"score"` // 0-100
	Weight          float64  `json:"weight"`
	EffectiveWeight float64  `json:"effectiveWeight"` // weight tras renormalizar entre dimensiones disponibles
	FindingIDs      []string `json:"findingIds,omitempty"`
}

// Score es el resultado de `husk score`: el score total de resiliencia
// (0-100) y su desglose por dimensión.
type Score struct {
	Total       float64          `json:"total"`
	Breakdown   []DimensionScore `json:"breakdown"`
	Findings    []Finding        `json:"findings"`
	GeneratedAt time.Time        `json:"generatedAt"`
}
