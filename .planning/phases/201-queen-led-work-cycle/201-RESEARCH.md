# Phase 201: Queen-Led Work Cycle - Research

**Researched:** 2026-09-10
**Domain:** Go runtime work-cycle consolidation (Queen team judgment, build/continue/run verification collapse, durable attempt/result model, bounded recovery, autopilot, per-job telemetry) inside the Aether Go binary. No new external services, frameworks, or packages.
**Confidence:** HIGH — every architectural claim below is grounded in files read this session (cited with path:line) or in the phase's own CONTEXT.md/REQUIREMENTS.md/capability ledger. The one LOW-confidence area is the numeric WORK-08 turnaround target, which D-13 itself says must come from measurement taken *during* this phase, not from research.

## Summary

Phase 201 is not a feature-build phase — it is the fourth in a chain of *mechanism-study-then-synthesize* phases (`SYNTH-03` mirrors `SYNTH-01`/`SYNTH-02` from Phases 199/200). Its job is to finish unifying three things that today are real, tested, and mostly *already built* in isolation but never joined into one story: (1) the Queen's team-selection judgment (`cmd/queen_judgement.go`, `cmd/coherent_jobs.go` — both already test-locked from v1.27), (2) three separate `continue` code paths that must collapse onto one accept/verify/advance function, and (3) a build-time verification stage that duplicates continue-time review on every phase. The planner's first wave must reproduce the exact workflow Phases 199 and 200 used: a plan that reads Classic evidence, audits the current Go mechanism, and signs a `201-CLASSIC-SYNTHESIS.md` (using `.planning/research/v1.28-classic-synthesis-template.md`) *before* any implementation plan is approved — this is a hard gate (SYNTH-07), not a nice-to-have.

The bulk of the phase's raw material already exists as tested Go code: `queenCasteJudgement` (cmd/queen_judgement.go:29-68) already reconciles a Queen proposal against safety floors; `coherentJobPlan` (cmd/coherent_jobs.go:47-84) already groups dependency-safe tasks into jobs with waves; `buildAttemptRecord` (cmd/build_attempt.go:190-244) already carries append-only attempt history, partial-credit parent/child linkage (`ParentAttemptID`, `ParentJobName`), and a `CheckFix` sub-record for one bounded automatic repair. `autopilotRepairLedger` (cmd/autopilot_policy.go:523-533) already implements a budgeted, receipted repair loop. What is genuinely missing is: (a) collapsing `cmd/codex_continue.go` / `cmd/codex_continue_plan.go` / `cmd/codex_continue_finalize.go` onto one authoritative function (D-04), (b) eliminating the double-verification named in `CONCERNS.md` (build-side Watcher + continue-side Probe/Auditor on the same phase), (c) making every outcome — not just success — render the full closeout ceremony (D-05), and (d) building the per-job telemetry that doesn't exist yet at all (WORK-08).

**Primary recommendation:** Treat this phase as *wiring and consolidation*, not new-system design. Every CONTEXT.md decision (D-01 through D-16) should resolve to "extend/unify existing tested Go type X" rather than "invent new type." The planner should sequence: Wave 1 = mechanism study + synthesis doc (blocks everything else, per SYNTH-07); Wave 2+ = one-verification-boundary collapse (D-01..D-04, touches the three continue lanes and `runDeterministicFloor`); parallel/later waves = result-truth cards (D-05..D-08), bounded repair (D-09..D-12, extends `autopilotRepairLedger`/`checkFixAttemptRecord`), and telemetry (D-13..D-16, net-new instrumentation on the existing attempt/dispatch types).

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Queen team/boundary judgment (D-01, D-02) | Go runtime (`cmd/queen_judgement.go`) | Wrapper narration (`.claude/commands/ant/build.md`, `.opencode/...`) | Judgment must be a validated decision with a recorded reason, not prompt text; wrappers only render it. `TestQueenChoiceReachesTheDispatchList` pattern already enforces this for team choice — D-01 extends the same discipline to verification-boundary choice. |
| One accept/verify/advance function (D-04) | Go runtime (new shared function, replacing per-lane logic in `cmd/codex_continue.go`, `cmd/codex_continue_plan.go`, `cmd/codex_continue_finalize.go`) | — | `CONCERNS.md` names this exact gap: "Three separate continue implementations ... must stay behaviorally identical. ... No test currently asserts end-to-end parity." Collapsing to one function makes parity structural instead of a maintenance burden. |
| Deterministic checks (build/vet/tests/lint) | Go runtime (`cmd/deterministic_floor.go:48 runDeterministicFloor`) | — | Already the sole source of a pass (`TestDeterministicFloorIsTheOnlySourceOfAPass`, STATE.md Phase 193). D-01 must not weaken this; it only changes *where reviewer judgment* sits, never the floor. |
| Result truth / outcome cards (D-05..D-08) | Go runtime (state/attempt persistence) | Wrapper rendering (Queen-voice closeout ceremony) | `buildAttemptRecord` already has the append-only history and partial-credit fields; D-05..D-08 mostly require *rendering* rules on data that is already durable, plus new fields for cost/time-per-attempt and orphan-file naming. |
| Bounded repair / checkpoint (D-09..D-12) | Go runtime (`cmd/autopilot_policy.go` repair ledger, `cmd/build_attempt.go` `checkFixAttemptRecord`) | Midden/signal pipeline (`pkg/colony/midden.go`, `cmd/memory_feed.go`) | Checkpoint/rollback and one-bounded-attempt semantics already exist for autopilot and for the single check-fix attempt; D-09..D-12 need to be the *general* recovery story both build/continue and Swarm/repair waves use, not a second competing implementation. |
| Per-job telemetry (D-13..D-16) | Go runtime (new instrumentation on `codexBuildDispatch`/`buildAttemptWorkerRun`) | `/ant-status` drill-down rendering | Nothing today measures queue/preflight/model/tool-call/context/work/verification/wait per job. This is genuinely new code, not a restoration — CONTEXT.md explicitly frames WORK-08 as "measure first, then set the target." |
| Goal-level autopilot transitions (WORK-07) | Go runtime (`cmd/codex_run.go`/`cmd/autopilot_*.go`) | Wrapper (`/ant-run`) | Autopilot policy (trigger specs, authority proposals, repair ledger) is already Go-owned; this phase's job is making it consume the *same* accepted result model as build/continue (D-04), not rebuild autopilot. |

