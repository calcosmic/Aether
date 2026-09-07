# Phase 200: Iterative Planning - Context

**Gathered:** 2026-09-07
**Status:** Ready for planning

<domain>
## Phase Boundary

Restore the visible February-April Scout → Route-Setter planning experience over the authoritative modern Go planning, revision, and finalization path. This phase owns evidence-grounded planning iterations, five-dimensional confidence, weakest-gap targeting, causal iteration deltas, material owner decisions, honest stop reasons, one approved owner-readable specification, and safe living-plan insertion and revision.

The result must let an owner watch a plan become better and understand why. In plain language: Scout investigates the uncertain parts, Route-Setter improves the route, and the Queen shows the evidence and asks only for decisions that genuinely belong to the owner. Go remains the source of truth for state, revision history, acceptance, and safety.

This phase does not absorb Queen-led build/continue execution and worker turnaround (Phase 201), substantive live Swarm/Watch/Oracle behavior (Phase 202), causal pheromone delivery or TypeScript-host preflight work (Phase 203), learning governance (Phase 204), final milestone acceptance (Phase 205), or Codex-native `$ant-*` lifecycle skills.

</domain>

<decisions>
## Implementation Decisions

### Iteration Presentation

- **D-01:** After every completed Scout → Route-Setter pass, `/ant-plan` shows a compact delta card containing fresh evidence, before→after confidence, the weakest remaining gap, semantic plan changes, and the explicit continue/stop reason. Full citations and detailed diffs remain available on demand rather than flooding the default view.
- **D-02:** Completed iteration cards append to a persistent, ordered timeline and remain attached to the accepted plan. Later passes may not overwrite or hide the visible history already persisted by the Go finalizer.
- **D-03:** Plan deltas are semantic rather than raw-text diffs. They enumerate added, changed, or removed phases, tasks, dependencies, acceptance checks, and recovery paths, and separately flag any proposed change that crosses an owner-authority boundary.
- **D-04:** Knowledge, requirements, risks, dependencies, and effort appear as whole-number before→after planning-readiness scores. Every changed dimension cites the fresh evidence that justified the movement and names its remaining gap; a score may not rise merely because prior evidence was restated.

### Owner Decision Boundaries

- **D-05:** Known material owner decisions are consolidated into one evidence-first checkpoint after Scout completes the first grounded pass. Planning must not begin with a generic category menu or ask questions that repository, survey, charter, Hive, outcome, or research evidence can answer.
- **D-06:** If a later pass discovers a genuinely new material decision, the current Scout → Route-Setter pass completes so the impact can be explained, then planning pauses at that pass boundary before an unauthorized revision is accepted.
- **D-07:** Every material-decision card states the decision, why it matters now, cited evidence, the Queen's recommendation, consequences of each viable choice, and what work resumes after the answer. The Queen advises rather than merely forwarding neutral alternatives.
- **D-08:** A prior owner answer is reused only while its goal, meaning, and consequences remain equivalent. If fresh evidence changes behavior, scope, risk, or acceptance impact, planning shows the prior answer and requests revalidation; answers remain bound to goal/session/revision and never become stale universal permission.

### Specification Lifecycle

- **D-09:** The normal guided journey hands off automatically from resolved `/ant-discuss` intent to a readable draft specification. `/ant-spec` remains a real public Claude/OpenCode command for opening, editing, approving, or revising that artifact later; it is not an empty presentation-only wrapper.
- **D-10:** A new or materially revised specification remains `draft` until the owner explicitly approves it. Only an approved revision may become the contract for a newly accepted plan; neither inferred intent nor resolved internal questions silently grants approval.
- **D-11:** Each colony goal has one canonical specification lineage. Feature-only work creates a scoped revision of identified requirements and acceptance criteria while preserving unaffected IDs and content; it does not create overlapping standalone sources of truth.
- **D-12:** A material specification edit creates an immutable revision linked to its predecessor, classifies unchanged/added/modified/removed requirements, preserves historical evidence, and invalidates only affected plan tasks and proof links. The changed contract must be approved and the affected plan scope reconciled before work or Seal may rely on it.

### Research Autonomy and Stop Policy

- **D-13:** When `/ant-plan` is invoked without explicit quality flags, the owner selects Fast, Balanced, Deep, or Exhaustive. The card exposes each existing preset's confidence target and iteration cap; there is no silent Queen-selected or fixed Deep default. Explicit valid flags continue to bypass this choice.
- **D-14:** Once the owner selects a preset, Scout and Route-Setter work autonomously within that agreed budget. Each later Scout pass targets the weakest evidenced gaps and the Queen explains why it ran; routine research does not need approval, while new material authority or safety boundaries still pause under D-06.
- **D-15:** Planning stops honestly for target sufficiency, diminishing returns, detected stall, or the selected iteration cap. A below-target draft may close automatically when every remaining gap is non-material, provided the timeline discloses the gaps, stop reason, and evidence that could change the next decision; a material gap produces the owner-decision card and pauses instead.
- **D-16:** The final plan becomes eligible for guided build or Autopilot only after explicit owner acceptance. The Queen presents the final plan, complete confidence history, remaining gaps, and recommendation first; specification approval does not double as plan acceptance.

