package score

import (
	"fmt"
	"time"

	"github.com/galexbh/husk/internal/config"
	"github.com/galexbh/husk/internal/model"
)

// Inputs agrupa todo lo que Compute necesita: los reportes ya producidos
// por internal/sizing, internal/capacity e internal/dr, y los pesos de
// score.weights.* de la configuración.
type Inputs struct {
	Weights config.ScoreWeights

	// Sizing puede ser nil (o no tener ningún contenedor evaluable) cuando
	// Prometheus/Thanos Querier no está disponible; en ese caso la
	// dimensión se marca no disponible y su peso se redistribuye entre las
	// demás, en vez de penalizar con un 0 que no reflejaría la realidad.
	Sizing       *model.SizingReport
	SizingReason string // por qué Sizing es nil/no evaluable, si aplica

	Capacity *model.CapacityReport
	DR       *model.DRReadiness
}

// Compute produce el model.Score: una función pura y determinista — la
// misma entrada siempre produce el mismo resultado.
func Compute(in Inputs) *model.Score {
	dims := make([]model.DimensionScore, 0, 5)
	allFindings := make([]model.Finding, 0)

	if s, findings, ok := scoreSizing(in.Sizing); ok {
		dims = append(dims, model.DimensionScore{Name: "sizing", Available: true, Score: s, Weight: in.Weights.Sizing, FindingIDs: findingIDs(findings)})
		allFindings = append(allFindings, findings...)
	} else {
		reason := in.SizingReason
		if reason == "" {
			reason = "no hay contenedores con consumo histórico observable"
		}
		dims = append(dims, model.DimensionScore{Name: "sizing", Available: false, Reason: reason, Weight: in.Weights.Sizing})
	}

	drScore, drFindings := scoreDR(in.DR)
	dims = append(dims, model.DimensionScore{Name: "dr", Available: true, Score: drScore, Weight: in.Weights.DR, FindingIDs: findingIDs(drFindings)})
	allFindings = append(allFindings, drFindings...)

	capScore, capFindings := scoreCapacity(in.Capacity)
	dims = append(dims, model.DimensionScore{Name: "capacity", Available: true, Score: capScore, Weight: in.Weights.Capacity, FindingIDs: findingIDs(capFindings)})
	allFindings = append(allFindings, capFindings...)

	pdbScore, pdbFindings := scorePDB(in.DR)
	dims = append(dims, model.DimensionScore{Name: "pdb", Available: true, Score: pdbScore, Weight: in.Weights.PDB, FindingIDs: findingIDs(pdbFindings)})
	allFindings = append(allFindings, pdbFindings...)

	topoScore, topoFindings := scoreTopology(in.DR)
	dims = append(dims, model.DimensionScore{Name: "topology", Available: true, Score: topoScore, Weight: in.Weights.Topology, FindingIDs: findingIDs(topoFindings)})
	allFindings = append(allFindings, topoFindings...)

	// Renormalizar: el peso de una dimensión no disponible se redistribuye
	// proporcionalmente entre las que sí lo están.
	var totalWeight float64
	for _, d := range dims {
		if d.Available {
			totalWeight += d.Weight
		}
	}
	if totalWeight <= 0 {
		totalWeight = 1
	}

	var total float64
	for i := range dims {
		if !dims[i].Available {
			continue
		}
		dims[i].EffectiveWeight = dims[i].Weight / totalWeight
		total += dims[i].Score * dims[i].EffectiveWeight
	}

	return &model.Score{
		Total:       clampScore(total),
		Breakdown:   dims,
		Findings:    allFindings,
		GeneratedAt: time.Now(),
	}
}

func findingIDs(findings []model.Finding) []string {
	ids := make([]string, 0, len(findings))
	for _, f := range findings {
		ids = append(ids, f.ID)
	}
	return ids
}

// scoreSizing pondera cada contenedor evaluable (HasData=true) según su
// veredicto. Si ningún workload tiene ningún contenedor evaluable, ok es
// false: la dimensión completa se reporta como no disponible.
func scoreSizing(r *model.SizingReport) (float64, []model.Finding, bool) {
	if r == nil {
		return 0, nil, false
	}

	score := 100.0
	var findings []model.Finding
	anyEvaluable := false

	for _, w := range r.Workloads {
		for _, c := range w.Containers {
			if !c.HasData || c.Verdict == "sidecar-ignorado" {
				continue
			}
			anyEvaluable = true
			resource := fmt.Sprintf("%s/%s/%s", w.Namespace, w.Name, c.Name)

			switch c.Verdict {
			case "sin-limites":
				score -= sizingNoLimitsDeduction
				findings = append(findings, model.NewFinding(model.RiskRed, "sizing-no-limits", w.Namespace, resource,
					fmt.Sprintf("%s %s/%s (%s): sin resources.limits", w.Kind, w.Namespace, w.Name, c.Name)))
			case "subaprovisionado":
				score -= sizingUnderProvisionedDeduction
				findings = append(findings, model.NewFinding(model.RiskRed, "sizing-under-provisioned", w.Namespace, resource,
					fmt.Sprintf("%s %s/%s (%s): subaprovisionado, riesgo de throttling/OOM", w.Kind, w.Namespace, w.Name, c.Name)))
			case "sobreaprovisionado":
				score -= sizingOverProvisionedDeduction
				findings = append(findings, model.NewFinding(model.RiskYellow, "sizing-over-provisioned", w.Namespace, resource,
					fmt.Sprintf("%s %s/%s (%s): sobreaprovisionado", w.Kind, w.Namespace, w.Name, c.Name)))
			}
		}
	}

	if !anyEvaluable {
		return 0, nil, false
	}
	return clampScore(score), findings, true
}

