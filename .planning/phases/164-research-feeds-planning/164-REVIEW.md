---
phase: 164-research-feeds-planning
reviewed: 2026-08-02T13:30:43Z
depth: standard
files_reviewed: 26
files_reviewed_list:
  - .aether/commands/plan.yaml
  - .aether/ts-host/src/host.ts
  - .aether/ts-host/src/research-confidence.ts
  - .aether/ts-host/test/research-confidence-loop.test.ts
  - .aether/ts-host/test/research-confidence.test.ts
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
  critical: 3
  warning: 7
  info: 6
  total: 16
status: issues_found
---

# Phase 164: Code Review Report

**Reviewed:** 2026-08-02T13:30:43Z
**Depth:** standard
**Files Reviewed:** 26
**Status:** issues_found

## Summary

Phase 164 wires a research decision/confidence/escalation pipeline through the Go runtime and TypeScript host: a batched research proposal (Queen recommends research/skip per phase), a `plan-research-approve` verb, approval-gated research Scout dispatches, an evidence-based per-phase confidence loop, and Scout-to-Oracle escalation on stalled deep/exhaustive research.

The Go-side proposal/decision/gating logic is solid and well-tested. However, the runtime execution path — the TS host actually dispatching the research Scouts this phase gates — has three critical defects that the passing test suites cannot see, because every test mocks or simulates the dispatch boundary. This is precisely the failure class this project's own CLAUDE.md documents: the machinery exists but has never been proven by execution. Specifically: (1) the phase downgraded Scout's canonical permission profile without updating the TS-side fallback, so every real Scout dispatched through `aether host plan` now fails a strict profile-equality check in the Go adapter; (2) the TS host drops the `brief` field when converting plan dispatches, so research Scouts never receive the six-section mission the confidence scorer grades against; (3) the escalation command resolves phases from colony state, which is empty during a fresh colony's planning loop — the only time research dispatches exist — so a stalled deep/exhaustive phase crashes the entire plan run.

Warnings cover state mutation on the plan-only/dry-run path, a flip-typo failure mode that silently commits the whole batch, a dead-end approval flow on the existing-plan branch, a citation "verifier" that verifies nothing, and a convergence-free iteration loop.

## Critical Issues

### CR-01: Scout permission downgrade breaks every real Scout dispatch on the TS-host path

**File:** `pkg/codex/permission_profile.go:53-55` (change site), `.aether/ts-host/src/host.ts:437-477` (`toWorkerDispatches`), `.aether/ts-host/src/worker-dispatch.ts:389-401` (fallback, out-of-list fix site)
**Issue:** This phase removed `"scout"` from `repositoryReadOnlyCastes`, making Scout's canonical profile `workspace_write` (necessary so research Scouts can write `.aether/data/phase-research/`). But two consumers were not updated:

1. `toWorkerDispatches` in host.ts does not copy `permission_profile` from the manifest dispatch (the Go manifest carries it — `codexPlanningDispatch.PermissionProfile`, stamped by `attachPlanningDispatchSkillAssignments`), so every plan/continue dispatch reaches `dispatchRealWorker` with `permission_profile` undefined.
2. `dispatchRealWorker` then falls back to the TS-side `permissionProfileForCaste`, which still maps `scout` → `repository_read_only`.

The Go adapter (`cmd/internal_worker_adapter.go:377` → `codex.ResolvePermissionProfile`) requires the requested profile to equal the canonical one exactly and errors otherwise ("permission profile mismatch for caste \"scout\": requested \"repository_read_only\" but canonical profile is \"workspace_write\""). Result: on any non-simulated `aether host plan` run, the base Scout and every research Scout dispatch fails immediately (marked `status: "failed"`, worker never runs). The research confidence loop then grades a nonexistent artifact at ~30 each round until it stalls. Before this phase, the TS fallback happened to match the canonical profile, which is why dropping `permission_profile` in `toWorkerDispatches` was latent rather than fatal. All ts-host tests mock `dispatchWorkers` or run `--simulate`, so nothing catches this.
**Fix:** Copy the manifest profile through the conversion, and align the TS fallback:
```ts
// host.ts toWorkerDispatches
if (dispatch.permission_profile !== undefined) {
  workerDispatch.permission_profile = dispatch.permission_profile;
}
```
```ts
// worker-dispatch.ts permissionProfileForCaste — scout is no longer read-only
const readOnly = normalized === "includer";
```
Add a non-mocked test that runs a plan dispatch request through `internalWorkerConfig`/`ResolvePermissionProfile` with the TS-built request for caste `scout` — per the project's Definition of Done, a command that fails when the two sides disagree.

