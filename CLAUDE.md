Quiero que entres en Plan Mode y NO escribas código todavía.

# Contexto del proyecto

Voy a construir un CLI en Go llamado `husk` para clusters de Kubernetes y OpenShift. El proyecto debe ser compatible tanto con ejecución nativa como binario como con ejecución dentro de contenedores Docker/OCI.

El CLI usa el kubeconfig activo del usuario. Debe soportar `--kubeconfig`, la variable `KUBECONFIG` y fallback a in-cluster config. Debe conectarse al API server de Kubernetes/OpenShift y al Prometheus/Thanos Querier integrado de OpenShift, normalmente en `openshift-monitoring`. Debe autenticarse con el bearer token del usuario actual; no debe requerir credenciales separadas.

# Objetivo funcional

`husk` debe generar reportes de:

1. **Sizing real:** comparar `resources.requests` y `resources.limits` de Deployments, StatefulSets y DaemonSets contra el consumo histórico real mediante PromQL (CPU P95; memoria P99 y picos), para detectar sobreaprovisionamiento, subaprovisionamiento y contenedores sin límites.

2. **DR readiness:** validar OADP/Velero, antigüedad de Backup CRs por namespace, estado de BackupStorageLocation, namespaces de aplicación sin backups asociados, antigüedad del snapshot de etcd y soporte CSI snapshot de los StorageClass utilizados.

3. **Capacity planning:** mostrar allocatable versus requests por nodo, enriquecido con uso histórico de Prometheus, headroom y riesgos de concentración de carga.

4. **Score de resiliencia:** calcular una puntuación agregada de 0 a 100 que combine sizing, DR, capacity, PodDisruptionBudgets y topology spread constraints. Debe mostrar el desglose de puntos y qué factores reducen más el score.

5. **Inventario de recursos:** generar un inventario completo de todos los recursos del cluster (Deployments, StatefulSets, DaemonSets, Services, PVCs, ConfigMaps, Secrets -solo nombres-, Nodes, StorageClasses, CRDs), con opción de resumen ejecutivo o detalle extendido (RBAC, NetworkPolicies, PDBs, ResourceQuotas, HPAs, Ingresses/Routes).

Los análisis deben excluir por defecto los namespaces de sistema: `default`, `kube-system`, `kube-public`, `kube-node-lease`, `openshift` y cualquier namespace con prefijo `openshift-`. Esta política debe ser configurable mediante YAML. Excepción: el estado del operador OADP en `openshift-adp` siempre debe evaluarse durante DR readiness aunque ese namespace esté excluido del análisis de aplicaciones.

# Requisitos de conexión a Kubernetes/OpenShift

El CLI debe conectarse directamente a las APIs de Kubernetes/OpenShift usando las librerías oficiales de Go `client-go`, `openshift/client-go`). **No debe ejecutar `oc` o `kubectl` como procesos externos.** El usuario debe tener un kubeconfig válido (típicamente creado mediante `oc login`), pero el binario de `oc` no necesita estar instalado en el PATH.

- Leer el kubeconfig usando `clientcmd.NewNonInteractiveDeferredLoadingClientConfig()`, soportando `--kubeconfig`, la variable `KUBECONFIG` y el archivo por defecto en `~/.kube/config`.

- Soportar fallback a in-cluster config cuando el binario corre dentro de un pod de Kubernetes (usar `rest.InClusterConfig()` si el kubeconfig falla).

- Detectar si el cluster es OpenShift intentando leer el CRD `config.openshift.io/v1` o el proyecto `openshift-apiserver`.

- Para acceder a Prometheus en OpenShift, obtener la ruta de Thanos Querier `thanos-querier` en `openshift-monitoring`) usando el cliente de Routes de OpenShift, y autenticar con el mismo bearer token del usuario.

- Si el kubeconfig no tiene credenciales válidas, mostrar un error claro indicando que el usuario debe ejecutar `oc login` o `kubectl` primero para autenticarse.

# Requisitos Docker y OCI

El proyecto debe incluir:

- Dockerfile multi-stage optimizado para Go; usa una imagen final mínima `distroless/static` o Alpine) y explica cuál eliges y por qué.

- Ejecución con usuario no-root por defecto `USER 65532` o equivalente).

