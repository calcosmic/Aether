---
phase: 200-iterative-planning
plan: 47
subsystem: build-lifecycle
tags: [go, build-attempts, path-containment, legacy-adapter, receipts, tdd]

# Dependency graph
requires:
  - phase: 200-34
    provides: receipt-last canonical build-start transaction
  - phase: 200-36
    provides: build-attempt callers and fixtures routed through canonical start
  - phase: 200-42
    provides: command lifecycle compatibility baseline
provides:
  - one checked internal build-attempt path representation with one-way public display conversion
  - fail-closed legacy/modern completion classification before binding validation
  - canonical legacy rebinding through accepted plan authority, attempt journal, and receipt-last commit
  - mutation-free refusal and exact legacy replay evidence
affects: [build-finalize, build-recovery, attempt-reconciliation, final-gate]

# Tech tracking
tech-stack:
  added: []
  patterns: [canonical repository-relative data path, explicit compatibility classifier, receipt-backed exact replay]

key-files:
  created: []
  modified:
    - cmd/build_attempt.go
    - cmd/build_start_transaction.go
    - cmd/codex_build_finalize.go
    - cmd/build_attempt_external_test.go
    - cmd/codex_build_finalize_test.go

key-decisions:
  - "Build-attempt filesystem operations consume only canonical data-root-relative paths; `.aether/data/` is added only when presenting a repository-relative public path."
  - "Only the complete pre-binding historical manifest shape enters the legacy adapter; any mixture of modern authority, attempt, path, or binding claims is refused without mutation."
  - "Legacy source owner/mode are classified against historical tuples, then the canonical transaction derives platform-wrapper/external-task identity and revalidates accepted plan authority under the repository session."
  - "Exact legacy replay is recognized by the submitted packet digest only after its canonical attempt manifest and receipt evidence both verify."

patterns-established:
  - "Path boundary: normalize internal or singly displayed input once, reject duplicate/absolute/traversal forms, and join the data root only after canonicalization."
  - "Compatibility boundary: classify before modern validation, derive missing trust fields inside canonical start, and retain the raw packet digest for exact replay."

requirements-completed: [CEC-03, PLAN-05, PLAN-06]

# Metrics
duration: 39min
completed: 2026-09-09
---

# Phase 200 Plan 47: Canonical Build Paths and Legacy Binding Summary

**Build reconciliation now resolves one contained relative path without doubled `.aether/data` prefixes, while genuinely old completion packets acquire trusted execution identity through the same receipt-last, D-16-authorized start transaction as modern builds.**

## Performance

- **Duration:** 39 min
- **Started:** 2026-09-09T11:02:11Z
- **Completed:** 2026-09-09T11:41:21Z
- **Tasks:** 2
- **Files modified:** 5

## Accomplishments

- Added checked normalization for empty, internal-relative, and singly displayed build-attempt paths while rejecting duplicate prefixes, absolute paths, platform-specific path ambiguity, and traversal.
- Replaced ad hoc path prefix handling across attempt derivation, start preparation, finalizer result serialization, and reconciliation with the shared checked boundary.
- Moved legacy classification ahead of strict modern validation and made mixed historical/modern claims an explicit zero-effect refusal.
- Routed genuine legacy completion starts through `commitBuildStart`, which reloads accepted plan authority under the repository session and derives canonical owner, mode, attempt, execution binding, journal, and receipt.
- Added receipt-verified, byte-stable exact replay for the original legacy packet while preserving the established plain-language warning and durable history event.

## Task Commits

Each TDD gate and implementation unit was committed atomically:

1. **Task 1 RED: Define canonical build-attempt path behavior** - `efef47e1` (test)
2. **Task 1 GREEN: Normalize contained paths and one-way display conversion** - `69961fea` (fix)
3. **Task 2 RED: Define legacy binding, refusal, and replay behavior** - `2fac86cf` (test)
4. **Task 2 GREEN: Rebind genuine legacy completions through canonical start** - `dc553593` (fix)

## Files Created/Modified

- `cmd/build_attempt.go` - Owns checked internal/public path conversion and strict manifest binding classification at attempt validation.
- `cmd/build_start_transaction.go` - Classifies historical versus modern manifest claims and derives trusted legacy execution fields within canonical start.
- `cmd/codex_build_finalize.go` - Uses canonical paths for reconciliation/output, starts genuine legacy input through D-16 authority, and verifies receipt-backed exact replay.
- `cmd/build_attempt_external_test.go` - Covers raw, displayed, duplicate-prefix, absolute, traversal, persistence, and byte-stable replay path forms.
- `cmd/codex_build_finalize_test.go` - Covers genuine legacy acceptance/warning, canonical identity, receipt existence, exact replay, and mutation-free ambiguous/forged refusals.

