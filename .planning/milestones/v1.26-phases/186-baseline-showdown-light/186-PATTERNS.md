# Phase 186: Baseline Showdown (Light) - Pattern Map

**Mapped:** 2026-08-17
**Files analyzed:** 12 (harness scaffolding — no production Go code changes)
**Analogs found:** 10 / 12

This phase creates a benchmark harness under `bench/` (no `cmd/` or `pkg/`
changes). There is no prior `bench/` directory, so every new file's closest
analog is a *role/data-flow* match from `scripts/`, not a literal predecessor.

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|--------------------|------|-----------|-----------------|---------------|
| `bench/lib/hermetic-home.sh` | utility (shared shell lib) | file-I/O (isolated HOME setup) | `scripts/smoke-daily-driver.sh` lines 16-46 | role-match (extract, don't copy whole script) |
| `bench/harness/run-lane.sh` | controller (orchestrates one lane's run) | request-response (shell process orchestration) | `scripts/smoke-daily-driver.sh` (whole-file structure) | exact (same genre: hermetic build+run+assert script) |
| `bench/harness/run-cell.sh` | controller (wraps run-lane for one task×lane cell) | request-response | `scripts/smoke-test-classic.sh` (whole-file structure) | role-match |
| `bench/tasks/*.md` (4 task specs) | config (declarative task definition) | transform (spec read by harness + acceptance script) | none in-repo — nearest analog is a `.aether/data/planning/*.md` PLAN.md shape, but that's colony-internal, not a fair pattern source | no analog (use RESEARCH.md/CONTEXT.md conventions) |
| `bench/acceptance/<task>.sh` | test (deterministic pass/fail script) | batch (exit-code verdict) | `scripts/smoke-daily-driver.sh` gate blocks, e.g. lines 73-119 (`step`/`fail` gate pattern) | role-match |
| `bench/lib/operator-log.sh` | utility (append-only structured log) | event-driven (timestamped input log) | `cmd/spend_session_capture.go` lines 88-123 (`recordSpendSessionFromHook`) — same shape: append/overwrite a small JSON record with an RFC3339 UTC timestamp | role-match |
| `bench/lib/measure-tokens.sh` (+ small Go or python helper) | service (harness-side usage measurement) | transform (parse transcript → usage numbers) | `pkg/codex/usage.go` `ParseUsage`/`usageFromEvent`/`usageFromFields` (lines 100-238); `cmd/spend_session_capture.go` `validateSpendTranscriptPath` (lines 46-86) for the transcript **location** convention | exact (event-shape parsing logic to port into the harness) |
| `bench/lib/git-cleanliness.sh` | test (checker script) | batch | `scripts/smoke-daily-driver.sh` gate 1/2 idempotency checks (lines 73-91, jq-based assertions) | role-match |
| `bench/results/generate-table.sh` (or `.py`) | utility (results-table generator) | transform (raw run JSON → markdown table) | `scripts/build_reliability_matrix.py` (nearest existing "aggregate raw data into a report" script in `scripts/`) | role-match |
| `bench/results/<date>/*` (raw evidence, gitignored-except-committed) | config/data (committed run evidence) | file-I/O | `.aether/data/COLONY_STATE.json` shape as a "durable JSON record of a run" analog only for field-naming conventions — not for structure | partial match |
| `bench/README.md` (one documented reproduce-a-run command) | config (docs) | n/a | `scripts/smoke-daily-driver.sh` header comment block (lines 1-15) — same "what this proves, how to run it, exit behavior" doc style | role-match |
| context-capture step (`aether build --print-brief --full` invocation + committed capture) | n/a — CLI invocation, not a new file's pattern | request-response (read-only) | `cmd/build_print_brief.go` (whole file) — this IS the command being invoked, not something to reimplement | exact (invoke as-is; do not reimplement) |

## Pattern Assignments

### `bench/harness/run-lane.sh` and `bench/harness/run-cell.sh` (controller, request-response)

**Analog:** `scripts/smoke-daily-driver.sh` (whole file, 209 lines) and `scripts/smoke-test-classic.sh` (whole file)

Both existing smoke scripts are the right shape for a harness lane runner: hermetic HOME setup, staged numbered gates, fail-fast with a labeled `fail()` helper, and a trap-based cleanup. Copy this skeleton directly.

**Shebang + strict mode + doc header** (`scripts/smoke-daily-driver.sh` lines 1-16):
```bash
#!/usr/bin/env bash
# smoke-daily-driver.sh — the daily-driver regression tripwire.
#
# Builds aether from source, installs it into a fully isolated HOME/hub,
# then proves the four things that must never break for a downstream user:
#   ...
# Exits non-zero on the first failure. Requires: go, node >= 20, npm, jq, git.
set -euo pipefail
```

**Working dir + temp workspace + trap cleanup** (`scripts/smoke-daily-driver.sh` lines 18-20):
```bash
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
WORK="$(mktemp -d)"
trap 'chmod -R u+w "$WORK" 2>/dev/null || true; rm -rf "$WORK"' EXIT
```

**Alternative cleanup pattern with worktree removal** (`scripts/smoke-test-classic.sh` lines 18-26) — use this variant if a lane needs a git worktree per run (the substrate repos are fresh-cloned per run, so `git worktree add`/`remove` or a plain `git clone` into `$WORK` both fit; the CONTEXT.md says "fresh clone per run" so plain `git clone` into `mktemp -d` is the simpler match):
```bash
cleanup() {
    if [[ -n "$FAKE_HOME" && -d "$FAKE_HOME" ]]; then
        rm -rf "$FAKE_HOME" 2>/dev/null || true
    fi
    if [[ -n "$WORKDIR" && -d "$WORKDIR" ]]; then
        git worktree remove "$WORKDIR" --force 2>/dev/null || true
    fi
}
trap cleanup EXIT
```

**Required-tool preflight** (`scripts/smoke-daily-driver.sh` lines 32-34):
```bash
for tool in go node npm jq git; do
  command -v "$tool" >/dev/null || fail "required tool missing: $tool"
done
```

**`fail`/`step` helper pair** (`scripts/smoke-daily-driver.sh` lines 29-30) — use this over `smoke-test-classic.sh`'s `pass`/`fail` pair; it prints a labeled section banner per gate, which maps directly onto "one gate per task category":
```bash
fail() { echo "SMOKE FAIL: $*" >&2; exit 1; }
step() { echo; echo "==> $*"; }
```

**Numbered gate pattern with jq assertions** (`scripts/smoke-daily-driver.sh` lines 73-91) — this is the shape for each acceptance/cleanliness check the harness runs after a lane finishes:
```bash
step "gate 1: aether update --force preserves foreign settings"
(cd "$REPO" && "$BIN" update --force >/dev/null) || fail "aether update --force failed"
SETTINGS="$REPO/.claude/settings.json"
jq -e '.gsdMarker == "do-not-touch"' "$SETTINGS" >/dev/null \
  || fail "custom top-level key lost by update --force"
...
echo "    settings merge OK"

step "gate 2: repeated update --force is idempotent"
BEFORE="$(shasum -a 256 "$SETTINGS")"
(cd "$REPO" && "$BIN" update --force >/dev/null) || fail "second aether update --force failed"
AFTER="$(shasum -a 256 "$SETTINGS")"
[ "$BEFORE" = "$AFTER" ] || fail "second update rewrote settings.json"
echo "    idempotency OK"
```

**Non-zero-exit assertion with captured stderr/stdout for diagnosis** (`scripts/smoke-daily-driver.sh` lines 99-114) — this is the pattern for the SIGKILL/resume task category, where the harness must assert a specific exit code and specific log content, not just "it ran":
```bash
HOST_OUT="$WORK/host-plan.out"
set +e
(cd "$REPO" && "$BIN" host plan --dry-run >"$HOST_OUT" 2>&1)
HOST_STATUS=$?
set -e
if grep -q "ERR_MODULE_NOT_FOUND\|Cannot find package" "$HOST_OUT"; then
  sed -n '1,10p' "$HOST_OUT" >&2
  fail "TS host died on missing dependencies (P0-04 regression)"
fi
[ "$HOST_STATUS" -eq 1 ] || {
  sed -n '1,10p' "$HOST_OUT" >&2
  fail "TS host without a colony: expected exit 1, got $HOST_STATUS"
}
```
This is the exact shape for the interrupted-execution task: `timeout`/background `$BIN` process + `sleep 120` + `kill -9 $PID`, capture the state at kill time, then run the resume command and assert against captured stdout/stderr the same way.

**Skip-loudly-never-silently pattern for an optional/unavailable lane** (`scripts/smoke-daily-driver.sh` lines 180-205) — apply this if a lane's tool (e.g. GSD's CLI, or a specific auth mechanism) is unavailable on a given machine; never let an unscored lane look like a passed one:
```bash
if command -v opencode >/dev/null 2>&1; then
  echo "    opencode binary present ($(opencode --version 2>/dev/null | head -1))"
else
  echo "    SKIPPED: opencode binary not on PATH — install it to exercise the live OpenCode lane" >&2
fi
```

---

### `bench/lib/hermetic-home.sh` (utility, file-I/O)

**Analog:** `scripts/smoke-daily-driver.sh` lines 22-46 (isolated HOME + fresh install)

This is the exact pattern CONTEXT.md's canonical-refs section names as "the existing hermetic-HOME pattern to extend, not reinvent." Extract into a shared lib sourced by both `run-lane.sh` (Aether lanes) and any GSD-lane equivalent.

**Go module cache pinning BEFORE HOME override** (lines 22-27) — critical: without this, isolating `HOME` also isolates `GOPATH`/`GOCACHE`, forcing a full module re-download per run, which breaks the "reproducible in one documented command" and "under a week" constraints:
```bash
# Pin Go caches to the real ones BEFORE overriding HOME, so tool invocations
# under the isolated HOME reuse the module cache instead of re-downloading
# into a throwaway (and read-only-flavored) GOPATH.
export GOPATH="$(go env GOPATH)"
export GOMODCACHE="$(go env GOMODCACHE)"
export GOCACHE="$(go env GOCACHE)"
```

**Isolated HOME + hub override + fresh install** (lines 40-46):
```bash
# Fully isolate: nothing touches the real ~/.claude or ~/.aether.
export HOME="$WORK/home"
export AETHER_HUB_DIR="$HOME/.aether"
mkdir -p "$HOME"

step "installing package into isolated hub"
(cd "$ROOT" && "$BIN" install --package-dir "$ROOT" >/dev/null) || fail "aether install failed"
```
For the benchmark harness this generalizes to: build once, then for lane 1/2/3 create three sibling `$HOME` directories, each installing only that lane's system (Aether hub install for lanes 1-2, GSD's file-copy install for lane 3 — GSD's own install mechanism is external to this repo and must be researched/documented in the harness README, not assumed).

**`smoke-test-classic.sh`'s alternate rationale for isolating HOME** (lines 47-52) is worth copying verbatim as a comment template — it documents *why* isolation matters, which the harness README should do too for auditability:
```bash
# Isolate HOME to prevent v5.4.0's delegation shim from finding ~/.aether/bin/aether.
# ... By setting HOME to a temp directory, we force Classic to use its own
# Node.js implementations.
```

---

### `bench/lib/measure-tokens.sh` + parser (service, transform)

**Analog:** `pkg/codex/usage.go` (whole file — `ParseUsage`, `usageFromEvent`, `usageFromFields`, `billedTotal`) and `cmd/spend_session_capture.go` `validateSpendTranscriptPath` (lines 46-86)

CONTEXT.md is explicit: tokens must be measured **harness-side from provider usage**, never a system's self-report — and Phase 185's `aether spend` self-report is something the harness *validates against*, never substitutes. `pkg/codex/usage.go` is the canonical parser logic for provider usage event shapes; it currently only parses raw worker stdout (NDJSON `token_count` events for Codex-shaped output, terminal `result` events with `usage`/`total_cost_usd` for Claude-shaped output), not the on-disk session transcript file directly — no such transcript-file parser exists yet in this repo, so the harness needs to write its own reader following the same event-shape logic.

**Where Claude Code session transcripts live** (confirmed via `cmd/spend_session_capture.go` lines 39-45, 63 and the test fixture in `cmd/spend_session_capture_test.go` lines 14-32):
```
$HOME/.claude/projects/<encoded-cwd>/<sessionID>.jsonl
```
Since the harness fully isolates `$HOME` per lane (see hermetic-home pattern above), this path is deterministic per run: `$WORK/home/.claude/projects/<encoded-repo-path>/<sessionID>.jsonl`. The encoding of `<encoded-cwd>` replaces path separators with `-` (see the fixture: `-Users-test-repo` for a repo at a path containing `/Users/test/repo`-like segments) — the harness's token-measurement script should glob `$HOME/.claude/projects/*/*.jsonl` rather than hand-construct the encoded name, since the exact encoding algorithm isn't itself documented as a stable contract in this repo.

**Path validation/boundary pattern to reuse for safety** (`cmd/spend_session_capture.go` lines 46-86) — not strictly required for a harness script (which controls its own isolated HOME and doesn't need to defend against adversarial input), but worth mirroring the *shape* — reject empty/relative paths, resolve symlinks, check containment — if the harness Go/python helper reads an arbitrary transcript path passed as an argument:
```go
func validateSpendTranscriptPath(claimed string) (string, error) {
    claimed = strings.TrimSpace(claimed)
    if claimed == "" {
        return "", fmt.Errorf("transcript path is empty")
    }
    ...
    root := filepath.Join(home, ".claude", "projects")
    ...
}
```

**Usage struct + measured/estimated distinction** (`pkg/codex/usage.go` lines 8-77) — the harness's own results table should carry the same `Source` distinction (provider / session-transcript / estimate) so a run's token figure is never silently presented as more authoritative than it is:
```go
type WorkerUsage struct {
    InputTokens         int64   `json:"input_tokens,omitempty"`
    CachedInputTokens   int64   `json:"cached_input_tokens,omitempty"`
    CacheCreationTokens int64   `json:"cache_creation_tokens,omitempty"`
    OutputTokens        int64   `json:"output_tokens,omitempty"`
    TotalTokens         int64   `json:"total_tokens,omitempty"`
    USDCost             float64 `json:"usd_cost,omitempty"`
    Model               string  `json:"model,omitempty"`
    Source              string  `json:"source,omitempty"` // "provider" | "session-transcript" | "estimate"
}
```

**Event-shape scan loop to port** (`pkg/codex/usage.go` lines 108-141) — this is the line-by-line NDJSON scan pattern; the harness's transcript reader does the same thing but reads a `.jsonl` file instead of captured stdout, and Claude Code session transcripts carry per-turn `usage` objects on assistant messages rather than a single terminal `result` event, so the "last one wins / supersede" reduction still applies but the event `type` to match on on-disk (`"type":"assistant"` with a nested `message.usage`) differs from the raw-stdout shape this function currently matches (`"type":"result"`) — verify the actual on-disk field names against a real transcript fixture before assuming they match `usageFromFields`'s key list:
```go
for _, line := range strings.Split(rawOutput, "\n") {
    line = strings.TrimSpace(stripANSIEscapeCodes(line))
    if line == "" || !strings.HasPrefix(line, "{") {
        continue
    }
    var event map[string]interface{}
    if err := json.Unmarshal([]byte(line), &event); err != nil {
        continue
    }
    if parsed, ok := usageFromEvent(event); ok {
        usage = parsed
        found = true
    }
}
```

**Billed-total derivation — do not re-derive by hand** (`pkg/codex/usage.go` lines 255-298) — this file documents a real 186x undercount bug from computing `input + output` instead of `input + cache_read + cache_creation + output`. The harness's token math must sum all four disjoint fields, matching `billedTotal()`/`BilledTotalTokens()` exactly, not reinvent the addition:
```go
func (u WorkerUsage) billedTotal() int64 {
    return u.InputTokens + u.CachedInputTokens + u.CacheCreationTokens + u.OutputTokens
}
```

---

### `bench/acceptance/<task>.sh` (test, batch)

**Analog:** `scripts/smoke-daily-driver.sh` gate blocks (e.g. lines 121-158, the `jq -e` dispatch-manifest assertion)

**Pattern: deterministic exit-code verdict against a fresh clone's file state** (lines 153-158):
```bash
BUILD_OUT="$WORK/build-plan.json"
(cd "$REPO" && "$BIN" build 1 --plan-only >"$BUILD_OUT" 2>"$WORK/build-plan.err") \
  || { cat "$WORK/build-plan.err" >&2; fail "aether build 1 --plan-only failed"; }
jq -e '.ok == true and (.result.dispatch_manifest.dispatches | length) >= 1' "$BUILD_OUT" >/dev/null \
  || { head -c 400 "$BUILD_OUT" >&2; fail "plan-only build did not emit a parseable dispatch_manifest"; }
```
Each `bench/acceptance/<task>.sh` should follow the same shape: take a path to the run's resulting clone as `$1`, run deterministic checks (test suite exit code, file contents, specific strings absent/present), exit 0 only on full pass, and print a one-line reason on failure. CONTEXT.md requires these scripts avoid both systems' own verification vocabularies (no GSD `must_haves`, no Aether criterion IDs) — this is a constraint on content, not on shape; the shape above still applies.

---

### `bench/lib/git-cleanliness.sh` (test, batch)

**Analog:** `scripts/smoke-daily-driver.sh` idempotency check (lines 86-91) — same "compare before/after state, fail on unexpected diff" shape, applied to `git status --porcelain` and `git branch` output instead of a settings file hash:
```bash
BEFORE="$(shasum -a 256 "$SETTINGS")"
(cd "$REPO" && "$BIN" update --force >/dev/null) || fail "second aether update --force failed"
AFTER="$(shasum -a 256 "$SETTINGS")"
[ "$BEFORE" = "$AFTER" ] || fail "second update rewrote settings.json"
```
Generalized for cleanliness checking: capture `git branch --list` and `git status --porcelain` in the substrate repo after a run, count orphan branches (branches other than the starting branch that were never merged/deleted) and residue paths matching tool-internal patterns (`.aether/`, `.planning/` beyond the lane's legitimate working state — CONTEXT.md says define the legitimate set per lane). Output a numeric cleanliness score, not just pass/fail, since results table format calls for a "cleanliness score" column.

---

### `bench/results/generate-table.sh` (utility, transform)

**Analog:** `scripts/build_reliability_matrix.py` — the only existing "aggregate raw per-run data into a report" script in `scripts/`. Not read in full here (python, not shell — role-match only, not literal-pattern-match); the planner should treat this as the closest sibling and inspect it directly if a Python-based generator is chosen over shell/jq. Given every other new script in this phase is bash, a jq-based markdown table generator consuming a directory of per-run JSON files (one per cell, written by `run-cell.sh`) is more consistent with the rest of `bench/`'s shell-only convention and CONTEXT.md's explicit "it is shell scripts, task specs, and a results table" scope constraint.

---

## Shared Patterns

### Strict-mode + fail-fast shell convention
**Source:** `scripts/smoke-daily-driver.sh` line 16, `scripts/smoke-test-classic.sh` line 11, `scripts/version-sync.sh` line 11
**Apply to:** every new `bench/**/*.sh` file
```bash
set -euo pipefail
```
This is a repo-wide convention with zero exceptions found across the three scripts inspected.

### Root-relative path resolution
**Source:** `scripts/smoke-daily-driver.sh` line 18, `scripts/version-sync.sh` line 13
**Apply to:** any harness script that needs to locate the repo root regardless of invocation directory
```bash
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
```

### Temp workspace + trap-based cleanup
**Source:** `scripts/smoke-daily-driver.sh` lines 19-20
**Apply to:** `run-lane.sh`, `run-cell.sh`, any script that clones a substrate repo or builds a binary
```bash
WORK="$(mktemp -d)"
trap 'chmod -R u+w "$WORK" 2>/dev/null || true; rm -rf "$WORK"' EXIT
```
Note the `chmod -R u+w` before `rm -rf` — this matters because `go build` / `npm install` artifacts and git-cloned repos can leave read-only files that a bare `rm -rf` chokes on mid-cleanup.

### Required-tool preflight
**Source:** `scripts/smoke-daily-driver.sh` lines 32-34
**Apply to:** the top-level `bench/run.sh` entry point (the "one documented command" CONTEXT.md requires)
```bash
for tool in go node npm jq git; do
  command -v "$tool" >/dev/null || fail "required tool missing: $tool"
done
```
Extend the tool list per-lane (the GSD lane likely needs whatever GSD's CLI depends on; document this in `bench/README.md`).

### Isolated-HOME hermetic pattern (the load-bearing shared pattern for this entire phase)
**Source:** `scripts/smoke-daily-driver.sh` lines 22-46
**Apply to:** `bench/lib/hermetic-home.sh`, sourced by every lane runner
This is explicitly named in CONTEXT.md's canonical references as the pattern to extend. See full excerpt under `bench/lib/hermetic-home.sh` above. The one addition the benchmark needs beyond the existing smoke script: three *separate* hermetic HOMEs must exist per cell (one per lane), never a shared one — CONTEXT.md's lane-independence requirement ("must never be pooled") extends to the filesystem-isolation level, not just the scoring level.

### Harness-side, provider-grade token measurement — never a system's self-report
**Source:** `pkg/codex/usage.go` (whole file) + `cmd/spend_session_capture.go` lines 88-140
**Apply to:** `bench/lib/measure-tokens.sh` and the results-table generator
The `Source` field convention (`"provider"` / `"session-transcript"` / `"estimate"`) and the `billedTotal()` arithmetic (four disjoint fields summed, never `input + output` alone) must be mirrored exactly, given this repo has a documented 186x-undercount incident from getting that arithmetic wrong once already.

## No Analog Found

| File | Role | Data Flow | Reason |
|------|------|-----------|--------|
| `bench/tasks/*.md` (4 task specs: bug fix, brownfield feature, interrupted execution, fresh-repo lifecycle) | config | transform | No prior task-spec format exists in this repo to imitate; nearest structural sibling (`.planning/phases/*/CONTEXT.md`) is colony-internal tooling output, not a hand-authored spec, and copying its shape would blur the CONTEXT.md requirement that acceptance scripts avoid both systems' own verification vocabularies. Author from scratch per CONTEXT.md's decisions/specifics sections. |
| GSD-lane install/invocation wrapper (`bench/lib/gsd-install.sh` or similar) | utility | file-I/O | GSD is an external tool with no analog inside this repo (per user's own memory record, "GSD is a dev tool for building Aether, not an Aether component" — its install mechanism is not documented anywhere in this codebase). Must be researched fresh against GSD's own install docs, not patterned from Aether's `aether install`. |

## Metadata

**Analog search scope:** `scripts/`, `.github/workflows/`, `cmd/build_print_brief.go`, `cmd/spend_session_capture.go`, `pkg/codex/usage.go`, `cmd/spend_session_capture_test.go`
**Files scanned:** 4 shell scripts, 3 Go source files, 1 Go test file, 1 CI workflow, 1 CONTEXT.md
**Pattern extraction date:** 2026-08-17
