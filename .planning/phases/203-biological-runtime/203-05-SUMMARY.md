---
phase: 203-biological-runtime
plan: "05"
subsystem: pheromones
tags: [pheromone-resolver, provenance, quarantine, ast-guards, go]

# Dependency graph
requires:
  - phase: 203-biological-runtime
    provides: "203-01's signed CLASSIC-SYNTHESIS.md (SYN-203-10 ruling f) naming the three disagreeing effective-strength predicates and requiring one canonical resolver plus a provenance/quarantine model"
provides:
  - "resolveEffectivePheromones (cmd/pheromone_resolver.go): the single decision for whether a pheromone note is in effect, how strong it is, and why not, replacing three disagreeing predicates"
  - "colony.PheromoneSignal.Provenance/Quarantined (pkg/colony/pheromones.go): pointer-backed, omitempty origin and quarantine state, plus the four write-time provenance constants and PheromoneProvenances()"
  - "Three AST-based ratchet tests (TestOneEffectivePheromonePredicate, TestOnePheromoneWriterOnly, TestEveryBriefReaderUsesTheResolver) proven to actually fail-by-name against a real regression"
  - "The real cross-project import path (cmd/exchange.go importPheromonesData) now stamps import provenance and quarantine, closing an end-to-end gap the plan's own must_haves required"
affects: [203-07, 203-08, "any future plan touching pheromone reads, writes, or the tick-to-approve queue (BIO-08)"]

# Actuals (#2632)
actuals:
  tokens: 12943
  tasks: 3
  commits: 6

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "AST-based structural ratchets (go/ast + go/parser walking non-test .go source in a package directory) to enforce singleness invariants that a grep or a comment cannot catch -- verified this session by deliberately breaking each guard and confirming it fails by name before restoring."
    - "Named exclusion-reason enum (pheromoneExcludedInactive/quarantined/expired/below-floor/malformed) on a resolver's decision struct, never a bare boolean, matching this codebase's established refusal-by-name convention (cmd/spawn.go)."

key-files:
  created:
    - cmd/pheromone_resolver.go
    - cmd/pheromone_resolver_test.go
  modified:
    - cmd/pheromone_loader.go
    - cmd/context.go
    - cmd/codex_plan.go
    - cmd/hook_cmds.go
    - cmd/signal_housekeeping.go
    - cmd/pheromone_write.go
    - cmd/exchange.go
    - cmd/exchange_import_sanitize_test.go
    - pkg/colony/pheromones.go

key-decisions:
  - "One resolver (resolveEffectivePheromones) is now the sole answer to 'is this note in effect'; extractSignalTexts, filterSignalsForPrompt, signalActiveForPrompt, extractSignalTextsFrom, and codex_plan.go's REDIRECT scan are all thin wrappers or direct callers -- proven by an AST call-graph walk (TestEveryBriefReaderUsesTheResolver), not a grep for a comment."
  - "An empty CreatedAt is treated as 'no decay data available' (matching computeEffectiveStrength's pre-existing behavior and this codebase's many test fixtures that omit it), while a non-empty but unparseable CreatedAt, an unparseable ExpiresAt, or a non-finite Strength are genuinely malformed and exclude the note. This distinction was required to avoid breaking dozens of pre-existing pheromone test fixtures that legitimately omit CreatedAt."
  - "Quarantine and provenance are pointer-backed and omitempty on colony.PheromoneSignal per the Phase 199 rule; a legacy signal with neither field reads as provenance 'unknown' and not quarantined, so existing colonies keep working unchanged."
  - "The plan's files_modified header omitted cmd/hook_cmds.go and cmd/signal_housekeeping.go, but Task 1's own acceptance criteria explicitly required their literal 0.1 comparisons to read the named pheromoneEffectiveFloor constant. Edited both (Rule 2 deviation) since neither is owned by the parallel sibling plan (203-02, which owns cmd/recruitment*.go/cmd/live_events.go/pkg/events/colony_live.go)."
  - "cmd/exchange.go's importPheromonesData (the real `aether import pheromones` / `/ant-import-signals` command) never called writePheromoneSignal and would have left every real cross-project import unquarantined -- the exact hazard D-10 exists to prevent, unenforced on the one pathway named for it. Wired it to stamp Provenance=import/Quarantined=true directly (Rule 2 deviation), since this plan's own must_haves truth requires imports to actually be held back, not merely provable in an isolated unit fixture."
  - "TestOnePheromoneWriterOnly and TestNoUngovernedQuarantineClear are ratchets with a named, reasoned allowlist (writePheromoneSignal, expireSignalsByType, runSignalHousekeepingWithState, entombTempSweep, importPheromonesData, syncPheromoneStores) rather than a single-writer absolute, because several of these pre-existing functions have distinct, legitimate purposes (lifecycle expiry, housekeeping GC, archival sweep, cross-project import, worktree sync) that were never candidates for consolidation into writePheromoneSignal -- only a genuinely new, undeclared writer fails the test."

