# mcpgen

[![Go](https://img.shields.io/badge/Go-1.26-blue?logo=go&logoColor=white)](https://go.dev)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

**Deterministic Go codegen for spec-compliant MCP servers — bootstrap in seconds, extend forever, verify until green.**

mcpgen generates MCP servers in Go on the official
[`go-sdk`](https://github.com/modelcontextprotocol/go-sdk) v1.8.0, targeting
MCP spec **2026-07-28**. It is built for agent-driven execution: every command
speaks a JSON envelope with semantic exit codes, so a coding agent can loop
`new → check → fix` until green without human help. Humans get an interactive
TUI.

## Install

```sh
go install github.com/alesr/mcpgen@latest
```

Requires Go 1.26+. Node.js/npx are optional (used for the conformance suite and
Inspector).

## The agent loop (canonical usage)

```bash
mcpgen new ./weather --name weather --module github.com/acme/weather \
  --transport http --tool tools/forecast.json --json
mcpgen check ./weather --json        # loop until exit 0
# ...agent implements handler logic in the user-owned files...
mcpgen add tool tools/health.json --dir ./weather --json
mcpgen check ./weather --json        # exit 0 = green
```

`mcpgen schema` emits the full machine-readable spec (commands, flags, exit
codes, envelope shape, the `mcpgen.json` JSON Schema) — the self-discovery
entry point, no docs needed.

## Commands

| Command | What it does |
|---|---|
| `mcpgen new [dir] [flags]` | Generate a project. No flags + TTY → interactive TUI. |
| `mcpgen add tool\|prompt\|resource <spec.json>` | Add a capability to an existing project (AST-aware marker insertion; never touches user files). |
| `mcpgen check [dir]` | Verify: config → tidy → fmt → vet → build → test → conformance. Exit 0 = green. |
| `mcpgen schema` | Machine-readable spec of the CLI contract. |
| `mcpgen version` | Print version (`--json` for the envelope). |
| `mcpgen mcp` | Run mcpgen itself as an MCP server (stdio): `mcpgen_new`, `mcpgen_add`, `mcpgen_check`, `mcpgen_schema`. |
| `mcpgen` | Interactive mode on a TTY. |

### Flags for `new`

`--name --module --version --transport stdio|http --port --protocol compat|latest-only`
`--tool spec.json ... --prompt spec.json ... --resource spec.json ...` (repeatable, `name=file.json` allowed)
`--features tasks,skills,oauth` `--no-stubs` `--json` `--dry-run` `--force` `--no-check`

`--dry-run` prints the full plan (file tree, create/overwrite/skip, owners —
including `mcpgen.json` and `.mcpgen/manifest.json`) and writes nothing.

### Removing a capability

`mcpgen.json` is the single source of truth. To remove a tool/prompt/resource:

1. Delete its entry from `mcpgen.json`.
2. Delete its user-owned files, e.g. `internal/mcpapp/tools/handlers/<tool>.go`
   and `<tool>_test.go` (delete these **before** regenerating — a regen only
   prunes manifest entries for files that are already gone).
3. Run `mcpgen new <dir> --force --no-check` (or `mcpgen check <dir>` after).

The registration is removed by the regen; step 2 takes care of the orphaned
handler files and their manifest entries.

## The JSON envelope

Every command with `--json` writes exactly one JSON document to stdout;
progress goes to stderr. Errors are always structured, even without `--json`.

```json
{
  "ok": true,
  "result": { "dir": "./weather", "files_written": ["cmd/weather/main.go"], "files_skipped": [], "next_actions": null },
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

## Exit codes

| Code | Meaning |
|---|---|
| 0 | success |
| 1 | general failure (build/test/conformance failed) |
| 2 | usage error (bad flags/args, unknown config version) |
| 3 | target dir exists without `--force` / not an mcpgen project |
| 4 | permission denied |
| 5 | conflict (duplicate id, markers missing) |
| 6 | prerequisite missing (go toolchain, node/npx) |

## What it generates

```
cmd/<name>/main.go                        entrypoint (mcpgen-owned)
internal/mcpapp/mcpapp.go                 server wiring: Stateless HTTP, SupportedProtocolVersions, SetCacheable
internal/mcpapp/tools.go                  registration with // mcpgen:begin/end tools markers (mcpgen-owned)
internal/mcpapp/tools/handlers/<tool>.go one user-owned handler + test per tool
internal/mcpapp/prompts/<prompt>.go       same pattern (user-owned)
internal/mcpapp/resources/<resource>.go   same pattern (user-owned)
mcpgen.json                               single source of truth (version 1)
.mcpgen/manifest.json                     ownership + sha256 manifest
Makefile, .github/workflows/ci.yml        CI incl. conformance gate
server.json                               MCP Registry descriptor (OCI package block)
Dockerfile                                OCI image build
AGENTS.md, README.md
```

Ownership rules: mcpgen-owned files carry `DO NOT EDIT region: mcpgen-owned`;
handler/prompt/resource files are user-owned and never overwritten (tracked via
the manifest). Re-running `mcpgen new --force` on a project regenerates
mcpgen-owned files but skips anything modified since generation. `mcpgen add`
inserts registrations into the marker regions using `go/parser` + `go/format`,
never regex.

## Verification

`mcpgen check` runs, in order: `config` (validate mcpgen.json), `tidy` (incl.
SDK pin check), `fmt`, `vet`, `build`, `test -race`, `conformance` (official
`@modelcontextprotocol/conformance` suite against a loopback instance when
node/npx are present and transport is http; protocol-level scenarios gate,
fixture scenarios are reported as n/a), and optionally `inspector` (Inspector
v2 CLI smoke). Every stage is reported in the envelope; failures carry a `fix`
with the first error's file:line.

## MCP server mode

```sh
mcpgen mcp   # stdio MCP server exposing mcpgen_new / mcpgen_add / mcpgen_check / mcpgen_schema
```

Same code path as the CLI — the two can never diverge.

## Development

```sh
make check    # tidy, fmt, vet, test -race
make golden   # regenerate golden snapshots (UPDATE_GOLDEN=1)
```

CI runs unit + golden tests, then the generate-and-verify matrix: each template
permutation is generated into a temp dir and pushed through `mcpgen check`,
including the conformance suite for http — catching SDK/spec drift the day it
lands.

## License

MIT
