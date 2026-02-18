# MCPGEN

A cookiecutter for MCP servers.

It asks a few questions, generates a clean MCP server in Go, runs checks, and leaves you with code you can extend.

> ⚠️ **Note:** This project is under active development. Features, APIs, and configurations may change until the first stable release.

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

![MCPGEN Screenshot 1](https://github.com/user-attachments/assets/b964dd20-88e3-4d62-9fd2-0045c9197b9b)
![MCPGEN Screenshot 2](https://github.com/user-attachments/assets/abba7ef7-6ba0-4d55-9741-a8be1afa4144)
![MCPGEN Screenshot 3](https://github.com/user-attachments/assets/403cfb5c-693f-49ba-89fb-32c5cd182f83)
![MCPGEN Screenshot 4](https://github.com/user-attachments/assets/b2f265a3-9d36-4525-9bc9-99f9f2ff35fd)

## TODO

 - unit tests
 - package API
 - generate middleware
 - generate http wrapper