### CR-02: `toWorkerDispatches` drops `brief` — research Scouts and escalated Oracles never receive their mission

**File:** `.aether/ts-host/src/host.ts:437-477` (`toWorkerDispatches`), consumed at `host.ts:821` and `host.ts:914` (`runResearchConfidenceLoop`), `.aether/ts-host/src/worker-dispatch.ts:278` (`task_brief: dispatch.task_brief ?? dispatch.task`)
**Issue:** Go emits the composed research mission in `codexPlanningDispatch.Brief` (JSON key `brief`): the six-section artifact template, territory-survey framing, scope constraints, and — for escalation — the "Escalation Notice" preamble (`renderPhaseResearchBrief`, `phaseResearchEscalationDispatch`). `toWorkerDispatches` copies `task_brief` (a host-injected field that plan dispatches never have) but never maps `brief`, so `dispatchRealWorker` sends `task_brief = dispatch.task` — the one-line "Research domain knowledge for phase 2: …". The worker is therefore never told the six-section contract that `ResearchConfidenceEvaluator` grades against (`REQUIRED_SECTIONS`, citation format, Files-to-Study). The confidence loop this phase built scores workers against a spec they never saw; the Oracle escalation loses both its stall context and the entire research mission. The wrapper-platform path (Claude/OpenCode Task spawning from the manifest) passes `brief` verbatim per the .md instructions, but the confidence loop and escalation dispatches run inside the TS host on all platforms, so the loop's own dispatches are always briefless.
**Fix:** Map the brief in the conversion:
```ts
if (dispatch.task_brief !== undefined) workerDispatch.task_brief = dispatch.task_brief;
else if ((dispatch as PlanDispatchLike).brief !== undefined) {
  workerDispatch.task_brief = (dispatch as PlanDispatchLike).brief;
}
```
Pin it with a test that asserts the `GoWorkerDispatchRequest.task_brief` for a `phase_research` dispatch contains `"## Output"` and `"## Files to Study"`.

### CR-03: `plan-research-escalate` cannot resolve phases during a fresh planning loop — a deep/exhaustive stall crashes the whole plan run

**File:** `cmd/phase_research_escalate.go:83` (`phaseResearchCandidates(state, codexPlanIterationState{})`), `.aether/ts-host/src/host.ts:899-906` (unguarded `_callGoJSONRef` call)
**Issue:** Research dispatches only exist from iteration 2 of a planning loop, sourced from `seed.PreviousPlanDraft` — which lives in the planning iteration state file, not in `COLONY_STATE.json` (intermediate `plan-finalize` saves `planningIterationStateRel`, not `state.Plan.Phases`; see `cmd/codex_plan_finalize.go:829-851`). The escalate command passes a zero-value `codexPlanIterationState{}`, so its candidate set falls through to `state.Plan.Phases`, which is empty for a fresh colony mid-loop. It returns `outputError(1, "phase N not found in the current plan")`; `_realCallGoJSON` throws on `ok:false`; `runResearchConfidenceLoop` has no try/catch around the escalation block, so the error propagates to `main` and the entire plan run exits 1 — after all research iterations already ran, and before `plan-finalize` is ever called. The exact scenario escalation was built for (deep/exhaustive fresh plan whose research stalls) is a guaranteed crash — directly contradicting the phase's own "research is enrichment, never a gate" invariant (D-08). On a refresh, `state.Plan.Phases` exists but iteration-≥2 draft candidates are renumbered `i+1` from the draft, so the escalate lookup can bind the wrong phase's name/description. The CLI test (`phase_research_escalate_test.go:96`) only exercises a colony state that already contains the phase, and the TS test mocks the Go call, so neither covers the live path.
**Fix:** In `planResearchEscalateCmd`, load the planning iteration state and pass it through:
```go
seed := loadCodexPlanIterationState(root) // the same seed source runCodexPlanPlanOnly uses
candidates := phaseResearchCandidates(state, seed)
```
And in `runResearchConfidenceLoop`, wrap each escalation call in try/catch: log a loud warning, skip that phase's escalation, and continue — a failed escalation must degrade, not gate.