requirements-completed: [BIO-07]

coverage:
  - id: D1
    description: "One resolver (resolveEffectivePheromones) owns effective scope, expiry and strength, replacing the three-way disagreement between extractSignalTexts (missed expiry), filterSignalsForPrompt (the correct superset), and codex_plan.go's REDIRECT scan (no floor at all)"
    requirement: "BIO-07"
    verification:
      - kind: unit
        ref: "cmd/pheromone_resolver_test.go#TestResolveEffectivePheromonesActiveStrongExpiredIsExcluded"
        status: pass
      - kind: unit
        ref: "cmd/pheromone_resolver_test.go#TestActiveStrongExpiredExcludedByEveryReader"
        status: pass
      - kind: unit
        ref: "cmd/pheromone_resolver_test.go#TestResolveEffectivePheromonesTieBreaksOnIdentifierAcrossRepeatedCalls"
        status: pass
      - kind: unit
        ref: "cmd/pheromone_resolver_test.go#TestResolveEffectivePheromonesMalformedTimestampNeverDefaultsIntoEffect"
        status: pass
      - kind: other
        ref: "go build ./... && go test ./cmd -run '^(TestPheromoneResolver|TestPheromone|TestSignal)' -count=1"
        status: pass
    human_judgment: false
  - id: D2
    description: "Every pheromone signal records its origin (owner/runtime/learning/import/unknown); a cross-project import is quarantined end-to-end (through the real import command, not only a unit fixture) and excluded from every worker brief until an owner-facing release path (not built in this plan) clears it"
    requirement: "BIO-07"
    verification:
      - kind: unit
        ref: "cmd/pheromone_resolver_test.go#TestLegacySignalReadsUnknownProvenanceAndNotQuarantined"
        status: pass
      - kind: unit
        ref: "cmd/pheromone_resolver_test.go#TestImportedNoteIsQuarantinedAndExcludedFromWorkerBrief"
        status: pass
      - kind: unit
        ref: "cmd/pheromone_resolver_test.go#TestWritePheromoneSignalStampsProvenanceAndQuarantinesImports"
        status: pass
      - kind: integration
        ref: "cmd/exchange_import_sanitize_test.go#TestImportPheromonesStampsImportProvenanceAndQuarantine"
        status: pass
      - kind: unit
        ref: "cmd/pheromone_resolver_test.go#TestPheromoneProvenanceRegistryIsComplete"
        status: pass
    human_judgment: false
  - id: D3
    description: "Structural guards ensure the consolidation cannot quietly come apart: a second effective-strength predicate, a second pheromones.json writer, or a brief reader that stops calling the resolver each fail a named test -- verified this session by deliberately introducing each violation and confirming the failure names the offending function, then restoring"
    requirement: "BIO-07"
    verification:
      - kind: unit
        ref: "cmd/pheromone_resolver_test.go#TestOneEffectivePheromonePredicate"
        status: pass
      - kind: unit
        ref: "cmd/pheromone_resolver_test.go#TestOnePheromoneWriterOnly"
        status: pass
      - kind: unit
        ref: "cmd/pheromone_resolver_test.go#TestEveryBriefReaderUsesTheResolver"
        status: pass
      - kind: unit
        ref: "cmd/pheromone_resolver_test.go#TestNoUngovernedQuarantineClear"
        status: pass
    human_judgment: false

