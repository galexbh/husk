---
title: husk connect health
description: Valida conectividad, tipo de cluster y permisos mínimos de lectura.
---

```sh
husk connect health
```

Valida, en orden y fallando rápido en el primer problema:

1. Que el API server sea alcanzable y devuelve su versión.
2. Si el cluster es OpenShift o Kubernetes vanilla (detectado vía el CRD
   `config.openshift.io/v1`, no por el binario `oc`).
3. Si Thanos Querier está accesible (solo en OpenShift) — un chequeo básico
   con el mismo bearer token del kubeconfig activo.
4. Los permisos mínimos de lectura sobre los recursos que husk necesita
   (`nodes`, `namespaces`, `pods`, `deployments`, `storageclasses`).

## Salida de ejemplo

```
husk connect health
====================
[OK]     API server alcanzable — versión v1.29.4 (linux/amd64)
[OK]     cluster detectado: OpenShift
[OK]     Thanos Querier alcanzable en https://thanos-querier-openshift-monitoring.apps.example.com
[OK]     permiso de lectura: nodes
[OK]     permiso de lectura: namespaces
[OK]     permiso de lectura: pods (todos los namespaces)
[OK]     permiso de lectura: deployments (todos los namespaces)
[OK]     permiso de lectura: storageclasses

husk connect health: todo en orden.
```

Si falta un permiso:

```
[FALTA]  permiso de lectura: deployments (todos los namespaces) (deployments.apps is forbidden: User "..." cannot list resource "deployments" in API group "apps" at the cluster scope)
```

```
Error: faltan permisos de lectura sobre 1 tipo(s) de recurso
Sugerencia: ejecuta `husk init rbac` para generar un ClusterRole/ClusterRoleBinding de solo lectura y aplícalo para tu usuario o ServiceAccount
```

## Sin kubeconfig válido

```
Error: no se pudo cargar un kubeconfig válido: invalid configuration: no configuration has been provided
Sugerencia: ejecuta `oc login` o `kubectl config use-context` para autenticarte, o especifica --kubeconfig/$KUBECONFIG. Si husk corre dentro de un pod, verifica que el ServiceAccount tenga un token montado.
```

## Flags

Solo hereda los [flags globales](/husk/referencia/flags-globales/); no
tiene flags propios.
