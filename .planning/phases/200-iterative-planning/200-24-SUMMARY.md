---
phase: 200-iterative-planning
plan: 24
subsystem: init-front-door
tags: [init, discuss, specification, lifecycle-authority, wrappers, claude, opencode, codex]

requires:
  - phase: 200-11
    provides: settled Discuss creates and renders a draft specification
  - phase: 200-12
    provides: shared lifecycle facts and authority-aware next-action projection
  - phase: 200-19
    provides: canonical planning and specification terminal presentation
provides:
  - runtime init persistence and closeout sourced from the shared lifecycle projection
  - synchronized Claude and OpenCode init wrappers that stop at Discuss
  - ordered front-door proof from init through specification, planning candidate, acceptance, and reconciliation
  - direct Codex proof using only runtime-native aether command spelling
affects: [200-23, 200-25, init, discuss, specification, command-wrappers, next-action]

tech-stack:
  added: []
  patterns: [shared lifecycle authority, exploratory-proposal separation, specification-first handoff, platform-native command spelling]

key-files:
  created: [cmd/front_door_200_test.go]
  modified:
    - cmd/init_cmd.go
    - cmd/front_door_199_test.go
    - cmd/init_proposals_test.go
    - cmd/front_door_init_wrapper_199_test.go
    - .aether/commands/init.yaml
    - .claude/commands/ant-init.md
    - .claude/commands/ant/init.md
    - .opencode/commands/ant/init.md

key-decisions:
  - "Repository-aware init proposals remain visible context, but only the shared lifecycle projection may determine persisted or displayed next-action authority."
  - "Init ends at Discuss; Discuss creates the draft specification, Specification owns exact approval, and only then may Planning begin."
  - "Claude and OpenCode advertise /ant-* commands while direct Codex output remains honest runtime-native aether syntax."

patterns-established:
  - "Authority separation: exploratory rankings may explain options but cannot override the lifecycle state table."
  - "Wrapper journey parity: canonical YAML and all managed init wrappers name the same sequential authority boundaries without host-side state writes."

requirements-completed: [SYNTH-02, CEC-03, PLAN-05, PLAN-06]

duration: 18 min
completed: 2026-09-08
---

# Phase 200 Plan 24: Init Specification-First Journey Summary

**Init now persists and presents the shared Discuss handoff, with Claude, OpenCode, and direct Codex all preserving the Discuss → draft Specification → exact approval → Planning authority sequence.**

## Performance

- **Duration:** 18 min
- **Started:** 2026-09-08T00:19:50Z
- **Completed:** 2026-09-08T00:37:48Z
- **Tasks:** 2
- **Files modified:** 9 plan-owned runtime, wrapper, and test files; 1 phase deferred-items log

## Accomplishments

- Replaced init's persisted top-proposal handoff with the shared lifecycle resolver, so `session.json`, recovery artifacts, result fields, and the visible closeout all agree on `aether discuss`.
- Kept repository-aware planning, discussion, and survey proposals as non-authoritative context instead of deleting their ranking behavior.
- Added an explicit closeout explanation that Discuss settles material intent before specification review and does not approve either a specification or plan.
- Synchronized canonical YAML, both Claude init entrypoints, and OpenCode on exact `/ant-discuss` closeout guidance and the later draft-`/ant-spec` boundary.
- Added an ordered six-state front-door regression covering init, draft specification review, exact specification approval, planning-candidate review, exact plan acceptance, and affected-scope reconciliation.
- Proved direct Codex presentation uses `aether` commands and never advertises unsupported `/ant-*` or `$ant-*` syntax.

## Task Commits

1. **Task 1 RED: Define specification-first init journey** — `c690096a` (test)
2. **Task 1 GREEN: Route init through lifecycle authority** — `1de40775` (feat)
3. **Task 2: Align init wrappers with specification journey** — `f49bd2a3` (feat)

## Files Created/Modified

