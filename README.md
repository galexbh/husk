# husk

**Documentación completa: [galexbh.github.io/husk](https://galexbh.github.io/husk/)**

`husk` es un CLI de solo lectura para clusters Kubernetes y OpenShift. Usa el
kubeconfig activo del usuario (el mismo que deja `oc login` o `kubectl`) para
generar reportes de:

- **Sizing real** — `resources.requests`/`limits` contra el consumo histórico
  real (CPU P95, memoria P99 y picos) vía PromQL.
- **DR readiness** — OADP/Velero, antigüedad de backups, BackupStorageLocation,
  namespaces sin backup, snapshot de etcd, soporte CSI snapshot.
- **Capacity planning** — allocatable vs requests por nodo, uso histórico,
  headroom y riesgos de concentración de carga.
- **Score de resiliencia** — 0–100, combinando sizing, DR, capacity, PDBs y
  topology spread constraints.
- **Inventario** — Deployments, StatefulSets, DaemonSets, Services, PVCs,
  ConfigMaps, Secrets (solo nombres), Nodes, StorageClasses, CRDs, con salida
  Excel formateada.

`husk` **nunca ejecuta `oc` o `kubectl` como procesos externos** — se conecta
directamente a las APIs de Kubernetes/OpenShift con `client-go` y
`openshift/client-go`, autenticando con el bearer token del usuario actual.
Todos los comandos de análisis y reporte son estrictamente de solo lectura.

> Estado actual: todas las fases del plan de construcción están implementadas
> (ver [CLAUDE.md](CLAUDE.md)) — sizing, DR readiness, capacity, score,
> inventario, reportes consolidados/diff, historial local, correlación con
> Alertmanager y exportación de dashboard de Grafana.

## Instalación

```sh
go install github.com/galexbh/husk@latest
```

O descarga un binario desde [Releases](https://github.com/galexbh/husk/releases),
o usa la imagen de contenedor:

```sh
docker run --rm -v ~/.kube:/home/nonroot/.kube:ro ghcr.io/galexbh/husk:latest connect health
```

## Uso

```sh
husk connect health              # valida conectividad al cluster y a Thanos/Alertmanager
husk init rbac                    # genera el ClusterRole/ClusterRoleBinding de solo lectura
husk inventory                    # inventario completo del cluster
husk inventory summary            # versión ejecutiva del inventario, con hallazgos de riesgo
husk sizing report                # sizing real vs consumo histórico (--dry-run: patch sugerido)
husk capacity nodes               # headroom por nodo y riesgos de concentración de carga
husk dr assess                    # preparación de disaster recovery
husk score                        # score de resiliencia (0-100), desglosado por dimensión
husk report generate              # reporte consolidado; guarda snapshot en ~/.husk/history/
husk report diff --from A --to B  # compara dos snapshots: regresiones y mejoras
husk export grafana-dashboard      # dashboard de Grafana con las métricas de sizing/capacity
husk completion <bash|zsh|fish>    # autocompletado de shell
```

Todos los comandos de análisis y reporte son estrictamente de solo lectura; el
CLI nunca ejecuta `create`/`update`/`patch`/`delete` contra el cluster.

Flags globales (heredados por todos los subcomandos):

| Flag | Descripción |
|---|---|
| `--kubeconfig` | ruta al kubeconfig (por defecto `$KUBECONFIG` o `~/.kube/config`) |
| `--context` | contexto del kubeconfig a usar |
| `-n, --namespace` | namespace por defecto |
| `-o, --output` | `json`\|`markdown`\|`table`\|`excel` (por defecto `table`) |
| `--output-file` | archivo de salida (requerido para `excel`) |
| `-v, --verbose` | logging de nivel debug (incluye queries PromQL y latencias) |
| `--config` | ruta a un YAML de configuración (por defecto `~/.husk/config.yaml`) |

Ver [`examples/config.yaml`](examples/config.yaml) para el esquema completo
de configuración.

## Documentación

La guía de uso completa (instalación, guía rápida, configuración, RBAC,
referencia de cada comando y arquitectura) vive en
**[galexbh.github.io/husk](https://galexbh.github.io/husk/)**, construida con
[Starlight](https://starlight.astro.build/es/) — fuente en
[`site/`](site/).

Versión en Markdown plano (para leer directo en GitHub, sin el sitio):

- [`docs/configuration.md`](docs/configuration.md) — esquema completo de `config.yaml`.
- [`docs/rbac.md`](docs/rbac.md) — permisos mínimos y `husk init rbac`.
- [`docs/metrics.md`](docs/metrics.md) — catálogo de queries PromQL de sizing/capacity.

## Desarrollo

```sh
go build ./...
go vet ./...
go test ./...
golangci-lint run ./...
```

Un test de arquitectura (`internal/archtest`) falla en CI si `internal/report`
o `internal/excel` importan directamente un paquete recolector
(`internal/k8sclient`, `internal/promclient`, `internal/alertmanager`),
haciendo cumplir la separación entre recolección y renderizado.

```sh
make build   # binario local con versión/commit/fecha embebidos
make docker  # imagen local
```

## Docker / OCI

La imagen final se basa en `gcr.io/distroless/static:nonroot`: el binario se
compila con `CGO_ENABLED=0` (no necesita libc), y distroless no incluye shell
ni gestor de paquetes, minimizando la superficie de ataque. Corre como
usuario no-root (UID 65532) por defecto.

Los releases se publican multi-arquitectura (`linux/amd64`, `linux/arm64`) en
`ghcr.io/galexbh/husk` y `docker.io/galexbh/husk`, con las etiquetas OCI
completas (`org.opencontainers.image.*`). Cada release incluye además:

- **SBOM** (software bill of materials) por artefacto, generado con `syft`.
- **Firma keyless (Sigstore/cosign)** de los checksums y de las imágenes,
  usando el token OIDC de GitHub Actions — sin gestionar una clave privada.
  Verificar: `cosign verify --certificate-identity-regexp=".*" --certificate-oidc-issuer="https://token.actions.githubusercontent.com" ghcr.io/galexbh/husk:vX.Y.Z`.

Ver [`.goreleaser.yaml`](.goreleaser.yaml) y
[`.github/workflows/release.yml`](.github/workflows/release.yml).

## Licencia

Apache-2.0. Ver [LICENSE](LICENSE).
