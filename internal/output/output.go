package output

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/table"
	"gopkg.in/yaml.v3"
)

type Format string

const (
	FormatTable Format = "table"
	FormatJSON  Format = "json"
	FormatJSONL Format = "jsonl"
	FormatYAML  Format = "yaml"
	FormatBrief Format = "brief"
)

type CacheMeta struct {
	Key    string `json:"key"`
	AgeMS  int64  `json:"age_ms"`
	TTLMS  int64  `json:"ttl_ms"`
	Stale  bool   `json:"stale"`
}

type Envelope struct {
	OK         bool            `json:"ok"`
	Command    string          `json:"command,omitempty"`
	HTTPStatus int             `json:"http_status,omitempty"`
	Cached     bool            `json:"cached,omitempty"`
	Cache      *CacheMeta      `json:"cache,omitempty"`
	TookMS     int64           `json:"took_ms,omitempty"`
	Data       any             `json:"data"`
	Task       any             `json:"task"`
	Warnings   []string        `json:"warnings,omitempty"`
	Error      *ErrorBody      `json:"error,omitempty"`
}

type ErrorBody struct {
	Type    string   `json:"type"`
	Code    string   `json:"code,omitempty"`
	Message string   `json:"message"`
	Fields  []any    `json:"fields,omitempty"`
}

type Printer struct {
	Out     io.Writer
	Err     io.Writer
	Format  Format
	Quiet   bool
	Verbose bool
	Command string
	Started time.Time
}

func NewPrinter(format string) *Printer {
	f := resolveFormat(format)
	return &Printer{
		Out:     os.Stdout,
		Err:     os.Stderr,
		Format:  f,
		Started: time.Now(),
	}
}

func resolveFormat(explicit string) Format {
	if explicit != "" {
		return Format(strings.ToLower(explicit))
	}
	if v := os.Getenv("NETCUP_FORMAT"); v != "" {
		return Format(strings.ToLower(v))
	}
	fi, err := os.Stdout.Stat()
	if err == nil && (fi.Mode()&os.ModeCharDevice) == 0 {
		return FormatJSON
	}
	return FormatTable
}

func (p *Printer) TookMS() int64 {
	return time.Since(p.Started).Milliseconds()
}

func (p *Printer) Success(data any, opts ...func(*Envelope)) error {
	env := Envelope{
		OK:       true,
		Command:  p.Command,
		TookMS:   p.TookMS(),
		Data:     data,
		Task:     nil,
		Warnings: []string{},
	}
	for _, o := range opts {
		o(&env)
	}
	return p.writeSuccess(env, data)
}

func WithHTTPStatus(code int) func(*Envelope) {
	return func(e *Envelope) { e.HTTPStatus = code }
}

func WithTask(task any) func(*Envelope) {
	return func(e *Envelope) { e.Task = task }
}

func WithCached(meta *CacheMeta) func(*Envelope) {
	return func(e *Envelope) {
		e.Cached = true
		e.Cache = meta
	}
}

func (p *Printer) writeSuccess(env Envelope, data any) error {
	switch p.Format {
	case FormatJSON:
		return p.writeJSON(env)
	case FormatJSONL:
		return p.writeJSONL(data)
	case FormatYAML:
		return yaml.NewEncoder(p.Out).Encode(env)
	case FormatBrief:
		return p.writeBrief(data)
	default:
		return p.writeTableOrText(data)
	}
}

func (p *Printer) Fail(exitCode int, errType, code, message string, httpStatus int, fields []any) error {
	env := Envelope{
		OK:         false,
		Command:    p.Command,
		HTTPStatus: httpStatus,
		TookMS:     p.TookMS(),
		Data:       nil,
		Task:       nil,
		Error: &ErrorBody{
			Type:    errType,
			Code:    code,
			Message: message,
			Fields:  fields,
		},
	}
	switch p.Format {
	case FormatJSON, FormatJSONL, FormatYAML:
		_ = p.writeJSON(env)
	default:
		style := lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
		fmt.Fprintln(p.Err, style.Render("Error: "+message))
	}
	return Exit(exitCode, message)
}

func (p *Printer) writeJSON(v any) error {
	enc := json.NewEncoder(p.Out)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func (p *Printer) writeJSONL(data any) error {
	enc := json.NewEncoder(p.Out)
	switch items := data.(type) {
	case []any:
		for _, item := range items {
			if err := enc.Encode(item); err != nil {
				return err
			}
		}
		return nil
	case []map[string]any:
		for _, item := range items {
			if err := enc.Encode(item); err != nil {
				return err
			}
		}
		return nil
	default:
		return enc.Encode(data)
	}
}

func (p *Printer) writeBrief(data any) error {
	switch rows := data.(type) {
	case []map[string]any:
		for _, row := range rows {
			fmt.Fprintln(p.Out, briefLine(row))
		}
		return nil
	case map[string]any:
		fmt.Fprintln(p.Out, briefLine(rows))
		return nil
	default:
		b, err := json.Marshal(data)
		if err != nil {
			return err
		}
		fmt.Fprintln(p.Out, string(b))
		return nil
	}
}

func briefLine(row map[string]any) string {
	keys := []string{"id", "name", "nickname", "state", "uuid", "hostname"}
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		if v, ok := row[k]; ok && v != nil {
			parts = append(parts, fmt.Sprint(v))
		}
	}
	if len(parts) == 0 {
		b, _ := json.Marshal(row)
		return string(b)
	}
	return strings.Join(parts, "\t")
}

// TableData is rendered in table format; other formats get Data as-is.
type TableData struct {
	Headers []string
	Rows    [][]string
	Raw     any
}

func (p *Printer) writeTableOrText(data any) error {
	if td, ok := data.(TableData); ok {
		t := table.New().
			Headers(td.Headers...).
			StyleFunc(func(row, col int) lipgloss.Style {
				if row == table.HeaderRow {
					return lipgloss.NewStyle().Bold(true)
				}
				return lipgloss.NewStyle()
			})
		for _, row := range td.Rows {
			t = t.Row(row...)
		}
		fmt.Fprintln(p.Out, t.String())
		return nil
	}
	if s, ok := data.(string); ok {
		fmt.Fprintln(p.Out, s)
		return nil
	}
	// Fallback: pretty JSON for humans when no table projection.
	enc := json.NewEncoder(p.Out)
	enc.SetIndent("", "  ")
	return enc.Encode(data)
}

func (p *Printer) Warn(msg string) {
	if p.Quiet {
		return
	}
	fmt.Fprintln(p.Err, msg)
}

func (p *Printer) Info(msg string) {
	if p.Quiet || p.Format == FormatJSON || p.Format == FormatJSONL || p.Format == FormatBrief {
		return
	}
	fmt.Fprintln(p.Err, msg)
}
