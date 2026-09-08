# spec -- Specification Lifecycle Contract

**Last verified:** 2026-09-08
**Status:** Phase 200 authoritative contract
**Schema:** `spec-command/v1`
**Authority:** canonical Go state; `.aether/SPEC.md` is a readable projection

In plain English: the Specification is the owner's signed brief. Editing it
creates a new draft; approving the exact draft allows planning against it. It
does not accept a plan, activate work, or make a candidate buildable.

## Inputs

`aether spec` accepts exactly one operation: inspect; add, modify, or remove one
typed item; approve one exact revision; or repair the readable projection.
Mutation inputs bind the section, stable identity or lineage, content/evidence,
scope, predecessor revision/hash, or approval revision/hash/token described
below.

## Outputs

Every successful operation returns the shared `spec-command/v1` result with the
complete nine-part body, immutable revision identity, classified delta,
affected scope, projection standing, receipt when issued, replay status, and
exact next action.

## State Mutations

Inspect never mutates. Add, modify, and remove create one immutable successor
draft; approve records authority for exactly one current draft; repair rewrites
only the readable projection. Exact replay is idempotent, while every refused
or divergent request leaves canonical Specification and Plan state unchanged.

## Preconditions

A canonical Specification must exist. Revision operations require the named
current predecessor and valid typed inputs; approval requires the exact current
draft revision ID, full content hash, and Go-issued approval token. File-backed
text must resolve to a contained, regular, non-empty repository file.

## Public Spelling

Codex uses direct `aether spec` commands. Claude Code and OpenCode expose the
same operations through `/ant-spec`. Wrappers may collect one explicit owner
operation and render its structured result; they do not gain authority from
their platform spelling.

## Typed Body and Readable Projection

Every revision preserves these nine categories separately and in this order:

1. `outcomes` — what the goal delivers
2. `included_behaviors` — behavior inside the contract
3. `exclusions` — explicit non-goals
4. `binding_decisions` — choices later work must honor
5. `requirements` — stable obligations
6. `acceptance_checks` — owner-checkable facts plus verification text
7. `negative_expectations` — behavior that must not occur
8. `recovery_expectations` — required failure/recovery behavior
9. `affected_public_paths` — externally visible paths and meanings

Each item has a Go-issued stable ID and content identity. The structured result
keeps each category as its own typed array; a renderer must not collapse them
into a generic list. `.aether/SPEC.md` renders the same complete owner-readable
body, revision lineage, status, scope, and impact, but is never parsed back to
make an authority decision.

## Shared Result

Every successful operation returns:

- `schema_version`, `command`, `operation`, `result_kind`, `outcome_kind`, and
  `state_effect`
- `specification_id`, `before_revision_id`, `after_revision_id`,
  `revision_number`, full `content_hash`, `status`, and `scope`
- all nine typed body arrays
- `classified_delta` with added, modified, removed, and unchanged stable IDs
  for every category
- `affected_scope` with affected Specification, requirement, task, and proof
  link IDs while completed and unaffected work remains preserved
- an approval and/or lifecycle `receipt` when the operation issued one,
  `replayed`, projection path/revision/expected and actual digest/drift state,
  `projection_repaired`, and exact `next_action`

Status is the closed set `draft`, `approved`, and `superseded`. Scope is
`whole_goal` or `feature`; feature scope binds the exact feature plus at least
one requirement or acceptance-check ID.

## Operations

### Inspect

```text
aether spec --inspect
```

`inspect` is read-only (`state_effect: none`). It returns the complete current
revision and projection state. A draft's exact `next_action` is the approval
command containing current revision ID, full content hash, and runtime-issued
approval token. An approved revision's next action is `aether plan`. Projection
drift takes precedence and returns `aether spec --repair-projection`.

### Add, modify, and remove

Every revision operation names one typed `--section` and exact predecessor:

```text
aether spec --add --section <section> --lineage <lineage> --text <owner-readable-text> --evidence <evidence-id> --predecessor-revision <revision-id> --predecessor-hash <content-hash>
aether spec --modify --section <section> --item-id <stable-id> --text <owner-readable-text> --evidence <evidence-id> --predecessor-revision <revision-id> --predecessor-hash <content-hash>
aether spec --remove --section <section> --item-id <stable-id> --predecessor-revision <revision-id> --predecessor-hash <content-hash>
```

