---
schema_version: "1.0"
id: phase-plan-template
kind: template
category: templates
title: Phase Plan Template
description: "Template for an Aether phase plan with dependencies, acceptance, and verification."
output_types: [phase-plan, plan, phase-plan-example]
agent_roles: [queen, architect, route-setter, oracle, builder]
task_types: [plan, phase, roadmap, dependency]
task_keywords: [phase, plan, dependencies, tasks, acceptance, roadmap, scope, verification, risk, worker]
workflow_triggers: [plan, discuss]
priority: high
version: "1.0"
source: "aether-native"
render:
  mode: full
  max_chars: 4200
---
# Phase Plan Template

## Goal

State the outcome in user terms.

## Scope

### In Scope

List what will be changed.

### Out Of Scope

List what will not be changed.

## Assumptions

Name assumptions and how they will be tested.

## Tasks

Use small, dependency-aware tasks:

```json
{
  "goal": "Concrete outcome",
  "depends_on": [],
  "hints": ["likely file or command"],
  "success_criteria": ["observable proof"]
}
```

Do not write an `id` field in `.aether/data/planning/phase-plan.json`. Aether assigns task ids from phase/task array order: phase 1 task 1 is `1.1`, phase 1 task 2 is `1.2`, and phase 2 task 1 is `2.1`. `depends_on` must reference only those runtime ids, never task prose, file paths, or aliases such as `P1-T1`.

## Worker Fit

Name which worker roles are likely needed and why.

## Verification

List phase-level checks, not just generic test commands.

## Risks

Name what could derail the phase.

For beginners: a phase plan is a build map, not a wish list.
