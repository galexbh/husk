package cli

import (
	"context"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/galexbh/husk/internal/capacity"
	"github.com/galexbh/husk/internal/huskerr"
	"github.com/galexbh/husk/internal/k8sclient"
	"github.com/galexbh/husk/internal/model"
	"github.com/galexbh/husk/internal/report"
)

func newCapacityCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "capacity",
		Short: "Analiza capacity planning del cluster",
	}
	cmd.AddCommand(&cobra.Command{
		Use:   "nodes",
		Short: "Allocatable vs requests por nodo, headroom y riesgos de concentración de carga",
		RunE:  runCapacityNodes,
	})
	return cmd
}

func runCapacityNodes(cmd *cobra.Command, _ []string) error {
	if flagOutput == "excel" {
		return huskerr.New(
			"husk capacity nodes no soporta --output excel en esta fase",
			"usa --output table|markdown|json",
			nil,
		)
	}

	ctx, cancel := context.WithTimeout(cmd.Context(), 2*time.Minute)
	defer cancel()

	logger := loggerFrom(cmd)
	cfg := configFrom(cmd)

	client, err := k8sclient.New(flagKubeconfig, flagContext)
	if err != nil {
		return err
	}

	// El consumo histórico es un enriquecimiento opcional: si no está
	// disponible, el reporte de capacity se genera igual, solo sin esos
	// campos — nunca falla por esto.
	promCli, reason := newOptionalPromClient(ctx, client, logger)
	if promCli == nil {
		logger.Debug("capacity continúa sin consumo histórico", "motivo", reason)
	}

	analyzer := capacity.New(client, promCli, cfg.Capacity)
	capacityReport, err := analyzer.Analyze(ctx)
	if err != nil {
		return err
	}

	return writeCapacityReport(cmd, capacityReport)
}

func writeCapacityReport(cmd *cobra.Command, r *model.CapacityReport) error {
	if flagOutputFile == "" {
		return report.RenderCapacity(cmd.OutOrStdout(), r, flagOutput)
	}

	f, err := os.Create(flagOutputFile)
	if err != nil {
		return huskerr.New("no se pudo crear "+flagOutputFile, "verifica permisos de escritura en la ruta indicada", err)
	}
	defer f.Close()

	if err := report.RenderCapacity(f, r, flagOutput); err != nil {
		return err
	}
	cmd.Printf("Reporte de capacity escrito en %s\n", flagOutputFile)
	return nil
}
