---
schema_version: "1.0"
id: bug-investigation-template
kind: template
category: templates
title: Bug Investigation Template
description: "Structured report format for reproducing, explaining, and fixing bugs."
output_types: [bug-investigation, root-cause-analysis]
agent_roles: [tracker, oracle, watcher, builder, scout]
task_types: [bug, debug, investigate, regression]
task_keywords: [bug, root cause, reproduce, failing, regression, investigate, symptom, hypothesis, diagnose]
workflow_triggers: [build, oracle, continue]
priority: high
version: "1.0"
source: "aether-native"
render:
  mode: full
  max_chars: 4200
---
# Bug Investigation Template

## Symptom

Describe the visible failure. Include command, workflow, file, user action, or test name.

## Impact

Explain what the bug prevents or risks.

For beginners: this is why the bug matters.

## Reproduction

State the smallest known reproduction:

```text
Command:
Expected:
Actual:
```

If not reproducible, say what evidence exists and what is missing.

## Timeline

List relevant recent changes, commits, generated files, user corrections, or state transitions.

## Hypotheses

Use a table:

| Hypothesis | Evidence For | Evidence Against | Status |
|---|---|---|---|

## Root Cause

Name the exact mechanism. Avoid vague causes like "state issue" or "race condition" unless the evidence proves it.

## Fix Boundary

State what should change and what must remain untouched.

## Verification

List the command or inspection that proves the original failure is fixed.

## Residual Risk

Name what remains unproven.
