---
schema_version: "1.0"
id: decision-record-template
kind: template
category: templates
title: Decision Record Template
description: "Template for recording architecture, runtime, and distribution decisions."
output_types: [decision-record, adr]
agent_roles: [architect, oracle, queen, chronicler, builder]
task_types: [decision, architecture, record, documentation]
task_keywords: [decision, ADR, rationale, tradeoff, architecture, reversal, consequence, alternative]
workflow_triggers: [plan, build, seal]
priority: normal
version: "1.0"
source: "aether-native"
render:
  mode: full
  max_chars: 3600
---
# Decision Record Template

## Status

Proposed, accepted, rejected, superseded, or deprecated.

## Context

Describe the problem and constraints.

## Decision

State the chosen path clearly.

## Alternatives

List realistic alternatives, including "do nothing" when relevant.

## Consequences

### Positive

What becomes simpler, safer, or more capable?

### Negative

What becomes harder, riskier, or more expensive?

## Verification

How will the decision be validated?

## Reversal Plan

How can Aether back out if the decision proves wrong?

For beginners: this keeps future workers from re-arguing the same choice without context.
