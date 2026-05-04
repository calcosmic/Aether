---
schema_version: "1.0"
id: documentation-accuracy-rubric
kind: rubric
category: rubrics
title: Documentation Accuracy Rubric
description: "Rubric for verifying that Aether docs match runtime behavior and source-of-truth files."
output_types: [documentation-review, source-audit]
agent_roles: [chronicler, watcher, scout, queen, architect]
task_types: [documentation, review, audit, source]
task_keywords: [docs, documentation, README, runbook, accurate, stale, drift, verify, count, command]
workflow_triggers: [build, continue, update]
priority: high
version: "1.0"
source: "aether-native"
render:
  mode: sections
  max_chars: 3400
  sections: [Use When, Checks, Blockers, Output]
---
# Documentation Accuracy Rubric

## Use When

Use this when docs, runbooks, command help, platform guides, or source-of-truth maps change.

For beginners: docs are only useful if they describe what the system really does.

## Checks

- Commands exist and flags match runtime help.
- Paths match current repo layout.
- Source/mirror/generated roles are clear.
- Platform-specific statements are accurate.
- Hub versus repo language is not confused.
- Protected state rules match update code.
- Counts are updated when inventories change.

## Blockers

- Docs tell users to edit generated or installed files as source.
- Docs claim a command exists when runtime does not.
- Docs imply target repos receive global-only assets.
- Docs hide known verification failures.

## Output

State which claims were verified, which were corrected, and which need follow-up.