## Standard Stack

This phase adds **no new external dependencies**. It is a Go-internal consolidation and instrumentation phase against the existing module.

| Component | Version | Purpose | Why Standard |
|---|---|---|---|
| Go toolchain | 1.26.5 `[VERIFIED: go.mod:3]` | Language/runtime for all changes | Existing project toolchain; no upgrade needed for this phase's scope. |
| `testing` (stdlib) | — | All new/extended tests | Project convention: no testify/gomega, table-driven Go tests only `[VERIFIED: .planning/codebase/TESTING.md:8-11]`. |
| `encoding/json` (stdlib) | — | Attempt/receipt/telemetry serialization | Every existing durable record (`buildAttemptRecord`, `autopilotRepairLedger`, `WorkerHandoff`) already uses stdlib JSON tags; new telemetry fields follow the same `omitempty` compatibility discipline `[VERIFIED: cmd/build_attempt.go:190-244]`. |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Extending `buildAttemptRecord`/`autopilotRepairLedger` in place | A new parallel "work-cycle" package/schema | Rejected direction: `.planning/codebase/CONCERNS.md` already documents duplicate-implementation drift risk (three continue lanes, dual verification) as this repo's *named, recurring* failure mode. A fourth parallel record type would reproduce the exact problem the phase is chartered to fix. |
| A generic percentage/aggregate telemetry field | Named per-segment timing (`queue, preflight, model, tool-call, context, work, verification, wait`) | CONTEXT.md D-13/D-14 explicitly name these eight segments and require "the biggest term" to be identifiable in the one-line closeout — an aggregate number cannot answer "6m of it verification." |

**Installation:** None — no `go get` / `npm install` required for this phase.

**Version verification:** Go module and toolchain confirmed via `go.mod:3` (`go 1.26.5`) read this session. No new packages are introduced, so no registry lookups apply.

## Package Legitimacy Audit

**Not applicable.** This phase installs no external packages (no npm/pip/cargo/go-get additions). All work extends existing Go types in `cmd/` and `pkg/colony/` using the stdlib only. If a planner discovers mid-phase that a new dependency is genuinely required (e.g., a specific timing/tracing library for WORK-08), the Package Legitimacy Gate protocol must be run at that point and this section amended — do not silently add a dependency without it.

## Architecture Patterns

### System Architecture Diagram

```text
                      /ant-build N                         /ant-continue                    /ant-run
                          │                                      │                                │
                          ▼                                      ▼                                ▼
          ┌───────────────────────────┐          ┌───────────────────────────┐      ┌───────────────────────────┐
          │  Queen team judgment      │          │  ONE accept/verify/       │      │  Autopilot policy         │
          │  (cmd/queen_judgement.go) │          │  advance function (NEW,   │      │  (cmd/autopilot_*.go)     │
          │  queenApplyJudgement()    │          │  replaces 3 continue      │      │  trigger specs, repair    │
          │  -> proposed/final/added/ │          │  lanes per D-04)          │      │  ledger, checkpoint pause │
          │  dropped/refused + reason │          └─────────────┬─────────────┘      └─────────────┬─────────────┘
          └─────────────┬─────────────┘                        │                                  │
                         ▼                                      │                                  │
          ┌───────────────────────────┐                         │                                  │
          │  Coherent job planning    │                         │  calls the SAME function ────────┘
          │  (cmd/coherent_jobs.go)   │                         │  (parity becomes structural,
          │  planCoherentJobs() ->    │                         │  not hand-maintained)
          │  jobs + waves + decisions │                         ▼
          └─────────────┬─────────────┘          ┌───────────────────────────┐
                         ▼                        │ runDeterministicFloor()   │   <- unskippable floor
          ┌───────────────────────────┐          │ (cmd/deterministic_floor  │      at every depth,
          │  Dispatch + execution     │          │  .go:48) build/vet/tests/ │      every boundary
          │  (codexBuildDispatch,     │          │  lint/claimed-files       │
          │  buildAttemptRecord)      │          └─────────────┬─────────────┘
          └─────────────┬─────────────┘                        ▼
                         ▼                        ┌───────────────────────────┐
          ┌───────────────────────────┐          │  D-01: Queen's per-phase   │
          │  Two-stage receipt        │◄────────►│  choice of WHERE reviewer  │
          │  admission/finalization   │          │  judgment lands (build-end │
          │  (cmd/codex_build_        │          │  OR continue), recorded    │
          │  finalize.go)             │          │  with a reason             │
          └─────────────┬─────────────┘          └─────────────┬─────────────┘
                         ▼                                      ▼
          ┌─────────────────────────────────────────────────────────────────┐
          │  Result truth (D-05..D-08): every outcome (success/no-change/   │
          │  partial/blocker/timeout/interrupted) renders the SAME full     │
          │  closeout ceremony, cost/time line, credited+orphan files       │
          └─────────────┬───────────────────────────────────────────────────┘
                         ▼
          ┌─────────────────────────────────────────────────────────────────┐
          │  Bounded repair on failure (D-09..D-12): checkpoint -> one      │
          │  repair wave -> re-verify once -> restore checkpoint if it      │
          │  still fails. Extends checkFixAttemptRecord (cmd/build_attempt  │
          │  .go:547) and autopilotRepairLedger (cmd/autopilot_policy.go    │
          │  :523) rather than inventing a third repair mechanism.          │
          └─────────────────────────────────────────────────────────────────┘
```

