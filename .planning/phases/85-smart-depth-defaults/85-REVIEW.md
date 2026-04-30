---
phase: 85-smart-depth-defaults
reviewed: 2026-04-30T00:30:00Z
depth: standard
files_reviewed: 4
files_reviewed_list:
  - cmd/codex_plan.go
  - cmd/codex_visuals.go
  - cmd/review_depth.go
  - cmd/review_depth_test.go
findings:
  critical: 1
  warning: 3
  info: 2
  total: 6
status: issues_found
---

# Phase 85: Code Review Report

**Reviewed:** 2026-04-30T00:30:00Z
**Depth:** standard
**Files Reviewed:** 4
**Status:** issues_found

## Summary

Reviewed the smart depth defaults implementation across four files: the core depth resolution logic (`review_depth.go`), its test suite (`review_depth_test.go`), the planning command integration (`codex_plan.go`), and the visual rendering layer (`codex_visuals.go`). The implementation introduces smart auto-detection of planning and verification depth based on phase position and risk signals from phase text. The core logic is sound, but one critical bug was found where `resolveReviewDepth` ignores its `lightFlag` parameter, and several warnings around dead code and missing edge-case coverage.

## Critical Issues

### CR-01: `resolveReviewDepth` silently ignores the `lightFlag` parameter

**File:** `cmd/review_depth.go:26-41`
**Issue:** The `resolveReviewDepth` function accepts a `lightFlag` parameter but never reads it. The function's default return path (line 40) always returns `ReviewDepthLight` regardless of whether `lightFlag` is true or false. This means any caller relying on `lightFlag` to explicitly opt into light mode for a phase that would otherwise match a keyword or get the default will get misleading behavior -- specifically, the function returns `ReviewDepthLight` for all non-final, non-heavy, non-keyword phases, making the `lightFlag` indistinguishable from the default.

While this function currently has no production callers (it is only called from its own tests), the function signature promises behavior it does not deliver. If a future caller passes `lightFlag=true` expecting it to force light mode (analogous to how `heavyFlag=true` forces heavy), it would silently get the same result as the default path.

```go
// Current code -- lightFlag is accepted but never checked:
func resolveReviewDepth(phase colony.Phase, totalPhases int, lightFlag, heavyFlag bool) ReviewDepth {
    if phase.ID == totalPhases {
        return ReviewDepthHeavy
    }
    if heavyFlag {
        return ReviewDepthHeavy
    }
    if phaseHasHeavyKeywords(phase.Name) {
        return ReviewDepthHeavy
    }
    // lightFlag is never checked here -- always falls through to ReviewDepthLight
    return ReviewDepthLight
}
```

**Fix:** Either remove the `lightFlag` parameter (since `ReviewDepthLight` is already the default and the 3-level `resolveVerificationDepth` is the active code path), or add explicit `lightFlag` handling to override keyword matches:

```go
func resolveReviewDepth(phase colony.Phase, totalPhases int, lightFlag, heavyFlag bool) ReviewDepth {
    if phase.ID == totalPhases {
        return ReviewDepthHeavy
    }
    if heavyFlag {
        return ReviewDepthHeavy
    }
    if lightFlag {
        return ReviewDepthLight
    }
    if phaseHasHeavyKeywords(phase.Name) {
        return ReviewDepthHeavy
    }
    return ReviewDepthLight
}
```

## Warnings

### WR-01: `resolveReviewDepth` and `ReviewDepth` type are dead production code

**File:** `cmd/review_depth.go:9-15, 26-41`
**Issue:** The `ReviewDepth` type (light/heavy 2-level) and `resolveReviewDepth` function are only called from their own test file. The production codebase has migrated to the 3-level `colony.VerificationDepth` system via `resolveVerificationDepth`. Keeping this dead code alongside the active system creates confusion about which depth model is canonical and increases maintenance burden.

**Fix:** Mark `resolveReviewDepth` and the `ReviewDepth` type as deprecated with a comment, or remove them entirely and update the tests to only cover `resolveVerificationDepth`. The keyword detection logic (`phaseHasHeavyKeywords`, `heavyKeywords`) is still used by `resolveVerificationDepth`, so those should be preserved.

### WR-02: `phasePositionLevel` produces no "early" result for 2-phase plans

