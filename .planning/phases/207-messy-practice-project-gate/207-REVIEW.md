---
phase: 207-messy-practice-project-gate
reviewed: 2026-09-22T00:00:00Z
depth: standard
files_reviewed: 19
files_reviewed_list:
  - cmd/journey.go
  - cmd/journey_traps.go
  - cmd/journey_seed.go
  - cmd/journey_expected_red.go
  - cmd/journey_fix_reverts.go
  - cmd/journey_live_test.go
  - cmd/journey_test.go
  - cmd/journey_expected_red_test.go
  - cmd/journey_fix_reverts_test.go
  - cmd/messy_practice_project_test.go
  - cmd/eval_gates.go
  - cmd/testdata/eval-gates/gates.json
  - cmd/testdata/journey/traps.json
  - cmd/testdata/journey/expected-red.json
  - cmd/testdata/journey/fix-reverts.json
  - scripts/build-messy-practice-project.sh
  - scripts/prove-journey-catches-the-2026-09-21-fixes.sh
  - Makefile
  - .aether/docs/publish-update-runbook.md
findings:
  critical: 2
  warning: 3
  info: 0
  total: 5
status: issues_found
---

# Phase 207: Code Review Report

**Reviewed:** 2026-09-22
**Depth:** standard
**Files Reviewed:** 19
**Status:** issues_found

## Summary

This phase builds a genuinely impressive, disciplined piece of testing infrastructure: a
messy-practice-project builder, a trap manifest, a live end-to-end lifecycle harness, an
expected-red register, a fix-revert proof harness, and a new `journey` eval gate — all
cross-referenced so the manifest, the shell script, the Go checkers and the offline tests can
never silently drift apart. The shell scripts respect the stated safety rules: neither script
ever runs `git checkout`/`stash`/`reset`/`restore` against the owner's own checkout, the revert
harness works only inside a disposable `git worktree`, and the builder script refuses to write
into a foreign, marker-less directory. `go vet` is clean under both the default and `journey`
build tags, and the transcript-parsing logic in `cmd/journey.go` is exercised against a real,
captured Claude Code transcript rather than a hand-typed fixture.

However, direct testing surfaced two genuine defects that block shipping as-is, plus three
smaller quality/consistency issues:

1. **A new hidden production command pair can corrupt a real project's specification and
   survey state**, with no safety guard, shipped unconditionally in the release binary
   (confirmed live against this very checkout).
2. **The new offline test suite fails deterministically** when run together with other tests
   in the same `cmd` package (exactly the invocation given for this review), due to an
   environment-variable leak that poisons every subsequently-spawned real `aether` binary
   subprocess.

Both were reproduced directly, not inferred from reading code.

## Critical Issues

### CR-01: Hidden production commands can fabricate approved specification history and overwrite real survey data in any project, with no guard

**File:** `cmd/journey_seed.go:31-56` (command registration), `:58-114` (`journeySeedStaleSurvey`), `:141-253` (`journeySeedSupersededPlan`)

**Issue:** `cmd/journey_seed.go` carries no `//go:build` constraint — it compiles into every
`aether` binary, including what ships to users. It registers two `Hidden: true` cobra
subcommands, `journey-seed-stale-survey <repo-root>` and `journey-seed-superseded-plan
<repo-root>`, that take an arbitrary repository-root argument and mutate real state there with
no confirmation, no marker-file check, and no restriction to a throwaway/practice project:

- `journey-seed-stale-survey` overwrites `<repo-root>/.aether/data/survey/territory-snapshot.json`
  and every other required territory artifact with placeholder content the code's own comment
  calls "irrelevant" — replacing a real project's actual codebase map with junk that still
  passes the digest self-consistency check, so downstream code (and any worker briefed from it)
  would trust it.
- `journey-seed-superseded-plan` calls the real specification writers
  (`createSpecificationDraft` / `approveSpecification` / `reviseSpecification`) to create and
  **auto-approve** two full specification revisions (`ApprovedBy: "journey-trap"`) directly
  into `<repo-root>`'s real specification lineage, and parks a planning run against them — i.e.
  it can inject a fabricated, indistinguishable-from-real "owner-approved" specification into
  any real, active colony.

Confirmed live against this exact checkout (not a hypothetical):

```
$ go run ./cmd/aether journey-seed-stale-survey --help
...
Usage:
  aether journey-seed-stale-survey <repo-root> [flags]
```

