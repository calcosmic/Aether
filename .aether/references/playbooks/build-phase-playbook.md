---
schema_version: "1.0"
id: build-phase-playbook
kind: playbook
category: playbooks
title: Build Phase Playbook
description: "Operational playbook for executing an Aether build phase safely."
output_types: [build-plan, workflow-playbook]
agent_roles: [queen, builder, watcher, architect, scout]
task_types: [build, phase, dispatch, execution]
task_keywords: [build, phase, worker, dispatch, task, verification, wave, decision, execution-policy, preflight]
workflow_triggers: [build]
priority: high
version: "1.0"
source: "aether-native"
render:
  mode: full
  max_chars: 4600
---
# Build Phase Playbook

## Use When

Use this when preparing or reviewing `aether build <phase>` behavior.

For beginners: build turns a planned phase into executed worker tasks.

## Preflight

- Confirm current repo and active colony.
- Load state and verify a phase exists.
- Respect active pheromones and user corrections.
- Check phase dependencies and selected tasks.
- Decide Queen execution policy from risk. The Queen classifies each gate into one of four tiers: `hard_block` (stops work, escalates), `soft_block` (eligible for auto-resolve if budget remains), `advisory` (non-blocking), or unclassified. Queen recommendations are: `pass`, `auto-resolve`, `dispatch-fixer`, or `escalate`. Auto-resolve is eligible only when tier is `soft_block` and gate status is `failed`.

## Dispatch

- Assign bounded tasks.
- Preserve dependencies.
- Include colony-prime context, skills, references, and relevant handoffs.
- Keep workers aware they are not alone in the codebase.
- Use Watcher or specialist review when risk requires it.

## During Build

- Stop on failed dependencies.
- Record worker reports.
- Capture changed files and commands.
- Avoid touching protected local state unless the task is explicitly state work.

## Completion

- Update build artifacts.
- Record claims.
- Prepare continue verification.
- Do not mark a phase complete just because dispatch finished.

## Failure Handling

If a worker fails, report the worker, task, command, and next recovery option.
