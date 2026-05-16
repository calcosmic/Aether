# Known Issues and Workarounds

Updated: 2026-05-15

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

### Provider availability preflight is not a post-launch API guarantee

- **Area:** Real worker dispatch through Codex, Claude, and OpenCode-compatible platforms
- **Impact:** Before worker launch, Aether checks whether a platform CLI exists and appears authenticated. That preflight can report categories such as `binary_missing`, `auth_probe_failed`, `auth_inactive`, `invalid_auth_output`, `credentials_missing`, or `probe_skipped`. It does not prove that the later worker request will be accepted by the selected model, account, proxy, or upstream API.
- **Mitigation:** Show only the sanitized provider, cause category, and next action returned by the runtime. Do not expose raw provider stdout/stderr, tokens, or auth probe output in docs, wrapper narration, debug summaries, or generated context.

### Post-launch provider/API/auth failures can look like worker parse failures

- **Area:** Real worker dispatch through hosted platforms after the worker process starts
- **Impact:** A real lifecycle smoke on 2026-05-15 launched OpenCode-compatible workers and rendered the expected ceremony, but the provider returned an auth/API failure payload instead of Aether worker claims JSON. Aether may report `parse worker output: no JSON found in output` or point to a worker-debug artifact because the terminal output was not valid worker claims.
- **Mitigation:** `aether continue` correctly blocks advancement and `aether watch --once` / `aether swarm --watch` render recovery guidance. The next hardening pass should classify post-launch provider/API/auth payloads before worker-result parsing so users see a sanitized setup/provider problem instead of a generic JSON-claims parse failure.

### Slash-command docs still require periodic parity sweeps

- **Area:** `.claude/commands/ant/*.md` and `.opencode/commands/ant/*.md`
- **Impact:** Those mirrors describe higher-level platform UX and can drift when the Go CLI surface changes.
- **Mitigation:** Treat the Go runtime in `cmd/` and the Codex guides (`AGENTS.md`, `.codex/CODEX.md`) as authoritative first; use command-doc sweeps to bring the markdown mirrors back in line.

### Visual output depends on terminal mode

- **Area:** Codex visual surfaces
- **Impact:** caste colors and live previews only render in visual/TTY mode.
- **Mitigation:** use an interactive terminal, or set `AETHER_FORCE_VISUAL=1`. JSON mode intentionally disables the visuals.
