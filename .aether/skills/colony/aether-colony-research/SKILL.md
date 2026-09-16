---
source: shipped
name: aether-colony-research
description: Use when Codex is asked to run Aether Oracle or discuss flows and should refine scope before research or clarification
type: colony
domains: [aether, codex, oracle, research, clarification, prd]
agent_roles: [oracle, scout, architect, queen]
workflow_triggers: [oracle, discuss]
task_keywords: [aether oracle, oracle research, prd, research brief, discuss, clarification, requirements]
priority: high
version: "1.0"
---

# Aether Colony Research

## Installed Codex entrypoints

Use `$ant-oracle` and `$ant-discuss`. This document is installed as
`support/aether-colony-research.md` beside the public skill directories;
it is private support, not a separately discoverable helper skill. Public skills
resolve its path relative to their installed `SKILL.md`, never the working directory.
The executable commands and runtime-issued IDs remain `aether` identities.
Full workflow coverage and native-worker parity remain pending.

## Purpose

Give Codex the wrapper-equivalent intelligence for `oracle` and `discuss`.
Codex should ask sharper questions, select the right Oracle template, and then
let the runtime own persistence and loop control.

For beginners: this is the part that turns "research this" into a focused brief
before the Oracle starts working.

## Required First Step

For Oracle:

```bash
aether command-guide oracle --platform codex
```

For discussion:

```bash
aether command-guide discuss --platform codex
```

Use `command-guide` as the source of truth. If it differs from this skill, follow
`command-guide` and update the skill.

## Raw Bypass

If the user explicitly says raw, exact, no interview, no orchestration, or "just
run this exact command", run the literal CLI command they provided. Say briefly
that the Codex research synthesis layer was bypassed.

## Oracle Flow

1. Read the user's topic and intended outcome.
2. Ask one compact batch of 3-6 questions when scope, audience, output shape,
   constraints, decision criteria, or confidence target are unclear.
3. Infer the template:
   - PRD, requirements, product brief, user stories: `prd`
   - technology comparison or vendor choice: `tech-eval`
   - architecture or design review: `architecture-review`
   - bug, incident, or root cause: `bug-investigation`
   - practices, conventions, or background research: `research-brief`
4. Present research depth as selectable options unless the user already gave
   one: `quick` (5 iterations), `balanced`/`standard` (15), `deep` (30), or
   `exhaustive`/`marathon` (50). Do not hide this behind a raw flag-only flow.
   If the user gives an exact iteration cap, pass `--max-iterations <1-50>`.
5. Ask the user to choose target confidence unless they already gave one: 80%,
   90%, 95% recommended, or 99%. Pass the selected number as
   `--confidence-target <percent>`. Oracle should iterate until it reaches that
   target, hits max iterations, or reports a hard blocker.
6. Synthesize the answers into a precise prompt. Do not pass a vague prompt
   through unchanged.
7. If the user asks for "everything", "all of the above", a full-system audit,
   or a large uncommitted diff review, split the topic into focused Oracle runs
   or start with explicit quick triage. Do not collapse every area into one
   blocking balanced-depth prompt.
8. Run:

```bash
AETHER_OUTPUT_MODE=visual aether oracle --depth <depth> --confidence-target <percent> --template <template> --background "<synthesized prompt>"
```

Use `--background` for long-running research, especially from OpenCode. The
runtime detaches a controller, writes progress under `.aether/oracle`, and
`aether oracle status` remains the inspection path. Omit `--background` only
when the user explicitly wants foreground execution.

For broad triage, prefer:

```bash
AETHER_OUTPUT_MODE=visual aether oracle --depth quick --confidence-target <percent> --template <template> "<focused triage prompt>"
```

If the shell/tool call times out, run `aether oracle status` before declaring
failure or switching to ad hoc agents.
If OpenCode subprocess dispatch is unavailable, let Oracle use its automatic
Codex/Claude fallback unless the user explicitly set `AETHER_WORKER_PLATFORM=opencode`.

9. After Oracle completes, suggest persisting high-value findings as pheromones
   or hive wisdom only after user approval.

## Discuss Flow

1. Run:

```bash
AETHER_OUTPUT_MODE=json aether discuss
```

2. Treat the Go result as authoritative. The runtime collects current charter,
   survey, codebase, constraint, clarification, specification-revision, and
   plan-revision evidence; it alone decides which gaps are evidence-answerable
   and which choices remain material. Do not compose fallback questions or
   impose a fixed question count.
3. When `material_batch.cards` is present, render the complete batch. For every
   card show `decision`, `why_now`, typed `evidence`,
   `queen_recommendation`, each choice and its `consequence`,
   `affected_semantic_ids`, `prior_answer`, `revalidation`,
   `planning_resumes`, and `exact_answer_syntax`. A recommendation is not an
   owner answer.
4. Ask the owner to answer a card using the exact runtime-issued decision ID
   and answer binding. Persist it only through the syntax returned by the card,
   equivalent to:

```bash
AETHER_OUTPUT_MODE=json aether discuss --resolve <exact-id> --answer "<answer>"
```

5. After each owner answer, rerun `AETHER_OUTPUT_MODE=json aether discuss` and
   render the fresh result. Do not assume the previous card list, ordering, or
   remaining count is still current.
6. Reuse a prior answer only when the runtime says it is equivalent across the
   exact goal, session, approved specification revision, base plan revision,
   decision meaning, behavior, authority, scope, risk, acceptance meaning, and
   affected semantic IDs. Otherwise surface the runtime's revalidation need.
7. When the runtime reports `discussion_status: settled`, render the returned
   `draft_spec` or existing `approved_spec` and direct the owner to the exact
   next command, `aether spec`. A draft is not plan approval: specification
   review and exact-revision approval remain a separate owner action before
   planning can begin.

## Guardrails

- Do not write `.aether/data`, Oracle output, pending decisions, or pheromone
  files by hand.
- Do not make the user run a separate PRD command before Oracle. PRD is an
  Oracle template and synthesis mode.
- Treat shell/tool timeouts as controller interruptions until `aether oracle
  status` proves the runtime is blocked, stopped, or complete.
- If Claude/OpenCode oracle or discuss wrapper behavior changes, update the
  matching YAML, this skill, and `cmd/command_guide.go` together.
