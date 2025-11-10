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
	protoc \
		--go_out=./ \
		--go_opt=module=github.com/spechtlabs/tka \
	    --go-grpc_out=./ \
		--go-grpc_opt=module=github.com/spechtlabs/tka \
		./pkg/cluster/messages.proto

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
all: generate pre-commit test
