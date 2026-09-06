package report

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/galexbh/husk/internal/model"
)

func sampleInventory() *model.Inventory {
	return &model.Inventory{
		Deployments: []model.WorkloadSummary{
			{Kind: "Deployment", Name: "api", Namespace: "shop", Replicas: 1, ReadyReplicas: 1, Risk: model.RiskRed},
		},
		Secrets: []model.SecretSummary{
			{Name: "db-creds", Namespace: "shop", Type: "Opaque", KeysCount: 2},
		},
	}
}

func TestRenderInventoryTable_ContainsExpectedContent(t *testing.T) {
	var buf bytes.Buffer
	if err := RenderInventoryTable(&buf, sampleInventory()); err != nil {
		t.Fatalf("RenderInventoryTable: %v", err)
	}
	out := buf.String()

	for _, want := range []string{"Deployments", "api", "ALTO", "Secrets", "db-creds"} {
		if !strings.Contains(out, want) {
			t.Errorf("la salida no contiene %q:\n%s", want, out)
		}
	}
	if !strings.Contains(out, "solo metadatos") {
		t.Error("se esperaba la nota de seguridad sobre Secrets/ConfigMaps")
	}
}

func TestRenderInventoryMarkdown_UsesMarkdownTableSyntax(t *testing.T) {
	var buf bytes.Buffer
	if err := RenderInventoryMarkdown(&buf, sampleInventory()); err != nil {
		t.Fatalf("RenderInventoryMarkdown: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "| Namespace") {
		t.Errorf("se esperaba sintaxis de tabla Markdown, got:\n%s", out)
	}
}

func TestRenderInventoryJSON_RoundTrips(t *testing.T) {
	inv := sampleInventory()
	var buf bytes.Buffer
	if err := RenderInventoryJSON(&buf, inv); err != nil {
		t.Fatalf("RenderInventoryJSON: %v", err)
	}

	var decoded model.Inventory
	if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
		t.Fatalf("el JSON producido no es válido: %v", err)
	}
	if len(decoded.Deployments) != 1 || decoded.Deployments[0].Name != "api" {
		t.Errorf("round-trip inválido: %+v", decoded.Deployments)
	}
}

func TestRenderInventory_UnsupportedFormat(t *testing.T) {
	var buf bytes.Buffer
	if err := RenderInventory(&buf, sampleInventory(), "yaml"); err == nil {
		t.Error("se esperaba un error para un formato no soportado")
	}
}

func TestRenderScoreTable_ShowsDimensionsAndTotal(t *testing.T) {
	s := &model.Score{
		Total: 82.5,
		Breakdown: []model.DimensionScore{
			{Name: "dr", Available: true, Score: 90, Weight: 0.3, EffectiveWeight: 0.3},
			{Name: "sizing", Available: false, Reason: "sin Prometheus", Weight: 0.3},
		},
	}
	var buf bytes.Buffer
	if err := RenderScoreTable(&buf, s); err != nil {
		t.Fatalf("RenderScoreTable: %v", err)
	}
	out := buf.String()
	for _, want := range []string{"82.5", "dr", "sizing", "sin Prometheus"} {
		if !strings.Contains(out, want) {
			t.Errorf("la salida no contiene %q:\n%s", want, out)
		}
	}
}
