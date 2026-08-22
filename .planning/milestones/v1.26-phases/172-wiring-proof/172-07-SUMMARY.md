---
phase: 172-wiring-proof
plan: "07"
subsystem: testing
tags: [go, ci, github-actions, static-analysis, wiring-proof, gap-closure]

# Dependency graph
requires:
  - phase: 172-wiring-proof (plan 05)
    provides: "TestWiringGateStepRunsEveryWiringTest, the named CI step, and wiringGateGuardFiles — this plan replaces the defective whole-file substring check inside the first and widens the escape-hatch scan that never used the second"
provides:
  - "blanketReleaseGateProblem(workflow string) error — a step-scoped, failing-capability-aware check that the blanket `go test ./...` release-gate step is present under its exact name and structurally capable of failing the build (no `if:`, no continue-on-error, no exit-status-swallowing fallback)"
  - "stepBlock(workflow, stepName string) (string, error) — the single, end-of-line-anchored step locator in cmd/ci_wiring_gate_test.go, replacing the unanchored strings.Index lookup that would have resolved a deleted `Run Go tests` step to `Run Go tests with race detection`"
  - "TestBlanketGateCheckRejectsADecoyStep — a hermetic, 6-subtest table test proving the decoy, the prefix collision, and each disabling condition are rejected, with a happy-path row proving the check isn't rejecting everything"
  - "TestWiringGuardsHaveNoRuntimeEscapeHatch now scans all five guard files this phase created (via the shared wiringGateGuardFiles inventory, with a <5-entries t.Fatalf floor) against seven escape-hatch spellings instead of three files against three spellings"
  - "T-172-18 honestly re-dispositioned: 172-05's 'mitigated by presence' claim recorded as false, replaced by a hermetic test and a reproduced red/green transcript"
affects: []

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Single shared step-locator function (stepBlock) used by both the -run coverage check and the failing-capability check, instead of two independently-drifting scans of the same workflow text"
    - "Escape-hatch pattern literals written with the `\\.` escape (matching the pre-existing buildConstraintRe concatenation trick) specifically so a widened pattern's own declaration line cannot self-match when the guard scans its own file"
    - "Hermetic table tests over synthetic input strings built in the test body (not fixture files) for guard logic that would otherwise only be proven once by a manual transcript"

key-files:
  created: []
  modified:
    - cmd/ci_wiring_gate_test.go
    - cmd/subcommand_reachability_ratchet_test.go
    - .github/workflows/ci.yml
    - .planning/phases/172-wiring-proof/deferred-items.md

key-decisions:
  - "stepBlock anchors the step-name marker to end-of-line by requiring the character immediately after the marker to be a newline or end-of-file, rather than using a `(?m)^...$` regexp. Chosen so the file's step locator remains a single plain-string scan (grep -c '\"- name: \"' returns exactly 1, per this task's own acceptance criterion) rather than mixing a regexp-based locator with the pre-existing string-based one."
  - "blanketReleaseGateProblem and extractWiringGateRunArg both call stepBlock and a shared runLineOf helper rather than each re-deriving the run: line independently — the task's acceptance criterion required exactly one step locator in the package, and a second independent run-line extractor would have been the same duplication in a different place."
  - "The widened forbiddenRe's new alternatives (os.LookupEnv, os.Environ, syscall.Getenv, testing.Short) are each written with the `\\.` escape, matching how the original three-entry pattern and buildConstraintRe already avoided self-matching — no alternative needed the fallback string-concatenation form the plan flagged as an option."
  - "Logged (not fixed) an unrelated pkg/codex test flake found while running the plan's own `go test ./... -count=1 -timeout 900s` acceptance check twice: TestCodexReadOnlyProfileSelectsReadOnlySandbox times out only under full-suite resource contention, passes in well under a second in isolation, and pkg/codex is untouched by this plan (confirmed via git log). Recorded in deferred-items.md per the Scope Boundary rule rather than investigated or fixed, since it is outside this plan's files_modified and outside cmd, which was fully green in both full-suite runs."

requirements-completed: [WIRE-01, WIRE-02, WIRE-03]

