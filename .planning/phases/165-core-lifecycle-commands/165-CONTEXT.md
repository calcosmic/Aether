# Phase 165: Core Lifecycle Commands - Context

**Gathered:** 2026-08-02
**Status:** Ready for planning

<domain>
## Phase Boundary

Rewrite the four core lifecycle wrappers (`.claude/commands/ant/init.md`,
`plan.md`, `build.md`, `continue.md`, mirrored in `.opencode/commands/ant/`)
to carry engineering method — stage purpose, files-to-read guidance, spawn
choreography, stop conditions — AND restore the rich Classic colony ceremony,
instead of protocol instructions whose primary job is parsing
`result.manifest.dispatch_manifest`. This phase is the sole structural owner
of `build.md` this milestone (Phase 160 made narrow call-argument fixes
before; Phase 168 appends a visual-guidance trailer after). Wrapper text only
— no Go/TS runtime behavior changes.

</domain>

<decisions>
## Implementation Decisions

The user's single directive: **"I just wanted to have that rich ceremony we
used to have."** All four gray areas were delegated to Claude. Decisions
below are locked; downstream agents treat them as chosen, not open.

### The governing insight (from v5.4.0 archaeology)
- **D-00: The Classic ceremony did not live in the old wrappers.** At v5.4.0,
  `build.md` (65 lines) and `continue.md` (60 lines) were thin pointers that
  Read-loaded ~4,400 lines of playbooks. The ceremony text lives in
  `.aether/docs/command-playbooks/*.md`, which still exist unchanged and are
  now reference-only. The rewrite MINES that text and INLINES what it
  restores. The playbook Read-loader is forbidden by
  `cmd/build_wrapper_ceremony_test.go` and `cmd/platform_doc_hygiene_test.go`
  and must never return.

