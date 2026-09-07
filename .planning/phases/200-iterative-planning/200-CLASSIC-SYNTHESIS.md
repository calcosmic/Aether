---
artifact: v1.28-classic-to-go-phase-synthesis
phase: "200"
area: planning
status: ready_for_planning
historical_refs:
  - 3a5b81c2
  - v5.0.0
  - v5.4
current_revision: "865f2f920067"
cap_rows:
  - CAP-005
  - CAP-010
  - CAP-011
  - CAP-012
  - CAP-056
  - CAP-061
  - CAP-069
last_updated: "2026-09-07"
---

# Phase 200 Classic-to-Go Synthesis

This is the mandatory pre-plan mechanism study for Phase 200. It treats Classic as evidence about a useful experience, not as code to restore, and treats the current Go runtime as the authority to extend. [VERIFIED: `.planning/research/v1.28-classic-synthesis-template.md`; `.planning/phases/200-iterative-planning/200-CONTEXT.md`]

## 1. Outcome under investigation

- **Owner-visible outcome:** the owner watches Scout investigate uncertainty, Route-Setter improve the route, and the Queen explain after every pass what changed, why confidence moved, what remains weakest, and why planning continues or stops. The owner chooses the quality preset, answers only material questions, approves the specification, and separately accepts the final plan. [VERIFIED: `200-CONTEXT.md` D-01..D-16]
- **Operational outcome:** one approved, owner-readable specification grounds a bounded sequence of evidence-backed planning iterations; the Go runtime validates every stage, persists an append-only timeline, produces an exact candidate plan, and activates it only after explicit acceptance. [VERIFIED: `200-CONTEXT.md` D-02, D-09..D-16; `.planning/REQUIREMENTS.md` PLAN-01..06]
- **Why this phase owns it:** the roadmap routes `SYNTH-02`, `CEC-03`, and `PLAN-01..06` here; the capability ledger routes `CAP-005`, `CAP-010..012`, `CAP-056`, `CAP-061`, and `CAP-069` here. [VERIFIED: `.planning/ROADMAP.md` Phase 200; `.planning/research/v1.28-classic-capability-ledger.md`:138-145,189-202]
- **Non-goals:** build/continue team behavior is Phase 201; substantive Swarm/Watch/Oracle is Phase 202; causal pheromone delivery and TypeScript-host preflight are Phase 203; learning governance is Phase 204; Codex-native `$ant-*` is later. [VERIFIED: `200-CONTEXT.md` Deferred Ideas]

For dummies: keep the modern engine and brakes, but restore the dashboard that shows the route getting better—and never start driving until the owner accepts the route.

## 2. Classic mechanism reconstruction

