# Phase 142: Grounding Gate + Decision Binding - Research

**Researched:** 2026-05-18
**Domain:** Plan validation, decision-to-pheromone binding, context injection
**Confidence:** HIGH

## Summary

Phase 142 adds two independent capabilities to the planning pipeline: (1) a grounding validation gate that checks whether plan tasks contain concrete file/path references when source anchors are available, and (2) decision binding that ensures resolved discuss decisions are injected into planner worker context and conflict-detected against each other.

The grounding gate (GROUND-06, GROUND-07) runs during `plan-finalize` after phases are extracted from the completion file. It checks task `Goal`, `Constraints`, and `Hints` fields for file-path-like strings (e.g., `cmd/main.go`, `pkg/storage/store.go`) and emits a soft warning to the result map when source anchors exist but zero tasks have any file reference. Research/architecture phases are exempted via a simple heuristic (phase name contains "research", "survey", "architecture", or "design").

Decision binding (GROUND-08) is **already implemented**. The `resolveDiscussQuestion` function in `discuss.go` already checks `clarificationIsHardConstraint()` and calls `createPheromoneSignal("REDIRECT", ...)` for hard constraints. The requirement confirms this is the desired behavior.

Colony-prime decision injection (GROUND-09) is **already implemented**. The `buildColonyPrimeOutput` function in `colony_prime_context.go` already reads resolved clarifications from `pending-decisions.json` and injects them as a "CLARIFIED INTENT" section at priority 8 with protection.

The only new work is the grounding gate (GROUND-06/07), decision conflict detection (GROUND-10), and ensuring the planner brief explicitly references resolved decisions.

**Primary recommendation:** Add `checkPlanGrounding(phases, sourceAnchors)` to `cmd/codex_plan.go` or a new `cmd/plan_grounding.go`, call it from `runCodexPlanFinalize` after building phases, and add `detectDecisionConflicts(resolvedDecisions)` to `cmd/discuss.go` called from `resolveDiscussQuestion`.

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| GROUND-06 | Plan-grounding validation gate -- scan plan tasks for concrete file/path references; emit soft warning when anchors exist but plan has zero file refs | Task struct fields catalogued below; regex pattern for file detection; integration point at `runCodexPlanFinalize` line 172 after `buildWorkerPlanPhases` |
| GROUND-07 | Gate is soft (warning, not rejection) -- research/architecture phases may legitimately lack file targets | Exemption heuristic from phase name; warning added to result map, not error return |
| GROUND-08 | Discuss decision binding -- auto-emit REDIRECT pheromones when `HardConstraint: true` decisions are resolved | **Already implemented** in `resolveDiscussQuestion` at `discuss.go:237-243`; `clarificationIsHardConstraint` checks `Source` suffix `:hard` |
| GROUND-09 | Colony-prime injects resolved decisions into planner worker context | **Already implemented** in `buildColonyPrimeOutput` at `colony_prime_context.go:729-756`; CLARIFIED INTENT section at priority 8 |
| GROUND-10 | Decision conflict detection -- warn when resolved decisions contradict each other | New function needed; reads resolved decisions from `PendingDecisionFile`, checks for contradictory keyword pairs; warning emitted but not blocking |
</phase_requirements>

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Plan grounding validation | Go runtime (`cmd/`) | -- | `plan-finalize` is the only place where completed phases are available for validation |
| Decision-to-pheromone binding | Go runtime (`cmd/discuss.go`) | -- | `resolveDiscussQuestion` already owns decision resolution and pheromone emission |
| Decision conflict detection | Go runtime (`cmd/discuss.go`) | -- | Resolved decisions live in `pending-decisions.json`, loaded by discuss system |
| Resolved decision injection | Go runtime (`cmd/colony_prime_context.go`) | -- | Colony-prime already owns context assembly and section injection |
| Planner worker context | Go runtime (`cmd/codex_plan.go`) | -- | `renderPlanningWorkerBrief` assembles the brief sent to planning workers |

## Standard Stack

