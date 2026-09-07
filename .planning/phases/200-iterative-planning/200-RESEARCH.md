# Phase 200: Iterative Planning - Research

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

#### Iteration Presentation

- **D-01:** After every completed Scout → Route-Setter pass, `/ant-plan` shows a compact delta card containing fresh evidence, before→after confidence, the weakest remaining gap, semantic plan changes, and the explicit continue/stop reason. Full citations and detailed diffs remain available on demand rather than flooding the default view.
- **D-02:** Completed iteration cards append to a persistent, ordered timeline and remain attached to the accepted plan. Later passes may not overwrite or hide the visible history already persisted by the Go finalizer.
- **D-03:** Plan deltas are semantic rather than raw-text diffs. They enumerate added, changed, or removed phases, tasks, dependencies, acceptance checks, and recovery paths, and separately flag any proposed change that crosses an owner-authority boundary.
- **D-04:** Knowledge, requirements, risks, dependencies, and effort appear as whole-number before→after planning-readiness scores. Every changed dimension cites the fresh evidence that justified the movement and names its remaining gap; a score may not rise merely because prior evidence was restated.

#### Owner Decision Boundaries

- **D-05:** Known material owner decisions are consolidated into one evidence-first checkpoint after Scout completes the first grounded pass. Planning must not begin with a generic category menu or ask questions that repository, survey, charter, Hive, outcome, or research evidence can answer.
- **D-06:** If a later pass discovers a genuinely new material decision, the current Scout → Route-Setter pass completes so the impact can be explained, then planning pauses at that pass boundary before an unauthorized revision is accepted.
- **D-07:** Every material-decision card states the decision, why it matters now, cited evidence, the Queen's recommendation, consequences of each viable choice, and what work resumes after the answer. The Queen advises rather than merely forwarding neutral alternatives.
- **D-08:** A prior owner answer is reused only while its goal, meaning, and consequences remain equivalent. If fresh evidence changes behavior, scope, risk, or acceptance impact, planning shows the prior answer and requests revalidation; answers remain bound to goal/session/revision and never become stale universal permission.

#### Specification Lifecycle

- **D-09:** The normal guided journey hands off automatically from resolved `/ant-discuss` intent to a readable draft specification. `/ant-spec` remains a real public Claude/OpenCode command for opening, editing, approving, or revising that artifact later; it is not an empty presentation-only wrapper.
- **D-10:** A new or materially revised specification remains `draft` until the owner explicitly approves it. Only an approved revision may become the contract for a newly accepted plan; neither inferred intent nor resolved internal questions silently grants approval.
- **D-11:** Each colony goal has one canonical specification lineage. Feature-only work creates a scoped revision of identified requirements and acceptance criteria while preserving unaffected IDs and content; it does not create overlapping standalone sources of truth.
- **D-12:** A material specification edit creates an immutable revision linked to its predecessor, classifies unchanged/added/modified/removed requirements, preserves historical evidence, and invalidates only affected plan tasks and proof links. The changed contract must be approved and the affected plan scope reconciled before work or Seal may rely on it.

#### Research Autonomy and Stop Policy

- **D-13:** When `/ant-plan` is invoked without explicit quality flags, the owner selects Fast, Balanced, Deep, or Exhaustive. The card exposes each existing preset's confidence target and iteration cap; there is no silent Queen-selected or fixed Deep default. Explicit valid flags continue to bypass this choice.
- **D-14:** Once the owner selects a preset, Scout and Route-Setter work autonomously within that agreed budget. Each later Scout pass targets the weakest evidenced gaps and the Queen explains why it ran; routine research does not need approval, while new material authority or safety boundaries still pause under D-06.
- **D-15:** Planning stops honestly for target sufficiency, diminishing returns, detected stall, or the selected iteration cap. A below-target draft may close automatically when every remaining gap is non-material, provided the timeline discloses the gaps, stop reason, and evidence that could change the next decision; a material gap produces the owner-decision card and pauses instead.
- **D-16:** The final plan becomes eligible for guided build or Autopilot only after explicit owner acceptance. The Queen presents the final plan, complete confidence history, remaining gaps, and recommendation first; specification approval does not double as plan acceptance.