| Mechanism ID | Historical evidence | Actors and control loop | State and information flow | Stop/failure/recovery behavior | Owner-visible effect | Why it worked |
|---|---|---|---|---|---|---|
| `OLD-01` | `3a5b81c2:.claude/commands/ant/plan.md:109-295` | Two Scouts split structural and edge/operations lenses, then a synthesis ant reconciled them. | Typed findings, gaps resolved/remaining, conflicts, confidence, and unique insights flowed into one synthesis. | Conflicts and missed gaps were explicit; each pass could target prior gaps. | Research looked like independent investigation rather than one opaque answer. | Different lenses created cross-checking and made uncertainty visible. [VERIFIED: cited ref/path/lines] |
| `OLD-02` | `3a5b81c2:.claude/commands/ant/plan.md:297-411` | Route-Setter consumed synthesized evidence and the previous draft on every iteration. | The output carried phases/tasks, five scores, `delta_reasoning`, and unresolved gaps. | The prompt prohibited changing the plan without new information. | The owner could understand that research caused a concrete plan change. | It formed a legible alternating feedback loop rather than a single generation. [VERIFIED: cited ref/path/lines] |
| `OLD-03` | `3a5b81c2:.claude/commands/ant/plan.md:418-471,542-568` | Queen tracked gap recurrence, score deltas, stall, diminishing returns, and a hard cap. | Five dimensions used 25/25/20/15/15 weights; stuck gaps could become human-input gaps. | Stall and diminishing returns asked the owner to continue, accept, or guide; the cap disclosed residual gaps and a recommendation. | Stops felt explained and owner-controlled. | It connected a stop to measurable lack of progress and remaining uncertainty. [VERIFIED: cited ref/path/lines] |
| `OLD-04` | `3a5b81c2:.claude/commands/ant/plan.md:473-535` | The wrapper treated confidence ≥99 or owner approval as acceptance, then rendered the plan. | Prompt instructions directly rewrote `COLONY_STATE.json` and watch/session files. | A failed write had little transaction protection. | The final plan, confidence, iterations, and next action were easy to see. | Strong ceremony closed the research story, but persistence depended on prompt/shell obedience. [VERIFIED: cited ref/path/lines] |
| `OLD-05` | `v5.0.0:.claude/commands/ant/plan.md:122-154,166-386` | One Scout ran broad discovery first, gap-only discovery later; one Route-Setter followed. | Territory survey context and compact JSON flowed directly to the planner. | Target 80, cap 4, and two low-delta passes ended the loop. | The rhythm stayed visible at lower cost. | Focused later passes avoided repeatedly surveying known ground. [VERIFIED: cited ref/path/lines] |
| `OLD-06` | `v5.0.0:.claude/commands/ant/plan.md:393-434,530-542` | Queen auto-finalized when target, cap, or stall fired. | Remaining gaps were deferred to builds and wrapper code wrote canonical state. | No owner acceptance was required. | Fast, low-friction completion. | It improved efficiency, but converted uncertainty and acceptance into silent policy. [VERIFIED: cited ref/path/lines] |
| `OLD-07` | `v5.4:.aether/commands/plan.yaml:65-96` and generated `v5.4:.claude/commands/ant/plan.md:68-97` | Queen offered Fast 80/4, Balanced 90/6, Deep 95/8, Exhaustive 99/12 and loaded a compact context capsule. | Preset bounded both target and passes. | Unclear input silently defaulted to Deep. | The owner could trade latency for confidence in plain language. | Named presets made a complex budget decision understandable. [VERIFIED: cited ref/path/lines] |
| `OLD-08` | `v5.4:.aether/commands/plan.yaml:131-287` and generated `v5.4:.claude/commands/ant/plan.md:168-289` | Territory evidence and a phase-domain Scout primed planning; Hive wisdom was read first. | Research was written as per-phase Markdown and summarized into Route-Setter context. | Failure was best-effort; rerun deleted old phase research before replacement. | The plan appeared grounded in the actual repository and remembered knowledge. | Durable research reduced rediscovery and made specialist knowledge inspectable. [VERIFIED: cited ref/path/lines] |
| `OLD-09` | `v5.4:.aether/commands/plan.yaml:288-555,658-690` | One Scout → one Route-Setter repeated with broad-first/gap-later prompts. | The previous draft, gaps, phase research, and context capsule fed the next pass. | Target, cap, or two <5-point passes auto-finalized; `--accept` was an escape hatch. | Mature planning felt active and bounded. | It combined April’s useful grounding and presets with February’s recognizable rhythm, but hid owner authority at finalization. [VERIFIED: cited ref/path/lines] |
| `OLD-10` | `3a5b81c2:.claude/commands/ant/council.md:47-176`; `v5.4:.claude/commands/ant/council.md:265-390` | Council opened with generic Project Direction/Quality/Constraints/Custom menus and translated answers into signals. | Wrapper logic wrote constraints/pheromones. | The owner could be repeatedly prompted without repository evidence. | It exposed control, but made the owner do classification work. | The memorable steering idea was valuable; the generic menu was not evidence-first. [VERIFIED: cited ref/path/lines] |
| `OLD-11` | `git ls-tree -r --name-only 3a5b81c2 v5.0.0 v5.4` | Classic had `init` and `council`, but no `discuss` or `spec` public command in these anchors. | Goal/clarifications flowed directly toward planning rather than an approved requirement lineage. | There was no spec revision/approval/invalidation boundary. | Planning could start quickly. | The missing contract explains why plan acceptance and intent approval became conflated. [VERIFIED: cited Git tree inspection] |

Classic’s value was causal: visible independent evidence fed a prior draft, scores and gaps changed, and the Queen narrated the consequence. Its unsafe parts were also causal: prompt code owned files, confidence was largely self-reported, research replacement could destroy history, and April auto-finalization treated a stop as acceptance. [VERIFIED: `3a5b81c2:.claude/commands/ant/plan.md:126-485`; `v5.4:.aether/commands/plan.yaml:177-185,540-583`]

## 3. Current Go mechanism audit

