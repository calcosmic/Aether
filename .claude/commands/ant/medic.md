<!-- Aether-managed: runtime spec at .aether/commands/medic.yaml. Synced by aether update. -->
---
name: ant-medic
description: "🩹 Diagnose colony health — scan all colony data for corruption, staleness, and configuration issues"
---

Use the Go `aether` CLI as the source of truth.

- Execute `AETHER_OUTPUT_MODE=visual aether medic $ARGUMENTS` directly.
- Do not read, upgrade, or rewrite raw colony state files from this command spec.
- If the runtime reports health issues, relay that exact output.
- If docs and runtime disagree, runtime wins.

**Flags:**
- `--fix` — enable repair mode (read-only by default; `--fix` required for any mutation)
- `--force` — allow destructive repairs (requires `--fix`)
- `--json` — output structured JSON report
- `--deep` — include wrapper parity, hub publish integrity, and ceremony checks
- `--trace <path>` — analyze a trace export JSON file instead of active colony data

**Exit codes:** 0 = healthy, 1 = warnings found, 2 = critical issues found.

**Stale session files:** when the diagnosis points at leftover per-command
session state (an old research run, a dead watch, stale swarm files), inspect
before clearing and clear only the named command's scope:

```bash
aether session-verify-fresh --command oracle
aether session-clear --command oracle --dry-run
aether session-clear --command oracle
```

First check whether the named command's state is stale, then preview what a
clear would remove, then clear it for real once the preview looks right.

Protected scopes (`init`, `seal`, `entomb`) refuse to clear — their state is
precious and the runtime enforces that; do not work around it.

**YAML source:** `.aether/commands/medic.yaml`
