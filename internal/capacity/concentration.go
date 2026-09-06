package capacity

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"

	"github.com/galexbh/husk/internal/huskerr"
	"github.com/galexbh/husk/internal/model"
)

// detectConcentration busca Deployments/StatefulSets con más de una
// réplica cuyos pods, en la práctica, corren todos en el mismo nodo o en
// una sola zona de disponibilidad — un riesgo que el número de réplicas
// por sí solo no muestra. DaemonSets quedan fuera: distribuirse en todos
// los nodos elegibles es justamente su diseño.
func (a *Analyzer) detectConcentration(ctx context.Context, pods []corev1.Pod, nodes []corev1.Node) ([]model.ConcentrationRisk, error) {
	nodeZone := make(map[string]string, len(nodes))
	zones := make(map[string]bool)
	for _, n := range nodes {
		zone := n.Labels[zoneLabel]
		if zone == "" {
			zone = n.Labels[failureDomainZoneLabel]
		}
		if zone != "" {
			nodeZone[n.Name] = zone
			zones[zone] = true
		}
	}
	multiZoneCluster := len(zones) > 1

	deployments, err := a.client.Kubernetes.AppsV1().Deployments("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, huskerr.New("no se pudo listar Deployments", "verifica el permiso de lectura sobre deployments", err)
	}
	statefulSets, err := a.client.Kubernetes.AppsV1().StatefulSets("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, huskerr.New("no se pudo listar StatefulSets", "verifica el permiso de lectura sobre statefulsets", err)
	}

	var risks []model.ConcentrationRisk
	for _, d := range deployments.Items {
		if r := checkWorkloadConcentration("Deployment", d.Namespace, d.Name, d.Spec.Replicas, d.Spec.Selector, pods, nodeZone, multiZoneCluster); r != nil {
			risks = append(risks, *r)
		}
	}
	for _, s := range statefulSets.Items {
		if r := checkWorkloadConcentration("StatefulSet", s.Namespace, s.Name, s.Spec.Replicas, s.Spec.Selector, pods, nodeZone, multiZoneCluster); r != nil {
			risks = append(risks, *r)
		}
	}

	return risks, nil
}

func checkWorkloadConcentration(
	kind, namespace, name string,
	replicas *int32,
	selector *metav1.LabelSelector,
	pods []corev1.Pod,
	nodeZone map[string]string,
	multiZoneCluster bool,
) *model.ConcentrationRisk {
	desired := int32(1)
	if replicas != nil {
		desired = *replicas
	}
	if desired <= 1 {
		return nil
	}

	sel, err := metav1.LabelSelectorAsSelector(selector)
	if err != nil || sel.Empty() {
		return nil
	}

	nodesSeen := make(map[string]bool)
	zonesSeen := make(map[string]bool)
	matched := 0
	for _, p := range pods {
		if p.Namespace != namespace || p.Spec.NodeName == "" {
			continue
		}
		if !sel.Matches(labels.Set(p.Labels)) {
			continue
		}
		matched++
		nodesSeen[p.Spec.NodeName] = true
		if zone, ok := nodeZone[p.Spec.NodeName]; ok {
			zonesSeen[zone] = true
		}
	}

	if matched <= 1 {
		return nil
	}

	switch {
	case len(nodesSeen) == 1:
		return &model.ConcentrationRisk{
			Kind: kind, Name: name, Namespace: namespace, Replicas: desired,
			DistinctNodes: 1, DistinctZones: len(zonesSeen),
			Message: fmt.Sprintf("%s %s/%s: sus %d réplicas corren en el mismo nodo", kind, namespace, name, matched),
		}
	case multiZoneCluster && len(zonesSeen) == 1:
		return &model.ConcentrationRisk{
			Kind: kind, Name: name, Namespace: namespace, Replicas: desired,
			DistinctNodes: len(nodesSeen), DistinctZones: 1,
			Message: fmt.Sprintf("%s %s/%s: sus %d réplicas corren en una sola zona de disponibilidad", kind, namespace, name, matched),
		}
	default:
		return nil
	}
}
