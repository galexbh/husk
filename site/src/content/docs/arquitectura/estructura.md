---
title: Estructura del proyecto
description: Responsabilidad de cada paquete de husk y las reglas de separación entre ellos.
---

```
husk/
├── main.go                # thin: llama a cli.Execute()
├── internal/
│   ├── cli/                # capa Cobra: parseo de flags y wiring. Cero lógica de negocio
│   ├── model/              # ClusterSnapshot y todos los tipos. Sin I/O
│   ├── config/             # defaults hardcodeados + merge de config.yaml
│   ├── nsfilter/           # filtro transversal de exclusión de namespaces
│   ├── logging/            # slog, --verbose
│   ├── huskerr/             # errores accionables (mensaje + sugerencia)
│   │
│   ├── k8sclient/           # recolección: kubeconfig, clientsets, detección OpenShift
│   ├── promclient/          # recolección: Thanos Querier, queries PromQL
│   ├── alertmanager/        # recolección: alertas activas + correlación
│   │
│   ├── inventory/           # analizador: inventario de recursos
│   ├── sizing/              # analizador: requests/limits vs consumo real
│   ├── capacity/            # analizador: headroom y concentración de carga
│   ├── dr/                  # analizador: OADP/Velero/etcd/CSI/PDB/topology
│   ├── score/               # combina sizing/dr/capacity/pdb/topology en un score
│   │
│   ├── report/              # renderizado: table/markdown/json de todos los reportes
│   ├── excel/                # renderizado: workbooks .xlsx (compartido por inventory y report)
│   ├── diff/                 # compara dos ClusterSnapshot
│   ├── history/               # persiste/lee snapshots en ~/.husk/history/
│   ├── grafana/               # genera el dashboard exportable de Grafana
│   │
│   ├── rbac/                  # plantillas embebidas para `husk init rbac`
│   └── archtest/               # test que hace cumplir la regla de separación de abajo
```

## Regla de separación: recolectar vs renderizar

Este es el límite arquitectónico más importante del proyecto, y el único que
se hace cumplir automáticamente en CI:

> `internal/report` e `internal/excel` consumen modelos ya poblados
> (`internal/model`); **nunca** llaman directamente a Kubernetes, Prometheus
> o Alertmanager.

En la práctica, esto significa que ninguno de esos dos paquetes importa
`internal/k8sclient`, `internal/promclient` ni `internal/alertmanager`. Todo
lo que necesitan renderizar ya les llega como un `model.Inventory`,
`model.SizingReport`, `model.CapacityReport`, `model.DRReadiness` o
`model.Score` — poblados por los paquetes analizadores/recolectores.

`internal/archtest` verifica esto parseando los imports de cada archivo
`.go` de esos dos paquetes (sin necesitar cargar el paquete completo) y
falla si encuentra una violación. Corre como parte de `go test ./...` y por
lo tanto en CI en cada push/PR.

## Por qué esta separación importa

- **Testeable sin cluster**: `internal/report`/`internal/excel` se prueban
  con datos sintéticos, sin fakes de client-go ni servidores HTTP simulados.
- **Reutilizable**: el mismo renderer sirve para datos que vinieron de un
  cluster real (`husk sizing report`) o de un snapshot cargado del historial
  (`husk report diff`).
- **Auditable**: cualquiera puede confirmar, con un test automatizado en vez
  de una revisión manual, que un cambio no mezcló recolección con
  presentación.

## El modelo compartido: `ClusterSnapshot`

`internal/model.ClusterSnapshot` es el contrato único que atraviesa todo el
proyecto: lo que produce cada analizador, lo que se guarda en
`~/.husk/history/`, lo que compara `report diff`, y lo que consume
`report generate` para ensamblar el reporte final. Cada comando llena solo
la sección que le corresponde (`Inventory`, `Sizing`, `Capacity`, `DR`,
`Score`); el resto queda `nil`. Ver
[Modelo de datos y seguridad](/husk/arquitectura/modelo-y-seguridad/) para
el detalle de sus campos.
