<!-- Aether-managed: runtime spec at .aether/commands/oracle.yaml. Synced by aether update. -->
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

Present research depth as selectable options unless the user already gave one:
- `quick` — fast first pass, up to 5 iterations
- `balanced` or `standard` — normal thoroughness, up to 15 iterations
- `deep` — comprehensive investigation, up to 30 iterations
- `exhaustive` or `marathon` — near-complete convergence, up to 50 iterations

If the user gives an exact iteration cap, pass `--max-iterations <1-50>`.

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

## Research Brief Gate

Before any tokens burn on the loop, present the synthesized brief as an
approvable artifact — this is the gate that prevents the Oracle from spending
thirty iterations answering a malformed question:

```
╭─ 🔮 Research Brief ─────────────────────────────╮
  Topic:            {one-line topic}
  Core Question:    {the single question the loop must answer}
  Context:          {2-3 lines: why now, what decision this feeds}
  Success Criteria: {what a done answer contains — bullet list}

  Template: {template}   Depth: {depth}   Target: {confidence}%
╰─────────────────────────────────────────────────╯
```

Ask: **approve**, **edit** (revise a field and re-present), or **cancel**.
Maximum 2 edit rounds — after the second, run with the latest brief or cancel.
Do not start the runtime loop until the brief is approved. The Core Question
must be a genuine question, scoped to one decision — never a directory listing,
a task list, or "everything about X". If the user's topic is that broad, split
it into focused briefs and gate each one.

**Focus signal:** if the approved brief names a specific area of the codebase
or a constraint the colony should honor during upcoming builds, offer once to
record it: `aether focus "<area or constraint from the brief>"`. Write it only
with the user's approval — research topics are not automatically colony
steering.

Run the Oracle after the brief is approved, passing the synthesized prompt
built from the approved brief:

```bash
AETHER_OUTPUT_MODE=visual aether oracle --depth <depth> --confidence-target <percent> --template <template> --background "<synthesized prompt>"
```

Use `--background` for long-running research, especially from OpenCode. The
runtime detaches a controller, writes progress under `.aether/oracle`, and
`aether oracle status` remains the inspection path. Omit `--background` only
when the user explicitly wants foreground execution — and if they do, tell them
first: foreground Oracle runs its research workers through the **Go subprocess
path**, not the Agent tool. There is no visible per-worker ceremony while it
runs; progress is CLI output and `.aether/oracle` files only.

When the runtime detects a hosted Claude/OpenCode agent session and the command
is not already backgrounded, it auto-detaches the Oracle controller. Treat that
as a normal background run: report the PID/log path and inspect progress through
`aether oracle status`.

## Promoting Findings Into Colony Memory

When a research loop completes (or the user asks to capture what Oracle
learned), offer promotion:

```bash
AETHER_OUTPUT_MODE=json aether oracle promote --dry-run   # preview
AETHER_OUTPUT_MODE=json aether oracle promote             # write
```

Promote takes every finding from questions at or above 80% confidence
(`--min-confidence` to change) and stores the admissible ones as learnings and
instincts. Findings that name no file, command, or error are rejected with the
reason shown — that is the admissibility gate working, not a failure. Show the
user the outcome table: what was promoted, what was skipped, and why. Colony
steering (FOCUS/REDIRECT) stays a separate, user-approved step.

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
If OpenCode subprocess dispatch is unavailable, let Oracle use its automatic
Codex/Claude fallback unless the user explicitly set `AETHER_WORKER_PLATFORM=opencode`.
Do not fake Oracle worker completion; if the runtime reports no dispatcher,
preserve the Oracle workspace and surface the blocker plainly.

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

**Next steps:**
- `aether oracle status` — check a running loop
- `aether oracle promote --dry-run` — preview capturing findings into colony memory
- `/ant-plan` (refresh with `--revision-evidence`) — fold findings into the plan
