---
title: Docker y OCI
description: Cómo correr husk en un contenedor y qué imagen usar.
---

## Imagen base

La imagen final se basa en `gcr.io/distroless/static:nonroot`: el binario se
compila con `CGO_ENABLED=0` (no necesita libc), y distroless no incluye
shell ni gestor de paquetes, minimizando la superficie de ataque. Corre como
usuario no-root (**UID 65532**) por defecto.

## Correr husk en un contenedor

Monta tu kubeconfig como volumen de solo lectura:

```sh
docker run --rm \
  -v ~/.kube:/home/nonroot/.kube:ro \
  ghcr.io/galexbh/husk:latest connect health
```

Para escribir archivos de salida (Excel, JSON, el historial en
`~/.husk/history/`), monta también un directorio de trabajo:

```sh
docker run --rm \
  -v ~/.kube:/home/nonroot/.kube:ro \
  -v "$(pwd)/out":/home/nonroot/out \
  -v "$(pwd)/.husk":/home/nonroot/.husk \
  ghcr.io/galexbh/husk:latest \
  report generate --output-file /home/nonroot/out/reporte.json
```

## Dentro de un pod (in-cluster)

Si corres `husk` dentro de un Pod con un ServiceAccount montado (por
ejemplo, un CronJob), no necesitas kubeconfig: husk detecta automáticamente
la configuración in-cluster (`rest.InClusterConfig()`) cuando no encuentra
un kubeconfig explícito. Genera el RBAC mínimo del ServiceAccount con la
[guía de RBAC](/husk/guias/rbac/).

## Etiquetas OCI

Todas las imágenes incluyen las etiquetas `org.opencontainers.image.*`
completas: `title`, `description`, `version`, `revision`, `created`,
`source`, `vendor`, `licenses`, `url`, `documentation`,
`base.name` y `base.digest`. `husk version` siempre coincide exactamente con
`org.opencontainers.image.version`/`.revision` de la imagen que lo contiene.

## SBOM y firma

Cada release incluye:

- **SBOM** (software bill of materials) por artefacto, generado con `syft`.
- **Firma keyless (Sigstore/cosign)** de los checksums y de las imágenes, con
  el token OIDC de GitHub Actions — sin gestionar una clave privada.

Verificar una imagen:

```sh
cosign verify \
  --certificate-identity-regexp=".*" \
  --certificate-oidc-issuer="https://token.actions.githubusercontent.com" \
  ghcr.io/galexbh/husk:v0.1.0
```

## Build local

```sh
make docker   # construye husk:<version> localmente con el Dockerfile del repo
```

El `Dockerfile` del repositorio es multi-stage (compila y empaqueta); el
release usa `Dockerfile.release`, que solo empaqueta el binario ya
compilado por GoReleaser — ver
[`.goreleaser.yaml`](https://github.com/galexbh/husk/blob/main/.goreleaser.yaml).
