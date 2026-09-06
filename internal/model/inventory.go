package model

// RiskLevel clasifica el riesgo de un recurso para el resaltado
// condicional (Excel) y para los hallazgos de inventory/sizing/dr/score.
type RiskLevel string

// Niveles de riesgo: Red (crítico, p. ej. sin límites o subaprovisionado),
// Yellow (advertencia, p. ej. sobreaprovisionado), Green (saludable) y
// Unknown (sin datos suficientes para evaluar).
const (
	RiskRed     RiskLevel = "red"
	RiskYellow  RiskLevel = "yellow"
	RiskGreen   RiskLevel = "green"
	RiskUnknown RiskLevel = ""
)

// ContainerSummary describe un contenedor de un workload, con lo mínimo
// necesario para sizing y para el resaltado de riesgo del inventario.
type ContainerSummary struct {
	Name          string `json:"name"`
	Image         string `json:"image"`
	CPURequest    string `json:"cpuRequest,omitempty"`
	CPULimit      string `json:"cpuLimit,omitempty"`
	MemoryRequest string `json:"memoryRequest,omitempty"`
	MemoryLimit   string `json:"memoryLimit,omitempty"`
	HasLimits     bool   `json:"hasLimits"`
}

// WorkloadSummary representa un Deployment, StatefulSet o DaemonSet.
type WorkloadSummary struct {
	Kind          string             `json:"kind"`
	Name          string             `json:"name"`
	Namespace     string             `json:"namespace"`
	Replicas      int32              `json:"replicas"`
	ReadyReplicas int32              `json:"readyReplicas"`
	Containers    []ContainerSummary `json:"containers"`
	Labels        map[string]string  `json:"labels,omitempty"`
	Risk          RiskLevel          `json:"risk"`
}

// ServiceSummary representa un Service.
type ServiceSummary struct {
	Name      string   `json:"name"`
	Namespace string   `json:"namespace"`
	Type      string   `json:"type"`
	ClusterIP string   `json:"clusterIP,omitempty"`
	Ports     []string `json:"ports,omitempty"`
}

// PVCSummary representa un PersistentVolumeClaim.
type PVCSummary struct {
	Name             string `json:"name"`
	Namespace        string `json:"namespace"`
	StorageClassName string `json:"storageClassName,omitempty"`
	Capacity         string `json:"capacity,omitempty"`
	Phase            string `json:"phase"`
	AccessModes      string `json:"accessModes,omitempty"`
}

// ConfigMapSummary representa un ConfigMap. Nunca incluye su contenido
// (Data/BinaryData): solo metadatos no sensibles.
type ConfigMapSummary struct {
	Name      string            `json:"name"`
	Namespace string            `json:"namespace"`
	KeysCount int               `json:"keysCount"`
	Labels    map[string]string `json:"labels,omitempty"`
}

// SecretSummary representa un Secret. Nunca incluye su contenido (Data) ni
// los nombres de sus claves: solo nombre, tipo, cantidad de claves y labels.
type SecretSummary struct {
	Name      string            `json:"name"`
	Namespace string            `json:"namespace"`
	Type      string            `json:"type"`
	KeysCount int               `json:"keysCount"`
	Labels    map[string]string `json:"labels,omitempty"`
}

// NodeSummary representa un Node.
type NodeSummary struct {
	Name            string            `json:"name"`
	Roles           []string          `json:"roles,omitempty"`
	Ready           bool              `json:"ready"`
	Unschedulable   bool              `json:"unschedulable"`
	KubeletVersion  string            `json:"kubeletVersion,omitempty"`
	AllocatableCPU  string            `json:"allocatableCPU,omitempty"`
	AllocatableMem  string            `json:"allocatableMemory,omitempty"`
	AllocatablePods string            `json:"allocatablePods,omitempty"`
	Labels          map[string]string `json:"labels,omitempty"`
}

// StorageClassSummary representa un StorageClass.
type StorageClassSummary struct {
	Name                 string `json:"name"`
	Provisioner          string `json:"provisioner"`
	ReclaimPolicy        string `json:"reclaimPolicy,omitempty"`
	VolumeBindingMode    string `json:"volumeBindingMode,omitempty"`
	AllowVolumeExpansion bool   `json:"allowVolumeExpansion"`
	IsDefault            bool   `json:"isDefault"`
}

// CRDSummary representa una CustomResourceDefinition.
type CRDSummary struct {
	Name       string   `json:"name"`
	Group      string   `json:"group"`
	Kind       string   `json:"kind"`
	Versions   []string `json:"versions,omitempty"`
	Scope      string   `json:"scope"`
	Conditions []string `json:"conditions,omitempty"`
}

// RBACSummary representa un Role/ClusterRole o RoleBinding/ClusterRoleBinding
// (--extended).
type RBACSummary struct {
	Kind      string   `json:"kind"`
	Name      string   `json:"name"`
	Namespace string   `json:"namespace,omitempty"`
	RoleRef   string   `json:"roleRef,omitempty"`
	Subjects  []string `json:"subjects,omitempty"`
}

