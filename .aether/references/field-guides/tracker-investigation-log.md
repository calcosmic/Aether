---
schema_version: "1.0"
id: tracker-investigation-log
kind: field-guide
category: field-guides
title: Tracker Investigation Log
description: "Bug investigation structure for Aether Tracker and Oracle workers."
output_types: [bug-investigation, root-cause-analysis, failure-analysis]
agent_roles: [tracker, oracle, watcher, scout, builder]
task_types: [bug, investigate, debug, failure, regression]
task_keywords: [bug, failure, flaky, regression, broken, root cause, reproduce, hypothesis, symptom, diagnose, debug]
workflow_triggers: [build, continue, oracle]
priority: high
version: "1.0"
source: "aether-native"
render:
  mode: sections
  max_chars: 4200
  sections: [Use When, Investigation Log, Hypothesis Discipline, Exit Criteria]
---
# Tracker Investigation Log

## Use When

Use this for bugs, regressions, stale-state problems, install/update failures, worker dispatch failures, and confusing test failures.

For beginners: this keeps debugging from becoming random poking. Each idea gets tested, then kept or discarded.

## Investigation Log

Record the investigation in this shape:

### Symptom

Describe what failed, where it appears, and who experiences it. Include the exact command, workflow, or user action.

### Reproduction

State whether the failure is reproducible. If it is, list the smallest command or steps. If not, list what evidence exists.

### Expected Behavior

Describe what should have happened in user terms and system terms.

### Observed Behavior

Name the file, state artifact, log, output, or test result proving the actual behavior.

### Candidate Causes

List hypotheses with evidence for and against each one. Do not collapse multiple causes into one vague theory.

### Root Cause

State the smallest mechanism that explains the symptom and the evidence.

### Fix Boundary

Name what should change and what should not change.

## Hypothesis Discipline

- Do not patch before identifying the failing path unless the bug is trivial.
- Do not treat correlation as root cause.
- Prefer a focused reproduction over broad speculation.
- After each test, update the hypothesis list.

## Exit Criteria

The investigation is complete only when:

1. The cause explains the symptom.
2. The fix boundary is clear.
3. Verification covers the original failure.
4. Residual risks are named.
