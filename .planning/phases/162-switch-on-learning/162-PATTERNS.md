# Phase 162: Switch On Learning - Pattern Map

**Mapped:** 2026-08-04
**Files analyzed:** 13 (5 modified-heavy, 4 modified-light, 4 new)
**Analogs found:** 13 / 13 (all in-repo; this phase invokes existing code, no external library patterns needed)

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|--------------------|------|-----------|-----------------|----------------|
| `cmd/consolidation_lifecycle.go` (NEW) | service/wrapper | event-driven (non-blocking side effect) | `cmd/codex_continue_finalize.go` `captureContinueLearning` (lines 1264-1363) | exact — same non-blocking-warn shape, same call-site pattern |
| `cmd/codex_continue.go` (~:973) | controller (insertion point) | CRUD (state advance + side effect) | same file, existing `captureContinueLearning(...)` call at :973 | exact — literal sibling call |
| `cmd/codex_continue_finalize.go` (~:489) | controller (insertion point) | CRUD (state advance + side effect) | same file, existing `captureContinueLearning(...)` call at :489 | exact — literal sibling call |
| `cmd/codex_workflow_cmds.go` (`completeSealRuntime`, :409-454) | controller (reconciliation) | CRUD (promotion loop) | `pkg/memory/consolidate.go` `QueenEligible` computation (:149) + `pkg/memory/queen.go` `PromoteInstinct` (:157) | role-match — same conceptual step, competing implementation to reconcile against |
| `cmd/hive_policy.go` (:18-27) | config | request-response (pure function) | itself — table-test extension of `currentHiveRuntimePolicy()` | exact |
| `cmd/hive.go` (:274-335, :544-563) | service (gate removal) | request-response | itself — 3-site consent-gate removal | exact |
| `cmd/context_weighting.go` (:18-27) | service (gate removal) | request-response | `cmd/hive.go` consent-gate call sites (same pattern, 2nd site) | exact |
| `cmd/colony_prime_context.go` (:652-654, :962) | service (context assembly) | transform (prompt building) | itself — existing `hiveRetrievalOptedIn()` warning-suppression block | exact |
| `cmd/codex_visuals.go` (caste maps, :36-121) | config (identity data) | transform (lookup) | itself — existing 24-entry caste map pattern, extend with 8-9 curation-ant entries | exact |
| `cmd/ceremony_emitter.go` (:304-327) | service (rendering) | event-driven | `emitContinueCeremonyFlowSequence` (same file, same function shape) | exact |
| `cmd/graph_consolidation_cmds.go` (:284-345) | controller (existing subcommand, extend output) | request-response | itself — `consolidationSealCmd`'s `stepInfo`/`steps` struct | exact |
| `cmd/consolidation_lifecycle_test.go` (NEW) | test | integration | `cmd/continue_learning_capture_test.go` (whole file) | exact — identical "call the shared function directly, assert store state" shape |
| `cmd/hive_policy_test.go` (NEW/extend) | test | unit (table test) | `cmd/hive_runtime_test.go` (existing `t.Setenv(hivePolicyEnv, ...)` usage) | exact |
| `cmd/colony_prime_context_test.go` (extend, D-08 Layer 1) | test | unit | `cmd/codex_build_test.go:3691` `TestBuildWorkerBriefIncludesSurveyAndResearch` | exact — populated-vs-absent fixture pattern |
| `cmd/seal_ceremony_test.go` (extend, LEARN-03) | test | integration | `TestSealHivePromotedCount` (same file, :798-844) | exact — same `runSealCmd` + CROWNED-ANTHILL.md assertion harness |
| `.aether/docs/learn-vs-memory-decision.md` (NEW, D-10) | doc | — | `.aether/docs/structural-learning-stack.md` (structure/tone) | role-match |

## Pattern Assignments

### `cmd/consolidation_lifecycle.go` (NEW — service, event-driven non-blocking side effect)

**Analog:** `cmd/codex_continue_finalize.go` — `captureContinueLearning` (lines 1264-1363) and the seal hive-promotion loop in `cmd/codex_workflow_cmds.go` (lines 409-454).

