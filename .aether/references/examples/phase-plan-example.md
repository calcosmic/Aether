---
schema_version: "1.0"
id: phase-plan-example
kind: example
category: examples
title: Phase Plan Example
description: "Example of a small Aether phase plan with acceptance criteria."
output_types: [phase-plan-example, plan]
agent_roles: [queen, route-setter, architect, builder]
task_types: [plan, example, phase]
task_keywords: [example, phase, plan, tasks, acceptance, dependency, verification, scope]
workflow_triggers: [plan]
priority: normal
version: "1.0"
source: "aether-native"
render:
  mode: full
  max_chars: 3200
---
# Phase Plan Example

## Goal

Make the reference library global-only and usable by Oracle.

## Tasks

```json
[
  {
    "id": "1.1",
    "goal": "Replace generic REFERENCE.md layout with named category files",
    "hints": [".aether/references/{category}/{id}.md"],
    "success_criteria": ["reference-list shows named files"]
  },
  {
    "id": "1.2",
    "goal": "Update runtime matching for named files",
    "depends_on": ["1.1"],
    "success_criteria": ["reference-match returns oracle-tech-evaluation for tech-eval"]
  },
  {
    "id": "1.3",
    "goal": "Prevent target repo reference sync",
    "depends_on": ["1.2"],
    "success_criteria": ["update test proves no .aether/references in target repo"]
  }
]
```

## Verification

- build passes
- focused reference tests pass
- CLI smoke checks pass
