default:
    @just --list

test: unittest lint fmt tidy build

unittest:
    go test -v ./...

lint:
    golangci-lint run

fmt:
    gofmt -w .

tidy:
    go mod tidy

build:
    go build -o bin/nutmeg-tokeniser ./cmd/nutmeg-tokeniser
