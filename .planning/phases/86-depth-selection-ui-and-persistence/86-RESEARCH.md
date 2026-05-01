# Phase 86: Depth Selection UI and Persistence - Research

**Researched:** 2026-05-01
**Domain:** Go CLI (cobra) console output formatting and JSON build packet persistence
**Confidence:** HIGH

## Summary

Phase 86 wires the smart depth defaults (implemented in Phase 85) into the user-facing `/ant-plan` command flow. The work is purely CLI-level: adding a `--verification-depth` flag to the plan command, displaying a depth selection banner with smart default reasons, and persisting the selected verification depth into the build packet JSON so `/ant-continue` reads it automatically.

The research confirms that all foundational pieces are already in place. The smart depth functions (`resolveSmartPlanningDepth`, `resolveSmartVerificationDepth`, `renderSmartDepthReason`, `renderReviewDepthLineWithReason`) are implemented and tested (44+ subtests passing). The pattern for wiring depth into plan options is established via `resolvePlanningDepthSmart`. The build packet struct (`codexBuildManifest`) and its write path are known. The continue read path already consumes `ReviewDepth` from the plan manifest. What is missing is the plumbing between plan-time depth selection and the build/continue packet.

**Primary recommendation:** Add `VerificationDepth` to `codexPlanOptions`, add `--verification-depth` flag to `planCmd`, emit verification depth in the plan result map, display the depth selection banner in `renderPlanVisual`, add `ReviewDepth` field to `codexBuildManifest`, and ensure the build packet write path stores it for continue to consume.

## User Constraints (from CONTEXT.md)

### Locked Decisions

- **D-01:** `/ant-plan` must display a depth selection banner showing both the planning depth and verification depth smart defaults, along with the reason each was selected. The `renderSmartDepthReason` and `renderReviewDepthLineWithReason` visual helpers from Phase 85 should be wired in for this output.
- **D-02:** The banner should use the existing stage marker style (`── Depth Selection ──`) for visual consistency with the rest of the Aether CLI output.
- **D-03:** Add a `--verification-depth` flag to `/ant-plan` as a direct mirror of the existing `--planning-depth` flag. Both flags accept the standard depth values (light/standard/deep for planning; light/standard/heavy for verification). When provided, the explicit flag overrides the smart default for that depth. When absent, the smart default is used.
- **D-04:** During `/ant-build`, the active verification depth must be shown in the build stage markers (e.g., `── Stage 2: Verification [standard] ──`) and in the build summary output. The `renderReviewDepthLineWithReason` visual helper from Phase 85 should be wired into the stage marker rendering in `cmd/codex_visuals.go`.

### Claude's Discretion

- Exact formatting of the depth selection banner (spacing, label casing, line width)
- Exact placement of the depth indicator in stage markers (prefix vs suffix, bracket style)
- How the depth value flows through the build packet JSON field naming

### Deferred Ideas (OUT OF SCOPE)

None -- discussion stayed within phase scope.

## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| DEPTH-04 | User depth selection at plan time: user sees smart defaults for both planning depth and verification depth, can accept or override either | Plan command flow, `codexPlanOptions` struct, `resolvePlanningDepthSmart` pattern, `renderSmartDepthReason` and `renderReviewDepthLineWithReason` visual helpers -- all ready to wire |
| DEPTH-05 | Depth persistence across continue: verification depth selected at plan time stored in build packet, honored by `/ant-continue` without re-specification | `codexBuildManifest` struct, build packet write path in `buildCodexBuildManifest`, continue read path in `codex_continue_finalize.go` lines 202-204 already reads `ReviewDepth` -- needs `ReviewDepth` field added to manifest and populated during build |

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Depth selection banner rendering | Go CLI (cmd/codex_visuals.go) | -- | Console output formatting is a runtime concern |
| `--verification-depth` flag parsing | Go CLI (cmd/codex_workflow_cmds.go) | -- | Cobra flag registration is runtime |
| Smart default resolution | Go runtime (cmd/review_depth.go) | -- | Pure logic, already implemented |
| Build packet JSON persistence | Go runtime (cmd/codex_build.go) | -- | Manifest construction and JSON serialization |
| Continue packet consumption | Go runtime (cmd/codex_continue_finalize.go) | -- | Already reads ReviewDepth from plan manifest |
| Plan result map enrichment | Go runtime (cmd/codex_plan.go) | -- | Map key additions for downstream rendering |
| Cross-command depth persistence | Go runtime (pkg/colony/colony.go) | -- | ColonyState field for plan-to-build depth propagation |

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| Go | 1.22+ (project Go version) | Runtime language | Project is Go-native |
| cobra | project dependency | CLI framework | All commands use cobra |
| pkg/colony | in-repo | Colony types (VerificationDepth, PlanningDepth, Phase) | Type definitions live here |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| fmt | stdlib | String formatting for console output | All visual rendering |
| strings | stdlib | String manipulation | Trimming, joining, casing |
| encoding/json | stdlib | JSON serialization for build packets | Manifest write/read |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Adding VerificationDepth to ColonyState | Plan manifest file on disk | ColonyState is simpler (already loaded by build) and follows the ColonyDepth pattern. Plan manifest is not written to a standalone file, so ColonyState is the practical choice. |
| Using existing --light/--heavy flags on plan | New --verification-depth flag | Existing flags are on build, not plan. A dedicated plan-time flag is cleaner and matches the --planning-depth precedent. |