### The Agent's Discretion

No owner-visible behavior was delegated. Research and planning may choose internal Go types, helper boundaries, the machine-readable SPEC index representation, exact terminal spacing/color, and bounded artifact-retention mechanics, provided they preserve the decisions above, thin-wrapper/runtime authority, immutable history, cross-platform semantic parity, and accessible drill-down evidence.

### Folded Todos

- `2026-08-20-spec-builder-feature.md` — The owner requested a user-facing flow that turns intent and decisions into a plain-language specification before building. Phase 200 now owns the real `/ant-spec` runtime contract, approval/revision lifecycle, automatic post-discuss handoff, owner-checkable acceptance, and plan traceability described in D-09 through D-12.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Milestone Authority

- `.planning/PROJECT.md` — Defines the active v1.28 Classic restoration goal, Claude/OpenCode-first platform order, modern Go authority, Phase 199-205 sequence, and approved todo routing.
- `.planning/ROADMAP.md` — Defines Phase 200's boundary, dependency, requirements, and five success criteria.
- `.planning/REQUIREMENTS.md` — Defines `SYNTH-02`, `CEC-03`, and `PLAN-01..06`, including the visible loop, causal confidence, material questions, owner-readable specification, and safe living-plan contract.
- `.planning/STATE.md` — Carries the current Phase 200 position and prior milestone decisions that planning must not reopen without new evidence.
- `.planning/research/priority-spec-v3-backlog.md` — Owner-ratified ordering for SPEC-first initialization, planning dimensions, Queen proposals, Go validation, and legacy-colony compatibility.

### Classic Evidence and Required Synthesis

- `.aether/dreams/2026-09-01-comprehensive-aether-colony-review.md` — Governing causal review of the February-April Classic experience versus the current runtime; reconstruct the planning mechanism rather than copying old files blindly.
- `.planning/research/v1.28-classic-capability-ledger.md` — Routes `CAP-005`, `CAP-010..012`, `CAP-056`, `CAP-061`, and `CAP-069` to Phase 200 with their required modern dispositions.
- `.planning/research/v1.28-classic-synthesis-template.md` — Mandatory keep-current/restore-modern/replace-better/retire-with-proof mechanism-study structure before implementation planning.
- Historical source anchor `3a5b81c2`, path `.claude/commands/ant/plan.md` — February dual-Scout/synthesis/Route-Setter loop, five confidence dimensions, gap targeting, visible progress, stall guidance, and owner acceptance.
- Historical source anchor `v5.0.0`, path `.claude/commands/ant/plan.md` — Worker Emergence-era compact planning loop and its changed confidence/iteration policy.
- Historical source anchor `v5.4`, paths `.aether/commands/plan.yaml` and `.claude/commands/ant/plan.md` — Mature April Scout → Route-Setter loop, territory/Hive/research priming, depth presets, gap-focused later passes, and automatic stopping.

### Prior Contract and Specification Authority

- `.planning/phases/199-front-door-and-classic-contract/199-CONTEXT.md` — Locks `/ant-*` as the ordinary Claude/OpenCode vocabulary, strong Queen voice, explicit plan acceptance before build/Autopilot, and the boundary between autonomous implementation detail and owner-authority changes.
- `.planning/phases/199-front-door-and-classic-contract/199-CLASSIC-SYNTHESIS.md` — Defines the modern front-door/runtime split and explicitly routes substantive planning restoration to Phase 200.
- `cmd/testdata/classic-contract/v1/schema.json` — Versioned semantic corpus schema that Phase 200 evidence should extend rather than replacing with snapshot-only tests.
- `cmd/testdata/classic-contract/v1/mechanisms.json` — Current machine-readable Classic mechanism inventory and historical-source citation pattern.
- `.aether/dreams/AETHER_PRIORITY_SPEC_V3_ANALYSIS_2026-08-21.md` §§15-16 — Defines SPEC identity, approval, supersession, impact-scoped invalidation, adaptive `/ant-spec`, planning presets, and Queen/Go authority separation.
- `.planning/todos/pending/2026-08-20-spec-builder-feature.md` — Original owner request and GSD-inspired acceptance model for whole-goal or feature-scoped readable specifications.

### Current Architecture and Contracts

- `.planning/codebase/ARCHITECTURE.md` — Maps current planning entry points, Go state authority, wrapper generation, and artifact/data flow.
- `.planning/codebase/CONVENTIONS.md` — Defines Go command organization, co-located tests, error handling, and concurrency conventions.
- `.planning/codebase/STACK.md` — Records the current Go/Cobra runtime and supported Claude/OpenCode presentation surfaces.
- `.aether/docs/wrapper-host-contract.md` — Defines the thin host-wrapper contract and prohibits wrappers from becoming independent state authorities.
- `RUNTIME UPDATE ARCHITECTURE.md` — Defines source, generated-wrapper, hub, install, and update synchronization requirements.

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets

