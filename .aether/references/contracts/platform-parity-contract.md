---
schema_version: "1.0"
id: platform-parity-contract
kind: contract
category: contracts
title: Platform Parity Contract
description: "Contract for keeping Claude, OpenCode, and Codex surfaces aligned without pretending they are identical."
output_types: [platform-parity, architecture-review, distribution-review, parity-review]
agent_roles: [architect, builder, watcher, queen, chronicler]
task_types: [platform, parity, update, wrapper, agent]
task_keywords: [claude, opencode, codex, parity, wrapper, agent, platform, drift, mirror, generated]
workflow_triggers: [plan, build, continue]
priority: critical
version: "1.0"
source: "aether-native"
render:
  mode: full
  max_chars: 4200
---
# Platform Parity Contract

## Purpose

Aether supports Claude Code, OpenCode, and Codex, but the platforms expose different surfaces. Parity means matching intent, lifecycle behavior, and safety guarantees. It does not mean byte-identical files everywhere.

For beginners: the same Aether job should work on each platform, even if the wrapper file format is different.

## Required Parity

- Same command intent and required arguments.
- Same source-of-truth CLI command underneath.
- Same safety rules for protected state and destructive actions.
- Same agent role purpose and escalation behavior.
- Same install/update distribution path.
- Same documented limitation when a platform cannot support a feature.

## Allowed Differences

- Claude uses markdown command wrappers.
- OpenCode uses markdown command wrappers and OpenCode agent frontmatter.
- Codex uses direct CLI flow and TOML agent definitions.
- Codex UX comes from Go visual output, not slash-command wrapper prose.

## Review Questions

1. Does the change affect a command, an agent, or runtime behavior?
2. Which platform surfaces need to change?
3. Is there a packaging mirror that must stay in sync?
4. Does `aether source-check` cover this surface?
5. Does the docs wording explain the platform difference honestly?

## Failure Signals

- A wrapper mentions behavior the CLI does not implement.
- A platform mirror is updated but the source file is not.
- Codex docs describe slash-command behavior.
- A generated file is edited manually without updating its source.
