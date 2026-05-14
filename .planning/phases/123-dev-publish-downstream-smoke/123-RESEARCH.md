# Phase 123: Dev Publish + Downstream Smoke - Research

**Gathered:** 2026-05-14
**Status:** Ready for planning

## Current State

- **Repo version:** 1.0.38 (`.aether/version.json` and `npm/package.json` agree)
- **Installed stable binary:** 1.0.38
- **Dev channel hub:** `~/.aether-dev/` at version 1.0.37 (stale — needs publish)
- **Dev channel registry:** empty (`{"repos":[]}`)

## Downstream Test Candidates

Available repos with Aether installed:
- `~/repos/litellm-proxy/.aether`
- `~/repos/Formica/.aether`
- `~/repos/CosmicDashboard/.aether`
- `~/repos/AbletonCompadre/.aether`
- `~/repos/cosmic-dev-system/.aether`
- `~/repos/CornettoDatabase/.aether`
- `~/repos/openclaude/.aether`
- `~/repos/agency-agents/.aether`
- `~/repos/CalCosmic Discord/.aether`
- `~/repos/Colony Creation Station/.aether`

**Best candidate:** `~/repos/Formica` — small Go project, likely clean working tree, no active development.

## Publish Process

Per `publish-update-runbook.md`:

```bash
# In Aether repo
aether publish --channel dev --binary-dest "$HOME/.local/bin"

# In target repo
aether-dev update --force
```

Dev channel behavior:
- Uses `~/.aether-dev/system/` instead of `~/.aether/system/`
- Binary installs as `aether-dev`
- Skips user-level platform asset sync (doesn't overwrite stable commands)
- Verifies binary and hub versions agree

## Smoke Test Steps

1. `aether-dev update --force` — refresh scaffolding
2. `aether-dev init "Test colony"` — initialize colony
3. `aether-dev plan` — generate phases
4. `aether-dev build 1` — build first phase
5. `aether-dev continue` — verify and advance
6. `aether-dev oracle` — run Oracle loop (or verify command exists)

## Known Risks

- `aether-dev` binary may not exist after publish if `--binary-dest` isn't on PATH
- Downstream repo may have uncommitted changes blocking update
- Dev channel intentionally skips platform asset sync — wrappers won't update globally
