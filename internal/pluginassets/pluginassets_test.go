package pluginassets

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	// internal/pluginassets → repo root
	return filepath.Clean(filepath.Join(filepath.Dir(file), "../.."))
}

func TestEmbeddedMatchesRepo(t *testing.T) {
	root := repoRoot(t)
	for embedPath, rel := range destByEmbed {
		wantPath := filepath.Join(root, filepath.FromSlash(rel))
		want, err := os.ReadFile(wantPath)
		if err != nil {
			t.Fatalf("read repo %s: %v (run: make sync-plugin-assets)", wantPath, err)
		}
		got, err := ReadEmbed(embedPath)
		if err != nil {
			t.Fatalf("read embed %s: %v", embedPath, err)
		}
		if !bytes.Equal(got, want) {
			t.Fatalf("drift: %s != %s\nrun: make sync-plugin-assets", embedPath, wantPath)
		}
	}
}

func TestMaterialize(t *testing.T) {
	dir := t.TempDir()
	if err := Materialize(dir, "9.9.9-test"); err != nil {
		t.Fatal(err)
	}
	manifest := filepath.Join(dir, ".codex-plugin", "plugin.json")
	if _, err := os.Stat(manifest); err != nil {
		t.Fatalf("missing manifest: %v", err)
	}
	stamp, err := os.ReadFile(filepath.Join(dir, stampName))
	if err != nil {
		t.Fatal(err)
	}
	if string(bytes.TrimSpace(stamp)) != "9.9.9-test" {
		t.Fatalf("stamp: %q", stamp)
	}

	// Second call with same version is a no-op (still succeeds).
	if err := Materialize(dir, "9.9.9-test"); err != nil {
		t.Fatal(err)
	}

	// Version bump rewrites stamp.
	if err := Materialize(dir, "9.9.10-test"); err != nil {
		t.Fatal(err)
	}
	stamp, err = os.ReadFile(filepath.Join(dir, stampName))
	if err != nil {
		t.Fatal(err)
	}
	if string(bytes.TrimSpace(stamp)) != "9.9.10-test" {
		t.Fatalf("stamp after bump: %q", stamp)
	}
}
