---
title: Guía rápida
description: De cero a tu primer reporte consolidado en unos minutos.
---

Esta guía asume que ya tienes sesión iniciada en tu cluster (`oc login` o
`kubectl config use-context`) y `husk` [instalado](/husk/empezando/instalacion/).

## 1. Valida la conectividad

```sh
husk connect health
```

`husk` reporta si el API server es alcanzable, si el cluster es OpenShift o
Kubernetes vanilla, si Thanos Querier está accesible, y si tu usuario tiene
los permisos mínimos de lectura que necesita. Si algo falla, el mensaje te
dice exactamente qué remediar — por ejemplo, correr `husk init rbac` o
volver a autenticarte con `oc login`.

## 2. Genera un inventario

```sh
husk inventory summary
```

Un resumen ejecutivo: conteos por tipo de recurso y hallazgos de riesgo
(workloads con una sola réplica, sin `resources.limits`, namespaces sin
`ResourceQuota`).

Para el inventario completo con todas las hojas formateadas en Excel:

```sh
husk inventory --output excel --output-file inventario.xlsx
```

## 3. Analiza sizing y capacity (requiere OpenShift)

```sh
husk sizing report
husk capacity nodes
```

`sizing report` compara los `requests`/`limits` declarados contra el consumo
real observado (CPU P95, memoria P99 y pico) de los últimos 7 días por
defecto. Agrega `--dry-run` para ver el patch YAML sugerido para cada
contenedor — nunca se aplica al cluster.

## 4. Evalúa disaster recovery

```sh
husk dr assess
```

Valida OADP/Velero, antigüedad de backups por namespace, BackupStorageLocation,
snapshot de etcd (si aplica) y soporte CSI snapshot.

## 5. Mira el score de resiliencia

```sh
husk score
```

Un número de 0 a 100 que combina las cinco dimensiones anteriores, con el
desglose de qué está restando más puntos.

## 6. Genera el reporte consolidado

```sh
husk report generate
```

Corre todo lo anterior en un solo pase: portada, resumen ejecutivo, hallazgos
priorizados por severidad, detalle técnico completo y recomendaciones
accionables. Guarda automáticamente un snapshot en `~/.husk/history/` para
que puedas compararlo más adelante:

```sh
husk report diff --from ~/.husk/history/<cluster>/20260101-000000.json \
                  --to   ~/.husk/history/<cluster>/20260201-000000.json
```

:::tip[Formatos de salida]
Todos los comandos anteriores aceptan `-o table|markdown|json` (y `excel`
para `inventory`/`report generate`). Usa `--output-file` para escribir a un
archivo en vez de stdout.
:::

## Siguientes pasos

- [Configura los umbrales](/husk/guias/configuracion/) (`~/.husk/config.yaml`)
  en vez de aceptar los valores por defecto.
- Si vas a correr `husk` desde un CronJob o pipeline de CI/CD (no un usuario
  interactivo), genera el RBAC mínimo con la [guía de RBAC](/husk/guias/rbac/).
- Usa el [buscador de comandos](/husk/referencia/buscador/) para filtrar en
  vivo por flag o palabra clave, en vez de recorrer cada página.
- Revisa la [referencia de comandos](/husk/referencia/connect/) para el
  detalle completo de cada uno.
