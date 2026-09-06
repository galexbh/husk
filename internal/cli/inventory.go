package cli

import (
	"context"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/galexbh/husk/internal/excel"
	"github.com/galexbh/husk/internal/huskerr"
	"github.com/galexbh/husk/internal/inventory"
	"github.com/galexbh/husk/internal/k8sclient"
	"github.com/galexbh/husk/internal/model"
	"github.com/galexbh/husk/internal/nsfilter"
	"github.com/galexbh/husk/internal/report"
)

func newInventoryCommand() *cobra.Command {
	var extended bool

	cmd := &cobra.Command{
		Use:   "inventory",
		Short: "Genera un inventario completo de los recursos del cluster",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runInventory(cmd, extended)
		},
	}
	cmd.Flags().BoolVar(&extended, "extended", false, "incluye RBAC, NetworkPolicies, PDBs, ResourceQuotas, LimitRanges, HPAs, Ingresses/Routes")
	cmd.AddCommand(newInventorySummaryCommand())
	return cmd
}

func newInventorySummaryCommand() *cobra.Command {
	var extended bool

	cmd := &cobra.Command{
		Use:   "summary",
		Short: "Versión ejecutiva del inventario, con métricas agregadas y hallazgos de riesgo",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runInventorySummary(cmd, extended)
		},
	}
	cmd.Flags().BoolVar(&extended, "extended", false, "incluye RBAC, NetworkPolicies, PDBs, ResourceQuotas, LimitRanges, HPAs, Ingresses/Routes (necesario para el hallazgo de namespaces sin ResourceQuota)")
	return cmd
}

// newCollector prepara el cliente de Kubernetes y el filtro de namespaces
// compartidos por `inventory` e `inventory summary`.
func newCollector(cmd *cobra.Command) (*inventory.Collector, error) {
	cfg := configFrom(cmd)

	client, err := k8sclient.New(flagKubeconfig, flagContext)
	if err != nil {
		return nil, err
	}

	filter, err := nsfilter.New(cfg.Namespaces.ExcludePatterns)
	if err != nil {
		return nil, huskerr.New("el archivo de configuración tiene un patrón de exclusión de namespaces inválido", "revisa namespaces.exclude_patterns en tu config.yaml", err)
	}

	return inventory.New(client, filter), nil
}

func runInventory(cmd *cobra.Command, extended bool) error {
	if flagOutput == "excel" && flagOutputFile == "" {
		return huskerr.New(
			"--output excel requiere --output-file",
			"especifica la ruta del archivo .xlsx a generar, por ejemplo: --output excel --output-file inventario.xlsx",
			nil,
		)
	}

	ctx, cancel := context.WithTimeout(cmd.Context(), 2*time.Minute)
	defer cancel()

	collector, err := newCollector(cmd)
	if err != nil {
		return err
	}

	inv, _, err := collector.Collect(ctx, inventory.Options{Namespace: flagNamespace, Extended: extended})
	if err != nil {
		return err
	}

	if flagOutput == "excel" {
		return writeInventoryExcel(cmd, inv)
	}

	return writeInventoryText(cmd, inv)
}

func runInventorySummary(cmd *cobra.Command, extended bool) error {
	if flagOutput == "excel" {
		return huskerr.New(
			"husk inventory summary no soporta --output excel",
			"usa `husk inventory --output excel` para el detalle completo en Excel, o --output table|markdown|json para el resumen",
			nil,
		)
	}

	ctx, cancel := context.WithTimeout(cmd.Context(), 2*time.Minute)
	defer cancel()

	collector, err := newCollector(cmd)
	if err != nil {
		return err
	}

	inv, namespaces, err := collector.Collect(ctx, inventory.Options{Namespace: flagNamespace, Extended: extended})
	if err != nil {
		return err
	}

	summary := inventory.BuildSummary(inv, namespaces)
	return writeInventorySummaryText(cmd, summary)
}

func writeInventoryText(cmd *cobra.Command, inv *model.Inventory) error {
	if flagOutputFile == "" {
		return report.RenderInventory(cmd.OutOrStdout(), inv, flagOutput)
	}

	f, err := os.Create(flagOutputFile)
	if err != nil {
		return huskerr.New("no se pudo crear "+flagOutputFile, "verifica permisos de escritura en la ruta indicada", err)
	}
	defer f.Close()

	if err := report.RenderInventory(f, inv, flagOutput); err != nil {
		return err
	}
	cmd.Printf("Inventario escrito en %s\n", flagOutputFile)
	return nil
}

func writeInventorySummaryText(cmd *cobra.Command, summary *model.InventorySummary) error {
	if flagOutputFile == "" {
		return report.RenderInventorySummary(cmd.OutOrStdout(), summary, flagOutput)
	}

	f, err := os.Create(flagOutputFile)
	if err != nil {
		return huskerr.New("no se pudo crear "+flagOutputFile, "verifica permisos de escritura en la ruta indicada", err)
	}
	defer f.Close()

	if err := report.RenderInventorySummary(f, summary, flagOutput); err != nil {
		return err
	}
	cmd.Printf("Resumen escrito en %s\n", flagOutputFile)
	return nil
}

func writeInventoryExcel(cmd *cobra.Command, inv *model.Inventory) error {
	cfg := configFrom(cmd)

	wb, err := excel.BuildInventoryWorkbook(inv, cfg.Inventory.Excel.FreezeColumns, cfg.Inventory.Excel.HighlightRisks)
	if err != nil {
		return huskerr.New("no se pudo construir el workbook de Excel", "esto es un bug de husk; por favor reporta el issue", err)
	}

	if err := wb.SaveAs(flagOutputFile); err != nil {
		return huskerr.New("no se pudo escribir "+flagOutputFile, "verifica permisos de escritura en la ruta indicada", err)
	}

	cmd.Printf("Inventario escrito en %s\n", flagOutputFile)
	return nil
}
