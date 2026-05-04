---
schema_version: "1.0"
id: test-strategy-rubric
kind: rubric
category: rubrics
title: Test Strategy Rubric
description: "Rubric for choosing the right depth and type of tests for Aether changes."
output_types: [test-plan, verification, quality-gate, acceptance-plan]
agent_roles: [watcher, probe, builder, queen, auditor]
task_types: [test, verify, coverage, regression]
task_keywords: [test, coverage, regression, smoke, fixture, unit, integration, evidence, depth, strategy]
workflow_triggers: [build, continue]
priority: high
version: "1.0"
source: "aether-native"
render:
  mode: sections
  max_chars: 3600
  sections: [Use When, Test Depth, Good Test Signals, Bad Test Signals]
---
# Test Strategy Rubric

## Use When

Use this when deciding how much verification a change needs.

For beginners: not every change needs every test, but every risky change needs the right proof.

## Test Depth

### Unit

Use for parsing, matching, scoring, formatting, migration helpers, and small pure functions.

### Integration

Use when multiple modules cooperate, such as install/update sync, prompt assembly, or dispatch preparation.

### CLI Smoke

Use when user-facing commands change.

### Regression Sweep

Use for core runtime, state, distribution, and platform parity changes.

## Good Test Signals

- The test fails before the fix.
- It asserts behavior, not implementation trivia.
- It uses realistic path layouts.
- It covers the user-visible command where relevant.
- It protects a previously broken case.

## Bad Test Signals

- Testing only that a file exists.
- Matching fragile pretty output when JSON exists.
- Ignoring errors from commands.
- Running a broad suite but not the changed path.
