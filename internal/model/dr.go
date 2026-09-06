package model

import "time"

// BSLStatus representa una BackupStorageLocation de Velero/OADP.
type BSLStatus struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	Provider  string `json:"provider,omitempty"`
	Phase     string `json:"phase"`
	Default   bool   `json:"default"`
}

// BackupSummary representa un Backup CR de Velero/OADP.
type BackupSummary struct {
	Name                string    `json:"name"`
	IncludedNamespaces  []string  `json:"includedNamespaces,omitempty"`
	Phase               string    `json:"phase"`
	CompletionTimestamp time.Time `json:"completionTimestamp,omitempty"`
	StartTimestamp      time.Time `json:"startTimestamp,omitempty"`
}

// EtcdSnapshotStatus es la antigüedad del snapshot de etcd, inferida del
// CronJob/Job de backup de etcd (no hay una API de solo lectura que lo
// exponga directamente). Solo se evalúa en OpenShift.
type EtcdSnapshotStatus struct {
	// Verifiable indica si se encontró evidencia (un Job exitoso) para
	// calcular la antigüedad. Si es false, AgeOK y LastSuccessTime no son
	// significativos.
	Verifiable      bool      `json:"verifiable"`
	Source          string    `json:"source,omitempty"` // ej. "CronJob/openshift-etcd/etcd-backup"
	LastSuccessTime time.Time `json:"lastSuccessTime,omitempty"`
	AgeOK           bool      `json:"ageOK"`
	// ClusterOperatorHealthy es el estado del ClusterOperator "etcd"
	// (Available=True, Degraded=False), informativo independientemente de
	// si se pudo verificar el snapshot.
	ClusterOperatorHealthy bool `json:"clusterOperatorHealthy"`
}

// WorkloadRef identifica un workload por tipo/namespace/nombre, usado por
// los hallazgos de PDB y topology spread.
type WorkloadRef struct {
	Kind      string `json:"kind"`
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
}

// DRReadiness es el resultado de `husk dr assess`.
type DRReadiness struct {
	IsOpenShift bool `json:"isOpenShift"`

	OADPNamespaceExists bool   `json:"oadpNamespaceExists"`
	OADPInstalled       bool   `json:"oadpInstalled"`
	OADPHealthy         bool   `json:"oadpHealthy"`
	OADPMessage         string `json:"oadpMessage,omitempty"`

	BackupStorageLocations []BSLStatus     `json:"backupStorageLocations,omitempty"`
	Backups                []BackupSummary `json:"backups,omitempty"`

	// ApplicationNamespaces son los namespaces evaluados para cobertura de
	// backup (tras el filtro de exclusión transversal; openshift-adp no se
	// incluye aquí, se evalúa aparte arriba).
	ApplicationNamespaces     []string `json:"applicationNamespaces,omitempty"`
	NamespacesWithoutBackup   []string `json:"namespacesWithoutBackup,omitempty"`
	NamespacesWithStaleBackup []string `json:"namespacesWithStaleBackup,omitempty"`

	// EtcdCheckApplicable es false en Kubernetes vanilla: el chequeo de
	// etcd no aplica y no penaliza el score.
	EtcdCheckApplicable bool                `json:"etcdCheckApplicable"`
	EtcdSnapshot        *EtcdSnapshotStatus `json:"etcdSnapshot,omitempty"`

	StorageClassesUsed               []string `json:"storageClassesUsed,omitempty"`
	StorageClassesWithoutCSISnapshot []string `json:"storageClassesWithoutCSISnapshot,omitempty"`

	// CriticalWorkloadsTotal es la cantidad de workloads "críticos"
	// evaluados para PDB/topology spread (Deployments/StatefulSets con
	// replicas > 1; los DaemonSets quedan fuera porque ya se distribuyen
	// por diseño en todos los nodos elegibles).
	CriticalWorkloadsTotal int           `json:"criticalWorkloadsTotal"`
	MissingPDBs            []WorkloadRef `json:"missingPDBs,omitempty"`
	MissingTopologySpread  []WorkloadRef `json:"missingTopologySpread,omitempty"`

	Findings    []Finding `json:"findings"`
	GeneratedAt time.Time `json:"generatedAt"`
}
