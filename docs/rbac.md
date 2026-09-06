# Permisos RBAC de husk

Todos los comandos de análisis y reporte de husk son estrictamente de solo
lectura: el CLI nunca requiere los verbos `create`, `update`, `patch` o
`delete` sobre ningún recurso del cluster.

## Uso local (`oc login` / `kubectl`)

Si ya tienes sesión iniciada con un usuario que tiene acceso de lectura al
cluster (y, para sizing/capacity, el rol `cluster-monitoring-view` en
OpenShift), no necesitas ningún manifiesto adicional: husk usa el mismo
bearer token de tu kubeconfig activo.

## Uso vía ServiceAccount (CronJobs, CI/CD)

Ejecuta:

```sh
husk init rbac --name husk-reader \
  --service-account husk \
  --service-account-namespace husk
```

Esto genera un `ClusterRole` y un `ClusterRoleBinding` con únicamente
`get`, `list` y `watch` sobre los recursos que husk necesita leer
(workloads, storage, RBAC, políticas de red, PDBs, HPAs, routes, CRs de
OADP/Velero, CustomResourceDefinitions, etc.). Aplícalo con:

```sh
husk init rbac --output-file rbac.yaml
kubectl apply -f rbac.yaml
```

Para acceder a Prometheus/Thanos en OpenShift, el usuario o ServiceAccount
también necesita el ClusterRole `cluster-monitoring-view` (fuera del alcance
de `husk init rbac`, ya que es un rol propio del stack de monitoreo de
OpenShift, no de husk).

`husk connect health` valida estos permisos en segundos y reporta con
precisión qué falta.
