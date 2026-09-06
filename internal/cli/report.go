package cli

import (
	"context"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/galexbh/husk/internal/capacity"
	"github.com/galexbh/husk/internal/diff"
	"github.com/galexbh/husk/internal/excel"
	"github.com/galexbh/husk/internal/history"
	"github.com/galexbh/husk/internal/huskerr"
	"github.com/galexbh/husk/internal/inventory"
	"github.com/galexbh/husk/internal/k8sclient"
	"github.com/galexbh/husk/internal/model"
	"github.com/galexbh/husk/internal/report"
	"github.com/galexbh/husk/internal/score"
	"github.com/galexbh/husk/internal/sizing"
	"github.com/galexbh/husk/internal/version"
)

func newReportCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "report",
		Short: "Genera y compara reportes consolidados",
	}

	var appendix bool
	generate := &cobra.Command{
		Use:   "generate",
		Short: "Reporte consolidado: resumen ejecutivo, hallazgos priorizados y recomendaciones",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runReportGenerate(cmd, appendix)
		},
	}
	generate.Flags().BoolVar(&appendix, "appendix", false, "incluye el snapshot completo como apéndice JSON")

	var from, to string
	diffCmd := &cobra.Command{
		Use:   "diff",
		Short: "Compara dos snapshots históricos y resalta regresiones y mejoras",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runReportDiff(cmd, from, to)
		},
	}
	diffCmd.Flags().StringVar(&from, "from", "", "snapshot inicial (JSON)")
	diffCmd.Flags().StringVar(&to, "to", "", "snapshot final (JSON)")
	_ = diffCmd.MarkFlagRequired("from")
	_ = diffCmd.MarkFlagRequired("to")

	cmd.AddCommand(generate, diffCmd)
	return cmd
}

func runReportGenerate(cmd *cobra.Command, appendix bool) error {
	ctx, cancel := context.WithTimeout(cmd.Context(), 10*time.Minute)
	defer cancel()

	logger := loggerFrom(cmd)
	cfg := configFrom(cmd)

	client, err := k8sclient.New(flagKubeconfig, flagContext)
	if err != nil {
		return err
	}
	kubeVersion, err := client.ServerVersion()
	if err != nil {
		return err
	}

	collector, err := newCollector(cmd)
	if err != nil {
		return err
	}
	inv, namespaces, err := collector.Collect(ctx, inventory.Options{Extended: cfg.Inventory.IncludeExtended})
	if err != nil {
		return err
	}
	invSummary := inventory.BuildSummary(inv, namespaces)

	promCli, promReason := newOptionalPromClient(ctx, client, logger)

	capAnalyzer := capacity.New(client, promCli, cfg.Capacity)
	capReport, err := capAnalyzer.Analyze(ctx)
	if err != nil {
		return err
	}

	drReadiness, err := assessDR(ctx, cmd)
	if err != nil {
		return err
	}

	var sizingReport *model.SizingReport
	if promCli != nil {
		workloads := make([]model.WorkloadSummary, 0, len(inv.Deployments)+len(inv.StatefulSets)+len(inv.DaemonSets))
		workloads = append(workloads, inv.Deployments...)
		workloads = append(workloads, inv.StatefulSets...)
		workloads = append(workloads, inv.DaemonSets...)

		sizingAnalyzer := sizing.New(promCli, cfg.Sizing)
		sizingReport, err = sizingAnalyzer.Analyze(ctx, workloads)
		if err != nil {
			return err
		}
	} else {
		logger.Debug("report generate continúa sin sizing", "motivo", promReason)
	}

	scoreResult := score.Compute(score.Inputs{
		Weights:      cfg.Score.Weights,
		Sizing:       sizingReport,
		SizingReason: promReason,
		Capacity:     capReport,
		DR:           drReadiness,
	})

	meta := model.Meta{
		ClusterName:       client.RESTConfig.Host,
		IsOpenShift:       client.IsOpenShift,
		KubernetesVersion: kubeVersion,
		GeneratedAt:       time.Now(),
		HuskVersion:       version.Version,
	}

	snapshot := &model.ClusterSnapshot{
		SchemaVersion:    model.SchemaVersion,
		Meta:             meta,
		Inventory:        inv,
		InventorySummary: invSummary,
		Sizing:           sizingReport,
		Capacity:         capReport,
		DR:               drReadiness,
		Score:            scoreResult,
	}

	store, err := history.New("")
	if err != nil {
		return err
	}
	historyPath, err := store.Save(snapshot)
	if err != nil {
		return err
	}
	if err := store.Prune(meta.ClusterName, cfg.History.RetainCount); err != nil {
		return err
	}

	var appendixSnapshot *model.ClusterSnapshot
	if appendix {
		appendixSnapshot = snapshot
	}
	rep := report.BuildReport(meta, invSummary, model.ReportDetail{
		Inventory: inv, Sizing: sizingReport, Capacity: capReport, DR: drReadiness,
	}, scoreResult, appendixSnapshot)
	rep.AlertCorrelation = buildAlertCorrelation(ctx, client, logger, rep.PrioritizedFindings)

	if err := writeReport(cmd, rep); err != nil {
		return err
	}
	cmd.Printf("Snapshot guardado en %s\n", historyPath)
	return nil
}

func writeReport(cmd *cobra.Command, r *model.Report) error {
	if flagOutput == "excel" {
		cfg := configFrom(cmd)
		if flagOutputFile == "" {
			return huskerr.New("--output excel requiere --output-file", "especifica la ruta del archivo .xlsx a generar", nil)
		}
		wb, err := excel.BuildReportWorkbook(r, cfg.Inventory.Excel.FreezeColumns, cfg.Inventory.Excel.HighlightRisks)
		if err != nil {
			return huskerr.New("no se pudo construir el workbook de Excel", "esto es un bug de husk; por favor reporta el issue", err)
		}
		if err := wb.SaveAs(flagOutputFile); err != nil {
			return huskerr.New("no se pudo escribir "+flagOutputFile, "verifica permisos de escritura en la ruta indicada", err)
		}
		cmd.Printf("Reporte escrito en %s\n", flagOutputFile)
		return nil
	}

	if flagOutputFile == "" {
		return report.RenderReport(cmd.OutOrStdout(), r, flagOutput)
	}

	f, err := os.Create(flagOutputFile)
	if err != nil {
		return huskerr.New("no se pudo crear "+flagOutputFile, "verifica permisos de escritura en la ruta indicada", err)
	}
	defer f.Close()

	if err := report.RenderReport(f, r, flagOutput); err != nil {
		return err
	}
	cmd.Printf("Reporte escrito en %s\n", flagOutputFile)
	return nil
}

func runReportDiff(cmd *cobra.Command, from, to string) error {
	fromSnapshot, err := history.LoadFile(from)
	if err != nil {
		return err
	}
	toSnapshot, err := history.LoadFile(to)
	if err != nil {
		return err
	}

	result := diff.Compute(fromSnapshot, toSnapshot)

	if flagOutputFile == "" {
		return diff.Render(cmd.OutOrStdout(), result, flagOutput)
	}
	f, err := os.Create(flagOutputFile)
	if err != nil {
		return huskerr.New("no se pudo crear "+flagOutputFile, "verifica permisos de escritura en la ruta indicada", err)
	}
	defer f.Close()
	if err := diff.Render(f, result, flagOutput); err != nil {
		return err
	}
	cmd.Printf("Diff escrito en %s\n", flagOutputFile)
	return nil
}
