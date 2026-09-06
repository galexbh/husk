---
title: husk score
description: Score de resiliencia agregado (0-100), con desglose por dimensión.
---

```sh
husk score [flags]
```

Calcula un score de 0 a 100 combinando cinco dimensiones, cada una
ponderada según `score.weights` (ver [Configuración](/husk/guias/configuracion/)):

| Dimensión | Peso por defecto | De dónde sale |
|---|---|---|
| Sizing | 30% | `husk sizing report` — requiere Prometheus. |
| DR | 30% | `husk dr assess` (OADP, backups, etcd, CSI snapshot). |
| Capacity | 20% | `husk capacity nodes` (headroom, concentración de carga). |
| PDB | 10% | `husk dr assess` (workloads críticos sin PodDisruptionBudget). |
| Topology spread | 10% | `husk dr assess` (workloads críticos sin `topologySpreadConstraints`). |

## Redistribución de pesos

Si una dimensión no se puede calcular — típicamente **Sizing** cuando no
hay Prometheus disponible — no se cuenta como 0: su peso se redistribuye
proporcionalmente entre las dimensiones que sí están disponibles, y la
tabla de salida lo marca explícitamente:

```
│ sizing    │ -     │ 30%              │ 0%            │ no (el cluster no es OpenShift) │
```

## Cómo se calcula cada dimensión

- **Sizing**: pondera cada contenedor evaluable según su veredicto — verde
  cuenta completo, amarillo (sobreaprovisionado) parcial, rojo (sin
  límites/subaprovisionado) resta con más peso.
- **DR**: 100 menos una deducción fija por OADP no instalado o no healthy;
  una deducción **proporcional** al porcentaje de namespaces de aplicación
  sin backup (el mismo hallazgo pesa igual sin importar el tamaño del
  cluster); deducciones fijas (con tope) por namespaces con backup vencido,
  snapshot de etcd no verificable o vencido, y StorageClasses sin soporte
  CSI.
- **Capacity**: 100 menos una deducción **proporcional** al porcentaje de
  nodos en riesgo — un nodo ALTO (headroom bajo el umbral en CPU **y**
  memoria a la vez, o no-Ready/unschedulable) cuenta entero, uno MEDIO (un
  solo eje bajo el umbral) cuenta la mitad — más una deducción fija (con
  tope) por cada riesgo de concentración de carga.
- **PDB** y **Topology spread**: proporcional — `100 × cubiertos / total`
  workloads críticos — para que el resultado no dependa del tamaño del
  cluster.

El cálculo es una función pura y determinista: la misma entrada siempre
produce el mismo resultado.

## Flags

Solo hereda los [flags globales](/husk/referencia/flags-globales/). No
soporta `--output excel` en esta fase.
