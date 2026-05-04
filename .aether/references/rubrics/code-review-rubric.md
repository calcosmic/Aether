---
schema_version: "1.0"
id: code-review-rubric
kind: rubric
category: rubrics
title: Code Review Rubric
description: "Aether code review rubric focused on correctness, regressions, maintainability, and missing tests."
output_types: [code-review, review-output]
agent_roles: [watcher, auditor, scout, architect, queen]
task_types: [review, code, audit, quality]
task_keywords: [review, code review, bug, regression, maintainability, tests, severity, findings, correctness]
workflow_triggers: [continue, seal]
priority: high
version: "1.0"
source: "aether-native"
render:
  mode: sections
  max_chars: 3600
  sections: [Use When, Severity Order, Review Checklist, Output Shape]
---
# Code Review Rubric

## Use When

Use this when reviewing changed source files or generated runtime behavior.

For beginners: code review should find what can break, not just comment on style.

## Severity Order

1. Correctness bugs.
2. Data loss, security, or destructive behavior.
3. Behavioral regressions.
4. Missing verification for changed behavior.
5. Maintainability risks.
6. Style and naming issues.

## Review Checklist

- Does the implementation satisfy the user request?
- Does it follow existing local patterns?
- Are error paths handled?
- Are state and filesystem operations safe?
- Are platform differences respected?
- Are tests targeted to the changed behavior?
- Is generated or mirrored content kept in sync?
- Is any unrelated refactor mixed in?

## Output Shape

Lead with findings. Include file and line references where possible. If there are no findings, say so and name residual test gaps.