- `ENTRYPOINT` y `CMD` apropiados para ejecutar `husk` directamente con `docker run`.

- Detección automática de ejecución in-cluster para usar in-cluster config dentro de un pod.

- GoReleaser con `dockers_v2`, builds multi-arquitectura para `linux/amd64` y `linux/arm64`, y tags semánticos `latest`, versión completa y major.minor).

- Todas las imágenes deben incluir estas etiquetas OCI:

  - `org.opencontainers.image.title=husk`

  - `org.opencontainers.image.description=CLI de sizing, DR readiness e inventario para Kubernetes/OpenShift`

  - `org.opencontainers.image.version`

  - `org.opencontainers.image.revision`

  - `org.opencontainers.image.created`

  - `org.opencontainers.image.source`

  - `org.opencontainers.image.vendor`

  - `org.opencontainers.image.licenses`

  - `org.opencontainers.image.url`

  - `org.opencontainers.image.documentation`

  - [`org.opencontainers.image.base.name`](http://org.opencontainers.image.base.name)

  - `org.opencontainers.image.base.digest`

# Features requeridas

- `husk version`: muestra versión semántica, commit SHA, fecha de build y versión de Go. Esta información debe coincidir con las etiquetas OCI de la imagen Docker.

- `husk connect health`: valida conectividad al API server, detecta si es OpenShift o Kubernetes vanilla, verifica acceso a Prometheus/Thanos Querier, y valida que el usuario tiene permisos de lectura mínimos. Debe fallar rápido si algo está mal, con mensajes claros de qué falla y cómo remediarlo.

- `husk init rbac`: genera el ClusterRole y ClusterRoleBinding mínimos de solo lectura para ejecutar el CLI mediante un ServiceAccount en CronJobs o CI/CD. No es necesario para uso local con un usuario ya autenticado por `oc login`.

- `husk inventory`: genera un inventario completo de los recursos del cluster (Deployments, StatefulSets, DaemonSets, Services, PVCs, ConfigMaps, Secrets -solo nombres-, Nodes, StorageClasses, CRDs). Soporta:

  - `--namespace` para filtrar a un namespace específico.

  - `--output` (markdown|json|table|excel).

  - `--output-file` para especificar el archivo de salida (requerido para excel).

  - `--extended` para incluir RBAC, NetworkPolicies, PDBs, ResourceQuotas, LimitRanges, HPAs, Ingresses/Routes.

  - Para Excel: múltiples hojas formateadas (una por tipo de recurso), con encabezados en negrita, filtros habilitados, colores condicionales para resaltar riesgos (rojo: sin limits/una réplica; amarillo: sobreaprovisionado; verde: saludable), columnas congeladas y hipervínculos entre hojas relacionadas.

- `husk inventory summary`: versión ejecutiva del inventario, con métricas agregadas y hallazgos de riesgo (namespaces sin quotas, workloads con una réplica, etc.).

- `husk sizing report`: tabla de sizing con recomendaciones.

- `husk capacity nodes`: headroom del cluster y riesgos por nodo.

- `husk dr assess`: semáforo de preparación DR; incluye chequeo de PodDisruptionBudgets faltantes y topology spread constraints en workloads críticos.

- `husk score`: score de resiliencia total y desglose por dimensión.

- `husk report generate --output [json|markdown|table]`: reporte consolidado con portada/metadatos, resumen ejecutivo, hallazgos priorizados, detalle técnico, recomendaciones accionables y apéndice JSON opcional.

- `husk report diff --from <snapshot1.json> --to <snapshot2.json>`: compara snapshots históricos y resalta regresiones y mejoras.

- `--dry-run`: en comandos que sugieren cambios, muestra el manifiesto YAML o patch propuesto y un diff contra el recurso actual; nunca aplica cambios al cluster, nunca abre Pull Requests y nunca provoca efectos externos.

- `husk export grafana-dashboard` (placeholder inicial): debe existir desde la Fase 0 pero puede generar solo un JSON de ejemplo mínimo o un mensaje de 'próximamente'. La implementación completa de dashboards se deja para una fase posterior, una vez el core del CLI esté validado.

- Integración con Alertmanager (fase posterior): correlacionar alertas activas de OOM, throttling y capacidad con hallazgos para distinguir incidentes actuales de riesgos preventivos.

- Historial local: guardar snapshots en `~/.husk/history/` para que `report diff` funcione sin gestión manual de archivos.

- Imágenes Docker/OCI multi-arquitectura listas para GHCR o Docker Hub.

# Flags globales

Todos los comandos deben heredar estos flags persistentes desde `rootCmd`:

- `--kubeconfig`: ruta al archivo kubeconfig (por defecto `~/.kube/config` o `$KUBECONFIG`).

- `--context`: nombre del contexto a usar dentro del kubeconfig.

- `--namespace`, `-n`: namespace por defecto para operaciones que lo requieran.

- `--output`, `-o`: formato de salida `json`, `markdown`, `table`, `excel`). Por defecto `table`.