**Installation:** No new dependencies required.

## Architecture Patterns

### System Architecture Diagram

```
/aether plan ──> planCmd.RunE()
    │
    ├── Read --planning-depth flag  ──> codexPlanOptions.PlanningDepth
    ├── Read --verification-depth flag (NEW) ──> codexPlanOptions.VerificationDepth
    │
    ├── resolvePlanningDepthSmart(depth, phase, total) ──> planningDepth string
    │       └── uses resolveSmartPlanningDepth() when no flag provided
    │
    ├── resolveVerificationDepthSmart(depth, phase, total) (NEW)
    │       └── uses resolveSmartVerificationDepth() when no flag provided
    │
    ├── Store verificationDepth in state.VerificationDepth ──> saveColonyState()
    │
    ├── Build result map (enriched with verification_depth, smart_default flags, planning_phase)
    │
    └── renderPlanVisual(result)
            └── Depth Selection Banner
                    ├── renderSmartDepthReason(phase, total) for planning reason
                    └── renderReviewDepthLineWithReason() for verification reason

/aether build ──> resolveVerificationDepth()
    │
    ├── Read state.VerificationDepth as depthStr (stored by plan)
    ├── Resolve with build flags + stored depth
    ├── Add ReviewDepth field to codexBuildManifest
    └── Write manifest.json with review_depth

/aether continue ──> loadCodexContinueManifest()
    │
    └── Read ReviewDepth from build manifest (ALREADY WORKS via plan.ReviewDepth)
```

### Recommended Project Structure

No new files needed. All changes are modifications to existing files:

```
cmd/
├── codex_workflow_cmds.go   ← Add --verification-depth flag, read it in planCmd.RunE
├── codex_plan.go            ← Add VerificationDepth to codexPlanOptions, resolve it, emit in result map, store in ColonyState
├── codex_build.go           ← Add ReviewDepth to codexBuildManifest, read state.VerificationDepth, populate it
├── codex_visuals.go         ← Add depth selection banner, wire into renderPlanVisual()
├── review_depth.go          ← Add resolveVerificationDepthSmart() wrapper (mirror of resolvePlanningDepthSmart)
└── review_depth_test.go     ← Tests for new function and banner rendering
pkg/colony/
└── colony.go                ← Add VerificationDepth field to ColonyState struct
```

### Pattern 1: Smart Depth Resolution Wrapper (established pattern)

**What:** `resolvePlanningDepthSmart` wraps `resolvePlanningDepth` with a smart-default fallback when no explicit flag is provided.

**When to use:** The same pattern must be replicated for verification depth.

**Example:**
```go
// Source: [VERIFIED: cmd/codex_plan.go:613-627]
func resolvePlanningDepthSmart(depth string, phase colony.Phase, totalPhases int) (string, error) {
    normalized, err := resolvePlanningDepth(depth)
    if err != nil {
        return "", err
    }
    if depth != "" {
        return normalized, nil
    }
    return string(resolveSmartPlanningDepth(phase, totalPhases)), nil
}
```

**Phase 86 implementation:** Add `resolveVerificationDepthSmart` as a mirror:
```go
func resolveVerificationDepthSmart(depth string, phase colony.Phase, totalPhases int) (string, error) {
    normalized := colony.NormalizeVerificationDepth(depth)
    if depth != "" {
        // Validate known values
        lower := strings.ToLower(strings.TrimSpace(depth))
        switch lower {
        case "light", "standard", "heavy":
            // known canonical values
        case "minimal", "coarse":
            // known aliases
        case "full", "thorough":
            // known aliases
        default:
            return "", fmt.Errorf("invalid verification depth %q: must be light, standard, or heavy", depth)
        }
        return string(normalized), nil
    }
    return string(resolveSmartVerificationDepth(phase, totalPhases)), nil
}
```

