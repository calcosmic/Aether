---
phase: 188-one-truth-for-failures-and-advances
plan: 04
subsystem: testing
tags: [go, go-ast, static-analysis, ratchet, colony-state, atomicity, call-graph]

# Dependency graph
requires:
  - phase: 188-one-truth-for-failures-and-advances (plan 02)
    provides: "advancePhase() -- the single named reachability root this plan's zero-tolerance check is rooted at"
provides:
  - "cmd/colony_state_atomicity_ratchet_test.go -- AST-based structural detector for non-atomic COLONY_STATE.json writes"
  - "TestColonyStateWriteAllowlistOnlyShrinks -- shrink-only baseline ratchet, self-regenerating via -update-colony-state-write-allowlist"
  - "TestNoFunctionReachableFromAdvancePhaseWritesNonAtomically -- zero-tolerance reachability check, no allowlist exemption possible for advancePhase's reachable set"
  - "cmd/testdata/colony_state_write_allowlist.json -- checked-in baseline of the 30 pre-existing, accepted non-atomic write sites"
affects: [191-ratchet-allowlist-may-only-shrink, future-LOCK-state-locking-migration]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Two-shape AST scanning for cobra command bodies: plain *ast.FuncDecl (RunE: namedFunc) AND inline RunE: func(...) {...} closures on &cobra.Command{...} composite literals found anywhere in a file (var-declared or inline AddCommand argument) -- confirmed necessary by direct inspection, not assumed, after the plan's own claim that 'every call is inside a *ast.FuncDecl' proved empirically false for roughly two-thirds of the real sites"
    - "One combined AST scan produces both the write-site list and the same-package call graph, so cmd/*.go is parsed once for both checks rather than twice"
    - "Zero-tolerance check intersects by function name only, never loading or consulting the allowlist file at all -- structurally impossible to spoof by adding an allowlist entry"

key-files:
  created:
    - cmd/colony_state_atomicity_ratchet_test.go
    - cmd/testdata/colony_state_write_allowlist.json

key-decisions:
  - "Scanner covers two syntactic shapes, not one, after direct inspection (not the plan's own text, which claimed a single shape) showed real sites live both inside ordinary *ast.FuncDecl bodies (e.g. cmd/abandon_cmd.go's RunE: runAbandon) AND inside inline RunE: func(...) {...} closures with no enclosing FuncDecl at all (e.g. cmd/init_cmd.go:300, cmd/worktree.go:220) -- a scanner that only walked FuncDecl bodies would have silently missed roughly 20 of the 30 real sites on day one"
  - "The cobra.Command composite-literal scan matches the shape ANYWHERE in a file (inline AddCommand(&cobra.Command{...}) as well as top-level var declarations), directly closing the exact blind spot 187-VERIFICATION.md's Addendum recorded as still open in the sibling worktree-destruction guard's root-set derivation"
  - "writeColonyStateWriteAllowlist deduplicates by the (File, Function, Primitive) key before writing -- several real functions legitimately contain more than one write call (e.g. runCodexPlanWithOptions has three, suggest_approve.go's RunE has six), and since two calls in the same function already collapse to one key for comparison purposes, a non-deduplicated file would carry repeated, identical JSON blocks that read like a scanner bug"
  - "Selector-name-only matching (SaveJSON/AtomicWrite) regardless of receiver, and call-graph edges recorded by selector name for qualified/method calls -- a deliberately conservative superset, not a precise call graph, per the plan's own explicit sanction"

patterns-established:
  - "Pattern: an AST ratchet's scanner must be verified against the real shapes production code actually uses, not against the plan's prose description of those shapes -- read the plan's factual claims as a hypothesis to confirm by direct inspection, not as ground truth, especially for a guard whose entire purpose is catching category-misses"

requirements-completed: []

# Metrics
duration: ~27min
completed: 2026-08-19
---

# Phase 188 Plan 04: COLONY_STATE Atomicity Ratchet Summary

**AST-based structural ratchet (not a grep) over every non-atomic `COLONY_STATE.json` write site in `cmd/*.go`, plus a zero-tolerance call-graph check that nothing reachable from `advancePhase()` can ever regress to a non-atomic write, no allowlist exemption possible — both proven by four separate adversarial source injections, all restored byte-for-byte.**

## Performance

