# Phase 201: Queen-Led Work Cycle - Context

**Gathered:** 2026-09-10
**Status:** Ready for planning

<domain>
## Phase Boundary

Restore visible Queen judgment, proportionate teams, coherent jobs and waves, one verification boundary, honest result truth, bounded recovery, and goal-level autopilot as a single work story over the modern Go kernel. The Queen selects the smallest capable team from the work and its risks and explains it; related tasks form dependency-safe jobs with workspace leases, receipts, and result lineage; build, continue, and run consume one accepted result model and verify exactly once at the authoritative boundary; every outcome — success, no-change, partial, blocker, timeout, interrupted — persists against the exact attempt and governs advance; failures get checkpointed, bounded, verified repair; and per-job timing is measured so representative one-plan turnaround can be improved without weakening TDD or the deterministic safety floor.

Required mechanism study first (SYNTH-03): reconstruct how Classic Queen judgment, caste selection, job/wave coordination, build, continue, run, checks, recovery, context relay, and ceremony actually worked; compare those causal mechanisms with the current attempt/manifest/receipt/verification kernel before choosing changes. A `CLASSIC-SYNTHESIS.md` following `.planning/research/v1.28-classic-synthesis-template.md` must exist before implementation plans are approved (SYNTH-07).

This phase does not absorb: substantive live Swarm/Watch/Oracle behavior (Phase 202), causal pheromone delivery or TypeScript-host preflight/timeout work (Phase 203), learning governance and outcome-backed promotion (Phase 204), final acceptance (Phase 205), or the planning loop and SPEC lifecycle already delivered in Phase 200.

</domain>

<decisions>
## Implementation Decisions

### One Verification Boundary (WORK-04)

- **D-01:** The Queen chooses, per phase, which boundary carries reviewer judgment — build-end or the continue/check step — and records the reason. Verification still happens exactly once per phase; the doubled build+continue review is eliminated. Deterministic checks (build, vet, tests, lint) remain the unskippable floor at every depth regardless of the Queen's choice (D11).
- **D-02:** The Queen's default with no strong signal is the continue/check step. Moving reviewer judgment to build-end requires a recorded reason (e.g. risky changes worth catching before dependent work continues).
- **D-03:** When a build finishes before verification has run, the closeout is an honest unverified card: built, deterministic checks passed, not yet verified, run `/ant-continue`. No "done" or success language before verification.
- **D-04:** One authoritative Go path owns accept/verify/advance. Every lane — direct `aether continue`, plan-only + finalize, and the autopilot/host lane — calls that single function. Lane parity becomes structural, not maintained by hand or by a parity test alone. — **Reversibility:** costly — undoing this after lanes are collapsed means re-splitting verification logic across three files and reintroducing the manual-parity burden the collapse removed.

### Result Truth and Outcome Cards (WORK-05, CEC-06)

- **D-05:** Every outcome — success, no-change, partial, blocker, timeout, interrupted — gets the same full Queen closeout ceremony (colony identity, participating ants, what happened, evidence, state changes, next choices). Only the verdict changes; a partial result looks as considered as a success. Nothing non-successful is ever dressed as success.
- **D-06:** Every build/continue/run closeout ends with elapsed time and reported cost for that exact attempt; `/ant-status` shows the running colony total. Unreported cost is named as unreported, never estimated or guessed.
- **D-07:** After a non-success outcome, the Queen recommends exactly one outcome-matched next action (retry unfinished tasks, answer the blocker, resume the interrupted attempt) and explains in a sentence why she recommends it over the alternatives, which are listed beneath.
- **D-08:** Result cards name both credited files (linked to completed tasks) and orphaned/uncredited edits with where they live. Exact-attempt file precision; no silent absorption and no hiding orphans behind a drill-down.

### Recovery and Bounded Repair (WORK-06)

- **D-09:** Automatic repair is one checkpointed round: checkpoint, one repair wave, re-verify once. If verification still fails, restore the checkpoint and pause with the evidence. No second automatic attempt.
- **D-10:** Checkpoint and rollback are both announced in the flow — "checkpoint saved" before repair, "restored to checkpoint" after a failed repair — so the owner always knows the project's real position.
- **D-11:** The failed-repair handback contains: a plain-language diagnosis of what is failing, what the repair attempted and why it didn't take, the restored safe position, and one concrete question or action for the owner.
- **D-12:** Repair waves consume failure-born signals from the same phase immediately — a signal created from this phase's own failures reaches the repair workers' briefs so the second attempt cannot repeat the first attempt's mistake.

### Turnaround Targets and Telemetry (WORK-08)

