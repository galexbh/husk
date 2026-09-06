package cli

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
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

// reportThanosHealth realiza un chequeo mínimo (best-effort) contra la route
// de Thanos Querier. Este es un probe simple; el cliente completo de
// Prometheus (descubrimiento, manejo de TLS, queries tipadas) se construye
// en la Fase 2 (internal/promclient).
func reportThanosHealth(ctx context.Context, out io.Writer, client *k8sclient.Client, logger interface {
	Debug(msg string, args ...any)
}) {
	if !client.IsOpenShift {
		fmt.Fprintln(out, "[N/A]    Thanos Querier: no aplica (cluster no es OpenShift)")
		return
	}

	url, err := client.ThanosQuerierURL(ctx)
	if err != nil {
		fmt.Fprintf(out, "[FALTA]  Thanos Querier: %v\n", err)
		return
	}

	token, err := client.BearerToken()
	if err != nil {
		fmt.Fprintf(out, "[FALTA]  Thanos Querier: %v\n", err)
		return
	}

	httpClient := &http.Client{
		Timeout: 5 * time.Second,
		// La route de Thanos no comparte CA con el API server; el cliente
		// completo de Prometheus (Fase 2) resolverá la validación de TLS
		// correctamente. Este probe mínimo solo confirma alcance y auth.
		Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}, //nolint:gosec
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url+"/-/healthy", nil)
	if err != nil {
		fmt.Fprintf(out, "[FALTA]  Thanos Querier: %v\n", err)
		return
	}
	req.Header.Set("Authorization", "Bearer "+token)

	logger.Debug("probing thanos querier", "url", url)
	resp, err := httpClient.Do(req)
	if err != nil {
		fmt.Fprintf(out, "[FALTA]  Thanos Querier no alcanzable en %s: %v\n", url, err)
		return
	}
	defer resp.Body.Close()

	switch {
	case resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusUnauthorized:
		fmt.Fprintf(out, "[FALTA]  Thanos Querier respondió %d en %s — falta el rol cluster-monitoring-view\n", resp.StatusCode, url)
	case resp.StatusCode >= 300:
		fmt.Fprintf(out, "[FALTA]  Thanos Querier respondió %d en %s\n", resp.StatusCode, url)
	default:
		fmt.Fprintf(out, "[OK]     Thanos Querier alcanzable en %s\n", url)
	}
}