| Mechanism ID | Current public path and evidence | Current actors and control loop | Durable authority/state | What is safer or better now | What is thinner, missing, or disconnected |
|---|---|---|---|---|---|
| `NOW-01` | `/ant-plan` canonical source `.aether/commands/plan.yaml`; runtime `aether host plan` → `plan-finalize`; contract `cmd/contracts/plan.md:1-128` | Manifest requires Scout then Route-Setter, with optional phase-research Scouts before Route-Setter. | Go validates root, goal, workspace, territory snapshot, worker identities, terminal results, evidence and dependencies. | Planned/spawned work cannot masquerade as completion; stale/replayed packets fail before state mutation. | The wrapper describes a loop, but default rendering shows only overall confidence and does not show a causal per-pass delta card. [VERIFIED: `cmd/codex_plan_finalize.go:299-384,693-768`; `cmd/codex_visuals.go:1440-1728`] |
| `NOW-02` | `cmd/codex_plan.go:192-298,1950-2145` | Presets and iteration identity carry target, cap, prior score/hash, selected gaps, previous draft and history. | Intermediate state and per-iteration Scout/phase-plan JSON are persisted under `.aether/data/planning/`. | A real bounded loop exists; it is not merely Classic-themed text. | History has overall score/delta and one summary string, not five before→after evidence chains, semantic diffs, materiality, or decision/stop causality. [VERIFIED: cited current code] |
| `NOW-03` | `cmd/codex_plan_finalize.go:771-930` | Go compares evidence hashes, computes stall and selects up to two gaps. | Evidence hash covers Scout findings/gaps/files and Route-Setter phases/gaps. | A changed overall score with an identical evidence hash is rejected. | Any changed evidence can justify any score movement; per-dimension evidence is absent, overall can be worker-supplied, explicit gaps are alphabetically limited rather than ranked by materiality, and diminishing returns is not a distinct stop. [VERIFIED: cited current code; `cmd/codex_plan.go:2619-2644`] |
| `NOW-04` | `cmd/codex_plan_finalize.go:932-1003` | A below-target pass records iteration artifacts and issues the next manifest command. | `iteration-state.json` binds run/goal/root/depth/target/cap/revision. | Resume/replay identity and gap-directed passes are substantially stronger than Classic variables. | Terminal activation deletes the locator state; per-pass files remain but the accepted plan revision does not attach their ordered cards/digests. [VERIFIED: `cmd/codex_plan_finalize.go:497,932-1003`; `pkg/colony/colony.go:422-481`] |
| `NOW-05` | `cmd/plan_revision.go:86-357`; tests `cmd/plan_revision_test.go`, `cmd/plan_revision_evidence_test.go` | Refresh binds a reason/type/evidence and preserves the completed prefix. | Plan revisions carry parent, hashes, evidence, planning run, full snapshots, preserved/superseded/replacement phase IDs. | Scope, symlink containment, active-attempt, stale-base, cycle, and immutable-completed-work checks are real. | Revision replaces the entire unfinished suffix; there is no approved spec revision, requirement-level impact map, task/proof invalidation, or separate candidate acceptance. [VERIFIED: cited current code/tests] |
| `NOW-06` | `cmd/codex_plan_finalize.go:401-564` | A terminal stop immediately calls `activateGeneratedPlan`. | One atomic state update sets READY and activates a revision. | State activation is atomic and evidence-bound. | `target_reached`, `stalled`, `max_iterations`, or pre-set `--accept` all become accepted plans before the owner sees the exact final candidate. [VERIFIED: cited current code; `cmd/codex_workflow_cmds.go:57-114,2452-2469`] |
| `NOW-07` | `cmd/discuss.go:1-720`; `.aether/commands/discuss.yaml` | Runtime generates or accepts grounded one-at-a-time clarification records; wrapper may compose questions. | Decisions are scoped by session/goal, stale records are quarantined, and hard answers can emit REDIRECT. | Existing scoping and stale-answer protection are reusable. | Runtime fallback still generates generic surface/integration/scope/verification categories; no evidence-backed batch, consequence model, revision binding, or spec handoff exists. [VERIFIED: `cmd/discuss.go:251-381,543-704`; `.aether/commands/discuss.yaml`] |
| `NOW-08` | `rg -n "aether spec|ant-spec|SpecRevision" cmd .aether/commands .claude/commands/ant .opencode/commands/ant pkg` | No public spec lifecycle is wired. | Accepted charter, plan revision and evidence fields exist, but no destination contract. | The missing feature is not hidden under another command. | There is no canonical spec lineage, approval, stable requirement/acceptance IDs, or impact-scoped invalidation. [VERIFIED: repository search; `.aether/dreams/AETHER_PRIORITY_SPEC_V3_ANALYSIS_2026-08-21.md:1016-1081`] |
| `NOW-09` | `cmd/state_extra.go:360-525`; `cmd/state_extra_test.go:218-834` | `aether insert-phase` gathers an issue and directly inserts/renumbers a phase. | State update is locked/atomic; recovery-specific insertions preserve a Swarm record. | It validates position and avoids partial state writes. | Ordinary insertion mutates the active plan outside `PlanRevision`, renumbers existing IDs, has no spec/impact check, and does not link planning research/history. [VERIFIED: cited current code/tests] |
| `NOW-10` | `cmd/codex_plan.go:850-1260`; `cmd/phase_research_*` | Runtime proposes a per-phase research batch and waits for approval before dispatch. | Decisions use the pending-decision store; research files are preserved rather than wholesale deleted. | Research is bounded, inspectable, and Route-Setter receives it; permission tests exist. | D-14 supersedes the second approval moment: routine research must be autonomous inside the selected preset. [VERIFIED: cited current code; `cmd/phase_research_dispatch_test.go`; `200-CONTEXT.md` D-14] |
| `NOW-11` | `pkg/colony/colony.go:422-481`; `.aether/dreams/AETHER_PRIORITY_SPEC_V3_ANALYSIS_2026-08-21.md:1016-1164` | Current Plan/PlanRevision provide a viable extension point; the prior audit already proposed hashable spec identity and predecessor-linked impact. | Go state is the existing authority boundary. | No parallel runtime is needed. | The spec and accepted-plan gates must be added without moving policy into wrappers or the deferred TypeScript-host phase. [VERIFIED: cited current code/governing audit] |

