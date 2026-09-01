---
phase: 198-put-the-thrown-away-data-back-on-screen
plan: 04
subsystem: cli-visuals
tags: [go, ceremony, plan, seal, closeout, D-12, SHOW-01]

# Dependency graph
requires:
  - phase: 198-01
    provides: "closeoutDirectVisual / closeoutContinueRenderInputs bridge architecture, dual-type rendering precedent, fixture-round-trip test pattern"
  - phase: 198-03
    provides: "runSealConfirmationGate's awaiting_owner_confirmation pending result shape"
provides:
  - "closeoutDirectVisual routes plan and seal workflows through renderPlanVisual / renderSealVisual (the same direct-path renderers), completing SHOW-01/D-12's three-workflow coverage (plan, continue, seal)"
  - "closeoutPlanRenderInputs / planConfidenceFromValue / planRevisionFromValue -- normalise a round-tripped completion map's confidence and plan_revision fields back into the exact struct types both plan construction paths always use"
  - "closeoutSealRenderInputs / defaultCrownedAnthillSummaryPath -- resolve renderSealVisual's summaryPath and report not-handled for a seal result still awaiting the D-04..D-07 confirmation gate's answer"
  - "TestCloseoutRefusesToHalfRenderAnIncompletePayload -- a completion map missing a branch's essential key reports not-handled rather than a half or misleading screen"
affects: [any-future-phase-touching-cmd/codex_visuals.go-confidence-or-plan_revision-rendering, 199-*]

# Actuals (#2632)
actuals:
  tokens: 6500
  tasks: 2
  commits: 2

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Normalise-back-to-native-type: when a renderer's own type-switch recognises only the in-process struct (or only the JSON-round-tripped map) for a given field, and the renderer itself cannot be edited (owned by another concurrent plan), reshape the completion-file value back into the exact dynamic type the in-process construction path always uses, in the bridge function -- not in the renderer. This achieves true parity (byte-equal to whatever the direct path currently does, bugs included) without touching the shared renderer file, and stays correct if a later plan later adds a dual-type switch to that same field."
    - "Essential-key gate: closeoutPlanRenderInputs/closeoutSealRenderInputs report not-handled (ok=false) when a field every real result of that shape always carries (plan's \"goal\", seal's \"sealed\"==true) is absent -- covers both a genuinely foreign payload and a legitimate in-progress state (seal's awaiting-confirmation pending result) that has nothing honest to show yet."

key-files:
  created: []
  modified:
    - cmd/closeout_direct_render.go
    - cmd/wrapper_path_parity_test.go
    - cmd/plan_wrapper_ceremony_test.go
    - .claude/commands/ant/plan.md
    - .opencode/commands/ant/plan.md
    - .claude/commands/ant-plan.md

key-decisions:
  - "confidence and plan_revision are normalised back into their native Go struct types (codexPlanConfidence, colony.PlanRevision) in closeoutPlanRenderInputs, rather than left as JSON-round-tripped maps -- renderPlanVisual's confidence branch only recognises map[string]interface{}, and its plan_revision struct/map branches render genuinely DIFFERENT text (the map branch omits Reason entirely), so passing the round-tripped map through unchanged would have made the chat path show content or wording the direct path does not, the exact divergence D-12 exists to prevent."
  - "closeoutSealRenderInputs treats raw[\"sealed\"] != true as not-handled, which correctly also covers the D-04..D-07 confirmation gate's own pending result (awaiting_owner_confirmation: true) -- verified that both real direct-path entry points (sealCmd's RunE and runSealFinalize) already hand that pending result to outputOK, which always writes raw JSON regardless of output mode, so neither path ever visually renders it as a screen. Falling through to the generic renderer here is genuine parity, not a new omission."
  - "Left the newly-discovered renderPlanVisual confidence-rendering bug (Confidence line never renders on the direct OR chat path, on any call site, because the map-only type assertion never matches the codexPlanConfidence struct both construction functions always store) unfixed in this plan -- fixing it requires editing cmd/codex_visuals.go, which this plan's own frontmatter prohibits (owned by a same-wave sibling plan). Recorded in .planning/WINDOWS.md (entry 5) rather than fixed here."

