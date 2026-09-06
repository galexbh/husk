# Catálogo de métricas PromQL

Este documento cataloga las queries PromQL que usa `husk sizing report`
contra el Thanos Querier de OpenShift (`openshift-monitoring`). Con
`--verbose`, husk registra en el log de debug cada query ejecutada y su
tiempo de respuesta (`internal/promclient.Client.Query`).

## Supuestos

- Requiere **kube-state-metrics** (incluido en el stack de monitoreo de
  OpenShift) para la métrica `kube_pod_owner`, que asocia cada Pod con su
  Deployment/StatefulSet/DaemonSet propietario (`owner_kind`/`owner_name`).
  Si kube-state-metrics no está disponible, las queries simplemente no
  devuelven series y el contenedor se reporta como **"sin datos"** — nunca
  como un error.
- El consumo de CPU/memoria se **suma entre todas las réplicas** del
  workload antes de calcular el percentil: el objetivo es comparar el
  footprint total del workload contra la suma de sus requests/limits, no el
  comportamiento de un pod individual.
- Los umbrales y la ventana histórica vienen de `sizing.*` en la
  configuración (ver [`configuration.md`](configuration.md)):
  `cpu_percentile`, `memory_percentile`, `lookback`,
  `cpu_limit_multiplier`, `memory_buffer_percent`.

## Queries (`internal/promclient/sizing_queries.go`)

### CPU — percentil (`CPUPercentileQuery`)

```promql
quantile_over_time(<cpu_percentile>,
  (
    sum(
      rate(container_cpu_usage_seconds_total{namespace="<ns>", container="<container>", container!="", container!="POD"}[5m])
      * on(pod) group_left() kube_pod_owner{namespace="<ns>", owner_kind="<Kind>", owner_name="<name>"}
    )
  )[<lookback>:5m]
)
```

Resultado: núcleos de CPU. Usado para el veredicto de sizing y, cuando el
contenedor no tiene `limits`, para recomendar
`request = P95` y `limit = P95 * cpu_limit_multiplier`.

### Memoria — percentil (`MemoryPercentileQuery`)

```promql
quantile_over_time(<memory_percentile>,
  (
    sum(
      container_memory_working_set_bytes{namespace="<ns>", container="<container>", container!="", container!="POD"}
      * on(pod) group_left() kube_pod_owner{namespace="<ns>", owner_kind="<Kind>", owner_name="<name>"}
    )
  )[<lookback>:5m]
)
```

Resultado: bytes de memoria working-set. Se usa como base de recomendación
cuando no hay dato de pico disponible.

### Memoria — pico (`MemoryPeakQuery`)

```promql
max_over_time(
  (
    sum(
      container_memory_working_set_bytes{namespace="<ns>", container="<container>", container!="", container!="POD"}
      * on(pod) group_left() kube_pod_owner{namespace="<ns>", owner_kind="<Kind>", owner_name="<name>"}
    )
  )[<lookback>:5m]
)
```

Resultado: bytes de memoria working-set, el máximo observado en la ventana.
Es la base preferida para recomendar `memory.request`/`memory.limit`
(`pico * (1 + memory_buffer_percent)`), ya que la memoria no se recupera
como la CPU y un OOMKill es más costoso que un pico transitorio de CPU.

## Veredictos de `husk sizing report`

| Veredicto | Condición |
|---|---|
| `sin-datos` | Ninguna de las tres queries devolvió una serie (kube-state-metrics ausente, workload sin tráfico en la ventana, etc.). |
| `sin-limites` | Al contenedor le falta `resources.limits.cpu` o `resources.limits.memory`. |
| `sobreaprovisionado` | El `request` declarado supera al consumo observado por más de `overProvisionFactor` (2×). |
| `subaprovisionado` | El consumo observado supera al `request` declarado (riesgo de throttling de CPU o OOMKill de memoria). |
| `saludable` | Ninguna de las condiciones anteriores. |

`--dry-run` (en `husk sizing report --dry-run`) imprime, para cada
contenedor con un veredicto distinto de `saludable`/`sin-datos`, un
fragmento de *strategic merge patch* en YAML con los valores recomendados
para `resources.requests`/`resources.limits` — solo texto local; husk nunca
lo aplica al cluster.