- **Duration:** ~27 min
- **Started:** 2026-08-19T22:05:52Z
- **Completed:** 2026-08-19T22:32:24Z
- **Tasks:** 2
- **Files modified:** 2 (both new)

## Accomplishments

- A structural (`go/ast`/`go/parser`-based) scanner finds every `store.SaveJSON("COLONY_STATE.json", ...)` and `store.AtomicWrite("COLONY_STATE.json", ...)` call site across `cmd/*.go`, covering two distinct syntactic shapes confirmed necessary by direct inspection (plain functions/methods, and inline `RunE: func(...) {...}` closures on `&cobra.Command{...}` literals, both var-declared and inline).
- A shrink-only, self-regenerating baseline allowlist (`cmd/testdata/colony_state_write_allowlist.json`, 30 entries across 22 files) checked in, with a `-update-colony-state-write-allowlist` flag mirroring the codebase's existing `-update-orphan-allowlist` idiom exactly. The count (30 unique sites from 41 raw call sites before dedup) matches the hand-derived expectation from 188-CONTEXT.md's own repo-wide grep (42 pre-188-02 sites minus the one `codex_continue_finalize.go` site 188-02's D-06 already removed = 41 raw sites) almost exactly.
- A second, independent, zero-tolerance check (`TestNoFunctionReachableFromAdvancePhaseWritesNonAtomically`) that structurally recomputes same-package call-graph reachability from `advancePhase` (188-02's shared phase-advance core) on every run and never loads or consults the allowlist file for anything in that reachable set.
- Both checks fail loudly, never vacuously, if the AST walker finds zero sites or the reachable set collapses to a suspiciously small size.
- All four required adversarial proofs performed against real source, output captured, source restored byte-for-byte and verified clean after each (see Proof section below).

## Task Commits

1. **Task 1: Baseline, shrink-only allowlist ratchet over every non-atomic COLONY_STATE.json write site** — `33a13bb5` (test)
2. **Task 2: Zero-tolerance reachability check from advancePhase()** — `28ab4ddd` (test)

_Both tasks touch the same file (`cmd/colony_state_atomicity_ratchet_test.go`) by the plan's own design — Task 2 explicitly reuses Task 1's scanner and parsed-file data rather than re-parsing the package a second time. Committed as two separate, verified-standalone diffs: Task 1's commit was built and tested in isolation (file temporarily trimmed to only Task 1's content, `go build`/`go vet`/`TestColonyStateWriteAllowlistOnlyShrinks` all confirmed passing alone) before Task 2's content was restored and committed as a pure addition (93 insertions, 0 deletions/modifications to Task 1's code)._

