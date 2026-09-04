---
phase: 199-front-door-and-classic-contract
plan: "25"
subsystem: archive-maintenance-integrity
tags: [go, archive, integrity, sha256, lifecycle-transaction, rollback]

requires:
  - phase: 199-front-door-and-classic-contract
    plan: "05"
    provides: Crash-safe multi-root lifecycle transactions and fault recovery
  - phase: 199-front-door-and-classic-contract
    plan: "06"
    provides: Expert maintenance inspection surface and result grammar
  - phase: 199-front-door-and-classic-contract
    plan: "16"
    provides: Entomb archive manifests, closure artifacts, and published-content verification
  - phase: 199-front-door-and-classic-contract
    plan: "18"
    provides: Shared maintenance preview, transaction, rollback, and receipt primitives
provides:
  - Zero-write typed archive inspection over manifest bytes, content digests, cross-references, seal disposition, and live context markers
  - Preview-approved manifest-owned archive/context repair using the shared recoverable maintenance transaction
  - One archive evidence result shared by chamber verification and the expert integrity view
affects: [plan-199-27, plan-199-28, plan-199-30, plan-199-34, maintenance, entomb]

tech-stack:
  added: []
  patterns: [single typed archive evidence source, preview-digest approval, manifest-owned repair targets, shared maintenance transaction]

key-files:
  created:
    - cmd/maintenance_archive_199_test.go
  modified:
    - cmd/chamber.go
    - cmd/context.go
    - cmd/integrity_cmd.go

key-decisions:
  - "Only the active state's exact archive reference makes live CONTEXT.md and HANDOFF.md part of a chamber inspection; historical chambers remain judged by their own published bytes."
  - "Repair targets are closed to manifest-owned archive entries or the exact active context/tombstone files, and desired archive bytes must still satisfy the published manifest and closure disposition."
  - "Archive repair approval binds the manifest, complete inspection baseline, seal disposition, and Plan 18 mutation preview before Plan 18's transaction is allowed to write."

patterns-established:
  - "Typed evidence reuse: chamber verification and expert integrity render the same archive inspection result instead of independently inferring truth from directory presence."
  - "Historical repair boundary: validate immutable published identity before preview, revalidate it at commit, and let the shared lifecycle transaction own journals, rollback, and receipts."

requirements-completed: [CEC-04, LIFE-05, LIFE-06]

duration: 59min
completed: 2026-09-04
---

# Phase 199 Plan 25: Archive and Context Truth Summary

**Archive maintenance now proves the published bytes and closure story before it writes, then commits only an explicitly approved repair through the existing recoverable transaction engine.**

## Performance

- **Duration:** 59 minutes
- **Started:** 2026-09-04T20:55:02Z
- **Completed:** 2026-09-04T21:54:02Z
- **Tasks:** 2/2
- **Files changed:** 4 implementation/test files

## Accomplishments

- Added a pure archive inspection result that validates the canonical manifest digest, every declared content digest and size, required archive kinds, closure cross-references, seal outcome, transaction identity, and forced-incomplete ownership evidence without changing any byte.
- Made `chamber-verify` and the expert integrity report consume that same typed evidence, replacing directory-existence inference with exact manifest/content/cross-reference truth.
- Added preview and commit builders that restrict repairs to manifest-owned archive entries or exact active context files, reject missing/conflicting forced markers before mutation, and bind approval to the full inspected baseline.
- Reused Plan 199-18's maintenance mutation primitives as the sole write engine, preserving journaled pre-images and returning changed files, state effect, verification, rollback, recovery, and next-action receipt fields.
- Proved successful repair, stale/tampered preview refusal, forced-marker refusal, injected-fault rollback, receipt auditability, and byte-for-byte inspection immutability in isolated roots.

## Task Commits

Each TDD gate was committed atomically:

1. **Task 1 RED: Specify zero-write archive/context inspection** — `a178a83c` (test)
2. **Task 1 GREEN: Verify archive and context truth** — `9cc73e83` (feat)
3. **Task 2 RED: Specify recoverable archive repair** — `b4e4c61f` (test)
4. **Task 2 GREEN: Transact archive/context repairs** — `ce12bc96` (feat)

**Plan metadata:** committed with this summary.

## Files Created/Modified

