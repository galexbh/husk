package inventory

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"

	"github.com/galexbh/husk/internal/huskerr"
	"github.com/galexbh/husk/internal/model"
)

// collectNetwork recolecta Services siempre, y NetworkPolicies/Ingresses
// (además de Routes de OpenShift) en ext cuando no es nil (--extended).
func (c *Collector) collectNetwork(ctx context.Context, inv *model.Inventory, ext *model.ExtendedInventory, namespaces []string) error {
	for _, ns := range namespaces {
		services, err := c.client.Kubernetes.CoreV1().Services(ns).List(ctx, listOpts)
		if err != nil {
			return huskerr.New("no se pudo listar Services en "+ns, "verifica el permiso de lectura sobre services", err)
		}
		for _, s := range services.Items {
			inv.Services = append(inv.Services, serviceSummaryFrom(s))
		}
	}

	if ext == nil {
		return nil
	}

	for _, ns := range namespaces {
		policies, err := c.client.Kubernetes.NetworkingV1().NetworkPolicies(ns).List(ctx, listOpts)
		if err != nil {
			return huskerr.New("no se pudo listar NetworkPolicies en "+ns, "verifica el permiso de lectura sobre networkpolicies", err)
		}
		for _, p := range policies.Items {
			ext.NetworkPolicies = append(ext.NetworkPolicies, networkPolicySummaryFrom(p))
		}

		ingresses, err := c.client.Kubernetes.NetworkingV1().Ingresses(ns).List(ctx, listOpts)
		if err != nil {
			return huskerr.New("no se pudo listar Ingresses en "+ns, "verifica el permiso de lectura sobre ingresses", err)
		}
		for _, ing := range ingresses.Items {
			ext.Ingresses = append(ext.Ingresses, ingressSummaryFrom(ing))
		}
	}

	if c.client.IsOpenShift && c.client.Route != nil {
		for _, ns := range namespaces {
			routes, err := c.client.Route.RouteV1().Routes(ns).List(ctx, listOpts)
			if err != nil {
				return huskerr.New("no se pudo listar Routes en "+ns, "verifica el permiso de lectura sobre routes.route.openshift.io", err)
			}
			for _, r := range routes.Items {
				toService := ""
				if r.Spec.To.Kind == "Service" || r.Spec.To.Kind == "" {
					toService = r.Spec.To.Name
				}
				ext.Routes = append(ext.Routes, model.RouteSummary{
					Name:      r.Name,
					Namespace: r.Namespace,
					Host:      r.Spec.Host,
					ToService: toService,
					TLS:       r.Spec.TLS != nil,
				})
			}
		}
	}

	return nil
}

func serviceSummaryFrom(s corev1.Service) model.ServiceSummary {
	ports := make([]string, 0, len(s.Spec.Ports))
	for _, p := range s.Spec.Ports {
		ports = append(ports, fmt.Sprintf("%d/%s", p.Port, p.Protocol))
	}
	return model.ServiceSummary{
		Name:      s.Name,
		Namespace: s.Namespace,
		Type:      string(s.Spec.Type),
		ClusterIP: s.Spec.ClusterIP,
		Ports:     ports,
	}
}

func networkPolicySummaryFrom(p networkingv1.NetworkPolicy) model.NetworkPolicySummary {
	types := make([]string, 0, len(p.Spec.PolicyTypes))
	for _, t := range p.Spec.PolicyTypes {
		types = append(types, string(t))
	}
	return model.NetworkPolicySummary{Name: p.Name, Namespace: p.Namespace, PolicyTypes: types}
}

func ingressSummaryFrom(ing networkingv1.Ingress) model.IngressSummary {
	hosts := make([]string, 0, len(ing.Spec.Rules))
	for _, rule := range ing.Spec.Rules {
		if rule.Host != "" {
			hosts = append(hosts, rule.Host)
		}
	}
	return model.IngressSummary{Name: ing.Name, Namespace: ing.Namespace, Hosts: hosts}
}
