---
title: Historial y report diff
description: Cómo funciona el historial local de snapshots y cómo comparar dos ejecuciones.
---

Cada vez que corres `husk report generate`, husk guarda automáticamente el
snapshot completo del análisis (`ClusterSnapshot`, el mismo modelo que
alimenta todos los reportes) en:

```
~/.husk/history/<cluster>/<timestamp>.json
```

`<cluster>` se deriva de la URL del API server (por ejemplo,
`https---api-demo-example-com-6443`), y `<timestamp>` es la fecha/hora UTC de
la ejecución. No necesitas gestionar estos archivos a mano: cada
`report generate` también poda automáticamente los snapshots más antiguos
según `history.retain_count` (30 por defecto — ver
[configuración](/husk/guias/configuracion/)).

## Comparar dos ejecuciones

```sh
husk report diff \
  --from ~/.husk/history/<cluster>/20260101-000000.json \
  --to   ~/.husk/history/<cluster>/20260201-000000.json
```

El resultado muestra:

- El **score total** antes/después y su delta, más el desglose por
  dimensión (solo para las dimensiones que estaban disponibles en ambos
  snapshots).
- **Hallazgos resueltos** (mejoras): estaban en `--from` y ya no están en
  `--to`.
- **Hallazgos nuevos** (regresiones): no estaban en `--from` y aparecen en
  `--to`.
- Cuántos hallazgos **persisten sin cambios**.

Si ambos snapshots son idénticos, `report diff` simplemente reporta
"Sin cambios" — no imprime tablas vacías.

:::note[No hace falta que sean de `~/.husk/history/`]
`report diff --from`/`--to` acepta cualquier archivo JSON generado por
`husk report generate` (o `husk score`/`dr assess`/etc. con `-o json`, si
tiene el campo `schemaVersion`), no solo los del historial local. Por
ejemplo, puedes versionar snapshots en tu propio repositorio y compararlos
en CI.
:::

## Cómo se identifican los hallazgos

Cada hallazgo tiene un `id` estable, derivado determinísticamente de su
categoría, namespace y recurso — no de un contador ni de un timestamp. Esto
es lo que permite que `report diff` reconozca "el mismo" hallazgo entre dos
ejecuciones aunque el orden interno cambie.

## Formatos de salida

```sh
husk report diff --from A.json --to B.json -o table     # por defecto
husk report diff --from A.json --to B.json -o markdown
husk report diff --from A.json --to B.json -o json
```

## Versión de esquema

Cada snapshot incluye un campo `schemaVersion`. Si intentas comparar un
snapshot generado por una versión de husk con un esquema incompatible,
`report diff` falla con un mensaje claro en vez de comparar datos que no
coinciden estructuralmente — regenera el snapshot con la versión actual de
husk.
