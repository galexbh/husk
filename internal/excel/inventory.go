package excel

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/galexbh/husk/internal/model"
)

// BuildInventoryWorkbook construye el workbook completo de `husk inventory
// --output excel`: una hoja por tipo de recurso y una hoja "Summary" con
// hipervínculos a cada una. highlightRisks y freezeColumns vienen de
// inventory.excel.{highlight_risks,freeze_columns} en la configuración.
func BuildInventoryWorkbook(inv *model.Inventory, freezeColumns int, highlightRisks bool) (*Workbook, error) {
	wb, err := New(freezeColumns, highlightRisks)
	if err != nil {
		return nil, err
	}

	entries := make([]SummaryEntry, 0, 16)
	addSheet := func(label string, sheet Sheet) error {
		if len(sheet.Rows) == 0 {
			entries = append(entries, SummaryEntry{Label: label, Count: 0})
			return nil
		}
		if err := wb.AddSheet(sheet); err != nil {
			return fmt.Errorf("hoja %s: %w", sheet.Name, err)
		}
		entries = append(entries, SummaryEntry{Label: label, Count: len(sheet.Rows), SheetLink: sheet.Name})
		return nil
	}

	if err := addSheet("Deployments", workloadSheet("Deployments", inv.Deployments)); err != nil {
		return nil, err
	}
	if err := addSheet("StatefulSets", workloadSheet("StatefulSets", inv.StatefulSets)); err != nil {
		return nil, err
	}
	if err := addSheet("DaemonSets", workloadSheet("DaemonSets", inv.DaemonSets)); err != nil {
		return nil, err
	}
	if err := addSheet("Services", servicesSheet(inv.Services)); err != nil {
		return nil, err
	}
	if err := addSheet("PVCs", pvcsSheet(inv.PVCs)); err != nil {
		return nil, err
	}
	if err := addSheet("ConfigMaps", configMapsSheet(inv.ConfigMaps)); err != nil {
		return nil, err
	}
	if err := addSheet("Secrets", secretsSheet(inv.Secrets)); err != nil {
		return nil, err
	}
	if err := addSheet("Nodes", nodesSheet(inv.Nodes)); err != nil {
		return nil, err
	}
	if err := addSheet("StorageClasses", storageClassesSheet(inv.StorageClasses)); err != nil {
		return nil, err
	}
	if err := addSheet("CRDs", crdsSheet(inv.CRDs)); err != nil {
		return nil, err
	}
	if err := addSheet("ResourceQuotas", resourceQuotasSheet(inv.ResourceQuotas)); err != nil {
		return nil, err
	}

	if ext := inv.Extended; ext != nil {
		if err := addSheet("Roles", rbacSheet("Roles", ext.Roles)); err != nil {
			return nil, err
		}
		if err := addSheet("RoleBindings", rbacSheet("RoleBindings", ext.RoleBindings)); err != nil {
			return nil, err
		}
		if err := addSheet("ClusterRoles", rbacSheet("ClusterRoles", ext.ClusterRoles)); err != nil {
			return nil, err
		}
		if err := addSheet("ClusterRoleBindings", rbacSheet("ClusterRoleBindings", ext.ClusterRoleBindings)); err != nil {
			return nil, err
		}
		if err := addSheet("NetworkPolicies", networkPoliciesSheet(ext.NetworkPolicies)); err != nil {
			return nil, err
		}
		if err := addSheet("PodDisruptionBudgets", pdbsSheet(ext.PodDisruptionBudgets)); err != nil {
			return nil, err
		}
		if err := addSheet("LimitRanges", limitRangesSheet(ext.LimitRanges)); err != nil {
			return nil, err
		}
		if err := addSheet("HPAs", hpasSheet(ext.HPAs)); err != nil {
			return nil, err
		}
		if err := addSheet("Ingresses", ingressesSheet(ext.Ingresses)); err != nil {
			return nil, err
		}
		if err := addSheet("Routes", routesSheet(ext.Routes)); err != nil {
			return nil, err
		}
	}

	if err := wb.FinalizeSummary("Inventario del cluster — husk", entries); err != nil {
		return nil, err
	}

	return wb, nil
}

