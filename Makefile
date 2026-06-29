.PHONY: build test lint cover run snapshot tidy clean

BINARY := ical-filter-proxy
PKG    := ./...

build: ## Build the binary
	go build -o $(BINARY) ./cmd/ical-filter-proxy

test: ## Run tests with race detector
	go test -race $(PKG)

cover: ## Run tests and print coverage for internal packages
	go test -coverprofile=coverage.out ./internal/...
	go tool cover -func=coverage.out | tail -1

lint: ## Run go vet and golangci-lint
	go vet $(PKG)
	golangci-lint run $(PKG)

run: ## Run locally against ./config.yml on :8000
	go run ./cmd/ical-filter-proxy --config ./config.yml --addr :8000

snapshot: ## Build a local snapshot release (binaries + images, no publish)
	goreleaser release --snapshot --clean

tidy: ## Tidy go modules
	go mod tidy

clean: ## Remove build artifacts
	$(RM) $(BINARY) coverage.out
	$(RM) -r dist
