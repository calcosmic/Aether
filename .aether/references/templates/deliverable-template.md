---
schema_version: "1.0"
id: deliverable-template
kind: template
category: templates
title: Deliverable Template
description: "Concise final output structure for completed Aether work."
output_types: [deliverable, output, final-response]
agent_roles: [builder, watcher, chronicler, queen, scout]
task_types: [deliver, summarize, complete, report]
task_keywords: [deliverable, summary, final, changed, verification, complete, receipt, outcome, residual]
workflow_triggers: [build, continue, seal]
priority: normal
version: "1.0"
source: "aether-native"
render:
  mode: full
  max_chars: 3400
---
# Deliverable Template

## Purpose

Use this when reporting completed work to the user or Queen.

For beginners: this is the short receipt. It says what changed, why it matters, and what proof exists.

## Required Shape

### Outcome

State the result in user terms.

### What Changed

Name the important files or behaviors changed. Do not list every tiny file when a grouped summary is clearer.

### Why It Matters

Explain the practical effect for the user or future workers.

### Verification

List commands run and whether they passed. If something was not run, say why.

### Residual Risk

Name any known gap, flaky area, blocked test, or follow-up that matters.

## Style Rules

- Keep it short.
- Lead with what the user can now do or trust.
- Do not hide failures.
- Do not end with vague offers.
