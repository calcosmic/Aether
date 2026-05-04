---
schema_version: "1.0"
id: distribution-safety-rubric
kind: rubric
category: rubrics
title: Distribution Safety Rubric
description: "Rubric for reviewing publish, install, update, embedded assets, and hub sync changes."
output_types: [distribution-review, safety-review, publish-review]
agent_roles: [watcher, auditor, gatekeeper, architect, queen, porter]
task_types: [distribution, publish, install, update, safety]
task_keywords: [publish, install, update, hub, embed, sync, overwrite, prune, stale, channel, drift, target repo]
workflow_triggers: [build, continue, seal]
priority: critical
version: "1.0"
source: "aether-native"
render:
  mode: sections
  max_chars: 4000
  sections: [Use When, Critical Checks, Blockers, Pass Conditions]
---
# Distribution Safety Rubric

## Use When

Use this for changes to `aether publish`, `aether install`, `aether update`, embedded assets, sync pairs, hub staging, or target repo companion files.

For beginners: this checks whether a change ships safely.

## Critical Checks

- Source files flow outward to hub, never the reverse.
- Hub staging and active hub paths are distinct.
- Target repo protected state remains untouched.
- Cleanup removes only managed stale files.
- `--force` has bounded overwrite behavior.
- Dev and stable channels stay isolated.
- Embedded assets include newly shipped source folders.
- Dry-run accurately describes real update behavior.

## Blockers

Block advancement if:

- Target repos receive global-only assets.
- User state can be overwritten or pruned.
- A publish path can refresh one platform but not hub staging.
- A binary/runtime change is needed but only companion files update.
- Source-check cannot detect mirror drift for the changed surface.

## Pass Conditions

Passing evidence should include at least one targeted unit test and one CLI or build smoke check when the changed path affects actual distribution.
