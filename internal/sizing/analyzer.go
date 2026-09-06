// Package sizing compara los requests/limits declarados de los workloads
// con su consumo histórico real, obtenido vía internal/promclient. Consume
// modelos ya poblados por internal/inventory; no recolecta datos de
// Kubernetes por sí mismo ni genera salida visual.
package sizing

import (
	"context"

	"github.com/galexbh/husk/internal/config"
	"github.com/galexbh/husk/internal/model"
	"github.com/galexbh/husk/internal/promclient"
)

// Analyzer produce un model.SizingReport combinando workloads (de
// internal/inventory) con el consumo histórico observado en Prometheus.
type Analyzer struct {
	prom *promclient.Client
	cfg  config.SizingConfig
}

// New construye un Analyzer.
func New(prom *promclient.Client, cfg config.SizingConfig) *Analyzer {
	return &Analyzer{prom: prom, cfg: cfg}
}

// Analyze produce el sizing de cada contenedor de cada workload dado.
func (a *Analyzer) Analyze(ctx context.Context, workloads []model.WorkloadSummary) (*model.SizingReport, error) {
	report := &model.SizingReport{
		Lookback:         a.cfg.Lookback,
		CPUPercentile:    a.cfg.CPUPercentile,
		MemoryPercentile: a.cfg.MemoryPercentile,
	}

	for _, w := range workloads {
		ws := model.WorkloadSizing{Kind: w.Kind, Name: w.Name, Namespace: w.Namespace}
		for _, ctr := range w.Containers {
			cs, err := a.analyzeContainer(ctx, w, ctr)
			if err != nil {
				return nil, err
			}
			ws.Containers = append(ws.Containers, cs)
		}
		report.Workloads = append(report.Workloads, ws)
	}

	return report, nil
}

func (a *Analyzer) analyzeContainer(ctx context.Context, w model.WorkloadSummary, ctr model.ContainerSummary) (model.ContainerSizing, error) {
	params := promclient.SizingQueryParams{
		Namespace: w.Namespace, OwnerKind: w.Kind, OwnerName: w.Name, Container: ctr.Name,
		Lookback: a.cfg.Lookback, CPUPercentile: a.cfg.CPUPercentile, MemPercentile: a.cfg.MemoryPercentile,
	}

	cpuVal, cpuOK, err := a.queryScalar(ctx, promclient.CPUPercentileQuery(params))
	if err != nil {
		return model.ContainerSizing{}, err
	}
	memVal, memOK, err := a.queryScalar(ctx, promclient.MemoryPercentileQuery(params))
	if err != nil {
		return model.ContainerSizing{}, err
	}
	peakVal, peakOK, err := a.queryScalar(ctx, promclient.MemoryPeakQuery(params))
	if err != nil {
		return model.ContainerSizing{}, err
	}

	return buildContainerSizing(ctr, a.cfg, observedValues{
		cpuP95: cpuVal, cpuOK: cpuOK,
		memP99: memVal, memOK: memOK,
		memPeak: peakVal, peakOK: peakOK,
	}), nil
}

func (a *Analyzer) queryScalar(ctx context.Context, query string) (float64, bool, error) {
	val, err := a.prom.Query(ctx, query)
	if err != nil {
		return 0, false, err
	}
	v, ok := promclient.ScalarOrNaN(val)
	return v, ok, nil
}
