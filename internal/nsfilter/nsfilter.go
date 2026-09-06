// Package nsfilter implementa el filtro transversal de namespaces: qué
// namespaces se excluyen de los análisis por defecto (namespaces de sistema
// de Kubernetes/OpenShift), configurable vía YAML, con soporte de
// excepciones explícitas (por ejemplo, openshift-adp siempre se evalúa
// durante DR readiness aunque esté excluido del análisis de aplicaciones).
package nsfilter

import (
	"fmt"
	"regexp"
)

// Filter decide si un namespace debe excluirse de un análisis.
type Filter struct {
	patterns   []*regexp.Regexp
	exceptions map[string]bool
}

// New compila los patrones de exclusión dados. alwaysInclude lista
// namespaces que nunca se excluyen, sin importar los patrones.
func New(patterns []string, alwaysInclude ...string) (*Filter, error) {
	compiled := make([]*regexp.Regexp, 0, len(patterns))
	for _, p := range patterns {
		re, err := regexp.Compile(p)
		if err != nil {
			return nil, fmt.Errorf("patrón de exclusión de namespaces inválido %q: %w", p, err)
		}
		compiled = append(compiled, re)
	}

	exceptions := make(map[string]bool, len(alwaysInclude))
	for _, ns := range alwaysInclude {
		exceptions[ns] = true
	}

	return &Filter{patterns: compiled, exceptions: exceptions}, nil
}

// Exclude reporta si ns coincide con algún patrón de exclusión y no es una
// excepción explícita.
func (f *Filter) Exclude(ns string) bool {
	if f.exceptions[ns] {
		return false
	}
	for _, re := range f.patterns {
		if re.MatchString(ns) {
			return true
		}
	}
	return false
}

// Apply devuelve el subconjunto de namespaces que no está excluido.
func (f *Filter) Apply(namespaces []string) []string {
	out := make([]string, 0, len(namespaces))
	for _, ns := range namespaces {
		if !f.Exclude(ns) {
			out = append(out, ns)
		}
	}
	return out
}
