---
phase: 93-gate-classification-infrastructure
reviewed: 2026-05-03T15:30:00Z
depth: standard
files_reviewed: 2
files_reviewed_list:
  - cmd/gate.go
  - cmd/gate_test.go
findings:
  critical: 0
  warning: 3
  info: 2
  total: 5
status: issues_found
---

# Phase 93: Code Review Report

**Reviewed:** 2026-05-03T15:30:00Z
**Depth:** standard
**Files Reviewed:** 2
**Status:** issues_found

## Summary

Reviewed gate classification infrastructure adding `GateClassificationTier` (hard_block/soft_block/advisory), a 13-entry classification map, lookup functions, `QueenAnnotation` audit trail struct, and a `gate-classify` CLI command. The core design is sound: deterministic code-level classification, fail-open for unclassified gates, and backward-compatible JSON serialization all work correctly. Tests pass (10 new test functions).

However, there are coverage gaps in the test suite for soft_block and advisory gates, the classification map is declared as a mutable `var` despite documentation claiming it is read-only, and a few internal gate names (phase_buildable, phase_built, all_tasks_completed) lack classification entries without any documented rationale for their exclusion.

## Warnings

### WR-01: `gateClassifications` is mutable despite documented immutability guarantee

**File:** `cmd/gate.go:587`
**Issue:** The `gateClassifications` map is declared as `var` (mutable), but the comment on line 585-586 states "This is a read-only constant -- no configuration can change these values." A `var` map can be mutated by any code in the `cmd` package via `gateClassifications["gatekeeper"] = gateClassificationEntry{advisory, "hacked"}`, which would silently change classification tiers. This is a defense-in-depth concern -- nothing currently mutates it, but the access control does not match the documented guarantee.

**Fix:**
```go
// Option A: Make the map unmodifiable by not exporting it and using
// a function that returns a copy:
func allGateClassifications() map[string]gateClassificationEntry {
    cp := make(map[string]gateClassificationEntry, len(gateClassifications))
    for k, v := range gateClassifications {
        cp[k] = v
    }
    return cp
}

// Option B: Add a sync.Once initializer and a package-internal function
// that freezes the map after init() completes.
```
At minimum, add a comment acknowledging the mutability tradeoff:
```go
// NOTE: Declared as var (not const) because Go maps cannot be const.
// Package-internal code must not mutate this map at runtime.
```

### WR-02: `TestIsHardBlockGate_SoftGates` only tests 3 of 6 soft_block gates and skips all advisory gates

**File:** `cmd/gate_test.go:952-959`
**Issue:** The test covers only `auditor`, `complexity`, and `tdd_evidence` as soft_block gates. The classification map defines 6 soft_block gates (`auditor`, `complexity`, `tdd_evidence`, `anti_pattern`, `verification_loop`, `spawn_gate`) and 2 advisory gates (`medic`, `runtime`). The uncovered soft_block gates (`anti_pattern`, `verification_loop`, `spawn_gate`) and all advisory gates are not tested for their non-hard-block behavior. If a gate were accidentally reclassified to `hardBlock`, this test would not catch it.

**Fix:**
```go
func TestIsHardBlockGate_SoftGates(t *testing.T) {
    softGates := []string{"auditor", "complexity", "tdd_evidence", "anti_pattern", "verification_loop", "spawn_gate"}
    for _, name := range softGates {
        if isHardBlockGate(name) {
            t.Errorf("expected isHardBlockGate(%q) to be false (soft_block)", name)
        }
    }
}

func TestIsHardBlockGate_AdvisoryGates(t *testing.T) {
    advisoryGates := []string{"medic", "runtime"}
    for _, name := range advisoryGates {
        if isHardBlockGate(name) {
            t.Errorf("expected isHardBlockGate(%q) to be false (advisory)", name)
        }
    }
}
```

### WR-03: Internal gate names (`phase_buildable`, `phase_built`, `all_tasks_completed`) are unclassified with no documented rationale

**File:** `cmd/gate.go:334-496` (gate functions), `cmd/gate.go:587-604` (classification map)
**Issue:** Three gate check functions produce named gate results -- `phase_buildable` (line 419), `phase_built` (line 461), and `all_tasks_completed` (line 338) -- but none of these appear in `gateClassifications`. The `gateClassify()` function returns `("", "")` for them, which means `isHardBlockGate()` returns `false` (fail-open). The comment on line 608 says "Unknown gates (like continue-flow structural gates) are intentionally unclassified," but `phase_buildable` and `all_tasks_completed` are not structural flow gates -- they are preconditions that directly control phase advancement. An unclassified `phase_buildable` failure would be silently treated as non-blocking if any caller uses `isHardBlockGate()` to decide whether to halt.

**Fix:** Either classify these gates explicitly (likely as `hardBlock` since phase-not-found or already-completed should halt), or add them to the documented exclusion list with rationale:
```go
var gateClassifications = map[string]gateClassificationEntry{
    // ... existing entries ...
    "phase_buildable":    {hardBlock, "Phase must exist and be in a buildable state"},
    "phase_built":        {hardBlock, "Phase must be built before continuing"},
    "all_tasks_completed": {hardBlock, "All phase tasks must be completed to advance"},
}
```

## Info

### IN-01: Table sort order is alphabetical by tier string, not semantic tier priority

**File:** `cmd/gate.go:846-851`
**Issue:** The `renderGateClassifyTable()` function sorts by `Tier < Tier` using string comparison. Since `advisory` < `hard_block` < `soft_block` alphabetically, advisory gates appear first, then hard_block, then soft_block. This is not the expected priority ordering (hard_block should come first). The JSON output uses `outputOK(gateClassifications)` which produces unsorted map output, so the table is the only human-readable view.

**Fix:**
```go
tierOrder := map[GateClassificationTier]int{
    hardBlock: 0,
    softBlock: 1,
    advisory:  2,
}
sort.Slice(entries, func(i, j int) bool {
    if entries[i].Tier != entries[j].Tier {
        return tierOrder[entries[i].Tier] < tierOrder[entries[j].Tier]
    }
    return entries[i].name < entries[j].name
})
```

### IN-02: `QueenAnnotation` struct is defined but never populated by production code

**File:** `cmd/gate.go:41-46, 58`
**Issue:** `QueenAnnotation` is added to `GateCheckResult` as an optional field with `omitempty`, and a JSON roundtrip test confirms it serializes correctly. However, no production code path actually creates or sets a `QueenAnnotation` -- it is only used in the test. This is likely intentional infrastructure for a future phase, but it currently adds dead code to the `GateCheckResult` struct that downstream consumers must handle.

**Fix:** This is acceptable if the annotation is planned for near-term use. If not, consider deferring the struct addition until the phase that actually populates it, to avoid carrying an unused field in persisted gate-results JSON files.

---

_Reviewed: 2026-05-03T15:30:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
