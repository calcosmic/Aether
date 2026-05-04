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
4. Ask the user to choose research depth unless they already gave one: `quick`,
   `balanced`, `deep`, or `exhaustive`.
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
AETHER_OUTPUT_MODE=visual aether oracle --depth <depth> --confidence-target <percent> --template <template> "<synthesized prompt>"
```

For broad triage, prefer:

```bash
AETHER_OUTPUT_MODE=visual aether oracle --depth quick --confidence-target <percent> --template <template> "<focused triage prompt>"
```

If the shell/tool call times out, run `aether oracle status` before declaring
failure or switching to ad hoc agents.

9. After Oracle completes, suggest persisting high-value findings as pheromones
   or hive wisdom only after user approval.

## Discuss Flow

1. Run:

```bash
AETHER_OUTPUT_MODE=json aether discuss-analyze --target .
```

2. Use the runtime suggestions to formulate codebase-aware questions. Cover
   architecture, dependencies, testing, deployment, performance, and product
   intent where relevant.
3. Present questions honestly. Do not invent answers.
4. Persist answers only through runtime commands such as:

```bash
AETHER_OUTPUT_MODE=visual aether discuss --resolve <id> --answer "<answer>"
```

5. When runtime reports `discussion_status: settled`, route back to `aether plan`.

## Guardrails

- Do not write `.aether/data`, Oracle output, pending decisions, or pheromone
  files by hand.
- Do not make the user run a separate PRD command before Oracle. PRD is an
  Oracle template and synthesis mode.
- Treat shell/tool timeouts as controller interruptions until `aether oracle
  status` proves the runtime is blocked, stopped, or complete.
- If Claude/OpenCode oracle or discuss wrapper behavior changes, update the
  matching YAML, this skill, and `cmd/command_guide.go` together.
