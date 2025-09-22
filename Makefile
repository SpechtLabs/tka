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

.PHONY: controller-gen
controller-gen: $(CONTROLLER_GEN) ## Download controller-gen locally if necessary.
$(CONTROLLER_GEN): $(LOCALBIN)
	$(call go-install-tool,$(CONTROLLER_GEN),sigs.k8s.io/controller-tools/cmd/controller-gen,$(CONTROLLER_TOOLS_VERSION))


.PHONY: generate
generate: controller-gen swag ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	$(CONTROLLER_GEN) object:headerFile="./hack/boilerplate.go.txt" paths="./..."

.PHONY: manifests
manifests: controller-gen swag
	$(CONTROLLER_GEN) rbac:roleName=tka-role crd paths="./..." output:crd:artifacts:config=config/crd/bases

.PHONY: swag
swag:
	swag init --dir ./pkg/api --generalInfo server.go --output ./pkg/swagger --parseDependency --parseDepth 3

.PHONY: test
test: lint
	go test -race ./... -coverprofile=coverage.out
	go tool cover -html=coverage.out

.PHONY: lint
lint:
	go vet ./...
	golangci-lint run ./...
