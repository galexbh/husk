package k8sclient

import (
	"context"

	configv1client "github.com/openshift/client-go/config/clientset/versioned"
	routev1client "github.com/openshift/client-go/route/clientset/versioned"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
)

// detectOpenShift construye los clientsets de Route y Config de OpenShift y
// prueba si el objeto ClusterVersion (config.openshift.io/v1) realmente
// existe — la señal estándar para distinguir OpenShift de Kubernetes
// vanilla sin depender del binario `oc`.
func detectOpenShift(restCfg *rest.Config) (routev1client.Interface, configv1client.Interface, bool) {
	routeClient, err := routev1client.NewForConfig(restCfg)
	if err != nil {
		return nil, nil, false
	}
	configClient, err := configv1client.NewForConfig(restCfg)
	if err != nil {
		return nil, nil, false
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultDetectTimeout)
	defer cancel()

	_, err = configClient.ConfigV1().ClusterVersions().Get(ctx, "version", metav1.GetOptions{})
	if err != nil {
		return routeClient, configClient, false
	}
	return routeClient, configClient, true
}