## Warnings

### WR-01: Plan-only manifest generation (including `aether host plan --dry-run`) mutates pending-decisions state

**File:** `cmd/codex_plan.go` (`computePhaseResearchProposalFields`, `persist=true` at the fresh-plan call site in `runCodexPlanPlanOnly`), `.aether/ts-host/src/host.ts:583-642` (`runDryRunDispatchedCommand`, "Fetch manifest (read-only)")
**Issue:** Every manifest fetch on the fresh-plan branch deletes unresolved in-scope research decisions and appends fresh ones to `pending-decisions.json`. The TS host's `--dry-run` path calls the same `plan --plan-only` args, so a dry run writes state — violating the project's documented corollary ("An inspection or `--dry-run` command must not mutate state", locked elsewhere by dedicated tests). Additionally, after the batch is answered, every subsequent manifest fetch re-appends unresolved duplicates of already-resolved decisions (only resolved ones are kept, then a fresh unresolved decision is appended for every proposal phase). These duplicates are never resolved or cleaned, so completed plans leave permanent unresolved `research-decision` entries in the decisions file.
**Fix:** Thread a `dryRun`/read-only flag from the host into the Go call (or have Go skip persistence when a `--dry-run` marker is present), and skip appending a fresh unresolved decision for a phase that already has a resolved decision in the current scope. Add `TestPlanPlanOnlyDryRunDoesNotMutatePendingDecisions` mirroring the existing dry-run mutation guards.

### WR-02: An entirely-invalid `--flip` still resolves the whole batch; `approve-all` flag is dead

**File:** `cmd/phase_research_decision_cmd.go:118-211`
**Issue:** `plan-research-approve --flip 33` (typo for 3) reports `invalid_flips: ["33"]` but still resolves every unresolved decision with the Queen's recommendation and returns `ok`. The user's intended override is silently lost; the decisions are now resolved and the gate is closed for this run. Separately, `approveAll` is assigned (including the no-flag default) but never read afterwards — the resolution loop unconditionally resolves everything in scope, so `--approve-all` is purely decorative and `--flip` with zero valid IDs behaves identically to `--approve-all`.
**Fix:** When `--flip` was provided and every token is invalid, return an error without resolving anything:
```go
if strings.TrimSpace(flip) != "" && len(flippedSet) == 0 && len(invalidFlips) > 0 {
    outputError(1, fmt.Sprintf("no valid phase IDs in --flip (%s); nothing was resolved", strings.Join(invalidFlips, ",")), nil)
    return nil
}
```
Either remove `approveAll` or make the batch resolution conditional on it.

### WR-03: Existing-plan branch shows the research card and "not answered" warning but approval can never take effect

**File:** `cmd/codex_plan.go` (existing-plan branch of `runCodexPlanPlanOnly`: `computePhaseResearchProposalFields(..., persist=false)`)
**Issue:** With an active plan and no `--refresh`, the manifest carries a non-empty `research_proposal_card` and `research_awaiting_approval: true` with the warning telling the user to run `aether plan-research-approve --approve-all`. But `persist=false` means no decisions were written, so `plan-research-approve` finds zero candidates and returns "no phases require research"; this branch also emits no dispatches (`requires_finalizer: false`). The wrapper instruction loop ("approve, then request a fresh manifest so gated `phase_research` dispatches appear") can never complete — the user is directed into a dead end.
**Fix:** In the existing-plan/no-refresh branch, either suppress the card and warning (set `AwaitingApproval=false`, empty card) or annotate the card that research applies only to a `--refresh` run.

### WR-04: Citation "verification" verifies nothing, and file-path extraction misses common bullet formats

