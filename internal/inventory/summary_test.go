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
	if s.HasQuotaData {
		t.Error("HasQuotaData debería ser false sin inv.Extended")
	}
	if len(s.Findings) != 2 {
		t.Errorf("len(Findings) = %d, want 2", len(s.Findings))
	}
}

func TestBuildSummary_NamespacesWithoutQuota(t *testing.T) {
	inv := &model.Inventory{
		Extended: &model.ExtendedInventory{
			ResourceQuotas: []model.ResourceQuotaSummary{{Name: "default", Namespace: "shop"}},
		},
	}

	s := BuildSummary(inv, []string{"shop", "billing"})

	if !s.HasQuotaData {
		t.Fatal("HasQuotaData debería ser true con inv.Extended")
	}
	if s.NamespacesWithoutQuota != 1 {
		t.Errorf("NamespacesWithoutQuota = %d, want 1", s.NamespacesWithoutQuota)
	}
}
