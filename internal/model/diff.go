package model

import "time"

// DimensionDelta es el cambio de score de una dimensión entre dos
// snapshots.
type DimensionDelta struct {
	Name  string  `json:"name"`
	From  float64 `json:"from"`
	To    float64 `json:"to"`
	Delta float64 `json:"delta"`
}

// DiffResult es el resultado de `husk report diff --from --to`.
type DiffResult struct {
	FromGeneratedAt time.Time `json:"fromGeneratedAt"`
	ToGeneratedAt   time.Time `json:"toGeneratedAt"`

	// ScoreAvailable indica si ambos snapshots tenían un Score calculado;
	// si no, ScoreDelta/DimensionDeltas no son significativos.
	ScoreAvailable  bool             `json:"scoreAvailable"`
	ScoreFrom       float64          `json:"scoreFrom,omitempty"`
	ScoreTo         float64          `json:"scoreTo,omitempty"`
	ScoreDelta      float64          `json:"scoreDelta,omitempty"`
	DimensionDeltas []DimensionDelta `json:"dimensionDeltas,omitempty"`

	// FindingsResolved son hallazgos presentes en "from" y ausentes en
	// "to" (mejoras); FindingsIntroduced es lo opuesto (regresiones).
	FindingsResolved   []Finding `json:"findingsResolved,omitempty"`
	FindingsIntroduced []Finding `json:"findingsIntroduced,omitempty"`
	FindingsPersisting int       `json:"findingsPersisting"`

	NoChanges bool `json:"noChanges"`
}
