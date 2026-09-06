package config

// Default devuelve la configuración por defecto de husk, tal como se
// documenta en CLAUDE.md. Estos valores están hardcodeados aquí y son
// sobrescribibles vía ~/.husk/config.yaml o --config (ver Load).
func Default() *Config {
	return &Config{
		Namespaces: NamespacesConfig{
			ExcludePatterns: []string{
				`^kube-.*$`,
				`^openshift.*$`,
				`^default$`,
			},
		},
		Sizing: SizingConfig{
			CPUPercentile:       0.95,
			MemoryPercentile:    0.99,
			Lookback:            "7d",
			CPULimitMultiplier:  3.0,
			MemoryBufferPercent: 0.2,
			OverProvisionFactor: 2.0,
			SidecarContainerNames: []string{
				"istio-proxy", "istio-init", "linkerd-proxy",
				"dynatrace-oneagent", "oneagent", "zabbix-agent", "zabbix-agent2",
				"datadog-agent", "filebeat", "fluentd", "fluent-bit", "vault-agent",
			},
		},
		Capacity: CapacityConfig{
			HeadroomThresholdPercent: 30.0,
			Lookback:                 "7d",
		},
		DR: DRConfig{
			BackupMaxAge:       "24h",
			EtcdSnapshotMaxAge: "7d",
		},
		Score: ScoreConfig{
			Weights: ScoreWeights{
				Sizing:   0.3,
				DR:       0.3,
				Capacity: 0.2,
				PDB:      0.1,
				Topology: 0.1,
			},
		},
		Inventory: InventoryConfig{
			IncludeExtended: false,
			Excel: InventoryExcelConfig{
				HighlightRisks: true,
				FreezeColumns:  2,
			},
		},
		History: HistoryConfig{
			RetainCount: 30,
		},
	}
}
