package dr

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// listOpts es el ListOptions por defecto para las llamadas de recolección
// de este paquete: sin límite, DR readiness necesita el conjunto completo.
var listOpts = metav1.ListOptions{}

// GVRs de los CRs de Velero/OADP, leídos vía el cliente dinámico para
// evitar depender de sus módulos Go (que no forman parte del árbol de
// dependencias de client-go/apimachinery ya usado en el resto de husk).
var (
	backupGVR                = schema.GroupVersionResource{Group: "velero.io", Version: "v1", Resource: "backups"}
	backupStorageLocationGVR = schema.GroupVersionResource{Group: "velero.io", Version: "v1", Resource: "backupstoragelocations"}
	dataProtectionAppGVR     = schema.GroupVersionResource{Group: "oadp.openshift.io", Version: "v1alpha1", Resource: "dataprotectionapplications"}
	volumeSnapshotClassGVR   = schema.GroupVersionResource{Group: "snapshot.storage.k8s.io", Version: "v1", Resource: "volumesnapshotclasses"}
)
