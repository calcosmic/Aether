# Phase 204: Learning Governor - Context (gap closure)

**Gathered:** 2026-09-14
**Status:** Ready for gap-closure planning
**Mode:** gap_closure — this context governs ONLY the closure plans for the seven gaps in `204-VERIFICATION.md`. The eleven executed plans (204-01..204-11) are not replanned.

<domain>
## Phase Boundary

Phase 204's goal is fixed by ROADMAP.md: make memory truthful and prove behavior changes through immutable outcomes, permanent evaluations, independent shadow comparison, bounded promotion, and rollback.

Verification (2026-09-14) found the second half of that goal built as a library of correct, individually-tested functions with no path connecting them to anything the running program does. The gap-closure boundary is therefore: **give every built mechanism a real production caller, on every lane that has one, proven by a command that fails when the caller is missing.** No redesign. Every piece needed already exists and is tested; it needs a caller (204-VERIFICATION.md, "Recommended next step").

</domain>

<decisions>
## Implementation Decisions

### How the running program reaches the three self-improvement features (owner decision, 2026-09-14)

The owner was asked how the program should reach (1) shadow comparison — trying a proposed change beside current behaviour, (2) canary promotion and atomic rollback, and (3) source-improvement proposals. The owner chose **"Automatically at each check, plus a command to look"**.

- **D-01:** Shadow comparison, promotion-gate admission, canary start/complete/rollback, and source-improvement proposal all run **automatically at the end of every check (`aether continue`), on BOTH check lanes** (the native/direct lane and the delegate/chat lane), immediately after phase-end consolidation — the same boundary where hive promotion already runs (`runPhaseEndConsolidation` → `recordPhaseApplicationCredit` → hive promotion). A lane on which the automatic pass cannot be reached is a failed criterion, not a documented limit.
- **D-02:** In addition, one **hand-run command** exists to inspect or trigger a comparison/canary by hand: a registered `aether` CLI surface (registering the already-built `shadow-declare` / `shadow-compare` commands is acceptable; a single umbrella command is also acceptable) with a matching `/ant-…` menu wrapper on the Claude and OpenCode surfaces (wrapper copies are byte-identical and parity-tested), so that `TestNoRegisteredSubcommandIsUnreferenced` has a real caller to point at and the command does not sit unused.
- **D-03:** The closing card the owner reads after a check names, in plain English, when a candidate was tried, promoted into a canary, rolled back, or when a source-improvement proposal was written — one line per event, nothing when nothing happened. No raw identifiers or internal state tokens on that card (voice-corpus rules apply).
- **D-04:** The automatic pass **never widens what may be changed**: it stays inside the two canary-promotable scopes (remembered project facts and task routing) and the nine retained-authority categories keep refusing structurally. A comparison, canary, or proposal that fails, is refused, or finds nothing must never block the check — it is logged and surfaced, never fatal (same non-blocking contract as hive promotion).

### What closure means for every gap (locked by the project's Definition of Done)

- **D-05:** Each of the seven verification gaps (SC3a, SC3b, SC3c, SC4b, SC5b, SC5c, SC5d) closes only with a **named test that fails when the production caller is removed** — a wiring test asserting the real production call site, not a fixture built in a shape the runtime cannot produce. "Covered by existing tests" does not close a gap.
- **D-06:** The `t.Skipf` on the swarm and recovery lanes in `TestEveryLifecycleLaneWritesADurableOutcome` is removed — both lanes must genuinely pass the same subtest the other four lanes pass, through the same `emitColonyLiveEpisodeStarted/Ended` boundary. A skip is not a pass.
- **D-07:** The nine writerless episode-ledger fields (`evidence_ids`, `hard_gate_results`, `changed_decision_ids`, `usage`, `reported_cost_usd`, `interventions`, `episode_revision`, `acceptance_digest`, `evaluator_digest`) get real production writers from the boundary callers, AND the episode ledger is registered as the seventh store in the memory-schema census (`liveMemoryStoreCensusTypes`) so `TestEveryMemoryStoreFieldHasALiveWriter` would fail if any of them went writerless again.
- **D-08:** The shadow grader stops being an always-pass placeholder: `FrozenEvaluator.Run(Candidate, Task) Result` is driven by a real per-fixture classifier, and a named test proves a beneficial candidate and a harmful/overfit candidate receive different verdicts through the production command path.
- **D-09:** The fixture bank's unguarded floor shrinks from 40. Every fixture that gains a guard cites a real, existing, named test for its own confirmed incident; the floor constant may only decrease. The planner decides how far the floor can honestly move in this closure (not every one of the 40 must be guarded here), but the number written down must be the number the tests prove.
- **D-10:** `episodeLedgerRecord.Interventions` gets a **closed intervention-kind vocabulary** that the real writers populate from, so `collectPreventableInterventions` classifies against a curated set — never each free-form string as its own category.
- **D-11:** REQUIREMENTS.md must end this closure telling the truth: LEARN-02, LEARN-06, LEARN-07 and LEARN-08 are marked satisfied only once the wiring tests above pass; WINDOWS.md entries 38–45 are closed (or narrowed with the remaining limit named) in the same change that closes each gap.

### Claude's Discretion

