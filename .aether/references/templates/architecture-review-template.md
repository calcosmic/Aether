---
schema_version: "1.0"
id: architecture-review-template
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
# Architecture Review Template

## Decision Summary

State the architecture decision in one paragraph. Name what changes and what remains stable.

For beginners: this is the "what shape should the system have?" answer.

## Current System Map

Describe the existing code path:

- Entry commands.
- Core runtime modules.
- State files.
- Companion files.
- Platform-specific surfaces.
- Install/update/publish behavior if affected.

## Proposed Shape

Explain the new structure and why it fits existing Aether patterns. Prefer small, named boundaries over broad rewrites.

## Ownership Boundaries

Define which module owns:

- Data model.
- Matching or routing.
- Prompt rendering.
- File distribution.
- User-local state.
- Tests and fixtures.

## Tradeoffs

Include:

- What gets simpler.
- What gets more complex.
- What future changes become easier.
- What failure modes are introduced.
- What migration cost exists.

## Alternatives Considered

Compare at least two alternatives, including "do nothing" when relevant.

## Validation Plan

Name the tests, smoke commands, and manual inspections needed to prove the architecture works.

## Decision

End with one of:

- `Proceed`.
- `Proceed with constraints`.
- `Spike first`.
- `Reject`.
