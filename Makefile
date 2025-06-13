TAG ?= latest

.PHONY: init
init:
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install github.com/golang/mock/mockgen@latest

.PHONY:mock
mock:
	mockgen -source=./internal/store/codebase/codebase_store.go -destination=./internal/store/codebase/mocks/codebase_store_mock.go -package=mocks
	mockgen -source=./internal/store/codebase/wrapper/minio_wrapper.go -destination=./internal/store/codebase/wrapper/mocks/minio_client_mock.go -package=mocks

.PHONY:test
test:
	go test ./internal/...

.PHONY:build
build:
	go mod tidy
	go build -ldflags="-s -w" -o ./bin/main ./cmd/main.go

.PHONY:docker
docker:
	docker build -t zgsm/codebase-indexer:$(TAG) .