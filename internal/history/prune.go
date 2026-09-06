package history

import (
	"os"

	"github.com/galexbh/husk/internal/huskerr"
)

// Prune conserva los retain snapshots más recientes de un cluster y borra
// el resto. retain <= 0 significa "sin límite" (no borra nada).
func (s *Store) Prune(clusterName string, retain int) error {
	if retain <= 0 {
		return nil
	}

	paths, err := s.List(clusterName)
	if err != nil {
		return err
	}
	if len(paths) <= retain {
		return nil
	}

	toDelete := paths[:len(paths)-retain]
	for _, p := range toDelete {
		if err := os.Remove(p); err != nil && !os.IsNotExist(err) {
			return huskerr.New("no se pudo podar el historial: no se pudo borrar "+p, "verifica permisos de escritura en el directorio de historial", err)
		}
	}
	return nil
}
