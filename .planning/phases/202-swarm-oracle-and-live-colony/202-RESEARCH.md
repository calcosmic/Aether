# Phase 202: Swarm, Oracle, and Live Colony - Research

**Researched:** 2026-09-11
**Domain:** Go runtime event modeling, live terminal rendering, multi-lens diagnostic orchestration, iterative research loops
**Confidence:** MEDIUM-HIGH — every claim below is grounded in code actually read this session or in the owner-approved CONTEXT.md/REQUIREMENTS.md/ledger; the one genuinely new subsystem (the unified typed event model) has no existing implementation to verify against, so its shape is a recommendation, not a restored fact.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

**Live View Experience (LIVE-01, LIVE-02, CEC-05)**
- **D-01:** The live view is a dashboard that updates in place — active workers, castes, lineage, waves, workspaces, current questions, confidence, contradictions, signals, findings, recovery state, elapsed time, reported cost — with a short event ticker strip at the bottom showing the last few things that happened. Both surfaces render from the same typed event stream.
- **D-02:** Default detail focuses on the active work: the current wave and its workers in depth (identity, current question, findings so far), with the wider colony compressed into one compact header line. Drill-down reaches the rest.
- **D-03:** With nothing running, the live view shows a replay-backed summary of the most recent episode — what ran, how it ended, what it cost — plus the one suggested next command. Real replayed history, never invented activity; falls back to the honest idle card only when no episode history exists.
- **D-04 (owner steer, 2026-09-11):** The visual character of this phase is the February–April Classic era. The live dashboard, Swarm investigation, and Oracle rounds should look and feel like the Classic colony did — caste emoji + ant glyphs, visible waves, ceremony framing, Queen voice — reconstructed via the SYNTH-04 mechanism study with the v5.4.0 tag as the visual anchor. Modern-minimal status screens are explicitly not the goal; the classic presentation is, over truthful typed events.

**Swarm Steering and Repair (LIVE-03, LIVE-04, LIVE-05)**
- **D-05:** Swarm applies its top-ranked repair automatically under the Phase 201 safety pattern: checkpoint saved (announced), repair applied through an authorized runtime path, re-verified, rolled back with announcement if verification fails. The owner gets the full story afterwards rather than a mid-flight approval gate.
- **D-06:** The four lenses are visible live: the investigators appear in the live dashboard like any other workers, each lens named, hypotheses and contradictions surfacing as they form. The hypothesis comparison and repair ranking render as one card at the end.
- **D-07:** Three failed repairs on the same problem escalate as an architectural question to the owner — a plain-language case naming what failed three times, why patching is not working, and the structural change Swarm believes is needed — then wait for the owner's decision. The proven three-strike escalation is preserved, not softened into a bare stop.

**Oracle Conversation Shape (LIVE-06, LIVE-07)**
- **D-08:** Oracle clarifies the actual question up front, then runs autonomously — rounds, confidence movement, and contradictions visible in the live view — and returns one final answer. No per-round check-ins.
- **D-09:** Research depth uses the same Fast / Balanced / Deep / Exhaustive picker Phase 200 established for planning, with each preset's confidence target and round cap shown. One habit across the product; no Oracle-specific dial in the owner's face (the existing confidence-target machinery may implement the presets internally).
- **D-10:** The final synthesis leads with the actionable recommendation in plain language, states confidence and unresolved questions honestly, and keeps sources and the full evidence trail beneath for on-demand reading.

**Durable, Discoverable Outcomes (LIVE-05, LIVE-07)**
- **D-11:** Finished Swarm episodes and Oracle syntheses surface through status and history: the status dashboard shows the most recent episode with its outcome; the history command lists all episodes with outcomes and cost, each pointing to its full write-up. Nothing lives only as a loose file the owner must remember.
- **D-12:** Useful partial work (partial research, unproven repair ideas, plan research, Dreams) is kept and listed alongside finished work, always labelled in plain words — e.g. "useful notes, not verified" — with the listing naming what would make it verified. Nothing half-done ever reads as checked truth.

### Claude's Discretion
- The typed event schema, versioning, replay/resume mechanics, and whether the existing `cmd/` event bus and `pkg/events` unify or bridge — provided one versioned model reaches all renderers (LIVE-01).
- What the four Swarm lenses actually are and how their evidence is compared and ranked, provided they are genuinely distinct and evidence-producing, not four cosmetically different prompts (LIVE-03).
- Internal Go types, episode storage shape, exact card spacing/color, refresh cadence, and how the ticker buffers — provided the decisions above, thin-wrapper/runtime authority, and Claude/OpenCode semantic parity hold.
- How the Phase 200 depth presets map onto Oracle's existing confidence-target and iteration-cap machinery.

### Deferred Ideas (OUT OF SCOPE)
- Recruitment, delegation, trophallaxis, and operational pheromones — Phase 203.
- Learning governance and outcome-backed credit for Swarm/Oracle lessons — Phase 204.
- Codex-native lifecycle skills — later milestone.
- `2026-08-01-ts-host-preflight-hardcoded-timeout.md` stays routed to Phase 203.
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| SYNTH-04 | Reconstruct exact Classic Swarm/Oracle/live-display control loops and compare with current issuance/research/event/renderer paths before choosing the synthesis. | Section "Classic Mechanism Evidence" below is the raw material; the mandatory `202-CLASSIC-SYNTHESIS.md` (SYNTH-07 gate) must be authored from it before plans are approved — see Metadata note. |
| CEC-05 | Planning, building, Swarm, Oracle, recovery, and verification expose typed live events, not simulated activity. | "Current Event Infrastructure" + "Don't Hand-Roll" sections identify the two existing event systems and the unification gap. |
| LIVE-01 | One versioned runtime event model to all renderers, with replay/resume. | "Current Event Infrastructure", "Architecture Patterns" Pattern 1. |
| LIVE-02 | Real live cockpit in `watch`/status. | "Current Watch Behavior", "Architecture Patterns" Pattern 2. |
| LIVE-03 | Substantive four-lens Swarm diagnosis, hypothesis comparison, ranked repair. | "Current Swarm Architecture", "Architecture Patterns" Pattern 3, "Common Pitfalls" 1/2. |
| LIVE-04 | Checkpointed, verified, rollback-safe Swarm repair; preserved three-strike escalation. | "Current Swarm Architecture", "Don't Hand-Roll" row 2 (reuse `work_repair.go`). |
| LIVE-05 | Durable, replay-safe Swarm episode reaching status/history. | "Current Swarm Architecture" (episode storage already exists), CAP-047/048. |
| LIVE-06 | Iterative Oracle: clarify, multi-round, visible confidence/contradictions, diminishing returns. | "Current Oracle Architecture" — most of this already exists; gap is visibility + preset alignment (D-09). |
| LIVE-07 | Durable Oracle synthesis; honest partial-work labelling. | "Current Oracle Architecture" (`oracle_research_doc.go`/`oracle_promote.go` already partially deliver this — Phase 198.2). |

</phase_requirements>

## Summary