**File:** `.aether/ts-host/src/research-confidence.ts:189-205` (`isCited`, `extractPath`)
**Issue:** `citationsVerified` counts bullets that merely contain `(Source: …)` with a path-shaped token or URL — the cited path's existence is never checked, so a researcher (an LLM) can fabricate `(Source: made/up.ts)` on every bullet and collect the full 25-point `CITATION_BONUS_MAX`. This undercuts D-11's core claim that the score is "dominated by checkable evidence" — two of the three evidence bonuses (citations 25, self-assessment 10) are self-reportable; only sections and file existence are checked. Meanwhile `extractPath` returns the whole bullet body, so the common format ``- `cmd/foo.go` — why it matters`` never resolves to an existing file and scores 0 in the files bonus, penalizing honest, well-annotated artifacts.
**Fix:** In `isCited`, when the source token is path-like (not a URL), resolve it against `repoRoot` with the same escape guard and require `fs.existsSync`. In `extractPath`, strip backticks and trailing annotations:
```ts
const stripped = bullet.replace(/^[-*]\s+/, "");
const m = /`([^`]+)`/.exec(stripped);
return (m ? m[1]! : stripped.split(/\s+[—–-]{1,2}\s+/)[0]!).trim();
```

### WR-05: Research loop re-dispatches an identical mission every iteration — it cannot converge, and `--accept` does not suppress escalation

**File:** `.aether/ts-host/src/host.ts:819-884` (`runResearchConfidenceLoop` round loop)
**Issue:** Unlike the build loop (which injects previous-iteration blockers into `task_brief`), each research round re-dispatches `state.dispatch` unchanged — the Scout is never told its score, unfilled sections, uncited bullets, or unverified files. With a deterministic scorer and an unchanged mission, the delta between rounds hovers near zero, so `diminishing_returns` (window 2, threshold 5) trips at iteration 3 in nearly every real run: two of the three Scout dispatches are budget spent re-running the same mission, and at deep/exhaustive depth the Oracle escalation becomes the near-certain path rather than an exception. Additionally, `parsed.accept` only converts still-continuing phases to `"accepted"` (`host.ts:866`); a phase whose loop returns `diminishing_returns` on the same iteration the user ran with `--accept` still escalates to a heavyweight Oracle, against the user's explicit accept.
**Fix:** Before re-dispatching an active phase, append the evaluator breakdown to the brief (e.g., "Previous iteration scored 62%: sections filled 4/6 (missing Gotchas, Files to Study); 1/4 bullets cited; 0/2 files verified"). Gate the escalation-candidate push on `!parsed.accept`.

### WR-06: Stale "Scout repository_read_only must remain host-enforced" instructions across YAML, wrappers, and command guide

**File:** `.aether/commands/plan.yaml:50`, `.claude/commands/ant/plan.md:71`, `.claude/commands/ant-plan.md:71`, `.opencode/commands/ant/plan.md:71`, `cmd/command_guide.go:211`
**Issue:** The same phase that changed Scout's canonical profile to `workspace_write` re-wrote all five of these files and left the mandate that "Scout's `repository_read_only` profile must remain host-enforced; never substitute … a prompt-only promise." The manifest now delivers scouts with `workspace_write` plus exactly the prompt-only behavioral restriction the guardrail forbids. A wrapper that obeys the instruction must either treat every manifest as a broadened-profile violation and refuse dispatch, or enforce a read-only boundary that prevents the research Scout from writing its only deliverable. The plan.yaml `drift_guard` explicitly requires these files to move together; they did not, and no doc-hygiene test pins the claim to the runtime (CLAUDE.md: "A documentation claim about runtime behaviour must be testable or removed").
**Fix:** Rewrite the guardrail in all five locations to match the runtime, e.g.: "Scouts run `workspace_write` with a behavioral restriction to `.aether/data/phase-research/`; treat `behavioral_restrictions` as required behavior, not a sandbox claim; do not broaden any profile." Add an assertion to `platform_doc_hygiene_test.go` that the plan wrappers do not claim a read-only Scout sandbox.

### WR-07: `resolvePhaseResearchSection` can truncate mid-rune

**File:** `cmd/phase_research.go:220-226`
**Issue:** `content[:phaseResearchBriefBudgetChars]` slices at a byte offset. When no `\n\n` exists past the midpoint, the cut lands wherever byte 3500 falls — possibly inside a multi-byte UTF-8 sequence (research artifacts routinely contain em dashes and typographic quotes), producing an invalid-UTF-8 brief that JSON marshaling mangles to U+FFFD.
**Fix:** Back the cut off to a rune boundary before appending the truncation notice:
```go
cut := content[:phaseResearchBriefBudgetChars]
for len(cut) > 0 && !utf8.ValidString(cut) {
    cut = cut[:len(cut)-1]
}
```
(or use `strings.ToValidUTF8` / walk back while `utf8.RuneStart` is false).

## Info

### IN-01: Decision description parsing breaks on phase names containing "): "

**File:** `cmd/phase_research_decision_cmd.go:18-53`
**Issue:** `parsePhaseResearchDecisionDescription` splits on the first `"): "`; a phase named `OAuth (v2): migration` yields a truncated name and a polluted reason in the resolution record. PhaseID and Recommend parse first, so flip/approve behavior is unaffected — only the durable record text is wrong.
**Fix:** Use `strings.LastIndex(rest, "): ")` or store the recommendation as structured fields rather than re-parsing the display string.

### IN-02: Flip-to-research resolution reads "user overrode: research research on phase N"

**File:** `cmd/phase_research_decision.go:225-236`
**Issue:** `"user overrode: %s research on phase %d"` with `oppositeRecommend("skip") == "research"` produces the doubled word. Cosmetic but this string is the durable audit record.
**Fix:** Use direction-specific phrasing: `"user overrode: research phase %d"` / `"user overrode: skip research on phase %d"`.

### IN-03: `omitempty` on struct-typed manifest field has no effect

**File:** `cmd/codex_plan.go:251` (`DepthProposal depthProposal json:"depth_proposal,omitempty"`)
**Issue:** encoding/json never treats a struct value as empty, so the tag is dead; the field always serializes.
**Fix:** Drop `omitempty` or make the field `*depthProposal`.

### IN-04: Planning-depth and verification-depth knobs share one synthetic reason

**File:** `cmd/plan_depth_proposal.go:112-117` (`smartOrExplicitReason`)
**Issue:** Both knobs call `renderSmartDepthReason(colony.Phase{ID: 1}, len(state.Plan.Phases))` — a fabricated phase — so the card prints the identical reason line twice, and for a fresh colony (`totalPhases == 0`) the position heuristics run on nonsense inputs. The reason also has nothing to do with task decomposition for the planning-depth knob.
**Fix:** Give each knob its own reason source (goal-size heuristic for planning depth; the actual smart-default inputs for verification depth).

### IN-05: Research dispatch results always report `status: "completed"`

**File:** `.aether/ts-host/src/host.ts:1296-1311` (`researchMappedResults`)
**Issue:** Every `phase_research` dispatch is mapped to `status: "completed"` regardless of whether its rounds actually failed (e.g., the CR-01 permission failure) or whether its `task_id` was malformed and therefore never dispatched at all (silently dropped by `researchDispatchPhaseId`). Finalize's template-marker detection catches the missing artifact, but the completion packet's per-worker record lies.
**Fix:** Derive the status from the loop summary (missing phase summary or a `stopReason` of empty/failed → `"failed"`).

### IN-06: PhaseName is interpolated unsanitized into the research card and decision records

**File:** `cmd/phase_research_decision.go:194-198, 209-219`
**Issue:** T-164-04 protects the Reason line via the fixed lookup table, but `rec.PhaseName` — Route-Setter/LLM-authored text — is interpolated verbatim into the card the wrapper prints and into `PendingDecision.Description`. A crafted phase name can inject instruction-looking lines into the card the Queen relays verbatim.
**Fix:** Run phase names through the existing pheromone-style sanitizer (or at minimum strip newlines and cap length) before rendering into the card and decision descriptions.

---

_Reviewed: 2026-08-02T13:30:43Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
