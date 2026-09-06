// Package rbac genera el ClusterRole y ClusterRoleBinding mínimos de solo
// lectura (get/list/watch) que necesita `husk init rbac`, para ejecutar el
// CLI mediante un ServiceAccount en CronJobs o CI/CD.
package rbac

import (
	"bytes"
	_ "embed"
	"text/template"
)

//go:embed templates/clusterrole.yaml.tmpl
var clusterRoleTemplate string

//go:embed templates/clusterrolebinding.yaml.tmpl
var clusterRoleBindingTemplate string

// Params parametriza los manifiestos generados.
type Params struct {
	// Name es el nombre compartido por el ClusterRole y el ClusterRoleBinding.
	Name string
	// ServiceAccountName y Namespace identifican el ServiceAccount al que se
	// le otorgan los permisos.
	ServiceAccountName string
	Namespace          string
}

// Render devuelve el ClusterRole y el ClusterRoleBinding como YAML,
// concatenados con el separador de documentos "---".
func Render(params Params) (string, error) {
	var buf bytes.Buffer
	for i, src := range []string{clusterRoleTemplate, clusterRoleBindingTemplate} {
		if i > 0 {
			buf.WriteString("---\n")
		}
		tmpl, err := template.New("rbac").Parse(src)
		if err != nil {
			return "", err
		}
		if err := tmpl.Execute(&buf, params); err != nil {
			return "", err
		}
	}
	return buf.String(), nil
}
