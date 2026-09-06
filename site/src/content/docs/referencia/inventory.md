---
title: husk inventory
description: Inventario completo de recursos del cluster, con export a Excel.
---

```sh
husk inventory [flags]
husk inventory summary [flags]
```

## `husk inventory`

Genera el inventario completo: Deployments, StatefulSets, DaemonSets,
Services, PVCs, ConfigMaps, Secrets (solo nombres y metadatos — **nunca**
contenido), Nodes, StorageClasses, ResourceQuotas y
CustomResourceDefinitions.

| Flag | Descripción |
|---|---|
| `--extended` | Incluye además RBAC (Roles/RoleBindings/ClusterRoles/ClusterRoleBindings), NetworkPolicies, PodDisruptionBudgets, LimitRanges, HorizontalPodAutoscalers e Ingresses/Routes. ResourceQuotas ya se incluye siempre, sin necesitar este flag. |

Soporta los cuatro formatos: `table`, `markdown`, `json` y `excel`.

```sh
husk inventory --output excel --output-file inventario.xlsx
```

El Excel generado tiene una hoja por tipo de recurso, más una hoja
`Summary` con hipervínculos a cada una. Cada hoja de detalle tiene:
encabezados en negrita, autofiltro, columnas congeladas
(`inventory.excel.freeze_columns`) y colores condicionales de riesgo
(`inventory.excel.highlight_risks`) — rojo para sin límites o una sola
réplica, verde para saludable.

:::danger[Seguridad de datos sensibles]
`husk inventory` **nunca** muestra el contenido de Secrets o ConfigMaps.
Solo nombre, namespace, tipo, cantidad de claves y labels — nunca los
valores ni los nombres de las claves.
:::

## `husk inventory summary`

La versión ejecutiva: conteos agregados por tipo de recurso y hallazgos de
riesgo (workloads con una sola réplica, sin `resources.limits`, namespaces
sin `ResourceQuota` — se recolecta siempre, sin necesitar `--extended`).

```sh
husk inventory summary --extended
```

No soporta `--output excel` (usa `husk inventory --output excel` para el
detalle completo).

## Filtrar por namespace

```sh
husk inventory --namespace mi-app
```

Cuando pasas `--namespace` explícitamente, ese namespace se analiza
**aunque coincida** con un patrón de `namespaces.exclude_patterns` — una
petición explícita tiene prioridad sobre la política de exclusión por
defecto.
