# Build mcpgen and run it as a stdio MCP server.
FROM golang:1.26 AS build
ENV CGO_ENABLED=0
RUN go install github.com/alesr/mcpgen@v0.3.1

FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates \
    && rm -rf /var/lib/apt/lists/*
COPY --from=build /go/bin/mcpgen /usr/local/bin/mcpgen
ENTRYPOINT ["mcpgen", "mcp"]
