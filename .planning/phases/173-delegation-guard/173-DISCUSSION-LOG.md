# Phase 173: Delegation Guard - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-08-12
**Phase:** 173-Delegation Guard
**Areas discussed:** The limits themselves (completed with user) — What happens at the limit, Watching it while it runs, Abandoned helpers (selected by user, then delegated to Claude)

---

## Area selection

| Option | Description | Selected |
|--------|-------------|----------|
| The limits themselves | How deep helpers may go, and how many one run may use in total | ✓ |
| What happens at the limit | Stop everything, refuse just that helper, or warn and carry on | ✓ |
| Watching it while it runs | Live view, on-demand command, or an alert | ✓ |
| Abandoned helpers | How long before a quiet helper is declared dead and its allowance returned | ✓ |

**User's choice:** All four.

---

## Loose ends (todo cross-reference)

| Option | Description | Selected |
|--------|-------------|----------|
| Neither (Recommended) | Both matched on generic word overlap only; neither is about delegation | ✓ |
| The waiting-time one | The hardcoded ts-host preflight timeout | |
| Both | Pull both in — advised against | |

**User's choice:** Neither.
**Notes:** Recorded in CONTEXT.md `<deferred>` § Reviewed Todos so future phases know they were considered and why they were left out.

---

## The limits themselves

### Q1 — Depth cap

| Option | Description | Selected |
|--------|-------------|----------|
| Two levels (Recommended) | Coordinator's workers may each call one round of helpers; those helpers call nobody | ✓ |
| Three levels | One more layer of specialisation; worst case grows very fast | |
| One level — nobody spawns | Absolutely predictable, but removes a capability the repo already instructs workers to use | |

**User's choice:** Two levels.

### Q2 — Whole-run helper budget

| Option | Description | Selected |
|--------|-------------|----------|
| 20 (Recommended) | The figure already in the plan; generous for normal work, stops a runaway early | ✓ |
| Around 40 | Double the headroom; a runaway costs roughly twice as much before it's caught | |
| Make it adjustable | Sensible default, changeable per run or project | |

**User's choice:** 20.
**Notes:** "Make it adjustable" was declined. The rationale offered with that option — *a limit you can raise under pressure is a limit that gets raised* — is preserved in CONTEXT.md `<specifics>` because it generalises beyond this decision. Consequence for planning: **do not add a config key for the tree budget in this phase.**

### Q3 — What the 20 counts

| Option | Description | Selected |
|--------|-------------|----------|
| Every helper (Recommended) | The coordinator's own workers count too; 8 workers consumes 8 of the 20 | ✓ |
| Only the extra ones | Coordinator's workers are free; keeps wave and run limits cleanly separate | |
| You decide | Claude's recommendation was the first option | |

**User's choice:** Every helper.
**Notes:** Rationale accepted — *a limit you have to do arithmetic on to understand is a limit that gets misread.*

### Q4 — Depth counting convention (SPAWN-06)

Not put to the user as a choice. Claude ruled and flagged it inline for objection; none was raised.

**Ruling:** coordinator = depth 0, its workers = depth 1, their helpers = depth 2, guard refuses at depth 3.
**Notes:** Surfaced because it changes what "two levels" permits, and because it makes `.aether/workers.md`'s `--depth 0` for children provably wrong — correcting that file is now in scope. Judged internal bookkeeping rather than a user-facing behaviour choice, per the project's decision boundaries.

---

## What happens at the limit

**Presented but not answered.** The user paused mid-question:

> *"Maybe let's just pause here for now because I don't know if we're just — are we actually doing stuff?"*

Options that had been put forward, for the record:

| Option | Description | Selected |
|--------|-------------|----------|
| Refuse the new helper only (Recommended) | Turn away the offending spawn; everything running finishes | Claude's call |
| Stop the whole run | Unambiguous, cheapest in a runaway, but discards work already done | |
| Let it through, warn loudly | Never obstructs, but makes the limit advisory | |
| Refusal shown in the run's own output (Recommended) | Named helper and reason, in the normal progress output | Claude's call |
| Also written to the failure record | System can learn the pattern, but mixes a working limit in with genuine faults | Partially — ceiling only |

**Outcome:** The user asked whether this step was actually building anything. Answered honestly: no — discuss produces a decisions document, only the execute step changes code. The user then chose *"finish it off and go straight to planning"*, delegating the three remaining areas to Claude with the understanding that they can push back once the behaviour is visible.

---

## Watching it while it runs

**Not discussed with the user.** Delegated. Decisions recorded as D-12 … D-14 in CONTEXT.md: on-demand tree view that works mid-run, a threshold warning line at ~75% of budget instead of a live dashboard, and a hard requirement that the rendered tree reads as English rather than raw identifiers.

---

## Abandoned helpers

**Not discussed with the user.** Delegated. Decisions recorded as D-15 … D-18 in CONTEXT.md: both an automatic reaper and a manual list-and-clear command, a deliberately conservative inactivity threshold that never reaps an active helper, that threshold configurable (the one place a config key is warranted), and budget released on reap.

---

## Claude's Discretion

- **SPAWN-06 depth convention** (D-05 … D-07) — ruled, flagged to the user, not objected.
- **What happens at the limit** (D-08 … D-11) — delegated after the user paused.
- **Watching it while it runs** (D-12 … D-14) — delegated.
- **Abandoned helpers** (D-15 … D-18) — delegated.

D-01 … D-07 should be treated as locked. D-08 … D-18 are strong defaults; any implementation-driven deviation must be surfaced to the user, not silently taken.

---

## Deferred Ideas

- Making the whole-run budget configurable — offered and explicitly declined.
- A live-refreshing delegation dashboard — a threshold warning line ships instead.
- Reaching the unreachable ts-host recursion policy engine — out of scope, later phase.
- Phase 172's CR-06 residue — tracked as Phase 172.1, not this phase.
- Both reviewed-but-not-folded todos — see CONTEXT.md `<deferred>`.
