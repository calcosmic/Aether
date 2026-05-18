# Phase 136: Production Foundation - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-05-18
**Phase:** 136-production-foundation
**Areas discussed:** Failure experience, Real dispatch UX, Dry-run output, Skill visibility

---

## Failure Experience

### Worker failure behavior

| Option | Description | Selected |
|--------|-------------|----------|
| Halt and tell me | Build stops immediately, clear message naming failed worker and why | |
| Continue and summarize | Failed worker marked, remaining continue, summary at wave end | |
| Claude decides per failure type | Auth errors halt, timeouts continue, crashes halt | |

**User's choice:** Claude decides (user said "you decide")
**Notes:** Claude chose: auth/config errors halt (user action needed), timeouts continue (worker may be slow), crashes halt. Summary at wave end. This matches the existing `continue` gate pattern.

### Error message style

| Option | Description | Selected |
|--------|-------------|----------|
| Plain English only | Single sentence, no error codes or paths | ✓ |
| Plain English + one detail | Plain English plus one technical detail in parentheses | |

**User's choice:** Plain English only
**Notes:** Go runtime already has `aether platform-diagnostic` for checks; user-facing messages hide the technical detail.

---

## Real Dispatch UX

### Ceremony difference from simulation

| Option | Description | Selected |
|--------|-------------|----------|
| Same ceremony, real results | Identical output to simulation, files just actually change | ✓ |
| One-time notice then same ceremony | Banner at start saying "this is real", then same ceremony | |
| More verbose than simulation | Extra detail: worker PID, command, file paths | |

**User's choice:** Same ceremony, real results
**Notes:** Silent upgrade. No "THIS IS REAL" banners.

### Worker progress display

| Option | Description | Selected |
|--------|-------------|----------|
| Spinner per worker | Spinner with caste name, updates in place | |
| Start/finish lines | Print line on start, line on finish | |
| Silent until wave completes | No output until wave finishes | |

**User's choice:** Existing swarm/ceremony display (user noted "I thought we had it where you could see them working in the terminal window with the caste colors?")
**Notes:** User remembered the existing swarm display with caste colors. Decision: reuse existing `swarm-display.ts` and `ceremony-adapter.ts` for real dispatch — they already render live worker activity with caste emoji, ANSI colors, and spinners.

---

## Dry-run Output

### What --dry-run shows

| Option | Description | Selected |
|--------|-------------|----------|
| Full ceremony preview | Same ceremony output with "DRY RUN" badge | ✓ |
| Summary table only | Wave/Workers/Tasks table, no ceremony | |
| Raw manifest JSON | Go manifest JSON output | |

**User's choice:** Full ceremony preview with "DRY RUN" badge
**Notes:** User sees exactly what the real build would look like, just marked as a preview.

---

## Skill Visibility

### Indication of skill injection

| Option | Description | Selected |
|--------|-------------|----------|
| One-line summary at build start | "Injecting N skills into worker prompts." | ✓ |
| Silent — skills are internal | No indication at all | |
| Per-worker skill count | Each worker's ceremony line shows skill count | |

**User's choice:** One-line summary at build start
**Notes:** Clean ceremony output. Skills do their work silently after the one-line notice.

---

## Claude's Discretion

- Technical implementation of simulation guard removal
- How to structure dry-run ceremony rendering
- Claims parsing for real platform output
- Wiring real dispatch results into Go finalizer pipeline
- Test strategy for real dispatch
- "DRY RUN" badge rendering style

## Deferred Ideas

None — discussion stayed within phase scope.