- `--output-file`: archivo de salida (requerido para `excel`, opcional para otros formatos).

- `--verbose`, `-v`: habilitar logging detallado (nivel debug), incluyendo queries PromQL ejecutadas y tiempos de respuesta.

- `--config`: ruta a un archivo de configuración YAML que sobrescribe valores por defecto.

# Archivo de configuración YAML

El CLI debe soportar un archivo de configuración en `~/.husk/config.yaml` o vía `--config <path>`. Este archivo debe permitir configurar:

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

Los valores por defecto deben estar hardcodeados en el código pero sobrescribibles vía este archivo.

# Reglas de arquitectura

- Separación estricta: `internal/k8sclient` e `internal/prometheus` solo recolectan datos; no generan salida visual.

- `internal/report` consume modelos ya poblados; nunca llama directamente a Kubernetes, Prometheus o Alertmanager.

- `internal/inventory` puede llamar directamente al API server para listar recursos, pero no debe modificar nada.

- Todos los comandos de análisis y reporte son estrictamente de solo lectura. El CLI debe funcionar con un ClusterRole que solo tenga `get`, `list` y `watch` en los recursos que necesita. Nunca debe requerir permisos de `create`, `update`, `patch` o `delete` para los comandos de análisis y reporte.

- `--dry-run` es estrictamente local y no tiene efectos externos.

- Usa `RunE`, nunca `Run`, para los comandos Cobra.

- Cada subcomando Cobra debe exponer `NewCommand()` y registrarse de forma modular; no conviertas `root.go` en un archivo monolítico.

- Los umbrales, la ventana histórica de métricas y los patrones de exclusión de namespaces deben ser configurables mediante YAML, no hardcodeados.

- Logging estructurado: usar `slog` (stdlib de Go 1.21+) o una librería como `zaplogrus` con niveles: error, warn, info, debug. El flag `--verbose` debe habilitar logs de nivel debug.

- Manejo de errores amigable: los errores deben mostrarse de forma clara, con mensajes entendibles y sugerencias de acción. Los stack traces solo deben mostrarse con `--verbose` o la variable de entorno `HUSK_DEBUG=1`.

- Habilitar autocompletado de shell para bash, zsh y fish vía `husk completion <shell>` (nativo de Cobra).

- **Seguridad de datos sensibles:** el comando `husk inventory` NUNCA debe mostrar el contenido de Secrets o ConfigMaps que puedan contener datos sensibles. Solo muestra nombres, labels, tipos y metadatos no sensibles.

# Score de resiliencia

El score de resiliencia (0-100) debe calcularse como un promedio ponderado de sub-scores por dimensión:

- Sizing (30%): 100 si no hay workloads críticos sin límites o con desviación extrema entre requests y uso real; se reduce proporcionalmente según la severidad y cantidad de hallazgos.

- DR (30%): 100 si todos los namespaces críticos tienen backups recientes (&lt; 24h), OADP está healthy y los PVs soportan CSI snapshot; se reduce por cada gap encontrado.

- Capacity (20%): 100 si hay headroom suficiente (&gt; 30% libre) en todos los nodos; se reduce si hay nodos saturados o riesgo de concentración de carga.

- PodDisruptionBudgets (10%): 100 si todos los workloads críticos tienen PDBs configurados correctamente; se reduce por cada workload sin PDB.

- Topology spread constraints (10%): 100 si los workloads críticos tienen topology spread para distribuirse en múltiples nodos/zonas; se reduce por cada workload sin esta protección.

El comando `husk score` debe mostrar el score total y el desglose por dimensión, con enlaces a los hallazgos específicos que reducen el score.

