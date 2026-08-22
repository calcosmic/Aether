---
phase: 162-switch-on-learning
plan: 02
subsystem: infra
tags: [go, consolidation-pipeline, queen-md, colony-prime, wisdom-pipeline]

# Dependency graph
requires:
  - phase: 160-fail-loudly
    provides: shared reliability/dry-run-purity foundation this plan extends
provides:
  - "consolidationQueenPath(): single resolved promotion target for the consolidation pipeline"
  - "ensureQueenInstinctsSection(): idempotent self-heal for legacy local QUEEN.md files"
  - "readQUEENMd ingesting the Instincts section, making promoted instincts reachable in worker prompts"
  - "dry-run purity coverage extended to the relocated QUEEN.md target"
affects: [162-03-consolidation-lifecycle-wiring, 162-04-seal-side-reconciliation]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Absolute-path promotion targets pass straight through storage.Store.resolvePath, avoiding accidental store-relative writes"
    - "Append-only, idempotent self-heal for legacy markdown section headers (never mid-file insert, since findSectionEnd treats a missing --- delimiter as 'runs to end of file')"

key-files:
  created:
    - cmd/consolidation_promotion_target_test.go
  modified:
    - cmd/graph_consolidation_cmds.go
    - cmd/queen.go
    - cmd/context.go
    - cmd/consolidation_dryrun_test.go

key-decisions:
  - "consolidationQueenPath() returns localQueenPath() directly rather than introducing a new path convention -- reuses the exact file colony-prime already reads"
  - "ensureQueenInstinctsSection() appends at the END of the file, never mid-file, because pkg/memory's findSectionEnd treats a missing --- delimiter as 'section runs to end of file'"
  - "Self-heal is wired only into the real (non-dry-run) branches of consolidation-phase-end and consolidation-seal; dry-run branches never call it, preserving 'Report without modifying'"
  - "readQUEENMd's allowlist widened with an exact-match plus prefix term for Instincts, in the same style as the existing Wisdom/Patterns/Philosophies terms -- no parser restructuring needed since PromoteInstinct's format already matches the existing key:value branch"
  - "Dry-run purity snapshot for the relocated QUEEN.md targets the exact file path, not a recursive walk of the parent .aether directory, because storage.Store's FileLocker creates a sibling .aether/locks/*.lock file on first touch that would otherwise register as a false-positive mutation"

patterns-established:
  - "New promotion/consolidation targets must resolve through a single named function (consolidationQueenPath) rather than repeating string literals across call sites -- makes a future path regression a one-line fix and a one-line test"
  - "End-to-end reachability tests that isolate one path from a second, unrelated path carrying the same data (here: instincts.json -> Active Instincts vs. QUEEN.md -> readQUEENMd) by explicitly clearing the confounding source before asserting"

requirements-completed: [LEARN-01, LEARN-03]

# Metrics
duration: ~35min
completed: 2026-08-04
---

# Phase 162 Plan 02: Switch On Learning — Promotion Target Reachability Summary

**Resolved the consolidation pipeline's instinct-promotion target from a dead store-relative file to the actual `.aether/QUEEN.md` colony-prime reads, and made the `## Instincts` section it writes visible to worker prompt assembly — with tests that fail if either link regresses.**

## Performance

- **Duration:** ~35 min
- **Started:** 2026-08-04T13:10Z (first commit)
- **Completed:** 2026-08-04T13:22Z (last commit)
- **Tasks:** 3 (all TDD except Task 3, which extended existing dry-run purity tests)
- **Files modified:** 4 (1 new test file, 3 modified source/test files)

## Accomplishments

- `pipelineConfigForStore()` and both dry-run construction sites now resolve `QueenPath` via `consolidationQueenPath()` instead of the bare literal `"QUEEN.md"`, which `storage.Store.resolvePath` was silently joining onto the store base directory (`.aether/data/QUEEN.md` — a file colony-prime never reads).
- `ensureQueenInstinctsSection()` self-heals legacy local `QUEEN.md` files (predating the `## Instincts` section) by appending the header at the end of the file — append-only, byte-for-byte preserving, and idempotent. Wired into the real (non-dry-run) branches of both `consolidation-phase-end` and `consolidation-seal` only.
- `queenDefaultContent` now includes `## Instincts` so newly created colonies get the section without needing the self-heal.
- `readQUEENMd`'s section allowlist now includes `Instincts` (exact match + prefix, matching the existing style), so a promoted instinct is ingested into the map `buildColonyPrimeOutput` assembles into worker prompts. No parser restructuring was needed — `PromoteInstinct`'s `- [instinct] **<domain>** (<conf>): When <trigger>, then <action>` format already matches the existing `key: value` parsing branch.
- Dry-run purity coverage (`TestConsolidationPhaseEndDryRunDoesNotMutate`, `TestConsolidationSealDryRunDoesNotMutate`) now also watches the relocated `localQueenPath()` file directly, plus a new `TestConsolidationDryRunDoesNotCreateInstinctsSection` pinning that the self-heal never runs on a dry-run branch.

## Task Commits

Each task was committed atomically, TDD RED/GREEN split into separate commits:

