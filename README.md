# mcpgen

[![CI](https://github.com/alesr/mcpgen/actions/workflows/ci.yaml/badge.svg)](https://github.com/alesr/mcpgen/actions/workflows/ci.yaml)
[![Go Reference](https://pkg.go.dev/badge/github.com/alesr/mcpgen.svg)](https://pkg.go.dev/github.com/alesr/mcpgen)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

mcpgen is a CLI that generates, extends, and verifies [MCP](https://modelcontextprotocol.io) (Model Context Protocol) servers in Go. Generated projects build on the official [`go-sdk`](https://github.com/modelcontextprotocol/go-sdk) v1.8.0 and target MCP spec **2026-07-28**.

Why it exists:

1. **Safe regeneration** — every generated file is `mcpgen`-owned or `user`-owned, tracked with a sha256 in `.mcpgen/manifest.json`; regen never overwrites your code.
2. **Built-in verification** — `mcpgen check` runs config validation, tidy, fmt, vet, build, `go test -race`, and the official MCP conformance suite; exit 0 means green.
3. **Agent-friendly** — every command emits one JSON envelope (`--json`) with stable exit codes and `fix` hints, so agents loop `new → check → fix` until exit 0.

## Install

Requires Go 1.26+. Node.js/npx are optional (conformance suite only).

```sh
go install github.com/alesr/mcpgen@latest
```

Release binaries for Linux, macOS, and Windows are on the [Releases page](https://github.com/alesr/mcpgen/releases) (tags `vX.Y.Z`, built by goreleaser).

## Quickstart (60 seconds)

```bash
# 1. Generate an HTTP MCP server (runs the check pipeline after writing)
mcpgen new ./weather --name weather --module example.com/weather --transport http --json

# 2. Verify until green (exit 0 = every stage passed)
mcpgen check ./weather --json

# 3. Run it (loopback by default) and probe it
cd weather && go run ./cmd/weather &
curl -s http://127.0.0.1:8080/healthz        # -> ok

# 4. Extend at any time; your files are never touched
mcpgen add tool tools/health.json --dir ./weather --json
mcpgen check ./weather --json
```

Real check output (trimmed):

```text
[passed] config — mcpgen.json valid (version 1, 1 tools, 1 prompts, 1 resources)
[passed] serverjson — server.json valid (MCP Registry descriptor)
[passed] tidy — go mod tidy
[passed] fmt — gofmt clean
[passed] vet — go vet clean
[passed] build — go build ok
[passed] test — go test -race ok
[passed] conformance — conformance passed against http://127.0.0.1:61255/mcp: 8/30 checks passed; ...
```

`mcpgen` with no arguments on a terminal runs an interactive flow. `mcpgen schema` prints the full machine-readable contract (commands, flags, exit codes, envelope, `mcpgen.json` JSON Schema) — the self-discovery entry point for agents.

## What you get

A fresh `mcpgen new ./weather --name weather --module example.com/weather` produces (verified against a real generation):

```text
weather/
├── cmd/weather/main.go                  entrypoint (stdio, or HTTP server with -addr)
├── internal/mcpapp/
│   ├── mcpapp.go                        server wiring (stateless HTTP, protocol versions)
│   ├── tools.go / prompts.go / resources.go   registration + // mcpgen:begin/end markers
│   ├── tools/handlers/greet.go (+_test.go)   user-owned: your implementation
│   ├── prompts/welcome.go (+_test.go)        user-owned
│   └── resources/readme.go (+_test.go)       user-owned
├── mcpgen.json                          single source of truth (version 1)
├── .mcpgen/manifest.json                owner + sha256 per file
├── server.json, Dockerfile, Makefile, .github/workflows/ci.yml, go.mod (pins go-sdk v1.8.0)
```

Tool handlers are typed (`mcp.AddTool[In, Out]`): `<Tool>Input`/`<Tool>Output` structs are derived from the tool's JSON Schema, and the SDK validates arguments before your handler runs.

## Features

- **Commands**: `new`, `add tool|prompt|resource`, `check`, `schema`, `version`, `mcp` (mcpgen itself as a stdio MCP server with `mcpgen_new`/`mcpgen_add`/`mcpgen_check`/`mcpgen_schema`), plus an interactive TUI.
- **Transports**: `stdio` (default) and stateless streamable `http` on `/mcp` with `/healthz`; loopback bind `127.0.0.1:<port>` by default, cross-site `Origin` rejected (403).
- **Protocol modes**: `compat` (default, negotiates down to legacy versions) or `latest-only` (2026-07-28 only).
- **Opt-in features** (`--features tasks,skills,oauth`): tasks stub, skills served from `skills/`, commented OAuth wiring for 2026-07-28 authorization.
- **Machine interface**: `--json` envelope (`ok`, `result`, `error{stage,code,message,fix,command,exit_code}`, `next_actions`); exit codes 0 ok · 1 failure · 2 usage · 3 exists/not-project · 4 permission · 5 conflict · 6 prerequisite missing.

## Status and limitations

- The conformance gate runs only the scenarios the official suite can run against a generated server: protocol-level scenarios gate; capability-fixture scenarios (which need well-known test tools) are reported as non-gating. A default server shows `8/30 checks passed` with 22 non-gating fixture failures — that is expected, not a bug.
- The generated HTTP server has **no authentication** unless you enable and wire the `oauth` feature; it binds to loopback and rejects cross-site Origins, which is the default protection.
- `mcpgen.json` supports config version 1 only.
- `mcpgen` v1.0.0; generated servers pin `go-sdk v1.8.0` (the `tidy` check stage enforces the pin).

## Docs and more

- **Documentation site**: <https://alesr.github.io/mcpgen/> — quickstart, end-to-end tutorial, concepts (ownership model, marker regions, security), how-to guides, and a generated reference (CLI, `mcpgen.json`, exit codes, file tree).
- [`AGENTS.md`](AGENTS.md) — repo guidance for coding agents; [`SKILL.md`](SKILL.md) — the agent loop as a skill.
- [Contributing](https://alesr.github.io/mcpgen/project/contributing/) — architecture, local checks (`make check`), releases.
- Development: `make check` (tidy, fmt, vet, race tests), `make golden`, `make docs-gen` / `make docs-build` for the docs site.

## License

MIT — see [LICENSE](LICENSE).