requirements-completed: [SHOW-01, SHOW-02]

coverage:
  - id: D1
    description: "The chat-path plan closeout (aether ceremony closeout --workflow plan) renders byte-identically to the direct plan-finalize/plan screen for a completed plan and a mid-loop iteration."
    requirement: "SHOW-01"
    verification:
      - kind: unit
        ref: "cmd/wrapper_path_parity_test.go#TestWrapperPathRendersSameCeremonyAsDirectPath/plan_completed"
        status: pass
      - kind: unit
        ref: "cmd/wrapper_path_parity_test.go#TestWrapperPathRendersSameCeremonyAsDirectPath/plan_mid-loop_iteration"
        status: pass
    human_judgment: false
  - id: D2
    description: "The chat-path seal closeout renders byte-identically to the direct seal screen for a plain completed seal and a force-sealed (override) seal."
    requirement: "SHOW-01"
    verification:
      - kind: unit
        ref: "cmd/wrapper_path_parity_test.go#TestWrapperPathRendersSameCeremonyAsDirectPath/seal_completed"
        status: pass
      - kind: unit
        ref: "cmd/wrapper_path_parity_test.go#TestWrapperPathRendersSameCeremonyAsDirectPath/seal_force-sealed_with_overrides"
        status: pass
    human_judgment: false
  - id: D3
    description: "All three workflows the roadmap names (plan, continue, seal) are covered by one named test, TestWrapperPathRendersSameCeremonyAsDirectPath, and codex_visuals.go is untouched by this plan."
    requirement: "SHOW-01"
    verification:
      - kind: unit
        ref: "cmd/wrapper_path_parity_test.go#TestWrapperPathRendersSameCeremonyAsDirectPath"
        status: pass
      - kind: other
        ref: "git diff --name-only 68203b5b HEAD -- cmd/codex_visuals.go"
        status: pass
    human_judgment: false
  - id: D4
    description: "A seal result still awaiting the owner's confirmation is not rendered as sealed on either path; an incomplete plan or seal payload reports not-handled rather than a half-screen."
    verification:
      - kind: unit
        ref: "cmd/wrapper_path_parity_test.go#TestWrapperPathRendersSameCeremonyAsDirectPath/seal_awaiting_owner_confirmation_is_not_rendered_as_sealed_on_either_path"
        status: pass
      - kind: unit
        ref: "cmd/wrapper_path_parity_test.go#TestCloseoutRefusesToHalfRenderAnIncompletePayload"
        status: pass
    human_judgment: false
  - id: D5
    description: "All three planning-wrapper copies describe the shared closing screen identically, and a drift is caught by a named test."
    requirement: "SHOW-01"
    verification:
      - kind: unit
        ref: "cmd/plan_wrapper_ceremony_test.go#TestPlanWrapperCeremonyContract"
        status: pass
      - kind: unit
        ref: "cmd/lifecycle_wrapper_contract_test.go#TestLifecycleFlatMirrorsMatchCanonical"
        status: pass
      - kind: other
        ref: "diff .claude/commands/ant/plan.md .opencode/commands/ant/plan.md && diff .claude/commands/ant/plan.md .claude/commands/ant-plan.md"
        status: pass
    human_judgment: false

duration: 55min
completed: 2026-08-29
status: complete
---

# Phase 198 Plan 04: Route Plan And Seal Closeout Through The Direct Renderer Summary

**`closeoutDirectVisual` now covers plan and seal alongside continue -- `TestWrapperPathRendersSameCeremonyAsDirectPath` proves byte-equal chat/direct output for all three roadmap-named workflows, without touching the shared renderer file a same-wave plan owns.**

## Performance

- **Duration:** ~55 min
- **Started:** 2026-08-29 (approx.)
- **Completed:** 2026-08-29
- **Tasks:** 2
- **Files modified:** 6

## Accomplishments