**Non-blocking-warn pattern to copy** (`cmd/codex_continue_finalize.go:1345-1347`):
```go
if err := learnStore.Add(entry); err != nil {
    // Non-blocking: learning failure must not prevent phase advancement
    fmt.Fprintf(os.Stderr, "warning: failed to capture learning: %v\n", err)
}
```

**Second precedent for the same shape** (`cmd/codex_workflow_cmds.go:446-451`):
```go
if err := promoteToHiveWithReference(entry.Action, domain, repoName, entry.Confidence, "seal:"+entry.ID); err != nil {
    log.Printf("seal: hive-promote failed for %s: %v", entry.ID, err)
    hivePromotionFailures++
} else {
    hivePromotedCount++
}
```

**What the new wrapper functions must do (per D-05, strengthen the wording beyond the two precedents above):**
- `runPhaseEndConsolidation(phaseID int) error-like-non-fatal` — builds `bus := events.NewBus(store, events.DefaultConfig())`, `pipeline := learn.NewPipeline(store, bus, pipelineConfigForStore())`, calls `pipeline.RunConsolidation(ctx)` (the REAL, non-dry-run path — see Pattern 2 below), and on error prints unmissably: `fmt.Fprintf(os.Stderr, "⚠ phase advanced WITHOUT consolidation — %v\n", err)`. Never returns an error to the caller that would abort the advance.
- `runSealConsolidation() (curation.RunResult, *learn.ConsolidationResult, error-like-non-fatal)` — calls `curation.NewOrchestrator(store, bus).Run(ctx, false)` then `pipeline.RunConsolidation(ctx)`, exactly as `consolidationSealCmd`'s `RunE` body already does (`cmd/graph_consolidation_cmds.go:290-327`) but returning the per-step data instead of discarding it into a single `stepInfo.Summary` string, so the ceremony renderer (Pattern 3) can render each of the 8 ants individually.
- Reuse `pipelineConfigForStore()` (`cmd/graph_consolidation_cmds.go:15-29`) verbatim — do not reinvent config construction.

**Dry-run purity constraint (copy exactly, do not invert):**
```go
// Source: cmd/graph_consolidation_cmds.go:226-234 — the ONLY correct branch shape
if dryRun {
    service := learn.NewDryRunConsolidationService(store, bus, "QUEEN.md", pipelineConfigForStore().ColonyName)
    result, err = service.Run(ctx)
} else {
    result, err = pipeline.RunConsolidation(ctx)
}
```
The new runtime callers (`runPhaseEndConsolidation`, `runSealConsolidation`) must always take the `else` (real) branch — they are never invoked with `--dry-run` semantics; `TestConsolidationPhaseEndDryRunDoesNotMutate` / `TestConsolidationSealDryRunDoesNotMutate` (`cmd/consolidation_dryrun_test.go`) pin the CLI subcommand's own dry-run path and must keep passing unmodified.

---

### `cmd/codex_continue.go` (~:973) — insertion point

**Analog:** the file's own existing call, directly above.

**Exact insertion context** (`cmd/codex_continue.go:966-976`, current code):
```go
// Durable learning capture on the DEFAULT path. This call is the fix for
// "the colony never learns": capture previously existed only inside
// continue-finalize, which the wrapper forbids for fast continue, so
// pkg/learn, hypothesis promotion, and auto-skill creation were unreachable
// in normal daily use. Gates have passed by this point; state is committed;
// learning failure is non-blocking inside the function.
captureContinueLearning(phase, workerFlow, gates, "", false, now)
emitContinueCeremonyFlowSequence("aether-continue", phase, workerFlow)
flowEvents := continueWorkerFlowEvents(now, workerFlow)
updated.Events = append(updated.Events, flowEvents...)
_ = appendRuntimeStateEventsIfCurrent(updated, flowEvents)
```
This is reached only after the atomic `store.UpdateJSONAtomically("COLONY_STATE.json", ...)` write (lines ~914-944) has already committed `updated.Plan.Phases[currentIdx].Status = colony.PhaseCompleted` — the exact "phase durably advanced" moment D-04 requires. `runPhaseEndConsolidation(phase.ID)` belongs directly beside `captureContinueLearning(...)`, before `emitContinueCeremonyFlowSequence` (so the 🧠 learning beat, D-06/D-07, can be folded into that same ceremony call or emitted immediately after it).

