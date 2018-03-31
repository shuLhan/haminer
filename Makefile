.PHONE: all build install

all: install

build:
	go build -v ./cmd/haminer

lint:
	-gometalinter --sort=path --disable=maligned ./...

install: build lint
	go install -v ./cmd/haminer
