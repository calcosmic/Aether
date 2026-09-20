# plan -- Iterative Planning and Candidate Contract

**Last verified:** 2026-09-08
**Status:** Phase 200 authoritative contract
**Machine authority:** Go runtime and canonical `.aether/data/` lifecycle records
**Readable projection:** wrappers and terminal renderers only

In plain English: approving a Specification gives Aether permission to plan;
stopping the planning loop creates a proposal to review; accepting that exact
proposal is the only action that makes a new plan active. Those are three
different owner decisions and none implies either of the others.

## Inputs

Planning accepts one approved Specification binding, one explicit quality
preset, and then only the current Go-issued stage manifest plus its one strict
Scout or Route-Setter result. Material decision answers and candidate
acceptance use the exact IDs, hashes, tokens, and commands returned by Go.

## Outputs

Every operation returns a structured state-machine result: the current stage,
state effect, receipt or refusal, exact recovery/next action, and only the next
artifact the caller is authorized to use. Detailed schemas are defined below.

## State Mutations

Preset selection, inspection, candidate review, and refused requests do not
change state. Valid stage finalization appends only Go-validated evidence,
receipts, cards, and candidates. Only exact candidate acceptance may activate
a new PlanRevision; replay and failure rules are defined below.

## Public Spelling

| Platform | Specification | Planning | Candidate review |
|----------|---------------|----------|------------------|
| Codex CLI | `aether spec` | `aether plan` | `aether plan --candidate` |
| Claude Code | `/ant-spec` | `/ant-plan` | `/ant-plan` candidate stage |
| OpenCode | `/ant-spec` | `/ant-plan` | `/ant-plan` candidate stage |

Claude and OpenCode wrappers invoke the same Go operations through the host
adapter. Codex uses direct `aether ...` spelling; there is no Codex-native
`$ant-*` alias. If prose and the structured runtime disagree, the runtime wins.

## Authority Terms

- **Specification approval** means the owner approved one immutable
  Specification revision, identified by revision ID and full content hash, as
  the contract planning must satisfy. It does not stop planning or activate a
  plan.
- **Planning stop** means Go selected one of `target_met`,
  `diminishing_returns`, `stalled_gap`, or `pass_cap` after validating a
  completed pass. It creates a `pending_review` candidate, not an active plan.
- **Candidate acceptance** means the owner accepted one exact stopped proposal
  through its receipt-bound `acceptance_command`. Only its
  `PlanAcceptanceReceipt` may activate a PlanRevision.
- **Receipt** means an immutable Go-issued record binding the exact request,
  inputs, result, and resulting state. A worker report, host confirmation, file
  save, or recommendation is not a receipt.
- **Authority** means permission to change canonical state. Scouts and
  Route-Setters propose evidence and plan content; Go validates it and owns
  every transition.

## Preconditions

Planning requires the current canonical Specification revision to be
`APPROVED`, with a valid approval receipt, readable projection, and reconciled
affected scope. Missing, draft, superseded, projection-drifted, or unreconciled
Specification state dispatches no worker and returns the exact recovery action,
normally `aether spec`, `aether spec --repair-projection`, or the current
reconciliation action.

### Preset selection

With no valid explicit policy, `aether plan` / `aether host plan` returns
`preset_required: true`, `state_effect: none`, and these four unbiased options:

| ID | Label | Target | Pass cap |
|----|-------|--------|----------|
| `fast` | Fast | 80 | 4 |
| `balanced` | Balanced | 90 | 6 |
| `deep` | Deep | 95 | 8 |
| `exhaustive` | Exhaustive | 99 | 12 |

No preset is selected or recommended by default. A wrapper may collect exactly
one owner choice and request `aether host plan --preset
<fast|balanced|deep|exhaustive>`. An exact `--target` plus `--max-iterations`
pair is accepted only when it maps to one preset. The old owner-facing depth
and phase-research approval ceremony is retired; routine read-only research is
automatic inside the selected policy.

## Staged State Machine

The closed stage vocabulary is:

`preset_required`, `scout_ready`, `scout_running`, `owner_decision`,
`spec_approval_required`, `reconciliation_required`, `route_ready`,
`route_running`, `continue_ready`, `candidate_ready`, `accepted`, and `failed`.

Only `scout_running` and `route_running` carry a worker manifest, and each
manifest authorizes exactly one worker. The ordinary path is:

1. `preset_required` -> `scout_ready` -> `scout_running`.
2. Finalize the exact Scout result. Go issues either an owner-decision boundary
   or a receipt-bound Route-Setter authorization.