### The Agent's Discretion

No owner-visible behavior was delegated. Research and planning may choose internal Go types, helper boundaries, the machine-readable SPEC index representation, exact terminal spacing/color, and bounded artifact-retention mechanics, provided they preserve the decisions above, thin-wrapper/runtime authority, immutable history, cross-platform semantic parity, and accessible drill-down evidence.

### Folded Todos

- `2026-08-20-spec-builder-feature.md` — The owner requested a user-facing flow that turns intent and decisions into a plain-language specification before building. Phase 200 now owns the real `/ant-spec` runtime contract, approval/revision lifecycle, automatic post-discuss handoff, owner-checkable acceptance, and plan traceability described in D-09 through D-12.

### Deferred Ideas (OUT OF SCOPE)

- Queen-led work execution, team sizing, verification boundaries, recovery, and worker-turnaround improvements belong to Phase 201.
- Substantive live Swarm/Watch/Oracle behavior belongs to Phase 202.
- Causal live pheromone influence and TypeScript-host preflight/timeout handling belong to Phase 203.
- Learning governance and outcome-backed promotion belong to Phase 204; Codex-native `$ant-*` lifecycle skills remain a later milestone.

#### Reviewed Todos (not folded)

- `2026-08-27-worker-turnaround-is-too-slow.md` — The one-plan-job latency and cost problem belongs to Phase 201's Queen-led work-cycle and attempt model, not planning synthesis.
- `2026-08-01-ts-host-preflight-hardcoded-timeout.md` — Host preflight timeout/configuration and repository-CWD handling remain routed to Phase 203; the keyword-only todo match does not change ownership.
</user_constraints>

**Researched:** 2026-09-07  
**Domain:** Go-authoritative iterative planning, specification identity, and evidence causality  
**Confidence:** HIGH — immutable Git refs, current code/tests, and binding decisions agree. [VERIFIED: `200-CLASSIC-SYNTHESIS.md`]

## Summary

Current Go already supplies the safety kernel: exact Scout/Route-Setter manifests, run and iteration identity, four presets, five scores, evidence hashes, persistent intermediate artifacts, replay/stale/workspace checks, atomic plan revisions, completed-prefix protection, and scoped clarification records. [VERIFIED: `cmd/codex_plan.go:23-298,1024-1278,1950-2145`; `cmd/codex_plan_finalize.go:299-1003`; `cmd/plan_revision.go:15-357`; `cmd/discuss.go:1-381,704-851`]

What is missing is a typed causal layer. The finalizer cannot pause between first Scout evidence and Route-Setter; score movement is not tied per dimension to applicable fresh evidence; accepted plans do not attach the ordered pass history; terminal stops immediately activate a plan; and there is no approved specification lineage. [VERIFIED: `cmd/codex_plan_finalize.go:401-564,693-930`; `pkg/colony/colony.go:422-481`; repository search for `ant-spec` and `SpecRevision`]

Implement four connected Go-owned concepts: immutable specification revisions, staged planning iterations, append-only evidence/delta cards, and exact plan candidates accepted in a separate transaction. Keep Claude/OpenCode wrappers as strong Queen presentation and dispatch surfaces, never as persistence or approval authority. [VERIFIED basis: `200-CONTEXT.md` D-01..D-16; `200-CLASSIC-SYNTHESIS.md` SYN-200-01..12]

For dummies: Scout brings receipts, Route-Setter changes the route, Go checks that the receipts explain the change, and the owner separately signs the destination contract and the exact route.

**Primary recommendation:** split a planning pass into Go-authorized Scout and Route-Setter stages, persist one validated semantic delta card after the route stage, stop into a non-active candidate, and activate only an exact candidate bound to an approved SPEC revision. [VERIFIED basis: `200-CLASSIC-SYNTHESIS.md` §5]

<phase_requirements>
## Phase Requirements