# Metrics
duration: ~25min
completed: 2026-08-11
---

# Phase 172 Plan 07: CI Wiring-Gate Hardening + Guard-Inventory Closure Summary

**Replaced the wiring gate's defective whole-file substring check with a step-scoped, failing-capability-aware check that a synthetic decoy cannot satisfy, and widened the escape-hatch scan from three files/three patterns to the full five-file guard inventory against seven patterns.**

## Performance

- **Duration:** ~25 min
- **Completed:** 2026-08-11
- **Tasks:** 2 (each landed in its own commit)
- **Files modified:** 4 (0 created, 4 modified — 3 source/workflow files, 1 tracking doc)

## Pre/Post Working-Tree Deletion Count (mandated check)

Before any commit in this plan: `git status --porcelain | wc -l` → **0** (this worktree was reset to the phase's clean base commit `08d6bfd2`; no pre-existing pending deletions).

After the final commit: `git status --porcelain | wc -l` → **0**.

`git status --porcelain .github/workflows/ci.yml cmd/spawn_enforce_test.go` returned no output after the final commit — every red-proof mutation across both tasks was reverted with a path-scoped edit, never a blanket `git checkout` or `git clean`.

## Accomplishments

- Task 1: Replaced `TestWiringGateStepRunsEveryWiringTest`'s unscoped whole-file `strings.Contains` check with `blanketReleaseGateProblem`, a pure function that locates the `Run Go tests` step by exact end-of-line-anchored name, confirms its run line contains the real command (and is not the `-race` step), and rejects `if:`, `continue-on-error`, and exit-status-swallowing fallbacks (`|| true`, `|| echo`, `|| :`).
- Added `stepBlock`, the single step locator now shared by both `blanketReleaseGateProblem` and `extractWiringGateRunArg`, anchored so `Run Go tests` cannot resolve to `Run Go tests with race detection` once the real step is deleted.
- Added `TestBlanketGateCheckRejectsADecoyStep`, a hermetic 6-subtest table test (decoy, prefix-collision, happy path, `if: always()`, `continue-on-error`, `|| true`) proving the fix holds on every CI run, not just once.
- Wired the new test into the named CI step's `-run` filter in `.github/workflows/ci.yml`.
- Reproduced and reverted all three of Task 1's mandatory red proofs (below).
- Task 2: Deleted the local three-entry `guardFiles` slice inside `TestWiringGuardsHaveNoRuntimeEscapeHatch` and switched it to iterate the shared `wiringGateGuardFiles` (all five guard files this phase created), with a `t.Fatalf` floor if the inventory ever drops below five entries.
- Widened `forbiddenRe` from `os\.Getenv|t\.Skip|t\.SkipNow` to also cover `os\.LookupEnv`, `os\.Environ`, `syscall\.Getenv`, and `testing\.Short`.
- Reproduced and reverted both of Task 2's mandatory red proofs (below), and confirmed the widened pattern passes on the clean tree — it does not self-match its own declaration.

## Task Commits

1. **Task 1: Scope the blanket-gate check to the real step and prove a decoy cannot satisfy it** — `8fdd247a` (fix)
2. **Task 2: Scan every guard file this phase created, from one inventory, for every spelling of an escape hatch** — `ac5b61f2` (fix)

## Files Created/Modified

- `cmd/ci_wiring_gate_test.go` — added `stepBlock`, `runLineOf`, `blanketReleaseGateProblem`, `blanketGateStepName`, `blanketGateRunSubstring`, `nameMarkerPrefix`, `TestBlanketGateCheckRejectsADecoyStep`; rewrote `extractWiringGateRunArg` to call `stepBlock`; updated `TestWiringGateStepRunsEveryWiringTest`'s doc comment and body.
- `cmd/subcommand_reachability_ratchet_test.go` — `TestWiringGuardsHaveNoRuntimeEscapeHatch` now iterates `wiringGateGuardFiles` with an anti-vacuity floor, and `forbiddenRe` covers seven spellings instead of three.
- `.github/workflows/ci.yml` — added `TestBlanketGateCheckRejectsADecoyStep` to the named wiring step's `-run` alternation. No other line changed.
- `.planning/phases/172-wiring-proof/deferred-items.md` — logged the unrelated `pkg/codex` full-suite-only flake.

## Required Red Proofs — Task 1 (blanket gate check)

All three run against the real `.github/workflows/ci.yml`, each restored with a path-scoped Python edit (never `git checkout .`) before the next.

**Red proof 1 — delete the real `Run Go tests` step, leave the `Test summary` decoy in place** (the exact reproduction from 172-VERIFICATION.md gap 2):

```
$ go test ./cmd -run TestWiringGateStepRunsEveryWiringTest -count=1 -v
=== RUN   TestWiringGateStepRunsEveryWiringTest
    ci_wiring_gate_test.go:97: .github/workflows/ci.yml has no step named "Run Go tests" — the blanket release gate has been removed or renamed; the named wiring step adds legibility to an existing failure, it does not replace the coverage that finds it
--- FAIL: TestWiringGateStepRunsEveryWiringTest (0.00s)
FAIL
```

Restored (step re-inserted with a path-scoped edit); re-ran:

```
=== RUN   TestWiringGateStepRunsEveryWiringTest
--- PASS: TestWiringGateStepRunsEveryWiringTest (0.01s)
PASS
```

`git diff .github/workflows/ci.yml` after restoration showed only this plan's intended 1-line `-run` addition (`TestBlanketGateCheckRejectsADecoyStep`) — no residue from the deletion/restoration cycle.

**Red proof 2 — add `if: always()` to the `Run Go tests` step:**

```
=== RUN   TestWiringGateStepRunsEveryWiringTest
    ci_wiring_gate_test.go:97: CI step "Run Go tests" has a conditional if: always() — a conditional step can be arranged not to run, so it cannot be relied on to gate
--- FAIL: TestWiringGateStepRunsEveryWiringTest (0.00s)
FAIL
```

Removed; re-ran:

```
=== RUN   TestWiringGateStepRunsEveryWiringTest
--- PASS: TestWiringGateStepRunsEveryWiringTest (0.01s)
PASS
```

**Red proof 3 — append `|| true` to the `Run Go tests` run line:**

```
=== RUN   TestWiringGateStepRunsEveryWiringTest
    ci_wiring_gate_test.go:97: CI step "Run Go tests"'s run line routes its exit status through "|| true", a fallback that cannot fail: run: go test ./... -count=1 -timeout 900s || true
--- FAIL: TestWiringGateStepRunsEveryWiringTest (0.00s)
FAIL
```

Removed; re-ran:

```
=== RUN   TestWiringGateStepRunsEveryWiringTest
--- PASS: TestWiringGateStepRunsEveryWiringTest (0.01s)
PASS
```

**Anti-regression note:** under the pre-fix `strings.Contains(workflow, "go test ./... -count=1 -timeout 900s")` logic, red proof 1's exact mutation (deleting `Run Go tests`, leaving `Test summary`'s decoy substring in place) passed — this was independently reproduced by the verifier and is recorded as 172-VERIFICATION.md gap 2 and as the (false) "mitigated" claim in 172-05-SUMMARY.md's threat register. The new failure in all three red proofs above is caused entirely by this task's `blanketReleaseGateProblem` replacing that check — no other file changed between the pre-fix and post-fix runs.

