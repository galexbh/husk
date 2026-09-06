package dr

import (
	"context"
	"sort"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/galexbh/husk/internal/huskerr"
	"github.com/galexbh/husk/internal/model"
)

// assessStorageClasses determina, para cada StorageClass efectivamente
// usada por al menos un PVC, si su provisioner tiene soporte de CSI
// snapshot (existe al menos un VolumeSnapshotClass cuyo driver coincide).
func (a *Assessor) assessStorageClasses(ctx context.Context, dr *model.DRReadiness) error {
	pvcs, err := a.client.Kubernetes.CoreV1().PersistentVolumeClaims("").List(ctx, listOpts)
	if err != nil {
		return huskerr.New("no se pudo listar PersistentVolumeClaims", "verifica el permiso de lectura sobre persistentvolumeclaims", err)
	}
	usedSet := make(map[string]bool)
	for _, pvc := range pvcs.Items {
		if pvc.Spec.StorageClassName != nil && *pvc.Spec.StorageClassName != "" {
			usedSet[*pvc.Spec.StorageClassName] = true
		}
	}

	storageClasses, err := a.client.Kubernetes.StorageV1().StorageClasses().List(ctx, listOpts)
	if err != nil {
		return huskerr.New("no se pudo listar StorageClasses", "verifica el permiso de lectura sobre storageclasses", err)
	}
	provisionerByName := make(map[string]string, len(storageClasses.Items))
	for _, sc := range storageClasses.Items {
		provisionerByName[sc.Name] = sc.Provisioner
	}

	supportedDrivers, err := a.csiSnapshotDrivers(ctx)
	if err != nil {
		return err
	}

	used := make([]string, 0, len(usedSet))
	for name := range usedSet {
		used = append(used, name)
	}
	sort.Strings(used)
	dr.StorageClassesUsed = used

	for _, name := range used {
		provisioner, ok := provisionerByName[name]
		if !ok || !supportedDrivers[provisioner] {
			dr.StorageClassesWithoutCSISnapshot = append(dr.StorageClassesWithoutCSISnapshot, name)
		}
	}

	return nil
}

// csiSnapshotDrivers lista los VolumeSnapshotClass (snapshot.storage.k8s.io/v1)
// del cluster y devuelve el conjunto de drivers (provisioners) con soporte
// de CSI snapshot. El CRD ausente (snapshot.storage.k8s.io no instalado)
// no es un error: se interpreta como "ningún provisioner con soporte".
func (a *Assessor) csiSnapshotDrivers(ctx context.Context) (map[string]bool, error) {
	drivers := make(map[string]bool)

	list, err := a.client.Dynamic.Resource(volumeSnapshotClassGVR).List(ctx, listOpts)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return drivers, nil
		}
		return nil, huskerr.New("no se pudo listar VolumeSnapshotClasses", "verifica el permiso de lectura sobre volumesnapshotclasses.snapshot.storage.k8s.io", err)
	}
	for _, item := range list.Items {
		if driver, found, _ := unstructured.NestedString(item.Object, "driver"); found && driver != "" {
			drivers[driver] = true
		}
	}
	return drivers, nil
}
