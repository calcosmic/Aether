# Phase 202: Swarm, Oracle, and Live Colony - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-09-11
**Phase:** 202-swarm-oracle-and-live-colony
**Areas discussed:** Live view experience, Swarm steering, Oracle conversation shape, Where finished work lives

---

## Live View Experience

| Option | Description | Selected |
|--------|-------------|----------|
| Dashboard + event ticker | Updates in place with a recent-events strip at the bottom | ✓ |
| Updating dashboard only | Current state only, no scrollback | |
| Scrolling feed only | Plain event log | |

| Option | Description | Selected |
|--------|-------------|----------|
| Focus on the active work | Current wave in detail, colony compressed to a header line | ✓ |
| Whole colony overview | Everything at equal weight | |

| Option | Description | Selected |
|--------|-------------|----------|
| Last episode summary | Replay-backed card of the most recent activity when idle | ✓ |
| Simple idle card | Today's honest fallback only | |

**Notes:** Mid-discussion the owner added a standing steer: "we want things to have a bit... the visual aspect of the February, April time" — the Classic-era look (caste emoji + ants, waves, ceremony) is a phase requirement for all three surfaces, anchored on v5.4.0 via the SYNTH-04 study. Recorded as D-04.

---

## Swarm Steering

| Option | Description | Selected |
|--------|-------------|----------|
| Auto-repair with safety net | Checkpoint, apply top-ranked repair, verify, roll back on failure | ✓ |
| Show me the theories first | Owner approves before any repair | |

| Option | Description | Selected |
|--------|-------------|----------|
| Live in the colony view | Four lenses visible as workers; comparison card at the end | ✓ |
| Summary at the end only | Quiet run, one final card | |

| Option | Description | Selected |
|--------|-------------|----------|
| Architectural question to you | Three strikes escalates with a structural-change case | ✓ |
| Just stop with the evidence | Report failures without a proposal | |

---

## Oracle Conversation Shape

| Option | Description | Selected |
|--------|-------------|----------|
| Clarify once, then run | Up-front clarification, autonomous rounds, one final answer | ✓ |
| Check in each round | Owner decides continuation per round | |

| Option | Description | Selected |
|--------|-------------|----------|
| Same picker as planning | Fast/Balanced/Deep/Exhaustive from Phase 200 | ✓ |
| Oracle-specific dial | Separate confidence-percentage target | |

| Option | Description | Selected |
|--------|-------------|----------|
| Recommendation first | Actionable conclusion up top, evidence beneath | ✓ |
| Evidence first | Report-style build-up to the conclusion | |

---

## Where Finished Work Lives

| Option | Description | Selected |
|--------|-------------|----------|
| Status + history | Latest episode in status; history lists all with outcomes and cost | ✓ |
| Files, pointed at once | Loose files named at finish | |

| Option | Description | Selected |
|--------|-------------|----------|
| Keep, label plainly | Partial work listed alongside finished, marked "not verified" | ✓ |
| Keep, but tucked away | Separate area, out of the main view | |

---

## Claude's Discretion

- Typed event schema/versioning/replay mechanics and unifying the two existing event systems
- What the four Swarm lenses concretely are, and hypothesis comparison/ranking mechanics
- Internal Go types, episode storage, card spacing/color, refresh cadence
- Mapping the Phase 200 presets onto Oracle's existing confidence-target machinery

## Deferred Ideas

- Recruitment/delegation/pheromone causality — Phase 203
- Learning governance for Swarm/Oracle lessons — Phase 204
