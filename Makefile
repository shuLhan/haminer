## SPDX-FileCopyrightText: 2018 M. Shulhan <ms@kilabit.info>
## SPDX-License-Identifier: GPL-3.0-or-later

.PHONY: all build lint install serve-doc
all: install

build:
	go build -v ./cmd/haminer

## Run all tests and generate coverage as HTML.

COVER_OUT:=cover.out
COVER_HTML:=cover.html

.PHONY: test
test:
	CGO_ENABLED=1 go test -failfast -timeout=1m -race \
		-coverprofile=$(COVER_OUT) ./...
	go tool cover -html=$(COVER_OUT) -o $(COVER_HTML)

lint:
	-fieldalignment ./...
	-shadow ./...
	-golangci-lint run \
		--presets bugs,metalinter,performance,unused \
		--disable exhaustive \
		--disable musttag \
		--disable bodyclose \
		./...

install: build test lint
	go install -v ./cmd/haminer

serve-doc:
	ciigo serve _doc
