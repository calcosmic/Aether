---
phase: 162-switch-on-learning
reviewed: 2026-08-04T13:56:20Z
depth: standard
files_reviewed: 36
files_reviewed_list:
  - .aether/docs/learning-system-authority.md
  - .aether/docs/structural-learning-stack.md
  - cmd/ceremony_emitter_test.go
  - cmd/ceremony_emitter.go
  - cmd/codex_continue_finalize.go
  - cmd/codex_continue.go
  - cmd/codex_prompt_context_test.go
  - cmd/codex_visuals.go
  - cmd/codex_workflow_cmds.go
  - cmd/colony_prime_audit_test.go
  - cmd/colony_prime_budget_test.go
  - cmd/colony_prime_context_test.go
  - cmd/colony_prime_context.go
  - cmd/consolidation_dryrun_test.go
  - cmd/consolidation_lifecycle_test.go
  - cmd/consolidation_lifecycle.go
  - cmd/consolidation_promotion_target_test.go
  - cmd/context_weighting.go
  - cmd/context.go
  - cmd/docs_truth_test.go
  - cmd/golden_workflow_test.go
  - cmd/graph_consolidation_cmds.go
  - cmd/hive_policy_test.go
  - cmd/hive_policy.go
  - cmd/hive_runtime_test.go
  - cmd/hive.go
  - cmd/learning_beat_test.go
  - cmd/memory_injection_test.go
  - cmd/pheromone_loader_test.go
  - cmd/queen.go
  - cmd/seal_ceremony_test.go
  - cmd/testdata/command_catalog.json
  - cmd/testdata/golden_continue.txt
  - cmd/testdata/parity_snapshot.json
  - cmd/testdata/regression_snapshot.json
  - pkg/events/ceremony.go
findings:
  critical: 1
  warning: 3
  info: 5
  total: 9
status: issues_found
---

# Phase 162: Code Review Report

**Reviewed:** 2026-08-04T13:56:20Z
**Depth:** standard
**Files Reviewed:** 36
**Status:** issues_found

## Summary

Reviewed the Phase 162 "Switch On Learning" changeset (~2,700 insertions): wiring
phase-end consolidation into both `/ant-continue` paths, seal consolidation into
`completeSealRuntime`, the QUEEN.md promotion-target fix, the D-09
double-promotion reconciliation, the hive default-on flip with consent-file
retirement, per-ant seal beats, and the learning-beat rendering.

Verification performed: `go build ./cmd/aether` passes, `go vet` clean, and all
26 phase-targeted tests pass (`go test ./cmd/ -run '...' -count=1` → ok, 3.7s).
CLAUDE.md and AGENTS.md were confirmed updated (no retired doc claims remain).
Hive text is sanitized at `hive-store` via `colony.SanitizeSignalContent`
(cmd/hive.go:382), which is the right compensating control for the default-on
retrieval flip; unrecognized `AETHER_HIVE_POLICY` values fail closed with a
one-shot stderr warning. The test suite for this phase is unusually strong —
mutation-detecting fixtures, isolation of the QUEEN.md route from the
instincts.json route, and proportion-style invariants.

However, the happy path is well-tested while the failure paths of
`runSealConsolidation` are not, and that is exactly where the defects live. The
critical finding is a reachable path where the seal both violates the D-09
"exactly one QUEEN.md entry per instinct" invariant this phase exists to enforce
AND reports "sealed WITHOUT consolidation" while consolidation actually ran and
mutated three files.

## Critical Issues

### CR-01: Curation failure does not short-circuit the mutating consolidation pipeline — reachable D-09 double-promotion plus a false "WITHOUT consolidation" report

**File:** `cmd/consolidation_lifecycle.go:268-341` (interacts with `cmd/codex_workflow_cmds.go:438-471`)
**Issue:** `runSealConsolidation` runs `curation.NewOrchestrator(...).Run` (line 268), and when that fails it still proceeds to run the **mutating** `pipeline.RunConsolidation(ctx)` (line 294). If curation fails but consolidation succeeds, the failure branch at lines 311-330 sets `Ran: false` and returns **before** line 339 populates `QueenPromotedIDs` — even though `RunConsolidation` just decayed trust scores, archived instincts, decayed observations, and **promoted every QueenEligible instinct into QUEEN.md's `## Instincts` section**.

