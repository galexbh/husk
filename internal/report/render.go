// Package report consume modelos ya poblados (internal/model) y produce
// texto: tablas, markdown o JSON. Nunca llama a Kubernetes, Prometheus ni
// Alertmanager directamente — solo internal/inventory, internal/sizing,
// internal/capacity y internal/dr recolectan datos.
package report

import (
	"fmt"

	"github.com/jedib0t/go-pretty/v6/table"

	"github.com/galexbh/husk/internal/model"
)

// newSectionTable crea una tabla go-pretty con un título de sección. El
// mismo table.Writer se usa tanto para salida "table" (Render) como
// "markdown" (RenderMarkdown).
func newSectionTable(title string) table.Writer {
	t := table.NewWriter()
	t.SetTitle(title)
	t.SetStyle(table.StyleLight)
	return t
}

func workloadsTable(title string, list []model.WorkloadSummary) table.Writer {
	t := newSectionTable(fmt.Sprintf("%s (%d)", title, len(list)))
	t.AppendHeader(table.Row{"Namespace", "Nombre", "Réplicas", "Listas", "Contenedores", "Riesgo"})
	for _, w := range list {
		t.AppendRow(table.Row{w.Namespace, w.Name, w.Replicas, w.ReadyReplicas, len(w.Containers), riskLabel(w.Risk)})
	}
	return t
}

func servicesTable(list []model.ServiceSummary) table.Writer {
	t := newSectionTable(fmt.Sprintf("Services (%d)", len(list)))
	t.AppendHeader(table.Row{"Namespace", "Nombre", "Tipo", "ClusterIP", "Puertos"})
	for _, s := range list {
		t.AppendRow(table.Row{s.Namespace, s.Name, s.Type, s.ClusterIP, joinList(s.Ports)})
	}
	return t
}

func pvcsTable(list []model.PVCSummary) table.Writer {
	t := newSectionTable(fmt.Sprintf("PersistentVolumeClaims (%d)", len(list)))
	t.AppendHeader(table.Row{"Namespace", "Nombre", "StorageClass", "Capacidad", "Fase", "AccessModes"})
	for _, p := range list {
		t.AppendRow(table.Row{p.Namespace, p.Name, p.StorageClassName, p.Capacity, p.Phase, p.AccessModes})
	}
	return t
}

func configMapsTable(list []model.ConfigMapSummary) table.Writer {
	t := newSectionTable(fmt.Sprintf("ConfigMaps (%d)", len(list)))
	t.AppendHeader(table.Row{"Namespace", "Nombre", "Claves"})
	for _, cm := range list {
		t.AppendRow(table.Row{cm.Namespace, cm.Name, cm.KeysCount})
	}
	return t
}

func secretsTable(list []model.SecretSummary) table.Writer {
	t := newSectionTable(fmt.Sprintf("Secrets (%d)", len(list)))
	t.AppendHeader(table.Row{"Namespace", "Nombre", "Tipo", "Claves"})
	for _, s := range list {
		t.AppendRow(table.Row{s.Namespace, s.Name, s.Type, s.KeysCount})
	}
	return t
}

func nodesTable(list []model.NodeSummary) table.Writer {
	t := newSectionTable(fmt.Sprintf("Nodes (%d)", len(list)))
	t.AppendHeader(table.Row{"Nombre", "Ready", "Unschedulable", "Roles", "CPU alloc.", "Mem alloc.", "Kubelet"})
	for _, n := range list {
		t.AppendRow(table.Row{n.Name, n.Ready, n.Unschedulable, joinList(n.Roles), n.AllocatableCPU, n.AllocatableMem, n.KubeletVersion})
	}
	return t
}

func storageClassesTable(list []model.StorageClassSummary) table.Writer {
	t := newSectionTable(fmt.Sprintf("StorageClasses (%d)", len(list)))
	t.AppendHeader(table.Row{"Nombre", "Provisioner", "ReclaimPolicy", "BindingMode", "Expansión", "Default"})
	for _, sc := range list {
		t.AppendRow(table.Row{sc.Name, sc.Provisioner, sc.ReclaimPolicy, sc.VolumeBindingMode, sc.AllowVolumeExpansion, sc.IsDefault})
	}
	return t
}

func crdsTable(list []model.CRDSummary) table.Writer {
	t := newSectionTable(fmt.Sprintf("CustomResourceDefinitions (%d)", len(list)))
	t.AppendHeader(table.Row{"Nombre", "Group", "Kind", "Versiones", "Scope"})
	for _, crd := range list {
		t.AppendRow(table.Row{crd.Name, crd.Group, crd.Kind, joinList(crd.Versions), crd.Scope})
	}
	return t
}