| ID | Required outcome | Research support |
|---|---|---|
| `SYNTH-02` | Reconstruct Classic evidence/scoring/gap/delta/stop/owner behavior against current Go. | Exact old/current mechanism and disposition study in `200-CLASSIC-SYNTHESIS.md`. [VERIFIED: synthesis §§2-4] |
| `CEC-03` | Each iteration, choice, and stop explains why and what evidence changes the next decision. | Typed evidence refs, delta card, reason code, and `evidence_that_would_change` field. [VERIFIED basis: D-01, D-07, D-14..D-15] |
| `PLAN-01` | Visible alternating loop over modern Go authority. | Scout-stage → Go checkpoint → Route-stage → Go finalizer. [VERIFIED basis: SYN-200-01/02/11] |
| `PLAN-02` | Five scored dimensions and weakest-gap targeting. | Per-dimension assessment, Go-recomputed weighted overall, material gap ranking. [VERIFIED basis: SYN-200-03/10] |
| `PLAN-03` | Explain change, evidence, and stop. | Semantic delta, append-only timeline, distinct stop taxonomy. [VERIFIED basis: SYN-200-02/03/06] |
| `PLAN-04` | Batch only material choices and scope answers. | First-pass evidence batch, later boundary pause, equivalence/revalidation key. [VERIFIED basis: SYN-200-04] |
| `PLAN-05` | Readable spec and accepted grounded plan with negative/recovery/public paths. | Go spec lineage, richer plan artifact, exact candidate acceptance and trace links. [VERIFIED basis: SYN-200-06..08/10] |
| `PLAN-06` | Evidence priming and safe insert/revision history, scope, and reason. | Typed evidence index and impact-scoped accepted PlanRevision. [VERIFIED basis: SYN-200-08..10] |
</phase_requirements>

## Project Constraints (from AGENTS.md)

- Go in `cmd/` owns validation, state transitions, and typed output; wrappers orchestrate and explain. [VERIFIED: `AGENTS.md` UX Architecture]
- Canonical `.aether/commands/*.yaml`, generated Claude/OpenCode commands, the Codex guide/skill, and `cmd/command_guide.go` must remain semantically aligned. [VERIFIED: `AGENTS.md` Codex Orchestration Layer]
- Never restore prompt/shell writes as canonical authority; `.aether/data/` is local runtime state. [VERIFIED: `AGENTS.md` Key Directories]
- Use real worker receipts; planned or spawned work is not completed work. [VERIFIED: `cmd/contracts/plan.md:93-128`]
- Public output explains what changed, why it matters, and what it means in plain English. [VERIFIED: root/global `AGENTS.md` Communication Style]
- Repository gates are `go test ./...` and `go test ./... -race`. [VERIFIED: `AGENTS.md` Quick Reference]
- Do not broaden into Phases 201–204 or later Codex-native lifecycle skills. [VERIFIED: `200-CONTEXT.md` Deferred Ideas]

## Architectural Responsibility Map

| Capability | Primary tier | Secondary tier | Why |
|---|---|---|---|
| SPEC lineage, approval, impact | Go state/domain | Markdown projection | Identity and execution eligibility require one authority. [VERIFIED basis: D-09..D-12] |
| Stage issue/finalization | Go coordinator | Host dispatcher | Go already validates manifest, workers, evidence, and state. [VERIFIED: `cmd/contracts/plan.md`] |
| Research and route proposal | Scout/Route-Setter | Go validator | Workers investigate/propose; they do not authorize. [VERIFIED: `cmd/codex_plan.go:1391-1658,2240-2340`] |
| Timeline and candidate | Go artifact/state | Visual renderer | Ordering, replay, attachment, and acceptance need durable identity. [VERIFIED basis: D-01..D-04, D-16] |
| Owner experience | Claude/OpenCode wrappers | Go visual/JSON | Wrappers present runtime-issued cards and actions. [VERIFIED: `.aether/commands/plan.yaml`] |
| Insert/revision | Go revision tier | SPEC impact mapper | Reuse current atomic safety; narrow invalidation by trace links. [VERIFIED: `cmd/plan_revision.go`; D-12] |

## Standard Stack