Phase 202 is not a greenfield build. Every one of its five owner-visible outcomes (live cockpit, four-lens Swarm, iterative Oracle, durable episodes, honest partial-work labelling) has a **partially-built current mechanism** that this session verified by reading the actual source. The single genuinely new piece of infrastructure is the unified typed event model (LIVE-01) — nothing in the current codebase emits the worker/wave/lineage/question/confidence/contradiction event stream the live cockpit needs, and the one artifact that tried (`cmd/event_types.go`/`event_stream.go`/`event_bridge.go`, a closed `EventType` enum writing NDJSON to `.aether/events/current.ndjson`) has **zero production callers** — it is dead code with only its own tests exercising it `[VERIFIED: cmd/event_stream.go, cmd/event_bridge.go, cmd/event_types.go; grep -rn "EmitPhaseStart|EmitAgentSelected|SyncEventsWithTS" cmd/*.go — no non-test, non-definition call sites]`. The live, wired event system is `pkg/events.Bus` (topic + `json.RawMessage` payload, JSONL-persisted, TTL-pruned), but it is only used today for narrow "ceremony" topics (spawn-tree announcements, loop-break signals) consumed by `status.go` and `spawn.go` — it carries none of the typed worker/wave/question/confidence fields LIVE-02 needs `[VERIFIED: pkg/events/event.go, pkg/events/bus.go:1-80; cmd/status.go:389-419; cmd/spawn.go:139,208,236,246]`.

`watch` today is unconditionally the "honest idle fallback" — its `RunE` calls `buildIdleWatchResult` with no live-data branch at all; the result literally sets `"live_capability": "unsupported", "active_count": 0` every time `[VERIFIED: cmd/compatibility_cmds.go:59-70,448-455]`. This is Phase 199's deliberate, tested floor ("Without a Phase 202 typed event source, Watch reports idle" — `.planning/phases/199-front-door-and-classic-contract/199-CONTEXT.md`) — Phase 202 is the phase that must replace it.

Swarm already has a caste roster that closely maps to Classic's four lenses (tracker≈Error Analyst, archaeologist≈Git Archaeologist, scout≈Pattern Hunter — but no analog of Classic's "External Researcher" web-research lens exists in Swarm today) `[VERIFIED: cmd/swarm_cmd.go:1015-1097]`. But the current `runSwarmDestroy` pipeline is strictly linear — one investigation wave whose combined text summary feeds one fix wave whose summary feeds one verification wave — there is no hypothesis comparison, no ranking, and no checkpoint before the fix wave is applied `[VERIFIED: cmd/swarm_cmd.go:274-372]`. The checkpoint/rollback primitive LIVE-04 needs already exists and is proven in Phase 201's bounded-repair round (`saveRepairCheckpoint`/`restoreRepairCheckpoint`/`emitRepairCheckpointSaved`/`emitRepairCheckpointRestored` in `cmd/work_repair.go`) — it should be reused, not reinvented `[VERIFIED: cmd/work_repair.go:240-459]`. Three-strike escalation already exists and is explicitly marked "already proven" in the capability ledger (CAP-046) — `evaluateSwarmStrikeHistory`/`ensureSwarmEscalationForHistory` in `cmd/swarm_strikes.go` `[VERIFIED: cmd/swarm_strikes.go:149-529; .planning/research/v1.28-classic-capability-ledger.md:179]`.

Oracle is the most mature of the three subsystems. Its 3,945-line loop already tracks `OverallConfidence`, `Contradictions`, `OpenGaps`, `ActiveQuestionText`, and a `noveltyTracker{ConsecutiveLow, Threshold}` that already implements diminishing-returns detection `[VERIFIED: cmd/oracle_loop.go:294-337]`. Phase 198.2 already gave Oracle a "one file/register/promote body" that labels partial research honestly (`oracleResearchPartialLabel`, `saveOracleResearchDocument`, `runResearchList`) `[VERIFIED: cmd/oracle_research_doc.go:47-335; STATE.md Phase 198.2 bullet]`. The one real gap for LIVE-06/07 is (a) none of this is emitted as a live event anyone can watch, and (b) Oracle's own depth-preset table uses different labels and different numeric targets than Phase 200's Fast/Balanced/Deep/Exhaustive picker — `oracleDepthLevels` calls its cheapest tier `"quick"` (not `"Fast"`) at 60% confidence/5 iterations, while Plan's `Fast` preset is 80% confidence/4 passes `[VERIFIED: cmd/oracle_loop.go:57-64; cmd/codex_plan.go:217-220]`. D-09 explicitly allows the underlying numbers to stay Oracle-specific as long as the four preset **names** are the one shared habit — this is a presentation-layer reconciliation, not new research-loop logic.

**Primary recommendation:** Treat this phase as *wiring and reconciling proven mechanisms behind one new typed event bridge*, not as building four new subsystems. Build the typed event model first (nothing else can be "live" without it), then thread Swarm's linear pipeline into a genuine four-lens compare-then-rank shape reusing Phase 201's checkpoint primitive, then align Oracle's preset labels with Phase 200's picker and emit its already-rich state as events, then make `watch`/status/history read the same event/episode records for both the live and replay-backed idle cases.

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Typed event emission (worker/wave/question/confidence/etc.) | Go runtime (`cmd/`, `pkg/events`) | — | Go owns all durable state transitions today; events must be emitted at the exact call sites that already mutate state (swarm wave dispatch, oracle iteration, build/continue dispatch), never re-derived by a renderer. |
| Live cockpit rendering (`watch`) | Go runtime (`cmd/`) | Platform wrapper (framing only) | Rendering must read the same projection status/history read; a wrapper cannot compute liveness without duplicating the Go projection (CLAUDE.md UX Architecture: "State mutations: Go runtime... Visual rendering: Go runtime"). |
| Swarm four-lens dispatch and hypothesis ranking | Go runtime (`cmd/swarm_cmd.go` + new lens/ranking logic) | — | Worker dispatch, evidence comparison, and repair ranking are all durable decisions; no wrapper authority here (established pattern for Queen judgement, `queenApplyJudgement`). |
| Swarm checkpoint/repair/rollback | Go runtime (reuse `cmd/work_repair.go`) | — | Phase 201 already built and proved this exact mechanism; a second implementation would reproduce the "three parallel implementations" anti-pattern CLAUDE.md documents. |
| Oracle iteration loop, confidence, diminishing returns | Go runtime (`cmd/oracle_loop.go`) | — | Already fully implemented; this phase's job is emission + preset alignment, not new loop logic. |
| Episode/synthesis durability (status/history surfacing) | Go runtime (`cmd/swarm_strikes.go`, `cmd/oracle_research_doc.go`, `cmd/status.go`, history command) | — | D-11 requires status/history to read one shared lineage; must not become a third parallel record type alongside `buildAttemptRecord`/`swarmResultRecord`/oracle research entries — bind them, don't replace them. |
| Colony framing / Queen voice / ceremony narration (D-04 classic visual character) | Platform wrapper markdown (`.claude/`, `.opencode/`) | Go (`cmd/codex_visuals.go` renderer) | Per CLAUDE.md UX Architecture: "Colony framing: Wrapper markdown... Queen persona, narration, pacing." The underlying caste-identity/stage-marker renderer is already Go (`casteEmojiMap`, `renderStageMarker`) and must be reused, not reinvented per-command. |

## Package Legitimacy Audit

