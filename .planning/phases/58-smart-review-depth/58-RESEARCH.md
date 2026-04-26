# Phase 58: Smart Review Depth - Research

**Researched:** 2026-04-27
**Domain:** Go CLI runtime review orchestration, multi-agent dispatch filtering
**Confidence:** HIGH

## Summary

Phase 58 adds a review depth system (light/heavy/auto) that controls which review agents run during builds and continues. The system is conceptually straightforward: a `resolveReviewDepth()` function computes depth based on phase position (final phase = heavy), keyword detection (security/release = heavy), and explicit flags (`--light`/`--heavy`). Both the build dispatch planner and the continue review dispatcher must respect this depth by conditionally skipping heavy agents (Auditor, Gatekeeper, Probe, Weaver, Medic, Measurer, Chaos) in light mode. Workers learn their depth through colony-prime context injection.

The existing codebase already has all the infrastructure needed. Build dispatch uses `normalizedBuildDepth()` and `plannedBuildDispatchesForSelection()` to gate specialist spawning. Continue review uses `codexContinueReviewSpecs` (currently always 3 specs: Gatekeeper, Auditor, Probe) and `plannedContinueReviewDispatches()` to build dispatch lists. The visual rendering layer (`codex_visuals.go`) already has progress-aware output. Colony-prime already injects sections with priorities and budgets.

**Primary recommendation:** Create `resolveReviewDepth()` as a pure function in a new file `cmd/review_depth.go`, extend the existing dispatch planners to filter by depth, add `--light`/`--heavy` flags to both `buildCmd` and `continueCmd`, inject depth into colony-prime as a short section, and add a depth line to the visual renderers.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
- **D-01:** Light mode runs Watcher only -- 7 heavy agents (Auditor, Gatekeeper, Probe, Weaver, Medic, Measurer, Chaos) are skipped on intermediate phases
- **D-02:** Heavy mode runs all agents (Watcher + all 7 heavy agents) -- full review gauntlet
- **D-03:** Chaos has 30% random sampling on light phases (deterministic by phase number hash -- same phase always gets Chaos or not across runs)
- **D-04:** Chaos always runs on heavy phases (final phase + security/release keyword phases)
- **D-05:** Final phase = last entry in COLONY_STATE.json phases array. Simple, deterministic, auto-adjusts when phases are inserted
- **D-06:** `--light` flag forces light review on any phase (except cannot override final phase -- DEPTH-03)
- **D-07:** `--heavy` flag forces heavy review on any phase -- gives user full control. Symmetrical with `--light`
- **D-08:** Without either flag, depth is auto-detected: heavy if final phase or security/release keywords, light otherwise
- **D-09:** Case-insensitive substring matching on phase name for 12 keywords: security, auth, crypto, secrets, permissions, compliance, audit, release, deploy, production, ship, launch
- **D-10:** Keyword list is hardcoded in Go runtime (not configurable)
- **D-11:** Review depth shown in wrapper output only, not runtime dispatch output
- **D-12:** Format: "Review depth: light (Phase 3 of 7 -- final phase gets full review)" or "Review depth: heavy (Phase 7 of 7 -- final phase)"
- **D-13:** Workers receive their review depth in colony-prime context: "Light review -- core verification only" or "Heavy review -- full quality gauntlet"
- **D-14:** Workers adapt thoroughness based on depth

### Claude's Discretion
- Exact hash function for deterministic Chaos sampling (simple modulo is fine)
- Exact phrasing of the worker depth context injection
- How resolveReviewDepth integrates with existing normalizedBuildDepth system
- Whether review_depth gets stored in COLONY_STATE.json or computed on-the-fly each time

