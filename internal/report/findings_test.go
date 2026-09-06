package report

import (
	"testing"

	"github.com/galexbh/husk/internal/model"
)

func TestPrioritizeFindings_DedupAndSeverityOrder(t *testing.T) {
	shared := model.NewFinding(model.RiskYellow, "missing-pdb", "shop", "Deployment/api", "sin pdb")
	red := model.NewFinding(model.RiskRed, "oadp-not-installed", "openshift-adp", "", "no instalado")
	green := model.NewFinding(model.RiskGreen, "healthy", "shop", "", "todo bien")

	merged := PrioritizeFindings(
		[]model.Finding{shared, green},
		[]model.Finding{shared, red}, // shared duplicado a propósito
	)

	if len(merged) != 3 {
		t.Fatalf("len(merged) = %d, want 3 (shared no debería duplicarse)", len(merged))
	}
	if merged[0].ID != red.ID {
		t.Errorf("merged[0] = %+v, want el hallazgo rojo primero", merged[0])
	}
	if merged[len(merged)-1].ID != green.ID {
		t.Errorf("merged[last] = %+v, want el hallazgo verde al final", merged[len(merged)-1])
	}
}

func TestBuildRecommendations_KnownAndUnknownCategory(t *testing.T) {
	known := model.NewFinding(model.RiskRed, "missing-pdb", "shop", "Deployment/api", "mensaje genérico")
	unknown := model.NewFinding(model.RiskYellow, "categoria-inventada", "shop", "", "mensaje original")

	recs := BuildRecommendations([]model.Finding{known, unknown})

	if len(recs) != 2 {
		t.Fatalf("len(recs) = %d, want 2", len(recs))
	}
	if recs[0].Action == known.Message {
		t.Error("una categoría conocida debería producir una acción específica, no repetir el mensaje")
	}
	if recs[1].Action != unknown.Message {
		t.Error("una categoría desconocida debería caer al mensaje original del hallazgo")
	}
}