func rbacTable(title string, list []model.RBACSummary) table.Writer {
	t := newSectionTable(fmt.Sprintf("%s (%d)", title, len(list)))
	t.AppendHeader(table.Row{"Namespace", "Nombre", "RoleRef", "Subjects"})
	for _, r := range list {
		t.AppendRow(table.Row{r.Namespace, r.Name, r.RoleRef, joinList(r.Subjects)})
	}
	return t
}

func networkPoliciesTable(list []model.NetworkPolicySummary) table.Writer {
	t := newSectionTable(fmt.Sprintf("NetworkPolicies (%d)", len(list)))
	t.AppendHeader(table.Row{"Namespace", "Nombre", "PolicyTypes"})
	for _, p := range list {
		t.AppendRow(table.Row{p.Namespace, p.Name, joinList(p.PolicyTypes)})
	}
	return t
}

func pdbsTable(list []model.PDBSummary) table.Writer {
	t := newSectionTable(fmt.Sprintf("PodDisruptionBudgets (%d)", len(list)))
	t.AppendHeader(table.Row{"Namespace", "Nombre", "MinAvailable", "MaxUnavailable", "Healthy actual/deseado"})
	for _, p := range list {
		t.AppendRow(table.Row{p.Namespace, p.Name, p.MinAvailable, p.MaxUnavailable, fmt.Sprintf("%d/%d", p.CurrentHealthy, p.DesiredHealthy)})
	}
	return t
}

func resourceQuotasTable(list []model.ResourceQuotaSummary) table.Writer {
	t := newSectionTable(fmt.Sprintf("ResourceQuotas (%d)", len(list)))
	t.AppendHeader(table.Row{"Namespace", "Nombre", "Hard", "Used"})
	for _, q := range list {
		t.AppendRow(table.Row{q.Namespace, q.Name, mapToString(q.Hard), mapToString(q.Used)})
	}
	return t
}

func limitRangesTable(list []model.LimitRangeSummary) table.Writer {
	t := newSectionTable(fmt.Sprintf("LimitRanges (%d)", len(list)))
	t.AppendHeader(table.Row{"Namespace", "Nombre", "Límites definidos"})
	for _, lr := range list {
		t.AppendRow(table.Row{lr.Namespace, lr.Name, lr.Limits})
	}
	return t
}

func hpasTable(list []model.HPASummary) table.Writer {
	t := newSectionTable(fmt.Sprintf("HorizontalPodAutoscalers (%d)", len(list)))
	t.AppendHeader(table.Row{"Namespace", "Nombre", "Target", "Min", "Max", "Actual"})
	for _, h := range list {
		t.AppendRow(table.Row{h.Namespace, h.Name, h.Target, h.MinReplicas, h.MaxReplicas, h.CurrentReplicas})
	}
	return t
}

func ingressesTable(list []model.IngressSummary) table.Writer {
	t := newSectionTable(fmt.Sprintf("Ingresses (%d)", len(list)))
	t.AppendHeader(table.Row{"Namespace", "Nombre", "Hosts"})
	for _, i := range list {
		t.AppendRow(table.Row{i.Namespace, i.Name, joinList(i.Hosts)})
	}
	return t
}

func routesTable(list []model.RouteSummary) table.Writer {
	t := newSectionTable(fmt.Sprintf("Routes (%d)", len(list)))
	t.AppendHeader(table.Row{"Namespace", "Nombre", "Host", "Servicio", "TLS"})
	for _, r := range list {
		t.AppendRow(table.Row{r.Namespace, r.Name, r.Host, r.ToService, r.TLS})
	}
	return t
}

func findingsTable(list []model.Finding) table.Writer {
	t := newSectionTable(fmt.Sprintf("Hallazgos (%d)", len(list)))
	t.AppendHeader(table.Row{"Severidad", "Categoría", "Namespace", "Recurso", "Mensaje"})
	for _, f := range list {
		t.AppendRow(table.Row{riskLabel(f.Severity), f.Category, f.Namespace, f.Resource, f.Message})
	}
	return t
}