## Required Red Proofs — Task 2 (guard-inventory + pattern widening)

**Red proof 1 — insert `os.LookupEnv("AETHER_SKIP_WIRING")` into `cmd/ci_wiring_gate_test.go`** (the guard file the old three-entry list never scanned):

```
$ go test ./cmd -run TestWiringGuardsHaveNoRuntimeEscapeHatch -count=1 -v
=== RUN   TestWiringGuardsHaveNoRuntimeEscapeHatch
    subcommand_reachability_ratchet_test.go:1225: ci_wiring_gate_test.go:259 contains a runtime escape hatch (an environment-variable read or a test-skip call): _, _ = os.LookupEnv("AETHER_SKIP_WIRING")
--- FAIL: TestWiringGuardsHaveNoRuntimeEscapeHatch (0.01s)
FAIL
```

This fails twice over under the pre-fix code: the file was not in the scanned list at all, and `os.LookupEnv` was not in the pattern even for files that were scanned. Removed; re-ran:

```
=== RUN   TestWiringGuardsHaveNoRuntimeEscapeHatch
--- PASS: TestWiringGuardsHaveNoRuntimeEscapeHatch (0.01s)
PASS
```

`git status --porcelain cmd/ci_wiring_gate_test.go` returned no output afterward — byte-identical to the committed Task 1 state.

