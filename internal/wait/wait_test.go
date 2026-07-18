package wait

import (
	"errors"
	"io"
	"testing"
)

func TestIsTransient(t *testing.T) {
	if !isTransient(io.ErrUnexpectedEOF) {
		t.Fatal("EOF")
	}
	if !isTransient(errors.New("read tcp: connection reset by peer")) {
		t.Fatal("reset")
	}
	if isTransient(errors.New("HTTP 404")) {
		t.Fatal("404")
	}
}
