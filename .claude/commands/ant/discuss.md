<!-- Aether-managed: runtime spec at .aether/commands/discuss.yaml. Synced by aether update. -->
---
name: ant-discuss
description: "💬 Capture clarifications before planning through the Aether CLI runtime"
---

Use the Go `aether` CLI as the source of truth.

## Compose the Questions (Queen-composed, default path)

The Queen composes clarifying questions from THIS goal and THIS codebase —
never the same generic questions every time. The runtime materializes what
you compose into the same decision pipeline the canned generator uses.

1. Gather signal: run `AETHER_OUTPUT_MODE=json aether status` for the goal
   and colony state, and `aether discuss-analyze --target .` for the codebase
   scan. Read any saved survey or research documents the runtime names.
2. Compose 3–5 clarification questions SPECIFIC to this goal and this
   codebase. Every question must be grounded in something you actually
   observed — a scan fact, a survey finding, the goal's own wording. If you
   cannot say what a question is based on, do not ask it.
3. Materialize each composed question with one call:
   `AETHER_OUTPUT_MODE=json aether discuss --add-question "<question>" --options "a|b|c" --category <surface|integration|scope|verification|analysis> --grounding "<what you observed that motivates this>" --source wrapper:<stable-slug>`.
   The runtime REFUSES ungrounded questions and dedups by the source slug
   (safe to re-run). Add the `--hard` flag to that call when the answer must
   become a hard constraint (its resolution then emits a REDIRECT signal).
4. Present ALL pending clarifications (composed and runtime-generated alike)
   to the user as real multiple-choice questions (the AskUserQuestion tool):
   plain-English options, the question's reasoning stated, a "none of
   these / let me answer freely" path. Nothing is recorded without the
   user's explicit pick.
5. Persist each pick with `AETHER_OUTPUT_MODE=visual aether discuss --resolve <id> --answer "<choice>"`.

**Canned fallback (typed condition, not vibes):** when `discuss-analyze`
returns no scan context or the goal is empty, fall back to plain
`AETHER_OUTPUT_MODE=visual aether discuss $ARGUMENTS` — the runtime's
generator asks its standard questions. Composed questions always come first
when both exist; the runtime's dedup prevents doubling.

- Do not write `pending-decisions.json`, `pheromones.json`, or `COLONY_STATE.json` by hand from this command spec.
- If the runtime returns clarification questions, present them honestly instead of inventing answers on the user's behalf.
- If the runtime reports `discussion_status: settled`, route wrapper users back to `/ant-plan`; direct CLI users can run `aether plan`.
- Use `/ant-council` only when the user wants multi-position deliberation; `/ant-discuss` is the lightweight pre-plan clarification gate.
- If docs and runtime disagree, runtime wins.

## Cross-Platform Drift Guard

If you change discuss analysis, question presentation, answer persistence, or
routing behavior here, update `.aether/commands/discuss.yaml`,
`cmd/command_guide.go`, and the Codex skill `aether-colony-research` in the same
change. Verify `aether command-guide discuss --platform codex` still describes
the matching Codex flow.

**Next steps:**
- `/ant-plan` — plan with clarified intent
- `/ant-assumptions` — surface plan assumptions after planning
