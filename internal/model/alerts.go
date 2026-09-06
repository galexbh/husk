package model

import "time"

// Alert es una alerta activa de Alertmanager, ya normalizada desde su
// formato de API v2 (labels/annotations) a los campos que husk necesita
// para correlacionar.
type Alert struct {
	Name      string    `json:"name"`
	Namespace string    `json:"namespace,omitempty"`
	Pod       string    `json:"pod,omitempty"`
	Container string    `json:"container,omitempty"`
	Severity  string    `json:"severity,omitempty"`
	State     string    `json:"state"` // active|suppressed|unprocessed
	StartsAt  time.Time `json:"startsAt,omitempty"`
	Summary   string    `json:"summary,omitempty"`
}

// AlertCorrelation es el resultado de cruzar los hallazgos de sizing y
// capacity con las alertas activas de Alertmanager: distingue un
// incidente actual (hallazgo con una alerta activa relacionada) de un
// riesgo preventivo (hallazgo sin ninguna alerta activa todavía).
//
// Available es false cuando Alertmanager no está accesible (Kubernetes
// vanilla, ruta inalcanzable, etc.): el reporte se genera igual, sin esta
// sección — nunca falla por esto.
type AlertCorrelation struct {
	Available    bool    `json:"available"`
	Reason       string  `json:"reason,omitempty"`
	ActiveAlerts []Alert `json:"activeAlerts,omitempty"`
	// CorrelatedFindingIDs son los Finding.ID que tienen al menos una
	// alerta activa relacionada en su mismo namespace.
	CorrelatedFindingIDs []string `json:"correlatedFindingIds,omitempty"`
}