**No new external packages are required for this phase.** `go.mod` already contains everything needed: `github.com/jedib0t/go-pretty/v6` (tables), `github.com/schollz/progressbar/v3` (progress rendering), and `github.com/spf13/cobra` for the command surface `[VERIFIED: go.mod:1-20]`. There is **no TUI framework** (`bubbletea`, `tview`, `tcell`) in the dependency tree — the "dashboard that updates in place" (D-01) must be built with the codebase's existing pattern of redrawing/reprinting terminal output on an interval (the `watch` command already accepts `--once`/`--interval` compatibility flags), not a new interactive TUI dependency. Introducing a TUI library would be a scope-expanding, unreviewed dependency add that the CONTEXT.md's "Claude's Discretion" area does not authorize; if a future plan believes one is genuinely needed, that is an owner-facing decision, not a research default.

| Package | Registry | Verdict | Disposition |
|---------|----------|---------|-------------|
| *(none required)* | — | — | No install step; reuse existing `go-pretty`/`progressbar`/cobra dependencies already vetted in prior phases. |

## Classic Mechanism Evidence (input to the mandatory SYNTH-04 study)

This is not the `202-CLASSIC-SYNTHESIS.md` itself — that artifact has its own mandatory template (`.planning/research/v1.28-classic-synthesis-template.md`) and must be authored separately before implementation plans are approved (SYNTH-07 gate, confirmed by the Phase 201 precedent at `.planning/phases/201-queen-led-work-cycle/201-CLASSIC-SYNTHESIS.md`). This section collects the raw evidence a synthesis author needs, cited from the authoritative Dreams review `[CITED: .aether/dreams/2026-09-01-comprehensive-aether-colony-review.md]`:

- **Classic Swarm (`swarm.yaml`/`ant:swarm`):** "With no bug, showed live colony activity. With a bug, launched Archaeologist, Pattern Hunter, Error Analyst and Web Researcher in parallel, cross-compared evidence, ranked fixes, applied the best, verified, rolled back failure, and deposited FOCUS/REDIRECT learning." Four named lenses, added 2026-02-09 (`ant:swarm` for "stubborn bug destruction") and given real-time display 2026-02-14, with specialist spawning added 2026-02-15 (commit `3a5b81c2`, "Open Chambers").
- **Classic Oracle:** "Reformulated the topic into an approved brief; offered templates, 5/15/30/50 iterations, 80/90/95/99 confidence, scope and research strategy; ran survey→investigate→synthesize→verify in visible `tmux`; preserved partial synthesis and offered high-confidence promotion." The `oracleDepthLevels` iteration counts already in Go (5/15/30/50) are an exact numeric match to the historical 5/15/30/50 — this is strong evidence the current Go loop already restored Oracle's numeric shape and only needs the *label* layer reconciled with Phase 200, not the iteration math itself `[VERIFIED: cmd/oracle_loop.go:57-64 shows 5/15/30/50; CITED: dreams review line 201 states "5/15/30/50 iterations"]`.
- **Classic `watch`:** "Created a four-pane 2×2 cockpit: colony status, progress, spawn tree and colorized activity log." The review explicitly reframes the old "do not restore tmux/second-terminal watch" decision as one to reopen: "the implementation may be modern and single-terminal; the capability must not be dismissed because the old transport was shell/tmux" (Decisions reopened, item 1).
- **The removal event:** commit `0063be8b` (2026-04-05) deleted "58 shell scripts... The removed core included the 5,645-line `.aether/aether-utils.sh`, 1,004-line Swarm engine, 1,023-line Oracle loop, and 277-line live Swarm renderer... Removed the old implementation and visibility substrate in one cut." Commit `2c9e2a98` (2026-05-05), "Restore host-orchestrated swarm flow," is the repository's own admission that this removal broke real capability.
- **Proposed typed event vocabulary** (the review's own recommendation, useful as a starting point for the LIVE-01 schema, not a locked spec): `goal_understood, charter_proposed, survey_worker_started/completed, planning_iteration_started, research_gap_selected, plan_confidence_changed, team_proposed/accepted/modified, worker_started/progress/completed, recruitment_requested/refused/admitted/completed, signal_consulted/decision_changed, verification_check_started/passed/failed, learning_proposed/promoted/rejected, phase_advanced, next_action_resolved`.
- **Trust boundary already closed:** the review's own Part II audit flagged the *external* Swarm finalizer as having two Critical vulnerabilities — CR-01 (caller-controlled `SwarmID` used in `MkdirAll`/JSON writes before containment validation) and CR-02 (self-asserted, replayable external manifest with no runtime-issued digest) `[VERIFIED: dreams review lines 1666-1667]`. **These are already fixed.** STATE.md's Phase 198.3 entry confirms: *"External swarm finalization now validates durable IDs before mutation, accepts only runtime-issued manifest digests, makes exact replay read-only, and rejects changed replay before rewriting strike truth."* `[VERIFIED: .planning/STATE.md Phase 198.3 bullet; cmd/swarm_issuance.go:42-245 shows validateDurableSwarmID, issueExternalSwarmManifest, validateExternalSwarmIssuanceRecord, replayExternalSwarmFinalization]` — Phase 202 builds the four-lens diagnosis on top of an already-safe substrate; it does not need to re-close CR-01/CR-02.

## Current Event Infrastructure (LIVE-01, CEC-05)

Two independent, non-interoperating event systems exist today:

| System | Files | Status | Shape |
|---|---|---|---|
| `cmd/event_types.go`/`event_stream.go`/`event_bridge.go` | 3 files, ~297 lines | **Dead code** — zero production callers outside own definitions/tests | Closed `EventType` enum (`phase_start`, `agent_selected`, `task_planned`, `worker_spawned`, `tool_call`, `verification_result`, `memory_update`, `run_seal`, `custom`); writes NDJSON to `.aether/events/current.ndjson`; explicitly documented as "stub mode" for a Go/TypeScript bridge that was never wired |
| `pkg/events.Bus` | `pkg/events/{bus,event,ceremony,agent_token}.go`, ~1567 lines | **Live**, but narrow | Generic `Event{Topic, Payload json.RawMessage, Source, Timestamp, TTLDays}`; JSONL-persisted via `storage.Store`; TTL-pruned; topic-pattern pub/sub. Used today only for `learning.*` topics (memory/curation pipeline) and `ceremony.*` topics (`CeremonyTopicBuildSpawn`, `CeremonyTopicLoopBreak` — spawn-tree announcements and loop-break signals consumed by `status.go`/`spawn.go`) |

`[VERIFIED: cmd/event_types.go full file; cmd/event_stream.go:1-60; cmd/event_bridge.go full file; pkg/events/event.go full file; pkg/events/bus.go:1-80; grep -rln "events.NewBus" cmd/*.go pkg/**/*.go → ceremony_emitter.go, consolidation_lifecycle.go, curation_cmds.go, graph_consolidation_cmds.go, internal_cmds.go, learning_cmds.go, learning.go, memory_feed.go, oracle_promote.go, serve.go, spawn.go, status.go]`

**Neither system carries the fields LIVE-02 needs**: no worker identity, caste, lineage, wave, workspace, question, confidence, contradiction, or finding is a typed field on either `Event` or `EventLine` today — both are generic envelopes (`RawMessage`/`map[string]interface{}`) that some other layer would have to interpret. This is exactly what CONTEXT.md's Claude's Discretion area anticipates: "whether the existing `cmd/` event bus and `pkg/events` unify or bridge — provided one versioned model reaches all renderers."

**Recommendation:** Retire the dead `cmd/event_types.go`/`event_stream.go`/`event_bridge.go` trio (or repurpose its file-write mechanics only if genuinely needed for a durable replay log) and extend `pkg/events.Bus` with new topic constants and typed payload structs for the live-cockpit domain (e.g. `swarm.lens_started`, `swarm.hypothesis_formed`, `oracle.round_started`, `oracle.confidence_changed`, `build.worker_progress`). This is additive to a bus three other subsystems already depend on — a from-scratch bus would fragment three already-wired callers (`memory_feed.go`, `status.go`, `oracle_promote.go`) across two buses, reproducing the exact "three parallel implementations" pattern CLAUDE.md's CONCERNS section already documents as this repo's own recurring failure mode.

## Current Watch Behavior (LIVE-02)

`watchCmd`'s `RunE` is exactly two lines: build the idle result, render it. `[VERIFIED: cmd/compatibility_cmds.go:59-70]`
```go
var watchCmd = &cobra.Command{
	Use:         "watch",
	Short:       "Show the honest idle watch fallback",
	Args:        cobra.NoArgs,
	Annotations: map[string]string{"aether.io/read-only": "true"},
	RunE: func(cmd *cobra.Command, args []string) error {
		_ = cmd // --once/--interval remain accepted compatibility flags.
		result := buildIdleWatchResult(resolveAetherRoot(), store, time.Now().UTC())
		outputWorkflow(result, renderIdleWatchVisual(result))
		return nil
	},
}
```
The idle result it always builds explicitly asserts no liveness: `[VERIFIED: cmd/compatibility_cmds.go:448-455]`
```go
	return map[string]interface{}{
		"schema_version": LifecycleResultSchemaVersion,
		"mode":           "idle_watch", "command": "watch",
		"outcome_kind":    colony.OutcomeKindNoChange,
		"state_effect":    colony.LifecycleStateEffectNone,
		"idle_message":    "No ants are active right now",
		"live_capability": "unsupported", "active_count": 0,
```
`--once`/`--interval` flags already exist on the command and are accepted-but-ignored compatibility flags today — this is the exact seam D-01's "updates in place... event ticker strip" refresh loop should attach to, rather than inventing a new flag surface.

D-03's "replay-backed summary of the most recent episode... falls back to the honest idle card only when no episode history exists" means `buildIdleWatchResult`'s current unconditional idle branch must become conditional: query the swarm/oracle/build episode stores first (all three already durably persist their own records — see below), and only fall back to the literal idle card when none exist.

## Current Swarm Architecture (LIVE-03, LIVE-04, LIVE-05)

**Caste roster and wave assignment** — `swarmWaveForCaste`/`swarmTaskForCaste`/`buildSwarmPlansForWave` `[VERIFIED: cmd/swarm_cmd.go:1015-1097]`:
```go
func swarmWaveForCaste(caste string) int {
	switch caste {
	case "builder", "weaver", "fixer":
		return 2
	case "watcher", "probe":
		return 3
	default:
		return 1
	}
}
```
Wave-1 (investigation) castes available today, per `buildSwarmPlansForWave`'s fixed order: `tracker, scout, archaeologist, gatekeeper, medic` — the Queen selects a subset via `queenSwarmSelectedCastes`. Mapped against Classic's four lenses:

| Classic lens | Current caste analog | Status |
|---|---|---|
| Git Archaeologist | `archaeologist` — "Inspect git history and prior fixes around the bug area" | Exists |
| Pattern Hunter | `scout` — "Search the repo for the most relevant files, patterns, tests, and documentation" | Exists |
| Error Analyst / Tracker | `tracker` — "Reproduce the issue, trace the failure path, and identify the most likely root cause" | Exists |
| External (Web) Researcher | *none* | **Gap** — no Swarm caste performs web/external research today |

`[VERIFIED: cmd/swarm_cmd.go:1071-1097 swarmTaskForCaste bodies quoted verbatim]`. The nearest existing capability for the missing fourth lens is Oracle's own research machinery (`oracle_loop.go`'s worker dispatch already understands `type: documentation | official | github | blog | forum | academic` evidence) — reusing an Oracle-shaped research dispatch as Swarm's fourth lens (rather than building a new web-research mechanism) directly satisfies "Don't Hand-Roll" and keeps LIVE-03's "genuinely distinct, evidence-producing" bar without a fifth parallel implementation.

