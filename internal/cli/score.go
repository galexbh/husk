package cli

import (
	"context"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/galexbh/husk/internal/capacity"
	"github.com/galexbh/husk/internal/huskerr"
	"github.com/galexbh/husk/internal/inventory"
	"github.com/galexbh/husk/internal/k8sclient"
	"github.com/galexbh/husk/internal/model"
	"github.com/galexbh/husk/internal/report"
	"github.com/galexbh/husk/internal/score"
	"github.com/galexbh/husk/internal/sizing"
)

func newScoreCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "score",
		Short: "Score de resiliencia agregado (0-100), con desglose por dimensión",
		RunE:  runScore,
	}
}

func runScore(cmd *cobra.Command, _ []string) error {
	if flagOutput == "excel" {
		return huskerr.New(
			"husk score no soporta --output excel en esta fase",
			"usa --output table|markdown|json",
			nil,
		)
	}

	ctx, cancel := context.WithTimeout(cmd.Context(), 5*time.Minute)
	defer cancel()

	logger := loggerFrom(cmd)
	cfg := configFrom(cmd)

	client, err := k8sclient.New(flagKubeconfig, flagContext)
	if err != nil {
		return err
	}

	// DR readiness: siempre se evalúa (los sub-chequeos que no aplican a
	// Kubernetes vanilla, como etcd, ya se marcan "no aplica" internamente).
	drReadiness, err := assessDR(ctx, cmd)
	if err != nil {
		return err
	}

	// Capacity: siempre se evalúa; Prometheus es un enriquecimiento
	// opcional (no afecta si se puede calcular headroom).
	promCli, promReason := newOptionalPromClient(ctx, client, logger)
	capAnalyzer := capacity.New(client, promCli, cfg.Capacity)
	capReport, err := capAnalyzer.Analyze(ctx)
	if err != nil {
		return err
	}

	// Sizing: requiere Prometheus. Si no está disponible, la dimensión se
	// marca no disponible (su peso se redistribuye) en vez de fallar todo
	// el comando.
	var sizingReport *model.SizingReport
	if promCli != nil {
		collector, err := newCollector(cmd)
		if err != nil {
			return err
		}
		inv, _, err := collector.Collect(ctx, inventory.Options{})
		if err != nil {
			return err
		}

		workloads := make([]model.WorkloadSummary, 0, len(inv.Deployments)+len(inv.StatefulSets)+len(inv.DaemonSets))
		workloads = append(workloads, inv.Deployments...)
		workloads = append(workloads, inv.StatefulSets...)
		workloads = append(workloads, inv.DaemonSets...)

		sizingAnalyzer := sizing.New(promCli, cfg.Sizing)
		sizingReport, err = sizingAnalyzer.Analyze(ctx, workloads)
		if err != nil {
			return err
		}
	}

	result := score.Compute(score.Inputs{
		Weights:      cfg.Score.Weights,
		Sizing:       sizingReport,
		SizingReason: promReason,
		Capacity:     capReport,
		DR:           drReadiness,
	})

	return writeScoreReport(cmd, result)
}

func writeScoreReport(cmd *cobra.Command, s *model.Score) error {
	if flagOutputFile == "" {
		return report.RenderScore(cmd.OutOrStdout(), s, flagOutput)
	}

	f, err := os.Create(flagOutputFile)
	if err != nil {
		return huskerr.New("no se pudo crear "+flagOutputFile, "verifica permisos de escritura en la ruta indicada", err)
	}
	defer f.Close()

	if err := report.RenderScore(f, s, flagOutput); err != nil {
		return err
	}
	cmd.Printf("Score escrito en %s\n", flagOutputFile)
	return nil
}
