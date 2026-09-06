package history

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/galexbh/husk/internal/model"
)

func snapshotAt(clusterName string, t time.Time, score float64) *model.ClusterSnapshot {
	return &model.ClusterSnapshot{
		SchemaVersion: model.SchemaVersion,
		Meta:          model.Meta{ClusterName: clusterName, GeneratedAt: t},
		Score:         &model.Score{Total: score, GeneratedAt: t},
	}
}

func TestStore_SaveListLatestPrune(t *testing.T) {
	dir := t.TempDir()
	store, err := New(dir)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	var paths []string
	for i := 0; i < 5; i++ {
		snap := snapshotAt("https://api.example.com:6443", base.Add(time.Duration(i)*time.Hour), float64(80+i))
		p, err := store.Save(snap)
		if err != nil {
			t.Fatalf("Save: %v", err)
		}
		paths = append(paths, p)
	}

	list, err := store.List("https://api.example.com:6443")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 5 {
		t.Fatalf("len(List) = %d, want 5", len(list))
	}

	latest, err := store.Latest("https://api.example.com:6443")
	if err != nil {
		t.Fatalf("Latest: %v", err)
	}
	if latest != paths[len(paths)-1] {
		t.Errorf("Latest = %s, want %s", latest, paths[len(paths)-1])
	}

	loaded, err := LoadFile(latest)
	if err != nil {
		t.Fatalf("LoadFile: %v", err)
	}
	if loaded.Score.Total != 84 {
		t.Errorf("Score.Total cargado = %v, want 84", loaded.Score.Total)
	}

	if err := store.Prune("https://api.example.com:6443", 2); err != nil {
		t.Fatalf("Prune: %v", err)
	}
	afterPrune, err := store.List("https://api.example.com:6443")
	if err != nil {
		t.Fatalf("List tras Prune: %v", err)
	}
	if len(afterPrune) != 2 {
		t.Fatalf("len(List) tras Prune = %d, want 2", len(afterPrune))
	}
	if filepath.Base(afterPrune[len(afterPrune)-1]) != filepath.Base(paths[len(paths)-1]) {
		t.Error("Prune debería conservar los snapshots más recientes")
	}
}

func TestStore_ListEmptyCluster(t *testing.T) {
	store, err := New(t.TempDir())
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	list, err := store.List("cluster-que-no-existe")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 0 {
		t.Errorf("List = %v, want vacío", list)
	}
}

func TestLoadFile_IncompatibleSchemaVersion(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.json")
	content := `{"schemaVersion": "999", "meta": {}}`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("os.WriteFile: %v", err)
	}
	if _, err := LoadFile(path); err == nil {
		t.Error("se esperaba un error por versión de esquema incompatible")
	}
}
