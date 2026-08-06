<!-- Aether-managed: runtime spec at .aether/commands/recover.yaml. Synced by aether update. -->
---
name: ant-recover
description: "🛟 Rescue a stuck colony — diagnose why it cannot make progress and optionally fix it"
---

Use the Go `aether` CLI as the source of truth.

- Execute `AETHER_OUTPUT_MODE=visual aether recover $ARGUMENTS` directly.
- Do not read, upgrade, or rewrite raw colony state files from this command spec.
- Relay the runtime's recovery guidance verbatim. It names the exact flags for
  the issue it found, and those flags differ per issue — do not generalise them.
- If docs and runtime disagree, runtime wins.

## When to use this

`/ant-recover` is for a colony that will not move: a build that appears to have
run but left no record, a phase that will not advance, stale spawn state, a
missing manifest. It diagnoses first and only changes anything when asked.

Related commands, which are not the same thing:

| Command | Use it for |
|---------|-----------|
| `/ant-recover` | The colony is stuck and you want it unstuck |
| `/ant-medic` | Colony data looks corrupt, stale, or misconfigured |
| `/ant-unblock` | A quality or security gate failed and you want the options |

**Flags:**
- `--apply` — apply fixes for detected issues (diagnosis only by default)
- `--force` — allow destructive repairs (requires `--apply`)
- `--json` — output structured JSON report

Typical escalation, and the runtime will name which one it needs:

```
/ant-recover                  # what is wrong?
/ant-recover --apply          # fix what can be fixed safely
/ant-recover --apply --force  # fix it even where the repair is destructive
```

**YAML source:** `.aether/commands/recover.yaml`
