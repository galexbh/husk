package dr

import "github.com/galexbh/husk/internal/model"

// workloadsMissingTopologySpread devuelve los workloads críticos cuyo pod
// template no declara ningún topology spread constraint — sin eso, sus
// réplicas pueden terminar todas en el mismo nodo o zona, sin que el
// scheduler tenga ninguna razón para distribuirlas.
func workloadsMissingTopologySpread(workloads []criticalWorkload) []model.WorkloadRef {
	var missing []model.WorkloadRef
	for _, w := range workloads {
		if len(w.TopologySpreadConstraints) == 0 {
			missing = append(missing, model.WorkloadRef{Kind: w.Kind, Namespace: w.Namespace, Name: w.Name})
		}
	}
	return missing
}