## 4. Comparative synthesis matrix

| Decision ID | Desired outcome/value | Classic mechanism | Current mechanism | Options considered | Selected disposition | Evidence and rationale | Safety/compatibility consequence |
|---|---|---|---|---|---|---|---|
| `SYN-200-01` | Real safe planning kernel | Prompt/shell loop and direct state writes | Fresh manifests, typed workers, evidence hashes, replay/stale checks, atomic revision activation | copy Classic / replace Go / retain kernel | `keep-current` | Current finalizer invariants are strictly stronger and already proven. [VERIFIED: `cmd/codex_plan_finalize_test.go:131-785,967-1024`] | Extend validation; never let YAML, Markdown, shell, or host become state authority. |
| `SYN-200-02` | Legible Scout → Route-Setter improvement | Visible alternating passes and prior-draft refinement | Real loop exists but presentation collapses it | leave hidden / copy old prose / typed modern timeline | `restore-modern` | Restore the causal rhythm as Go-emitted stage and delta records, not worker theatre. [VERIFIED: OLD-01..03; NOW-01..04] | Each card is derived from validated persisted facts and attached to its plan revision. |
| `SYN-200-03` | Honest five-dimensional confidence | Five scores and weighted overall, mostly honor-system | Five scores plus coarse evidence-hash guard | keep coarse / remove scores / evidence-chain scores | `replace-better` | Keep historical dimensions/weights but require fresh evidence refs and remaining gap per changed dimension; Go recomputes overall. [VERIFIED: `3a5b81c2...:542-555`; `cmd/codex_plan.go:77-84,2619-2644`; D-04] | A dimension cannot move on unrelated evidence or restatement; scores may decrease. |
| `SYN-200-04` | Ask only material owner questions | Generic council menus and ad hoc guidance | Scoped pending decisions, generic plan boundary candidates | keep menus / ask nothing / evidence-backed batch | `replace-better` | Reuse scoping but add a typed batch after first Scout evidence and pass-boundary handling for later discoveries. [VERIFIED: OLD-10; NOW-07; D-05..D-08] | No Route-Setter acceptance crosses an unresolved behavior/authority/risk/scope decision; stale answers require revalidation. |
| `SYN-200-05` | Understandable budget with autonomous research | April presets plus automatic loop | Three-knob proposal plus a separate research-approval batch | silent Queen default / retain two decisions / one preset choice | `restore-modern` | Owner selects the existing preset if flags are absent; within it routine Scout/phase research is autonomous. [VERIFIED: OLD-07..09; NOW-10; D-13..D-14] | Explicit valid flags bypass selection; no silent Deep default and no routine research approval gate. |
| `SYN-200-06` | Honest stop distinct from acceptance | February owner stop; April auto-finalize | target/stall/cap/`accepted` stop immediately activates | keep current / always prompt every pass / candidate then accept | `replace-better` | Stops become sufficiency, diminishing returns, stall, cap, or material decision; every viable final draft becomes a non-active candidate until explicit owner acceptance. [VERIFIED: OLD-03, OLD-06, NOW-06; D-15..D-16] | Pre-authorizing an unseen candidate with bare `--accept` is forbidden; acceptance binds candidate/spec/base hashes. |
| `SYN-200-07` | One readable destination contract | No Classic spec lifecycle | Accepted charter but no spec command/lineage | plan only / Markdown authority / Go-backed spec lineage | `replace-better` | Add real `/ant-spec` open/edit/approve/revise backed by immutable Go schema and a readable projection; resolved discuss automatically creates/updates a draft. [VERIFIED: OLD-11; NOW-08; D-09..D-11] | Only an approved exact revision can ground a new accepted plan; wrapper cannot infer approval. |
| `SYN-200-08` | Safe feature-scoped change | Whole plan replacement | Completed-prefix preservation, unfinished-suffix replacement | mutate in place / keep suffix model / impact graph | `replace-better` | Spec revision classifies requirement deltas and computes affected plan task/proof links; unchanged IDs/content remain. [VERIFIED: NOW-05; priority analysis §15.3; D-12] | Historical evidence is retained as historical; build/Seal blocks only on unresolved current affected scope. |
| `SYN-200-09` | Living plan with truthful history | Insert-phase was convenient but not revision-safe | Atomic insert plus separate revision machinery | delete insert / keep mutation / route through revision | `restore-modern` | Retain the proven corrective UX, but make insert/revision consume approved spec/context, record reason/evidence/impact, and activate only as an accepted PlanRevision. [VERIFIED: NOW-05, NOW-09; CAP-010..012] | No renumbering or direct active-plan mutation outside revision acceptance; active attempts remain a hard stop. |
| `SYN-200-10` | Attributable evidence priming | Territory, context, phase research and raw Hive text | Survey snapshots, context capsule, research docs, Hive injection exist | restore semantic shell / opaque prompts / typed evidence index | `replace-better` | Preserve current services and attach source ID/path/hash/scope/admissibility to every planning input; do not revive `semantic-cli.sh`. [VERIFIED: OLD-08; `cmd/codex_plan.go`; CAP-056,061,069] | Inadmissible/stale Hive or context cannot silently change scope; changed evidence invalidates stale packets. |
| `SYN-200-11` | Cross-platform Queen experience | Large Claude-only prompt owned behavior | Canonical YAML, generated Claude/OpenCode, Go JSON/visual renderers | wrapper logic / runtime-only bland output / thin orchestration | `restore-modern` | YAML orchestrates and narrates only Go-issued facts/actions; Go exposes all cards/states. [VERIFIED: `.aether/commands/plan.yaml`, `.aether/commands/discuss.yaml`, AGENTS.md] | Update canonical YAML, generated Claude/OpenCode, command guide, docs and semantic parity tests together. |
| `SYN-200-12` | Proof of behavior, not nostalgia | Prompt text and watch files | Strong unit/finalizer tests plus Phase 199 semantic corpus | snapshots / unit-only / extend corpus | `replace-better` | Extend `classic-contract/v1` with planning/spec/living-plan groups and causal public-path cases. [VERIFIED: `cmd/testdata/classic-contract/v1/schema.json`, `mechanisms.json`] | Schema must broaden `SYN-199-*` decision validation compatibly; existing Phase 199 fixtures remain valid. |

