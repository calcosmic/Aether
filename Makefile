VERSION := $(shell \
	if [ -f .aether/version.json ]; then \
		sed -n 's/.*"version": *"\([^"]*\)".*/\1/p' .aether/version.json | head -1; \
	else \
		git describe --tags --abbrev=0 2>/dev/null | sed 's/^v//'; \
	fi \
)
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
