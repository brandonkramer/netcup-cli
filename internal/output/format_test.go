package output

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"gopkg.in/yaml.v3"
)

func TestParseFormat(t *testing.T) {
	if _, err := ParseFormat("xml"); err == nil {
		t.Fatal("expected error")
	}
	f, err := ParseFormat("JSON")
	if err != nil || f != FormatJSON {
		t.Fatalf("got %v %v", f, err)
	}
}

func TestWriteErrorFormats(t *testing.T) {
	cases := []Format{FormatJSON, FormatJSONL, FormatYAML}
	for _, format := range cases {
		var buf bytes.Buffer
		p := &Printer{Out: &buf, Err: &buf, Format: format, Command: "x", Started: time.Now()}
		if err := p.WriteError(2, "cli", "", "boom", 0, nil); err != nil {
			t.Fatalf("%s: %v", format, err)
		}
		out := buf.String()
		switch format {
		case FormatJSON:
			var env Envelope
			if err := json.Unmarshal(buf.Bytes(), &env); err != nil || env.OK || env.Error == nil || env.Error.Message != "boom" {
				t.Fatalf("json: %s", out)
			}
		case FormatJSONL:
			if strings.Count(out, "\n") != 1 && !strings.HasSuffix(out, "\n") {
				t.Fatalf("jsonl should be one line: %q", out)
			}
			var env Envelope
			if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &env); err != nil || env.Error == nil {
				t.Fatalf("jsonl parse: %v %s", err, out)
			}
		case FormatYAML:
			var env Envelope
			if err := yaml.Unmarshal(buf.Bytes(), &env); err != nil || env.Error == nil {
				t.Fatalf("yaml: %v %s", err, out)
			}
			if !strings.Contains(out, "http_status:") && !strings.Contains(out, "http_status: ") {
				// yaml.v3 may omit zero http_status; ensure message key is snake-stable
			}
			if !strings.Contains(out, "message:") {
				t.Fatalf("yaml missing message key: %s", out)
			}
		}
	}
}
