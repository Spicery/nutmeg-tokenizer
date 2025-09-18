default:
    @just --list

just test: unittest lint fmt tidy build

just unittest:
    go test -v ./...

just lint:
    golangci-lint run

just fmt:
    gofmt -w .

just tidy:
    go mod tidy

just build:
    go build -o bin/nutmeg-tokeniser ./cmd/nutmeg-tokeniser
