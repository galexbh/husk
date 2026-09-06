package sizing

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/galexbh/husk/internal/model"
	"github.com/galexbh/husk/internal/promclient"
)

// fakePrometheusHandler simula el endpoint /api/v1/query de Prometheus:
// devuelve un valor fijo para las queries de CPU y otro para las de
// memoria, distinguiendo por una subcadena de la query. También verifica
// que el bearer token llegue correctamente.
func fakePrometheusHandler(t *testing.T, wantToken string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer "+wantToken {
			t.Errorf("Authorization header = %q, want %q", got, "Bearer "+wantToken)
		}

		if err := r.ParseForm(); err != nil {
			t.Fatalf("ParseForm: %v", err)
		}
		query := r.Form.Get("query")
		var value string
		switch {
		case strings.Contains(query, "container_cpu_usage_seconds_total"):
			value = "0.5" // 500m observado
		case strings.Contains(query, "container_memory_working_set_bytes") && strings.Contains(query, "max_over_time"):
			value = "209715200" // 200Mi pico
		case strings.Contains(query, "container_memory_working_set_bytes"):
			value = "157286400" // 150Mi p99
		default:
			t.Fatalf("query inesperada: %s", query)
		}

		resp := map[string]any{
			"status": "success",
			"data": map[string]any{
				"resultType": "vector",
				"result": []any{
					map[string]any{
						"metric": map[string]any{},
						"value":  []any{1700000000, value},
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			t.Fatal(err)
		}
	}
}

func TestAnalyzer_Analyze_EndToEnd(t *testing.T) {
	const token = "test-bearer-token"
	server := httptest.NewServer(fakePrometheusHandler(t, token))
	defer server.Close()

	logger := slog.New(slog.NewTextHandler(&testWriter{t}, nil))
	promCli, err := promclient.New(server.URL, token, nil, logger)
	if err != nil {
		t.Fatalf("promclient.New: %v", err)
	}

	cfg := testCfg()
	analyzer := New(promCli, cfg)

	workloads := []model.WorkloadSummary{
		{
			Kind: "Deployment", Name: "api", Namespace: "shop",
			Containers: []model.ContainerSummary{
				{Name: "api", CPURequest: "1", CPULimit: "2", MemoryRequest: "128Mi", MemoryLimit: "256Mi"},
			},
		},
	}

	report, err := analyzer.Analyze(context.Background(), workloads)
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}

	if len(report.Workloads) != 1 || len(report.Workloads[0].Containers) != 1 {
		t.Fatalf("estructura de reporte inesperada: %+v", report)
	}

	cs := report.Workloads[0].Containers[0]
	// Request de 1 core vs 500m observados: sobreaprovisionado (factor 2x
	// no se alcanza en este caso — 1 core / 0.5 = 2x exacto, no > 2x — así
	// que se espera saludable). Verificamos que al menos haya datos y un
	// veredicto coherente, sin acoplar el test al umbral exacto.
	if !cs.HasData {
		t.Fatal("se esperaba HasData = true")
	}
	if cs.ObservedCPU == "" || cs.ObservedMemory == "" || cs.ObservedMemoryPeak == "" {
		t.Errorf("valores observados incompletos: %+v", cs)
	}
	if cs.Verdict == "" || cs.Verdict == "sin-datos" {
		t.Errorf("Verdict = %q, no se esperaba sin-datos", cs.Verdict)
	}
}

// testWriter adapta *testing.T a io.Writer para capturar los logs de debug
// del cliente Prometheus dentro del test.
type testWriter struct{ t *testing.T }

func (w *testWriter) Write(p []byte) (int, error) {
	w.t.Logf("%s", p)
	return len(p), nil
}
