# Phase 201: Queen-Led Work Cycle - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-09-10
**Phase:** 201-queen-led-work-cycle
**Areas discussed:** One verification boundary, Result truth & outcome cards, Recovery & bounded repair, Turnaround targets & telemetry

---

## Todo Routing

| Option | Description | Selected |
|--------|-------------|----------|
| Fold turnaround only | Fold worker-turnaround into 201; ts-host stays Phase 203; spec-builder done | ✓ |
| Pull ts-host into 201 too | Override Phase 200's routing | |
| Fold all three as chosen | Include already-delivered spec-builder | |

**User's choice:** Initially selected all three todos, then on the routing-conflict follow-up chose "Fold turnaround only" (the recommended option matching PROJECT.md routing).

---

## One Verification Boundary

| Option | Description | Selected |
|--------|-------------|----------|
| Continue owns judgment | All reviewer judgment at continue; build deterministic-only | |
| Build owns judgment | Reviewers at build-end; continue pure gate | |
| Queen picks per phase | Queen decides boundary per phase, recorded reason | ✓ |

**Follow-up — Queen's default:** Default to continue (✓) vs default to build-end vs always judged.
**Build-end view:** Honest unverified card (✓) vs auto-chain into verify vs chain on autopilot only.
**Lane architecture:** One Go path, thin lanes (✓) vs keep lanes + parity test vs you decide.

---

## Result Truth & Outcome Cards

| Option | Description | Selected |
|--------|-------------|----------|
| Same ceremony, honest verdict | Full Queen closeout for every outcome | ✓ |
| Short card for failures | Compact card for non-success | |
| You decide | Claude picks per-outcome rendering | |

**Cost line:** Every closeout + status (✓) vs status only vs autopilot report only.
**Next Up after non-success:** Queen recommends + why (✓) vs one repair-first action vs full option menu.
**Changed-file truth:** Credited + orphaned, both named (✓) vs credited only with orphan drill-down vs you decide.

---

## Recovery & Bounded Repair

| Option | Description | Selected |
|--------|-------------|----------|
| One round | One checkpointed repair wave, re-verify once, then pause | ✓ |
| Two rounds | Second targeted attempt | |
| Queen decides, capped at 3 | Queen picks bound from failure shape | |

**Checkpoints:** Announce both (✓) vs announce rollback only vs quiet.
**Handback:** Diagnosis + tried + ask (✓) vs evidence dump + options vs you decide.
**Learning loop:** Yes — same-phase signal consumption by repair waves (✓) vs next phase only vs you decide.

---

## Turnaround Targets & Telemetry

| Option | Description | Selected |
|--------|-------------|----------|
| Under 10 minutes | ~3× faster target | |
| Under 15 minutes | ~2× faster target | |
| Measure first, then set | Build telemetry, record baseline, set target from data | ✓ |

**Telemetry display:** One line + drill-down (✓) vs full table each closeout vs on demand only.
**Speed levers (multi-select):** Targeted test lanes (✓), slimmer briefs & handoffs (✓), model routing by caste (✓); parallel waves by default NOT selected.
**Auto-tuning:** Report only this phase (✓) vs bounded auto-tuning now vs you decide.

## Claude's Discretion

- Internal Go types, receipt schema details, card spacing/color, checkpoint storage mechanics, test-lane selection heuristics.
- Handback design beyond the required contents.
- Per-phase boundary judgment stays the Queen's (within the locked five-signal reviewer rules).

## Deferred Ideas

- Live Swarm/Watch/Oracle + typed events → Phase 202
- Causal pheromones, ts-host preflight → Phase 203
- Telemetry-driven auto-tuning → Phase 204
- Codex-native `$ant-*` skills → later milestone
