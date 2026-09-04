.PHONY: all build test coverage lint clean

BINARY_NAME=mailru-mcp-server

all: lint test build

build:
	go build -v -o $(BINARY_NAME) .

test:
	go test -v -race -cover ./internal/...

coverage:
	go test -v -race -coverprofile=coverage.txt -covermode=atomic ./internal/...
	go tool cover -func=coverage.txt

lint:
	go vet ./...

clean:
	rm -f $(BINARY_NAME) coverage.txt *.out
