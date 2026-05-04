---
schema_version: "1.0"
id: platform-surface-map
kind: field-guide
category: field-guides
title: Platform Surface Map
description: "Guide for identifying which Claude, OpenCode, Codex, and hub files are affected by a change."
output_types: [platform-map, parity-review]
agent_roles: [architect, scout, watcher, queen, builder, chronicler]
task_types: [platform, parity, map, source]
task_keywords: [platform, claude, opencode, codex, hub, mirror, wrapper, drift, surface, agent, generated]
workflow_triggers: [plan, build, update]
priority: high
version: "1.0"
source: "aether-native"
render:
  mode: sections
  max_chars: 4000
  sections: [Use When, Surface Types, Mapping Questions, Output]
---
# Platform Surface Map

## Use When

Use this when a change may touch platform-specific files.

For beginners: this tells you which copy belongs to which host tool.

## Surface Types

- Runtime: `cmd/`, `pkg/`
- Shared command source: `.aether/commands/*.yaml`
- Claude wrappers: `.claude/commands/ant/*.md`
- OpenCode wrappers: `.opencode/commands/ant/*.md`
- Claude agents: `.claude/agents/ant/*.md`
- OpenCode agents: `.opencode/agents/*.md`
- Codex agents: `.codex/agents/*.toml`
- Packaging mirrors: `.aether/agents-*`, `.aether/commands/*`
- Hub staging: `~/.aether/system/`

## Mapping Questions

- Is this source, generated, mirror, or installed output?
- Which platform consumes it directly?
- Which command publishes or updates it?
- Does source-check verify parity?
- Does Codex need docs or runtime UX instead of wrapper changes?

## Output

List affected surfaces and required verification for each.
