SHELL := /bin/sh

.DEFAULT_GOAL := help

.PHONY: help install-web format format-check lint typecheck test test-race build build-web build-go verify clean

help:
	@printf '%s\n' 'Dock-Weaver development targets:'
	@printf '%s\n' '  make install-web  Install pinned frontend dependencies'
	@printf '%s\n' '  make format       Format Go source'
	@printf '%s\n' '  make verify       Run formatting, lint, tests, and builds'

install-web:
	pnpm --dir web install --frozen-lockfile

format:
	gofmt -w cmd internal

format-check:
	@test -z "$$(gofmt -l cmd internal)"

lint:
	go vet ./...
	npm --prefix web run lint
	bash -n install.sh

typecheck:
	npm --prefix web run typecheck

test:
	go test ./...
	npm --prefix web run test

test-race:
	go test -race ./...

build-web:
	npm --prefix web run build

build-go:
	mkdir -p bin
	go build -o bin/dock-weaver ./cmd/dock-weaver

build:
	npm --prefix web run build
	./scripts/sync-web-assets.sh
	mkdir -p bin
	go build -o bin/dock-weaver ./cmd/dock-weaver

verify: format-check lint typecheck test test-race build

clean:
	go clean
