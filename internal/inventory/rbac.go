package inventory

import (
	"context"

	rbacv1 "k8s.io/api/rbac/v1"

	"github.com/galexbh/husk/internal/huskerr"
	"github.com/galexbh/husk/internal/model"
)

// collectRBAC recolecta Roles/RoleBindings por namespace y ClusterRoles/
// ClusterRoleBindings (cluster-scoped, una sola vez). Solo se llama con
// --extended.
func (c *Collector) collectRBAC(ctx context.Context, ext *model.ExtendedInventory, namespaces []string) error {
	for _, ns := range namespaces {
		roles, err := c.client.Kubernetes.RbacV1().Roles(ns).List(ctx, listOpts)
		if err != nil {
			return huskerr.New("no se pudo listar Roles en "+ns, "verifica el permiso de lectura sobre roles", err)
		}
		for _, r := range roles.Items {
			ext.Roles = append(ext.Roles, model.RBACSummary{Kind: "Role", Name: r.Name, Namespace: r.Namespace})
		}

		roleBindings, err := c.client.Kubernetes.RbacV1().RoleBindings(ns).List(ctx, listOpts)
		if err != nil {
			return huskerr.New("no se pudo listar RoleBindings en "+ns, "verifica el permiso de lectura sobre rolebindings", err)
		}
		for _, rb := range roleBindings.Items {
			ext.RoleBindings = append(ext.RoleBindings, roleBindingSummary("RoleBinding", rb.Name, rb.Namespace, rb.RoleRef, rb.Subjects))
		}
	}

	clusterRoles, err := c.client.Kubernetes.RbacV1().ClusterRoles().List(ctx, listOpts)
	if err != nil {
		return huskerr.New("no se pudo listar ClusterRoles", "verifica el permiso de lectura sobre clusterroles", err)
	}
	for _, cr := range clusterRoles.Items {
		ext.ClusterRoles = append(ext.ClusterRoles, model.RBACSummary{Kind: "ClusterRole", Name: cr.Name})
	}

	clusterRoleBindings, err := c.client.Kubernetes.RbacV1().ClusterRoleBindings().List(ctx, listOpts)
	if err != nil {
		return huskerr.New("no se pudo listar ClusterRoleBindings", "verifica el permiso de lectura sobre clusterrolebindings", err)
	}
	for _, crb := range clusterRoleBindings.Items {
		ext.ClusterRoleBindings = append(ext.ClusterRoleBindings, roleBindingSummary("ClusterRoleBinding", crb.Name, "", crb.RoleRef, crb.Subjects))
	}

	return nil
}

func roleBindingSummary(kind, name, namespace string, roleRef rbacv1.RoleRef, subjects []rbacv1.Subject) model.RBACSummary {
	subjectStrs := make([]string, 0, len(subjects))
	for _, s := range subjects {
		subjectStrs = append(subjectStrs, s.Kind+":"+s.Name)
	}
	return model.RBACSummary{
		Kind:      kind,
		Name:      name,
		Namespace: namespace,
		RoleRef:   roleRef.Kind + "/" + roleRef.Name,
		Subjects:  subjectStrs,
	}
}
