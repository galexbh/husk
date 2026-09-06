package capacity

import (
	"context"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/galexbh/husk/internal/config"
	"github.com/galexbh/husk/internal/k8sclient"
)

func testCfg() config.CapacityConfig {
	return config.CapacityConfig{HeadroomThresholdPercent: 30.0, Lookback: "7d"}
}

func int32Ptr(v int32) *int32 { return &v }

func node(name string, cpu, mem, pods string, taints []corev1.Taint, ready bool) *corev1.Node {
	return &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Spec:       corev1.NodeSpec{Taints: taints},
		Status: corev1.NodeStatus{
			Allocatable: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse(cpu),
				corev1.ResourceMemory: resource.MustParse(mem),
				corev1.ResourcePods:   resource.MustParse(pods),
			},
			Conditions: []corev1.NodeCondition{{Type: corev1.NodeReady, Status: readyStatus(ready)}},
		},
	}
}

func readyStatus(ready bool) corev1.ConditionStatus {
	if ready {
		return corev1.ConditionTrue
	}
	return corev1.ConditionFalse
}

func pod(name, namespace, nodeName string, labelsMap map[string]string, cpuReq, memReq string) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace, Labels: labelsMap},
		Spec: corev1.PodSpec{
			NodeName: nodeName,
			Containers: []corev1.Container{{
				Name: "app",
				Resources: corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse(cpuReq),
						corev1.ResourceMemory: resource.MustParse(memReq),
					},
				},
			}},
		},
		Status: corev1.PodStatus{Phase: corev1.PodRunning},
	}
}

func TestAnalyze_HeadroomAndRequests(t *testing.T) {
	n1 := node("node-1", "4", "8Gi", "110", nil, true)
	p1 := pod("app-1", "shop", "node-1", map[string]string{"app": "api"}, "1", "2Gi")
	p2 := pod("app-2", "shop", "node-1", map[string]string{"app": "api"}, "1", "2Gi")

	client := &k8sclient.Client{Kubernetes: fake.NewSimpleClientset(n1, p1, p2)}
	a := New(client, nil, testCfg())

	report, err := a.Analyze(context.Background())
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}

	if len(report.Nodes) != 1 {
		t.Fatalf("len(Nodes) = %d, want 1", len(report.Nodes))
	}
	nc := report.Nodes[0]
	if nc.RequestedCPU != "2" {
		t.Errorf("RequestedCPU = %q, want 2", nc.RequestedCPU)
	}
	if nc.PodCount != 2 {
		t.Errorf("PodCount = %d, want 2", nc.PodCount)
	}
	// 2 cores pedidos de 4 allocatable = 50% headroom.
	if nc.CPUHeadroomPercent != 50 {
		t.Errorf("CPUHeadroomPercent = %v, want 50", nc.CPUHeadroomPercent)
	}
	if nc.Risk != "green" {
		t.Errorf("Risk = %q, want green (50%% > 30%% umbral)", nc.Risk)
	}
	if report.HasMetrics {
		t.Error("HasMetrics debería ser false sin cliente Prometheus")
	}
}

func TestAnalyze_TaintedNodeExcluded(t *testing.T) {
	n1 := node("node-1", "4", "8Gi", "110", nil, true)
	n2 := node("node-2-infra", "4", "8Gi", "110", []corev1.Taint{{Key: "infra", Effect: corev1.TaintEffectNoSchedule}}, true)

	client := &k8sclient.Client{Kubernetes: fake.NewSimpleClientset(n1, n2)}
	a := New(client, nil, testCfg())

	report, err := a.Analyze(context.Background())
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}

	if len(report.Nodes) != 1 || report.Nodes[0].Name != "node-1" {
		t.Errorf("Nodes = %+v, want solo node-1", report.Nodes)
	}
	if len(report.ExcludedNodes) != 1 || report.ExcludedNodes[0].Name != "node-2-infra" {
		t.Errorf("ExcludedNodes = %+v, want solo node-2-infra", report.ExcludedNodes)
	}
}

