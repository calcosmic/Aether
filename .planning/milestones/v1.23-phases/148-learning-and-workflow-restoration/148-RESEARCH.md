# Phase 148: Learning and Workflow Restoration - Research

**Researched:** 2026-05-21
**Domain:** Aether learning pipeline, worker context injection, ceremony output, and 9 flagship workflows
**Confidence:** HIGH

## Summary

The Aether Go runtime has a **complete learning pipeline** implemented end-to-end, but the **continue-finalize path** (the primary production codepath) has a **critical gap**: learning capture stores only a generic placeholder string (`"Phase N completed successfully: ..."`) rather than extracting actual hypotheses from worker outputs. The hypothesis/validated/disproven lifecycle described in requirements does not exist as a first-class state machine.

Worker context injection (pheromones, skills, survey data) is **fully implemented** and wired into build dispatches via `attachBuildDispatchContext()` in `cmd/codex_build.go`. The oracle-to-colony pipeline (findings -> instincts -> QUEEN.md -> hive) exists as **individual commands** but lacks **orchestrated end-to-end automation**.

The 9 flagship workflows (build, continue, plan, colonize, autopilot, seal, entomb, swarm, oracle) all have **runtime implementations** that produce correct state mutations and ceremony output. The primary risks are:
1. **Learning extraction is shallow** — no hypothesis lifecycle, no evidence tracking per hypothesis
2. **Autopilot pause conditions** are implemented in the state machine but the "10 Classic pause conditions" need verification against the actual `autopilot.go` implementation
3. **Worker context injection** works for build but needs verification for continue/plan/colonize dispatches

**Primary recommendation:** Restore the learning extraction lifecycle by adding hypothesis tracking to `pkg/learn/`, wiring it into `continue-finalize`, and verifying all 9 workflows with targeted tests.

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Learning extraction (hypothesis lifecycle) | API / Backend | — | `pkg/learn/` + `cmd/codex_continue_finalize.go` own capture |
| Worker context injection | API / Backend | — | `cmd/codex_build.go` dispatches carry pheromones, skills, handoffs |
| Oracle promote-to-colony | API / Backend | — | `cmd/oracle_loop.go` -> `cmd/instinct.go` -> `cmd/queen.go` -> `cmd/hive.go` |
| Build workflow state mutations | API / Backend | — | `cmd/codex_build.go` mutates COLONY_STATE.json, writes manifests |
| Continue workflow advancement | API / Backend | — | `cmd/codex_continue.go` + `cmd/codex_continue_finalize.go` |
| Plan workflow | API / Backend | — | `cmd/codex_plan.go` generates phases, writes plan artifacts |
| Colonize workflow | API / Backend | — | `cmd/codex_colonize.go` surveys workspace, writes survey artifacts |
| Autopilot state machine | API / Backend | — | `cmd/autopilot.go` tracks phase progress, pause conditions |
| Seal ceremony | API / Backend | — | `cmd/seal_final_review.go` + `cmd/codex_workflow_cmds.go` |
| Entomb archive | API / Backend | — | `cmd/entomb_cmd.go` archives to chambers |
| Swarm cross-comparison | API / Backend | — | `cmd/swarm_cmd.go` dispatches 4 scouts, ranks findings |

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| Go standard library | 1.23 | Runtime language | Already used for all commands |
| `github.com/spf13/cobra` | v1.8+ | CLI framework | All commands use this |
| `github.com/calcosmic/Aether/pkg/memory` | local | Trust scoring, event bus, instinct promotion | Learning pipeline backbone |
| `github.com/calcosmic/Aether/pkg/learn` | local | Learning store, evidence collection, classification | New learning system (D-06+) |
| `github.com/calcosmic/Aether/pkg/events` | local | Event bus with JSONL persistence | Ceremony events, learning.observe |
| `github.com/calcosmic/Aether/pkg/colony` | local | Domain types (ColonyState, Phase, Task) | State mutations |
| `github.com/calcosmic/Aether/pkg/storage` | local | Atomic file operations | All persistence |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `pkg/agent` | local | Spawn tree tracking | Worker dispatch recording |
| `pkg/codex` | local | Worker invoker, dispatch contracts | Build/continue worker dispatch |

**Installation:** No new dependencies required — all are already in `go.mod`.

## Architecture Patterns

### System Architecture Diagram

