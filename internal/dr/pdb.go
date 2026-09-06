package dr

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"

	"github.com/galexbh/husk/internal/huskerr"
	"github.com/galexbh/husk/internal/model"
)

// criticalWorkload es un Deployment o StatefulSet con más de una réplica:
// el conjunto que se evalúa para PodDisruptionBudgets y topology spread
// constraints. Los DaemonSets quedan fuera: ya se distribuyen por diseño
// en todos los nodos elegibles.
type criticalWorkload struct {
	Kind                      string
	Namespace                 string
	Name                      string
	TemplateLabels            map[string]string
	TopologySpreadConstraints []corev1.TopologySpreadConstraint
}

// assessPDBAndTopology identifica los workloads críticos del cluster y
// reporta cuáles no tienen un PodDisruptionBudget que los cubra y cuáles
// no declaran topology spread constraints.
func (a *Assessor) assessPDBAndTopology(ctx context.Context, dr *model.DRReadiness) error {
	workloads, err := a.criticalWorkloads(ctx, dr.ApplicationNamespaces)
	if err != nil {
		return err
	}
	dr.CriticalWorkloadsTotal = len(workloads)

	pdbsByNamespace, err := a.pdbsByNamespace(ctx, dr.ApplicationNamespaces)
	if err != nil {
		return err
	}

	for _, w := range workloads {
		if !hasMatchingPDB(w, pdbsByNamespace[w.Namespace]) {
			dr.MissingPDBs = append(dr.MissingPDBs, model.WorkloadRef{Kind: w.Kind, Namespace: w.Namespace, Name: w.Name})
		}
	}
	dr.MissingTopologySpread = workloadsMissingTopologySpread(workloads)

	return nil
}

func (a *Assessor) criticalWorkloads(ctx context.Context, namespaces []string) ([]criticalWorkload, error) {
	var out []criticalWorkload

	for _, ns := range namespaces {
		deployments, err := a.client.Kubernetes.AppsV1().Deployments(ns).List(ctx, listOpts)
		if err != nil {
			return nil, huskerr.New("no se pudo listar Deployments en "+ns, "verifica el permiso de lectura sobre deployments", err)
		}
		for _, d := range deployments.Items {
			if isCriticalReplicaCount(d.Spec.Replicas) {
				out = append(out, criticalWorkload{
					Kind: "Deployment", Namespace: d.Namespace, Name: d.Name,
					TemplateLabels:            d.Spec.Template.Labels,
					TopologySpreadConstraints: d.Spec.Template.Spec.TopologySpreadConstraints,
				})
			}
		}

		statefulSets, err := a.client.Kubernetes.AppsV1().StatefulSets(ns).List(ctx, listOpts)
		if err != nil {
			return nil, huskerr.New("no se pudo listar StatefulSets en "+ns, "verifica el permiso de lectura sobre statefulsets", err)
		}
		for _, s := range statefulSets.Items {
			if isCriticalReplicaCount(s.Spec.Replicas) {
				out = append(out, criticalWorkload{
					Kind: "StatefulSet", Namespace: s.Namespace, Name: s.Name,
					TemplateLabels:            s.Spec.Template.Labels,
					TopologySpreadConstraints: s.Spec.Template.Spec.TopologySpreadConstraints,
				})
			}
		}
	}

	return out, nil
}

func isCriticalReplicaCount(replicas *int32) bool {
	return replicas != nil && *replicas > 1
}

func (a *Assessor) pdbsByNamespace(ctx context.Context, namespaces []string) (map[string][]policyv1.PodDisruptionBudget, error) {
	out := make(map[string][]policyv1.PodDisruptionBudget, len(namespaces))
	for _, ns := range namespaces {
		pdbs, err := a.client.Kubernetes.PolicyV1().PodDisruptionBudgets(ns).List(ctx, listOpts)
		if err != nil {
			return nil, huskerr.New("no se pudo listar PodDisruptionBudgets en "+ns, "verifica el permiso de lectura sobre poddisruptionbudgets", err)
		}
		out[ns] = pdbs.Items
	}
	return out, nil
}

// hasMatchingPDB reporta si algún PDB del namespace cubre a w, comparando
// su selector contra las labels del pod template de w.
func hasMatchingPDB(w criticalWorkload, pdbs []policyv1.PodDisruptionBudget) bool {
	for _, pdb := range pdbs {
		sel, err := metav1.LabelSelectorAsSelector(pdb.Spec.Selector)
		if err != nil || sel.Empty() {
			continue
		}
		if sel.Matches(labels.Set(w.TemplateLabels)) {
			return true
		}
	}
	return false
}
