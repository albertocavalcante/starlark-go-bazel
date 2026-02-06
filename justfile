# starlark-go-bazel development commands
# Run `just --list` to see available commands

# Default recipe: run tests
default: test

# Build all packages
build:
    go build ./...

# Build WASM target
build-wasm:
    GOOS=js GOARCH=wasm go build -o main.wasm ./wasm/

# Run all tests
test:
    go tool -modfile=tools.go.mod gotestsum --format pkgname-and-test-fails -- ./...

# Run tests with verbose output
test-v:
    go tool -modfile=tools.go.mod gotestsum --format standard-verbose -- ./...

# Run tests with race detector
test-race:
    go tool -modfile=tools.go.mod gotestsum --format pkgname-and-test-fails -- -race ./...

# Run tests with coverage
test-cover:
    go tool -modfile=tools.go.mod gotestsum --format pkgname-and-test-fails -- -coverprofile=coverage.out ./...
    go tool cover -html=coverage.out -o coverage.html
    @echo "Coverage report: coverage.html"

# Run benchmarks
bench:
    go test -bench=. -benchmem ./...

# Run golangci-lint
lint:
    golangci-lint run

# Run go vet
vet:
    go vet ./...

# Format code
fmt:
    gofmt -w .

# Check if code is formatted
fmt-check:
    #!/usr/bin/env bash
    unformatted=$(gofmt -l .)
    if [ -n "$unformatted" ]; then
        echo "Code is not formatted. Run 'just fmt'"
        echo "$unformatted"
        exit 1
    fi

# Tidy all go.mod files
tidy:
    go mod tidy
    go mod tidy -modfile=tools.go.mod

# Verify dependencies
verify:
    go mod verify

# Clean build artifacts
clean:
    go clean -cache
    rm -f main.wasm coverage.out coverage.html test-output.json

# Run all checks (CI)
ci: build test lint vet fmt-check

# Development workflow: format, lint, test
dev: fmt lint test

# Show module info
info:
    @echo "Module: $(go list -m)"
    @echo "Go version: $(go version)"
    @echo ""
    @echo "Packages:"
    @go list ./...

# Generate documentation server
doc:
    @echo "Starting godoc server at http://localhost:6060"
    @echo "View at: http://localhost:6060/pkg/github.com/albertocavalcante/starlark-go-bazel/"
    godoc -http=:6060
