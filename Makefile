MODULE      := github.com/galexbh/husk
VERSION     ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT      ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
BUILD_DATE  ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

LDFLAGS := -s -w \
	-X $(MODULE)/internal/version.Version=$(VERSION) \
	-X $(MODULE)/internal/version.Commit=$(COMMIT) \
	-X $(MODULE)/internal/version.BuildDate=$(BUILD_DATE)

.PHONY: build
build:
	CGO_ENABLED=0 go build -trimpath -ldflags "$(LDFLAGS)" -o husk .

.PHONY: test
test:
	go test ./...

.PHONY: vet
vet:
	go vet ./...

.PHONY: lint
lint:
	golangci-lint run ./...

.PHONY: fmt
fmt:
	gofmt -l -w .

.PHONY: docker
docker:
	docker build \
		--build-arg VERSION=$(VERSION) \
		--build-arg COMMIT=$(COMMIT) \
		--build-arg BUILD_DATE=$(BUILD_DATE) \
		-t husk:$(VERSION) .

.PHONY: clean
clean:
	rm -f husk husk.exe
	rm -rf dist/

.PHONY: ci
ci: vet test build
