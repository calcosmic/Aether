---
phase: 199-front-door-and-classic-contract
plan: "07"
subsystem: lifecycle-front-door
tags: [go, cobra, help, init, lifecycle-projection, territory, wrappers]

requires:
  - phase: 199-front-door-and-classic-contract
    plan: "04"
    provides: Immutable lifecycle facts, pure projection, and shared Next Up decisions
  - phase: 199-front-door-and-classic-contract
    plan: "05"
    provides: Lifecycle state-effect and transaction vocabulary
  - phase: 199-front-door-and-classic-contract
    plan: "06"
    provides: Expert maintenance landing and read-only root command boundaries
  - phase: 199-front-door-and-classic-contract
    plan: "08"
    provides: Typed territory freshness evidence and its four public outcome labels
provides:
  - Progressive three-group help with projection-derived empty and active standing
  - Automatic first-run init bootstrap with a five-stage guided ceremony
  - Versioned accepted-charter episode state carrying goal, provenance, time, and material constraints
  - One canonical help specification with byte-identical Claude and OpenCode runtime delegates
affects: [plan-front-door, lifecycle-status, init-wrappers, command-help, generated-surfaces]

tech-stack:
  added: []
  patterns: [projection-backed orientation, pre-store refusal, typed UI evidence, passive runtime wrappers]

key-files:
  created:
    - cmd/front_door_199_test.go
  modified:
    - cmd/root.go
    - cmd/init_cmd.go
    - cmd/lifecycle_facts.go
    - pkg/colony/colony.go
    - .aether/commands/help.yaml
    - .claude/commands/ant-help.md
    - .claude/commands/ant/help.md
    - .opencode/commands/ant/help.md

key-decisions:
  - "Root help is a curated journey map backed by the shared lifecycle projection; the raw Cobra inventory remains reachable only through command-specific/expert paths."
  - "Guided Claude/OpenCode init refuses an existing colony before opening storage, while the raw Codex/runtime confirmation escape hatch remains backward compatible."
  - "Accepted intent is durable data in accepted-charter/v1, not presentation text reconstructed after initialization."
  - "Help wrappers run the visual Go command once and return stdout unchanged rather than maintaining a second help implementation."

patterns-established:
  - "Pre-store guard: a mutating lifecycle command may inspect retained state directly before constructing any write-backed store."
  - "Typed display boundary: territory UI copy comes only from SurveyFreshnessResult.OutcomeLabel."
  - "Canonical wrapper body: YAML owns one passive body copied byte-for-byte to every managed platform surface."

requirements-completed: [CEC-01, CEC-02, LIFE-01]

duration: 26min
completed: 2026-09-03
---

# Phase 199 Plan 07: Guided Front Door Summary

**Projection-backed journey help and a five-stage init now bootstrap the repository, persist an accepted episode charter, report typed territory truth, and hand the owner directly to `/ant-plan`.**

## Performance

- **Duration:** 26 minutes
- **Started:** 2026-09-03T21:55:57Z
- **Completed:** 2026-09-03T22:21:37Z
- **Tasks:** 2/2
- **Files modified:** 10 production, test, wrapper, and planning files

## Accomplishments

- Replaced the hundreds-command default root inventory with `Normal journey`, `Steer and inspect`, and `Expert maintenance`, including responsive rendering and exact empty/active standing from the lifecycle projection.
- Made init own first-run setup without a separate setup detour, refuse a guided active-colony replacement before storage can write, and render Queen opening, setup, accepted intent, territory, and closeout in order.
- Added `accepted-charter/v1` state with episode ID, accepted goal, owner provenance, acceptance time, and optional detailed charter so reload reproduces the visible intent summary.
- Synchronized canonical YAML, Claude flat/nested, and OpenCode help around one runtime command and one approved public command order.

## Task Commits

Task 1 retained its required TDD gates, and Task 2 was committed atomically:

1. **Task 1: Make help and init the one normal front door** — `4609a35e` (RED), `1a4e3286` (GREEN)
2. **Task 2: Regenerate the public help surface from canonical YAML** — `35f6aac8`

**Plan metadata:** committed with this summary.

## Files Created/Modified

- `cmd/front_door_199_test.go` — golden group, standing, width, bootstrap, stage, territory, charter, zero-write, and wrapper parity coverage.
- `cmd/root.go` — lazy Cobra group configuration, curated responsive help, read-only fact loading, and compact projection-derived standing.
- `cmd/init_cmd.go` — pre-store active-state guard, automatic local scaffold, accepted episode construction, typed territory stage, and exact `/ant-plan` closeout.
- `cmd/lifecycle_facts.go` — accepted episode propagation through the shared identity projection.
- `pkg/colony/colony.go` — versioned `AcceptedCharter` state contract.
- `.aether/commands/help.yaml` — canonical standing, group membership, descriptions, and passive wrapper body.
- `.claude/commands/ant-help.md`, `.claude/commands/ant/help.md`, `.opencode/commands/ant/help.md` — identical managed runtime delegates.
- `deferred-items.md` — isolated evidence for broader Phase 199 migration failures outside this plan.

## Decisions Made

