---
title: husk init rbac
description: Genera el ClusterRole y ClusterRoleBinding mínimos de solo lectura.
---

```sh
husk init rbac [flags]
```

Genera el `ClusterRole` y `ClusterRoleBinding` de solo lectura
(`get`/`list`/`watch`) necesarios para ejecutar husk mediante un
ServiceAccount — por ejemplo, en un CronJob o pipeline de CI/CD. **No es
necesario** para uso local con un usuario ya autenticado por `oc login` o
`kubectl`.

## Flags

| Flag | Default | Descripción |
|---|---|---|
| `--name` | `husk-reader` | Nombre del ClusterRole/ClusterRoleBinding a generar. |
| `--service-account` | `husk` | Nombre del ServiceAccount que usará el rol. |
| `--service-account-namespace` | `husk` | Namespace del ServiceAccount. |

Además de los [flags globales](/husk/referencia/flags-globales/):
`--output-file` escribe el YAML a un archivo en vez de stdout.

## Ejemplo

```sh
husk init rbac \
  --name husk-reader \
  --service-account husk \
  --service-account-namespace husk \
  --output-file rbac.yaml

kubectl apply -f rbac.yaml
```

Ver la [guía de RBAC](/husk/guias/rbac/) para el detalle de qué recursos
cubre y por qué `cluster-monitoring-view` queda fuera de este comando.