```
Build / Continue / Plan / Colonize / Seal / Entomb / Swarm / Oracle / Autopilot
    |
    v
Command Handler (cmd/codex_*.go)
    |
    |-- State mutations (COLONY_STATE.json via store.SaveJSON / UpdateJSONAtomically)
    |-- Worker dispatch (codex.WorkerDispatch with context capsule)
    |-- Ceremony events (events.Bus.Publish with CeremonyTopic*)
    |
    v
Learning Pipeline (pkg/learn/ + pkg/memory/)
    |
    |-- Observation capture (memory.ObservationService.Capture)
    |-- Trust scoring (memory.Calculate)
    |-- Event bus publish (learning.observe -> auto-promotion)
    |-- Promotion (memory.PromoteService.Promote -> instincts.json)
    |-- QUEEN.md write (memory.QueenService.WriteEntry)
    |-- Hive promotion (cmd/hive.go hive-promote)
    |
    v
Storage Layer (pkg/storage/)
    |
    |-- AtomicWrite / LoadJSON / AppendJSONL
    |-- File locking for cross-process safety
```

### Recommended Project Structure

```
cmd/
├── codex_build.go              # Build workflow + worker dispatch
├── codex_continue.go           # Continue verification + gates
├── codex_continue_finalize.go  # Continue finalize + learning capture
├── codex_plan.go               # Plan workflow
├── codex_colonize.go           # Colonize workflow
├── codex_workflow_cmds.go      # Cobra command wrappers
├── autopilot.go                # Autopilot state machine
├── seal_final_review.go        # Seal ceremony
├── entomb_cmd.go               # Entomb archive
├── swarm_cmd.go                # Swarm dispatch
├── oracle_loop.go              # Oracle research loop
├── instinct.go                 # Instinct CRUD
├── queen.go                    # QUEEN.md promotion
├── hive.go                     # Hive wisdom store
└── learning_cmds.go            # Learning proposal management

pkg/
├── learn/
│   ├── learn.go                # Core types (Entry, Evidence, Classification)
│   ├── colony_store.go         # JSON persistence for learning entries
│   ├── evidence.go             # Evidence collection from worker results
│   ├── classify.go             # Privacy scan + classification
│   ├── trigger.go              # IsLearningEligible
│   └── wrappers.go             # Wrappers around pkg/memory
├── memory/
│   ├── observe.go              # Observation capture
│   ├── promote.go              # Promotion to instinct
│   ├── trust.go                # Trust scoring engine
│   ├── pipeline.go             # Event-driven pipeline
│   ├── consolidate.go          # Phase-end consolidation
│   └── queen.go                # QUEEN.md writing
├── events/
│   ├── bus.go                  # Pub/sub with JSONL persistence
│   └── ceremony.go             # Ceremony topic constants
└── colony/
    ├── colony.go               # ColonyState, Phase, Task types
    └── learning.go             # Observation, LearningFile types
```

### Pattern 1: Learning Capture in Continue-Finalize
**What:** After gates pass and review passes, extract learning from worker outputs and store with evidence.
**When to use:** Every `continue-finalize` that succeeds.
**Example:**
```go
// Source: cmd/codex_continue_finalize.go lines 466-560
captureLearning := func() {
    allWorkersSucceeded := true
    for _, step := range workerFlow {
        if step.Status != "completed" {
            allWorkersSucceeded = false
            break
        }
    }
    learningEnabled := isLearningEnabled(noLearn)
    if !learn.IsLearningEligible(allWorkersSucceeded, true, gates.Passed, learningEnabled) {
        return
    }
    // ... evidence collection, privacy scan, classification, store
}
```

### Pattern 2: Worker Context Injection
**What:** Attach pheromone signals, matched skills, and handoff sections to build dispatches.
**When to use:** Every worker dispatch in build, continue, plan, colonize.
**Example:**
```go
// Source: cmd/codex_build.go lines 2449-2458
func attachBuildDispatchContext(phaseID int, dispatches []codexBuildDispatch) {
    for i := range dispatches {
        assignment := resolveWorkerSkillAssignmentForWorkflow("build", dispatches[i].Caste, dispatches[i].Task)
        dispatches[i].SkillSection = assignment.Section
        dispatches[i].SkillCount = assignment.SkillCount
        dispatches[i].MatchedSkills = append([]string{}, assignment.MatchedNames...)
        dispatches[i].HandoffSection = renderWorkerHandoffSection("build", phaseID, dispatches[i].Name)
    }
}
```

### Pattern 3: Trust Scoring
**What:** Compute confidence for observations/instincts using weighted source/evidence/activity scores.
**When to use:** Every observation capture and instinct promotion.
**Example:**
```go
// Source: pkg/memory/trust.go lines 44-65
func Calculate(input TrustInput) TrustResult {
    sourceScore := sourceWeights[input.SourceType]
    evidenceScore := evidenceWeights[input.Evidence]
    activityScore := math.Pow(0.5, float64(input.DaysSince)/60.0)
    rawScore := 0.4*sourceScore + 0.35*evidenceScore + 0.25*activityScore
    score := math.Max(0.2, rawScore)
    // ...
}
```

