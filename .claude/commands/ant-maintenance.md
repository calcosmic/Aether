<!-- Aether-managed: runtime spec at .aether/commands/maintenance.yaml. Synced by aether update. -->
---
name: ant-maintenance
description: "Inspect or repair Aether internals with preview and rollback."
---

Use the Go `aether` CLI as the source of truth.

Run `AETHER_OUTPUT_MODE=visual aether maintenance $ARGUMENTS` exactly once and return its stdout unchanged. With no arguments, the runtime shows the expert landing. Pass supplied arguments through unchanged; do not select or execute a second operation from wrapper prose.

The runtime owns the operation inventory and target resolution. The runtime owns every state and evidence read. The runtime owns every mutation, validation, preview, checkpoint, staged write, verification, transaction receipt, commit, and automatic rollback. Relay its typed result and current Next Up answer without adding or inferring success.

## Inspection

- `recovery.inspect`
- `integrity.inspect`
- `source.parity.inspect`
- `registry.inspect`
- `chamber.inspect`
- `context.inspect`
- `archive.inspect`
- `skills.inspect`
- `skills.diff`

## Mutation

- `update.apply`
- `state.migration.apply`
- `data.cleanup.apply`
- `backup.prune.apply`
- `temp.cleanup.apply`
- `registry.update`
- `chamber.create`

Live skills remain nested below the maintenance landing:

- `/ant-maintenance skills inspect`
- `/ant-maintenance skills diff --before <receipt> --after <receipt>`

Do not expose root-level skill list, diff, or cache commands.

- Do not write colony state, session, evidence, registry, chamber, hub, or generated files from this wrapper.
- Do not parse visual output, ANSI text, or stdout as state or evidence. Return runtime output unchanged.
- A process exit alone does not prove a repair or verification result; relay only the runtime's typed outcome.
- Keep Codex runtime-native; do not add a Codex maintenance command skill.
