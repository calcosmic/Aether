# Requirements

This file is the explicit capability and coverage contract for the active milestone.

## Active (`v1.4`)

### R039 — Medic Ant: Colony Health Diagnosis
- Class: core-capability
- Status: active
- Description: A Medic ant command (`/ant:medic` or `aether medic`) that scans colony data files and reports health status. It checks COLONY_STATE.json, pheromones.json, session.json, constraints.json, and trace.jsonl for corruption, staleness, inconsistency, or missing fields.
- Why it matters: Colony data drift is the #1 cause of "mysterious" Aether failures. A Medic gives users (and us) visibility into what's broken before it breaks runtime behavior.
- Source: direct user milestone brief
- Primary owning slice: M022/S01
- Validation: unmapped
- Notes: Must be read-only by default; repair requires explicit `--fix` flag.

### R040 — Auto-Repair for Common Issues
- Class: operational-safety
- Status: active
- Description: When invoked with `--fix`, the Medic can repair common colony data issues: clear stale spawn state, remove orphaned worktree entries, rebuild missing indexes, fix corrupted JSON structures, and reconcile wrapper/runtime drift.
- Why it matters: Many colony issues are recoverable with simple data fixes. Automating these reduces support burden and user frustration.
- Source: direct user milestone brief
- Primary owning slice: M022/S01
- Validation: unmapped
- Notes: Every repair action must be logged to trace.jsonl with before/after state.

### R041 — Medic Skill: Healthy State Specification
- Class: architecture
- Status: active
- Description: A dedicated skill file (`.aether/skills/colony/medic.md`) documents the "healthy state" for every colony data file — schema, required fields, valid values, relationships, and common failure modes. This skill is loaded by the Medic worker and kept up to date as the system evolves.
- Why it matters: The Medic's diagnostic rules must be maintainable and versioned alongside the codebase. A skill file makes the health specification explicit and reviewable.
- Source: direct user milestone brief
- Primary owning slice: M022/S01
- Validation: unmapped
- Notes: Skill should be auto-loaded by colony-prime when the Medic worker spawns.

### R042 — Ceremony Integrity Verification
- Class: differentiator
- Status: active
- Description: The Medic can verify that wrapper files match runtime expectations: stage markers present, emoji consistency maintained, context-clear guidance emitted, no duplicate ceremony. Reports mismatches between `.claude/commands/ant/`, `.opencode/commands/ant/`, and the Go runtime renderer.
- Why it matters: Wrapper/runtime drift was a major issue in v1.3. Catching it automatically prevents regression.
- Source: direct user milestone brief
- Primary owning slice: M022/S02
- Validation: unmapped
- Notes: Should reference commandEmojiMap and casteEmojiMap as ground truth.

### R043 — Trace-Based Remote Debugging
- Class: differentiator
- Status: active
- Description: The Medic can analyze a user's trace.jsonl (shared via export) to diagnose issues without repo access. Reads state transitions, phase changes, errors, and interventions to reconstruct what went wrong and suggest fixes.
- Why it matters: The trace logging from v1.3 was built precisely for this. The Medic makes it actionable.
- Source: direct user milestone brief
- Primary owning slice: M022/S02
- Validation: unmapped
- Notes: Uses `trace-replay`, `trace-summary`, and `trace-inspect` commands under the hood.

### R044 — Medic Worker Integration
- Class: product-surface
- Status: active
- Description: The Medic is a first-class worker in the Aether caste system. It can be spawned manually (`/ant:medic`) or automatically when the colony detects health issues (e.g., stale session, corrupted state). It produces a structured health report and recommends next steps.
- Why it matters: Making the Medic a caste member (not just a CLI command) lets it participate in the colony's self-awareness. A colony that can heal itself feels alive.
- Source: direct user milestone brief
- Primary owning slice: M022/S01
- Validation: unmapped
- Notes: Medic caste should have its own emoji, color, and label in codex_visuals.go.

## Completed (`v1.3`)

### R027-R038
- Status: completed
- Summary: Visual UX restoration, core path hardening, recovery and continuity, trace logging, slash command audit. All 12 requirements satisfied.

## Deferred

### R016 — Pheromone Markets and Reputation Exchange
- Class: differentiator
- Status: deferred

### R017 — Federation and Inter-Colony Coordination
- Class: expansion
- Status: deferred

### R018 — Evolution Engine / Self-Modifying Agents
- Class: speculative-differentiator
- Status: deferred

## Out of Scope

### R009 — Codex wrapper ceremony
- Class: anti-feature
- Status: out-of-scope

### R019 — More agents before curation closure
- Class: anti-feature
- Status: out-of-scope

## Traceability

| ID | Class | Status | Phase | Proof |
|---|---|---|---|---|
| R039 | core-capability | active | Phase 25 | unmapped |
| R040 | operational-safety | active | Phase 26 | unmapped |
| R041 | architecture | active | Phase 27 | unmapped |
| R042 | differentiator | active | Phase 28 | unmapped |
| R043 | differentiator | active | Phase 29 | unmapped |
| R044 | product-surface | active | Phase 25 | unmapped |
| R027 | primary-user-loop | completed | Phase 18 | validated |
| R028 | primary-user-loop | completed | Phase 19 | validated |
| R029 | differentiator | completed | Phase 20 | validated |
| R030 | product-surface | completed | Phase 21 | validated |
| R031 | operational-safety | completed | Phase 22 | validated |
| R032 | operational-safety | completed | Phase 22 | validated |
| R033 | recovery | completed | Phase 22 | validated |
| R034 | operational-safety | completed | Phase 22 | validated |
| R035 | recovery | completed | Phase 23 | validated |
| R036 | recovery | completed | Phase 23 | validated |
| R037 | differentiator | completed | Phase 24 | validated |
| R038 | operational-safety | completed | Phase 17 | validated |

## Coverage Summary

- Active requirements: 6
- Completed milestone requirements: 12
- Mapped to phases: 18
- Validated: 12
- Unmapped active requirements: 0
