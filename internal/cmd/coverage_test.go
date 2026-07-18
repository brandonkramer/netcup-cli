package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var httpMethods = map[string]bool{
	"GET": true, "POST": true, "PUT": true, "PATCH": true, "DELETE": true, "HEAD": true,
}

// requiredCurated lists daily operations that must have a named curated command (not only api|call).
var requiredCurated = []string{
	"GET /api/ping",
	"GET /api/v1/maintenance",
	"GET /api/v1/openapi",
	"POST /api/v1/openapi/mcp",
	"GET /api/v1/servers",
	"GET /api/v1/servers/{serverId}",
	"PATCH /api/v1/servers/{serverId}",
	"POST /api/v1/servers/{serverId}/interfaces",
	"GET /api/v1/servers/{serverId}/interfaces",
	"PUT /api/v1/servers/{serverId}/interfaces/{mac}",
	"PUT /api/v1/servers/{serverId}/interfaces/{mac}/firewall",
	"POST /api/v1/users/{userId}/firewall-policies",
	"PUT /api/v1/users/{userId}/firewall-policies/{id}",
	"PUT /api/v1/users/{userId}/vlans/{vlanId}",
	"PUT /api/v1/users/{userId}",
	"POST /api/v1/users/{userId}/ssh-keys",
	"POST /api/v1/servers/{serverId}/snapshots",
	"GET /api/v1/tasks",
}

func findOpenAPIPath(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		candidate := filepath.Join(dir, "openapi.json")
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	t.Fatal("openapi.json not found")
	return ""
}

func loadOpenAPIOperations(t *testing.T, path string) []string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := json.Unmarshal(b, &doc); err != nil {
		t.Fatal(err)
	}
	paths, _ := doc["paths"].(map[string]any)
	var ops []string
	for p, item := range paths {
		opsMap, ok := item.(map[string]any)
		if !ok {
			continue
		}
		for method := range opsMap {
			ml := strings.ToUpper(method)
			if !httpMethods[ml] {
				continue
			}
			ops = append(ops, ml+" "+p)
		}
	}
	return ops
}

func TestCoverageGate(t *testing.T) {
	openAPIPath := findOpenAPIPath(t)
	ops := loadOpenAPIOperations(t, openAPIPath)

	if len(ops) != 87 {
		t.Fatalf("expected 87 OpenAPI operations, got %d", len(ops))
	}
	t.Logf("OpenAPI operations: %d", len(ops))

	if len(curatedCoverage) != 87 {
		t.Fatalf("curatedCoverage has %d entries, want 87", len(curatedCoverage))
	}

	opSet := make(map[string]struct{}, len(ops))
	for _, op := range ops {
		opSet[op] = struct{}{}
		val, ok := curatedCoverage[op]
		if !ok {
			t.Errorf("missing curatedCoverage entry for %s", op)
			continue
		}
		if strings.TrimSpace(val) == "" {
			t.Errorf("empty curatedCoverage value for %s", op)
		}
	}

	for key := range curatedCoverage {
		if _, ok := opSet[key]; !ok {
			t.Errorf("curatedCoverage key not in OpenAPI: %s", key)
		}
	}

	for _, op := range requiredCurated {
		val, ok := curatedCoverage[op]
		if !ok {
			t.Errorf("requiredCurated op missing from curatedCoverage: %s", op)
			continue
		}
		if val == "api|call" {
			t.Errorf("requiredCurated op %s must not be only api|call", op)
		}
	}
}
