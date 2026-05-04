---
schema_version: "1.0"
id: platform-parity-rubric
kind: rubric
category: rubrics
title: Platform Parity Rubric
description: "Rubric for evaluating whether Claude, OpenCode, and Codex behavior remain aligned."
output_types: [platform-parity, parity-review]
agent_roles: [watcher, auditor, architect, queen, builder, chronicler]
task_types: [platform, parity, wrapper, agent, review]
task_keywords: [platform, parity, claude, opencode, codex, wrapper, agent, drift, mirror, source-check]
workflow_triggers: [build, continue]
priority: high
version: "1.0"
source: "aether-native"
render:
  mode: sections
  max_chars: 3400
  sections: [Use When, Parity Checks, Acceptable Drift, Blockers]
---
# Platform Parity Rubric

## Use When

Use this when commands, agents, docs, or runtime UX change across supported host platforms.

For beginners: each platform can speak its own format, but it should not tell a different story.

## Parity Checks

- Same lifecycle command maps to same CLI behavior.
- Same safety warnings and protected paths.
- Same agent role purpose.
- Same generated wrapper source.
- Same distribution expectations.
- Same restart or update guidance where relevant.

## Acceptable Drift

- Codex uses direct CLI, not slash commands.
- Platform metadata differs by format.
- Visual polish may differ where a platform lacks a wrapper surface.

## Blockers

- One platform advances without a required gate.
- One wrapper writes a different state file.
- One agent omits a critical safety boundary.
- Mirrors are stale after source changes.
