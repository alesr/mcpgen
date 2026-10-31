// Package checks implements the `mcpgen check` verification pipeline:
// config → tidy → fmt → vet → build → test → conformance (optional) →
// inspector (optional). Every stage reports a structured status.
package checks

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/alesr/mcpgen/internal/cli"
	"github.com/alesr/mcpgen/internal/config"
)

// Stage statuses.
const (
	StatusPassed  = "passed"
	StatusFailed  = "failed"
	StatusSkipped = "skipped"
)

// Stage is one pipeline stage result.
type Stage struct {
	Name     string        `json:"name"`
	Status   string        `json:"status"`
	Command  string        `json:"command,omitempty"`
	Duration string        `json:"duration,omitempty"`
	Message  string        `json:"message,omitempty"`
	Fix      string        `json:"fix,omitempty"`
	Code     cli.ErrorCode `json:"code,omitempty"`
	ExitCode int           `json:"exit_code,omitempty"`
}

// Result is the full pipeline verdict.
type Result struct {
	Dir    string  `json:"dir"`
	OK     bool    `json:"ok"`
	Stages []Stage `json:"stages"`
}

// Options controls the pipeline.
type Options struct {
	Conformance *bool         // nil = auto (on when npx present and transport is http)
	Inspector   bool          // run Inspector v2 CLI smoke
	Timeout     time.Duration // overall timeout; default 10m
}

// DefaultTimeout is the default overall pipeline timeout.
const DefaultTimeout = 10 * time.Minute

// SDKPin is the go-sdk version generated projects must require.
const SDKPin = "v1.8.0"

