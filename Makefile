# SPDX-FileCopyrightText: 2018 M. Shulhan <ms@kilabit.info>
#
# SPDX-License-Identifier: GPL-3.0-or-later

.PHONY: all lint serve-doc
all: build lint test

embed:
	go run ./internal/cmd/memfs

.PHONY: build-wui
build-wui:
	tsc --project _wui

.PHONY: build
build: embed
	go build -o ./_bin/ ./cmd/...

##---- Run all tests and generate coverage as HTML.

COVER_OUT:=cover.out
COVER_HTML:=cover.html

.PHONY: test
test:
	CGO_ENABLED=1 go test -failfast -timeout=1m -race \
		-coverprofile=$(COVER_OUT) ./...
	go tool cover -html=$(COVER_OUT) -o $(COVER_HTML)

.PHONY: test-integration
test-integration:
	CGO_ENABLED=1 go test -failfast -timeout=1m -race \
		-coverprofile=$(COVER_OUT) -integration ./...
	go tool cover -html=$(COVER_OUT) -o $(COVER_HTML)

##----

lint:
	go run ./internal/cmd/gocheck ./...
	go vet ./...

install:
	go install -v ./cmd/haminer

serve-doc:
	ciigo serve _doc


##---- Run haminer for local development.

.PHONY: dev
dev:
	go run ./cmd/haminer -dev \
		-config _ops/haminer-test/mkosi.extra/etc/haminer.conf


##---- Initialize local development by creating image using mkosi.
## NOTE: only works on GNU/Linux OS.

MACHINE_NAME:=haminer-test

.PHONY: haminer-dummy-backend
haminer-dummy-backend:
	go build -o ./_bin/ ./internal/cmd/...

.PHONY: init-local-dev
init-local-dev: build haminer-dummy-backend
	@echo ">>> Stopping container ..."
	-sudo machinectl poweroff $(MACHINE_NAME)

	cp -f _bin/* _ops/haminer-test/mkosi.extra/data/haminer/bin/

	@echo ">>> Building container $(MACHINE_NAME) ..."
	sudo mkosi --directory=_ops/$(MACHINE_NAME)/ --force build

	sudo machinectl --force import-fs _ops/$(MACHINE_NAME)/$(MACHINE_NAME)
	sudo machinectl start $(MACHINE_NAME)

	## Once the container is imported, we can enable and run them any
	## time without rebuilding again.
