VERSION := $(shell \
	if [ -f .aether/version.json ]; then \
		sed -n 's/.*"version": *"\([^"]*\)".*/\1/p' .aether/version.json | head -1; \
	else \
		git describe --tags --abbrev=0 2>/dev/null | sed 's/^v//'; \
	fi \
)
BINARY  := aether
LDFLAGS := -X github.com/calcosmic/Aether/cmd.Version=$(VERSION)

.PHONY: build test lint clean install smoke version-sync vulncheck

smoke:
	./scripts/smoke-daily-driver.sh

# Fails on any vulnerability this code actually calls. Modules we require but
# never call into are reported separately and do not fail the gate.
# The install line puts the binary in $(go env GOPATH)/bin, which is not on
# PATH on a default macOS setup — so the gate installed the tool and then died
# with "govulncheck: No such file or directory", reporting a tooling failure as
# if it were a scan result. Resolve the path instead of assuming PATH.
vulncheck:
	@command -v govulncheck >/dev/null 2>&1 || go install golang.org/x/vuln/cmd/govulncheck@latest
	@GOVULNCHECK=$$(command -v govulncheck || echo "$$(go env GOPATH)/bin/govulncheck"); \
	if [ ! -x "$$GOVULNCHECK" ]; then \
		echo "vulncheck: govulncheck not found after install (looked for $$GOVULNCHECK)" >&2; \
		exit 1; \
	fi; \
	"$$GOVULNCHECK" ./...

version-sync:
	./scripts/version-sync.sh

build:
	go build -ldflags "$(LDFLAGS)" -o $(BINARY) ./cmd/aether/

test:
	go test -race -count=1 ./...

lint:
	go vet ./...

clean:
	rm -f $(BINARY)

install: build
