# Phase 164: Research Feeds Planning - Context

**Gathered:** 2026-08-01
**Status:** Ready for planning

<domain>
## Phase Boundary

Before a phase is planned, the Queen decides whether it needs research and
states why; the user can override either way. When research is warranted, a
worker runs automatically through the existing `confidence-loop.ts` (kept,
not rebuilt) with a visible confidence readout; findings persist to a durable
per-phase artifact and feed the planner's context directly; re-planning
re-researches from scratch; territory survey context reaches both the research
worker and the planner. At plan time the Queen proposes plan granularity,
task decomposition depth, and verification depth with a plain-English reason
each, presented as multiple-choice selections.

**Premise corrections this phase must plan against:** the roadmap text
("plan.md contains zero references to Oracle; research is a standalone command
the user must paste in") is stale. `cmd/phase_research.go` already dispatches
one research Scout per drafted phase in wave 1 of `/ant-plan`, writing to
`.aether/data/phase-research/phase-N-research.md`, and `plan.md` documents
this choreography. What's genuinely missing: the Queen's per-phase decision
with reason and override (research currently runs unconditionally), the
confidence loop wired into research (it drives only build/continue today),
re-research on replan (current code deliberately reuses findings — the
opposite of RESEARCH-04), depth-bound targets, and the Queen's depth
proposals. This phase is wiring and completion, not greenfield.

</domain>

<decisions>
## Implementation Decisions

### The Queen's research decision (RESEARCH-01, RESEARCH-02)
- **D-01: One batched, tick-to-approve proposal.** After the route is drafted,
  the Queen presents a single list — "Phase 2: research (new external API);
  Phase 3: skip (pure refactor, codebase mapped)" — and the user approves or
  flips entries in one interaction. Same interaction pattern as
  `suggest-approve`. No per-phase interruptions, no decide-and-proceed.
- **D-02: Queen judgment grounded by runtime hints.** The Go runtime computes
  cheap signals per phase (external tech/API mentions, domain absent from the
  colonize survey map, phase mode) and passes them to the orchestrating Queen,
  who makes the call and writes the plain-English reason. Not pure keyword
  heuristics (Phase 167 is moving away from keyword inference) and not
  ungrounded LLM vibes.
- **D-03: Overrides are recorded as decisions.** A user flip (either
  direction) lands in the existing decision/assumption model (RESEARCH-06: no
  new planning store). Plan artifacts show "user overrode: skip research on
  phase 3". Replans see the record but still re-propose — overrides are not
  sticky.
- **D-04: Scout by default, Oracle on escalation.** The per-phase researcher
  stays a Scout (cheap, parallel, already wired). When the confidence loop
  stalls below target at deep/exhaustive depth, the Queen may escalate that
  one phase to an Oracle. Depth buys a heavier researcher only where earned.

### Re-research on replan (RESEARCH-04)
- **D-05: Replans always re-research.** The current `hasWorkerAuthoredResearch`
  skip-if-exists behavior in `cmd/phase_research.go` is inverted for replans:
  a re-run of plan re-researches flagged phases from scratch. Within a single
  plan run's iterations, research still runs once per phase (unchanged).
- **D-06: The batched proposal is the cost control.** On replan the Queen
  re-proposes research per phase with reasons; everything approved
  re-researches fresh. The user sees the full list before workers spawn and
  can flip entries off. No silent 10-worker surprises, no staleness-threshold
  reuse (that quietly drifts back to the failure mode RESEARCH-04 targets).
- **D-07: Findings overwrite in place.** `phase-N-research.md` regenerates —
  one durable artifact per phase, no timestamped archive siblings.
- **D-08: Research failure warns loudly, planning proceeds.** Research is
  enrichment, not a gate (Phase 160 classification). If a research worker
  fails or produces nothing usable, the planner runs anyway and both the plan
  artifact and finalize output state "phase N planned WITHOUT its research —
  worker failed". Never blocks, never silent.

### Confidence readout & early accept (RESEARCH-07, RESEARCH-08)
- **D-09: Full colony ceremony.** Each research Scout announces with caste
  emoji + ANSI color + deterministic ant name (🔍 Scout Antenna-42), and every
  loop iteration prints the ceremony line — confidence %, delta, budget
  remaining, stop reason at the end. `renderIterationCeremony()` in
  `confidence-loop.ts` already produces the line format. Consistent with
  Phase 168's living-colony direction.
- **D-10: Early-accept prompts only on stall or near-target.** The loop runs
  on its own; the user is asked only when diminishing returns are detected
  below target, or confidence is within ~5% of target with iterations left:
  "At 87%, gaining 2%/iteration — accept now or keep digging?" No per-iteration
  nagging, no upfront-only flag.
- **D-11: Confidence is evidence-scored with a self-check blend.** The runtime
  scores checkable evidence — required sections filled, patterns/gotchas cited
  with real paths or URLs, files-to-study verified to exist — blended with the
  researcher's own gap assessment. The number must be reproducible so a test
  can fail when scoring breaks (Definition of Done). A bare self-score from a
  cheap model is not trusted alone.
- **D-12: Depth binds target and iteration budget** per RESEARCH-08: fast
  80%/4, balanced 90%/6, deep 95%/8, exhaustive 99%/12 — fed into the existing
  `ConfidenceLoopOptions` (`confidenceTarget`, `maxIterations`), not a new loop.

### Depth proposal ceremony (RESEARCH-09, RESEARCH-10)
- **D-13: Multiple-choice proposal card, zero typing.** At plan start the
  Queen presents granularity, task decomposition depth, and verification depth
  as selectable options with her recommended pick pre-marked and a one-line
  plain-English reason each ("milestone granularity — goal spans several
  subsystems"). Accepting the recommendations is one tap; changing any knob is
  a tap on a different option. The user's words: typing everything is "a pain
  in the ass" — selection, not forms.
- **D-14: The plan flow has exactly two decision moments.** The depth proposal
  card and the research batch (D-01) — both tap-to-approve, both selection-
  based. No third ceremony.
- **D-15: Fast preset — Queen leans skip.** On a fast run the research
  proposal defaults every phase to "skip", but the batch still appears so the
  user can flip a phase on; a flipped-on phase runs at 80%/4. Speed stays
  fast's default contract, RESEARCH-08's numbers apply whenever research
  actually runs, nothing is silently impossible.
- **D-16: Autopilot auto-accepts, records, and logs.** Under `/ant-run` both
  decision moments auto-accept the Queen's recommendations, record them
  through the existing decision model, and print them in the run log
  ("auto-accepted: research phases 2,4; granularity milestone"). No smart
  pauses for proposals.

### Claude's Discretion
- How territory survey context (RESEARCH-05) is plumbed into the research
  brief and planner context — ride the manifest-level context path Phase 163
  restored, mechanics up to the planner.
- The evidence-scoring formula internals (weights, section checks, citation
  verification mechanics) — bound by D-11's reproducibility rule.
- Runtime-hint computation details for D-02 (which signals, how surfaced to
  the Queen).
- Oracle escalation mechanics (threshold, brief shape) for D-04.
- Where depth→target/iteration binding lives (Go vs TS boundary), given the
  existing `--target`/`--max-iterations` flags in `cmd/codex_plan.go`.
- Exact ceremony wording, proposal card copy, and log-line formats.
- Whether `renderPhaseResearchBrief`'s six-section format changes to support
  evidence scoring.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Requirements and roadmap
- `.planning/REQUIREMENTS.md` §RESEARCH-01..10 — requirement text (lines 84-93)
- `.planning/ROADMAP.md` §Phase 164 — goal, dependencies (160, 163), success
  criteria; note the stale "zero references" premise corrected in <domain>

### Existing research machinery (extend, don't rebuild)
- `cmd/phase_research.go` — the whole file: candidates, dispatch emission,
  `hasWorkerAuthoredResearch` (D-05 inverts this on replan),
  `renderPhaseResearchBrief`, `resolvePhaseResearchSection` (3500-char build
  brief budget)
- `.claude/commands/ant/plan.md` — documents today's wave-1 research Scout
  choreography (lines ~74-93); must stay accurate after this phase
- `.aether/data/phase-research/` — existing per-phase findings artifacts

### The confidence loop (use as-is — roadmap constraint)
- `.aether/ts-host/src/confidence-loop.ts` — `ConfidenceLoopOptions`
  (`maxIterations`, `confidenceTarget` — D-12's binding points),
  `renderIterationCeremony` (D-09's line format), stop conditions
- `.aether/ts-host/src/confidence-evaluator.ts` — existing bridge to
  ConfidenceMetric (D-11's natural home)
- `.aether/ts-host/src/host.ts:842-852` — how build/continue already
  constructs the loop; the pattern research wiring should mirror

### Plan command surfaces
- `cmd/codex_plan.go` — depth presets fast|balanced|deep|exhaustive,
  `--target`/`--max-iterations` flags (:997), `resolvePlanningLoopOptions`,
  `planBoundaryQuestionCandidates` (:833 — existing mechanism passing
  granularity/planning-depth/verification-depth; natural home for D-13)
- `pkg/colony/granularity.go` — `GranularityRange` (sprint 1-3, milestone 4-7,
  quarter 8-12, major 13-20) for RESEARCH-09's proposal copy

### Prior phase constraints
- `.planning/phases/163-context-reaches-workers/163-CONTEXT.md` — research
  findings ride the manifest-level context path (roadmap dependency); D-10
  no-silent-automation precedent; D-11 end-of-build suggestion pattern D-01
  mirrors
- `.planning/phases/160-fail-loudly/160-CONTEXT.md` — gate-vs-enrichment
  classification behind D-08

### Presentation constraints
- `cmd/codex_visuals.go` — `casteIdentity()`, `casteEmoji()`, `casteColorMap`
  for D-09's ceremony identity
- `.aether/docs/wrapper-runtime-ux-contract.md` — wrappers present, runtime
  owns truth; the proposal cards are wrapper-rendered from runtime-emitted data

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `plannedPhaseResearchDispatches` (cmd/phase_research.go): the dispatch
  emission this phase gates behind the Queen's decision instead of
  running unconditionally
- `ConfidenceLoop` (confidence-loop.ts, 250 lines): target/iteration knobs
  already exist as constructor options — D-12 is configuration, not code
- `planBoundaryQuestionCandidates` (cmd/codex_plan.go:833): existing
  plan-time question mechanism to carry the D-13 proposal card
- `suggest-approve` tick-to-approve UI: the interaction pattern for D-01
- Existing decision/assumption model (`pending-decisions.json`,
  `assumptions.json`): where D-03/D-16 records land — RESEARCH-06 forbids a
  new store

### Established Patterns
- Runtime owns truth, wrappers own presentation: decision logic, scoring, and
  depth binding live in Go/runtime; Claude Code and OpenCode wrappers render
  the proposal cards and ceremony
- Definition of Done: each decision needs a command/test that fails when
  unmet — e.g. a test that fails if replan reuses findings (D-05), if the
  confidence number is not reproducible (D-11), or if fast silently makes
  research impossible (D-15)
- No automatic model selection, ever (standing user rule) — Oracle escalation
  (D-04) changes the caste/worker, never silently swaps models

### Integration Points
- Research findings → planner context and build briefs via the manifest-level
  context path Phase 163 restored (`resolvePhaseResearchSection` already
  handles the build-brief side)
- Phase 165 (Core Lifecycle Commands) documents whatever research step this
  phase adds to `plan.md` — keep the choreography describable in prose
- ts-host changes require `npm --prefix .aether/ts-host run build` (dist is
  embedded); fixes need both Go AND ts-host paths plus dist rebuild

</code_context>

<specifics>
## Specific Ideas

- The user's north star, verbatim (mid-discussion, 2026-08-01): "I just want
  this to have all of the functionality that we enjoyed and we've developed
  through… all the beautiful shit that we did… the oracle was to do like a
  ralph loop thing so it just iteratively go over stuff until it finds like
  the best things… it's just important that all of that stuff is networked
  together that's fundamentally the thing… we wanted all the emojis and the
  caste colors and all this shit to come back… it's just to get this back as
  a working framework." This phase restores the iterate-until-confident
  research behavior as part of that larger switching-on.
- On the proposal UX: "for a user to have to type everything that just seems
  like a pain in the ass" — selections, never forms (D-13).
- The acceptance story: run `/ant-plan`, tap through two selection cards
  (depths, research batch), watch named colorful Scouts iterate with visible
  confidence climbing, get a plan whose per-phase sections visibly cite the
  fresh research.

</specifics>

<deferred>
## Deferred Ideas

### Reviewed Todos (not folded)
- **ts-host preflight timeout hardcoded** — matched by keyword only; it is
  Phase 163.2's executed scope, not research. Remains with 163.2.
- **continue-finalize does not count --reconcile-task for the
  implementation_evidence gate** — continue-finalize bug, unrelated to
  research-feeds-planning; already reviewed-and-deferred by 163.2. Remains
  pending for a future phase.

</deferred>

---

*Phase: 164-research-feeds-planning*
*Context gathered: 2026-08-01*