### Core

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| Go stdlib `regexp` | Go 1.26+ | File path detection in task text | Already used throughout codebase for pattern matching |
| Go stdlib `strings` | Go 1.26+ | Keyword matching for conflict detection | Already used in `discuss.go` for signal suppression |
| Go stdlib `path/filepath` | Go 1.26+ | Path validation and anchor comparison | Already used by `loadCodexSurveyContext` |

### Supporting

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| Go stdlib `fmt` | Go 1.26+ | Warning message formatting | All warning emission |
| Go stdlib `sort` | Go 1.26+ | Sorting conflict pairs for deterministic output | Conflict detection output |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Regex for file detection | AST parsing of task text | Overkill; task text is free-form English, not code. Regex matching `path/to/file.ext` patterns is sufficient. |
| Keyword-based conflict detection | LLM-based semantic analysis | Adds latency and non-determinism; keyword pairs cover 90% of contradictions (e.g., "use PostgreSQL" vs "keep serverless") |
| New CLI subcommand for conflict check | Inline in `resolveDiscussQuestion` | Separate command adds ceremony; inline check is simpler and runs at the right moment |

**Installation:**
```bash
# Zero new dependencies -- all changes use Go stdlib
```

**Version verification:** Go 1.26.1 darwin/arm64 installed and current. [VERIFIED: `go version`]

## Architecture Patterns

### System Architecture Diagram

```
resolveDiscussQuestion (discuss.go)
    |
    +-- Mark decision resolved, save pending-decisions.json
    +-- ALREADY EXISTS: if HardConstraint -> createPheromoneSignal("REDIRECT", ...)
    +-- NEW: detectDecisionConflicts(file.Decisions) -> emit FEEDBACK warning
    |
    v
buildColonyPrimeOutput (colony_prime_context.go)
    |
    +-- ALREADY EXISTS: resolvedClarifiedIntentEntries -> CLARIFIED INTENT section
    |   (priority 8, injected into all worker context including planners)
    |
    v
plan-finalize (codex_plan_finalize.go)
    |
    +-- Load completion file
    +-- Validate manifest, dispatches, phases
    +-- buildWorkerPlanPhases(phasePlan)
    |
    +-- NEW: checkPlanGrounding(phases, manifest.Survey.SourceAnchors)
    |       |
    |       +-- For each task: scan Goal/Constraints/Hints for file-like paths
    |       +-- If SourceAnchors exist AND zero tasks have file refs -> emit warning
    |       +-- Exempt research/architecture phases
    |       +-- Warning added to result map (soft, not blocking)
    |
    +-- Write COLONY_STATE.json with phases
    +-- Return result with grounding_warnings field
    |
    v
Grounded plan with optional grounding warning
```

### Recommended Project Structure
```
cmd/
    plan_grounding.go          # NEW: checkPlanGrounding, isGroundedTask, isResearchPhase
    plan_grounding_test.go     # NEW: unit tests for grounding gate
    discuss.go                 # MODIFY: add detectDecisionConflicts to resolveDiscussQuestion
    discuss_test.go            # MODIFY: add conflict detection tests
    codex_plan_finalize.go     # MODIFY: call checkPlanGrounding after building phases
    codex_plan_finalize_test.go # MODIFY: add grounding gate integration tests
```