3. `route_ready` -> `route_running`. Finalize the exact Route-Setter result.
4. Go persists and returns the complete iteration card before choosing
   `continue_ready`, `owner_decision`, or `candidate_ready`.
5. A continuation issues a new Scout manifest for the weakest evidenced gap.
   A candidate remains non-active until exact acceptance moves
   `candidate_ready` -> `accepted`.

The host must never dispatch Scout and Route-Setter as one whole-chain wave,
predict a later stage, reuse a consumed authorization, or treat a worker's
claimed next action as authority.

## Stage Manifest and Finalization Contracts

`planning-stage-manifest/v1` is content addressed. Every manifest requires:

- `id`, `content_hash`, and one-use `authorization_id`
- `run_id`, positive `pass`, selected `preset`
- approved Specification `revision_id`, `content_hash`, status, and approval
  receipt ID/hash
- `base_plan_revision_id` and `base_plan_revision_hash`
- `prior_card_hash` and `input_frontier_hash`
- `expected_caste` and matching `expected_result_type`

A Scout manifest additionally carries the authorized `evidence_frontier` and
one `weakest_gap`; it cannot carry future Route-Setter bindings. A Route-Setter
manifest additionally carries the exact Scout receipt and
`candidate_snapshot_hash`; it cannot carry a fresh Scout evidence frontier.

The finalizer command is:

```text
aether plan-finalize --completion-file <completion-file>
```

The completion packet submits the unchanged current `plan_manifest` plus
exactly one of `scout_result` or `route_result`. Strict decoding rejects legacy
whole-chain worker arrays, combined Scout/Route results, state patches,
acceptance fields, invented next stages, and unknown fields.

### Scout proposal

`planning-scout-result/v1` repeats the manifest ID/hash, run/pass/caste,
Specification and base-plan bindings, and input frontier. It may propose only
`findings`, `new_evidence`, `unresolved_gaps`, and
`material_decision_candidates`. It cannot propose scores, semantic plan
content, stops, candidates, acceptance, activation, or state mutations.

On success Go returns `stage_receipt`, the normalized `scout_artifact`, admitted
evidence, remaining gaps, and exactly one current boundary: `decision_cards`,
`route_stage_manifest`, or `successor_specification` approval/reconciliation.
The StageReceipt binds its ID/hash and request digest to run/pass, manifest
ID/hash, input-frontier hash, output path/hash, caste, prior receipt, resulting
state, and any decision-resume or candidate-snapshot boundary.

### Route-Setter proposal and Go validation

`planning-route-setter-result/v1` repeats every current manifest binding,
including the Scout receipt and candidate snapshot. It may propose plan-shaped
content, proposal evidence IDs, five `dimension_assessments`, and material
decision candidates. It cannot set an overall, stop reason, candidate status,
acceptance, activation, state patch, or next stage.

The five dimensions are exactly `knowledge`, `requirements`, `risks`,
`dependencies`, and `effort`. Each Route-Setter assessment proposes whole-number
`before` and `after` values, applicable `fresh_evidence_ids`,
`resolved_gap_ids`, a typed `remaining_gap`, rationale, and producer receipt.
Go validates evidence freshness/admissibility and the before value, rejects a
supplied or mismatched overall, then derives weighted overall readiness,
semantic delta, weakest gap, and stop policy. A score never rises because prose
was restated or unrelated evidence appeared.

On success Go returns `route_stage_receipt`, proposal hash, the persisted
`iteration_card`, stop policy, and exactly one boundary:
`scout_stage_manifest`, `decision_cards`, `successor_specification`, or
`plan_candidate`.

## Material Owner Decisions

Only behavior, authority, risk tolerance, scope, or acceptance meaning may
require an owner decision. Each decision card identifies the exact decision,
why it is needed now, cited evidence, Queen recommendation, viable choices and
consequences, affected semantic IDs, prior-answer/revalidation status, and the
condition under which planning resumes. Evidence-answerable research,
mechanics, scoring, or formatting questions do not become owner prompts.

The host may collect every exact answer in the current batch. Go binds them to
goal, session, approved Specification, base plan, batch ID/hash, frontier
receipt or completed-card hash, decision/choice IDs, and equivalence hashes.
The closed resolution is:

- `direct_resume`: the answer does not change the approved contract; Go resumes
  the exact previously authorized Scout or Route-Setter boundary.
- `successor_spec_required`: the answer changes behavior, authority, risk,
  scope, acceptance meaning, or affected semantic IDs; Go creates a distinct
  successor `DRAFT`, moves to `spec_approval_required`, and requires exact
  approval plus affected-scope reconciliation before a new Scout can run.

