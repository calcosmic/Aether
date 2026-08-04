# Phase 162: Switch On Learning - Research

**Researched:** 2026-08-04
**Domain:** Go CLI runtime wiring — invoking two existing consolidation subcommands from the lifecycle runtime, reconciling two same-named-but-different Go packages, and flipping a policy default
**Confidence:** HIGH (all claims below are `[VERIFIED: repo read]` against current `cmd/`, `pkg/memory/`, `pkg/learn/`, `pkg/agent/curation/` — this is a wiring phase, not a library-adoption phase, so there is no external ecosystem to research)

## Summary

This phase has no external dependency surface — it is 100% invoking, reconciling, and documenting code that already exists in this repository. All research below is direct code reading, not framework research. The two subcommands (`consolidation-phase-end`, `consolidation-seal`) are fully implemented, tested for dry-run purity, and simply uncalled. The exact insertion points for D-04 exist and are unambiguous: both continue paths already have a single "phase just durably advanced" moment (right after `COLONY_STATE.json` is atomically written with `PhaseCompleted`), and it already hosts a non-blocking learning call (`captureContinueLearning`) that this phase's consolidation call should sit beside. The seal path has an existing, independent instinct-promotion loop (`completeSealRuntime`, `cmd/codex_workflow_cmds.go:409`) that duplicates part of what `consolidation-seal` would do — this is the single most important reconciliation finding of this research and directly extends D-09's "genuinely duplicated stage" test.

The `pkg/learn` vs `pkg/memory` question research turned up a wrinkle CONTEXT.md's canonical refs did not fully separate: `pkg/learn` is two different things under one name. `pkg/learn/wrappers.go` is a **thin forwarding shim** (`learn.NewPipeline` → `memory.NewPipeline`, `learn.ConsolidationResult = memory.ConsolidationResult`) that `cmd/graph_consolidation_cmds.go` already uses to call `pkg/memory` — it is not a competing implementation, it is a rename-avoidance import-cycle workaround, and D-09 does not need to "retire" it. The actual competing system is `pkg/learn/learn.go` + `curator.go` + `colony_store.go` (Entry/Evidence/Classification, written to `entries.json` and `colony.db`), which `captureContinueLearning` (`cmd/codex_continue_finalize.go:1274`, called from both continue paths) already runs live and which already feeds a `## LEARNED MEMORY` section into worker context. `pkg/memory`'s `LearningValidator` bridge field exists in `PipelineConfig` but is never populated by `pipelineConfigForStore()` (`cmd/graph_consolidation_cmds.go:15`) — the bridge CONTEXT.md's canonical refs describe as "already anticipating this relationship" is wired in the struct but not connected at the call site.