func workloadSheet(name string, list []model.WorkloadSummary) Sheet {
	rows := make([][]string, 0, len(list))
	risks := make([]model.RiskLevel, 0, len(list))
	for _, w := range list {
		rows = append(rows, []string{
			w.Namespace, w.Name, strconv.Itoa(int(w.Replicas)), strconv.Itoa(int(w.ReadyReplicas)),
			strconv.Itoa(len(w.Containers)), string(w.Risk),
		})
		risks = append(risks, w.Risk)
	}
	return Sheet{
		Name:     name,
		Headers:  []string{"Namespace", "Nombre", "Réplicas", "Listas", "Contenedores", "Riesgo"},
		Rows:     rows,
		RowRisks: risks,
	}
}

func servicesSheet(list []model.ServiceSummary) Sheet {
	rows := make([][]string, 0, len(list))
	for _, s := range list {
		rows = append(rows, []string{s.Namespace, s.Name, s.Type, s.ClusterIP, strings.Join(s.Ports, ", ")})
	}
	return Sheet{Name: "Services", Headers: []string{"Namespace", "Nombre", "Tipo", "ClusterIP", "Puertos"}, Rows: rows}
}

func pvcsSheet(list []model.PVCSummary) Sheet {
	rows := make([][]string, 0, len(list))
	for _, p := range list {
		rows = append(rows, []string{p.Namespace, p.Name, p.StorageClassName, p.Capacity, p.Phase, p.AccessModes})
	}
	return Sheet{Name: "PVCs", Headers: []string{"Namespace", "Nombre", "StorageClass", "Capacidad", "Fase", "AccessModes"}, Rows: rows}
}

func configMapsSheet(list []model.ConfigMapSummary) Sheet {
	rows := make([][]string, 0, len(list))
	for _, cm := range list {
		rows = append(rows, []string{cm.Namespace, cm.Name, strconv.Itoa(cm.KeysCount)})
	}
	return Sheet{Name: "ConfigMaps", Headers: []string{"Namespace", "Nombre", "Claves"}, Rows: rows}
}

func secretsSheet(list []model.SecretSummary) Sheet {
	rows := make([][]string, 0, len(list))
	for _, s := range list {
		rows = append(rows, []string{s.Namespace, s.Name, s.Type, strconv.Itoa(s.KeysCount)})
	}
	return Sheet{Name: "Secrets", Headers: []string{"Namespace", "Nombre", "Tipo", "Claves"}, Rows: rows}
}

func nodesSheet(list []model.NodeSummary) Sheet {
	rows := make([][]string, 0, len(list))
	risks := make([]model.RiskLevel, 0, len(list))
	for _, n := range list {
		risk := model.RiskGreen
		if !n.Ready || n.Unschedulable {
			risk = model.RiskRed
		}
		rows = append(rows, []string{
			n.Name, strconv.FormatBool(n.Ready), strconv.FormatBool(n.Unschedulable),
			strings.Join(n.Roles, ", "), n.AllocatableCPU, n.AllocatableMem, n.KubeletVersion,
		})
		risks = append(risks, risk)
	}
	return Sheet{
		Name:     "Nodes",
		Headers:  []string{"Nombre", "Ready", "Unschedulable", "Roles", "CPU alloc.", "Mem alloc.", "Kubelet"},
		Rows:     rows,
		RowRisks: risks,
	}
}

func storageClassesSheet(list []model.StorageClassSummary) Sheet {
	rows := make([][]string, 0, len(list))
	for _, sc := range list {
		rows = append(rows, []string{
			sc.Name, sc.Provisioner, sc.ReclaimPolicy, sc.VolumeBindingMode,
			strconv.FormatBool(sc.AllowVolumeExpansion), strconv.FormatBool(sc.IsDefault),
		})
	}
	return Sheet{Name: "StorageClasses", Headers: []string{"Nombre", "Provisioner", "ReclaimPolicy", "BindingMode", "Expansión", "Default"}, Rows: rows}
}

func crdsSheet(list []model.CRDSummary) Sheet {
	rows := make([][]string, 0, len(list))
	for _, crd := range list {
		rows = append(rows, []string{crd.Name, crd.Group, crd.Kind, strings.Join(crd.Versions, ", "), crd.Scope})
	}
	return Sheet{Name: "CRDs", Headers: []string{"Nombre", "Group", "Kind", "Versiones", "Scope"}, Rows: rows}
}

