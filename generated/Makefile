SERVER := example_mcp

.PHONY: build test vet fmt tidy check run conformance clean

build:
	go build ./...

test:
	go test -race ./...

vet:
	go vet ./...

fmt:
	gofmt -w .

tidy:
	go mod tidy

check: tidy fmt vet build test

run:
	go run ./cmd/$(SERVER)

# Runs the official MCP conformance suite against a local instance.
# Requires node/npx. Only meaningful for the http transport.
conformance: build
	@go build -o dist/$(SERVER) ./cmd/$(SERVER)
	@dist/$(SERVER) -addr 127.0.0.1:18080 & echo $$! > dist/server.pid
	@sleep 1
	@npx --yes @modelcontextprotocol/conformance server --url http://127.0.0.1:18080/mcp; status=$$?; kill $$(cat dist/server.pid) 2>/dev/null; exit $$status

clean:
	rm -rf dist
