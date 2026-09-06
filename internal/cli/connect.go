package cli

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"time"

	"github.com/spf13/cobra"

	"github.com/galexbh/husk/internal/huskerr"
	"github.com/galexbh/husk/internal/k8sclient"
)

func newConnectCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "connect",
		Short: "Valida conectividad hacia el cluster y sus dependencias",
	}
	cmd.AddCommand(newConnectHealthCommand())
	return cmd
}

func newConnectHealthCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "health",
		Short: "Valida API server, tipo de cluster, Prometheus/Thanos y permisos mínimos de lectura",
		RunE:  runConnectHealth,
	}
}

func runConnectHealth(cmd *cobra.Command, _ []string) error {
	ctx, cancel := context.WithTimeout(cmd.Context(), 10*time.Second)
	defer cancel()

	logger := loggerFrom(cmd)
	out := cmd.OutOrStdout()

	client, err := k8sclient.New(flagKubeconfig, flagContext)
	if err != nil {
		return err
	}

	fmt.Fprintln(out, "husk connect health")
	fmt.Fprintln(out, "====================")

	v, err := client.ServerVersion()
	if err != nil {
		return err
	}
	fmt.Fprintf(out, "[OK]     API server alcanzable — versión %s\n", v)

	if client.IsOpenShift {
		fmt.Fprintln(out, "[OK]     cluster detectado: OpenShift")
	} else {
		fmt.Fprintln(out, "[OK]     cluster detectado: Kubernetes vanilla")
	}
	logger.Debug("conectividad validada", "isOpenShift", client.IsOpenShift)

	reportThanosHealth(ctx, out, client, logger)

	failures := 0
	for _, r := range client.ProbeReadAccess(ctx) {
		if r.OK {
			fmt.Fprintf(out, "[OK]     permiso de lectura: %s\n", r.Resource)
			continue
		}
		failures++
		fmt.Fprintf(out, "[FALTA]  permiso de lectura: %s (%v)\n", r.Resource, r.Err)
	}

	if failures > 0 {
		return huskerr.New(
			fmt.Sprintf("faltan permisos de lectura sobre %d tipo(s) de recurso", failures),
			"ejecuta `husk init rbac` para generar un ClusterRole/ClusterRoleBinding de solo lectura y aplícalo para tu usuario o ServiceAccount",
			nil,
		)
	}

	fmt.Fprintln(out, "\nhusk connect health: todo en orden.")
	return nil
}

// reportThanosHealth valida Thanos Querier ejecutando una consulta PromQL
// mínima a través de newOptionalPromClient — el mismo constructor que usan
// sizing/capacity/score. Antes golpeaba /-/healthy directamente: esa ruta
// puede no estar expuesta por la route/oauth-proxy de Thanos Querier en
// OpenShift aunque /api/v1/query funcione con normalidad, lo que producía
// un falso negativo aquí mientras el resto de la CLI sí obtenía datos.
func reportThanosHealth(ctx context.Context, out io.Writer, client *k8sclient.Client, logger *slog.Logger) {
	if !client.IsOpenShift {
		fmt.Fprintln(out, "[N/A]    Thanos Querier: no aplica (cluster no es OpenShift)")
		return
	}

	url, urlErr := client.ThanosQuerierURL(ctx)

	promCli, reason := newOptionalPromClient(ctx, client, logger)
	if promCli == nil {
		fmt.Fprintf(out, "[FALTA]  Thanos Querier: %s\n", reason)
		return
	}

	if _, err := promCli.Query(ctx, "vector(1)"); err != nil {
		if urlErr == nil {
			fmt.Fprintf(out, "[FALTA]  Thanos Querier no respondió a una consulta PromQL de prueba en %s: %v\n", url, err)
		} else {
			fmt.Fprintf(out, "[FALTA]  Thanos Querier no respondió a una consulta PromQL de prueba: %v\n", err)
		}
		return
	}

	if urlErr == nil {
		fmt.Fprintf(out, "[OK]     Thanos Querier alcanzable en %s\n", url)
	} else {
		fmt.Fprintln(out, "[OK]     Thanos Querier alcanzable")
	}
}
