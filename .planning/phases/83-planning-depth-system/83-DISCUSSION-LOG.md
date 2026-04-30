# Phase 83: Planning Depth System - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-04-30
**Phase:** 83-planning-depth-system
**Areas discussed:** Naming, Light mode, Deep mode, Integration

---

## Naming

| Option | Description | Selected |
|--------|-------------|----------|
| Use light/standard/deep | Match ROADMAP spec. Existing fast/balanced/deep/exhaustive stays for PlanGranularity. | ✓ |
| Align both to same names | Rename both planning depth and PlanGranularity to light/standard/deep. | |
| Use fast/balanced/deep instead | Match existing codebase, update ROADMAP. | |

**User's choice:** Use light/standard/deep (Recommended)
**Notes:** User confirmed the two concepts are different — PlanGranularity = phase count, planning depth = task detail.

---

## Light mode

| Option | Description | Selected |
|--------|-------------|----------|
| Objectives only | 1-2 tasks per plan: goal statement and completion check. | |
| High-level steps | 2-3 tasks per plan: broad strokes. | |
| You decide | Let planner determine based on phase context. | ✓ |

**User's choice:** You decide

---

## Deep mode

| Option | Description | Selected |
|--------|-------------|----------|
| Subtasks with edge cases | 5-8 tasks per plan: implementation + edge cases + tests. | |
| Exhaustive breakdown | 8+ tasks per plan: every possible path. | |
| You decide | Let planner determine based on phase context. | ✓ |

**User's choice:** You decide

---

## Integration

| Option | Description | Selected |
|--------|-------------|----------|
| Runtime only | Go binary handles the flag, no wrapper changes. | |
| Runtime + wrapper | Go binary + wrapper markdown updated for help text. | ✓ |

**User's choice:** Runtime + wrapper (Recommended)
**Notes:** User initially selected "unsure" — asked a follow-up clarifying the two options in plain English, then selected runtime + wrapper.

---

## Claude's Discretion

- Light mode task count — delegated to planner
- Deep mode task count — delegated to planner

## Deferred Ideas

None.
