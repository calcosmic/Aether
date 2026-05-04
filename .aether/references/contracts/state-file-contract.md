---
schema_version: "1.0"
id: state-file-contract
kind: contract
category: contracts
title: State File Contract
description: "Contract for reading, writing, and migrating Aether state files safely."
output_types: [state-review, migration-plan, safety-review, migration-review]
agent_roles: [builder, watcher, medic, architect, queen, fixer]
task_types: [state, migration, storage, repair, update]
task_keywords: [COLONY_STATE, state, migration, lock, atomic, data, repair, corruption, protected, fixture, JSON]
workflow_triggers: [build, continue, update, seal]
priority: critical
version: "1.0"
source: "aether-native"
render:
  mode: full
  max_chars: 4200
---
# State File Contract

## Purpose

Aether state files drive colony lifecycle, dispatch, verification, recovery, and memory. Corrupt state can block work or advance unsafe phases.

For beginners: state files are the colony's memory. Write them carefully.

## Rules

- Use structured parsers for JSON, not string replacement.
- Use existing storage helpers and locks where available.
- Prefer atomic writes for state mutations.
- Preserve unknown fields unless a migration intentionally removes them.
- Never update state based only on visual output.
- Record migration assumptions and failure handling.
- Do not use repo-local state as package source.

## Protected Files

Treat these as user/runtime state:

- `.aether/data/COLONY_STATE.json`
- `.aether/data/pheromones.json`
- `.aether/data/midden/`
- `.aether/data/handoffs/`
- `.aether/data/oracle/` and `.aether/oracle/`
- `.aether/QUEEN.md`
- `.aether/HANDOFF.md`

## Verification

State changes need at least one of:

- unit tests for migration logic
- CLI smoke test for the lifecycle command
- invalid-state fixture
- recovery-path test

## Failure Signals

- State is written without validation.
- An update path overwrites runtime data.
- A migration cannot be rerun safely.
- Tests only check that a file exists, not that its content is valid.
