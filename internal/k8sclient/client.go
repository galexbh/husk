package k8sclient

import (
	"fmt"
	"os"
	"strings"

	configv1client "github.com/openshift/client-go/config/clientset/versioned"
	routev1client "github.com/openshift/client-go/route/clientset/versioned"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/galexbh/husk/internal/huskerr"
)

// Client agrupa los clientsets que husk necesita para hablar con
// Kubernetes y, cuando está presente, OpenShift. Solo recolecta datos: no
// renderiza nada ni contiene lógica de negocio.
//
// CustomResourceDefinitions y otros recursos sin clientset tipado propio
// (por ejemplo, las CRs de Velero en fases posteriores) se leen vía Dynamic
// con un GroupVersionResource explícito, para no arrastrar el módulo
// k8s.io/apiextensions-apiserver completo (trae apiserver/etcd/otel como
// dependencias transitivas) solo por sus tipos.
type Client struct {
	RESTConfig *rest.Config

	Kubernetes kubernetes.Interface
	Dynamic    dynamic.Interface

	// Clientes específicos de OpenShift. Son nil cuando el cluster no es
	// OpenShift o cuando la API correspondiente no está disponible.
	Route  routev1client.Interface
	Config configv1client.Interface

	IsOpenShift bool
}

// New construye un Client a partir del kubeconfig indicado (kubeconfigPath
// y contextName vacíos usan los defaults), detectando además si el cluster
// es OpenShift.
func New(kubeconfigPath, contextName string) (*Client, error) {
	restCfg, err := LoadRESTConfig(kubeconfigPath, contextName)
	if err != nil {
		return nil, err
	}

	kubeClient, err := kubernetes.NewForConfig(restCfg)
	if err != nil {
		return nil, huskerr.New(
			"no se pudo crear el cliente de Kubernetes",
			"verifica que el kubeconfig tenga credenciales válidas (`oc whoami` o `kubectl auth whoami`)",
			err,
		)
	}

	dynClient, err := dynamic.NewForConfig(restCfg)
	if err != nil {
		return nil, huskerr.New(
			"no se pudo crear el cliente dinámico de Kubernetes",
			"verifica que el kubeconfig tenga credenciales válidas",
			err,
		)
	}

	c := &Client{
		RESTConfig: restCfg,
		Kubernetes: kubeClient,
		Dynamic:    dynClient,
	}

	routeClient, configClient, isOS := detectOpenShift(restCfg)
	c.Route = routeClient
	c.Config = configClient
	c.IsOpenShift = isOS

	return c, nil
}

// ServerVersion devuelve la versión del API server de Kubernetes/OpenShift.
func (c *Client) ServerVersion() (string, error) {
	v, err := c.Kubernetes.Discovery().ServerVersion()
	if err != nil {
		return "", huskerr.New(
			"no se pudo consultar la versión del API server",
			"verifica conectividad de red hacia el cluster y que el token de sesión no haya expirado (`oc whoami -t`)",
			err,
		)
	}
	return fmt.Sprintf("%s (%s/%s)", v.GitVersion, v.Platform, v.BuildDate), nil
}

// BearerToken devuelve el bearer token usado para autenticar contra el API
// server, resolviendo BearerTokenFile cuando el token no está inline. Los
// kubeconfigs generados por `oc login`/`kubectl` normalmente traen un token
// inline; autenticación por certificado de cliente o por plugin exec no
// está soportada aquí y produce un error explícito.
func (c *Client) BearerToken() (string, error) {
	if c.RESTConfig.BearerToken != "" {
		return c.RESTConfig.BearerToken, nil
	}
	if c.RESTConfig.BearerTokenFile != "" {
		data, err := os.ReadFile(c.RESTConfig.BearerTokenFile)
		if err != nil {
			return "", huskerr.New(
				"no se pudo leer el BearerTokenFile del kubeconfig",
				"verifica permisos de lectura sobre el archivo de token",
				err,
			)
		}
		return strings.TrimSpace(string(data)), nil
	}
	return "", huskerr.New(
		"el kubeconfig activo no usa un bearer token (autenticación por certificado de cliente o plugin exec)",
		"husk necesita un bearer token para autenticar contra Prometheus/Thanos; vuelve a autenticarte con `oc login` usando un token",
		nil,
	)
}