### Pattern 2: Stage Marker Rendering (established pattern)

**What:** `renderStageMarker(title)` produces `── Title ──\n` format.

**When to use:** Depth selection banner uses this for the `── Depth Selection ──` header.

**Example:**
```go
// Source: [VERIFIED: cmd/codex_visuals.go:274-280]
func renderStageMarker(title string) string {
    title = strings.TrimSpace(title)
    if title == "" {
        return ""
    }
    return "── " + title + " ──\n"
}
```

### Pattern 3: Plan Result Map Enrichment (established pattern)

**What:** Plan functions return `map[string]interface{}` with keys consumed by `renderPlanVisual`.

**When to use:** New keys `verification_depth`, `verification_smart_default`, and `planning_smart_default` must be added.

**Example:**
```go
// Source: [VERIFIED: cmd/codex_plan.go:424-453]
result := map[string]interface{}{
    "planned":        true,
    "planning_depth": planningDepth,
    // ... existing keys
}
// Add:
// "verification_depth":         verificationDepth,
// "verification_smart_default": verificationDepthWasSmartDefault,
// "planning_smart_default":     planningDepthWasSmartDefault,
// "planning_phase":             planningPhase,
```

### Anti-Patterns to Avoid
- **Don't duplicate the smart depth logic in the plan command:** Use the existing `resolveSmartVerificationDepth` function, just like `resolvePlanningDepthSmart` delegates to `resolveSmartPlanningDepth`.
- **Don't break the existing --light/--heavy flags on build:** Those are independent of plan-time depth selection. The build command should still accept its own flags that override whatever was stored in the packet.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Depth normalization | Custom string switch | `colony.NormalizeVerificationDepth()` | Already handles aliases, casing, trimming, unknown-value defaulting |
| Planning depth validation | New validation logic | Pattern from `resolvePlanningDepth` (cmd/codex_plan.go:598-611) | Established validation with good error messages |
| Stage marker formatting | Manual string concat | `renderStageMarker()` | Consistent with existing CLI visual style |
| Smart depth reason rendering | New formatting | `renderSmartDepthReason()` and `renderReviewDepthLineWithReason()` | Already implemented in Phase 85, produce human-readable reasons |

**Key insight:** Phase 85 already built all the "smart" pieces. This phase is purely plumbing and display work.

## Common Pitfalls

### Pitfall 1: Banner placement after plan generation
**What goes wrong:** If the depth banner is inserted after plan generation, the user sees it too late -- after phases are already generated.
**Why it happens:** `renderPlanVisual` receives the full result map, so it's tempting to append the banner at the end.
**How to avoid:** Insert the depth banner early in `renderPlanVisual`, immediately after the goal line and before the confidence/phase list. This matches the CONTEXT.md guidance that the banner should appear "before plan generation begins" from the user's perspective (it shows what was selected, not what happened after).
**Warning signs:** User sees the depth info after scrolling past a long phase list.

### Pitfall 2: Forgetting to track whether depth was smart-defaulted
**What goes wrong:** `renderReviewDepthLineWithReason` takes a `smartDefault bool` parameter. If this is always true or always false, the reason display is misleading.
**Why it happens:** The resolve function returns only the depth value, not whether it was auto-selected.
**How to avoid:** Track the "was smart default" flag explicitly. The pattern is: if `opts.VerificationDepth == ""` (no flag), then it was smart-defaulted; otherwise it was user-specified. Return this as a boolean alongside the resolved value.
**Warning signs:** Banner always shows "auto:" prefix even when user explicitly passed a flag.

### Pitfall 3: Build manifest not persisting ReviewDepth
**What goes wrong:** Continue reads `ReviewDepth` from the plan manifest but it is empty because build never wrote it.
**Why it happens:** The `codexBuildManifest` struct currently has no `ReviewDepth` field. The build writes the manifest without this field, and continue's `plan.ReviewDepth` is empty string, which defaults to `VerificationDepthLight` via `NormalizeVerificationDepth`.
**How to avoid:** Add `ReviewDepth string \`json:"review_depth,omitempty"\`` to `codexBuildManifest` and populate it in `buildCodexBuildManifest`. The continue read path at `codex_continue_finalize.go:202-204` already handles this field.
**Warning signs:** Continue always uses light review depth regardless of what was selected at plan time.

