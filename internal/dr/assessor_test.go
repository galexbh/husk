package dr

import (
	"context"
	"testing"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/galexbh/husk/internal/config"
	"github.com/galexbh/husk/internal/k8sclient"
	"github.com/galexbh/husk/internal/nsfilter"
)

var gvrToListKind = map[schema.GroupVersionResource]string{
	backupGVR:                "BackupList",
	backupStorageLocationGVR: "BackupStorageLocationList",
	dataProtectionAppGVR:     "DataProtectionApplicationList",
	volumeSnapshotClassGVR:   "VolumeSnapshotClassList",
}

func testClient(k8sObjs []runtime.Object, dynObjs ...runtime.Object) *k8sclient.Client {
	return &k8sclient.Client{
		Kubernetes: fake.NewSimpleClientset(k8sObjs...),
		Dynamic:    dynamicfake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), gvrToListKind, dynObjs...),
	}
}

func testDRCfg() config.DRConfig {
	return config.DRConfig{BackupMaxAge: "24h", EtcdSnapshotMaxAge: "168h"}
}

func int32Ptr(v int32) *int32 { return &v }

func dpa(namespace string, healthy bool) *unstructured.Unstructured {
	status := "True"
	message := "OADP reconciliado correctamente"
	if !healthy {
		status = "False"
		message = "error de configuración"
	}
	return &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "oadp.openshift.io/v1alpha1",
		"kind":       "DataProtectionApplication",
		"metadata":   map[string]interface{}{"name": "dpa", "namespace": namespace},
		"status": map[string]interface{}{
			"conditions": []interface{}{
				map[string]interface{}{"type": "Reconciled", "status": status, "message": message},
			},
		},
	}}
}

func backup(name string, includedNamespaces []string, phase string, completion time.Time) *unstructured.Unstructured {
	ns := make([]interface{}, len(includedNamespaces))
	for i, n := range includedNamespaces {
		ns[i] = n
	}
	return &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "velero.io/v1",
		"kind":       "Backup",
		"metadata":   map[string]interface{}{"name": name, "namespace": oadpNamespace},
		"spec":       map[string]interface{}{"includedNamespaces": ns},
		"status": map[string]interface{}{
			"phase":               phase,
			"completionTimestamp": completion.Format(time.RFC3339),
		},
	}}
}

func TestAssess_OADPNotInstalled(t *testing.T) {
	client := testClient(nil)
	filter, _ := nsfilter.New(nil)
	a := New(client, filter, testDRCfg())

	result, err := a.Assess(context.Background())
	if err != nil {
		t.Fatalf("Assess: %v", err)
	}
	if result.OADPInstalled {
		t.Error("OADPInstalled debería ser false sin namespace openshift-adp")
	}
	if result.OADPMessage == "" {
		t.Error("se esperaba un mensaje explicando por qué OADP no está instalado")
	}
}

func TestAssess_OADPHealthy(t *testing.T) {
	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: oadpNamespace}}
	client := testClient([]runtime.Object{ns}, dpa(oadpNamespace, true))
	filter, _ := nsfilter.New(nil)
	a := New(client, filter, testDRCfg())

	result, err := a.Assess(context.Background())
	if err != nil {
		t.Fatalf("Assess: %v", err)
	}
	if !result.OADPInstalled || !result.OADPHealthy {
		t.Errorf("OADP debería estar instalado y healthy: %+v", result)
	}
}

func TestAssess_BackupCoverage(t *testing.T) {
	now := time.Now()
	nsShop := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "shop"}}
	nsBilling := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "billing"}}
	nsLegacy := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "legacy"}}

	recentBackup := backup("recent", []string{"shop"}, "Completed", now.Add(-1*time.Hour))
	staleBackup := backup("old", []string{"billing"}, "Completed", now.Add(-48*time.Hour))

	client := testClient([]runtime.Object{nsShop, nsBilling, nsLegacy}, recentBackup, staleBackup)
	filter, _ := nsfilter.New(nil)
	a := New(client, filter, testDRCfg())

	result, err := a.Assess(context.Background())
	if err != nil {
		t.Fatalf("Assess: %v", err)
	}

	if !contains(result.NamespacesWithoutBackup, "legacy") {
		t.Errorf("NamespacesWithoutBackup = %v, want incluir legacy", result.NamespacesWithoutBackup)
	}
	if !contains(result.NamespacesWithStaleBackup, "billing") {
		t.Errorf("NamespacesWithStaleBackup = %v, want incluir billing", result.NamespacesWithStaleBackup)
	}
	if contains(result.NamespacesWithoutBackup, "shop") || contains(result.NamespacesWithStaleBackup, "shop") {
		t.Errorf("shop tiene un backup reciente, no debería aparecer en ninguna lista de gaps")
	}
}

