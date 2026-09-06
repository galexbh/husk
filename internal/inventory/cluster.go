package inventory

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/galexbh/husk/internal/huskerr"
	"github.com/galexbh/husk/internal/model"
)

// crdGVR identifica a las CustomResourceDefinitions (apiextensions.k8s.io/v1)
// para leerlas vía el cliente dinámico, evitando depender del módulo
// k8s.io/apiextensions-apiserver (ver el comentario en k8sclient.Client).
var crdGVR = schema.GroupVersionResource{Group: "apiextensions.k8s.io", Version: "v1", Resource: "customresourcedefinitions"}

// collectConfigAndSecrets recolecta ConfigMaps y Secrets por namespace. Solo
// metadatos: nunca Data/BinaryData ni los nombres de las claves — ver la
// regla de seguridad de datos sensibles en CLAUDE.md.
func (c *Collector) collectConfigAndSecrets(ctx context.Context, inv *model.Inventory, namespaces []string) error {
	for _, ns := range namespaces {
		configMaps, err := c.client.Kubernetes.CoreV1().ConfigMaps(ns).List(ctx, listOpts)
		if err != nil {
			return huskerr.New("no se pudo listar ConfigMaps en "+ns, "verifica el permiso de lectura sobre configmaps", err)
		}
		for _, cm := range configMaps.Items {
			inv.ConfigMaps = append(inv.ConfigMaps, model.ConfigMapSummary{
				Name:      cm.Name,
				Namespace: cm.Namespace,
				KeysCount: len(cm.Data) + len(cm.BinaryData),
				Labels:    cm.Labels,
			})
		}

		secrets, err := c.client.Kubernetes.CoreV1().Secrets(ns).List(ctx, listOpts)
		if err != nil {
			return huskerr.New("no se pudo listar Secrets en "+ns, "verifica el permiso de lectura sobre secrets", err)
		}
		for _, s := range secrets.Items {
			inv.Secrets = append(inv.Secrets, model.SecretSummary{
				Name:      s.Name,
				Namespace: s.Namespace,
				Type:      string(s.Type),
				KeysCount: len(s.Data),
				Labels:    s.Labels,
			})
		}
	}
	return nil
}

// collectClusterScoped recolecta Nodes y CustomResourceDefinitions, ambos
// cluster-scoped (no aplican filtro de namespace).
func (c *Collector) collectClusterScoped(ctx context.Context, inv *model.Inventory) error {
	nodes, err := c.client.Kubernetes.CoreV1().Nodes().List(ctx, listOpts)
	if err != nil {
		return huskerr.New("no se pudo listar Nodes", "verifica el permiso de lectura sobre nodes", err)
	}
	for _, n := range nodes.Items {
		inv.Nodes = append(inv.Nodes, nodeSummaryFrom(n))
	}

	crds, err := c.client.Dynamic.Resource(crdGVR).List(ctx, listOpts)
	if err != nil {
		return huskerr.New("no se pudo listar CustomResourceDefinitions", "verifica el permiso de lectura sobre customresourcedefinitions", err)
	}
	for _, item := range crds.Items {
		inv.CRDs = append(inv.CRDs, crdSummaryFromUnstructured(item))
	}

	return nil
}

func crdSummaryFromUnstructured(item unstructured.Unstructured) model.CRDSummary {
	name, _, _ := unstructured.NestedString(item.Object, "metadata", "name")
	group, _, _ := unstructured.NestedString(item.Object, "spec", "group")
	kind, _, _ := unstructured.NestedString(item.Object, "spec", "names", "kind")
	scope, _, _ := unstructured.NestedString(item.Object, "spec", "scope")

	versions := make([]string, 0)
	if raw, found, _ := unstructured.NestedSlice(item.Object, "spec", "versions"); found {
		for _, v := range raw {
			if vm, ok := v.(map[string]interface{}); ok {
				if vName, _, _ := unstructured.NestedString(vm, "name"); vName != "" {
					versions = append(versions, vName)
				}
			}
		}
	}

	conditions := make([]string, 0)
	if raw, found, _ := unstructured.NestedSlice(item.Object, "status", "conditions"); found {
		for _, cnd := range raw {
			cm, ok := cnd.(map[string]interface{})
			if !ok {
				continue
			}
			status, _, _ := unstructured.NestedString(cm, "status")
			condType, _, _ := unstructured.NestedString(cm, "type")
			if status == "True" && condType != "" {
				conditions = append(conditions, condType)
			}
		}
	}

	return model.CRDSummary{
		Name:       name,
		Group:      group,
		Kind:       kind,
		Versions:   versions,
		Scope:      scope,
		Conditions: conditions,
	}
}

