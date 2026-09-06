---
title: husk sizing report
description: Sizing real de workloads contra su consumo histórico, con recomendaciones.
---

```sh
husk sizing report [flags]
```

Compara los `resources.requests`/`limits` declarados de cada contenedor
(en Deployments, StatefulSets y DaemonSets) contra su consumo histórico real
observado en Prometheus/Thanos Querier — CPU P95, memoria P99 y pico.

:::caution[Requiere OpenShift]
Este comando necesita Thanos Querier, disponible en el stack de monitoreo de
OpenShift. En Kubernetes vanilla falla explícitamente con un mensaje claro
en vez de intentarlo.
:::

## Flags

| Flag | Descripción |
|---|---|
| `--dry-run` | Además de la tabla, imprime un fragmento de *strategic merge patch* en YAML por cada contenedor con un veredicto distinto de `saludable`/`sin-datos`, con los valores recomendados. **Nunca** aplica cambios al cluster. |

No soporta `--output excel` en esta fase.

## Veredictos

`sin-datos`, `sin-limites`, `sobreaprovisionado`, `subaprovisionado`,
`saludable` — ver el detalle de cada condición y las queries PromQL exactas
en [Métricas PromQL](/husk/guias/metricas/).

## Ejemplo

```sh
husk sizing report --dry-run
```

```
Sizing report — lookback 7d, CPU P95, memoria P99

┌─────────────────────────────────────────────────────────┐
│ Deployment shop/api                                     │
├──────────┬─────────────┬──────────────┬─────────────────┤
│ ...      │ CPU req/lim │ CPU observada│ Veredicto       │
├──────────┼─────────────┼──────────────┼─────────────────┤
│ api      │ 1 / 2       │ 100m         │ SOBREAPROVISIONADO │
└──────────┴─────────────┴──────────────┴─────────────────┘

--dry-run: patches sugeridos (solo texto local; husk nunca los aplica al cluster)

# Deployment/api en namespace shop, contenedor "api" (sobreaprovisionado)
spec:
  template:
    spec:
      containers:
        - name: api
          resources:
            requests:
              cpu: 100m
              memory: 96Mi
            limits:
              cpu: 300m
              memory: 96Mi
```

## Configuración relacionada

`sizing.cpu_percentile`, `sizing.memory_percentile`, `sizing.lookback`,
`sizing.cpu_limit_multiplier`, `sizing.memory_buffer_percent`,
`sizing.over_provision_factor` — ver [Configuración](/husk/guias/configuracion/).
