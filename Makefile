# Makefile for configforge

BINARY=configforge
PKG=./cmd

.PHONY: all build run test clean fmt

all: build

build:
	go build -o $(BINARY) $(PKG)

run: build
	./$(BINARY)

test:
	go test ./...

fmt:
	gofmt -w .

clean:
	rm -f $(BINARY)
