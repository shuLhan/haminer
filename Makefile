## SPDX-FileCopyrightText: 2018 M. Shulhan <ms@kilabit.info>
## SPDX-License-Identifier: GPL-3.0-or-later

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