---

### `cmd/codex_continue_finalize.go` (~:489) — insertion point

**Analog:** the file's own existing call, `captureContinueLearning` at line 489, and the note in RESEARCH.md's Code Examples about placement relative to `advanceExternalContinue`.

**Exact insertion context** (`cmd/codex_continue_finalize.go:483-497`, current code):
```go
// --- Learning capture (D-01, D-02, D-03, D-04) ---
// Shared with the default continue path; see captureContinueLearning.
runID := ""
if runHandle != nil {
    runID = runHandle.Run.ID
}
captureContinueLearning(phase, workerFlow, gates, runID, noLearn, now)

result, updated, nextPhase, housekeeping, final, err := advanceExternalContinue(root, state, phase, manifest, verification, assessment, gates, review, reviewReportRel, watcherFlow, workerFlow, now, verificationReportRel, gateReportRel, finalizeReviewDepth)
if err != nil {
    return nil, state, phase, nil, housekeeping, final, err
}
runStatus = "completed"
return result, updated, phase, nextPhase, housekeeping, final, nil
```
Recommended placement (per RESEARCH.md's assumption A1, stricter-correct): call `runPhaseEndConsolidation(phase.ID)` **after** `advanceExternalContinue` returns with `err == nil`, not colocated with `captureContinueLearning` before it — `PhaseCompleted` is written *inside* `advanceExternalContinue`, so only the post-return point guarantees the phase truly advanced. The colocated placement (matching the default-path precedent exactly) is defensible too since both statements are unreachable on the error return; document whichever choice is made.

---

### `cmd/codex_workflow_cmds.go` (`completeSealRuntime`, :409-454) — reconciliation target

**Analog:** own existing loop, must be refactored/reconciled with `pkg/memory`'s consolidation output per D-09 (pkg/memory wins).

**Existing loop to reconcile (do not just add a second, unreconciled call beside this):**
```go
// Source: cmd/codex_workflow_cmds.go:429-454 (current, pre-Phase-162)
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
            domain := entry.Domain
            if domain == "" {
                domain = "general"
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
This writes to QUEEN.md's **"Wisdom"** section (`cmd/queen.go` — `appendEntryToQueenSection(text, "Wisdom", entry)`, lines 278-287) using threshold `Confidence >= 0.8 && Action != ""`. `pkg/memory`'s pipeline writes to QUEEN.md's **"Instincts"** section (`pkg/memory/queen.go:157-161`, `PromoteInstinct` → `WriteEntry(ctx, queenPath, "Instincts", entry, colonyName)`) using threshold `Confidence >= 0.75 && Applications >= 3` (`pkg/memory/consolidate.go:149`). Per D-09, `pkg/memory`'s path wins; the recommended shape (per RESEARCH.md Open Question 1) is to refactor this block into a thin ceremony-reporting wrapper that consumes `runSealConsolidation()`'s already-computed `QueenEligible` list instead of independently re-scanning `instincts.json` with a different threshold, while preserving the existing `hivePromotedCount`/`hiveEligibleCount`/`hivePromotionFailures` variables that `sealEnrichment` (line 497) and `TestSealHivePromotedCount` already depend on.

**Downstream consumer that must keep working** (`cmd/codex_workflow_cmds.go:496-508`):
```go
enrichment := sealEnrichment{
    LearningsCount:        len(state.Memory.PhaseLearnings),
    InstinctsPromoted:     promotedInstinctNames,
    HiveEligible:          hiveEligibleCount,
    HivePromoted:          hivePromotedCount,
    HivePromotionFailures: hivePromotionFailures,
    SignalsExpired:        expiredFOCUSCount,
    FlagsResolved:         countResolvedFlags(store),
    ShelfCandidates:       candidates,
    FinalReview:           finalReview,
    ReviewBacklog:         reviewBacklog,
}
```

---

### `cmd/hive_policy.go` (:18-27) — default flip (D-01/D-02)

**Analog:** itself, table-test extension.

**Current code (exact target):**
```go
func currentHiveRuntimePolicy() hiveRuntimePolicy {
    switch strings.ToLower(strings.TrimSpace(os.Getenv(hivePolicyEnv))) {
    case "read", "inject", "on":
        return hivePolicyRead
    case "promote", "full":
        return hivePolicyPromote
    default:
        return hivePolicyOff
    }
}
```
**Required change (Pitfall 4 — do not collapse unset and explicit "off"):**
```go
func currentHiveRuntimePolicy() hiveRuntimePolicy {
    switch strings.ToLower(strings.TrimSpace(os.Getenv(hivePolicyEnv))) {
    case "off":
        return hivePolicyOff
    case "read", "inject", "on":
        return hivePolicyRead
    case "promote", "full":
        return hivePolicyPromote
    default:
        return hivePolicyPromote // D-02: unset now means "on, full"
    }
}
```
`automaticHiveReadEnabled()` (:29-32) and `automaticHivePromotionEnabled()` (:34-36) need no changes — they already derive correctly from the policy value.

---

### `cmd/hive.go`, `cmd/context_weighting.go`, `cmd/colony_prime_context.go` — consent-gate retirement (D-02)

**Analog:** the 3 call sites are each other's closest analog; retire all 3 in the same change per Pitfall 3.

**Site 1 — `cmd/hive.go:544-563` (`hive-read --for-worker`), retire lines 554-563 only (keep the policy check at 544):**
```go
if forWorker && !automaticHiveReadEnabled() {
    outputOK(map[string]interface{}{ /* policy-disabled reason */ })
    return nil
}
if forWorker && !hiveRetrievalOptedIn() {   // <-- DELETE this whole block
    outputOK(map[string]interface{}{
        "entries": []hiveWisdomEntry{},
        "total":   0,
        "policy":  string(currentHiveRuntimePolicy()),
        "enabled": false,
        "reason":  "this colony has not consented to cross-project wisdom; run `aether hive-opt-in` to enable it here",
    })
    return nil
}
```

**Site 2 — `cmd/context_weighting.go:18-27` (`readHiveWisdomEntriesForDomains`), retire lines 22-27:**
```go
func readHiveWisdomEntriesForDomains(hubDir string, limit int, domains []string, fallbacks *[]string) []hiveWisdomEntry {
    if !automaticHiveReadEnabled() {
        return nil
    }
    if !hiveRetrievalOptedIn() {              // <-- DELETE this whole block
        if fallbacks != nil {
            *fallbacks = append(*fallbacks, "hive_wisdom: colony has not opted in to cross-project wisdom")
        }
        return nil
    }
    // ... existing wisdomPath read continues unchanged
```

**Site 3 — `cmd/colony_prime_context.go:642-654` (warning-suppression, CHANGES MEANING, not pure deletion):**
```go
// Surface why hive wisdom was withheld — but only once the colony has
// actually opted in. ... (this whole comment block's premise is retired by D-02)
if hiveRetrievalOptedIn() {                    // <-- retire the gate; fallbacks
    result.Warnings = append(result.Warnings, fallbacks...)   //   should now always surface,
}                                                                //   since opt-in no longer exists
```
Since retrieval is default-on now, `fallbacks` (e.g. "domain did not match", "everything decayed to dormant") should surface unconditionally — the old rationale ("don't warn about the default state of every colony") no longer applies because there is no more not-opted-in default state.

**Also retire (per Pitfall 3, same commit):** `hiveRetrievalConsentFile`, `hiveRetrievalConsentPath()`, `writeHiveRetrievalConsent()` (`cmd/hive.go:274-335`), and the `hive-opt-in`/`hive-opt-out` cobra commands (`cmd/hive.go:874`, `:891`). Update `cmd/hive_test_optin.go`'s `enableHiveForTest` helper (currently calls `writeHiveRetrievalConsent(true)` at line 24) — since promote is now the default, this helper likely simplifies to just `t.Setenv(hivePolicyEnv, "read")` or is removed if tests no longer need to opt in at all. 6 test files call it: `cmd/codex_prompt_context_test.go`, `cmd/colony_prime_context_test.go`, `cmd/colony_prime_audit_test.go`, `cmd/colony_prime_budget_test.go`, `cmd/hive_runtime_test.go`, `cmd/pheromone_loader_test.go`.

---

### `cmd/codex_visuals.go` — caste identity for curation ants (D-06)

**Analog:** the existing 24-entry maps, same file.

**Current pattern (`cmd/codex_visuals.go:36-121`), extend all three maps with matching keys:**
```go
var casteEmojiMap = map[string]string{
    "queen":         "👑",
    "builder":       "🔨",
    "watcher":       "👁️",
    // ... 21 more existing entries ...
}
var casteColorMap = map[string]string{
    "queen":         "35",
    "builder":       "33",
    // ... ANSI 30-97 codes ...
}
var casteLabelMap = map[string]string{
    "queen":         "Queen",
    "builder":       "Builder",
    // ...
}
```
**Fallback behavior if left unmapped (`cmd/codex_visuals.go:3555-3593`):**
```go
func casteEmoji(caste string) string {
    caste = normalizeCasteKey(caste)
    if loaded := loadVisualsConfig(); loaded != nil {
        if emoji, ok := loaded.CasteEmojiMap[caste]; ok { return emoji }
    }
    if emoji, ok := casteEmojiMap[caste]; ok { return emoji }
    return "🐜"   // <-- what all 9 curation-ant names render as today
}
```
Add entries for the 9 curation-ant keys used by `pkg/agent/curation/orchestrator.go:94-101`: `sentinel`, `nurse`, `critic`, `herald`, `janitor`, `archivist`, `librarian`, `scribe` (plus optionally `orchestrator`/`curation` for the aggregate line). Exact emoji/color/label choices are Claude's discretion (CONTEXT.md), but must use unclaimed ANSI codes and follow the existing lowercase-snake-key convention (`normalizeCasteKey`).

---

### `cmd/ceremony_emitter.go` — 🧠 learning beat + 8-ant seal announcements (D-06/D-07)

**Analog:** `emitContinueCeremonyFlowSequence` (same file, lines 304-327).

```go
// Source: cmd/ceremony_emitter.go:304-327 — exact shape to extend/copy
func emitContinueCeremonyFlowSequence(source string, phase colony.Phase, workerFlow []codexContinueWorkerFlowStep) {
    steps := make([]lifecycleCeremonyStep, 0, len(workerFlow))
    for idx, step := range workerFlow {
        taskID := fmt.Sprintf("continue-%d", idx)
        if stage := strings.TrimSpace(step.Stage); stage != "" {
            taskID = fmt.Sprintf("continue-%s-%d", stage, idx)
        }
        steps = append(steps, lifecycleCeremonyStep{
            Wave:    continueCeremonyWaveForStage(step.Stage),
            SpawnID: taskID,
            Caste:   step.Caste,
            Name:    step.Name,
            TaskID:  taskID,
            Task:    continueWorkerFlowTask(step),
            Status:  step.Status,
            Message: step.Summary,
        })
    }
    emitLifecycleCeremonySequence(lifecycleCeremonyTopics{
        WaveStart: events.CeremonyTopicContinueWaveStart,
        Spawn:     events.CeremonyTopicContinueSpawn,
        WaveEnd:   events.CeremonyTopicContinueWaveEnd,
    }, source, "continue", phase.ID, phase.Name, steps)
}
```
**D-06/D-07 learning beat:** append one synthetic `lifecycleCeremonyStep{Caste: "queen"-or-new-"librarian"-key, Name: <deterministic>, Status: "completed", Message: "3 observations → 2 learnings → 1 new instinct (confidence 0.75)"}` (or the D-07 zero-state text "colony observed nothing new this phase") built from `runPhaseEndConsolidation`'s `learn.ConsolidationResult` (`PromotionCandidates`, `QueenEligible`, `ObservationsDecayed` fields — same fields surfaced today by `consolidationPhaseEndCmd`'s JSON output at `cmd/graph_consolidation_cmds.go:255-264`). Zero-state message should trigger on `promotion_candidates == 0 && queen_eligible == 0` per RESEARCH.md's Open Question 3 recommendation.

**8-ant seal announcements:** the data is `curation.StepResult{Name, Success, Summary map[string]any}` per ant, from `orch.Run(ctx, dryRun)` (`pkg/agent/curation/orchestrator.go:94-101, 140-197`) — currently discarded into one aggregate string by `consolidationSealCmd` (`cmd/graph_consolidation_cmds.go:298-303`). `runSealConsolidation()` in the new `consolidation_lifecycle.go` should call the orchestrator directly (not go through the CLI subcommand) so each `StepResult` becomes one `lifecycleCeremonyStep` with `Caste` set to the ant's lowercase name (matching the new `casteEmojiMap` keys above), rendered via the same `emitLifecycleCeremonySequence` mechanism.

---

### `cmd/graph_consolidation_cmds.go` (:284-345) — existing subcommand output shape (reference only, keep working)

**Analog:** itself — the `stepInfo`/`steps` pattern already exists and is the shape `runSealConsolidation()` should either reuse or extend to expose the richer `curation.StepResult` data:
```go
// Source: cmd/graph_consolidation_cmds.go:284-303
type stepInfo struct {
    Name    string `json:"name"`
    Success bool   `json:"success"`
    Summary string `json:"summary,omitempty"`
}
var steps []stepInfo
orch := curation.NewOrchestrator(store, bus)
curResult, err := orch.Run(ctx, dryRun)
if err != nil {
    steps = append(steps, stepInfo{Name: "curation", Success: false, Summary: err.Error()})
} else {
    steps = append(steps, stepInfo{
        Name:    "curation",
        Success: true,
        Summary: fmt.Sprintf("succeeded=%d failed=%d", curResult.Succeeded, curResult.Failed),
    })
}
```
Do not modify this subcommand's own dry-run behavior (Pattern 2); only extend what it outputs, or have `runSealConsolidation` call `curation.NewOrchestrator` and `pipeline.RunConsolidation` directly (bypassing the CLI wrapper) to get the un-aggregated per-ant results.

---

## Shared Patterns

### Non-blocking failure with unmissable warning (D-05)
**Source:** `cmd/codex_continue_finalize.go:1345-1347` (learn store) and `cmd/codex_workflow_cmds.go:446-451` (hive promote)
**Apply to:** `runPhaseEndConsolidation`, `runSealConsolidation` in the new `cmd/consolidation_lifecycle.go`
```go
if err := learnStore.Add(entry); err != nil {
    fmt.Fprintf(os.Stderr, "warning: failed to capture learning: %v\n", err)
}
```
D-05's literal wording requirement ("phase advanced WITHOUT consolidation — <reason>") is stronger than either existing precedent's wording — copy the *shape* (non-blocking, stderr, continues execution), not the exact string.

### Dry-run purity via structurally separate service
**Source:** `cmd/graph_consolidation_cmds.go:226-234`, pinned by `cmd/consolidation_dryrun_test.go`
**Apply to:** any new code path that could accidentally route through `NewDryRunConsolidationService` when it shouldn't, or vice versa. New runtime callers always use the real (`else`) branch.

### Caste-styled ceremony rendering
**Source:** `cmd/ceremony_emitter.go:304-327` + `cmd/codex_visuals.go:3591-3593` (`casteIdentity()`)
**Apply to:** the 🧠 learning beat and 8 named ant beats (D-06). Format target: `casteEmoji(caste) + " " + colorizeCaste(caste, casteLabel(caste))` — never hand-roll `fmt.Println` banners (RESEARCH.md's Don't Hand-Roll table).

### Test pattern: call the shared function directly against a seeded test store
**Source:** `cmd/continue_learning_capture_test.go` (whole file, 84 lines) — `TestLearningCaptureOnDefaultContinue` / `TestLearningCaptureRespectsFailedWorkers`
**Apply to:** `cmd/consolidation_lifecycle_test.go` (LEARN-01/02 wiring tests). Pattern: `saveGlobals(t)` → `newTestStore(t)` → seed fixture (`colony.Phase`, `codexContinueWorkerFlowStep`, `codexContinueGateReport`) → call the function under test directly (not via `rootCmd.Execute()`) → assert on the resulting file in `s.BasePath()`.

### Test pattern: populated-vs-absent fixture, assert prompt text presence/absence
**Source:** `cmd/codex_build_test.go:3691-3753` (`TestBuildWorkerBriefIncludesSurveyAndResearch`)
**Apply to:** D-08 Layer 1 test extending `cmd/colony_prime_context_test.go`. Pattern: two `t.Run` subtests, each builds its own `t.TempDir()` + `storage.NewStore(dataDir)`, writes (or omits) a fixture file, calls `buildColonyPrimeOutput(true).PromptSection` (or `resolveCodexWorkerContext()`), asserts `strings.Contains`/`!strings.Contains`. Since `## Active Instincts` reads `instincts.json` unconditionally (no hive-policy gate), seed/omit `instincts.json` for the simplest, most deterministic assertion — no `AETHER_HIVE_POLICY` coupling needed.

### Test pattern: seal-ceremony end-to-end via `runSealCmd` + CROWNED-ANTHILL.md assertion
**Source:** `cmd/seal_ceremony_test.go:798-844` (`TestSealHivePromotedCount`)
**Apply to:** LEARN-03's reconciliation test (no double-promotion). Pattern: `setupSealTestStore(t)` → seed `instincts.json` with known-confidence entries → seed hub `wisdom.json` under `AETHER_HUB_DIR` → `runSealCmd(t, s, tmpDir, nil)` → read `CROWNED-ANTHILL.md` → assert exact table row text (e.g. `"| Hive-promoted instincts | 2 |"`). For LEARN-03, additionally assert QUEEN.md contains each promoted instinct's action text exactly once (not once under "Wisdom" and again under "Instincts").

### Test pattern: dry-run purity via file hashing
**Source:** `cmd/consolidation_dryrun_test.go` (whole file, 147 lines)
**Apply to:** any new test asserting the new runtime callers never accidentally take the dry-run branch. Critical gotcha documented in the file's own comments (lines 37-41): watch `store.BasePath()`, not the tmp root — the tmp root watch previously made purity assertions pass vacuously.

## No Analog Found

None. Every file in this phase's scope has an exact or role-match analog already in the codebase, consistent with RESEARCH.md's framing: this is 100% connective-tissue work invoking, reconciling, and documenting code that already exists.

| File | Role | Data Flow | Reason |
|------|------|-----------|--------|
| `.aether/docs/learn-vs-memory-decision.md` (D-10 ADR) | doc | — | No prior ADR-style doc exists in `.aether/docs/`; use `structural-learning-stack.md`'s prose structure and heading style as the closest tonal analog, not a strict template |

## Metadata

**Analog search scope:** `cmd/` (continue/seal/hive/visuals/ceremony/consolidation files and their existing `_test.go` siblings), `pkg/memory/`, `pkg/learn/`, `pkg/agent/curation/`
**Files scanned:** ~20 read directly (line-cited above) + targeted greps across `cmd/*.go` for `hiveRetrievalOptedIn`, `automaticHivePromotionEnabled`, `casteEmoji`/`casteColorMap`/`casteLabelMap`, `captureContinueLearning`
**Pattern extraction date:** 2026-08-04
