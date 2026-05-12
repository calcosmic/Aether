# Phase 111: Follow-up Migration Map - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-05-12
**Phase:** 111-follow-up-migration-map
**Areas discussed:** Migration granularity, Prioritization approach, Scope boundaries

---

## Migration Granularity

| Option | Description | Selected |
|--------|-------------|----------|
| Milestone-ready | Each item gets a milestone with phases, requirements, and success criteria — ready for /gsd-plan-phase | ✓ |
| Roadmap sketch | High-level phases only, no requirements or success criteria | |
| Implementation-ready | Detailed enough for a builder to start coding, with task breakdowns and file lists | |

**User's choice:** Milestone-ready
**Notes:** Recommended option selected. Balances upfront effort with downstream speed.

---

## Prioritization Approach

| Option | Description | Selected |
|--------|-------------|----------|
| Oracle first | Build TS host's ability to run the full Oracle RALF loop first — proves complex flow handling | ✓ |
| Swarm first | Get swarm display working through TS host — simpler warmup | |
| Parallel tracks | Tackle all three in parallel across different milestones | |

**User's choice:** Oracle first
**Notes:** Sequential ordering: Oracle → Swarm → Parity. Each builds on patterns proven by the previous.

---

## Scope Boundaries

| Option | Description | Selected |
|--------|-------------|----------|
| Migration only | Stick to migrating existing Go behavior to TS host. No new features. | ✓ |
| Migration + improvement | Allow some improvement alongside migration — better confidence scoring, richer formats | |

**User's choice:** Migration only
**Notes:** Respect the Go/TS boundary contract from Phase 106. No scope creep.

---

## Claude's Discretion

- Exact milestone version numbers (v1.17, v1.18, etc.)
- Phase count per milestone
- Requirement ID naming convention
- Document format (combined vs separate)
- How to structure the map output

## Deferred Ideas

None — discussion stayed within phase scope.
