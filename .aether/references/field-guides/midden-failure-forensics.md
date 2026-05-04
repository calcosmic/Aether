---
schema_version: "1.0"
id: midden-failure-forensics
kind: field-guide
category: field-guides
title: Midden Failure Forensics
description: "Guide for using failure records to avoid repeating known Aether mistakes."
output_types: [failure-analysis, recovery-plan]
agent_roles: [tracker, medic, watcher, queen, builder]
task_types: [failure, forensics, recovery, debug]
task_keywords: [midden, failure, repeated, recovery, test failure, blocker, acknowledge, forensics, history]
workflow_triggers: [build, continue, plan]
priority: high
version: "1.0"
source: "aether-native"
render:
  mode: sections
  max_chars: 3800
  sections: [Use When, What To Look For, How To Use Findings, Output]
---
# Midden Failure Forensics

## Use When

Use this when a command, test, worker, update, or plan fails in a way that may have happened before.

For beginners: midden is the failure notebook. Read it before repeating the same mistake.

## What To Look For

- recent command failures
- repeated test failures
- known stale state
- previous abandoned approaches
- user corrections after bad assumptions
- failed fixes that looked plausible

## How To Use Findings

- Treat old failures as leads, not proof.
- Check whether the code has changed since the failure.
- Avoid retrying rejected approaches.
- Convert repeated failures into a targeted test or guard.
- Record new failure details if the issue persists.

## Output

State:

- relevant prior failures
- whether they still apply
- likely cause
- repair plan
- verification