// scoreDR reutiliza los hallazgos ya consolidados por
// internal/dr.Assessor.Assess (dr.Findings), filtrando los que
// corresponden a esta dimensión, para no duplicar la lógica de mensajes.
func scoreDR(dr *model.DRReadiness) (float64, []model.Finding) {
	score := 100.0

	switch {
	case !dr.OADPInstalled:
		score -= drOADPNotInstalledDeduction
	case !dr.OADPHealthy:
		score -= drOADPUnhealthyDeduction
	}

	if total := len(dr.ApplicationNamespaces); total > 0 {
		score -= drMissingBackupMaxDeduction * float64(len(dr.NamespacesWithoutBackup)) / float64(total)
	}
	score -= capDeduction(float64(len(dr.NamespacesWithStaleBackup))*drStaleBackupDeduction, drStaleBackupCap)

	if dr.EtcdCheckApplicable && dr.EtcdSnapshot != nil {
		switch {
		case !dr.EtcdSnapshot.Verifiable:
			score -= drEtcdUnverifiableDeduction
		case !dr.EtcdSnapshot.AgeOK:
			score -= drEtcdStaleDeduction
		}
	}

	score -= capDeduction(float64(len(dr.StorageClassesWithoutCSISnapshot))*drCSIUnsupportedDeduction, drCSIUnsupportedCap)

	findings := filterFindings(dr.Findings,
		"oadp-not-installed", "oadp-unhealthy", "missing-backup", "stale-backup",
		"etcd-snapshot-unverifiable", "etcd-snapshot-stale", "csi-snapshot-unsupported",
	)
	return clampScore(score), findings
}

// scorePDB y scoreTopology son proporcionales al número de workloads
// críticos cubiertos, en vez de deducciones fijas: así el score refleja la
// misma severidad relativa sin importar el tamaño del cluster.
func scorePDB(dr *model.DRReadiness) (float64, []model.Finding) {
	if dr.CriticalWorkloadsTotal == 0 {
		return 100, nil
	}
	covered := dr.CriticalWorkloadsTotal - len(dr.MissingPDBs)
	score := 100 * float64(covered) / float64(dr.CriticalWorkloadsTotal)
	return clampScore(score), filterFindings(dr.Findings, "missing-pdb")
}

func scoreTopology(dr *model.DRReadiness) (float64, []model.Finding) {
	if dr.CriticalWorkloadsTotal == 0 {
		return 100, nil
	}
	covered := dr.CriticalWorkloadsTotal - len(dr.MissingTopologySpread)
	score := 100 * float64(covered) / float64(dr.CriticalWorkloadsTotal)
	return clampScore(score), filterFindings(dr.Findings, "missing-topology-spread")
}

// scoreCapacity pondera cada nodo según su nivel de riesgo (ver
// capacity.nodeRisk): ALTO cuenta entero, MEDIO (un solo eje de headroom
// bajo el umbral, el otro sano) cuenta la mitad
// (capacityHeadroomWarningWeight), para no tratar igual un nodo con un
// único eje ajustado que uno realmente saturado en ambos.
func scoreCapacity(c *model.CapacityReport) (float64, []model.Finding) {
	score := 100.0
	var findings []model.Finding

	var altoCount, medioCount int
	for _, n := range c.Nodes {
		switch n.Risk {
		case model.RiskRed:
			altoCount++
			findings = append(findings, model.NewFinding(model.RiskRed, "node-saturated", "", n.Name, nodeSaturatedMessage(n)))
		case model.RiskYellow:
			medioCount++
			findings = append(findings, model.NewFinding(model.RiskYellow, "node-headroom-warning", "", n.Name, nodeHeadroomWarningMessage(n)))
		}
	}
	if total := len(c.Nodes); total > 0 {
		weighted := float64(altoCount) + capacityHeadroomWarningWeight*float64(medioCount)
		score -= capacitySaturatedMaxDeduction * weighted / float64(total)
	}

	score -= capDeduction(float64(len(c.ConcentrationRisks))*capacityConcentrationDeduction, capacityConcentrationCap)
	for _, r := range c.ConcentrationRisks {
		findings = append(findings, model.NewFinding(model.RiskYellow, "concentration-risk", r.Namespace, r.Kind+"/"+r.Name, r.Message))
	}

	return clampScore(score), findings
}

func nodeSaturatedMessage(n model.NodeCapacity) string {
	if len(n.RiskAxes) == 0 {
		return fmt.Sprintf("nodo %s no está Ready o está marcado unschedulable", n.Name)
	}
	return fmt.Sprintf("nodo %s con headroom crítico en CPU y memoria (CPU %.1f%%, memoria %.1f%%)", n.Name, n.CPUHeadroomPercent, n.MemoryHeadroomPercent)
}

func nodeHeadroomWarningMessage(n model.NodeCapacity) string {
	if len(n.RiskAxes) == 1 && n.RiskAxes[0] == "cpu" {
		return fmt.Sprintf("nodo %s con headroom ajustado en CPU (%.1f%%); memoria saludable (%.1f%%)", n.Name, n.CPUHeadroomPercent, n.MemoryHeadroomPercent)
	}
	return fmt.Sprintf("nodo %s con headroom ajustado en memoria (%.1f%%); CPU saludable (%.1f%%)", n.Name, n.MemoryHeadroomPercent, n.CPUHeadroomPercent)
}

func filterFindings(all []model.Finding, categories ...string) []model.Finding {
	set := make(map[string]bool, len(categories))
	for _, c := range categories {
		set[c] = true
	}
	out := make([]model.Finding, 0)
	for _, f := range all {
		if set[f.Category] {
			out = append(out, f)
		}
	}
	return out
}
