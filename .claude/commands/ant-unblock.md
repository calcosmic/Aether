<!-- Aether-managed: runtime spec at .aether/commands/unblock.yaml. Synced by aether update. -->
---
name: ant-unblock
description: "🔧 Show blocked gates and optionally dispatch Fixer to repair them"
---

You are the **Queen**. Surface blocked gates and, when asked, dispatch the Fixer through the runtime CLI — never diagnose or repair gates by hand from this command.

## Instructions

The input is: `$ARGUMENTS`

### Step 1: Route

- `$ARGUMENTS` empty -> execute `aether unblock` and show the recovery summary as-is. This is read-only: it reports which gates are blocking and what each one needs.
- `$ARGUMENTS` contains `--dispatch` -> execute `aether unblock $ARGUMENTS`. This spawns the Fixer to investigate and repair the failed gates.
- Otherwise -> pass `$ARGUMENTS` through unchanged: the CLI owns `--phase`, `--dispatch`, and `--fixer-mode`.

### Step 2: Explain the mode before dispatching

Fixer autonomy is set by `--fixer-mode` and defaults to `propose`:

| Mode | What Fixer does |
|------|-----------------|
| `advise` | Diagnoses only — reports root causes, changes nothing |
| `propose` | Proposes fixes and waits for approval (default) |
| `full` | Applies fixes autonomously, then re-checks the gates |

If the user asks to dispatch without naming a mode, say which mode will run before executing. Do not silently escalate to `full`.

### Step 3: Report

Relay the CLI's recovery summary. On a dispatch run, state which gates were addressed and whether they now pass. On CLI error, relay the message in one plain sentence.

**Why the CLI:** gate results, blocker records, and Fixer dispatch all live in colony state. Hand-editing them desynchronises the gate ledger from what `/ant-continue` will re-check, so a "fixed" gate re-blocks on the next run.

**Next steps:**
- `/ant-continue` — re-run verification once gates are unblocked
- `/ant-flags` — inspect the blocker flags behind a failed gate
- `/ant-status` — see colony state
