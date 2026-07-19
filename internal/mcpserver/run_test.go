package mcpserver

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestGuardCLIArgs(t *testing.T) {
	t.Parallel()
	cases := []struct {
		args    []string
		wantErr string
	}{
		{nil, "non-empty"},
		{[]string{"tui"}, "not allowed"},
		{[]string{"mcp"}, "not allowed"},
		{[]string{"install-mcp"}, "not allowed"},
		{[]string{"completion"}, "not allowed"},
		{[]string{"nope"}, "not allowlisted"},
		{[]string{"auth", "login"}, "status|whoami"},
		{[]string{"auth"}, "status|whoami"},
		{[]string{"servers", "list", "tui"}, "tui/completion"},
		{[]string{"disks", "list"}, ""},
		{[]string{"auth", "whoami"}, ""},
		{[]string{"ping"}, ""},
	}
	for _, tc := range cases {
		err := guardCLIArgs(tc.args)
		if tc.wantErr == "" {
			if err != nil {
				t.Fatalf("args %v: unexpected err %v", tc.args, err)
			}
			continue
		}
		if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
			t.Fatalf("args %v: got %v, want substring %q", tc.args, err, tc.wantErr)
		}
	}
}

func TestLooksSecretFlag(t *testing.T) {
	t.Parallel()
	if !looksSecretFlag("--password-file") || !looksSecretFlag("--password=/x") {
		t.Fatal("expected secret flags")
	}
	if looksSecretFlag("--format") || looksSecretFlag("-s") {
		t.Fatal("expected non-secret flags")
	}
}

func TestEnsureJSONFormat(t *testing.T) {
	t.Parallel()
	got := ensureJSONFormat([]string{"ping"})
	if len(got) < 3 || got[0] != "--format" || got[1] != "json" {
		t.Fatalf("got %v", got)
	}
	same := ensureJSONFormat([]string{"--format", "yaml", "ping"})
	if same[0] != "--format" || same[1] != "yaml" {
		t.Fatalf("should keep existing format: %v", same)
	}
}

func TestWithServer(t *testing.T) {
	t.Parallel()
	if got := withServer([]string{"a"}, ""); len(got) != 1 {
		t.Fatalf("empty server: %v", got)
	}
	got := withServer([]string{"servers", "get"}, "99")
	if len(got) != 4 || got[2] != "-s" || got[3] != "99" {
		t.Fatalf("got %v", got)
	}
}

func TestRunCLIBlocksSecrets(t *testing.T) {
	t.Parallel()
	_, err := runCLI(context.Background(), []string{"servers", "set", "root-password", "--password-file", "x"}, true, time.Second)
	if err == nil || !strings.Contains(err.Error(), "secret-bearing") {
		t.Fatalf("got %v", err)
	}
}

func TestRunCLIFakeBinary(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell fake binary")
	}
	dir := t.TempDir()
	script := filepath.Join(dir, "fake-netcup")
	argsFile := filepath.Join(dir, "args.txt")
	body := "#!/bin/sh\nprintf '%s\\n' \"$*\" > '" + argsFile + "'\necho '{\"ok\":true}'\n"
	if err := os.WriteFile(script, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("NETCUP_BIN", script)

	out, err := runCLI(context.Background(), []string{"ping"}, false, 5*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, `"ok"`) {
		t.Fatalf("out=%s", out)
	}
	argBytes, err := os.ReadFile(argsFile)
	if err != nil {
		t.Fatal(err)
	}
	args := string(argBytes)
	if !strings.Contains(args, "--format") || !strings.Contains(args, "json") || !strings.Contains(args, "-q") {
		t.Fatalf("expected format/quiet flags: %s", args)
	}

	if _, err = runCLI(context.Background(), []string{"servers", "reboot"}, true, 5*time.Second); err != nil {
		t.Fatal(err)
	}
	argBytes, err = os.ReadFile(argsFile)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(argBytes), "-y") {
		t.Fatalf("confirm should add -y: %s", argBytes)
	}
}

func TestRunCLIExitCode(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell fake binary")
	}
	dir := t.TempDir()
	script := filepath.Join(dir, "fake-netcup")
	if err := os.WriteFile(script, []byte("#!/bin/sh\necho nope\nexit 2\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("NETCUP_BIN", script)
	_, err := runCLI(context.Background(), []string{"ping"}, false, 5*time.Second)
	if err == nil || !strings.Contains(err.Error(), "exited 2") {
		t.Fatalf("got %v", err)
	}
}

func TestNetcupBinEnv(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bin")
	t.Setenv("NETCUP_BIN", path)
	got, err := netcupBin()
	if err != nil || got != path {
		t.Fatalf("got %q %v", got, err)
	}
}