duration: 55min
completed: 2026-09-13
status: complete
---

# Phase 203 Plan 05: One Pheromone Resolver, Provenance, and Quarantine Summary

**One `resolveEffectivePheromones` function replaces three disagreeing effective-strength predicates, every stored signal now records its origin, cross-project imports are quarantined end-to-end through the real import command, and three AST-based ratchet tests -- verified this session to actually fail by name -- lock the consolidation against regression.**

## Performance

- **Duration:** 55 min (approximate; research and reading preceded the first commit)
- **Started:** 2026-09-13T00:55:00+02:00 (approx.)
- **Completed:** 2026-09-13T01:49:00+02:00
- **Tasks:** 3 completed
- **Files modified:** 11 (2 created, 9 modified)

## Accomplishments

- Created `cmd/pheromone_resolver.go` with `resolveEffectivePheromones`, the one function every pheromone reader now calls (or transitively reaches) to decide whether a note is in effect, replacing `extractSignalTexts` (which never checked expiry), `filterSignalsForPrompt` (the correct-but-duplicated superset), and `codex_plan.go`'s REDIRECT scan (which used `<= 0`, no floor at all, drifting from Classic's own 0.1 convention).
- `pheromoneEffectiveFloor` is one named constant (0.1) replacing scattered literals across `pheromone_loader.go`, `context.go`, `hook_cmds.go`, and `signal_housekeeping.go`.
- Added `colony.PheromoneSignal.Provenance`/`Quarantined` (pointer-backed, `omitempty`), the four write-time provenance constants, and `PheromoneProvenances()`. `writePheromoneSignal` classifies its `source` input into owner/runtime/import and quarantines exactly the import case.
- Wired the REAL cross-project import path (`cmd/exchange.go`'s `importPheromonesData`, the actual `aether import pheromones` command) to stamp `Provenance=import`/`Quarantined=true` -- a gap that would otherwise have left every genuine cross-project import unquarantined despite the resolver and writer both being correct in isolation.
- Three AST-based structural guards (`TestOneEffectivePheromonePredicate`, `TestOnePheromoneWriterOnly`, `TestEveryBriefReaderUsesTheResolver`) plus a fourth governing quarantine assignment (`TestNoUngovernedQuarantineClear`), each manually verified this session to fail by name against a deliberately introduced violation before being restored to the passing state.

## Task Commits

Each task was committed atomically (TDD RED/GREEN pairs, per task):

1. **Task 1: One resolver owns effective scope, expiry and strength**
   - `e783ab0f` (test) -- RED: failing resolver-behavior tests, `resolveEffectivePheromones` did not exist
   - `23fb937c` (feat) -- GREEN: `cmd/pheromone_resolver.go` + rewired `pheromone_loader.go`, `context.go`, `codex_plan.go`, `hook_cmds.go`, `signal_housekeeping.go`
2. **Task 2: Record where every note came from and quarantine what arrives from elsewhere**
   - `2a59e8b4` (test) -- RED: failing provenance/quarantine tests, fields/constants did not exist
   - `c8c8ea60` (feat) -- GREEN: `pkg/colony/pheromones.go` fields+constants, `pheromone_write.go` provenance classification
3. **Task 3: Lock the singleness of the writer and the predicate**
   - `695b9912` (test) -- guard tests; no production code needed (Tasks 1-2 already established the invariants), all three pass immediately
4. **Deviation fix** (see below)
   - `1e62a406` (fix) -- wired the real cross-project import path to actually quarantine, closing an end-to-end gap in the must_haves truth

**Plan metadata:** commit pending (this SUMMARY)

## Files Created/Modified

- `cmd/pheromone_resolver.go` -- `resolveEffectivePheromones`, `resolvedPheromone`, `pheromoneEffectiveFloor`, exclusion-reason constants, malformed/quarantine/provenance helpers
- `cmd/pheromone_resolver_test.go` -- 24 tests: resolver behavior, provenance/quarantine, and three AST-based structural guards plus shared `parseCmdPackageFuncs` helper
- `cmd/pheromone_loader.go` -- `signalActiveForPrompt`/`filterSignalsForPrompt`/`extractSignalTextsFrom` rewritten as thin resolver adapters
- `cmd/context.go` -- `extractSignalTexts` rewritten as a thin wrapper over `extractSignalTextsFrom`
- `cmd/codex_plan.go` -- `activeRedirectPlanConstraints` now calls the resolver instead of its own `<= 0` predicate
- `cmd/hook_cmds.go`, `cmd/signal_housekeeping.go` -- literal `0.1` replaced with `pheromoneEffectiveFloor`
- `cmd/pheromone_write.go` -- `pheromoneProvenanceFromSource`, provenance/quarantine stamping in `writePheromoneSignal`
- `cmd/exchange.go` -- `importPheromonesData` stamps import provenance and quarantine on every sanitized signal
- `cmd/exchange_import_sanitize_test.go` -- `TestImportPheromonesStampsImportProvenanceAndQuarantine` (end-to-end, real import command)
- `pkg/colony/pheromones.go` -- `Provenance`/`Quarantined` fields, provenance constants, `PheromoneProvenances()`

## Decisions Made

See `key-decisions` in frontmatter for the full list. The most consequential: an empty `CreatedAt` is NOT treated as malformed (only a non-empty, unparseable one is), because dozens of pre-existing pheromone test fixtures across this codebase omit `CreatedAt` entirely and a strict "any unparseable value is malformed" rule would have excluded all of them from effect -- this was discovered by running the target test suite and seeing `TestPheromonePrime` fail before the distinction was added.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - Missing Critical] Two files outside the plan's files_modified header needed the floor-constant fix**
- **Found during:** Task 1
- **Issue:** The plan's frontmatter `files_modified` list omits `cmd/hook_cmds.go` and `cmd/signal_housekeeping.go`, but Task 1's own acceptance criteria explicitly required their literal `0.1` strength comparisons to read the named constant.
- **Fix:** Replaced `< 0.1` with `< pheromoneEffectiveFloor` in both files.
- **Files modified:** `cmd/hook_cmds.go`, `cmd/signal_housekeeping.go`
- **Verification:** `grep -n '0\.1\b'` in all five acceptance-criteria files returns no comparisons (only comments); neither file is owned by the parallel sibling plan (203-02).
- **Committed in:** `23fb937c`

