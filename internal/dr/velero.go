package dr

import (
	"context"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/galexbh/husk/internal/huskerr"
	"github.com/galexbh/husk/internal/model"
)

// assessVelero recolecta los Backup y BackupStorageLocation CRs de Velero
// (instalados por OADP), de todo el cluster —no solo openshift-adp—, por
// si el usuario instaló Velero en un namespace distinto. La ausencia de
// estos CRDs no es un error: OADP podría no estar instalado del todo, lo
// que ya se refleja en assessOADP.
func (a *Assessor) assessVelero(ctx context.Context, dr *model.DRReadiness) error {
	backups, err := a.client.Dynamic.Resource(backupGVR).List(ctx, listOpts)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		return huskerr.New("no se pudo listar Backups (velero.io)", "verifica el permiso de lectura sobre backups.velero.io", err)
	}
	for _, item := range backups.Items {
		dr.Backups = append(dr.Backups, backupSummaryFrom(item))
	}

	bsls, err := a.client.Dynamic.Resource(backupStorageLocationGVR).List(ctx, listOpts)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		return huskerr.New("no se pudo listar BackupStorageLocations (velero.io)", "verifica el permiso de lectura sobre backupstoragelocations.velero.io", err)
	}
	for _, item := range bsls.Items {
		dr.BackupStorageLocations = append(dr.BackupStorageLocations, bslStatusFrom(item))
	}

	return nil
}

func backupSummaryFrom(item unstructured.Unstructured) model.BackupSummary {
	name, _, _ := unstructured.NestedString(item.Object, "metadata", "name")
	phase, _, _ := unstructured.NestedString(item.Object, "status", "phase")

	var included []string
	if raw, found, _ := unstructured.NestedStringSlice(item.Object, "spec", "includedNamespaces"); found {
		included = raw
	}

	return model.BackupSummary{
		Name:                name,
		IncludedNamespaces:  included,
		Phase:               phase,
		CompletionTimestamp: parseTimeField(item, "status", "completionTimestamp"),
		StartTimestamp:      parseTimeField(item, "status", "startTimestamp"),
	}
}

func bslStatusFrom(item unstructured.Unstructured) model.BSLStatus {
	name, _, _ := unstructured.NestedString(item.Object, "metadata", "name")
	namespace, _, _ := unstructured.NestedString(item.Object, "metadata", "namespace")
	provider, _, _ := unstructured.NestedString(item.Object, "spec", "provider")
	phase, _, _ := unstructured.NestedString(item.Object, "status", "phase")
	isDefault, _, _ := unstructured.NestedBool(item.Object, "spec", "default")

	return model.BSLStatus{
		Name: name, Namespace: namespace, Provider: provider, Phase: phase, Default: isDefault,
	}
}

// parseTimeField lee un campo string en formato RFC3339 dentro de un
// unstructured.Unstructured; devuelve el time.Time cero si el campo no
// existe o no se puede interpretar.
func parseTimeField(item unstructured.Unstructured, fields ...string) time.Time {
	raw, found, _ := unstructured.NestedString(item.Object, fields...)
	if !found || raw == "" {
		return time.Time{}
	}
	t, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return time.Time{}
	}
	return t
}