| Component | Version/source | Use |
|---|---|---|
| Go | `go 1.26.5` | Typed contracts, hashes, validators, transactions. [VERIFIED: `go.mod`] |
| Cobra / pflag | `v1.10.2` / `v1.0.9` | Public `aether spec` and internal stage/accept actions. [VERIFIED: `go.mod`; `cmd/codex_workflow_cmds.go`] |
| Repository JSON store | existing package | Locked/atomic colony state changes. [VERIFIED: `cmd/codex_plan_finalize.go:463-495`; `cmd/state_extra.go:389-511`] |
| SHA-256 + `encoding/json` | standard library | Evidence, candidate, revision, and timeline identity. [VERIFIED: `cmd/codex_plan.go`; `cmd/plan_revision.go`] |
| Canonical YAML generator | repository-native | Primary Claude/OpenCode command projections. [VERIFIED: `RUNTIME UPDATE ARCHITECTURE.md`] |

No external dependency or install is needed; use existing survey snapshots, charter/intent renderers, phase research, colony-prime/Hive inputs, and the Phase 199 Classic corpus. [VERIFIED: `cmd/codex_workflow_cmds.go:119-167`; `cmd/discuss.go`; `cmd/phase_research_*`; `cmd/testdata/classic-contract/v1/`]

## Architecture Patterns

### Authoritative flow

```text
owner resolves /ant-discuss
  -> Go creates/updates draft SPEC
  -> owner reviews and explicitly approves exact SPEC revision
  -> /ant-plan selects preset unless valid flags already supplied
  -> Go issues Scout-stage manifest
  -> Go validates Scout receipt
       -> first-pass material batch, or Route-Setter-stage manifest
  -> Go validates Route-Setter receipt
  -> Go computes dimension movement + semantic delta + timeline card
       -> next Scout targets weakest material gap
       -> material decision pauses at completed pass boundary
       -> honest stop persists a non-active PlanCandidate
  -> Queen presents exact candidate, history, gaps, recommendation
  -> owner explicitly accepts candidate
  -> Go atomically activates PlanRevision bound to SPEC + timeline
```

The whole-chain completion packet must become stage-scoped because it cannot enforce D-05 after first Scout evidence but before Route-Setter. Each next-stage manifest should bind the prior receipt hash and the same run/goal/root/spec/base-plan identity. [VERIFIED: `cmd/codex_plan_finalize.go:299-384,693-768`; D-05]

### Core contracts

| Contract | Required fields and invariant |
|---|---|
| `Specification` / `SpecRevision` | One goal lineage; stable requirement/acceptance IDs; `draft/approved/superseded`; predecessor/content hash/scope/approval; classified requirement delta. Markdown is regenerable projection, not authority. [VERIFIED basis: priority analysis §15.2-15.3; D-09..D-12] |
| `PlanningEvidenceRef` | Source kind, stable ref, repo-relative path when applicable, hash, scope/revision, freshness/admissibility, excerpt digest. Prompts project this index. [VERIFIED basis: D-04/D-08/D-14] |
| `PlanningDimensionAssessment` | Dimension, whole-number before/after, fresh evidence IDs, resolved gaps, remaining gap, rationale. Go validates applicability and recomputes historical weighted overall. [VERIFIED basis: D-04; `3a5b81c2:.claude/commands/ant/plan.md:542-555`] |
| `PlanningDelta` | Go-computed added/changed/removed phases, tasks, dependencies, checks, negative/recovery/public paths over stable semantic IDs; authority impacts separate. [VERIFIED basis: D-03] |
| `PlanningIterationCard` | Run/pass/stage IDs, receipt hashes, new evidence, five movements, weakest material gap, semantic delta, boundary, reason, evidence needed next. Append and hash ordered timeline. [VERIFIED basis: D-01..D-04, CEC-03] |
| `PlanCandidate` | Exact plan/hash, base revision/hash, approved spec ID/hash, run/timeline digest, stop reason, residual gaps/materiality, status. It does not replace active plan. [VERIFIED basis: D-15..D-16] |
| accepted `PlanRevision` additions | Spec/candidate/timeline bindings, affected requirement/task/proof IDs, owner receipt, impact disposition. Legacy stays explicit rather than backfilled. [VERIFIED basis: `pkg/colony/colony.go:459-481`; D-10..D-12] |

