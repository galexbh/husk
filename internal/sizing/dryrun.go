package sizing

import (
	"fmt"
	"strings"

	"github.com/galexbh/husk/internal/model"
)

// BuildDryRunPatches genera un model.DryRunPatch por cada contenedor con
// una recomendación pendiente (se omiten los veredictos "saludable" y
// "sin-datos"). --dry-run es estrictamente local: esta función solo
// construye texto, nunca toca el cluster.
func BuildDryRunPatches(report *model.SizingReport) []model.DryRunPatch {
	patches := make([]model.DryRunPatch, 0)
	for _, w := range report.Workloads {
		for _, c := range w.Containers {
			if c.Verdict == "saludable" || c.Verdict == "sin-datos" || c.Verdict == "" {
				continue
			}
			if c.RecommendedCPURequest == "" && c.RecommendedMemRequest == "" {
				continue
			}
			patches = append(patches, model.DryRunPatch{
				Kind: w.Kind, Namespace: w.Namespace, Name: w.Name, Container: c.Name, Verdict: c.Verdict,
				YAML: renderPatchYAML(w, c),
			})
		}
	}
	return patches
}

// renderPatchYAML produce un fragmento de strategic merge patch, listo
// para pegarse en el manifiesto del workload o aplicarse manualmente con
// `kubectl patch` — husk nunca lo aplica por sí mismo.
func renderPatchYAML(w model.WorkloadSizing, c model.ContainerSizing) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# %s/%s en namespace %s, contenedor %q (%s)\n", w.Kind, w.Name, w.Namespace, c.Name, c.Verdict)
	b.WriteString("spec:\n  template:\n    spec:\n      containers:\n")
	fmt.Fprintf(&b, "        - name: %s\n          resources:\n", c.Name)

	if c.RecommendedCPURequest != "" || c.RecommendedMemRequest != "" {
		b.WriteString("            requests:\n")
		if c.RecommendedCPURequest != "" {
			fmt.Fprintf(&b, "              cpu: %s\n", c.RecommendedCPURequest)
		}
		if c.RecommendedMemRequest != "" {
			fmt.Fprintf(&b, "              memory: %s\n", c.RecommendedMemRequest)
		}
	}
	if c.RecommendedCPULimit != "" || c.RecommendedMemLimit != "" {
		b.WriteString("            limits:\n")
		if c.RecommendedCPULimit != "" {
			fmt.Fprintf(&b, "              cpu: %s\n", c.RecommendedCPULimit)
		}
		if c.RecommendedMemLimit != "" {
			fmt.Fprintf(&b, "              memory: %s\n", c.RecommendedMemLimit)
		}
	}

	return b.String()
}
