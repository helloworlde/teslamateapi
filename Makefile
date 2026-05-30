.PHONY: help build run docs docs-clean tidy fmt vet test lint clean docker-build docker-run install-tools

# Pinned to match the swaggo runtime locked in go.mod
SWAG_VERSION := v1.16.4
SWAG := $(shell go env GOPATH)/bin/swag

API_VERSION  ?= dev
DOCKER_IMAGE ?= teslamateapi
DOCKER_TAG   ?= local

help: ## Show this help
	@awk 'BEGIN {FS = ":.*##"} /^[a-zA-Z_-]+:.*##/ {printf "  \033[36m%-18s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

install-tools: ## Install swag CLI at the pinned version
	@command -v $(SWAG) >/dev/null 2>&1 || go install github.com/swaggo/swag/cmd/swag@$(SWAG_VERSION)

docs: install-tools ## Regenerate OpenAPI spec from swag annotations
	$(SWAG) init --dir cmd/teslamateapi,internal,pkg/dto -g main.go -o docs --outputTypes go,yaml,json --parseDependency --parseInternal

docs-clean: ## Remove generated docs
	rm -f docs/docs.go docs/swagger.json docs/swagger.yaml

build: docs ## Regenerate docs and build the binary into ./bin/teslamateapi
	mkdir -p bin
	CGO_ENABLED=0 go build -ldflags="-w -s -X 'main.apiVersion=$(API_VERSION)'" -o bin/teslamateapi ./cmd/teslamateapi

run: docs ## Regenerate docs and run via dev/run-api.sh
	./dev/run-api.sh

tidy: ## go mod tidy
	go mod tidy

fmt: ## gofmt all sources
	gofmt -s -w cmd internal pkg

vet: ## go vet
	go vet ./...

test: ## go test
	go test ./...

lint: fmt vet ## fmt + vet

clean: docs-clean ## Remove build artefacts and generated docs
	rm -rf bin

docker-build: ## Build the production Docker image
	docker build --build-arg apiVersion=$(API_VERSION) -t $(DOCKER_IMAGE):$(DOCKER_TAG) .

docker-run: docker-build ## Build and run the Docker image on :8080
	docker run --rm -p 8080:8080 $(DOCKER_IMAGE):$(DOCKER_TAG)
