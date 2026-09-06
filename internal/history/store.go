// Package history persiste snapshots de husk en ~/.husk/history/ para que
// `report diff` funcione sin gestión manual de archivos. No recolecta
// datos de Kubernetes ni genera salida visual: solo lee/escribe JSON en
// disco.
package history

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"time"

	"github.com/galexbh/husk/internal/huskerr"
	"github.com/galexbh/husk/internal/model"
)

const timestampFormat = "20060102-150405"

// nonFilenameSafe son los caracteres que se reemplazan al derivar un
// nombre de directorio a partir del nombre del cluster (que puede ser una
// URL o contener caracteres no válidos en un nombre de archivo, sobre todo
// en Windows).
var nonFilenameSafe = regexp.MustCompile(`[^A-Za-z0-9._-]+`)

// Store guarda y lee snapshots bajo un directorio base, uno por cluster.
type Store struct {
	baseDir string
}

// New construye un Store. Si baseDir está vacío, usa ~/.husk/history.
func New(baseDir string) (*Store, error) {
	if baseDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, huskerr.New("no se pudo resolver el directorio home del usuario", "especifica un directorio de historial explícito", err)
		}
		baseDir = filepath.Join(home, ".husk", "history")
	}
	return &Store{baseDir: baseDir}, nil
}

// clusterDirName deriva un nombre de directorio seguro para archivos a
// partir del nombre del cluster (puede ser una URL de API server).
func clusterDirName(clusterName string) string {
	if clusterName == "" {
		clusterName = "default"
	}
	safe := nonFilenameSafe.ReplaceAllString(clusterName, "-")
	if safe == "" {
		safe = "default"
	}
	return safe
}

// Save escribe snapshot como un nuevo archivo JSON bajo
// <base>/<cluster>/<timestamp>.json y devuelve la ruta escrita.
func (s *Store) Save(snapshot *model.ClusterSnapshot) (string, error) {
	dir := filepath.Join(s.baseDir, clusterDirName(snapshot.Meta.ClusterName))
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return "", huskerr.New("no se pudo crear el directorio de historial "+dir, "verifica permisos de escritura en "+s.baseDir, err)
	}

	ts := snapshot.Meta.GeneratedAt
	if ts.IsZero() {
		ts = time.Now()
	}
	path := filepath.Join(dir, ts.UTC().Format(timestampFormat)+".json")

	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return "", huskerr.New("no se pudo serializar el snapshot", "esto es un bug de husk; por favor reporta el issue", err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return "", huskerr.New("no se pudo escribir "+path, "verifica permisos de escritura en "+dir, err)
	}

	return path, nil
}

// List devuelve las rutas de los snapshots guardados para un cluster,
// ordenadas de más antiguo a más reciente (por nombre de archivo, que es
// el timestamp).
func (s *Store) List(clusterName string) ([]string, error) {
	dir := filepath.Join(s.baseDir, clusterDirName(clusterName))
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, huskerr.New("no se pudo listar el historial en "+dir, "verifica permisos de lectura", err)
	}

	paths := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".json" {
			continue
		}
		paths = append(paths, filepath.Join(dir, e.Name()))
	}
	sort.Strings(paths)
	return paths, nil
}

// Latest devuelve la ruta del snapshot más reciente de un cluster, o "" si
// no hay ninguno.
func (s *Store) Latest(clusterName string) (string, error) {
	paths, err := s.List(clusterName)
	if err != nil {
		return "", err
	}
	if len(paths) == 0 {
		return "", nil
	}
	return paths[len(paths)-1], nil
}

// LoadFile carga un ClusterSnapshot desde una ruta arbitraria (usado tanto
// por el historial como por `report diff --from/--to`, que aceptan
// cualquier archivo JSON, no solo los del historial).
func LoadFile(path string) (*model.ClusterSnapshot, error) {
	data, err := os.ReadFile(path) //nolint:gosec // path es explícitamente provisto por el usuario (--from/--to o el propio historial de husk); es el propósito de esta función, no una vulnerabilidad
	if err != nil {
		return nil, huskerr.New("no se pudo leer "+path, "verifica que la ruta exista y sea legible", err)
	}
	var snapshot model.ClusterSnapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return nil, huskerr.New("no se pudo interpretar "+path+" como un snapshot de husk", "verifica que el archivo sea un JSON válido generado por husk (schemaVersion "+model.SchemaVersion+")", err)
	}
	if snapshot.SchemaVersion != "" && snapshot.SchemaVersion != model.SchemaVersion {
		return nil, huskerr.New(
			"el snapshot "+path+" tiene una versión de esquema incompatible ("+snapshot.SchemaVersion+", se esperaba "+model.SchemaVersion+")",
			"regenera el snapshot con esta versión de husk",
			nil,
		)
	}
	return &snapshot, nil
}
