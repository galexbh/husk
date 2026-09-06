// Package archtest hace cumplir en CI la regla de arquitectura de
// CLAUDE.md: "internal/report consume modelos ya poblados; nunca llama
// directamente a Kubernetes, Prometheus o Alertmanager." Si alguien agrega
// por accidente un import de un paquete recolector a internal/report (o a
// internal/excel, que comparte esa misma regla), este test falla.
package archtest

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// renderingPackages son los paquetes que solo deben consumir modelos ya
// poblados (internal/model) y nunca recolectar datos por sí mismos.
var renderingPackages = []string{
	"internal/report",
	"internal/excel",
}

// forbiddenImports son los paquetes recolectores que renderingPackages no
// debe importar, ni directa ni transitivamente (aquí se verifica el
// import directo, que es donde ocurriría la violación real).
var forbiddenImports = []string{
	"github.com/galexbh/husk/internal/k8sclient",
	"github.com/galexbh/husk/internal/promclient",
	"github.com/galexbh/husk/internal/alertmanager",
}

// repoRoot resuelve la raíz del módulo desde este archivo de test
// (internal/archtest/arch_test.go), sin depender del directorio de trabajo
// desde el que se invoque `go test`.
func repoRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("os.Getwd: %v", err)
	}
	// internal/archtest -> sube dos niveles hasta la raíz del módulo.
	return filepath.Join(wd, "..", "..")
}

func TestReportAndExcelNeverImportCollectors(t *testing.T) {
	root := repoRoot(t)

	for _, pkgRelPath := range renderingPackages {
		pkgDir := filepath.Join(root, filepath.FromSlash(pkgRelPath))
		entries, err := os.ReadDir(pkgDir)
		if err != nil {
			t.Fatalf("no se pudo leer %s: %v", pkgDir, err)
		}

		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") {
				continue
			}
			if strings.HasSuffix(entry.Name(), "_test.go") {
				continue
			}

			filePath := filepath.Join(pkgDir, entry.Name())
			imports, err := fileImports(filePath)
			if err != nil {
				t.Fatalf("no se pudo parsear %s: %v", filePath, err)
			}

			for _, imp := range imports {
				for _, forbidden := range forbiddenImports {
					if imp == forbidden {
						t.Errorf(
							"violación de arquitectura: %s importa %s directamente — %s debe consumir solo internal/model, nunca recolectar datos",
							filepath.Join(pkgRelPath, entry.Name()), imp, pkgRelPath,
						)
					}
				}
			}
		}
	}
}

// fileImports devuelve las rutas de import de un archivo .go, vía el
// parser estándar de Go (sin necesitar cargar el paquete completo).
func fileImports(path string) ([]string, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
	if err != nil {
		return nil, err
	}

	imports := make([]string, 0, len(f.Imports))
	for _, imp := range f.Imports {
		unquoted, err := strconv.Unquote(imp.Path.Value)
		if err != nil {
			continue
		}
		imports = append(imports, unquoted)
	}
	return imports, nil
}
