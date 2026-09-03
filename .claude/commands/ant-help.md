<!-- Aether-managed: runtime spec at .aether/commands/help.yaml. Synced by aether update. -->
---
name: ant-help
description: "🐜 Show the guided Aether journey from runtime truth"
---

You are the colony's guide. Use the Go `aether` CLI as the source of truth.

Run `AETHER_OUTPUT_MODE=visual aether help $ARGUMENTS` exactly once and return its stdout unchanged. The runtime owns the standing line, responsive layout, command descriptions, and state-derived Next Up answer. If an argument names one command, pass it through in `$ARGUMENTS`; do not invent a second command catalog.

The runtime output must preserve these visible contracts:

- Empty line 1: `No colony is active`
- Empty line 2: `Start a guided colony for one goal with /ant-init "goal".`
- Active standing: `Colony: {identity} | Goal: {accepted goal} | Episode: {episode} | Phase: {current}/{total} | Standing: {standing} | Ants: {acting castes or No ants are active} | Blockers: {count} | Next Up: {exact action}`

## Normal journey

- `/ant-init "goal"` — Start a guided colony for one goal.
- `/ant-plan` — Turn the accepted goal and territory evidence into an executable phase plan.
- `/ant-build` — Execute one accepted phase with guided checkpoints.
- `/ant-run` — Autopilot the remaining accepted phases within the displayed safety contract.
- `/ant-status` — Show the complete authoritative colony snapshot.
- `/ant-pause` — Stop at a safe boundary and save one resumable handoff.
- `/ant-resume` — Validate and restore the safest honest recovery point.
- `/ant-seal` — Close a verified colony, or explicitly record an owner-forced incomplete closure.
- `/ant-entomb` — Archive and clear the sealed colony.

## Steer and inspect

- `/ant-focus` — Guide colony attention toward one area.
- `/ant-feedback` — Add a gentle correction for future work.
- `/ant-redirect` — Record a hard constraint the colony must avoid.
- `/ant-watch` — Show live worker activity.
- `/ant-phase` — Inspect the current phase and its accepted work.
- `/ant-history` — Review recorded colony events.
- `/ant-swarm` — Route a problem or inspect the live swarm.
- `/ant-oracle status` — Inspect the current deep-research run.
- `/ant-dream` — Let the colony reflect on the codebase.
- `/ant-interpret` — Interpret saved Dreams without changing colony state.
- `/ant-memory-details` — Inspect retained learning and memory.
- `/ant-flags` — Inspect retained blockers, issues, and findings.

## Expert maintenance

- `/ant-maintenance` — Inspect or repair Aether internals with preview and rollback.
