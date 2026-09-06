---
title: husk dr assess
description: Semáforo de preparación de disaster recovery.
---

```sh
husk dr assess [flags]
```

Evalúa, en un solo pase:

- **OADP/Velero**: si el operador está instalado en `openshift-adp` y si la
  `DataProtectionApplication` está reconciliada (`Reconciled=True`).
  Este chequeo **siempre** se ejecuta, aunque `openshift-adp` coincida con
  `namespaces.exclude_patterns`.
- **BackupStorageLocations**: nombre, provider, fase y si es la default.
- **Cobertura de backups por namespace**: para cada namespace de
  aplicación, busca el `Backup` completado más reciente que lo incluya y
  reporta si no hay ninguno, o si el más reciente supera
  `dr.backup_max_age`.
- **Snapshot de etcd**: solo en OpenShift. No hay una API de solo lectura
  que exponga directamente su antigüedad, así que se infiere del último Job
  exitoso de un CronJob/Job cuyo nombre sugiera que es un backup de etcd
  (patrón `(?i)(etcd.*(backup|snapshot))|((backup|snapshot).*etcd)`). Si no
  se encuentra evidencia, se reporta como "no verificable" — nunca como un
  error — y esto sí resta puntos al score (severidad media), a diferencia
  de en Kubernetes vanilla, donde el chequeo completo se marca "no aplica"
  y no penaliza.
- **Soporte CSI snapshot**: para cada StorageClass efectivamente usada por
  al menos un PVC, si existe un `VolumeSnapshotClass` cuyo `driver`
  coincida con su provisioner.
- **PodDisruptionBudgets y topology spread constraints**: para los
  workloads "críticos" (Deployments/StatefulSets con más de una réplica),
  si tienen un PDB que los cubra y si declaran
  `topologySpreadConstraints`. Los DaemonSets quedan fuera — ya se
  distribuyen por diseño en todos los nodos elegibles.

## Flags

Solo hereda los [flags globales](/husk/referencia/flags-globales/). No
soporta `--output excel` en esta fase.

## Configuración relacionada

`dr.backup_max_age`, `dr.etcd_snapshot_max_age` — ver
[Configuración](/husk/guias/configuracion/).

## Ver también

[`husk score`](/husk/referencia/score/) usa los mismos hallazgos de DR,
PDB y topology spread para calcular tres de sus cinco dimensiones.
