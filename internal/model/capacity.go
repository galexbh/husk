package model

import "time"

// NodeCapacity compara lo allocatable de un nodo contra la suma de
// requests de los pods efectivamente programados en él, enriquecido con el
// uso histórico real de Prometheus cuando está disponible.
type NodeCapacity struct {
	Name          string   `json:"name"`
	Roles         []string `json:"roles,omitempty"`
	Ready         bool     `json:"ready"`
	Unschedulable bool     `json:"unschedulable"`
	// Tainted indica que el nodo tiene al menos un taint NoSchedule o
	// NoExecute: se excluye del headroom asignable del cluster y se
	// reporta aparte (ver CapacityReport.ExcludedNodes).
	Tainted bool   `json:"tainted"`
	Zone    string `json:"zone,omitempty"`

	AllocatableCPU    string `json:"allocatableCPU"`
	AllocatableMemory string `json:"allocatableMemory"`
	AllocatablePods   string `json:"allocatablePods"`

	RequestedCPU    string `json:"requestedCPU"`
	RequestedMemory string `json:"requestedMemory"`
	PodCount        int    `json:"podCount"`

	CPUHeadroomPercent    float64 `json:"cpuHeadroomPercent"`
	MemoryHeadroomPercent float64 `json:"memoryHeadroomPercent"`

	// Consumo histórico real (opcional: requiere Thanos Querier
	// disponible). HasMetrics indica si estos campos son significativos.
	ObservedCPUAvg     string `json:"observedCPUAvg,omitempty"`
	ObservedCPUPeak    string `json:"observedCPUPeak,omitempty"`
	ObservedMemoryAvg  string `json:"observedMemoryAvg,omitempty"`
	ObservedMemoryPeak string `json:"observedMemoryPeak,omitempty"`
	HasMetrics         bool   `json:"hasMetrics"`

	Risk RiskLevel `json:"risk"`
	// RiskAxes indica qué eje(s) de headroom dispararon Risk: "cpu",
	// "memory", ambos, o vacío (Risk=green, o el nodo no está Ready/está
	// unschedulable, donde el problema no es de headroom sino de
	// disponibilidad del nodo).
	RiskAxes []string `json:"riskAxes,omitempty"`
}

// ConcentrationRisk señala un workload cuyas réplicas están concentradas en
// muy pocos nodos o zonas (riesgo si ese nodo/zona falla).
type ConcentrationRisk struct {
	Kind          string `json:"kind"`
	Name          string `json:"name"`
	Namespace     string `json:"namespace"`
	Replicas      int32  `json:"replicas"`
	DistinctNodes int    `json:"distinctNodes"`
	DistinctZones int    `json:"distinctZones"`
	Message       string `json:"message"`
}

// CapacityReport es el resultado de `husk capacity nodes`.
type CapacityReport struct {
	Nodes []NodeCapacity `json:"nodes"`
	// ExcludedNodes son los nodos con taints NoSchedule/NoExecute: se
	// listan con sus mismas métricas informativas, pero fuera del cálculo
	// de headroom asignable del cluster.
	ExcludedNodes            []NodeCapacity      `json:"excludedNodes,omitempty"`
	ConcentrationRisks       []ConcentrationRisk `json:"concentrationRisks,omitempty"`
	HeadroomThresholdPercent float64             `json:"headroomThresholdPercent"`
	HasMetrics               bool                `json:"hasMetrics"`
	GeneratedAt              time.Time           `json:"generatedAt"`
}
