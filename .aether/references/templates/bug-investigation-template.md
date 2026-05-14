---
schema_version: "1.0"
id: bug-investigation-template
template: bug-investigation
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
# Bug Investigation: {{title}}

## Symptoms
<!-- What was observed -->

## Root Cause Analysis
<!-- Why it happened -->

## Fix Applied
<!-- What was changed -->

## Verification Steps
<!-- How we confirmed the fix -->

## Prevention
<!-- How to prevent recurrence -->