// inventoryTables devuelve, en orden, todas las tablas no vacías del
// inventario dado.
func inventoryTables(inv *model.Inventory) []table.Writer {
	tables := make([]table.Writer, 0, 16)
	add := func(t table.Writer, n int) {
		if n > 0 {
			tables = append(tables, t)
		}
	}

	add(workloadsTable("Deployments", inv.Deployments), len(inv.Deployments))
	add(workloadsTable("StatefulSets", inv.StatefulSets), len(inv.StatefulSets))
	add(workloadsTable("DaemonSets", inv.DaemonSets), len(inv.DaemonSets))
	add(servicesTable(inv.Services), len(inv.Services))
	add(pvcsTable(inv.PVCs), len(inv.PVCs))
	add(configMapsTable(inv.ConfigMaps), len(inv.ConfigMaps))
	add(secretsTable(inv.Secrets), len(inv.Secrets))
	add(nodesTable(inv.Nodes), len(inv.Nodes))
	add(storageClassesTable(inv.StorageClasses), len(inv.StorageClasses))
	add(crdsTable(inv.CRDs), len(inv.CRDs))

	if ext := inv.Extended; ext != nil {
		add(rbacTable("Roles", ext.Roles), len(ext.Roles))
		add(rbacTable("RoleBindings", ext.RoleBindings), len(ext.RoleBindings))
		add(rbacTable("ClusterRoles", ext.ClusterRoles), len(ext.ClusterRoles))
		add(rbacTable("ClusterRoleBindings", ext.ClusterRoleBindings), len(ext.ClusterRoleBindings))
		add(networkPoliciesTable(ext.NetworkPolicies), len(ext.NetworkPolicies))
		add(pdbsTable(ext.PodDisruptionBudgets), len(ext.PodDisruptionBudgets))
		add(resourceQuotasTable(ext.ResourceQuotas), len(ext.ResourceQuotas))
		add(limitRangesTable(ext.LimitRanges), len(ext.LimitRanges))
		add(hpasTable(ext.HPAs), len(ext.HPAs))
		add(ingressesTable(ext.Ingresses), len(ext.Ingresses))
		add(routesTable(ext.Routes), len(ext.Routes))
	}

	return tables
}

func summaryTable(s *model.InventorySummary) table.Writer {
	t := newSectionTable("Resumen del inventario")
	t.AppendHeader(table.Row{"Métrica", "Valor"})
	t.AppendRow(table.Row{"Namespaces analizados", s.Namespaces})
	t.AppendRow(table.Row{"Deployments", s.DeploymentsCount})
	t.AppendRow(table.Row{"StatefulSets", s.StatefulSetsCount})
	t.AppendRow(table.Row{"DaemonSets", s.DaemonSetsCount})
	t.AppendRow(table.Row{"Services", s.ServicesCount})
	t.AppendRow(table.Row{"PVCs", s.PVCsCount})
	t.AppendRow(table.Row{"ConfigMaps", s.ConfigMapsCount})
	t.AppendRow(table.Row{"Secrets", s.SecretsCount})
	t.AppendRow(table.Row{"Nodes", s.NodesCount})
	t.AppendRow(table.Row{"StorageClasses", s.StorageClassesCount})
	t.AppendRow(table.Row{"CRDs", s.CRDsCount})
	t.AppendRow(table.Row{"Workloads con una sola réplica", s.SingleReplicaWorkloads})
	t.AppendRow(table.Row{"Workloads sin resources.limits", s.WorkloadsWithoutLimits})
	if s.HasQuotaData {
		t.AppendRow(table.Row{"Namespaces sin ResourceQuota", s.NamespacesWithoutQuota})
	} else {
		t.AppendRow(table.Row{"Namespaces sin ResourceQuota", "no disponible (usa --extended)"})
	}
	return t
}

func sizingTable(w model.WorkloadSizing) table.Writer {
	t := newSectionTable(fmt.Sprintf("%s %s/%s", w.Kind, w.Namespace, w.Name))
	t.AppendHeader(table.Row{"Contenedor", "CPU req/lim", "CPU observada (percentil)", "Mem req/lim", "Mem observada (percentil/pico)", "Veredicto"})
	for _, c := range w.Containers {
		t.AppendRow(table.Row{
			c.Name,
			fmt.Sprintf("%s / %s", orDash(c.CurrentCPURequest), orDash(c.CurrentCPULimit)),
			orDash(c.ObservedCPU),
			fmt.Sprintf("%s / %s", orDash(c.CurrentMemRequest), orDash(c.CurrentMemLimit)),
			fmt.Sprintf("%s / %s", orDash(c.ObservedMemory), orDash(c.ObservedMemoryPeak)),
			verdictLabel(c.Verdict),
		})
	}
	return t
}

