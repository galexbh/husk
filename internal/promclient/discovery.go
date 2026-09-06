package promclient

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"net/http"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// El router de OpenShift (por donde pasa la route de Thanos Querier) usa el
// certificado del ingress por defecto del cluster, cuya CA se publica en
// este ConfigMap para que otros componentes puedan validarla.
const (
	routerCANamespace = "openshift-config-managed"
	routerCAConfigMap = "default-ingress-cert"
	routerCAKey       = "ca-bundle.crt"
)

// DiscoverTransport intenta resolver la CA del router de OpenShift para
// validar TLS correctamente contra la route de Thanos Querier. Si no puede
// (ConfigMap ausente, sin permiso de lectura, cluster con una CA distinta),
// degrada a InsecureSkipVerify y lo advierte con nivel Warn (visible sin
// --verbose) — nunca falla por esto, para no bloquear el sizing por un
// detalle de TLS.
func DiscoverTransport(ctx context.Context, kube kubernetes.Interface, logger Logger) http.RoundTripper {
	tlsConfig := &tls.Config{}

	cm, err := kube.CoreV1().ConfigMaps(routerCANamespace).Get(ctx, routerCAConfigMap, metav1.GetOptions{})
	if err == nil {
		if caPEM, ok := cm.Data[routerCAKey]; ok && caPEM != "" {
			pool, poolErr := x509.SystemCertPool()
			if poolErr != nil || pool == nil {
				pool = x509.NewCertPool()
			}
			if pool.AppendCertsFromPEM([]byte(caPEM)) {
				tlsConfig.RootCAs = pool
				logger.Debug("CA del router de OpenShift resuelta", "configmap", routerCANamespace+"/"+routerCAConfigMap)
				return &http.Transport{TLSClientConfig: tlsConfig}
			}
		}
	}

	logger.Warn("no se pudo resolver la CA del router de OpenShift; se usará InsecureSkipVerify para Thanos Querier — el bearer token viajará sin validar el certificado del servidor", "error", err)
	tlsConfig.InsecureSkipVerify = true //nolint:gosec // fallback documentado cuando no se puede resolver la CA real del router
	return &http.Transport{TLSClientConfig: tlsConfig}
}
