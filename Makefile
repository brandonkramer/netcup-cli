.PHONY: generate build test lint coverage

coverage:
	go test ./internal/cmd -run TestCoverageGate -count=1

generate:
	go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@v2.5.1 -config oapi-codegen.yaml openapi.json

build:
	mkdir -p dist
	go build -o dist/netcup ./cmd/netcup

test:
	go test ./...

lint:
	golangci-lint run ./...
