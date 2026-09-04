---
phase: 199-front-door-and-classic-contract
plan: "21"
subsystem: lifecycle-recovery-guidance
tags: [go, cobra, resume, provenance, wrappers, codex, seal, entomb]

requires:
  - phase: 199-front-door-and-classic-contract
    plan: "13"
    provides: Transactional pause/resume, four-class recovery provenance, zero-write conflict stops, and hidden bounded legacy redirects
  - phase: 199-front-door-and-classic-contract
    plan: "15"
    provides: Retained verified/forced seal outcomes with status primary and entomb optional
  - phase: 199-front-door-and-classic-contract
    plan: "16"
    provides: Separate owner-confirmed verified entomb transaction
  - phase: 199-front-door-and-classic-contract
    plan: "17"
    provides: Hidden zero-write legacy recovery compatibility and resume as the sole restoration owner
  - phase: 199-front-door-and-classic-contract
    plan: "20"
    provides: Existing Codex command-guide parity pattern without native lifecycle skill expansion
provides:
  - One canonical resume wrapper contract with exact description, runtime delegation, four provenance groups, receipts, and zero-write conflict stops
  - Byte-identical managed resume wrappers for flat Claude, nested Claude, and OpenCode surfaces
  - Existing Codex lifecycle guidance with canonical pause/resume and retained sealed-state review through status before optional entomb
  - Named semantic ratchets preventing public retired recovery routes, host-side evidence selection, and automatic entomb guidance
affects: [plan-199-23, plan-199-27, plan-199-30, plan-199-32, phase-202, lifecycle-recovery, platform-sync]

tech-stack:
  added: []
  patterns: [runtime-owned recovery adapter, provenance-preserving wrapper, status-primary retained closure]

key-files:
  created:
    - cmd/resume_wrapper_contract_199_test.go
  modified:
    - cmd/command_guide_test.go
    - .aether/commands/resume.yaml
    - .claude/commands/ant-resume.md
    - .claude/commands/ant/resume.md
    - .opencode/commands/ant/resume.md
    - .aether/skills/colony/colony-lifecycle/SKILL.md
    - cmd/command_guide.go

key-decisions:
  - "Resume wrappers invoke the Go runtime exactly once and render its Confirmed, Reconstructed, Conflicting, or Unknown result without inspecting or selecting recovery evidence themselves."
  - "The existing Codex command guide exposes pause and resume only; bounded legacy parser compatibility remains invisible runtime plumbing."
  - "After seal, status is the primary review action and entomb remains a separate optional owner-confirmed archive-and-clear action."

patterns-established:
  - "Recovery adapter: one runtime invocation, provenance-preserving rendering, and no host-side evidence or lifecycle mutation."
  - "Retained closure guidance: seal keeps active state reviewable through status; entomb is never automatic or folded into seal."

requirements-completed: [CEC-04, LIFE-04, LIFE-05]

duration: 16min
completed: 2026-09-04
---

# Phase 199 Plan 21: Resume Front Door and Lifecycle Guidance Summary

**Resume now has one Go-owned recovery surface across Claude, OpenCode, and existing Codex guidance, with provenance preserved and status kept primary after seal.**

## Performance

- **Duration:** 16 minutes
- **Started:** 2026-09-04T20:14:48Z
- **Completed:** 2026-09-04T20:31:13Z
- **Tasks:** 3/3
- **Files changed:** 8 implementation/test/wrapper files

## Accomplishments

- Added exact `TestResumeWrapperContract199` and `TestCommandGuideLifecycle199` ratchets over runtime metadata, canonical YAML, generated wrappers, Go provenance stops, and existing Codex lifecycle guidance.
- Replaced the stale resume copy with the exact `Validate and restore the safest honest recovery point.` contract and one `AETHER_OUTPUT_MODE=visual aether resume $ARGUMENTS` invocation.
- Made every public wrapper preserve Confirmed, Reconstructed, Conflicting, and Unknown evidence classes while stopping Conflicting/Unknown results with state effect none.
- Removed retired pause/recovery names from the command-guide catalog while leaving hidden bounded runtime input compatibility intact.
- Corrected the lifecycle skill and seal guide so sealed state is reviewed through status first and entomb is a distinct optional owner action.

## Task Commits

Each task was committed atomically:

