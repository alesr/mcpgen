.PHONY: build
build:
	go build ./...

.PHONY: fmt
fmt:
	gofmt -w .

.PHONY: vet
vet:
	go vet ./...

.PHONY: test
test:
	go test -count=1 -race -v ./...

.PHONY: tidy
tidy:
	go mod tidy

.PHONY: gen
gen:
	@go run .

.PHONY: check
check: tidy fmt vet test

.PHONY: golden
golden:
	UPDATE_GOLDEN=1 go test ./internal/generator -run '^TestGolden$' -count=1
