# Phase 162: Switch On Learning - Context

**Gathered:** 2026-08-04
**Status:** Ready for planning

<domain>
## Phase Boundary

Invoke the consolidation pipeline that already exists. `pkg/memory/pipeline.go`
wires Observe → Promote → Queen → Consolidate and is constructed in exactly two
places — `consolidation-phase-end` and `consolidation-seal`
(`cmd/graph_consolidation_cmds.go:217` and `:315`) — and neither subcommand has
ever been invoked by any wrapper, playbook, or Go call site. This phase gives
them their callers: phase-end consolidation runs when `/ant-continue` advances a
phase, and the full eight-ant pass + decay + archive runs at `/ant-seal`. It
also reconciles the two competing learning systems (`pkg/learn` runs live on
every continue via `captureContinueLearning`; `pkg/memory` is the documented,
dormant one), flips the Hive Brain from accidental-off to deliberate-on, and
proves the loop end-to-end with a before/after memory comparison (LEARN-01..05).

This phase invokes and reconciles existing code; it does not build a learning
system.

</domain>

<decisions>
## Implementation Decisions

### Hive Brain default (LEARN-04 — user-decided)
- **D-01: On, full.** The default becomes read + promote: colonies
  automatically read cross-colony wisdom into worker context AND promote
  high-confidence instincts at seal. Matches the milestone's
  everything-networked-together goal; promotion is already non-blocking and
  capped (200 entries, LRU), so blast radius is small.
- **D-02: One switch — the env var.** When `AETHER_HIVE_POLICY` is unset, the
  policy is `promote` (full). `=off` and `=read` remain as explicit overrides.
  The consent-file mechanism (`hiveRetrievalOptedIn()`, `cmd/hive.go:303`) is
  retired entirely — one control surface, honestly documented, with no second
  hidden gate that silently vetoes the first.
- **D-03: Docs become true, not softened.** With the default flipped, CLAUDE.md
  and the hive docs' descriptions of automatic hive flow become accurate;
  update any text describing the opt-in mechanics to match the new single
  switch. Per REQUIREMENTS.md's Non-Goals, hive *trust redesign* stays shelved
  — this is a default change plus honest documentation, nothing more.

