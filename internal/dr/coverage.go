package dr

import (
	"time"

	"github.com/galexbh/husk/internal/model"
)

// assessBackupCoverage es una función pura (opera solo sobre los campos ya
// recolectados de dr): para cada namespace de aplicación, busca el backup
// completado más reciente que lo incluya y determina si hay uno, y si su
// antigüedad está dentro de dr.backup_max_age.
func (a *Assessor) assessBackupCoverage(dr *model.DRReadiness) {
	maxAge, err := time.ParseDuration(a.cfg.BackupMaxAge)
	if err != nil {
		maxAge = 24 * time.Hour
	}
	now := time.Now()

	for _, ns := range dr.ApplicationNamespaces {
		latest, found := latestCompletedBackupFor(dr.Backups, ns)
		switch {
		case !found:
			dr.NamespacesWithoutBackup = append(dr.NamespacesWithoutBackup, ns)
		case now.Sub(latest) > maxAge:
			dr.NamespacesWithStaleBackup = append(dr.NamespacesWithStaleBackup, ns)
		}
	}
}

// latestCompletedBackupFor devuelve el CompletionTimestamp más reciente
// entre los backups completados que incluyen ns (explícitamente, o porque
// el backup no restringe namespaces — comportamiento por defecto de
// Velero cuando spec.includedNamespaces está vacío).
func latestCompletedBackupFor(backups []model.BackupSummary, ns string) (time.Time, bool) {
	var latest time.Time
	found := false

	for _, b := range backups {
		if b.Phase != "Completed" {
			continue
		}
		if !backupIncludesNamespace(b, ns) {
			continue
		}
		if b.CompletionTimestamp.After(latest) {
			latest = b.CompletionTimestamp
			found = true
		}
	}

	return latest, found
}

func backupIncludesNamespace(b model.BackupSummary, ns string) bool {
	if len(b.IncludedNamespaces) == 0 {
		return true
	}
	for _, n := range b.IncludedNamespaces {
		if n == "*" || n == ns {
			return true
		}
	}
	return false
}
