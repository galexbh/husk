package k8sclient

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/galexbh/husk/internal/huskerr"
)

const (
	monitoringNamespace   = "openshift-monitoring"
	thanosRouteName       = "thanos-querier"
	alertmanagerRouteName = "alertmanager-main"
)

// ThanosQuerierURL resuelve la URL externa de la route de Thanos Querier en
// openshift-monitoring. Solo aplica en clusters OpenShift; el acceso es
// siempre cluster-wide (requiere el ClusterRole cluster-monitoring-view),
// el modo tenancy por namespace queda fuera de alcance.
func (c *Client) ThanosQuerierURL(ctx context.Context) (string, error) {
	return c.monitoringRouteURL(ctx, thanosRouteName, "Thanos Querier")
}

// AlertmanagerURL resuelve la URL externa de la route de Alertmanager en
// openshift-monitoring, usada por internal/alertmanager para correlacionar
// alertas activas con hallazgos de sizing/capacity.
func (c *Client) AlertmanagerURL(ctx context.Context) (string, error) {
	return c.monitoringRouteURL(ctx, alertmanagerRouteName, "Alertmanager")
}

// monitoringRouteURL resuelve la URL externa de una route del stack de
// monitoreo de OpenShift (openshift-monitoring), compartido por Thanos
// Querier y Alertmanager.
func (c *Client) monitoringRouteURL(ctx context.Context, routeName, label string) (string, error) {
	if !c.IsOpenShift || c.Route == nil {
		return "", huskerr.New(
			label+" solo está disponible en clusters OpenShift",
			"este chequeo no aplica en Kubernetes vanilla",
			nil,
		)
	}

	route, err := c.Route.RouteV1().Routes(monitoringNamespace).Get(ctx, routeName, metav1.GetOptions{})
	if err != nil {
		return "", huskerr.New(
			fmt.Sprintf("no se pudo obtener la route %s/%s", monitoringNamespace, routeName),
			"verifica que el monitoring stack de OpenShift esté instalado y que tengas permiso de lectura sobre routes en openshift-monitoring",
			err,
		)
	}

	host := route.Spec.Host
	if host == "" && len(route.Status.Ingress) > 0 {
		host = route.Status.Ingress[0].Host
	}
	if host == "" {
		return "", huskerr.New(
			fmt.Sprintf("la route %s no tiene host asignado", routeName),
			fmt.Sprintf("revisa el estado de la route %s en openshift-monitoring", routeName),
			nil,
		)
	}

	return "https://" + host, nil
}