1. **Task 1 RED: add failing test for consolidation promotion target** - `e16f6cdc` (test)
2. **Task 1 GREEN: resolve consolidation promotion target to readable QUEEN.md** - `521b9d59` (feat)
3. **Task 2 RED: add failing test for Instincts reaching worker prompt** - `abf32fae` (test)
4. **Task 2 GREEN: make Instincts section visible to worker context assembly** - `a2b6a8b4` (feat)
5. **Task 3: extend dry-run purity coverage to the relocated QUEEN.md** - `65dc4980` (test)

_Note: Task 3 has no separate feat commit — its "action" was extending existing tests, not writing new production code; nothing in Task 3 required a source change beyond what Tasks 1-2 already shipped._

## Files Created/Modified

- `cmd/consolidation_promotion_target_test.go` - New test file: proves `consolidationQueenPath()` resolves to `localQueenPath()`, `ensureQueenInstinctsSection()` self-heals correctly and idempotently, a real consolidation run promotes into `.aether/QUEEN.md` under `## Instincts` (never `.aether/data/QUEEN.md`), and the promoted instinct reaches `buildColonyPrimeOutput(true).PromptSection` (isolated from the unrelated instincts.json → Active Instincts path)
- `cmd/graph_consolidation_cmds.go` - Added `consolidationQueenPath()`; replaced all four `"QUEEN.md"` literals; wired `ensureQueenInstinctsSection()` into the real branches of both consolidation subcommands
- `cmd/queen.go` - Added `## Instincts` to `queenDefaultContent`; added `ensureQueenInstinctsSection()`
- `cmd/context.go` - Widened `readQUEENMd`'s section allowlist to include `Instincts`
- `cmd/consolidation_dryrun_test.go` - Widened the watched-file list in both existing dry-run purity tests to include the relocated `localQueenPath()`; added `TestConsolidationDryRunDoesNotCreateInstinctsSection`

## Decisions Made

- `consolidationQueenPath()` is a pure resolver with no side effects (no writes, no directory creation) because two of its four callers are dry-run paths that must remain read-only.
- The self-heal appends at the **end** of the local `QUEEN.md`, never mid-file, because `pkg/memory`'s `findSectionEnd` treats "no `\n---` after the header" as "section runs to end of file" — a mid-file `## Instincts` header on the legacy (no-delimiter) local template would silently redirect entries meant for other sections.
- Dry-run purity for the relocated file is asserted via an explicit file-path watch rather than a recursive snapshot of the parent `.aether` directory, because `storage.Store`'s `FileLocker` creates a sibling `.aether/locks/*.lock` file the first time any path is locked — a recursive walk would register that as a false-positive mutation on every run, dry or not.
- `TestPromotedInstinctReachesWorkerPrompt` explicitly clears `instincts.json` after the consolidation run and before assembling the prompt, because the pre-existing `## Active Instincts` section in `buildColonyPrimeOutput` reads `instincts.json` directly and unconditionally (a known, accepted exposure per this plan's threat register, T-162-09) — without clearing it, the test would pass vacuously even if the `readQUEENMd`/`Instincts` link were cut.

## Deviations from Plan

None — plan executed exactly as written. All acceptance criteria in all three tasks were verified directly (including the "removing the fix causes the test to fail" proof-of-linkage checks specified in the plan), not merely asserted.

## Issues Encountered

The plan's Task 2 behavior spec asked for a test proving the pipeline → QUEEN.md → readQUEENMd → prompt link reaches the worker, but a naive version of that test passed even before the `readQUEENMd` fix, because the pre-existing `## Active Instincts` section (reading `instincts.json` directly) already surfaced the same sentinel text through an unrelated path. Resolved by clearing `instincts.json` after the real consolidation run and before calling `buildColonyPrimeOutput`, isolating the QUEEN.md-specific link the test exists to prove. Confirmed by temporarily removing the `Instincts` allowlist term and observing the test then fail as expected.

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness

The consolidation pipeline's promotion output is now reachable end-to-end: `pkg/memory`'s `QueenService.PromoteInstinct` writes into the same `.aether/QUEEN.md` file colony-prime reads, under a section (`Instincts`) that is actually ingested into worker prompts, and dry-run purity now covers the relocated file. Plans 03 (consolidation lifecycle wiring) and 04 (seal-side reconciliation) can now switch consolidation on via runtime callers without switching it on into a void — the load-bearing prerequisite this plan's objective called out is satisfied.

No blockers. `pkg/memory/promoteInstinctLocal`'s separate `## Wisdom`-target write path (in `cmd/queen.go`) was deliberately left untouched per the plan's scope — Plan 04 owns reconciling the two writers.

---
*Phase: 162-switch-on-learning*
*Completed: 2026-08-04*

## Self-Check: PASSED

All created/modified files verified present on disk:
- cmd/consolidation_promotion_target_test.go
- cmd/graph_consolidation_cmds.go
- cmd/queen.go
- cmd/context.go
- cmd/consolidation_dryrun_test.go
- .planning/phases/162-switch-on-learning/162-02-SUMMARY.md

All task commits verified present in `git log`:
- e16f6cdc, 521b9d59, abf32fae, a2b6a8b4, 65dc4980

Full `go build ./...` and `go test ./cmd/ -count=1` both green.