Add requires semantic lineage and lets Go derive the stable ID. Modify preserves
the exact stable ID. Remove keeps the item in immutable history. Add/modify may
use exactly one `--text` value or contained, regular, non-empty repository
`--file`; acceptance checks also require `--verification`, and affected public
paths also require `--path`. Go creates a predecessor-linked successor
`DRAFT`, classifies its delta, calculates affected scope, updates the readable
projection, and commits all canonical targets atomically.

### Approve

```text
aether spec --approve --revision-id <revision-id> --revision-hash <content-hash> --approval-token <approval-token>
```

Approval requires explicit owner confirmation and all three exact values from
the current inspection result. The `SpecApprovalReceipt` binds Specification
ID, revision ID, revision content hash, hashed approval token, owner, and time.
Approval changes only Specification authority; it does not stop planning,
accept a candidate, activate a PlanRevision, or authorize Build/Autopilot.

### Repair the projection

```text
aether spec --repair-projection
```

Repair regenerates `.aether/SPEC.md` from canonical state. It cannot change the
Specification revision, status, approval, affected scope, plan, candidate, or
active revision. An already-correct projection is an idempotent no-op.

## Exact Replay, Refusal, and Recovery

Every mutation uses immutable IDs/hashes and a lifecycle transaction:

- An exact retry of add/modify/remove or approval returns the same successor and
  durable receipt with `replayed: true`; an exact projection-repair retry keeps
  the synchronized projection and creates no duplicate state.
- A divergent replay — same predecessor or revision identity but changed
  content, lineage, scope, evidence, approval hash, or token — is refused. A
  stale predecessor or stale approval is also refused. State unchanged: the
  active plan remains unchanged.
- Closed refusal classes are `missing_specification`, `invalid_operation`,
  `invalid_scope`, `invalid_source`, `stale_predecessor`,
  `divergent_revision_replay`, `approval_binding_mismatch`,
  `divergent_approval_replay`, `projection_drift`, and
  `atomic_persistence_failure`. Rendered wording may add context, but it must
  preserve the typed class, exact current/requested IDs and hashes, zero state
  effect, and the runtime's exact recovery command.

Recovery is read-only first: rerun `aether spec --inspect`; if and only if it
reports projection drift, run `aether spec --repair-projection`. Never silently
substitute the newest revision into a stale operation. Planning resumes with
`aether plan` only after the exact current revision is approved and affected
scope is reconciled.

## Separation from Planning Authority

The three owner boundaries are independent:

| Boundary | What it authorizes | What it never authorizes |
|----------|--------------------|--------------------------|
| Specification approval | Plan against one exact contract revision | Planning stop, candidate acceptance, plan activation |
| Planning stop | Review one evidence-backed candidate | Candidate acceptance, plan activation |
| Candidate acceptance | Activate one exact candidate as a PlanRevision | Rewriting the approved Specification |

A material planning answer with `direct_resume` leaves the approved contract
unchanged. An answer with `successor_spec_required` creates a successor draft;
that draft must pass this exact approval flow and affected-scope reconciliation
before the staged planning run resumes.

## Wrapper Boundary

Wrappers may render the complete structured body, collect one section/item/text
choice, ask the exact No-default approval question, and execute the exact
runtime command after owner confirmation. They may not:

- write `COLONY_STATE.json`, `.aether/SPEC.md`, canonical revisions, plan
  artifacts, transaction journals, projection digests, or affected-scope data
- parse the Markdown projection as authority; derive stable IDs; classify
  deltas; calculate scope; mint tokens or receipts; or rewrite `next_action`
- treat editor completion, file existence, a generic Yes/OK, Discuss
  completion, preset selection, planning stop, or candidate readiness as
  Specification approval
- treat Specification approval as exact plan-candidate acceptance

See `cmd/contracts/plan.md` for the staged planning and candidate-acceptance
contracts, and `.aether/docs/wrapper-host-contract.md` for the host boundary.
