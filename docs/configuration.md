# Configuración de husk

husk carga su configuración en dos capas:

1. **Defaults hardcodeados** en `internal/config/defaults.go`.
2. **Archivo YAML opcional**, en `~/.husk/config.yaml` o vía `--config <path>`,
   que sobrescribe solo las claves declaradas.

Ver [`examples/config.yaml`](../examples/config.yaml) para un ejemplo
completo y comentado.

## Esquema

| Clave | Tipo | Default | Descripción |
|---|---|---|---|
| `namespaces.exclude_patterns` | `[]string` | `["^kube-.*$", "^openshift.*$", "^default$"]` | Regex de namespaces excluidos por defecto de los análisis. |
| `sizing.cpu_percentile` | `float` | `0.95` | Percentil de CPU usado para el sizing. |
| `sizing.memory_percentile` | `float` | `0.99` | Percentil de memoria usado para el sizing. |
| `sizing.lookback` | `string` (duración) | `"7d"` | Ventana histórica de métricas. |
| `sizing.cpu_limit_multiplier` | `float` | `3.0` | Multiplicador sugerido para limits de CPU. |
| `sizing.memory_buffer_percent` | `float` | `0.2` | Buffer sugerido sobre el pico de memoria observado. |
| `sizing.over_provision_factor` | `float` | `2.0` | Si el request declarado supera al consumo observado por este factor, se marca "sobreaprovisionado". |
| `capacity.headroom_threshold_percent` | `float` | `30.0` | Por debajo de este % de headroom libre (CPU o memoria), un nodo se marca en riesgo de saturación. |
| `capacity.lookback` | `string` (duración) | `"7d"` | Ventana histórica para el consumo observado por nodo. |
| `dr.backup_max_age` | `string` (duración) | `"24h"` | Antigüedad máxima aceptable de un backup. |
| `dr.etcd_snapshot_max_age` | `string` (duración) | `"7d"` | Antigüedad máxima aceptable del snapshot de etcd. |
| `score.weights.*` | `float` | `sizing 0.3, dr 0.3, capacity 0.2, pdb 0.1, topology 0.1` | Pesos del score de resiliencia; deben sumar 1.0. |
| `inventory.include_extended` | `bool` | `false` | Incluye RBAC/NetworkPolicies/PDBs/ResourceQuotas/LimitRanges/HPAs/Ingresses por defecto. |
| `inventory.excel.highlight_risks` | `bool` | `true` | Colores condicionales de riesgo en el Excel generado. |
| `inventory.excel.freeze_columns` | `int` | `2` | Columnas congeladas junto con la fila de encabezados. |
| `history.retain_count` | `int` | `30` | Snapshots recientes conservados por cluster en `~/.husk/history/`; los más antiguos se podan automáticamente en cada `report generate`. |

**Excepción de namespaces:** el estado del operador OADP en `openshift-adp`
siempre se evalúa durante `dr assess`, aunque ese namespace coincida con
`namespaces.exclude_patterns`.
