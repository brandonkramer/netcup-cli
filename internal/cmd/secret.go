package cmd

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/brandonkramer/netcup-cli/internal/output"
	"github.com/charmbracelet/x/term"
)

// readSecretFile reads a secret from path, or from stdin when path is "-".
func readSecretFile(path string) (string, error) {
	var (
		b   []byte
		err error
	)
	if path == "-" {
		b, err = io.ReadAll(os.Stdin)
	} else {
		b, err = os.ReadFile(path)
	}
	if err != nil {
		return "", output.Exit(output.ExitUsage, "read secret file: "+err.Error())
	}
	v := strings.TrimSpace(string(b))
	if v == "" {
		return "", output.Exit(output.ExitUsage, "secret file is empty")
	}
	return v, nil
}

// readSecret resolves a secret from --*-file (use "-" for stdin) or an interactive TTY prompt.
func readSecret(prompt, file string) (string, error) {
	if file != "" {
		return readSecretFile(file)
	}
	if !term.IsTerminal(os.Stdin.Fd()) {
		return "", output.Exit(output.ExitUsage, "password required: use --password-file PATH (or - for stdin), or an interactive TTY")
	}
	fmt.Fprint(os.Stderr, prompt)
	b, err := term.ReadPassword(os.Stdin.Fd())
	fmt.Fprintln(os.Stderr)
	if err != nil {
		return "", err
	}
	v := strings.TrimSpace(string(b))
	if v == "" {
		return "", output.Exit(output.ExitUsage, "password cannot be empty")
	}
	return v, nil
}
