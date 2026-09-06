package model

import "time"

// ContainerSizing compara los requests/limits actuales de un contenedor
// contra su consumo histórico real (CPU P95, memoria P99 y pico), con una
// recomendación cuando el veredicto no es "saludable" ni "sin-datos".
//
// Verdict es uno de: "sin-datos", "sin-limites", "sobreaprovisionado",
// "subaprovisionado", "saludable".
type ContainerSizing struct {
	Name string `json:"name"`

	CurrentCPURequest string `json:"currentCPURequest,omitempty"`
	CurrentCPULimit   string `json:"currentCPULimit,omitempty"`
	CurrentMemRequest string `json:"currentMemoryRequest,omitempty"`
	CurrentMemLimit   string `json:"currentMemoryLimit,omitempty"`

	ObservedCPU        string `json:"observedCPU,omitempty"`
	ObservedMemory     string `json:"observedMemory,omitempty"`
	ObservedMemoryPeak string `json:"observedMemoryPeak,omitempty"`
	HasData            bool   `json:"hasData"`

	RecommendedCPURequest string `json:"recommendedCPURequest,omitempty"`
	RecommendedCPULimit   string `json:"recommendedCPULimit,omitempty"`
	RecommendedMemRequest string `json:"recommendedMemoryRequest,omitempty"`
	RecommendedMemLimit   string `json:"recommendedMemoryLimit,omitempty"`

	Verdict string    `json:"verdict"`
	Risk    RiskLevel `json:"risk"`
}

// WorkloadSizing agrupa el sizing de todos los contenedores de un workload.
type WorkloadSizing struct {
	Kind       string            `json:"kind"`
	Name       string            `json:"name"`
	Namespace  string            `json:"namespace"`
	Containers []ContainerSizing `json:"containers"`
}

// SizingReport es el resultado de `husk sizing report`.
type SizingReport struct {
	Workloads        []WorkloadSizing `json:"workloads"`
	Lookback         string           `json:"lookback"`
	CPUPercentile    float64          `json:"cpuPercentile"`
	MemoryPercentile float64          `json:"memoryPercentile"`
	GeneratedAt      time.Time        `json:"generatedAt"`
}

// DryRunPatch es el patch YAML sugerido para un contenedor (ver
// internal/sizing.BuildDryRunPatches). --dry-run es estrictamente local:
// husk nunca lo aplica al cluster, nunca abre Pull Requests y no tiene
// ningún efecto externo.
type DryRunPatch struct {
	Kind      string `json:"kind"`
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
	Container string `json:"container"`
	Verdict   string `json:"verdict"`
	YAML      string `json:"yaml"`
}
