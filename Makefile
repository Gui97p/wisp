.PHONY: build prod ext run clean version testgen test test-lua test-x64 test-interop
APP=wisp

VERSION = $(shell go run ./cmd/main version --short)
NAME = wisp-$(VERSION)-linux-amd64

build:
	go build -o bin/$(APP) ./cmd/main

version:
	@go run ./cmd/main version

prod:
	rm -rf dist/$(APP)
	mkdir -p dist/$(APP)

	cp -r bin dist/$(APP)
	cp -r std dist/$(APP)
	
	tar -C dist -czf dist/$(NAME).tar.gz $(APP)

	rm -rf dist/$(APP)

ext:
	mkdir -p dist/vscode
	cd vscode && npx @vscode/vsce package
	cp vscode/*.vsix dist/vscode

run:
	go run ./cmd/main

testgen:
	go run ./tests/gen

test-lua: build testgen
	WISP=bin/$(APP) tests/run.sh lua $(FILTER)

test-x64: build testgen
	WISP=bin/$(APP) tests/run.sh linux_x64 $(FILTER)

test-interop: build
	WISP=bin/$(APP) tests/interop/run.sh $(FILTER)

test: test-lua test-x64

clean:
	rm -rf bin
	rm -rf dist
