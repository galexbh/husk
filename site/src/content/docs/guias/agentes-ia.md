---
title: Guía para agentes de IA
description: Cómo un agente de IA puede correr husk e interpretar su salida de forma segura.
---

`husk` no expone un servidor MCP ni ninguna API propia para agentes: un
agente de IA lo consume exactamente igual que un humano — invocando el
binario o la imagen Docker y parseando su salida. Esta página explica cómo
hacerlo de forma segura y cómo interpretar el resultado.

:::note[Si sos un agente de IA]
Leé esta página antes de ejecutar comandos de `husk` o de sacar
conclusiones del JSON de salida.
:::

## Qué es seguro invocar

Todos los comandos de análisis y reporte de `husk` son estrictamente de
solo lectura: el `ClusterRole` que genera `husk init rbac` solo tiene
`get`, `list` y `watch`, nunca `create`/`update`/`patch`/`delete`. Podés
correr cualquiera de ellos (`connect health`, `inventory`, `sizing
report`, `capacity nodes`, `dr assess`, `score`, `report generate`,
`report diff`, `export grafana-dashboard`) sin riesgo de modificar el
cluster.

:::caution[Excepción: `husk init rbac`]
Este comando **genera** un YAML de `ClusterRole`/`ClusterRoleBinding` — no
lo aplica. Aplicar ese YAML (`kubectl apply -f rbac.yaml`) sí modifica el
cluster y es una acción aparte que requiere autorización humana explícita;
no la ejecutes por tu cuenta como si fuera parte de un análisis de solo
lectura.
:::

`--dry-run` (en `husk sizing report`) es igualmente inocuo: solo imprime un
patch YAML sugerido en texto, nunca lo aplica, nunca abre Pull Requests.

## Cómo invocarlo

`husk` usa el kubeconfig activo del usuario — nunca credenciales propias.
Vía Docker (el patrón recomendado si no tenés el binario compilado):

```sh
docker run --rm \
  -v ~/.kube/config:/tmp/kubeconfig:ro \
  ghcr.io/galexbh/husk:latest \
  --kubeconfig /tmp/kubeconfig score --output json
```

Montá el kubeconfig en modo `:ro` (solo lectura). Usá `--output json` para
obtener una salida estructurada en vez de la tabla pensada para humanos.
Los comandos que aceptan `--namespace`/`-n` (`inventory`, `sizing report`)
lo heredan como flag global; los comandos cluster-wide lo ignoran a
propósito. Ver [Docker y OCI](/husk/guias/docker/) para más detalle sobre
la imagen.

## Esquema del JSON

El bloque central, reutilizado por `score`, `dr assess`, `sizing report`
(vía `report generate`), `capacity nodes` e `inventory summary`, es
`Finding` (ver [Modelo de datos y seguridad](/husk/arquitectura/modelo-y-seguridad/)):

```json
{
  "id": "sizing-under-provisioned:shop/deployment/api",
  "severity": "red",
  "category": "sizing-under-provisioned",
  "message": "Deployment shop/api (api): subaprovisionado, riesgo de throttling/OOM",
  "namespace": "shop",
  "resource": "Deployment/api",
  "explanation": "El consumo observado (CPU P95 o memoria P99) supera el request declarado: el contenedor corre en riesgo de throttling de CPU u OOM kill.",
  "recommendation": "Aumenta requests/limits: el consumo observado supera lo declarado (riesgo de throttling/OOM). Ver `husk sizing report --dry-run`."
}
```

- `severity`: `"red"` (crítico), `"yellow"` (advertencia), `"green"`
  (saludable) o `""` (sin datos suficientes).
- `id`: estable — la misma combinación de categoría/namespace/recurso
  produce siempre el mismo `id`, útil para deduplicar entre comandos o
  para `husk report diff`.
- `explanation`/`recommendation`: vienen de un catálogo único
  (`internal/model/finding_catalog.go`) indexado por `category`. Si ves una
  categoría con `explanation` igual a *"sin explicación documentada para
  esta categoría de hallazgo"*, es una categoría nueva a la que todavía no
  se le agregó guía — no es un error tuyo, repórtalo.

`husk score` además devuelve un desglose por dimensión — ver
[husk score](/husk/referencia/score/) para el detalle de cada una:

```json
{
  "total": 62.4,
  "breakdown": [
    { "name": "sizing", "score": 40.0, "weight": 0.3, "effectiveWeight": 0.3, "available": true, "findingIds": ["..."] },
    { "name": "dr", "score": 55.0, "weight": 0.3, "effectiveWeight": 0.3, "available": true, "findingIds": ["..."] }
  ],
  "findings": ["... Finding[] con explanation/recommendation ..."]
}
```

Si una dimensión no está disponible (`available: false`, típicamente
`sizing` sin Prometheus), su peso se redistribuye entre las demás — no
cuenta como 0. `reason` explica por qué.

`husk report generate` devuelve, además, un resumen ejecutivo
(`executiveSummary`), la lista de hallazgos ya priorizados y deduplicados
(`prioritizedFindings`) y una lista de acciones sugeridas
(`recommendations`), en el mismo orden.

## Cómo interpretar el score

El score de resiliencia (0-100) es **relativo y direccional**, no un
certificado absoluto: sirve para (a) comparar el mismo cluster en el
tiempo, vía `husk report diff` entre dos snapshots, y (b) priorizar qué
dimensión atacar primero dentro de una misma ejecución (mirá primero los
`Finding` con `severity: "red"`, luego la cantidad de hallazgos por
dimensión — no solo el número total).

:::caution[No inventes bandas de interpretación]
`husk` **no** define bandas oficiales (por ejemplo, "80 a 100 es
saludable"); no las inventes ni las presentes como si vinieran del propio
CLI. Si necesitás una banda de este tipo para reportar hacia negocio, es
una decisión que debe tomar y aprobar el dueño del producto, no asumirse
en la salida de una herramienta de solo lectura.
:::

## Ver también

- [Configuración (config.yaml)](/husk/guias/configuracion/)
- [Modelo de datos y seguridad](/husk/arquitectura/modelo-y-seguridad/)
- `CLAUDE.md` en la raíz del repositorio — convenciones de arquitectura y
  trabajo, si además vas a modificar el código de `husk`.
