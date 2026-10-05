---
phase: 209-light-default-path
plan: 04
subsystem: cli
tags: [cobra, cli, specification, planning, single-door]

requires:
  - phase: 209-light-default-path (plan 01)
    provides: "/ant-go's small route, resolveJobSizeRoute, runGoJob/renderGoVisual"
  - phase: 209-light-default-path (plan 02)
    provides: "the big route's own branch in runGoJob and D-03's self-escalation, which this plan's ensureGoPlanningSpecification call is wired into"
provides:
  - "ensureGoPlanningSpecification -- derives and approves a specification from the owner's own sentence when a project has none, or none approved, so requireApprovedPlanningSpecification never refuses the single door's big route"
  - "goDefaultPlanningPreset (\"fast\") -- the shallowest named planning preset, wired into all four /ant-go wrapper sources so the planning route the big route hands off to asks no depth question"
  - "the four /ant-go wrapper sources documenting the full goal-to-built-work sequence with no owner-typed step in between, while /ant-discuss, /ant-spec and deeper presets remain fully available on request"
affects: [210, any later phase touching /ant-go's routing, the specification machinery, or the planning wrapper]

actuals:
  tokens: 12900
  tasks: 3
  commits: 4

tech-stack:
  added: []
  patterns:
    - "Two writers for one lifecycle, never a third: createGoDerivedSpecificationDraft calls createSpecificationDraft (the specification machinery's other entry point, used when no specification exists at all -- the ordinary shape of a project reached purely through init-then-go, since aether init never creates one) while addGoDerivedSpecificationOutcome calls runSpecCommand's Add operation (used when an unapproved draft already exists); both then approve through runSpecCommand's Approve operation exactly once. Never a bespoke third specification writer."
    - "Check first, touch nothing if already settled: ensureGoPlanningSpecification's first act is requireApprovedPlanningSpecification itself -- if that already succeeds (owner-approved, or a prior derived approval), the function returns immediately having read but not written anything."
    - "Every identifier is derived, never typed: the goal ID (pendingDecisionGoalHash), the approval token (colony.CanonicalSpecificationApprovalToken), and every revision id/content hash are all read from the specification machinery's own return values -- grep -nE '\"[0-9a-f]{16,}\"' cmd/go_default_path.go returns nothing."

key-files:
  created:
    - cmd/go_default_path.go
    - cmd/go_default_path_test.go
  modified:
    - cmd/go_cmd.go
    - cmd/go_escalation_test.go
    - .aether/commands/go.yaml
    - .claude/commands/ant-go.md
    - .claude/commands/ant/go.md
    - .opencode/commands/ant/go.md

key-decisions:
  - "aether init never creates a specification (only an accepted charter), so a project reached purely through init-then-go has no specification at all the first time the single door's big route runs. The plan's own <action> text described only inspect/add/approve through runSpecCommand, which all require an existing specification -- verified empirically (a bare `aether spec --inspect` after `aether init` errors \"no specification exists\"). Fixed by adding a bootstrap branch (createGoDerivedSpecificationDraft) that builds a full nine-section draft via createSpecificationDraft, the specification machinery's own initial-draft entry point, when state.Specification is nil; the add-one-item path (addGoDerivedSpecificationOutcome) is used only when a specification already exists but is not yet approved. Documented here rather than silently deviating from the plan's literal text."
  - "goDefaultPlanningPreset is \"fast\" (not \"quick\"): resolvePlanningPreset's --preset flag path (aliases=false) only recognises the four literal planningStagePreset values (fast/balanced/deep/exhaustive); \"quick\" only resolves through the legacy --depth alias path. Using the exact preset name avoids relying on an alias the wrapper's own `aether plan --preset` invocation does not take."
  - "TestGoBigRouteTakesTheJobToPlanning (plan 02) asserted the big route wrote nothing under .aether/data for a recorded project. That was true before this plan and is now factually superseded by this plan's own explicit deliverable (derive and approve a specification there) -- updated the assertion to a scoped check that still fails on any write outside the specification machinery's own bookkeeping (COLONY_STATE.json, transactions/, .aether-transactions/), rather than weakening or deleting the check. See Deviations."
  - "Verified end-to-end against the real compiled binary (aether go, aether spec --inspect, aether plan) before trusting the Go tests, including the bug reproduction for --synthetic planning against an already-approved specification (plan_authority_legacy_invalid) -- confirmed pre-existing and unrelated by finding e2e_lifecycle_test.go already works around it with the same acceptStagedPlanningCandidate200 helper this plan's own real-flow test (Task 2) reuses."

requirements-completed: [UED-17]

coverage:
  - id: D1
    description: "A big-route job for a project with a recorded goal and no approved specification leaves the project in a state where requireApprovedPlanningSpecification returns no error, with the owner's own sentence as the one outcome item -- and a second run with the same sentence does not create a second approved revision."
    requirement: "UED-17"
    verification:
      - kind: unit
        ref: "cmd/go_default_path_test.go#TestGoBigRouteLeavesPlanningReadyToRun"
        status: pass
      - kind: unit
        ref: "cmd/go_default_path_test.go#TestGoNeverTouchesAnOwnerApprovedSpecification"
        status: pass
    human_judgment: false
  - id: D2
    description: "One sentence reaches built work with the owner typing exactly one command and zero further owner-typed steps, and the planning route the single door hands off to asks no depth question."
    requirement: "UED-17"
    verification:
      - kind: unit
        ref: "cmd/go_default_path_test.go#TestGoalReachesBuiltWorkWithNoExtraSteps"
        status: pass
    human_judgment: false
  - id: D3
    description: "Clarifying intent, hand-drafting and approving a specification, and a deeper planning preset all still work exactly as before this plan."
    verification:
      - kind: unit
        ref: "cmd/go_default_path_test.go#TestDiscussAndSpecStillBehaveExactlyAsBefore"
        status: pass
    human_judgment: false
  - id: D4
    description: "The four /ant-go wrapper sources document the full goal-to-built-work sequence -- planning route with the default preset supplied, then build, then the check command -- while explicitly naming that /ant-discuss, /ant-spec and a deeper preset remain available on request."
    requirement: "UED-17"
    verification: []
    human_judgment: true
    rationale: "Wrapper prose read by an AI orchestrator/chat, not asserted by an automated test beyond the parity/audit checks already covering command names and flags (TestClassicCommandParity, TestCommandCallsMatchCobraContracts, TestDocumentedCommandNamesResolve, all passing) -- a human should read the actual wording once."

duration: 2h 30min
completed: 2026-09-24
status: complete
---

# Phase 209 Plan 04: Planning Through the Single Door Never Stops for a Missing Specification Summary

**`ensureGoPlanningSpecification` derives and approves a specification from the owner's own sentence whenever `/ant-go`'s big route reaches a project with none (or none approved), so `requireApprovedPlanningSpecification` never refuses — and the four `/ant-go` wrappers now run the planning route with a fixed "fast" preset and go straight to build, with zero owner-typed steps between the goal and built work.**

## Performance

- **Duration:** ~2h 30min
- **Started:** 2026-09-24T16:10:00Z (approx)
- **Completed:** 2026-09-24T18:40:00Z
- **Tasks:** 3
- **Files modified:** 8 (2 created, 6 modified)

## Accomplishments

- `ensureGoPlanningSpecification` (`cmd/go_default_path.go`) checks `requireApprovedPlanningSpecification` first and touches nothing if it already succeeds; otherwise it derives one outcome item from the owner's sentence — via `createGoDerivedSpecificationDraft` when no specification exists at all (the ordinary shape of `aether init` then `aether go`, since `aether init` never creates a specification), or `addGoDerivedSpecificationOutcome` when an unapproved draft already exists — then approves it through `runSpecCommand`'s Approve operation exactly once.
- `runGoJob`'s big-route branch now calls this whenever a project is recorded, reports `specification_source` (`derived-from-request` / `owner-approved`), and `renderGoVisual` prints one plain-English line saying which happened, before planning starts.
- Verified end-to-end against the real compiled binary: `aether init` → `aether go "<big sentence>"` → `aether plan --preset fast --synthetic` succeeded immediately (no missing-specification refusal), and a second `aether go` run with the same sentence reported `owner-approved` with the revision count unchanged.
- `goDefaultPlanningPreset` ("fast") is now supplied by all four `/ant-go` wrapper sources when they hand off to the planning route, so `resolvePlanningPreset` never puts a depth question to the owner; the wrappers also now say explicitly that clarifying intent, drafting, and approving a specification by hand are skipped on this path but remain fully available (`/ant-discuss`, `/ant-spec`, a deeper preset) any time the owner asks for them by name.
- `TestGoalReachesBuiltWorkWithNoExtraSteps` proves the real flow: a project whose only recorded input is the goal reaches built work (`colony.StateBUILT`) through exactly one owner-typed command, with the planning stage faked the same way `e2e_lifecycle_test.go` already fakes real Scout/Route-Setter dispatch (`acceptStagedPlanningCandidate200`) and the build stage run through the real `build --synthetic` command.
- `TestDiscussAndSpecStillBehaveExactlyAsBefore` is the standing "nothing is taken away" guard: clarifying intent still asks about and records an unresolved material decision, hand-drafting and approving a specification still produces the same revision/approval shape, and a deeper preset (balanced/deep/exhaustive) still resolves to its own policy from `planningPresetPolicies` rather than the single door's default.

## Task Commits

1. **Task 1: Planning reached through the single door never stops for a missing specification** - `9957da1d` (feat)
2. **Task 2: One pass, no depth question, and the route from goal to built work has no steps in between** - `35e29487` (feat)
3. **Task 3: Nothing that worked before works differently now** - `7dbeee65` (test)
4. **Deviation fix: update plan 02's big-route test for the new derivation write** - `d14e9a6c` (fix)

**Plan metadata:** commit follows this SUMMARY (docs).

## Files Created/Modified

- `cmd/go_default_path.go` - `ensureGoPlanningSpecification`, `createGoDerivedSpecificationDraft`, `addGoDerivedSpecificationOutcome`, `goAcceptedGoalEvidenceID`, and the `goDerivedSpecificationLineage`/`goDefaultPlanningPreset`/`goSpecificationSource*`/`goAcceptedGoalEvidenceOrigin` constants
- `cmd/go_cmd.go` - wires `ensureGoPlanningSpecification` into `runGoJob`'s big-route branch and adds the specification-source line to `renderGoVisual`
- `cmd/go_default_path_test.go` - `TestGoBigRouteLeavesPlanningReadyToRun`, `TestGoNeverTouchesAnOwnerApprovedSpecification`, `TestGoalReachesBuiltWorkWithNoExtraSteps`, `TestDiscussAndSpecStillBehaveExactlyAsBefore`
- `cmd/go_escalation_test.go` - `TestGoBigRouteTakesTheJobToPlanning`'s "recorded project" subtest updated for the new, intentional specification-derivation write (see Deviations)
- `.aether/commands/go.yaml`, `.claude/commands/ant-go.md`, `.claude/commands/ant/go.md`, `.opencode/commands/ant/go.md` - the planning-route routing step now names the default preset, the build step, the check command, and what is deliberately skipped but still available

## Decisions Made

See `key-decisions` in frontmatter above.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Specification bootstrap when none exists at all**
- **Found during:** Task 1
- **Issue:** The plan's `<action>` described `ensureGoPlanningSpecification` purely as inspect → add → approve, all through `runSpecCommand`. Empirically verified (real compiled binary, fresh temp project) that `aether init` never creates a specification — only an accepted charter — so `runSpecCommand`'s Inspect and Add operations both fail with "no specification exists" on the ordinary init-then-go path, which is exactly this plan's primary target scenario.
- **Fix:** Added `createGoDerivedSpecificationDraft`, which builds a full nine-section draft (all sections populated, most with plain-English filler text honestly labelled as derived from the request) via `createSpecificationDraft` — the specification machinery's own initial-draft entry point, used by `discuss.go`'s settled-draft flow and nowhere else previously. `ensureGoPlanningSpecification` calls this only when `state.Specification == nil`; otherwise it uses the plan's literal inspect→add sequence (`addGoDerivedSpecificationOutcome`) against the existing lineage. Both paths converge on the same single `runSpecCommand` Approve call.
- **Files modified:** `cmd/go_default_path.go`
- **Verification:** `TestGoBigRouteLeavesPlanningReadyToRun` builds exactly this scenario (goal-only project, no specification) and passes; manually verified against the real binary end-to-end.
- **Committed in:** `9957da1d` (Task 1 commit)

**2. [Rule 1 - Bug/stale test] Updated `TestGoBigRouteTakesTheJobToPlanning` for the new derivation write**
- **Found during:** Task 1 verification (running the plan's own `<verification>` block surfaced this failure)
- **Issue:** This plan-02 test asserted the big route wrote nothing under `.aether/data` for a project with a recorded goal — an assertion that was correct only because Task 1's specification derivation did not exist yet. This plan's own explicit deliverable is exactly that derivation, so the assertion's premise is now factually superseded, not accidentally broken.
- **Fix:** Replaced the byte-exact before/after equality check with `aetherDataChangeOutsideSpecification`, which still fails if the big route writes anything outside the specification machinery's own bookkeeping (`COLONY_STATE.json`, `transactions/`, `.aether-transactions/`) — preserving the test's real protective intent (no stray writes, no helper dispatch) while accommodating the new, intentional write. Added an assertion that `specification_source` is `derived-from-request` for this fixture.
- **Files modified:** `cmd/go_escalation_test.go`
- **Verification:** `TestGoBigRouteTakesTheJobToPlanning` passes; the full `go test ./cmd -run 'TestGo|TestSpec|TestDiscuss|TestPlanningPreset'` run shows only the three pre-existing, unrelated failures named in this project's own repo-specific rules.
- **Committed in:** `d14e9a6c`

---

**Total deviations:** 2 auto-fixed (1 blocking bootstrap gap, 1 stale test premise). **Impact on plan:** Both were necessary to make the plan's own primary scenario (init-then-go with no specification) actually work, and to keep the build green afterward. No scope creep beyond what Task 1's own deliverable required.

## Issues Encountered

While reproducing the real flow manually against the compiled binary, `aether plan --synthetic` against a project that already has an approved specification (derived or hand-approved) refused with `plan_authority_legacy_invalid` when combined with a plain `aether build --synthetic` immediately after — a pre-existing gap in synthetic planning's interaction with specification-bound plans, unrelated to this plan's changes. Confirmed pre-existing by finding `cmd/e2e_lifecycle_test.go` already documents and works around exactly this ("so a Specification-bearing plan can never enter build as legacy_unbound") using `acceptStagedPlanningCandidate200` instead of driving `aether plan --synthetic` directly. `TestGoalReachesBuiltWorkWithNoExtraSteps` (Task 2) uses that same existing helper rather than the buggy CLI path. Not fixed here — out of this plan's scope (a pre-existing runtime gap in the `--synthetic` flag, not a defect in `/ant-go`'s routing or specification derivation) and not recorded in WINDOWS.md since it was already implicitly documented by the existing test's own comment.

## User Setup Required

None - no external service configuration required.

## Verification

```
go build ./cmd/aether && go vet ./cmd
go test ./cmd -count=1 -timeout 480s -run 'TestGo|TestSpec|TestDiscuss|TestPlanningPreset'
```

Result: 100 subtests/tests passed. The only 3 failures are the pre-existing, unrelated ones named in this repo's own CLAUDE.md repo-specific rules (`TestGoSourceHintsMatchCobraContracts`, `TestGoldenBuildVisualOutput`, `TestGoldenContinueVisualOutput`) — none touched, none caused by this plan.

`grep -nE '"[0-9a-f]{16,}"' cmd/go_default_path.go` returns nothing (no hash, id, or token written as a literal).

## Next Phase Readiness

- `/ant-go`'s big route now reaches a planning-ready state unconditionally for any recorded project, and the wrapper carries the job to built work with zero further owner-typed steps.
- 209-05 (the timing writeup, ROADMAP success criterion 4) remains to execute this phase.
- No blockers carried forward from this plan. The pre-existing `--synthetic` planning + specification interaction gap (see Issues Encountered) is not new and was not introduced by this plan; it does not block `/ant-go`'s real (non-synthetic) planning path, which routes through real Scout/Route-Setter dispatch, never `--synthetic`.

## Known Stubs

None. Both the specification derivation and the wrapper routing are real, production-quality implementations with no placeholder branches.

## Self-Check: PASSED

- `FOUND: cmd/go_default_path.go`
- `FOUND: cmd/go_default_path_test.go`
- `FOUND: 9957da1d` (Task 1 commit) in `git log --oneline --all`
- `FOUND: 35e29487` (Task 2 commit) in `git log --oneline --all`
- `FOUND: 7dbeee65` (Task 3 commit) in `git log --oneline --all`
- `FOUND: d14e9a6c` (deviation fix commit) in `git log --oneline --all`

---
*Phase: 209-light-default-path*
*Completed: 2026-09-24*
