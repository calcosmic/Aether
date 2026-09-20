---
phase: 200-iterative-planning
plan: 10
subsystem: specification-command
tags: [go, cobra, specification, approval, immutable-revisions, platform-help, tdd]

# Dependency graph
requires:
  - phase: 200-iterative-planning
    plan: 08
    provides: Immutable whole-goal and feature specification lineages, exact approval receipts, and deterministic projection repair
provides:
  - Public aether spec inspection, add, modify, remove, exact approval, and projection-repair operations
  - Renderer-neutral specification results retaining every typed stable-ID body category, delta, affected scope, and receipt
  - Platform-native root journey ordered as init, discuss, spec, plan, build, and run
affects: [200-11-discuss, 200-12-lifecycle-facts, 200-13-plan-command, 200-19-terminal-projection, 200-23-semantic-proof, 200-25-spec-wrappers]

# Tech tracking
tech-stack:
  added: []
  patterns: [one-to-one Cobra operations, validate-before-state-access, renderer-neutral typed results, platform-aware journey projection]

key-files:
  created:
    - cmd/spec_cmd.go
    - cmd/spec_cmd_test.go
  modified:
    - cmd/root.go
    - cmd/root_test.go

key-decisions:
  - "Expose one renderer-neutral specification result whose nine body categories remain separately typed rather than collapsing owner contracts into generic items."
  - "Validate operation exclusivity and exact revision inputs before state access, and constrain file-backed source text to regular files inside the repository."
  - "Derive the restored journey at root-help render time so Codex receives native aether commands while the Phase 199 canonical wrapper catalogue remains unchanged until the parity plan."

patterns-established:
  - "Specification command boundary: validate a single explicit operation, invoke the existing immutable engine, then project the exact state and transaction evidence into one visual/JSON-neutral result."
  - "Journey projection boundary: preserve canonical host data and add platform-specific lifecycle steps only in the runtime renderer until managed wrappers are updated together."

requirements-completed: [PLAN-05]

# Metrics
duration: 16 min
completed: 2026-09-07
---

# Phase 200 Plan 10: Public Specification Command Summary

**A real `aether spec` command now exposes exact specification inspection, immutable revision, approval, and projection repair, while Codex root help presents the restored intent-to-spec-to-plan journey in commands users can actually type.**

## Performance

- **Duration:** 16 min
- **Started:** 2026-09-07T17:10:13Z
- **Completed:** 2026-09-07T17:25:44Z
- **Tasks:** 2
- **Files modified:** 4

## Accomplishments

- Added a public `aether spec` surface with mutually exclusive inspect, add, modify, remove, approve, and projection-repair operations mapped directly to the existing immutable specification engine.
- Preserved all nine owner-readable body categories as distinct stable-ID arrays in a shared visual/JSON result, alongside revision lineage, classified delta, affected downstream scope, exact receipts, projection state, replay status, and next action.
- Enforced exact predecessor revision for mutations and exact revision ID, content hash, and approval token for approval; a generic confirmation cannot cross the authority boundary.
- Added repository-contained text-file input with symlink, regular-file, and path-containment checks before source bytes are read.
- Registered `spec` as a normal command and rendered the lifecycle journey as `init → discuss → spec → plan → build/run`, with distinct clarification, specification-approval, and candidate-plan authority descriptions.
- Made root help platform-aware: Codex now receives only native `aether ...` spellings while Claude/OpenCode retain their wrapper spellings.

## Task Commits

Each task was committed atomically; the TDD task has separate RED and GREEN commits:

1. **Task 1: Add inspect, revise, approve, and repair operations** — `fc6c333b` (test/RED), `20bf06fc` (feat/GREEN)
2. **Task 2: Register spec and the restored lifecycle journey in root help** — `bd53c101` (feat)

**Plan metadata:** committed separately after state synchronization.

## Files Created/Modified

- `cmd/spec_cmd.go` — Public command options and validation, safe text/file input, engine calls, typed result assembly, JSON output, and visual rendering.
- `cmd/spec_cmd_test.go` — Whole-goal and feature inspection, add/modify/remove, exact approval/replay, repair/no-op, invalid combination, stale predecessor, file input, and help-contract coverage.
- `cmd/root.go` — Public command registration, normal-journey grouping, platform-native journey rendering, and platform-correct compact lifecycle projection.
- `cmd/root_test.go` — Registration preservation, Codex lifecycle order, native syntax, and separate specification-versus-plan authority assertions.

