---
phase: 199-front-door-and-classic-contract
plan: "19"
subsystem: command-surfaces
tags: [go, cobra, yaml, claude-code, opencode, maintenance]

requires:
  - phase: 199-front-door-and-classic-contract
    plan: "06"
    provides: Expert maintenance landing, typed operation catalog, and live nested skill inspection
provides:
  - Canonical YAML contract for the expert maintenance landing
  - Byte-aligned Claude flat, Claude nested, and OpenCode maintenance wrappers
  - Named semantic test binding wrapper inventory and authority to the Go runtime
affects: [maintenance, platform-sync, source-hygiene, classic-contract]

tech-stack:
  added: []
  patterns: [YAML-owned managed wrappers, runtime-catalog semantic ratchet, one-call presentation adapter]

key-files:
  created:
    - cmd/maintenance_wrapper_contract_199_test.go
    - .aether/commands/maintenance.yaml
    - .claude/commands/ant-maintenance.md
    - .claude/commands/ant/maintenance.md
    - .opencode/commands/ant/maintenance.md
  modified: []

key-decisions:
  - "Use the live Go maintenance catalog as the operation-inventory authority and require canonical/generated surfaces to match its ordered IDs."
  - "Keep every public maintenance wrapper to one visual runtime invocation; Go alone reads evidence, mutates state, verifies results, emits receipts, and rolls back."
  - "Expose live skill inspection only below /ant-maintenance and add no Codex-native $ant-* maintenance surface in this milestone."

patterns-established:
  - "Semantic wrapper ratchet: a named test checks exact description, managed source, runtime delegation, inventory, and forbidden authority drift independently of generic hygiene."
  - "Mechanical peer parity: the YAML wrapper body and all three generated Claude/OpenCode bodies remain byte-equivalent after their managed headers."

requirements-completed: [CEC-02, LIFE-06]

duration: 7min
completed: 2026-09-03
---

# Phase 199 Plan 19: Maintenance Wrapper Contract Summary

**One source-linked maintenance adapter now gives Claude Code and OpenCode the same Go-owned expert catalog while preventing wrappers from writing state, parsing visuals, or inventing repair evidence.**

## Performance

- **Duration:** 7 min
- **Started:** 2026-09-03T22:56:10Z
- **Completed:** 2026-09-03T23:03:38Z
- **Tasks:** 2/2
- **Files modified:** 5

## Accomplishments

- Added exact `TestMaintenanceWrapperContract` coverage that derives the ordered inspection/mutation inventory from `buildMaintenanceCatalog` and rejects missing operations, extra invocations, direct file/state authority, visual parsing, invented success, root-level skill commands, and deferred Codex-native surfaces.
- Added `.aether/commands/maintenance.yaml` as the canonical source for the approved description, one runtime invocation, operation inventory, and state/evidence/rollback guardrails.
- Published semantically identical Claude flat, Claude nested, and OpenCode wrappers with managed headers and live skill inspection available only under `/ant-maintenance`.
- Kept `TestCommandSourceHygiene` independent and green, and confirmed the broader production source checker accepts the new surface family.

## Task Commits

Each task was committed atomically:

1. **Task 1: Ratchet maintenance wrapper semantics (RED)** — `fc3d2f79` (test)
2. **Task 2: Publish canonical and generated maintenance surfaces (GREEN)** — `6c883ab8` (feat)

**Plan metadata:** committed with this summary.

## Files Created/Modified

- `cmd/maintenance_wrapper_contract_199_test.go` — exact semantic and negative-fixture contract for all maintenance wrapper surfaces.
- `.aether/commands/maintenance.yaml` — canonical description, runtime delegation, ordered catalog IDs, wrapper body, and authority guardrails.
- `.claude/commands/ant-maintenance.md` — managed flat Claude command surface.
- `.claude/commands/ant/maintenance.md` — managed nested Claude command surface.
- `.opencode/commands/ant/maintenance.md` — managed OpenCode command surface.

## Decisions Made

- Bound the wrapper inventory directly to the existing Go catalog rather than maintaining a second unverified list in tests.
- Required exactly one `AETHER_OUTPUT_MODE=visual aether maintenance $ARGUMENTS` call and unchanged stdout, so presentation adapters cannot become a second state/evidence authority.
- Kept the public operation IDs visible for parity while leaving target resolution, previews, transactions, verification, receipts, and rollback entirely to Go.
- Preserved the later-milestone Codex boundary and Phase 191 deletion of root-level skill list/diff/cache commands.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

- The first GREEN run caught that the canonical prose did not contain the exact explicit phrase assigning every state/evidence read to the runtime. The wording was tightened across the YAML and three generated peers before the task commit; all semantic and hygiene checks then passed.
- The requirements update handler could not match this repository's checkbox style, where the bold span includes the requirement description. `CEC-02` and `LIFE-06` were already checked complete, so no direct requirements-file edit was needed.

## Verification

- `go test ./cmd -run '^TestMaintenanceWrapperContract$' -count=1` — passed (12 named test/subtest cases).
- `go test ./cmd -run '^TestCommandSourceHygiene$' -count=1` — passed (7 named test/subtest cases).
- `go test ./cmd -run '^TestSourceCheckValidatesCurrentSourceSurfaces$' -count=1` — passed.
- `git diff --check HEAD~2..HEAD` — passed.
- Acceptance checks confirmed all four public surface files exist and the test file defines exactly one top-level `TestMaintenanceWrapperContract` entry point.
- Stub scan found no TODO, FIXME, placeholder copy, coming-soon behavior, or goal-blocking unwired values; matched empty values are ordinary Go test initialization.

## Threat Flags

| Flag | File | Description |
|------|------|-------------|
| threat_flag: command-delegation | `.aether/commands/maintenance.yaml` | The new public adapter forwards owner-supplied arguments to the existing maintenance runtime. The contract restricts this to one direct invocation and leaves validation, reads, mutation, receipts, verification, and rollback in Go. |

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- The Wave 7 front-door and Autopilot wrapper work can rely on one published expert-maintenance entry across Claude Code and OpenCode.
- Later mutation routing can change Go operations without granting wrappers state authority; the ordered catalog ratchet will surface intentional inventory changes explicitly.
- No Plan 199-19 blocker remains.

## Self-Check: PASSED

- All five planned implementation/test files and this summary exist.
- RED commit `fc3d2f79` and GREEN commit `6c883ab8` are present in Git history.
- The exact semantic, hygiene, and production source-surface gates pass, and the summary is whitespace-clean.

---
*Phase: 199-front-door-and-classic-contract*
*Completed: 2026-09-03*