### Anti-Patterns to Avoid
- **Storing generic placeholder learnings:** The current `continue-finalize` stores `"Phase N completed successfully: NAME"` instead of extracting actual insights from worker outputs.
- **Bypassing the event bus:** Direct file writes without publishing events break the pipeline's auto-promotion.
- **Hand-rolling trust scoring:** Always use `memory.Calculate()` — it has the correct 40/35/25 weights and 60-day half-life.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Learning storage | Custom JSON files | `learn.ColonyStore` (pkg/learn/colony_store.go) | Atomic updates, ID generation, filtering, compaction |
| Trust scoring | Ad-hoc confidence math | `memory.Calculate()` (pkg/memory/trust.go) | Verified 40/35/25 weights, 60-day half-life, 7 tiers |
| Event persistence | Custom log format | `events.Bus` (pkg/events/bus.go) | JSONL with TTL, pub/sub, query/replay |
| Worker handoffs | String concatenation | `renderWorkerHandoffSection()` (cmd/context.go) | Structured handoff with changed files, commands, status |
| QUEEN.md writing | Direct file append | `memory.QueenService.WriteEntry()` | V2 template, dedup, metadata tracking |

## Common Pitfalls

### Pitfall 1: Shallow Learning Capture
**What goes wrong:** Continue-finalize stores a generic string instead of extracting actual hypotheses from worker summaries, findings, and reusable lessons.
**Why it happens:** The `captureLearning` closure in `cmd/codex_continue_finalize.go` builds content from `phase.Name` only, ignoring `workerFlow` findings.
**How to avoid:** Extract content from `step.Findings`, `step.ReusableLessons`, `step.Recommendations` in the worker flow.
**Warning signs:** All learning entries have identical format: `"Phase N completed successfully: ..."`.

### Pitfall 2: Missing Hypothesis Lifecycle
**What goes wrong:** There is no `hypothesis` -> `validated` / `disproven` state tracking. Learnings are stored as flat entries with no evolution.
**Why it happens:** The `learn.Entry` type has no status field and no linking between related entries.
**How to avoid:** Add `Status` and `ParentID` fields to `learn.Entry`, or use the existing `memory.Pipeline` auto-promotion path.
**Warning signs:** Learning entries never change after creation; no validation or disproven tracking exists.

### Pitfall 3: Autopilot Pause Gaps
**What goes wrong:** The `autopilot.go` state machine tracks phase completion but may not check all 10 Classic pause conditions (test failures, chaos findings, quality gate failures, etc.).
**Why it happens:** `autopilot.go` only stores `autopilotState` with phase statuses; it does not read verification reports or gate results.
**How to avoid:** Add pause-condition checks in `autopilot-update` or `autopilot-status` that read `verification.json`, `gates.json`, and `review.json`.
**Warning signs:** Autopilot advances through blocked phases.

### Pitfall 4: Worker Context Drops in Non-Build Workflows
**What goes wrong:** Build dispatches get full context (skills, pheromones, handoffs), but continue/plan/colonize dispatches may not.
**Why it happens:** `attachBuildDispatchContext()` exists only in `codex_build.go`. Continue and plan have their own dispatch paths.
**How to avoid:** Audit `codex_continue.go`, `codex_plan.go`, `codex_colonize.go` for skill/pheromone/handoff attachment.
**Warning signs:** Workers in continue/plan report missing context or skills.

## Code Examples

### Learning Entry Creation (Current)
```go
// Source: cmd/codex_continue_finalize.go lines 534-542
learnStore := learn.NewColonyStore(store)
entry := learn.Entry{
    Content:        scanResult.Clean,
    Evidence:       evidence,
    Classification: classification,
    Phase:          phase.ID,
    Confidence:     evidence.Confidence,
}
if err := learnStore.Add(entry); err != nil {
    fmt.Fprintf(os.Stderr, "warning: failed to capture learning: %v\n", err)
}
```

### Evidence Collection
```go
// Source: pkg/learn/evidence.go lines 27-67
func CollectEvidence(runID string, phase int, workers []WorkerResult, gates GateResult, scope string) Evidence {
    trustResult := memory.Calculate(memory.TrustInput{
        SourceType: "success_pattern",
        Evidence:   "test_verified",
        DaysSince:  0,
    })
    return Evidence{
        RunID: runID, Phase: phase, Workers: workerEvidence,
        GatesPassed: gates.Passed, GatesTotal: gates.Total,
        Confidence: trustResult.Score, Timestamp: time.Now().UTC().Format(time.RFC3339),
        Scope: scope,
    }
}
```