- **D-13:** Measure first, then set the target: this phase builds per-job telemetry (queue, preflight, model, tool-call, context, work, verification, wait), records a real baseline for a representative one-plan job, and sets the numeric latency target from that data mid-phase — not from aspiration. Today's reference points: ~30 min per plan, test suite ~12–20 min as the largest term.
- **D-14:** Timing renders as one line in each closeout — total plus the biggest term (e.g. "ran 9m — 6m of it verification") — with the full per-segment breakdown in `/ant-status` drill-down and history.
- **D-15:** Approved speed levers: (a) targeted test lanes — a change to one area runs that area's tests during the loop while the full suite still gates the phase boundary; (b) slimmer briefs and compact handoffs — workers receive only task-relevant context; (c) model routing by caste — faster/cheaper models for mechanical roles where quality is provably unaffected, with the cost line showing what each worker cost. Parallel-wave expansion was NOT approved as a lever (past timing flakes came from contention). TDD and the deterministic safety floor are untouchable.
- **D-16:** Telemetry is report-only in this phase. No automatic self-tuning of routing from timing data; that belongs to Phase 204's learning governance once the numbers are trusted.

### Claude's Discretion

- Internal Go types, attempt/receipt schema details, exact card spacing/color, checkpoint storage mechanics, and test-lane selection heuristics — provided the decisions above, thin-wrapper/runtime authority, and cross-platform (Claude/OpenCode) semantic parity hold.
- The handback design details beyond D-11's required contents.
- Whether any given phase's risk signals justify a build-end boundary is the Queen's judgment (per D-01/D-02), not a keyword list to hardcode — but the five named risk signals forcing reviewers (credentials/auth, payments, release sign-off, data deletion, database migration) remain as ruled in v1.27.

### Folded Todos

- `2026-08-27-worker-turnaround-is-too-slow.md` — A one-plan job costs ~30 minutes; owner-named priority. Folded into WORK-08 scope via D-13..D-16: measure the real per-job breakdown, then cut latency through test lanes, slimmer briefs, and caste model routing without weakening TDD or safety gates.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Milestone Authority

- `.planning/PROJECT.md` — v1.28 Classic restoration goal, Phase 199-205 sequence, todo routing (turnaround → 201).
- `.planning/ROADMAP.md` — Phase 201 boundary, dependency on Phases 199-200, requirements, five success criteria.
- `.planning/REQUIREMENTS.md` — SYNTH-03, CEC-06, WORK-01..08 full text.
- `.planning/STATE.md` — Current position and prior milestone decisions not to reopen without new evidence.
- `.planning/research/priority-spec-v3-backlog.md` — Owner-ratified ordering; D11 (deterministic floor, reviewers are judgment) and D12 governing rulings at `.planning/decisions/2026-08-21-owner-rulings-priority-spec-v3.md`.

### Classic Evidence and Required Synthesis

- `.aether/dreams/2026-09-01-comprehensive-aether-colony-review.md` — Governing causal review of Classic vs current runtime; reconstruct mechanisms, don't copy files.
- `.planning/research/v1.28-classic-capability-ledger.md` — CAP rows routed to Phase 201 with required modern dispositions.
- `.planning/research/v1.28-classic-synthesis-template.md` — Mandatory mechanism-study structure; a `201-CLASSIC-SYNTHESIS.md` must precede approved implementation plans (SYNTH-07).
- Historical anchor `v5.4.0` tag — Classic behaviour baseline for build/continue/run ceremony, team sizes (3–4 workers for a small fix), and wave presentation.

### Prior Phase Contracts

- `.planning/phases/199-front-door-and-classic-contract/199-CONTEXT.md` — Locked `/ant-*` vocabulary, autopilot boundaries (D-05..D-08), single pause/resume, Queen voice rules, no invented activity.
- `.planning/phases/200-iterative-planning/200-CONTEXT.md` — Locked plan acceptance before build/autopilot (D-16 there), Go state authority, thin wrappers, evidence-first cards.
- `.planning/phases/199-front-door-and-classic-contract/199-CLASSIC-SYNTHESIS.md` — Modern front-door/runtime split that routes substantive work-cycle restoration to this phase.
- `cmd/testdata/classic-contract/v1/schema.json` and `cmd/testdata/classic-contract/v1/mechanisms.json` — Versioned semantic Classic contract corpus to extend with Phase 201 mechanism coverage.

### Current Architecture and Known Defects