**The critical gap — no hypothesis comparison or ranking exists.** `runSwarmDestroy`'s control flow is strictly linear: `[VERIFIED: cmd/swarm_cmd.go:274-372]`
```go
	investigation := buildSwarmInvestigationPlans(root, target)
	...
	investigationRuns, err := executeSwarmWave(ctx, root, swarmID, target, investigation, "", invoker)
	...
	findingSummary := renderSwarmFindingSummary(investigationRuns)
	fixPlans := buildSwarmFixPlans(root, target)
	...
	builderRuns, err := executeSwarmWave(ctx, root, swarmID, target, fixPlans, findingSummary, invoker)
```
All wave-1 workers' outputs are concatenated into one `findingSummary` string and handed to the fix wave — there is no step that compares competing hypotheses, ranks candidate repairs, or surfaces contradictions between lenses as a distinct artifact. `renderSwarmFindingSummary` produces prose, not a structured, comparable hypothesis list. This is the exact gap LIVE-03/D-06 requires closing: a genuine comparison step (structured hypothesis + evidence + confidence per lens, cross-compared, contradictions surfaced, one ranked repair selected) must be inserted between the investigation wave and the fix wave, and rendered as "one card at the end" per D-06.

**No checkpoint exists before the fix wave is applied.** `runSwarmDestroy` goes straight from investigation to `executeSwarmWave` for the fix plans with no save-state step. This must reuse Phase 201's already-proven checkpoint/repair primitive rather than invent a second one — CONTEXT.md's own Integration Points section states this explicitly ("Swarm's checkpoint/rollback must be the same checkpoint concept as Phase 201's bounded repair and Phase 199's pause/resume, not a third mechanism"). The reusable pieces `[VERIFIED: cmd/work_repair.go:240-459 function signatures]`:
```
func repairCheckpointIdentity(phase int, check string) string
func saveRepairCheckpoint(root, checkpointID string, paths []string) (repairCheckpoint, error)
func restoreRepairCheckpoint(checkpoint repairCheckpoint) error
func emitRepairCheckpointSaved(phase int, check string)
func emitRepairCheckpointRestored(phase int, check string)
func applyBoundedCheckFixRepair(ctx context.Context, root string, state colony.ColonyState, phase colony.Phase, manifest codexContinueManifest, floor deterministicFloorResult, buildWatcher codexWatcherVerification, workerTimeout, verificationTimeout time.Duration, reviewerDispatched bool) (deterministicFloorResult, *checkFixAttemptRecord, *repairHandback)
```
Swarm's repair round is not phase/check-shaped the way `applyBoundedCheckFixRepair` is (Swarm operates on a free-text `target`, not a phase+check pair) — the *checkpoint save/restore primitive* (`saveRepairCheckpoint`/`restoreRepairCheckpoint`, keyed by a generalized identity, not literally `repairCheckpointIdentity(phase int, check string)`) is what should be reused; the phase-specific wrapper around it is Phase 201's own concern and should not be called directly by Swarm.

