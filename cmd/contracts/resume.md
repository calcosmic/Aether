# resume — Lifecycle Contract

**Last verified:** 2026-09-04
**Source files:** `cmd/session_flow_cmds.go`, `cmd/lifecycle_facts.go`, `cmd/lifecycle_transaction.go`, `cmd/normalize_args.go`

## Inputs

### Public command

`aether resume` is the runtime's only recovery entry point. Its generated
wrapper is `/ant-resume`.

- Arguments: none.
- `--no-handoff` disables `HANDOFF.md` reconstruction when durable colony state
  is not runnable.

### Evidence read

Before declaring any write, `resume` reads the lifecycle fact bundle and checks
the handoff identity against durable state, session state, the pause receipt,
repository bytes, worktree evidence, and worker activity.

## Outputs

Visual and structured output report the recovery provenance, transaction
receipt and state effect, whether the result was replayed, and the next safe
action.

### Evidence and provenance

| Provenance | Meaning | State effect |
|------------|---------|--------------|
| `confirmed` | The referenced pause handoff and all independent evidence agree. | Commit or replay the named resume transaction. |
| `reconstructed` | The runtime can derive one honest recovery point from durable evidence. | Commit or replay the named resume transaction and persist the reconstructed handoff evidence. |
| `conflicting` | Independent evidence disagrees or a supposedly finished transaction cannot be validated. | None; render the named conflict and safe inspection step. |
| `unknown` | There is not enough readable evidence to choose an honest recovery point. | None; render the missing evidence and safe inspection step. |

Confirmed and reconstructed facts remain visibly distinct in both visual and
structured output. Conflicting and unknown outcomes never mutate lifecycle
state.

## State Mutations

### Transaction contract

The transaction identifier is derived from the handoff identity. An existing
intent or receipt is resumed instead of creating a second recovery operation,
so retries are idempotent.

A successful transaction updates `.aether/data/COLONY_STATE.json`,
`.aether/data/session.json`, and `.aether/CONTEXT.md`; it removes the consumed
`.aether/HANDOFF.md`. Reconstructed recovery also writes
`.aether/data/pause-handoff.json`. When stale worker activity is part of the
validated recovery point, the transaction removes the stale spawn records.
The lifecycle transaction journal retains the intent and receipt that prove the
committed state effect.

## Preconditions

- The lifecycle store must be initialized before recovery can proceed.
- A mutation requires confirmed or explicitly reconstructed evidence.
- Conflicting or unknown evidence returns a zero-state-effect result and an
  inspection action instead of making the colony runnable.

## Bounded parser compatibility

During the 1.28 milestone only, process startup rewrites one exact historical
suffixed token to `resume` before Cobra parses arguments. The rewrite expires at
1.29 and is deliberately absent from command metadata, help, completion,
canonical YAML, and generated wrappers. It is hidden migration plumbing, not a
flag, alias, or user-selectable route.