### Deferred Ideas (OUT OF SCOPE)
- Configurable keyword list for auto-heavy detection -- hardcoding is simpler for now
- Random (non-deterministic) Chaos sampling -- deterministic is better for reproducibility
- Depth-based time budgets for agents -- future enhancement
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| DEPTH-01 | `resolveReviewDepth()` helper determines light vs heavy review based on phase position, keyword detection, and `--light` flag | New pure function in `cmd/review_depth.go`. Inputs: phase (colony.Phase), totalPhases (int), lightFlag (bool), heavyFlag (bool). Returns "light" or "heavy". See Architecture Patterns below. |
| DEPTH-02 | `--light` flag on build and continue commands skips heavy agents on intermediate phases | Add `buildCmd.Flags().Bool("light", false, ...)` and `continueCmd.Flags().Bool("light", false, ...)`. Filter dispatches in `plannedBuildDispatchesForSelection()` and `plannedContinueReviewDispatches()`. |
| DEPTH-03 | Final phase always gets heavy review regardless of `--light` flag | `resolveReviewDepth()` enforces this: if `phase.ID == totalPhases`, return "heavy" regardless of `--light` flag. |
| DEPTH-04 | Phases with security/release keywords in name auto-detect as heavy | `phaseHasHeavyKeywords()` checks 12 keywords against `strings.ToLower(phase.Name)` using `strings.Contains`. |
| DEPTH-05 | Continue playbooks skip heavy agents when depth is light | Filter `codexContinueReviewSpecs` in `plannedContinueReviewDispatches()` based on resolved depth. The 3 existing review specs (Gatekeeper, Auditor, Probe) are all heavy and get skipped entirely in light mode. |
| DEPTH-06 | Review depth displayed to user in wrapper output | Add depth line to `renderBuildVisualWithDispatches()` and `renderContinueVisual()` in `codex_visuals.go`. Format per D-12. |
</phase_requirements>

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| resolveReviewDepth() logic | Go Runtime | -- | Pure computation, must be deterministic and testable |
| Build dispatch filtering | Go Runtime | -- | `plannedBuildDispatchesForSelection()` already gates by build depth |
| Continue dispatch filtering | Go Runtime | -- | `plannedContinueReviewDispatches()` builds dispatch list |
| CLI flag registration | Go Runtime (Cobra) | -- | `buildCmd` and `continueCmd` flags registered in `codex_workflow_cmds.go` |
| Depth display in visuals | Go Runtime | -- | `renderBuildVisualWithDispatches()` and `renderContinueVisual()` in `codex_visuals.go` |
| Worker depth awareness | Go Runtime (colony-prime) | -- | Context injection in `buildColonyPrimeOutput()` |
| Wrapper markdown depth display | Wrapper (Claude/OpenCode) | -- | `build.md` and `continue.md` wrapper files read runtime output |

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| Go stdlib (strings, fmt) | Go 1.26 | String matching, formatting | Built-in, no dependencies |
| cobra | existing | CLI flag registration | Already used for all commands |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| pkg/colony | existing | ColonyState and Phase types | Type access for phase position and name |
| pkg/storage | existing | JSON store for state loading | Reading COLONY_STATE.json |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Computed on-the-fly review depth | Stored in COLONY_STATE.json | Computed is simpler, no state migration, no backward compat risk. Storing adds write surface and must handle `omitempty`. Computed is recommended. |

**Installation:**
No new packages needed. Everything uses existing Go stdlib and project packages.

## Architecture Patterns

### System Architecture Diagram

```
resolveReviewDepth(phase, totalPhases, --light, --heavy)
        |
        v
   +----+----+
   | "light" |  or  | "heavy" |
   +----+----+           |
        |                |
        v                v
  BUILD PATH:         BUILD PATH:
  plannedBuild        plannedBuild
  DispatchesFor       DispatchesFor
  Selection()         Selection()
  - Watcher only      - Watcher
  - Skip: Measurer,   - Measurer
    Chaos             - Chaos
  - Chaos 30% sample  (always)

  CONTINUE PATH:      CONTINUE PATH:
  plannedContinue     plannedContinue
  ReviewDispatches()  ReviewDispatches()
  - Review specs      - Review specs
    = [] (empty)      = [Gatekeeper,
  - No review wave      Auditor, Probe]

  COLONY-PRIME:       COLONY-PRIME:
  "Light review --    "Heavy review --
   core only"          full gauntlet"
```

