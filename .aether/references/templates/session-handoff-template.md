---
schema_version: "1.0"
id: session-handoff-template
kind: template
category: templates
title: Session Handoff Template
description: "Template for pausing Aether work with enough context to resume safely."
output_types: [session-handoff, handoff]
agent_roles: [builder, watcher, queen, chronicler, scout]
task_types: [handoff, session, resume, pause]
task_keywords: [handoff, resume, pause, session, next steps, freshness, context, do not repeat, recovery]
workflow_triggers: [build, continue, seal]
priority: normal
version: "1.0"
source: "aether-native"
render:
  mode: full
  max_chars: 3600
---
# Session Handoff Template

## Current Goal

State the active user goal and phase.

## What Changed

List changed files and behavior.

## What Was Verified

List commands, tests, inspections, and outcomes.

## Known Failures

Name failures directly. Include whether they are related or unrelated.

## Open Decisions

List decisions still needed from user, Queen, or next worker.

## Next Action

Give the next worker one bounded action.

## Do Not Repeat

List dead ends, rejected approaches, and user-corrected assumptions.

## Freshness

State whether evidence was collected after the latest edits.

For beginners: this is the note that lets work resume without guessing.