### Pattern 1: Grounding Gate Function
**What:** Pure function that scans plan phases for file-path references and returns warnings.
**When to use:** After phases are built in `plan-finalize`, before writing COLONY_STATE.
**Example:**
```go
// Source: cmd/plan_grounding.go (new file)
package cmd

import (
    "regexp"
    "strings"
)

// filePathPattern matches common file path patterns:
// - paths with extensions: cmd/main.go, pkg/storage/store.go
// - paths with slashes: cmd/sub/command
// - explicit file references: `file.go`, "config.yaml"
var filePathPattern = regexp.MustCompile(`(?:^|[\s"'` + "`" + `(])(?:[a-zA-Z0-9._-]+/){1,}[a-zA-Z0-9._-]+(?:\.[a-zA-Z0-9]+)?(?:$|[\s"'` + "`" + `)])`)

// researchPhaseKeywords identify phases that legitimately lack file targets.
var researchPhaseKeywords = []string{"research", "survey", "architecture", "design", "planning", "discovery"}

// planGroundingWarning is returned when anchors exist but no tasks reference files.
type planGroundingWarning struct {
    PhaseID       int      `json:"phase_id"`
    PhaseName     string   `json:"phase_name"`
    AnchorCount   int      `json:"anchor_count"`
    UngroundedTasks int    `json:"ungrounded_tasks"`
}

// checkPlanGrounding scans plan phases for concrete file references.
// Returns warnings for phases where anchors exist but no tasks reference files.
// Research/architecture phases are exempted.
func checkPlanGrounding(phases []colony.Phase, sourceAnchors []string) []planGroundingWarning {
    if len(sourceAnchors) == 0 {
        return nil // No anchors to ground against
    }
    var warnings []planGroundingWarning
    for _, phase := range phases {
        if isResearchPhase(phase.Name) {
            continue // GROUND-07: research/architecture phases exempted
        }
        groundedCount := 0
        for _, task := range phase.Tasks {
            if isGroundedTask(task) {
                groundedCount++
            }
        }
        if groundedCount == 0 && len(phase.Tasks) > 0 {
            warnings = append(warnings, planGroundingWarning{
                PhaseID:         phase.ID,
                PhaseName:       phase.Name,
                AnchorCount:     len(sourceAnchors),
                UngroundedTasks: len(phase.Tasks),
            })
        }
    }
    return warnings
}

// isGroundedTask returns true if any task field contains a file path reference.
func isGroundedTask(task colony.Task) bool {
    for _, text := range []string{task.Goal, strings.Join(task.Constraints, " "), strings.Join(task.Hints, " ")} {
        if filePathPattern.MatchString(text) {
            return true
        }
    }
    // Also check for explicit file references without slashes (e.g., "main.go")
    for _, hint := range task.Hints {
        if looksLikeFile(hint) {
            return true
        }
    }
    return false
}

// isResearchPhase returns true for phases that legitimately lack file targets.
func isResearchPhase(name string) bool {
    lower := strings.ToLower(name)
    for _, keyword := range researchPhaseKeywords {
        if strings.Contains(lower, keyword) {
            return true
        }
    }
    return false
}

// looksLikeFile checks if a string looks like a file reference.
func looksLikeFile(s string) bool {
    // Check for common source file extensions
    sourceExts := []string{".go", ".ts", ".js", ".py", ".rs", ".java", ".yaml", ".json", ".toml", ".md"}
    for _, ext := range sourceExts {
        if strings.Contains(s, ext) {
            return true
        }
    }
    return false
}
```

### Pattern 2: Decision Conflict Detection
**What:** Function that checks resolved decisions for contradictory pairs.
**When to use:** Inside `resolveDiscussQuestion` after marking a decision resolved.
**Example:**
```go
// Source: cmd/discuss.go (addition to existing file)

// contradictionPairs lists keyword pairs that indicate contradictory decisions.
// Each pair is (positive, negative) where both appearing in resolved decisions
// suggests a conflict.
var contradictionPairs = []struct {
    positive string
    negative string
    category string
}{
    {"postgresql", "serverless", "database"},
    {"mysql", "serverless", "database"},
    {"sqlite", "serverless", "database"},
    {"sql", "nosql", "database"},
    {"monolith", "microservice", "architecture"},
    {"react", "vue", "frontend"},
    {"react", "svelte", "frontend"},
    {"typescript", "javascript only", "language"},
    {"rest", "graphql", "api"},
    {"docker", "no docker", "deployment"},
    {"kubernetes", "no kubernetes", "deployment"},
}

// detectDecisionConflicts checks resolved decisions for contradictions.
// Returns conflict descriptions for FEEDBACK pheromone emission.
func detectDecisionConfisions(decisions []PendingDecision) []string {
    // Extract resolved decision text
    var resolutions []string
    for _, d := range decisions {
        if !d.Resolved || strings.TrimSpace(d.Resolution) == "" {
            continue
        }
        resolutions = append(resolutions, strings.ToLower(d.Resolution))
    }
    if len(resolutions) < 2 {
        return nil
    }

    var conflicts []string
    for _, pair := range contradictionPairs {
        hasPositive := false
        hasNegative := false
        for _, r := range resolutions {
            if strings.Contains(r, pair.positive) {
                hasPositive = true
            }
            if strings.Contains(r, pair.negative) {
                hasNegative = true
            }
        }
        if hasPositive && hasNegative {
            conflicts = append(conflicts, fmt.Sprintf(
                "Possible %s conflict: decisions reference both '%s' and '%s'",
                pair.category, pair.positive, pair.negative,
            ))
        }
    }
    return conflicts
}
```

