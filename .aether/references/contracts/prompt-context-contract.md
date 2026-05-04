---
schema_version: "1.0"
id: prompt-context-contract
kind: contract
category: contracts
title: Prompt Context Contract
description: "Contract for what may be injected into worker prompts and how context should be budgeted."
output_types: [prompt-review, context-review, architecture-review]
agent_roles: [architect, builder, watcher, oracle, queen, scout]
task_types: [prompt, context, injection, budget, worker]
task_keywords: [prompt, context, skill, reference, pheromone, budget, injection, trim, weight, capsule, colony-prime]
workflow_triggers: [build, continue, oracle]
priority: high
version: "1.0"
source: "aether-native"
render:
  mode: full
  max_chars: 4000
---
# Prompt Context Contract

## Purpose

Worker prompts combine colony-prime context, skills, references, pheromones, handoffs, and task details. Each source has a different trust level and budget.

For beginners: the prompt is the worker's briefing packet. It needs the right facts, not every fact.

## Context Classes

- Colony state: current goal, phase, tasks, blockers.
- Pheromones: user steering signals.
- Skills: behavior and domain guidance.
- References: structure, rubrics, templates, examples.
- Handoffs: recent worker relay notes.
- External docs: only when explicitly relevant and trustworthy.

## Budget

Colony-prime prompt section has a character budget:

- Normal: 6000 characters.
- Compact: 3000 characters (triggered by `--compact` flag or auto-detection).

### Trim Order

When the budget is exceeded, sections are trimmed in this order (first trimmed = lowest priority):

1. ROLLING SUMMARY
2. PHASE LEARNINGS
3. KEY DECISIONS
4. HIVE WISDOM
5. PROJECT REQUIREMENTS
6. CONTEXT CAPSULE
7. QUEEN WISDOM (Global)
8. QUEEN WISDOM (Local)
9. PROJECT BRAIN CORE
10. USER PREFERENCES
11. ACTIVE SIGNALS

BLOCKERS is NEVER trimmed.

## Rules

- Keep references precise: top 2 matches (scoring: output_type = 4, role = 3, task = 2).
- Do not let references replace evidence.
- Keep skills and references separate in meaning, even if rendered together.
- Preserve high-priority blockers before broad background.
- Avoid injecting stale local state.
- Label inferred context as inference.

## Review Questions

1. Is this context relevant to the worker's task?
2. Is it trusted, stale, user-provided, generated, or inferred?
3. Does it fit within the budget?
4. Should this be a skill, a reference, a state artifact, or a doc?
5. Does trim order preserve safety-critical content?
