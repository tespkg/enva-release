SHELL := /bin/bash -o pipefail
BIN_DIR=${PWD}/.bin
PB_DIR=${PWD}/pb

.PHONY: help
help:
	@echo "Usage: make <TARGET>"
	@echo ""
	@echo "Available targets are:"
	@echo ""
	@echo "    generate-mock               Generate mock code for interfaces"
	@echo ""
	@echo "    run-fmt                     Run gofmt/goimports for the code"
	@echo "    run-lint                    Run golangci-lint for the code static lint"
	@echo "    run-test                    Run coverage test"
	@echo ""
	@echo "    vendor                      Create/Update vendor packages"
	@echo "    build                       Build the binary"
	@echo ""

.PHONY: get-protoc
get-protoc:
	@scripts/get_protoc.sh

.PHONY: generate-mock
generate-mock: get-protoc
	@rm -f ./pkg/store/store_mock.go
	@.bin/mockgen --source ./pkg/store/store.go --package store --destination ./pkg/store/store_mock.go

.PHONY: run-fmt
run-fmt:
	@scripts/run_gofmt.sh

.PHONY: run-lint 
run-lint:
	@scripts/run_golangci.sh

.PHONY: run-test
run-test:
	@go test -race -mod vendor ./... -coverprofile .testCoverage.txt | tee .testPkg.txt
	@scripts/avg_coverage.sh

.PHONY: vendor
vendor:
	@go mod vendor

.PHONY: build
build:
	@mkdir -p bin
	@go build -mod vendor -o bin/enva meera.tech/envs/cmd/enva
	@go build -mod vendor -o bin/envi meera.tech/envs/cmd/envi
