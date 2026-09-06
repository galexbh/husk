package k8sclient

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ProbeResult reporta si husk pudo leer un tipo de recurso dado.
type ProbeResult struct {
	Resource string
	OK       bool
	Err      error
}

// ProbeReadAccess ejecuta List baratos (Limit: 1) contra los recursos
// centrales que husk necesita, para validar el RBAC sin requerir el verbo
// `create` que exigiría un SelfSubjectAccessReview.
func (c *Client) ProbeReadAccess(ctx context.Context) []ProbeResult {
	opts := metav1.ListOptions{Limit: 1}
	results := make([]ProbeResult, 0, 5)

	check := func(name string, fn func() error) {
		err := fn()
		results = append(results, ProbeResult{Resource: name, OK: err == nil, Err: err})
	}

	check("nodes", func() error {
		_, err := c.Kubernetes.CoreV1().Nodes().List(ctx, opts)
		return err
	})
	check("namespaces", func() error {
		_, err := c.Kubernetes.CoreV1().Namespaces().List(ctx, opts)
		return err
	})
	check("pods (todos los namespaces)", func() error {
		_, err := c.Kubernetes.CoreV1().Pods("").List(ctx, opts)
		return err
	})
	check("deployments (todos los namespaces)", func() error {
		_, err := c.Kubernetes.AppsV1().Deployments("").List(ctx, opts)
		return err
	})
	check("storageclasses", func() error {
		_, err := c.Kubernetes.StorageV1().StorageClasses().List(ctx, opts)
		return err
	})

	return results
}
