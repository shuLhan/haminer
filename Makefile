## SPDX-FileCopyrightText: 2018 M. Shulhan <ms@kilabit.info>
## SPDX-License-Identifier: GPL-3.0-or-later

.PHONY: all build lint install serve-doc
all: install

build:
	go build -o ./_bin/ ./cmd/...

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

## Initialize local development by creating image using mkosi.
## NOTE: only works on GNU/Linux OS.

MACHINE_NAME:=haminer-test

.PHONY: init-local-dev
init-local-dev:
	@echo ">>> Stopping container ..."
	-sudo machinectl poweroff $(MACHINE_NAME)

	@echo ">>> Building container $(MACHINE_NAME) ..."
	sudo mkosi --directory=_ops/$(MACHINE_NAME)/ --force build

	sudo machinectl --force import-tar _ops/$(MACHINE_NAME)/$(MACHINE_NAME)
	sudo machinectl start $(MACHINE_NAME)

	## Once the container is imported, we can enable and run them any
	## time without rebuilding again.