### Pitfall 4: --verification-depth flag on plan but not on plan-finalize
**What goes wrong:** `aether plan --plan-only` emits the depth, but `aether plan-finalize` does not pass it through, so the finalized plan loses the depth selection.
**Why it happens:** `planFinalizeCmd` reads from a completion file, not from flags.
**How to avoid:** The plan manifest (`codexPlanManifest`) should include the `VerificationDepth` field so that plan-finalize can persist it. The plan-finalize path already stores the manifest.
**Warning signs:** After `aether plan --plan-only` followed by `aether plan-finalize`, the verification depth is lost.

### Pitfall 5: Build re-resolving depth without reading plan-time selection
**What goes wrong:** Build calls `resolveVerificationDepth(phase, totalPhases, false, false, "")` with empty depthStr, ignoring the user's explicit `--verification-depth` selection from plan time.
**Why it happens:** The build command loads ColonyState fresh and does not read any plan-time depth preference.
**How to avoid:** Store the resolved verification depth in `ColonyState.VerificationDepth` during plan (parallel to `ColonyDepth`). Build reads `state.VerificationDepth` and passes it as `depthStr` to `resolveVerificationDepth`, so the plan-time selection propagates.
**Warning signs:** User runs `aether plan --verification-depth heavy` then `aether build 1`, but build uses standard depth.

## Code Examples

Verified patterns from codebase:

### Existing plan depth resolution flow
```go
// Source: [VERIFIED: cmd/codex_plan.go:162-176]
granularity, planDepth, err := resolvePlanGranularityDepth(state.PlanGranularity, opts.Depth)
// ...
planningDepth, err := resolvePlanningDepthSmart(opts.PlanningDepth, planningPhase, len(state.Plan.Phases))
```

### Existing build verification depth resolution
```go
// Source: [VERIFIED: cmd/codex_build.go:148-151]
reviewDepth := resolveVerificationDepth(phase, len(state.Plan.Phases), false, false, "")
dispatches := plannedBuildDispatchesForSelection(phase, depth, selectedTaskIDs, reviewDepth)
```

### Existing continue read of ReviewDepth
```go
// Source: [VERIFIED: cmd/codex_continue_finalize.go:202-206]
finalizeReviewDepth := colony.VerificationDepthLight
if plan.ReviewDepth != "" {
    finalizeReviewDepth = colony.NormalizeVerificationDepth(plan.ReviewDepth)
}
review := externalContinueReviewReport(phase.ID, workerFlow, now, skipMissing, finalizeReviewDepth)
```

### Existing plan visual rendering (current -- no depth banner)
```go
// Source: [VERIFIED: cmd/codex_visuals.go:849-873]
func renderPlanVisual(result map[string]interface{}) string {
    var b strings.Builder
    b.WriteString(renderBanner(commandEmoji("plan"), "Plan"))
    b.WriteString(visualDivider)
    // ... goal, granularity, planning_depth (only if non-standard), confidence ...
    // NO verification depth currently
}
```

### Proposed depth banner insertion point
```go
// Insert after planning_depth line (line ~873) and before confidence
// This is the natural location: after basic config, before results

// Depth Selection Banner
planningPhase := result["planning_phase"].(colony.Phase)
b.WriteString(renderStageMarker("Depth Selection"))
b.WriteString(fmt.Sprintf("Planning depth: %s (%s)\n", planningDepth, renderSmartDepthReason(planningPhase, totalPhases)))
b.WriteString(renderReviewDepthLineWithReason(verificationDepth, phase.ID, totalPhases, planningPhase, verificationSmartDefault))
if planningSmartDefault || verificationSmartDefault {
    b.WriteString("Override: --planning-depth <light|standard|deep> --verification-depth <light|standard|heavy>\n")
}
```

## State of the Art

N/A -- This phase uses only established project patterns. No external dependencies or framework evolution involved.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | `codexPlanManifest` is the correct place to store verification depth for plan-finalize path | Architecture Patterns | If plan-finalize has a different persistence path, the depth could be lost after plan-only + finalize flow |
| A2 | The continue read path at `codex_continue_finalize.go:202-204` is the only place continue reads ReviewDepth | Build Packet Persistence | If other continue code paths also read depth, they need updating too |
| A3 | ColonyState.VerificationDepth follows the same persistence pattern as ColonyDepth | ColonyState Storage | If ColonyDepth has special handling we missed, VerificationDepth may need the same |

## Open Questions (RESOLVED)

1. RESOLVED: **Should the depth banner show per-phase depths or the "next buildable phase" depth?**
   - Resolution: Show depth for the first buildable phase only (matches the existing `planningDepth` display behavior). Per-phase depth is a future enhancement.

