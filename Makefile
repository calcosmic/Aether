VERSION := $(shell \
	if [ -f .aether/version.json ]; then \
		sed -n 's/.*"version": *"\([^"]*\)".*/\1/p' .aether/version.json | head -1; \
	else \
		git describe --tags --abbrev=0 2>/dev/null | sed 's/^v//'; \
	fi \
)
BINARY  := aether
LDFLAGS := -X github.com/calcosmic/Aether/cmd.Version=$(VERSION)

.PHONY: build test lint clean install smoke bench-selftest bench-acceptance-order bench-dry-run version-sync vulncheck eval-gate-fast eval-gate-focused eval-gate-integration eval-gate-provider eval-gate-overnight eval-gate-race eval-gate-release eval-gate-journey prove-journey-fix-reverts

smoke:
	./scripts/smoke-daily-driver.sh

bench-selftest:
	./bench/lib/selftest.sh

bench-acceptance-order:
	./bench/acceptance/verify-predates-runs.sh

bench-dry-run:
	./bench/run.sh --dry-run

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
	# -timeout 90m is load-bearing, not padding: the cmd suite needs ~21 minutes
	# and go test's default per-package timeout is 10. On timeout the suite's own
	# controller stops early and prints a complete-looking per-lane summary of the
	# fraction it finished, so a truncated run is indistinguishable from a clean one
	# unless you read the FULL-SUITE discovered=/executed= headline. Always check
	# that those two numbers are equal before trusting the FAIL list.
	go test -race -count=1 -timeout 90m ./...

lint:
	go vet ./...

clean:
	rm -f $(BINARY)

install: build

# LEARN-05 (204-07-PLAN.md): the seven named eval gates (see
# cmd/testdata/eval-gates/gates.json for the authoritative declaration each
# target below mirrors). Every target pipes its run through the same
# discovered==executed accounting cmd/eval_gates.go's assertEvalGateCoverage
# encodes for Go callers -- a truncated run is indistinguishable from a
# clean one without this check, exactly the defect behind three entries in
# this project's own defect register (see CLAUDE.md's Verification
# Commands section). A gate that only partly ran fails the target instead
# of reporting clean.
define EVAL_GATE_CHECK
	@echo "eval-gate-$(1): go test $(2)"
	@DISCOVERED=$$(go test $(3) 2>/dev/null | grep -Ec '^(Test|Example)[A-Za-z0-9_]*$$' || true); \
	go test $(2) -v > /tmp/eval-gate-$(1).out 2>&1; STATUS=$$?; \
	EXECUTED=$$(grep -Ec '^=== RUN   (Test|Example)[A-Za-z0-9_]*$$' /tmp/eval-gate-$(1).out || true); \
	if [ "$$STATUS" -ne 0 ]; then tail -n 200 /tmp/eval-gate-$(1).out; fi; \
	rm -f /tmp/eval-gate-$(1).out; \
	echo "eval-gate-$(1): discovered=$$DISCOVERED executed=$$EXECUTED"; \
	if [ "$$DISCOVERED" != "$$EXECUTED" ]; then \
		echo "eval-gate-$(1): FAILED -- discovered=$$DISCOVERED executed=$$EXECUTED, a truncated run is indistinguishable from a clean one without this check"; \
		exit 1; \
	fi; \
	if [ "$$STATUS" -ne 0 ]; then \
		echo "eval-gate-$(1): FAILED -- go test exited $$STATUS"; \
		exit 1; \
	fi; \
	echo "eval-gate-$(1): PASS"
endef

# fast: the default-tagged tests in the small packages only, no network.
# Excludes pkg/codex (its own suite alone measures over a minute) -- same
# explicit package list as gates.json's own "fast" entry. Budget 60s
# (gates.json).
EVAL_GATE_FAST_PACKAGES := ./pkg/agent/... ./pkg/cache/... ./pkg/codegraph/... ./pkg/colony/... ./pkg/downloader/... ./pkg/events/... ./pkg/exchange/... ./pkg/graph/... ./pkg/learn/... ./pkg/llm/... ./pkg/memory/... ./pkg/smoke/... ./pkg/storage/... ./pkg/terminal/... ./pkg/trace/...