**Three-strike escalation already works and must not be touched.** `evaluateSwarmStrikeHistory`, `ensureSwarmEscalationForHistory`, `upsertSwarmEscalationFlag` in `cmd/swarm_strikes.go` already implement exactly D-07's requirement, and the capability ledger marks this row `restore-modern (already proven)` `[VERIFIED: cmd/swarm_strikes.go:149-529; .planning/research/v1.28-classic-capability-ledger.md:179 "CAP-046 | restore-modern (already proven) | LIVE-04 | 202 | Preserve the Phase 198.3 three-strike escalation inside the restored substantive Swarm."]`. This phase's job for CAP-046 is *preservation through the refactor*, not reimplementation — the new lens/checkpoint work must feed strike history through the exact same `swarmResultRecord`/`persistSwarmResultOutcome` path unchanged.

**Episode durability already exists, retention/archive does not.** Every `runSwarmDestroy` run already calls `persistSwarmResultOutcome(store, swarmResultRecord{...})` with `SwarmID, Target, Status, RootCause, Solution, Recommendation, Workers, Files, Tests, Blockers, CompletedAt` `[VERIFIED: cmd/swarm_cmd.go:349-361]` — this is the durable "one issuance-bound replay-safe episode" LIVE-05/CAP-047 asks for; it needs status/history to *read* it, not a new store. What genuinely does not exist: a cleanup/archive/retention mechanism — `grep -rn "func.*[Ss]warmCleanup\|func.*[Ss]warmArchive\|retention" cmd/swarm*.go` (non-test) returns nothing `[VERIFIED: no matches]` — matching the ledger's `replace-better` disposition for CAP-048 ("Cleanup/archive operates on one durable replay-safe Swarm episode with retention metadata" — this is new work, not restoration, since Classic itself did not have durable replay-safe episodes to retain).

## Current Oracle Architecture (LIVE-06, LIVE-07)

Oracle's loop already tracks nearly everything LIVE-06 asks to make visible `[VERIFIED: cmd/oracle_loop.go:300-337, full struct definitions quoted]`:
```go
type oracleStateFile struct {
	...
	Iteration          int            `json:"iteration,omitempty"`
	MaxIterations      int            `json:"max_iterations,omitempty"`
	TargetConfidence   int            `json:"target_confidence,omitempty"`
	OverallConfidence  int            `json:"overall_confidence,omitempty"`
	...
	ActiveQuestionID   string         `json:"active_question_id,omitempty"`
	ActiveQuestionText string         `json:"active_question_text,omitempty"`
	...
	OpenGaps           []string       `json:"open_gaps,omitempty"`
	Contradictions     []string       `json:"contradictions,omitempty"`
	...
	Novelty            noveltyTracker `json:"novelty,omitempty"`
	CoreQuestion    string   `json:"core_question,omitempty"`
	SuccessCriteria []string `json:"success_criteria,omitempty"`
}

type noveltyTracker struct {
	LastKeywords   map[string]bool
	ConsecutiveLow int
	Threshold      float64
}
```
`noveltyTracker` is diminishing-returns detection already implemented: a consecutive-low-novelty counter against a threshold is exactly the "detects diminishing returns" clause of LIVE-06. This is a `keep-current` case, not new logic to write.

**The real gap is visibility, not capability.** None of `ActiveQuestionText`, `OverallConfidence`, `Contradictions`, or `Novelty` is emitted as a live event anywhere — they are written to `oracleStateFile` on disk and only surfaced when `oracle status` is explicitly polled (`oracleStatusResult`, `cmd/oracle_loop.go:493-565`). LIVE-06's "visibly targets uncertainty... confidence and contradictions" requirement is satisfied by threading these existing fields through the new event bus (Current Event Infrastructure section above) as `oracle.round_started`/`oracle.confidence_changed`/`oracle.contradiction_found` events at the exact points `oracleStateFile` is already mutated during `runOracleLoop` — not by adding new state.

**Preset naming/value mismatch with Phase 200 (D-09).** `[VERIFIED: cmd/oracle_loop.go:57-64]`
```go
var oracleDepthLevels = map[string]oracleDepthConfig{
	"quick":      {5, 60, "Quick", "Fast overview, up to 5 iterations"},
	"balanced":   {15, 85, "Balanced", "Standard research, up to 15 iterations"},
	"standard":   {15, 85, "Balanced", "Standard research, up to 15 iterations"},
	"deep":       {30, 95, "Deep", "Deep dive, up to 30 iterations"},
	"exhaustive": {50, 99, "Exhaustive", "Marathon convergence, up to 50 iterations"},
	"marathon":   {50, 99, "Exhaustive", "Marathon convergence, up to 50 iterations"},
}
```
versus Phase 200's plan preset table `[VERIFIED: cmd/codex_plan.go:217-220]`:
```go
{ID: planningStagePresetFast, Label: "Fast", TargetConfidence: 80, PassCap: 4},
{ID: planningStagePresetBalanced, Label: "Balanced", TargetConfidence: 90, PassCap: 6},
{ID: planningStagePresetDeep, Label: "Deep", TargetConfidence: 95, PassCap: 8},
{ID: planningStagePresetExhaustive, Label: "Exhaustive", TargetConfidence: 99, PassCap: 12},
```
The cheapest tier is labelled `"quick"` in Oracle vs `"Fast"` in Plan, and every confidence target differs (Oracle 60/85/95/99 vs Plan 80/90/95/99) except Deep and Exhaustive, which already agree at 95/99. D-09 explicitly permits keeping Oracle's own numeric targets/iteration caps internally — "the existing confidence-target machinery may implement the presets internally" — so the fix is presentation-layer: rename the picker's label surface to the shared `Fast/Balanced/Deep/Exhaustive` vocabulary (mapping `quick→Fast`, `standard`/`balanced`→`Balanced`, `deep`→`Deep`, `marathon`/`exhaustive`→`Exhaustive`) and show each preset's *own* confidence target and round cap next to it, exactly as D-09 asks — not force Oracle's research-round economics to match planning-pass economics, which are legitimately different costs.

**Durable, honestly-labelled partial work already exists for Oracle (Phase 198.2).** `[VERIFIED: cmd/oracle_research_doc.go:47-335 function signatures; STATE.md Phase 198.2 bullet: "Oracle finish and stop paths share one file/register/promote body; useful partial research is labelled, strong findings carry origin labels, and empty runs write nothing."]` `oracleResearchPartialLabel`, `saveOracleResearchDocument`, and `runResearchList` already implement D-12's "useful notes, not verified" labelling contract for Oracle specifically. LIVE-07's job is (a) confirm this still holds after the event-bridge refactor, and (b) extend the *same discipline* to Swarm's unproven repair ideas and to D-11's status/history surfacing — not reinvent partial-work labelling from scratch.

## Architecture Patterns

### System Architecture Diagram