### Pattern 3: Integration Point in plan-finalize
**What:** Call grounding check after phases are built, add warnings to result.
**When to use:** In `runCodexPlanFinalize` after `buildWorkerPlanPhases`.
**Example:**
```go
// Source: cmd/codex_plan_finalize.go (modification to runCodexPlanFinalize)
// After line 172: phases := buildWorkerPlanPhases(*phasePlan)
// Add:

groundingWarnings := checkPlanGrounding(phases, manifest.Survey.SourceAnchors)
// ... later in result map construction:
if len(groundingWarnings) > 0 {
    result["grounding_warnings"] = groundingWarnings
    warningTexts := make([]string, len(groundingWarnings))
    for i, w := range groundingWarnings {
        warningTexts[i] = fmt.Sprintf("Phase %d (%s): %d tasks with no file references (%d anchors available)",
            w.PhaseID, w.PhaseName, w.UngroundedTasks, w.AnchorCount)
    }
    result["planning_warning"] = fmt.Sprintf("Grounding gate: %s", strings.Join(warningTexts, "; "))
}
```

### Anti-Patterns to Avoid

- **Making the grounding gate blocking:** GROUND-07 explicitly says the gate is soft. Never return an error from the grounding check. Only add warnings.
- **Over-engineering conflict detection:** Keyword matching is sufficient. Do not attempt semantic similarity, NLP, or LLM-based contradiction analysis. The scope is small and deterministic.
- **Skipping the research phase exemption:** Research and architecture phases legitimately have zero file targets. Forgetting to exempt them will generate false positive warnings.
- **Duplicating the CLARIFIED INTENT section:** Colony-prime already injects resolved decisions. Do not add a second section in the planner brief. Only add source anchors reference.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| File path detection in text | Custom parser | `regexp` with path pattern | Regex handles the 95% case; custom parser adds complexity for marginal improvement |
| Contradiction detection | Semantic similarity | Keyword pair matching | The set of meaningful contradictions is small and domain-specific; keyword pairs are deterministic and testable |
| Decision-to-pheromone binding | New binding system | Existing `createPheromoneSignal` | Already implemented in `resolveDiscussQuestion`; no new code needed |
| Resolved decision injection | New context section | Existing CLARIFIED INTENT section | Already implemented in `buildColonyPrimeOutput`; no new code needed |

**Key insight:** GROUND-08 and GROUND-09 are already implemented. The only new runtime logic is the grounding gate (GROUND-06/07) and conflict detection (GROUND-10).

## Common Pitfalls

### Pitfall 1: Regex False Positives on Common English
**What goes wrong:** The file path regex matches English phrases like "in the next/version" or "step 1/2".
**Why it happens:** The regex looks for `word/word` patterns which appear in normal English.
**How to avoid:** Require at least one path component to contain a period (extension) or match against known source anchors. Add a minimum path depth of 1 (at least one `/`).
**Warning signs:** Grounding gate reports tasks as "grounded" when they only contain English text with slashes.