### Where consolidation runs & failure behavior (LEARN-01/02 — user-decided)
- **D-04: Runtime-owned, at phase end.** The Go continue-finalize path invokes
  phase-end consolidation automatically when a phase actually completes and
  advances — not on every mid-phase continue, and not as a wrapper-instructed
  step. No new wrapper protocol (Phase 165 owns continue.md's structure), and
  an LLM can't skip a step that lives in the runtime — the exact failure mode
  this milestone exists to kill. Seal consolidation is likewise invoked by the
  runtime seal path.
- **D-05: Warn loudly, never block.** A consolidation failure never stops the
  phase advance or the seal. The output states unmissably: "phase advanced
  WITHOUT consolidation — <reason>". Consistent with Phase 160's
  gate-vs-enrichment classification and the existing non-blocking hive
  promotion rule. Learning is enrichment, not a gate.
- **D-06: Full colony ceremony for the report.** Phase end: a caste-styled
  learning beat (e.g. 🧠 "3 observations → 2 learnings → 1 new instinct
  (confidence 0.75)"). Seal: each of the eight curation ants announces with
  emoji + name + what it did, followed by decay counts and the archive/report
  artifact path. Consistent with Phase 164's ceremony precedent (D-09 there)
  and the living-colony direction.
- **D-07: The learning beat always appears — even at zero.** A phase end with
  nothing to consolidate still prints "colony observed nothing new this
  phase". Silence and not-wired must be impossible to confuse.

### The before/after proof (LEARN-05 — user-decided)
- **D-08: Two-layer proof.**
  - **Layer 1 (permanent):** a deterministic automated test assembles the same
    worker brief twice — colony memory populated vs wiped — and asserts the
    instinct/wisdom text is present in one and absent in the other. Runs in CI
    forever; fails if memory injection silently breaks (Definition of Done).
  - **Layer 2 (recorded once):** during phase verification, one real worker
    run each way, with the differing outputs saved as a before/after artifact
    in the phase directory — the requirement's "recorded, not assumed"
    comparison.
  This reconciles Phase 163 D-08 (no staged benchmarks) with LEARN-05: the
  permanent layer is prompt-level and deterministic (no model calls in CI);
  the model-call layer happens exactly once as a recorded exhibit. A
  repeatable `learning-proof` harness is NOT built here — that overlaps
  Phase 171 (Prove It).

### Which system is authoritative (LEARN-03 — delegated to Claude, locked)
- **D-09: `pkg/memory` is authoritative; `pkg/learn` is explicitly
  subordinate as the live capture layer.** Reasoning: `pkg/memory` is the
  pipeline the documentation describes (the Wisdom Pipeline in CLAUDE.md), the
  one with consolidation, decay, and the eight curation ants — the destination
  this phase switches on. `pkg/learn` already runs live on every continue and
  retiring it would delete the only working capture path; it stays, defined as
  the capture/trust-scoring front-end feeding the authoritative pipeline (the
  `LearningValidator` bridge in `pkg/memory/pipeline.go` already anticipates
  this relationship). If research finds a stage genuinely implemented in both,
  `pkg/memory`'s implementation wins and the duplicate in `pkg/learn` is
  retired.
- **D-10: The decision is written down in the repo,** not only implemented —
  a decision record stating authoritative/subordinate roles and the fate of
  any duplicated stage (LEARN-03 and LEARN-04 are decision-shaped
  requirements). Exact location is Claude's discretion (an ADR-style doc under
  `.aether/docs/` is the natural home); CLAUDE.md and
  `structural-learning-stack.md` must agree with it afterward.

### Claude's Discretion
- Exact ceremony wording, emoji, and line formats (bound by D-06/D-07 and the
  caste identity system in `cmd/codex_visuals.go`).
- Seal archive/report artifact location and contents.
- Mechanics of detecting "phase actually advanced" inside continue-finalize
  for D-04, and where in the seal path the eight-ant pass fires.
- The decision-record location and format for D-10.
- How the Layer-1 test populates/wipes fixture memory, and where the Layer-2
  before/after artifact lives in the phase directory.
- Whether `consolidation-phase-end`/`consolidation-seal` remain user-invocable
  standalone subcommands after gaining runtime callers (they should — they are
  the inspection/manual path).

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Requirements and roadmap
- `.planning/REQUIREMENTS.md` §LEARN-01..05 (lines ~58-64) — requirement text,
  including the file:line cites for both learning systems and the hive gates
- `.planning/ROADMAP.md` §Phase 162 — goal, dependency (160), five success
  criteria (two decision-shaped)

### The dormant pipeline this phase switches on
- `pkg/memory/pipeline.go` — the Observe → Promote → Queen → Consolidate
  wiring, `PipelineConfig`, the `LearningValidator` bridge to `pkg/learn`
- `cmd/graph_consolidation_cmds.go` (`:217`, `:315`) — the only two
  construction sites: `consolidation-phase-end` and `consolidation-seal`;
  note the v1.25 dry-run fix pinned by
  `TestConsolidationPhaseEndDryRunDoesNotMutate` /
  `TestConsolidationSealDryRunDoesNotMutate` — dry-run must never mutate

### The live system being subordinated
- `cmd/codex_continue_finalize.go` (~:482-555) — `captureContinueLearning`,
  the `pkg/learn` capture that runs on every continue today
- `pkg/learn/learn.go`, `pkg/learn/curator.go`, `pkg/learn/hive_store.go` —
  Entry/Evidence types, trust scoring, hive storage

### Hive gates being reconciled (D-01/D-02)
- `cmd/hive_policy.go` — `AETHER_HIVE_POLICY` env var, `hivePolicyOff`
  default, off/read/promote values
- `cmd/hive.go:303` — `hiveRetrievalOptedIn()` consent-file gate (retired by
  D-02)

### Docs that must end up true (D-03/D-10)
- `CLAUDE.md` §Wisdom Pipeline, §Hive Brain, §Structural Learning Stack —
  currently describe the dormant pipeline as running and hive flow as
  automatic; both claims become true, and the lifecycle-integration caveat
  ("no lifecycle command invokes either one yet (Phase 162 wires this)")
  comes out
- `.aether/docs/structural-learning-stack.md` — same reconciliation
- `.planning/ROADMAP.md` §Phase 160 success criterion 4 — records that the
  consolidation-runs doc claims were corrected in 160 to become true only
  when this phase ships

### Prior phase constraints
- `.planning/phases/165-core-lifecycle-commands/165-CONTEXT.md` — Phase 165 is
  sole structural owner of build.md/continue.md; 162's wiring stays
  runtime-side (D-04); whatever ceremony 162 adds must be runtime-rendered,
  never wrapper-hand-rendered
- `.planning/phases/164-research-feeds-planning/164-CONTEXT.md` §D-09 — the
  ceremony precedent D-06 follows (caste emoji + ANSI color + deterministic
  name)
- `.planning/phases/163-context-reaches-workers/163-CONTEXT.md` §D-08 — the
  no-staged-benchmark precedent D-08 reconciles with

### Presentation surface (invoke, never re-render in wrappers)
- `cmd/codex_visuals.go` — `casteIdentity()`, `casteEmoji()`, caste color maps
  for the D-06 ceremony beats

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `consolidation-phase-end` / `consolidation-seal` subcommands: fully working
  CLI entry points — this phase adds runtime callers, not new commands
- `pkg/memory` services (ObservationService, PromoteService, QueenService,
  ConsolidationService): already wired together by `NewPipeline`
- The eight curation ants (orchestrator, archivist, critic, herald, janitor,
  librarian, nurse, scribe, sentinel): already orchestrated by the seal-side
  consolidation; D-06 makes their pass visible
- Hive machinery (`hive-read`, `hive-promote`, `hive-store`, 200-cap LRU,
  multi-repo confidence boosting): all built; D-01 flips the policy default
- Colony ceremony renderers (`cmd/ceremony_cmd.go`, `cmd/codex_visuals.go`):
  the identity/format system the learning beats should use

### Established Patterns
- Runtime owns truth, wrappers present: consolidation invocation and ceremony
  rendering live in Go; wrappers at most narrate around runtime output
- Definition of Done: every wiring claim needs a command/test that fails when
  the wiring is absent — e.g. a test that fails if continue-finalize stops
  invoking consolidation, and the D-08 Layer-1 injection test
- Non-blocking enrichment: hive promotion at seal is already NON-BLOCKING by
  documented design; D-05 extends the same classification to consolidation
- Dry-run purity: pinned by the two v1.25 dry-run tests — new callers must
  use the real (non-dry-run) path deliberately

### Integration Points
- `cmd/codex_continue_finalize.go` — where phase-advance detection lives and
  where the phase-end consolidation call lands (D-04), adjacent to the
  existing `captureContinueLearning` call
- The seal path (`cmd/seal.go` / codex seal flow) — where the eight-ant pass,
  decay, and archive fire
- Worker context assembly (`cmd/colony_prime_context.go`, brief renderers) —
  where hive wisdom injection becomes default-on (D-01) and where the D-08
  Layer-1 test asserts presence/absence
- Phase 170's RECLAIM-09 (`queen-seed-from-hive`) explicitly waits on this
  phase's hive decision — D-01 (on, full) is the input that phase needs

</code_context>

<specifics>
## Specific Ideas

- The user's standing north star (from Phase 164 discussion, applies here):
  "it's just important that all of that stuff is networked together that's
  fundamentally the thing" — this phase is the learning half of that
  networking.
- The acceptance story: finish a phase, watch a 🧠 learning beat report real
  promotions (or honestly report zero); seal a colony, watch eight named ants
  do their pass; open a second repo and see wisdom from the first arrive
  without setting any env var.

</specifics>

<deferred>
## Deferred Ideas

- **Repeatable `learning-proof` harness** (run a real worker twice on demand)
  — rejected for this phase (D-08); Phase 171 (Prove It) territory.
- **`queen-seed-from-hive` reconnection** — Phase 170 (RECLAIM-09), which
  consumes this phase's hive decision.
- **Hive trust redesign** — explicitly shelved at the milestone level
  (REQUIREMENTS.md Non-Goals); D-01/D-02 change the default and controls only.

### Reviewed Todos (not folded)
- "continue-finalize does not count --reconcile-task as recorded
  reconciliation for the implementation_evidence gate" — keyword match only;
  a continue-finalize gate bug unrelated to learning. Already reviewed and
  deferred by Phases 163.2, 164, and 165. Stays in backlog.
- "ts-host preflight timeout hardcoded, not configurable, runs in repo cwd" —
  keyword match only; Phase 163.2's executed scope. Stays in backlog.

</deferred>

---

*Phase: 162-Switch On Learning*
*Context gathered: 2026-08-04*
