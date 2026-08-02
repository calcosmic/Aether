---
phase: 164-research-feeds-planning
reviewed: 2026-08-02T16:28:17Z
depth: standard
files_reviewed: 32
files_reviewed_list:
  - .aether/commands/plan.yaml
  - .aether/ts-host/src/host.ts
  - .aether/ts-host/src/research-confidence.ts
  - .aether/ts-host/src/types.ts
  - .aether/ts-host/src/worker-dispatch.ts
  - .aether/ts-host/test/dispatch-field-fidelity.test.ts
  - .aether/ts-host/test/research-confidence-loop.test.ts
  - .aether/ts-host/test/research-confidence.test.ts
  - .aether/ts-host/test/research-escalation-degrade.test.ts
  - .claude/commands/ant-plan.md
  - .claude/commands/ant/plan.md
  - .opencode/commands/ant/plan.md
  - cmd/codex_plan_finalize.go
  - cmd/codex_plan.go
  - cmd/command_guide.go
  - cmd/phase_research_decision_cmd.go
  - cmd/phase_research_decision_test.go
  - cmd/phase_research_decision.go
  - cmd/phase_research_dispatch_test.go
  - cmd/phase_research_escalate_test.go
  - cmd/phase_research_escalate.go
  - cmd/phase_research_permission_boundary_test.go
  - cmd/phase_research_preserve_test.go
  - cmd/phase_research.go
  - cmd/plan_depth_proposal_manifest_test.go
  - cmd/plan_depth_proposal_test.go
  - cmd/plan_depth_proposal.go
  - cmd/plan_wrapper_cards_test.go
  - cmd/plan_wrapper_ceremony_test.go
  - cmd/platform_doc_hygiene_test.go
  - pkg/codex/permission_profile.go
findings:
  critical: 0
  warning: 7
  info: 4
  total: 11
status: issues_found
---

# Phase 164: Code Review Report (Re-review after gap closure)

**Reviewed:** 2026-08-02T16:28:17Z
**Depth:** standard
**Files Reviewed:** 32
**Status:** issues_found

## Summary

Re-review of Phase 164 (research-feeds-planning) covering the original implementation plus gap-closure plans 164-10 (`toWorkerDispatches` field fidelity) and 164-11 (`plan-research-escalate` fresh-colony fix, escalation degrade-to-warning).

**Verification performed:**
- `go build ./cmd/aether` and `go vet ./cmd ./pkg/codex` — clean.
- All phase-relevant Go tests (`PhaseResearch|PlanResearch|DepthProposal|PlanWrapper|OracleEscalation|...`) — pass.
- All four TS test files (37 tests) run against real `src/` — pass.
- `tsc --noEmit` — clean.
- `dist/` verified in sync with `src/`: `dist/host.js` and `dist/worker-dispatch.js` carry the CR-01/CR-03 fixes (`permission_profile` copy-through at dist/host.js:412, degraded `plan-research-escalate` at dist/host.js:754-769); `dist/types.d.ts` matches `src/types.ts`.
- All three prior critical findings (CR-01 dropped permission_profile, CR-02 dropped brief/silent field drops, CR-03 escalation crash) are confirmed fixed in source, covered by non-mocked tests (`dispatch-field-fidelity.test.ts` drives the real exported function; `research-escalation-degrade.test.ts` drives the real Go binary against a stateless cwd), and present in the compiled dist. See the verification table at the end.

No new critical findings. The remaining findings are correctness and robustness gaps in the surrounding orchestration; several (WR-06, WR-07, IN-04) carry over from the first review and were not in scope for the gap-closure plans — they remain open and are restated here so this report is the complete current picture. The most significant cluster: the TS host fabricates `status: "completed"` for research workers regardless of actual outcome, reports escalations as sent before the dispatch actually succeeds, and the evidence scorer's "verified citations" are never actually verified.

## Warnings

### WR-01: Research worker results are always reported as "completed" to plan-finalize, regardless of actual outcome

