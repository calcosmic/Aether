---
schema_version: "1.0"
id: publish-update-playbook
kind: playbook
category: playbooks
title: Publish Update Playbook
description: "Playbook for publishing Aether source changes and validating downstream update behavior."
output_types: [publish-plan, update-plan, distribution-review]
agent_roles: [builder, watcher, queen, architect, porter]
task_types: [publish, update, install, distribution]
task_keywords: [publish, update, install, hub, downstream, release, source-check, stale, channel, drift, dev, binary]
workflow_triggers: [build, continue, seal]
priority: critical
version: "1.0"
source: "aether-native"
render:
  mode: full
  max_chars: 4600
---
# Publish Update Playbook

## Use When

Use this when a change needs to move from the Aether repo to the shared hub and then into other repos.

For beginners: publish pushes from the Aether repo; update pulls into target repos.

## Source Workflow

```bash
aether source-check
aether publish
```

For dev isolation:

```bash
aether publish --channel dev --binary-dest "$HOME/.local/bin"
aether-dev update
```

## What Publish Does

- Builds the binary unless skipped.
- Syncs companion files to hub staging.
- Refreshes platform home assets on stable.
- Verifies hub and binary versions agree.

## What Update Does

- Pulls companion files from hub.
- Prunes stale managed files.
- Preserves local state.
- Refreshes active global references from hub staging.
- Does not publish source changes.

## Verification

- `go build ./cmd/aether`
- `aether source-check`
- targeted update test
- downstream dry-run or temp repo smoke test

## Common Mistakes

- Running update in target repos before publishing source.
- Editing hub files directly.
- Forgetting embedded assets.
- Confusing dev and stable hubs.
