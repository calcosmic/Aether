---
schema_version: "1.0"
id: performance-gate-rubric
kind: rubric
category: rubrics
title: Performance Gate Rubric
description: "Rubric for reviewing runtime cost, prompt size, filesystem scans, and worker dispatch overhead."
output_types: [performance-review, quality-gate]
agent_roles: [measurer, auditor, watcher, architect, queen]
task_types: [performance, budget, optimization, review]
task_keywords: [performance, budget, latency, scan, prompt, cache, cost, trim, weight, capsule, index]
workflow_triggers: [continue, seal]
priority: normal
version: "1.0"
source: "aether-native"
render:
  mode: sections
  max_chars: 3400
  sections: [Use When, Checks, Warning Signs, Evidence]
---
# Performance Gate Rubric

## Use When

Use this for prompt assembly, recursive scans, index building, worker dispatch, install/update, or repeated CLI commands.

For beginners: this asks whether the change is going to get slow or expensive as Aether grows.

## Checks

- Is the scan bounded or cached?
- Does prompt injection cap content?
- Does the code avoid repeated filesystem walks in hot paths?
- Are indexes invalidated predictably?
- Does matching sort and limit results?
- Are expensive commands avoided in normal status paths?

## Warning Signs

- Full repository scans on every worker prompt.
- Unlimited markdown injection.
- Rebuilding indexes without need.
- Large generated JSON printed in visual paths.
- Tests that pass only on tiny fixtures.

## Evidence

Use timing output, fixture scale tests, cache behavior tests, or code inspection for bounded loops and limits.