### Voice & audience (delegated — locked)
- **D-01: Queen voice in the Classic register, anchored to method.** Each
  stage opens with one colony-framing beat, then engineering substance
  (Purpose / Reads / Spawns / Stop conditions). The signature Classic idiom
  to restore: rule-with-reason-in-colony-terms at the moment the rule fires
  (e.g. "🐜 Why this matters: Builders verify their own work = confirmation
  bias; independent Watchers catch bugs builders miss"). Persona is the
  presentation layer; method is the content. Success criterion 3 (a user can
  describe what each stage does without opening Go source) is the acid test.
- **D-02: Restore these specific Classic wrapper beats** (verbatim exemplars
  in the archaeology report; playbook line refs included there):
  - "You are the **Queen**. You DIRECTLY spawn multiple workers" opener
    with the no-Prime-Worker rule stated as colony law.
  - 🐜 "The colony requires actual parallelism" / "Why this matters" reason
    blocks at spawn and verification stages.
  - 👑 intention-setting beat in init ("Queen has set the colony's
    intention: '{approved_intent}'").
  - The Next Up closing idiom and phase framing ("Phase N of M — name, one-line
    purpose") — narrated by the wrapper, rendered by runtime commands where
    they exist.
  - Classic separator conventions in wrapper-authored narration: stage beats
    may use the `━━━ ... ━━━` marker style, but NEVER hand-render what a
    runtime ceremony command already renders (spawn plans, wave starts,
    worker completions, next-up blocks come from `aether ceremony ...` /
    `aether pheromone-display` / `aether status`).

### Where the JSON protocol goes (delegated — locked)
- **D-03: Envelope mechanics leave the wrapper prose.** The wrapper's spawn
  stage says, as method: fetch the manifest (`aether host build --dry-run N`),
  spawn each worker with the runtime-composed brief verbatim, respect
  `execution_plan` waves, print runtime ceremony output as each result
  arrives. Field-by-field parsing instructions, envelope shape documentation,
  and temp-file bookkeeping move to `.aether/docs/wrapper-host-contract.md`
  (already the named contract doc), referenced once per wrapper. Success
  criterion 2 (grep for parse-the-envelope-as-primary-job returns zero) is
  the test. No runtime behavior changes — this phase moves text, not code.
- **D-04: What the wrapper may still say about the handshake:** one line per
  mechanical step, subordinated under the stage's method ("the manifest is
  the colony's marching order — fetch it, then spawn exactly what it names").
  Nothing that reads as "parse JSON field X into Y".

### Stage structure & template (delegated — locked)
- **D-05: One uniform stage skeleton across all four wrappers:** stage name
  (aligned with the runtime's `── Stage Name ──` markers: Context, Tasks,
  Dispatch, Verification, Housekeeping, Next Phase) → colony beat → Purpose →
  Reads (specific files/commands) → Spawn choreography (if any) → Stop
  conditions. init.md keeps its own stage names but the same skeleton.
- **D-06: Restore the three Classic method assets archaeology marked as
  unambiguously safe wrapper content:**
  - `<success_criteria>` / `<failure_modes>` / `<read_only>` blocks per
    wrapper (Classic had them in init and build-prep; all four get them).
  - The named cross-stage state-carry contract (the v5.4.0 "Required
    Cross-Stage State" list: phase_id, depth, prompt_section, wave_results,
    verification_status, next_action — updated to current names).
  - Termination-condition prose (stop conditions summarized per command:
    confidence target reached, stall detection, iteration caps, escape
    hatches — matching what the runtime actually enforces today; wrappers
    describe, never re-implement).
- **D-07: Spawn choreography rules stay wrapper-owned and explicit:** all
  wave-1 workers in a single message via multiple Task calls, wait for wave
  completion before next wave, never run_in_background, never invent castes
  or names — use the manifest.

### build.md ownership handshake (delegated — locked)
- **D-08: Both mechanisms, pinned by test.** An HTML comment header in
  build.md records what Phase 160 fixed (broken call arguments, merged
  first) and reserves a named trailer marker (e.g.
  `<!-- PHASE-168: visual-guidance trailer appends below this line -->`) for
  Phase 168. The rewrite commit message states the ownership chain. A test
  greps for both the header record and the reserved marker so the merge
  order is traceable per success criterion 5.

### Ceremony Definition of Done (Claude-added, from archaeology escalation)
- **D-09: Ship tests that fail when the ceremony or method is absent.**
  Ceremony has been "restored" seven times since v5.4.0 and evaporated each
  time. Per the project's Definition of Done, Phase 165 extends the existing
  wrapper test pattern (`build_wrapper_ceremony_test.go`,
  `continue_wrapper_ceremony_test.go`, `plan_wrapper_ceremony_test.go`,
  `platform_doc_hygiene_test.go`) with assertions that (a) each wrapper
  carries the required stage skeleton sections, (b) required narration beats
  are present (prefer proportion/invariant assertions over named-section
  greps), (c) every already-forbidden regression stays forbidden, and (d)
  .claude/.opencode heading parity holds (TestPlanWrapperCardsParity style).
- **D-10: Regression fence — the eleven "do NOT bring back" items are hard
  constraints:** playbook Read-loader (CRITICAL), wrapper state mutation of
  COLONY_STATE.json/pheromones/constraints (CRITICAL), plan.md watch-file/tmux
  writes, caste legend or speculative caste naming in prose, context-clear
  ceremony in continue.md, gate threshold arithmetic in markdown, synthetic
  build/plan forcing, verbatim user transcripts in prompt files, wrapper-driven
  git stash/commit, `.aether/aether-utils.sh` or LiteLLM-proxy assumptions,
  and the retired four-value `colony_depth` vocabulary. Full evidence and
  commit refs in the archaeology report (see canonical refs).

### Platform parity
- **D-11: .claude and .opencode rewritten together in the same plan(s),**
  parity pinned by ordered-heading-set tests. Codex is runtime-native — no
  wrapper work. `.aether/commands/{init,plan,build,continue}.yaml` guardrail
  entries update in lockstep with wrapper guardrails.

### Claude's Discretion
- Exact wording of colony beats and narration (mine the playbooks freely).
- Which runtime ceremony commands each wrapper invokes at which stage,
  provided no hand-rendering of runtime-owned visuals.
- How to split the work into plans/waves.
- Whether `oracle.md`-style short wrappers need touch-ups is OUT of scope —
  only the four named files (both platforms) plus their YAML sources.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Ceremony source text (mine and inline; loader forbidden)
- `.aether/docs/command-playbooks/build-prep.md`, `build-context.md`,
  `build-wave.md`, `build-verify.md`, `build-complete.md` — Classic build
  ceremony and method text (~2,160 lines)
- `.aether/docs/command-playbooks/continue-verify.md`, `continue-gates.md`,
  `continue-advance.md`, `continue-finalize.md` — Classic continue ceremony
  (~2,252 lines)
- Git tag `v5.4.0` — Classic baseline for init.md (516 lines, thick) and
  plan.md (693 lines, thick): `git show v5.4.0:.claude/commands/ant/init.md`

### Contracts and ownership
- `.aether/docs/wrapper-runtime-ux-contract.md` — wrapper vs runtime
  ownership rules and the four anti-patterns; §139-150 forbid state
  mutation and duplicated gating
- `.aether/docs/wrapper-host-contract.md` — the host/manifest handshake;
  destination for envelope mechanics leaving the wrappers (D-03)

### Enforcement tests (extend, never weaken)
- `cmd/build_wrapper_ceremony_test.go` — forbids playbook loading; asserts
  build wrapper contract
- `cmd/continue_wrapper_ceremony_test.go` — forbids context-clear ceremony
  in wrapper (runtime owns it)
- `cmd/plan_wrapper_ceremony_test.go`, `cmd/plan_wrapper_cards_test.go` —
  Phase 164 decision-moment contract; two cards, printed verbatim
- `cmd/platform_doc_hygiene_test.go` — per-wrapper forbidden-string lists

### Runtime ceremony surface (invoke, never re-render)
- `cmd/ceremony_cmd.go` — `spawn-plan`, `wave-start`, `worker-complete`,
  `skill-assignments`, `closeout`, `queen-frame` renderers
- `cmd/codex_visuals.go` — caste maps, `spacedTitle`, `renderBanner`,
  `renderNextUp`, `milestoneIcon`

### Prior phase contracts the wrappers must document (not restructure)
- `.planning/phases/164-research-feeds-planning/164-CONTEXT.md` — two
  decision cards in plan.md (D-14/D-16), research choreography
- `.planning/phases/163-context-reaches-workers/163-CONTEXT.md` — manifest-
  level context injection the wrappers describe

### Archaeology report
- The Phase 165 archaeology findings (32-element ceremony inventory with
  verbatim exemplars, delta table with RT/WR ownership per element, 11
  regression warnings D-1..D-11 with commit evidence) were persisted to the
  bugs/history review ledger by the archaeologist agent and are reproduced
  in `.planning/phases/165-core-lifecycle-commands/165-ARCHAEOLOGY.md`

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `aether ceremony spawn-plan / wave-start / worker-complete / closeout` —
  already render Classic-style spawn plans, wave separators, streaming
  completion lines; current build.md already invokes some. Wrappers narrate
  around them.
- `aether pheromone-display` — renders the signals table; Classic pattern
  was "run it, then narrate the legend" (build-context.md:23-31).
- `AETHER_OUTPUT_MODE=visual aether status` — colony grounding at stage open.

### Established Patterns
- Wrapper tests use ordered-heading-set equality and forbidden-string lists —
  extend these for the new stage skeleton and ceremony beats (D-09).
- YAML source (`.aether/commands/*.yaml`) carries guardrails that must match
  wrapper guardrail sections; update in lockstep (D-11).

### Integration Points
- 20 of 32 Classic ceremony elements belong in the Go renderer, not wrapper
  text (delta table in archaeology report). Nine have NO current owner
  (wave-failure banner, escalation banner, verification report grid, pattern
  announce, graveyard caution, survey-loaded banner, visual checkpoint,
  project-complete block, resumption line). These are Phase 168 input — see
  deferred.

</code_context>

<specifics>
## Specific Ideas

- "I just wanted to have that rich ceremony shit that we used to have" — the
  user's core desire. The measure of success is the felt experience of
  running /ant-build: Queen voice, caste identity moments, wave drama,
  loud failure banners — grounded in real runtime state, never invented.
- The strongest Classic assets per archaeology: the escalation banner
  ("⚠ ESCALATION — QUEEN NEEDS YOU" with tried/options/awaiting-choice
  structure) and the rule-with-reason 🐜 blocks. Wrapper-side beats restore
  now; renderer-side banners are 168's.

</specifics>

<deferred>
## Deferred Ideas

- **Nine renderer-owned ceremony gaps** (wave-failure banner, escalation
  banner, verification report grid, workflow pattern announce — computed at
  `orchestrator.ts:95` but written to stderr, graveyard caution, survey-loaded
  banner, visual checkpoint, project-complete block, resumption line) plus
  richer build-summary lines (Pattern/Tools/Duration) — these need Go renderer
  work and belong to **Phase 168 (Your Eyes Back)**. The archaeology delta
  table is the shopping list.
- **Milestone/maturity framing in build/continue** — net-new (Classic barely
  had it); Phase 168 decides.

### Reviewed Todos (not folded)
- "continue-finalize does not count --reconcile-task as recorded
  reconciliation for the implementation_evidence gate" — Go runtime behavior
  fix, out of scope for a wrapper-text phase. Stays in backlog.
- "ts-host preflight timeout hardcoded" — largely addressed by Phase 163.2;
  weak match. Stays in backlog.

</deferred>

---

*Phase: 165-Core Lifecycle Commands*
*Context gathered: 2026-08-02*
