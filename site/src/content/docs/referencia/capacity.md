---
title: husk capacity nodes
description: Allocatable vs requests por nodo, headroom y riesgos de concentración de carga.
---

```sh
husk capacity nodes [flags]
```

Para cada nodo del cluster: allocatable de CPU/memoria/pods vs la suma de
`requests` de los pods efectivamente programados en él (equivalente a la
tabla "Allocated resources" de `kubectl describe node`, sin necesitar el
binario `kubectl`), el % de headroom libre, y — cuando Thanos Querier está
disponible — el consumo histórico real (promedio y pico).

También detecta **riesgos de concentración de carga**: Deployments o
StatefulSets con más de una réplica cuyos pods, en la práctica, corren
todos en el mismo nodo o en una sola zona de disponibilidad.

:::tip[Funciona sin Prometheus]
A diferencia de `sizing report`, este comando **no requiere** OpenShift ni
Thanos Querier: el headroom (allocatable vs requests) se calcula igual en
Kubernetes vanilla. El consumo histórico es solo un enriquecimiento
opcional.
:::

## Nodos excluidos del headroom

Los nodos con un taint `NoSchedule` o `NoExecute` se excluyen del cálculo de
headroom asignable del cluster y se reportan aparte, en su propia sección
("Nodes excluidos del headroom").

## Flags

Solo hereda los [flags globales](/husk/referencia/flags-globales/). No
soporta `--output excel` en esta fase.

## Configuración relacionada

`capacity.headroom_threshold_percent`, `capacity.lookback` — ver
[Configuración](/husk/guias/configuracion/).
