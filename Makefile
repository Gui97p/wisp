.PHONY: build run clean
APP=wisp

build:
	go build -o bin/$(APP) ./cmd/main

run:
	go run ./cmd/main

clean:
	rm -rf bin
