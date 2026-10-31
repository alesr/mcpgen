---
name: mcpgen
description: "Deterministic Go codegen for spec-compliant MCP servers (MCP 2026-07-28, go-sdk v1.8.0). Bootstrap a server with `mcpgen new`, extend it idempotently with `mcpgen add`, and verify with `mcpgen check --json` until exit 0. Use when the user wants to create, extend, or verify a Go MCP server, or when an agent should drive MCP server development in a loop."
license: MIT
compatibility: "Requires the mcpgen CLI (go install github.com/alesr/mcpgen@latest), Go 1.26+, and optionally Node.js/npx for the conformance suite."
metadata:
  author: alesr
  version: "1.0.0"
---

# mcpgen — Go MCP server generator

mcpgen is the deterministic, spec-current codegen tool for Go MCP servers.
Run `mcpgen schema` first: it emits the machine-readable contract (commands,
flags, exit codes, JSON envelope, mcpgen.json schema).

## The loop (always follow this)

1. **Discover**: `mcpgen schema` (or read this skill).
2. **Bootstrap**: `mcpgen new <dir> --name <name> --module <go module> [--transport stdio|http] [--tool spec.json ...] --json`
   - Prefer `--json` and read the envelope: `ok`, `result`, `error`, `next_actions`.
   - `--dry-run` previews the file plan without writing.
3. **Implement**: edit the user-owned files (`internal/mcpapp/tools/handlers/<tool>.go`, prompts, resources). Never edit mcpgen-owned files or marker regions.
4. **Verify**: `mcpgen check <dir> --json` — loop until exit code 0. On failure, read `error.stage`, `error.code`, and `error.fix` (contains file:line of the first failure), fix, re-run.
5. **Extend**: `mcpgen add tool|prompt|resource <spec.json> --dir <dir> --json`, then implement the new stub, then `mcpgen check` again. Duplicate ids exit 5; never hand-edit `// mcpgen:begin/end` marker regions.

## Exit codes

0 ok · 1 failure (build/test/conformance) · 2 usage · 3 exists/not-project · 4 permission · 5 conflict · 6 prerequisite missing (go, node/npx).

## Envelope contract

stdout carries one JSON document (`--json`) or clean human text; logs go to
stderr. Errors are always structured. `next_actions` is always present on
success and recoverable failure — follow it.

## Spec targets

Generated servers pin `github.com/modelcontextprotocol/go-sdk v1.8.0` and MCP
spec 2026-07-28 (stateless streamable HTTP by default, `SupportedProtocolVersions`
and `SetCacheable` wired). Never downgrade the SDK pin by hand.

## MCP mode

Hosts without shell access can run `mcpgen mcp` (stdio) and call
`mcpgen_new`, `mcpgen_add`, `mcpgen_check`, `mcpgen_schema`.