**File:** `.aether/ts-host/src/host.ts:1342-1358`
**Issue:** After `runResearchConfidenceLoop` finishes, `runDispatchedPlanCommand` maps every `phase_research` dispatch to a `WorkerResult` with hardcoded `status: "completed"` — even when every provider dispatch for that phase failed, timed out, or returned blockers on every iteration. The actual `DispatchResult`s from the loop (files_created, duration, blockers, real status) are discarded entirely. Go's `mergeExternalPlanResults` (cmd/codex_plan_finalize.go:928-934) hard-rejects any non-completed status, so this fabrication is what keeps a failed research Scout from aborting the whole plan — but it means the durable completion packet, spawn tree, and dispatch records all claim a worker completed work it never did. The Go-native path warns loudly on research failure (`TestResearchWorkerFailureWarnsLoudlyWithoutBlocking`); the TS host path is silent about worker failure and relies solely on the confidence score staying low.
**Fix:** Track per-phase worker outcomes in `ResearchLoopPhaseSummary` (e.g., `lastWorkerStatus`, `dispatchFailures`). When the final round's real status is not `completed`, keep the non-gating behavior but (a) emit a loud `Warning: research worker for phase N never completed (...)` ceremony line, and (b) put the real outcome in the summary text, e.g. `"Research phase 3: worker failed on all 4 iterations; artifact scored 20% (max_iterations_met)"`, so the completion packet's summary is honest even where its status must satisfy the finalizer contract.

### WR-02: `summary.escalations` records a phase as escalated before the Oracle dispatch is actually sent

**File:** `.aether/ts-host/src/host.ts:934-960`
**Issue:** In the escalation round, `escalations.push(state.phaseId)` happens immediately after `plan-research-escalate` returns a dispatch — before the Oracle worker is dispatched. If the subsequent `_dispatchWorkersRef(...)` wave throws (line 953), the catch only warns, and `summary.escalations` still reports every candidate phase as escalated. The `ResearchLoopSummary` doc comment promises "the escalated Oracle dispatch was sent exactly once" — after a wave failure that is false for every phase in the batch. `research-escalation-degrade.test.ts` covers the `callGoJSON` failure ("a failed escalation must not be recorded as successful") but not the dispatch-wave failure, so this path is unguarded by tests.
**Fix:** Collect phase IDs alongside their dispatches and only append to `escalations` after the dispatch wave resolves:
```ts
try {
  await _dispatchWorkersRef(escalationDispatchOpts, toWorkerDispatches(escalationDispatches));
  escalations.push(...escalatedPhaseIds);
} catch (err: unknown) { /* existing warning */ }
```

### WR-03: Build path parses `--max-iterations` / `--target` without the NaN guard the research path has

**File:** `.aether/ts-host/src/host.ts:1160-1166`
**Issue:** `runDispatchedBuildCommand` does `loopOpts.maxIterations = parseInt(parsed.maxIterations, 10)` with no `Number.isNaN` check and no clamping. `--max-iterations abc` stores NaN in `ConfidenceLoop`; `iterationCount >= NaN` and `confidence >= NaN` are both always false, silently disabling the iteration cap and the confidence target. The loop then only stops on budget exhaustion — up to ~20 real provider dispatch waves. The research path added exactly these guards this phase (`clampResearchConfidenceTarget` / `clampResearchMaxIterations` plus `isNaN` checks at host.ts:804-815); the build path was left inconsistent.
**Fix:** Mirror the research path:
```ts
if (parsed.maxIterations) {
  const n = parseInt(parsed.maxIterations, 10);
  if (!Number.isNaN(n)) loopOpts.maxIterations = Math.min(12, Math.max(1, n));
}
```
(Same shape for `targetConfidence` with the 70-99 clamp.)

### WR-04: `plan-research-escalate` consumes planning iteration state without the staleness check every other consumer applies

**File:** `cmd/phase_research_escalate.go:94-98`
**Issue:** The 164-11 fix seeds candidates from `loadPlanningIterationState()`, but skips the `planningIterationStateMatches` guard (goal/root/depth equality, cmd/codex_plan.go:1924-1932) that `planningManifestIterationSeed` applies before trusting that file. `planning/iteration-state.json` is only removed on a *successful* finalize (cmd/codex_plan_finalize.go:429); an abandoned planning loop leaves it behind indefinitely. On a later refresh/replan for the same colony (or after the goal changed), escalation resolves phase IDs against the stale abandoned draft's `PreviousPlanDraft` instead of the active colony plan — producing an Oracle brief for the wrong phase name/description, or "phase N not found" for a phase that does exist in the active plan. The test suite's `writeFreshColonyIterationState` fixture (phase_research_escalate_test.go:165-183) demonstrates the file is trusted with nothing but a run ID and iteration count.
**Fix:** Reject cross-run staleness before trusting the loaded state:
```go
seed := codexPlanIterationState{}
if loaded, ok := loadPlanningIterationState(); ok &&
    strings.TrimSpace(loaded.Goal) == strings.TrimSpace(goal) &&
    strings.TrimSpace(loaded.Root) == strings.TrimSpace(root) {
    seed = loaded
}
```
(Load state after `goal` is resolved; goal+root equality is available to this command even though the full loop-options match is not.)