### Enforcement rules

- Per-dimension score increases require fresh applicable evidence; restatement cannot raise readiness and scores may fall. Keep the global evidence digest as an additional replay guard. [VERIFIED basis: D-04; current coarse guard `cmd/codex_plan_finalize.go:771-781`]
- Rank gaps by materiality then weakness and deterministic tie-break, never by alphabetic order alone. [VERIFIED: current alphabetic limit `cmd/codex_plan.go:206-216`; D-05/D-15]
- `target_sufficient`, `diminishing_returns`, `stalled`, and `iteration_cap` may create a candidate only with non-material residual gaps; `material_decision` pauses without acceptance. [VERIFIED basis: D-15/D-16]
- A later material decision is shown after the completed pass; any resulting material SPEC change creates a successor draft and invalidates the pending candidate. [VERIFIED basis: D-06/D-12]
- Replayed/wrong-stage/stale packets, changed spec/base/evidence, path escape, missing citations, score-only changes, or candidate mismatch fail before active-plan mutation. [VERIFIED extension of `cmd/codex_plan_finalize.go:299-384,693-781`; `cmd/plan_revision.go:184-247`]
- A crash after Scout resumes at the next authorized stage; after candidate persistence it re-renders the same candidate; repeated exact acceptance is idempotent. [VERIFIED basis: D-02/D-16 and current replay guards]
- Preset selection authorizes ordinary read-only research inside its cap; phase-research failure is disclosed and limits readiness rather than silently substituting content. [VERIFIED basis: D-14; `cmd/phase_research_dispatch_test.go`]

### Current seams to reuse

| Seam | Reuse / change |
|---|---|
| `cmd/codex_plan.go` | Retain manifests, presets, run identity, snapshot/context/phase research; add stage and typed evidence/draft IDs. [VERIFIED: current code] |
| `cmd/codex_plan_finalize.go` | Retain validation/persistence/replay; split stage advancement, compute card/stop/candidate, remove immediate activation. [VERIFIED: current code] |
| `cmd/plan_revision.go` | Retain hashes, atomicity, stale-base, active-attempt and completed-prefix guards; add exact candidate/spec/impact binding. [VERIFIED: current code/tests] |
| `cmd/discuss.go` | Retain goal/session scope and stale quarantine; replace generic planning questions with evidence-backed material batches and draft handoff. [VERIFIED: current code] |
| `cmd/state_extra.go` | Route `insert-phase` through revision/candidate machinery; avoid direct renumbering mutation. [VERIFIED: `cmd/state_extra.go:360-525`] |
| YAML/renderers/generator | Present runtime-issued preset/decision/card/spec/candidate states identically on Claude/OpenCode. [VERIFIED: AGENTS.md; Phase 199 contract] |

## Classic Lessons and Dispositions

- Restore the causal rhythm and named presets, not February’s fixed dual-Scout staffing or April’s silent Deep default. [VERIFIED: `200-CLASSIC-SYNTHESIS.md` OLD-01, OLD-07..09, SYN-200-02/05]
- Keep current Go replay, identity, evidence, atomicity, and completed-work protection. [VERIFIED: synthesis NOW-01..06, SYN-200-01]
- Replace self-reported confidence, wrapper authority, destructive research replacement, generic menus, and automatic acceptance with typed modern contracts. [VERIFIED: synthesis SYN-200-03/04/06/10]
- Add the missing spec lineage; Classic had no public `spec` command at the inspected refs. [VERIFIED: synthesis OLD-11/NOW-08/SYN-200-07]
- Preserve CAP-010..012 living-plan value through accepted revisions; treat CAP-005 as evidence-backed material decisions, CAP-056 as attributable Hive input, CAP-061 as typed current context rather than `semantic-cli.sh`, and CAP-069 as versioned context evidence. [VERIFIED: synthesis §6; capability ledger]

## Don't Hand-Roll

