---
schema_version: "1.0"
id: command-wrapper-contract
kind: contract
category: contracts
title: Command Wrapper Contract
description: "Contract for Claude/OpenCode command wrappers and their YAML sources."
output_types: [wrapper-review, source-audit, platform-parity, wrapper-plan]
agent_roles: [builder, watcher, chronicler, queen, architect]
task_types: [command, wrapper, generated, parity, update]
task_keywords: [command, wrapper, yaml, generated, claude, opencode, source-check, header, argument, route]
workflow_triggers: [build, continue, publish]
priority: high
version: "1.0"
source: "aether-native"
render:
  mode: full
  max_chars: 3600
---
# Command Wrapper Contract

## Purpose

Command wrappers are platform entrypoints, but the Go CLI is the runtime source of truth. Wrappers should route to the CLI and explain only platform-specific handling.

For beginners: the wrapper is the button, not the engine.

## Source Layout

- Shared YAML source: `.aether/commands/*.yaml`
- Claude generated wrapper: `.claude/commands/ant/*.md`
- OpenCode generated wrapper: `.opencode/commands/ant/*.md`
- Published copies: global hub and platform command homes created from the
  canonical wrappers.

## Rules

- Generated wrappers need the generated header.
- Do not hand-edit generated wrappers without updating the YAML source.
- Use direct `aether ...` runtime commands.
- Do not invent behavior that the Go CLI does not implement.
- Keep argument handling explicit.
- Use visual mode for lifecycle commands unless JSON is required.

## Review Questions

1. Does the YAML source exist?
2. Do Claude and OpenCode wrappers match the generated source?
3. Is Codex unaffected because it uses direct CLI?
4. Does source-check cover the changed wrapper?
5. Does the wrapper avoid stale command names?
