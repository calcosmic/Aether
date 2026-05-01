# Phase 86: Depth Selection UI and Persistence - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-05-01
**Phase:** 86-depth-selection-ui-and-persistence
**Areas discussed:** Plan output format, Override mechanism, Build depth display

---

## Plan Output Format

| Option | Description | Selected |
|--------|-------------|----------|
| Banner with values + reasons | A clean section showing both depths and why they were selected | ✓ |
| Compact single-line | Single line in plan header, no explanation | |
| Claude's discretion | You decide | |

**User's choice:** Banner with values + reasons
**Notes:** User wants the full information visible — both depths and the reason for each.

---

## Override Mechanism

| Option | Description | Selected |
|--------|-------------|----------|
| Add --verification-depth flag | Direct mirror of --planning-depth on /ant-plan | ✓ |
| Reuse --light/--heavy flags | Make build flags available on plan command | |
| Claude's discretion | You decide | |

**User's choice:** Add --verification-depth flag
**Notes:** Consistent UX — same flag pattern for both depths.

---

## Build Depth Display

| Option | Description | Selected |
|--------|-------------|----------|
| Show in stage markers + summary | Display depth in every stage marker and final summary | ✓ |
| Show in summary only | Only in final build summary | |
| Claude's discretion | You decide | |

**User's choice:** Show in stage markers + summary
**Notes:** User always wants to know what depth is running.

---

## Claude's Discretion

No areas deferred to Claude — all three areas had explicit user selections.