| Problem | Do not build | Use instead |
|---|---|---|
| State safety | Wrapper/file rewrites | Existing locked atomic store and Go validation. [VERIFIED: current revision/finalizer] |
| Evidence identity | Prompt citations alone | Stable typed refs plus SHA-256 digests. [VERIFIED: current evidence/snapshot patterns] |
| Plan diff | Raw Markdown diff as semantics | Stable nodes and Go structural comparison. [VERIFIED basis: D-03] |
| Acceptance | Prose “yes”, target, cap, or stale flag | Exact runtime-issued candidate/spec token and current hashes. [VERIFIED basis: D-10/D-16] |
| Impact | Replace every unfinished task | Requirement→task→proof trace links and affected closure. [VERIFIED basis: D-12] |
| Context retrieval | Restored `semantic-cli.sh` | Existing survey/context/research/Hive services wrapped as evidence. [VERIFIED: CAP-056/061/069] |

## Common Pitfalls

1. **Stop equals acceptance:** target/stall/cap currently activates immediately. Persist a candidate first and prove build/Autopilot refuse it. [VERIFIED: `cmd/codex_plan_finalize.go:401-564`; D-16]
2. **Unrelated evidence inflates a score:** global hash change is too coarse. Validate evidence applicability per dimension. [VERIFIED: `cmd/codex_plan_finalize.go:771-781`; D-04]
3. **Decision arrives at the wrong boundary:** first-pass choices must precede Route-Setter; later choices follow the completed pass. Test both transitions. [VERIFIED basis: D-05/D-06]
4. **Markdown becomes dual authority:** free-form SPEC is readable but not atomic. Store canonical schema in Go and regenerate projection. [VERIFIED basis: D-09..D-12]
5. **Legacy migration invents consent:** mark old accepted plans `legacy_accepted`; never synthesize approval/evidence timestamps. [VERIFIED basis: compatibility requirement]
6. **Task renumbering breaks links:** retain ordinal execution IDs but add immutable semantic IDs for diff/impact. [VERIFIED: `cmd/plan_revision.go:360-385`; `cmd/state_extra.go:451-466`]
7. **Research cleanup erases history:** content-address iteration/card records and retain every accepted timeline digest. [VERIFIED: D-02; destructive Classic behavior `v5.4:.aether/commands/plan.yaml:177-185`]
8. **Routine research asks twice:** remove the phase-research approval gate after preset choice; pause only for material authority/safety. [VERIFIED: current proposal flow; D-14]
9. **Wrapper and runtime diverge:** one typed `next_action` and stage token must drive both generated platforms. [VERIFIED: Phase 199 projection pattern]

## Likely Files to Modify or Create

| Area | Expected work |
|---|---|
| `pkg/colony/colony.go` or new spec domain file | Spec lineage, candidate/timeline/revision bindings, legacy decode. |
| `cmd/codex_plan.go`, `cmd/codex_plan_finalize.go` | Staged manifest, evidence/gap/draft contracts, scoring, semantic delta, stop/candidate persistence. |
| `cmd/plan_revision.go`, `cmd/state_extra.go` | Exact candidate activation and impact-scoped revision/insert safety. |
| `cmd/discuss.go`, pending-decision helpers, new `cmd/spec*.go` | Material batch scope and real draft/open/edit/approve/revise lifecycle. |
| `cmd/codex_visuals.go`, closeout/next action | Preset, delta/timeline, decision, spec, candidate, and acceptance projections. |
| `.aether/commands/{plan,discuss,spec}.yaml`, generated commands, guide/docs | Thin cross-platform orchestration and parity. |
| `cmd/testdata/classic-contract/v1/*`, co-located tests | Phase 200 decisions/groups and causal public/state cases. |

