# Known Issues and Workarounds

Updated: 2026-04-25

This file tracks live Aether limitations that are still relevant to the current Go-based runtime.
Historical bash/npm migration bugs were removed once the affected paths stopped existing.

## Current State

- Core Codex colony lifecycle is now aligned enough for day-to-day use:
  - `colonize`, `plan`, `build`, `continue`, `resume`, `status`, `run`, `watch`, and `oracle` all have active regression coverage.
  - `go test ./...` and `go test ./... -race` are expected to stay clean for release readiness.

## Open Limitations

### Interrupted build workers can require explicit recovery

- **Area:** Codex and wrapper build lifecycle
- **Impact:** If a worker run is interrupted before the build packet is finalized, the colony can remain on an active phase with no usable worker manifest.
- **Mitigation:** Run `aether continue` first. It now writes blocked recovery guidance when the build packet is missing. Use `aether build <phase> --force` to redispatch the active phase. If the phase is intentionally abandoned, use the audited escape hatch: `aether skip-phase <phase> --force --reason "<why>"`.

### Worker artifact contracts are only as good as the worker following them

- **Area:** Codex `colonize` / `plan`
- **Impact:** Real workers are now allowed to author survey and planning artifacts directly, and `plan` can consume a worker-written `phase-plan.json`. If a worker ignores that contract, Aether falls back to local synthesis.
- **Mitigation:** The command output now reports explicit provenance (`dispatch_mode`, `artifact_source`, `plan_source`) so fallback behavior is visible instead of silent.

### Slash-command docs still require periodic parity sweeps

- **Area:** `.claude/commands/ant/*.md` and `.opencode/commands/ant/*.md`
- **Impact:** Those mirrors describe higher-level platform UX and can drift when the Go CLI surface changes.
- **Mitigation:** Treat the Go runtime in `cmd/` and the Codex guides (`AGENTS.md`, `.codex/CODEX.md`) as authoritative first; use command-doc sweeps to bring the markdown mirrors back in line.

### Visual output depends on terminal mode

- **Area:** Codex visual surfaces
- **Impact:** caste colors and live previews only render in visual/TTY mode.
- **Mitigation:** use an interactive terminal, or set `AETHER_FORCE_VISUAL=1`. JSON mode intentionally disables the visuals.
