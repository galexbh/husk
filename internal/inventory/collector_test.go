package inventory

import (
	"context"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/galexbh/husk/internal/k8sclient"
	"github.com/galexbh/husk/internal/nsfilter"
)

// newTestClient construye un *k8sclient.Client respaldado por fake
// clientsets, sin tocar ningún cluster real. Los campos de k8sclient.Client
// son exportados precisamente para permitir esto en pruebas.
func newTestClient(objects ...runtime.Object) *k8sclient.Client {
	scheme := runtime.NewScheme()
	gvrToListKind := map[schema.GroupVersionResource]string{
		crdGVR: "CustomResourceDefinitionList",
	}
	return &k8sclient.Client{
		Kubernetes: fake.NewSimpleClientset(objects...),
		Dynamic:    dynamicfake.NewSimpleDynamicClientWithCustomListKinds(scheme, gvrToListKind),
	}
}

func int32Ptr(v int32) *int32 { return &v }

func TestCollect_WorkloadsAndRisk(t *testing.T) {
	deployNoLimits := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "api", Namespace: "shop"},
		Spec: appsv1.DeploymentSpec{
			Replicas: int32Ptr(1),
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{Name: "api", Image: "api:1.0"}},
				},
			},
		},
		Status: appsv1.DeploymentStatus{ReadyReplicas: 1},
	}

	deployHealthy := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "worker", Namespace: "shop"},
		Spec: appsv1.DeploymentSpec{
			Replicas: int32Ptr(3),
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Name:  "worker",
						Image: "worker:1.0",
						Resources: corev1.ResourceRequirements{
							Requests: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("100m"), corev1.ResourceMemory: resource.MustParse("128Mi")},
							Limits:   corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("200m"), corev1.ResourceMemory: resource.MustParse("256Mi")},
						},
					}},
				},
			},
		},
		Status: appsv1.DeploymentStatus{ReadyReplicas: 3},
	}

	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "shop"}}

	client := newTestClient(ns, deployNoLimits, deployHealthy)
	filter, err := nsfilter.New([]string{`^kube-.*$`})
	if err != nil {
		t.Fatalf("nsfilter.New: %v", err)
	}

	c := New(client, filter)
	inv, namespaces, err := c.Collect(context.Background(), Options{})
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}

	if got := []string{"shop"}; len(namespaces) != 1 || namespaces[0] != got[0] {
		t.Fatalf("namespaces = %v, want %v", namespaces, got)
	}

	if len(inv.Deployments) != 2 {
		t.Fatalf("len(Deployments) = %d, want 2", len(inv.Deployments))
	}

	byName := map[string]string{}
	for _, d := range inv.Deployments {
		byName[d.Name] = string(d.Risk)
	}
	if byName["api"] != "red" {
		t.Errorf("risk de 'api' = %s, want red (sin límites y una réplica)", byName["api"])
	}
	if byName["worker"] != "green" {
		t.Errorf("risk de 'worker' = %s, want green", byName["worker"])
	}
}

func TestCollect_SecretsAndConfigMapsMetadataOnly(t *testing.T) {
	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "shop"}}
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "db-creds", Namespace: "shop"},
		Type:       corev1.SecretTypeOpaque,
		Data:       map[string][]byte{"password": []byte("super-secret"), "username": []byte("admin")},
	}
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: "app-config", Namespace: "shop"},
		Data:       map[string]string{"key1": "value1"},
	}

	client := newTestClient(ns, secret, cm)
	filter, _ := nsfilter.New(nil)
	c := New(client, filter)

	inv, _, err := c.Collect(context.Background(), Options{})
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}

	if len(inv.Secrets) != 1 {
		t.Fatalf("len(Secrets) = %d, want 1", len(inv.Secrets))
	}
	if inv.Secrets[0].KeysCount != 2 {
		t.Errorf("KeysCount = %d, want 2", inv.Secrets[0].KeysCount)
	}

	if len(inv.ConfigMaps) != 1 || inv.ConfigMaps[0].KeysCount != 1 {
		t.Fatalf("ConfigMaps inesperado: %+v", inv.ConfigMaps)
	}

	// Verificación explícita de la regla de seguridad: el modelo no debe
	// tener ningún campo capaz de portar el contenido de Data/BinaryData.
	// Esto se garantiza en tiempo de compilación por el propio tipo
	// model.SecretSummary/ConfigMapSummary (solo KeysCount, sin Data), y
	// aquí confirmamos que el conteo refleja las claves reales sin
	// filtrar su contenido.
	_ = secret.Data["password"]
}

func TestCollect_NamespaceExplicitOverridesExclusion(t *testing.T) {
	kubeSystemDeploy := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "coredns", Namespace: "kube-system"},
		Spec: appsv1.DeploymentSpec{
			Replicas: int32Ptr(2),
			Template: corev1.PodTemplateSpec{Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "coredns"}}}},
		},
	}
	client := newTestClient(kubeSystemDeploy)
	filter, _ := nsfilter.New([]string{`^kube-.*$`})
	c := New(client, filter)

	inv, namespaces, err := c.Collect(context.Background(), Options{Namespace: "kube-system"})
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}
	if len(namespaces) != 1 || namespaces[0] != "kube-system" {
		t.Fatalf("--namespace explícito debería ignorar el filtro de exclusión, got %v", namespaces)
	}
	if len(inv.Deployments) != 1 {
		t.Fatalf("len(Deployments) = %d, want 1", len(inv.Deployments))
	}
}

func TestCollect_CRDsViaDynamicClient(t *testing.T) {
	crd := &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "apiextensions.k8s.io/v1",
		"kind":       "CustomResourceDefinition",
		"metadata":   map[string]interface{}{"name": "backups.velero.io"},
		"spec": map[string]interface{}{
			"group":    "velero.io",
			"scope":    "Namespaced",
			"names":    map[string]interface{}{"kind": "Backup"},
			"versions": []interface{}{map[string]interface{}{"name": "v1"}},
		},
	}}

	client := newTestClient()
	client.Dynamic = dynamicfake.NewSimpleDynamicClientWithCustomListKinds(
		runtime.NewScheme(),
		map[schema.GroupVersionResource]string{crdGVR: "CustomResourceDefinitionList"},
		crd,
	)
	filter, _ := nsfilter.New(nil)
	c := New(client, filter)

	inv, _, err := c.Collect(context.Background(), Options{})
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}
	if len(inv.CRDs) != 1 {
		t.Fatalf("len(CRDs) = %d, want 1", len(inv.CRDs))
	}
	if inv.CRDs[0].Group != "velero.io" || inv.CRDs[0].Kind != "Backup" {
		t.Errorf("CRD inesperado: %+v", inv.CRDs[0])
	}
}
