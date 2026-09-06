package report

import (
	"strings"
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
	// Una categoría sin entrada en el catálogo (internal/model/finding_catalog.go)
	// cae al marcador visible de model.Guidance, no al mensaje original en
	// silencio: así un olvido se nota en la salida.
	if !strings.Contains(recs[1].Action, "falta agregar la categoría") {
		t.Errorf("recs[1].Action = %q, want el marcador visible de categoría sin documentar", recs[1].Action)
	}
}

func TestBuildRecommendations_FallsBackToMessageWhenRecommendationEmpty(t *testing.T) {
	// Un Finding sin Recommendation (ej. deserializado de un snapshot
	// histórico generado antes de este campo) cae al Message original, en
	// vez de quedar con una acción vacía.
	f := model.Finding{ID: "x", Category: "missing-pdb", Message: "mensaje histórico"}

	recs := BuildRecommendations([]model.Finding{f})

	if recs[0].Action != "mensaje histórico" {
		t.Errorf("Action = %q, want el Message original como fallback", recs[0].Action)
	}
}
