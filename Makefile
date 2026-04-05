VERSION := $(shell sed -n 's/.*"version": *"\([^"]*\)".*/\1/p' package.json | head -1)
BINARY  := aether
LDFLAGS := -X github.com/calcosmic/Aether/cmd.Version=$(VERSION)

.PHONY: build test lint clean install

build:
	go build -ldflags "$(LDFLAGS)" -o $(BINARY) ./cmd/aether/

test:
	go test -race -count=1 ./...

lint:
	go vet ./...

clean:
	rm -f $(BINARY)

install: build
