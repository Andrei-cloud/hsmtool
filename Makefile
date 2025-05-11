.PHONY: help run run-human build test fmt lint clean plugins cli install

help: ## Display this help screen.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } ' $(MAKEFILE_LIST)

gen: ## Geenerate plugins code.
	go generate ./...

run: ## Start HSM server (structured JSON logs).
	go run ./cmd/hsmtool/main.go

build: ## Build HSM binary.
	CGO_ENABLED=0 go build -o bin/hsmtool ./cmd/hsmtool/main.go

test: ## Run tests.
	go test ./... -v

all: ## Build, test and clean.
	make gen && make plugins && make run

