---
phase: 86-depth-selection-ui-and-persistence
reviewed: 2026-05-01T12:00:00Z
depth: standard
files_reviewed: 8
files_reviewed_list:
  - cmd/codex_build.go
  - cmd/codex_build_finalize.go
  - cmd/codex_build_finalize_test.go
  - cmd/codex_plan.go
  - cmd/codex_visuals.go
  - cmd/codex_visuals_test.go
  - cmd/review_depth.go
  - cmd/review_depth_test.go
findings:
  critical: 1
  warning: 3
  info: 2
  total: 6
status: issues_found
---

# Phase 86: Code Review Report

**Reviewed:** 2026-05-01T12:00:00Z
**Depth:** standard
**Files Reviewed:** 8
**Status:** issues_found

## Summary

Reviewed 8 files implementing depth selection UI, smart depth defaults with position/risk signals, verification depth persistence in colony state, atomic state commits in build-finalize, and removal of the unbounded filesystem walk fallback. The depth resolution logic is well-tested with thorough coverage of priority ordering, keyword matching, and edge cases. The visual rendering correctly surfaces depth selection banners in the plan output. Three genuine issues were found: a misleading reason annotation in build visuals, a behavioral inconsistency between the legacy 2-level and current 3-level depth resolvers, and a brittle structural test that scans source text.

## Critical Issues

### CR-01: renderBuildVisualWithDispatches and renderBuildPlanOnlyVisual hardcode smartDefault=true, misrepresenting user intent

**File:** `cmd/codex_visuals.go:1163` and `cmd/codex_visuals.go:1217`
**Issue:** Both `renderBuildVisualWithDispatches` and `renderBuildPlanOnlyVisual` call `renderReviewDepthLineWithReason` with `smartDefault=true` hardcoded. This means the reason annotation (e.g., "auto: security risk") is always displayed, even when the user explicitly set the depth via `--verification-depth heavy` or `--light`. The user is misled into believing the depth was auto-detected when it was their explicit choice.

By contrast, the plan visual at `codex_visuals.go:896` correctly reads `verificationSmartDefault` from the result map and only shows the reason when the depth was actually smart-defaulted.

The root cause is that `renderBuildVisualWithDispatches` receives only the resolved `colony.VerificationDepth` value and the colony state, with no way to distinguish whether the depth came from a smart default or an explicit user flag. The `smartDefault=true` parameter is therefore a lie.

**Fix:**
Either pass `smartDefault` through the call chain from `runCodexBuildWithOptions`, or stop showing the reason annotation when the caller cannot determine whether the depth was user-specified:

```go
// Option A: Remove reason from build visuals (safest short-term fix)
b.WriteString(renderReviewDepthLine(reviewDepth, phase.ID, len(state.Plan.Phases)))
```

```go
// Option B: Thread smartDefault through the call chain
func renderBuildVisualWithDispatches(state colony.ColonyState, phase colony.Phase, dispatches []codexBuildDispatch, reviewDepth colony.VerificationDepth, smartDefault bool) string {
    // ...
    b.WriteString(renderReviewDepthLineWithReason(reviewDepth, phase.ID, len(state.Plan.Phases), phase, smartDefault))
    // ...
}
```

## Warnings

### WR-01: resolveReviewDepth (2-level legacy) ignores lightFlag for keyword phases, inconsistent with resolveVerificationDepth (3-level)

**File:** `cmd/review_depth.go:27-42`
**Issue:** `resolveReviewDepth` treats keyword-matched phases as always heavy regardless of `lightFlag`. A phase named "Auth refactor" with `lightFlag=true` returns `ReviewDepthHeavy`. The 3-level `resolveVerificationDepth` (line 64) correctly allows `lightFlag` to override keyword match. This creates an inconsistency between the two resolution functions. While `resolveReviewDepth` appears to be a legacy function with no production callers, it has 8 tests asserting this behavior, making a future alignment change harder.