### Recommended Approach to File Organization

This phase should **not** create a large number of new top-level files; it should extend files that already own the relevant concept, and split only where a file crosses ~2000 lines per existing convention hints (`CONCERNS.md` already flags `codex_continue.go` at 4155 lines as tech debt — do not add to that file uncritically; consider whether D-04's new shared function belongs in a *new*, focused file like `cmd/codex_verify_advance.go` that the three lanes call into, per the "refactor into focused packages" fix approach already recorded in `.planning/codebase/CONCERNS.md`).

```
cmd/
├── codex_continue.go            # direct lane — becomes a thin caller of the new shared function
├── codex_continue_plan.go       # plan-only lane — becomes a thin caller
├── codex_continue_finalize.go   # finalize lane — becomes a thin caller
├── codex_verify_advance.go      # NEW (suggested): the one accept/verify/advance function (D-04)
├── codex_build.go               # build-time verification stage narrows to deterministic-only
│                                 # when Queen chooses continue-boundary (D-01/D-02)
├── build_attempt.go             # buildAttemptRecord gains cost/time/orphan-file fields (D-06/D-08)
├── autopilot_policy.go          # repair ledger extended for D-09..D-12's general bounded-repair story
├── queen_judgement.go           # extended: judgement also carries the boundary choice + reason (D-01)
├── coherent_jobs.go             # unchanged in shape; consumed by the unified verify/advance path
└── job_telemetry.go             # NEW (suggested): per-job segment timing (D-13..D-16)
```

### Pattern 1: Queen judgment as validated proposal, never prompt authority

**What:** A Queen (or the deterministic fallback) proposes a team/boundary choice; a pure Go function reconciles it against floors/ceilings and returns exactly what changed and why.
**When to use:** Every decision this phase adds that involves "the Queen chooses X" — team composition (already done) and now verification-boundary placement (D-01) and next-action recommendation (D-07).
**Example:**
```go
// Source: cmd/queen_judgement.go:29-68 (read this session)
// queenCasteJudgement is the outcome of reconciling a proposed team with the
// floors and ceiling the runtime owns.
type queenCasteJudgement struct {
	Proposed        []string
	Final           []string
	Added           []string          // safety castes the proposal omitted and the runtime restored
	Dropped         []string          // proposed castes removed to fit the worker budget
	Refused         []string          // proposed castes the phase gives no work to
	Unknown         []string          // proposed names that are not dispatchable castes
	Rationale       string            // the Queen's TEAM summary (one string for the whole proposal)
	Reasons         map[string]string // per-worker reason, keyed by caste (D-08/D-09/D-10 discipline)
	RefusedNoReason []string          // proposed, not required, no stated reason -> refused
	Source          string            // "queen" or "deterministic"
}
```
D-01's boundary-choice record should follow the identical shape: a `Proposed`/`Final`/`Reason` triple that a later step cannot silently override, mirroring the existing `TestQueenChoiceReachesTheDispatchList` guarantee.

### Pattern 2: Coherent jobs and waves — planning before ownership, never per-task

**What:** Related tasks become one dependency-safe job with one workspace lease before anyone owns a file; disjoint jobs run in parallel waves.
**When to use:** WORK-03's "coherent jobs and waves" requirement — already delivered in v1.27/Phase 195 per STATE.md; this phase's job is to make sure the *verification and result model* downstream of jobs (receipts, findings, fan-in) refers to the same job/wave identity, not a re-derived one.
**Example:**
```go
// Source: cmd/coherent_jobs.go:47-84 (read this session)
type coherentJob struct {
	Name        string
	Tasks       []coherentJobTask
	TaskIDs     []string // preserve execution order
	OwnerCaste  string
	JobReason   string
	Source      string
	DependsOn   []string // names other jobs
	Wave        int      // deterministic job-DAG wave derived from those edges
	OwnerReason string
}

type coherentJobPlan struct {
	Jobs      []coherentJob
	Decisions []coherentJobDecision // what happened to each Queen proposal
	Waves     [][]string            // job names in runnable order
}
```
WORK-03's "receipts, result lineage, and visible waves" (success criterion 3) is satisfied by making the D-04 unified verify/advance path read `coherentJobPlan.Waves`/`Decisions` directly rather than re-deriving wave membership from task lists.

### Pattern 3: Append-only attempt journal with narrow-setter discipline

**What:** A build attempt is a single durable JSON record with an append-only `History` and narrowly-scoped setter functions (`attachCheckFixAttempt`, `attachBuildFreeCheckReport`) that touch exactly one field and nothing else — never overwriting `Status`, `Dispatches`, or `Claims`.
**When to use:** Every new field this phase adds to the attempt record (cost, time-per-segment, orphaned files) must follow this same narrow-setter discipline, proven by tests like `TestFixAttemptNeverOverwritesTheFirstResult`.
**Example:**
```go
// Source: cmd/build_attempt.go:547-591 (read this session)
type checkFixAttemptRecord struct {
	RecordedAt      string
	Phase           int
	Check           string // e.g. "tests" — the failing verification step
	Reason          string // "fixing the failed tests check" (D-03 clause)
	ParentAttemptID string // the attempt this fix follows up on, never replaces
	FailureIndex    checkFailureIndex
	Outcome         string // "fixed" or "still_failing"
}
```
This is D-09's "one checkpointed round: checkpoint, one repair wave, re-verify once" already half-built for the narrow "check-fix" case. D-09..D-12 generalize this pattern (and `autopilotRepairLedger`'s budget/receipt discipline, `cmd/autopilot_policy.go:523-533`) to the phase's full bounded-repair story rather than inventing a third repair record type.

