package output

import "fmt"

const (
	ExitOK          = 0
	ExitAPI         = 1
	ExitUsage       = 2
	ExitAuth        = 3
	ExitTimeout     = 4
	ExitNotFound    = 5
	ExitInterrupted = 130
)

// ExitError carries a process exit code.
type ExitError struct {
	Code     int
	Message  string
	Rendered bool // true when Fail already wrote structured/human output
}

func (e *ExitError) Error() string {
	if e.Message == "" {
		return "exit"
	}
	return e.Message
}

func Exit(code int, msg string) *ExitError {
	return &ExitError{Code: code, Message: msg}
}

func Exitf(code int, format string, args ...any) *ExitError {
	return Exit(code, fmt.Sprintf(format, args...))
}