func nodeSummaryFrom(n corev1.Node) model.NodeSummary {
	ready := false
	for _, cond := range n.Status.Conditions {
		if cond.Type == corev1.NodeReady {
			ready = cond.Status == corev1.ConditionTrue
			break
		}
	}

	roles := make([]string, 0)
	for label := range n.Labels {
		const rolePrefix = "node-role.kubernetes.io/"
		if len(label) > len(rolePrefix) && label[:len(rolePrefix)] == rolePrefix {
			roles = append(roles, label[len(rolePrefix):])
		}
	}

	cpu := n.Status.Allocatable[corev1.ResourceCPU]
	mem := n.Status.Allocatable[corev1.ResourceMemory]
	pods := n.Status.Allocatable[corev1.ResourcePods]

	return model.NodeSummary{
		Name:            n.Name,
		Roles:           roles,
		Ready:           ready,
		Unschedulable:   n.Spec.Unschedulable,
		KubeletVersion:  n.Status.NodeInfo.KubeletVersion,
		AllocatableCPU:  cpu.String(),
		AllocatableMem:  mem.String(),
		AllocatablePods: pods.String(),
		Labels:          n.Labels,
	}
}

// collectResourceQuotas recolecta ResourceQuotas por namespace. A diferencia
// del resto de --extended, esto se recolecta siempre: es un único tipo de
// recurso, barato de listar, que alimenta el hallazgo "namespaces sin
// ResourceQuota" de `inventory summary` sin necesitar el resto del detalle
// extendido (RBAC, NetworkPolicies, PDBs, LimitRanges, HPAs, Ingresses/Routes).
func (c *Collector) collectResourceQuotas(ctx context.Context, inv *model.Inventory, namespaces []string) error {
	for _, ns := range namespaces {
		quotas, err := c.client.Kubernetes.CoreV1().ResourceQuotas(ns).List(ctx, listOpts)
		if err != nil {
			return huskerr.New("no se pudo listar ResourceQuotas en "+ns, "verifica el permiso de lectura sobre resourcequotas", err)
		}
		for _, q := range quotas.Items {
			inv.ResourceQuotas = append(inv.ResourceQuotas, model.ResourceQuotaSummary{
				Name:      q.Name,
				Namespace: q.Namespace,
				Hard:      resourceListToStrings(q.Status.Hard),
				Used:      resourceListToStrings(q.Status.Used),
			})
		}
	}
	return nil
}

// collectExtended recolecta los recursos que solo se piden con --extended y
// no encajan en las demás categorías: PodDisruptionBudgets, LimitRanges,
// HorizontalPodAutoscalers y RBAC (delegado a collectRBAC).
func (c *Collector) collectExtended(ctx context.Context, ext *model.ExtendedInventory, namespaces []string) error {
	for _, ns := range namespaces {
		pdbs, err := c.client.Kubernetes.PolicyV1().PodDisruptionBudgets(ns).List(ctx, listOpts)
		if err != nil {
			return huskerr.New("no se pudo listar PodDisruptionBudgets en "+ns, "verifica el permiso de lectura sobre poddisruptionbudgets", err)
		}
		for _, pdb := range pdbs.Items {
			minAvail, maxUnavail := "", ""
			if pdb.Spec.MinAvailable != nil {
				minAvail = pdb.Spec.MinAvailable.String()
			}
			if pdb.Spec.MaxUnavailable != nil {
				maxUnavail = pdb.Spec.MaxUnavailable.String()
			}
			ext.PodDisruptionBudgets = append(ext.PodDisruptionBudgets, model.PDBSummary{
				Name:           pdb.Name,
				Namespace:      pdb.Namespace,
				MinAvailable:   minAvail,
				MaxUnavailable: maxUnavail,
				CurrentHealthy: pdb.Status.CurrentHealthy,
				DesiredHealthy: pdb.Status.DesiredHealthy,
			})
		}

		limitRanges, err := c.client.Kubernetes.CoreV1().LimitRanges(ns).List(ctx, listOpts)
		if err != nil {
			return huskerr.New("no se pudo listar LimitRanges en "+ns, "verifica el permiso de lectura sobre limitranges", err)
		}
		for _, lr := range limitRanges.Items {
			ext.LimitRanges = append(ext.LimitRanges, model.LimitRangeSummary{
				Name:      lr.Name,
				Namespace: lr.Namespace,
				Limits:    len(lr.Spec.Limits),
			})
		}

		hpas, err := c.client.Kubernetes.AutoscalingV2().HorizontalPodAutoscalers(ns).List(ctx, listOpts)
		if err != nil {
			return huskerr.New("no se pudo listar HorizontalPodAutoscalers en "+ns, "verifica el permiso de lectura sobre horizontalpodautoscalers", err)
		}
		for _, hpa := range hpas.Items {
			target := hpa.Spec.ScaleTargetRef.Kind + "/" + hpa.Spec.ScaleTargetRef.Name
			minReplicas := int32(1)
			if hpa.Spec.MinReplicas != nil {
				minReplicas = *hpa.Spec.MinReplicas
			}
			ext.HPAs = append(ext.HPAs, model.HPASummary{
				Name:            hpa.Name,
				Namespace:       hpa.Namespace,
				Target:          target,
				MinReplicas:     minReplicas,
				MaxReplicas:     hpa.Spec.MaxReplicas,
				CurrentReplicas: hpa.Status.CurrentReplicas,
			})
		}
	}

	return c.collectRBAC(ctx, ext, namespaces)
}

func resourceListToStrings(list corev1.ResourceList) map[string]string {
	out := make(map[string]string, len(list))
	for k, v := range list {
		out[string(k)] = v.String()
	}
	return out
}
