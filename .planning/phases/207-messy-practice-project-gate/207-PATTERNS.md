# Phase 207: A Messy Practice Project Is the Release Gate - Pattern Map

**Mapped:** 2026-09-22
**Files analyzed:** 6 (builder script, trap-assertion Go test, journey harness, trial/flake report writer, gates.json entry, Makefile + runbook wiring)
**Analogs found:** 6 / 6

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|---|---|---|---|---|
| `scripts/build-messy-practice-project.sh` | utility (fixture builder, shell) | file-I/O (filesystem+git construction) | `scripts/proof-screens-reach-the-owner.sh` | role-match (isolation pattern, not content) |
| `cmd/messy_practice_project_test.go` (or similar, asserting the 9 traps) | test | file-I/O / assertion | `cmd/eval_gates_test.go` (manifest-reading test pattern) + `scripts/proof-screens-reach-the-owner.sh` gate-assertion style | role-match |
| `cmd/journey_test.go` (`//go:build journey`, `TestJourney`) | test (integration/journey harness) | streaming (JSONL) + event-driven (chat turns) | `scripts/proof-screens-reach-the-owner.sh` gate 3/4 (`claude -p` invocation + JSONL assertion) | exact (same mechanics, longer chain) |
| journey trial/flake JSON report writer (Go, inside the journey test or a small helper file) | utility (report writer) | transform / batch | none exact — model on `cmd/eval_gates.go`'s manifest read/write shape + the transient-failure regex in `proof-screens-reach-the-owner.sh` | role-match |
| `cmd/testdata/eval-gates/gates.json` (add `journey` entry) | config | CRUD (declarative manifest, read by `cmd/eval_gates.go`) | itself — existing `provider`/`overnight` entries (`requires` gated, no test declares the tag yet) | exact |
| `Makefile` (`eval-gate-journey` target) + `.aether/docs/publish-update-runbook.md` (Preflight bullet) | config / docs | request-response (manual invocation) | `Makefile` `eval-gate-provider`/`eval-gate-overnight` targets; runbook's existing `git status --porcelain` Preflight bullet | exact |

## Pattern Assignments

### `scripts/build-messy-practice-project.sh` (utility, file-I/O)

**Analog:** `scripts/proof-screens-reach-the-owner.sh` (full 203 lines read)

**Header/isolation pattern** (lines 1-29):
```bash
#!/usr/bin/env bash
# ISOLATION RULE, followed throughout this script: nothing here writes to the
# real $HOME/.claude/ or the real ~/.aether/ hub. ...
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
WORK="$(mktemp -d)"
trap 'rm -rf "$WORK"' EXIT

fail() { echo "PROOF FAIL: $*" >&2; exit 1; }
step() { echo; echo "==> $*"; }

for tool in go git jq claude; do
  command -v "$tool" >/dev/null 2>&1 || fail "required tool missing: $tool"
done
```
Copy this exact preamble shape (tool-check loop, `trap ... EXIT` cleanup, `fail`/`step` helpers, `set -euo pipefail`). The builder script differs in that it must accept a destination directory argument (not always a `mktemp -d`) so `TestMessyPracticeProjectHasEveryTrap` can point at a fixed, inspectable path, and it must be safely re-runnable (idempotent — "runs clean twice" per UED-07) rather than using `trap ... EXIT` to always delete itself.

**Scratch repo construction pattern** (lines 54-60):
```bash
REPO="$WORK/repo"
mkdir -p "$REPO"
git init -q "$REPO"
git -C "$REPO" config user.email "proof@example.com"
git -C "$REPO" config user.name "proof"
```
Reuse verbatim for the practice project's own git init.

**Case-only-collision trap — construct via two different directories, never two siblings** (per RESEARCH.md Pitfall 4 and the real `57e563ba` bug shape): e.g. `.aether/HANDOFF.md` + `.aether/data/handoff.md`. Do not attempt `Foo.md`/`foo.md` in one directory (APFS aliases them).

