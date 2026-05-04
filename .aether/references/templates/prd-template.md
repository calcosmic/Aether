---
schema_version: "1.0"
id: prd-template
kind: template
category: templates
title: Product Requirements Document Template
description: "Structure for turning research or user intent into actionable product requirements."
output_types: [prd, requirements, product-spec]
agent_roles: [oracle, architect, queen, route-setter, scout]
task_types: [research, planning, requirements, product]
task_keywords: [prd, requirements, user stories, acceptance, product, spec, define, scope, goals]
workflow_triggers: [discuss, plan, oracle]
priority: high
version: "1.0"
source: "aether-native"
render:
  mode: full
  max_chars: 4600
---
# Product Requirements Document Template

## Problem Statement

Explain the user problem in plain language. Avoid implementation details until the problem is clear.

For beginners: this is the "why anyone should care" section.

## Target Users

Name the people or agents affected by the change. Include direct users, maintainers, and downstream workers if relevant.

## Goals

List what must become true when the work is complete.

## Non-Goals

List what is intentionally out of scope. This protects the implementation from drifting.

## User Stories

Use this shape:

1. As a `<user or worker>`, I want `<capability>`, so that `<outcome>`.

Each story must connect to an observable behavior.

## Functional Requirements

State concrete behaviors:

- Inputs accepted.
- Outputs produced.
- Files or state affected.
- CLI flags or commands involved.
- Platform parity requirements.

## Safety Requirements

State protected paths, overwrite rules, migration constraints, and destructive-operation boundaries.

## Implementation Notes

Name likely modules, interfaces, and existing patterns. Do not over-design. The implementation section should guide builders without pretending every line is known.

## Acceptance Criteria

List checks that prove completion. Include at least one user-facing workflow check when behavior crosses a CLI or distribution boundary.

## Risks And Open Questions

Name assumptions, unknowns, and how they will be resolved.
