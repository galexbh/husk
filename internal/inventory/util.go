package inventory

import "strings"

// joinStrings concatena parts con sep, sin depender de strings.Join
// directamente en cada call site (mantiene los otros archivos más legibles).
func joinStrings(parts []string, sep string) string {
	return strings.Join(parts, sep)
}