## 5. Selected architecture

### Authoritative flow

```text
/ant-discuss
  -> Go-scoped material answers
  -> idempotent draft SPEC revision
  -> /ant-spec review + explicit approval
  -> /ant-plan preset selection (only when no valid explicit flag)
  -> Go issues Scout stage manifest
  -> Go validates Scout evidence
       -> first pass: one material-decision batch if needed
       -> otherwise: Go issues Route-Setter stage manifest
  -> Go validates Route-Setter candidate and computes semantic delta + score movement
       -> later material decision: persist pass, pause before candidate acceptance
       -> continue: target weakest gaps with next Scout
       -> stop: persist exact non-active plan candidate
  -> Queen shows final plan + full timeline + residual gaps + recommendation
  -> owner explicitly accepts candidate
  -> Go atomically activates PlanRevision bound to SPEC revision and timeline digest
```

The present whole-chain completion packet cannot enforce D-05 after Scout and before the first Route-Setter. Split one iteration into Go-authorized `scout` and `route` stages while retaining `plan-finalize` as the validation boundary: a completion may advance only the stage named by the manifest, and each returned next-stage manifest binds the previous result hash. [VERIFIED: current limitation in `cmd/codex_plan_finalize.go:299-384,693-768`; D-05]

### Core data contracts

