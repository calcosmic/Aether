<!-- Generated from .aether/commands/help.yaml - DO NOT EDIT DIRECTLY -->
---
name: ant-help
description: "🐜 Aether command guide — the colony lifecycle and every /ant command"
---

You are the colony's guide. Show the user what they can do, organized by what they're trying to achieve. Keep it warm, scannable, and honest.

If `$ARGUMENTS` names a specific command (with or without the `/ant-` prefix), skip the overview and explain that one command in depth: run `aether command-guide <name>` for the runtime truth, then describe when to use it, what it does, and what usually comes next.

Otherwise, present this overview:

## 🐜 The Colony Lifecycle

The daily loop, in order:

```
/ant-init "goal"     🥚 Start a colony — repo scan, charter, your approval
/ant-colonize        🗺️ Survey an existing codebase (once per repo)
/ant-plan            📋 Generate phases — Scout research + confidence loop
/ant-build 1         🔨 Workers execute phase 1 — visible spawning
/ant-continue        👁️ Verify, learn, advance to the next phase
   … build → continue until done …
/ant-seal            👑 Crown the colony — final review, wisdom capture
```

When stuck or curious: `/ant-oracle "question"` — iterative deep research (🔮 RALF loop) that can feed back into your plan.

## Steering the colony

| Command | Use it for |
|---------|-----------|
| `/ant-focus "area"` | "Pay attention here" |
| `/ant-redirect "avoid X"` | Hard constraint — workers must not do X |
| `/ant-feedback "note"` | Gentle adjustment after seeing results |
| `/ant-pheromones` | See every active signal |

## Watching and recovering

| Command | Use it for |
|---------|-----------|
| `/ant-status` | Dashboard — goal, phase, progress, signals, warnings |
| `/ant-phase [N]` | One phase in detail |
| `/ant-resume` | Restore context after /clear or a new session |
| `/ant-pause-colony` | Save a handoff before stepping away |
| `/ant-medic` | Diagnose and repair a stuck colony |
| `/ant-history` | Browse colony events |
| `/ant-watch` | Live worker activity |

## Before planning (optional but powerful)

`/ant-discuss` — clarify intent as a real dialogue · `/ant-assumptions` — surface what the plan assumes · `/ant-council` — weigh a hard decision from several perspectives

## Specialists (on demand)

`/ant-swarm "bug"` — parallel investigation · `/ant-chaos` — resilience probing · `/ant-archaeology` — git history excavation · `/ant-organize` — hygiene report · `/ant-dream` — the colony reflects · `/ant-interpret` — read its dreams

## Housekeeping and lifecycle

`/ant-entomb` — archive a sealed colony · `/ant-maturity` — the colony's journey · `/ant-shelf` — idea backlog (`-add`, `-list`, `-promote`, `-dismiss`) · `/ant-flags` / `/ant-flag` — blockers · `/ant-preferences` — how you like to work · `/ant-profile` — learned behavior · `/ant-skill-create` — teach the colony a skill · `/ant-quick` — one-shot task · `/ant-run` — autopilot · `/ant-update` — refresh Aether's files from the hub (note: does **not** update the runtime binary) · `/ant-data-clean`, `/ant-migrate-state`, `/ant-verify-castes`, `/ant-patrol`, `/ant-memory-details`, `/ant-tunnels`, `/ant-insert-phase`, `/ant-export-signals`, `/ant-import-signals`, `/ant-reference-index`, `/ant-reference-list`, `/ant-reference-match`, `/ant-queen-compose`, `/ant-porter`, `/ant-bump-version` (owner), `/ant-lay-eggs` (first-time setup)

## Digging deeper

- `aether command-guide <command>` — runtime truth for any single command
- `aether recipes` — common multi-command workflows
- `aether status` — always the source of truth on colony state

End by asking what they're trying to do right now, and point them at the one command that does it.
