---
schema_version: "1.0"
id: planning-source-audit
kind: field-guide
category: field-guides
title: Planning Source Audit
description: "How planners inspect existing Aether state, docs, and code before proposing work."
output_types: [phase-plan, planning-review, source-audit, source-audit-example]
agent_roles: [queen, architect, route-setter, scout, oracle]
task_types: [plan, research, audit, architecture, roadmap]
task_keywords: [plan, phase, roadmap, architecture, source, existing, state, docs, inspect, truth, priority]
workflow_triggers: [discuss, colonize, plan]
priority: high
version: "1.0"
source: "aether-native"
render:
  mode: sections
  max_chars: 4200
  sections: [Use When, Source Order, Audit Questions, Output Shape]
---
# Planning Source Audit

## Use When

Use this before creating or reviewing an Aether plan. It is especially important when the user references existing files, previous colonies, global hub behavior, agent definitions, command wrappers, or distribution pipelines.

For beginners: this tells the planner where to look before inventing a plan. Plans should come from the real system, not guesses.

## Source Order

Inspect sources in this order:

1. Current user instruction and recent corrections.
2. Current git worktree and changed files.
3. Aether runtime code under `cmd/` and shared packages.
4. Source-of-truth companion files under `.aether/`.
5. Platform surfaces under `.claude/`, `.opencode/`, and `.codex/`.
6. Distributed docs and command playbooks.
7. Hub behavior only as a distribution target, not as source truth.

If sources disagree, the newest explicit user correction wins unless it would corrupt state or contradict executable code.

## Audit Questions

- What is the source of truth?
- What is generated, mirrored, installed, or user-local?
- Which platform surfaces must stay in parity?
- Does this change affect target repositories, the global hub, or only the Aether source repo?
- What files are protected from update/install overwrite?
- What tests or smoke commands prove the pipeline still works?

## Output Shape

A planning source audit should produce:

- `Facts`: observations with file references.
- `Assumptions`: beliefs that still need verification.
- `Risks`: how the plan could damage users or confuse agents.
- `Decision`: the recommended implementation boundary.
- `Verification`: commands or inspections needed before completion.
