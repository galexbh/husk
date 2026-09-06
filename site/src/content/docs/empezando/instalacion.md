---
title: Instalación
description: Cómo instalar husk como binario, con Go, o como imagen de contenedor.
---

`husk` es un único binario estático (compilado con `CGO_ENABLED=0`), sin
dependencias de sistema. Elige la opción que prefieras.

## Binario de release

Descarga el binario para tu plataforma desde la página de
[Releases](https://github.com/galexbh/husk/releases) de GitHub. Cada release
incluye:

- Binarios para `linux`, `darwin` y `windows` (`amd64`/`arm64`).
- `checksums.txt` y su firma keyless (Sigstore/cosign).
- Un SBOM (software bill of materials) por artefacto.

## Con Go

Si tienes Go 1.26 o superior instalado:

```sh
go install github.com/galexbh/husk@latest
```

## Imagen de contenedor

`husk` también se publica como imagen multi-arquitectura (`linux/amd64`,
`linux/arm64`) en GHCR y Docker Hub:

```sh
docker pull ghcr.io/galexbh/husk:latest
# o
docker pull docker.io/galexbh/husk:latest
```

La imagen corre como usuario no-root (UID 65532) sobre
`gcr.io/distroless/static:nonroot`. Ver la [guía de Docker/OCI](/husk/guias/docker/)
para cómo montar tu kubeconfig al correrla.

## Verificar la instalación

```sh
husk version
```

```
husk v0.1.0
  commit:     a1b2c3d
  build date: 2026-01-15T10:00:00Z
  go version: go1.26.5
```

## Autocompletado de shell

```sh
husk completion bash   # también: zsh, fish
```

Sigue las instrucciones que imprime `husk completion <shell> --help` para
cargarlo en tu shell.

## Requisitos previos

`husk` usa el kubeconfig activo del usuario — el mismo que deja `oc login` o
`kubectl`. No necesita el binario `oc`/`kubectl` instalado, pero sí:

- Un kubeconfig válido con credenciales vigentes (`--kubeconfig`,
  `$KUBECONFIG`, o el archivo por defecto `~/.kube/config`).
- Para sizing/capacity con consumo histórico, acceso al stack de monitoreo de
  OpenShift (el rol `cluster-monitoring-view`).

Corre `husk connect health` para validar todo esto en segundos — ver la
[guía rápida](/husk/empezando/guia-rapida/).