- How to implement the real per-fixture classifier behind `FrozenEvaluator.Run` (as long as D-08 holds).
- Which of the 40 unguarded fixtures gain guards in this closure and how the guard index records them (as long as D-09 holds).
- The exact members of the closed intervention-kind vocabulary (as long as D-10 holds and the report's two figures stay separate).
- Whether the automatic pass is one new function called from `runPhaseEndConsolidation` or a sibling called beside it, and how it is budgeted so an ordinary check with no declared candidate costs nothing measurable.
- Name and shape of the hand-run command and its wrapper, within D-02.
- How the source-improvement proposal is triggered automatically (e.g. only when the improvement report shows a repeated preventable intervention), as long as D-01/D-04 hold and it can never self-approve, merge, publish, or deploy.
- Grouping of the seven gaps into plans and waves.

</decisions>

<specifics>
## Specific Ideas

- The owner's own words for the chosen option: things should "run on their own at the end of every check, the same moment the program already promotes lessons it has learned — no new command to remember", the closing card should say "when something was tried, promoted, or undone", and a separate command should let them "look at or trigger one by hand".
- This project's recorded failure mode is exactly the shape of these gaps: 18 of 25 milestones have been framed around restoring something previously marked done, and features "built but never called" are its signature (CLAUDE.md "Definition of Done"; memory: "Machinery exists but was never switched on"). Closure plans must assert the call site, never the existence of the function.
- Anything with more than one lane gets both lanes proved (CLAUDE.md "How much proof a change needs"): the check has a native lane and a delegate lane; the episode boundary has six lanes (build, continue, swarm, oracle, recovery, plan).

</specifics>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### The gaps themselves
- `.planning/phases/204-learning-governor/204-VERIFICATION.md` — the seven failed truths, each with the exact file, the missing caller, and what "missing" means.
- `.planning/WINDOWS.md` entries 38–45 — the project's own defect register for the same gaps, including the swarm/recovery routing detail (41), the fixture floor (42), the validator gap for `learn.StatusValidated` (44) and the closed-vocabulary requirement (45).
- `.planning/phases/204-learning-governor/204-REVIEW.md` and `204-REVIEW-FIX.md` — the post-execution code review and the five fixes already applied (do not re-fix).

### What was built (read the SUMMARY, then the code it names)
- `204-04-SUMMARY.md` — episode ledger; `204-08-SUMMARY.md` — shadow package and `cmd/shadow_cmds.go`; `204-09-SUMMARY.md` — promotion gate and rollback; `204-10-SUMMARY.md` — improvement report and source proposal (its "Next Phase Readiness" names the callers that are missing); `204-05-SUMMARY.md` / `204-07-SUMMARY.md` — fixture bank and guard index; `204-03-SUMMARY.md` — memory-schema census.
- `cmd/consolidation_lifecycle.go` (`runPhaseEndConsolidation`) — the one boundary D-01 hangs off; `cmd/live_events.go` (the unwired emitters); `cmd/episode_ledger.go`; `cmd/shadow_cmds.go`; `cmd/promotion_gate.go`; `cmd/rollback.go`; `cmd/improvement_report.go`; `cmd/source_proposal.go`; `cmd/memory_schema.go`; `cmd/testdata/fixture-bank/v1/bank.json`.

### Rules that bind the closure
- `CLAUDE.md` — "Definition of Done" and "How much proof a change needs" (full rigour: this decides work is complete/credited; both lanes get proved); "Communication Style" (the closing-card line in D-03 must be plain English); "Wrapper-Runtime Contract" and the wrapper-triplet parity rule for D-02.
- `.aether/docs/learning-system-authority.md` — the non-blocking contract hive promotion already follows, which D-04 reuses.

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `runPhaseEndConsolidation` already reaches both check lanes and already chains credit → hive promotion non-blockingly — the automatic pass in D-01 should sit on that exact chain, reusing its lane-reachability test pattern (`TestPhaseApplicationCreditIsReachedFromBothCheckLanes`).
- `emitColonyLiveEpisodeStarted/Ended` — the boundary four lanes already call; swarm and recovery route through it too (D-06).
- `emitColonyLiveOutcomeRecorded` / `emitColonyLiveInterventionRecorded` — built, tested, uncalled (D-07).
- `FrozenEvaluator.Run` seam in `pkg/shadow` — proven isolation; only the grading function behind it is a placeholder (D-08).
- `admitCandidateToCanary`, `startCanary`, `rollbackCanary`, `completeCanary` — correct, tested, uncalled.
- `proposeSourceImprovement` + `sourceProposalReachabilityEntryPoints` — extend the entry-point list in the same change that adds the caller (WINDOWS #45).
- `seedBankUnguardedFloor` and the fixture-to-guard index — the ratchet D-09 moves.

### Established Patterns
- Wiring tests assert the real spawn list / call site, never an intermediate record (`TestQueenChoiceReachesTheDispatchList` pattern).
- Allowlists and floors only shrink (`TestMemoryStoreFieldExceptionsOnlyShrink`, `seedBankUnguardedFloor`).
- Dry-run/inspection commands never mutate state.
- Voice corpus: any new owner-facing screen line carries the shared glyph table and no raw state token.

### Integration Points
- Phase-end consolidation (both check lanes) — D-01.
- `cmd/swarm_cmd.go` and the recovery entry point — D-06.
- Build/continue/swarm/oracle/recovery boundary callers — D-07.
- cobra root command registration + `.claude/commands/ant/` and `.opencode/commands/ant/` wrappers — D-02.
- The check's closing card renderer — D-03.

</code_context>

<deferred>
## Deferred Ideas

- Extending shadow isolation to code-valued candidates (WINDOWS #40, second half) — LEARN-06 scoped code-valued candidates out; not part of this closure.
- WINDOWS #43 (three knowingly-empty memory fields) — closes only when a future phase names a consumer; not a Phase 204 gap.
- Guarding every one of the 40 unguarded fixtures — D-09 requires the floor to shrink honestly, not to reach zero in this closure.

</deferred>

---

*Phase: 204-learning-governor (gap closure)*
*Context gathered: 2026-09-14 via /gsd-plan-phase 204 --gaps (one owner question; remaining constraints derived from 204-VERIFICATION.md and WINDOWS.md 38–45)*
