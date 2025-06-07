
.PHONY:mock
mock:
	mockgen -source=./internal/store/codebase/codebase_store.go -destination=./internal/store/codebase/mocks/codebase_store_mock.go -package=mocks
	mockgen -source=./internal/store/codebase/wrapper/minio_wrapper.go -destination=./internal/store/codebase/mocks/minio_client_mock.go -package=mocks

.PHONY:test
test:
	go test ./internal/...