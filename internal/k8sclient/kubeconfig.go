// Package k8sclient recolecta datos de los API servers de Kubernetes y
// OpenShift: carga de kubeconfig, clientsets, detección de OpenShift y
// probes de acceso de lectura. No genera salida visual ni contiene lógica
// de reporte — eso vive en internal/report.
package k8sclient

import (
	"fmt"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/galexbh/husk/internal/huskerr"
)

// LoadRESTConfig resuelve un *rest.Config a partir de --kubeconfig,
// $KUBECONFIG o el kubeconfig por defecto (~/.kube/config), respetando
// contextName cuando se indica. Si no hay kubeconfig utilizable, hace
// fallback a la configuración in-cluster (rest.InClusterConfig), para
// cuando husk corre dentro de un pod vía ServiceAccount.
func LoadRESTConfig(kubeconfigPath, contextName string) (*rest.Config, error) {
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	if kubeconfigPath != "" {
		loadingRules.ExplicitPath = kubeconfigPath
	}

	overrides := &clientcmd.ConfigOverrides{}
	if contextName != "" {
		overrides.CurrentContext = contextName
	}

	clientConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, overrides)
	cfg, err := clientConfig.ClientConfig()
	if err == nil {
		return cfg, nil
	}

	if inClusterCfg, inClusterErr := rest.InClusterConfig(); inClusterErr == nil {
		return inClusterCfg, nil
	}

	return nil, huskerr.New(
		fmt.Sprintf("no se pudo cargar un kubeconfig válido: %v", err),
		"ejecuta `oc login` o `kubectl config use-context` para autenticarte, o especifica --kubeconfig/$KUBECONFIG. "+
			"Si husk corre dentro de un pod, verifica que el ServiceAccount tenga un token montado.",
		err,
	)
}