- `cmd/init_cmd.go` — Persists the shared next-action command and explains the Discuss-before-Specification boundary.
- `cmd/front_door_200_test.go` — Covers the complete authority state table and direct Codex spelling.
- `cmd/front_door_199_test.go` — Migrates the Phase 199 init closeout and authority-safe help expectations.
- `cmd/init_proposals_test.go` — Preserves proposal safety while asserting the authoritative Discuss handoff and migrated wrapper closeout.
- `cmd/front_door_init_wrapper_199_test.go` — Verifies canonical, flat Claude, nested Claude, and OpenCode journey parity plus forbidden writes.
- `.aether/commands/init.yaml` — Defines the canonical init-to-discuss-to-spec-to-plan journey and authority guardrails.
- `.claude/commands/ant-init.md` — Aligns the flat Claude entrypoint with the runtime-backed Discuss closeout.
- `.claude/commands/ant/init.md` — Aligns the canonical Claude entrypoint with the specification-first journey.
- `.opencode/commands/ant/init.md` — Aligns OpenCode semantics and command spelling with Claude.
- `.planning/phases/200-iterative-planning/deferred-items.md` — Records unrelated repository-wide migration failures without crossing plan ownership.

## Decisions Made

- The shared resolver is the sole authority for post-init next action. The proposal ranker remains useful for repository context but cannot cause a direct jump to Colonize or Plan.
- Init does not invoke Discuss automatically. It renders runtime truth, offers the owner the exact next command, and leaves all later state changes to their own Go-backed lifecycle commands.
- Draft specification creation belongs to settled Discuss, exact specification approval belongs to Specification, and exact candidate acceptance remains distinct from both.
- Platform parity means equal lifecycle outcomes with honest spelling: `/ant-discuss` and `/ant-spec` on Claude/OpenCode, `aether discuss` and `aether spec` on Codex.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - State integrity] Preserved the actual next incomplete plan**
- **Found during:** Post-plan state synchronization
- **Issue:** `STATE.md` correctly identified Plan 21 as the next incomplete plan because Plan 24 was executed out of numerical order. Blindly incrementing that pointer would falsely skip unfinished Plan 21.
- **Fix:** Left the current-position pointer at Plan 21 while the summary-count, metrics, session, decisions, and roadmap handlers recorded Plan 24 independently.
- **Files modified:** `.planning/STATE.md`, `.planning/ROADMAP.md`
- **Commit:** Final metadata commit

## Issues Encountered

- The repository-wide `go test ./...` sweep reported 5,822 passes, 34 failures, and 6 skips in legacy Plan/Discuss/schema/vocabulary migration gates owned by other plans. No failure names a Plan 24 init file. The exact Plan 24 gate passes, and the out-of-scope result is recorded in `deferred-items.md`.
- `state.update-progress` found no legacy Markdown progress-bar field; the state frontmatter still recalculated from 54 to 55 completed plans through the metric/state transaction.
- `requirements.mark-complete` does not parse this milestone's bold-ID requirement syntax. All four Plan 24 requirement rows were already checked before execution, so no requirement text needed changing.

## TDD Gate Compliance

- RED commit `c690096a` established three expected failures: persisted init handoff, visible specification-first explanation, and direct Codex closeout.
- GREEN commit `1de40775` followed it and made all 31 Task 1 focused tests pass before wrapper synchronization began.

## Verification

- `go test ./cmd -run 'Test(InitSuggestedNextMatchesTopProposal|InitWrapper.*Discuss|FrontDoorHelp|FrontDoorInit|FrontDoor200)' -count=1` — 31 passed.
- `go test ./cmd -run 'TestFrontDoorInitWrapperParity|TestInitSuggestedNextMatchesTopProposal|Test.*Init.*Wrapper|TestFrontDoor200' -count=1` — 24 passed.
- `go run ./cmd/aether source-check` — passed all 16 canonical-source, 5 retired-mirror, and 124 generated-wrapper checks with no findings.
- `git diff --check` — passed before task commits.
- Stub scan across all nine plan-owned files — no functional stubs; matches were ordinary Go empty-value guards, test fixtures, and the existing phrase “colony todos.”

## Known Stubs

None.

## User Setup Required

None - no external service or local configuration is required.

## Next Phase Readiness

- Plan 200-23 can use the ordered state-table proof and exact-init regression as part of the complete public journey.
- Plan 200-25 can rely on canonical init wrappers already using the restored lifecycle authority and platform spellings.
- The broader repository migration failures remain assigned to their existing owners and do not block the Plan 24 contract.

## Self-Check: PASSED

- All nine plan-owned implementation/test/wrapper files and this summary exist.
- Task commits `c690096a`, `1de40775`, and `f49bd2a3` are present in repository history.

---
*Phase: 200-iterative-planning*
*Completed: 2026-09-08*
