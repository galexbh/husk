---
title: Configuración (config.yaml)
description: Esquema completo del archivo de configuración de husk.
---

`husk` carga su configuración en dos capas:

1. **Defaults hardcodeados** en el binario (documentados en la tabla de abajo).
2. **Archivo YAML opcional**, en `~/.husk/config.yaml` o vía `--config <path>`,
   que sobrescribe solo las claves que declares — no hace falta repetir todo
   el esquema.

```sh
husk score --config ./mi-config.yaml
```

## Esquema completo

```yaml
namespaces:
  exclude_patterns:
    - "^kube-.*$"
    - "^openshift.*$"
    - "^default$"

sizing:
  cpu_percentile: 0.95
  memory_percentile: 0.99
  lookback: "7d"
  cpu_limit_multiplier: 3.0
  memory_buffer_percent: 0.2
  over_provision_factor: 2.0

capacity:
  headroom_threshold_percent: 30.0
  lookback: "7d"

dr:
  backup_max_age: "24h"
  etcd_snapshot_max_age: "7d"

score:
  weights:
    sizing: 0.3
    dr: 0.3
    capacity: 0.2
    pdb: 0.1
    topology: 0.1

inventory:
  include_extended: false
  excel:
    highlight_risks: true
    freeze_columns: 2

history:
  retain_count: 30
```

## Referencia de claves

| Clave | Tipo | Default | Descripción |
|---|---|---|---|
| `namespaces.exclude_patterns` | `[]string` | ver arriba | Regex de namespaces excluidos por defecto de los análisis. |
| `sizing.cpu_percentile` | `float` | `0.95` | Percentil de CPU usado para el sizing. |
| `sizing.memory_percentile` | `float` | `0.99` | Percentil de memoria usado para el sizing. |
| `sizing.lookback` | `string` (duración) | `"7d"` | Ventana histórica de métricas de sizing. |
| `sizing.cpu_limit_multiplier` | `float` | `3.0` | Multiplicador sugerido para `limits.cpu` cuando un contenedor no tiene límites. |
| `sizing.memory_buffer_percent` | `float` | `0.2` | Buffer sugerido sobre el pico de memoria observado. |
| `sizing.over_provision_factor` | `float` | `2.0` | Si el request declarado supera al consumo observado por este factor, se marca "sobreaprovisionado". |
| `capacity.headroom_threshold_percent` | `float` | `30.0` | Por debajo de este % de headroom libre (CPU o memoria), un nodo se marca en riesgo de saturación. Coincide con el criterio de la dimensión Capacity del score. |
| `capacity.lookback` | `string` (duración) | `"7d"` | Ventana histórica para el consumo observado por nodo. |
| `dr.backup_max_age` | `string` (duración) | `"24h"` | Antigüedad máxima aceptable del backup más reciente de un namespace. |
| `dr.etcd_snapshot_max_age` | `string` (duración) | `"7d"` | Antigüedad máxima aceptable del snapshot de etcd. |
| `score.weights.*` | `float` | `sizing 0.3, dr 0.3, capacity 0.2, pdb 0.1, topology 0.1` | Pesos del score de resiliencia por dimensión; deben sumar 1.0. |
| `inventory.include_extended` | `bool` | `false` | Incluye RBAC/NetworkPolicies/PDBs/ResourceQuotas/LimitRanges/HPAs/Ingresses por defecto (equivalente a `--extended`). |
| `inventory.excel.highlight_risks` | `bool` | `true` | Colores condicionales de riesgo (rojo/amarillo/verde) en el Excel generado. |
| `inventory.excel.freeze_columns` | `int` | `2` | Columnas congeladas junto con la fila de encabezados en cada hoja. |
| `history.retain_count` | `int` | `30` | Snapshots recientes conservados por cluster en `~/.husk/history/`; los más antiguos se podan automáticamente en cada `report generate`. |

:::note[Excepción de namespaces]
El estado del operador OADP en `openshift-adp` **siempre** se evalúa durante
`husk dr assess`, aunque ese namespace coincida con
`namespaces.exclude_patterns` — no hay forma de excluirlo, es intencional.
:::

:::caution[Los pesos del score no se normalizan silenciosamente entre sí]
Si `score.weights` no suma 1.0, el score se sigue calculando (y se
renormaliza automáticamente entre las dimensiones que estén *disponibles* en
esa ejecución, por ejemplo cuando falta Prometheus) pero el resultado puede
sorprenderte. Mantén la suma en 1.0 salvo que sepas exactamente lo que
buscas.
:::