**2. [Rule 2 - Missing Critical] The real cross-project import path never quarantined anything**
- **Found during:** Post-Task-3 review of the plan's own must_haves truths
- **Issue:** `cmd/exchange.go`'s `importPheromonesData` (the actual `aether import pheromones` command) never called `writePheromoneSignal` and never set `Provenance`/`Quarantined`. A real cross-project import would have read as legacy "unknown" provenance, NOT quarantined, and reached worker briefs immediately -- exactly the hazard D-10 exists to prevent, unenforced on the one pathway named for it, despite the resolver and writer being individually correct.
- **Fix:** `importPheromonesData` now stamps `Provenance=import`/`Quarantined=true` on every sanitized signal before merging.
- **Files modified:** `cmd/exchange.go`, `cmd/exchange_import_sanitize_test.go`, `cmd/pheromone_resolver_test.go` (added `importPheromonesData` to `TestNoUngovernedQuarantineClear`'s allowlist)
- **Verification:** `TestImportPheromonesStampsImportProvenanceAndQuarantine` proves the real import command quarantines and excludes from worker-brief text; existing import tests unaffected.
- **Committed in:** `1e62a406`

---

**Total deviations:** 2 auto-fixed (both Rule 2 -- missing critical functionality required by the plan's own acceptance criteria/must_haves). **Impact:** No scope creep; both closed a genuine gap between the plan's stated intent and what the code would otherwise have done. No file owned by the parallel sibling plan (203-02: `cmd/recruitment*.go`, `cmd/live_events.go`, `pkg/events/colony_live.go`) was touched.

## Issues Encountered

- The plan's `files_modified` header and Task 1's acceptance criteria disagreed on scope (see Deviation 1). Resolved in favor of the more specific, actionable acceptance criteria, per CLAUDE.md's Definition of Done.
- `TestNoUngovernedQuarantineClear`'s AST detection initially produced a false positive against `cmd/hive.go`'s unrelated `HiveWisdomEntry.Quarantined` field (a different struct, same field name). Narrowed the detector to a name-based heuristic (receiver identifiers containing "sig") plus a `colony.PheromoneSignal`-typed composite-literal check, documented inline as an intentional precision/scope tradeoff since pure AST inspection cannot type-check.
- Building the tick-to-approve UI surface that lets an owner actually SEE and release a quarantined note (BIO-08, D-07/D-10's shared queue) is explicitly out of this plan's scope -- `pkg/colony.PendingSuggestion` already exists as the reuse target per 203-PATTERNS.md, but adding a quarantine lifecycle state to it belongs to a later plan. Today, nothing in this codebase can clear a `Quarantined` flag once set (proven by `TestNoUngovernedQuarantineClear`), which is the deliberately conservative, safe starting state.
- `cmd/pheromone_mgmt.go`'s `pheromone-display` listing command (the `aether pheromones` / `/ant-pheromones` admin view) filters only on `sig.Active` and type -- it does not call the resolver and will continue to show expired, below-floor, or quarantined signals (with no quarantine-state label yet). This is unchanged, pre-existing behavior for a raw admin listing, not a regression, and satisfies the must_haves' "quarantined note still appears in listings" requirement trivially; making the quarantine state visibly labeled there is BIO-08 UI work, not this plan's read-side resolver consolidation.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- The must_haves truths for BIO-07 are satisfied and test-locked: one resolver, one predicate, recorded provenance on every signal, and quarantine that survives the real import path.
- A future BIO-08 plan can add a quarantine lifecycle state to `colony.PendingSuggestion` and reuse `cmd/suggest_approve.go`'s existing approve/dismiss commands as the single tick-to-approve surface for both suggested notes and quarantined imports (per 203-PATTERNS.md and D-07/D-10) -- no new queue needed.
- No blockers for the sibling parallel plan (203-02); no file overlap occurred.

---
*Phase: 203-biological-runtime*
*Completed: 2026-09-13*

## Self-Check: PASSED

- FOUND: `cmd/pheromone_resolver.go`
- FOUND: `cmd/pheromone_resolver_test.go`
- FOUND: commit `e783ab0f` (test: RED, Task 1) in `git log --oneline`
- FOUND: commit `23fb937c` (feat: GREEN, Task 1) in `git log --oneline`
- FOUND: commit `2a59e8b4` (test: RED, Task 2) in `git log --oneline`
- FOUND: commit `c8c8ea60` (feat: GREEN, Task 2) in `git log --oneline`
- FOUND: commit `695b9912` (test: Task 3 guards) in `git log --oneline`
- FOUND: commit `1e62a406` (fix: deviation) in `git log --oneline`
- Re-ran plan-level task `<verify>` commands: Task 1 `go test ./cmd -run '^(TestPheromoneResolver|TestPheromone|TestSignal)' -count=1 && go build ./...` -- PASS; Task 2 `go test ./cmd ./pkg/colony -run '^(TestPheromoneProvenance|TestPheromoneQuarantine)' -count=1` -- PASS; Task 3 `go test ./cmd -run '^(TestOneEffectivePheromonePredicate|TestOnePheromoneWriterOnly|TestEveryBriefReaderUsesTheResolver)$' -count=1` -- PASS
- Guard tests independently verified to fail-by-name against manually introduced violations, then restored to clean/passing state (`git diff --stat` empty on all touched-but-reverted files)
