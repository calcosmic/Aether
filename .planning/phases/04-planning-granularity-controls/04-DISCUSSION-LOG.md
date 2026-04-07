# Phase 4: Planning Granularity Controls - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-04-07
**Phase:** 04-planning-granularity-controls
**Areas discussed:** Range names and sizes, Default behavior, Out-of-range handling, Granularity and depth interaction

---

## Range Names and Sizes

| Option | Description | Selected |
|--------|-------------|----------|
| Keep as defined | Sprint (1-3), Milestone (4-7), Quarter (8-12), Major (13-20) | ✓ |
| Cap at Quarter | Remove Major (13-20), max 12 phases | |
| Custom ranges | Different names or ranges | |

**User's choice:** Keep as defined
**Notes:** The 4 ranges cover quick fixes through full releases. Matches PLAN-01 requirements.

---

## Default Behavior

| Option | Description | Selected |
|--------|-------------|----------|
| Default to milestone | If no granularity set, use milestone (4-7). Matches current ~6 phases default. | |
| Always ask | Every /ant:plan prompts for granularity. No silent default. | ✓ |
| Persist + default milestone | Like depth: set once, persists. Default milestone if never set. | |

**User's choice:** Always ask
**Notes:** No default — user always picks. Once picked, it persists for the colony. More control, slightly more friction on first use.

---

## Out-of-Range Handling

| Option | Description | Selected |
|--------|-------------|----------|
| Warn + user chooses | Show warning with actual vs chosen count. User decides: accept, adjust range, or replan. | ✓ |
| Auto-trim to fit | Silently trim phases to fit the range. | |
| Reject and replan | Hard reject, force route-setter to retry. | |

**User's choice:** Warn + user chooses
**Notes:** Matches PLAN-03 intent. Respects user judgment — sometimes a 5-phase sprint is intentional.

---

## Granularity and Depth Interaction

| Option | Description | Selected |
|--------|-------------|----------|
| Fully independent | Granularity = phase count, depth = thoroughness. No cross-influence. | ✓ |
| Soft recommendations | Major plan suggests light depth, sprint suggests deep. Not enforced. | |

**User's choice:** Fully independent
**Notes:** Simple, predictable. 4x4 = 16 valid combinations.

---

## Claude's Discretion

- Exact enum implementation (iota vs string constants)
- Whether to add --granularity flag to /ant:init
- How the "always ask" prompt appears in /ant:plan
- Exact warning message format
- Whether state-mutate should validate granularity values
- Whether to add plan-granularity get/set command pair

## Deferred Ideas

None — discussion stayed within phase scope.