eval-gate-fast:
	$(call EVAL_GATE_CHECK,fast,-count=1 -timeout=60s $(EVAL_GATE_FAST_PACKAGES),-list=. $(EVAL_GATE_FAST_PACKAGES))

# focused: a name pattern supplied at invocation against the default build.
# Budget 300s (gates.json). Usage: make eval-gate-focused PATTERN=TestFoo.*
eval-gate-focused:
	@if [ -z "$(PATTERN)" ]; then \
		echo "eval-gate-focused requires PATTERN=<test-name-regexp>, e.g. make eval-gate-focused PATTERN=TestFoo"; \
		exit 1; \
	fi
	$(call EVAL_GATE_CHECK,focused,-run='$(PATTERN)' -count=1 -timeout=300s ./...,-list='$(PATTERN)' ./...)

# integration: the tests behind the integration build tag, excluded from
# the default build, no network required. Budget 1400s (gates.json).
eval-gate-integration:
	$(call EVAL_GATE_CHECK,integration,-tags=integration -count=1 -timeout=1400s ./...,-tags=integration -list=. ./...)

# provider: the tests behind the provider build tag, requiring provider
# credentials, never run by default. Budget 1400s (gates.json).
eval-gate-provider:
	$(call EVAL_GATE_CHECK,provider,-tags=provider -count=1 -timeout=1400s ./...,-tags=provider -list=. ./...)

# overnight: the tests behind the overnight build tag, long wall clock, the
# whole corpus plus anything too slow for any other gate. Budget 7200s
# (gates.json).
eval-gate-overnight:
	$(call EVAL_GATE_CHECK,overnight,-tags=overnight -count=1 -timeout=7200s ./...,-tags=overnight -list=. ./...)

# race: the full default build with the race detector. Budget 1260s (21m),
# CLAUDE.md's own stated figure for the cmd suite, not a guess.
eval-gate-race:
	$(call EVAL_GATE_CHECK,race,-race -count=1 -timeout=1260s ./...,-list=. ./...)

# release: the union of fast, focused, integration and race, plus the
# sentinel list -- the gate that must pass before anything is published.
# Budget 1800s (gates.json).
eval-gate-release:
	$(call EVAL_GATE_CHECK,release,-race -tags=integration -count=1 -timeout=1800s ./...,-tags=integration -list=. ./...)

# journey: the full messy-practice-project lifecycle, driven through a real
# claude -p chat with hooks and menu commands loaded. Requires a local,
# interactively-authenticated claude CLI; never runs in CI. Sets no
# environment variable that could turn this into a skip. Budget 6662s
# (PROVISIONAL, 208-08-PLAN.md Task 2): the 207-06 run's three trials each
# stopped at step 2 of 14 ("survey"), so its 1100s figure only ever covered
# two steps -- it cannot fit a full fourteen-step chain. This provisional
# number is the sum of all fourteen steps' own per-step wall-clock caps
# (journeyStepWallClockSecs in cmd/journey_live_test.go: 120+600+400+400+
# 700+700+1200+1200+120+120+120+400+400+120 = 6600s) for one trial (the
# owner-approved single walk, AETHER_JOURNEY_TRIALS=1), plus 207-06's own
# measured per-trial build/setup headroom (61.93s), rounded up. Replaced
# below by a figure re-measured from this phase's own real run.
eval-gate-journey:
	$(call EVAL_GATE_CHECK,journey,-tags=journey -run=TestJourney -count=1 -timeout=6662s ./...,-tags=journey -list=TestJourney ./...)

# prove-journey-fix-reverts (207-05-PLAN.md, UED-09, owner ruling D-01): the
# journey gate above proves the product works; this proves the gate is
# worth running -- reverting each of the five landed 2026-09-21 fixes in a
# throwaway working copy and watching the journey fail at the step it
# protects. Deliberately NOT folded into eval-gate-journey: the two cost
# very differently (this rebuilds a binary and drives a real chat per
# entry) and answer different questions. Requires a local, interactively-
# authenticated claude CLI; never runs in CI.
prove-journey-fix-reverts:
	./scripts/prove-journey-catches-the-2026-09-21-fixes.sh