// sizingTables devuelve, en orden, las tablas de sizing de los workloads
// con al menos un contenedor.
func sizingTables(r *model.SizingReport) []table.Writer {
	tables := make([]table.Writer, 0, len(r.Workloads))
	for _, w := range r.Workloads {
		if len(w.Containers) > 0 {
			tables = append(tables, sizingTable(w))
		}
	}
	return tables
}

func verdictLabel(v string) string {
	switch v {
	case "sin-limites":
		return "SIN LÍMITES"
	case "sobreaprovisionado":
		return "SOBREAPROVISIONADO"
	case "subaprovisionado":
		return "SUBAPROVISIONADO"
	case "saludable":
		return "SALUDABLE"
	case "sin-datos":
		return "SIN DATOS"
	default:
		return v
	}
}

func orDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

func capacityNodesTable(title string, list []model.NodeCapacity) table.Writer {
	t := newSectionTable(fmt.Sprintf("%s (%d)", title, len(list)))
	t.AppendHeader(table.Row{
		"Nombre", "Ready", "Zona", "CPU alloc.", "CPU pedida", "Headroom CPU", "Mem alloc.", "Mem pedida", "Headroom Mem", "Pods", "Riesgo",
	})
	for _, n := range list {
		t.AppendRow(table.Row{
			n.Name, n.Ready, n.Zone,
			n.AllocatableCPU, n.RequestedCPU, fmt.Sprintf("%.1f%%", n.CPUHeadroomPercent),
			n.AllocatableMemory, n.RequestedMemory, fmt.Sprintf("%.1f%%", n.MemoryHeadroomPercent),
			n.PodCount, riskLabel(n.Risk),
		})
	}
	return t
}

func concentrationRisksTable(list []model.ConcentrationRisk) table.Writer {
	t := newSectionTable(fmt.Sprintf("Riesgos de concentración de carga (%d)", len(list)))
	t.AppendHeader(table.Row{"Kind", "Namespace", "Nombre", "Réplicas", "Nodos distintos", "Zonas distintas", "Mensaje"})
	for _, r := range list {
		t.AppendRow(table.Row{r.Kind, r.Namespace, r.Name, r.Replicas, r.DistinctNodes, r.DistinctZones, r.Message})
	}
	return t
}

// capacityTables devuelve, en orden, todas las tablas no vacías de un
// CapacityReport.
func capacityTables(r *model.CapacityReport) []table.Writer {
	tables := make([]table.Writer, 0, 3)
	if len(r.Nodes) > 0 {
		tables = append(tables, capacityNodesTable("Nodes", r.Nodes))
	}
	if len(r.ExcludedNodes) > 0 {
		tables = append(tables, capacityNodesTable("Nodes excluidos del headroom (taint NoSchedule/NoExecute)", r.ExcludedNodes))
	}
	if len(r.ConcentrationRisks) > 0 {
		tables = append(tables, concentrationRisksTable(r.ConcentrationRisks))
	}
	return tables
}

func drOverviewTable(dr *model.DRReadiness) table.Writer {
	t := newSectionTable("Resumen de DR readiness")
	t.AppendHeader(table.Row{"Chequeo", "Estado"})
	t.AppendRow(table.Row{"OADP instalado", dr.OADPInstalled})
	t.AppendRow(table.Row{"OADP healthy", dr.OADPHealthy})
	if dr.OADPMessage != "" {
		t.AppendRow(table.Row{"OADP detalle", dr.OADPMessage})
	}
	t.AppendRow(table.Row{"BackupStorageLocations", len(dr.BackupStorageLocations)})
	t.AppendRow(table.Row{"Namespaces de aplicación evaluados", len(dr.ApplicationNamespaces)})
	t.AppendRow(table.Row{"Namespaces sin backup", len(dr.NamespacesWithoutBackup)})
	t.AppendRow(table.Row{"Namespaces con backup vencido", len(dr.NamespacesWithStaleBackup)})

	switch {
	case !dr.EtcdCheckApplicable:
		t.AppendRow(table.Row{"Snapshot de etcd", "no aplica (no es OpenShift)"})
	case dr.EtcdSnapshot == nil || !dr.EtcdSnapshot.Verifiable:
		t.AppendRow(table.Row{"Snapshot de etcd", "no verificable"})
	default:
		t.AppendRow(table.Row{"Snapshot de etcd", fmt.Sprintf("%s (última vez: %s, dentro de la antigüedad máxima: %v)", dr.EtcdSnapshot.Source, dr.EtcdSnapshot.LastSuccessTime.Format("2006-01-02 15:04"), dr.EtcdSnapshot.AgeOK)})
	}

	t.AppendRow(table.Row{"StorageClasses usadas sin soporte CSI snapshot", len(dr.StorageClassesWithoutCSISnapshot)})
	t.AppendRow(table.Row{"Workloads críticos evaluados", dr.CriticalWorkloadsTotal})
	t.AppendRow(table.Row{"Workloads críticos sin PDB", len(dr.MissingPDBs)})
	t.AppendRow(table.Row{"Workloads críticos sin topology spread", len(dr.MissingTopologySpread)})
	return t
}

