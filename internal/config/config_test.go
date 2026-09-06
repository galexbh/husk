package config

import "testing"

func TestDefault_WeightsSumToOne(t *testing.T) {
	w := Default().Score.Weights
	total := w.Sizing + w.DR + w.Capacity + w.PDB + w.Topology
	if total < 0.999 || total > 1.001 {
		t.Fatalf("los pesos del score deben sumar 1.0, suman %f", total)
	}
}

func TestLoad_NoPath_ReturnsDefaults(t *testing.T) {
	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Load(\"\") no debería fallar: %v", err)
	}
	if cfg.Sizing.CPUPercentile != Default().Sizing.CPUPercentile {
		t.Fatalf("se esperaban los defaults de sizing, got %+v", cfg.Sizing)
	}
}

func TestLoad_MissingExplicitFile_ReturnsError(t *testing.T) {
	if _, err := Load("testdata/does-not-exist.yaml"); err == nil {
		t.Fatal("se esperaba un error al indicar un archivo de configuración inexistente")
	}
}