- Configured help groups lazily inside the help hook so the complete cross-file Cobra tree exists before any `GroupID` is assigned.
- Loaded only state, actor, and blocker facts needed by root help with direct read-only loaders; asking for help never constructs `storage.Store` or creates lock directories.
- Kept the existing raw/Codex `--confirm-reinit` compatibility path, but made guided Claude/OpenCode init an unconditional zero-write refusal when a goal-bearing colony already exists.
- Reused `ensureRepoLocalScaffold` for the automatic setup stage and classified its result as `Ready`, `Bootstrapped`, or an actionable failure.
- Kept richer repo-aware proposals available for compatibility while making the visible fifth-stage closeout end with the exact normal next command `/ant-plan`.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - Missing Critical] Propagated the accepted episode through lifecycle identity**

- **Found during:** Task 1 active-standing implementation
- **Issue:** The shared lifecycle projection exposed colony name and goal but had no field for the required accepted episode, so help could only obtain it by bypassing the projection.
- **Fix:** Added `LifecycleIdentityFacts.Episode` and populated it from the persisted `AcceptedCharter` in both in-memory and disk-backed fact loaders.
- **Files modified:** `cmd/lifecycle_facts.go`
- **Verification:** `TestFrontDoorHelpActiveStanding` compares the full projection-derived standing byte-for-byte.
- **Committed in:** `1a4e3286`

**2. [Rule 1 - Test Bug] Made exact-copy assertions compatible with responsive wrapping**

- **Found during:** Task 1 GREEN verification at widths 63, 64, and 100
- **Issue:** The initial RED test simultaneously required long descriptions to remain on one physical line and required every physical line to fit the terminal width.
- **Fix:** Normalized whitespace only for exact semantic-copy assertions while retaining strict per-line width checks.
- **Files modified:** `cmd/front_door_199_test.go`
- **Verification:** All width fixtures preserve complete commands/descriptions with no line beyond the selected width.
- **Committed in:** `1a4e3286`

**3. [Rule 2 - Missing Critical] Aligned runtime inspection entries with actual public wrappers**

- **Found during:** Task 2 wrapper parity implementation
- **Issue:** The first runtime group draft named raw research/review-ledger commands that do not have matching guided wrapper surfaces.
- **Fix:** Used the existing Oracle-status, Dream, interpretation, memory, and flags views and included `cmd/root.go` in the wrapper-parity commit so runtime and generated membership cannot drift.
- **Files modified:** `cmd/root.go`, `cmd/front_door_199_test.go`
- **Verification:** `TestFrontDoorHelpWrapperParity` compares canonical, runtime, Claude, and OpenCode membership/order.
- **Committed in:** `35f6aac8`

---

**Total deviations:** 3 auto-fixed (1 Rule 1, 2 Rule 2).
**Impact on plan:** Each adjustment was required to make the approved projection and wrapper contracts internally consistent; no new lifecycle command or dependency was added.

## Known Stubs

None. The `Not available` fallback in help is an explicit unavailable-evidence label, not mock data; accepted init and active standing are wired to persisted state and typed facts.

## Threat Flags

| Flag | File | Description |
|------|------|-------------|
| threat_flag: local-state-read | `cmd/root.go` | Root help now reads colony state, actor ledger, and blocker facts directly through existing read-only compatibility loaders; malformed or absent evidence remains typed and no store/lock path is opened. |

## Verification

- Passed all ten `TestFrontDoor*` cases, including the exact active line, widths 63/64/100, automatic bootstrap, five ordered stages, four typed territory labels, durable charter reload, zero-write refusal, and platform wrapper parity.
- Passed `TestCommandSourceHygiene`, source-check regressions, the existing init/root/lifecycle compatibility selection, and `go test ./pkg/colony`.
- Manually exercised root help at 64 columns and command-specific init help; commands remain complete and expert detail stays reachable.
- `git diff --check` passed before each task commit, and forbidden retired names are absent from all four canonical/generated help files.
- The broader `go test ./cmd -count=1` run reached the known staged Phase 199 migration failures: legacy status/Next Up expectations, the archived Phase 196 fixture, and stale maturity/status catalog metadata. Each class is isolated and recorded in `deferred-items.md`; no front-door test failed.

## Issues Encountered

- The repository-wide command suite is not yet green because current Phase 199 plans intentionally supersede older status and recommendation snapshots. Standalone checks show these failures do not exercise Plan 199-07 code.
- The installed GSD requirement updater could not recognize this repository's `**ID — label:**` checkbox format; after confirming CEC-01 and CEC-02 were already complete, LIFE-01 was checked directly and the resulting one-line diff was verified.
- No authentication gates, package installs, or external services were encountered.

## User Setup Required

None - init now performs the required local setup automatically.

## Next Phase Readiness

- Plan can rely on one accepted goal/episode, a typed territory status, and an exact `/ant-plan` handoff without teaching setup or registry internals.
- Init wrapper/skill parity work can consume the persisted accepted-charter contract and five-stage runtime result rather than reconstructing ceremony text.
- Later Phase 199 snapshot cleanup must resolve the deferred legacy expectations before the final normal/race repository gate.

## Self-Check: PASSED

- All nine implementation/wrapper files and this summary exist.
- RED, GREEN, and wrapper task commits resolve to committed Git objects.
- Stub scan found only the intentional explicit `Not available` evidence fallback; no mock or placeholder data feeds the public surfaces.
- Protected pre-existing paths remain unstaged and unchanged by this plan.

---
*Phase: 199-front-door-and-classic-contract*
*Completed: 2026-09-03*
