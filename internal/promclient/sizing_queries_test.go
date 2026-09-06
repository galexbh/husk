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

func TestMemoryPeakQuery_UsesMaxOverTime(t *testing.T) {
	p := SizingQueryParams{Namespace: "shop", OwnerKind: "Deployment", OwnerName: "api", Container: "api", Lookback: "7d"}
	q := MemoryPeakQuery(p)
	if !strings.HasPrefix(q, "max_over_time(") {
		t.Errorf("MemoryPeakQuery() = %q, se esperaba que empezara con max_over_time(", q)
	}
}
