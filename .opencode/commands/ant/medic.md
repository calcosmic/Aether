<!-- Generated from .aether/commands/medic.yaml - DO NOT EDIT DIRECTLY -->
---
name: ant:medic
description: "🏥 Diagnose colony health — scan all colony data for corruption, staleness, and configuration issues"
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
- `--deep` — include wrapper parity and ceremony checks

**Exit codes:** 0 = healthy, 1 = warnings found, 2 = critical issues found.

**YAML source:** `.aether/commands/medic.yaml`
