.PHONY: build prod run clean
APP=wisp

VERSION=0.1.0
NAME=wisp-$(VERSION)-linux-amd64

build:
	go build -o bin/$(APP) ./cmd/main

prod:
	rm -rf dist/$(NAME)
	mkdir -p dist/$(NAME)

	cp -r bin dist/$(NAME)
	cp -r std dist/$(NAME)
	
	tar -C dist -czf dist/$(NAME).tar.gz $(NAME)

	rm -rf dist/$(NAME)

run:
	go run ./cmd/main

clean:
	rm -rf bin
	rm -rf dist