```text
                         ┌─────────────────────────────┐
                         │   Go runtime state mutators  │
                         │  (swarm wave dispatch, oracle │
                         │   iteration, build dispatch)  │
                         └───────────────┬───────────────┘
                                         │ emit typed event
                                         ▼
                         ┌─────────────────────────────┐
                         │   pkg/events.Bus (extended)  │
                         │  new topics: swarm.*, oracle.*│
                         │  existing: learning.*,       │
                         │  ceremony.*                  │
                         │  JSONL-persisted, replayable  │
                         └───────────────┬───────────────┘
                     ┌───────────────────┼───────────────────┐
                     ▼                   ▼                   ▼
            ┌────────────────┐  ┌────────────────┐  ┌────────────────┐
            │ watch (live)   │  │ status/history  │  │ transcript/test │
            │ dashboard +    │  │ (replay-backed  │  │ fixtures        │
            │ ticker strip   │  │  summary, D-03) │  │ (replay proof)  │
            └────────────────┘  └────────────────┘  └────────────────┘

Swarm episode path (already durable):
  runSwarmDestroy → [NEW: lens wave → hypothesis compare/rank card]
    → [NEW: checkpoint save (reuse work_repair.go) → fix wave
       → re-verify → pass: announce | fail: rollback + announce]
    → persistSwarmResultOutcome (existing) → swarmResultRecord
    → evaluateSwarmStrikeHistory (existing, unchanged) → 3-strike escalation

Oracle episode path (already durable):
  runOracleLoop → oracleStateFile mutated each iteration (existing)
    → [NEW: emit oracle.* events at each mutation point]
    → finalizeOracleLoop → saveOracleResearchDocument (existing, Phase 198.2)
    → runResearchList / oraclePromote (existing) → status/history surfacing (NEW)
```

### Recommended Project Structure
```
cmd/
├── event_types.go          # RETIRE or fold into pkg/events topic constants
├── event_stream.go         # RETIRE (dead code, zero callers)
├── event_bridge.go         # RETIRE (dead code, zero callers)
├── swarm_cmd.go            # ADD: lens comparison/ranking step between
│                           #   investigation wave and fix wave
├── swarm_lens.go           # NEW (suggested): hypothesis struct, compare/rank
├── swarm_repair_checkpoint.go  # NEW (suggested): thin adapter calling
│                           #   work_repair.go's save/restore primitives
├── oracle_loop.go          # ADD: event emission at existing mutation points
├── oracle_preset.go        # NEW (suggested): Fast/Balanced/Deep/Exhaustive
│                           #   label mapping over existing oracleDepthLevels
├── compatibility_cmds.go   # MODIFY: watchCmd branches live vs replay-backed
│                           #   vs idle, per D-01/D-03
└── status.go / history.go  # MODIFY: read swarm/oracle episode records (D-11)

pkg/events/
└── bus.go                  # ADD: new topic constants + typed payload structs
                             #   for swarm.*, oracle.*, watch.* domains
```

### Pattern 1: Typed event emission at existing mutation points, not a new state machine
**What:** Add `bus.Publish(ctx, "swarm.lens_started", payload, "swarm")`-style calls at the exact lines that already mutate `swarmWorkerExecution`/`oracleStateFile`, rather than building a parallel "live state" struct that could drift from the durable record.
**When to use:** Every place LIVE-01/CEC-05 requires a live signal.
**Example:**
```go
// Source: pkg/events/bus.go:47-50 (existing signature to call against)
func (b *Bus) Publish(ctx context.Context, topic string, payload json.RawMessage, source string) (*Event, error)
```
`[VERIFIED: pkg/events/bus.go:47]`

### Pattern 2: `watch` as a three-way branch, not a single idle path
**What:** `buildIdleWatchResult`'s unconditional idle branch (`cmd/compatibility_cmds.go:420-460`) becomes: (1) live episode running → subscribe to the event bus and render D-01's dashboard; (2) no live episode but replay history exists → D-03's replay-backed summary; (3) no history at all → today's literal idle card, unchanged.
**When to use:** `watchCmd` and the equivalent status/history surfacing.
**Anti-pattern to avoid:** Do not make the wrapper decide which of the three branches applies — the Go runtime must resolve this the same way `LifecycleFacts`/`LifecycleProjection` resolve every other "what state are we in" question today (established pattern throughout Phase 199).

### Pattern 3: Lens comparison as a first-class artifact between waves
**What:** Insert a structured step between the investigation wave and the fix wave that takes each lens's `swarmWorkerExecution`/response, builds a comparable hypothesis (claim, evidence, confidence), diffs them for contradictions, and ranks candidate repairs — rendered as "one card at the end" (D-06) before the fix wave proceeds.
**When to use:** `runSwarmDestroy`'s gap between `investigationRuns` and `fixPlans` (`cmd/swarm_cmd.go:307-317`).
**Reuse:** `oracleWorkerResponse{Confidence, Findings, Contradictions, Recommendation}` (`cmd/oracle_loop.go:418-428`) is an already-designed, already-tested shape for exactly this "evidence + confidence + contradictions" comparison unit — reuse its shape (or the type itself) for Swarm's per-lens hypothesis rather than inventing a fourth parallel evidence type (Swarm's `swarmWorkerResponse` today has no `Confidence` or structured evidence field at all — `[VERIFIED: cmd/swarm_cmd.go:44-57 swarmWorkerResponse type definition has no confidence field]`).

### Anti-Patterns to Avoid
- **A second checkpoint primitive for Swarm:** CONTEXT.md is explicit — "not a third mechanism." Reuse `work_repair.go`'s save/restore functions.
- **A second event bus:** extending `pkg/events.Bus` (already used by three live subsystems) beats building a new bus that fragments existing callers.
- **A fourth parallel evidence/result envelope type:** Phase 201's own synthesis calls this out by name as "a structural anti-pattern this repository has already paid for three times" — reuse `oracleWorkerResponse`'s shape for Swarm's hypothesis type rather than inventing a new one.
- **Forcing Oracle's iteration math to match Plan's:** D-09 explicitly allows different underlying numbers; only the four preset *names* need to be shared.
- **Rebuilding Swarm's external-manifest trust boundary work:** CR-01/CR-02 are already fixed (Phase 198.3); do not re-derive `validateDurableSwarmID`/issuance logic.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|--------------|-----|
| Repair checkpoint/rollback for Swarm | A new Swarm-specific save/restore mechanism | `cmd/work_repair.go`'s `saveRepairCheckpoint`/`restoreRepairCheckpoint`/`emitRepairCheckpointSaved`/`emitRepairCheckpointRestored` | Already built, already proven in Phase 201, and CONTEXT.md explicitly forbids a third checkpoint concept. |
| Live event transport | A new pub/sub bus or a revival of the dead `cmd/event_types.go` NDJSON stream | `pkg/events.Bus` (topic + payload, JSONL-persisted, already used by 3 subsystems) | Extending a wired bus keeps `memory_feed.go`/`status.go`/`oracle_promote.go` on one source of truth; reviving dead code recreates the exact two-buses problem LIVE-01 exists to resolve. |
| Web/external-research Swarm lens | A new HTTP/search-integration mechanism inside `swarm_cmd.go` | Oracle's existing research worker dispatch (`oracle_loop.go`'s evidence types: `documentation \| official \| github \| blog \| forum \| academic`) | Oracle already has this capability fully built and tested; Swarm's fourth lens should invoke the same shape rather than duplicate web-research plumbing. |
| Three-strike escalation | Any new strike-counting logic | `cmd/swarm_strikes.go`'s `evaluateSwarmStrikeHistory`/`ensureSwarmEscalationForHistory` | Ledger explicitly marks this "already proven" — CAP-046. Touching it is regression risk, not delivery. |
| Partial-work honesty labelling | A new "unverified" flag system | `oracleResearchPartialLabel` pattern (`cmd/oracle_research_doc.go`) | Phase 198.2 already solved this exact problem for Oracle; extend the same discipline to Swarm rather than inventing new vocabulary. |
| Depth/confidence presets for Oracle | A new preset system | Existing `oracleDepthLevels` map, relabelled to match Phase 200's `Fast/Balanced/Deep/Exhaustive` names | The underlying iteration math (5/15/30/50) already matches Classic evidence; only the label layer needs to change. |