**Fix:** Update `resolveReviewDepth` to respect `lightFlag`, matching the 3-level behavior:
```go
func resolveReviewDepth(phase colony.Phase, totalPhases int, lightFlag, heavyFlag bool) ReviewDepth {
    if phase.ID == totalPhases {
        return ReviewDepthHeavy
    }
    if heavyFlag {
        return ReviewDepthHeavy
    }
    if phaseHasHeavyKeywords(phase.Name) && !lightFlag {
        return ReviewDepthHeavy
    }
    return ReviewDepthLight
}
```
Then update `TestResolveReviewDepth_KeywordPhaseWithLightFlag` to expect `ReviewDepthLight`.

### WR-02: TestDepthKeysPresentInFreshPlanResultMap scans source text, creating fragile coupling

**File:** `cmd/review_depth_test.go:1036-1060`
**Issue:** This test reads `codex_plan.go` as raw source text and counts occurrences of JSON key strings (e.g., `"verification_depth":`). If a key appears in a comment, string literal, or generated code, the test would produce a false positive. The test is validating a regression fix but does so in a way that is fragile to refactoring (e.g., extracting a helper function could change the count).

**Fix:** Convert to an integration test that calls `runCodexPlanWithOptions` and checks the returned result map directly, or add a clear comment explaining the intentional tradeoff:
```go
// Structural regression guard: verifies depth keys appear in all three
// result-map paths. This intentionally scans source text because it is
// checking that a prior regression (missing keys) does not recur. If a
// refactor moves keys into a shared helper, update the expected count.
```

### WR-03: resolveVerificationDepthSmart validates raw input separately from NormalizeVerificationDepth, creating drift risk

**File:** `cmd/review_depth.go:218-231`
**Issue:** `resolveVerificationDepthSmart` first calls `colony.NormalizeVerificationDepth(depth)` which maps aliases like "full" to "heavy", "minimal" to "light". Then it validates the raw user input against a hardcoded switch statement. If a new alias is added to `NormalizeVerificationDepth` but not to the switch in `resolveVerificationDepthSmart`, the alias would be normalized successfully but then rejected as invalid. The validation is redundant and diverges from the canonical normalization.

**Fix:** Validate against the normalized value instead of the raw input, or remove the redundant validation and rely on `NormalizeVerificationDepth` as the single source of truth:
```go
func resolveVerificationDepthSmart(depth string, phase colony.Phase, totalPhases int) (string, error) {
    normalized := colony.NormalizeVerificationDepth(depth)
    if depth != "" {
        return string(normalized), nil
    }
    return string(resolveSmartVerificationDepth(phase, totalPhases)), nil
}
```

## Info

### IN-01: Duplicate keyword lists between heavyKeywords and securityRiskKeywords

**File:** `cmd/review_depth.go:19-23` and `cmd/review_depth.go:106-110`
**Issue:** `heavyKeywords` (used by `phaseHasHeavyKeywords`) and `securityRiskKeywords` (used by `phaseRiskLevel`) contain overlapping entries. Both include "security", "auth", "secrets", "permissions", "compliance", "audit". `securityRiskKeywords` adds "token", "session", "password" while `heavyKeywords` adds "release", "deploy", "production", "ship", "launch". These parallel lists will drift if one is updated without the other.

**Fix:** Consolidate into a single authoritative keyword list with metadata tags, or derive `heavyKeywords` from `securityRiskKeywords` plus deployment keywords.

### IN-02: phasePositionLevel produces "early" for phase 1 of a 1-phase plan

**File:** `cmd/review_depth.go:121-134`
**Issue:** For a single-phase plan (totalPhases=1, phaseID=1), `phasePositionLevel` returns "final" because of the `phaseID == totalPhases` check on line 122. This is correct. However, for a 2-phase plan where phase 1 passes the 25% threshold check (`phaseID <= 0.25 * totalPhases` i.e. `1 <= 0.5` which is true), it returns "early". This means a 2-phase plan always starts with "early" for phase 1 regardless of whether the keyword/risk signals suggest otherwise. This is by design but could surprise users who expect the first phase of a 2-phase plan to get "standard" depth.

This is informational only -- the behavior is documented and tested. No action needed.

---

_Reviewed: 2026-05-01T12:00:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
