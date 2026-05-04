---
schema_version: "1.0"
id: clarity-gate
kind: rubric
category: rubrics
title: Clarity Gate
description: "Rubric for deciding whether a plan, spec, or handoff is clear enough to execute."
output_types: [planning-review, spec-review, handoff-review, clarity-gate]
agent_roles: [queen, architect, route-setter, watcher, oracle, builder]
task_types: [plan, spec, review, handoff, clarify]
task_keywords: [ambiguous, assumptions, unclear, scope, plan, spec, handoff, acceptance, vague, ready, execute]
workflow_triggers: [discuss, plan, build, continue]
priority: high
version: "1.0"
source: "aether-native"
render:
  mode: sections
  max_chars: 3600
  sections: [Use When, Pass Criteria, Common Clarity Failures, Required Repairs]
---
# Clarity Gate

## Use When

Use this before execution when a plan, spec, research brief, worker assignment, or handoff may be too vague to implement safely.

For beginners: this is the "do we actually know what to build?" check.

## Pass Criteria

A plan or handoff is clear enough when it states:

- The user-facing outcome.
- The files, modules, commands, or workflows likely involved.
- The acceptance criteria that prove completion.
- The constraints that must not be violated.
- The open assumptions and how they will be tested.

For engineering work, "make it better" is not enough. The work needs an observable before-and-after result.

## Common Clarity Failures

- The task describes activity instead of outcome.
- The plan names agents but not responsibilities.
- Verification is generic, such as "run tests", without naming what the tests prove.
- External constraints are implied but not recorded.
- A handoff says "continue from here" but does not say what changed or what remains.
- There is no explicit out-of-scope boundary.

## Required Repairs

When clarity fails, repair the plan before execution:

1. Restate the user outcome in one sentence.
2. List the concrete surfaces touched.
3. Add acceptance checks.
4. Mark unknowns as assumptions, not facts.
5. Assign the first worker a bounded task with a clear stop condition.