### Anti-Patterns to Avoid

- **A fourth parallel "verification result" type:** `CONCERNS.md` already documents three continue lanes as an unmanaged-parity risk. Do not add a fourth data shape for "the unified result" that the three lanes each translate into — D-04 requires the three lanes to become thin callers of *one* function operating on the *existing* `codexContinueManifest`/`codexContinueAssessment`/`buildAttemptRecord` types, not a new envelope type layered on top.
- **Re-deriving job/wave membership downstream of dispatch:** Success criterion 3 explicitly requires the Queen card, jobs, dependencies, waves, workspaces, receipts, findings, and fan-in to refer to the *same* durable identity. A second computation of "which job owns this file" anywhere downstream (e.g., in a rendering function) is a parity bug waiting to happen — reuse `coherentJobPlan` fields verbatim.
- **Silent second automatic repair attempt:** D-09 is explicit: "If verification still fails, restore the checkpoint and pause with the evidence. No second automatic attempt." `checkFixAttemptRecord`'s single-record-per-attempt design already enforces this at the type level for the narrow check-fix case; the general repair path (D-09..D-12) must preserve the same one-shot guarantee, not add a retry loop.
- **Cost/time estimation instead of measurement:** D-06 explicitly forbids estimating or guessing unreported cost — "Unreported cost is named as unreported, never estimated or guessed." The existing ledger already enforces this discipline (`STATE.md` [Phase 196]: "'Reported' keys on the usage source tag, never on a number ... An estimated ledger row renders the dash sentinel exactly like an empty one"). WORK-08's per-job timing must follow the identical rule: a segment with no evidence renders as unmeasured, never as zero or an inferred average.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Verifying only proportionate-team floors (Builder-only for a low-risk one-task change) | A new "is this simple" heuristic | `queenApplyJudgement` + the required-caste floor already in `cmd/queen_judgement.go` (proven by `TestOneTaskBugFixIsOneWorkerPlusChecks`, STATE.md [Phase 194]) | This exact behavior is already delivered and test-locked from v1.27; WORK-02 is a *non-regression* requirement for this phase, not new design. |
| Dependency-safe task grouping | A custom DAG grouping pass | `planCoherentJobs` (`cmd/coherent_jobs.go`) — already validates the whole phase graph, refuses unsafe orderings by name, and substitutes dependency-safe jobs for only the offending group | Delivered and test-locked in Phase 195 (`TestCoherentJobProposalOrderRefusedByName`, `TestCoherentJobGraphErrorsHaveNoPlan`, STATE.md). Re-deriving this for "waves" in D-04's unified path would create the exact parity risk the phase exists to close. |
| Deterministic pass/fail floor (build/vet/tests/lint) | A per-lane copy of "run the checks" | `runDeterministicFloor` (`cmd/deterministic_floor.go:48`) — already the *sole* source of a pass across both existing continue lanes (`TestBothContinueLanesApplyTheSameFloor`) | D-01/D-04 must route the *third* (plan-only+finalize) lane through this same function too, not reimplement it a third time. |
| Bounded, budgeted repair with receipts | A bespoke retry counter | `autopilotRepairLedger`/`autopilotRepairReceipt` (`cmd/autopilot_policy.go:494-533`) — already has `InitialBudget`/`Remaining`, per-attempt `BudgetBefore`/`BudgetRemaining`, and a `Verification` sub-record | This is D-09's exact shape (checkpoint → one repair wave → re-verify) already implemented for autopilot's repair loop; extend its use to build/continue's own bounded repair rather than building parallel budget tracking. |
| Worker context slimming / compact handoff | A new "brief compression" scheme | `codex.WorkerHandoff` (`pkg/codex/handoff.go:10-20`) — already carries `ChangedFiles`, `CommandsRun`, `VerificationStatus`, `KnownFailures`, `OpenDecisions`, `Assumptions`, `NextWorkerInstructions`, `DoNotRepeat`, `Freshness` | D-15(b) ("slimmer briefs and compact handoffs") is a rendering/selection change over data this type already carries — do not add a second handoff schema. |
| Targeted test-lane selection scoped to changed packages | A new package-dependency scanner | `scopedGoTestCommand` (`cmd/verification_scope.go:189`) — already narrows `go test` to the Go packages a phase's changed files touched, falling back to the full run when scope can't be honestly derived (Phase 193 D-07, STATE.md) | D-15(a) ("targeted test lanes — a change to one area runs that area's tests during the loop while the full suite still gates the phase boundary") is exactly this existing function's documented behavior; WORK-08 should measure and extend it, not replace it. |

**Key insight:** This phase's single biggest risk is *not* missing capability — it is re-inventing a slightly different version of something that already exists and is already test-locked, thereby recreating the exact "three parallel implementations that must stay in sync by hand" failure mode `CONCERNS.md` names as this repo's own documented pattern. The mechanism-study gate (SYNTH-03/SYNTH-07) exists specifically to force an audit of current Go before any new type is proposed.

## Common Pitfalls

### Pitfall 1: Treating "collapse three continue lanes" as a rewrite instead of a refactor-to-shared-function

