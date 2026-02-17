---
phase: 05-pheromone-system
plan: 03
subsystem: pheromone-signals
tags: [pheromones, midden, eternal-memory, auto-emission, signal-expiration, bash, jq]
dependency_graph:
  requires: ["05-01", "05-02"]
  provides: [pheromone-expire, eternal-init, auto-emit-on-phase-advance, midden-archival]
  affects: [continue.md, pheromones.json, midden.json, eternal/memory.json]
tech_stack:
  added: []
  patterns: [pause-aware-ttl, midden-archival, auto-emit-pheromones, idempotent-init]
key_files:
  created:
    - .aether/data/midden/midden.json
  modified:
    - .aether/aether-utils.sh
    - .claude/commands/ant/continue.md
decisions:
  - "pheromone-expire does NOT delete signals from pheromones.json — sets active=false only (archive not destroy)"
  - "Phase_end expiry only triggered from continue.md, never build.md — signals must survive through builds"
  - "Auto FEEDBACK strength 0.6 vs auto REDIRECT strength 0.7 — failures produce stronger signals than successes"
  - "Pause-aware TTL: adds pause_duration to expires_at before comparing against now (epoch seconds, macOS-safe)"
  - "eternal-init is fully idempotent — safe to call on every /ant:continue invocation"
  - "All Step 2.1 pheromone operations are silent and non-blocking (2>/dev/null || true)"
metrics:
  duration: 3 minutes
  completed: "2026-02-17"
  tasks_completed: 2
  files_modified: 2
---

# Phase 05 Plan 03: Pheromone Signal Lifecycle Summary

**`pheromone-expire` and `eternal-init` added to aether-utils.sh; continue.md now auto-emits FEEDBACK and REDIRECT pheromones on phase advance, expires phase_end signals to the midden archive, and initializes eternal cross-session memory — completing the self-learning loop.**

## What Was Built

**pheromone-expire subcommand** — Archives expired pheromone signals to the midden. Two modes: `--phase-end-only` expires only signals where `expires_at == "phase_end"` (used by continue.md); without the flag, it also expires timestamp-based signals past their TTL. Pause-aware TTL: reads `paused_at`/`resumed_at` from COLONY_STATE.json and adds the pause duration to each signal's expiry before comparison, so signals don't tick down during paused colonies. Signals are marked `active: false` in pheromones.json (never deleted) and appended to midden.json with an `archived_at` timestamp.

**eternal-init subcommand** — Creates `~/.aether/eternal/` directory and `memory.json` with schema `{version, created_at, colonies, high_value_signals, cross_session_patterns}`. Fully idempotent — returns `{already_existed: true}` on subsequent calls without modifying the file.

**Midden directory** — `.aether/data/midden/midden.json` created as the persistent archive for expired signals. Grows over time, never capped (signals are never truly deleted per design decision).

**continue.md Step 2.1 (Auto-Emit Phase Pheromones, SILENT)** — Inserted between Step 2 (state update) and Step 2.2 (wisdom promotion):
- **2.1a:** Auto-emits FEEDBACK (strength 0.6, source "worker:continue", TTL "phase_end") summarizing phase learnings
- **2.1b:** Reads `errors.flagged_patterns` and auto-emits REDIRECT (strength 0.7, TTL 30d) for each pattern with count >= 2
- **2.1c:** Calls `pheromone-expire --phase-end-only` to archive phase_end signals to midden
- **2.1d:** Calls `eternal-init` to ensure eternal memory structure exists

All pheromone operations in Step 2.1 are wrapped with `2>/dev/null || true` — phase advancement never fails due to pheromone errors.

## Task Commits

Each task was committed atomically:

1. **Task 1: pheromone-expire and eternal-init subcommands** - `8b874a2` (feat)
2. **Task 2: Wire auto-emission and expiration into continue.md** - `21c9672` (feat)

## Files Created/Modified

- `.aether/aether-utils.sh` - Added `pheromone-expire` case (~115 lines) and `eternal-init` case (~18 lines); updated help command registry to include both
- `.claude/commands/ant/continue.md` - Inserted Step 2.1 (68 lines) between Step 2 and Step 2.2
- `.aether/data/midden/midden.json` - Created on first pheromone-expire call (contains 5 archived signals from verification runs)

## Decisions Made

1. **Active=false, not deletion** — Expired signals stay in pheromones.json with `active: false`. They're also copied to midden.json. Midden is the historical record; pheromones.json retains the signal body for potential future inspection.

2. **Phase_end expiry only in continue.md** — Pitfall 3 from plan research: signals emitted before a build must survive through the build. Build.md never calls pheromone-expire. Only continue.md, at phase advance time, expires phase_end signals.

3. **Asymmetric signal strength** — Auto FEEDBACK at 0.6, auto REDIRECT at 0.7. Anti-patterns and failures produce stronger signals than successes. This aligns with the signal strength hierarchy: user-emitted (0.7-1.0) > auto REDIRECT (0.7) > auto FEEDBACK (0.6).

4. **Pause-aware TTL uses macOS-safe date** — `date -j -f "%Y-%m-%dT%H:%M:%SZ"` with fallback to `date -d` for Linux. Epoch seconds for all arithmetic.

5. **eternal-init is idempotent** — Every /ant:continue call will safely call eternal-init. On first call it creates the directory and file. Subsequent calls detect the file and return `already_existed: true` without touching it.

## Deviations from Plan

None - plan executed exactly as written.

## Phase 05 Completion

With this plan complete, the full pheromone system learning loop is operational:
- **05-01:** Signals written (pheromone-write), counted (pheromone-count), read with decay (pheromone-read)
- **05-02:** Signals consumed — injected into every builder and watcher prompt (pheromone-prime, build.md)
- **05-03:** Signals lifecycle complete — auto-emitted on phase advance, expired to midden, eternal memory initialized

## Self-Check: PASSED

Files verified:
- FOUND: `.aether/aether-utils.sh`
- FOUND: `.claude/commands/ant/continue.md`
- FOUND: `.aether/data/midden/midden.json`
- FOUND: `~/.aether/eternal/memory.json`

Commits verified:
- FOUND: `8b874a2` (feat(05-03): add pheromone-expire and eternal-init subcommands to aether-utils.sh)
- FOUND: `21c9672` (feat(05-03): wire auto-emission and signal expiration into continue.md)
