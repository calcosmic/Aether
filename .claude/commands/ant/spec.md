<!-- Aether-managed: runtime spec at .aether/commands/spec.yaml. Synced by aether update. -->
---
name: ant-spec
description: "📜 Review, revise, approve, or repair the owner-readable specification"
---

Use the Go `aether` CLI as the source of truth.

## Runtime-First Specification Boundary

Run `AETHER_OUTPUT_MODE=json aether spec $ARGUMENTS` and consume the structured
result. With no operation flag, `aether spec` performs the same read-only
inspection as `aether spec --inspect`. Go owns the canonical specification,
immutable revision lineage, stable IDs, content hashes, classified delta,
affected scope, approval token and receipt, projection repair, and every state
write. This wrapper only presents Go-issued facts and submits the owner's exact
chosen action.

Always show the exact specification ID, revision number and ID, full content
hash where an action binds it, and the distinct `DRAFT`, `APPROVED`, or
`SUPERSEDED` status. Render all nine typed body categories in this fixed order:

1. `outcomes`
2. `included_behaviors`
3. `exclusions`
4. `binding_decisions`
5. `requirements`
6. `acceptance_checks`
7. `negative_expectations`
8. `recovery_expectations`
9. `affected_public_paths`

Keep each category and its runtime-issued stable IDs separate. Do not collapse
them into generic items, derive an ID, or parse `.aether/SPEC.md` back into
authority.

## Open and Inspect

Use `AETHER_OUTPUT_MODE=json aether spec --inspect` for a read-only view. Render
the scope and predecessor lineage, exact body, approval receipt when present,
projection status, classified delta, affected scope, state effect, and exact
next action returned by Go. Inspection, a readable file, and an editor save do
not approve anything.

## Create an Immutable Successor Revision

Every material edit creates a predecessor-linked successor `DRAFT`; prior
revision bytes remain historical. Submit one operation at a time through these
runtime-backed routes:

- Add: `aether spec --add --section <section> --lineage <lineage> --text <text> --evidence <evidence-id> --predecessor-revision <revision-id> --predecessor-hash <content-hash>`
- Modify: `aether spec --modify --section <section> --item-id <stable-id> --text <text> --evidence <evidence-id> --predecessor-revision <revision-id> --predecessor-hash <content-hash>`
- Remove: `aether spec --remove --section <section> --item-id <stable-id> --predecessor-revision <revision-id> --predecessor-hash <content-hash>`

Use `--file <repository-relative-regular-file>` instead of `--text` only when
the owner supplies a contained repository file. For `acceptance_checks`, also
pass `--verification <owner-checkable-verification>`. For
`affected_public_paths`, also pass `--path <public-path>`. A feature-scoped
successor uses the exact `--scope feature`, `--feature-id`,
`--scope-requirement`, and/or `--scope-acceptance` bindings accepted by Go.
Never widen the scope or silently substitute the latest predecessor.

After an add, modify, or remove result, render the semantic revision impact for
all nine categories:

- unchanged IDs and content retained
- added IDs and summaries
- modified stable IDs and changed meaning
- removed IDs with their retained-history status
- affected specification, requirement, task, and proof-link IDs
- preserved completed and unaffected work

A changed contract must be explicitly approved and its exact affected plan
scope reconciled before work or Seal may rely on it. The wrapper does not
compute, clear, or infer reconciliation.

## Exact Specification Approval

For a current draft, show the exact No-default prompt:

`Approve SPEC <spec-id> revision <N> (<short-hash>) as the planning contract? [y/N]`

Only after the owner explicitly approves that exact reviewed revision, execute
the runtime-issued approval action verbatim. Its shape is:

`aether spec --approve --revision-id <revision-id> --revision-hash <content-hash> --approval-token <approval-token>`

Do not mint, reconstruct, shorten, or update this command. Editor completion,
file existence, discussion completion, generic confirmation, a planning stop,
or candidate readiness is not approval. A stale revision, predecessor, hash,
goal/session, or token must be refused with `State: unchanged`; never approve a
newer revision instead. Exact replay may retain the existing approval receipt.

## Projection Repair

When inspection reports `.aether/SPEC.md` missing or drifted, use
`AETHER_OUTPUT_MODE=json aether spec --repair-projection`. Render the runtime's
projection result and say plainly that the readable file was regenerated from
canonical state. Projection repair changes no specification revision,
approval, affected scope, or plan authority.

## Authority Wall

Specification approval establishes only the planning contract. It does not
accept a plan candidate, activate a plan revision, make anything buildable,
authorize Autopilot, or prove Seal eligibility. Preset selection and an honest
planning stop remain separate again. Exact candidate review and acceptance
belong to `/ant-plan`; this surface must never execute or construct a plan
acceptance command.

Do not write `COLONY_STATE.json`, `.aether/SPEC.md`, planning files,
transaction journals, approval receipts, or affected-scope records by hand.
Do not classify deltas, calculate impact, mint tokens, fabricate receipts, or
infer any authority in wrapper prose. If docs and runtime disagree, runtime
wins.

## Platform Spelling and Drift Guard

Claude Code and OpenCode expose this workflow as `/ant-spec`. Direct Codex uses
`aether spec`; it has no native `$ant-spec` alias. Keep this file,
`.claude/commands/ant/spec.md`, `.opencode/commands/ant/spec.md`, and
`.aether/commands/spec.yaml` synchronized.

**Next steps:**
- Draft: continue an exact add/modify/remove route, or explicitly approve the reviewed revision.
- Approved and reconciled: `/ant-plan` — choose a planning preset.
- Candidate ready: stay in `/ant-plan` to review and separately accept the exact plan candidate.
