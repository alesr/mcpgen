# syntax=docker/dockerfile:1
# Multi-stage build for the OCI artifact referenced by server.json.
FROM golang:1.26 AS build
WORKDIR /src
COPY . .
RUN go mod tidy && CGO_ENABLED=0 go build -o /out/server ./cmd/bare

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/server /server
EXPOSE 8080
ENTRYPOINT ["/server"]
