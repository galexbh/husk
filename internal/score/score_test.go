package score

import (
	"fmt"
	"testing"

	"github.com/galexbh/husk/internal/config"
	"github.com/galexbh/husk/internal/model"
)

func testWeights() config.ScoreWeights {
	return config.ScoreWeights{Sizing: 0.3, DR: 0.3, Capacity: 0.2, PDB: 0.1, Topology: 0.1}
}

func healthyDR() *model.DRReadiness {
	return &model.DRReadiness{
		OADPInstalled:       true,
		OADPHealthy:         true,
		EtcdCheckApplicable: false,
	}
}

func TestCompute_PerfectCluster(t *testing.T) {
	in := Inputs{
		Weights:  testWeights(),
		Sizing:   &model.SizingReport{},
		Capacity: &model.CapacityReport{},
		DR:       healthyDR(),
	}
	// Sizing sin ningún contenedor evaluable -> no disponible; el resto de
	// dimensiones parten de un estado sin hallazgos -> 100.
	s := Compute(in)

	for _, d := range s.Breakdown {
		if d.Name == "sizing" {
			if d.Available {
				t.Error("sizing sin contenedores evaluables debería marcarse no disponible")
			}
			continue
		}
		if d.Score != 100 {
			t.Errorf("dimensión %s: score = %v, want 100", d.Name, d.Score)
		}
	}
	if s.Total != 100 {
		t.Errorf("Total = %v, want 100 (sin hallazgos en ninguna dimensión disponible)", s.Total)
	}
}

func TestCompute_SizingUnavailable_RedistributesWeight(t *testing.T) {
	in := Inputs{
		Weights:  testWeights(),
		Sizing:   nil, // no disponible
		Capacity: &model.CapacityReport{},
		DR:       healthyDR(),
	}
	s := Compute(in)

	var totalEffective float64
	for _, d := range s.Breakdown {
		if d.Name == "sizing" {
			if d.Available {
				t.Fatal("sizing debería quedar no disponible con Sizing=nil")
			}
			if d.EffectiveWeight != 0 {
				t.Errorf("sizing.EffectiveWeight = %v, want 0", d.EffectiveWeight)
			}
			continue
		}
		totalEffective += d.EffectiveWeight
	}
	if totalEffective < 0.999 || totalEffective > 1.001 {
		t.Errorf("la suma de EffectiveWeight de las dimensiones disponibles = %v, want ~1.0", totalEffective)
	}
	if s.Total != 100 {
		t.Errorf("Total = %v, want 100 (todas las dimensiones disponibles están limpias)", s.Total)
	}
}

func TestCompute_OADPNotInstalled_PenalizesDR(t *testing.T) {
	dr := &model.DRReadiness{OADPInstalled: false, OADPMessage: "no instalado"}
	dr.Findings = []model.Finding{model.NewFinding(model.RiskRed, "oadp-not-installed", "openshift-adp", "", "no instalado")}

	in := Inputs{Weights: testWeights(), Capacity: &model.CapacityReport{}, DR: dr}
	s := Compute(in)

	for _, d := range s.Breakdown {
		if d.Name == "dr" && d.Score >= 100 {
			t.Errorf("dr.Score = %v, debería reducirse por OADP no instalado", d.Score)
		}
	}
	if s.Total >= 100 {
		t.Errorf("Total = %v, debería ser menor a 100", s.Total)
	}
}

func TestCompute_PDBAndTopologyAreProportional(t *testing.T) {
	dr := healthyDR()
	dr.CriticalWorkloadsTotal = 4
	dr.MissingPDBs = []model.WorkloadRef{{Kind: "Deployment", Namespace: "shop", Name: "api"}}
	dr.MissingTopologySpread = nil

	in := Inputs{Weights: testWeights(), Capacity: &model.CapacityReport{}, DR: dr}
	s := Compute(in)

	var pdbScore, topoScore float64
	for _, d := range s.Breakdown {
		switch d.Name {
		case "pdb":
			pdbScore = d.Score
		case "topology":
			topoScore = d.Score
		}
	}
	if pdbScore != 75 {
		t.Errorf("pdbScore = %v, want 75 (3/4 workloads cubiertos)", pdbScore)
	}
	if topoScore != 100 {
		t.Errorf("topoScore = %v, want 100 (ningún workload sin topology spread)", topoScore)
	}
}

func TestCompute_CapacitySaturatedNode(t *testing.T) {
	capReport := &model.CapacityReport{
		Nodes: []model.NodeCapacity{
			{Name: "node-1", Risk: model.RiskRed, RiskAxes: []string{"cpu", "memory"}},
			{Name: "node-2", Risk: model.RiskGreen},
		},
	}
	in := Inputs{Weights: testWeights(), Capacity: capReport, DR: healthyDR()}
	s := Compute(in)

	for _, d := range s.Breakdown {
		if d.Name == "capacity" && d.Score != 70 {
			t.Errorf("capacity.Score = %v, want 70 (100 - 60*(1/2): 1 de 2 nodos en riesgo ALTO)", d.Score)
		}
	}
}

