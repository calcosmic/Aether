---
schema_version: "1.0"
id: quality-gate-output-contract
kind: contract
category: contracts
title: Quality Gate Output Contract
description: "Required structure for Watcher, Auditor, Gatekeeper, Probe, and Measurer gate outputs."
output_types: [quality-gate, gate-output, review-output, gate-output-example]
agent_roles: [watcher, auditor, gatekeeper, probe, measurer, queen]
task_types: [gate, review, verify, audit, quality]
task_keywords: [gate, pass, fail, warning, blocker, evidence, review, severity, residual, decision, advisory]
workflow_triggers: [continue, seal]
priority: critical
version: "1.0"
source: "aether-native"
render:
  mode: full
  max_chars: 3600
---
# Quality Gate Output Contract

## Purpose

Gate outputs decide whether Aether advances. They must be precise, evidence-backed, and severity-aware.

For beginners: this is the review verdict format. It prevents vague "looks good" reviews.

## Required Fields

- `decision`: pass, pass_with_warnings, or block.
- `scope_reviewed`: files, commands, artifacts, or workflows inspected.
- `evidence`: tests, logs, code inspection, screenshots, or source references.
- `findings`: ordered by severity.
- `blockers`: required fixes before advancement.
- `warnings`: risks accepted if advancement continues.
- `advisories`: optional improvements.
- `residual_risk`: what remains unproven.

## Rules

- A blocker must include a concrete next action.
- A warning must explain why it does not block.
- An advisory must not be phrased as mandatory.
- Evidence must be fresh enough to cover the latest edit.
- Do not approve if verification was impossible and the task affects runtime behavior.

## Minimum Passing Output

```text
Decision: pass
Scope reviewed: <files or workflow>
Evidence: <commands or inspection>
Findings: none blocking
Residual risk: <remaining gap or none>
```