1. **`Specification` / `SpecRevision` in Go state:** one lineage per episode; stable requirement and acceptance IDs; statuses `draft`, `approved`, `superseded`; predecessor, content hash, goal/session scope, approval actor/time, and unchanged/added/modified/removed IDs. Keep `.aether/SPEC.md` as a readable, regenerable projection, never as independently parsed authority. [VERIFIED basis: priority analysis §15.2-15.3; D-09..D-12]
2. **`PlanningEvidenceRef`:** typed source kind (`spec`, `survey`, `charter`, `decision`, `context`, `research`, `hive`, `outcome`), stable reference, repository-relative path where applicable, content hash, scope/revision, freshness/admissibility, and excerpt digest. The manifest stores the index; prompts are projections from it. [VERIFIED basis: D-04, D-08, D-14; current snapshot patterns in `cmd/codex_plan.go`]
3. **`PlanningDimensionAssessment`:** dimension, before/after whole-number score, fresh evidence IDs, resolved gap IDs, remaining gap, and rationale. Go checks cited refs exist in this pass, prohibits an increase without fresh applicable evidence, and recomputes the historical weighted overall. [VERIFIED basis: D-04; historical weights at `3a5b81c2...:542-555`]
4. **`PlanningDelta`:** Go-computed semantic changes over stable draft-node IDs: added/changed/removed phases, tasks, dependencies, acceptance checks, negative cases, recovery paths, and public paths; a separate authority-impact list comes from typed spec/decision links. Raw text diffs are drill-down only. [VERIFIED basis: D-03]
5. **`PlanningIterationCard`:** run/iteration/stage IDs, Scout and Route-Setter receipt hashes, new evidence, five score movements, weakest material gap, semantic delta, decision boundary, stop/continue reason, and evidence that would change the next decision. Append immutably and hash the ordered timeline. [VERIFIED basis: D-01..D-04, CEC-03]
6. **`PlanCandidate`:** exact plan content/hash, base plan revision/hash, approved spec revision/hash, planning-run/timeline digest, stop reason, residual gaps/materiality, and expiry/status. Finalization stores this without replacing the active plan. [VERIFIED basis: D-15..D-16]
7. **Accepted `PlanRevision` additions:** approved spec revision ID/hash, candidate ID/hash, timeline location/digest, affected requirement/task/proof IDs, acceptance actor/time, and impact disposition. Legacy accepted plans migrate as `legacy_accepted` without retroactively inventing a spec. [VERIFIED basis: current revision compatibility in `pkg/colony/colony.go:459-481`; D-10..D-12]

### Responsibilities

| Actor/component | Responsibility |
|---|---|
| Scout | Read only, investigate the issued gaps, return evidence refs, resolved/remaining gaps and potential material decisions; never approve intent or mutate state. |
| Route-Setter | Consume the exact Scout receipt, approved spec and prior draft; propose the improved plan, dimension assessments and candidate recovery/negative/public-path coverage. |
| Queen/host wrapper | Present Go-issued preset, decision, iteration and acceptance cards; dispatch only authorized manifests; advise with the persisted recommendation; never compute acceptance or edit state. |
| Go runtime/finalizer | Validate identity/freshness/evidence, classify stage, compute semantic diff/overall/stop eligibility, persist append-only history, gate material decisions/spec approval, and atomically accept an exact candidate. |
| Owner | Select preset, decide material behavior/authority/risk/scope, approve spec revisions, and explicitly accept the exact final plan candidate. |

### Stop, failure, and recovery semantics

- `target_sufficient`, `diminishing_returns`, `stalled`, and `iteration_cap` may produce a candidate only when all residual gaps are classified non-material; `material_decision` pauses without acceptance. Stop and acceptance are separate fields. [VERIFIED: D-15..D-16]
- A later material choice is recorded only after the full Scout → Route-Setter pass, then the candidate stays pending until the answer and any resulting spec revision are approved. [VERIFIED: D-06]
- Replayed stage completions, changed spec/base plan/evidence, wrong worker/stage, stale workspace, path escape, missing citations, score-only changes, or candidate digest mismatch fail before active plan mutation. [VERIFIED extension of `cmd/codex_plan_finalize.go:299-384,693-781`; `cmd/plan_revision.go:184-247`]
- A crash after persisting a Scout result reissues only the next authorized stage; a crash after candidate persistence re-renders the same candidate; a repeated acceptance returns the existing receipt. No worker result is redone merely to reconstruct presentation. [VERIFIED design requirement from D-02/D-16 and current replay guards]
- Routine phase research is issued automatically inside the preset cap. Failure is visible evidence and lowers/limits the appropriate dimension; it does not cause silent template substitution. [VERIFIED: D-14; current fail-loudly pattern in `cmd/phase_research_dispatch_test.go:950`]