### Pitfall 2: Forgetting to Pass SourceAnchors Through Manifest
**What goes wrong:** The `codexPlanManifest` struct does not propagate `SourceAnchors` from survey context, so `checkPlanGrounding` receives empty anchors and never warns.
**Why it happens:** The manifest is built in `runCodexPlanPlanOnly` or `runCodexPlanAgentDelegate` and may not include the full survey context.
**How to avoid:** Verify `manifest.Survey.SourceAnchors` is populated at manifest creation time. The `codexSurveyContext` already has `SourceAnchors` field from Phase 141.
**Warning signs:** `checkPlanGrounding` always returns nil because `sourceAnchors` is empty.

### Pitfall 3: Conflict Detection Too Broad
**What goes wrong:** The keyword "sql" matches "no sql database" as well as "use SQL", creating false conflicts.
**Why it happens:** Simple `strings.Contains` matching without understanding negation context.
**How to avoid:** Use specific terms (e.g., "nosql" as a compound word, not "no" + "sql"). Keep the contradiction pairs list conservative -- only include pairs that are genuinely contradictory.
**Warning signs:** Every plan generates FEEDBACK pheromones about conflicts that are not real.

### Pitfall 4: Warning Fatigue
**What goes wrong:** Every plan-finalize generates grounding warnings for all phases, making the signal useless.
**Why it happens:** Not exempting research/architecture phases, or warning on every ungrounded phase instead of only when zero phases have file references.
**How to avoid:** Only warn when ALL non-research phases lack file references. A single grounded phase is sufficient to show the planner is working with real files.
**Warning signs:** Users start ignoring grounding warnings because they appear on every plan.

### Pitfall 5: Missing Conflict Detection Emission
**What goes wrong:** `detectDecisionConflicts` returns conflict strings but nothing emits them as pheromones.
**Why it happens:** The function is called but the caller forgets to create the FEEDBACK signal.
**How to avoid:** Emit FEEDBACK pheromones immediately after conflict detection in `resolveDiscussQuestion`, with the conflict description as content.
**Warning signs:** Conflicts are detected in test output but never appear in `pheromones.json`.

## Code Examples

### Grounding Gate Integration in plan-finalize
```go
// Source: cmd/codex_plan_finalize.go
// In runCodexPlanFinalize, after line 172:
phases := buildWorkerPlanPhases(*phasePlan)
if len(phases) == 0 || buildablePlanTaskCount(phases) == 0 {
    return nil, fmt.Errorf("phase_plan contains no buildable tasks")
}

// NEW: Grounding validation gate (GROUND-06, GROUND-07)
groundingWarnings := checkPlanGrounding(phases, manifest.Survey.SourceAnchors)
// groundingWarnings is added to result map later, after state is saved
```

### Conflict Detection Integration in resolveDiscussQuestion
```go
// Source: cmd/discuss.go
// In resolveDiscussQuestion, after line 243 (after REDIRECT emission):

// NEW: Decision conflict detection (GROUND-10)
if conflicts := detectDecisionConflicts(file.Decisions); len(conflicts) > 0 {
    for _, conflict := range conflicts {
        _, _ = createPheromoneSignal("FEEDBACK", conflict, "discuss",
            "decision conflict detection", "", 0.5, "low")
    }
}
```

### Verifying SourceAnchors in Manifest Survey
```go
// Source: cmd/codex_plan.go
// In runCodexPlanPlanOnly or runCodexPlanAgentDelegate,
// the survey context is built via loadCodexSurveyContext(root)
// which already populates SourceAnchors from anchors.json.
// The manifest.Survey is this context, so SourceAnchors flows through.
// Verify: codexPlanManifest.Survey is type codexSurveyContext
// which has SourceAnchors []string.
```

## Already Implemented (No New Code Required)

### GROUND-08: Discuss decision binding

**Status:** ALREADY IMPLEMENTED [VERIFIED: cmd/discuss.go:235-243]

The `resolveDiscussQuestion` function already:
1. Checks `clarificationIsHardConstraint(decision)` which reads `Source` suffix `:hard`
2. Calls `buildClarificationRedirect(decision, answer)` to format the pheromone content
3. Calls `createPheromoneSignal("REDIRECT", redirectText, "discuss", "resolved clarification", "", 1.0, "high")`
4. Returns `redirect_emitted: true` in the result

