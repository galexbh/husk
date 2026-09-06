package inventory

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"

	"github.com/galexbh/husk/internal/huskerr"
	"github.com/galexbh/husk/internal/model"
)

const defaultStorageClassAnnotation = "storageclass.kubernetes.io/is-default-class"

// collectStorage recolecta PersistentVolumeClaims por namespace y
// StorageClasses (cluster-scoped).
func (c *Collector) collectStorage(ctx context.Context, inv *model.Inventory, namespaces []string) error {
	for _, ns := range namespaces {
		pvcs, err := c.client.Kubernetes.CoreV1().PersistentVolumeClaims(ns).List(ctx, listOpts)
		if err != nil {
			return huskerr.New("no se pudo listar PersistentVolumeClaims en "+ns, "verifica el permiso de lectura sobre persistentvolumeclaims", err)
		}
		for _, pvc := range pvcs.Items {
			inv.PVCs = append(inv.PVCs, pvcSummaryFrom(pvc))
		}
	}

	storageClasses, err := c.client.Kubernetes.StorageV1().StorageClasses().List(ctx, listOpts)
	if err != nil {
		return huskerr.New("no se pudo listar StorageClasses", "verifica el permiso de lectura sobre storageclasses", err)
	}
	for _, sc := range storageClasses.Items {
		inv.StorageClasses = append(inv.StorageClasses, storageClassSummaryFrom(sc))
	}

	return nil
}

func pvcSummaryFrom(pvc corev1.PersistentVolumeClaim) model.PVCSummary {
	capacity := ""
	if q, ok := pvc.Status.Capacity[corev1.ResourceStorage]; ok {
		capacity = q.String()
	}
	modes := make([]string, 0, len(pvc.Spec.AccessModes))
	for _, m := range pvc.Spec.AccessModes {
		modes = append(modes, string(m))
	}
	scName := ""
	if pvc.Spec.StorageClassName != nil {
		scName = *pvc.Spec.StorageClassName
	}
	return model.PVCSummary{
		Name:             pvc.Name,
		Namespace:        pvc.Namespace,
		StorageClassName: scName,
		Capacity:         capacity,
		Phase:            string(pvc.Status.Phase),
		AccessModes:      joinStrings(modes, ","),
	}
}

func storageClassSummaryFrom(sc storagev1.StorageClass) model.StorageClassSummary {
	reclaim := ""
	if sc.ReclaimPolicy != nil {
		reclaim = string(*sc.ReclaimPolicy)
	}
	bindingMode := ""
	if sc.VolumeBindingMode != nil {
		bindingMode = string(*sc.VolumeBindingMode)
	}
	allowExpansion := sc.AllowVolumeExpansion != nil && *sc.AllowVolumeExpansion
	isDefault := sc.Annotations[defaultStorageClassAnnotation] == "true"

	return model.StorageClassSummary{
		Name:                 sc.Name,
		Provisioner:          sc.Provisioner,
		ReclaimPolicy:        reclaim,
		VolumeBindingMode:    bindingMode,
		AllowVolumeExpansion: allowExpansion,
		IsDefault:            isDefault,
	}
}
