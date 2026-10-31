# mcpgen

[![Go](https://img.shields.io/badge/Go-1.26-blue?logo=go&logoColor=white)](https://go.dev)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

mcpgen is a CLI that generates, extends, and verifies [MCP](https://modelcontextprotocol.io) servers in Go. Generated projects build on the official [`go-sdk`](https://github.com/modelcontextprotocol/go-sdk) v1.8.0 and target MCP spec **2026-07-28**. Generation is deterministic — same `mcpgen.json` in, same project out — and an ownership manifest keeps your code safe when you regenerate.

Two surfaces, one code path: coding agents drive the CLI through a JSON envelope with semantic exit codes (`new → check → fix`, loop until exit 0), while humans get an interactive TUI and plain terminal output.

## Install

Requires Go 1.26+. Node.js/npx are optional — used only for the conformance suite and the Inspector smoke check.

```sh
go install github.com/alesr/mcpgen@latest
```

## Quickstart

```bash
# 1. Scaffold a server from spec files
mcpgen new ./weather --name weather --module github.com/acme/weather \
  --transport http --tool tools/forecast.json --json

# 2. Implement the handler stubs (user-owned files), then verify until green
mcpgen check ./weather --json        # exit 0 = every stage passed

# 3. Extend at any time; your files are never touched
mcpgen add tool tools/health.json --dir ./weather --json
mcpgen check ./weather --json
```

Run `mcpgen` with no arguments on a terminal for the interactive flow. `mcpgen schema` prints the full machine-readable contract (commands, flags, defaults, exit codes, envelope shape, and the `mcpgen.json` JSON Schema) — the self-discovery entry point, no docs needed.

## Commands

| Command | What it does |
|---|---|
| `mcpgen new [dir] [flags]` | Generate a project. No args on a TTY runs the interactive TUI. |
| `mcpgen add <tool\|prompt\|resource> <spec.json> [flags]` | Add a capability to an existing project. |
| `mcpgen check [dir] [flags]` | Run the verification pipeline. Exit 0 = green. |
| `mcpgen schema` | Emit the machine-readable spec of the CLI contract. |
| `mcpgen version [--json]` | Print the version. |
| `mcpgen mcp` | Run mcpgen itself as an MCP server over stdio. |
| `mcpgen` (no args, TTY) | Interactive mode. |

Flags-only invocation (`mcpgen --name weather ...`) is accepted as shorthand for `mcpgen new --name weather ...`.

### `mcpgen new`

| Flag | Default | Meaning |
|---|---|---|
| `--name` | `example-mcp` | Server name |
| `--module` | derived from name (`example.com/<name>`) | Go module path |
| `--version` | `0.1.0` | Server version (semver) |
| `--transport` | `stdio` | `stdio` or `http` (stateless streamable HTTP on `/mcp`, plus `/healthz`) |
| `--port` | `8080` | HTTP listen port when `--transport http` |
| `--protocol` | `compat` | `compat` (negotiate down to legacy versions) or `latest-only` (2026-07-28 only) |
| `--tool`, `--prompt`, `--resource` | — | Spec JSON file, repeatable; `name=file.json` overrides the spec's `id` |
| `--features` | — | Comma-separated opt-in features: `tasks`, `skills`, `oauth` |
| `--no-stubs` | off | Don't generate the default `greet` tool / `welcome` prompt / `readme` resource |
| `--dry-run` | off | Print the full plan (every file: create/overwrite/skip + owner) and write nothing |
| `--force` | off | Allow generating into a non-empty directory (regen) |
| `--no-check` | off | Skip the post-generation verification pipeline |
| `--json` | off | Emit the JSON envelope on stdout |

The output directory defaults to `./generated`. With no spec flags and no `--no-stubs`, you get one stub per capability kind to start from. After writing files, `new` runs the same pipeline as `mcpgen check` unless `--no-check` is set.

Re-running `new --force` on an existing project regenerates mcpgen-owned files, but only flags you pass explicitly override the existing `mcpgen.json` — everything else is preserved. `--features` are additive on regen; remove a feature by editing `mcpgen.json` directly.

### `mcpgen add`

```bash
mcpgen add tool tools/health.json --dir ./weather --json
```

One command does three things: the entry is appended to `mcpgen.json`, the user-owned implementation and test files are created, and the registration statement is inserted into the `// mcpgen:begin … / // mcpgen:end …` marker region of the matching registration file. Insertion goes through `go/parser` + `go/ast` + `go/format` — never regex — so the file stays valid and formatted. A duplicate id exits with code 5.

Flags: `--dir` (default `.`), `--json`, `--dry-run`, `--force` (overwrite an existing user-owned handler file), `--no-check`, `--conformance` / `--no-conformance`.

### `mcpgen check`

```bash
mcpgen check ./weather --json
```

Runs the pipeline in order and reports every stage in the envelope:

```
prerequisite → config → serverjson → tidy → fmt → vet → build → test → conformance → inspector*
```

- **prerequisite** — the Go toolchain is on `PATH` (else exit 6)
- **config** — `mcpgen.json` validates (version 1)
- **serverjson** — `server.json` is a valid MCP Registry descriptor
- **tidy** — `go mod tidy`, plus a check that `go.mod` still pins `go-sdk v1.8.0`
- **fmt / vet / build / test** — `gofmt -l`, `go vet`, `go build`, `go test -race`
- **conformance** — the official `@modelcontextprotocol/conformance` suite via `npx`, run against the built server on a loopback address. Auto-enabled when node/npx are present, transport is `http`, and protocol is `compat`. Protocol-level scenarios gate; capability-fixture scenarios (which require well-known test tools) are reported as n/a. Unparseable suite output fails closed. Skipped for stdio or `latest-only`; forcing it without node/npx exits 6.
- **inspector*** — optional (`--inspector`) Inspector v2 CLI smoke check; a debugging aid, not a gate

Flags: `--json`, `--conformance` / `--no-conformance`, `--inspector`, `--timeout` (default `10m`). Exit 0 means green; on failure `error.fix` in the envelope names the first failure with its `file:line`.

## The generated project

```
<dir>/
├── cmd/<name>/main.go                        mcpgen-owned — entrypoint (stdio, or http server with -addr)
├── internal/mcpapp/
│   ├── mcpapp.go                             mcpgen-owned — server wiring (stateless HTTP, SetCacheable, protocol versions)
│   ├── instructions.go                       mcpgen-owned
│   ├── tools.go / prompts.go / resources.go  mcpgen-owned — registration + // mcpgen:begin/end markers
│   ├── tasks.go / skills.go / oauth.go       mcpgen-owned — only with --features
│   ├── tools/handlers/doc.go + <tool>.go, _test.go   mcpgen-owned doc.go; user-owned — your implementation
│   ├── prompts/doc.go + <prompt>.go, _test.go         same pattern
│   └── resources/doc.go + <resource>.go, _test.go     same pattern
├── mcpgen.json                               mcpgen-owned — single source of truth (version 1)
├── .mcpgen/manifest.json                     mcpgen-owned — owner + sha256 per file
├── server.json                               MCP Registry descriptor (OCI package block)
├── Dockerfile                                OCI image build
├── Makefile, .github/workflows/ci.yml        CI with a conformance gate
├── go.mod                                    pins go-sdk v1.8.0
└── AGENTS.md, README.md, .gitignore
```

Ownership rules:

- mcpgen-owned files carry a `DO NOT EDIT region: mcpgen-owned` header. User-owned handler/prompt/resource files are never overwritten (the only exception is `add --force`).
- `.mcpgen/manifest.json` records an owner and sha256 for every file. On regen, mcpgen-owned files you modified since generation are skipped (hash mismatch), and untracked files are never clobbered.
- Never hand-edit the `// mcpgen:begin/end` marker regions — use `mcpgen add`. If a marker is missing, `add` exits 5 with a fix hint.

### `mcpgen.json`

Abridged shape (the full JSON Schema is embedded in `mcpgen schema`):

```json
{
  "version": 1,
  "server": { "name": "weather", "version": "0.1.0", "module": "github.com/acme/weather", "description": "…", "instructions": "…" },
  "transport": { "type": "http", "httpPort": 8080 },
  "protocol": "compat",
  "tools": [{ "id": "forecast", "title": "Forecast", "inputSchema": { "type": "object" }, "readOnly": true }],
  "prompts": [{ "id": "welcome", "arguments": [{ "name": "name", "required": true }] }],
  "resources": [{ "id": "readme", "uri": "file:///readme", "mimeType": "text/plain", "text": "…" }],
  "features": { "tasks": true, "skills": false, "oauth": false }
}
```

To remove a capability: delete its entry from `mcpgen.json`, delete its user-owned files **first** (e.g. `internal/mcpapp/tools/handlers/<tool>.go` and its test), then regenerate with `mcpgen new <dir> --force --no-check`. The regen removes the registration and prunes the orphaned manifest entries.

## JSON contract for agents

Every command with `--json` writes exactly one JSON document to stdout; progress and logs go to stderr. Errors are always structured, even without `--json`.

```json
{
  "ok": true,
  "result": { "dir": "./weather", "files_written": ["cmd/weather/main.go"], "files_skipped": [] },
  "error": null,
  "next_actions": ["cd weather", "mcpgen check ./weather --json"]
}
```

```json
{
  "ok": false,
  "result": null,
  "error": {
    "stage": "test",
    "code": "TESTS_FAILED",
    "message": "1 of 3 tests failed",
    "fix": "first failure: TestGreet (greet_test.go:30); run `go test -race ./...` in ./weather",
    "command": "go test -race ./...",
    "exit_code": 1
  },
  "next_actions": ["fix the failing test, then re-run mcpgen check ./weather --json"]
}
```

Exit codes are stable and safe to branch on:

| Code | Meaning |
|---|---|
| 0 | Success |
| 1 | General failure (build/test/conformance failed) |
| 2 | Usage error (bad flags/args, unknown config version) |
| 3 | Target dir exists without `--force`, or dir is not an mcpgen project |
| 4 | Permission denied |
| 5 | Conflict (duplicate id, markers missing or edited) |
| 6 | Prerequisite missing (Go toolchain, node/npx) |

## MCP server mode

```sh
mcpgen mcp
```

Runs mcpgen itself as a stdio MCP server exposing `mcpgen_new`, `mcpgen_add`, `mcpgen_check`, and `mcpgen_schema` — for hosts without shell access. It serves the same operations package the CLI uses, so the two interfaces cannot diverge. Client config:

```json
{ "mcpServers": { "mcpgen": { "command": "mcpgen", "args": ["mcp"] } } }
```

## Development

```sh
make check     # go mod tidy, gofmt, go vet, go test -race
make golden    # regenerate golden snapshots (UPDATE_GOLDEN=1)
```

CI runs unit + golden tests with `-race` and coverage, fmt/vet/gomodernize gates, golangci-lint, and a goreleaser snapshot — then a generate-and-verify matrix (`stdio-default`, `http-full`, `http-latest-only` with `tasks,skills,oauth`, `stdio-empty`): each permutation is generated into a temp dir and pushed through `mcpgen check`, including the conformance suite for http, so SDK/spec drift fails CI the day it lands. Releases are goreleaser-built binaries for linux/darwin/windows, tagged `vX.Y.Z` (the version is stamped via `-ldflags`). The `generated/` directory is a checked-in sample project regenerated with `mcpgen new`.

## License

MIT
