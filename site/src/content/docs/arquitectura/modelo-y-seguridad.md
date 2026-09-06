---
title: Modelo de datos y seguridad
description: ClusterSnapshot, hallazgos con ID estable, y las garantías de solo lectura y datos sensibles.
---

## `ClusterSnapshot`

```go
type ClusterSnapshot struct {
	SchemaVersion string
	Meta          Meta

	Inventory        *Inventory
	InventorySummary *InventorySummary
	Sizing           *SizingReport
	Capacity         *CapacityReport
	DR               *DRReadiness
	Score            *Score
}
```

`SchemaVersion` se incrementa cada vez que el modelo cambia de forma
incompatible, para que `report diff` y el historial local puedan detectar
snapshots de versiones distintas de husk en vez de comparar datos que no
coinciden estructuralmente.

## Hallazgos (`Finding`) con ID estable

Todo lo que husk considera un riesgo — desde un workload con una sola
réplica hasta un namespace sin backup — se modela como un `Finding`:

```go
type Finding struct {
	ID             string    // determinista: category + namespace + resource
	Severity       RiskLevel // red | yellow | green | ""
	Category       string
	Message        string
	Namespace      string
	Resource       string
	Explanation    string // qué significa el hallazgo y por qué importa
	Recommendation string // acción concreta sugerida
}
```

El `ID` se deriva determinísticamente de la categoría, el namespace y el
recurso — nunca de un contador ni de un timestamp. Esto es lo que permite
que:

- `husk score` enlace cada punto restado a un hallazgo concreto.
- `husk report diff` reconozca "el mismo" hallazgo entre dos snapshots,
  aunque el orden interno de la lista cambie entre ejecuciones.

`Explanation` y `Recommendation` vienen de un catálogo único
(`internal/model/finding_catalog.go`), indexado por `Category`, y se
adjuntan a todo `Finding` en el momento en que se construye — no solo en
`report generate`. El objetivo es que cualquier `--output json` de
cualquier comando (`score`, `dr assess`, `sizing report`, `capacity nodes`,
`inventory summary`) sea interpretable sin contexto adicional, incluido por
un agente de IA. Ver [Guía para agentes de IA](/husk/guias/agentes-ia/).

## Garantías de solo lectura

Todos los comandos de análisis y reporte son estrictamente de solo lectura:

- El ClusterRole que genera `husk init rbac` solo tiene `get`, `list` y
  `watch` — nunca `create`, `update`, `patch` ni `delete`.
- `--dry-run` (en `husk sizing report`) es estrictamente local: muestra un
  patch YAML propuesto y nunca lo aplica al cluster, ni abre Pull Requests,
  ni tiene ningún otro efecto externo.
- husk nunca ejecuta `oc` ni `kubectl` como procesos externos — se conecta
  directamente a las APIs con `client-go`/`openshift-client-go`.

## Datos sensibles

`husk inventory` **nunca** muestra el contenido de Secrets o ConfigMaps.
`model.SecretSummary` y `model.ConfigMapSummary` solo tienen nombre,
namespace, tipo, cantidad de claves y labels — no existe ningún campo en el
modelo capaz de portar `Data`/`BinaryData`, así que esta garantía se cumple
en tiempo de compilación, no solo por convención en el código que arma el
reporte.
