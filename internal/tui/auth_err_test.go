package tui

import (
	"errors"
	"fmt"
	"testing"
)

func TestIsAuthFailure(t *testing.T) {
	if !isAuthFailure(newAuthError("HTTP 401")) {
		t.Fatal("AuthError")
	}
	if !isAuthFailure(fmt.Errorf("list servers: HTTP 401 — run netcup auth login")) {
		t.Fatal("string 401")
	}
	if !isAuthFailure(errors.New("not logged in")) {
		t.Fatal("not logged in")
	}
	if isAuthFailure(fmt.Errorf("HTTP 403 — denied")) {
		t.Fatal("403 should not gate")
	}
	if isAuthFailure(nil) {
		t.Fatal("nil")
	}
}
