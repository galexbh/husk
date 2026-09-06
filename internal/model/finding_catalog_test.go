package model

import "testing"

func TestFindingCatalog_NoEmptyEntries(t *testing.T) {
	for category, g := range findingCatalog {
		if g.Explanation == "" {
			t.Errorf("categoría %q: Explanation vacía", category)
		}
		if g.Recommendation == "" {
			t.Errorf("categoría %q: Recommendation vacía", category)
		}
	}
}

func TestGuidance_KnownCategory(t *testing.T) {
	g := Guidance("missing-pdb")
	if g.Explanation == "" || g.Recommendation == "" {
		t.Error("Guidance para una categoría catalogada no debería devolver campos vacíos")
	}
}

func TestGuidance_UnknownCategory_IsVisible(t *testing.T) {
	g := Guidance("categoria-inventada")
	if g.Explanation != unmappedCategoryFallback {
		t.Errorf("Explanation = %q, want el marcador de categoría no documentada", g.Explanation)
	}
	if g.Recommendation == "" {
		t.Error("Recommendation debería señalar visiblemente la categoría faltante, no quedar vacía")
	}
}

func TestNewFinding_PopulatesGuidance(t *testing.T) {
	f := NewFinding(RiskRed, "missing-pdb", "shop", "Deployment/api", "mensaje")
	if f.Explanation == "" || f.Recommendation == "" {
		t.Error("NewFinding debería poblar Explanation/Recommendation desde el catálogo")
	}
}