// Run executes the pipeline in dir and returns the structured verdict. It
// never exits; callers map Result to exit codes.
func Run(ctx context.Context, dir string, opts Options) Result {
	if opts.Timeout <= 0 {
		opts.Timeout = DefaultTimeout
	}
	ctx, cancel := context.WithTimeout(ctx, opts.Timeout)
	defer cancel()

	res := Result{Dir: dir}

	stage := func(name string, fn func() (string, string, error)) {
		if ctx.Err() != nil {
			res.Stages = append(res.Stages, Stage{Name: name, Status: StatusSkipped, Message: "timeout/cancelled before stage"})
			return
		}
		start := time.Now()
		msg, fix, err := fn()
		s := Stage{Name: name, Duration: time.Since(start).Round(time.Millisecond).String()}
		var sk skippedError
		switch {
		case err != nil && errors.As(err, &sk):
			s.Status = StatusSkipped
			s.Message = sk.reason
		case err != nil:
			s.Status = StatusFailed
			s.Message = msg
			s.Fix = fix
			// Preserve structured exit codes (e.g. usage errors from the
			// config stage, prerequisite failures).
			if ee := cli.AsExitError(err); ee != nil {
				s.Code = ee.Code
				s.ExitCode = ee.ExitCode
			}
		default:
			s.Status = StatusPassed
			s.Message = msg
		}
		res.Stages = append(res.Stages, s)
	}

	// Prerequisite: the go toolchain must be present (exit code 6).
	if _, err := exec.LookPath("go"); err != nil {
		stage("prerequisite", func() (string, string, error) {
			ee := cli.NewExitError(cli.ExitPrerequisite, "prerequisite", cli.CodePrerequisite,
				"go toolchain not found on PATH").WithFix("install Go 1.26+ (https://go.dev/dl) and re-run")
			return ee.Message, ee.Fix, ee
		})
		res.OK = false
		return res
	}

	cfg, cfgErr := config.Load(dir)

	stage("config", func() (string, string, error) {
		if cfgErr != nil {
			if errors.Is(cfgErr, config.ErrConfigVersionUnsupported) {
				ee := cli.NewExitError(cli.ExitUsage, "config", cli.CodeConfigVersion, "%v", cfgErr).
					WithFix(fmt.Sprintf("set \"version\": %d in %s, or regenerate the project with `mcpgen new --force`", config.ConfigVersion, filepath.Join(dir, config.FileName)))
				return ee.Message, ee.Fix, ee
			}
			return errText(cfgErr), fixFor(cfgErr, dir), cfgErr
		}
		return fmt.Sprintf("mcpgen.json valid (version %d, %d tools, %d prompts, %d resources)",
			cfg.Version, len(cfg.Tools), len(cfg.Prompts), len(cfg.Resources)), "", nil
	})

	stage("serverjson", func() (string, string, error) {
		if cfgErr != nil {
			return "", "", errSkipped("config invalid")
		}
		if msg, fix, err := validateServerJSON(dir); err != nil {
			return msg, fix, err
		}
		return "server.json valid (MCP Registry descriptor)", "", nil
	})

	stage("tidy", func() (string, string, error) {
		if cfgErr != nil {
			return "", "", errSkipped("config invalid")
		}
		if pinErr := checkSDKPin(dir); pinErr != nil {
			return pinErr.Error(), fmt.Sprintf("update go.mod to require github.com/modelcontextprotocol/go-sdk %s, or regenerate with `mcpgen new --force`", SDKPin), pinErr
		}
		out, err := runCmd(ctx, dir, "go", "mod", "tidy")
		if err != nil {
			return out, fmt.Sprintf("run `go mod tidy` in %s; if the network is unavailable, vendor dependencies", dir), err
		}
		return "go mod tidy", "", nil
	})

	stage("fmt", func() (string, string, error) {
		out, err := runCmd(ctx, dir, "gofmt", "-l", ".")
		if err != nil {
			return out, fmt.Sprintf("run `gofmt -w .` in %s", dir), err
		}
		if strings.TrimSpace(out) != "" {
			files := strings.Fields(out)
			return fmt.Sprintf("unformatted files: %s", strings.Join(files, ", ")),
				fmt.Sprintf("run `gofmt -w %s` in %s", strings.Join(files, " "), dir), fmt.Errorf("gofmt -l reported files")
		}
		return "gofmt clean", "", nil
	})

	stage("vet", func() (string, string, error) {
		out, err := runCmd(ctx, dir, "go", "vet", "./...")
		if err != nil {
			loc := firstGoLocation(out)
			fix := fmt.Sprintf("run `go vet ./...` in %s", dir)
			if loc != "" {
				fix = fmt.Sprintf("first issue at %s; run `go vet ./...` in %s", loc, dir)
			}
			return out, fix, err
		}
		return "go vet clean", "", nil
	})

	stage("build", func() (string, string, error) {
		out, err := runCmd(ctx, dir, "go", "build", "./...")
		if err != nil {
			loc := firstGoLocation(out)
			fix := fmt.Sprintf("run `go build ./...` in %s", dir)
			if loc != "" {
				fix = fmt.Sprintf("first error at %s; run `go build ./...` in %s", loc, dir)
			}
			return out, fix, err
		}
		return "go build ok", "", nil
	})

	stage("test", func() (string, string, error) {
		out, err := runCmd(ctx, dir, "go", "test", "-race", "./...")
		if err != nil {
			loc, test := firstTestFailure(out)
			fix := fmt.Sprintf("run `go test -race ./...` in %s", dir)
			if test != "" {
				if loc != "" {
					fix = fmt.Sprintf("first failure: %s (%s); run `go test -race ./...` in %s", test, loc, dir)
				} else {
					fix = fmt.Sprintf("first failure: %s; run `go test -race ./...` in %s", test, dir)
				}
			}
			return out, fix, err
		}
		return "go test -race ok", "", nil
	})

	// Conformance: optional, auto when npx is present and transport is http.
	confWant := opts.Conformance
	if confWant == nil {
		v := npxAvailable() && cfgErr == nil && cfg.Transport.Type == "http"
		confWant = &v
	}
	stage("conformance", func() (string, string, error) {
		if !*confWant {
			return "", "", errSkipped("disabled")
		}
		if !npxAvailable() {
			// Auto mode skips gracefully; an explicit --conformance without
			// node/npx is a missing prerequisite (exit 6).
			if opts.Conformance != nil && *opts.Conformance {
				ee := cli.NewExitError(cli.ExitPrerequisite, "conformance", cli.CodePrerequisite,
					"node/npx not found but conformance was explicitly requested (--conformance)").
					WithFix("install Node.js (https://nodejs.org) or re-run with --no-conformance")
				return ee.Message, ee.Fix, ee
			}
			return "", "", errSkipped("node/npx not found; install Node.js to run the MCP conformance suite")
		}
		if cfgErr != nil || cfg.Transport.Type != "http" {
			return "", "", errSkipped("server mode requires the http transport")
		}
		if cfg.Protocol == config.ProtocolLatestOnly {
			// The conformance suite (<=0.1.x) speaks the legacy initialize
			// handshake (protocol <= 2025-11-25), which a latest-only server
			// rejects by design. Run the conformance gate with protocol
			// "compat" (the default), which still serves 2026-07-28 to
			// capable clients.
			return "", "", errSkipped("conformance suite speaks the legacy handshake (<=2025-11-25); latest-only servers reject it by design — use protocol \"compat\" for the conformance gate")
		}
		msg, fix, err := runConformance(ctx, dir, cfg)
		return msg, fix, err
	})

	if opts.Inspector {
		stage("inspector", func() (string, string, error) {
			if !npxAvailable() {
				return "", "", errSkipped("node/npx not found")
			}
			if cfgErr != nil {
				return "", "", errSkipped("config invalid")
			}
			msg, fix, err := runInspectorSmoke(ctx, dir, cfg)
			return msg, fix, err
		})
	}

	res.OK = true
	for _, s := range res.Stages {
		if s.Status == StatusFailed {
			res.OK = false
			break
		}
	}
	return res
}