**File:** `cmd/review_depth.go:119-132`
**Issue:** When `totalPhases=2`, the 25% threshold is `0.5`. Phase 1 (`phaseID=1`) does not satisfy `1.0 <= 0.5`, so it falls through to "intermediate" instead of "early". This means a 2-phase plan will classify phase 1 as "intermediate" and phase 2 as "final" -- no phase gets the "early" classification, so `resolveSmartPlanningDepth` returns `PlanningDepthStandard` instead of `PlanningDepthLight` for the first phase.

```go
// totalPhases=2: threshold25 = 0.5
// phaseID=1: 1.0 <= 0.5 is false -> "intermediate" (should arguably be "early")
```

**Fix:** Use `<=` with integer comparison or floor the threshold:

```go
func phasePositionLevel(phaseID, totalPhases int) string {
    if phaseID == totalPhases || totalPhases <= 1 {
        return "final"
    }
    // Use integer-based boundaries to ensure phase 1 is always "early"
    earlyBound := max(1, int(float64(totalPhases)*0.25+0.5))
    lateBound := int(float64(totalPhases)*0.75 + 0.5)
    if phaseID <= earlyBound {
        return "early"
    }
    if phaseID >= lateBound && phaseID != totalPhases {
        return "late"
    }
    return "intermediate"
}
```

### WR-03: `runCodexPlanWithOptions` dereferences `*state.Goal` without nil check

**File:** `cmd/codex_plan.go:198, 278, 309, 310, 315, 324, 428`
**Issue:** `runCodexPlanWithOptions` dereferences `*state.Goal` at 7 locations without first checking whether `state.Goal` is nil. The sibling function `runCodexPlanPlanOnly` correctly guards against this at line 463 with `if state.Goal == nil`. While `loadActiveColonyState()` likely guarantees a non-nil Goal, an inconsistency between these two functions means a corrupted or manually-edited state file could cause a panic.

**Fix:** Add a nil check for `state.Goal` early in `runCodexPlanWithOptions`, matching the pattern in `runCodexPlanPlanOnly`:

```go
func runCodexPlanWithOptions(root string, opts codexPlanOptions) (map[string]interface{}, error) {
    if store == nil {
        return nil, fmt.Errorf("no store initialized")
    }
    state, err := loadActiveColonyState()
    if err != nil {
        return nil, fmt.Errorf("%s", colonyStateLoadMessage(err))
    }
    if state.Goal == nil || strings.TrimSpace(*state.Goal) == "" {
        return nil, fmt.Errorf("No active colony goal. Run `aether init \"goal\"` first.")
    }
    // ... rest of function
}
```

## Info

### IN-01: `heavyKeywords` and `securityRiskKeywords` are separate lists with significant overlap

**File:** `cmd/review_depth.go:18-22, 105-108`
**Issue:** `heavyKeywords` (used by `phaseHasHeavyKeywords`) and `securityRiskKeywords` (used by `phaseRiskLevel`) overlap on "security", "auth", "crypto", "secrets", "permissions", "compliance", "audit" -- 7 of 12 entries. Both lists serve similar purposes but through different resolution paths (`resolveReviewDepth` vs `resolveVerificationDepth`). Maintaining two separate keyword lists increases the risk of them drifting apart.

**Fix:** Consider extracting the shared security keywords into a single authoritative list and composing the two keyword sets from it:

```go
var baseSecurityKeywords = []string{
    "security", "auth", "crypto", "secrets", "permissions",
    "compliance", "audit",
}

var heavyKeywords = append(baseSecurityKeywords,
    "release", "deploy", "production", "ship", "launch",
)

var securityRiskKeywords = append(baseSecurityKeywords,
    "token", "session", "password",
)
```

### IN-02: `resolveReviewDepthFlag` function is unused in production code

**File:** `cmd/review_depth.go:92-100`
**Issue:** `resolveVerificationDepthFlag` is only called from its own test (`TestResolveVerificationDepthFlag_BoolPriority`). It is not called by any production function. It appears to be a utility that was written for potential use but never wired into the call chain.

**Fix:** If this function is intended for future use, add a comment documenting its purpose. If it was superseded by the inline flag priority logic in `resolveVerificationDepth`, consider removing it along with its test.

---

_Reviewed: 2026-04-30T00:30:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
