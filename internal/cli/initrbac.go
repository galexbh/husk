package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/galexbh/husk/internal/huskerr"
	"github.com/galexbh/husk/internal/rbac"
)

func newInitCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Genera manifiestos de apoyo para ejecutar husk",
	}
	cmd.AddCommand(newInitRBACCommand())
	return cmd
}

func newInitRBACCommand() *cobra.Command {
	var name, saName, saNamespace string

	cmd := &cobra.Command{
		Use:   "rbac",
		Short: "Genera el ClusterRole y ClusterRoleBinding mínimos de solo lectura para un ServiceAccount",
		Long: `Genera el ClusterRole y ClusterRoleBinding de solo lectura (get/list/watch)
necesarios para ejecutar husk mediante un ServiceAccount, por ejemplo en un
CronJob o pipeline de CI/CD. No es necesario para uso local con un usuario
ya autenticado por 'oc login' o 'kubectl'.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			manifest, err := rbac.Render(rbac.Params{
				Name:               name,
				ServiceAccountName: saName,
				Namespace:          saNamespace,
			})
			if err != nil {
				return huskerr.New("no se pudo generar el manifiesto de RBAC", "esto es un bug de husk; por favor reporta el issue", err)
			}

			if flagOutputFile != "" {
				if err := os.WriteFile(flagOutputFile, []byte(manifest), 0o600); err != nil {
					return huskerr.New(fmt.Sprintf("no se pudo escribir %s", flagOutputFile), "verifica permisos de escritura en la ruta indicada", err)
				}
				fmt.Fprintf(cmd.OutOrStdout(), "RBAC escrito en %s\n", flagOutputFile)
				return nil
			}

			fmt.Fprint(cmd.OutOrStdout(), manifest)
			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "husk-reader", "nombre del ClusterRole/ClusterRoleBinding a generar")
	cmd.Flags().StringVar(&saName, "service-account", "husk", "nombre del ServiceAccount que usará el rol")
	cmd.Flags().StringVar(&saNamespace, "service-account-namespace", "husk", "namespace del ServiceAccount")

	return cmd
}
