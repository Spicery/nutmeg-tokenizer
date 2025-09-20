default:
    @just --list

test: unittest lint fmt-check tidy build

unittest:
    go test ./...

lint:
    echo "Running linter..."
    @if command -v ~/.tools/ext/bin/golangci-lint >/dev/null 2>&1; then \
        ~/.tools/ext/bin/golangci-lint run; \
    elif command -v golangci-lint >/dev/null 2>&1; then \
        golangci-lint run; \
    else \
        echo "golangci-lint not found, falling back to go vet"; \
        echo "To install golangci-lint locally, run: just install-golangci-lint"; \
        go vet ./...; \
    fi

# Install golangci-lint to ~/.tools/ext/bin  
install-golangci-lint:
    @mkdir -p ~/.tools/ext/bin
    GOBIN=~/.tools/ext/bin go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
    @echo "golangci-lint installed locally to this project in ~/.tools/ext/bin/"
    @echo "Note that ~/.tools/ext/bin is not assumed to be in your PATH"

fmt:
    go fmt ./...

# Check formatting without modifying files
fmt-check:
    ./.tools/repo/bin/go-fmt-check

tidy:
    go mod tidy

build:
    go build -o bin/nutmeg-tokeniser ./cmd/nutmeg-tokeniser
