package cmd

import (
	"path/filepath"
	"testing"
)

func TestDetectInstallMethodHomebrew(t *testing.T) {
	cases := []string{
		"/opt/homebrew/Cellar/netcup/0.1.0/bin/netcup",
		"/usr/local/Cellar/netcup/0.2.0/bin/netcup",
		"/home/linuxbrew/.linuxbrew/Cellar/netcup/0.1.0/bin/netcup",
	}
	for _, p := range cases {
		if got := detectInstallMethod(p); got != installHomebrew {
			t.Fatalf("%s: got %q want %q", p, got, installHomebrew)
		}
	}
}

func TestDetectInstallMethodGo(t *testing.T) {
	root := t.TempDir()
	gobin := filepath.Join(root, "gobin")
	gopath := filepath.Join(root, "gopath")
	t.Setenv("GOBIN", gobin)
	t.Setenv("GOPATH", gopath)

	for _, p := range []string{
		filepath.Join(gobin, "netcup"),
		filepath.Join(gopath, "bin", "netcup"),
	} {
		if got := detectInstallMethod(p); got != installGo {
			t.Fatalf("%s: got %q want %q", p, got, installGo)
		}
	}
}

func TestDetectInstallMethodManual(t *testing.T) {
	t.Setenv("GOBIN", filepath.Join(t.TempDir(), "empty-gobin"))
	t.Setenv("GOPATH", filepath.Join(t.TempDir(), "empty-gopath"))
	p := "/usr/local/bin/netcup"
	if got := detectInstallMethod(p); got != installManual {
		t.Fatalf("%s: got %q want %q", p, got, installManual)
	}
}

func TestUpdatePlan(t *testing.T) {
	brew := updatePlan(installHomebrew)
	if len(brew.commands) != 2 || brew.commands[1][2] != brewFormula {
		t.Fatalf("homebrew plan: %+v", brew)
	}
	goPlan := updatePlan(installGo)
	if len(goPlan.commands) != 1 || goPlan.commands[0][2] != goInstallPkg {
		t.Fatalf("go plan: %+v", goPlan)
	}
	manual := updatePlan(installManual)
	if manual.note == "" || len(manual.commands) != 0 {
		t.Fatalf("manual plan: %+v", manual)
	}
}
