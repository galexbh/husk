package sizing

import (
	"fmt"

	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/galexbh/husk/internal/config"
	"github.com/galexbh/husk/internal/model"
)

// observedValues son los resultados (ya sea que hayan tenido datos o no) de
// las tres queries de sizing para un contenedor.
type observedValues struct {
	cpuP95 float64
	cpuOK  bool

	memP99 float64
	memOK  bool

	memPeak float64
	peakOK  bool
}

// buildContainerSizing es una función pura: combina los requests/limits
// actuales de ctr con obs y produce el veredicto y la recomendación,
// siguiendo los umbrales de cfg (cpu_limit_multiplier, memory_buffer_percent,
// over_provision_factor).
func buildContainerSizing(ctr model.ContainerSummary, cfg config.SizingConfig, obs observedValues) model.ContainerSizing {
	cs := model.ContainerSizing{
		Name:              ctr.Name,
		CurrentCPURequest: ctr.CPURequest,
		CurrentCPULimit:   ctr.CPULimit,
		CurrentMemRequest: ctr.MemoryRequest,
		CurrentMemLimit:   ctr.MemoryLimit,
		HasData:           obs.cpuOK || obs.memOK || obs.peakOK,
	}

	if obs.cpuOK {
		cs.ObservedCPU = formatCPU(obs.cpuP95)
	}
	if obs.memOK {
		cs.ObservedMemory = formatMemory(obs.memP99)
	}
	if obs.peakOK {
		cs.ObservedMemoryPeak = formatMemory(obs.memPeak)
	}

	if !obs.cpuOK && !obs.memOK && !obs.peakOK {
		cs.Verdict = "sin-datos"
		cs.Risk = model.RiskUnknown
		return cs
	}

	hasLimits := ctr.CPULimit != "" && ctr.MemoryLimit != ""
	if !hasLimits {
		cs.Verdict = "sin-limites"
		cs.Risk = model.RiskRed
		applyRecommendation(&cs, cfg, obs)
		return cs
	}

	over := exceedsFactor(ctr.CPURequest, obs.cpuP95, obs.cpuOK, cfg.OverProvisionFactor) ||
		exceedsFactor(ctr.MemoryRequest, obs.memP99, obs.memOK, cfg.OverProvisionFactor)
	under := belowObserved(ctr.CPURequest, obs.cpuP95, obs.cpuOK) ||
		belowObserved(ctr.MemoryRequest, obs.memP99, obs.memOK)

	switch {
	case under:
		cs.Verdict = "subaprovisionado"
		cs.Risk = model.RiskRed
	case over:
		cs.Verdict = "sobreaprovisionado"
		cs.Risk = model.RiskYellow
	default:
		cs.Verdict = "saludable"
		cs.Risk = model.RiskGreen
		return cs
	}

	applyRecommendation(&cs, cfg, obs)
	return cs
}

// exceedsFactor reporta si currentQty (un request/limit ya parseado desde
// el modelo) supera observed*factor. Un request vacío o no parseable, o sin
// dato observado, nunca cuenta como sobreaprovisionado.
func exceedsFactor(currentQty string, observed float64, observedOK bool, factor float64) bool {
	if currentQty == "" || !observedOK || observed <= 0 {
		return false
	}
	q, err := resource.ParseQuantity(currentQty)
	if err != nil {
		return false
	}
	return q.AsApproximateFloat64() > observed*factor
}

// belowObserved reporta si el consumo observado supera al request
// declarado (riesgo de throttling/OOM).
func belowObserved(currentQty string, observed float64, observedOK bool) bool {
	if currentQty == "" || !observedOK {
		return false
	}
	q, err := resource.ParseQuantity(currentQty)
	if err != nil {
		return false
	}
	return observed > q.AsApproximateFloat64()
}

// applyRecommendation rellena los campos Recommended* de cs a partir de obs
// y los multiplicadores/buffers configurados.
func applyRecommendation(cs *model.ContainerSizing, cfg config.SizingConfig, obs observedValues) {
	if obs.cpuOK {
		cs.RecommendedCPURequest = formatCPU(obs.cpuP95)
		cs.RecommendedCPULimit = formatCPU(obs.cpuP95 * cfg.CPULimitMultiplier)
	}

	memBase, memBaseOK := obs.memPeak, obs.peakOK
	if !memBaseOK {
		memBase, memBaseOK = obs.memP99, obs.memOK
	}
	if memBaseOK {
		recommended := formatMemory(memBase * (1 + cfg.MemoryBufferPercent))
		cs.RecommendedMemRequest = recommended
		cs.RecommendedMemLimit = recommended
	}
}

func formatCPU(cores float64) string {
	milli := cores * 1000
	return fmt.Sprintf("%dm", int64(milli+0.5))
}

func formatMemory(bytes float64) string {
	q := resource.NewQuantity(int64(bytes), resource.BinarySI)
	return q.String()
}