// NetworkPolicySummary representa una NetworkPolicy (--extended).
type NetworkPolicySummary struct {
	Name        string   `json:"name"`
	Namespace   string   `json:"namespace"`
	PolicyTypes []string `json:"policyTypes,omitempty"`
}

// PDBSummary representa un PodDisruptionBudget (--extended).
type PDBSummary struct {
	Name           string `json:"name"`
	Namespace      string `json:"namespace"`
	MinAvailable   string `json:"minAvailable,omitempty"`
	MaxUnavailable string `json:"maxUnavailable,omitempty"`
	CurrentHealthy int32  `json:"currentHealthy"`
	DesiredHealthy int32  `json:"desiredHealthy"`
}

// ResourceQuotaSummary representa un ResourceQuota (--extended).
type ResourceQuotaSummary struct {
	Name      string            `json:"name"`
	Namespace string            `json:"namespace"`
	Hard      map[string]string `json:"hard,omitempty"`
	Used      map[string]string `json:"used,omitempty"`
}

// LimitRangeSummary representa un LimitRange (--extended).
type LimitRangeSummary struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	Limits    int    `json:"limits"`
}

// HPASummary representa un HorizontalPodAutoscaler (--extended).
type HPASummary struct {
	Name            string `json:"name"`
	Namespace       string `json:"namespace"`
	Target          string `json:"target,omitempty"`
	MinReplicas     int32  `json:"minReplicas"`
	MaxReplicas     int32  `json:"maxReplicas"`
	CurrentReplicas int32  `json:"currentReplicas"`
}

// IngressSummary representa un Ingress (--extended).
type IngressSummary struct {
	Name      string   `json:"name"`
	Namespace string   `json:"namespace"`
	Hosts     []string `json:"hosts,omitempty"`
}

// RouteSummary representa una Route de OpenShift (--extended, solo OpenShift).
type RouteSummary struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	Host      string `json:"host,omitempty"`
	ToService string `json:"toService,omitempty"`
	TLS       bool   `json:"tls"`
}

// ExtendedInventory agrupa los recursos que solo se recolectan con
// --extended.
type ExtendedInventory struct {
	Roles                []RBACSummary          `json:"roles,omitempty"`
	RoleBindings         []RBACSummary          `json:"roleBindings,omitempty"`
	ClusterRoles         []RBACSummary          `json:"clusterRoles,omitempty"`
	ClusterRoleBindings  []RBACSummary          `json:"clusterRoleBindings,omitempty"`
	NetworkPolicies      []NetworkPolicySummary `json:"networkPolicies,omitempty"`
	PodDisruptionBudgets []PDBSummary           `json:"podDisruptionBudgets,omitempty"`
	ResourceQuotas       []ResourceQuotaSummary `json:"resourceQuotas,omitempty"`
	LimitRanges          []LimitRangeSummary    `json:"limitRanges,omitempty"`
	HPAs                 []HPASummary           `json:"hpas,omitempty"`
	Ingresses            []IngressSummary       `json:"ingresses,omitempty"`
	Routes               []RouteSummary         `json:"routes,omitempty"`
}

// Inventory es el inventario completo de recursos del cluster (o de un
// namespace, cuando se filtra con --namespace).
type Inventory struct {
	Deployments    []WorkloadSummary     `json:"deployments"`
	StatefulSets   []WorkloadSummary     `json:"statefulSets"`
	DaemonSets     []WorkloadSummary     `json:"daemonSets"`
	Services       []ServiceSummary      `json:"services"`
	PVCs           []PVCSummary          `json:"pvcs"`
	ConfigMaps     []ConfigMapSummary    `json:"configMaps"`
	Secrets        []SecretSummary       `json:"secrets"`
	Nodes          []NodeSummary         `json:"nodes"`
	StorageClasses []StorageClassSummary `json:"storageClasses"`
	CRDs           []CRDSummary          `json:"crds"`

	Extended *ExtendedInventory `json:"extended,omitempty"`
}

// InventorySummary es la versión ejecutiva del inventario: conteos
// agregados y hallazgos de riesgo (`husk inventory summary`).
type InventorySummary struct {
	Namespaces int `json:"namespaces"`

	DeploymentsCount    int `json:"deploymentsCount"`
	StatefulSetsCount   int `json:"statefulSetsCount"`
	DaemonSetsCount     int `json:"daemonSetsCount"`
	ServicesCount       int `json:"servicesCount"`
	PVCsCount           int `json:"pvcsCount"`
	ConfigMapsCount     int `json:"configMapsCount"`
	SecretsCount        int `json:"secretsCount"`
	NodesCount          int `json:"nodesCount"`
	StorageClassesCount int `json:"storageClassesCount"`
	CRDsCount           int `json:"crdsCount"`

	SingleReplicaWorkloads int `json:"singleReplicaWorkloads"`
	WorkloadsWithoutLimits int `json:"workloadsWithoutLimits"`

	// NamespacesWithoutQuota solo se calcula cuando el inventario incluyó
	// --extended (necesita ResourceQuotas); HasQuotaData indica si el
	// campo anterior es significativo.
	NamespacesWithoutQuota int  `json:"namespacesWithoutQuota,omitempty"`
	HasQuotaData           bool `json:"hasQuotaData"`

	Findings []Finding `json:"findings"`
}