The file's own comment cites `cmd/abandon_cmd.go` and `cmd/hook_cmds.go` as precedent for
`Hidden: true`, but that precedent does not hold: `abandonCmd` is explicitly documented as
"remains parseable only to explain... It never invokes that route and never changes colony
state" and carries `Annotations: {"aether.io/read-only": "true", "aether.io/internal-only":
"true"}`. `hook_cmds.go`'s hidden commands act on the current session's own repo context, never
on an attacker/user-supplied path argument. Neither precedent ships a command that mutates an
arbitrary caller-supplied directory's approval history.

This is exactly the kind of untrusted-input trust-boundary issue CLAUDE.md's own Definition of
Done calls out ("Anything a worker or a wrapper can influence... Treat it as untrusted input"):
a script, a misremembered command, or a curious `aether --help`-reading user pointed at a real
project would silently corrupt that project's specification and survey state.

**Fix:** Gate this file out of release builds entirely (e.g. `//go:build journey`, updating
`scripts/build-messy-practice-project.sh`'s own `go build -o "$BIN" ./cmd/aether` call to add
`-tags=journey` so the practice-project builder still gets it), and/or add a hard runtime guard
requiring the target directory to already carry a recognizable practice-project marker (the
same `.journey-practice-project.json` the builder script itself writes and checks) before either
seed function is allowed to mutate anything:

```go
func journeySeedStaleSurvey(root string) error {
    canonicalRoot, err := canonicalTerritoryRoot(root)
    if err != nil {
        return fmt.Errorf("resolve repository root: %w", err)
    }
    if _, err := os.Stat(filepath.Join(canonicalRoot, ".journey-practice-project.json")); err != nil {
        return fmt.Errorf("refusing to seed a stale survey into %s: no .journey-practice-project.json marker -- this command only operates on a practice project the builder script created", canonicalRoot)
    }
    ...
```

### CR-02: The new offline test suite fails deterministically when run with other `cmd` package tests, due to an environment leak

