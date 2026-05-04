---
schema_version: "1.0"
id: state-safety-rubric
kind: rubric
category: rubrics
title: State Safety Rubric
description: "Rubric for reviewing Aether state file reads, writes, migrations, locks, and recovery behavior."
output_types: [state-review, migration-review, safety-review]
agent_roles: [watcher, medic, auditor, architect, queen, fixer, builder]
task_types: [state, migration, storage, recovery, safety]
task_keywords: [state, migration, lock, atomic, JSON, recovery, corruption, protected, fixture, COLONY_STATE]
workflow_triggers: [build, continue, update]
priority: critical
version: "1.0"
source: "aether-native"
render:
  mode: sections
  max_chars: 3800
  sections: [Use When, Review Checks, Blockers, Evidence]
---
# State Safety Rubric

## Use When

Use this when state files, storage helpers, lifecycle advancement, recovery, locks, or migrations change.

For beginners: this is the "will this corrupt the colony?" review.

## Review Checks

- Does the code parse state structurally?
- Are writes atomic?
- Is locking used when shared state can be touched concurrently?
- Are protected files excluded from install/update?
- Can old state still load?
- Are unknown fields preserved or intentionally migrated?
- Does failure leave a recoverable file?
- Is there a backup or checkpoint path for risky operations?

## Blockers

- String editing JSON state.
- Writing state without error handling.
- Advancing phase state without verification.
- Removing user-local state during update.
- Migration that cannot handle missing or legacy fields.

## Evidence

Good evidence includes fixture tests, migration tests, invalid JSON handling, and a lifecycle smoke command.
