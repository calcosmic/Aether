---
schema_version: "1.0"
id: quality-gate-template
kind: template
category: templates
title: Quality Gate Template
description: "Template for pass/block gate results during continue and seal."
output_types: [quality-gate, gate-output, gate-output-example]
agent_roles: [watcher, auditor, gatekeeper, probe, measurer, queen]
task_types: [gate, verify, audit, continue]
task_keywords: [gate, pass, block, warning, evidence, continue, severity, blocker, advisory, residual, decision]
workflow_triggers: [continue, seal]
priority: critical
version: "1.0"
source: "aether-native"
render:
  mode: full
  max_chars: 3600
---
# Quality Gate Template

## Decision

Choose one:

- `pass`
- `pass_with_warnings`
- `block`

## Scope

Name what was reviewed:

- files
- commands
- generated artifacts
- workflows
- state files

## Evidence

List fresh evidence:

```text
Command/inspection:
Result:
What it proves:
```

## Findings

### Blockers

Issues that prevent advancement.

### Warnings

Known risks that can advance with explicit acceptance.

### Advisories

Non-blocking improvements.

## Required Next Action

If blocked, state the exact next action. If passed, state the next lifecycle command.

For beginners: this is the stop/go report.
