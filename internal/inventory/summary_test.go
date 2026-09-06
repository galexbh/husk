package inventory

import (
	"testing"

	"github.com/galexbh/husk/internal/model"
)

func TestBuildSummary_CountsAndFindings(t *testing.T) {
	inv := &model.Inventory{
		Deployments: []model.WorkloadSummary{
			{
				Kind: "Deployment", Name: "api", Namespace: "shop", Replicas: 1,
				Containers: []model.ContainerSummary{{Name: "api", HasLimits: false}},
			},
			{
				Kind: "Deployment", Name: "worker", Namespace: "shop", Replicas: 3,
				Containers: []model.ContainerSummary{{Name: "worker", HasLimits: true}},
			},
		},
		DaemonSets: []model.WorkloadSummary{
			{Kind: "DaemonSet", Name: "agent", Namespace: "shop", Replicas: 3, Containers: []model.ContainerSummary{{Name: "agent", HasLimits: true}}},
		},
	}

	s := BuildSummary(inv, []string{"shop"})

	if s.DeploymentsCount != 2 {
		t.Errorf("DeploymentsCount = %d, want 2", s.DeploymentsCount)
	}
	if s.SingleReplicaWorkloads != 1 {
		t.Errorf("SingleReplicaWorkloads = %d, want 1", s.SingleReplicaWorkloads)
	}
	if s.WorkloadsWithoutLimits != 1 {
		t.Errorf("WorkloadsWithoutLimits = %d, want 1", s.WorkloadsWithoutLimits)
	}
	if !s.HasQuotaData {
		t.Error("HasQuotaData debería ser true: ResourceQuotas ya no depende de --extended")
	}
	if s.NamespacesWithoutQuota != 1 {
		t.Errorf("NamespacesWithoutQuota = %d, want 1 (shop sin ResourceQuota)", s.NamespacesWithoutQuota)
	}
	if len(s.Findings) != 3 {
		t.Errorf("len(Findings) = %d, want 3 (single-replica + missing-limits + missing-resourcequota)", len(s.Findings))
	}
}

func TestBuildSummary_NamespacesWithoutQuota(t *testing.T) {
	inv := &model.Inventory{
		ResourceQuotas: []model.ResourceQuotaSummary{{Name: "default", Namespace: "shop"}},
	}

	s := BuildSummary(inv, []string{"shop", "billing"})

	if !s.HasQuotaData {
		t.Fatal("HasQuotaData debería ser true con inv.Extended")
	}
	if s.NamespacesWithoutQuota != 1 {
		t.Errorf("NamespacesWithoutQuota = %d, want 1", s.NamespacesWithoutQuota)
	}
}