- `cmd/maintenance_archive_199_test.go` — Exercises read-only inspection, manifest and forced-marker refusal, transaction rollback, and successful receipt evidence.
- `cmd/chamber.go` — Builds typed archive evidence and validates manifest content, cross-references, closure disposition, and active context markers.
- `cmd/context.go` — Prepares and commits approved manifest-owned repairs through the shared maintenance transaction.
- `cmd/integrity_cmd.go` — Renders the chamber/context integrity line from the exact typed inspection evidence.

## Decisions Made

- Live `CONTEXT.md` and `HANDOFF.md` markers are required only when the inspected chamber is the active state's exact archive reference. This avoids falsely judging an older chamber against whichever context happens to be live now.
- A repair may create or replace only bytes whose ownership and desired identity are already declared by the archive manifest, plus the exact active context/tombstone targets. Maintenance cannot use this path to normalize history or invent a new closure story.
- Preview approval covers the manifest digest, complete inspection baseline, seal disposition, and the shared Plan 18 mutation preview. Commit recomputes those facts before the existing transaction receives authority to write.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Corrected archive verification failure classification**
- **Found during:** Task 2 rollback/receipt verification
- **Issue:** The shared archive verifier's content-size/digest mismatch message also contained the word `manifest`, causing content tamper to be mislabeled as a manifest-format failure.
- **Fix:** Prioritized content size/digest/missing-entry evidence and narrowed the manifest classification to canonical manifest-digest and archive-manifest failures.
- **Files modified:** `cmd/chamber.go`
- **Commit:** `ce12bc96`

## Automated Checks

- PASS — exact Plan 25 inspection suite: `TestMaintenanceArchive199InspectReadOnly`, `ManifestRefusal`, and `ForcedMarker`.
- PASS — exact Plan 25 mutation suite: `TestMaintenanceArchive199RepairRollback` and `Receipt`.
- PASS — combined five-test Plan 25 selection (`16.394s` on final focused run).
- PASS — existing chamber and integrity regression selections.
- PASS — existing lifecycle-transaction and Plan 18 maintenance-mutation regression selections (`12.645s`).
- PASS — `go build -o <temporary>/aether ./cmd/aether`.
- PASS — `git diff --check` across all plan-owned changes.
- CARRIED — `go test ./cmd -count=1` exited after `542.319s` with the handoff's mapped staged failures in unrelated retirement/status/audit/ceremony contracts; no Plan 25, chamber, integrity, lifecycle-transaction, or maintenance-mutation test failed.

## Known Stubs

None. The new tests use real entomb output, archive manifests, live lifecycle state, fault injection, and the production transaction/receipt path.

## Issues Encountered

- No Plan 199-25-owned failure remains. The repository-wide staged failure corpus was carried as instructed and did not leave a background test process.
- The generic state progress rebuild derived `0%` from this milestone's zero completed phases even though it counted 25 completed plans. The registered frontmatter setter was therefore the final state mutation, restoring the required out-of-order readback of 25/34 (`74%`) while leaving Plan 23 as the next pointer.
- The requirements handler does not parse this repository's bold em-dash checkbox format and reported all three IDs as not found. `CEC-04`, `LIFE-05`, and `LIFE-06` were already checked complete, so `REQUIREMENTS.md` correctly needed no edit.
- Protected pre-existing `.planning/config.json`, `.gsd/`, and `199-PATTERNS.md` changes remained untouched and unstaged.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Plans 199-27 and 199-28 can exercise archive integrity through one content-backed evidence source rather than directory presence.
- Plans 199-30 and 199-34 can finish vocabulary and stored-state migration while retaining forced-incomplete closure truth and recoverable maintenance semantics.
- Plan 199-23 remains the honest first incomplete plan; this out-of-order completion must not move the execution pointer past it.

## Self-Check: PASSED

- This summary and all four declared implementation/test files exist.
- TDD commits `a178a83c`, `9cc73e83`, `b4e4c61f`, and `ce12bc96` resolve in Git and contain only Plan 25-owned files.
- The focused regressions, build, stub scan, and whitespace check pass; no unplanned network, authentication, dependency, or schema surface was introduced.

---
*Phase: 199-front-door-and-classic-contract*
*Completed: 2026-09-04*
