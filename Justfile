default:
    @just --list

test: unittest lint fmt-check tidy build

unittest:
    go test -v ./...

lint:
    echo "Running linter..."
    @if command -v ~/.tools/bin/golangci-lint >/dev/null 2>&1; then \
        ~/.tools/bin/golangci-lint run; \
    elif command -v golangci-lint >/dev/null 2>&1; then \
        golangci-lint run; \
    else \
        echo "golangci-lint not found, falling back to go vet"; \
        echo "To install golangci-lint locally, run: just install-golangci-lint"; \
        go vet ./...; \
    fi

# Install golangci-lint to ~/.tools/bin
install-golangci-lint:
    @mkdir -p ~/.tools/bin
    GOBIN=~/.tools/bin go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
    @echo "golangci-lint installed locally to this project in ~/.tools/bin/"
    @echo "Note that ~/.tools/bin is not assumed to be in your PATH"

fmt:
    go fmt ./...

# Check formatting without modifying files
fmt-check:
    ./.tools/scripts/go_fmt_dryrun.sh

tidy:
    go mod tidy

build:
    go build -o bin/nutmeg-tokeniser ./cmd/nutmeg-tokeniser
