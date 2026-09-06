package nsfilter

import "testing"

func TestFilter_ExcludesSystemNamespaces(t *testing.T) {
	f, err := New([]string{`^kube-.*$`, `^openshift.*$`, `^default$`})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	cases := map[string]bool{
		"kube-system":     true,
		"kube-public":     true,
		"kube-node-lease": true,
		"default":         true,
		"openshift":       true,
		"openshift-adp":   true,
		"my-app":          false,
	}
	for ns, want := range cases {
		if got := f.Exclude(ns); got != want {
			t.Errorf("Exclude(%q) = %v, want %v", ns, got, want)
		}
	}
}

func TestFilter_AlwaysIncludeException(t *testing.T) {
	f, err := New([]string{`^openshift.*$`}, "openshift-adp")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if f.Exclude("openshift-adp") {
		t.Error("openshift-adp debería estar exento de la exclusión durante DR readiness")
	}
	if !f.Exclude("openshift-monitoring") {
		t.Error("otros namespaces openshift-* deberían seguir excluidos")
	}
}

func TestFilter_Apply(t *testing.T) {
	f, err := New([]string{`^kube-.*$`})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	got := f.Apply([]string{"kube-system", "my-app", "kube-public"})
	if len(got) != 1 || got[0] != "my-app" {
		t.Errorf("Apply = %v, want [my-app]", got)
	}
}

func TestNew_InvalidPattern(t *testing.T) {
	if _, err := New([]string{"("}); err == nil {
		t.Error("se esperaba un error con un patrón regex inválido")
	}
}
