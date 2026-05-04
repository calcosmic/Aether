---
schema_version: "1.0"
id: command-wrapper-generation-playbook
kind: playbook
category: playbooks
title: Command Wrapper Generation Playbook
description: "Playbook for creating and verifying cross-platform command wrappers."
output_types: [wrapper-plan, platform-parity]
agent_roles: [builder, watcher, chronicler, queen, architect]
task_types: [command, wrapper, generation, parity]
task_keywords: [command, wrapper, yaml, claude, opencode, generated, header, mirror, source-check, argument]
workflow_triggers: [build, continue]
priority: high
version: "1.0"
source: "aether-native"
render:
  mode: full
  max_chars: 4000
---
# Command Wrapper Generation Playbook

## Use When

Use this when adding or changing Aether command wrappers.

For beginners: write the command once, generate the platform wrappers, then verify the live surfaces.

## Steps

1. Add or edit `.aether/commands/<name>.yaml`.
2. Generate/update Claude wrapper in `.claude/commands/ant/<name>.md`.
3. Generate/update OpenCode wrapper in `.opencode/commands/ant/<name>.md`.
4. Publish from the Aether repo so the hub and platform command homes refresh
   from those canonical wrappers.
5. Keep wrappers routed to the Go CLI.
6. Run source-check or targeted parity tests.

## Required Wrapper Behavior

- Use the runtime CLI as source truth.
- Do not implement separate logic in wrapper prose.
- Keep argument handling explicit.
- Keep generated header intact.

## Verification

- wrapper files contain generated header
- Claude and OpenCode wrappers match each other
- command appears in help/docs where expected
- runtime command exists