These are implementation seams, not a mandate for exact file splitting; the owner delegated helper/type layout. [VERIFIED: Agent's Discretion]

## Schema and Migration Guidance

1. Version new shapes (`specification/v1`, `planning-evidence/v1`, `planning-iteration/v2`, `plan-candidate/v1`); reject unknown future major versions and treat absent fields as legacy. [VERIFIED basis: existing AcceptedCharter/Classic corpus versioning]
2. Broaden the Classic corpus schema to enumerate `SYN-200-*` and new planning/spec/living-plan groups without invalidating Phase 199 fixtures. [VERIFIED: current `schema.json`/`mechanisms.json`]
3. Do not mutate historical PlanRevision snapshots; optional bindings appear only on new revisions. [VERIFIED basis: current `omitempty` compatibility; D-02]
4. Index/digest existing iteration artifacts as `legacy_unbound`; never invent per-dimension citations. [VERIFIED basis: D-02/D-04]
5. Existing active legacy plans stay buildable; a new plan or material revision must use approved current contracts. [VERIFIED basis: D-10..D-12 and legacy compatibility]
6. Approval and acceptance are stale-base/replay safe; regenerating a failed projection must not change canonical receipts. [VERIFIED basis: current replay/stale patterns]

## Validation Architecture

Nyquist automation is disabled, but Phase 200 explicitly requires a concrete validation design. [VERIFIED: `.planning/config.json`; phase research contract]

| Property | Value |
|---|---|
| Framework | Go `testing`, existing package/black-box helpers. [VERIFIED: `cmd/*_test.go`] |
| Focused | `go test ./cmd -run 'Test(Plan|Spec|Discuss|PhaseInsert|ClassicContract200)' -count=1` |
| Domain | `go test ./pkg/colony ./cmd -count=1` |
| Full | `go test ./...` |
| Race | `go test ./... -race` |

### Requirement-to-test map

| Req | Behavior | Test type / command | Gap |
|---|---|---|---|
| `SYNTH-02` | Corpus maps old/current mechanisms to decisions. | fixture/schema: `go test ./cmd -run TestClassicContract -count=1` | Extend loader/schema. |
| `CEC-03` | Every pass/stop has causal evidence. | unit/black-box: `go test ./cmd -run TestPlanningIterationCard -count=1` | New card suite. |
| `PLAN-01` | Exact staged visible loop. | integration: `go test ./cmd -run TestPlanStagedLoop -count=1` | Adapt chain tests. |
| `PLAN-02` | Five cited dimensions and material-gap order. | table: `go test ./cmd -run TestPlanningConfidenceEvidence -count=1` | Extend evidence tests. |
| `PLAN-03` | Semantic delta and distinct stop reasons. | unit/renderer: `go test ./cmd -run 'TestPlanning(SemanticDelta|StopPolicy)' -count=1` | New matrices. |
| `PLAN-04` | First batch, later boundary, revalidation. | integration: `go test ./cmd -run TestPlanningMaterialDecision -count=1` | Extend discuss tests. |
| `PLAN-05` | Discuss→draft→approve→candidate→accept. | black-box/state: `go test ./cmd -run 'TestSpec|TestPlanCandidateAcceptance' -count=1` | New spec/accept suites. |
| `PLAN-06` | Provenance, scoped impact, safe insert/revision. | integration/fault: `go test ./cmd -run 'Test(PlanImpact|PhaseInsert|PlanRevision)' -count=1` | Extend revision tests. |

### Mandatory matrices

- Stage: initial Scout, material/no-decision first checkpoint, Route-Setter, later decision, next gap, every stop, research/provider failure, interruption, replay. [VERIFIED basis: D-05/D-06/D-14/D-15]
- Confidence: relevant fresh evidence, restatement, decrease, unrelated/missing/removed evidence, supplied-overall mismatch, gap tie/materiality. [VERIFIED basis: D-04]
- SPEC: draft, exact approval, successor draft, scoped add/modify/remove, stale token, malformed ID, unknown schema, projection failure/repair, legacy. [VERIFIED basis: D-09..D-12]
- Candidate/revision: every stop, material pause, build refusal, exact/stale/replayed acceptance, atomic failure, completed/unaffected preservation, affected-only invalidation, active-attempt refusal. [VERIFIED basis: D-12/D-15/D-16]
- Platform: discuss handoff, all spec operations, preset select/flag bypass, autonomous research, pass cards/drill-down, explicit acceptance, Claude/OpenCode parity. [VERIFIED basis: D-09/D-13/D-14]

### Semantic public-path proofs

- `V-200-E2E-01`: discuss → draft → approval → two passes → candidate → exact acceptance → READY plan bound to SPEC/timeline.
- `V-200-CONFIDENCE-01`: repeated or unrelated evidence cannot raise a dimension.
- `V-200-DECISION-01`: evidence-answerable questions are absent and a material batch pauses at the correct stage.
- `V-200-STOP-01`: each reason is distinct and residual gaps remain visible.
- `V-200-ACCEPT-01`: no stopped candidate is buildable before explicit acceptance.
- `V-200-REVISION-01`: one requirement change invalidates only linked tasks/proof.
- `V-200-REPLAY-01`: stale/replayed stage, approval, and acceptance cause zero canonical mutation.
- `V-200-PLATFORM-01`: generated Claude/OpenCode paths drive equal Go transitions and never write state.

### Sampling and Wave 0

- Per task: named focused test; per wave: domain suite plus corpus/parity; phase gate: full and race suites plus one real-repository two-pass Claude and OpenCode walkthrough. [VERIFIED: roadmap success criterion 2; AGENTS.md gates]
- Wave 0: deterministic stage receipts; goal/session/survey/Hive/spec fixtures; expanded Classic schema/loader; state digest helpers for zero-mutation checks; generated-wrapper parity fixture. [VERIFIED basis: current test gaps]

## Environment Availability

No external service/package is required. Verify the existing Go toolchain and command generator at execution start; no install checkpoint belongs in the plan. [VERIFIED: `go.mod`; selected architecture]

## Assumptions Log

| # | Claim | Risk |
|---|---|---|
| — | None; recommendations derive from locked decisions and repository evidence. | — |

## Open Questions

1. **SPEC body storage:** embed revisions in colony state or index immutable bodies. Prefer the design that makes approval plus impact one atomic transaction and prove crash/replay behavior. [VERIFIED scope: Agent's Discretion]
2. **Diminishing threshold:** historical `<2 after pass 5` and current `<5 twice` are inputs, not a locked answer. Encode a named policy using novelty/materiality and table-test it. [VERIFIED: `3a5b81c2:.claude/commands/ant/plan.md:428-456`; `cmd/codex_plan_finalize.go:820-860`]
3. **Semantic versus ordinal IDs:** retain ordinals for build compatibility; add immutable semantic IDs for trace/diff/impact. [VERIFIED: `cmd/contracts/plan.md:130-176`; `cmd/plan_revision.go:360-385`]

None blocks planning because exact internal representation and thresholds are delegated while the observable/safety contracts are locked. [VERIFIED: Agent's Discretion]

## Sources

- Immutable Classic: `3a5b81c2:.claude/commands/ant/plan.md`, `v5.0.0:.claude/commands/ant/plan.md`, `v5.4:.aether/commands/plan.yaml`, generated `v5.4:.claude/commands/ant/plan.md`.
- Authority: `200-CONTEXT.md`, `.planning/{REQUIREMENTS,ROADMAP,STATE,PROJECT}.md`, capability ledger, synthesis template.
- Current: `cmd/codex_plan.go`, `cmd/codex_plan_finalize.go`, `cmd/plan_revision.go`, `cmd/discuss.go`, `cmd/state_extra.go`, `pkg/colony/colony.go`, command YAML and tests.
- Governing studies: comprehensive review, priority analysis §§15-16, spec-builder todo, Phase 199 context/synthesis, Classic corpus fixtures.

No web/training source or external package claim was used. [VERIFIED: cited repository source set]

## Metadata

- Standard stack: HIGH — existing module/runtime only. [VERIFIED: `go.mod`]
- Architecture: HIGH — locked behavior and directly observed seams agree. [VERIFIED: context and current code]
- Historical reconstruction: HIGH — all required immutable refs inspected. [VERIFIED: `200-CLASSIC-SYNTHESIS.md`]
- Validation: HIGH — current safety tests plus explicit new gaps. [VERIFIED: co-located tests and validation map]

**Research date:** 2026-09-07  
**Valid until:** Phase 200 implementation or a change to its locked decisions/current plan kernel.
