---
schema_version: "1.0"
id: risk-and-pitfall-field-guide
kind: field-guide
category: field-guides
title: Risk And Pitfall Field Guide
description: "Guide for identifying likely traps before implementation starts."
output_types: [risk-register, pitfall-summary, plan-review]
agent_roles: [architect, oracle, scout, watcher, queen, builder]
task_types: [risk, pitfall, plan, review]
task_keywords: [risk, pitfall, failure mode, trap, assumption, regression, mitigate, state, distribution, prompt]
workflow_triggers: [discuss, plan, build]
priority: high
version: "1.0"
source: "aether-native"
render:
  mode: sections
  max_chars: 3800
  sections: [Use When, Pitfall Categories, Output, Review Use]
---
# Risk And Pitfall Field Guide

## Use When

Use this before implementation when the task crosses state, distribution, platform, prompt, or user-facing workflow boundaries.

For beginners: this is the "what could go wrong?" scan before building.

## Pitfall Categories

- source versus mirror confusion
- hub versus repo confusion
- stale generated files
- state corruption
- weak verification
- hidden platform drift
- prompt bloat
- unsafe shell or file operations
- broad refactor mixed into a narrow change
- old docs contradicting current code

## Output

Use a table:

| Risk | Why It Matters | Mitigation | Verification |
|---|---|---|---|

## Review Use

Risks are not blockers by default. They become blockers when unmitigated and likely enough to affect correctness, safety, or user trust.