### Recommended Project Structure
```
cmd/
├── review_depth.go              # NEW: resolveReviewDepth(), heavy keyword detection, Chaos sampling
├── review_depth_test.go         # NEW: unit tests for all depth logic
├── codex_build.go               # MODIFY: pass review depth to dispatch planning
├── codex_continue.go            # MODIFY: filter review specs by depth
├── codex_visuals.go             # MODIFY: add depth line to build/continue visual renderers
├── colony_prime_context.go      # MODIFY: add review depth section
├── codex_workflow_cmds.go       # MODIFY: add --light/--heavy flags
├── codex_workflow_cmds_test.go  # MODIFY: test flag exposure (if exists)
├── codex_build_test.go          # MODIFY: test depth-aware dispatch planning
└── codex_continue_test.go       # MODIFY: test depth-aware review filtering
```

### Pattern 1: Pure Depth Resolution Function
**What:** A single pure function that takes phase metadata and flags, returns "light" or "heavy".
**When to use:** Called from both build and continue code paths.
**Example:**
```go
// Source: [VERIFIED: cmd/codex_build.go line 859 normalizedBuildDepth pattern]
// cmd/review_depth.go

var heavyKeywords = []string{
    "security", "auth", "crypto", "secrets", "permissions",
    "compliance", "audit", "release", "deploy", "production",
    "ship", "launch",
}

type ReviewDepth string

const (
    ReviewDepthLight ReviewDepth = "light"
    ReviewDepthHeavy ReviewDepth = "heavy"
)

func resolveReviewDepth(phase colony.Phase, totalPhases int, lightFlag, heavyFlag bool) ReviewDepth {
    // D-03: Final phase always heavy
    if phase.ID == totalPhases {
        return ReviewDepthHeavy
    }
    // D-07: --heavy overrides everything (except final, handled above)
    if heavyFlag {
        return ReviewDepthHeavy
    }
    // D-04: Keyword detection triggers heavy
    if phaseHasHeavyKeywords(phase.Name) {
        return ReviewDepthHeavy
    }
    // D-08: Auto-detect default is light for non-final, non-keyword phases
    return ReviewDepthLight
}

func phaseHasHeavyKeywords(name string) bool {
    lower := strings.ToLower(name)
    for _, kw := range heavyKeywords {
        if strings.Contains(lower, kw) {
            return true
        }
    }
    return false
}

func chaosShouldRunInLightMode(phaseID int) bool {
    // D-03: 30% deterministic sampling via modulo
    return phaseID%10 < 3
}
```

### Pattern 2: Depth-Aware Build Dispatch Filtering
**What:** Extend `plannedBuildDispatchesForSelection()` to accept review depth and conditionally skip heavy agents.
**When to use:** During build dispatch planning.
**Example:**
```go
// In codex_build.go, modify plannedBuildDispatchesForSelection:
func plannedBuildDispatchesForSelection(phase colony.Phase, depth string, selectedTaskIDs []string, reviewDepth ReviewDepth) []codexBuildDispatch {
    // ... existing dispatch logic ...

    // Probe always runs (light verification)
    if len(selected) == 0 {
        dispatches = append(dispatches, codexBuildSpecialistDispatch(phase, "probe", ...))
    }

    // Watcher always runs (light + heavy)
    dispatches = append(dispatches, codexBuildDispatch{
        Stage: "verification", Caste: "watcher", ...
    })

    // Heavy-only agents
    if reviewDepth == ReviewDepthHeavy {
        if len(selected) == 0 && (depth == "deep" || depth == "full") {
            dispatches = append(dispatches, codexBuildSpecialistDispatch(phase, "measurement", ...))
        }
        if depth == "full" {
            dispatches = append(dispatches, codexBuildDispatch{Stage: "resilience", Caste: "chaos", ...})
        }
    }

    // Light mode: Chaos with 30% deterministic sampling
    if reviewDepth == ReviewDepthLight && chaosShouldRunInLightMode(phase.ID) {
        dispatches = append(dispatches, codexBuildDispatch{Stage: "resilience", Caste: "chaos", ...})
    }

    return dispatches
}
```

