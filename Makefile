.PHONY: help build tidy fmt test bench lint cover clean check sloc
.DEFAULT_GOAL := help

COVERAGE_DATA := coverage.out
COVERAGE_HTML := coverage.html

export GOEXPERIMENT := loopvar

help: ## Show the list of available tasks
	@echo "Available Tasks:\n"
	@grep -E '^[a-zA-Z_0-9%-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "%-10s %s\n", $$1, $$2}'

build: ## Compile the binary
	go build -trimpath -ldflags '-s -w' -o ./bin/flyio

tidy: ## Tidy dependencies in go.mod
	go mod tidy

fmt: ## Run go fmt on all source files
	go fmt ./...

test: ## Run the test suite
	go test -race ./...

bench: ## Run benchmarks
	go test ./... -run=None -benchmem -bench .

lint: ## Run the linters and auto-fix if possible
	golangci-lint run --fix

cover: ## Calculate test coverage and render the html
	go test -race -cover -covermode atomic -coverprofile $(COVERAGE_DATA) ./...
	go tool cover -html $(COVERAGE_DATA) -o $(COVERAGE_HTML)
	open $(COVERAGE_HTML)

clean: ## Remove build artifacts and other clutter
	go clean ./...
	rm -rf $(COVERAGE_DATA) $(COVERAGE_HTML) ./bin ./dist

check: test lint ## Run tests and linting in one go

sloc: ## Print lines of code (for fun)
	find . -name "*.go" | xargs wc -l | sort -nr | head
