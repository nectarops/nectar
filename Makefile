SHELL := /bin/sh
SHELLCHECK_IMAGE := koalaman/shellcheck-alpine@sha256:c82fe42504fbc9fc68f15d36638e5ee2324ebb8b94e96a3c4e395bf361c49183
SHFMT_IMAGE := mvdan/shfmt@sha256:307d265ffd25ce832899ae17c93ed5062fc3375c514bba8f52cbf52792735c4d

.DEFAULT_GOAL := help

.PHONY: help dev install-web format format-check lint typecheck test test-installer test-race build build-web build-go verify clean

help:
	@printf '%s\n' 'Nectar development targets:'
	@printf '%s\n' '  make dev          Build and start the local web application'
	@printf '%s\n' '  make install-web  Install pinned frontend dependencies'
	@printf '%s\n' '  make format       Format Go and installer Shell source'
	@printf '%s\n' '  make test-installer  Test installer distribution detection in containers'
	@printf '%s\n' '  make verify       Run formatting, lint, tests, and builds'

dev:
	./scripts/dev.sh

install-web:
	pnpm --dir web install --frozen-lockfile

format:
	gofmt -w cmd internal
	docker run --rm -v "$(CURDIR):/mnt" $(SHFMT_IMAGE) \
		-w -i 2 -ci /mnt/install.sh /mnt/test/installer/dry_run.sh

format-check:
	@test -z "$$(gofmt -l cmd internal)"
	docker run --rm -v "$(CURDIR):/mnt:ro" $(SHFMT_IMAGE) \
		-d -i 2 -ci /mnt/install.sh /mnt/test/installer/dry_run.sh

lint:
	go vet ./...
	npm --prefix web run lint
	bash -n install.sh
	sh -n scripts/dev.sh scripts/sync-web-assets.sh
	docker run --rm -v "$(CURDIR):/mnt:ro" $(SHELLCHECK_IMAGE) \
		shellcheck -x /mnt/install.sh /mnt/test/installer/dry_run.sh

typecheck:
	npm --prefix web run typecheck

test:
	go test ./...
	npm --prefix web run test
	./test/installer/dry_run.sh

test-installer:
	./test/installer/dry_run.sh

test-race:
	go test -race ./...

build-web:
	npm --prefix web run build

build-go:
	mkdir -p bin
	go build -o bin/nectar ./cmd/nectar

build:
	npm --prefix web run build
	./scripts/sync-web-assets.sh
	mkdir -p bin
	go build -o bin/nectar ./cmd/nectar

verify: format-check lint typecheck test test-race build

clean:
	go clean