- Added `plan` and `seal` branches to `closeoutDirectVisual`, each mirroring `closeoutContinueDirectVisual`'s `completion_raw` / double-envelope-unwrap handling (a plan or seal result has no manifest key and no dispatches/results/workers key either, so the shared `closeoutCompletionDetails` unwrap heuristic doesn't recognize it as an envelope -- the same gap 198-01 already documented for continue).
- `closeoutPlanRenderInputs` normalises a round-tripped completion map's `confidence` and `plan_revision` fields back into the exact `codexPlanConfidence` / `colony.PlanRevision` struct types both `runCodexPlanWithOptions` and `runCodexPlanFinalize` always populate them with -- discovered mid-task that `renderPlanVisual`'s confidence branch only recognises the map form (never the struct form both construction paths actually use), so without this normalisation the chat path would have shown a confidence line the direct terminal path does not, the exact divergence this whole plan exists to close.
- `closeoutSealRenderInputs` resolves the seal summary path (from the completion map's own key, falling back to the standard `CROWNED-ANTHILL.md` location) and reports not-handled when `sealed` is not `true` -- correctly covering both a foreign payload shape and the D-04..D-07 confirmation gate's own "awaiting the owner's answer" pending result, which neither the direct nor chat path has ever visually rendered (both hand it to `outputOK`, which always writes raw JSON regardless of output mode).
- Extended `TestWrapperPathRendersSameCeremonyAsDirectPath` with five new subtests (plan completed, plan mid-loop, seal completed, seal force-sealed, seal awaiting confirmation) and added `TestCloseoutRefusesToHalfRenderAnIncompletePayload` (four subtests: plan missing goal, plan empty payload, seal missing sealed flag, seal sealed=false).
- Updated all three byte-identical planning-wrapper copies with plain-English prose confirming the shared closing screen, and extended `TestPlanWrapperCeremonyContract`'s required-substring list.

## Task Commits

Each task was committed atomically:

1. **Task 1: Route planning and finishing through the same one renderer** - `f0c81ffe` (feat)
2. **Task 2: Update all three planning-wrapper copies and lock them** - `e48e8f46` (docs)

**Plan metadata:** committed with this SUMMARY.

## Files Created/Modified

- `cmd/closeout_direct_render.go` - `closeoutPlanDirectVisual`, `closeoutPlanRenderInputs`, `planConfidenceFromValue`, `planRevisionFromValue`, `closeoutSealDirectVisual`, `closeoutSealRenderInputs`, `defaultCrownedAnthillSummaryPath`; `closeoutDirectVisual`'s switch gains `plan` and `seal` cases
- `cmd/wrapper_path_parity_test.go` - new fixtures (`wrapperParityPlanFinalizeResult`, `wrapperParityPlanIterationResult`, `wrapperParitySealResult`, `wrapperParityForceSealedResult`, `wrapperParitySealConfirmationPendingResult`); five new `TestWrapperPathRendersSameCeremonyAsDirectPath` subtests; new `TestCloseoutRefusesToHalfRenderAnIncompletePayload`
- `cmd/plan_wrapper_ceremony_test.go` - extended `required` substring list with the new prose
- `.claude/commands/ant/plan.md`, `.opencode/commands/ant/plan.md`, `.claude/commands/ant-plan.md` - byte-identical new sentence after the closeout call, explaining the shared closing screen in plain English

## Decisions Made

See `key-decisions` in frontmatter above (confidence/plan_revision normalised back to native struct types; seal's `sealed != true` not-handled gate also correctly covers the confirmation-pending state; the discovered confidence-rendering bug is recorded, not fixed, per this plan's own file-ownership prohibition).

## Deviations from Plan

### Auto-fixed Issues

None -- this plan required no in-scope bug fixes; the discovered issue below is out of this plan's file scope by the plan's own written prohibition, so it is recorded rather than auto-fixed.

### Discovered, Deferred (out of scope for this plan)

**1. [Pre-existing bug, cmd/codex_visuals.go -- out of scope] `renderPlanVisual`'s confidence line never renders on any path**
- **Found during:** Task 1, while investigating why the round-tripped closeout map would have diverged from the direct render for `confidence`
- **Issue:** `renderPlanVisual` (cmd/codex_visuals.go:1489) type-asserts `result["confidence"]` only to `map[string]interface{}`. Both `runCodexPlanWithOptions` (cmd/codex_plan.go:786) and `runCodexPlanFinalize` (cmd/codex_plan_finalize.go:455,872) always store `confidence` as a `codexPlanConfidence` struct, which never satisfies that assertion. Verified empirically: a direct call to `renderPlanVisual` with a real, non-zero `codexPlanConfidence{Overall: 85}` produces NO "Confidence: N%" line at all -- on any call site, direct or chat. RESEARCH.md's Q5 finding ("Yes, rendered on direct path") did not catch this, because the map-shaped rendering code is real, it is just never reached by either construction path's actual value.
- **Why not fixed here:** the fix location is `cmd/codex_visuals.go`, which this plan's own frontmatter prohibits editing (`git diff --name-only HEAD -- cmd/codex_visuals.go` must produce no output; "a same-wave plan owns that file"). Fixing it would require adding a `codexPlanConfidence` struct branch to that type-switch, mirroring `planning_loop`'s existing dual-type precedent one line below it.
- **What this plan did instead:** normalised the closeout side's `confidence` value back into the SAME struct type the direct side already uses (`planConfidenceFromValue`), so both paths behave identically -- neither renders the line today, and once a future plan adds the struct branch to `renderPlanVisual`, both paths will pick it up identically too, because they will then carry the exact same dynamic type.
- **Recorded:** `.planning/WINDOWS.md` entry 5 (kind: deviation, phase 198).
- **Verification:** confirmed via a throwaway scratch test (`renderPlanVisual` called directly with a real `codexPlanConfidence{Overall: 85}` fixture) that the direct path genuinely omits the line; removed the scratch file before committing.

---

**Total deviations:** 0 auto-fixed; 1 discovered-and-deferred (recorded in WINDOWS.md, not fixed, per this plan's own file-ownership prohibition).
**Impact on plan:** None on this plan's own scope. The deferred item is real and user-visible (plan confidence never shows anywhere today) but requires editing a file this plan is explicitly forbidden from touching; it is now tracked for whichever plan legitimately owns `cmd/codex_visuals.go`'s confidence/duration rendering additions.

## Issues Encountered

None beyond the deviation documented above.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- SHOW-01/D-12's roadmap criterion 1 is closed: `TestWrapperPathRendersSameCeremonyAsDirectPath` now covers plan, continue, and seal in one named test, and `cmd/codex_visuals.go` remains untouched by this plan.
- A genuine, previously-invisible rendering gap was found and recorded (WINDOWS.md entry 5): plan confidence never displays on any surface today. Whichever plan next touches `cmd/codex_visuals.go` (likely a SHOW-02 sibling restoring worker duration/tool-count rendering) should add the `codexPlanConfidence` struct branch alongside `planning_loop`'s existing one, and this plan's normalisation in `closeoutPlanRenderInputs` will make both paths pick it up automatically once that lands.
- No blockers for the remaining plans in this phase (05-09).

---
*Phase: 198-put-the-thrown-away-data-back-on-screen*
*Completed: 2026-08-29*

## Self-Check: PASSED

- Verified `cmd/closeout_direct_render.go`, `cmd/wrapper_path_parity_test.go`, `cmd/plan_wrapper_ceremony_test.go`, `.claude/commands/ant/plan.md`, `.opencode/commands/ant/plan.md`, `.claude/commands/ant-plan.md` all exist on disk with the expected content.
- Verified both task commit hashes (`f0c81ffe`, `e48e8f46`) exist in `git log`.
- Re-ran the plan's full `<verification>` command: `go build ./...` and `go vet ./cmd` exit 0; `go test ./cmd -run 'TestWrapperPathRendersSameCeremonyAsDirectPath|TestCloseoutRefusesToHalfRenderAnIncompletePayload|TestCloseoutUnhandledWorkflowsKeepTheGenericRenderer|TestPlanWrapperCeremonyContract|TestSealWrapperCeremonyContract|TestContinueWrapperCeremonyContract|TestLifecycleFlatMirrorsMatchCanonical' -count=1` -- ok.
- Confirmed `git diff --name-only 68203b5b HEAD -- cmd/codex_visuals.go` produces no output.
- Ran the full `go test ./cmd/...` suite once, 249s, all green, zero failures.
- Confirmed `diff .claude/commands/ant/plan.md .opencode/commands/ant/plan.md` and `diff .claude/commands/ant/plan.md .claude/commands/ant-plan.md` both exit 0.
