# MCPGEN
[![Go](https://img.shields.io/badge/Go-1.25-blue?logo=go&logoColor=white)](https://go.dev)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

A cookiecutter for bootstrapping MCP servers in Go.

Answer a few questions → get a clean, tested, inspector-ready MCP server in seconds.

## Requirements

- Go 1.25
- Node.js (for inspector checks)
- `npx`

## Quick start

```sh
git clone https://github.com/alesr/mcpgen.git

cd mcpgen

go run .

# or
make gen
```

You’ll get a small interactive flow. Press Enter to accept defaults.

## What it generates

- `cmd/<server>/main.go` – entrypoint
- `internal/mcpapp/` – server wiring + handlers
- `internal/mcpapp/tools/handlers/` – stub tool handler(s)
- `internal/mcpapp/prompts/` – stub prompts
- `internal/mcpapp/resources/` – stub resources
- `internal/mcpapp/stubs/` – shared stub responses
- unit tests for tools/prompts/resources (only for enabled features)

## What happens after generation

MCPGEN runs a quick sanity pipeline in this order:

- `go mod tidy`
- `gofmt -w .`
- `go vet ./...`
- `go test ./...`

If everything passes, it runs the inspector checks for any enabled features.

## Customize

Start by editing:

- `internal/mcpapp/tools/handlers/handlers.go`
- `internal/mcpapp/prompts/prompts.go`
- `internal/mcpapp/resources/resources.go`

Replace the stub logic with your real implementation.

## Notes

- Default transport is **stdio** (best for local tools).
- You can switch to HTTP in the setup flow if you want a networked server.

## Screenshots

<div style="display: flex; justify-content: center;">
    <img src="https://github.com/user-attachments/assets/d572ffc7-aa04-48d4-a047-62f38d0c16d2" width="400" />
</div>
<div style="display: flex; flex-wrap: wrap; gap: 10px; justify-content: center;">
    <img src="https://github.com/user-attachments/assets/4bb1e652-6a9c-4bd1-9abf-5637bbf2b3f9" width="300" style="margin-bottom: 10px;" />
    <img src="https://github.com/user-attachments/assets/dab94556-37a0-4adf-b41c-a1ef409d3e2a" width="300" style="margin-bottom: 10px;" />
</div>
<div style="display: flex; flex-wrap: wrap; gap: 10px; justify-content: center;">
    <img src="https://github.com/user-attachments/assets/874b8fae-a131-41b8-ab90-d382d46ce84d" width="300" style="margin-bottom: 10px;" />
</div>

## TODO

 - unit tests
 - package API
 - generate middleware
 - generate http wrapper
