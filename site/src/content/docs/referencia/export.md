---
title: husk export grafana-dashboard
description: Genera un dashboard de Grafana con las métricas de sizing/capacity.
---

```sh
husk export grafana-dashboard [flags]
```

Genera un dashboard de Grafana (JSON exportable, con `__inputs`) con
paneles de CPU/memoria por namespace y por nodo, más alertas activas — las
mismas familias de métricas PromQL que usan `husk sizing report` y
`husk capacity nodes` (ver [Métricas PromQL](/husk/guias/metricas/)),
generalizadas por namespace/nodo en vez de por workload específico.

Este comando **no necesita un kubeconfig** ni acceso a un cluster: el
dashboard es estático, no una captura del estado actual.

## Importar en Grafana

Al importar el JSON, Grafana pide elegir el datasource de Prometheus/Thanos
(variable `${DS_PROMETHEUS}`) — el dashboard no queda atado a un UID de
datasource específico de un cluster, así que el mismo archivo sirve en
cualquier instalación de Grafana apuntada al Prometheus/Thanos correcto.

## Flags

Solo hereda los [flags globales](/husk/referencia/flags-globales/).
`--output-file` escribe el JSON a un archivo en vez de stdout.

```sh
husk export grafana-dashboard --output-file dashboard.json
```

## Paneles incluidos

- CPU por namespace
- Memoria working-set por namespace
- CPU por nodo (máquina completa)
- Memoria por nodo (máquina completa)
- Alertas activas (`ALERTS{alertstate="firing"}`)
