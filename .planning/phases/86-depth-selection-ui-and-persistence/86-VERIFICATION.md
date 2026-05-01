---
phase: 86-depth-selection-ui-and-persistence
verified: 2026-05-01T14:30:00Z
status: gaps_found
score: 3/4 must-haves verified
overrides_applied: 0
gaps:
  - truth: "When /ant-plan runs, the user is shown the smart defaults for planning depth and verification depth before plan generation"
    status: partial
    reason: "The main plan generation result map (codex_plan.go line 441-470) is missing verification_depth, verification_smart_default, planning_smart_default, and planning_phase keys. The existing-plan and plan-only paths include them, but the primary fresh-generation path does not. The depth selection banner renders (planning_depth is present), but verification depth is not shown and reason annotations are suppressed because verification_smart_default defaults to false and planningPhase is zero-value."
    artifacts:
      - path: "cmd/codex_plan.go"
        issue: "Main plan generation result map (lines 441-470) missing 4 keys: verification_depth, verification_smart_default, planning_smart_default, planning_phase"
    missing:
      - "Add verification_depth, verification_smart_default, planning_smart_default, planning_phase to the result map at lines 441-470 in cmd/codex_plan.go (mirroring lines 216-219 in the existing-plan path)"
---

# Phase 86: Depth Selection UI and Persistence Verification Report

**Phase Goal:** At `/ant-plan` start, users see the smart default for both depths and can accept or override either, and the selected verification depth persists into the build packet for `/ant-continue`
**Verified:** 2026-05-01T14:30:00Z
**Status:** gaps_found
**Re-verification:** No -- initial verification

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | When `/ant-plan` runs, the user is shown the smart defaults for planning depth and verification depth before plan generation | FAILED | Depth selection banner exists in renderPlanVisual (line 869-909 of codex_visuals.go), but the main plan generation result map (codex_plan.go lines 441-470) is missing verification_depth, verification_smart_default, planning_smart_default, and planning_phase keys. The existing-plan path (line 208-225) and plan-only path (line 566-577) both include them. Only the fresh plan generation path is affected. Banner renders planning_depth without reason; verification depth line is silently skipped. |
| 2 | The user can accept both defaults with a single confirmation, or override either depth individually | VERIFIED | `--verification-depth` flag registered on planCmd (codex_workflow_cmds.go line 963), read in RunE (line 65), passed to codexPlanOptions (line 77). `--planning-depth` flag already existed. resolveVerificationDepthSmart (review_depth.go line 216) validates explicit values and falls through to smart default when empty. |
| 3 | The verification depth selected at plan time is stored in the build packet JSON | VERIFIED | codexBuildManifest struct has `ReviewDepth string json:"review_depth,omitempty"` (codex_build.go line 73). buildCodexBuildManifest populates it (line 1050). Both build flows pass state.VerificationDepth to resolveVerificationDepth (lines 151-152 and 245-246). |
| 4 | `/ant-continue` reads the verification depth from the build packet and uses it without requiring the user to re-specify | VERIFIED | codex_continue_finalize.go lines 202-205 read plan.ReviewDepth and normalize it: `if plan.ReviewDepth != "" { finalizeReviewDepth = colony.NormalizeVerificationDepth(plan.ReviewDepth) }`. This value is used in externalContinueReviewReport (line 206). Tests confirm light and heavy depths propagate correctly (codex_continue_test.go lines 277-278, 320-321). |

**Score:** 3/4 truths verified

### Deferred Items

