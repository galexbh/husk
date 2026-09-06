package grafana

import (
	"encoding/json"
	"testing"
)

func TestBuildDashboard_ValidJSON(t *testing.T) {
	d := BuildDashboard()

	data, err := json.Marshal(d)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}

	var generic map[string]interface{}
	if err := json.Unmarshal(data, &generic); err != nil {
		t.Fatalf("el dashboard no es JSON válido: %v", err)
	}

	if generic["title"] == "" {
		t.Error("el dashboard debería tener título")
	}
	if _, ok := generic["__inputs"]; !ok {
		t.Error("el dashboard debería declarar __inputs (datasource exportable)")
	}
	panels, ok := generic["panels"].([]interface{})
	if !ok || len(panels) == 0 {
		t.Fatal("el dashboard debería tener al menos un panel")
	}

	for i, p := range panels {
		panel, ok := p.(map[string]interface{})
		if !ok {
			t.Fatalf("panel %d no es un objeto", i)
		}
		ds, ok := panel["datasource"].(map[string]interface{})
		if !ok {
			t.Fatalf("panel %d sin datasource", i)
		}
		if ds["uid"] != "${DS_PROMETHEUS}" {
			t.Errorf("panel %d datasource.uid = %v, want ${DS_PROMETHEUS}", i, ds["uid"])
		}
		targets, ok := panel["targets"].([]interface{})
		if !ok || len(targets) == 0 {
			t.Errorf("panel %d sin targets", i)
		}
	}
}

func TestBuildDashboard_UniquePanelIDs(t *testing.T) {
	d := BuildDashboard()
	seen := make(map[int]bool)
	for _, p := range d.Panels {
		if seen[p.ID] {
			t.Errorf("panel ID duplicado: %d", p.ID)
		}
		seen[p.ID] = true
	}
}
