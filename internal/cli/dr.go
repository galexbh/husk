package cli

import (
	"context"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/galexbh/husk/internal/dr"
	"github.com/galexbh/husk/internal/huskerr"
	"github.com/galexbh/husk/internal/k8sclient"
	"github.com/galexbh/husk/internal/model"
	"github.com/galexbh/husk/internal/nsfilter"
	"github.com/galexbh/husk/internal/report"
)

func newDRCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dr",
		Short: "Evalúa disaster recovery readiness",
	}
	cmd.AddCommand(&cobra.Command{
		Use:   "assess",
		Short: "Semáforo de preparación DR (OADP, backups, etcd, CSI snapshots, PDBs, topology spread)",
		RunE:  runDRAssess,
	})
	return cmd
}

func runDRAssess(cmd *cobra.Command, _ []string) error {
	if flagOutput == "excel" {
		return huskerr.New(
			"husk dr assess no soporta --output excel en esta fase",
			"usa --output table|markdown|json",
			nil,
		)
	}

	ctx, cancel := context.WithTimeout(cmd.Context(), 3*time.Minute)
	defer cancel()

	drReadiness, err := assessDR(ctx, cmd)
	if err != nil {
		return err
	}

	return writeDRReport(cmd, drReadiness)
}

// assessDR construye el cliente/filtro y ejecuta la evaluación de DR
// readiness. Se reutiliza desde `husk score`.
func assessDR(ctx context.Context, cmd *cobra.Command) (*model.DRReadiness, error) {
	cfg := configFrom(cmd)

	client, err := k8sclient.New(flagKubeconfig, flagContext)
	if err != nil {
		return nil, err
	}
	filter, err := nsfilter.New(cfg.Namespaces.ExcludePatterns)
	if err != nil {
		return nil, huskerr.New("el archivo de configuración tiene un patrón de exclusión de namespaces inválido", "revisa namespaces.exclude_patterns en tu config.yaml", err)
	}

	assessor := dr.New(client, filter, cfg.DR)
	return assessor.Assess(ctx)
}

func writeDRReport(cmd *cobra.Command, r *model.DRReadiness) error {
	if flagOutputFile == "" {
		return report.RenderDR(cmd.OutOrStdout(), r, flagOutput)
	}

	f, err := os.Create(flagOutputFile)
	if err != nil {
		return huskerr.New("no se pudo crear "+flagOutputFile, "verifica permisos de escritura en la ruta indicada", err)
	}
	defer f.Close()

	if err := report.RenderDR(f, r, flagOutput); err != nil {
		return err
	}
	cmd.Printf("Reporte de DR readiness escrito en %s\n", flagOutputFile)
	return nil
}