Concretely reachable: corrupt `pheromones.json` (or `event-bus.jsonl`, `instinct-graph.json`, `COLONY_STATE.json` — see `sentinelCheckedStores`, pkg/agent/curation/sentinel.go:28) with a valid `instincts.json` triggers a sentinel abort while `RunConsolidation` runs cleanly. Three consequences:

1. **D-09 violation.** `completeSealRuntime` builds its skip-set from the empty `QueenPromotedIDs` (cmd/codex_workflow_cmds.go:438), so the subordinate loop calls `promoteInstinctLocal` (line 470) for the same instinct the pipeline just wrote — one instinct, two QUEEN.md entries (`## Instincts` via `QueenService.PromoteInstinct`, `## Wisdom` via `promoteInstinctLocal`; the section-scoped dedup in `appendEntriesToQueenSection` cannot catch a cross-section duplicate, and the entry formats differ). This is precisely the double-write `TestSealDoesNotDoublePromoteInstincts` pins — but only on the happy path. `TestRunSealConsolidationNonBlockingOnFailure` only tests corrupt `instincts.json`, where both stages fail together, so this asymmetric path has zero coverage.
2. **False report.** The operator sees "colony sealed WITHOUT consolidation — curation: sentinel abort: ..." while decay, archival, observation decay, and queen promotion all ran. The summary's decay/archive counts also read zero despite real mutations.
3. **Mutation after corruption detected.** The sentinel's entire purpose is to guard the stores before curation acts; running the mutating pipeline against a colony the sentinel just flagged corrupt defeats that guard. (The `consolidation-seal` subcommand shares this ordering, but this changeset makes it fire automatically on every seal, which raises the stakes.)

**Fix:** Short-circuit before the pipeline when curation fails, and keep the two stages' outcomes independent in the summary:
```go
curResult, curErr := curation.NewOrchestrator(store, bus).Run(ctx, false)
// ... build ants/reportPath as today ...
if curErr != nil {
    reason := curErr.Error()
    if strings.Contains(reason, "sentinel abort") {
        reason = "curation sentinel detected corrupt stores: " + reason
    }
    fmt.Fprintf(os.Stderr, "colony sealed WITHOUT consolidation — %v\n", reason)
    return sealConsolidationSummary{Ants: ants, ReportPath: reportPath, Ran: false, Reason: reason}
}
consResult, consErr := pipeline.RunConsolidation(ctx)
```
If deliberately continuing past a non-sentinel curation failure is desired, then `QueenPromotedIDs` must be populated from `consResult` whenever `consResult != nil` — regardless of `curErr` — so the skip-set always reflects what the pipeline actually wrote. Either way, add a test with corrupt `pheromones.json` + a dual-eligible instinct asserting the action text appears exactly once in QUEEN.md.

## Warnings

### WR-01: `QueenPromotedIDs` conflates "eligible" with "actually promoted" — a failed promotion is skipped by the subordinate writer and still reported as promoted

**File:** `cmd/consolidation_lifecycle.go:339` (root cause context: `pkg/memory/pipeline.go:181-191`)
**Issue:** `summary.QueenPromotedIDs = append([]string{}, consResult.QueenEligible...)` — but `Pipeline.RunConsolidation`'s queen-promotion loop treats a failed `PromoteInstinct` as log-and-continue (`log.Printf("pipeline: queen promote %s failed: %v", ...)`) without removing the ID from `QueenEligible`. The doc comment on `QueenPromotedIDs` (lines 194-197) claims it "carries the actual instinct IDs pkg/memory's RunConsolidation promoted into QUEEN.md" — that is not what the code guarantees. When a promotion write fails (e.g., transient QUEEN.md permission error) for a dual-eligible instinct: the ID enters the skip-set, `completeSealRuntime` skips its own `promoteInstinctLocal` fallback (cmd/codex_workflow_cmds.go:465-471) **and** appends the ID to `promotedInstinctNames` — so CROWNED-ANTHILL.md reports an instinct as promoted that never reached QUEEN.md at all, and the one writer that could have recovered it was told not to.
**Fix:** Have `RunConsolidation` return the actually-promoted set — add a `QueenPromoted []string` field to `ConsolidationResult` appended only after `PromoteInstinct` returns nil — and build `QueenPromotedIDs` (and the skip-set) from that, keeping `QueenEligible` as the eligibility report it is.

