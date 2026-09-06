package diff

import (
	"testing"
	"time"

	"github.com/galexbh/husk/internal/model"
)

func snapshotWithFindings(score float64, findings []model.Finding, breakdown []model.DimensionScore, at time.Time) *model.ClusterSnapshot {
	return &model.ClusterSnapshot{
		Meta:  model.Meta{GeneratedAt: at},
		Score: &model.Score{Total: score, Findings: findings, Breakdown: breakdown, GeneratedAt: at},
	}
}

func TestCompute_NoChanges(t *testing.T) {
	findings := []model.Finding{model.NewFinding(model.RiskRed, "missing-backup", "shop", "", "sin backup")}
	now := time.Now()
	snap := snapshotWithFindings(80, findings, nil, now)

	result := Compute(snap, snap)
	if !result.NoChanges {
		t.Error("comparar un snapshot consigo mismo debería reportar NoChanges=true")
	}
	if len(result.FindingsResolved) != 0 || len(result.FindingsIntroduced) != 0 {
		t.Errorf("no se esperaban hallazgos resueltos/introducidos: %+v / %+v", result.FindingsResolved, result.FindingsIntroduced)
	}
}

func TestCompute_ResolvedAndIntroducedFindings(t *testing.T) {
	resolved := model.NewFinding(model.RiskRed, "missing-backup", "legacy", "", "sin backup")
	persisting := model.NewFinding(model.RiskYellow, "missing-pdb", "shop", "Deployment/api", "sin pdb")
	introduced := model.NewFinding(model.RiskRed, "oadp-unhealthy", "openshift-adp", "", "no reconciliado")

	from := snapshotWithFindings(70, []model.Finding{resolved, persisting}, nil, time.Now().Add(-time.Hour))
	to := snapshotWithFindings(60, []model.Finding{persisting, introduced}, nil, time.Now())

	result := Compute(from, to)

	if result.NoChanges {
		t.Fatal("no debería reportar NoChanges cuando hay hallazgos resueltos/introducidos")
	}
	if len(result.FindingsResolved) != 1 || result.FindingsResolved[0].ID != resolved.ID {
		t.Errorf("FindingsResolved = %+v, want solo %s", result.FindingsResolved, resolved.ID)
	}
	if len(result.FindingsIntroduced) != 1 || result.FindingsIntroduced[0].ID != introduced.ID {
		t.Errorf("FindingsIntroduced = %+v, want solo %s", result.FindingsIntroduced, introduced.ID)
	}
	if result.FindingsPersisting != 1 {
		t.Errorf("FindingsPersisting = %d, want 1", result.FindingsPersisting)
	}
	if result.ScoreDelta != -10 {
		t.Errorf("ScoreDelta = %v, want -10", result.ScoreDelta)
	}
}

func TestCompute_DimensionDeltas(t *testing.T) {
	from := snapshotWithFindings(80, nil, []model.DimensionScore{
		{Name: "dr", Available: true, Score: 90},
		{Name: "capacity", Available: false, Score: 0},
	}, time.Now().Add(-time.Hour))
	to := snapshotWithFindings(75, nil, []model.DimensionScore{
		{Name: "dr", Available: true, Score: 70},
		{Name: "capacity", Available: true, Score: 100},
	}, time.Now())

	result := Compute(from, to)

	if len(result.DimensionDeltas) != 1 {
		t.Fatalf("DimensionDeltas = %+v, want solo 'dr' (capacity no estaba disponible en 'from')", result.DimensionDeltas)
	}
	if result.DimensionDeltas[0].Name != "dr" || result.DimensionDeltas[0].Delta != -20 {
		t.Errorf("DimensionDeltas[0] = %+v, want dr con delta -20", result.DimensionDeltas[0])
	}
}