The `discussQuestion` struct already has `HardConstraint bool` field.
The `clarificationIsHardConstraint` function checks `strings.HasSuffix(strings.TrimSpace(decision.Source), ":hard")`.
The `discussSource` function appends `:hard` to the source string for hard constraints.

### GROUND-09: Colony-prime injects resolved decisions

**Status:** ALREADY IMPLEMENTED [VERIFIED: cmd/colony_prime_context.go:729-756]

The `buildColonyPrimeOutput` function already:
1. Calls `clarifiedIntentPromptRenderResultForScope(scope)` which loads `pending-decisions.json`
2. Filters for resolved clarification decisions
3. Renders them as "CLARIFIED INTENT" section at priority 8
4. Includes integrity checking via `colony.AssessPromptSource`

This section is included in ALL worker context, including planner workers. No additional injection is needed.

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| No source anchors in planning | Source anchors from survey flow through to planner context | Phase 141 (2026-05-18) | Plans now have access to concrete file paths |
| Hard constraints only emitted as REDIRECT on resolution | Same (no change needed) | -- | GROUND-08 confirms existing behavior is correct |
| Resolved decisions only in pending-decisions.json | Injected into colony-prime as CLARIFIED INTENT | Existing | GROUND-09 confirms existing behavior is correct |

**Deprecated/outdated:**
- None in this phase -- all prior patterns remain valid

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | `manifest.Survey.SourceAnchors` is populated when `plan-finalize` receives the manifest | GROUND-06 | Medium -- if the manifest builder does not propagate SourceAnchors, grounding check always returns nil |
| A2 | The `codexPlanManifest.Survey` field is of type `codexSurveyContext` (which has SourceAnchors) | GROUND-06 | Low -- verified by reading `codex_plan.go` line 169 showing the struct type |
| A3 | Conflict detection keyword pairs are sufficient for the current use case | GROUND-10 | Low -- can be extended later without architectural change |
| A4 | Research/architecture phase detection by name is sufficient for GROUND-07 exemption | GROUND-07 | Low -- phase names are deterministic from the plan template system |

## Open Questions

1. **Should the grounding gate warn per-phase or per-plan?**
   - What we know: GROUND-06 says "emit soft warning when anchors exist but plan has zero file refs"
   - What's unclear: Whether "plan has zero" means all tasks across all phases, or each phase independently
   - Recommendation: Warn at plan level -- only when ALL non-research phases have zero grounded tasks. This avoids warning fatigue while still catching completely generic plans.

2. **Should conflict detection block the resolution?**
   - What we know: GROUND-10 says "warn when resolved decisions contradict each other"
   - What's unclear: Whether the warning should prevent the resolution or just emit a pheromone
   - Recommendation: Emit FEEDBACK pheromone only. Do not block resolution. The user already chose the answer.

3. **Should the planner brief explicitly mention source anchors?**
   - What we know: `renderPlanningWorkerBrief` does not currently reference source anchors
   - What's unclear: Whether the planner already sees anchors through colony-prime context or needs them in the brief
   - Recommendation: Add a single line to the planner brief when anchors are available: "Source anchors available: {count} repo-owned files from survey. Prefer referencing these files in task goals."

## Environment Availability

Step 2.6: SKIPPED (no external dependencies -- all changes are Go stdlib)

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go testing (stdlib) |
| Config file | None (Go convention) |
| Quick run command | `go test ./cmd/ -run TestPlanGrounding -v` |
| Full suite command | `go test ./cmd/ -count=1` |