## Decisions Made

- Kept every owner contract category separately typed in `specCommandResult`. This prevents JSON consumers or visual renderers from losing whether a stable ID is a requirement, exclusion, recovery expectation, public path, or another distinct contract kind.
- Performed operation-shape validation in Cobra's argument phase before the shared store initializes. Contradictory flags and incomplete exact-authority inputs therefore fail before reading or writing colony state.
- Allowed `--file` only for a resolved regular file contained by the active repository. Text convenience does not become arbitrary filesystem access through absolute paths or symlinks.
- Kept specification approval visibly separate from plan acceptance in both result copy and root-help copy. Approval authorizes the owner-readable specification; `plan` only generates an evidence-backed candidate.
- Left the Phase 199 canonical help-group slice byte-compatible and derived the restored journey for runtime output. Plan 200-25 can update managed Claude/OpenCode wrappers and inventory as one bounded parity change without making Codex lie in the meantime.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Routed restored journey rows through native platform translation**

- **Found during:** Task 2 root-help verification
- **Issue:** The first implementation appended the new `discuss` and `spec` rows after translating existing entries, so Codex help still displayed unsupported `/ant-discuss` and `/ant-spec` syntax.
- **Fix:** Applied the same platform command renderer while constructing both inserted rows.
- **Files modified:** `cmd/root.go`
- **Verification:** `TestRootCodexHelpShowsRestoredLifecycleOrder` now passes and manual temporary-repository help contains no `/ant-*` or `$ant-*` spelling.
- **Committed in:** `bd53c101`

---

**Total deviations:** 1 auto-fixed bug.
**Impact on plan:** The fix was required for the explicit Codex-facing help contract and introduced no scope beyond the planned platform rendering.

## Issues Encountered

- The installed requirement helper did not recognize this milestone's legacy bold-ID label for `PLAN-05`; the requirement was already checked complete, so no requirement-file mutation was necessary.
- Pre-existing changes in `.planning/config.json`, `.gsd/`, and `.planning/phases/199-front-door-and-classic-contract/199-PATTERNS.md` remained unstaged and uncommitted throughout execution.

## TDD Gate Compliance

- Task 1 RED (`fc6c333b`) failed at compile time because the public command and typed result contracts did not yet exist.
- Task 1 GREEN (`20bf06fc`) made all nine command-contract tests pass, including exact replay and stale-input refusal.
- The RED commit precedes the GREEN implementation in repository history.

## Known Stubs

None.

## Threat Flags

| Flag | File | Description |
|------|------|-------------|
| threat_flag: repository-file-read | `cmd/spec_cmd.go` | The new `--file` input reads owner-provided text only after canonical path, symlink containment, and regular-file checks keep access inside the repository. |

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Plan 200-11 can hand evidence-settled discussion output to the public draft/review/approval surface without inventing another specification command path.
- Plan 200-12 can use the registered specification authority when extending shared lifecycle facts and next actions.
- Plan 200-25 can add real Claude/OpenCode wrappers and update the canonical inventory without changing the Go command contract.
- No Plan 200-10 implementation blocker remains.

## Self-Check: PASSED

- All four plan-owned Go files and this summary exist; all three task commits are present in repository history in RED→GREEN→root-integration order.
- The required combined suite passes all 12 specification-command/root-help tests, the legacy Phase 199 root-help suite passes all eight tests, and the combined suite also passes under `-race`.
- `go vet ./cmd`, whitespace checks, manual `aether spec --help`, and manual Codex root-help inspection are clean.
- Help exposes every exact approval input, renders native Codex syntax in the required lifecycle order, and keeps specification approval separate from candidate-plan generation.
- Stub scanning found no TODO, FIXME, placeholder, or coming-soon implementation marker, and the public command contains no plan-acceptance or activation call.
- The only new file-read surface resolves symlinks and refuses paths outside the repository or non-regular files before reading bytes.

---
*Phase: 200-iterative-planning*
*Completed: 2026-09-07*
