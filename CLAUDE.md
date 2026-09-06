# husk

CLI en Go, de solo lectura, para clusters Kubernetes y OpenShift: sizing real
contra consumo histórico, DR readiness, capacity planning, score de
resiliencia agregado e inventario completo. Compatible con ejecución nativa
(binario) y dentro de contenedores Docker/OCI.

**Estado: todas las fases del plan original están implementadas y en
producción** (Fase 0 a Fase 7 — ver "Historial de fases" al final). Este
documento ya no es un brief de planificación: es la referencia de
arquitectura y las reglas que gobiernan el trabajo futuro sobre este
repositorio. Si vas a agregar una feature o corregir algo, estas son las
convenciones a seguir; no hace falta re-planificar desde cero.

## Notas del proyecto

- Nombre del binario: `husk` (en minúsculas). Módulo Go:
  `github.com/galexbh/husk`.
- Repositorio: `github.com/galexbh/husk`, público. Licencia: Apache-2.0.
- Imágenes de contenedor: `ghcr.io/galexbh/husk` y `docker.io/galexbh/husk`
  (multi-arquitectura, `linux/amd64`/`linux/arm64`).
- Documentación de usuario (Starlight): `site/` — desplegada en
  **https://galexbh.github.io/husk/** vía
  `.github/workflows/docs.yml` en cada push a `main` que toque `site/**`.
  Es un proyecto Node.js independiente del módulo Go; no comparte
  dependencias con `go.mod`.

## El CLI usa el kubeconfig activo del usuario

Soporta `--kubeconfig`, la variable `KUBECONFIG` y fallback a in-cluster
config. Se conecta al API server de Kubernetes/OpenShift y al
Prometheus/Thanos Querier integrado de OpenShift (normalmente en
`openshift-monitoring`), autenticando con el bearer token del usuario
actual — nunca requiere credenciales separadas.

# Objetivo funcional

`husk` genera reportes de:

1. **Sizing real:** compara `resources.requests` y `resources.limits` de
   Deployments, StatefulSets y DaemonSets contra el consumo histórico real
   vía PromQL (CPU P95; memoria P99 y picos), para detectar
   sobreaprovisionamiento, subaprovisionamiento y contenedores sin límites.
   (`husk sizing report`, `internal/sizing`).

2. **DR readiness:** OADP/Velero, antigüedad de Backup CRs por namespace,
   estado de BackupStorageLocation, namespaces de aplicación sin backups
   asociados, antigüedad del snapshot de etcd y soporte CSI snapshot de los
   StorageClass utilizados. (`husk dr assess`, `internal/dr`).

3. **Capacity planning:** allocatable versus requests por nodo, enriquecido
   con uso histórico de Prometheus (opcional), headroom y riesgos de
   concentración de carga. (`husk capacity nodes`, `internal/capacity`).

4. **Score de resiliencia:** puntuación agregada de 0 a 100 que combina
   sizing, DR, capacity, PodDisruptionBudgets y topology spread
   constraints, con desglose de qué factores reducen más el score.
   (`husk score`, `internal/score` — ver fórmula abajo).

5. **Inventario de recursos:** inventario completo del cluster
   (Deployments, StatefulSets, DaemonSets, Services, PVCs, ConfigMaps,
   Secrets -solo nombres-, Nodes, StorageClasses, CRDs), con resumen
   ejecutivo o detalle extendido (RBAC, NetworkPolicies, PDBs,
   ResourceQuotas, HPAs, Ingresses/Routes). (`husk inventory`,
   `internal/inventory`).

6. **Reporte consolidado e historial:** `husk report generate` corre las
   cinco dimensiones anteriores en un solo pase, guarda un snapshot en
   `~/.husk/history/` (`internal/history`) y `husk report diff` compara dos
   snapshots (`internal/diff`).

Los análisis excluyen por defecto los namespaces de sistema: `default`,
`kube-system`, `kube-public`, `kube-node-lease`, `openshift` y cualquier
namespace con prefijo `openshift-` (`internal/nsfilter`, configurable por
YAML). **Excepción:** el estado del operador OADP en `openshift-adp`
siempre se evalúa durante DR readiness aunque ese namespace esté excluido
del análisis de aplicaciones.

# Conexión a Kubernetes/OpenShift

