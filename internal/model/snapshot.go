// Package model define el contrato compartido de husk: ClusterSnapshot y
// los tipos que lo componen. Este paquete no realiza I/O ni depende de
// ningún otro paquete de husk — es el único paquete que todos los demás
// pueden importar con seguridad.
package model

// SchemaVersion identifica la forma de ClusterSnapshot. Se incrementa cada
// vez que se agrega, renombra o quita un campo, para que `report diff` y el
// historial local (~/.husk/history/) puedan detectar snapshots
// incompatibles entre versiones de husk.
const SchemaVersion = "1"

// ClusterSnapshot es el modelo único que comparten todos los reportes de
// husk, el historial local (~/.husk/history/) y `report diff`. Cada
// comando de análisis llena solo la sección que produce; el resto queda
// nil. `report generate` es el único comando que las llena todas.
type ClusterSnapshot struct {
	SchemaVersion string `json:"schemaVersion"`
	Meta          Meta   `json:"meta"`

	Inventory        *Inventory        `json:"inventory,omitempty"`
	InventorySummary *InventorySummary `json:"inventorySummary,omitempty"`
	Sizing           *SizingReport     `json:"sizing,omitempty"`
	Capacity         *CapacityReport   `json:"capacity,omitempty"`
	DR               *DRReadiness      `json:"dr,omitempty"`
	Score            *Score            `json:"score,omitempty"`
}

// NewSnapshot crea un snapshot vacío con la versión de esquema actual y los
// metadatos dados.
func NewSnapshot(meta Meta) *ClusterSnapshot {
	return &ClusterSnapshot{SchemaVersion: SchemaVersion, Meta: meta}
}