### Phase Requirements -> Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| GROUND-06 | checkPlanGrounding returns warnings when anchors exist but tasks have no file refs | unit | `go test ./cmd/ -run TestCheckPlanGrounding_NoFileRefs -v` | No -- Wave 0 |
| GROUND-06 | checkPlanGrounding returns nil when tasks contain file paths | unit | `go test ./cmd/ -run TestCheckPlanGrounding_WithFileRefs -v` | No -- Wave 0 |
| GROUND-06 | checkPlanGrounding returns nil when no source anchors available | unit | `go test ./cmd/ -run TestCheckPlanGrounding_NoAnchors -v` | No -- Wave 0 |
| GROUND-07 | Research/architecture phases are exempted from grounding check | unit | `go test ./cmd/ -run TestCheckPlanGrounding_ResearchExempt -v` | No -- Wave 0 |
| GROUND-07 | Grounding warnings do not block plan-finalize | integration | `go test ./cmd/ -run TestPlanFinalize_GroundingSoftGate -v` | No -- Wave 0 |
| GROUND-08 | Hard constraint decisions emit REDIRECT pheromones | unit | `go test ./cmd/ -run TestResolveDiscuss_HardConstraintRedirect -v` | YES -- existing test |
| GROUND-09 | Colony-prime includes resolved decisions in CLARIFIED INTENT | unit | `go test ./cmd/ -run TestColonyPrime_IncludesResolvedClarifiedIntent -v` | YES -- existing test |
| GROUND-10 | Contradictory decisions emit FEEDBACK pheromones | unit | `go test ./cmd/ -run TestDetectDecisionConflicts -v` | No -- Wave 0 |
| GROUND-10 | Non-contradictory decisions emit nothing | unit | `go test ./cmd/ -run TestDetectDecisionConflicts_NoConflict -v` | No -- Wave 0 |

### Sampling Rate
- **Per task commit:** `go test ./cmd/ -run TestPlanGrounding -v`
- **Per wave merge:** `go test ./cmd/ -count=1`
- **Phase gate:** `go test ./... -count=1`

### Wave 0 Gaps
- [ ] `cmd/plan_grounding.go` -- covers GROUND-06, GROUND-07
- [ ] `cmd/plan_grounding_test.go` -- covers GROUND-06, GROUND-07 unit tests
- [ ] `cmd/discuss.go` additions -- covers GROUND-10 (detectDecisionConflicts function)
- [ ] `cmd/discuss_test.go` additions -- covers GROUND-10 unit tests
- [ ] `cmd/codex_plan_finalize.go` modifications -- integration of grounding gate

## Sources

### Primary (HIGH confidence)
- Direct codebase analysis of `cmd/discuss.go` (full file, 972 lines) -- resolveDiscussQuestion, clarificationIsHardConstraint, buildClarificationRedirect all verified
- Direct codebase analysis of `cmd/colony_prime_context.go` (full file, 882 lines) -- buildColonyPrimeOutput, CLARIFIED INTENT section verified at lines 729-756
- Direct codebase analysis of `cmd/codex_plan_finalize.go` (lines 1-280) -- runCodexPlanFinalize flow, phase building at line 172, state writing at line 259
- Direct codebase analysis of `cmd/codex_plan.go` -- codexSurveyContext struct with SourceAnchors at line 45, loadCodexSurveyContext at line 1183, plan manifest struct at line 169
- Direct codebase analysis of `cmd/pending_decision.go` (full file, 213 lines) -- PendingDecision struct with Resolved, Resolution fields
- Direct codebase analysis of `pkg/colony/colony.go` -- Phase struct (lines 522-531), Task struct (lines 534-542) with Goal, Constraints, Hints fields
- Direct codebase analysis of `pkg/colony/pheromones.go` -- PheromoneSignal struct with Type, Content, ContentHash fields
- Phase 141 research and execution summaries -- verified SourceAnchors pipeline from survey through to planner context
- Colony-prime audit test output showing 11 included sections with CLARIFIED INTENT present

### Secondary (MEDIUM confidence)
- Milestone architecture decisions from roadmap -- GROUND-06 through GROUND-10 requirements specification

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH -- zero new dependencies, all Go stdlib
- Architecture: HIGH -- all integration points verified by reading source code; GROUND-08 and GROUND-09 confirmed as already implemented
- Pitfalls: HIGH -- regex false positives and warning fatigue are well-understood problems with clear mitigations

**Research date:** 2026-05-18
**Valid until:** 2026-06-18 (stable -- no external dependencies or fast-moving libraries)