El CLI se conecta directamente a las APIs de Kubernetes/OpenShift con las
librerías oficiales `client-go` y `openshift/client-go` (`internal/k8sclient`).
**Nunca ejecuta `oc` o `kubectl` como procesos externos.** El usuario debe
tener un kubeconfig válido (típicamente vía `oc login`), pero el binario
`oc` no necesita estar instalado en el PATH.

- Kubeconfig vía `clientcmd.NewNonInteractiveDeferredLoadingClientConfig()`,
  con `--kubeconfig`, `$KUBECONFIG` y el archivo por defecto
  `~/.kube/config`.
- Fallback a in-cluster config (`rest.InClusterConfig()`) cuando el
  kubeconfig no está disponible — para correr dentro de un pod.
- Detección de OpenShift vía el CRD `config.openshift.io/v1`
  (`ClusterVersion`), nunca inspeccionando el binario `oc`.
- Prometheus/Alertmanager en OpenShift: resuelve la route de
  `thanos-querier`/`alertmanager-main` en `openshift-monitoring`
  (`internal/k8sclient/thanos.go`) y autentica con el mismo bearer token del
  usuario (`internal/promclient`, `internal/alertmanager`).
- **TLS hacia Thanos/Alertmanager:** `promclient.DiscoverTransport`
  (`internal/promclient/discovery.go`) es el único punto que arma el
  transporte HTTP para estas rutas — intenta validar contra la CA real del
  router de OpenShift (ConfigMap
  `openshift-config-managed/default-ingress-cert`) y solo degrada a
  `InsecureSkipVerify` si no puede resolverla, advirtiéndolo con `Warn`
  (visible sin `--verbose`), nunca en silencio. **Todo código que hable con
  Thanos/Alertmanager con el bearer token del usuario debe reusar esta
  función** — no armar un `http.Transport` ad-hoc con `InsecureSkipVerify`
  hardcodeado (ya pasó una vez en `husk connect health`, que tenía su propio
  transporte inseguro incondicional; corregido para reusar
  `DiscoverTransport` igual que `sizing`/`alertmanager`/`prom`).
- Si el kubeconfig no tiene credenciales válidas, el error indica
  claramente que hace falta `oc login`/`kubectl` (`internal/huskerr`).

# Docker y OCI

- `Dockerfile` (build local) y `Dockerfile.release` (solo empaqueta el
  binario ya compilado por GoReleaser) sobre `gcr.io/distroless/static:nonroot`:
  el binario se compila con `CGO_ENABLED=0`, y distroless minimiza la
  superficie de ataque frente a Alpine (sin shell ni gestor de paquetes).
- Corre como usuario no-root por defecto (`USER 65532:65532`).
- Detección automática de ejecución in-cluster para usar in-cluster config
  dentro de un pod.
- `.goreleaser.yaml` con `dockers_v2`, builds multi-arquitectura
  (`linux/amd64`, `linux/arm64`) y tags `latest`, versión completa y
  `major.minor`.
- Cada release además incluye **SBOM** (`sboms`, vía `syft`) y **firma
  keyless (Sigstore/cosign)** de checksums e imágenes (`signs`,
  `docker_signs`), usando el token OIDC de GitHub Actions — ver
  `.github/workflows/release.yml` (necesita `id-token: write`).
- Todas las imágenes incluyen las etiquetas OCI: `title`, `description`,
  `version`, `revision`, `created`, `source`, `vendor`, `licenses`, `url`,
  `documentation`, `base.name`, `base.digest`. `husk version` debe coincidir
  exactamente con `.version`/`.revision`.

# Comandos implementados

- `husk version` — versión, commit, fecha de build y versión de Go;
  coincide con las etiquetas OCI de la imagen.
- `husk connect health` — valida API server, tipo de cluster (OpenShift vs
  vanilla), Thanos Querier y permisos mínimos de lectura. Falla rápido con
  mensajes claros.
- `husk init rbac [--name --service-account --service-account-namespace]` —
  genera el ClusterRole/ClusterRoleBinding de solo lectura para un
  ServiceAccount (CronJobs, CI/CD). No hace falta para uso local ya
  autenticado.
- `husk inventory [--extended]` / `husk inventory summary [--extended]` —
  inventario completo o resumen ejecutivo con hallazgos de riesgo. Soporta
  `--output table|markdown|json|excel` (Excel: hojas por tipo de recurso,
  encabezados en negrita, autofiltro, colores condicionales de riesgo,
  columnas congeladas, hipervínculos entre hojas — `internal/excel`).
