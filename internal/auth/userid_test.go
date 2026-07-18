package auth

import "testing"

func TestParseSCPUserID(t *testing.T) {
	tests := []struct {
		in      string
		want    int32
		wantErr bool
	}{
		{"12345", 12345, false},
		{"f:aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee:12345", 12345, false},
		{"", 0, true},
		{"f:uuid:notanumber", 0, true},
		{"nope", 0, true},
	}
	for _, tt := range tests {
		got, err := ParseSCPUserID(tt.in)
		if tt.wantErr {
			if err == nil {
				t.Fatalf("%q: expected error", tt.in)
			}
			continue
		}
		if err != nil {
			t.Fatalf("%q: %v", tt.in, err)
		}
		if got != tt.want {
			t.Fatalf("%q: got %d want %d", tt.in, got, tt.want)
		}
	}
}
