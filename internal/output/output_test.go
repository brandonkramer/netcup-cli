package output

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"
)

func TestJSONEnvelope(t *testing.T) {
	var buf bytes.Buffer
	p := &Printer{Out: &buf, Err: &buf, Format: FormatJSON, Command: "ping", Started: time.Now()}
	if err := p.Success(map[string]any{"ok": true}, WithHTTPStatus(200)); err != nil {
		t.Fatal(err)
	}
	var env Envelope
	if err := json.Unmarshal(buf.Bytes(), &env); err != nil {
		t.Fatal(err)
	}
	if !env.OK || env.Command != "ping" || env.HTTPStatus != 200 {
		t.Fatalf("unexpected envelope: %+v", env)
	}
}
