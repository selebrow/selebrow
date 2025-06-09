-include .env
export

SHELL := /bin/bash

FALLBACK_BROWSERS_URI?=https://selebrow.dev/browsers.yaml
REGISTRY_IMAGE?=selebrow
BIN_NAME?=selebrow
BUILD_TARGET=selebrow
REF=$$(git symbolic-ref --short HEAD)
TAG=$$(echo ${REF} | sed 's/^master$$/latest/' | sed 's/^feature\///')

GOPATH?=$(shell go env GOPATH)
GOARCH?=$(shell go env GOARCH)
GOOS?=$(shell go env GOOS)

GCOVER_COBERTURA=$(GOPATH)/bin/gocover-cobertura
GO_JUNIT_REPORT=$(GOPATH)/bin/go-junit-report
MOCKERY=$(GOPATH)/bin/mockery
HELM_DOCS=$(GOPATH)/bin/helm-docs

ifeq ($(GOOS), windows)
BIN_SUFFIX?=.exe
else
BIN_SUFFIX?=
endif

ifndef CI
GitRef=$(shell git rev-parse --abbrev-ref HEAD)
GitSha=$(shell git rev-parse --short HEAD)
else
GitRef=${CI_COMMIT_REF_NAME}
GitSha=${CI_COMMIT_SHORT_SHA}
endif

ldflags=-s -w -X main.GitRef=$(GitRef) -X main.GitSha=$(GitSha) -X github.com/selebrow/selebrow/pkg/config.DefaultFallbackBrowsersURI=$(FALLBACK_BROWSERS_URI)

default:

.PHONY: fmt vet build docker-build lint lint-fix test test-report coverage helm-docs

fmt:
	go fmt ./...

vet:
	go vet ./...

build:
	go build -trimpath -ldflags "$(ldflags)" -o ./bin/$(BIN_NAME)-$(GOOS)-$(GOARCH)$(BIN_SUFFIX) ./cmd/$(BUILD_TARGET)

docker-build:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "$(ldflags)" -o ./bin/$(BIN_NAME)-linux-amd64 ./cmd/$(BUILD_TARGET)
	docker build --pull -t $(REGISTRY_IMAGE):$(TAG) .

lint: golangci-lint
	${GOLANGCI-LINT} run --timeout=5m ./... -v

lint-fix: golangci-lint
	${GOLANGCI-LINT} fmt

test:
	set -o pipefail && go test -tags release --race --vet= --count=1 --covermode=atomic --coverprofile=coverage.out --coverpkg=./internal/...,./pkg/... ./... -v | tee report.txt

junit-report: go-junit-report
	$(GO_JUNIT_REPORT) -set-exit-code < report.txt > report.xml

coverage: gcover-cobertura
	go tool cover --func=coverage.out
	# $(GCOVER_COBERTURA) < coverage.out > coverage.xml

mocks: mockery
	rm -rf mocks/
	$(MOCKERY)

helm-docs: install-helm-docs
	$(HELM_DOCS)

# find or download golangci-lint
# download golangci-lint if necessary
golangci-lint:
ifeq (, $(shell which golangci-lint))
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.1.6
GOLANGCI-LINT=$(GOPATH)/bin/golangci-lint
else
GOLANGCI-LINT=$(shell which golangci-lint)
endif

gcover-cobertura:
ifeq (, $(shell which $(GCOVER_COBERTURA)))
	go install github.com/boumenot/gocover-cobertura@latest
endif

go-junit-report:
ifeq (, $(shell which $(GO_JUNIT_REPORT)))
	go install github.com/jstemmer/go-junit-report@latest
endif

mockery:
ifeq (, $(shell which $(MOCKERY)))
	go install github.com/vektra/mockery/v2@latest
endif

install-helm-docs:
ifeq (, $(shell which $(HELM_DOCS)))
	go install github.com/norwoodj/helm-docs/cmd/helm-docs@latest
endif
