// Package cli defines mcpgen's machine-first CLI contract: the JSON envelope,
// semantic exit codes, and structured errors shared by every command.
package cli

import (
	"errors"
	"fmt"
)

// Semantic exit codes. These are stable: agents may rely on them.
const (
	ExitOK           = 0 // success
	ExitFailure      = 1 // general failure (build/test/conformance failed, etc.)
	ExitUsage        = 2 // usage error (bad flags/args, unknown config version)
	ExitExists       = 3 // target dir exists without --force, or project is not mcpgen-generated
	ExitPermission   = 4 // permission denied (filesystem, port)
	ExitConflict     = 5 // conflict (name already registered, markers missing/edited)
	ExitPrerequisite = 6 // prerequisite missing (go toolchain, node/npx)
)

// ErrorCode is a stable machine-readable code attached to failures.
type ErrorCode string

const (
	CodeInternal          ErrorCode = "INTERNAL"
	CodeUsage             ErrorCode = "USAGE"
	CodeDirExists         ErrorCode = "DIR_EXISTS"
	CodeNotMCPGenProject  ErrorCode = "NOT_MCPGEN_PROJECT"
	CodePermission        ErrorCode = "PERMISSION_DENIED"
	CodeConflict          ErrorCode = "CONFLICT"
	CodePrerequisite      ErrorCode = "PREREQUISITE_MISSING"
	CodeConfigInvalid     ErrorCode = "CONFIG_INVALID"
	CodeConfigVersion     ErrorCode = "CONFIG_VERSION_UNSUPPORTED"
	CodeStageFailed       ErrorCode = "STAGE_FAILED"
	CodeTestsFailed       ErrorCode = "TESTS_FAILED"
	CodeBuildFailed       ErrorCode = "BUILD_FAILED"
	CodeVetFailed         ErrorCode = "VET_FAILED"
	CodeFmtFailed         ErrorCode = "FMT_FAILED"
	CodeTidyFailed        ErrorCode = "TIDY_FAILED"
	CodeConformanceFailed ErrorCode = "CONFORMANCE_FAILED"
	CodeInspectorFailed   ErrorCode = "INSPECTOR_FAILED"
	CodeGenerateFailed    ErrorCode = "GENERATE_FAILED"
	CodeSpecInvalid       ErrorCode = "SPEC_INVALID"
	CodeIO                ErrorCode = "IO_ERROR"
)

// ExitError is an error carrying a semantic exit code and, optionally, the
// structured error payload emitted in the JSON envelope.
type ExitError struct {
	ExitCode int
	Stage    string
	Code     ErrorCode
	Message  string
	Fix      string
	Command  string
}

func (e *ExitError) Error() string { return e.Message }

// ExitCodeOf extracts the semantic exit code from err (ExitOK for nil,
// ExitFailure for errors without a code).
func ExitCodeOf(err error) int {
	if err == nil {
		return ExitOK
	}
	if ee, ok := errors.AsType[*ExitError](err); ok {
		return ee.ExitCode
	}
	return ExitFailure
}

// AsExitError extracts the structured payload from err, or nil.
func AsExitError(err error) *ExitError {
	ee, _ := errors.AsType[*ExitError](err)
	return ee
}

// NewExitError builds an ExitError.
func NewExitError(exitCode int, stage string, code ErrorCode, format string, args ...any) *ExitError {
	return &ExitError{
		ExitCode: exitCode,
		Stage:    stage,
		Code:     code,
		Message:  fmt.Sprintf(format, args...),
	}
}

// WithFix returns a copy of e with the fix hint set.
func (e *ExitError) WithFix(fix string) *ExitError {
	c := *e
	c.Fix = fix
	return &c
}

// WithCommand returns a copy of e with the offending command set.
func (e *ExitError) WithCommand(cmd string) *ExitError {
	c := *e
	c.Command = cmd
	return &c
}
