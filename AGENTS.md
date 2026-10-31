# AGENTS.md

Operational guidance for coding agents working in `mcpgen`.

## Instruction precedence
- Apply instructions in this order:
  1) system/developer/user runtime instructions,
  2) repository-local Cursor/Copilot rule files,
  3) this `AGENTS.md`.

## Agent quickstart checklist
- Read runtime instructions and active rule files before touching code.
- Skim `main.go` and `internal/app` first to understand control flow.
- Check `Makefile` and `go.mod` to confirm command and version expectations.
- Keep scope tight; do not refactor unrelated packages.
- Prefer targeted tests during iteration; run full format/tests before handoff.
- Update docs when flags, workflows, or generation behavior change.

## Cursor and Copilot rules
- Always check for additional repository rules in:
  - `.cursor/rules/**`
  - `.cursorrules`
  - `.github/copilot-instructions.md`
- Current status in this repository:
  - `.cursor/rules/**`: not present
  - `.cursorrules`: not present
  - `.github/copilot-instructions.md`: not present
- If these files appear later, treat them as higher priority than this file.

## Repository snapshot
- Module: `github.com/alesr/mcpgen`
- Go version: `1.26` (CLI + templates); generated projects pin `go-sdk v1.8.0`, MCP spec 2026-07-28
- CLI entrypoint: `main.go` → `internal/app.Run(args, stdin, stdout, stderr) int` (semantic exit codes)
- Core packages:
  - `internal/cli`: JSON envelope, exit codes, `mcpgen schema` document
  - `internal/config`: `mcpgen.json` v1 model, validation, load/save, JSON Schema (`schema.json`)
  - `internal/ops`: core operations shared by CLI and MCP mode (`new`/`add`/`check`/`schema`)
  - `internal/generator`: templates (`templates/*.gotmpl`, go:embed), ownership model, `.mcpgen/manifest.json`, golden tests (`testdata/golden`, regen via `make golden`)
  - `internal/adder`: `mcpgen add` — marker-region insertion via go/parser + go/ast + go/format
  - `internal/checks`: `mcpgen check` staged pipeline (config/tidy/fmt/vet/build/test/conformance/inspector)
  - `internal/mcpserver`: `mcpgen mcp` (stdio MCP server, dogfoods go-sdk)
  - `internal/scaffold`, `internal/ui`, `internal/pkg/utils`
- Extra module: `generated/` has its own `go.mod` and is a checked-in sample project regenerated with `mcpgen new generated ...`

## Build, lint, and test commands

### Preferred Make targets
- `make build` -> `go build ./...`
- `make fmt` -> `gofmt -w .`
- `make vet` -> `go vet ./...`
- `make test` -> `go test -count=1 -race -v ./...`
- `make tidy` -> `go mod tidy`
- `make check` -> run `tidy`, `fmt`, `vet`, `test`
- `make gen` -> run CLI from source (`go run .`)

### Direct commands (repo root)
- Build all packages: `go build ./...`
- Run the CLI: `go run .`
- Format all files: `gofmt -w .`
- Check format drift only: `gofmt -l .`
- Static analysis: `go vet ./...`
- Run all tests: `go test ./...`
- Run all tests uncached: `go test ./... -count=1`
- Run all tests with race: `go test ./... -race`
- CI-like test run: `go test -race -count=1 -v -coverprofile=coverage.out ./...`
- Dependency health: `go mod download && go mod verify`

### Single-test commands (important)
- Run one package:
  - `go test ./internal/config`
- Run one test function (exact match):
  - `go test ./internal/config -run '^TestConfig_Validate$'`
- Run one subtest (exact parent/subtest path):
  - `go test ./internal/config -run '^TestConfig_Validate/accumulates multiple errors$'`
- Run one test with verbose + no cache:
  - `go test ./internal/app -run '^TestParseRunOptions$' -count=1 -v`
- Run a second concrete example from generator package:
  - `go test ./internal/generator -run '^TestGenerator_Run$' -count=1 -v`

## Local workflow expectations
- Minimum before finishing a code change:
  - `gofmt -w .`
  - `go test ./...`
- Preferred parity with CI:
  - `go test -race -count=1 -v ./...`
- If dependencies were added/removed:
  - `go mod tidy`

## Code style guidelines (Go)

### Imports
- Use standard grouping: stdlib, blank line, third-party/local.
- Avoid dot imports.
- Use blank imports only for explicit side effects.
- Remove unused imports before finalizing.

### Formatting and structure
- Always rely on `gofmt`; do not manually align code.
- Keep functions focused and short where practical.
- Prefer early returns over deep nesting.
- Keep package APIs cohesive and file layout predictable.
- Add comments only when intent is non-obvious.

### Types and interfaces
- Prefer concrete types by default.
- Introduce interfaces at call sites when they improve tests/decoupling.
- Use pointer receivers for mutable or large structs.
- Use value receivers for small immutable structs where clear.
- Keep zero values useful when practical.

### Naming conventions
- Package names: short, lowercase, no underscores.
- Exported identifiers: `CamelCase`.
- Unexported identifiers: `lowerCamelCase`.
- Test functions: `TestXxx`; subtests should use descriptive strings.
- Avoid stutter and redundant suffixes in type names.

### Error handling
- Return errors instead of logging-and-swallowing.
- Wrap errors with context using `%w`:
  - `fmt.Errorf("read config: %w", err)`
- Use `errors.Is` / `errors.As` compatible wrapping.
- Use `errors.Join` when aggregating validation failures.
- Keep user-facing errors actionable and specific.

### Context and concurrency
- Accept `context.Context` as first arg for blocking/cancellable work.
- Propagate context down call chains; do not store it in structs.
- Guard shared mutable state with clear synchronization.
- Avoid goroutine leaks; define clear shutdown/lifetime behavior.

### Testing conventions
- Prefer table-driven tests for multi-case logic.
- Use `t.Run` for scenario isolation.
- Use `t.Parallel()` only when deterministic and safe.
- In this repo, `require` is preferred for setup/preconditions and `assert` for value checks.
- Cover happy paths, edge cases, and validation failures.

## Templates and generated output
- Do not patch generated artifacts when template/source logic is the root cause.
- For generation behavior changes, edit sources under `internal/generator`.
- Keep `generated/` edits intentional, minimal, and directly related to the task.

## Dependency and module hygiene
- Add dependencies only when necessary.
- Keep `go.mod` / `go.sum` changes tightly scoped.
- Run `go mod tidy` after dependency changes.

## Safety and repo hygiene
- Never commit secrets, credentials, or tokens.
- Validate and sanitize path/name inputs used by generation code.
- Avoid destructive git operations unless explicitly requested.
- Do not reformat unrelated files as drive-by cleanup.

## Documentation expectations
- Update `README.md` when CLI flags or workflows change.
- Keep command examples copy-pastable and current.

## When unsure
- Follow nearby existing patterns first.
