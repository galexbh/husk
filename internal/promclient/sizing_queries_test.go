package promclient

import (
	"strings"
	"testing"
)

func TestCPUPercentileQuery_ContainsExpectedLabels(t *testing.T) {
	p := SizingQueryParams{
		Namespace: "shop", OwnerKind: "Deployment", OwnerName: "api", Container: "api",
		Lookback: "7d", CPUPercentile: 0.95,
	}
	q := CPUPercentileQuery(p)

	for _, want := range []string{`namespace="shop"`, `container="api"`, `owner_kind="Deployment"`, `owner_name="api"`, "0.95", "[7d:5m]"} {
		if !strings.Contains(q, want) {
			t.Errorf("CPUPercentileQuery() = %q, se esperaba que contuviera %q", q, want)
		}
	}
}

func TestMemoryPercentileQuery_ContainsExpectedLabels(t *testing.T) {
	p := SizingQueryParams{Namespace: "shop", OwnerKind: "StatefulSet", OwnerName: "db", Container: "db", Lookback: "7d", MemPercentile: 0.99}
	q := MemoryPercentileQuery(p)
	for _, want := range []string{`namespace="shop"`, `container="db"`, `owner_kind="StatefulSet"`, "0.99"} {
		if !strings.Contains(q, want) {
			t.Errorf("MemoryPercentileQuery() = %q, se esperaba que contuviera %q", q, want)
		}
	}
}

func TestCPUPercentileQuery_DeploymentJoinsThroughReplicaSet(t *testing.T) {
	p := SizingQueryParams{
		Namespace: "shop", OwnerKind: "Deployment", OwnerName: "api", Container: "api",
		Lookback: "7d", CPUPercentile: 0.95,
	}
	q := CPUPercentileQuery(p)

	// kube_pod_owner nunca tiene owner_kind="Deployment" — el owner directo
	// del pod es siempre el ReplicaSet intermedio. La query debe saltar por
	// kube_replicaset_owner en vez de matchear Deployment directamente ahí.
	for _, want := range []string{
		`kube_pod_owner{namespace="shop", owner_kind="ReplicaSet"}`,
		`label_replace(`,
		`"replicaset", "$1", "owner_name", "(.*)"`,
		`on(replicaset, namespace) group_left()`,
		`kube_replicaset_owner{namespace="shop", owner_kind="Deployment", owner_name="api"}`,
		// El join de dos saltos debe quedar agrupado entre paréntesis justo
		// detrás del `group_left()` externo: sin esa agrupación, PromQL lo
		// evalúa como una cadena plana de `*` de izquierda a derecha y el
		// primer group_left() (sin lista de labels) descarta la label
		// "replicaset" antes de que el segundo join la necesite — volviendo
		// a producir vector vacío en silencio.
		`group_left() (label_replace(`,
	} {
		if !strings.Contains(q, want) {
			t.Errorf("CPUPercentileQuery() = %q, se esperaba que contuviera %q", q, want)
		}
	}
}

func TestMemoryPeakQuery_UsesMaxOverTime(t *testing.T) {
	p := SizingQueryParams{Namespace: "shop", OwnerKind: "Deployment", OwnerName: "api", Container: "api", Lookback: "7d"}
	q := MemoryPeakQuery(p)
	if !strings.HasPrefix(q, "max_over_time(") {
		t.Errorf("MemoryPeakQuery() = %q, se esperaba que empezara con max_over_time(", q)
	}
}
