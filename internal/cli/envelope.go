package cli

import (
	"encoding/json"
	"fmt"
	"io"
)

// Envelope is the JSON contract for every mcpgen command. stdout carries only
// the envelope (in --json mode) or clean human output; progress and logs go to
// stderr. Errors are always emitted as a structured envelope.
type Envelope struct {
	OK          bool       `json:"ok"`
	Result      any        `json:"result"`
	Error       *ErrorInfo `json:"error"`
	NextActions []string   `json:"next_actions,omitempty"`
}

// ErrorInfo is the structured failure payload.
type ErrorInfo struct {
	Stage    string    `json:"stage"`
	Code     ErrorCode `json:"code"`
	Message  string    `json:"message"`
	Fix      string    `json:"fix,omitempty"`
	Command  string    `json:"command,omitempty"`
	ExitCode int       `json:"exit_code"`
}

// EnvelopeFromError builds a failure envelope from err.
func EnvelopeFromError(err error, nextActions []string) Envelope {
	env := Envelope{OK: false, Result: nil, NextActions: nextActions}
	if ee := AsExitError(err); ee != nil {
		env.Error = &ErrorInfo{
			Stage:    ee.Stage,
			Code:     ee.Code,
			Message:  ee.Message,
			Fix:      ee.Fix,
			Command:  ee.Command,
			ExitCode: ee.ExitCode,
		}
		return env
	}
	env.Error = &ErrorInfo{
		Stage:    "internal",
		Code:     CodeInternal,
		Message:  err.Error(),
		ExitCode: ExitCodeOf(err),
	}
	return env
}

// EnvelopeOK builds a success envelope.
func EnvelopeOK(result any, nextActions []string) Envelope {
	return Envelope{OK: true, Result: result, Error: nil, NextActions: nextActions}
}

// WriteEnvelope writes the envelope as indented JSON to w (stdout).
func WriteEnvelope(w io.Writer, env Envelope) error {
	raw, err := json.MarshalIndent(env, "", "  ")
	if err != nil {
		return err
	}
	_, err = w.Write(append(raw, '\n'))
	return err
}

// Log writes a progress/log line to stderr. Never mixed into stdout.
func Log(w io.Writer, format string, args ...any) {
	_, _ = io.WriteString(w, "mcpgen: "+fmt.Sprintf(format, args...)+"\n")
}
