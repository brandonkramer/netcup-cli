package apiraw

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveExactAndFill(t *testing.T) {
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	idx, err := Load(filepath.Join(root, "openapi.json"))
	if err != nil {
		t.Fatal(err)
	}
	op, _, err := idx.Best("GET /api/v1/servers")
	if err != nil {
		t.Fatal(err)
	}
	if op.Method != "GET" || op.Path != "/api/v1/servers" {
		t.Fatalf("got %+v", op)
	}
	op2, _, err := idx.Best("snapshot create")
	if err != nil {
		t.Fatal(err)
	}
	if op2.Method != "POST" || op2.Path != "/api/v1/servers/{serverId}/snapshots" {
		t.Fatalf("snapshot create => %+v", op2)
	}
	filled, err := FillPath(op2.Path, map[string]string{"serverId": "42"})
	if err != nil || filled != "/api/v1/servers/42/snapshots" {
		t.Fatalf("fill: %q %v", filled, err)
	}
}

func TestAmbiguous(t *testing.T) {
	b, err := os.ReadFile(filepath.Join("..", "..", "openapi.json"))
	if err != nil {
		t.Fatal(err)
	}
	idx, err := Parse(b)
	if err != nil {
		t.Fatal(err)
	}
	_, matches, err := idx.Best("server")
	if err == nil {
		t.Fatal("expected ambiguity")
	}
	if len(matches) < 2 {
		t.Fatalf("expected candidates, got %d", len(matches))
	}
}