func TestAssess_EtcdNotApplicableOnVanillaKubernetes(t *testing.T) {
	client := testClient(nil)
	client.IsOpenShift = false
	filter, _ := nsfilter.New(nil)
	a := New(client, filter, testDRCfg())

	result, err := a.Assess(context.Background())
	if err != nil {
		t.Fatalf("Assess: %v", err)
	}
	if result.EtcdCheckApplicable {
		t.Error("EtcdCheckApplicable debería ser false en Kubernetes vanilla")
	}
	if result.EtcdSnapshot != nil {
		t.Error("EtcdSnapshot debería quedar nil cuando el chequeo no aplica")
	}
}

func TestAssess_EtcdUnverifiableOnOpenShift(t *testing.T) {
	client := testClient(nil)
	client.IsOpenShift = true
	filter, _ := nsfilter.New(nil)
	a := New(client, filter, testDRCfg())

	result, err := a.Assess(context.Background())
	if err != nil {
		t.Fatalf("Assess: %v", err)
	}
	if !result.EtcdCheckApplicable {
		t.Error("EtcdCheckApplicable debería ser true en OpenShift")
	}
	if result.EtcdSnapshot == nil || result.EtcdSnapshot.Verifiable {
		t.Errorf("se esperaba EtcdSnapshot no verificable sin ningún CronJob/Job de backup, got %+v", result.EtcdSnapshot)
	}
}

func TestAssess_StorageClassesCSISupport(t *testing.T) {
	scName := "gp3"
	scWithoutCSI := "nfs"
	pvc1 := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{Name: "data-1", Namespace: "shop"},
		Spec:       corev1.PersistentVolumeClaimSpec{StorageClassName: &scName},
	}
	pvc2 := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{Name: "data-2", Namespace: "shop"},
		Spec:       corev1.PersistentVolumeClaimSpec{StorageClassName: &scWithoutCSI},
	}
	sc1 := &storagev1.StorageClass{ObjectMeta: metav1.ObjectMeta{Name: scName}, Provisioner: "ebs.csi.aws.com"}
	sc2 := &storagev1.StorageClass{ObjectMeta: metav1.ObjectMeta{Name: scWithoutCSI}, Provisioner: "nfs.csi.k8s.io"}

	vsc := &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "snapshot.storage.k8s.io/v1",
		"kind":       "VolumeSnapshotClass",
		"metadata":   map[string]interface{}{"name": "ebs-vsc"},
		"driver":     "ebs.csi.aws.com",
	}}

	client := testClient([]runtime.Object{pvc1, pvc2, sc1, sc2}, vsc)
	filter, _ := nsfilter.New(nil)
	a := New(client, filter, testDRCfg())

	result, err := a.Assess(context.Background())
	if err != nil {
		t.Fatalf("Assess: %v", err)
	}
	if !contains(result.StorageClassesWithoutCSISnapshot, scWithoutCSI) {
		t.Errorf("StorageClassesWithoutCSISnapshot = %v, want incluir %s", result.StorageClassesWithoutCSISnapshot, scWithoutCSI)
	}
	if contains(result.StorageClassesWithoutCSISnapshot, scName) {
		t.Errorf("%s tiene VolumeSnapshotClass, no debería aparecer como sin soporte", scName)
	}
}

func TestAssess_PDBAndTopologySpread(t *testing.T) {
	nsShop := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "shop"}}
	labelsMap := map[string]string{"app": "api"}

	uncovered := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "api", Namespace: "shop"},
		Spec: appsv1.DeploymentSpec{
			Replicas: int32Ptr(3),
			Template: corev1.PodTemplateSpec{ObjectMeta: metav1.ObjectMeta{Labels: labelsMap}},
		},
	}
	covered := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "worker", Namespace: "shop"},
		Spec: appsv1.DeploymentSpec{
			Replicas: int32Ptr(3),
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"app": "worker"}},
				Spec: corev1.PodSpec{
					TopologySpreadConstraints: []corev1.TopologySpreadConstraint{{MaxSkew: 1}},
				},
			},
		},
	}
	pdb := &policyv1.PodDisruptionBudget{
		ObjectMeta: metav1.ObjectMeta{Name: "worker-pdb", Namespace: "shop"},
		Spec:       policyv1.PodDisruptionBudgetSpec{Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": "worker"}}},
	}

	client := testClient([]runtime.Object{nsShop, uncovered, covered, pdb})
	filter, _ := nsfilter.New(nil)
	a := New(client, filter, testDRCfg())

	result, err := a.Assess(context.Background())
	if err != nil {
		t.Fatalf("Assess: %v", err)
	}

	if result.CriticalWorkloadsTotal != 2 {
		t.Fatalf("CriticalWorkloadsTotal = %d, want 2", result.CriticalWorkloadsTotal)
	}
	if len(result.MissingPDBs) != 1 || result.MissingPDBs[0].Name != "api" {
		t.Errorf("MissingPDBs = %+v, want solo 'api'", result.MissingPDBs)
	}
	if len(result.MissingTopologySpread) != 1 || result.MissingTopologySpread[0].Name != "api" {
		t.Errorf("MissingTopologySpread = %+v, want solo 'api'", result.MissingTopologySpread)
	}
}

func contains(list []string, want string) bool {
	for _, v := range list {
		if v == want {
			return true
		}
	}
	return false
}