// Err returns an ExitError for a failed pipeline, pointing at the first
// failed stage. Stages that carried a structured exit code (usage errors,
// prerequisites) keep it.
func (r Result) Err() error {
	if r.OK {
		return nil
	}
	for _, s := range r.Stages {
		if s.Status == StatusFailed {
			code := s.Code
			if code == "" {
				code = stageErrorCode(s.Name)
			}
			exitCode := s.ExitCode
			if exitCode == 0 {
				exitCode = cli.ExitFailure
			}
			ee := cli.NewExitError(exitCode, s.Name, code, "%s", s.Message)
			if s.Fix != "" {
				ee = ee.WithFix(s.Fix)
			}
			if s.Command != "" {
				ee = ee.WithCommand(s.Command)
			}
			return ee
		}
	}
	return cli.NewExitError(cli.ExitFailure, "check", cli.CodeStageFailed, "check pipeline failed")
}

func stageErrorCode(stage string) cli.ErrorCode {
	switch stage {
	case "config":
		return cli.CodeConfigInvalid
	case "tidy":
		return cli.CodeTidyFailed
	case "fmt":
		return cli.CodeFmtFailed
	case "vet":
		return cli.CodeVetFailed
	case "build":
		return cli.CodeBuildFailed
	case "test":
		return cli.CodeTestsFailed
	case "conformance":
		return cli.CodeConformanceFailed
	case "inspector":
		return cli.CodeInspectorFailed
	default:
		return cli.CodeStageFailed
	}
}

// errSkipped marks a stage skipped without failing the pipeline.
type skippedError struct{ reason string }

func (e skippedError) Error() string { return "skipped: " + e.reason }

func errSkipped(reason string) error { return skippedError{reason: reason} }

// runCmd runs a command in dir with GOWORK=off, capturing combined output.
func runCmd(ctx context.Context, dir, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GOWORK=off")
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	err := cmd.Run()
	return strings.TrimSpace(buf.String()), err
}

var (
	goLocRe = regexp.MustCompile(`([^\s:]+\.go):(\d+):`)
	failRe  = regexp.MustCompile(`^--- FAIL: (\S+)`)
)

// firstGoLocation returns the first file:line in go tool output.
func firstGoLocation(out string) string {
	if m := goLocRe.FindStringSubmatch(out); m != nil {
		return m[1] + ":" + m[2]
	}
	return ""
}

// firstTestFailure extracts the first failing test name and its file:line.
func firstTestFailure(out string) (loc, test string) {
	lines := strings.Split(out, "\n")
	for _, line := range lines {
		if m := failRe.FindStringSubmatch(strings.TrimSpace(line)); m != nil {
			test = m[1]
		}
		if loc == "" {
			if m := goLocRe.FindStringSubmatch(line); m != nil {
				loc = m[1] + ":" + m[2]
			}
		}
		if test != "" && loc != "" {
			return loc, test
		}
	}
	return loc, test
}

func errText(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func fixFor(err error, dir string) string {
	return fmt.Sprintf("fix %s: %v", filepath.Join(dir, config.FileName), err)
}

// checkSDKPin verifies go.mod requires the pinned go-sdk version.
func checkSDKPin(dir string) error {
	raw, err := os.ReadFile(filepath.Join(dir, "go.mod"))
	if err != nil {
		return fmt.Errorf("read go.mod: %w", err)
	}
	want := "github.com/modelcontextprotocol/go-sdk " + SDKPin
	if !strings.Contains(string(raw), want) {
		return fmt.Errorf("go.mod does not pin %s", want)
	}
	return nil
}

// npxAvailable reports whether node and npx are on PATH.
func npxAvailable() bool {
	if _, err := exec.LookPath("node"); err != nil {
		return false
	}
	_, err := exec.LookPath("npx")
	return err == nil
}
