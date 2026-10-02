.PHONY: build prod run clean version sync-vscode check-version testgen test test-lua test-x64
APP=wisp

VERSION = $(shell go run ./cmd/main version --short)
NAME = wisp-$(VERSION)-linux-amd64

build:
	go build -o bin/$(APP) ./cmd/main

version:
	@go run ./cmd/main version

sync-vscode:
	sed -i 's/^    "version": ".*"/    "version": "$(VERSION)"/' vscode/package.json

check-version:
	@grep -q '^    "version": "$(VERSION)"' vscode/package.json || { echo "vscode/package.json is out of sync with $(VERSION), run: make sync-vscode"; exit 1; }

prod: check-version
	rm -rf dist/$(APP)
	mkdir -p dist/$(APP)

	cp -r bin dist/$(APP)
	cp -r std dist/$(APP)
	
	tar -C dist -czf dist/$(NAME).tar.gz $(APP)

	rm -rf dist/$(APP)

ext:
	mkdir -p dist/vscode
	cd vscode
	npx @vscode/vsce package
	cp *.vsix ../dist/vscode

run:
	go run ./cmd/main

testgen:
	go run ./tests/gen

test-lua: build testgen
	WISP=bin/$(APP) tests/run.sh lua $(FILTER)

test-x64: build testgen
	WISP=bin/$(APP) tests/run.sh linux_x64 $(FILTER)

test: test-lua test-x64

clean:
	rm -rf bin
	rm -rf dist
