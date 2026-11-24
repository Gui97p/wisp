.PHONY: build prod prod-windows release check-toolchain-linux check-toolchain-windows toolchain install test-windows test-lsp test-cli test-fmt ext install-ext run clean version testgen test test-x64 bench
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

prod: build check-toolchain-linux
	rm -f $(DIST)/$(NAME).tar.gz
	rm -rf $(STAGE)
	mkdir -p $(DIST) $(STAGE)/$(APP)

	cp -r bin $(STAGE)/$(APP)
	cp -r std $(STAGE)/$(APP)
	mkdir -p $(STAGE)/$(APP)/toolchain && cp -a dependencies/toolchain/linux-amd64/. $(STAGE)/$(APP)/toolchain

	tar -C $(STAGE) -czf $(DIST)/$(NAME).tar.gz $(APP)

	rm -rf $(STAGE)

toolchain:
	scripts/fetch-toolchain.sh all

check-toolchain-linux:
	@test -x dependencies/toolchain/linux-amd64/bin/ld.lld && test -x dependencies/toolchain/linux-amd64/bin/nasm || { echo "dependencies/toolchain/linux-amd64 is missing: run 'make toolchain' first, a release without the toolchain is not allowed" >&2; exit 1; }

check-toolchain-windows:
	@test -f dependencies/toolchain/windows-amd64/bin/nasm.exe && test -f dependencies/toolchain/windows-amd64/bin/ld.lld.exe || { echo "dependencies/toolchain/windows-amd64 is missing: run 'make toolchain' first, a release without the toolchain is not allowed" >&2; exit 1; }

release: prod prod-windows
	cd $(DIST) && sha256sum $(NAME).tar.gz $(WINNAME).zip > SHA256SUMS

prod-windows: check-toolchain-windows
	rm -f $(DIST)/$(WINNAME).zip
	rm -rf $(STAGE)
	mkdir -p $(DIST) $(STAGE)/$(APP)/bin
	GOOS=windows GOARCH=amd64 go build -o $(STAGE)/$(APP)/bin/$(APP).exe ./cmd/main
	cp -r std $(STAGE)/$(APP)
	mkdir -p $(STAGE)/$(APP)/toolchain && cp -a dependencies/toolchain/windows-amd64/. $(STAGE)/$(APP)/toolchain
	cd $(STAGE) && zip -qr $(CURDIR)/$(DIST)/$(WINNAME).zip $(APP)
	rm -rf $(STAGE)

PREFIX ?= $(HOME)/.$(APP)

install: prod
	rm -rf $(PREFIX).new
	mkdir -p $(PREFIX).new
	tar -C $(PREFIX).new --strip-components=1 -xzf $(DIST)/$(NAME).tar.gz
	rm -rf $(PREFIX)
	mv $(PREFIX).new $(PREFIX)

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

test-cli:
	go test -count=1 ./tests/testcmd

test-fmt:
	go test -count=1 ./tests/fmt

test: test-x64 test-lsp test-cli test-fmt

bench:
	python3 benchmark/bench.py $(ARGS)

clean:
	rm -rf bin
	rm -rf dist
