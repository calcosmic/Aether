---
name: ant-discuss
description: "💬 Resolve evidence-backed material decisions and hand settled intent to a draft specification"
---
<!-- Aether-managed: runtime spec at .aether/commands/discuss.yaml. Synced by aether update. -->

Use the Go `aether` CLI as the source of truth.

## Runtime-First Evidence Boundary

Run `AETHER_OUTPUT_MODE=json aether discuss $ARGUMENTS` and consume that
structured result. Go assembles the current evidence frontier and decides
which unresolved choices are material or already answerable. It also owns
answer reuse: an answer is reusable only while the exact goal, session,
specification revision, base plan, meaning, behavior, authority, risk, scope,
acceptance impact, and affected semantic IDs remain equivalent.

`material_batch.cards` is the complete batch of decisions Go itself found
material. When it is present, render every card first, with these Go-issued fields:

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

## Owner Interview

The owner's ruling (2026-09-26): before a draft description goes to `/ant-spec`,
interview the owner. After Go's own `material_batch.cards` are answered, or when
there are none, compose about 20 multiple-choice questions specific to this goal
and this project: what the result looks like, what is in and out of scope, how it
should behave, the quality and style it must meet, what must never happen, and how
the owner will check it. Ground every question in the goal, the charter, the
codebase map and files you can read. Never ask what the goal, Go's resolved
answers, or an earlier interview already settled, and never fall back to a
generic canned question list.

Ask them up to 4 at a time with the platform's own multiple-choice question tool
(AskUserQuestion in Claude Code; a numbered list where there is none), 2-4 options
each, the recommended option first and marked "(Recommended)", every option
saying its real-world consequence in plain words. The owner may always give his
own answer, or say "enough" to stop early. Fewer than 20 is fine only when the
goal genuinely has fewer real decisions; say so when that happens.

Record each question before asking it:
`aether discuss --add-question "<question>" --options "<option 1>|<option 2>" --grounding "<what it is based on>" --category <surface|integration|scope|verification|analysis> --source <short-stable-slug>`
(add `--hard` when the answer is a must-never rule), and each pick after he answers:
`aether discuss --resolve <id> --answer "<the option he chose, or his own words>"`.
Nothing is recorded without the owner's explicit pick. Never answer for him.

When the interview is done, rerun `AETHER_OUTPUT_MODE=json aether discuss`. Go
writes every answer into the draft description; show the returned page, including
its "What you decided" section, then route to `/ant-spec`. If this project already
had an interview, ask once whether he wants more questions instead of starting over.

## Settled Intent Is a Draft Boundary

When `discussion_status` is `settled`, render the exact `draft_spec` or retained
`approved_spec` returned by Go. A newly settled specification is `DRAFT`. Say
plainly that it does not authorize planning, build, or any other approval until
the owner reviews and approves that exact revision through `/ant-spec`.

- Never route settled discuss directly to `/ant-plan`; the next public boundary is `/ant-spec`.
- Do not write `pending-decisions.json`, `pheromones.json`, `COLONY_STATE.json`, `.aether/SPEC.md`, or lifecycle receipts by hand.
- Do not synthesize evidence, receipts, draft state, specification approval, or plan approval; interview questions go through `--add-question` only.
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
