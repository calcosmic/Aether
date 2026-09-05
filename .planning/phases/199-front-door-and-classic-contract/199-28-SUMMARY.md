---
phase: 199-front-door-and-classic-contract
plan: "28"
subsystem: classic coverage and lifecycle vocabulary ratchet
tags: [go, coverage-ledger, lifecycle, vocabulary, compatibility, d17]
requires:
  - phase: 199-32
    provides: executed recovery-route fixtures
  - phase: 199-34
    provides: expiring parser compatibility and recovery proof
provides:
  - exact 73-row Phase 199 capability, decision, and requirement coverage ledger
  - git-aware four-bucket lifecycle vocabulary inventory
  - status-primary D-17 and canonical recovery vocabulary ratchets
affects: [199-29, phase verification, lifecycle documentation]
tech-stack:
  added: []
  patterns:
    - tracked-byte inventories classify each path/family/count exactly once
    - historical evidence remains immutable while current runtime vocabulary is executable-proofed
key-files:
  created:
    - .planning/phases/199-front-door-and-classic-contract/199-CLASSIC-COVERAGE.json
    - .planning/phases/199-front-door-and-classic-contract/199-CLASSIC-COVERAGE.md
    - cmd/classic_coverage_199_test.go
    - cmd/current_vocabulary_199_test.go
    - cmd/testdata/current-vocabulary-199.json
  modified:
    - .planning/phases/199-front-door-and-classic-contract/199-CLASSIC-SYNTHESIS.md
    - .planning/research/v1.28-classic-capability-ledger.md
    - cmd/abandon_cmd.go
    - cmd/next_action.go
key-decisions:
  - "Use a 73-row JSON ledger as the exact source for Phase 199 capability, synthesis, decision, and requirement proof."
  - "Treat every tracked retired lifecycle spelling as an exact path-and-family inventory item; only normalize_args.go accepts old input tokens through 1.29."
  - "Current preserved-work advice is maintenance recovery-inspect followed by owner-selected aether resume, never a retired recovery command."
requirements-completed: [SYNTH-01, CEC-01, CEC-02, CEC-04, CEC-08, LIFE-01, LIFE-02, LIFE-03, LIFE-04, LIFE-05, LIFE-06, PROOF-01]
duration: 1d 1h
completed: 2026-09-06
---

# Phase 199 Plan 28: Exact Coverage and Current Vocabulary Summary

**Phase 199 now has a machine-validated 73-row restoration ledger and an exhaustive tracked-byte ratchet that preserves historical evidence while enforcing canonical current lifecycle guidance.**

## Performance

- **Duration:** 1d 1h (including an interrupted transport continuation)
- **Started:** 2026-09-05T01:04:02Z
- **Completed:** 2026-09-06
- **Tasks:** 2/2
- **Files modified:** 10

## Accomplishments

- Added the exact 34 CAP, 10 synthesis, 17 locked-decision, and 12 requirement coverage rows; each points to a current artifact and named passing proof.
- Added the four-table human audit generated from the signed JSON source and preserved the original Classic historical evidence/dispositions.
- Added a `git ls-files` inventory that classifies 206 tracked path/family token keys into current documentation, active runtime, hidden parser compatibility, or historical/test evidence.
- Enforced D-17 status-first review ordering across all eight current/generated guide surfaces and reused Plan 32/34 executed proofs.
- Removed two discovered active suggestions that leaked retired vocabulary.

## Task Commits

1. **Task 1 RED: failing exact coverage ledger contract** — `95fc9e5d` (test)
2. **Task 1 GREEN: signed exact Phase 199 coverage ledger** — `34bc4fbd` (feat)
3. **Task 2 RED: failing exhaustive vocabulary ratchet** — `2b69b93c` (test)
4. **Task 2 GREEN: canonical lifecycle vocabulary ratchet** — `adfe9ade` (feat)

## Files Created/Modified

- `199-CLASSIC-COVERAGE.json` and `199-CLASSIC-COVERAGE.md` — the exact source and four-table audit.
- `classic_coverage_199_test.go` — exact-set, field, proof, passing-status, and mutation rejection validators.
- `current_vocabulary_199_test.go` and `current-vocabulary-199.json` — exhaustive tracked-byte occurrence and current-guidance proof.
- `abandon_cmd.go` and `next_action.go` — Rule 1 removal of leaked current recovery/legacy-resume suggestions.

## Verification

- `go test ./cmd -run '^TestClassicCoverage199(ExactSets|RequiredFields|ProofResolution|PassingStatus|RejectsInvalidRows)$' -count=1` — passed (11 assertions/subtests).
- `go test ./cmd -run '^(TestCurrentVocabulary199|TestCurrentVocabularyDocs199|TestRuntimeRecoveryRoutes199|TestRuntimeRecoveryCompatibility199)$' -count=1` — passed (43 tests).
- Focused `go test -race ./cmd` for the vocabulary/document/route/compatibility proofs — passed (43 tests).
- `go build ./cmd` — passed.

## Decisions Made

- The coverage JSON is authoritative; Markdown is a checked human rendering rather than a second source of truth.
- The old spellings remain only as explicit historical/fixture evidence or bounded parser input, never as a current teaching route.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Removed active retired recovery suggestions discovered by the exhaustive inventory**
- **Found during:** Task 2
- **Issue:** `cmd/abandon_cmd.go` emitted `aether recover`, and `cmd/next_action.go` emitted `aether resume-colony`; both were current executable suggestions and could not be classified as historical evidence.
- **Fix:** Replaced the abandonment route with explicit read-only maintenance inspection followed by owner-selected `aether resume`, and removed the legacy resume candidate/route membership.
- **Files modified:** `cmd/abandon_cmd.go`, `cmd/next_action.go`, `cmd/current_vocabulary_199_test.go`
- **Verification:** Current vocabulary, Plan 32 route, Plan 34 compatibility, docs, race, and build proofs passed.
- **Committed in:** `adfe9ade`

**Total deviations:** 1 auto-fixed (1 Rule 1 bug)
**Impact on plan:** The ratchet uncovered and closed exactly the current-source leaks it was designed to prevent; no historical evidence was rewritten.

## Known Stubs

None.

## Issues Encountered

- `git ls-files` exposes a tracked symlink-directory entry with no readable working-tree byte stream. The ratchet deliberately handles it as `git grep` does: it is not a text occurrence and cannot be an exemption for a token.
- The SDK progress updater did not recognize this STATE frontmatter shape and left progress at zero; the committed metadata corrects it from the on-disk 33/34 plan count.

## User Setup Required

None.

## Next Phase Readiness

- Plan 199-29 can rely on signed source coverage and a current vocabulary boundary that rejects unmanaged lifecycle spelling drift.
- No unresolved current occurrence remains; the only retired spellings are exact inventory rows for historical evidence, test/fixture assertions, or the 1.29 parser boundary.

## Self-Check: PASSED

All five created artifacts exist and all four RED/GREEN task commits are reachable.

---
*Phase: 199-front-door-and-classic-contract*
*Completed: 2026-09-06*
