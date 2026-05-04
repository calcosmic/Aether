---
schema_version: "1.0"
id: update-safety-contract
kind: contract
category: contracts
title: Update Safety Contract
description: "Rules for safe install/update behavior across the Aether hub and target repos."
output_types: [update-plan, distribution-review, safety-review]
agent_roles: [builder, watcher, architect, queen, porter]
task_types: [update, install, publish, distribution, safety]
task_keywords: [update, install, publish, overwrite, preserve, prune, hub, target repo, stale, channel, drift, protected, force]
workflow_triggers: [build, continue, seal]
priority: critical
version: "1.0"
source: "aether-native"
render:
  mode: full
  max_chars: 4400
---
# Update Safety Contract

## Purpose

Install and update paths can damage user work if they confuse shipped files with local state. This contract defines what may be copied, overwritten, preserved, or pruned.

For beginners: this is the "do not trash the user's project" contract.

## Protected Repo State

Normal update must preserve:

- `.aether/data/`
- `.aether/dreams/`
- `.aether/oracle/`
- `.aether/QUEEN.md`
- `.aether/CONTEXT.md`
- `.aether/HANDOFF.md`
- `.env*`
- user-created platform files outside Aether-managed paths

## Shipped Companion Files

Update may sync managed Aether companion files from the hub:

- command wrappers
- agent definitions
- docs
- templates
- skills for Codex
- rules and schemas

References are global-only in v1. They sync from `~/.aether/system/references/` to `~/.aether/references/`, not into target repo `.aether/references/`.

## Force Rules

`--force` may overwrite locally modified shipped companion files. It must not override protected repo state.

## Review Questions

1. Could this copy into a user-owned file?
2. Could cleanup remove a custom file?
3. Is the destination source, mirror, hub staging, active hub, or target repo?
4. Does dry-run tell the truth?
5. Is there a regression test for the protected path?
