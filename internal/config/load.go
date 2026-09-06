package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"

	"github.com/galexbh/husk/internal/huskerr"
)

// Load construye la configuración efectiva: los defaults combinados con un
// archivo YAML opcional. path tiene prioridad; si está vacío, se usa
// ~/.husk/config.yaml cuando existe. Que no exista el archivo no es un
// error cuando path está vacío; se aplican los defaults. Un path explícito
// que no se puede leer sí es un error.
func Load(path string) (*Config, error) {
	cfg := Default()

	resolved := path
	if resolved == "" {
		if home, err := os.UserHomeDir(); err == nil {
			candidate := filepath.Join(home, ".husk", "config.yaml")
			if _, statErr := os.Stat(candidate); statErr == nil {
				resolved = candidate
			}
		}
	}

	if resolved == "" {
		return cfg, nil
	}

	v := viper.New()
	v.SetConfigFile(resolved)
	if err := v.ReadInConfig(); err != nil {
		return nil, huskerr.New(
			fmt.Sprintf("no se pudo leer el archivo de configuración %s", resolved),
			"verifica que la ruta exista y que el YAML sea válido",
			err,
		)
	}
	if err := v.Unmarshal(cfg); err != nil {
		return nil, huskerr.New(
			fmt.Sprintf("no se pudo interpretar %s", resolved),
			"revisa que las claves y tipos coincidan con el esquema documentado en docs/configuration.md",
			err,
		)
	}
	return cfg, nil
}
