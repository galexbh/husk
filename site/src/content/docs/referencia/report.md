---
title: husk report
description: Reporte consolidado, historial local y comparación entre snapshots.
---

```sh
husk report generate [flags]
husk report diff --from <A.json> --to <B.json> [flags]
```

## `husk report generate`

Corre inventario, sizing (si hay Prometheus), capacity, DR y score **en un
solo pase**, y produce un reporte consolidado con:

1. **Portada**: cluster, tipo (OpenShift/vanilla), versión, versión de
   husk, fecha de generación.
2. **Resumen ejecutivo**: score total, conteos de hallazgos por severidad,
   métricas agregadas del inventario.
3. **Hallazgos priorizados**: todos los hallazgos de inventory summary +
   score (que ya agrega sizing/DR/capacity/PDB/topology), deduplicados por
   ID y ordenados por severidad.
4. **Recomendaciones accionables**: una acción sugerida concreta por cada
   hallazgo.
5. **Detalle técnico completo**: las mismas tablas que producirían
   `inventory`, `sizing report`, `capacity nodes` y `dr assess` por
   separado.
6. **Correlación con Alertmanager** (si está disponible): qué hallazgos de
   sizing/capacity tienen una alerta activa relacionada en su namespace —
   distingue un incidente actual de un riesgo preventivo.

Además, **guarda automáticamente** el snapshot completo en
`~/.husk/history/<cluster>/<timestamp>.json` — ver
[Historial y report diff](/husk/guias/historial-y-diff/).

### Flags

| Flag | Descripción |
|---|---|
| `--appendix` | Incluye el snapshot completo (`ClusterSnapshot`) como apéndice JSON al final del reporte. |

Soporta los cuatro formatos, incluido `excel` (hojas de Findings,
Recommendations y, cuando el detalle de inventario está presente,
Deployments/StatefulSets/DaemonSets/Nodes).

```sh
husk report generate --appendix -o json --output-file reporte.json
husk report generate -o excel --output-file reporte.xlsx
```

## `husk report diff`

```sh
husk report diff --from A.json --to B.json
```

| Flag | Requerido | Descripción |
|---|---|---|
| `--from` | sí | Snapshot inicial (JSON). |
| `--to` | sí | Snapshot final (JSON). |

Ver [Historial y report diff](/husk/guias/historial-y-diff/) para el
detalle completo de qué compara y cómo se identifican los hallazgos entre
ejecuciones.
