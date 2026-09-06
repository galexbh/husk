package dr

import (
	"fmt"

	"github.com/galexbh/husk/internal/model"
)

// collectFindings consolida todos los hallazgos de una evaluación de DR ya
// completa en una sola lista, para report generate y husk score.
func collectFindings(dr *model.DRReadiness) []model.Finding {
	findings := make([]model.Finding, 0)

	switch {
	case !dr.OADPInstalled:
		findings = append(findings, model.NewFinding(model.RiskRed, "oadp-not-installed", oadpNamespace, "", dr.OADPMessage))
	case !dr.OADPHealthy:
		findings = append(findings, model.NewFinding(model.RiskRed, "oadp-unhealthy", oadpNamespace, "", dr.OADPMessage))
	}

	for _, ns := range dr.NamespacesWithoutBackup {
		findings = append(findings, model.NewFinding(
			model.RiskRed, "missing-backup", ns, "",
			fmt.Sprintf("el namespace %s no tiene ningún backup completado que lo incluya", ns),
		))
	}
	for _, ns := range dr.NamespacesWithStaleBackup {
		findings = append(findings, model.NewFinding(
			model.RiskYellow, "stale-backup", ns, "",
			fmt.Sprintf("el backup más reciente de %s supera la antigüedad máxima configurada", ns),
		))
	}

	if dr.EtcdCheckApplicable && dr.EtcdSnapshot != nil {
		switch {
		case !dr.EtcdSnapshot.Verifiable:
			findings = append(findings, model.NewFinding(
				model.RiskYellow, "etcd-snapshot-unverifiable", "", "",
				"no se encontró evidencia de un Job/CronJob de backup de etcd; verifica manualmente la antigüedad del último snapshot",
			))
		case !dr.EtcdSnapshot.AgeOK:
			findings = append(findings, model.NewFinding(
				model.RiskRed, "etcd-snapshot-stale", "", dr.EtcdSnapshot.Source,
				fmt.Sprintf("el último snapshot de etcd (%s) supera la antigüedad máxima configurada", dr.EtcdSnapshot.Source),
			))
		}
	}

	for _, sc := range dr.StorageClassesWithoutCSISnapshot {
		findings = append(findings, model.NewFinding(
			model.RiskYellow, "csi-snapshot-unsupported", "", sc,
			fmt.Sprintf("la StorageClass %s no tiene un VolumeSnapshotClass que soporte CSI snapshot", sc),
		))
	}

	for _, ref := range dr.MissingPDBs {
		resource := ref.Kind + "/" + ref.Name
		findings = append(findings, model.NewFinding(
			model.RiskYellow, "missing-pdb", ref.Namespace, resource,
			fmt.Sprintf("%s %s/%s no tiene un PodDisruptionBudget que lo cubra", ref.Kind, ref.Namespace, ref.Name),
		))
	}
	for _, ref := range dr.MissingTopologySpread {
		resource := ref.Kind + "/" + ref.Name
		findings = append(findings, model.NewFinding(
			model.RiskYellow, "missing-topology-spread", ref.Namespace, resource,
			fmt.Sprintf("%s %s/%s no declara topology spread constraints", ref.Kind, ref.Namespace, ref.Name),
		))
	}

	return findings
}