### Pattern 3: Depth-Aware Continue Review Filtering
**What:** Filter `codexContinueReviewSpecs` based on review depth before dispatching.
**When to use:** During continue review wave planning.
**Example:**
```go
// In codex_continue.go, modify plannedContinueReviewDispatches:
func plannedContinueReviewDispatches(root string, phase colony.Phase, manifest codexContinueManifest, verification codexContinueVerificationReport, assessment codexContinueAssessment, invoker codex.WorkerInvoker, workerTimeout time.Duration, reviewDepth ReviewDepth) []codex.WorkerDispatch {
    // ... existing setup ...

    specs := codexContinueReviewSpecs
    if reviewDepth == ReviewDepthLight {
        specs = []codexContinueReviewSpec{} // Skip all review agents in light mode
    }

    for idx, spec := range specs {
        // ... existing dispatch building ...
    }
    return dispatches
}
```

### Pattern 4: Colony-Prime Depth Injection
**What:** Add a short section to colony-prime context informing workers of review depth.
**When to use:** During `buildColonyPrimeOutput()`.
**Example:**
```go
// In colony_prime_context.go, add after state section:
if state.CurrentPhase > 0 && state.CurrentPhase <= len(state.Plan.Phases) {
    phase := state.Plan.Phases[state.CurrentPhase-1]
    reviewDepth := resolveReviewDepth(phase, len(state.Plan.Phases), false, false)
    var depthText string
    if reviewDepth == ReviewDepthLight {
        depthText = "Light review -- core verification only"
    } else {
        depthText = "Heavy review -- full quality gauntlet"
    }
    sections = append(sections, colonyPrimeSection{
        name:           "review_depth",
        title:          "Review Depth",
        source:         statePath,
        content:        fmt.Sprintf("## Review Depth\n\n%s\n", depthText),
        priority:       6,
        freshnessScore: 1.0,
    })
}
```

### Anti-Patterns to Avoid
- **Storing review depth in COLONY_STATE.json:** This adds backward-compat surface (`omitempty` on a new field), state migration complexity, and staleness risk. The depth should be computed fresh each time from the same inputs (phase position, keywords, flags). `[VERIFIED: pkg/colony/colony.go line 168 ColonyDepth already uses omitempty but is a separate concept]`
- **Mixing review depth with build depth:** `normalizedBuildDepth()` controls specialist spawning (archaeologist, oracle, architect, ambassador). Review depth controls quality gate agents (gatekeeper, auditor, probe, chaos, etc.). They are orthogonal concepts. Do not merge them. `[VERIFIED: cmd/codex_build.go lines 580-654 shows build depth gates specialists, not review agents]`
- **Duplicating depth logic in wrapper markdown:** Wrapper files should read depth from runtime JSON output, not compute it independently. The runtime is the truth layer. `[VERIFIED: .claude/commands/ant/build.md and continue.md are thin wrappers that read runtime output]`

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Case-insensitive matching | Custom Unicode-aware comparison | `strings.Contains(strings.ToLower(name), keyword)` | Go's `strings.ToLower` handles ASCII keywords correctly; all 12 keywords are ASCII |
| Deterministic sampling | Random number generator with seed | Simple modulo: `phaseID % 10 < 3` | No need for `math/rand`, no seed management, fully deterministic across runs |
| Phase position detection | Complex state parsing | `phase.ID == len(state.Plan.Phases)` | The phases array is 1-indexed by ID, last element = final phase `[VERIFIED: pkg/colony/colony.go Phase struct has ID int, Plan has Phases []Phase]` |

