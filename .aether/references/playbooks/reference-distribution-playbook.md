---
schema_version: "1.0"
id: reference-distribution-playbook
kind: playbook
category: playbooks
title: Reference Distribution Playbook
description: "Install/update/publish flow for global Aether references."
output_types: [distribution-review, install-plan, update-plan, publish-plan]
agent_roles: [builder, watcher, queen, architect, porter]
task_types: [install, update, publish, distribution, reference]
task_keywords: [publish, install, update, hub, global, system, references, sync, stale, channel, drift, lookup]
workflow_triggers: [build, continue, seal]
priority: critical
version: "1.0"
source: "aether-native"
render:
  mode: full
  max_chars: 4600
---
# Reference Distribution Playbook

## Use When

Use this when modifying how references move from the Aether source repo to the global hub.

For beginners: this is the shipping route. It says where the files start, where they are staged, and where agents read them.

## Source Of Truth

Edit references only in the Aether repo:

```text
.aether/references/{category}/{id}.md
```

Do not author reference files directly in target project repositories.

## Publish And Install Flow

1. Source repo contains `.aether/references/`.
2. `aether publish` is the primary publish command. It builds the binary, syncs companion files (including references) to hub staging, and verifies version agreement.
3. Hub staging lives at `~/.aether/system/references/`.
4. Active runtime references live at `~/.aether/references/`.
5. `aether update` refreshes active hub references from hub staging.
6. Target repos do not receive `.aether/references/`.
7. `aether install --package-dir "$PWD"` is the legacy path and still works for backward compatibility.

## Runtime Lookup

When running in the Aether source checkout, development commands may read `.aether/references/` directly so tests and local smoke checks use the edited source files.

Outside the Aether source checkout, reference matching reads the global hub library under `~/.aether/references/` and `~/.aether/system/references/`.

## Safety Checks

- Do not overwrite unrelated user-local state.
- Do not prune target repo files based on the reference manifest.
- Do not treat `~/.aether/` as source truth during repo development.
- Do not create per-project reference libraries in v1.

## Verification

Run:

```bash
go build ./cmd/aether
go test ./cmd -run Reference -count=1
go run ./cmd/aether reference-list
go run ./cmd/aether reference-match --role oracle --task "evaluate React vs Vue" --output-type tech-eval
```
