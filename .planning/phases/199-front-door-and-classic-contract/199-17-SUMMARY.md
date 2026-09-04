---
phase: 199-front-door-and-classic-contract
plan: "17"
subsystem: lifecycle-recovery
tags: [go, cobra, recovery, maintenance, lifecycle-facts, provenance, wrappers]

requires:
  - phase: 199-front-door-and-classic-contract
    plan: "06"
    provides: Expert maintenance catalog and typed read-only inspection result vocabulary
  - phase: 199-front-door-and-classic-contract
    plan: "13"
    provides: Resume-owned evidence recovery with typed handoff provenance
  - phase: 199-front-door-and-classic-contract
    plan: "15"
    provides: Explicit owner-only forced seal contract and incomplete-closure evidence
provides:
  - Hidden, zero-write parser compatibility for legacy recover and abandon inputs
  - Read-only maintenance recovery inspection backed by LifecycleFacts provenance and existing stuck-state detectors
  - Canonical and generated recover-wrapper absence across Claude and OpenCode
affects: [plan-199-24, plan-199-27, plan-199-29, plan-199-30, plan-199-32, plan-199-34, resume, maintenance]

tech-stack:
  added: []
  patterns: [hidden zero-write compatibility route, direct-file read-only diagnostics, single recovery owner]

key-files:
  created:
    - cmd/legacy_recovery_199_test.go
  modified:
    - cmd/recover.go
    - cmd/abandon_cmd.go
    - cmd/maintenance_cmd.go
    - .aether/commands/recover.yaml
    - .claude/commands/ant-recover.md
    - .claude/commands/ant/recover.md
    - .opencode/commands/ant/recover.md

key-decisions:
  - "Keep legacy recover and abandon tokens parseable but hidden, store-free, and zero-write so old automation receives safe migration guidance without retaining authority."
  - "Read recovery evidence directly through LifecycleFacts and raw manifest/claims readers, while reserving all restoration for aether resume."

patterns-established:
  - "Retired lifecycle input: accept the exact old token, emit a typed no-change result, and never initialize storage or invoke a replacement command."
  - "Expert diagnosis: preserve detector value through direct reads with named provenance/evidence, then return one canonical recovery action."

requirements-completed: [CEC-04, LIFE-01, LIFE-04, LIFE-05]

duration: 36min
completed: 2026-09-04
---

# Phase 199 Plan 17: Legacy Recovery Retirement Summary

**Recover and abandon are now hidden, zero-write compatibility inputs; expert maintenance retains evidence-backed diagnosis, and resume is the sole public recovery action.**

## Performance

- **Duration:** 36 minutes
- **Started:** 2026-09-04T14:56:20Z
- **Completed:** 2026-09-04T15:32:09Z
- **Tasks:** 2/2
- **Files changed:** 8 implementation/test/wrapper files

## Accomplishments

- Replaced legacy `recover`, including accepted `--apply --force` input, with a hidden, store-free, typed no-change route to `aether resume`.
- Replaced legacy `abandon --confirm` with a hidden, store-free explanation of the direct-owner `aether seal --force --reason` action; the command neither constructs an invocation nor changes state.
- Added `aether maintenance recovery-inspect`, which reuses the useful stale-worker, build-packet, phase, manifest, worktree, survey, agent, and reconciliation diagnoses through causally read-only loaders.
- Returned stable issue, finding, evidence, verification, lifecycle, and `LifecycleFacts` provenance fields while making `aether resume` the only recovery action.
- Deleted the canonical recover YAML and all three managed repository wrappers while leaving every resume source/wrapper byte unchanged.

## Task Commits

Each task was committed atomically:

1. **Task 1: Route legacy recovery and abandonment to canonical owner choices**
   - `f3a09ce6` — test(199-17): add failing legacy recovery contract tests (RED)
   - `c12861a8` — feat(199-17): retire legacy recovery mutators (GREEN)
   - `934a8303` — fix(199-17): bind recovery scan to lifecycle data root
2. **Task 2: Remove recover from public source/generated surfaces**
   - `934a1a71` — chore(199-17): remove public recover command surfaces

**Plan metadata:** committed with this summary.

## Files Created/Modified

- `cmd/legacy_recovery_199_test.go` — fingerprints repository, lifecycle data, hub/registry, and Git state around real legacy and maintenance command executions.
- `cmd/recover.go` — hidden store-free compatibility result that points only to resume and never scans or repairs.
- `cmd/abandon_cmd.go` — hidden store-free compatibility result that explains, but never invokes, owner-controlled forced seal.
- `cmd/maintenance_cmd.go` — read-only recovery inspection, typed evidence/provenance result, direct manifest/claims readers, and retained stuck-state detectors.
- `.aether/commands/recover.yaml` — deleted canonical public recover wrapper source.
- `.claude/commands/ant-recover.md` — deleted managed flat Claude recover wrapper.
- `.claude/commands/ant/recover.md` — deleted managed nested Claude recover wrapper.
- `.opencode/commands/ant/recover.md` — deleted managed OpenCode recover wrapper.

## Decisions Made