The wrapper never decides which branch applies and never edits the
Specification. A first-pass decision occurs after Scout and before Route-Setter;
a later decision occurs only after the completed iteration card is visible.

## Iteration Cards and Read-Only Detail

Every completed Scout -> Route-Setter pass appends one immutable
`planning-iteration/v2` card. The card binds ID/hash, run and iteration, both
stage receipt IDs/hashes, evidence IDs, all five assessments, Go-derived
overall/target facts, weakest gap, semantic delta, stop decision, and
`evidence_that_would_change`.

Gaps, iteration cards, stop decisions, and plan candidates all carry explicit
`evidence_that_would_change`; below-target uncertainty is never hidden. Read one
card without mutation through:

```text
aether plan --candidate --details --show-iteration <positive-pass>
```

The result operation is `candidate_iteration_detail` and binds `candidate_id`,
`timeline_id`, `timeline_digest`, and the exact card.

## Candidate Review, Queen Recommendation, and Acceptance

`aether plan --candidate` is read-only. It returns `candidate_review`, the
`pending_review` candidate ID/hash and expiry, approved Specification and base
plan bindings, proposal/hash, five scores and derived actual/target readiness,
stop decision, residual gaps, `evidence_that_would_change`, semantic delta,
complete timeline/digest/cards, and `acceptance_command`.

The candidate's recommendation is advice, never authority. It requires a
content-addressed recommendation ID/hash, disposition `accept` or `revise`,
candidate ID, evidence IDs, rationale, producer `queen`, producer ID, and
creation time. A wrapper may render it but may not create or modify it.

After the owner explicitly accepts the exact reviewed candidate, execute the
returned `acceptance_command` verbatim:

```text
aether plan --accept-candidate <candidate-id> --spec-revision <revision-id> --spec-hash <content-hash> --base-plan-revision <revision-id> --timeline-digest <digest> --proposal-hash <hash> --acceptance-token <token>
```

The deprecated `--accept` flag never accepts or activates a plan. Successful
acceptance atomically writes a `plan-acceptance/v1` receipt binding candidate
ID/hash, Specification revision ID/hash, base-plan ID/hash, timeline ID/digest,
proposal hash, hashed acceptance token, owner, time, and activated PlanRevision
ID/hash. Only then may the UI offer guided build and Autopilot as coequal next
actions.

## Replay, Conflict, and Recovery

Every mutating operation is content-addressed and retry-safe:

- An exact retry of stage dispatch, stage output, stage finalization,
  decision-resume, Specification mutation/approval, or candidate acceptance
  returns the existing artifact/receipt with `replayed: true` or an equivalent
  idempotent result. It appends no duplicate state.
- A retry with the same identity but different bytes, hashes, answers,
  frontier, Specification, base plan, proposal, timeline, or token is a
  divergent replay. It fails non-zero, leaves canonical state unchanged, and
  returns the exact read-only recovery/inspection command for the current
  boundary.
- A stale worker result retains completed prior receipts and issues no next
  stage. A stale candidate acceptance leaves the active PlanRevision unchanged
  and routes to `aether plan --candidate`.

The closed execution-authority refusal codes shared by Build and Autopilot are:

`plan_authority_state_unavailable`, `plan_authority_policy_missing`,
`plan_authority_no_active_plan`, `plan_authority_legacy_invalid`,
`plan_authority_specification_not_approved`,
`plan_authority_stale_specification`,
`plan_authority_candidate_not_accepted`, `plan_authority_candidate_invalid`,
`plan_authority_stale_base`, `plan_authority_broken_timeline`,
`plan_authority_acceptance_invalid`, and `plan_authority_affected_scope`.

Each refusal carries `eligible: false`, `state_effect: none`, a diagnostic, and
an exact `recovery_command`. Renderers may translate the explanation but must
not infer or rewrite the code or command.

## Host Boundary

Wrappers may render structured Go results, collect one selected preset, collect
the exact owner answers for the current decision batch, dispatch only the one
authorized worker, and execute a runtime-returned exact command. They may not:

- edit `COLONY_STATE.json`, `.aether/SPEC.md`, canonical Specification or plan
  artifacts, transaction journals, projections, or active revision pointers
- derive or alter stable IDs, hashes, evidence admissibility, scores, overall
  readiness, gaps, semantic deltas, materiality, stop reasons, or next actions
- mint or alter stage/approval/acceptance receipts, decision tokens, Queen
  recommendations, candidate status, or acceptance commands
- treat Specification approval, preset selection, worker completion, planning
  stop, generic confirmation, or candidate readiness as plan acceptance

See `.aether/docs/wrapper-host-contract.md` for the shared host envelope and
wrapper obligations.