No deferred items. This is the final phase in the v1.12 depth controls milestone.

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `cmd/review_depth.go` | resolveVerificationDepthSmart function | VERIFIED | Function exists at line 216-229. Validates explicit values, falls through to resolveSmartVerificationDepth when empty. |
| `cmd/codex_workflow_cmds.go` | --verification-depth flag on plan command | VERIFIED | Flag registered at line 963, read at line 65, passed to codexPlanOptions at line 77. |
| `cmd/codex_visuals.go` | Depth selection banner in plan visual output | VERIFIED | Banner at lines 869-909 in renderPlanVisual. Uses renderStageMarker("Depth Selection"), renderSmartDepthReason, renderReviewDepthLineWithReason. Includes override hint. |
| `cmd/codex_plan.go` | VerificationDepth on codexPlanOptions and codexPlanManifest | VERIFIED | codexPlanOptions has VerificationDepth string (line 122). codexPlanManifest has VerificationDepth string (line 138). Both populated correctly. |
| `cmd/review_depth_test.go` | Tests for resolveVerificationDepthSmart | VERIFIED | TestResolveVerificationDepthSmart_ExplicitValue (line 962), TestResolveVerificationDepthSmart_EmptyUsesSmartDefault (line 987), TestResolveVerificationDepthSmart_InvalidValue (line 1012). All pass. |
| `pkg/colony/colony.go` | VerificationDepth field on ColonyState | VERIFIED | Field at line 288: `VerificationDepth string json:"verification_depth,omitempty"`. |
| `cmd/codex_build.go` | ReviewDepth on codexBuildManifest, populated from ColonyState | VERIFIED | Field at line 73. Populated at line 1050. Both build flows read state.VerificationDepth (lines 151, 245). |

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|-----|--------|---------|
| codex_workflow_cmds.go | codex_plan.go | planCmd RunE reads verification-depth flag, passes to codexPlanOptions | WIRED | Line 65 reads flag, line 77 passes to options struct |
| codex_plan.go | review_depth.go | runCodexPlanWithOptions calls resolveVerificationDepthSmart | WIRED | Line 179 in main flow, line 487 in plan-only flow |
| codex_plan.go | codex_visuals.go | result map consumed by renderPlanVisual | PARTIAL | Keys present in existing-plan and plan-only paths. Missing from main plan generation result map (line 441-470). |
| codex_visuals.go | review_depth.go | renderReviewDepthLineWithReason and renderSmartDepthReason called from banner | WIRED | Lines 887, 896 in renderPlanVisual |
| codex_plan.go | pkg/colony/colony.go | plan stores resolved verification depth in state.VerificationDepth | WIRED | Lines 186-189 in main flow, lines 495-498 in plan-only flow |
| codex_build.go | pkg/colony/colony.go | build reads state.VerificationDepth as depthStr | WIRED | Lines 151-152 in plan-only flow, lines 245-246 in full build flow |
| codex_build.go | codex_build.go | buildCodexBuildManifest populates ReviewDepth | WIRED | Function signature accepts reviewDepth parameter (line 1010), populated at line 1050 |
| codex_continue_finalize.go | codex_build.go | Continue reads plan.ReviewDepth from build manifest | WIRED | Lines 202-205 read plan.ReviewDepth, normalize, and use as finalizeReviewDepth |

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
|----------|---------------|--------|-------------------|--------|
| renderPlanVisual (plan-only path) | verification_depth | resolveVerificationDepthSmart result via result map | FLOWING | Plan-only path correctly passes verification_depth, planning_phase, and smart default flags |
| renderPlanVisual (existing-plan path) | verification_depth | resolveVerificationDepthSmart result via result map | FLOWING | Existing-plan path correctly passes all 4 keys |
| renderPlanVisual (main generation path) | verification_depth | MISSING from result map | DISCONNECTED | Main plan generation result map (lines 441-470) does not include verification_depth key |
| renderBuildVisualWithDispatches | reviewDepth | resolveVerificationDepth with state.VerificationDepth | FLOWING | Build reads stored depth from ColonyState, resolves, passes to visual |
| codex_continue_finalize | finalizeReviewDepth | plan.ReviewDepth from build manifest | FLOWING | Manifest stores ReviewDepth, continue reads and normalizes it |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| resolveVerificationDepthSmart tests pass | `go test ./cmd/ -run "TestResolveVerificationDepthSmart" -count=1` | All 5 subtests PASS | PASS |
| Render plan visual tests pass | `go test ./cmd/ -run "TestRenderPlan" -count=1` | All 8 tests PASS | PASS |
| Full cmd test suite passes | `go test ./cmd/ -count=1` | ok (69.275s) | PASS |
| Build-related tests pass | `go test ./cmd/ -run "TestBuild" -count=1` | All 20+ tests PASS | PASS |
| Continue review tests pass | `go test ./cmd/ -run "TestContinueReview" -count=1` | All 9 tests PASS | PASS |
| Go binary compiles | `go build ./cmd/` | No errors | PASS |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|-------------|-------------|--------|----------|
| DEPTH-04 | 86-01-PLAN, 86-02-PLAN | User depth selection at plan time -- present smart defaults, allow override | PARTIAL | Flag, resolver, ColonyState persistence, and banner all exist. Main plan generation result map missing verification keys -- banner partially broken for fresh plans. |
| DEPTH-05 | 86-02-PLAN | Depth persistence across continue -- store in build packet, honored by /ant-continue | SATISFIED | ReviewDepth in build manifest (codex_build.go line 73), populated at line 1050. Continue reads plan.ReviewDepth at codex_continue_finalize.go lines 202-205. Tests confirm propagation. |

### Anti-Patterns Found

No anti-patterns detected. No TODO/FIXME/PLACEHOLDER comments in modified files. No empty implementations or hardcoded stubs.

### Human Verification Required

None. All behaviors are programmatically verifiable through code inspection and tests.

### Gaps Summary

One gap blocks full goal achievement: the main plan generation result map in `cmd/codex_plan.go` (lines 441-470) is missing four keys that the depth selection banner requires: `verification_depth`, `verification_smart_default`, `planning_smart_default`, and `planning_phase`. These keys ARE present in the existing-plan path (lines 216-219) and the plan-only path (lines 574-577), but were not added to the primary plan generation path. The fix is straightforward: add the same four key-value pairs to the result map at lines 441-470, using the `verificationDepth`, `verificationSmartDefault`, `planningSmartDefault`, and `planningPhase` variables that are already resolved earlier in the function (lines 179-184).

The practical impact: when a user runs `/ant-plan` and a fresh plan is generated (the normal first-run case), the depth selection banner will show only the planning depth value without reason annotations and will not show the verification depth at all. The verification depth IS correctly resolved and persisted to ColonyState (so the build/continue path works), but the user never sees it in the plan output.

All other aspects of the phase goal are fully achieved: the flag exists, the resolver works with smart defaults, the build manifest carries the depth, and continue reads it automatically.

---

_Verified: 2026-05-01T14:30:00Z_
_Verifier: Claude (gsd-verifier)_