func rbacSheet(name string, list []model.RBACSummary) Sheet {
	rows := make([][]string, 0, len(list))
	for _, r := range list {
		rows = append(rows, []string{r.Namespace, r.Name, r.RoleRef, strings.Join(r.Subjects, ", ")})
	}
	return Sheet{Name: name, Headers: []string{"Namespace", "Nombre", "RoleRef", "Subjects"}, Rows: rows}
}

func networkPoliciesSheet(list []model.NetworkPolicySummary) Sheet {
	rows := make([][]string, 0, len(list))
	for _, p := range list {
		rows = append(rows, []string{p.Namespace, p.Name, strings.Join(p.PolicyTypes, ", ")})
	}
	return Sheet{Name: "NetworkPolicies", Headers: []string{"Namespace", "Nombre", "PolicyTypes"}, Rows: rows}
}

func pdbsSheet(list []model.PDBSummary) Sheet {
	rows := make([][]string, 0, len(list))
	risks := make([]model.RiskLevel, 0, len(list))
	for _, p := range list {
		risk := model.RiskGreen
		if p.CurrentHealthy < p.DesiredHealthy {
			risk = model.RiskRed
		}
		rows = append(rows, []string{
			p.Namespace, p.Name, p.MinAvailable, p.MaxUnavailable,
			fmt.Sprintf("%d/%d", p.CurrentHealthy, p.DesiredHealthy),
		})
		risks = append(risks, risk)
	}
	return Sheet{
		Name:     "PodDisruptionBudgets",
		Headers:  []string{"Namespace", "Nombre", "MinAvailable", "MaxUnavailable", "Healthy actual/deseado"},
		Rows:     rows,
		RowRisks: risks,
	}
}

func resourceQuotasSheet(list []model.ResourceQuotaSummary) Sheet {
	rows := make([][]string, 0, len(list))
	for _, q := range list {
		rows = append(rows, []string{q.Namespace, q.Name, mapToString(q.Hard), mapToString(q.Used)})
	}
	return Sheet{Name: "ResourceQuotas", Headers: []string{"Namespace", "Nombre", "Hard", "Used"}, Rows: rows}
}

func limitRangesSheet(list []model.LimitRangeSummary) Sheet {
	rows := make([][]string, 0, len(list))
	for _, lr := range list {
		rows = append(rows, []string{lr.Namespace, lr.Name, strconv.Itoa(lr.Limits)})
	}
	return Sheet{Name: "LimitRanges", Headers: []string{"Namespace", "Nombre", "Límites definidos"}, Rows: rows}
}

func hpasSheet(list []model.HPASummary) Sheet {
	rows := make([][]string, 0, len(list))
	for _, h := range list {
		rows = append(rows, []string{
			h.Namespace, h.Name, h.Target,
			strconv.Itoa(int(h.MinReplicas)), strconv.Itoa(int(h.MaxReplicas)), strconv.Itoa(int(h.CurrentReplicas)),
		})
	}
	return Sheet{Name: "HPAs", Headers: []string{"Namespace", "Nombre", "Target", "Min", "Max", "Actual"}, Rows: rows}
}

func ingressesSheet(list []model.IngressSummary) Sheet {
	rows := make([][]string, 0, len(list))
	for _, i := range list {
		rows = append(rows, []string{i.Namespace, i.Name, strings.Join(i.Hosts, ", ")})
	}
	return Sheet{Name: "Ingresses", Headers: []string{"Namespace", "Nombre", "Hosts"}, Rows: rows}
}

func routesSheet(list []model.RouteSummary) Sheet {
	rows := make([][]string, 0, len(list))
	for _, r := range list {
		rows = append(rows, []string{r.Namespace, r.Name, r.Host, r.ToService, strconv.FormatBool(r.TLS)})
	}
	return Sheet{Name: "Routes", Headers: []string{"Namespace", "Nombre", "Host", "Servicio", "TLS"}, Rows: rows}
}

func mapToString(m map[string]string) string {
	if len(m) == 0 {
		return ""
	}
	parts := make([]string, 0, len(m))
	for k, v := range m {
		parts = append(parts, k+"="+v)
	}
	return strings.Join(parts, ", ")
}