2. RESOLVED: **Should verification depth also be persisted in ColonyState for non-build-plan flows?**
   - Resolution: Yes. ColonyState.VerificationDepth stores the resolved depth during plan, following the ColonyDepth pattern. Build reads it as `depthStr` for `resolveVerificationDepth`. This ensures the user's explicit `--verification-depth` flag propagates from plan to build without re-resolution.

## Environment Availability

Step 2.6: SKIPPED (no external dependencies -- all changes are code/config-only Go modifications within the existing repo).

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go testing (stdlib) |
| Config file | none -- Go test convention |
| Quick run command | `go test ./cmd/ -run "TestDepth" -count=1` |
| Full suite command | `go test ./cmd/ -count=1` |

### Phase Requirements -> Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| DEPTH-04 | `--verification-depth` flag on plan accepts light/standard/heavy | unit | `go test ./cmd/ -run "TestResolveVerificationDepthSmart" -count=1` | No -- Wave 0 |
| DEPTH-04 | Smart default used when flag absent | unit | `go test ./cmd/ -run "TestResolveVerificationDepthSmart/empty_flag" -count=1` | No -- Wave 0 |
| DEPTH-04 | Depth selection banner renders correctly | unit | `go test ./cmd/ -run "TestRenderDepthSelectionBanner" -count=1` | No -- Wave 0 |
| DEPTH-04 | Plan result map includes verification_depth | unit | `go test ./cmd/ -run "TestPlanResultIncludesVerificationDepth" -count=1` | No -- Wave 0 |
| DEPTH-05 | Build manifest includes ReviewDepth field | unit | `go test ./cmd/ -run "TestBuildManifestReviewDepth" -count=1` | No -- Wave 0 |
| DEPTH-05 | Continue reads ReviewDepth from manifest | integration | `go test ./cmd/ -run "TestContinueReadsStoredReviewDepth" -count=1` | Partial -- existing test at codex_continue_test.go:277 |

### Sampling Rate
- **Per task commit:** `go test ./cmd/ -run "TestDepth|TestSmartDepth|TestReviewDepth" -count=1`
- **Per wave merge:** `go test ./cmd/ -count=1`
- **Phase gate:** Full suite green before `/gsd-verify-work`

### Wave 0 Gaps
- [ ] `cmd/review_depth_test.go` -- tests for `resolveVerificationDepthSmart()` function
- [ ] `cmd/review_depth_test.go` -- tests for `renderDepthSelectionBanner()` or equivalent visual rendering
- [ ] `cmd/codex_plan_test.go` -- integration test that plan result map includes `verification_depth` key
- [ ] `cmd/codex_build_test.go` -- test that build manifest includes `ReviewDepth` when populated

## Security Domain

Not applicable -- this phase adds CLI flags and console output formatting only. No authentication, session management, access control, input validation beyond enum values, or cryptography changes.

## Sources

### Primary (HIGH confidence)
- [VERIFIED: cmd/review_depth.go] -- Smart depth functions, verification depth resolver, all Phase 85 logic
- [VERIFIED: cmd/codex_plan.go] -- Plan command flow, codexPlanOptions, resolvePlanningDepthSmart pattern
- [VERIFIED: cmd/codex_build.go] -- Build manifest struct, buildCodexBuildManifest function, review depth resolution
- [VERIFIED: cmd/codex_visuals.go] -- renderStageMarker, renderBanner, renderPlanVisual, renderReviewDepthLineWithReason
- [VERIFIED: cmd/codex_workflow_cmds.go] -- Plan command cobra registration, flag definitions
- [VERIFIED: cmd/codex_continue_finalize.go] -- Continue reads ReviewDepth from plan manifest
- [VERIFIED: pkg/colony/colony.go] -- VerificationDepth type, NormalizeVerificationDepth, PlanningDepth type, ColonyState struct
- [VERIFIED: cmd/review_depth_test.go] -- Existing 44+ subtests for smart depth functions

### Secondary (MEDIUM confidence)
- [VERIFIED: cmd/codex_plan_test.go] -- Existing plan command integration tests showing result map structure

### Tertiary (LOW confidence)
- None -- all findings verified against codebase.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - all existing project code, no new dependencies
- Architecture: HIGH - every integration point verified against source
- Pitfalls: HIGH - all pitfalls derived from verified code paths

**Research date:** 2026-05-01
**Valid until:** 30 days (stable internal architecture, no external dependencies)
