## SPDX-FileCopyrightText: 2018 M. Shulhan <ms@kilabit.info>
## SPDX-License-Identifier: GPL-3.0-or-later

.PHONY: all build test lint install serve-doc
all: install

build:
	go build -v ./cmd/haminer

test:
	CGO_ENABLED=1 go test -race ./...

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
