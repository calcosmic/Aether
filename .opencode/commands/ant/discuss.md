<!-- Aether-managed: runtime spec at .aether/commands/discuss.yaml. Synced by aether update. -->
---
name: ant-discuss
description: "💬 Resolve evidence-backed material decisions and hand settled intent to a draft specification"
---

Use the Go `aether` CLI as the source of truth.

## Runtime-First Evidence Boundary

Run `AETHER_OUTPUT_MODE=json aether discuss $ARGUMENTS` and consume that
structured result. Go assembles the current evidence frontier and decides
which unresolved choices are material or already answerable. It also owns
answer reuse: an answer is reusable only while the exact goal, session,
specification revision, base plan, meaning, behavior, authority, risk, scope,
acceptance impact, and affected semantic IDs remain equivalent.

Do not compose questions, add a fixed question quota, or fall back to a generic
category menu. `material_batch.cards` is the complete currently known batch.
When it is present, render every card with these Go-issued fields:

- `decision` and `why_now`
- cited `evidence`
- `queen_recommendation`
- every viable choice and its `consequence`
- `prior_answer` and `revalidation`
- `planning_resumes`
- `exact_answer_syntax`

A platform-native question card may render those structured choices, but it may
not add, drop, rewrite, classify, or answer a card. Nothing is recorded without
the owner's explicit pick. Submit the pick only through the card's exact
`aether discuss --resolve <id> --answer "<answer>"` syntax.

After an answer, request a fresh `AETHER_OUTPUT_MODE=json aether discuss`
result. Continue only from the newly returned remaining batch or exact next
command; never reuse the prior batch or fabricate a resume token, receipt, or
state transition.

## Compose the Questions — Retired Compatibility Marker

This heading and the legacy `--add-question`, `--grounding`, and
`AskUserQuestion` names remain only so older installed wrappers can recognize
the retired contract. They are not executable guidance. This wrapper must not
compose or submit questions. The former promise was to make questions SPECIFIC to this goal and this
codebase; Go now satisfies it through the complete evidence-backed batch.
Nothing is recorded without the
   user's explicit pick, and that pick is valid only when submitted through the
runtime-issued `exact_answer_syntax`.

## Canned fallback (typed condition — retired compatibility only)

The historical fallback applied when there was no scan context or the goal is empty.
It is disabled here: the wrapper never runs a canned generator because
only Go may decide whether a material question exists.

## Settled Intent Is a Draft Boundary

When `discussion_status` is `settled`, render the exact `draft_spec` or retained
`approved_spec` returned by Go. A newly settled specification is `DRAFT`. Say
plainly that it does not authorize planning, build, or any other approval until
the owner reviews and approves that exact revision through `/ant-spec`.

- Never route settled discuss directly to `/ant-plan`; the next public boundary is `/ant-spec`.
- Do not write `pending-decisions.json`, `pheromones.json`, `COLONY_STATE.json`, `.aether/SPEC.md`, or lifecycle receipts by hand.
- Do not synthesize questions, evidence, recommendations, receipts, draft state, specification approval, or plan approval.
- Do not render Scout/Builder/Watcher theatre for discuss; no planning worker is dispatched here.
- Use `/ant-council` only when the owner wants multi-position deliberation.
- If docs and runtime disagree, runtime wins.

## Cross-Platform Drift Guard

If you change discuss evidence, decision-card presentation, answer persistence,
or draft routing here, update `.aether/commands/discuss.yaml`, the matching
Claude/OpenCode wrappers, `cmd/command_guide.go`, the public discuss contract,
and the Codex skill `aether-colony-research` in the same change. Verify
`aether command-guide discuss --platform codex` still describes the same flow.

**Next steps:**
- `/ant-spec` — review, revise, or explicitly approve the exact specification
- `/ant-assumptions` — surface plan assumptions after planning
