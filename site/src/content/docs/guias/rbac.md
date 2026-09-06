---
title: RBAC y permisos
description: Qué permisos necesita husk y cómo generarlos para un ServiceAccount.
---

Todos los comandos de análisis y reporte de `husk` son estrictamente de solo
lectura: el CLI **nunca** requiere los verbos `create`, `update`, `patch` o
`delete` sobre ningún recurso del cluster.

## Uso local (`oc login` / `kubectl`)

Si ya tienes sesión iniciada con un usuario que tiene acceso de lectura al
cluster, no necesitas ningún manifiesto adicional: `husk` usa el mismo
bearer token de tu kubeconfig activo. Para `sizing report`, `capacity nodes`
(consumo histórico) y la correlación de alertas en `report generate`,
también necesitas el rol `cluster-monitoring-view` en OpenShift.

## Uso vía ServiceAccount (CronJobs, CI/CD)

Cuando `husk` no lo ejecuta un usuario interactivo sino un ServiceAccount
(por ejemplo, en un CronJob que genera un reporte periódico), genera el
`ClusterRole`/`ClusterRoleBinding` mínimos con:

```sh
husk init rbac \
  --name husk-reader \
  --service-account husk \
  --service-account-namespace husk \
  --output-file rbac.yaml

kubectl apply -f rbac.yaml
```

Esto genera únicamente `get`, `list` y `watch` sobre los recursos que husk
necesita leer: workloads (`apps`), storage (PVCs, StorageClasses,
VolumeSnapshotClasses), RBAC, políticas de red, PodDisruptionBudgets, HPAs,
routes de OpenShift, CustomResourceDefinitions, y las CRs de OADP/Velero.

:::note
`husk init rbac` **no** otorga `cluster-monitoring-view`: ese es un rol
propio del stack de monitoreo de OpenShift, fuera del alcance de husk.
Agrégalo aparte si el ServiceAccount también correrá `sizing report` o
`capacity nodes`.
:::

## Validar los permisos

```sh
husk connect health
```

Corre en segundos y reporta con precisión qué falta — por ejemplo:

```
[FALTA]  permiso de lectura: deployments (todos los namespaces) (forbidden)
```

Si ves un `[FALTA]`, el mensaje de error de `husk connect health` te indica
exactamente aplicar el ClusterRole generado por `husk init rbac`.

## Manifiesto de referencia

Si prefieres aplicar el YAML directamente sin correr el CLI, el repositorio
incluye una copia estática con los valores por defecto en
[`deploy/rbac/`](https://github.com/galexbh/husk/tree/main/deploy/rbac).
