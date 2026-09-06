package inventory

import (
	"context"

	corev1 "k8s.io/api/core/v1"

	"github.com/galexbh/husk/internal/huskerr"
	"github.com/galexbh/husk/internal/model"
)

// collectWorkloads recolecta Deployments, StatefulSets y DaemonSets en los
// namespaces dados.
func (c *Collector) collectWorkloads(ctx context.Context, inv *model.Inventory, namespaces []string) error {
	for _, ns := range namespaces {
		deployments, err := c.client.Kubernetes.AppsV1().Deployments(ns).List(ctx, listOpts)
		if err != nil {
			return huskerr.New("no se pudo listar Deployments en "+ns, "verifica el permiso de lectura sobre deployments", err)
		}
		for _, d := range deployments.Items {
			replicas := int32(1)
			if d.Spec.Replicas != nil {
				replicas = *d.Spec.Replicas
			}
			inv.Deployments = append(inv.Deployments, buildWorkloadSummary(
				"Deployment", d.Name, d.Namespace, replicas, d.Status.ReadyReplicas,
				d.Spec.Template.Spec.Containers, d.Labels, true,
			))
		}

		statefulSets, err := c.client.Kubernetes.AppsV1().StatefulSets(ns).List(ctx, listOpts)
		if err != nil {
			return huskerr.New("no se pudo listar StatefulSets en "+ns, "verifica el permiso de lectura sobre statefulsets", err)
		}
		for _, s := range statefulSets.Items {
			replicas := int32(1)
			if s.Spec.Replicas != nil {
				replicas = *s.Spec.Replicas
			}
			inv.StatefulSets = append(inv.StatefulSets, buildWorkloadSummary(
				"StatefulSet", s.Name, s.Namespace, replicas, s.Status.ReadyReplicas,
				s.Spec.Template.Spec.Containers, s.Labels, true,
			))
		}

		daemonSets, err := c.client.Kubernetes.AppsV1().DaemonSets(ns).List(ctx, listOpts)
		if err != nil {
			return huskerr.New("no se pudo listar DaemonSets en "+ns, "verifica el permiso de lectura sobre daemonsets", err)
		}
		for _, ds := range daemonSets.Items {
			// Un DaemonSet corre un pod por nodo elegible: el número de
			// réplicas no es un parámetro que el usuario configure, así que
			// el riesgo de "una sola réplica" no aplica.
			inv.DaemonSets = append(inv.DaemonSets, buildWorkloadSummary(
				"DaemonSet", ds.Name, ds.Namespace, ds.Status.DesiredNumberScheduled, ds.Status.NumberReady,
				ds.Spec.Template.Spec.Containers, ds.Labels, false,
			))
		}
	}
	return nil
}

// buildWorkloadSummary construye un model.WorkloadSummary y calcula su
// riesgo. checkSingleReplica controla si "una sola réplica" cuenta como
// riesgo para este tipo de workload (no aplica a DaemonSets).
func buildWorkloadSummary(kind, name, namespace string, replicas, ready int32, containers []corev1.Container, labels map[string]string, checkSingleReplica bool) model.WorkloadSummary {
	containerSummaries := make([]model.ContainerSummary, 0, len(containers))
	anyMissingLimits := false
	for _, ctr := range containers {
		cs := containerSummaryFrom(ctr)
		if !cs.HasLimits {
			anyMissingLimits = true
		}
		containerSummaries = append(containerSummaries, cs)
	}

	risk := model.RiskGreen
	if anyMissingLimits || (checkSingleReplica && replicas <= 1) {
		risk = model.RiskRed
	}

	return model.WorkloadSummary{
		Kind:          kind,
		Name:          name,
		Namespace:     namespace,
		Replicas:      replicas,
		ReadyReplicas: ready,
		Containers:    containerSummaries,
		Labels:        labels,
		Risk:          risk,
	}
}

func containerSummaryFrom(ctr corev1.Container) model.ContainerSummary {
	cpuReq := ctr.Resources.Requests.Cpu()
	memReq := ctr.Resources.Requests.Memory()
	cpuLim := ctr.Resources.Limits.Cpu()
	memLim := ctr.Resources.Limits.Memory()

	hasCPULimit := cpuLim != nil && !cpuLim.IsZero()
	hasMemLimit := memLim != nil && !memLim.IsZero()

	cs := model.ContainerSummary{
		Name:      ctr.Name,
		Image:     ctr.Image,
		HasLimits: hasCPULimit && hasMemLimit,
	}
	if cpuReq != nil && !cpuReq.IsZero() {
		cs.CPURequest = cpuReq.String()
	}
	if memReq != nil && !memReq.IsZero() {
		cs.MemoryRequest = memReq.String()
	}
	if hasCPULimit {
		cs.CPULimit = cpuLim.String()
	}
	if hasMemLimit {
		cs.MemoryLimit = memLim.String()
	}
	return cs
}