# Lo que necesito en esta etapa de Plan Mode

1. Explora el estado actual del repositorio.

2. Propón una estructura de carpetas completa y explica la responsabilidad de cada paquete.

3. Propón fases pequeñas, verificables y ordenadas. Prioriza: esqueleto/conectividad, cliente Prometheus, filtro transversal de namespaces, inventario (Fase 1), sizing, capacity, DR/score, reportes/diff, Alertmanager/Grafana/historial, pruebas/distribución.

4. Para cada fase, especifica librerías Go, archivos nuevos, dependencias y criterios verificables de finalización. Para la feature de inventario con Excel, sugiere la librería `excelize` y describe cómo estructurar las hojas.

5. Identifica decisiones de arquitectura con alternativas razonables —por ejemplo PDF, modelo `ClusterSnapshot`, integración OpenShift versus Kubernetes genérico, y distroless versus Alpine— y recomienda una opción con sus trade-offs.

6. Incluye el diseño propuesto del Dockerfile multi-stage y de `dockers_v2` en `.goreleaser.yaml` con las etiquetas OCI requeridas.

# Punto de control obligatorio

Cuando termines el plan completo, DETENTE. No escribas código ni archivos todavía. Espera mi aprobación explícita del plan.

Después de mi aprobación, implementa SOLAMENTE la Fase 0: esqueleto del proyecto, Cobra, estructura inicial de carpetas, `go.mod`, `.gitignore`, README inicial, Dockerfile base, `.goreleaser.yaml` base con OCI labels y [`CLAUDE.md`](http://CLAUDE.md) con estas reglas.

Al finalizar la Fase 0, DETENTE y presenta:

- archivos creados;

- comandos ejecutados y resultado;

- decisiones tomadas;

- cualquier desviación respecto del plan;

- los siguientes módulos que consideres listos para trabajo paralelo.

Espera mi confirmación de que el esqueleto es aceptado antes de implementar la Fase 1.

# Autonomía para subagentes después de aprobar el esqueleto

Una vez que yo confirme explícitamente que el esqueleto de la Fase 0 está aceptado, continúa el proyecto de forma iterativa y toma tú la decisión de usar o no subagentes.

Usa subagentes de manera autónoma solo cuando identifiques tareas realmente independientes, con límites de archivos claros y bajo riesgo de conflictos. Antes de iniciar cada subagente:

1. Define para el subagente una meta concreta, archivos/directorios permitidos, criterios de aceptación y pruebas que debe ejecutar.

2. Usa git worktrees o ramas aisladas para trabajo paralelo cuando haya posibilidad de editar archivos comunes.

3. No delegues simultáneamente tareas que modifiquen el mismo contrato público, los mismos modelos compartidos, `cmd/root.go`, `go.mod`, configuración global o los mismos archivos de integración.

4. Mantente en la sesión principal la responsabilidad de decisiones de arquitectura, integración final, cambios de contratos compartidos, resolución de conflictos y revisión de seguridad/RBAC.

5. Prefiere una sola sesión para trabajo secuencial, cambios fundacionales, refactors transversales y la integración del reporte consolidado.

Puedes usar subagentes para ejemplos como estos, si el estado real del repositorio lo permite:

- Motor de inventario aislado en `internal/inventory/`.

- Motor de sizing aislado en `internal/sizing/`.

- Evaluador DR aislado en `internal/dr/`.

- Generador de Excel aislado en `internal/excel/` (puede ser compartido por inventory y reportes).

- Integración Alertmanager aislada en `internal/alertmanager/`.

- Exportador de dashboard en `internal/grafana/` (fase posterior).

- Historial local y comparación de snapshots en `internal/history/` e `internal/diff/`.

- Pruebas unitarias aisladas que no exijan cambiar los contratos de producción.

- Revisión de seguridad/RBAC como subagente de solo lectura.

- Empaquetado/release como subagente dedicado, después de estabilizar las interfaces de build.

Al terminar cada iteración, informa: qué implementaste, si usaste subagentes, qué responsabilidad tuvo cada uno, pruebas ejecutadas, resultados, riesgos pendientes y la siguiente propuesta de trabajo. No continúes con una fase nueva que implique cambios arquitectónicos significativos sin explicar primero el plan breve de esa fase.