### WR-02: The 30-second timeout cannot cancel the consolidation pipeline — the "can never hang a phase advance (T-162-12)" comment claims a guarantee the code does not enforce

**File:** `cmd/consolidation_lifecycle.go:19-24, 105, 263` (context: `pkg/memory/consolidate.go:64`, `pkg/storage/lock.go`)
**Issue:** `context.WithTimeout` does not preempt synchronous code — it only helps if the callee polls the context. `ConsolidationService.Run` never checks `ctx.Done()` between its steps; its `LoadJSON`/`SaveJSON` calls go through `storage.FileLocker`, whose `lock()` is a blocking flock with no timeout or retry deadline. A stale or contended write lock on `instincts.json` (e.g., a crashed process that held it, or a concurrent command) blocks `runPhaseEndConsolidation` indefinitely inside a single step, and the phase advance hangs with it — exactly the failure the const's comment says "can never" happen. The curation orchestrator checks `ctx.Done()` only *between* ant steps (pkg/agent/curation/orchestrator.go:153), so an in-step block is equally uncancellable at seal. Additionally, at seal one 30s budget is shared across the eight-ant pass *and* the pipeline, so a slow curation pass can starve consolidation even in the cooperative case.
**Fix:** Run the pipeline call in a goroutine and select on `ctx.Done()`, returning `Ran: false, Reason: "consolidation timed out after 30s"` on expiry (accepting the leaked goroutine for a process that exits shortly after), e.g.:
```go
done := make(chan struct{})
var result *learn.ConsolidationResult
var err error
go func() { result, err = pipeline.RunConsolidation(ctx); close(done) }()
select {
case <-done:
case <-ctx.Done():
    return phaseEndConsolidationSummary{Ran: false, Reason: "consolidation timed out"}
}
```
Alternatively wire `ctx` checks into `pkg/memory` — or at minimum soften the comment so it does not document a hang-proofing property that a test cannot currently verify (per CLAUDE.md's Definition of Done, an unenforceable claim should be removed).

### WR-03: "Report the full promoted set" is not true — pipeline-promoted instincts below the 0.8 bar are invisible to CROWNED-ANTHILL.md and hive promotion

**File:** `cmd/codex_workflow_cmds.go:462-490`
**Issue:** The comment at lines 465-471 says the skip-branch still counts pipeline-promoted instincts "so sealEnrichment.InstinctsPromoted and CROWNED-ANTHILL.md report the full promoted set." But the loop only ever inspects entries with `entry.Confidence >= 0.8` (line 464). `QueenEligible` requires only confidence >= 0.75 (plus 3 applications), evaluated against the *post-decay* confidence, while the loop filters on the *pre-consolidation snapshot* confidence. An instinct at snapshot confidence 0.78 with 3 successful applications is promoted into `## Instincts` by the pipeline, but never enters `promotedInstinctNames` — CROWNED-ANTHILL.md's "Promoted Instincts" section and the hive-eligible count silently omit it. Relatedly, any *new* instinct created by the pipeline's observation-promotion during this very seal is absent from `sealEligibleEntries` (snapshotted before consolidation at line 419), so it is skipped by hive promotion this seal even if it qualifies.
**Fix:** After the loop, append any `QueenPromotedIDs` entry not already in `promotedInstinctNames`:
```go
seen := make(map[string]struct{}, len(promotedInstinctNames))
for _, id := range promotedInstinctNames { seen[id] = struct{}{} }
for _, id := range sealConsolidation.QueenPromotedIDs {
    if _, ok := seen[id]; !ok { promotedInstinctNames = append(promotedInstinctNames, id) }
}
```
and either document the one-seal hive-promotion lag for newly created instincts or re-read entries for the hive loop after consolidation.

## Info

### IN-01: Comment claims `AETHER_HIVE_POLICY=off` is surfaced in withheld-wisdom warnings, but the off path emits no warning at all

**File:** `cmd/colony_prime_context.go:642-648` (context: `cmd/context_weighting.go:18-21`)
**Issue:** The comment justifying unconditional warning surfacing says a colony "needs to know whether the hub is empty, the domain didn't match, everything decayed to dormant, or AETHER_HIVE_POLICY=off disabled retrieval entirely." The fourth case never produces a warning: `readHiveWisdomEntriesForDomains` returns `nil` immediately when `automaticHiveReadEnabled()` is false, appending nothing to `fallbacks`. A user with the policy set to off (or a typo'd value, after the once-only stderr warning has fired) gets silence again.
**Fix:** Append a `"hive_wisdom: retrieval disabled by AETHER_HIVE_POLICY"` fallback in the early-return branch of `readHiveWisdomEntriesForDomains`, or trim the comment's claim.

### IN-02: `consolidation.seal` event published even when both curation and consolidation failed

**File:** `cmd/consolidation_lifecycle.go:300-306`
**Issue:** `bus.Publish(ctx, "consolidation.seal", ...)` fires unconditionally, before the failure check at line 311 — so the event stream records a seal consolidation on runs where `Ran` is false and nothing consolidated. The `consolidation-seal` subcommand shares this behavior, but a consumer of the event bus cannot distinguish a real consolidation from a failed one.
**Fix:** Publish only when `curErr == nil && consErr == nil`, or include a `"success"` field in the payload.

### IN-03: Two write disciplines on the same QUEEN.md file — atomic+locked vs. bare read-modify-write

**File:** `cmd/queen.go:663-737` (`writeLocalQueenText`, `ensureQueenInstinctsSection`, `promoteInstinctLocal`)
**Issue:** `pkg/memory`'s `QueenService` writes `.aether/QUEEN.md` via `store.AtomicWrite` (locked, temp-file rename), while `ensureQueenInstinctsSection` and `promoteInstinctLocal` use unlocked `os.ReadFile` + `os.WriteFile` on the same file. In-process the calls are serial, but a concurrent process (autopilot continue vs. a manual `consolidation-phase-end`) interleaving with the read-modify-write can lose an update or observe a torn write on non-atomic-rename filesystems.
**Fix:** Route `writeLocalQueenText` through `store.AtomicWrite(localQueenPath(), ...)` (absolute paths pass through `resolvePath`), matching the pipeline's discipline.

### IN-04: Dead parameter and obscure conditional construct

**File:** `cmd/consolidation_lifecycle.go:85`, `cmd/codex_workflow_cmds.go:462`
**Issue:** (a) `runPhaseEndConsolidation(phaseID int)` immediately discards its parameter (`_ = phaseID`); both callers pass `phase.ID` for nothing. (b) `if entries, err := sealEligibleEntries, sealEligibleErr; err == nil {` is a needless re-binding that shadows `err` and obscures that this is just a nil-error check on values computed 40 lines earlier.
**Fix:** (a) Drop the parameter until phase-scoped reporting exists, or use it in the summary/event payload. (b) Replace with `if sealEligibleErr == nil { for _, entry := range sealEligibleEntries {`.

### IN-05: Stale "opt in" phrasing in the disabled worker-read message

**File:** `cmd/hive.go:485` (hive-read `--for-worker` disabled branch)
**Issue:** The reason string reads "automatic cross-project wisdom injection is disabled; set AETHER_HIVE_POLICY=read to opt in". Post D-01/D-02 the feature is on by default; this branch is only reachable when the operator explicitly set `off` (or typo'd a value). "Opt in" misdescribes the state — the operator opted *out* — and suggesting `read` rather than clearing the variable (which restores full `promote`) may not be what they want.
**Fix:** Reword to e.g. "disabled by AETHER_HIVE_POLICY; unset it or set it to read/promote to re-enable".

---

_Reviewed: 2026-08-04T13:56:20Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
