.PHONY: all build install

all: install

build:
	go build -v ./cmd/haminer

lint:
	-golangci-lint run --enable-all ./...

install: build lint
	go install -v ./cmd/haminer