**Key insight:** This entire feature is pure computation and conditional filtering. No new data structures, no new storage, no new external dependencies.

## Common Pitfalls

### Pitfall 1: Confusing Build Depth with Review Depth
**What goes wrong:** `normalizedBuildDepth()` (standard/deep/full) already exists for build dispatch planning. Review depth (light/heavy) is a separate axis.
**Why it happens:** Both control "how many agents run" but at different lifecycle points.
**How to avoid:** Keep them completely separate. Build depth controls specialist research agents. Review depth controls quality gate agents. Pass both parameters through to dispatch planning independently.
**Warning signs:** If `resolveReviewDepth` takes or returns a build depth string, the concepts have been conflated.

### Pitfall 2: Chaos Over-Spawning in Light Mode
**What goes wrong:** Chaos runs on every light phase instead of 30% of them.
**Why it happens:** Forgetting the deterministic sampling check or making it non-deterministic.
**How to avoid:** Use `phaseID % 10 < 3` which gives exactly 3 out of every 10 phases Chaos. Document that the same phase always gets the same result across runs.
**Warning signs:** Chaos appears in every build dispatch manifest for intermediate phases.

### Pitfall 3: Final Phase Override Bypass
**What goes wrong:** `--light` flag somehow overrides final-phase heavy requirement.
**Why it happens:** The check order in `resolveReviewDepth()` puts the flag check before the final-phase check.
**How to avoid:** Final-phase check MUST be the first condition in `resolveReviewDepth()`. If `phase.ID == totalPhases`, return "heavy" unconditionally, before checking any flags.
**Warning signs:** `--light` accepted on the final phase without an error or warning.

### Pitfall 4: Continue Review Wave Always Spawning 3 Agents
**What goes wrong:** The continue review wave still spawns Gatekeeper, Auditor, and Probe even on light phases.
**Why it happens:** `plannedContinueReviewDispatches()` iterates `codexContinueReviewSpecs` unconditionally.
**How to avoid:** Filter specs by depth BEFORE the dispatch loop. If light, set specs to empty slice. The review wave becomes a no-op, and `runCodexContinueReview` returns `report.Passed = true` immediately.
**Warning signs:** Review wave shows Gatekeeper/Auditor/Probe in continue output for intermediate phases.

### Pitfall 5: Depth Not Reaching Wrapper Markdown
**What goes wrong:** Wrapper markdown cannot show review depth because it is not in the runtime JSON output.
**Why it happens:** The visual renderer adds the depth line but the structured JSON result does not include it.
**How to avoid:** Add `"review_depth": "light"` or `"review_depth": "heavy"` to the result map in both build and continue code paths. The wrapper reads this from the JSON.
**Warning signs:** Wrapper shows "Review depth: unknown" or omits the line entirely.

### Pitfall 6: Colony-Prime Depth Injection During Wrong Phase State
**What goes wrong:** Colony-prime injects review depth when colony is not in EXECUTING state, showing stale or wrong depth.
**Why it happens:** `buildColonyPrimeOutput()` runs for all colony-prime invocations, not just during builds.
**How to avoid:** Only inject the review depth section when `state.CurrentPhase > 0 && state.CurrentPhase <= len(state.Plan.Phases)`. This matches the existing pattern for phase-specific sections.
**Warning signs:** Colony-prime shows review depth during init or plan phases.

## Code Examples

