package dr

import (
	"context"
	"fmt"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/galexbh/husk/internal/huskerr"
	"github.com/galexbh/husk/internal/model"
)

// assessOADP valida la presencia y salud del operador OADP en
// openshift-adp. Este chequeo siempre se ejecuta, aunque el namespace
// openshift-adp coincida con los patrones de exclusión configurados.
func (a *Assessor) assessOADP(ctx context.Context, dr *model.DRReadiness) error {
	_, err := a.client.Kubernetes.CoreV1().Namespaces().Get(ctx, oadpNamespace, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			dr.OADPMessage = fmt.Sprintf("el namespace %s no existe: OADP no está instalado", oadpNamespace)
			return nil
		}
		return huskerr.New("no se pudo verificar el namespace "+oadpNamespace, "verifica el permiso de lectura sobre namespaces", err)
	}
	dr.OADPNamespaceExists = true

	dpas, err := a.client.Dynamic.Resource(dataProtectionAppGVR).Namespace(oadpNamespace).List(ctx, listOpts)
	if err != nil {
		if apierrors.IsNotFound(err) {
			dr.OADPMessage = "el CRD DataProtectionApplication (oadp.openshift.io) no está instalado: OADP no está instalado"
			return nil
		}
		return huskerr.New("no se pudo listar DataProtectionApplication en "+oadpNamespace, "verifica el permiso de lectura sobre dataprotectionapplications.oadp.openshift.io", err)
	}
	if len(dpas.Items) == 0 {
		dr.OADPMessage = fmt.Sprintf("no hay ninguna DataProtectionApplication en %s: OADP no está configurado", oadpNamespace)
		return nil
	}
	dr.OADPInstalled = true

	healthy, message := dpaHealth(dpas.Items[0])
	dr.OADPHealthy = healthy
	dr.OADPMessage = message
	return nil
}

// dpaHealth interpreta status.conditions de una DataProtectionApplication,
// buscando la condición type=Reconciled.
func dpaHealth(dpa unstructured.Unstructured) (bool, string) {
	conditions, found, _ := unstructured.NestedSlice(dpa.Object, "status", "conditions")
	if !found {
		return false, "la DataProtectionApplication no reporta status.conditions todavía"
	}
	for _, c := range conditions {
		cond, ok := c.(map[string]interface{})
		if !ok {
			continue
		}
		condType, _, _ := unstructured.NestedString(cond, "type")
		if condType != "Reconciled" {
			continue
		}
		status, _, _ := unstructured.NestedString(cond, "status")
		message, _, _ := unstructured.NestedString(cond, "message")
		if status == "True" {
			return true, "OADP reconciliado correctamente"
		}
		return false, "OADP no está healthy: " + message
	}
	return false, "la DataProtectionApplication no tiene una condición Reconciled"
}