- `husk sizing report [--dry-run]` — tabla de sizing con recomendaciones;
  `--dry-run` imprime el patch YAML sugerido por contenedor, nunca aplica
  cambios. Requiere OpenShift (Thanos Querier).
- `husk capacity nodes` — headroom por nodo y riesgos de concentración de
  carga; funciona sin Prometheus (el consumo histórico es un
  enriquecimiento opcional).
- `husk dr assess` — semáforo de preparación DR, incluidos PDBs faltantes y
  topology spread constraints en workloads críticos.
- `husk score` — score de resiliencia total y desglose por dimensión, con
  redistribución de pesos si una dimensión no está disponible (ver fórmula
  abajo).
- `husk report generate [--appendix]` — reporte consolidado (portada,
  resumen ejecutivo, hallazgos priorizados, detalle técnico,
  recomendaciones, apéndice JSON opcional con `--appendix`, correlación con
  Alertmanager si está disponible); guarda el snapshot en
  `~/.husk/history/`. Soporta `--output` incluido `excel`.
- `husk report diff --from <snapshot1.json> --to <snapshot2.json>` —
  compara dos snapshots: score, hallazgos resueltos/introducidos/persistentes.
- `husk export grafana-dashboard` — dashboard de Grafana exportable (JSON
  con `__inputs`) con las mismas métricas PromQL que sizing/capacity; no
  requiere cluster.
- `--dry-run` (en `sizing report`) es estrictamente local: muestra el patch
  YAML propuesto; nunca aplica cambios al cluster, nunca abre Pull
  Requests, nunca tiene efectos externos.
- Alertmanager: correlaciona alertas activas de OOM/throttling/capacidad
  con hallazgos de sizing/capacity dentro de `report generate`
  (`internal/alertmanager`), distinguiendo incidente actual de riesgo
  preventivo.
- Autocompletado de shell nativo de Cobra: `husk completion bash|zsh|fish`.

# Flags globales

Todos los comandos heredan estos flags persistentes desde `rootCmd`
(`internal/cli/root.go`):

- `--kubeconfig`: ruta al kubeconfig (por defecto `~/.kube/config` o
  `$KUBECONFIG`).
- `--context`: contexto del kubeconfig a usar.
- `--namespace`, `-n`: namespace por defecto (comandos que lo aceptan:
  `inventory`, `sizing report`; los cluster-wide lo ignoran a propósito).
- `--output`, `-o`: `table` (default) | `markdown` | `json` | `excel`.
- `--output-file`: archivo de salida (requerido para `excel`, opcional para
  el resto).
- `--verbose`, `-v`: logging de nivel debug, incluidas las queries PromQL y
  sus tiempos de respuesta.
- `--config`: ruta a un archivo de configuración YAML.

# Archivo de configuración YAML

`~/.husk/config.yaml` o `--config <path>` (`internal/config`). Los defaults
están hardcodeados en `internal/config/defaults.go` y son sobrescribibles
declarando solo las claves que se quieran cambiar:

```yaml
namespaces:
  exclude_patterns:
    - "^kube-.*$"
    - "^openshift.*$"
    - "^default$"

sizing:
  cpu_percentile: 0.95
  memory_percentile: 0.99
  lookback: "7d"
  cpu_limit_multiplier: 3.0
  memory_buffer_percent: 0.2
  over_provision_factor: 2.0

capacity:
  headroom_threshold_percent: 30.0
  lookback: "7d"

dr:
  backup_max_age: "24h"
  etcd_snapshot_max_age: "7d"

score:
  weights:
    sizing: 0.3
    dr: 0.3
    capacity: 0.2
    pdb: 0.1
    topology: 0.1

inventory:
  include_extended: false
  excel:
    highlight_risks: true
    freeze_columns: 2

history:
  retain_count: 30
```

Si agregas una clave de configuración nueva, actualiza en el mismo cambio:
este archivo, `internal/config/{config,defaults}.go`, `examples/config.yaml`,
`docs/configuration.md` y `site/src/content/docs/guias/configuracion.md`.
No hardcodees umbrales de negocio en el código — ya pasó dos veces
(`capacity.headroom_threshold_percent`, `sizing.over_provision_factor`) y
la corrección fue moverlos aquí.

# Reglas de arquitectura

