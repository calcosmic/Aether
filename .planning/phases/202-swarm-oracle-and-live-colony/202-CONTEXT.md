# Phase 202: Swarm, Oracle, and Live Colony - Context

**Gathered:** 2026-09-11
**Status:** Ready for planning

<domain>
## Phase Boundary

Restore three owner-visible capabilities over the modern Go kernel: a real live colony cockpit (`watch`/status) driven by replayable typed runtime events; substantive Swarm diagnosis that attacks a stubborn defect through four genuinely distinct evidence-producing lenses, compares hypotheses, and safely applies a ranked repair; and iterative Oracle research that clarifies the question, runs multi-round evidence gathering with visible confidence and contradictions, and produces a source-grounded synthesis. Useful partial work stays durable, discoverable, and honestly labelled.

Required mechanism study first (SYNTH-04): reconstruct the exact Classic Swarm, Oracle, and live-display control loops — including their presentation — and compare them with the current issuance, research, event, and renderer paths before selecting the modern synthesis. A `202-CLASSIC-SYNTHESIS.md` following `.planning/research/v1.28-classic-synthesis-template.md` must exist before implementation plans are approved (SYNTH-07).

This phase does not absorb: recruitment/delegation/pheromone causality or TypeScript-host preflight work (Phase 203), learning governance and outcome-backed promotion (Phase 204), final acceptance (Phase 205), or the build/continue/run work cycle delivered in Phase 201.

</domain>

<decisions>
## Implementation Decisions

### Live View Experience (LIVE-01, LIVE-02, CEC-05)

- **D-01:** The live view is a dashboard that updates in place — active workers, castes, lineage, waves, workspaces, current questions, confidence, contradictions, signals, findings, recovery state, elapsed time, reported cost — with a short event ticker strip at the bottom showing the last few things that happened. Both surfaces render from the same typed event stream.
- **D-02:** Default detail focuses on the active work: the current wave and its workers in depth (identity, current question, findings so far), with the wider colony compressed into one compact header line. Drill-down reaches the rest.
- **D-03:** With nothing running, the live view shows a replay-backed summary of the most recent episode — what ran, how it ended, what it cost — plus the one suggested next command. Real replayed history, never invented activity; falls back to the honest idle card only when no episode history exists.
- **D-04 (owner steer, 2026-09-11):** The visual character of this phase is the February–April Classic era. The live dashboard, Swarm investigation, and Oracle rounds should look and feel like the Classic colony did — caste emoji + ant glyphs, visible waves, ceremony framing, Queen voice — reconstructed via the SYNTH-04 mechanism study with the v5.4.0 tag as the visual anchor. Modern-minimal status screens are explicitly not the goal; the classic presentation is, over truthful typed events.

### Swarm Steering and Repair (LIVE-03, LIVE-04, LIVE-05)

- **D-05:** Swarm applies its top-ranked repair automatically under the Phase 201 safety pattern: checkpoint saved (announced), repair applied through an authorized runtime path, re-verified, rolled back with announcement if verification fails. The owner gets the full story afterwards rather than a mid-flight approval gate.
- **D-06:** The four lenses are visible live: the investigators appear in the live dashboard like any other workers, each lens named, hypotheses and contradictions surfacing as they form. The hypothesis comparison and repair ranking render as one card at the end.
- **D-07:** Three failed repairs on the same problem escalate as an architectural question to the owner — a plain-language case naming what failed three times, why patching is not working, and the structural change Swarm believes is needed — then wait for the owner's decision. The proven three-strike escalation is preserved, not softened into a bare stop.

### Oracle Conversation Shape (LIVE-06, LIVE-07)

- **D-08:** Oracle clarifies the actual question up front, then runs autonomously — rounds, confidence movement, and contradictions visible in the live view — and returns one final answer. No per-round check-ins.
- **D-09:** Research depth uses the same Fast / Balanced / Deep / Exhaustive picker Phase 200 established for planning, with each preset's confidence target and round cap shown. One habit across the product; no Oracle-specific dial in the owner's face (the existing confidence-target machinery may implement the presets internally).
- **D-10:** The final synthesis leads with the actionable recommendation in plain language, states confidence and unresolved questions honestly, and keeps sources and the full evidence trail beneath for on-demand reading.

