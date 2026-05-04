---
schema_version: "1.0"
id: verification-evidence-ladder
kind: rubric
category: rubrics
title: Verification Evidence Ladder
description: "Evidence standard for proving Aether work is actually complete."
output_types: [quality-gate, verification, completion-review, gate-output]
agent_roles: [watcher, probe, auditor, queen, builder]
task_types: [verify, test, evidence, complete, continue]
task_keywords: [verified, evidence, pass, fail, regression, acceptance, complete, done, ladder, severity, prove]
workflow_triggers: [build, continue, seal]
priority: critical
version: "1.0"
source: "aether-native"
render:
  mode: sections
  max_chars: 4200
  sections: [Use When, Evidence Ladder, Failure Signals, Output Requirements]
---
# Verification Evidence Ladder

## Use When

Use this whenever a worker claims a phase, task, fix, migration, command, wrapper, or generated artifact is complete. The goal is to prevent "looks done" from being treated as "proven done."

For beginners: this is the checklist that says what kind of proof is strong enough. A screenshot, test run, command output, file diff, and reviewer note are not equal evidence.

## Evidence Ladder

### Level 0: Claim Only

The worker says it is done but gives no reproducible proof. This is not acceptable for completion.

Required response: ask for evidence or mark verification incomplete.

### Level 1: Static Inspection

The changed files were inspected and the approach appears coherent. This is useful but weak because runtime behavior is still unproven.

Acceptable for: tiny documentation edits, generated wrappers with no executable path, or planning-only artifacts.

### Level 2: Targeted Command

A focused command was run against the changed area, such as a unit test package, a CLI subcommand, a linter, or a renderer smoke test.

Acceptable for: narrow code changes when the command directly exercises the behavior.

### Level 3: End-to-End Workflow

The user-facing workflow was executed through the same entrypoint the user will run, such as `aether build`, `aether update`, `aether oracle`, or a generated wrapper command.

Acceptable for: CLI behavior, install/update paths, prompt injection, and anything crossing module boundaries.

### Level 4: Regression Sweep

Focused verification plus broader regression coverage ran cleanly. Examples: package tests plus full `go test ./cmd/...`, wrapper generation plus install dry-run, or CLI smoke tests plus fixture checks.

Acceptable for: core runtime, distribution, state migration, worker dispatch, and quality gates.

## Failure Signals

- Evidence is from a stale command run before the latest edit.
- The command proves formatting but not behavior.
- The changed code path is behind a different platform or workflow than the one tested.
- The worker reports "should work" or "likely fixed" without command output.
- Tests pass only because the test data avoids the changed branch.

## Output Requirements

Verification summaries must name:

1. The command or inspection performed.
2. The behavior it proves.
3. The remaining gap, if any.
4. Whether the evidence is enough to advance the phase.