func TestCompute_CapacitySaturation_ScalesWithClusterSize(t *testing.T) {
	nodes := []model.NodeCapacity{{Name: "node-1", Risk: model.RiskRed, RiskAxes: []string{"cpu", "memory"}}}
	for i := 2; i <= 10; i++ {
		nodes = append(nodes, model.NodeCapacity{Name: fmt.Sprintf("node-%d", i), Risk: model.RiskGreen})
	}
	capReport := &model.CapacityReport{Nodes: nodes}
	in := Inputs{Weights: testWeights(), Capacity: capReport, DR: healthyDR()}
	s := Compute(in)

	for _, d := range s.Breakdown {
		if d.Name == "capacity" && d.Score != 94 {
			t.Errorf("capacity.Score = %v, want 94 (100 - 60*(1/10): el mismo hallazgo pesa menos en un cluster más grande)", d.Score)
		}
	}
}

func TestCompute_CapacityMedioWeightsHalfOfAlto(t *testing.T) {
	capReport := &model.CapacityReport{
		Nodes: []model.NodeCapacity{
			{Name: "node-1", Risk: model.RiskYellow, RiskAxes: []string{"cpu"}},
			{Name: "node-2", Risk: model.RiskGreen},
		},
	}
	in := Inputs{Weights: testWeights(), Capacity: capReport, DR: healthyDR()}
	s := Compute(in)

	for _, d := range s.Breakdown {
		if d.Name == "capacity" && d.Score != 85 {
			t.Errorf("capacity.Score = %v, want 85 (100 - 60*(0.5/2): un nodo MEDIO pesa la mitad que uno ALTO)", d.Score)
		}
	}
}

func TestCompute_DRMissingBackup_Proportional(t *testing.T) {
	dr := healthyDR()
	dr.ApplicationNamespaces = []string{"shop", "billing", "auth", "reports"}
	dr.NamespacesWithoutBackup = []string{"shop"}

	in := Inputs{Weights: testWeights(), Capacity: &model.CapacityReport{}, DR: dr}
	s := Compute(in)

	for _, d := range s.Breakdown {
		if d.Name == "dr" && d.Score != 90 {
			t.Errorf("dr.Score = %v, want 90 (100 - 40*(1/4))", d.Score)
		}
	}
}

func TestCompute_DRMissingBackup_ScalesWithClusterSize(t *testing.T) {
	namespaces := make([]string, 20)
	for i := range namespaces {
		namespaces[i] = fmt.Sprintf("ns-%d", i)
	}
	dr := healthyDR()
	dr.ApplicationNamespaces = namespaces
	dr.NamespacesWithoutBackup = []string{"ns-0"}

	in := Inputs{Weights: testWeights(), Capacity: &model.CapacityReport{}, DR: dr}
	s := Compute(in)

	for _, d := range s.Breakdown {
		if d.Name == "dr" && d.Score != 98 {
			t.Errorf("dr.Score = %v, want 98 (100 - 40*(1/20)): el mismo hallazgo pesa menos en un cluster con más namespaces", d.Score)
		}
	}
}

func TestCompute_SizingSidecarIgnored_NoDeduction(t *testing.T) {
	sizing := &model.SizingReport{
		Workloads: []model.WorkloadSizing{{
			Kind: "Deployment", Namespace: "shop", Name: "api",
			Containers: []model.ContainerSizing{
				{Name: "istio-proxy", HasData: true, Verdict: "sidecar-ignorado", Risk: model.RiskUnknown},
			},
		}},
	}
	in := Inputs{Weights: testWeights(), Sizing: sizing, Capacity: &model.CapacityReport{}, DR: healthyDR()}
	s := Compute(in)

	for _, d := range s.Breakdown {
		if d.Name == "sizing" && d.Available {
			t.Error("sizing con solo contenedores sidecar-ignorado debería marcarse no disponible, no evaluarse")
		}
	}
}

func TestCompute_Deterministic(t *testing.T) {
	dr := healthyDR()
	dr.CriticalWorkloadsTotal = 2
	dr.MissingPDBs = []model.WorkloadRef{{Kind: "Deployment", Namespace: "shop", Name: "api"}}

	in := Inputs{Weights: testWeights(), Capacity: &model.CapacityReport{}, DR: dr}
	s1 := Compute(in)
	s2 := Compute(in)

	if s1.Total != s2.Total {
		t.Errorf("Compute no es determinista: %v != %v", s1.Total, s2.Total)
	}
}
