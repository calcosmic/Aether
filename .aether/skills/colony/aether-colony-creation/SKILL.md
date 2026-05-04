---
source: shipped
name: aether-colony-creation
description: Use when Codex is asked to initialize or set up an Aether colony and should refine intent before running init
type: colony
domains: [aether, codex, lifecycle, init, orchestration]
agent_roles: [queen, scout, architect, route_setter]
workflow_triggers: [init, lay-eggs]
task_keywords: [aether init, initialize colony, new colony, charter, intent refinement, colony setup]
priority: high
version: "1.0"
---

# Aether Colony Creation

## Purpose

Give Codex the same intelligent init behavior that Claude Code and OpenCode get
from slash-command wrappers. The Go runtime still owns state. Codex owns the
conversation before the state is created.

For beginners: the binary can create files, but Codex can ask the human better
questions first. This skill is the question-and-synthesis layer.

## Required First Step

Run or inspect:

```bash
aether command-guide init --platform codex
```

Use that result as the source of truth for current orchestration steps. If this
skill and `command-guide` disagree, follow `command-guide` and update the skill.

## Raw Bypass

If the user explicitly says raw, exact, no interview, no orchestration, or "just
run this exact command", run the literal CLI command they provided. Say briefly
that the Codex synthesis layer was bypassed.

## Init Flow

1. Read the user's raw goal.
2. Run deterministic context gathering:

```bash
AETHER_OUTPUT_MODE=json aether init-research --goal "<raw goal>" --target .
```

3. Ask one compact batch of 4-7 questions when the goal is vague or broad.
   Cover target users, success criteria, non-goals, constraints, risk tolerance,
   affected systems, and the first useful milestone.
4. Synthesize the raw goal, answers, and init-research findings into:
   - `refined_goal`
   - `problem_statement`
   - `success_criteria`
   - `constraints`
   - `non_goals`
   - `risks`
   - `first_milestone`
5. Keep deterministic housekeeping separate from strategy. README, changelog,
   license, formatter, or CI suggestions are scan warnings, not strategic
   pheromones.
6. Suggest at most 3 strategic pheromones only when they are specific to the
   clarified user intent. Ask approval before writing any signal.
7. Start the colony through the runtime:

```bash
AETHER_OUTPUT_MODE=visual aether init --charter-json '<synthesized charter JSON>' "<refined goal>"
```

## Guardrails

- Do not write `.aether/data/COLONY_STATE.json`, `session.json`, handoff files,
  or pheromone files by hand.
- Do not treat `init-research` as the final charter. It is only deterministic
  context.
- Do not keep asking serial questions. Ask one compact batch, then synthesize.
- If Claude/OpenCode init wrapper behavior changes, update
  `.aether/commands/init.yaml`, this skill, and `cmd/command_guide.go` together.
