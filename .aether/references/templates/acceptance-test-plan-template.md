---
schema_version: "1.0"
id: acceptance-test-plan-template
kind: template
category: templates
title: Acceptance Test Plan Template
description: "Template for mapping acceptance criteria to concrete verification commands."
output_types: [test-plan, acceptance-plan]
agent_roles: [watcher, probe, builder, queen, auditor]
task_types: [test, acceptance, verification, plan]
task_keywords: [acceptance, test plan, verification, criteria, prove, fixture, negative, regression, evidence]
workflow_triggers: [plan, build, continue]
priority: high
version: "1.0"
source: "aether-native"
render:
  mode: full
  max_chars: 3800
---
# Acceptance Test Plan Template

## Acceptance Criteria

List each requirement as an observable statement.

## Test Mapping

Use this table:

| Criterion | Test/Command | Evidence | Owner | Status |
|---|---|---|---|---|

## Required Fixtures

Name fixture files, temp repos, state files, or mocked inputs needed.

## Negative Cases

List what must fail safely:

- invalid input
- missing files
- stale state
- permission errors
- platform mismatch

## Regression Surface

Name existing behavior that must not break.

## Completion Standard

State the minimum evidence needed to call the work complete.

For beginners: this connects "what should be true" to "how we prove it."
