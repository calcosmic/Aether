<!-- Generated from .aether/commands/oracle.yaml - DO NOT EDIT DIRECTLY -->
---
name: ant-oracle
description: "🔮 Run the autonomous Oracle loop through the Aether CLI runtime"
---

Use the Go `aether` CLI as the source of truth.

- Execute `AETHER_OUTPUT_MODE=visual aether oracle "$ARGUMENTS"` directly only for `status`, `stop`, or when the user explicitly asks to skip refinement.
- For inspection or control, prefer `aether oracle status` and `aether oracle stop`.
- Do not describe legacy loop control files or shell-managed orchestration from this command spec.
- Report the CLI result directly.

## Intent Refinement

When the user provides a research topic, do a short AI-led scoping pass before
starting the runtime loop.

For beginners: this is the part where you turn "look into this" into a useful
research brief, so Oracle does not spend iterations answering the wrong
question.

Ask one compact batch of 3-6 questions if any of these are unclear:
- the decision the user needs to make
- the desired output shape, such as PRD, tech evaluation, architecture review,
  bug investigation, or research brief
- target users or affected worker roles
- constraints, non-goals, deadlines, or risk tolerance
- evidence sources to prefer: repo, web/current docs, or both
- what would make the answer actionable

After the user answers, synthesize:
- a precise Oracle prompt
- the template to pass to the runtime
- the scope flag if it is obvious

Template mapping:
- PRD, requirements, user stories, acceptance criteria, product scope:
  `--template prd`
- technology/library/tool comparison: `--template tech-eval`
- system design or architecture review: `--template architecture-review`
- bug, regression, failure, or root cause: `--template bug-investigation`
- best practices, conventions, or patterns: `--template research-brief`
- otherwise: `--template custom`

Ask the user to choose research depth unless they already gave one:
- `quick` — fast first pass
- `balanced` — normal thoroughness
- `deep` — comprehensive investigation
- `exhaustive` — near-complete convergence

Ask the user to choose target confidence unless they already gave one:
- **80% confidence** — good enough for a first pass
- **90% confidence** — solid understanding
- **95% confidence (recommended)** — thorough, few gaps remaining
- **99% confidence** — near-exhaustive

Pass the selected target as `--confidence-target <percent>`. Oracle should keep
iterating until it reaches that target, hits max iterations, or reports a hard
blocker.

The PRD template/reference is automatic. Do not ask the user to run
`aether reference-match`; that command is only diagnostic.

Run the Oracle after the user confirms the refined prompt:

```bash
AETHER_OUTPUT_MODE=visual aether oracle --depth <depth> --confidence-target <percent> --template <template> "<synthesized prompt>"
```

## Broad Scope And Timeout Handling

If the user asks for "everything", "all of the above", a full-system audit, or a
large uncommitted diff review, do not collapse every concern into one blocking
balanced-depth Oracle prompt. Either split the work into focused Oracle prompts
or run an explicit quick triage first:

```bash
AETHER_OUTPUT_MODE=visual aether oracle --depth quick --confidence-target <percent> --template <template> "<focused triage prompt>"
```

If the shell/tool call times out, immediately inspect the runtime state:

```bash
aether oracle status
```

Report that status. Do not assume Oracle failed, and do not bypass it with ad
hoc agents until the runtime status says it is blocked, stopped, or complete.

## Cross-Platform Drift Guard

If you change Oracle interview, template selection, persistence suggestions, or
closeout behavior here, update `.aether/commands/oracle.yaml`,
`cmd/command_guide.go`, and the Codex skill `aether-colony-research` in the same
change. Verify `aether command-guide oracle --platform codex` still describes
the matching Codex flow.

## Post-Completion Persistence Suggestions

After the oracle loop completes and you have reported the research results, check whether the findings are worth preserving for the colony:

**Only suggest persistence when:**
- The oracle completed successfully (status is "complete" or "max_iterations_reached")
- Do NOT suggest persistence if the oracle was blocked or stopped early

**How to identify high-value findings:**
- Findings with high confidence that apply to the colony's current goal
- Actionable recommendations that would benefit future workers
- Domain-specific insights that are not obvious from the codebase alone
- Architecture or design decisions documented in the research

**Suggestion format:**
Present findings worth preserving as a tick-to-approve list:

```
🔮 Research findings worth preserving:

1. [x] {finding title}
   → Suggest: aether pheromone-write --type "FOCUS" --content '{"text":"{finding summary}"}' --priority "normal" --source "oracle" --reason "High-value research finding"

2. [ ] {finding title}
   → Suggest: aether hive-store --text "{generalized finding}" --domain "{detected domain}" --source-repo "{repo name}"
```

For each approved finding:
- If it is colony-specific guidance: use `aether pheromone-write --type "FOCUS" --content '{"text":"{finding}"}' --priority "normal" --source "oracle" --reason "High-value research finding"`
- If it is generalizable cross-colony wisdom: use `aether hive-store --text "{generalized finding}" --domain "{domain}" --source-repo "{repo name}"`
- Let the user approve each suggestion before persisting

**Do NOT suggest persistence for:** low-confidence findings, obvious observations, or findings already captured as pheromones.