func bslTable(list []model.BSLStatus) table.Writer {
	t := newSectionTable(fmt.Sprintf("BackupStorageLocations (%d)", len(list)))
	t.AppendHeader(table.Row{"Namespace", "Nombre", "Provider", "Fase", "Default"})
	for _, b := range list {
		t.AppendRow(table.Row{b.Namespace, b.Name, b.Provider, b.Phase, b.Default})
	}
	return t
}

func workloadRefsTable(title string, list []model.WorkloadRef) table.Writer {
	t := newSectionTable(fmt.Sprintf("%s (%d)", title, len(list)))
	t.AppendHeader(table.Row{"Kind", "Namespace", "Nombre"})
	for _, r := range list {
		t.AppendRow(table.Row{r.Kind, r.Namespace, r.Name})
	}
	return t
}

// drTables devuelve, en orden, todas las tablas no vacías de un
// DRReadiness.
func drTables(dr *model.DRReadiness) []table.Writer {
	tables := []table.Writer{drOverviewTable(dr)}
	if len(dr.BackupStorageLocations) > 0 {
		tables = append(tables, bslTable(dr.BackupStorageLocations))
	}
	if len(dr.MissingPDBs) > 0 {
		tables = append(tables, workloadRefsTable("Workloads sin PodDisruptionBudget", dr.MissingPDBs))
	}
	if len(dr.MissingTopologySpread) > 0 {
		tables = append(tables, workloadRefsTable("Workloads sin topology spread constraints", dr.MissingTopologySpread))
	}
	if len(dr.Findings) > 0 {
		tables = append(tables, findingsTable(dr.Findings))
	}
	return tables
}

func scoreBreakdownTable(s *model.Score) table.Writer {
	t := newSectionTable(fmt.Sprintf("Score de resiliencia: %.1f/100", s.Total))
	t.AppendHeader(table.Row{"Dimensión", "Score", "Peso configurado", "Peso efectivo", "Disponible", "Hallazgos"})
	for _, d := range s.Breakdown {
		available := "sí"
		scoreStr := fmt.Sprintf("%.1f", d.Score)
		if !d.Available {
			available = "no (" + d.Reason + ")"
			scoreStr = "-"
		}
		t.AppendRow(table.Row{d.Name, scoreStr, fmt.Sprintf("%.0f%%", d.Weight*100), fmt.Sprintf("%.0f%%", d.EffectiveWeight*100), available, len(d.FindingIDs)})
	}
	return t
}

// scoreTables devuelve, en orden, todas las tablas del Score.
func scoreTables(s *model.Score) []table.Writer {
	tables := []table.Writer{scoreBreakdownTable(s)}
	if len(s.Findings) > 0 {
		tables = append(tables, findingsTable(s.Findings))
	}
	return tables
}

func reportCoverTable(r *model.Report) table.Writer {
	t := newSectionTable("husk report generate")
	t.AppendHeader(table.Row{"Campo", "Valor"})
	t.AppendRow(table.Row{"Cluster", r.Meta.ClusterName})
	t.AppendRow(table.Row{"Tipo", clusterTypeLabel(r.Meta.IsOpenShift)})
	t.AppendRow(table.Row{"Versión de Kubernetes/OpenShift", r.Meta.KubernetesVersion})
	t.AppendRow(table.Row{"Versión de husk", r.Meta.HuskVersion})
	t.AppendRow(table.Row{"Generado", r.GeneratedAt.Format("2006-01-02 15:04:05 MST")})
	return t
}

func clusterTypeLabel(isOpenShift bool) string {
	if isOpenShift {
		return "OpenShift"
	}
	return "Kubernetes vanilla"
}

