// Package dr evalúa la preparación de disaster recovery: OADP/Velero,
// antigüedad de backups, BackupStorageLocations, soporte CSI snapshot,
// antigüedad del snapshot de etcd, y PodDisruptionBudgets/topology spread
// constraints en workloads críticos. Consume internal/k8sclient
// directamente para recolectar (CRs de Velero/OADP vía cliente dinámico,
// Nodes/Pods/PVCs vía clientset tipado) pero no genera salida visual — eso
// vive en internal/report.
package dr

import (
	"context"
	"time"

	"github.com/galexbh/husk/internal/config"
	"github.com/galexbh/husk/internal/k8sclient"
	"github.com/galexbh/husk/internal/model"
	"github.com/galexbh/husk/internal/nsfilter"
)

// oadpNamespace es el namespace donde vive el operador OADP y, típicamente,
// las CRs de Velero. Siempre se evalúa aunque coincida con los patrones de
// exclusión (excepción documentada en CLAUDE.md).
const oadpNamespace = "openshift-adp"

// Assessor produce un model.DRReadiness.
type Assessor struct {
	client *k8sclient.Client
	filter *nsfilter.Filter
	cfg    config.DRConfig
}

// New construye un Assessor. filter aplica la política de exclusión de
// namespaces al evaluar cobertura de backups (openshift-adp siempre se
// evalúa aparte, sin importar el filtro).
func New(client *k8sclient.Client, filter *nsfilter.Filter, cfg config.DRConfig) *Assessor {
	return &Assessor{client: client, filter: filter, cfg: cfg}
}

// Assess ejecuta la evaluación completa de DR readiness.
func (a *Assessor) Assess(ctx context.Context) (*model.DRReadiness, error) {
	dr := &model.DRReadiness{
		IsOpenShift: a.client.IsOpenShift,
		GeneratedAt: time.Now(),
	}

	namespaces, err := a.applicationNamespaces(ctx)
	if err != nil {
		return nil, err
	}
	dr.ApplicationNamespaces = namespaces

	if err := a.assessOADP(ctx, dr); err != nil {
		return nil, err
	}
	if err := a.assessVelero(ctx, dr); err != nil {
		return nil, err
	}
	a.assessBackupCoverage(dr)

	if err := a.assessEtcdSnapshot(ctx, dr); err != nil {
		return nil, err
	}
	if err := a.assessStorageClasses(ctx, dr); err != nil {
		return nil, err
	}
	if err := a.assessPDBAndTopology(ctx, dr); err != nil {
		return nil, err
	}

	dr.Findings = collectFindings(dr)
	return dr, nil
}

// applicationNamespaces resuelve los namespaces de aplicación a evaluar
// para cobertura de backup: todos los namespaces del cluster tras el
// filtro de exclusión, sin incluir openshift-adp (que se evalúa aparte).
func (a *Assessor) applicationNamespaces(ctx context.Context) ([]string, error) {
	list, err := a.client.Kubernetes.CoreV1().Namespaces().List(ctx, listOpts)
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(list.Items))
	for _, ns := range list.Items {
		if ns.Name == oadpNamespace {
			continue
		}
		names = append(names, ns.Name)
	}
	return a.filter.Apply(names), nil
}