### Durable, Discoverable Outcomes (LIVE-05, LIVE-07)

- **D-11:** Finished Swarm episodes and Oracle syntheses surface through status and history: the status dashboard shows the most recent episode with its outcome; the history command lists all episodes with outcomes and cost, each pointing to its full write-up. Nothing lives only as a loose file the owner must remember.
- **D-12:** Useful partial work (partial research, unproven repair ideas, plan research, Dreams) is kept and listed alongside finished work, always labelled in plain words — e.g. "useful notes, not verified" — with the listing naming what would make it verified. Nothing half-done ever reads as checked truth.

### Claude's Discretion

- The typed event schema, versioning, replay/resume mechanics, and whether the existing `cmd/` event bus and `pkg/events` unify or bridge — provided one versioned model reaches all renderers (LIVE-01).
- What the four Swarm lenses actually are and how their evidence is compared and ranked, provided they are genuinely distinct and evidence-producing, not four cosmetically different prompts (LIVE-03).
- Internal Go types, episode storage shape, exact card spacing/color, refresh cadence, and how the ticker buffers — provided the decisions above, thin-wrapper/runtime authority, and Claude/OpenCode semantic parity hold.
- How the Phase 200 depth presets map onto Oracle's existing confidence-target and iteration-cap machinery.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Milestone Authority

- `.planning/PROJECT.md` — v1.28 Classic restoration goal and Phase 199-205 sequence.
- `.planning/ROADMAP.md` — Phase 202 boundary, dependency on Phase 201, requirements, five success criteria.
- `.planning/REQUIREMENTS.md` — SYNTH-04, CEC-05, LIVE-01..07 full text.
- `.planning/STATE.md` — Current position; prior milestone decisions not to reopen without new evidence.

### Classic Evidence and Required Synthesis

- `.aether/dreams/2026-09-01-comprehensive-aether-colony-review.md` — Governing causal review of Classic vs current runtime.
- `.planning/research/v1.28-classic-capability-ledger.md` — CAP rows routed to Phase 202 with required modern dispositions.
- `.planning/research/v1.28-classic-synthesis-template.md` — Mandatory mechanism-study structure; `202-CLASSIC-SYNTHESIS.md` must precede approved implementation plans (SYNTH-07).
- Historical anchor `v5.4.0` tag — Classic behavior AND visual baseline for watch, Swarm, and Oracle presentation (D-04).
- `cmd/testdata/classic-contract/v1/schema.json` and `cmd/testdata/classic-contract/v1/mechanisms.json` — Versioned Classic contract corpus to extend with Phase 202 coverage.

### Prior Phase Contracts

- `.planning/phases/199-front-door-and-classic-contract/199-CONTEXT.md` — Locked vocabulary, Queen voice, no invented activity.
- `.planning/phases/200-iterative-planning/200-CONTEXT.md` — The Fast/Balanced/Deep/Exhaustive preset pattern (D-13 there) that D-09 here reuses; confidence-dimension presentation conventions.
- `.planning/phases/201-queen-led-work-cycle/201-CONTEXT.md` — Checkpointed repair with announced save/restore (D-09/D-10 there), closeout ceremony/cost/elapsed rules (D-05..D-08 there), and attempt-bound evidence that D-05/D-11 here build on.
- `.planning/phases/201-queen-led-work-cycle/201-CLASSIC-SYNTHESIS.md` — Phase 201's mechanism study; the work-cycle synthesis Phase 202's study extends.

### Current Architecture

- `.planning/codebase/ARCHITECTURE.md`, `.planning/codebase/CONCERNS.md`, `.planning/codebase/CONVENTIONS.md`, `.planning/codebase/TESTING.md` — Layering, known defects, Go conventions, test rules.
- `.aether/docs/wrapper-host-contract.md` — Thin-wrapper contract; wrappers render, never own lifecycle truth.

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets

- `cmd/eventbus.go`, `cmd/event_bridge.go`, `cmd/event_stream.go`, `cmd/event_types.go`, `cmd/event_reader.go`/`event_writer.go` and `pkg/events/` (bus, event, ceremony, agent tokens) — two existing event systems that LIVE-01's one versioned model must unify or bridge.
- `cmd/compatibility_cmds.go` `watchCmd` + `cmd/watch_idle_199_test.go` — today's honest idle watch fallback; D-03 keeps it as the no-history floor.
- `cmd/swarm.go`, `cmd/swarm_cmd.go`, `cmd/swarm_issuance.go`, `cmd/swarm_strikes.go`, `cmd/swarm_display.go` — runtime-issued identities, three-strike machinery, cleanup truth, and display scaffolding to build the four-lens diagnosis on. No lens concept exists yet.
- `cmd/oracle_loop.go`, `cmd/oracle_iterate_cmd.go`, `cmd/oracle_progress.go`, `cmd/oracle_research_doc.go`, `cmd/oracle_promote.go` — the confidence-target loop (default 95), iteration, progress, research-doc, and promotion machinery D-08..D-10 present through the classic ceremony.
- Phase 201's WorkOutcome/closeout ceremony, checkpointed-repair round (`cmd/work_repair.go`), and attempt-bound evidence — the safety and ceremony substrate D-05 and D-11 reuse.
- `cmd/codex_visuals.go` caste identity system (emoji + ant glyphs, ANSI labels, deterministic names, stage markers) — the classic house style D-04 extends to the live view.

### Established Patterns

- Go owns state, verification, issuance, and advancement; wrappers render and spawn but never compute truth.
- No invented activity: every rendered fact traces to a typed, replayable event or durable record (Phase 199 rule; CEC-05 makes it structural).
- Evidence, cost, and elapsed time bind to the exact attempt (CEC-06, shipped in 201).
- Tests prove behavior through fixtures derived the way the runtime derives them, never plausible literals.

### Integration Points

- The live dashboard and the event ticker must render from the same typed stream the status/history commands replay — one source, three surfaces.
- Swarm's checkpoint/rollback must be the same checkpoint concept as Phase 201's bounded repair and Phase 199's pause/resume, not a third mechanism.
- Oracle's preset picker shares presentation with `/ant-plan`'s Phase 200 picker.
- Episode/synthesis records join the attempt-bound evidence model so status, history, and later phases (203-204) consume one lineage.

</code_context>

<specifics>
## Specific Ideas

- **Owner steer (verbatim intent, 2026-09-11):** "we want things to have a bit... the visual aspect of the February, April time" — the classic-era visual presentation is a phase requirement, not polish. The SYNTH-04 study must reconstruct what the Feb–Apr watch/Swarm/Oracle screens actually looked like and why they felt alive, and the modern synthesis must deliver that feel over truthful events.
- The owner took every recommended option quickly and consistently: prefer the alive, classic, watchable experience with automatic safe action and honest reporting over approval gates and minimal screens.

</specifics>

<deferred>
## Deferred Ideas

- Recruitment, delegation, trophallaxis, and operational pheromones — Phase 203.
- Learning governance and outcome-backed credit for Swarm/Oracle lessons — Phase 204.
- Codex-native lifecycle skills — later milestone.

### Reviewed Todos (not folded)

- `2026-08-01-ts-host-preflight-hardcoded-timeout.md` — stays routed to Phase 203 (host preflight ownership), per Phases 200 and 201 review.
- `2026-08-20-spec-builder-feature.md` — delivered by Phase 200; nothing remains.
- `2026-08-27-worker-turnaround-is-too-slow.md` — folded into and delivered by Phase 201 (WORK-08); nothing remains for this phase.

</deferred>

---

*Phase: 202-swarm-oracle-and-live-colony*
*Context gathered: 2026-09-11*