**Plan metadata:** (this commit, SUMMARY.md only per this plan's parallel-executor instructions — STATE.md/ROADMAP.md are owned by the orchestrator)

## Files Created/Modified

- `cmd/colony_state_atomicity_ratchet_test.go` (588 lines) — the two-check ratchet: `scanColonyStateSource`/`scanFileForColonyStateWrites`/`recordColonyStateCallsAndSites` (combined AST scanner producing both the site list and the call graph in one parse pass), `findColonyStateWriteSites`, `loadColonyStateWriteAllowlist`/`writeColonyStateWriteAllowlist`, `TestColonyStateWriteAllowlistOnlyShrinks`, `colonyStateReachableFrom` (BFS), `TestNoFunctionReachableFromAdvancePhaseWritesNonAtomically`.
- `cmd/testdata/colony_state_write_allowlist.json` (30 entries) — the checked-in baseline, generated by the scanner's own `-update-colony-state-write-allowlist` flag, not hand-written.

## Decisions Made

- **The scanner covers two syntactic shapes, discovered by direct inspection rather than trusting the plan's own text.** The plan's `<action>` section for Task 1 states: "every one of these calls is inside a `*ast.FuncDecl` in this codebase." Before writing the scanner, I checked this claim against several of the real sites named in 188-CONTEXT.md's canonical_refs grep list. It was false for the majority of them: `cmd/init_cmd.go:300` and `cmd/worktree.go:220` (among many others) live inside `RunE: func(cmd *cobra.Command, args []string) error {...}` inline closures attached to `var X = &cobra.Command{...}` composite literals — there is no enclosing `*ast.FuncDecl` at all for these. A scanner that only walked `FuncDecl` bodies (as the plan's own model description would have produced) would have silently missed roughly 20 of the 30 real sites, and — much more importantly — would have missed any *future* write added inside a new cobra command's `RunE` closure, which is the single most common place a new CLI command's logic lives in this codebase. The scanner was built with two passes from the start to close this before it ever shipped, matching the `<why_this_plan_is_the_important_one>` brief's explicit warning: "Entry-point discovery that only recognises one syntactic form misses the others."
- **The `&cobra.Command{...}` composite-literal scan matches anywhere in a file, not only top-level `var` declarations.** `cmd/host_cmd.go` registers nine subcommands via inline `AddCommand(&cobra.Command{...})` composite literals. `187-VERIFICATION.md`'s Addendum recorded this exact shape as a confirmed, still-open blind spot in the sibling `worktree_destruction_reachability_test.go` guard's root-set derivation (`indexCobraCommandLiteral` is only invoked from the `var`-declaration path there). This ratchet's scan is a whole-file `ast.Inspect` for the composite-literal shape itself, independent of what container it sits inside, so it does not repeat that specific miss even though no currently-shipped `COLONY_STATE.json` write happens to live in `host_cmd.go` today.
- **`writeColonyStateWriteAllowlist` deduplicates by `(File, Function, Primitive)` before writing.** The raw scan found 41 call sites; three real functions legitimately contain more than one write call at different lines (`runCodexPlanWithOptions` has three, `colony_cmds.go`'s `set` command's `RunE` has three, `suggest_approve.go`'s `RunE` has six). Since D-08 keys allowlist entries by file+function (not line number), two calls in the same function already collapse to the same comparison key — writing them as separate, byte-identical JSON blocks would read like a scanner bug to a future reviewer rather than the genuine multiple call sites they represent. The comparison logic (set-membership, unaffected either way) was verified unchanged by this choice.
- **A log-message accuracy bug was caught and fixed before the first commit.** `writeColonyStateWriteAllowlist`'s caller originally logged `len(sites)` (the raw, pre-dedup count, 41) as the number of entries written, when the file actually held 30 deduplicated entries. Fixed by having the write function return the true written count and logging that instead — both counts are now reported explicitly ("wrote ... with 30 unique (file, function, primitive) entries from 41 raw scanned site(s)").

## Proof: All Four Required Adversarial Injections

Every injection was performed against real, unmodified source; each was reverted and confirmed clean (`git diff --exit-code`, or byte-for-byte diff against a pre-injection snapshot for the not-yet-tracked allowlist file) before moving to the next. `go build ./cmd/` and `go vet ./cmd/` were confirmed clean after every restoration.

**1. New non-atomic write not in the allowlist → `TestColonyStateWriteAllowlistOnlyShrinks` FAILS.**
Added `func throwawayColonyStateProbeFn() error { return store.SaveJSON("COLONY_STATE.json", nil) }` to `cmd/root.go`. Observed output:
```
1 new non-atomic COLONY_STATE.json write site(s) found that are not in testdata/colony_state_write_allowlist.json:
      cmd/root.go:throwawayColonyStateProbeFn (SaveJSON)
    Either make the write atomic (store.UpdateJSONAtomically) or, if this is a genuinely reviewed exception, run with -update-colony-state-write-allowlist to regenerate the allowlist from the scanner's real output (never hand-edit the JSON file).
--- FAIL: TestColonyStateWriteAllowlistOnlyShrinks (0.10s)
```
Reverted; `git diff --exit-code -- cmd/root.go` clean; test passes again.

**2. Non-atomic write inside `advancePhase`'s reachable set, AND added to the allowlist → `TestNoFunctionReachableFromAdvancePhaseWritesNonAtomically` STILL FAILS (zero exemption on the critical path — the property that matters most).**
Edited `cmd/advance_phase.go`: kept the mutate closure's full logic (preserving its calls to `validateRuntimeStateStillCurrent`/`trimmedEvents` so the call graph stayed populated) but replaced the outer `store.UpdateJSONAtomically(...)` wrapper with a plain closure invocation followed by a bare `store.SaveJSON("COLONY_STATE.json", &updated)` — this is the plan's own mandatory Task 2 adversarial self-proof. Observed output with NO allowlist entry present:
```
1 non-atomic COLONY_STATE.json write site(s) found inside a function reachable from advancePhase -- NO allowlist exemption is possible for this set, by design:
      cmd/advance_phase.go:advancePhase (SaveJSON)
    Route this write through store.UpdateJSONAtomically instead.
--- FAIL: TestNoFunctionReachableFromAdvancePhaseWritesNonAtomically (0.11s)
```
Then added a matching entry (`cmd/advance_phase.go` / `advancePhase` / `SaveJSON`) to `testdata/colony_state_write_allowlist.json` and reran both tests. `TestColonyStateWriteAllowlistOnlyShrinks` now PASSED (the site is accepted there), but `TestNoFunctionReachableFromAdvancePhaseWritesNonAtomically` produced the **identical failure output above, unchanged** — confirming the allowlist has zero effect on the critical-path check. Reverted `cmd/advance_phase.go` (`git diff --exit-code` clean, matches saved pre-injection snapshot byte-for-byte) and regenerated the allowlist via `-update-colony-state-write-allowlist` to drop the temporary entry; both tests pass again.

**3. Delete a real call site from source but leave its allowlist entry → `TestColonyStateWriteAllowlistOnlyShrinks` FAILS (no stale rot).**
Changed `cmd/abandon_cmd.go`'s real site (`runAbandon`) from `store.SaveJSON("COLONY_STATE.json", reset)` to `store.SaveJSON("PROBE_188_04_NOT_COLONY_STATE.json", reset)`, simulating the call site disappearing from the scanner's perspective while its checked-in allowlist entry remains. Observed output:
```
1 testdata/colony_state_write_allowlist.json entry(ies) no longer match any real write site in source:
      cmd/abandon_cmd.go:runAbandon (SaveJSON)
    The file/function was removed or renamed, or the write became atomic -- run with -update-colony-state-write-allowlist to regenerate the allowlist so it always reflects the scanner's real output (never hand-edit the JSON file to drop a stale entry alone).
--- FAIL: TestColonyStateWriteAllowlistOnlyShrinks (0.11s)
```
Reverted; `git diff --exit-code -- cmd/abandon_cmd.go` clean; test passes again.

**4. Break the walker so it finds zero sites → both ratchets FAIL loudly rather than passing vacuously.**
Changed the scanner's literal-match target inside `recordColonyStateCallsAndSites` from `"COLONY_STATE.json"` to a string that matches nothing real. Observed output:
```
findColonyStateWriteSites found zero non-atomic COLONY_STATE.json write sites across cmd/*.go -- the AST walker likely broke (wrong selector names, wrong string literal match, wrong directory), not that every write site became atomic.
--- FAIL: TestColonyStateWriteAllowlistOnlyShrinks (0.11s)

scanColonyStateSource found zero non-atomic COLONY_STATE.json write sites across cmd/*.go -- the AST walker likely broke, not that the codebase became fully atomic overnight.
--- FAIL: TestNoFunctionReachableFromAdvancePhaseWritesNonAtomically (0.10s)
```
Both tests fail loudly via their respective vacuous-check guards, not vacuously pass. Reverted; `go vet ./cmd/` clean; both tests pass again.

After all four injections, a final sweep (`grep -rn "PROBE-188-04\|PROBE_188_04\|throwawayColonyStateProbeFn" cmd/*.go` and `git status --short`) confirmed zero leftover artifacts anywhere in the tree — only the two intended new files remain untracked/added.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Plan's own factual premise about call-site shape was wrong; scanner built to cover the real shape instead**
- **Found during:** Task 1, before writing any scanner code
- **Issue:** The plan's action text asserted "every one of these calls is inside a `*ast.FuncDecl` in this codebase." Direct inspection of the real call sites (`cmd/init_cmd.go:300`, `cmd/worktree.go:220`, and others) showed this is false — the majority of real sites live inside `RunE: func(...) {...}` closures on `&cobra.Command{...}` composite literals, which are not `*ast.FuncDecl` nodes. A scanner built strictly to the plan's stated premise would have silently missed most of the real sites and, more importantly, any future write added inside a new command's `RunE` closure.
- **Fix:** Built the scanner with two independent passes from the start: top-level `*ast.FuncDecl` bodies (functions and methods), and a whole-file scan for `&cobra.Command{...}` composite literals (var-declared or inline) whose `RunE` field is an inline closure, attributed to a synthetic `"cobra:<Use>"` name mirroring `worktree_destruction_reachability_test.go`'s own `cobraEntryNodes` convention.
- **Verification:** The resulting scan count (41 raw sites, 30 deduplicated) matches the hand-derived expectation from 188-CONTEXT.md's own repo-wide grep almost exactly (42 pre-188-02 minus 1 already-removed site = 41), which would not have been possible with a FuncDecl-only scanner (it would have undercounted by roughly two-thirds).
- **Committed in:** `33a13bb5` (Task 1 commit)

**2. [Rule 1 - Bug] Log message reported the wrong entry count**
- **Found during:** Task 1, immediately after first generating the baseline
- **Issue:** `writeColonyStateWriteAllowlist`'s caller logged `len(sites)` (41, the raw pre-dedup scan count) as "entries written," but after adding deduplication the file actually held 30 entries — the log message was factually wrong about what had just been written to disk.
- **Fix:** `writeColonyStateWriteAllowlist` now returns the true deduplicated entry count; the caller logs both numbers explicitly ("wrote ... with 30 unique (file, function, primitive) entries from 41 raw scanned site(s)").
- **Files modified:** `cmd/colony_state_atomicity_ratchet_test.go`
- **Verification:** Re-ran `-update-colony-state-write-allowlist`; log output and actual file entry count now agree (confirmed via a direct JSON parse count).
- **Committed in:** `33a13bb5` (Task 1 commit)

---

**Total deviations:** 2 auto-fixed (both Rule 1 — one a plan-premise bug that would have produced a category-miss ratchet, one a cosmetic log-accuracy bug caught before any commit).
**Impact on plan:** The first deviation is the most consequential thing in this entire plan — without it, the shipped ratchet would have had exactly the kind of undetected blind spot 187-VERIFICATION.md documents as catastrophic, on a codebase where the majority of command logic lives inside `RunE` closures. No scope creep: both fixes stayed inside this plan's two declared files.

## Issues Encountered

None beyond the two auto-fixed items above. The `cmd` package's full test run takes ~317s (no `-race`); this is pre-existing test suite weight, not something this plan added meaningfully to (it added two new tests, both completing in ~0.1s each).

## User Setup Required

None — no external service configuration required.

## Verification

```
go build ./...                                    clean
go vet ./...                                       clean
go test ./cmd/ -run 'TestColonyStateWriteAllowlistOnlyShrinks|TestNoFunctionReachableFromAdvancePhaseWritesNonAtomically' -count=1 -v
    --- PASS: TestColonyStateWriteAllowlistOnlyShrinks (0.10s)
    --- PASS: TestNoFunctionReachableFromAdvancePhaseWritesNonAtomically (0.10s)
go test ./cmd/... -count=1                         ok  	github.com/calcosmic/Aether/cmd	317.217s
go test ./... -count=1                             all 18 tested packages ok, 0 FAIL, exit code 0
```

## Next Phase Readiness

- ROADMAP criterion 4 ("grep-ratchet against non-atomic COLONY_STATE writes") is satisfied as a genuine structural property check, scoped per D-07 (the phase-advance path only, not a full atomicity migration of the ~30 pre-existing one-shot-command sites — that remains the ROADMAP's own deferred `LOCK` requirement category).
- Plan 05 (autopilot retry-exhaustion record) touches entirely different files (`cmd/compatibility_cmds.go`, `cmd/run_autopilot_test.go`) and has no dependency on this plan's output.
- Any future phase that migrates one of the 30 baseline-accepted sites to `UpdateJSONAtomically` will see `TestColonyStateWriteAllowlistOnlyShrinks` correctly flag the now-stale allowlist entry, prompting a `-update-colony-state-write-allowlist` regeneration as part of that migration's own commit — the ratchet's shrink-only design makes that migration's progress externally measurable via the baseline file's shrinking entry count.

---
*Phase: 188-one-truth-for-failures-and-advances*
*Completed: 2026-08-19*

## Self-Check: PASSED

- FOUND: `cmd/colony_state_atomicity_ratchet_test.go`
- FOUND: `cmd/testdata/colony_state_write_allowlist.json`
- FOUND: `.planning/phases/188-one-truth-for-failures-and-advances/188-04-SUMMARY.md`
- FOUND: commit `33a13bb5` (Task 1)
- FOUND: commit `28ab4ddd` (Task 2)