func TestAnalyze_LowHeadroomIsRisky(t *testing.T) {
	n1 := node("node-1", "4", "8Gi", "110", nil, true)
	// CPU: 87.5% pedido -> 12.5% headroom (< 30%). Memoria: 75% pedido ->
	// 25% headroom (< 30%). Ambos ejes bajo el umbral -> riesgo ALTO.
	p1 := pod("app-1", "shop", "node-1", nil, "3.5", "6Gi")

	client := &k8sclient.Client{Kubernetes: fake.NewSimpleClientset(n1, p1)}
	a := New(client, nil, testCfg())

	report, err := a.Analyze(context.Background())
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}
	if report.Nodes[0].Risk != "red" {
		t.Errorf("Risk = %q, want red (headroom bajo en ambos ejes)", report.Nodes[0].Risk)
	}
}

func TestAnalyze_SingleAxisIsMedio(t *testing.T) {
	n1 := node("node-1", "4", "8Gi", "110", nil, true)
	// CPU: 87.5% pedido -> 12.5% headroom (< 30%). Memoria: 12.5% pedido ->
	// 87.5% headroom (sana). Un solo eje bajo el umbral -> riesgo MEDIO.
	p1 := pod("app-1", "shop", "node-1", nil, "3.5", "1Gi")

	client := &k8sclient.Client{Kubernetes: fake.NewSimpleClientset(n1, p1)}
	a := New(client, nil, testCfg())

	report, err := a.Analyze(context.Background())
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}
	nc := report.Nodes[0]
	if nc.Risk != "yellow" {
		t.Errorf("Risk = %q, want yellow (un solo eje con headroom bajo)", nc.Risk)
	}
	if len(nc.RiskAxes) != 1 || nc.RiskAxes[0] != "cpu" {
		t.Errorf("RiskAxes = %v, want [cpu]", nc.RiskAxes)
	}
}

func TestDetectConcentration_SameNode(t *testing.T) {
	n1 := node("node-1", "4", "8Gi", "110", nil, true)
	labelsMap := map[string]string{"app": "api"}
	p1 := pod("api-1", "shop", "node-1", labelsMap, "100m", "128Mi")
	p2 := pod("api-2", "shop", "node-1", labelsMap, "100m", "128Mi")
	p3 := pod("api-3", "shop", "node-1", labelsMap, "100m", "128Mi")

	deploy := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "api", Namespace: "shop"},
		Spec: appsv1.DeploymentSpec{
			Replicas: int32Ptr(3),
			Selector: &metav1.LabelSelector{MatchLabels: labelsMap},
		},
	}

	client := &k8sclient.Client{Kubernetes: fake.NewSimpleClientset(n1, p1, p2, p3, deploy)}
	a := New(client, nil, testCfg())

	report, err := a.Analyze(context.Background())
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}

	if len(report.ConcentrationRisks) != 1 {
		t.Fatalf("ConcentrationRisks = %+v, want 1 hallazgo", report.ConcentrationRisks)
	}
	r := report.ConcentrationRisks[0]
	if r.Name != "api" || r.DistinctNodes != 1 {
		t.Errorf("riesgo inesperado: %+v", r)
	}
}

func TestDetectConcentration_NoRiskWithDistinctNodes(t *testing.T) {
	n1 := node("node-1", "4", "8Gi", "110", nil, true)
	n2 := node("node-2", "4", "8Gi", "110", nil, true)
	labelsMap := map[string]string{"app": "api"}
	p1 := pod("api-1", "shop", "node-1", labelsMap, "100m", "128Mi")
	p2 := pod("api-2", "shop", "node-2", labelsMap, "100m", "128Mi")

	deploy := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "api", Namespace: "shop"},
		Spec: appsv1.DeploymentSpec{
			Replicas: int32Ptr(2),
			Selector: &metav1.LabelSelector{MatchLabels: labelsMap},
		},
	}

	client := &k8sclient.Client{Kubernetes: fake.NewSimpleClientset(n1, n2, p1, p2, deploy)}
	a := New(client, nil, testCfg())

	report, err := a.Analyze(context.Background())
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}
	if len(report.ConcentrationRisks) != 0 {
		t.Errorf("ConcentrationRisks = %+v, want ninguno (réplicas en nodos distintos)", report.ConcentrationRisks)
	}
}
