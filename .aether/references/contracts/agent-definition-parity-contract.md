---
schema_version: "1.0"
id: agent-definition-parity-contract
kind: contract
category: contracts
title: Agent Definition Parity Contract
description: "Contract for keeping Aether agent role intent aligned across Claude, OpenCode, Codex, and hub-published platform assets."
output_types: [agent-review, platform-parity, source-audit]
agent_roles: [architect, builder, watcher, queen, chronicler]
task_types: [agent, platform, parity, source, update]
task_keywords: [agent, claude, opencode, codex, hub, parity, role, drift, TOML, definition, alignment]
workflow_triggers: [plan, build, continue]
priority: high
version: "1.0"
source: "aether-native"
render:
  mode: full
  max_chars: 3800
---
# Agent Definition Parity Contract

## Purpose

Agent definitions tell platforms what each worker is for. Role drift causes incorrect delegation, weak verification, and confused orchestration.

For beginners: Builder, Watcher, Oracle, and the rest should mean the same job everywhere.

## Source Surfaces

- Claude: `.claude/agents/ant/aether-*.md`
- OpenCode: `.opencode/agents/aether-*.md`
- Codex: `.codex/agents/aether-*.toml`
- Published copies: global hub and platform homes created by `aether publish`
  or `aether install --package-dir`.

## Required Alignment

- role purpose
- when to use
- safety boundaries
- output expectations
- tool permissions
- handoff behavior
- evidence standard

## Allowed Differences

- File format and metadata.
- Platform-specific tool names.
- Codex can be less polished where direct CLI flow differs.

## Failure Signals

- A published platform copy differs from canonical source without explanation.
- A role gains platform-only powers.
- Codex TOML omits a safety rule present elsewhere.
- OpenCode frontmatter is invalid or too vague.