**Key insight:** every subsystem this phase touches already has a working, tested, durable core (episode storage, checkpoint/rollback, confidence tracking, partial-work labelling). The actual engineering work is (1) building the one missing piece — a typed live event bridge — and (2) wiring existing mechanisms to it and to each other, not building four new features from a blank page.

## Common Pitfalls

### Pitfall 1: Treating "four lenses" as four prompts instead of four evidence-producing, comparable outputs
**What goes wrong:** Selecting four castes for wave 1 (as today's code already does) and calling that LIVE-03 done — the roadmap and CONTEXT.md are explicit that "four cosmetically different prompts" is the failure mode being guarded against.
**Why it happens:** The caste-selection machinery already exists and looks like most of the work; the missing 20% (structured hypothesis comparison + ranking) is invisible until you look for the comparison step and find it isn't there.
**How to avoid:** The acceptance bar is a **ranking artifact** — one card, at the end, showing compared hypotheses and why one was selected — not merely four workers running.
**Warning signs:** If `renderSwarmFindingSummary` (free-text concatenation) is still the only thing feeding the fix wave, the comparison step has not actually been built.

### Pitfall 2: Applying the repair without a checkpoint because "it's just Swarm"
**What goes wrong:** `runSwarmDestroy` today applies the fix wave directly with no save-state step; a plan that reuses today's flow unchanged silently ships LIVE-04 as unmet.
**Why it happens:** Swarm's flow already "looks" like build's flow (waves, dispatch, verification), so it's easy to assume the safety net is already there.
**How to avoid:** Explicitly verify a checkpoint call happens before the fix wave and a restore call happens on verification failure, both announced (D-05's literal requirement).
**Warning signs:** No call to `saveRepairCheckpoint`-equivalent anywhere in the modified `runSwarmDestroy`.

### Pitfall 3: Building a new dashboard state model instead of projecting the event stream
**What goes wrong:** Writing a separate "current live state" struct that the dashboard reads, updated by ad-hoc setters scattered through swarm/oracle/build code — this is exactly the dead-code pattern `cmd/event_types.go` already demonstrates (a nice-looking schema nobody actually calls).
**Why it happens:** It's tempting to build the renderer first and "wire events later."
**How to avoid:** Build the event emission first, at the real mutation sites, and make the dashboard a pure projection of replayed events — this is also what makes D-03's replay-backed idle summary possible for free (same projection, historical data).
**Warning signs:** A dashboard that renders correctly in a demo but shows nothing when driven purely from replayed JSONL events — that's the dead-code pattern recurring.

### Pitfall 4: Reconciling Oracle and Plan presets by changing Oracle's confidence targets
**What goes wrong:** "Fixing" the 60/85/95/99 vs 80/90/95/99 mismatch by rewriting `oracleDepthLevels`'s numbers to match Plan's, which changes Oracle's actual research behavior (a real regression) to solve what is only a display inconsistency.
**Why it happens:** The mismatch looks like a bug because the numbers don't match; but D-09 only requires the same four *names* be shown, with each preset's *own* target/cap displayed.
**How to avoid:** Add a label-mapping layer; leave `oracleDepthLevels`'s actual confidence/iteration values untouched unless a specific product reason (not merely cross-system tidiness) requires changing them.

### Pitfall 5: Re-litigating the already-fixed swarm trust boundary
**What goes wrong:** A plan spends effort re-verifying/rebuilding `validateDurableSwarmID`/issuance-digest validation because the Dreams review's CR-01/CR-02 findings read as unresolved.
**Why it happens:** The review document (dated 2026-09-01) predates the Phase 198.3 fix; reading it without cross-checking STATE.md gives a false impression the vulnerability is still open.
**How to avoid:** Trust STATE.md's Phase 198.3 entry and the actual `cmd/swarm_issuance.go` code (already read this session) over the review document's Part II findings table — the review's own text elsewhere (Part I) already reflects the fix status; only the raw finding table is dated.

## Code Examples

### Watch's current idle-only branch (to be made conditional)
```go
// Source: cmd/compatibility_cmds.go:59-70
var watchCmd = &cobra.Command{
	Use:         "watch",
	Short:       "Show the honest idle watch fallback",
	Args:        cobra.NoArgs,
	Annotations: map[string]string{"aether.io/read-only": "true"},
	RunE: func(cmd *cobra.Command, args []string) error {
		_ = cmd // --once/--interval remain accepted compatibility flags.
		result := buildIdleWatchResult(resolveAetherRoot(), store, time.Now().UTC())
		outputWorkflow(result, renderIdleWatchVisual(result))
		return nil
	},
}
```

### Swarm's fixed caste-to-lens task descriptions
```go
// Source: cmd/swarm_cmd.go:1071-1093
func swarmTaskForCaste(caste string) string {
	switch caste {
	case "tracker":
		return "Reproduce the issue, trace the failure path, and identify the most likely root cause."
	case "scout":
		return "Search the repo for the most relevant files, patterns, tests, and documentation tied to the reported bug."
	case "archaeologist":
		return "Inspect git history and prior fixes around the bug area to identify historical context, fragile zones, and regressions."
	case "gatekeeper":
		return "Inspect security, auth, permission, dependency, and release-integrity risks tied to the reported bug."
	case "medic":
		return "Diagnose colony/runtime health risks and recovery constraints tied to the reported bug."
	...
```

### Oracle's already-durable partial-research labelling entry points
```go
// Source: cmd/oracle_research_doc.go — function signatures (Phase 198.2)
func oracleResearchPartialLabel(state oracleStateFile) string
func saveOracleResearchDocument(paths oraclePaths, state oracleStateFile, plan oraclePlanFile, name string) (string, error)
func runResearchList(root string) (map[string]interface{}, error)
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|---------------|--------|
| Classic four-lens Swarm as a shell-script control loop over `tmux` | Go-owned caste dispatch (`buildSwarmPlansForWave`) with a linear pipeline, no comparison step | Removed 2026-04-05 (`0063be8b`), partially restored since | The caste roster survived; the comparison/ranking behavior did not. |
| Classic Oracle wizard over `tmux` with visible round-by-round research | Go `oracle_loop.go` (3,945 lines) with the same 5/15/30/50 iteration shape but backgrounded, poll-only visibility | April 2026 collapse | Confidence/contradiction/novelty tracking is materially equal or better; only live visibility regressed. |
| Classic `watch` four-pane `tmux` cockpit | Compatibility `watch` — permanently idle | Deliberately retired ("do not restore tmux/second-terminal watch"), now reopened by owner steer (D-01/D-04) | This phase is the first to build a real replacement. |
| Swarm external finalizer trusted caller-supplied manifest/ID (CR-01/CR-02) | Runtime-issued, digest-bound, replay-protected manifest issuance (`cmd/swarm_issuance.go`) | Phase 198.3 | Safe substrate for this phase's new lens/ranking work — no re-verification needed. |

**Deprecated/outdated:**
- `cmd/event_types.go`/`event_stream.go`/`event_bridge.go`'s NDJSON stub stream: no production caller; superseded in practice by `pkg/events.Bus`, which should absorb its intended purpose rather than the stub being revived.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|----------------|
| A1 | The suggested fourth Swarm lens (external/web research) should reuse Oracle's existing research-dispatch shape rather than being a wholly new mechanism. | Don't Hand-Roll; Pattern 3 | If the owner/planner decides the fourth lens must be something else entirely (e.g., a specialist code-quality lens), this recommendation doesn't apply — but the underlying gap (no fourth wave-1 caste exists today) still holds regardless of which lens is chosen. |
| A2 | `pkg/events.Bus` (not a rewritten `cmd/event_types.go`) is the correct unification target for LIVE-01. | Current Event Infrastructure; Pattern 1 | This is architecture guidance, not a locked decision — CONTEXT.md explicitly leaves "whether the existing `cmd/` event bus and `pkg/events` unify or bridge" to Claude's discretion. If a planner has evidence favoring a bridge instead of a merge, that is within the granted discretion. |
| A3 | No TUI framework dependency is needed; the live dashboard should redraw via the existing interval/reprint pattern. | Package Legitimacy Audit | If D-01's "dashboard that updates in place" is judged to require true interactive TUI behavior (mouse/keyboard nav, split panes), a TUI dependency decision becomes owner-facing and this assumption would need revisiting. |

## Open Questions

1. **Exact typed event schema and versioning strategy (LIVE-01).**
   - What we know: `pkg/events.Event` already has a generic, extensible shape (topic + raw payload); Classic's proposed vocabulary (`worker_started/progress/completed`, `plan_confidence_changed`, etc.) is a reasonable starting point per the Dreams review.
   - What's unclear: the exact versioning scheme (a `schema_version` field on payloads? a topic-name version suffix?) and precisely which fields are mandatory vs optional per event kind.
   - Recommendation: this is explicitly Claude's Discretion per CONTEXT.md — the mandatory `202-CLASSIC-SYNTHESIS.md` should record the chosen schema and its rationale (this is exactly the kind of decision SYNTH-04/SYNTH-07 require to be justified, not merely implemented).

2. **How the Swarm lens hypothesis type relates to `oracleWorkerResponse`.**
   - What we know: `oracleWorkerResponse{Confidence, Findings, Contradictions, Recommendation}` is an already-tested shape for exactly the fields a Swarm hypothesis needs.
   - What's unclear: whether to literally reuse the Go type (creating a cross-package/cross-domain dependency between Swarm and Oracle code) or define a sibling type with the same shape.
   - Recommendation: a sibling type (same field shape, Swarm-owned) is likely safer — it avoids coupling Swarm's evolution to Oracle's, matching the Phase 201 synthesis precedent of "a small sibling type... never extending it" for comparable design questions.

## Environment Availability

Skipped — this phase has no external service/CLI/runtime dependencies beyond the existing Go toolchain already verified in every prior phase of this milestone.

## Validation Architecture

Skipped — `.planning/config.json` sets `workflow.nyquist_validation: false` explicitly `[VERIFIED: .planning/config.json]`.

## Security Domain

Skipped — `.planning/config.json` sets `workflow.security_enforcement: false` explicitly `[VERIFIED: .planning/config.json]`. Note for the planner: this phase does touch an area with historical trust-boundary findings (Swarm's external finalizer, CR-01/CR-02) — those are independently confirmed fixed (Phase 198.3) and do not need re-verification, but any *new* external-facing surface this phase adds (e.g., if the new lens/checkpoint code introduces a new external completion path) should be checked against the same `validateDurableSwarmID`/issuance-digest pattern already proven in `cmd/swarm_issuance.go`, even with security_enforcement off, simply because it is directly adjacent proven-safe code.

## Sources

### Primary (HIGH confidence — read directly this session)
- `cmd/compatibility_cmds.go` — `watchCmd`, `buildIdleWatchResult`
- `cmd/swarm_cmd.go`, `cmd/swarm.go`, `cmd/swarm_issuance.go`, `cmd/swarm_strikes.go`, `cmd/swarm_display.go` — full function/type inventory, key bodies read verbatim
- `cmd/oracle_loop.go`, `cmd/oracle_research_doc.go`, `cmd/oracle_promote.go` — full struct/preset definitions, key function inventory
- `cmd/codex_plan.go` — Phase 200 preset table
- `cmd/work_repair.go` — checkpoint/repair primitives
- `pkg/events/event.go`, `pkg/events/bus.go` — event bus shape
- `cmd/event_types.go`, `cmd/event_stream.go`, `cmd/event_bridge.go` — dead-code confirmation via grep for callers
- `cmd/status.go`, `cmd/spawn.go` — live callers of `pkg/events.Bus`
- `cmd/codex_visuals.go` — caste identity/stage marker functions (grep-confirmed existence)
- `go.mod` — dependency inventory
- `.planning/config.json` — workflow flags
- `.planning/research/v1.28-classic-capability-ledger.md` — CAP-021, CAP-044..048, CAP-063, CAP-072 rows routed to Phase 202
- `.planning/research/v1.28-classic-synthesis-template.md` — mandatory synthesis structure
- `.planning/phases/201-queen-led-work-cycle/201-CLASSIC-SYNTHESIS.md` — precedent for synthesis depth, reusable-primitive reasoning, and "sibling type, not extension" pattern
- `.planning/STATE.md` — Phase 198.2/198.3 delivery confirmations

### Secondary (MEDIUM confidence)
- `.aether/dreams/2026-09-01-comprehensive-aether-colony-review.md` — Classic mechanism reconstruction, proposed event vocabulary, CR-01/CR-02 findings (cross-checked against current code and STATE.md; the finding table itself predates the Phase 198.3 fix, but the mechanism/vocabulary content is used as historical evidence input, not as a claim about current state)
- `.planning/ROADMAP.md`, `.planning/REQUIREMENTS.md` — phase boundary, requirement text, success criteria (owner-approved documents, read directly)

### Tertiary (LOW confidence)
- None — no WebSearch was needed for this phase; every claim traces to a file read this session or an owner-approved planning document.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — no new packages, verified against `go.mod` directly.
- Current-architecture findings (event systems, watch, swarm, oracle): HIGH — every claim cites a specific file and line range read this session.
- Recommended synthesis (lens comparison shape, event schema, preset reconciliation): MEDIUM — these are informed recommendations for the mandatory `202-CLASSIC-SYNTHESIS.md` to formalize, not restored facts; the synthesis document itself (SYNTH-04/SYNTH-07 gate) still needs to be authored with its own comparative matrix before implementation plans are approved, per this milestone's established pattern (Phases 199-201 each did this).

**Research date:** 2026-09-11
**Valid until:** Re-verify current-architecture claims if `cmd/swarm_cmd.go`, `cmd/oracle_loop.go`, or `pkg/events/bus.go` change substantially before planning begins (30-day estimate for a fast-moving milestone branch).
