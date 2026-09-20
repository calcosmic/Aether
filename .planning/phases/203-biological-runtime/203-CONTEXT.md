# Phase 203: Biological Runtime - Context

**Gathered:** 2026-09-12
**Status:** Ready for planning

<domain>
## Phase Boundary

Complete the real worker-discovery → governed recruitment → child result → knowledge transfer → downstream decision loop, then make pheromones operational decision inputs. Requirements: SYNTH-05, CEC-07, BIO-01..08. A worker mid-task can emit a typed recruitment intent; Go — never the LLM — admits or refuses it atomically under parent, depth, tree, cycle, permission, path, and cost rules; the child's result returns exactly once and provably changes a recorded downstream decision; pheromone signals move from display-only to outcome-governed decision inputs on one canonical bus.

</domain>

<decisions>
## Implementation Decisions

### Helper Freedom (owner, 2026-09-12)
- **D-01:** Recruitment is automatic within program-enforced limits and **visible as it happens** — no approval prompt by default. The program (BIO-02 admission) is the leash, not an owner gate. Consistent with the Team Check-In fast-path philosophy: nothing pending on the owner → no pause.
- **D-02:** Recruited helpers may recruit further (chains) as long as depth/budget/permission rules hold. Chains are the point of the phase; the limits are the safety.
- **D-03:** A refused recruitment never stalls work: the helper carries on alone, and the refusal is recorded and surfaced (see D-06). No mid-run pause, no owner question on refusal.

### What the Owner Sees (owner, 2026-09-12)
- **D-04:** Live view: **one line per recruit** joining the run (who, why, cost so far), rendered inline in the working terminal. **Hard constraint carried from 202 UAT: the owner never uses a second terminal — all liveness renders inline in the session where the command runs.** No second-terminal watch flows.
- **D-05:** End-of-run summary shows the **family tree with costs** — who recruited whom, what each branch did and cost, refusals noted.
- **D-06:** Refusals show **inline at the moment they happen AND in the summary**. The owner always knows when a helper wanted backup and didn't get it.

### Pheromone Influence / Note Power (owner, 2026-09-12)
- **D-07:** **Suggested** pheromones (system-proposed notes) take effect only after owner approval via a quick tick-to-approve queue. The system never speaks in the owner's voice unasked. (BIO-08 accept/edit/reject surface.)
- **D-08:** When a note changes a decision mid-run: one inline line at the moment it bites, plus a summary list of every decision the owner's notes changed. (CEC-07 credit records make this renderable.)
- **D-09:** Outcome-weighted auto-tuning is ON: good outcomes strengthen a note, bad/neutral outcomes weaken or quarantine it, full immutable history kept, owner can pin or revoke any note. — **Reversibility:** costly — once strengths start moving on outcomes, freezing them later means untangling learned strengths from owner-set ones; the immutable history (BIO-08) is what keeps this undoable at all.
- **D-10:** Cross-project (imported) notes start quarantined and are released **only by the owner**, through the same tick-to-approve queue as suggestions (BIO-07 quarantine + one approval surface, not two).

### Limits (owner, 2026-09-12)
- **D-11:** Default chain depth is **2 levels** (a helper can recruit, and that recruit can recruit once more). Raisable per-run by explicit flag; the default stays modest because speed is the owner's stated priority.
- **D-12:** Recruits draw from the **same team budget** the phase already has — no separate recruit allowance, no new dial, no way for recruitment to silently grow the bill.

### Claude's Discretion
- Wire shapes, schema versions, ledger/manifest formats for RecruitmentIntent/RecruitmentResult/TrophallaxisPacket (BIO-01, BIO-04, BIO-05).
- Probe design for platform availability (BIO-03): configurable, bounded, outside the repository — engineering choices are Claude's.
- Consolidation of the five existing pheromone write sites into one canonical bus (BIO-07) — pure internals.
- How CEC-07 credit records are stored and joined to outcomes.

