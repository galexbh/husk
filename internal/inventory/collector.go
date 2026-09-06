// Package inventory recolecta el inventario de recursos del cluster
// directamente del API server, vía internal/k8sclient. Solo lee: nunca crea,
// actualiza, parchea ni borra nada. No genera salida visual — eso vive en
// internal/report e internal/excel.
package inventory

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/galexbh/husk/internal/huskerr"
	"github.com/galexbh/husk/internal/k8sclient"
	"github.com/galexbh/husk/internal/model"
	"github.com/galexbh/husk/internal/nsfilter"
)

// listOpts es el ListOptions por defecto para todas las llamadas de
// recolección: sin límite (el inventario necesita el conjunto completo).
var listOpts = metav1.ListOptions{}

// Options controla el alcance de la recolección.
type Options struct {
	// Namespace, si no está vacío, acota la recolección a un único
	// namespace y omite el filtro de exclusión (una petición explícita del
	// usuario tiene prioridad sobre la política de exclusión por defecto).
	Namespace string
	// Extended incluye RBAC, NetworkPolicies, PDBs, ResourceQuotas,
	// LimitRanges, HPAs e Ingresses/Routes.
	Extended bool
}

// Collector recolecta el inventario de un cluster.
type Collector struct {
	client *k8sclient.Client
	filter *nsfilter.Filter
}

// New construye un Collector. filter aplica la política de exclusión de
// namespaces cuando Options.Namespace está vacío.
func New(client *k8sclient.Client, filter *nsfilter.Filter) *Collector {
	return &Collector{client: client, filter: filter}
}

// Collect recolecta el inventario completo según opts. Devuelve también la
// lista de namespaces efectivamente analizados (tras el filtro de
// exclusión), que BuildSummary necesita para hallazgos como "namespaces sin
// ResourceQuota".
func (c *Collector) Collect(ctx context.Context, opts Options) (*model.Inventory, []string, error) {
	inv := &model.Inventory{}

	namespaces, err := c.namespacesToScan(ctx, opts.Namespace)
	if err != nil {
		return nil, nil, err
	}

	var ext *model.ExtendedInventory
	if opts.Extended {
		ext = &model.ExtendedInventory{}
	}

	if err := c.collectWorkloads(ctx, inv, namespaces); err != nil {
		return nil, nil, err
	}
	if err := c.collectNetwork(ctx, inv, ext, namespaces); err != nil {
		return nil, nil, err
	}
	if err := c.collectStorage(ctx, inv, namespaces); err != nil {
		return nil, nil, err
	}
	if err := c.collectConfigAndSecrets(ctx, inv, namespaces); err != nil {
		return nil, nil, err
	}
	if err := c.collectClusterScoped(ctx, inv); err != nil {
		return nil, nil, err
	}
	if ext != nil {
		if err := c.collectExtended(ctx, ext, namespaces); err != nil {
			return nil, nil, err
		}
		inv.Extended = ext
	}

	return inv, namespaces, nil
}

// namespacesToScan resuelve la lista de namespaces a recolectar: el
// namespace explícito si se indicó, o todos los namespaces del cluster tras
// aplicar el filtro de exclusión.
func (c *Collector) namespacesToScan(ctx context.Context, explicit string) ([]string, error) {
	if explicit != "" {
		return []string{explicit}, nil
	}

	list, err := c.client.Kubernetes.CoreV1().Namespaces().List(ctx, listOpts)
	if err != nil {
		return nil, huskerr.New(
			"no se pudo listar namespaces",
			"verifica el permiso de lectura sobre namespaces (`husk connect health`)",
			err,
		)
	}

	names := make([]string, 0, len(list.Items))
	for _, ns := range list.Items {
		names = append(names, ns.Name)
	}
	return c.filter.Apply(names), nil
}