**Symlink-to-folder / symlink-loop / nested-git traps**: plain `ln -s`, `git init` in a subdirectory — no existing script pattern to copy; construct directly with shell primitives, left uncommitted where the trap requires "unsaved changes" (per RESEARCH.md Q3/Q4 trap table).

---

### `cmd/messy_practice_project_test.go` (test, file-I/O assertion)

**Analog:** `scripts/proof-screens-reach-the-owner.sh` gate-assertion style (jq/grep-based `fail` calls) + `cmd/eval_gates.go`'s manifest-path constant pattern (lines 36-53) for how this repo names committed testdata paths.

**Core assertion pattern to copy** — one small, named check per trap, each producing a clear failure reason (mirrors `fail "..."` calls in the proof script, lines 61-63, 84-85, 121 etc.):
```bash
step "gate 1: the install really registered both routes"
[ -f "$SETTINGS" ] || fail "no .claude/settings.json after aether update --force"
jq -e '...' "$SETTINGS" >/dev/null || fail "... -- run aether update --force"
```
Translate this "one assertion, one named failure reason" discipline into Go subtests (`t.Run("shortcut-to-a-folder", ...)`, etc.) — each subtest asserts on a filesystem/git fact (`os.Lstat`, `filepath.EvalSymlinks`, `git show`), never on prose, per CLAUDE.md's "asserting on files produced ... never wording" rule.

---

### `cmd/journey_test.go` (`//go:build journey`, integration/streaming)

**Analog:** `scripts/proof-screens-reach-the-owner.sh` gate 3 (chat invocation) and gate 4 (assertion), full text read above.

