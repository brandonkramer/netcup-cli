.PHONY: generate build test lint coverage sync-plugin-assets

coverage:
	go test ./internal/cmd -run TestCoverageGate -count=1

generate:
	go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@v2.5.1 -config oapi-codegen.yaml openapi.json

# Refresh go:embed copies from the canonical plugin trees at repo root.
sync-plugin-assets:
	mkdir -p internal/pluginassets/files/codex-plugin \
		internal/pluginassets/files/claude-plugin \
		internal/pluginassets/files/cursor-plugin \
		internal/pluginassets/files/skills/netcup
	cp .codex-plugin/plugin.json internal/pluginassets/files/codex-plugin/
	cp .claude-plugin/plugin.json .claude-plugin/marketplace.json internal/pluginassets/files/claude-plugin/
	cp .cursor-plugin/plugin.json .cursor-plugin/mcp.json internal/pluginassets/files/cursor-plugin/
	cp skills/netcup/SKILL.md internal/pluginassets/files/skills/netcup/

build:
	mkdir -p bin
	go build -o bin/netcup ./cmd/netcup

test:
	go test ./...

lint:
	golangci-lint run ./...
