---
schema_version: "1.0"
id: worker-dispatch-field-guide
kind: field-guide
category: field-guides
title: Worker Dispatch Field Guide
description: "Guide for designing and reviewing worker dispatch, handoff, and context behavior."
output_types: [dispatch-review, architecture-review]
agent_roles: [queen, architect, builder, watcher, scout]
task_types: [dispatch, worker, prompt, handoff]
task_keywords: [dispatch, worker, handoff, context, prompt, wave, colony-prime, skill, reference, assignment]
workflow_triggers: [build, continue]
priority: high
version: "1.0"
source: "aether-native"
render:
  mode: sections
  max_chars: 4200
  sections: [Use When, Dispatch Concerns, Handoff Concerns, Verification]
---
# Worker Dispatch Field Guide

## Use When

Use this when changing worker assignment, prompt assembly, wave planning, task dispatch, or result handling.

For beginners: this is how Aether gives the right job to the right worker and gets useful results back.

## Dispatch Concerns

- role matches task risk
- task scope is bounded
- dependencies are preserved
- context is relevant and fresh
- references and skills are injected only when helpful
- workers know they are not alone in the codebase
- failures stop dependent work

## Handoff Concerns

- changed files are named
- commands and outcomes are recorded
- known failures are honest
- assumptions are separated from facts
- next action is bounded
- stale evidence is identified

## Verification

Use tests or smoke commands that inspect dispatch records, worker briefs, prompt sections, and state updates.