**What goes wrong:** A planner reads "one authoritative Go path owns accept/verify/advance" (D-04) and designs a brand-new unified continue implementation from scratch, discarding the already-proven per-lane logic (replay guards, stale-workspace checks, `TestDeterministicFloorIsTheOnlySourceOfAPass`, evidence gates).
**Why it happens:** The three files are large (4155/589/1112+ lines across `codex_continue.go`/`codex_continue_plan.go`/`codex_continue_finalize.go`) and their logic is deeply interleaved with lane-specific I/O (reading a completion JSON from disk vs. running inline).
**How to avoid:** Extract the *decision* logic (assessment, review dispatch, gate evaluation, advancement) into one function that all three lanes call with lane-specific inputs already normalized into the same in-memory types (`codexContinueManifest`, `codexContinueVerificationReport`, `codexContinueAssessment` — all already shared across the three files per this session's `grep`). The lane-specific parts (how results arrive: inline dispatch vs. external JSON completion) stay lane-specific; the decision core does not.
**Warning signs:** A plan task titled "rewrite continue" instead of "extract shared verify/advance function"; a plan that doesn't cite `cmd/codex_continue_finalize.go`'s existing bug (`CONCERNS.md`: "Evidence gate blocks finalize when reconciliation is supplied ... The direct `aether continue` path correctly accepts the reconciliation") as evidence for *why* the collapse is needed.

### Pitfall 2: Double-verification silently reappearing after D-01 is implemented

**What goes wrong:** D-01 lets the Queen choose build-end *or* continue as the verification boundary, but if `queenBuildPostWaveDispatches`/the build verification stage isn't also gated on that same choice, a Watcher still runs at build-end even when the Queen chose continue — reproducing exactly the `CONCERNS.md` "Same phase verified twice" bug this phase is chartered to fix.
**Why it happens:** Build-side reviewer dispatch (`cmd/codex_build.go` verification stage) and continue-side reviewer dispatch (`queenContinueDispatchesWithJudgement`, `cmd/codex_continue.go:1405`) are two independently-computed caste sets today (already flagged in STATE.md [Phase 193]: "probe, auditor and gatekeeper still legitimately double-dispatch on production/security phases with no explicit Queen proposal, because build and continue independently derive the same required caste from the same phase content").
**How to avoid:** D-01's boundary decision must be a single stored fact (like `queenCasteJudgement`) that *both* the build-time dispatch code and the continue-time dispatch code read — never two independent derivations that happen to usually agree.
**Warning signs:** A test that asserts "Watcher dispatched at build" and a separate test that asserts "Probe dispatched at continue" for the *same* phase fixture without a shared boundary-choice record between them.

### Pitfall 3: Recovery/repair introduces a competing checkpoint concept

**What goes wrong:** D-09/D-10 need "checkpoint saved" / "restored to checkpoint" announcements, and Phase 199 already built `/ant-pause`/`/ant-resume` checkpoint/handoff machinery. Building a *second* checkpoint concept for build/continue repair (rather than reusing the pause/resume handoff identity) creates two "what does checkpoint mean" stories the owner has to reconcile.
**Why it happens:** The repair checkpoint (D-09) is triggered automatically by a failing verification, while `/ant-pause` is owner-invoked — different triggers can tempt a separate implementation.
**How to avoid:** CONTEXT.md's Integration Points section says this explicitly: "Repair checkpoints (D-09/D-10) interact with `/ant-pause`/`/ant-resume` handoff machinery from Phase 199 — one checkpoint concept, not two competing ones." The mechanism study (Wave 1) must resolve whether repair-checkpoint reuses the pause/resume handoff ID/idempotency-key pattern (`cmd/pause` uses "the handoff ID as their idempotency key, so interrupted commits resume one journal and verified replay returns one existing receipt" — STATE.md [Phase 199]) as its SYN-201-* decision, before any repair code is written.
**Warning signs:** A new "repair checkpoint" JSON file format that doesn't reuse any type from the pause/resume handoff machinery (`.aether/data/handoffs/`).

### Pitfall 4: Telemetry becomes an estimate when a segment has no real evidence

**What goes wrong:** WORK-08's per-job timing (queue, preflight, model, tool-call, context, work, verification, wait) is implemented with a fallback that computes a missing segment by subtracting known segments from total elapsed time, producing a plausible-looking number for a segment that was never actually measured.
**Why it happens:** Total wall-clock time is trivial to capture (start/end timestamps already exist on `buildAttemptRecord`); per-segment breakdowns require instrumenting each sub-step, which is more work, so "just derive the missing one" looks efficient.
**How to avoid:** D-06 and the project's own cost-ledger precedent forbid this pattern outright: "'Reported' keys on the usage source tag, never on a number" and "An estimated ledger row renders the dash sentinel exactly like an empty one" (STATE.md [Phase 196]). A telemetry segment with no direct instrumentation must render as unmeasured (a dash/sentinel), never as a derived remainder.
**Warning signs:** A telemetry struct field with a comment like "computed as total minus other segments" instead of "captured at instrumentation point X."

### Pitfall 5: Parallel-wave expansion sneaking back in as a "speed lever"

**What goes wrong:** WORK-08's turnaround-improvement work naturally suggests "run more workers in parallel" as an obvious lever, but D-15 explicitly rules this out: "Parallel-wave expansion was NOT approved as a lever (past timing flakes came from contention)."
**Why it happens:** Parallel execution is the most intuitive way to cut wall-clock time, and the codebase already has worktree-mode parallelism infrastructure that a planner might reach for.
**How to avoid:** Confine speed work to the three approved levers only: (a) targeted test lanes via `scopedGoTestCommand`, (b) slimmer briefs/compact handoffs via `WorkerHandoff`, (c) model routing by caste with a visible cost line. Any plan task proposing to raise worker-count ceilings or expand parallel waves for speed (as opposed to correctness/proportionality per WORK-01..03) should be flagged against D-15 explicitly.
**Warning signs:** A plan task that changes `queenBuildTaskCaste`/depth-adjusted worker caps (`cmd/codex_build.go`) in the name of "faster turnaround" rather than "proportionate team."

## Code Examples

### Reading the deterministic floor as the sole pass source
```go
// Source: cmd/deterministic_floor.go:48 (signature read this session)
func runDeterministicFloor(ctx context.Context, root string, phase colony.Phase,
	manifest codexContinueManifest, watcher codexWatcherVerification,
	verificationTimeout time.Duration) deterministicFloorResult
```
Both existing continue lanes already call this one function (`TestBothContinueLanesApplyTheSameFloor`). D-04's unified accept/verify/advance function must be the single caller for the third (plan-only+finalize) lane too — not a fourth independent invocation site.

### The durable attempt journal's append-only transition history
```go
// Source: cmd/build_attempt.go:190-244 (read this session)
type buildAttemptRecord struct {
	SchemaVersion    int
	ID               string
	Phase            int
	Status           string
	StartedAt        string
	UpdatedAt        string
	CompletedAt      string
	Dispatches       []codexBuildDispatch
	Claims           *codexBuildClaims
	Recoverable      bool
	RecoveryCommand  string
	RecoveryTaskIDs  []string // exact unfinished-only task set, per D-10-style precedent
	History          []buildAttemptTransition
	OutOfBandVerification *outOfBandVerificationRecord
	FreeChecks       *buildFreeCheckReport
	CheckFix         *checkFixAttemptRecord // D-02/D-03's single bounded automatic fix attempt
	ParentAttemptID  string                 // begin-then-attach linkage to the attempt this followed up on
	ParentJobName    string                 // the coherent job this recovery attempt was derived from
}
```
This is the "durable attempt and result model" success criterion 3 requires the Queen card, jobs, receipts, and findings to all refer to. D-05's outcome cards and D-06's cost/time line should be new fields on this same record (or a sibling record keyed by the same `ID`), never a separate result envelope.

### Bounded, receipted repair (the pattern D-09 generalizes)
```go
// Source: cmd/autopilot_policy.go:494-533 (read this session)
type autopilotRepairEvaluation struct {
	Eligible bool
	Pause    bool
	Reason   string
}

type autopilotRepairReceipt struct {
	ID              string
	Phase           int
	Attempt         string
	Check           string
	Status          autopilotRepairStatus
	PreparedAt      string
	CompletedAt     string
	BudgetBefore    int
	BudgetRemaining int
	Verification    autopilotRepairVerification
}

type autopilotRepairLedger struct {
	SchemaVersion  int
	InvocationID   string
	InitialBudget  int
	Remaining      int
	Receipts       []autopilotRepairReceipt
	Debt           []colony.LifecycleIssue
	Blockers       []colony.LifecycleIssue
}
```

## State of the Art

| Old Approach (Classic, pre-v1.25) | Current Approach (this repo today) | When Changed | Impact |
|--------------------------------|-------------------------------------|---------------|--------|
| Queen team choice via keyword-only relevance scoring | `queenApplyJudgement` reconciles an LLM-authored proposal against floors/ceilings with recorded per-worker reasons | v1.27 (Phase 194) | D-01/D-02 extend the *same mechanism* to the verification-boundary decision rather than inventing a second judgment engine. |
| Build and continue each independently computed required reviewer castes | Still independently computed today (named defect in `CONCERNS.md`) | Not yet fixed | This is exactly what D-01/D-04 must close — see Pitfall 2. |
| Three separate continue code paths with only manual-review parity | Still three separate lanes today, no automated parity test | Not yet fixed | D-04's core deliverable. |
| No per-job timing telemetry | None exists today — total build/continue wall-clock is captured, but no per-segment breakdown | Never built | WORK-08 is genuinely new instrumentation, not a restoration. |
| Check-fix repair: single bounded automatic attempt, narrow to one failing check | Exists and is test-locked (`checkFixAttemptRecord`) | Phase 193 (v1.27) | D-09..D-12 generalize this narrow case to the phase's full bounded-repair story. |

**Deprecated/outdated:** None of the Classic (pre-v1.25) `.claude`/`.aether` command-YAML era's build/continue prompt logic should be restored as *implementation* — per `.planning/research/v1.28-classic-synthesis-template.md`'s disposition model, Classic mechanisms here are evidence for *why* a behavior mattered (e.g., "3-4 workers for a small fix," visible waves, honest partial closeouts), never code to copy. The current Go kernel (attempts, receipts, deterministic floor, coherent jobs) is the substrate every synthesis decision must build on.

## Mandatory Mechanism-Study Sequencing (SYNTH-03 / SYNTH-07)

Phases 199 and 200 both produced their `CLASSIC-SYNTHESIS.md` in an early, gating plan before any implementation plan — the planner must reproduce this exact shape for Phase 201:

- **Phase 199 precedent:** `199-02-PLAN.md` (wave 2, `depends_on: ["199-01"]`) has `requirements: [SYNTH-01, PROOF-01]` and its sole deliverable is `.planning/phases/199-front-door-and-classic-contract/199-CLASSIC-SYNTHESIS.md` plus the versioned `cmd/testdata/classic-contract/v1/{schema.json,mechanisms.json,cases.json}` corpus and its loader tests (`cmd/classic_contract_test.go`) `[VERIFIED: .planning/phases/199-front-door-and-classic-contract/199-02-PLAN.md:1-33]`.
- **Phase 200 precedent:** `200-CLASSIC-SYNTHESIS.md` follows the exact template structure (`.planning/research/v1.28-classic-synthesis-template.md`): outcome under investigation → Classic mechanism reconstruction table (`OLD-*` rows citing `git show <tag>:<path>:<lines>`) → current Go mechanism audit table (`NOW-*` rows citing `path:line`/test names) → comparative synthesis matrix (`SYN-200-*` rows with `keep-current`/`restore-modern`/`replace-better`/`retire-with-proof` dispositions) → selected architecture → research-to-plan linkage table → verification contract `[VERIFIED: .planning/phases/200-iterative-planning/200-CLASSIC-SYNTHESIS.md]`.
- **Phase 201 must do the same:** produce `201-CLASSIC-SYNTHESIS.md` with `SYN-201-*` decision IDs, citing the eight routed CAP rows below plus the Classic build/continue/run/swarm evidence already gathered in `.aether/dreams/2026-09-01-comprehensive-aether-colony-review.md` (section D, "Command-by-command reconstruction," rows for `build`/`continue`/`run`). This document must exist and be signed **before** any Phase 201 implementation plan is approved (SYNTH-07 is a planning gate, not a suggestion).
- **The versioned Classic contract corpus** (`cmd/testdata/classic-contract/v1/`) currently has 22 `mechanisms.json` entries (`SYN-199-*` × 10, `SYN-200-*` × 12) and 108 `cases.json` executable cases `[VERIFIED: python parse of cmd/testdata/classic-contract/v1/mechanisms.json and cases.json this session]`. Phase 201 extends both files with `SYN-201-*` mechanism entries and new cases in a `work-cycle`-style group, following the exact same required schema fields: `id, group, kind, historical_anchors, synthesis_decision, cap_ids, source_citations, platform, command, expected` for a case `[VERIFIED: cmd/testdata/classic-contract/v1/schema.json $defs.case.required]`, and the same registry shape for a mechanism entry (`id, title, disposition, cap_ids, public_commands, source_citations, groups`) `[VERIFIED: cmd/testdata/classic-contract/v1/mechanisms.json first entry, read this session]`.

## CAP Row Routing for Phase 201

The capability ledger routes exactly 8 of the 72 frozen dropped-capability rows to Phase 201 `[VERIFIED: .planning/research/v1.28-classic-capability-ledger.md:136,137,155,157,162,184,199,204]`:

| CAP ID | Classic capability | Disposition | Requirement | Report-grounded target |
|---|---|---|---|---|
| `CAP-003` | build: midden-write on worker/build failure | `restore-modern` | WORK-06 | Worker and build failures continue to write canonical midden evidence consumed by bounded recovery. |
| `CAP-004` | build: persist worker blockers as flags | `restore-modern` | WORK-05 | Worker blockers remain durable attempt-bound result truth and appear consistently in advance/status/seal decisions. |
| `CAP-022` | patrol: plan-vs-reality task evidence verification | `restore-modern` | WORK-05 | Plan-versus-reality evidence is part of the authoritative verification and completion result. |
| `CAP-024` | patrol: unresolved flags + recurring midden review | `restore-modern` | WORK-06 | Patrol/recovery consumes unresolved flags and recurring midden classes through one bounded recovery model. |
| `CAP-029` | quick | `replace-better` | WORK-07 | Quick work becomes a proportionate one-goal fast path through the same attempt, checks, and evidence model. |
| `CAP-051` | status: escalated-flags count | `restore-modern` (already proven, Phase 198.3) | WORK-05 | Preserve the Phase 198.3 escalated-blocker count and bind it to exact blocker truth. |
| `CAP-066` | chamber-compare `diff` (content-level new decisions/learnings) | `replace-better` | CEC-06 | Content-level decision/learning deltas become attempt-bound evidence shown where build/continue consumes them. |
| `CAP-071` | `.aether/data/verification.json` / `build-claims.json` / `claims.json` (root-level legacy names) | `replace-better` | WORK-05 | Legacy root filenames migrate to exact attempt-bound claims/verification artifacts; compatibility never weakens validation. |

Every routed row's disposition above is a **hypothesis from the comprehensive report**, not a pre-judged final decision — the `201-CLASSIC-SYNTHESIS.md` must independently confirm or revise each via its own `SYN-201-*` comparative synthesis row, citing old and current evidence, per the template's planning gate checklist.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | A new shared file (e.g. `cmd/codex_verify_advance.go`) is the right home for D-04's unified accept/verify/advance function, rather than growing one of the three existing continue files further | Recommended Approach to File Organization | Low — this is Claude's Discretion per CONTEXT.md ("Internal Go types ... provided the decisions above ... hold"); the planner may choose a different file layout as long as the single-function/single-caller guarantee holds. |
| A2 | Repair-checkpoint (D-09/D-10) should reuse the `/ant-pause`/`/ant-resume` handoff-ID idempotency pattern from Phase 199 rather than a new checkpoint primitive | Pitfall 3 | Medium — if the mechanism study concludes the two checkpoint concepts are genuinely different (owner-invoked pause vs. verification-triggered repair-restore), a distinct-but-related design may be correct instead. This must be resolved as an explicit `SYN-201-*` row, not assumed silently. |
| A3 | WORK-08's numeric turnaround target cannot be researched — it must come from D-13's own mid-phase measurement of a representative one-plan job | Turnaround Targets section (implicit throughout) | Low — this is explicitly stated in CONTEXT.md D-13 itself, not an inference; flagged here only so the planner does not mistake this research document's silence on a target number for an oversight. |

## Open Questions

1. **Does D-01's Queen boundary choice reuse `queenCasteJudgement`'s exact struct shape, or does it need a distinct type?**
   - What we know: `queenCasteJudgement` already has the `Proposed`/`Final`/`Rationale`/`Source` shape D-01 needs (Queen proposes, runtime validates/records reason).
   - What's unclear: Whether "which boundary carries reviewer judgment" is naturally a *caste-judgment*-shaped decision or a different kind of typed choice (e.g., an enum `{build_end, continue}` with a reason string) that doesn't fit the multi-caste reconciliation shape.
   - Recommendation: Resolve in the `201-CLASSIC-SYNTHESIS.md` comparative synthesis as an explicit `SYN-201-*` row; likely a small new type (`verificationBoundaryDecision{ Choice, Reason, Source }`) that *references* but does not extend `queenCasteJudgement`.

2. **How does D-04's single function's failure/error handling reconcile the three lanes' currently-different error surfaces (in-process error return vs. external JSON completion contract violations)?**
   - What we know: `codex_continue_finalize.go` has a distinct `completionContractError`-shaped validation path (`validateCompletionPacketSemantics`, `contractViolation`) that the direct lane doesn't need, because the direct lane never parses untrusted external JSON.
   - What's unclear: Whether the shared function should accept a pre-validated, lane-normalized input type (recommended) or whether contract validation itself should also become shared.
   - Recommendation: The shared function should operate on already-normalized Go types; each lane keeps its own input-validation/parsing responsibility (untrusted JSON parsing is lane-specific; the *decision* is not).

3. **What does "cost line showing what each worker cost" (D-15c, model routing) require from the existing spend ledger that isn't already there?**
   - What we know: `.planning/codebase` and STATE.md [Phase 196] show a mature, fail-closed spend ledger (`spendTotals`, per-worker `codexBuildDispatch.Name`-keyed accounting, "Reported" vs. unreported distinction) already exists.
   - What's unclear: Whether model routing by caste (D-15c) needs new ledger fields, or whether it only needs a policy layer (which model to request per caste) sitting *above* the existing ledger, which already accounts whatever model was actually used.
   - Recommendation: Treat as a policy-layer addition over the existing ledger unless the mechanism study finds the ledger cannot currently attribute cost to a specific *model choice* per caste (as opposed to per worker) — verify this by reading `cmd/spend_*.go` in the synthesis plan before assuming new fields are needed.

## Environment Availability

Skipped — this phase has no external service/tool dependencies beyond the existing Go toolchain and git, both already present in this repository's CI and dev environment (`go.mod:3`, `.planning/codebase/TESTING.md`).

## Sources

### Primary (HIGH confidence — read directly this session)
- `.planning/phases/201-queen-led-work-cycle/201-CONTEXT.md` — full D-01..D-16 decision set, canonical refs, code context.
- `.planning/REQUIREMENTS.md:25,38,62-69,130` — SYNTH-03, CEC-06, WORK-01..08 full text and phase routing.
- `.planning/STATE.md` — prior milestone decisions, Phase 193-200 test-locked precedents cited throughout.
- `.planning/research/v1.28-classic-synthesis-template.md` — mandatory synthesis structure.
- `.planning/research/v1.28-classic-capability-ledger.md` — 8 CAP rows routed to Phase 201, dispositions, requirement mapping.
- `.planning/codebase/CONCERNS.md` — named defects: dual continue lanes, dual verification, large monolithic files.
- `.aether/dreams/2026-09-01-comprehensive-aether-colony-review.md` (sections D, I, J) — Classic command-by-command reconstruction, proposed architecture, phased recovery program (Phase 4 = this phase's scope).
- `.planning/phases/199-front-door-and-classic-contract/199-CLASSIC-SYNTHESIS.md` and `199-02-PLAN.md` — precedent for synthesis-doc sequencing.
- `.planning/phases/200-iterative-planning/200-CLASSIC-SYNTHESIS.md` — precedent for synthesis-doc structure and comparative matrix format.
- `cmd/queen_judgement.go:29-68`, `cmd/coherent_jobs.go:30-90`, `cmd/build_attempt.go:190-620`, `cmd/autopilot_policy.go:478-538`, `pkg/codex/handoff.go:10-50`, `cmd/deterministic_floor.go:48`, `cmd/verification_scope.go:189` — existing Go types/functions this phase extends.
- `cmd/testdata/classic-contract/v1/schema.json`, `mechanisms.json`, `cases.json` — versioned Classic contract corpus structure to extend.
- `go.mod:3` — Go 1.26.5 toolchain confirmation.
- `.planning/codebase/TESTING.md:8-45` — test framework/conventions confirmation.
- `.planning/config.json` — confirms `nyquist_validation: false`, `security_enforcement: false` (both sections omitted from this document per those settings).

### Secondary (MEDIUM confidence)
- `git log`/`git rev-parse` output this session confirming repository state (commit `702f03c8`, date 2026-09-10) — used only for dating, not for technical claims.

### Tertiary (LOW confidence)
- None — all claims trace to files read this session or to the phase's own CONTEXT.md/REQUIREMENTS.md.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — no new dependencies; Go toolchain version read directly from `go.mod`.
- Architecture: HIGH — every cited type/function/line was read this session; the "what's missing" analysis is grounded in `CONCERNS.md`'s own named defects, not inference.
- Pitfalls: HIGH — each pitfall cites either an existing STATE.md-recorded defect/finding or an explicit CONTEXT.md decision it would violate.
- Turnaround numeric target: LOW by design — D-13 requires this phase itself to measure and set it; no research can supply it in advance.

**Research date:** 2026-09-10
**Valid until:** Should be re-verified if the mechanism-study plan (Wave 1) discovers the current Go code has materially changed since this session's reads (unlikely within the same milestone, but the synthesis doc's own `NOW-*` audit is the authoritative re-check, not this document).
