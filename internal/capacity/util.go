package capacity

import (
	"fmt"
	"math"

	"k8s.io/apimachinery/pkg/api/resource"
)

func formatCPU(cores float64) string {
	milli := cores * 1000
	return fmt.Sprintf("%dm", int64(milli+0.5))
}

func formatMemory(bytes float64) string {
	q := resource.NewQuantity(int64(bytes), resource.BinarySI)
	return q.String()
}

func round2(v float64) float64 {
	return math.Round(v*100) / 100
}

// rolesFromLabels extrae los roles de un Node desde sus labels
// node-role.kubernetes.io/<rol>, igual que internal/inventory.
func rolesFromLabels(nodeLabels map[string]string) []string {
	const rolePrefix = "node-role.kubernetes.io/"
	roles := make([]string, 0)
	for label := range nodeLabels {
		if len(label) > len(rolePrefix) && label[:len(rolePrefix)] == rolePrefix {
			roles = append(roles, label[len(rolePrefix):])
		}
	}
	return roles
}