- `cmd/codex_plan.go`: already models the five historical confidence dimensions, exact Fast/Balanced/Deep/Exhaustive preset bounds, planning-run and iteration identity, context capsules, Scout/Route-Setter dispatch, selected gaps, and intermediate artifacts.
- `cmd/codex_plan_finalize.go`: already rejects confidence movement without fresh evidence, chooses up to two weakest next gaps, distinguishes target/max/stall outcomes, persists intermediate iterations, and activates the accepted plan through the Go state path.
- `cmd/plan_revision.go`: already supplies plan hashes, immutable revision lineage, reason/evidence fields, completed-prefix preservation, scope validation, and replacement/supersession accounting for living plans.
- `cmd/discuss.go`: already scopes pending decisions by goal/session and quarantines stale answers, but currently emits generic categories and one-question-at-a-time runtime records rather than the locked evidence-first owner checkpoint and readable SPEC flow.
- `.aether/commands/plan.yaml` and `.aether/commands/discuss.yaml`: canonical presentation/orchestration sources for Claude and OpenCode. The plan wrapper already describes Scout then Route-Setter dispatch and current decision cards, providing the main public integration seam.

### Established Patterns

- Go owns authoritative validation, state mutation, revision/finalization, stop policy, approval state, and evidence truth. Wrappers may orchestrate host workers and render the Queen-led experience but may not compute or persist competing lifecycle truth.
- Planning manifests and finalizer results use freshness, workspace, ownership, identity, and evidence-hash checks. New SPEC and iteration behavior must extend these contracts rather than bypass them with Markdown-only state.
- Current confidence weights and preset ranges are working runtime primitives. Historical sources are behavioral evidence; they are not permission to restore shell-owned watch/state writes or prompt-only acceptance.
- Canonical `.aether/commands/*.yaml` sources and the generated `.claude/commands/ant/*.md` and `.opencode/commands/ant/*.md` surfaces must remain synchronized.
- Tests should prove outcome, behavior, experience, replay, and forbidden mutations through semantic fixtures and state/evidence assertions—not ANSI or prose snapshots alone.

### Integration Points

- `/ant-discuss` completion needs a truthful transition into a real Go-backed `/ant-spec` draft/approve/revise lifecycle, with an approved revision identity available to planning manifests and plan revisions.
- `/ant-plan` needs a preset-selection boundary, visible iteration events/cards, persistent confidence/delta history, owner-decision checkpoints, and explicit final acceptance projected consistently by Claude and OpenCode.
- Plan insertion and revision must consume the active approved SPEC revision, preserve completed work and old evidence as historical, identify affected tasks, and explain the reason/scope of every change.
- Survey, charter, Hive wisdom, prior outcomes, discussion deltas, and phase research enter Scout prompts as attributable evidence; stale answers or inadmissible cross-colony context may not silently narrow questions or widen the goal.
- The Phase 199 Classic contract corpus should gain Phase 200 mechanism/case coverage tying visible `/ant-*` behavior to the same authoritative Go transitions and artifacts.

</code_context>

<specifics>
## Specific Ideas

- The owner's explicit reminder is the phase thesis: bring back the “cool things” from February and April Aether. Use the actual `3a5b81c2`, `v5.0.0`, and `v5.4` planning sources as evidence and synthesize their best experience over modern safeguards; do not redesign planning from a blank page.
- Preserve February's legible research, confidence, gap, and owner-guidance story; preserve April's focused Scout → Route-Setter rhythm, context priming, depth presets, and autonomous routine passes; reject both eras' wrapper/shell state authority and any automatic handling of genuinely material owner choices.
- The default iteration view should feel like an understandable progress ledger, not an internal protocol dump: what Scout learned, what Route-Setter changed, why confidence moved, what remains weakest, and why the Queen continues or stops.
- The owner deliberately chose to select the planning preset rather than delegate that first depth decision to the Queen. Once selected, the Queen operates the loop without repeatedly asking permission.

</specifics>

<deferred>
## Deferred Ideas

- Queen-led work execution, team sizing, verification boundaries, recovery, and worker-turnaround improvements belong to Phase 201.
- Substantive live Swarm/Watch/Oracle behavior belongs to Phase 202.
- Causal live pheromone influence and TypeScript-host preflight/timeout handling belong to Phase 203.
- Learning governance and outcome-backed promotion belong to Phase 204; Codex-native `$ant-*` lifecycle skills remain a later milestone.

### Reviewed Todos (not folded)

- `2026-08-27-worker-turnaround-is-too-slow.md` — The one-plan-job latency and cost problem belongs to Phase 201's Queen-led work-cycle and attempt model, not planning synthesis.
- `2026-08-01-ts-host-preflight-hardcoded-timeout.md` — Host preflight timeout/configuration and repository-CWD handling remain routed to Phase 203; the keyword-only todo match does not change ownership.

</deferred>

---

*Phase: 200-iterative-planning*
*Context gathered: 2026-09-07*
