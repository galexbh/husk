package inventory

import (
	"fmt"

	"github.com/galexbh/husk/internal/model"
)

// BuildSummary calcula la versión ejecutiva de un inventario ya
// recolectado: conteos agregados y hallazgos de riesgo. Es una función pura
// (sin I/O), separada de Collect para que sea trivialmente testeable.
// namespaces es la lista de namespaces efectivamente analizados (tras el
// filtro de exclusión), usada para el hallazgo de namespaces sin
// ResourceQuota.
func BuildSummary(inv *model.Inventory, namespaces []string) *model.InventorySummary {
	s := &model.InventorySummary{
		Namespaces:          len(namespaces),
		DeploymentsCount:    len(inv.Deployments),
		StatefulSetsCount:   len(inv.StatefulSets),
		DaemonSetsCount:     len(inv.DaemonSets),
		ServicesCount:       len(inv.Services),
		PVCsCount:           len(inv.PVCs),
		ConfigMapsCount:     len(inv.ConfigMaps),
		SecretsCount:        len(inv.Secrets),
		NodesCount:          len(inv.Nodes),
		StorageClassesCount: len(inv.StorageClasses),
		CRDsCount:           len(inv.CRDs),
	}

	findings := make([]model.Finding, 0)

	scalableWorkloads := make([]model.WorkloadSummary, 0, len(inv.Deployments)+len(inv.StatefulSets))
	scalableWorkloads = append(scalableWorkloads, inv.Deployments...)
	scalableWorkloads = append(scalableWorkloads, inv.StatefulSets...)

	for _, w := range scalableWorkloads {
		if w.Replicas <= 1 {
			s.SingleReplicaWorkloads++
			findings = append(findings, model.NewFinding(
				model.RiskRed, "single-replica", w.Namespace, w.Kind+"/"+w.Name,
				fmt.Sprintf("%s %s/%s corre con una sola réplica", w.Kind, w.Namespace, w.Name),
			))
		}
	}

	allWorkloads := make([]model.WorkloadSummary, 0, len(scalableWorkloads)+len(inv.DaemonSets))
	allWorkloads = append(allWorkloads, scalableWorkloads...)
	allWorkloads = append(allWorkloads, inv.DaemonSets...)

	for _, w := range allWorkloads {
		missing := false
		for _, ctr := range w.Containers {
			if !ctr.HasLimits {
				missing = true
				break
			}
		}
		if missing {
			s.WorkloadsWithoutLimits++
			findings = append(findings, model.NewFinding(
				model.RiskRed, "missing-limits", w.Namespace, w.Kind+"/"+w.Name,
				fmt.Sprintf("%s %s/%s tiene contenedores sin resources.limits definidos", w.Kind, w.Namespace, w.Name),
			))
		}
	}

	if inv.Extended != nil {
		s.HasQuotaData = true
		withQuota := make(map[string]bool, len(inv.Extended.ResourceQuotas))
		for _, q := range inv.Extended.ResourceQuotas {
			withQuota[q.Namespace] = true
		}
		for _, ns := range namespaces {
			if !withQuota[ns] {
				s.NamespacesWithoutQuota++
				findings = append(findings, model.NewFinding(
					model.RiskYellow, "missing-resourcequota", ns, "",
					fmt.Sprintf("el namespace %s no tiene ningún ResourceQuota", ns),
				))
			}
		}
	}

	s.Findings = findings
	return s
}