**Chat invocation + caps pattern** (lines 87-97):
```bash
PROMPT='Run the exact command ... Add no commentary.'
ARGS=(-p "$PROMPT" --output-format stream-json --verbose --allowedTools Bash --settings "$SETTINGS" --max-turns 6)
CAPS=("max-turns=6")
if claude --help 2>&1 | grep -q -- '--max-budget-usd'; then
  ARGS+=(--max-budget-usd 1.00)
  CAPS+=("max-budget-usd=1.00")
fi
CAPS+=("wall-clock=300s (timeout 300)")
```
Copy this defensive "probe `claude --help` before using a flag" pattern for `--max-budget-usd`. For the 12-step journey, use `--resume "$session_id"` (captured from the first call's `--output-format json | jq -r '.session_id'`) for every step after the first, per RESEARCH.md Q1 — this script has no precedent for multi-turn chaining (it is single-turn), so that part is new composition, not copyable code.

**Transient-failure retry pattern** (lines 105-116):
```bash
RETRIED="no"
set +e
run_chat
CHAT_STATUS=$?
set -e
if [ "$CHAT_STATUS" -ne 0 ]; then
  if grep -qiE 'rate.?limit|overloaded|timed?.?out' "$CHAT_ERR" "$CHAT_OUT" 2>/dev/null; then
    echo "    transient failure (exit $CHAT_STATUS) -- retrying once, as instructed; not counted as a real failure unless it recurs"
    RETRIED="yes"
    set +e; run_chat; CHAT_STATUS=$?; set -e
  fi
fi
```
Generalize this exact regex (`rate.?limit|overloaded|timed?.?out`) into the journey's 3-trial classifier per RESEARCH.md Q6 — do not import cline's full 5-category system; keep the regex table small per the "one review round" framing.

**Assertion-on-transcript pattern, never prose** (lines 120-131, primary + documented fallback):
```bash
BASH_CALLS=$(jq -c 'select(.message.content != null) | .message.content[] | select(.type=="tool_use" and .name=="Bash")' "$CHAT_OUT" 2>/dev/null | wc -l | tr -d ' ')
[ "$BASH_CALLS" -ge 1 ] || fail "the chat never ran a Bash tool call ..."
```
Copy this `tool_use`/Bash-call JSONL query verbatim as the base assertion primitive; extend with a `<command-name>` tag check (per RESEARCH.md Q1, verified against `cmd/testdata/stop-hook/menu-command-transcript.jsonl`) to prove a menu command, not a raw CLI call, drove each step:
```
jq -r 'select(.type=="user") | .message.content' transcript.jsonl | grep -F '<command-name>/ant-init</command-name>'
```

**Fixture reference for transcript shape** — `cmd/testdata/stop-hook/menu-command-transcript.jsonl` (already in-repo, read this session): first line's `message.content` is literally `"<command-message>ant-status</command-message>\n<command-name>/ant-status</command-name>"`. Use as the ground-truth shape when writing the journey's own transcript-parsing assertions, and as a Go test fixture if the journey harness needs an offline unit test for its own JSONL parser (separate from the `journey`-tagged live test) — mirrors how `cmd/stop_hook_screen_test.go` already reads this same fixture file for `TestStopHookSendsBackAReplyThatHidTheScreen` and friends.

---

### journey trial/flake JSON report writer

**No exact analog** — first writer of this shape in the repo (per RESEARCH.md Q6: "no existing precedent"). Nearest structural analog: `cmd/eval_gates.go`'s manifest read/write conventions (JSON marshal/unmarshal against a committed schema-versioned file, `evalGateManifestSchemaVersion` constant) — copy the "schema_version" top-level field convention for the new `journey-report.json` shape, and write it to `.aether/data/worker-debug/journey-report.json` (already inside `sanctionedDataWritePrefixes`, `cmd/hook_cmds.go:846-871` — no new allowlist entry needed).

---

### `cmd/testdata/eval-gates/gates.json` (config, CRUD)

**Analog:** itself — the existing `provider`/`overnight` entries model exactly this shape ("no test declares this tag yet").

**Pattern to copy** (from the file read above):
```json
{
  "name": "journey",
  "packages": ["./..."],
  "build_tags": ["journey"],
  "run_pattern": "TestJourney",
  "requires": "claude_cli_credentials",
  "budget_seconds": <sized from Q6 instrumentation>,
  "purpose": "The full messy-practice-project lifecycle journey, driven through a real claude -p chat with hooks and menu commands loaded; requires a local, interactively-authenticated claude CLI. Never run in CI (no credential is configured there). The gate for every release from here on -- see .aether/docs/publish-update-runbook.md's Preflight checklist."
}
```
Reader: `cmd/eval_gates.go` (`evalGateManifestPath` constant, schema-version check, `requires` field semantics) — read this file's header comment (lines 1-53) for the exact conventions to follow (constants, override vars for test isolation).

---

### `Makefile` + `.aether/docs/publish-update-runbook.md`

**Analog:** `Makefile` lines 72-134 (`EVAL_GATE_CHECK` macro + `eval-gate-provider`/`eval-gate-overnight` targets), and the runbook's existing Preflight bullet.

**Makefile target pattern** (lines 116-117, 122-123):
```makefile
eval-gate-provider:
	$(call EVAL_GATE_CHECK,provider,-tags=provider -count=1 -timeout=1400s ./...,-tags=provider -list=. ./...)
```
Add `eval-gate-journey` following this exact macro-call shape, with `-tags=journey -run=TestJourney`, and register it in the `.PHONY` line (Makefile line 11) alongside the other six.

**Runbook Preflight bullet pattern**: mirror the existing `git status --porcelain` must-be-empty bullet in `.aether/docs/publish-update-runbook.md`'s Preflight section (first ~60 lines) — add a new bullet naming `make eval-gate-journey` as a manual, local, pre-publish step. This is a procedural/documentation change only, consistent with the milestone's "no new strict rules in the `aether` binary" constraint (RESEARCH.md's Architectural Responsibility Map: "Release-gate wiring ... Process/documentation layer, not the `aether` binary").

## Shared Patterns

### Isolation discipline (hub only, never HOME)
**Source:** `scripts/proof-screens-reach-the-owner.sh` lines 18-29, 51-53
**Apply to:** the builder script and the journey test — both must isolate `AETHER_HUB_DIR` and leave `HOME` untouched (Claude Code's own login lives there).
```bash
export AETHER_HUB_DIR="$WORK/hub"
mkdir -p "$AETHER_HUB_DIR"
```

### Never assert on prose
**Source:** CLAUDE.md ("Assertions are on files produced and commands the chat ran, never on wording") + `scripts/proof-screens-reach-the-owner.sh`'s own documented primary/fallback split (lines 120-152)
**Apply to:** every assertion in the journey test and the trap-assertion test — resolve to (a) a `tool_use`/Bash-call JSONL entry, (b) a `<command-name>` tag, or (c) on-disk state (`COLONY_STATE.json`, file existence, `aether status --output-format json`). Never `grep -i "success"` against chat text.

### Transient-vs-real failure classification
**Source:** `scripts/proof-screens-reach-the-owner.sh` lines 105-116 (single retry, regex `rate.?limit|overloaded|timed?.?out`)
**Apply to:** the journey's 3-trial harness — generalize the same regex table (kept small per "one review round"), classify each of the 3 trials, write results via the new JSON report writer.

### discovered==executed truncation check
**Source:** `Makefile` `EVAL_GATE_CHECK` macro, lines 72-88 (`DISCOVERED`/`EXECUTED` comparison, per CLAUDE.md's Verification Commands warning)
**Apply to:** the journey's own trial accounting — a chat that silently stops mid-journey is the same failure class as a truncated test run; reuse the "count what was declared vs. what actually ran" discipline rather than trusting a clean-looking summary.

### `--bare` must never be passed, and its absence is not enough
**Source:** RESEARCH.md Q1 / Pitfall 1 (docs cited: `code.claude.com/docs/en/headless`)
**Apply to:** every `claude -p` invocation in both the builder-adjacent smoke check and the journey test — must positively assert hooks/commands fired (via `<command-name>` tag or `--include-hook-events`), never merely rely on omitting `--bare`.

## No Analog Found

| File | Role | Data Flow | Reason |
|---|---|---|---|
| journey trial/flake JSON report writer | utility | transform/batch | No existing writer of this shape in the repo (RESEARCH.md Q6: "no existing precedent" for flaky/pass@k reporting); use RESEARCH.md's Code Examples/`eval_gates.go` conventions instead |
| Nested-git / nested-`.aether` trap handling assertion | test | file-I/O | RESEARCH.md Q3/Q4 explicitly names this "genuinely unexplored territory" — no existing detection code to point at; treat as exploratory per Open Question 3 |
| Revert-loop harness for UED-09 (staging old commits per fix, in a throwaway worktree) | utility (test orchestration) | event-driven (per-fix loop) | No existing revert-and-rerun harness in this repo; RESEARCH.md recommends a throwaway worktree per CLAUDE.md's own "Concurrent sessions" / gate-worktree-path lesson, but there is no committed script doing this today to copy from |

## Metadata

**Analog search scope:** `scripts/`, `cmd/eval_gates.go`, `cmd/testdata/eval-gates/`, `Makefile`, `.aether/docs/publish-update-runbook.md`, `cmd/testdata/stop-hook/`
**Files scanned:** `scripts/proof-screens-reach-the-owner.sh` (full), `scripts/smoke-daily-driver.sh` (referenced, not re-read — already fully read during research per RESEARCH.md sources), `cmd/eval_gates.go` (header + constants), `cmd/testdata/eval-gates/gates.json` (full), `Makefile` (eval-gate section), `cmd/testdata/stop-hook/menu-command-transcript.jsonl` (referenced from research)
**Pattern extraction date:** 2026-09-22
