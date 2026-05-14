---
schema_version: "1.0"
id: architecture-review-template
template: architecture-review
kind: template
category: templates
title: Architecture Review Template
description: "Structure for reviewing Aether architecture changes and cross-module tradeoffs."
output_types: [architecture-review, architecture-decision]
agent_roles: [oracle, architect, queen, auditor, builder]
task_types: [architecture, review, design, refactor]
task_keywords: [architecture, design, module, boundary, abstraction, tradeoff, refactor, decision, ownership, migration]
workflow_triggers: [plan, build, oracle]
priority: high
version: "1.0"
source: "aether-native"
render:
  mode: full
  max_chars: 5000
---
# Architecture Review: {{title}}

## Context
<!-- Current state and problem statement -->

## Constraints
<!-- Technical, business, and team constraints -->

## Options Considered
<!-- List of architectural options -->

## Trade-off Analysis
<!-- Pros/cons of each option -->

## Decision
<!-- Selected option with rationale -->