**Red proof 2 — insert `t.Skip("temporary")` as the first statement of `TestSpawnCanSpawnAcceptsDocumentedInvocation` in `cmd/spawn_enforce_test.go`** (the other previously-unscanned file):

```
=== RUN   TestWiringGuardsHaveNoRuntimeEscapeHatch
    subcommand_reachability_ratchet_test.go:1225: spawn_enforce_test.go:65 contains a runtime escape hatch (an environment-variable read or a test-skip call): t.Skip("temporary")
--- FAIL: TestWiringGuardsHaveNoRuntimeEscapeHatch (0.01s)
FAIL
```

Removed; re-ran:

```
=== RUN   TestWiringGuardsHaveNoRuntimeEscapeHatch
--- PASS: TestWiringGuardsHaveNoRuntimeEscapeHatch (0.01s)
PASS
```

**Anti-self-trip proof:** on the clean tree (both red-proof mutations reverted), `go test ./cmd -run TestWiringGuardsHaveNoRuntimeEscapeHatch -count=1 -v` **PASSES**. The widened pattern's own declaration line reads `os\.Getenv|os\.LookupEnv|os\.Environ|syscall\.Getenv|t\.Skip|t\.SkipNow|testing\.Short` — every alternative is written with the `\.` (backslash-dot) escape, so the *literal source text* contains a backslash character immediately before each dot. The regexp alternative `os\.Getenv` (as a *pattern*) requires a literal dot with no backslash, so it does not match the string `os\.Getenv` (which contains a backslash), matching how the pre-existing `buildConstraintRe`'s string-concatenation trick already avoided self-matching. No alternative required the fallback concatenation form the plan allowed for.

## T-172-18 Re-Disposition (mandatory statement)

**172-05-SUMMARY.md's threat register recorded T-172-18 as "Mitigated by the assertion that the blanket step is still present." This was false.** Independently reproduced by the verifier and reproduced again here in Task 1's red proof 1: with only the `Run Go tests` step's two lines deleted from `.github/workflows/ci.yml`, `TestWiringGateStepRunsEveryWiringTest` stayed green, because the whole-file `strings.Contains(workflow, "go test ./... -count=1 -timeout 900s")` check was satisfied by the same substring sitting inside the non-blocking `Test summary` step's `echo "Go tests: $(go test ./... -count=1 -timeout 900s ...) passed"` line — a step that runs `if: always()` and pipes its result through `grep -c '--- PASS' || echo 'unknown'`, whose exit status can never fail the job.

**T-172-18 is now genuinely mitigated by `blanketReleaseGateProblem`** (`cmd/ci_wiring_gate_test.go`), which locates the step named exactly `Run Go tests` via the end-of-line-anchored `stepBlock`, asserts its own run line contains the real command, and rejects `if:`, `continue-on-error`, and any exit-status-swallowing fallback on that step specifically — the `Test summary` step's decoy text is never consulted, because the check is scoped to the named step's block, not the whole file. This is carried forward by a hermetic test (`TestBlanketGateCheckRejectsADecoyStep`), not by a one-time transcript: the decoy case is now a permanent row in that table, asserted on every `go test ./cmd` run.

## Decisions Made

See `key-decisions` in the frontmatter for the full list. Summarized:
- `stepBlock` anchors via next-character-after-marker rather than a `(?m)^...$` regexp, keeping the file's step locator a single plain-string scan (the acceptance criterion `grep -c '"- name: "' cmd/ci_wiring_gate_test.go` returns exactly 1).
- `blanketReleaseGateProblem` and `extractWiringGateRunArg` share `stepBlock` and a `runLineOf` helper — one step locator in the package, per the task's acceptance criterion.
- The four new `forbiddenRe` alternatives all use the `\.` escape form; none needed the string-concatenation fallback the plan allowed for.
- Logged (not fixed) an unrelated `pkg/codex` flake discovered while running the plan's own `go test ./... -count=1 -timeout 900s` acceptance check — see Issues Encountered and `deferred-items.md`.

