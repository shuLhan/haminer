.PHONY: all build install

all: install

build:
	go build -v ./cmd/haminer

test:
	CGO_ENABLED=1 go test -race ./...

lint:
	-golangci-lint run ./...

install: build test lint
	go install -v ./cmd/haminer
