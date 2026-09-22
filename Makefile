.PHONY: build prod run clean
APP=wisp

VERSION=0.1.0
NAME=wisp-$(VERSION)-linux-amd64

build:
	go build -o bin/$(APP) ./cmd/main

prod:
	rm -rf dist/$(APP)
	mkdir -p dist/$(APP)

	cp -r bin dist/$(APP)
	cp -r std dist/$(APP)
	
	tar -C dist -czf dist/$(NAME).tar.gz $(APP)

	rm -rf dist/$(APP)

run:
	go run ./cmd/main

clean:
	rm -rf bin
	rm -rf dist