func executiveSummaryTable(s model.ExecutiveSummary) table.Writer {
	t := newSectionTable(fmt.Sprintf("Resumen ejecutivo — score de resiliencia: %.1f/100", s.ScoreTotal))
	t.AppendHeader(table.Row{"Métrica", "Valor"})
	t.AppendRow(table.Row{"Hallazgos críticos", s.CriticalFindings})
	t.AppendRow(table.Row{"Hallazgos de advertencia", s.WarningFindings})
	t.AppendRow(table.Row{"Total de hallazgos", s.TotalFindings})
	t.AppendRow(table.Row{"Namespaces analizados", s.Inventory.Namespaces})
	t.AppendRow(table.Row{"Deployments", s.Inventory.DeploymentsCount})
	t.AppendRow(table.Row{"StatefulSets", s.Inventory.StatefulSetsCount})
	t.AppendRow(table.Row{"DaemonSets", s.Inventory.DaemonSetsCount})
	t.AppendRow(table.Row{"Nodes", s.Inventory.NodesCount})
	return t
}

func recommendationsTable(list []model.Recommendation) table.Writer {
	t := newSectionTable(fmt.Sprintf("Recomendaciones accionables (%d)", len(list)))
	t.AppendHeader(table.Row{"Severidad", "Categoría", "Acción"})
	for _, r := range list {
		t.AppendRow(table.Row{riskLabel(r.Severity), r.Category, r.Action})
	}
	return t
}

// reportTables devuelve, en orden, todas las tablas de un Report: portada,
// resumen ejecutivo, score, hallazgos priorizados, recomendaciones y el
// detalle técnico completo (reutilizando los mismos builders que
// inventory/sizing/capacity/dr por separado).
func reportTables(r *model.Report) []table.Writer {
	tables := []table.Writer{reportCoverTable(r), executiveSummaryTable(r.ExecutiveSummary)}
	tables = append(tables, scoreBreakdownTable(&model.Score{Total: r.ExecutiveSummary.ScoreTotal, Breakdown: r.ExecutiveSummary.ScoreBreakdown}))

	if len(r.PrioritizedFindings) > 0 {
		tables = append(tables, findingsTable(r.PrioritizedFindings))
	}
	if len(r.Recommendations) > 0 {
		tables = append(tables, recommendationsTable(r.Recommendations))
	}

	if r.Detail.Inventory != nil {
		tables = append(tables, inventoryTables(r.Detail.Inventory)...)
	}
	if r.Detail.Sizing != nil {
		tables = append(tables, sizingTables(r.Detail.Sizing)...)
	}
	if r.Detail.Capacity != nil {
		tables = append(tables, capacityTables(r.Detail.Capacity)...)
	}
	if r.Detail.DR != nil {
		tables = append(tables, drTables(r.Detail.DR)...)
	}

	if ac := r.AlertCorrelation; ac != nil {
		tables = append(tables, alertCorrelationTable(ac))
	}

	return tables
}

func alertCorrelationTable(ac *model.AlertCorrelation) table.Writer {
	if !ac.Available {
		t := newSectionTable("Correlación con Alertmanager")
		t.AppendHeader(table.Row{"Estado"})
		t.AppendRow(table.Row{"no disponible: " + ac.Reason})
		return t
	}

	t := newSectionTable(fmt.Sprintf("Alertmanager: %d alertas activas, %d hallazgos son incidente actual", len(ac.ActiveAlerts), len(ac.CorrelatedFindingIDs)))
	t.AppendHeader(table.Row{"Alerta", "Namespace", "Severidad", "Estado", "Resumen"})
	for _, a := range ac.ActiveAlerts {
		t.AppendRow(table.Row{a.Name, a.Namespace, a.Severity, a.State, a.Summary})
	}
	return t
}

func riskLabel(r model.RiskLevel) string {
	switch r {
	case model.RiskRed:
		return "ALTO"
	case model.RiskYellow:
		return "MEDIO"
	case model.RiskGreen:
		return "SALUDABLE"
	default:
		return "-"
	}
}

func joinList(items []string) string {
	if len(items) == 0 {
		return ""
	}
	out := items[0]
	for _, s := range items[1:] {
		out += ", " + s
	}
	return out
}

func mapToString(m map[string]string) string {
	if len(m) == 0 {
		return ""
	}
	out := ""
	for k, v := range m {
		if out != "" {
			out += ", "
		}
		out += k + "=" + v
	}
	return out
}
