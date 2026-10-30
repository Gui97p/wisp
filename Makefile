.PHONY: build prod install ext run clean version testgen test test-cc test-x64
APP=wisp
DIST=dist/wisp
STAGE=dist/wisp-tmp

VERSION = $(shell go run ./cmd/main version --short)
NAME = wisp-$(VERSION)-linux-amd64

build:
	go build -o bin/$(APP) ./cmd/main

version:
	@go run ./cmd/main version

prod: build
	rm -f $(DIST)/$(NAME).tar.gz
	rm -rf $(STAGE)
	mkdir -p $(DIST) $(STAGE)/$(APP)

	cp -r bin $(STAGE)/$(APP)
	cp -r std $(STAGE)/$(APP)

	tar -C $(STAGE) -czf $(DIST)/$(NAME).tar.gz $(APP)

	rm -rf $(STAGE)

install: prod
	sudo rm -rf /usr/local/$(APP)
	sudo tar -C /usr/local -xzf $(DIST)/$(NAME).tar.gz

ext:
	mkdir -p dist/vscode
	cd vscode && npx @vscode/vsce package
	mv vscode/*.vsix dist/vscode

run:
	go run ./cmd/main

testgen:
	go run ./tests/gen

test-cc: build testgen
	WISP=bin/$(APP) tests/run.sh cc $(FILTER)

test-x64: build testgen
	WISP=bin/$(APP) tests/run.sh linux_x64 $(FILTER)

test: test-cc test-x64

clean:
	rm -rf bin
	rm -rf dist