Hive gating is a genuine double-gate today only for **retrieval** (`AETHER_HIVE_POLICY` policy AND a per-colony consent file), not for **promotion** (seal's promotion loop only checks the policy). D-02 retires the consent file; three call sites and one shared test helper depend on it and must all change together.

**Primary recommendation:** Insert one non-blocking `runPhaseEndConsolidation(...)` call directly after the atomic `PhaseCompleted` write in both `cmd/codex_continue.go` (~line 973, beside the existing `captureContinueLearning` call) and `cmd/codex_continue_finalize.go` (~line 489); insert one non-blocking `consolidation-seal`-equivalent call inside `completeSealRuntime` (`cmd/codex_workflow_cmds.go:409`), and either replace or explicitly keep-alongside its existing ad hoc `promoteInstinctLocal`/`promoteToHiveWithReference` loop — do not let both run unreconciled. Flip `currentHiveRuntimePolicy()`'s default branch (`cmd/hive_policy.go:24-26`) to `hivePolicyPromote`, add an explicit `"off"` case, and delete the `hiveRetrievalOptedIn()` gate at its three call sites plus the `hive-opt-in`/`hive-opt-out` commands and consent-file plumbing. Target `resolveCodexWorkerContext()` → `buildColonyPrimeOutput(true).PromptSection` (`cmd/colony_prime_context.go:962`) as the D-08 Layer-1 assertion point — it is already the single funnel every dispatch path uses.

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Phase-end consolidation trigger | API/Backend (Go runtime, `cmd/codex_continue*.go`) | — | Runtime owns state transitions; D-04 forbids wrapper-instructed invocation |
| Seal consolidation trigger (8-ant pass) | API/Backend (Go runtime, `cmd/codex_workflow_cmds.go`) | — | Same rule; seal is a Go-owned ceremony function already |
| Consolidation pipeline itself (decay/archive/promote) | API/Backend (`pkg/memory`) | — | Already implemented, pure Go, no UI concern |
| Curation ants (8-ant pass) | API/Backend (`pkg/agent/curation`) | — | Already implemented |
| Live learning capture (`captureContinueLearning`) | API/Backend (`cmd/`, `pkg/learn`) | Storage (`entries.json`, `colony.db`) | Existing capture front-end; D-09 keeps it subordinate |
| Hive policy default | API/Backend (`cmd/hive_policy.go`) | — | Single env-var-driven control surface, no UI |
| Worker context injection (instincts/hive/learned-memory sections) | API/Backend (`cmd/colony_prime_context.go`) | — | Feeds text into worker prompts; still Go-rendered, not wrapper-rendered |
| Ceremony rendering (🧠 beat, 8-ant announcements) | API/Backend (`cmd/codex_visuals.go`, `cmd/ceremony_emitter.go`) | CLI/stdout | Runtime-owned per wrapper-runtime contract; wrappers narrate only |
| Documentation truth (CLAUDE.md, AGENTS.md, structural-learning-stack.md) | Docs | — | No runtime tier; must track the above once wired |

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| LEARN-01 | `consolidation-phase-end` runs at the end of every phase | Exact insertion points identified in both continue paths (§Code Examples, §Architecture Patterns); non-blocking pattern already exists to copy (`captureContinueLearning`) |
| LEARN-02 | `consolidation-seal` runs at seal — full 8-ant pass, decay, archive, scribe report | Seal path located (`completeSealRuntime`); existing overlapping promotion loop identified and must be reconciled, not just left running alongside a new call |
| LEARN-03 | Two competing learning systems reconciled, one authoritative | `pkg/learn/wrappers.go` (shim, not competitor) vs `pkg/learn/learn.go` (real competitor) distinguished; `LearningValidator` bridge found unwired; QUEEN.md promotion duplication (different sections, different formats, different eligibility thresholds) found between `completeSealRuntime`'s loop and `pkg/memory.QueenService.PromoteInstinct` |
| LEARN-04 | Hive Brain default decided deliberately, documented honestly | Exact default-flip line found (`cmd/hive_policy.go:24-26`); all 3 consent-gate call sites + 1 shared test helper + 2 cobra commands + AGENTS.md's "opt-in, twice over" section enumerated |
| LEARN-05 | Worker output demonstrably differs, memory populated vs wiped | Exact assembly funnel found (`resolveCodexWorkerContext` → `buildColonyPrimeOutput`); existing test pattern to copy identified (`TestBuildWorkerBriefIncludesSurveyAndResearch`) |
</phase_requirements>

## Standard Stack

Not applicable in the conventional sense — this phase adds zero new dependencies. All work is in the existing Go module (`github.com/calcosmic/Aether`), existing packages (`pkg/memory`, `pkg/learn`, `pkg/agent/curation`, `cmd/`). `go.mod` requires no changes.

**Version verification:** N/A — no new packages.

## Architecture Patterns

### System Architecture Diagram

```
                         DEFAULT CONTINUE PATH                    EXTERNAL/CODEX CONTINUE PATH
                    cmd/codex_continue.go:~900-975          cmd/codex_continue_finalize.go:459-497
                              │                                              │
              verification + gates + review all pass          verification + gates + review all pass
                              │                                              │
              store.UpdateJSONAtomically("COLONY_STATE.json")  store.UpdateJSONAtomically(...) [inside advanceExternalContinue]
              → Plan.Phases[idx].Status = PhaseCompleted        → same, called from advanceExternalContinue
                              │                                              │
              captureContinueLearning(...)  ◄── existing        captureContinueLearning(...)  ◄── existing
              (pkg/learn Entry capture, non-blocking)            (same, non-blocking)
                              │                                              │
              ┌───────────────▼───────────────┐            ┌────────────────▼────────────────┐
              │ NEW (LEARN-01, D-04):          │            │ NEW (LEARN-01, D-04):            │
              │ consolidation-phase-end call   │            │ consolidation-phase-end call     │
              │ (non-blocking, D-05)           │            │ (non-blocking, D-05)             │
              └───────────────┬───────────────┘            └────────────────┬────────────────┘
                              │                                              │
                    emitContinueCeremonyFlowSequence            advanceExternalContinue(...)
                    (existing ceremony emitter — NEW             (calls the same emitter)
                    🧠 learning beat joins this, D-06/D-07)
                              │
                       "aether build N+1" or "aether seal"


                                   SEAL PATH
                        cmd/codex_workflow_cmds.go:409
                          completeSealRuntime(state)
                              │
        ┌─────────────────────┴──────────────────────┐
        │ EXISTING Ceremony Step 1 (lines 425-454):   │
        │ loadActiveInstinctEntriesFromStore()        │
        │ → for confidence>=0.8 && Action!="":        │
        │     promoteInstinctLocal() → QUEEN.md        │
        │     "Wisdom" section                         │
        │   + promoteToHiveWithReference() if          │
        │     automaticHivePromotionEnabled()          │
        │                                              │
        │ ⚠ OVERLAPS with what consolidation-seal's   │
        │   pipeline.RunConsolidation() would ALSO do  │
        │   via Queen.PromoteInstinct() → QUEEN.md     │
        │   "Instincts" section (different format,     │
        │   different threshold: >=0.75 AND            │
        │   applications>=3, not >=0.8)                │
        └─────────────────────┬──────────────────────┘
                              │
              ┌────────────────▼────────────────┐
              │ NEW (LEARN-02, D-04):           │
              │ consolidation-seal equivalent   │
              │ call: curation.Orchestrator.Run │
              │ (8 ants) → pipeline.Run          │
              │ Consolidation (decay/archive/   │
              │ promote) → publish seal event   │
              │ RECONCILE with Step 1 above     │
              │ (D-09: pick one winner)         │
              │ Render 8 named ant beats, D-06  │
              └────────────────┬────────────────┘
                              │
                    rest of completeSealRuntime
                    (FOCUS expiry, shelf detection,
                    CROWNED-ANTHILL.md enrichment)


                    WORKER CONTEXT INJECTION (read-side, all dispatch paths)
        cmd/{codex_build,codex_continue,codex_colonize,codex_plan,swarm_cmd,
             seal_final_review,build_print_brief,internal_worker_adapter}.go
                              │
              resolveCodexWorkerContext()  (cmd/colony_prime_context.go:962)
                              │
              buildColonyPrimeOutput(true).PromptSection
                              │
        ┌─────────────────────┼──────────────────────┐
        │                     │                       │
  "## Active Instincts"  "## HIVE WISDOM       "## LEARNED MEMORY
  reads instincts.json    (Cross-Colony)"       (Verified Outcomes)"
  UNCONDITIONALLY         readHiveWisdomEntries  learn.ColonyStore.List()
  (no policy gate)        ForDomains() — gated   (pkg/learn Entry system)
                           on automaticHiveRead   UNCONDITIONAL
                           Enabled() + (until     (min confidence 0.3)
                           D-02) hiveRetrieval
                           OptedIn()

  ◄── D-08 Layer-1 test target: call buildColonyPrimeOutput(true) with
      instincts.json/hive wisdom.json populated vs. absent/empty, assert
      PromptSection contains/excludes the text.
```

### Recommended Project Structure

No new files/directories are required by shape — this is glue code inside existing files. Suggested new files (Claude's discretion per CONTEXT.md, but consistent with existing naming):

```
cmd/
├── consolidation_lifecycle.go      # NEW: shared non-blocking wrapper functions
│                                    #   runPhaseEndConsolidation(phaseID int) — called from both continue paths
│                                    #   runSealConsolidation() (curResult, consResult, error) — called from completeSealRuntime
├── consolidation_lifecycle_test.go # NEW: wiring regression tests (LOCK-01-shaped, scoped to this phase's own connections)
├── colony_prime_context_hive_default_test.go  # NEW or extend colony_prime_context_test.go: D-08 Layer 1
.aether/docs/
├── learn-vs-memory-decision.md     # NEW (D-10): ADR-style decision record, or fold into structural-learning-stack.md
```

### Pattern 1: Non-blocking runtime side effect with unmissable warning (D-05)
**What:** A side-effect call that must never abort the caller's success path, but must print a loud warning on failure — the pattern this codebase already uses for hive promotion and for `captureContinueLearning`.
**When to use:** Both the phase-end and seal consolidation calls.
**Example (existing precedent to copy, `cmd/codex_workflow_cmds.go:446-451`):**
```go
// Source: cmd/codex_workflow_cmds.go (existing hive promotion loop)
if err := promoteToHiveWithReference(entry.Action, domain, repoName, entry.Confidence, "seal:"+entry.ID); err != nil {
    log.Printf("seal: hive-promote failed for %s: %v", entry.ID, err)
    hivePromotionFailures++
} else {
    hivePromotedCount++
}
```
**Example (existing precedent for non-blocking capture, `cmd/codex_continue_finalize.go:1345-1347`):**
```go
// Source: cmd/codex_continue_finalize.go (existing learning capture)
if err := learnStore.Add(entry); err != nil {
    // Non-blocking: learning failure must not prevent phase advancement
    fmt.Fprintf(os.Stderr, "warning: failed to capture learning: %v\n", err)
}
```
D-05 requires the message to state unmissably "phase advanced WITHOUT consolidation — <reason>" — follow this exact shape but strengthen the wording per D-05's literal text.

### Pattern 2: Dry-run purity via a separate no-write service, never a flag inside the mutating path
**What:** `NewDryRunConsolidationService` (`pkg/memory/consolidate.go:55`, forwarded via `pkg/learn/wrappers.go:100`) is a **structurally distinct** service, not `RunConsolidation` with an `if dryRun { skip writes }` branch inside it.
**When to use:** If the plan adds a `--dry-run` surface to any new wrapper function, it MUST route to `NewDryRunConsolidationService`/`pipeline.RunConsolidation` exactly as `cmd/graph_consolidation_cmds.go:226-234` already does — copy this branching, do not reinvent it.
**Example (existing code, already correct — do not modify unless extending it):**
```go
// Source: cmd/graph_consolidation_cmds.go:226-234
if dryRun {
    service := learn.NewDryRunConsolidationService(store, bus, "QUEEN.md", pipelineConfigForStore().ColonyName)
    result, err = service.Run(ctx)
} else {
    result, err = pipeline.RunConsolidation(ctx)
}
```
**Constraint:** `TestConsolidationPhaseEndDryRunDoesNotMutate` / `TestConsolidationSealDryRunDoesNotMutate` (`cmd/consolidation_dryrun_test.go`) pin this. Any new call site that invokes `consolidation-phase-end`/`consolidation-seal` from the runtime must call them **without** `--dry-run` (the real path) — this phase is explicitly "use the real path deliberately" per CONTEXT.md's Established Patterns.

### Pattern 3: Ceremony emission via `events.CeremonyTopic*` + `emitLifecycleCeremony`/`emitLifecycleCeremonySequence`
**What:** The existing mechanism for rendering caste-styled beats during continue/build.
**Example (existing pattern to extend for D-06's 🧠 beat and the 8 named ant beats, `cmd/ceremony_emitter.go:304-327`):**
```go
// Source: cmd/ceremony_emitter.go
func emitContinueCeremonyFlowSequence(source string, phase colony.Phase, workerFlow []codexContinueWorkerFlowStep) {
	steps := make([]lifecycleCeremonyStep, 0, len(workerFlow))
	for idx, step := range workerFlow {
		steps = append(steps, lifecycleCeremonyStep{
			Caste: step.Caste, Name: step.Name, Status: step.Status, Message: step.Summary,
			/* ... */
		})
	}
	emitLifecycleCeremonySequence(lifecycleCeremonyTopics{ /* topic constants */ }, source, "continue", phase.ID, phase.Name, steps)
}
```
For the 8-ant seal announcements, the data is already available per-ant as `curation.StepResult{Name, Success, Summary map[string]any}` from `orch.Run(ctx, dryRun)` (`pkg/agent/curation/orchestrator.go:140-197`) — `consolidationSealCmd` currently **discards** this into one aggregate string (`cmd/graph_consolidation_cmds.go:298-303`, `Summary: fmt.Sprintf("succeeded=%d failed=%d", ...)`). The seal wiring must either call the orchestrator directly (bypassing the CLI subcommand's aggregation) or extend `consolidation-seal`'s JSON output to include the per-step `Steps` array so the ceremony renderer can consume it.

### Pattern 4: Caste identity fallback for unmapped names
**What:** `casteEmoji()`/`casteLabel()`/`casteANSIColor()` (`cmd/codex_visuals.go:3555-3591`) look up `casteEmojiMap`/`casteColorMap`/`casteLabelMap` by a normalized key and fall back to `"🐜"`/`"Ant"` (no color) if the key is absent.
**Finding:** None of the 9 curation-ant names (`sentinel`, `nurse`, `critic`, `herald`, `janitor`, `archivist`, `librarian`, `scribe`, and the orchestrator itself) exist as keys in any of the three maps today — they will all render as generic "🐜 Ant" until entries are added.
**What to do:** Add 8-9 new entries to `casteEmojiMap`/`casteColorMap`/`casteLabelMap` (exact emoji/color/label choices are Claude's discretion per CONTEXT.md) so D-06's "each of the eight curation ants announces with emoji + name" produces distinct identities rather than 8 identical "🐜 Ant" lines.

### Anti-Patterns to Avoid
- **Wrapper-instructed consolidation invocation:** Do not add a step to `.claude/commands/ant/continue.md` or `.opencode/commands/ant/*.md` instructing the model to run `aether consolidation-phase-end`. D-04 and Phase 165's ownership of `continue.md`/`build.md` (CONTEXT.md's Prior phase constraints) both forbid this — it must be a Go call inside `cmd/codex_continue*.go` / `cmd/codex_workflow_cmds.go`.
- **Letting both promotion loops run unreconciled:** Do not simply add a call to `consolidation-seal`'s pipeline inside `completeSealRuntime` while leaving the existing `promoteInstinctLocal`/`promoteToHiveWithReference` loop (lines 425-454) untouched. They write to different QUEEN.md sections with different formats and different eligibility thresholds (0.8+Action!="" vs 0.75+applications>=3) for what is conceptually the same event. Decide which wins per D-09's rule ("if a stage is genuinely implemented in both, pkg/memory's implementation wins and the duplicate is retired") and document the decision in the D-10 record.
- **Testing dry-run purity by watching the wrong directory:** `cmd/consolidation_dryrun_test.go`'s own comment (lines 37-41) documents a real historical bug — the test store writes under `<tmp>/.aether/data`, not the tmp root; watching the tmp root makes purity assertions pass vacuously. Any new test in this area must use `store.BasePath()`, exactly as `seedConsolidationFixture` does.
- **Retiring `pkg/learn/wrappers.go` under a mistaken belief it's the "competing" system:** It is a thin forwarding shim to `pkg/memory` used by the two consolidation subcommands themselves. The real subordinate system is `pkg/learn/learn.go`/`curator.go`/`colony_store.go` (the Entry/Evidence/hypothesis capture layer `captureContinueLearning` uses). Do not delete or rename `wrappers.go`'s exports — `cmd/graph_consolidation_cmds.go` depends on them directly.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Decay/archive/promotion-candidate logic | A new consolidation algorithm | `pkg/memory/consolidate.go`'s `ConsolidationService.Run()` (via `consolidation-phase-end`/`consolidation-seal`) | Already implemented, already has dry-run purity tests, already computes `QueenEligible`/`ReviewCandidates`/`RereadCandidates` |
| 8-ant curation pass | Sequential curation logic in `cmd/` | `pkg/agent/curation.NewOrchestrator(store, bus).Run(ctx, dryRun)` | Already implements the fixed order (sentinel→nurse→critic→herald→janitor→archivist→librarian→scribe) with sentinel-abort semantics |
| Instinct-to-QUEEN.md promotion | A second ad hoc promotion writer | `pkg/memory.QueenService.PromoteInstinct` (`pkg/memory/queen.go:157`) — **but reconcile with the existing `completeSealRuntime` loop first (D-09)** | Two writers to two different QUEEN.md sections already exist; adding a third makes it worse |
| Hive policy gate | A new env var or config key | `cmd/hive_policy.go`'s existing `AETHER_HIVE_POLICY` three-value enum | Single control surface is D-02's explicit goal; do not add a second knob |
| Ceremony rendering | Custom `fmt.Println` banners in the seal/continue functions | `cmd/ceremony_emitter.go`'s `emitLifecycleCeremony`/`emitLifecycleCeremonySequence` + `cmd/codex_visuals.go`'s `casteIdentity()` | Matches wrapper-runtime UX contract; keeps caste color/emoji consistent across the whole ceremony system |

**Key insight:** Every primitive this phase needs (decay, archive, promotion, 8-ant orchestration, ceremony rendering, caste identity, non-blocking-warn pattern) already exists in the codebase in working, tested form. The engineering content of this phase is entirely in the *connective tissue* (where to call, what to reconcile, what default to flip) — treat any task that proposes writing new consolidation/promotion/decay logic as a signal something has been misunderstood.

## Common Pitfalls

### Pitfall 1: Assuming `consolidation-phase-end` runs the curation ants
**What goes wrong:** `.aether/docs/structural-learning-stack.md:211-216` currently states phase-end mode "Executes three ants only: `nurse → herald → janitor`." This is **false today, independent of Phase 162's wiring** — `consolidationPhaseEndCmd` (`cmd/graph_consolidation_cmds.go:204-267`) calls `pipeline.RunConsolidation(ctx)`, which calls `pkg/memory/consolidate.go`'s `ConsolidationService.Run()`, a monolithic algorithmic implementation with **zero references to `pkg/agent/curation`** (verified: `grep -n "nurse\|herald\|janitor" pkg/memory/consolidate.go` returns nothing). Only `consolidation-seal` touches the curation ants, via `curation.NewOrchestrator(...).Run(ctx, dryRun)` as a separate first step.
**Why it happens:** The doc appears to describe an intended/earlier design where phase-end selectively ran a subset of ants; the implementation diverged.
**How to avoid:** Correct `structural-learning-stack.md`'s phase-end description as part of D-03/D-10's docs pass — don't just remove the "Phase 162 wires this" caveat, fix the ant-list claim underneath it too.
**Warning signs:** Any task description or plan text that says "phase-end runs nurse/herald/janitor" is working from the stale doc, not the code.

### Pitfall 2: Double-promoting instincts to QUEEN.md at seal
**What goes wrong:** If the plan adds a call to run full seal consolidation (which internally calls `Queen.PromoteInstinct` for `QueenEligible` instincts) without touching `completeSealRuntime`'s existing Step 1 loop (lines 425-454, `promoteInstinctLocal`/`promoteToHiveWithReference`), the same instinct can be written to QUEEN.md twice, in two different formats, under two different section headers ("Instincts" vs "Wisdom"), using two different eligibility thresholds.
**Why it happens:** The two code paths were built independently — `completeSealRuntime`'s loop predates this phase and was never designed to compose with `consolidation-seal`.
**How to avoid:** Decide explicitly (and record via D-10) whether `completeSealRuntime`'s loop is retired in favor of `consolidation-seal`'s pipeline, or whether the two are scoped to non-overlapping instinct sets. Given D-09's stated rule, the default answer is: `pkg/memory`'s path wins; refactor `completeSealRuntime`'s Step 1 to be a thin ceremony-reporting wrapper around `consolidation-seal`'s already-computed `QueenEligible` results rather than a second independent instinct scan.
**Warning signs:** QUEEN.md gaining duplicate-looking entries for the same instinct ID under different headers after seal.

### Pitfall 3: Breaking the double-gated hive-read call sites incompletely
**What goes wrong:** `hiveRetrievalOptedIn()` is checked at 3 call sites (`cmd/hive.go:554`, `cmd/context_weighting.go:22`, `cmd/colony_prime_context.go:652`) plus referenced by 1 shared test helper (`cmd/hive_test_optin.go`'s `enableHiveForTest`) and documented explicitly in `AGENTS.md:694-706` ("Retrieval is opt-in, twice over"). Removing the gate at only some of these sites leaves an inconsistent half-retired mechanism — e.g., `hive-read --for-worker` still refusing entries while `colony_prime_context.go`'s injection succeeds, or vice versa.
**Why it happens:** The 3 call sites are in 3 different files with no shared helper function to change once.
**How to avoid:** Grep for all 4 identifiers (`hiveRetrievalOptedIn`, `writeHiveRetrievalConsent`, `hiveRetrievalConsentPath`, `hiveRetrievalConsentFile`) before starting and change every hit in the same task/commit. Six test files call `enableHiveForTest(t)` (`cmd/codex_prompt_context_test.go`, `cmd/colony_prime_context_test.go`, `cmd/colony_prime_audit_test.go`, `cmd/colony_prime_budget_test.go`, `cmd/hive_runtime_test.go`, `cmd/pheromone_loader_test.go`) — decide whether the helper keeps its name but drops the consent-file line, or is removed entirely in favor of `t.Setenv(hivePolicyEnv, "read")` alone (or is deleted, since promote is now default and tests may not need to opt in at all).
**Warning signs:** `go test ./cmd/... -run TestColonyPrime` failing after the hive_policy.go default change but before the gate-removal changes land (or vice versa) — these two halves must land together or CI will show a real regression, not a false one.

### Pitfall 4: `currentHiveRuntimePolicy()`'s default-case rewrite silently changing the meaning of explicit `off`
**What goes wrong:** The current switch (`cmd/hive_policy.go:18-27`) has no explicit case for `"off"` — it relies on the `default:` branch to return `hivePolicyOff` for *both* unset and explicitly-set-to-`"off"` env values. If D-02's implementation naively changes only the `default:` return value to `hivePolicyPromote`, setting `AETHER_HIVE_POLICY=off` would stop working (it would fall into the new default and become `promote`).
**Why it happens:** The unset-vs-explicit-off distinction currently collapses into one code path.
**How to avoid:** Add an explicit `case "off":` returning `hivePolicyOff`, and change only the *implicit* unset path to `hivePolicyPromote`. Verify with a table test: `""` → promote, `"off"` → off, `"read"`/`"inject"`/`"on"` → read, `"promote"`/`"full"` → promote, and (decide) unrecognized garbage values → probably `promote` (matching new default) or `off` (fail-safe) — this is a small decision-shaped detail worth stating explicitly in the plan.
**Warning signs:** A test that sets `AETHER_HIVE_POLICY=off` and expects retrieval/promotion disabled starts failing after the default flip.

### Pitfall 5: Treating `LearningValidator` as already-wired
**What goes wrong:** `pkg/memory/pipeline.go:17-18` defines `LearningValidator` as a callback field anticipating a bridge to `pkg/learn`, and `handleObserveEvent` (`pkg/memory/pipeline.go:132-137`) already calls it when trust score >= 0.8. But `pipelineConfigForStore()` (`cmd/graph_consolidation_cmds.go:15-29`) — the only place `PipelineConfig`/`learn.PipelineConfig` is constructed for the two consolidation subcommands — never sets this field, so it is always `nil` and the bridge never fires.
**Why it happens:** The field was added in anticipation of the D-09 relationship but the call site was never updated.
**How to avoid:** Decide explicitly whether this phase wires `LearningValidator` (e.g., to something in `pkg/learn` that validates/promotes a matching hypothesis entry) or leaves it deliberately unwired with a documented reason. Silently leaving it `nil` while claiming "pkg/memory is authoritative, pkg/learn is subordinate" per D-09 is an incomplete implementation of that decision, not a wrong one — but it should be a stated choice, not an oversight.
**Warning signs:** Grep for `LearningValidator` returning only the struct definition and the `nil`-safe call site, never a real assignment.

## Code Examples

### Detecting "phase actually advanced" for D-04 (default path)
```go
// Source: cmd/codex_continue.go:900-975 (existing code, condensed to the relevant shape)
if err := store.UpdateJSONAtomically("COLONY_STATE.json", &updated, func() error {
    // ... validation ...
    updated.Plan.Phases[currentIdx].Status = colony.PhaseCompleted
    // ... final vs. next-phase branching ...
    return nil
}); err != nil {
    // ... error handling; if this branch is taken, phase did NOT advance ...
}
// Reaching here means the atomic write succeeded: the phase durably advanced.
// captureContinueLearning already fires exactly here (non-blocking).
// NEW: runPhaseEndConsolidation(phase.ID) belongs right beside it.
captureContinueLearning(phase, workerFlow, gates, "", false, now)
```

### Detecting "phase actually advanced" for D-04 (external/finalize path)
```go
// Source: cmd/codex_continue_finalize.go:483-491
// --- Learning capture (D-01, D-02, D-03, D-04) ---
// Shared with the default continue path; see captureContinueLearning.
runID := ""
if runHandle != nil {
    runID = runHandle.Run.ID
}
captureContinueLearning(phase, workerFlow, gates, runID, noLearn, now)
// advanceExternalContinue is only reached after verification+gates+review all
// passed; it performs the same atomic PhaseCompleted write as the default path.
result, updated, nextPhase, housekeeping, final, err := advanceExternalContinue(root, state, phase, manifest, verification, assessment, gates, review, reviewReportRel, watcherFlow, workerFlow, now, verificationReportRel, gateReportRel, finalizeReviewDepth)
```
Note: in this path, `PhaseCompleted` is actually written *inside* `advanceExternalContinue` (line 980), which runs *after* `captureContinueLearning`. Since reaching line 489 always implies `advanceExternalContinue` will be called and will succeed in normal operation (its own atomic-write error path returns early with an error, same shape as the default path), inserting the consolidation call either just before line 489 (alongside learning capture) or just after `advanceExternalContinue` returns without error both satisfy D-04 — the latter is more precisely "after advance," the former matches the existing `captureContinueLearning` precedent more closely. Recommend the latter for stricter correctness (only run consolidation if `advanceExternalContinue` truly returned success), unless the failure window between them is judged negligible enough that colocating with the existing call is preferred for symmetry with the default path.

### The seal path's existing (pre-Phase-162) promotion loop that must be reconciled
```go
// Source: cmd/codex_workflow_cmds.go:409-454 (completeSealRuntime, existing)
if entries, err := loadActiveInstinctEntriesFromStore(store); err == nil {
    for _, entry := range entries {
        if entry.Confidence >= 0.8 && entry.Action != "" {
            if err := promoteInstinctLocal(store, entry.ID, entry.Action); err == nil {
                promotedInstinctNames = append(promotedInstinctNames, entry.ID)
            }
        }
        if entry.Confidence >= 0.8 && entry.Action != "" {
            hiveEligibleCount++
            if !automaticHivePromotionEnabled() {
                continue
            }
            if err := promoteToHiveWithReference(entry.Action, domain, repoName, entry.Confidence, "seal:"+entry.ID); err != nil {
                log.Printf("seal: hive-promote failed for %s: %v", entry.ID, err)
                hivePromotionFailures++
            } else {
                hivePromotedCount++
            }
        }
    }
}
```
This uses `colony.InstinctEntry.Confidence >= 0.8 && Action != ""` as its eligibility bar and writes to QUEEN.md's "Wisdom" section via `cmd/queen.go:676-684`'s `promoteInstinctLocal`. `pkg/memory`'s consolidation pipeline uses `Confidence >= 0.75 && applications >= 3` (`pkg/memory/consolidate.go:149`) and writes to QUEEN.md's "Instincts" section via a different formatter (`pkg/memory/queen.go:157-161`). These must be reconciled per D-09, not run side by side unexamined.

### Hive policy default flip target
```go
// Source: cmd/hive_policy.go:18-27 (current)
func currentHiveRuntimePolicy() hiveRuntimePolicy {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(hivePolicyEnv))) {
	case "read", "inject", "on":
		return hivePolicyRead
	case "promote", "full":
		return hivePolicyPromote
	default:
		return hivePolicyOff // <-- D-02 target: unset should become hivePolicyPromote;
	}                          //     add an explicit case "off": return hivePolicyOff
}                              //     so explicit opt-out still works (Pitfall 4).
```

### The consent gate to retire (3 call sites)
```go
// Source: cmd/hive.go:554-563 (hive-read --for-worker) — retire this block
if forWorker && !hiveRetrievalOptedIn() {
    outputOK(map[string]interface{}{ /* ... reason: "this colony has not consented..." */ })
    return nil
}

// Source: cmd/context_weighting.go:22-27 (readHiveWisdomEntriesForDomains) — retire this block
if !hiveRetrievalOptedIn() {
    if fallbacks != nil {
        *fallbacks = append(*fallbacks, "hive_wisdom: colony has not opted in to cross-project wisdom")
    }
    return nil
}

// Source: cmd/colony_prime_context.go:652-654 (warning surfacing) — this one changes
// meaning, not just deletion, since the warning-suppression logic was keyed on opt-in status:
if hiveRetrievalOptedIn() {
    result.Warnings = append(result.Warnings, fallbacks...)
}
```

### D-08 Layer-1 test target (worker brief assembly, populated vs wiped)
```go
// Source: cmd/colony_prime_context.go:962-970 (the function to call from the test)
func resolveCodexWorkerContext() string {
	context := strings.TrimSpace(buildColonyPrimeOutput(true).PromptSection)
	// ...
	return context
}
```
Follow the existing pattern from `TestBuildWorkerBriefIncludesSurveyAndResearch` (`cmd/codex_build_test.go:3691-3721`): seed `instincts.json` (and/or hive `wisdom.json` under a `t.Setenv("AETHER_HUB_DIR", ...)` fixture) with a distinctive instinct/wisdom string in one subtest, run with an empty/absent store in a sibling subtest, and assert `strings.Contains`/`!strings.Contains` on `buildColonyPrimeOutput(true).PromptSection`. Since "## Active Instincts" reads `instincts.json` **unconditionally** (no hive-policy gate), it is the simplest, most deterministic target — it does not require setting `AETHER_HIVE_POLICY` at all, avoiding any coupling to the D-01/D-02 default-flip work landing first.

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|---------------|--------|
| Dry-run mutated state | `NewDryRunConsolidationService` is a separate no-write service | v1.25 (this milestone, Phase 160) | Already fixed; do not regress — pinned by `TestConsolidationPhaseEndDryRunDoesNotMutate`/`TestConsolidationSealDryRunDoesNotMutate` |
| Learning capture only inside `continue-finalize` | `captureContinueLearning` also fires on the default `/ant-continue` fast path | Recent, per comment at `cmd/codex_continue.go:966-971` ("This call is the fix for 'the colony never learns'") | Precedent this phase should follow for consolidation: make it fire on both paths, not just the external/finalize one |

**Deprecated/outdated:**
- `.aether/docs/structural-learning-stack.md`'s phase-end "three ants only" description — inaccurate today regardless of this phase's wiring (see Pitfall 1).
- `AGENTS.md:694-706`'s "Retrieval is opt-in, twice over" section — becomes false the moment D-02 lands; must be rewritten to describe the single-switch model, not merely have a caveat removed.

## Project Constraints (from CLAUDE.md)

- **Definition of Done:** "A requirement is satisfied only when a command exists that someone can run, and that command fails when the requirement is unmet." Applies directly: LEARN-01/02's satisfaction cannot be a checked box — it needs a test that fails if the consolidation call is removed from the continue/seal path (see Validation Architecture below), matching the CLAUDE.md-cited precedent of `consolidation-phase-end`/`consolidation-seal` having been "declared restored" in three prior milestones with no caller each time.
- **Dry-run purity corollary:** "An inspection or `--dry-run` command must not mutate state." Any new code touching `consolidation-phase-end`/`consolidation-seal` must not weaken `TestConsolidationPhaseEndDryRunDoesNotMutate`/`TestConsolidationSealDryRunDoesNotMutate`.
- **Documentation claim corollary:** "A documentation claim about runtime behaviour must be testable or removed." Directly governs D-03/D-10's docs pass — CLAUDE.md itself names this exact learning pipeline as the historical example of the violation (CLAUDE.md's own Definition of Done section, "The learning pipeline has been declared restored in v1.10, v1.11, v1.13 and v1.23").
- **Wrapper-runtime UX contract:** "Wrappers may add colony framing and narration but must not mutate state" / "Wrappers must not duplicate verification or gating logic." Directly governs D-04/D-06 — consolidation invocation and ceremony rendering must be Go-owned (`cmd/`), never added as an instruction inside `.claude/commands/ant/continue.md` or `.claude/commands/ant/seal.md`.
- **Phase 165 sole ownership of `build.md`/`continue.md` structure** (CLAUDE.md-adjacent, from CONTEXT.md's Prior phase constraints): this phase's wiring is entirely runtime-side by design (D-04), so it should not need to touch those wrapper files' structure at all — if a plan draft proposes editing `continue.md`, that is a signal the approach has drifted from D-04.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | Inserting the phase-end consolidation call *after* `advanceExternalContinue` returns success (rather than colocated with `captureContinueLearning` before it) is the stricter-correct choice for the external/finalize path | Code Examples, "Detecting phase actually advanced (external path)" | Low — both placements are defensible; if the planner picks the colocated placement instead, it still satisfies D-04's "when a phase actually completes and advances" as long as consolidation firing is conditioned on the atomic write having already succeeded, which it is either way since both statements are unreachable on an error return |
| A2 | The default answer to Pitfall 2 (pkg/memory's promotion path should win over `completeSealRuntime`'s existing loop) is correctly inferred from D-09's stated rule, but the *exact* mechanics of reconciliation (retire vs. refactor Step 1 into a reporting wrapper) were not specified in CONTEXT.md and are Claude's discretion at plan time | Common Pitfalls #2, Code Examples | Medium — if the planner instead decides both loops coexist deliberately (e.g., scoped to different instinct subsets), that would contradict this research's reading of D-09 and should be explicitly re-justified in the plan, not silently accepted |
| A3 | Unrecognized/garbage values of `AETHER_HIVE_POLICY` (e.g. a typo) should resolve to the new default (`promote`), matching "unset" behavior, rather than fail-safe to `off` | Common Pitfalls #4 | Low-Medium — a typo like `AETHER_HIVE_POLICY=raed` silently promoting instead of silently doing nothing is a slightly larger blast radius; worth a one-line decision in the plan rather than leaving it to whichever branch the refactor happens to produce |

**If this table is empty:** N/A — see above; all three assumptions are low-to-medium risk implementation-detail choices within an otherwise fully-verified research base, not factual claims about the domain.

## Open Questions

1. **Should `completeSealRuntime`'s existing instinct-promotion loop (lines 425-454) be deleted, or refactored to consume `consolidation-seal`'s already-computed `QueenEligible` list?**
   - What we know: Both loops promote the same underlying `instincts.json` entries to QUEEN.md, using different thresholds and different target sections. `pipeline.RunConsolidation()`'s `QueenEligible` computation (`pkg/memory/consolidate.go:149`) is more selective (requires application history) and is the "documented" pipeline.
   - What's unclear: Whether the seal ceremony's existing hive-promotion reporting (hivePromotedCount, hiveEligibleCount, etc., feeding `sealEnrichment` and `CROWNED-ANTHILL.md`) can be cleanly re-derived from `consolidation-seal`'s output without breaking `cmd/seal_ceremony_test.go`'s existing assertions (e.g., "CROWNED-ANTHILL.md should show 2 hive-promoted instincts").
   - Recommendation: Read `cmd/seal_ceremony_test.go` in full at plan time before deciding the refactor shape; treat this as the highest-risk single decision in the phase because it touches an existing, tested ceremony surface.

2. **Does the `LOCK-01`-style "regression test that fails if consolidation is disconnected" belong to this phase, or is it deferred to a later, unassigned LOCK phase?**
   - What we know: `LOCK-01..04` exist in REQUIREMENTS.md but have **no phase assignment** in the Traceability table (verified: `grep -n "LOCK-0" .planning/REQUIREMENTS.md` shows the requirement text but the Traceability section has no `LOCK-0*` rows at all).
   - What's unclear: Whether Phase 162 should write its own narrow "consolidation is called" regression test as a down payment toward LOCK-01 (recommended — it costs little and satisfies this phase's own Definition-of-Done obligation independent of LOCK's eventual home), or whether that's considered LOCK's exclusive territory.
   - Recommendation: Write the narrow test regardless — CLAUDE.md's Definition of Done requires it for LEARN-01/02 on its own terms, independent of whether a later milestone-wide LOCK suite also exists.

3. **Exact wording/threshold for what counts as "colony observed nothing new this phase" (D-07)?**
   - What we know: `consolidation-phase-end`'s JSON output already reports `instincts_decayed`, `instincts_archived`, `observations_decayed`, `promotion_candidates`, `queen_eligible`, `review_candidates`, `reread_candidates` (`cmd/graph_consolidation_cmds.go:255-264`) — all zero is a well-defined "nothing to report" state.
   - What's unclear: Whether "nothing new" should be all-fields-zero, or just `promotion_candidates == 0 && queen_eligible == 0` (the two D-06 explicitly calls out: "3 observations → 2 learnings → 1 new instinct").
   - Recommendation: Base the zero-state message on `promotion_candidates == 0 && queen_eligible == 0`, consistent with D-06's example wording, and treat decay/archive counts as detail available on request rather than gating the headline message.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go `testing` (stdlib) |
| Config file | none — `go test ./...` |
| Quick run command | `go test ./cmd/... -run 'Consolidation\|HivePolicy\|LearningCapture\|ColonyPrime' -count=1` |
| Full suite command | `go test ./... -count=1 -timeout 900s` (matches `.github/workflows/ci.yml:42`) |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| LEARN-01 | Phase-end consolidation is invoked when a phase durably advances (both continue paths) | integration (Go, calls the real continue-advance function against a seeded test store) | `go test ./cmd -run TestPhaseEndConsolidationInvokedOnAdvance -v` | ❌ Wave 0 — new test, model on `cmd/continue_learning_capture_test.go`'s direct-call pattern |
| LEARN-02 | Seal invokes the full 8-ant pass + decay + archive + report | integration | `go test ./cmd -run TestSealInvokesConsolidation -v` | ❌ Wave 0 — new test, model on `cmd/consolidation_dryrun_test.go`'s fixture-seeding + real-run-mutates pattern (`TestConsolidationRealRunStillMutates`) |
| LEARN-02 | Dry-run purity is preserved after wiring | unit | `go test ./cmd -run TestConsolidationPhaseEndDryRunDoesNotMutate\|TestConsolidationSealDryRunDoesNotMutate -v` | ✅ exists (`cmd/consolidation_dryrun_test.go`) — must keep passing unmodified |
| LEARN-03 | Duplicate promotion paths reconciled — no double-write to QUEEN.md for the same instinct at seal | unit/integration | `go test ./cmd -run TestSealDoesNotDoublePromoteInstincts -v` | ❌ Wave 0 — new test |
| LEARN-04 | Unset `AETHER_HIVE_POLICY` resolves to `promote`; explicit `off` still resolves to `off` | unit (table test) | `go test ./cmd -run TestHiveRuntimePolicyDefault -v` | ❌ Wave 0 — new test, model on existing `t.Setenv` usage in `cmd/hive_runtime_test.go` |
| LEARN-04 | Consent-file gate no longer blocks retrieval; all 3 call sites updated together | unit/integration | `go test ./cmd -run TestHiveReadNoLongerRequiresOptIn\|TestColonyPrimeHiveWisdom -v` | ❌ Wave 0 (new) / ✅ partial (`cmd/hive_runtime_test.go` has related coverage to extend) |
| LEARN-05 | Worker brief text differs, memory populated vs wiped (Layer 1, permanent, deterministic) | unit | `go test ./cmd -run TestColonyPrimeMemoryPresenceAffectsBrief -v` | ❌ Wave 0 — new test, model on `cmd/codex_build_test.go:3691` (`TestBuildWorkerBriefIncludesSurveyAndResearch`) |
| LEARN-05 | Worker brief text differs, memory populated vs wiped (Layer 2, recorded once, real model call) | manual/exhibit | N/A — produced during phase verification, not CI; saved as a phase-directory artifact per D-08 | N/A by design (D-08 explicitly scopes this to a one-time recorded exhibit, not a repeatable harness) |

### Sampling Rate
- **Per task commit:** `go test ./cmd/... -run 'Consolidation|HivePolicy|LearningCapture|ColonyPrime|Seal' -count=1`
- **Per wave merge:** `go test ./... -count=1 -timeout 900s` (matches CI's non-race job, `.github/workflows/ci.yml:42`)
- **Phase gate:** Full suite green (`go test ./... -race -count=1 -timeout 2400s`, matching `.github/workflows/ci.yml:45`) before `/ant-continue`/verify-work

### Wave 0 Gaps
- [ ] `cmd/consolidation_lifecycle_test.go` (or equivalent) — covers LEARN-01 (both continue paths call consolidation-phase-end on advance) and LEARN-02 (seal calls consolidation-seal)
- [ ] `cmd/hive_policy_test.go` (extend or create) — covers LEARN-04's default-flip table test
- [ ] Extension to `cmd/colony_prime_context_test.go` — covers LEARN-05 Layer 1 (brief text presence/absence)
- [ ] A seal-ceremony-level test extending `cmd/seal_ceremony_test.go` — covers LEARN-03's reconciliation (no double-promotion) without breaking the existing hive-promoted-count assertions

*(No framework install gap — `go test` is already the project's only test framework and is fully configured.)*

## Security Domain

`security_enforcement` not found disabled in `.planning/config.json` (absent = enabled per instructions), but this phase has no meaningful new attack surface: it invokes existing, already-reviewed subcommands and flips a default for an already-existing, non-network-facing policy switch (`AETHER_HIVE_POLICY` only ever controls local file reads/writes under `~/.aether/hive/` and local `.aether/data/`). No new user input parsing, no new network calls, no new auth/session concerns are introduced.

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | no | No auth surface touched |
| V3 Session Management | no | N/A |
| V4 Access Control | no | N/A — hive policy is a local operator-controlled env var, not a multi-tenant access boundary |
| V5 Input Validation | marginal | `currentHiveRuntimePolicy()`'s switch already handles unrecognized string values safely (falls through to a defined default); no new parsing introduced by the default flip |
| V6 Cryptography | no | N/A |

### Known Threat Patterns for this stack

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Cross-repo wisdom pollution (a malicious/broken repo promotes garbage into the shared hub, now read by more repos by default) | Tampering | Already mitigated by existing 200-cap LRU, dedup-by-content-hash, and quarantine-on-contradiction logic in `pkg/learn/hive_store.go`/`cmd/hive.go`; the D-01/D-02 default flip increases the *number* of repos reading hive content by default but does not change the write-side controls, which are unaffected by this phase |
| Non-blocking consolidation failure silently masking a real corruption (sentinel abort) | Denial of Service (soft) | `curation.Orchestrator.Run`'s sentinel-abort behavior (skips remaining steps, returns error) already exists; D-05's "warn loudly" requirement is itself the mitigation — ensure the loud warning surfaces sentinel-abort specifically, not just generic failure text |

## Sources

### Primary (HIGH confidence — direct repo reads this session)
- `pkg/memory/pipeline.go`, `pkg/memory/consolidate.go`, `pkg/memory/queen.go`, `pkg/memory/promote.go` — pipeline wiring, consolidation algorithm, QUEEN.md promotion formats
- `cmd/graph_consolidation_cmds.go` — the two subcommands' full implementation, dry-run branching
- `pkg/learn/wrappers.go`, `pkg/learn/learn.go` — the shim vs. the real competing capture system
- `cmd/codex_continue.go` (~900-975), `cmd/codex_continue_finalize.go` (459-497, 1264-1363) — both continue paths' advance detection and existing learning capture call
- `cmd/codex_workflow_cmds.go` (409-508) — `completeSealRuntime`, the existing seal promotion loop
- `cmd/hive_policy.go`, `cmd/hive.go` (274-335, 533-563, 867-900), `cmd/context_weighting.go`, `cmd/colony_prime_context.go` (600-700, 962-970) — hive gating, worker-context injection funnel
- `cmd/codex_visuals.go` (30-130, 3550-3610), `cmd/ceremony_emitter.go` (304-345) — caste identity system, ceremony emission pattern
- `pkg/agent/curation/orchestrator.go`, `pkg/agent/curation/archivist.go` — 8-ant fixed order, per-step `StepResult` shape
- `cmd/consolidation_dryrun_test.go`, `cmd/continue_learning_capture_test.go`, `cmd/codex_build_test.go` (3691-3760), `cmd/hive_test_optin.go`, `cmd/hive_runtime_test.go` — existing test patterns to model new tests on
- `.aether/docs/structural-learning-stack.md`, `AGENTS.md` (655-714, 899), `CLAUDE.md` (project instructions, this session's system context) — doc claims requiring correction
- `.github/workflows/ci.yml` — CI test invocation commands
- `.planning/REQUIREMENTS.md`, `.planning/STATE.md`, `.planning/phases/162-switch-on-learning/162-CONTEXT.md` — requirement text, locked decisions, roadmap placement notes

### Secondary (MEDIUM confidence)
- None — no web/ecosystem research was needed for this phase; everything is repo-internal.

### Tertiary (LOW confidence)
- None.

## Metadata

**Confidence breakdown:**
- Standard stack: N/A — no new dependencies
- Architecture: HIGH — every insertion point, gate, and data shape cited above was read directly from current source, not inferred
- Pitfalls: HIGH — all 5 pitfalls are concrete, verified code facts (e.g., the phase-end/ants doc mismatch, the double-promotion-loop overlap, the 3-call-site consent gate), not speculative risks

**Research date:** 2026-08-04
**Valid until:** Until this phase's plan lands — this research is tied to exact line numbers in a fast-moving milestone (v1.25); re-verify line numbers if Phase 161 (which merges first) touches any of the same files before Phase 162 executes.
