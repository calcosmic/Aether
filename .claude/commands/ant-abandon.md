<!-- Aether-managed: runtime spec at .aether/commands/abandon.yaml. Synced by aether update. -->
---
name: ant-abandon
description: "🗑️ Discard a colony that is not worth finishing and start fresh"
---

Use the Go `aether` CLI as the source of truth.

- Execute `AETHER_OUTPUT_MODE=visual aether abandon $ARGUMENTS` directly.
- Do not write colony state files by hand from this command spec.
- If docs and runtime disagree, runtime wins.

## When to use this

Sometimes a goal turns out not to be worth finishing. `/ant-abandon` throws that
colony away and clears the repo for a new one.

| Situation | Command |
|-----------|---------|
| The work is finished | `/ant-seal`, then `/ant-entomb` to archive it |
| The work is not worth finishing | `/ant-abandon` |
| The colony is stuck but still wanted | `/ant-recover` |

Use `/ant-seal` for finished work. Sealing a colony you are discarding runs the
whole completion ceremony over it and promotes its lessons into the
cross-colony hive — abandoned work teaching every other project.

## Two steps, on purpose

Run without a flag first. The runtime prints exactly what would be lost — the
goal, how many phases were completed, how much the colony learned — and stops.

```
/ant-abandon
```

**Show that preview to the user and let them decide.** Never pass `--confirm` on
their behalf from a first request; "scrap this" is a decision they should make
with the cost in front of them. If they confirm:

```
/ant-abandon --confirm
```

The state is backed up to `.aether/data/backups/` first and can be restored by
copying the `.bak` file back over `COLONY_STATE.json`. Relay the backup path —
someone who changes their mind an hour later needs it.

**Stop conditions:** If there is no active colony, the runtime says so and does
nothing. That is not an error; do not retry with `--confirm`.

**YAML source:** `.aether/commands/abandon.yaml`
