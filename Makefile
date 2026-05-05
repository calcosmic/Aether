VERSION := $(shell \
	if [ -f .aether/version.json ]; then \
		sed -n 's/.*"version": *"\([^"]*\)".*/\1/p' .aether/version.json | head -1; \
	else \
		git describe --tags --abbrev=0 2>/dev/null | sed 's/^v//'; \
	fi \
)
BINARY  := aether
LDFLAGS := -X github.com/calcosmic/Aether/cmd.Version=$(VERSION)

.PHONY: build test test-race lint version-check goreleaser-check parity-test npm-test narrator-test snapshot doctor-fast doctor doctor-ci clean install

build:
	go build -ldflags "$(LDFLAGS)" -o $(BINARY) ./cmd/aether/

test:
	go test ./... -count=1 -timeout 300s

test-race:
	go test ./... -race -count=1 -timeout 600s

lint:
	go vet ./...

version-check:
	test "$$(node -p 'require("./npm/package.json").version')" = "$$(node -p 'require("./.aether/version.json").version')"

goreleaser-check:
	goreleaser check

parity-test:
	go test ./cmd/... -run "(Parity|GoOnly)" -count=1 -timeout 300s -v

npm-test:
	npm --prefix npm test

narrator-test:
	npm --prefix .aether/ts ci
	npm --prefix .aether/ts audit --package-lock-only --audit-level=low
	npm --prefix .aether/ts run build
	git diff --exit-code -- .aether/ts/dist/narrator.js
	npm --prefix .aether/ts run typecheck
	npm --prefix .aether/ts test

snapshot:
	goreleaser build --snapshot --clean
	./dist/Aether_linux_amd64_v1/Aether version

doctor-fast: build lint version-check parity-test

doctor: build lint test version-check parity-test npm-test

doctor-ci: goreleaser-check doctor test-race snapshot narrator-test

clean:
	rm -f $(BINARY)

install: build