## Decisions Made

- Kept the durable attempt record's established public path fields for compatibility, but every filesystem consumer now converts them back through one checked internal representation before joining the data root.
- Treated plan revision, plan-state hash, accepted authority, attempt ID/path, and execution binding as a modern claim family. A partially present family is ambiguous and cannot select compatibility behavior.
- Retained only the two historical wrapper source tuples (`host-queen` with `plan-only` or `queen-led`) as legacy candidates; the transaction does not trust those values as the resulting execution identity.
- Preserved the canonical transaction's trusted normalized-unbound internal shape for existing callers, while the public finalizer refuses that same shape when it arrives as an ambiguous external artifact.
- Kept D-16 intact: the adapter resolves current accepted authority before constructing its request, and `commitBuildStart` reloads and compares that authority while holding the repository mutation session.

## Deviations from Plan

None - both failures and their compatibility/security boundaries were repaired within the planned files and explicit adapter contract.

## Issues Encountered

- The first legacy RED failed because strict modern manifest validation ran before compatibility could derive current authority; a modern manifest stripped only of attempt ID/path was also incorrectly accepted as legacy.
- The first broader GREEN run exposed canonical transaction fixtures that intentionally exercise a trusted normalized-but-unbound internal request. Transaction validation now distinguishes that fully request-matching internal shape from public inbound compatibility classification, without widening the external adapter.

## TDD Gate Compliance

- **Task 1 RED (`efef47e1`):** the new table proved empty paths were displayed as `.aether/data`, already displayed paths were doubled, hostile paths were accepted, and reconciliation tried the doubled data root.
- **Task 1 GREEN (`69961fea`):** the exact two-test gate passed after checked normalization and single display conversion were wired through derivation and reconciliation.
- **Task 2 RED (`2fac86cf`):** genuine old input failed at strict plan-authority validation, while a modern authority-bearing manifest with its attempt fields stripped incorrectly finalized successfully.
- **Task 2 GREEN (`dc553593`):** genuine legacy input receives canonical authority/binding/receipt, all partial/contradictory/forged cases are read-only refusals, and an exact raw replay is byte-stable.

## Verification

- `go test ./cmd -run '^Test(BuildAttemptPathContract200|BuildFinalizeReconcilesJournalAfterBuiltStateCommit)$' -count=1` - PASS (`ok`, 1.271s).
- `go test ./cmd -run '^Test(BuildFinalizeWarnsOnLegacyUnboundManifest|BuildStartLegacyBinding200)$' -count=1` - PASS (`ok`, 5.417s).
- `go test ./cmd -run '^Test(BuildAttempt|BuildStart|BuildFinalizeWarnsOnLegacyUnboundManifest)' -count=1` - PASS (`ok`, 73.441s).
- `git diff --check` - PASS for all Plan 47 implementation and test changes.

## Known Stubs

None.

## Threat Flags

None. The changed path and legacy-artifact trust boundaries are exactly T-200-47-01 and T-200-47-02 from the plan: paths are contained before reads/writes, and legacy execution fields are derived through current authority rather than trusted from the packet. No endpoint, credential path, schema, dependency, or broader filesystem authority was introduced.

## User Setup Required

None - no dependency, credential, service, or configuration change is required.

## Next Phase Readiness

- Build-finalize reconciliation no longer addresses `.aether/data/.aether/data`, so the failed standalone journal gate is ready for re-verification.
- The retained legacy compatibility lane now goes through canonical build-start authority and cannot be selected by partial modern claims.
- Plans 48-54 can continue their owned gap closures; Plan 47 introduces no new lifecycle policy or public command.

## Self-Check: PASSED

- All five modified implementation/test files and this summary exist.
- TDD commits `efef47e1`, `69961fea`, `2fac86cf`, and `dc553593` exist in repository history.
- Both required key-link families resolve, the three scoped verification commands pass, and production code contains no doubled `.aether/data/.aether/data` literal.
- `.planning/config.json`, `.gsd/`, and untracked Phase 199 `199-PATTERNS.md` were neither staged nor changed by Plan 47.

---
*Phase: 200-iterative-planning*
*Completed: 2026-09-09*
