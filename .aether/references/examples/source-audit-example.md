---
schema_version: "1.0"
id: source-audit-example
kind: example
category: examples
title: Source Audit Example
description: "Example of resolving source, hub, and target-repo ownership."
output_types: [source-audit-example, source-audit]
agent_roles: [scout, architect, watcher, queen]
task_types: [source, audit, example]
task_keywords: [example, source, hub, target repo, references, truth, drift, global, staging]
workflow_triggers: [plan, build]
priority: normal
version: "1.0"
source: "aether-native"
render:
  mode: full
  max_chars: 3200
---
# Source Audit Example

## Question

Where should Aether references live?

## Facts

- Source files are edited in the Aether repo under `.aether/references/{category}/{id}.md`.
- Hub staging is `~/.aether/system/references/`.
- Active global runtime library is `~/.aether/references/`.
- Target project repos should not receive `.aether/references/` in v1.

## Decision

Keep references global-only. Runtime may read repo source references only when running inside the Aether source checkout for development and tests.

## Risks

- If target repos get `.aether/references/`, users will not know which copy is real.
- If hub files are treated as source, stale installed content can overwrite the repo.

## Verification

- `reference-list` in Aether source shows repo references.
- update sync test proves target repos do not get `.aether/references/`.
