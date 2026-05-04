---
schema_version: "1.0"
id: source-of-truth-contract
kind: contract
category: contracts
title: Source Of Truth Contract
description: "Contract for deciding which Aether file wins when source, mirror, generated, hub, and runtime files disagree."
output_types: [source-audit, conflict-review, architecture-review]
agent_roles: [queen, architect, chronicler, scout, watcher, builder]
task_types: [source, audit, documentation, update, conflict]
task_keywords: [source of truth, mirror, generated, hub, stale, conflict, docs, drift, authority, precedence]
workflow_triggers: [colonize, plan, build, update]
priority: critical
version: "1.0"
source: "aether-native"
render:
  mode: full
  max_chars: 4200
---
# Source Of Truth Contract

## Purpose

This contract prevents Aether workers from repairing the wrong file or copying stale hub output back into source.

For beginners: it answers "which copy is real?"

## Authority Order

1. Latest explicit user correction.
2. Go runtime source in `cmd/` and `pkg/`.
3. Source companion files in `.aether/`.
4. Platform source surfaces in `.claude/`, `.opencode/`, and `.codex/`.
5. Generated wrappers and packaging mirrors.
6. Installed hub files in `~/.aether/`.
7. Local runtime state under `.aether/data/`, `.aether/dreams/`, `.aether/oracle/`, and locks.

## Source Rules

- Do not edit hub files as if they are source.
- Do not recover source files from installed hub output unless the user explicitly asks.
- Generated wrappers must be regenerated or mirrored from their YAML source.
- Packaging mirrors must match their source surfaces.
- Target repo state must never be treated as Aether package source.

## Required Output

When a worker resolves a source conflict, it should report:

- Winning source.
- Losing stale source.
- Files changed.
- Files intentionally left alone.
- Verification command or inspection.