### Folded Todos
- **ts-host preflight timeout hardcoded, not configurable, runs in repo cwd** (`.planning/todos/2026-08-01-ts-host-preflight-hardcoded-timeout.md`) — BIO-03 requires probes to be *configurable, bounded, process-tree-safe, and isolated from the repository*; this todo is that requirement's existing counter-example and must be retired by the BIO-03 work.
- **Worker turnaround is too slow (~30 min/plan)** (`.planning/todos/2026-08-27-worker-turnaround-is-too-slow.md`) — folded as a *constraint*, not a work item: recruitment admission must add no meaningful latency to the plain no-recruitment path, and nothing in this phase may add a new mandatory ceremony step. Never buy capability with turnaround time.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Phase requirements and synthesis discipline
- `.planning/ROADMAP.md` §Phase 203 — goal, five success criteria.
- `.planning/REQUIREMENTS.md` — SYNTH-05 (mechanism study gate: study BEFORE design is fixed), CEC-07 (credit only with recorded decision + verified outcome), BIO-01..08 (full requirement text).
- `.planning/research/v1.28-classic-synthesis-template.md` — the CLASSIC-SYNTHESIS.md artifact this phase must produce (SYNTH-07 ratchet checks it exists and cites evidence).

### Prior phase contracts this phase builds on
- `.planning/phases/201-queen-led-work-cycle/201-CONTEXT.md` — one verification boundary, one authoritative Go accept/verify/advance path (D-04 there): recruitment results must flow into THAT path, not a new lane.
- `.planning/phases/202-swarm-oracle-and-live-colony/202-CONTEXT.md` — typed event trail feeding all screens (D-01), Classic visual voice (D-04): recruit lines and family trees render through the same event boundary, in the same voice.

### Codebase
- `.planning/codebase/ARCHITECTURE.md`, `.planning/codebase/CONCERNS.md` — current structure and known debt.

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `pkg/agent/spawn_tree.go` — existing spawn tree with parent/child tracking; the natural substrate for depth/tree/cycle admission checks (BIO-02) rather than a new structure.
- `pkg/agent/pool.go` + stream manager — worker lifecycle and streaming already sync with the spawn tree.
- Phase 202's typed live-event boundary — recruit/refusal lines and Oracle-style live rendering reuse it (`TestEveryLiveEventGoesThroughOneBoundary` locks the single boundary).
- Worker handoff store (`.aether/data/handoffs/worker-handoffs.json`) — precedent for scoped knowledge transfer; TrophallaxisPacket (BIO-05) is its governed, acknowledged sibling.

### Established Patterns
- Admission/refusal-by-name pattern from coherent jobs (`TestCoherentJobProposalOrderRefusedByName`): refuse with the offending item named, substitute safe alternatives, never blow the whole plan apart. BIO-02 refusals should read the same way.
- "The program decides, not the wrapper": grouping, depth, and checkin decisions are all Go-owned. Recruitment admission must follow.
- Nothing exists named RecruitmentIntent today — BIO-01 is green-field; no legacy shim needed.

### Integration Points
- Pheromone writes currently spread across five files (`cmd/pheromone_write.go`, `cmd/agency_contract.go`, `cmd/codex_workflow_cmds.go`, `cmd/internal_cmds.go`, `cmd/phase_end_signals.go`) — BIO-07's canonical bus consolidates these call sites onto one writer.
- Colony-prime prompt assembly — where effective pheromone scope/strength lands in worker briefs today; the resolver (BIO-07) must be its single source.
- The 201 accept/verify/advance path — where a recruitment result's "recorded downstream decision" (BIO-04/BIO-05, CEC-07) must land.

</code_context>

<specifics>
## Specific Ideas

- Owner's standing voice: "I never cared to have a second terminal window to view the colony stuff... view within the one terminal that you're working" (2026-09-12, during 202 UAT). Every liveness surface in this phase is inline-first; `aether watch` compatibility is not a design driver.
- One approval surface for everything owner-gated: suggested notes and quarantined imports land in the same tick-to-approve queue (the existing suggest-approve pattern), never two different ceremonies.

</specifics>

<deferred>
## Deferred Ideas

- None raised during discussion.

### Reviewed Todos (not folded)
- **Spec builder — user-facing command developing a readable specification before building** (`.planning/todos/2026-08-20-spec-builder-feature.md`) — planning-surface feature, unrelated to the biological runtime; belongs in its own phase if it survives prioritization.

</deferred>
