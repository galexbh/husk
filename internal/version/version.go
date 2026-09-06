// Package version holds the build-time identity of the husk binary.
package version

import "runtime"

// Version, Commit y BuildDate se inyectan en build time vía -ldflags, tanto
// en builds locales (Makefile/Dockerfile) como en CI (GoReleaser/dockers_v2).
// Estos valores por defecto solo se usan con `go run` o en tests.
var (
	Version   = "dev"
	Commit    = "none"
	BuildDate = "unknown"
)

// GoVersion devuelve la versión de Go con la que se compiló el binario.
func GoVersion() string {
	return runtime.Version()
}