### Compatibility and migration

- Decode absent new fields as legacy. Existing accepted plans remain buildable. New planning or a material revision must create/approve a spec and reconcile affected scope; do not invalidate completed legacy work retroactively. [VERIFIED basis: current pointer/omitempty compatibility patterns in `pkg/colony/colony.go`; D-11..D-12]
- Keep current preset numbers, plan revision hashes, completed-prefix protection, evidence containment, real-worker contract and phase-research files. Migrate iteration files into a versioned timeline index rather than deleting them. [VERIFIED: NOW-01..05]
- Keep direct `aether` as runtime plumbing and `/ant-*` as ordinary Claude/OpenCode vocabulary. Do not add Codex-native `$ant-*` in this phase. [VERIFIED: Phase 199 context/synthesis; `200-CONTEXT.md` Deferred Ideas]

### Rejected alternatives

| Alternative | Why considered | Why rejected | Evidence that would reopen it |
|---|---|---|---|
| Restore February’s two Scouts plus synthesis every pass | Strong independent cross-check | D-14 asks autonomous work within a bounded preset, not fixed overstaffing; one targeted Scout plus optional justified escalation preserves the causal loop at lower cost. [VERIFIED: OLD-01; D-14] | Measurements show one Scout systematically misses material gaps and independence is worth the cost for a named preset/risk. |
| Keep current whole-chain manifest | Minimal code change | It cannot enforce the first evidence-backed checkpoint between Scout and Route-Setter. [VERIFIED: NOW-01; D-05] | A proof demonstrates equivalent enforcement without pre-dispatching or preauthorizing Route-Setter. |
| Treat confidence target as plan acceptance | Current behavior is simple | D-16 explicitly separates readiness from owner acceptance. [VERIFIED: D-16] | Only a new owner decision can change this. |
| Make `.aether/SPEC.md` the parsed authority | Human-readable and portable | Free-form Markdown makes stable identity, atomic approval and impact invalidation brittle. [VERIFIED: priority analysis §15.2] | A schema-validated embedded representation proves lossless round trips and atomicity. |
| Restore `semantic-cli.sh` | Classic command name suggests context value | Current survey/context/evidence services provide the useful outcome; restoring shell index authority would duplicate truth. [VERIFIED: CAP-061; AGENTS.md] | Public-path evidence identifies a unique missing retrieval outcome not provided by current services. |
| Preserve research approval card | Existing tested behavior | It violates autonomous routine research after preset selection. [VERIFIED: D-14] | A research action crosses an external/destructive/authority boundary rather than ordinary read-only investigation. |

## 6. Research-to-plan linkage

| Decision ID | Requirement IDs | CAP rows | Planned implementation task(s) | Public-path test/evaluation | Expected owner-visible change |
|---|---|---|---|---|---|
| `SYN-200-01` | SYNTH-02, PLAN-01 | — | Preserve/extend manifest and finalizer invariants | stale/replay/wrong-stage/no-mutation tests | Same safe kernel, now visibly driving the loop |
| `SYN-200-02` | CEC-03, PLAN-01, PLAN-03 | — | Staged iteration state + card/timeline renderer | real two-pass `/ant-plan` corpus case | Each pass says what changed and why |
| `SYN-200-03` | PLAN-02, PLAN-03 | — | Dimension evidence schema and Go scoring validator | unrelated/restated evidence cannot raise score | Five evidence-backed before→after scores |
| `SYN-200-04` | PLAN-04 | CAP-005 | Typed material-decision batch/revalidation | generic/evidence-answerable questions absent; stale answer case | One useful checkpoint, only when needed |
| `SYN-200-05` | PLAN-01, PLAN-02 | CAP-056 | Preset gate + autonomous research policy | no-flags selects; flags bypass; routine research does not pause | Owner chooses cost once; Queen then runs |
| `SYN-200-06` | PLAN-03, PLAN-05 | — | Stop classifier, candidate store, explicit plan acceptance | target/stall/cap/diminishing/material and accept replay | Stop reason is honest; plan waits for owner |
| `SYN-200-07` | PLAN-04, PLAN-05 | — | Go spec types/commands, discuss handoff, `/ant-spec` | draft cannot plan; approval exact; edit makes revision | Readable contract before planning |
| `SYN-200-08` | PLAN-05, PLAN-06 | CAP-012 | Spec impact graph and scoped invalidation | modified requirement invalidates only linked tasks/proof | Feature revisions preserve unaffected work |
| `SYN-200-09` | PLAN-06 | CAP-010, CAP-011, CAP-012 | Route insert-phase through plan revision acceptance | inserted phase inherits evidence and preserves prior revision | Living-plan change explains why it exists |
| `SYN-200-10` | PLAN-02, PLAN-06 | CAP-056, CAP-061, CAP-069 | Typed evidence index and provenance snapshots | inadmissible Hive/stale context rejected or disclosed | Queen can show what grounded planning |
| `SYN-200-11` | PLAN-01, PLAN-03, PLAN-05 | — | YAML/generated/guide/Go renderer alignment | Claude/OpenCode semantic parity | Same Queen story on both primary platforms |
| `SYN-200-12` | SYNTH-02, PLAN-01..06 | all routed CAP rows | Extend Classic semantic corpus/loader | public invocation + state/artifact causality | Restoration is proven, not just styled |

