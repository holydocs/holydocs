GO=go
BUILD_PATH=./bin
GOLANGCI_LINT=$(BUILD_PATH)/golangci-lint
GOLANGCI_LINT_VERSION=v2.5.0

.PHONY: build clean test lint help docker gen-test-docs

build: ## build app
	$(GO) build -o $(BUILD_PATH)/holydocs ./cmd/holydocs

clean: ## remove build and clean go cache
	$(GO) clean
	rm -rf $(BUILD_PATH)

test: ## run tests
	$(GO) test ./... -race -v -covermode=atomic -coverprofile=coverage.out

lint: $(GOLANGCI_LINT) ## run linters
	$(GOLANGCI_LINT) run

$(GOLANGCI_LINT): ## install local golangci-lint
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/$(GOLANGCI_LINT_VERSION)/install.sh | sh -s -- -b $(BUILD_PATH) $(GOLANGCI_LINT_VERSION)

docker: ## build docker image
	docker build -t holydocs:latest .

gen-test-docs: ## generate docs from testdata
	HOLYDOCS_INPUT_DIR=internal/adapters/secondary/schema/testdata HOLYDOCS_OUTPUT_DIR=generated go run cmd/holydocs/main.go gen-docs

# self documenting command
help:
	@grep -E '^[a-zA-Z\\._-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

.DEFAULT_GOAL := help