- **Separación estricta recolección/renderizado:** `internal/k8sclient`,
  `internal/promclient` y `internal/alertmanager` solo recolectan datos; no
  generan salida visual. `internal/report` e `internal/excel` consumen
  modelos ya poblados (`internal/model`); **nunca** llaman directamente a
  Kubernetes, Prometheus o Alertmanager. Esta regla la hace cumplir
  automáticamente `internal/archtest` (parsea los imports de
  `internal/report`/`internal/excel` y falla si aparece un import
  prohibido) — corre en `go test ./...` y por lo tanto en CI. No la rompas;
  si un renderer necesita un dato nuevo, agrégalo al modelo y pobláalo en
  el analizador correspondiente, no en el renderer.
- Los analizadores (`internal/inventory`, `internal/sizing`,
  `internal/capacity`, `internal/dr`) pueden llamar directamente a las APIs
  recolectoras para obtener datos, pero nunca modifican nada.
- Todos los comandos de análisis y reporte son estrictamente de solo
  lectura. El ClusterRole que genera `husk init rbac` solo tiene `get`,
  `list` y `watch`. Nunca requieras `create`/`update`/`patch`/`delete` para
  comandos de análisis/reporte.
- `--dry-run` es estrictamente local y no tiene efectos externos.
- Usa `RunE`, nunca `Run`, para los comandos Cobra.
- Cada subcomando expone `newXxxCommand()` y se registra de forma modular
  desde `internal/cli/root.go`; no lo conviertas en un archivo monolítico.
- Los umbrales, la ventana histórica y los patrones de exclusión de
  namespaces son configurables por YAML, no hardcodeados (ver arriba).
- Logging estructurado con `slog` (stdlib); niveles error/warn/info/debug;
  `--verbose` habilita debug.
- Errores amigables vía `internal/huskerr`: mensaje + sugerencia siempre;
  causa técnica y stack trace solo con `--verbose` o `HUSK_DEBUG=1`.
- Autocompletado de shell nativo de Cobra (bash/zsh/fish).
- **Seguridad de datos sensibles:** `husk inventory` nunca muestra el
  contenido de Secrets o ConfigMaps. `model.SecretSummary`/`ConfigMapSummary`
  solo tienen nombre, namespace, tipo, cantidad de claves y labels — no
  existe ningún campo capaz de portar `Data`/`BinaryData`.
- **Inyección de fórmulas en Excel (CWE-1236):** los valores de fila en los
  reportes `.xlsx` vienen de nombres/labels/hosts leídos en vivo del
  cluster — texto que cualquier actor con permiso para crear un recurso en
  un namespace de aplicación controla. `excel.Workbook.AddSheet`
  (`internal/excel/workbook.go`) es el único punto que escribe celdas de
  datos y sanea cada valor con `sanitizeCellValue` (antepone `'` si empieza
  con `= + - @` o tab/CR) antes de `SetCellValue`. Cualquier renderer nuevo
  que escriba celdas debe pasar por `Workbook.AddSheet`, no llamar a
  `excelize` directamente para volcar datos de fila.
- **Lint:** el repo debe pasar `golangci-lint run ./...` con 0 issues (ver
  `.golangci.yml`). Al fijar la versión del binario/action en CI, verifica
  que sea igual o más nueva que la versión de Go declarada en `go.mod` — un
  `golangci-lint` compilado con un Go más viejo se niega a correr (ya pasó
  una vez en `ci.yml`, fijar una versión v2.x explícita lo resuelve).
- `site/` (documentación Starlight) es un proyecto Node.js aparte: no
  agregues sus dependencias a `go.mod` ni mezcles su build con `go build`.
  Si cambias un comando/flag, actualiza también
  `site/src/data/commands.ts` (alimenta el buscador interactivo en
  `/referencia/buscador/`) y la página de referencia correspondiente bajo
  `site/src/content/docs/referencia/`.

# Score de resiliencia

Promedio ponderado de cinco sub-scores (`internal/score`):

- **Sizing (30%):** 100 menos deducciones por contenedor evaluable según
  su veredicto (sin límites, subaprovisionado, sobreaprovisionado). Si
  ningún contenedor tiene consumo histórico observable (sin Prometheus),
  la dimensión se marca no disponible y su peso se redistribuye entre las
  demás en vez de contar como 0.
- **DR (30%):** 100 menos deducciones fijas (con tope) por OADP no
  instalado/no healthy, namespaces sin backup o con backup vencido,
  snapshot de etcd no verificable o vencido, StorageClasses sin soporte CSI
  snapshot.
