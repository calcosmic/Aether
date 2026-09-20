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

## Installed Codex entrypoints

Use `$ant-init`. This document is installed as
`support/aether-colony-creation.md` beside the public skill directories;
it is private support, not a separately discoverable helper skill. Public skills
resolve its path relative to their installed `SKILL.md`, never the working directory.
The executable commands and runtime-issued IDs remain `aether` identities.
Full workflow coverage and native-worker parity remain pending.

## Purpose

Give Codex the same intelligent init behavior that Claude Code and OpenCode get
from their guided wrappers through the public `$ant-init` skill.
Codex owns the clarification conversation; the Go runtime owns every durable
fact and mutation.

For beginners: the binary can create files, but Codex can ask the human better
questions first. This skill is the question-and-synthesis layer.

Go owns setup, registry updates, accepted-charter persistence, colony state creation, territory evidence, and init result truth.

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

## Stage 1 — Queen opening

1. Read the owner's raw goal and name the repository being initialized.
2. Run deterministic context gathering:

```bash
AETHER_OUTPUT_MODE=json aether init-research --goal "<raw goal>" --target .
```

If the runtime reports an existing active colony, relay its identity and goal,
then stop on its native status or seal guidance.

An existing active colony is refused before storage opens; the refusal changes no files.

## Stage 2 — Setup

Do not require a separate setup command. The final `aether init` call performs
safe automatic bootstrap and reports `Ready`, `Bootstrapped`, or an actionable
failure. Do not reproduce setup, registry, or storage work in this skill.

## Stage 3 — Accepted intent

1. Ask one compact batch of 4-7 questions when the goal is vague or broad.
   Cover target users, success criteria, non-goals, constraints, risk tolerance,
   affected systems, and the first useful milestone.
2. Synthesize the raw goal, answers, and init-research findings into:
   - `refined_goal`
   - `problem_statement`
   - `success_criteria`
   - `constraints`
   - `non_goals`
   - `risks`
   - `first_milestone`
3. Ask the user to choose the colony mode before state creation:
   - Colony Mode: the existing default lifecycle with fewer prompts.
   - Orchestrator Mode: guided boundary questions at phase points for tighter
     user control.
   Default to Colony Mode when the user skips the choice or the host is
   non-interactive.
4. Keep deterministic housekeeping separate from strategy. README, changelog,
   license, formatter, or CI suggestions are scan warnings, not strategic
   pheromones.
5. Suggest at most 3 strategic pheromones only when they are specific to the
   clarified user intent. Ask approval before writing any signal.
6. Start the colony through the runtime:

```bash
AETHER_OUTPUT_MODE=visual aether init --colony-mode <selected colony|orchestrator> --charter-json '<synthesized charter JSON>' "<refined goal>"
```

Persist the owner-approved goal and material constraints as accepted-charter/v1 through aether init.

## Stage 4 — Territory

Read the territory outcome from the successful runtime result; do not inspect
survey files or infer freshness in the skill.

Territory result is exactly one of Fresh, Refreshed, Stale—refresh required, or Unavailable.

Relay its evidence and any safe recovery guidance exactly.

## Stage 5 — Closeout

Summarize the colony name, accepted goal, runtime-created artifacts, and typed
territory result. Display `$ant-plan` as the next public Codex action; its
executable route remains `aether plan`.

## Guardrails

- Do not write `.aether/data/COLONY_STATE.json`, `session.json`, handoff files,
  or pheromone files by hand.
- Do not treat `init-research` as the final charter. It is only deterministic
  context.
- Do not keep asking serial questions. Ask one compact batch, then synthesize.
- If Claude/OpenCode init wrapper behavior changes, update
  `.aether/commands/init.yaml`, this skill, and `cmd/command_guide.go` together.

Next Up: $ant-plan