### WR-05: `plan-research-approve` clobbers a corrupt pending-decisions.json with an empty file

**File:** `cmd/phase_research_decision_cmd.go:133-137,191`
**Issue:** `store.LoadJSON(pendingDecisionsFile, &file)` failure — which includes a *parse* failure on a corrupt file, not just file-not-found — resets `file` to `PendingDecisionFile{Decisions: []}`, and the command then unconditionally `SaveJSON`s that empty struct. `pending-decisions.json` is a shared file: it also carries discuss clarifications and plan-finalize failure blocker flags (`loadPlanFinalizeFlagsFile`, cmd/codex_plan_finalize.go:188-206). One `plan-research-approve` invocation against a corrupt file silently destroys all of it, eliminating any chance of manual recovery from the raw bytes.
**Fix:** Distinguish not-exist from parse failure:
```go
if err := store.LoadJSON(pendingDecisionsFile, &file); err != nil {
    if !errors.Is(err, os.ErrNotExist) {
        outputError(2, fmt.Sprintf("pending-decisions.json is unreadable, refusing to overwrite: %v", err), nil)
        return nil
    }
    file = PendingDecisionFile{Decisions: []PendingDecision{}}
}
```

### WR-06: "citationsVerified" citations are never verified — a fabricated `(Source: fake/path.ts)` earns the full 25-point bonus

**File:** `.aether/ts-host/src/research-confidence.ts:189-201,246`
**Issue:** (Carried over from the first review; not addressed by gap closure.) The module contract (D-11 header comment, lines 14-19) says the score is "dominated by checkable evidence (filled sections, verified citations, verified file paths)", and the result field is named `citationsVerified`. But `isCited` only pattern-matches that a bullet contains `(Source: ...)` with something path-shaped or URL-shaped — the path is never checked against the repo, unlike the Files-to-Study section, which does a real `fs.existsSync` with a traversal guard right below (lines 261-273). A Scout that invents `(Source: does/not/exist.go)` for every bullet collects the entire `CITATION_BONUS_MAX` (25 of 100 points) in the loop that decides whether to re-dispatch workers and whether to escalate to Oracle at deep/exhaustive depth.
**Fix:** For path-like sources, resolve against `input.repoRoot` with the same escape guard and count only existing paths as verified (URLs stay pattern-only, since the scorer is offline by design). At minimum, rename the field/comment to `citationsPresent` so the contract stops overclaiming.

### WR-07: An entirely-invalid `--flip` still resolves the whole research batch, discarding the user's override permanently

**File:** `cmd/phase_research_decision_cmd.go:153-189`
**Issue:** (Carried over from the first review; not addressed by gap closure.) `parseFlipPhaseIDs` correctly rejects unknown/unparsable tokens into `invalid_flips`, but the command then resolves every unresolved research decision per the Queen's recommendation anyway. A user who runs `aether plan-research-approve --flip 5` with a typo (phase 5 not in the batch) gets the entire batch consumed as-recommended; their intended flip was dropped, and it cannot be retried because all decisions are now `Resolved: true`. The `invalid_flips` field in the result is the only signal, and nothing pauses on it.
**Fix:** When `--flip` was supplied, all parsed tokens were invalid, and no `--approve-all`/`--auto` accompanied it, return an error without mutating any decision:
```go
if strings.TrimSpace(flip) != "" && len(flippedSet) == 0 && len(invalidFlips) > 0 && !approveAll && !auto {
    outputError(1, fmt.Sprintf("no valid phase IDs in --flip (%s); batch left unanswered", strings.Join(invalidFlips, ",")), nil)
    return nil
}
```

## Info

### IN-01: `planDepth` parameter of `plannedPhaseResearchDispatches` is dead

