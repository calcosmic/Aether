# discuss -- Lifecycle Contract

**Last verified:** 2026-09-08
**Source files:** `cmd/discuss.go`, `cmd/discuss_analyze.go`, `cmd/planning_decision.go`, `cmd/specification.go`, `cmd/pending_decision.go`

## Purpose

`aether discuss` turns the current goal and repository evidence into either one
complete batch of material owner decisions or a runtime-owned draft
specification. It is an evidence-first lifecycle command, not a generic
interview or a fixed question-count ceremony.

For beginners: Aether checks what the project already proves, asks only the
choices that still need a human, and then writes a draft agreement for the
owner to review. The wrapper displays that work; it does not invent questions
or edit state itself.

## Inputs

### Flags

| Flag | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `--max-questions` | int | no | 3 | Legacy compatibility flag. It does not truncate the complete material-decision batch. |
| `--dry-run` | bool | no | false | Analyze and preview the same decision or draft-spec result without committing lifecycle writes. |
| `--resolve` | string | no | `""` | Exact runtime-issued clarification decision ID to resolve. |
| `--answer` | string | with `--resolve` | `""` | Exact owner answer bound to the ID supplied by `--resolve`. |
| `--add-question` | string | no | `""` | Compatibility path for adding an explicitly sourced clarification. |
| `--options` | string | no | `""` | Pipe-delimited choices for `--add-question`. |
| `--category` | string | with `--add-question` | `""` | Material decision domain for an explicitly added question. |
| `--grounding` | string | with `--add-question` | `""` | Evidence-grounded reason the explicitly added question is necessary. |
| `--hard` | bool | no | false | Marks an added question so its resolved answer emits a hard `REDIRECT` constraint. |
| `--source` | string | no | derived | Stable deduplication slug for an explicitly added question. |

### Arguments

None.

### Evidence frontier

The Go runtime reads the current accepted charter (or charter draft), current
survey and codebase context, active owner constraints, current-goal pending
decisions, and resolved clarification evidence. Every evidence item is bound to
the exact goal, session, approved specification revision, and base plan
revision before the runtime classifies a decision.

## Decision ownership

The Go runtime exclusively owns:

- evidence collection and normalization;
- evidence-answerability and materiality classification;
- creation, ordering, persistence, resolution, and suppression of decisions;
- prior-answer equivalence and revalidation;
- draft-spec creation or replay after the material frontier is settled; and
- lifecycle receipts, projections, and next-command truth.

Claude, OpenCode, and Codex wrappers only invoke the runtime once per owner
action and render its structured result. They must not compose fallback
questions, impose a 3-5 question quota, infer that an answer is reusable,
write lifecycle files, or skip directly from `settled` to planning.

## Outputs

Stdout is an `outputWorkflow` envelope with visual and structured forms.

### Material decisions

When owner input is still required, `discussion_status` is
`new_questions` or `pending_questions`, `planning_paused` is true, and
`material_batch.cards` contains the complete material batch. Each card names:

| Field | Meaning |
|-------|---------|
| `decision_id` / `id` | Stable decision identity and the exact resolvable runtime ID. |
| `decision` | The owner choice that remains material. |
| `why_now` | Why planning cannot safely proceed without this answer. |
| `evidence` | Typed evidence references supporting the decision boundary. |
| `queen_recommendation` | A bounded recommendation; it is not an owner answer. |
| `choices[].label` | A selectable answer. |
| `choices[].consequence` | What becomes true if that choice is selected. |
| `choices[].impact` | Behavior, authority, scope, risk, or acceptance impact. |
| `affected_semantic_ids` | Exact contract IDs affected by the choice. |
| `prior_answer` | A reusable answer only when runtime equivalence succeeds. |
| `revalidation` | Why a prior answer can or cannot be reused. |
| `planning_resumes` | The runtime-owned condition for continuing. |
| `exact_answer_syntax` | Exact `aether discuss --resolve ... --answer ...` binding. |

All cards are returned together. Answering one card is one owner action; the
caller must rerun `aether discuss` afterward and render the fresh runtime
result. A generic response, card position, or wrapper-generated ID is not a
valid binding.

### Answer equivalence

The runtime may reuse a prior answer only when all equivalence dimensions are
unchanged: exact goal, exact session, approved specification revision, base
plan revision, decision meaning, behavior, authority, scope, risk, acceptance
meaning, and affected semantic IDs. Any mismatch requires explicit
revalidation or a new owner answer.

### Settled discussion and specification handoff

When no material owner decisions remain, `discussion_status` is `settled` and
the runtime returns either `draft_spec` or an existing `approved_spec`. A new
draft contains nine contract categories:

1. outcomes;
2. included behaviors;
3. exclusions;
4. binding decisions;
5. requirements;
6. acceptance checks;
7. negative expectations;
8. recovery expectations; and
9. affected public paths.

The exact next command is `aether spec` (rendered as `/ant-spec` by slash-command
wrappers). A draft remains unapproved and cannot authorize planning. Approval
is a separate `aether spec --approve ...` owner action bound to the exact
revision ID, content hash, and approval token returned by the runtime. An
existing approved specification is preserved rather than silently replaced.

### Resolve mode

`--resolve <id> --answer <answer>` returns the exact binding (`resolved`, `id`,
`answer`), hard-constraint signal status, remaining decision count, and, when
the final material decision was answered, the same settled specification
closeout and exact next command.

## Ceremony class

`discuss` is a runtime-owned evidence and owner-decision ceremony. Renderers
show the returned cards, resolution, or specification closeout. They do not
claim Scout, Builder, Watcher, or other worker dispatch unless a separate
lifecycle command emitted a worker manifest.

## Files created or modified

| File | Operation | When |
|------|-----------|------|
| `.aether/data/pending-decisions.json` | update | Runtime materializes or resolves current-scope clarification decisions. |
| `.aether/data/pheromones.json` | update | A resolved hard constraint emits a `REDIRECT` signal. |
| `.aether/data/COLONY_STATE.json` | transactional update | A settled discussion creates/replays the canonical draft specification or repairs its projection metadata. |
| `.aether/SPEC.md` | create or repair | Runtime projects the canonical draft or approved specification for owner review. |
| lifecycle receipt files | append | Runtime records committed specification transitions. |

Wrappers never write these files directly.

## State mutations

- Pending clarification creation and resolution stay scoped to the current
  goal/session.
- A settled discussion may commit a draft specification transition; it never
  commits specification approval or plan approval.
- `--dry-run` reports `would_create` or equivalent preview state without
  committing the transition.

## Exit codes

| Code | Meaning |
|------|---------|
| 0 | Successful surface, resolution, dry-run preview, or settled closeout. |
| 1 | Invalid flag combination, missing colony/goal, stale or unknown decision ID, empty answer, or failed lifecycle transaction. |

## Preconditions

- The colony is initialized with a non-empty current goal.
- The repository store is initialized.
- Resolution uses the exact current-scope decision ID and a non-empty answer.
