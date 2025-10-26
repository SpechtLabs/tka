##@ Dependencies

## Location to install dependencies to
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

## Tool Binaries
KUBECTL ?= kubectl
KIND ?= kind
KUSTOMIZE ?= $(LOCALBIN)/kustomize
CONTROLLER_GEN ?= $(LOCALBIN)/controller-gen
TKALINT ?= $(LOCALBIN)/tkalint

## Build

.PHONY: build
build: generate
	goreleaser build --clean --snapshot --config .goreleaser.pr.yaml

.PHONY: build-release
build-release: generate
	goreleaser build --clean --config .goreleaser.yaml

## Code Generations

.PHONY: controller-gen
controller-gen: swag ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	controller-gen object:headerFile="./hack/boilerplate.go.txt" paths="./..."

.PHONY: protoc
protoc:
	protoc --go_out=./ ./pkg/cluster/messages.proto

.PHONY: manifests
manifests:
	controller-gen rbac:roleName=tka-role crd paths="./..." output:crd:artifacts:config=config/crd/bases

.PHONY: swag
swag:
	swag init --dir ./pkg/service/ --generalInfo docs.go --output ./pkg/swagger --parseDependency --parseDepth 3

.PHONY: generate
generate: controller-gen swag protoc manifests

## Testing

.PHONY: test
test: lint
	go test -race ./... -coverprofile=coverage.out
	go tool cover -html=coverage.out

.PHONY: lint
lint:
	go vet ./...
	golangci-lint run ./...

.PHONY: pre-commit
pre-commit:
	pre-commit run --all-files

## All

.PHONY: all
all: generate manifests pre-commit test

##@ Custom Analyzers (TKA-specific linting)

.PHONY: build-analyzers
build-analyzers: $(LOCALBIN) ## Build the custom TKA analyzers
	cd tools/analyzers && go build -o ../../$(LOCALBIN)/tkalint ./cmd/tkalint

.PHONY: tkalint
tkalint: build-analyzers ## Run TKA-specific linters on the codebase
	$(TKALINT) ./...

.PHONY: tkalint-fix
tkalint-fix: build-analyzers ## Run TKA linters and show detailed output
	$(TKALINT) -json ./... | jq .

.PHONY: lint-all
lint-all: lint tkalint ## Run both standard and TKA-specific linters

##@ Architecture Checks

.PHONY: arch
arch: ## Run arch-go architecture checks
	arch-go

.PHONY: arch-verbose
arch-verbose: ## Run arch-go with verbose output
	arch-go -v

.PHONY: install-arch-go
install-arch-go: ## Install arch-go tool
	go install github.com/fdaines/arch-go@latest

##@ Analyzer Development

.PHONY: test-analyzers
test-analyzers: ## Run tests for custom analyzers
	cd tools/analyzers && go test -v ./...

.PHONY: analyzer-coverage
analyzer-coverage: ## Run analyzer tests with coverage
	cd tools/analyzers && go test -coverprofile=coverage.out ./... && go tool cover -html=coverage.out