**File:** `cmd/phase_research.go:76`
**Issue:** The `planDepth` parameter is never used in the function body — depth gating moved entirely into `computePhaseResearchProposal`'s recommendations (the doc comment even explains this). The dead parameter misleads readers into thinking the dispatcher is still depth-aware, and every call site threads a value through for nothing.
**Fix:** Drop the parameter (or rename to `_` with a comment) and update the call sites/tests.

### IN-02: `extractPath` fails on realistic Files-to-Study bullets, silently forfeiting the 15-point files bonus

**File:** `.aether/ts-host/src/research-confidence.ts:204-206`
**Issue:** `extractPath` strips only the bullet marker; a bullet like `` - `cmd/exporter.go` — retry entry point `` (backticks plus an annotation — the natural way an LLM Scout writes this section) resolves to a "path" containing backticks and prose, so `existsSync` always fails and `filesVerified` is 0. The 15-point bonus quietly disappears for well-formed research, biasing the loop toward extra iterations.
**Fix:** Strip surrounding backticks and take the first whitespace-delimited token: `bullet.replace(/^[-*]\s+/, "").replace(/`/g, "").trim().split(/\s+/)[0]`.

### IN-03: `parsePhaseResearchDecisionDescription` mis-splits phase names containing "): "

**File:** `cmd/phase_research_decision_cmd.go:40-45`
**Issue:** The parser finds the *first* `"): "` in the remainder, so a user-authored phase name like `"Fix exporter (v2): cleanup"` splits as name=`"Fix exporter (v2"`, reason=`"cleanup): <real reason>"`. Recommend and phase ID (parsed earlier) survive, so approve/flip still work — only the resolution record's name/reason text is mangled. Round-tripping structured data through a human-readable Description string is inherently brittle.
**Fix:** Use `strings.LastIndex(rest, "): ")` — the reason comes from the fixed `phaseResearchReasons` lookup table and cannot contain `"): "`, so the last occurrence is always the real delimiter.

### IN-04: Existing-plan manifest branch shows the research warning but `plan-research-approve` cannot act on it

**File:** `cmd/codex_plan.go:1018-1024`, `cmd/codex_plan.go:791-795`
**Issue:** (Carried over from the first review.) The existing-plan early-return branch calls `computePhaseResearchProposalFields(..., persist=false)`, so no `PendingDecision` records are written — yet the result still carries `research_awaiting_approval: true` and the warning telling the user to run `aether plan-research-approve --approve-all`. Running it on that path finds zero unresolved candidates and reports "no phases require research", contradicting the warning the runtime just displayed. Harmless (the refresh path re-proposes and persists), but the runtime is issuing an instruction it knows cannot take effect on this branch.
**Fix:** On the `persist=false` branch, suppress `AwaitingApproval`/`Warning` (or swap the warning text for "run `aether plan --refresh` to open the research batch").

---

## Prior critical-finding verification (CR-01 / CR-02 / CR-03)

| Prior finding | Status | Evidence |
|---|---|---|
| CR-01 — `toWorkerDispatches` dropped `permission_profile`; every real Scout dispatch rejected at Go's exact-equality check | Fixed | src/host.ts:477-479 copies it through; dist/host.js:412-413 matches; `dispatch-field-fidelity.test.ts` drives the real function and mirrors Go's `repositoryReadOnlyCastes` map from source; `phase_research_permission_boundary_test.go` proves a real dispatch's own emitted profile resolves through `internalWorkerConfig`/`ResolvePermissionProfile` |
| CR-02 — `brief` and other fields silently dropped at the Go-to-worker boundary | Fixed | Test 1 of `dispatch-field-fidelity.test.ts` enforces a full-field invariant with an explicit `INTENTIONALLY_UNMAPPED` classification set; `brief` maps to `task_brief` with host-injected precedence, covered by a dedicated test |
| CR-03 — `plan-research-escalate` crashed fresh colonies mid-loop; any escalation failure killed the plan run | Fixed | cmd/phase_research_escalate.go:94-98 seeds candidates from planning iteration state with colony-plan fallback (see WR-04 for a remaining staleness gap); host.ts:927-960 wraps both the Go call and the dispatch wave in degrade-to-warning handling; `research-escalation-degrade.test.ts` drives the real binary against a stateless cwd (see WR-02 for a remaining reporting gap on the dispatch-wave path) |

---

_Reviewed: 2026-08-02T16:28:17Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