- Legacy flags remain parse-compatible but hidden and inert. This gives old scripts a deterministic migration message without letting `--apply`, `--force`, or `--confirm` retain mutation authority.
- Maintenance diagnosis does not call `performStuckStateScan`, because that path loads compatibility state through write-backed storage. It uses `LifecycleFacts` plus direct `os.ReadFile` equivalents for manifest and claims evidence, preserving raw bytes and avoiding lock-file side effects.
- Every recovery finding names `aether resume`; maintenance never repairs, and abandon never launches forced seal on the owner's behalf.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Bound recovery scanners to the configured lifecycle data root**

- **Found during:** Final Task 1 correctness audit
- **Issue:** `LifecycleFacts` correctly loaded a configured `COLONY_DATA_DIR`, but the retained stuck-state scanners reconstructed the default `<repo>/.aether/data` path and could omit evidence stored elsewhere.
- **Fix:** Derived the scan root from the state fact's provenance path, with the repository default only as a fallback, and moved the zero-write fixture data outside the repository to lock the behavior.
- **Files modified:** `cmd/maintenance_cmd.go`, `cmd/legacy_recovery_199_test.go`
- **Verification:** The strengthened maintenance fixture failed by losing `stale_spawned` before the fix, then all 13 combined normal and race cases passed.
- **Committed in:** `934a8303`

---

**Total deviations:** 1 auto-fixed (1 Rule 1 bug)
**Impact on plan:** The fix keeps the planned diagnostic contract correct for supported custom data roots without expanding public surface or mutation authority.

## Automated Checks

- PASS — mandatory RED gate failed before implementation: 0 passed, 6 failed because both legacy commands were public/mutating and `maintenance recovery-inspect` did not exist.
- PASS — exact Task 1 gate: 6 cases in `TestLegacyRecoveryCommands199(Hidden|RecoverRoutesToResume|AbandonZeroWrite|MaintenanceDiagnosis)`.
- PASS — broader maintenance slice: 15 focused cases covering the new contract plus landing catalog, output modes, and read-only behavior.
- PASS — root help and Bash completion emit neither hidden legacy command.
- PASS — Task 2 source hygiene: 7 `TestCommandSourceHygiene` cases with all four recover surfaces absent.
- PASS — combined Plan 199-17 selection: 13 cases across the legacy runtime contract and source hygiene.
- PASS — the same 13-case focused selection with the Go race detector after binding scanners to the configured data root.
- PASS — all four canonical/generated resume files retain their pre-deletion Git object hashes.
- PASS — `git diff --check` across the RED, GREEN, and public-surface commits.

## TDD Gate Compliance

- RED commit `f3a09ce6` captured all intended failures before production edits: public discovery, mutating recover/abandon behavior, and missing maintenance diagnosis.
- GREEN commit `c12861a8` follows the RED commit and makes the exact six-case contract pass without weakening the tests.
- Post-GREEN regression commit `934a8303` strengthens the existing zero-write fixture with a non-default lifecycle data root and fixes the path binding it exposed.

## Known Stubs

None. Empty issue/evidence slices are stable typed results for genuinely empty diagnoses, not placeholder UI data; no TODO/FIXME or mock recovery path was introduced.

## Threat Flags

| Flag | File | Description |
|------|------|-------------|
| threat_flag: recovery-evidence-read | `cmd/maintenance_cmd.go` | The new expert inspection reads colony state, manifest, claims, worker, survey, and Git evidence. Reads are constrained to the resolved repository/data roots, use direct read-only APIs, retain provenance, and expose no repair capability. |

## Issues Encountered

- The repository-wide suite was not used as this plan's acceptance gate because the execution handoff records 199 known staged Phase 199 migration failures. The exact plan-owned runtime, maintenance, help/completion, source-hygiene, and whitespace checks are green; no new focused Plan 199-17 failure remains.
- Existing internal recovery scanners and legacy renderer/repair helpers remain available for the later scoped runtime vocabulary migration in Plan 199-34; no active public command calls those mutators from this plan's handlers.
- The installed progress updater looks only for a body `Progress:` line, while state-writing handlers derive the frontmatter percentage from completed phases and reset 20/34 plans to `0%`. A post-handoff audit caught the final session update undoing the first correction; the registered frontmatter handler was therefore made the last state mutation and a post-commit readback verified 20/34 (`59%`), Plan 18, and matching ROADMAP progress. The requirements handler does not parse this repository's bold em-dash checkbox format; all four plan requirements were already checked complete, so `REQUIREMENTS.md` correctly needed no edit.
- Protected pre-existing `.planning/config.json`, `.gsd/`, and `199-PATTERNS.md` changes remained untouched and unstaged.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Plan 199-24 can delete public abandon wrappers while retaining this plan's zero-write parser proof.
- Plans 199-30, 199-32, and 199-34 can replace remaining current guidance with the established maintenance-inspect versus resume boundary.
- Plan 199-29 can enforce the final repository-wide normal/race gate after the staged migration corpus is updated.
- No Plan 199-17 blocker remains.

## Self-Check: PASSED

- The four implementation/test files and this summary exist; the four declared recover surfaces are absent, and all four resume surfaces remain present.
- RED/GREEN commits `f3a09ce6` and `c12861a8`, cleanup commit `934a1a71`, and data-root fix `934a8303` resolve in Git in the documented order.
- The combined 13-case focused gate passes, STATE directly reads 20/34 (`59%`) with Plan 18 next, ROADMAP reads 20/34, summary whitespace is clean, and protected pre-existing paths remain unstaged.

---
*Phase: 199-front-door-and-classic-contract*
*Completed: 2026-09-04*