**File:** `cmd/messy_practice_project_test.go:97-127` (`journeyTestSharedProject`), interacting
with `cmd/journey_expected_red_test.go` (any test using `newTestStore`, e.g. line 66) and
`cmd/journey_live_test.go`'s equivalent build/install steps; root cause in the pre-existing
`newTestStore` helper (`cmd/write_cmds_test.go:21-35`, not in this phase's diff).

**Issue:** Reproduced directly with the exact command given for this review:

```
$ go test ./cmd -run 'TestSixthBlockerCheckIsStillRed|TestMessyPracticeProjectHasEveryTrap' -count=1 -timeout 200s -v
--- PASS: TestSixthBlockerCheckIsStillRed (0.21s)
--- FAIL: TestMessyPracticeProjectHasEveryTrap (2.22s)
    ==> installing package into isolated hub
    Error: failed to validate repository store: storage: repository containment refused: data root path must not be empty
    BUILD FAIL: aether install failed
```

One prior test that calls `newTestStore` is sufficient to break every later test in the same
binary that spawns a real `aether` subprocess. `TestMessyPracticeProjectHasEveryTrap` passes in
isolation (91.79s) and fails in 2.3s when run after `TestSixthBlockerCheckIsStillRed`.

Root cause: `newTestStore` (`cmd/write_cmds_test.go:21-35`) saves the pre-test value of
`COLONY_DATA_DIR` with `os.Getenv` (which cannot distinguish "unset" from "set to empty") and
restores it in `t.Cleanup` with `os.Setenv("COLONY_DATA_DIR", origColonyDataDir)`. When the var
was originally unset (the normal case), this *sets* it to `""` instead of unsetting it — and
that leaked, explicitly-empty env var then persists for the rest of the `go test` process. The
in-process guard in `cmd/root.go` (`colonyDataDirConfiguredAtProcessStart`) was written
precisely to tolerate this leak for cobra commands invoked *inside* the same test process — but
it cannot protect a **freshly spawned subprocess** (the real `aether-under-test`/`aether-bin`
binary `journeyTestSharedProject` and `journeyRunOneTrial` build and `exec.Command` invoke),
because that subprocess re-evaluates `colonyDataDirConfiguredAtProcessStart` fresh at its own
startup, sees the leaked `COLONY_DATA_DIR=""` as `configured=true`, and takes it as an explicit
(empty) data-root override — exactly the "repository containment refused: data root path must
not be empty" failure observed.

This is not a flaky/incidental failure: it is deterministic, and it will reproduce for anyone
running the reviewed test command (or the default `go test ./cmd/...`) whenever any
`newTestStore`-using test happens to execute before the new journey/messy-practice-project
tests in the same binary — which, given Go's source-order test discovery, is common.

**Fix:** Two independent, complementary fixes:

1. Fix the leak at its source in `newTestStore` (even though outside this phase's file list,
   this phase's own new tests are what first exposes it):
   ```go
   origColonyDataDir, hadColonyDataDir := os.LookupEnv("COLONY_DATA_DIR")
   t.Cleanup(func() {
       if hadColonyDataDir {
           os.Setenv("COLONY_DATA_DIR", origColonyDataDir)
       } else {
           os.Unsetenv("COLONY_DATA_DIR")
       }
   })
   ```
2. Defensively, since this phase's new tests are the first in the package to spawn a real
   built binary as a subprocess, don't trust the ambient/inherited process environment at all:
   in `journeyTestSharedProject` (`cmd/messy_practice_project_test.go`) and in
   `journeyRunOneTrial`'s build/init calls (`cmd/journey_live_test.go`), set `cmd.Env`
   explicitly (e.g. a filtered copy of `os.Environ()` with `COLONY_DATA_DIR` removed) rather
   than relying on whatever any other test in the package left behind.

## Warnings

### WR-01: `eval-gate-journey`'s Makefile budget is stale and inconsistent with `gates.json`

**File:** `Makefile` (the `eval-gate-journey` target and its preceding comment block)

**Issue:** Every other eval gate keeps its Makefile `-timeout=<N>s` value identical to that
gate's `budget_seconds` in `cmd/testdata/eval-gates/gates.json` (fast: 60/60, focused: 300/300,
integration: 1400/1400, provider: 1400/1400, overnight: 7200/7200, race: 1260/1260, release:
1800/1800). `eval-gate-journey` breaks this pattern: the Makefile uses `-timeout=5400s` and its
comment says `Budget 5400s, provisional (gates.json)`, while `gates.json`'s own `journey` entry
declares `"budget_seconds": 1100` with an explicit note that this figure is *"measured, not
provisional"*, derived from the real 207-06 three-trial run. The Makefile comment was not
updated when the manifest was finalized with the real measured figure, leaving a
self-contradicting pair (one file says "provisional 5400", the other says "measured, not
provisional, 1100") that a future maintainer will trust the wrong half of.

**Fix:** Update the Makefile's comment and `-timeout` to match `gates.json`'s `1100`, or (if a
larger process-level safety margin is intentional) change the comment to say so explicitly and
stop calling it "provisional":
```make
# journey: ... Budget 1100s, measured from 207-06-PLAN.md's real three-trial run
# (gates.json) -- -timeout is set with headroom above that measured figure.
eval-gate-journey:
	$(call EVAL_GATE_CHECK,journey,-tags=journey -run=TestJourney -count=1 -timeout=1500s ./...,-tags=journey -list=TestJourney ./...)
```

### WR-02: The persisted failure classification can disagree with the retry decision that was actually made

**File:** `cmd/journey_live_test.go:1009-1047` (`journeyRunClaudeWithRetry`), `:295-337`
(`journeyDriveStep`)

**Issue:** `journeyRunClaudeWithRetry` decides whether to retry by classifying **combined
stdout+stderr** (`classifyJourneyFailure(string(out)+"\n"+string(errOut))`), but only returns
`out` (stdout). `journeyRunStep` writes exactly that returned `out` to `streamOutPath`, and
`journeyDriveStep` later derives the *persisted* `result.FailureKind` from that same file
(`classifyJourneyFailure(string(streamed))`) — stderr is never available at that point. If the
diagnostic text for a genuinely transient failure (rate limit / overloaded / timed out) appears
only on stderr of the final attempt — a common place for a CLI to emit such messages — the live
retry logic would correctly detect and act on it as transient, while the persisted report would
independently reclassify the same failure as `"real"`. Since `journeyGateVerdict` refuses the
whole run outright on any single `real-failure` trial, this discrepancy can cause the release
gate to be refused for what the harness's own retry logic already identified as noise,
undermining the documented "failures sorted into flaky and real" guarantee.

**Fix:** Return (or otherwise persist) the same combined stdout+stderr text
`journeyRunClaudeWithRetry` used for its own retry decision, and derive
`result.FailureKind` from that instead of from stdout alone — e.g. write both streams to
`streamOutPath` (or a sibling file) and classify from the concatenation, so the retry decision
and the persisted classification can never diverge.

### WR-03: The harness safety-net test doesn't check for `git clean`

**File:** `cmd/journey_fix_reverts_test.go:335-350` (`TestFixRevertHarnessNeverTouchesTheOwnersCheckout`)

**Issue:** The test's forbidden-operations list is `{"git checkout", "git stash", "git reset",
"git restore"}`. The script itself does not use `git clean` today, so this is not an active
bug, but the stated safety contract (both in this review's own scope and in the script's own
WORKING-COPY RULE comment) is about protecting the owner's checkout from any
working-copy-switching or content-destroying git operation. `git clean -fd` would silently
delete untracked files in `$ROOT` and is at least as dangerous as the four operations already
guarded against, yet a future edit adding it would pass this test unnoticed.

**Fix:** Add `"git clean"` to the forbidden list in both the test and the script's own
WORKING-COPY RULE comment, matching the five operations (`checkout`/`stash`/`reset`/`restore`/`clean`)
that must never run against `$ROOT`.

---

_Reviewed: 2026-09-22_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
