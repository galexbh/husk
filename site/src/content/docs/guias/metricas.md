---
title: Métricas PromQL
description: Catálogo de las queries PromQL que usan husk sizing report y husk capacity nodes.
---

Este documento cataloga las queries PromQL que usa `husk sizing report` y
`husk capacity nodes` contra el Thanos Querier de OpenShift
(`openshift-monitoring`). Con `--verbose`, husk registra en el log de debug
cada query ejecutada y su tiempo de respuesta.

## Supuestos

- Requiere **kube-state-metrics** (incluido en el stack de monitoreo de
  OpenShift) para la métrica `kube_pod_owner`, que asocia cada Pod con su
  Deployment/StatefulSet/DaemonSet propietario (`owner_kind`/`owner_name`).
  Si kube-state-metrics no está disponible, las queries de sizing
  simplemente no devuelven series y el contenedor se reporta como
  **"sin datos"** — nunca como un error.
- El consumo de CPU/memoria de sizing se **suma entre todas las réplicas**
  del workload antes de calcular el percentil: el objetivo es comparar el
  footprint total del workload contra la suma de sus requests/limits, no el
  comportamiento de un pod individual.
- Las queries de capacity usan las métricas de cAdvisor con `id="/"`
  (cgroup raíz de la máquina completa), cuya etiqueta `node` coincide
  directamente con el nombre del Node de Kubernetes — a diferencia de
  node-exporter, cuya etiqueta `instance` (host:puerto) no siempre coincide.
- Los umbrales y la ventana histórica vienen de la
  [configuración](/husk/guias/configuracion/) (`sizing.*`, `capacity.*`).

## Queries de sizing

### CPU — percentil

```txt
quantile_over_time(<cpu_percentile>,
  (
    sum(
      rate(container_cpu_usage_seconds_total{namespace="<ns>", container="<container>", container!="", container!="POD"}[5m])
      * on(pod) group_left() kube_pod_owner{namespace="<ns>", owner_kind="<Kind>", owner_name="<name>"}
    )
  )[<lookback>:5m]
)
```

Resultado: núcleos de CPU. Se usa para el veredicto de sizing y, cuando el
contenedor no tiene `limits`, para recomendar `request = P95` y
`limit = P95 * cpu_limit_multiplier`.

### Memoria — percentil

```txt
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

### Memoria — pico

```txt
max_over_time(
  (
    sum(
      container_memory_working_set_bytes{namespace="<ns>", container="<container>", container!="", container!="POD"}
      * on(pod) group_left() kube_pod_owner{namespace="<ns>", owner_kind="<Kind>", owner_name="<name>"}
    )
  )[<lookback>:5m]
)
```

Resultado: el máximo de memoria working-set observado en la ventana. Es la
base preferida para recomendar `memory.request`/`memory.limit`
(`pico * (1 + memory_buffer_percent)`), ya que la memoria no se recupera
como la CPU y un OOMKill es más costoso que un pico transitorio de CPU.

## Queries de capacity (por nodo)

```txt
# CPU promedio
avg_over_time(rate(container_cpu_usage_seconds_total{id="/", node="<node>"}[5m])[<lookback>:5m])

# CPU pico
max_over_time(rate(container_cpu_usage_seconds_total{id="/", node="<node>"}[5m])[<lookback>:5m])

# Memoria promedio
avg_over_time(container_memory_working_set_bytes{id="/", node="<node>"}[<lookback>])

# Memoria pico
max_over_time(container_memory_working_set_bytes{id="/", node="<node>"}[<lookback>])
```

Este consumo histórico es un **enriquecimiento opcional** del reporte de
capacity: si Thanos Querier no está disponible, `husk capacity nodes` se
genera igual, solo sin estos campos.

## Veredictos de `husk sizing report`

| Veredicto | Condición |
|---|---|
| `sin-datos` | Ninguna de las tres queries devolvió una serie (kube-state-metrics ausente, workload sin tráfico en la ventana, etc.). |
| `sin-limites` | Al contenedor le falta `resources.limits.cpu` o `resources.limits.memory`. |
| `sobreaprovisionado` | El `request` declarado supera al consumo observado por más de `over_provision_factor` (2× por defecto). |
| `subaprovisionado` | El consumo observado supera al `request` declarado (riesgo de throttling de CPU o OOMKill de memoria). |
| `saludable` | Ninguna de las condiciones anteriores. |

`husk sizing report --dry-run` imprime, para cada contenedor con un
veredicto distinto de `saludable`/`sin-datos`, un fragmento de *strategic
merge patch* en YAML con los valores recomendados para
`resources.requests`/`resources.limits` — solo texto local; husk nunca lo
aplica al cluster.