## Deviations from Plan

None — plan executed exactly as written. The only unplanned discovery (the `pkg/codex` flake below) was out of scope per the Scope Boundary rule and was logged, not fixed.

## Issues Encountered

**Unrelated `pkg/codex` test flake under full-suite load.** Running the plan's own acceptance command `go test ./... -count=1 -timeout 900s` (required by both tasks' `<acceptance_criteria>`) failed both times it was run, at `TestCodexReadOnlyProfileSelectsReadOnlySandbox`:

```
--- FAIL: TestCodexReadOnlyProfileSelectsReadOnlySandbox (3.00s)
    permission_profile_test.go:111: worker startup failed: codex login status failed: timed out; sensitive details omitted
FAIL
FAIL	github.com/calcosmic/Aether/pkg/codex	24.926s
```

Run in isolation (`go test ./pkg/codex -run TestCodexReadOnlyProfileSelectsReadOnlySandbox -count=1 -v`), it passes in well under a second. `pkg/codex/permission_profile_test.go` was last touched by an unrelated 163-03 commit (confirmed via `git log`) and is nowhere in this plan's `files_modified`. `go test ./cmd` — the package every file this plan touched lives in — was fully green in both full-suite runs (298s and 332s, 0 failures). This is a pre-existing, environment/resource-contention-dependent flake (a subprocess `codex login status` check timing out when run concurrently with the ~300s `cmd` package under this sandboxed worktree), not something this plan's changes could cause. Logged to `.planning/phases/172-wiring-proof/deferred-items.md` per the Scope Boundary rule rather than investigated further.

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness

- **Verification gap 2 (172-VERIFICATION.md) is closed.** ROADMAP success criterion 4's durability half is now TRUE: the blanket release-gate step is verified capable of failing the build via `blanketReleaseGateProblem`, not merely present as text, and a hermetic table test (`TestBlanketGateCheckRejectsADecoyStep`) keeps that true on every CI run.
- **D-11's "no escape hatch" now covers all five guard files this phase created**, from one shared inventory (`wiringGateGuardFiles`), against seven escape-hatch spellings — including the one guard (`cmd/ci_wiring_gate_test.go`) that could previously have been `t.Skip`'d with nothing noticing.
- **Full release-gate suite:** `cmd` package green on both full-suite runs (18 packages total; the one `pkg/codex` failure is a pre-existing, unrelated, load-dependent flake, logged in `deferred-items.md`, not blocking this plan's own scope).
- No further known gaps against this plan's own threat register (T-172-18, T-172-31 through T-172-35) — all six are mitigated and covered by the tests named in this summary.

## Working-Tree Safety (mandated section)

- This worktree started at the phase's clean base commit `08d6bfd2` (`git status --porcelain | wc -l` = 0) with no pre-existing pending deletions.
- Every mutation performed during the five red-proof reproductions (two in `.github/workflows/ci.yml`, one further in the same file, two across `cmd/ci_wiring_gate_test.go` and `cmd/spawn_enforce_test.go`) was made with explicit, path-scoped Python file writes verified to match exactly one occurrence before writing — no `git stash`, `git reset --hard`, `git checkout .`, `git clean`, or blanket `git add`. Every commit staged only the exact files named in this summary.
- `git status --porcelain | wc -l` was **0** before this plan's first commit and **0** after its last commit.

---
*Phase: 172-wiring-proof*
*Completed: 2026-08-11*

## Self-Check: PASSED

- FOUND: `cmd/ci_wiring_gate_test.go`
- FOUND: `cmd/subcommand_reachability_ratchet_test.go`
- FOUND: `.planning/phases/172-wiring-proof/172-07-SUMMARY.md`
- FOUND commit: `8fdd247a` (Task 1)
- FOUND commit: `ac5b61f2` (Task 2)
