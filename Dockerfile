# syntax=docker/dockerfile:1
#
# Build local/reproducible de husk. En release, GoReleaser NO usa este
# Dockerfile: inyecta el binario ya compilado usando Dockerfile.release.
# Se elige gcr.io/distroless/static:nonroot como imagen final: el binario se
# compila con CGO_ENABLED=0 (no necesita libc), distroless no trae shell ni
# gestor de paquetes (menor superficie de ataque y menos CVEs que mantener)
# y ya incluye ca-certificates, un usuario no-root (UID 65532) y zoneinfo.

FROM --platform=$BUILDPLATFORM golang:1.26-alpine AS builder
WORKDIR /src

RUN apk add --no-cache git ca-certificates

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download

COPY . .

ARG TARGETOS
ARG TARGETARCH
ARG VERSION=dev
ARG COMMIT=none
ARG BUILD_DATE=unknown

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -trimpath \
      -ldflags="-s -w \
        -X github.com/galexbh/husk/internal/version.Version=${VERSION} \
        -X github.com/galexbh/husk/internal/version.Commit=${COMMIT} \
        -X github.com/galexbh/husk/internal/version.BuildDate=${BUILD_DATE}" \
      -o /out/husk .

FROM gcr.io/distroless/static:nonroot
COPY --from=builder /out/husk /usr/local/bin/husk

USER 65532:65532
WORKDIR /home/nonroot

ENTRYPOINT ["/usr/local/bin/husk"]
CMD ["--help"]

LABEL org.opencontainers.image.title="husk" \
      org.opencontainers.image.description="CLI de sizing, DR readiness e inventario para Kubernetes/OpenShift" \
      org.opencontainers.image.source="https://github.com/galexbh/husk" \
      org.opencontainers.image.url="https://github.com/galexbh/husk" \
      org.opencontainers.image.documentation="https://github.com/galexbh/husk/blob/main/README.md" \
      org.opencontainers.image.vendor="galexbh" \
      org.opencontainers.image.licenses="Apache-2.0" \
      org.opencontainers.image.base.name="gcr.io/distroless/static:nonroot"
