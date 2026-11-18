.PHONY: build prod prod-windows toolchain install test-windows test-lsp ext install-ext run clean version testgen test test-x64 bench
APP=wisp
DIST=dist/wisp
STAGE=dist/wisp-tmp

VERSION = $(shell go run ./cmd/main version --short)
NAME = wisp-$(VERSION)-linux-amd64
WINNAME = wisp-$(VERSION)-windows-amd64

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
	if [ -d dependencies/toolchain/linux-amd64 ]; then mkdir -p $(STAGE)/$(APP)/toolchain && cp -a dependencies/toolchain/linux-amd64/. $(STAGE)/$(APP)/toolchain; fi

	tar -C $(STAGE) -czf $(DIST)/$(NAME).tar.gz $(APP)

	rm -rf $(STAGE)

toolchain:
	scripts/fetch-toolchain.sh all

prod-windows:
	rm -f $(DIST)/$(WINNAME).zip
	rm -rf $(STAGE)
	mkdir -p $(DIST) $(STAGE)/$(APP)/bin
	GOOS=windows GOARCH=amd64 go build -o $(STAGE)/$(APP)/bin/$(APP).exe ./cmd/main
	cp -r std $(STAGE)/$(APP)
	if [ -d dependencies/toolchain/windows-amd64 ]; then mkdir -p $(STAGE)/$(APP)/toolchain && cp -a dependencies/toolchain/windows-amd64/. $(STAGE)/$(APP)/toolchain; fi
	cd $(STAGE) && zip -qr $(CURDIR)/$(DIST)/$(WINNAME).zip $(APP)
	rm -rf $(STAGE)

install: prod
	sudo rm -rf /usr/local/$(APP)
	sudo tar -C /usr/local -xzf $(DIST)/$(NAME).tar.gz

ext:
	mkdir -p dist/vscode
	cd vscode && npx @vscode/vsce package
	mv vscode/*.vsix dist/vscode

install-ext: ext
	code --install-extension "dist/vscode/$$(ls -t dist/vscode/*.vsix | head -n 1 | xargs basename)"

run:
	go run ./cmd/main

testgen:
	go run ./tests/gen

test-x64: build testgen
	WISP=bin/$(APP) tests/run.sh linux $(FILTER)

test-windows: build testgen
	WISP_TOOLCHAIN=$(CURDIR)/dependencies/toolchain/linux-amd64/bin WISP=bin/$(APP) tests/run.sh windows $(FILTER)

test-lsp:
	go test -count=1 ./tests/lsp

test: test-x64 test-lsp

bench:
	python3 benchmark/bench.py $(ARGS)

clean:
	rm -rf bin
	rm -rf dist
