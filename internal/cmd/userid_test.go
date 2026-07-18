package cmd

import (
	"context"
	"testing"

	"github.com/brandonkramer/netcup-cli/internal/output"
)

func TestResolveUserIDFlag(t *testing.T) {
	a := &App{}
	for _, tc := range []struct {
		in      string
		wantErr bool
		want    int32
	}{
		{"1", false, 1},
		{"42", false, 42},
		{"123junk", true, 0},
		{"0", true, 0},
		{"-1", true, 0},
	} {
		a.Flags.UserID = tc.in
		id, err := a.ResolveUserID(context.Background())
		if tc.wantErr {
			if err == nil {
				t.Fatalf("%q: expected error", tc.in)
			}
			var ee *output.ExitError
			if !asExit(err, &ee) {
				t.Fatalf("%q: want ExitError, got %T", tc.in, err)
			}
			continue
		}
		if err != nil || id != tc.want {
			t.Fatalf("%q: got %d %v", tc.in, id, err)
		}
	}
}

func asExit(err error, ee **output.ExitError) bool {
	if e, ok := err.(*output.ExitError); ok {
		*ee = e
		return true
	}
	return false
}
