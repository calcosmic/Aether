# Phase 58: Smart Review Depth - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-04-27
**Phase:** 58-smart-review-depth
**Areas discussed:** Light mode agent scope, Final phase detection, Depth display format, Keyword detection rules, Chaos Ant spawning, --heavy flag, Chaos deterministic seed, Worker depth awareness

---

## Light Mode Agent Scope

| Option | Description | Selected |
|--------|-------------|----------|
| Watcher only | Light mode = just Watcher verification. Fastest path. | ✓ |
| Watcher + Gatekeeper | Keep security gate even in light mode. Slower but catches secrets. | |
| Watcher + Gatekeeper + Auditor | Keep security and quality gates. Only skip Probe, Weaver, Medic, Measurer, Chaos. | |

**User's choice:** Watcher only — matches the requirement goal of "saving time without sacrificing safety"
**Notes:** The 7 heavy agents (Auditor, Gatekeeper, Probe, Weaver, Medic, Measurer, Chaos) only run on final and security-sensitive phases.

---

## Final Phase Detection

| Option | Description | Selected |
|--------|-------------|----------|
| Last in phases array | Check if current phase is the last entry in COLONY_STATE.json phases array. | ✓ |
| Marker flag in colony state | Add `is_final: true` flag during /ant-plan. Explicit but could get stale. | |
| Highest phase number | Compare against highest phase number. Breaks if backlog items have high numbers. | |

**User's choice:** Last in phases array — simple, deterministic, auto-adjusts when phases are inserted.

---

## Depth Display Format

| Option | Description | Selected |
|--------|-------------|----------|
| Wrapper output only | Show depth message in build/continue wrapper output only. | ✓ |
| Both wrapper and runtime | Show in both surfaces. More visible but duplicates. | |
| Minimal — just depth label | Show "Review depth: light" without phase position context. | |

**User's choice:** Wrapper output only — runtime stays quiet, wrapper shows "Review depth: light (Phase 3 of 7 — final phase gets full review)".

---

## Keyword Detection Rules

| Option | Description | Selected |
|--------|-------------|----------|
| Case-insensitive substring | Match on phase name. Catches "Security Hardening" → "security". | ✓ |
| Exact word boundary match | More precise but might miss "Authentication" matching "auth". | |
| Configurable keyword list | Keywords from config file. Flexible but adds config surface. | |

**User's choice:** Case-insensitive substring matching on the phase name.

---

## Chaos Ant Spawning

| Option | Description | Selected |
|--------|-------------|----------|
| Chaos covered by heavy mode | Only runs on heavy phases (final + security keywords). | |
| Random Chaos sampling on light | 30% chance on light phases too. Catches resilience issues. | ✓ |
| Specific Chaos issue | User has a specific problem to address. | |

**User's choice:** Random Chaos sampling on light phases at 30%.

### Chaos Probability

| Option | Description | Selected |
|--------|-------------|----------|
| 30% chance | 1 in 3 light phases. Noticeable without being noisy. | ✓ |
| 20% chance | 1 in 5. Rare but catches issues over a milestone. | |
| 50% chance | Every other light phase. More thorough but slower. | |

**User's choice:** 30% chance.

### Chaos Determinism

| Option | Description | Selected |
|--------|-------------|----------|
| Deterministic by phase | Hash phase number. Same result across runs. | ✓ |
| Truly random | Different each run. More surprising but harder to reproduce. | |

**User's choice:** Deterministic by phase — reproducible builds.

---

## --Heavy Flag

| Option | Description | Selected |
|--------|-------------|----------|
| Yes, add --heavy | Both --light and --heavy. Symmetrical control. | ✓ |
| No, --light only | Only --light as specified. Rely on auto-detection for heavy. | |

**User's choice:** Add --heavy flag alongside --light for full user control.

---

## Worker Depth Awareness

| Option | Description | Selected |
|--------|-------------|----------|
| Yes, inject depth context | Workers get "Light review" or "Heavy review" context. | ✓ |
| No, just control spawning | Workers always do their best. Simpler. | |

**User's choice:** Inject depth context so workers adapt their thoroughness.

---

## Claude's Discretion

- Exact hash function for deterministic Chaos sampling
- Exact phrasing of worker depth context injection
- How resolveReviewDepth integrates with existing normalizedBuildDepth system
- Whether review_depth gets stored in COLONY_STATE.json or computed on-the-fly

## Deferred Ideas

- Configurable keyword list for auto-heavy detection
- Non-deterministic Chaos sampling
- Depth-based time budgets for agents