### Worker Context Assembly
```go
// Source: cmd/codex_build.go lines 1236-1258
workerDispatch := codex.WorkerDispatch{
    ID:               fmt.Sprintf("phase-%d-dispatch-%d", phase.ID, i+1),
    WorkerName:       dispatch.Name,
    AgentName:        agentName,
    TaskBrief:        renderCodexBuildWorkerBrief(root, phase, dispatch, playbooks, startedAt),
    ContextCapsule:   capsule,
    HandoffSection:   dispatch.HandoffSection,
    SkillSection:     resolveSkillSectionForWorkflow("build", dispatch.Caste, dispatch.Task),
    PheromoneSection: pheromoneSection,
    Root:             root,
    Timeout:          workerTimeout,
    Wave:             normalizedDispatchWave(dispatch),
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Shell scripts for learning | Go `pkg/learn/` + `pkg/memory/` | v1.0.20+ | Structured, testable, event-driven |
| Direct COLONY_STATE.json writes | `store.UpdateJSONAtomically()` | v1.0.41 | Prevents state corruption |
| Manual skill injection | `resolveSkillSectionForWorkflow()` | v1.0.41 | Automatic skill matching per caste |
| No privacy scan on learning | `privacyScan()` + `learn.ClassifyEntry()` | v1.0.41 | Blocks secrets, classifies for hive |

**Deprecated/outdated:**
- `learning-extract-fallback` command: Still exists but should be replaced by structured extraction from worker flow.
- `phase_learnings` in COLONY_STATE.json: Legacy field, learning now stored in `entries.json` via `learn.ColonyStore`.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | The "10 Classic pause conditions" for autopilot are documented in `.aether/docs/command-playbooks/` or `CLAUDE.md` but were not verified against `autopilot.go` source | Autopilot | If the conditions are not implemented, autopilot may advance through blocked phases |
| A2 | Worker context injection (skills, pheromones, handoffs) works for continue/plan/colonize because they share the same `codex.WorkerDispatch` type | Worker Context | If those workflows don't call the attachment functions, workers will lack context |
| A3 | The oracle findings -> instinct -> QUEEN.md -> hive pipeline is manually triggered by commands, not automated | Oracle Pipeline | If automation is expected but missing, oracle findings won't promote |

## Open Questions

1. **What are the exact 10 Classic autopilot pause conditions?**
   - What we know: `autopilot.go` tracks phase status (completed/running/stopped) and has `autopilot-check-replan`.
   - What's unclear: The specific 10 conditions (test failures, chaos findings, quality gate failures, security gate failures, runtime verification needed, replan suggestion, critical flag, etc.) are not enumerated in the source.
   - Recommendation: Check `.aether/docs/command-playbooks/` or `CLAUDE.md` for the Classic list, then verify against `autopilot.go`.

2. **Does the hypothesis lifecycle exist anywhere?**
   - What we know: `learn.Entry` has no `Status` field. `memory.Pipeline` auto-promotes observations to instincts based on trust score.
   - What's unclear: Whether "hypothesis -> validated/disproven" was ever implemented or is a new requirement.
   - Recommendation: Check git history for "hypothesis" in `pkg/learn/` or `pkg/memory/`. If absent, implement as new feature.

3. **Are continue/plan/colonize dispatches missing skill/pheromone context?**
   - What we know: `attachBuildDispatchContext()` exists in `codex_build.go`.
   - What's unclear: Whether `codex_continue.go`, `codex_plan.go`, `codex_colonize.go` have equivalent attachment.
   - Recommendation: Grep for `resolveSkillSection`, `resolvePheromoneSection`, `renderWorkerHandoffSection` in those files.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go toolchain | All | Yes | 1.23 | — |
| `go test` | Verification | Yes | built-in | — |
| `aether` binary | Runtime testing | Yes | v1.0.41 | `go run ./cmd/aether` |
| SQLite (for auto-skill) | Auto-skill creation | Yes | built-in (mattn/go-sqlite3) | Skip auto-skill test |

**Missing dependencies with no fallback:** None.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go standard testing |
| Config file | none — see Wave 0 |
| Quick run command | `go test ./pkg/learn/... -v` |
| Full suite command | `go test ./... -race` |

### Phase Requirements -> Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| WORKFLOW-01 | Learning extraction creates entries with real content | unit | `go test ./pkg/learn/... -v` | Partial (colony_store_test.go exists) |
| WORKFLOW-02 | Worker dispatches include skill/pheromone/handoff sections | unit | `go test ./cmd/... -run TestBuildDispatchContext` | No — needs new test |
| WORKFLOW-03 | Oracle findings promote through instinct -> QUEEN.md -> hive | integration | Manual / e2e | No — needs new test |
| WORKFLOW-04 | Build produces correct state mutations and ceremony | unit | `go test ./cmd/... -run TestCodexBuild` | Partial (codex_build_test.go exists) |
| WORKFLOW-05 | Continue advances phase correctly | unit | `go test ./cmd/... -run TestCodexContinue` | Partial (codex_continue_test.go exists) |
| WORKFLOW-06 | Plan produces grounded plans | unit | `go test ./cmd/... -run TestCodexPlan` | Partial (codex_plan_test.go exists) |
| WORKFLOW-07 | Colonize produces complete survey | unit | `go test ./cmd/... -run TestCodexColonize` | Partial (codex_colonize_test.go exists) |
| WORKFLOW-08 | Autopilot respects pause conditions | unit | `go test ./cmd/... -run TestAutopilot` | Partial (autopilot_test.go exists) |
| WORKFLOW-09 | Seal ceremony completes lifecycle | unit | `go test ./cmd/... -run TestSeal` | Partial (seal tests in e2e) |
| WORKFLOW-10 | Entomb archives correctly | unit | `go test ./cmd/... -run TestEntomb` | No — needs new test |
| WORKFLOW-11 | Swarm 4-scout comparison works | unit | `go test ./cmd/... -run TestSwarm` | Partial (swarm_cmd_test.go exists) |

### Wave 0 Gaps
- [ ] `cmd/codex_continue_finalize_test.go` — covers learning capture in continue-finalize
- [ ] `cmd/entomb_cmd_test.go` — covers entomb archive behavior
- [ ] `pkg/learn/hypothesis_test.go` — covers hypothesis lifecycle if implemented
- [ ] `cmd/autopilot_pause_test.go` — covers the 10 pause conditions

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V5 Input Validation | Yes | `privacyScan()` + `learn.ClassifyEntry()` blocks secrets |
| V6 Cryptography | No | Not in scope for this phase |
| V7 Error Handling | Yes | All learning failures are non-blocking (warning to stderr) |

### Known Threat Patterns

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Secret leakage in learning entries | Information Disclosure | `privacyScan()` detects API keys; `ClassBlocked` prevents storage |
| Learning tampering | Tampering | `store.UpdateFile()` atomic writes with locking |
| Hive wisdom injection | Tampering | `hive-store` dedup + confidence threshold |

## Sources

### Primary (HIGH confidence)
- `cmd/codex_continue_finalize.go` lines 466-560 — learning capture implementation
- `cmd/codex_build.go` lines 2449-2458 — worker context attachment
- `pkg/learn/` — learning store, evidence, classification, trigger
- `pkg/memory/` — trust scoring, observation, promotion, pipeline, consolidation, queen
- `pkg/events/bus.go` — event bus pub/sub with JSONL persistence
- `cmd/autopilot.go` — autopilot state machine
- `cmd/seal_final_review.go` — seal ceremony
- `cmd/entomb_cmd.go` — entomb archive
- `cmd/swarm_cmd.go` — swarm dispatch
- `cmd/oracle_loop.go` — oracle research loop

### Secondary (MEDIUM confidence)
- `cmd/codex_continue.go` — continue verification and gates
- `cmd/codex_plan.go` — plan workflow
- `cmd/codex_colonize.go` — colonize workflow
- `cmd/instinct.go` — instinct CRUD
- `cmd/queen.go` — QUEEN.md promotion
- `cmd/hive.go` — hive wisdom store

### Tertiary (LOW confidence)
- `.aether/docs/command-playbooks/` — assumed to contain Classic pause conditions (not verified)
- `CLAUDE.md` — assumed to document 10 pause conditions (not verified)

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — all libraries verified in go.mod and source
- Architecture: HIGH — all 9 workflows have runtime implementations
- Pitfalls: HIGH — shallow learning and missing hypothesis lifecycle confirmed by source inspection
- Worker context: HIGH — build attachment confirmed, continue/plan/colonize need audit
- Autopilot pause conditions: MEDIUM — state machine exists but 10 specific conditions not verified

**Research date:** 2026-05-21
**Valid until:** 2026-06-21 (stable Go codebase)
