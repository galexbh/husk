package model

import "strings"

// Finding es un hallazgo de riesgo genérico, reutilizado por inventory
// summary, dr assess, score y report generate.
//
// ID es un identificador estable (determinista para la misma entrada:
// mismo recurso + misma categoría) para que `husk score` pueda enlazar cada
// punto restado a un hallazgo concreto, y para que `report diff` pueda
// reconocer el "mismo" hallazgo entre dos snapshots.
type Finding struct {
	ID        string    `json:"id,omitempty"`
	Severity  RiskLevel `json:"severity"`
	Category  string    `json:"category"`
	Message   string    `json:"message"`
	Namespace string    `json:"namespace,omitempty"`
	Resource  string    `json:"resource,omitempty"`
}

// NewFinding construye un Finding con un ID estable derivado de category,
// namespace y resource: la misma combinación produce siempre el mismo ID,
// independientemente de en qué orden se recolecten los hallazgos.
func NewFinding(severity RiskLevel, category, namespace, resource, message string) Finding {
	id := category
	if namespace != "" {
		id += ":" + namespace
	}
	if resource != "" {
		id += "/" + resource
	}
	return Finding{
		ID:        strings.ToLower(id),
		Severity:  severity,
		Category:  category,
		Message:   message,
		Namespace: namespace,
		Resource:  resource,
	}
}