### Complete resolveReviewDepth() with All Rules
```go
// Source: [ASSUMED -- follows CONTEXT.md decisions D-01 through D-14]
// cmd/review_depth.go

package cmd

import (
    "strings"
    "github.com/calcosmic/Aether/pkg/colony"
)

type ReviewDepth string

const (
    ReviewDepthLight ReviewDepth = "light"
    ReviewDepthHeavy ReviewDepth = "heavy"
)

var heavyKeywords = []string{
    "security", "auth", "crypto", "secrets", "permissions",
    "compliance", "audit", "release", "deploy", "production",
    "ship", "launch",
}

// resolveReviewDepth determines whether a phase gets light or heavy review.
// Priority order: final phase > --heavy flag > keyword detection > auto (light).
func resolveReviewDepth(phase colony.Phase, totalPhases int, lightFlag, heavyFlag bool) ReviewDepth {
    // D-03: Final phase always heavy, regardless of --light
    if phase.ID == totalPhases {
        return ReviewDepthHeavy
    }
    // D-07: --heavy forces heavy on any non-final phase
    if heavyFlag {
        return ReviewDepthHeavy
    }
    // D-04: Security/release keywords auto-detect as heavy
    if phaseHasHeavyKeywords(phase.Name) {
        return ReviewDepthHeavy
    }
    // D-06: --light explicitly requests light (honored because not final and not keyword)
    // D-08: Default is light for intermediate phases
    return ReviewDepthLight
}

func phaseHasHeavyKeywords(name string) bool {
    lower := strings.ToLower(name)
    for _, kw := range heavyKeywords {
        if strings.Contains(lower, kw) {
            return true
        }
    }
    return false
}

// chaosShouldRunInLightMode returns true for ~30% of phases deterministically.
// Uses simple modulo: phases with ID % 10 in [0, 1, 2] get Chaos.
func chaosShouldRunInLightMode(phaseID int) bool {
    return phaseID%10 < 3
}
```

### Adding Flags to Cobra Commands
```go
// Source: [VERIFIED: cmd/codex_workflow_cmds.go lines 685-693 shows existing flag pattern]
// In codex_workflow_cmds.go init() function:

buildCmd.Flags().Bool("light", false, "Force light review (skip heavy agents on intermediate phases)")
buildCmd.Flags().Bool("heavy", false, "Force heavy review (full quality gauntlet on any phase)")

continueCmd.Flags().Bool("light", false, "Force light review (skip heavy review agents)")
continueCmd.Flags().Bool("heavy", false, "Force heavy review (full review gauntlet)")
```

### Depth Display in Visual Renderer
```go
// Source: [VERIFIED: cmd/codex_visuals.go lines 903-951 shows renderBuildVisualWithDispatches]
// Add after line 911 (Phase: name):

// Review depth display (D-11, D-12)
if depthLine := renderReviewDepthLine(reviewDepth, phase.ID, len(state.Plan.Phases)); depthLine != "" {
    b.WriteString(depthLine)
    b.WriteString("\n")
}

// Helper function:
func renderReviewDepthLine(depth ReviewDepth, phaseNum, totalPhases int) string {
    if depth == ReviewDepthHeavy {
        if phaseNum == totalPhases {
            return "Review depth: heavy (final phase)"
        }
        return fmt.Sprintf("Review depth: heavy (Phase %d of %d)", phaseNum, totalPhases)
    }
    return fmt.Sprintf("Review depth: light (Phase %d of %d -- final phase gets full review)", phaseNum, totalPhases)
}
```