1. **Task 1: Ratchet resume and lifecycle-skill semantics** — `e6ddcdf5` (test, RED)
2. **Task 2: Synchronize canonical and generated resume wrappers** — `aad05324` (feat, GREEN)
3. **Task 3: Correct existing Codex lifecycle guidance** — `925dea56` (fix, GREEN)

**Plan metadata:** committed with this summary and the required state/roadmap bookkeeping.

## Files Created/Modified

- `cmd/resume_wrapper_contract_199_test.go` — named runtime/YAML/wrapper provenance, authority, route, and parity contract.
- `cmd/command_guide_test.go` — named existing-Codex lifecycle guidance and retained-seal ordering contract.
- `.aether/commands/resume.yaml` — canonical exact resume description, Go authority, provenance groups, conflict stop, and adapter guardrails.
- `.claude/commands/ant-resume.md`, `.claude/commands/ant/resume.md`, `.opencode/commands/ant/resume.md` — byte-identical managed wrappers that invoke and render one runtime result.
- `.aether/skills/colony/colony-lifecycle/SKILL.md` — canonical pause/resume recovery guidance and status-before-optional-entomb lifecycle order.
- `cmd/command_guide.go` — canonical pause/resume/entomb definitions, retired public route removal, and retained-seal closeout guidance.

## Decisions Made

- Wrappers name all four provenance groups because flattening reconstructed or conflicting evidence into a generic success/failure would discard recovery truth the Go runtime already supplies.
- Conflicting and unknown recovery remain rendered stops with state effect none; a wrapper cannot inspect files, choose an authoritative source, or make the colony runnable.
- Seal and entomb remain separate transitions in guidance as well as runtime behavior: status reviews retained sealed state, while entomb requires a later explicit owner choice.

## Deviations from Plan

None - plan executed exactly as written.

## Verification

- PASS — `go test ./cmd -run '^(TestResumeWrapperContract199|TestCommandGuideLifecycle199|TestCommandSourceHygiene)$' -count=1`.
- PASS — focused pause/resume integration including confirmed, reconstructed, conflict-zero-write, hidden-redirect, pause-wrapper, and resume-wrapper cases.
- PASS — exact Task 2 checks: `TestResumeWrapperContract199` and `TestCommandSourceHygiene` independently.
- PASS — exact Task 3 checks: `TestCommandGuideLifecycle199` and `TestResumeWrapperContract199` independently.
- PASS — `go build -o /tmp/aether-plan19921-build ./cmd/aether`.
- PASS — `git diff --check HEAD`.
- EXPECTED STAGED MIGRATION — the broader `^TestCommandGuide` family now reports only `maintenance` missing from the guide and no extra retired pause/resume/recovery names. This pre-existing Plan 199-19/22 handoff is already mapped in `deferred-items.md`; it is not Plan 21-owned.

## TDD Gate Compliance

- RED `e6ddcdf5` failed for the intended reasons: stale resume copy, absent provenance/authority fields, public retired command-guide routes, and immediate-entomb lifecycle guidance.
- GREEN wrapper commit `aad05324` made the resume and source-hygiene contracts pass.
- GREEN guidance commit `925dea56` made the Codex lifecycle contract pass without weakening the runtime compatibility window or adding a native `$ant-*` skill.

## Known Stubs

None. The eight plan-owned files contain no TODO/FIXME, placeholder/coming-soon copy, mock-only result, or empty data source feeding a user-facing surface.

## Issues Encountered

- The broad command-guide test retains one already-documented failure: `.aether/commands/maintenance.yaml` has no guide entry. Plan 21 removed all three extra retired lifecycle entries and did not absorb the separate maintenance migration.
- Protected pre-existing `.planning/config.json`, `.gsd/`, and `199-PATTERNS.md` changes remained untouched and unstaged.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Plan 199-23 is the first incomplete plan and can consume the corrected recovery/closure guidance.
- Later platform inventory and executable-corpus plans can assert one public resume door without reviving hidden compatibility tokens.
- No Plan 199-21 blocker remains.

## Self-Check: PASSED

- All eight declared implementation/test/wrapper files and this summary exist.
- RED `e6ddcdf5`, wrapper GREEN `aad05324`, and guidance GREEN `925dea56` resolve in Git in the documented order.
- All exact plan semantic/source checks pass after summary creation, and the summary is whitespace-clean.
- Protected pre-existing paths remain unstaged and uncommitted.

---
*Phase: 199-front-door-and-classic-contract*
*Completed: 2026-09-04*
