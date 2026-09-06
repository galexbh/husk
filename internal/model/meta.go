package model

import "time"

// Meta describe el cluster y la ejecución que produjeron un ClusterSnapshot.
type Meta struct {
	ClusterName       string    `json:"clusterName,omitempty"`
	IsOpenShift       bool      `json:"isOpenShift"`
	KubernetesVersion string    `json:"kubernetesVersion,omitempty"`
	GeneratedAt       time.Time `json:"generatedAt"`
	HuskVersion       string    `json:"huskVersion,omitempty"`
}
