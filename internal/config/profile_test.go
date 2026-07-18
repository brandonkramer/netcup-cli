package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInvalidProfileRejected(t *testing.T) {
	dir := t.TempDir()
	for _, bad := range []string{"../evil", "..", ".", "-bad", "bad-", "a/b"} {
		if err := ValidateProfile(bad); err == nil {
			t.Fatalf("expected rejection for %q", bad)
		}
	}
	if _, err := Load("ok-profile", dir); err != nil {
		t.Fatal(err)
	}
	if err := ValidateProfile("a"); err != nil {
		t.Fatal(err)
	}
}

func TestConfigYAMLProfileValidated(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte("profile: ../../outside\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load("default", dir); err == nil {
		t.Fatal("expected config.yaml profile rejection")
	}
}

func TestCacheDirStaysUnderCache(t *testing.T) {
	cfg := &Config{ConfigDir: "/tmp/netcup-cfg", Profile: "default"}
	dir := cfg.CacheDir()
	if !strings.HasPrefix(dir, filepath.Clean("/tmp/netcup-cfg/cache")+string(os.PathSeparator)) && dir != filepath.Clean("/tmp/netcup-cfg/cache/default") {
		t.Fatalf("unexpected cache dir %q", dir)
	}
	cfg.Profile = ".."
	if got := cfg.CacheDir(); got != filepath.Clean("/tmp/netcup-cfg/cache/default") {
		t.Fatalf("escaped profile should fall back, got %q", got)
	}
}