Planning gate:

- [x] Every in-scope requirement has a synthesis decision.
- [x] Every routed CAP row has a proposed disposition backed by old/current evidence.
- [x] No task is justified only by a Classic feature name or screenshot.
- [x] No current Go safety or behavior is replaced without a documented comparative reason.
- [x] Genuine unresolved owner choices are separated from engineering conclusions.

## 7. Verification contract

| Dimension | What must be true | Evidence and public path | Failure/negative case |
|---|---|---|---|
| Outcome | A real repo plan improves over at least two passes, has a readable approved spec and exact accepted plan. | `/ant-discuss` → `/ant-spec` → `/ant-plan`; `V-200-E2E-01` | No spec, unchanged plan dressed as progress, or build eligible before acceptance fails. |
| Behavior | Go issues/validates alternating stages, targets weakest evidenced gaps, computes delta/stop, and attaches the timeline. | `V-200-LOOP-01`, `V-200-CONFIDENCE-01`, `V-200-TIMELINE-01` | Reused packet, unrelated score evidence, skipped stage, or overwritten history fails before mutation. |
| Experience | Preset, material decision, per-pass delta, final candidate, history, gaps, recommendation and Next Up are understandable. | Claude/OpenCode generated public paths; `V-200-PLATFORM-01` | Wrapper-invented facts, generic menu, hidden residual gap, or divergent platforms fails. |
| Safety | Spec/plan/candidate/revision identities and hashes are checked; affected scope only is invalidated; explicit acceptance is required. | `V-200-SPEC-01`, `V-200-ACCEPT-01`, `V-200-REVISION-01`, `V-200-REPLAY-01` | Draft/stale spec, active attempt, path escape, base drift, candidate mismatch, or partial mutation fails. |

Verification gate:

- [x] Selected synthesis defines behavior rather than repeating the phase title.
- [x] Proposed tests cover public paths and causal state transitions, not snapshots alone.
- [x] Negative cases prove prompt/shell authority and automatic acceptance do not return.
- [x] CAP dispositions link to decisions and verification IDs.
- [ ] The phase summary must record final differences from both Classic and pre-phase current behavior after implementation.

## 8. Open decisions and confidence

| Open item | Why evidence cannot decide it | Who has authority | Blocking? | Next evidence/action |
|---|---|---|---|---|
| Exact Go file/type split and whether spec revisions are embedded fully in colony state or referenced from an immutable data file | Both can preserve the locked behavior; this is explicitly delegated implementation detail. [VERIFIED: D-11/D-12 and Agent's Discretion] | Planner/implementer under Go atomicity constraints | No | Prefer the smallest design that makes approval + impact state one atomic transaction; prove crash/replay behavior. |
| Exact numerical diminishing-return threshold | Owner selected the behavior, not the internal threshold. [VERIFIED: D-15; Agent's Discretion] | Planner/implementer | No | Calibrate against historical `<2 after pass 5` and current `<5 twice`; encode as named policy and test boundaries. |
| Bounded artifact-retention mechanics | The owner requires the timeline remain attached but delegated retention mechanics. [VERIFIED: D-02; Agent's Discretion] | Planner/implementer | No | Never remove an accepted plan's timeline index/digest; deduplicate content only with verifiable references. |

**Synthesis confidence:** `high` — all locked behavior is explicit, the three historical refs were inspected at source, the current public/runtime/finalizer/revision paths and tests were traced, and every requirement/CAP row has a disposition. Implementation details listed above remain deliberately open, not owner decisions.

---

*Evidence sources are repository-local or immutable Git refs. No external package or web claim is required for this mechanism study.*