- `.planning/codebase/ARCHITECTURE.md` — Wrapper/runtime/state layering; build/continue entry points.
- `.planning/codebase/CONCERNS.md` — Named defects this phase resolves: same phase verified twice (build Watcher + continue Probe/Auditor); three continue lanes with manual-parity risk (`cmd/codex_continue.go`, `cmd/codex_continue_plan.go`, `cmd/codex_continue_finalize.go`); plan-only+finalize lane rejecting reconciliation the direct lane accepts.
- `.planning/codebase/CONVENTIONS.md` and `.planning/codebase/TESTING.md` — Go command organization, co-located tests, test conventions for the test-lane work.
- `.aether/docs/wrapper-host-contract.md` — Thin-wrapper contract; wrappers never own competing lifecycle truth.
- `.planning/todos/pending/2026-08-27-worker-turnaround-is-too-slow.md` — Original owner turnaround complaint and measured breakdown.

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets

- `cmd/codex_build.go` (3582 lines) — Build orchestration, dispatch, depth-adjusted worker caps, coherent-job grouping, one-worker fast path, team check-in. The Queen-choice and proportionate-team machinery from v1.27 already exists and is test-locked; this phase builds on it rather than re-deciding it.
- `cmd/codex_continue.go` / `cmd/codex_continue_plan.go` / `cmd/codex_continue_finalize.go` — The three continue lanes to be collapsed onto one authoritative Go accept/verify/advance path (D-04).
- `cmd/codex_build_finalize.go` — Two-stage receipt admission/finalization credit model; the exact-attempt result lineage D-08 renders comes from here.
- `.aether/data/handoffs/worker-handoffs.json` + relay-note machinery — The compact handoff channel D-15's slimmer briefs build on.
- Midden → signal pipeline (3-failures-of-one-kind → one REDIRECT) — The failure-learning loop D-12 wires into same-phase repair briefs.

### Established Patterns

- Go owns all state mutation, verification, gating, advancement; wrappers render and spawn but never compute lifecycle truth.
- Deterministic checks run on every phase at every depth; a smaller worker count never means less checking (D11, test-locked).
- Reviewers forced only by five named risk signals; only the owner can waive one; waivers persist per signal per phase.
- Tests prove behavior through semantic fixtures and state/evidence assertions, never fixture values the runtime cannot produce.
- Canonical `.aether/commands/*.yaml` sources generate `.claude/` and `.opencode/` wrappers; the three surfaces stay synchronized.

### Integration Points

- The Queen's per-phase boundary choice (D-01) must reach the actual dispatch list, not an intermediate record a later step can override (`TestQueenChoiceReachesTheDispatchList` pattern).
- The honest unverified build card (D-03) and single-verify story must hold on both the interactive wrapper lane and the autopilot/host lane.
- Repair checkpoints (D-09/D-10) interact with `/ant-pause`/`/ant-resume` handoff machinery from Phase 199 — one checkpoint concept, not two competing ones.
- Per-job telemetry (D-13) instruments the existing attempt/receipt records so timing binds to the same durable attempt identity that evidence and cost already use (CEC-06).
- Phase 199's Classic contract corpus gains Phase 201 mechanism/case coverage tying visible build/continue/run behavior to authoritative Go transitions.

</code_context>

<specifics>
## Specific Ideas

- The owner's standing bar: Aether should feel as snappy as plain Claude on simple tasks; the ~30-minute one-plan job is why daily use stopped. Speed work is the phase's business value, but never bought by dropping TDD or the deterministic floor.
- The measured Classic reference: v5.4.0 sent 3–4 workers for a small fix where the modern path once sent 8; v1.27 got the ordinary case to 1 Builder + free checks. This phase completes the story — one verification, one narrative, honest cards — without regressing that win.
- The default closeout should read like the Queen reporting to the owner, not a protocol dump: who went, what happened, what it proves, what it cost, and the one thing to do next with the reason.

</specifics>

<deferred>
## Deferred Ideas

- Substantive live Swarm/Watch/Oracle behavior and the typed event bridge — Phase 202.
- Causal pheromone delivery and shared-environment effects — Phase 203.
- Automatic self-tuning of routing/test lanes from telemetry — Phase 204 learning governance (per D-16).
- Codex-native `$ant-*` lifecycle skills — later milestone.

### Reviewed Todos (not folded)

- `2026-08-01-ts-host-preflight-hardcoded-timeout.md` — Considered again here (preflight time is one WORK-08 segment) but stays routed to Phase 203 per Phase 200's review; folding it would split host preflight ownership across two phases.
- `2026-08-20-spec-builder-feature.md` — Already folded into and delivered by Phase 200; nothing remains for this phase.

</deferred>

---

*Phase: 201-queen-led-work-cycle*
*Context gathered: 2026-09-10*
