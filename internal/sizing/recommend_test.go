package sizing

import (
	"testing"

	"github.com/galexbh/husk/internal/config"
	"github.com/galexbh/husk/internal/model"
)

func testCfg() config.SizingConfig {
	return config.SizingConfig{
		CPUPercentile:         0.95,
		MemoryPercentile:      0.99,
		Lookback:              "7d",
		CPULimitMultiplier:    3.0,
		MemoryBufferPercent:   0.2,
		OverProvisionFactor:   2.0,
		SidecarContainerNames: []string{"istio-proxy", "oneagent"},
	}
}

func TestBuildContainerSizing_NoData(t *testing.T) {
	ctr := model.ContainerSummary{Name: "api", CPURequest: "100m", CPULimit: "200m", MemoryRequest: "128Mi", MemoryLimit: "256Mi"}
	cs := buildContainerSizing(ctr, testCfg(), observedValues{})

	if cs.Verdict != "sin-datos" {
		t.Errorf("Verdict = %q, want sin-datos", cs.Verdict)
	}
	if cs.Risk != model.RiskUnknown {
		t.Errorf("Risk = %q, want unknown", cs.Risk)
	}
	if cs.RecommendedCPURequest != "" {
		t.Error("no debería haber recomendación sin datos observados")
	}
}

func TestBuildContainerSizing_NoLimits(t *testing.T) {
	ctr := model.ContainerSummary{Name: "api", CPURequest: "100m"}
	obs := observedValues{cpuP95: 0.05, cpuOK: true, memPeak: 100 * 1024 * 1024, peakOK: true}
	cs := buildContainerSizing(ctr, testCfg(), obs)

	if cs.Verdict != "sin-limites" {
		t.Errorf("Verdict = %q, want sin-limites", cs.Verdict)
	}
	if cs.Risk != model.RiskRed {
		t.Errorf("Risk = %q, want red", cs.Risk)
	}
	if cs.RecommendedCPURequest == "" || cs.RecommendedCPULimit == "" {
		t.Error("se esperaba una recomendación de CPU")
	}
	if cs.RecommendedMemRequest == "" {
		t.Error("se esperaba una recomendación de memoria basada en el pico")
	}
}

func TestBuildContainerSizing_OverProvisioned(t *testing.T) {
	// Request de 1 core, pero el consumo observado es de solo 100m: más de
	// 2x de sobra -> sobreaprovisionado.
	ctr := model.ContainerSummary{Name: "api", CPURequest: "1", CPULimit: "2", MemoryRequest: "128Mi", MemoryLimit: "256Mi"}
	obs := observedValues{cpuP95: 0.1, cpuOK: true, memP99: 64 * 1024 * 1024, memOK: true}
	cs := buildContainerSizing(ctr, testCfg(), obs)

	if cs.Verdict != "sobreaprovisionado" {
		t.Errorf("Verdict = %q, want sobreaprovisionado", cs.Verdict)
	}
	if cs.Risk != model.RiskYellow {
		t.Errorf("Risk = %q, want yellow", cs.Risk)
	}
	if cs.RecommendedCPURequest == "" {
		t.Error("se esperaba una recomendación de reducción")
	}
}

func TestBuildContainerSizing_UnderProvisioned(t *testing.T) {
	// Request de 100m, pero el consumo observado es de 500m: el
	// contenedor está siendo throttled -> subaprovisionado (más severo que
	// sobreaprovisionado).
	ctr := model.ContainerSummary{Name: "api", CPURequest: "100m", CPULimit: "200m", MemoryRequest: "128Mi", MemoryLimit: "256Mi"}
	obs := observedValues{cpuP95: 0.5, cpuOK: true, memP99: 64 * 1024 * 1024, memOK: true}
	cs := buildContainerSizing(ctr, testCfg(), obs)

	if cs.Verdict != "subaprovisionado" {
		t.Errorf("Verdict = %q, want subaprovisionado", cs.Verdict)
	}
	if cs.Risk != model.RiskRed {
		t.Errorf("Risk = %q, want red", cs.Risk)
	}
}

func TestBuildContainerSizing_Sidecar(t *testing.T) {
	// Sin límites (candidato a "sin-limites"), pero el nombre matchea un
	// patrón de sidecar_container_names -> se ignora en vez de marcarse.
	ctr := model.ContainerSummary{Name: "istio-proxy", CPURequest: "10m"}
	obs := observedValues{cpuP95: 0.005, cpuOK: true}
	cs := buildContainerSizing(ctr, testCfg(), obs)

	if cs.Verdict != "sidecar-ignorado" {
		t.Errorf("Verdict = %q, want sidecar-ignorado", cs.Verdict)
	}
	if cs.Risk != model.RiskUnknown {
		t.Errorf("Risk = %q, want unknown", cs.Risk)
	}
	if cs.RecommendedCPURequest != "" {
		t.Error("un sidecar ignorado no debería llevar recomendación")
	}
}

func TestBuildContainerSizing_Healthy(t *testing.T) {
	ctr := model.ContainerSummary{Name: "api", CPURequest: "200m", CPULimit: "400m", MemoryRequest: "256Mi", MemoryLimit: "512Mi"}
	obs := observedValues{cpuP95: 0.15, cpuOK: true, memP99: 200 * 1024 * 1024, memOK: true}
	cs := buildContainerSizing(ctr, testCfg(), obs)

	if cs.Verdict != "saludable" {
		t.Errorf("Verdict = %q, want saludable", cs.Verdict)
	}
	if cs.Risk != model.RiskGreen {
		t.Errorf("Risk = %q, want green", cs.Risk)
	}
	if cs.RecommendedCPURequest != "" {
		t.Error("un contenedor saludable no debería llevar recomendación")
	}
}
