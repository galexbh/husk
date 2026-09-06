package cli

import (
	"context"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/galexbh/husk/internal/huskerr"
	"github.com/galexbh/husk/internal/inventory"
	"github.com/galexbh/husk/internal/k8sclient"
	"github.com/galexbh/husk/internal/model"
	"github.com/galexbh/husk/internal/promclient"
	"github.com/galexbh/husk/internal/report"
	"github.com/galexbh/husk/internal/sizing"
)

func newSizingCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sizing",
		Short: "Analiza sizing real de workloads contra su consumo histórico",
	}

	var dryRun bool
	reportCmd := &cobra.Command{
		Use:   "report",
		Short: "Tabla de sizing con recomendaciones de requests/limits",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runSizingReport(cmd, dryRun)
		},
	}
	reportCmd.Flags().BoolVar(&dryRun, "dry-run", false, "muestra el patch YAML sugerido para cada contenedor; nunca aplica cambios al cluster")
	cmd.AddCommand(reportCmd)

	return cmd
}

func runSizingReport(cmd *cobra.Command, dryRun bool) error {
	if flagOutput == "excel" {
		return huskerr.New(
			"husk sizing report no soporta --output excel en esta fase",
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
	if !client.IsOpenShift {
		return huskerr.New(
			"husk sizing report necesita Thanos Querier, disponible en el stack de monitoreo de OpenShift",
			"este comando no aplica en Kubernetes vanilla en esta fase; ejecútalo contra un cluster OpenShift",
			nil,
		)
	}

	thanosURL, err := client.ThanosQuerierURL(ctx)
	if err != nil {
		return err
	}
	token, err := client.BearerToken()
	if err != nil {
		return err
	}
	transport := promclient.DiscoverTransport(ctx, client.Kubernetes, logger)
	promCli, err := promclient.New(thanosURL, token, transport, logger)
	if err != nil {
		return err
	}

	collector, err := newCollector(cmd)
	if err != nil {
		return err
	}
	inv, _, err := collector.Collect(ctx, inventory.Options{Namespace: flagNamespace})
	if err != nil {
		return err
	}

	workloads := make([]model.WorkloadSummary, 0, len(inv.Deployments)+len(inv.StatefulSets)+len(inv.DaemonSets))
	workloads = append(workloads, inv.Deployments...)
	workloads = append(workloads, inv.StatefulSets...)
	workloads = append(workloads, inv.DaemonSets...)

	analyzer := sizing.New(promCli, cfg.Sizing)
	sizingReport, err := analyzer.Analyze(ctx, workloads)
	if err != nil {
		return err
	}

	if err := writeSizingReport(cmd, sizingReport); err != nil {
		return err
	}

	if dryRun {
		patches := sizing.BuildDryRunPatches(sizingReport)
		return report.RenderDryRunPatches(cmd.OutOrStdout(), patches)
	}
	return nil
}

func writeSizingReport(cmd *cobra.Command, r *model.SizingReport) error {
	if flagOutputFile == "" {
		return report.RenderSizing(cmd.OutOrStdout(), r, flagOutput)
	}

	f, err := os.Create(flagOutputFile)
	if err != nil {
		return huskerr.New("no se pudo crear "+flagOutputFile, "verifica permisos de escritura en la ruta indicada", err)
	}
	defer f.Close()

	if err := report.RenderSizing(f, r, flagOutput); err != nil {
		return err
	}
	cmd.Printf("Reporte de sizing escrito en %s\n", flagOutputFile)
	return nil
}