### Passing Review Depth Through Build Path
```go
// Source: [VERIFIED: cmd/codex_build.go line 189 runCodexBuildWithOptions and line 231 depth handling]
// In runCodexBuildWithOptions, after computing build depth:

lightFlag, _ := cmd.Flags().GetBool("light")  // passed from caller
heavyFlag, _ := cmd.Flags().GetBool("heavy")  // passed from caller
reviewDepth := resolveReviewDepth(phase, len(state.Plan.Phases), lightFlag, heavyFlag)

dispatches := plannedBuildDispatchesForSelection(phase, depth, selectedTaskIDs, reviewDepth)

// Add to result map:
result["review_depth"] = string(reviewDepth)
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| All phases get same review agents | Review depth varies by phase position and content | This phase (58) | Intermediate phases become faster; final/security phases keep full review |

**Deprecated/outdated:**
- The build-verify playbook's Chaos depth check (`DEPTH CHECK: Skip if colony depth is not "full"`) will need updating to also consider review depth. Currently Chaos only runs at build depth "full". With review depth, Chaos can also run during "standard" builds if review depth is heavy. `[VERIFIED: .aether/docs/command-playbooks/build-verify.md line 259]`

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | `phase.ID` is always equal to the phase's 1-based index in `state.Plan.Phases` | Architecture Patterns | Final phase detection breaks; verify by checking existing colony states |
| A2 | Colony-prime context injection does not need to know about `--light`/`--heavy` flags -- it always uses auto-detected depth | Colony-Prime Depth Injection | Workers may not know when user explicitly forced a depth mode |
| A3 | The `codexContinueReviewSpecs` list currently has exactly 3 entries (Gatekeeper, Auditor, Probe) -- no other review specs exist elsewhere | Pattern 3 | Filtering logic misses review agents that are defined outside this slice |
| A4 | `runCodexContinueReview` handles empty dispatch lists gracefully (returns `report.Passed = true`) | Pattern 3 | Light-mode continue would block on empty review wave |
| A5 | Wrapper markdown does not need modification for depth display -- the visual renderer output already reaches wrappers | Architecture Patterns | Users may not see depth information |

**Verifiable in session but not yet checked:**
- A1: Read any COLONY_STATE.json to verify phase ID = index+1
- A4: Check if `runCodexContinueReview` handles 0 dispatches

## Open Questions

1. **How does review depth interact with the Measurer conditional in build-verify?**
   - What we know: Measurer already has a conditional spawn based on performance keywords. Review depth adds another layer.
   - What's unclear: Should Measurer run in light mode if the phase IS performance-sensitive?
   - Recommendation: Measurer is a heavy agent per D-01. Even if performance-sensitive, light mode skips it. The performance keyword check is a different concept from review depth.

2. **Should `--light` and `--heavy` be mutually exclusive?**
   - What we know: Both flags are boolean, could both be set to true.
   - What's unclear: What happens if user passes `--light --heavy`?
   - Recommendation: If both are set, `--heavy` wins (it is the safer option). Document this. The order in `resolveReviewDepth` already handles this: final phase check > heavy flag > keyword check > light default.

3. **Should the build-verify playbook's Chaos depth check be updated?**
   - What we know: The playbook currently gates Chaos on `colony_depth == "full"`. With review depth, Chaos should also run when review depth is heavy AND build depth is "standard".
   - What's unclear: Whether the playbook is the only place Chaos is gated, or if the Go runtime also gates it.
   - Recommendation: The Go runtime `plannedBuildDispatchesForSelection()` gates Chaos on `depth == "full"` at line 646. This is the runtime truth. The playbook is for wrapper-based execution. Both need review depth awareness. `[VERIFIED: cmd/codex_build.go line 646]`

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go toolchain | Build/test | ✓ | go1.26.1 | -- |
| pkg/colony | Type definitions | ✓ | existing | -- |
| pkg/storage | State loading | ✓ | existing | -- |
| cobra | CLI flags | ✓ | existing | -- |

**Missing dependencies with no fallback:**
- None

**Missing dependencies with fallback:**
- None

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go testing (stdlib) |
| Config file | none |
| Quick run command | `go test ./cmd/ -run TestReview -v -count=1` |
| Full suite command | `go test ./... -race` |

### Phase Requirements to Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| DEPTH-01 | resolveReviewDepth returns correct depth for all input combos | unit | `go test ./cmd/ -run TestResolveReviewDepth -v` | Wave 0 |
| DEPTH-01 | phaseHasHeavyKeywords detects all 12 keywords | unit | `go test ./cmd/ -run TestPhaseHasHeavyKeywords -v` | Wave 0 |
| DEPTH-01 | chaosShouldRunInLightMode deterministic sampling | unit | `go test ./cmd/ -run TestChaosShouldRunInLightMode -v` | Wave 0 |
| DEPTH-02 | --light flag registered on build and continue commands | unit | `go test ./cmd/ -run TestBuildCommandExposesLightFlag -v` | Wave 0 |
| DEPTH-02 | --heavy flag registered on build and continue commands | unit | `go test ./cmd/ -run TestBuildCommandExposesHeavyFlag -v` | Wave 0 |
| DEPTH-03 | Final phase always heavy regardless of --light | unit | `go test ./cmd/ -run TestFinalPhaseAlwaysHeavy -v` | Wave 0 |
| DEPTH-04 | Security keywords auto-detect heavy | unit | `go test ./cmd/ -run TestSecurityKeywordsAutoHeavy -v` | Wave 0 |
| DEPTH-05 | Continue skips review agents in light mode | unit | `go test ./cmd/ -run TestContinueLightSkipsReview -v` | Wave 0 |
| DEPTH-06 | Visual renderer includes depth line | unit | `go test ./cmd/ -run TestRenderReviewDepthLine -v` | Wave 0 |

### Sampling Rate
- **Per task commit:** `go test ./cmd/ -run TestReview -v -count=1`
- **Per wave merge:** `go test ./... -race`
- **Phase gate:** `go test ./... -race` (full suite green)

### Wave 0 Gaps
- [ ] `cmd/review_depth_test.go` -- covers DEPTH-01, DEPTH-03, DEPTH-04
- [ ] `cmd/codex_build_test.go` additions -- covers DEPTH-02 (flag tests)
- [ ] `cmd/codex_continue_test.go` additions -- covers DEPTH-05

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | no | -- |
| V3 Session Management | no | -- |
| V4 Access Control | no | -- |
| V5 Input Validation | yes | Cobra flag parsing + Go type system |
| V6 Cryptography | no | -- |

### Known Threat Patterns for Go CLI

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Flag injection via arguments | Tampering | Cobra handles argument parsing; flags are booleans, no string injection possible |
| Phase ID out of bounds | Tampering | Existing bounds checks in `runCodexBuildWithOptions` and `runCodexContinue` |

## Sources

### Primary (HIGH confidence)
- `cmd/codex_build.go` -- normalizedBuildDepth() (line 859), plannedBuildDispatchesForSelection() (lines 569-657), Chaos spawn gate (line 646)
- `cmd/codex_continue.go` -- codexContinueReviewSpecs (lines 793-808), plannedContinueReviewDispatches() (lines 892-916)
- `cmd/colony_prime_context.go` -- buildColonyPrimeOutput() (lines 333-768), section assembly pattern
- `cmd/codex_visuals.go` -- renderBuildVisualWithDispatches() (lines 903-951), renderContinueVisual() (lines 1021-1085)
- `cmd/codex_workflow_cmds.go` -- buildCmd (line 82), continueCmd (line 140), flag registration (lines 685-693)
- `pkg/colony/colony.go` -- Phase struct (line 279), ColonyState struct (line 151)
- `.aether/docs/command-playbooks/build-verify.md` -- Chaos depth check (line 259)

### Secondary (MEDIUM confidence)
- `.claude/commands/ant/build.md` -- wrapper pattern for build
- `.claude/commands/ant/continue.md` -- wrapper pattern for continue
- `.opencode/commands/ant/build.md` -- OpenCode mirror
- `.opencode/commands/ant/continue.md` -- OpenCode mirror

### Tertiary (LOW confidence)
- None

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - no new dependencies, all verified in codebase
- Architecture: HIGH - existing patterns (depth gating, flag registration, visual rendering) well established
- Pitfalls: HIGH - identified from direct code reading and understanding the dual-depth-axis problem

**Research date:** 2026-04-27
**Valid until:** 2026-05-27 (stable codebase, no external dependencies)