- **Capacity (20%):** 100 menos deducciones fijas por nodo saturado
  (headroom bajo `capacity.headroom_threshold_percent`, default 30%) y por
  riesgo de concentración de carga.
- **PodDisruptionBudgets (10%)** y **Topology spread (10%):**
  proporcionales (`100 × workloads críticos cubiertos / total`), para que
  el resultado no dependa del tamaño del cluster.

El cálculo es una función pura y determinista. `husk score` muestra el
total y el desglose por dimensión, con los `Finding.ID` (estables,
derivados de categoría+namespace+recurso) que explican cada deducción —
los mismos que usa `report diff` para reconocer un hallazgo entre dos
snapshots.

# Cómo trabajar en este repo de aquí en adelante

Antes de dar por terminado cualquier cambio en el CLI:

1. `go build ./... && go vet ./... && go test ./... && gofmt -l .` — todo
   limpio.
2. `golangci-lint run ./...` — 0 issues.
3. Si tocaste `internal/report` o `internal/excel`, `internal/archtest`
   debe seguir pasando (ya corre dentro de `go test ./...`).
4. Si agregaste/cambiaste un comando o flag: actualiza `site/src/data/commands.ts`,
   la página de referencia en `site/src/content/docs/referencia/`, y corré
   `npm run build` dentro de `site/` para confirmar que el sitio sigue
   generando sin errores.
5. Si agregaste una clave de configuración: actualiza `CLAUDE.md`,
   `internal/config`, `examples/config.yaml`, `docs/configuration.md` y
   la guía de configuración del sitio.
6. Prefiere probar contra un cluster real cuando esté disponible; si no,
   usa fakes de `client-go` (`k8s.io/client-go/kubernetes/fake`,
   `k8s.io/client-go/dynamic/fake`) o un servidor HTTP simulado para
   Prometheus/Alertmanager — es el patrón ya usado en
   `internal/{inventory,capacity,dr,sizing}/*_test.go`.

## Subagentes

Usa subagentes de forma autónoma cuando identifiques tareas realmente
independientes, con límites de archivos claros y bajo riesgo de conflictos
— por ejemplo, una feature aislada en un paquete que nadie más esté
tocando, o una tarea de solo lectura (auditoría, investigación). Antes de
lanzar uno:

1. Define una meta concreta, los archivos/directorios permitidos, criterios
   de aceptación y qué pruebas debe correr.
2. Usa git worktrees o ramas aisladas si hay riesgo de editar los mismos
   archivos.
3. No delegues en paralelo dos tareas que toquen el mismo contrato público,
   los mismos modelos compartidos (`internal/model`), `internal/cli/root.go`,
   `go.mod`, la configuración global o `site/src/data/commands.ts` a la
   vez.
4. Mantén en la sesión principal: decisiones de arquitectura, cambios de
   contratos compartidos, resolución de conflictos, y revisión de
   seguridad/RBAC.
5. Prefiere una sola sesión para refactors transversales y cambios
   fundacionales.

Al terminar cualquier cambio no trivial, informa: qué se implementó, si se
usaron subagentes y con qué responsabilidad, qué pruebas se corrieron y su
resultado, riesgos pendientes, y la siguiente propuesta de trabajo. No
inicies un cambio con impacto arquitectónico significativo sin explicar
primero el plan breve.

# Historial de fases (referencia)

El proyecto se construyó en fases incrementales, cada una verificada antes
de continuar:

- **Fase 0** — esqueleto, Cobra, `husk version`/`connect health`/`init rbac`,
  Dockerfile, `.goreleaser.yaml` base.
- **Fase 1** — inventario completo + Excel (`internal/inventory`,
  `internal/excel`).
- **Fase 2** — cliente Prometheus (`internal/promclient`) y
  `sizing report`.
- **Fase 3** — `capacity nodes` (`internal/capacity`).
- **Fase 4** — `dr assess` y `score` (`internal/dr`, `internal/score`).
- **Fase 5** — `report generate`/`diff`, historial local
  (`internal/report`, `internal/history`, `internal/diff`).
- **Fase 6** — integración Alertmanager y `export grafana-dashboard`
  completo (`internal/alertmanager`, `internal/grafana`).
- **Fase 7** — endurecimiento: `internal/archtest`, `golangci-lint` limpio,
  cobertura ampliada, release firmado con cosign + SBOM.
- **Post-Fase 7** — documentación de usuario en Starlight (`site/`),
  desplegada en GitHub Pages, con buscador de comandos interactivo.
