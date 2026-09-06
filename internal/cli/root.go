// Package cli implementa la capa Cobra de husk: parseo de flags y wiring
// hacia los colectores/analizadores. No contiene lógica de negocio — cada
// subcomando expone NewXxxCommand() y usa RunE, nunca Run.
package cli

import (
	"context"
	"log/slog"
	"os"

	"github.com/spf13/cobra"

	"github.com/galexbh/husk/internal/config"
	"github.com/galexbh/husk/internal/huskerr"
	"github.com/galexbh/husk/internal/logging"
)

type ctxKey string

const (
	ctxKeyConfig ctxKey = "husk-config"
	ctxKeyLogger ctxKey = "husk-logger"
)

// Flags globales, heredados por todos los subcomandos desde rootCmd.
var (
	flagKubeconfig string
	flagContext    string
	flagNamespace  string
	flagOutput     string
	flagOutputFile string
	flagVerbose    bool
	flagConfigPath string
)

// NewRootCommand construye el comando raíz de husk con todos los
// subcomandos registrados. root.go solo declara flags persistentes y
// registra hijos; ningún subcomando vive en este archivo.
func NewRootCommand() *cobra.Command {
	root := &cobra.Command{
		Use:   "husk",
		Short: "husk analiza sizing, DR readiness e inventario de clusters Kubernetes/OpenShift",
		Long: `husk es un CLI de solo lectura para clusters Kubernetes y OpenShift.

Genera reportes de sizing real contra consumo histórico, preparación de
disaster recovery, capacity planning, un score de resiliencia agregado e
inventario completo de recursos. Usa el kubeconfig activo del usuario; no
requiere credenciales separadas.`,
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			return bootstrap(cmd)
		},
	}

	root.PersistentFlags().StringVar(&flagKubeconfig, "kubeconfig", "", "ruta al archivo kubeconfig (por defecto $KUBECONFIG o ~/.kube/config)")
	root.PersistentFlags().StringVar(&flagContext, "context", "", "contexto del kubeconfig a usar")
	root.PersistentFlags().StringVarP(&flagNamespace, "namespace", "n", "", "namespace por defecto para operaciones que lo requieran")
	root.PersistentFlags().StringVarP(&flagOutput, "output", "o", "table", "formato de salida: json|markdown|table|excel")
	root.PersistentFlags().StringVar(&flagOutputFile, "output-file", "", "archivo de salida (requerido para excel, opcional para el resto)")
	root.PersistentFlags().BoolVarP(&flagVerbose, "verbose", "v", false, "habilita logging de nivel debug")
	root.PersistentFlags().StringVar(&flagConfigPath, "config", "", "ruta a un archivo de configuración YAML (por defecto ~/.husk/config.yaml)")

	root.AddCommand(
		newVersionCommand(),
		newConnectCommand(),
		newInitCommand(),
		newInventoryCommand(),
		newSizingCommand(),
		newCapacityCommand(),
		newDRCommand(),
		newScoreCommand(),
		newReportCommand(),
		newExportCommand(),
	)

	return root
}

// bootstrap prepara el logger y la configuración para el comando en
// ejecución y los deja disponibles vía el contexto del comando.
func bootstrap(cmd *cobra.Command) error {
	logger := logging.New(flagVerbose)

	cfg, err := config.Load(flagConfigPath)
	if err != nil {
		return err
	}

	ctx := context.WithValue(cmd.Context(), ctxKeyLogger, logger)
	ctx = context.WithValue(ctx, ctxKeyConfig, cfg)
	cmd.SetContext(ctx)

	logger.Debug("husk bootstrap completado",
		"kubeconfig", flagKubeconfig,
		"context", flagContext,
		"namespace", flagNamespace,
		"output", flagOutput,
		"config", flagConfigPath,
	)
	return nil
}

// loggerFrom recupera el logger preparado en bootstrap.
func loggerFrom(cmd *cobra.Command) *slog.Logger {
	if l, ok := cmd.Context().Value(ctxKeyLogger).(*slog.Logger); ok {
		return l
	}
	return logging.New(false)
}

// configFrom recupera la configuración preparada en bootstrap.
func configFrom(cmd *cobra.Command) *config.Config {
	if c, ok := cmd.Context().Value(ctxKeyConfig).(*config.Config); ok {
		return c
	}
	return config.Default()
}

// Execute corre el comando raíz y maneja la salida de error amigable.
func Execute() {
	root := NewRootCommand()
	if err := root.ExecuteContext(context.Background()); err != nil {
		huskerr.PrintFriendly(os.Stderr, err, flagVerbose)
		os.Exit(1)
	}
}
