PROJECT_NAME := "erouska-backend"
PKG := "github.com/covid19cz/erouska-backend"
PKG_LIST := $(shell go list ${PKG}/... | grep -v /vendor/)
GO_FILES := $(shell find . -name '*.go' | grep -v /vendor/ | grep -v _test.go)

.PHONY: all dep build clean test coverage coverhtml lint

all: build

help:
	@grep -E '^[a-zA-Z0-9_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

functions := $(shell find cmd -name \*main.go | awk -F'/' '{print $$2}')

build: ## Build golang binaries
	@for function in $(functions) ; do \
		go build -mod vendor -ldflags="-s -w" -o bin/$$function cmd/$$function/*.go ; \
	done

lint: ## Lint the files
	@golint -set_exit_status ${PKG_LIST}

test: ## Run unittests
	@go test -mod vendor -short ${PKG_LIST}

dep: ## Get the dependencies
	@go get -u golang.org/x/lint/golint
