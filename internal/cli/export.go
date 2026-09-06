package cli

import (
	"encoding/json"
	"os"

	"github.com/spf13/cobra"

	"github.com/galexbh/husk/internal/grafana"
	"github.com/galexbh/husk/internal/huskerr"
)

func newExportCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "export",
		Short: "Exporta artefactos derivados de los reportes de husk",
	}
	cmd.AddCommand(newExportGrafanaDashboardCommand())
	return cmd
}

func newExportGrafanaDashboardCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "grafana-dashboard",
		Short: "Genera un dashboard de Grafana con las métricas de sizing/capacity que usa husk",
		Long: `Genera un dashboard de Grafana (JSON exportable con __inputs) con paneles
de CPU/memoria por namespace y por nodo, más alertas activas — las mismas
familias de métricas PromQL que usan 'husk sizing report' y
'husk capacity nodes'. Al importarlo en Grafana, se pide elegir el
datasource de Prometheus/Thanos.`,
		RunE: runExportGrafanaDashboard,
	}
}

func runExportGrafanaDashboard(cmd *cobra.Command, _ []string) error {
	dashboard := grafana.BuildDashboard()

	data, err := json.MarshalIndent(dashboard, "", "  ")
	if err != nil {
		return huskerr.New("no se pudo serializar el dashboard", "esto es un bug de husk; por favor reporta el issue", err)
	}
	data = append(data, '\n')

	if flagOutputFile == "" {
		_, err := cmd.OutOrStdout().Write(data)
		return err
	}

	if err := os.WriteFile(flagOutputFile, data, 0o600); err != nil {
		return huskerr.New("no se pudo escribir "+flagOutputFile, "verifica permisos de escritura en la ruta indicada", err)
	}
	cmd.Printf("Dashboard escrito en %s\n", flagOutputFile)
	return nil
}
