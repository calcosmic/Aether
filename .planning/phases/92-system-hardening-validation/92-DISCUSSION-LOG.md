# Phase 92: System Hardening & Validation - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-05-02
**Phase:** 92-System Hardening & Validation
**Areas discussed:** Worker heartbeats, Context refresh (SAFE-05/06), E2E smoke test scope, Update integrity (VAL-02)

---

## Worker Heartbeats

### Heartbeat Mechanism

| Option | Description | Selected |
|--------|-------------|----------|
| File-based heartbeat | Worker writes timestamp to .aether/data/heartbeat-{worker-id}.json. Works across all platforms. | ✓ |
| Output-based heartbeat | Worker emits structured comment in output stream. Tight coupling to output format. | |
| Skip heartbeats | Existing cleanup + process groups may be sufficient. | |

**User's choice:** File-based heartbeat
**Notes:** Works everywhere since workers can already write files.

### Heartbeat Writer

| Option | Description | Selected |
|--------|-------------|----------|
| Prompt instruction | Worker prompt includes heartbeat write instruction. Honest detection. | ✓ |
| Runtime-managed heartbeat | Background goroutine touches file. Can't distinguish alive vs stuck. | |
| Wrapper-driven heartbeat | Wrapper markdown writes between steps. Only works for Claude/OpenCode. | |

**User's choice:** Prompt instruction (recommended by Claude)
**Notes:** User asked "what do you recommend?" — Claude recommended prompt instruction for honesty and cross-platform consistency.

### Heartbeat Checker

| Option | Description | Selected |
|--------|-------------|----------|
| On-demand check | Check at colony-prime assembly or build-wave status points. Simpler. | |
| Background monitor goroutine | Periodic scan of heartbeat files with warnings and auto-cleanup. More responsive. | ✓ |

**User's choice:** Background monitor goroutine

---

## Context Refresh (SAFE-05/06)

### Context Gap Assessment

| Option | Description | Selected |
|--------|-------------|----------|
| Gap is timing only | 12+ sections already cover requirements. Fix is ensuring fresh assembly at spawn. | |
| Sections are missing | Some AAC-005 sections absent from current assembly. | |
| Needs audit first | Proper comparison of current sections against AAC-005 before deciding. | ✓ |

**User's choice:** Needs audit first
**Notes:** Planner will audit buildColonyPrimeOutput() against AAC-005 requirements.

### Context Freshness

| Option | Description | Selected |
|--------|-------------|----------|
| Fresh per-spawn | Colony-prime called right before each worker spawn. Not cached. | ✓ |
| Cached per-wave | Same context for all workers in a wave. Refresh between waves. | |
| Depends on audit | Let audit determine if per-spawn matters. | |

**User's choice:** Fresh per-spawn (recommended)

---

## E2E Smoke Test Scope (VAL-01)

### Test Form

| Option | Description | Selected |
|--------|-------------|----------|
| Go integration test | Single test calling commands in sequence. Matches e2e_recovery_test.go pattern. | ✓ |
| CLI smoke test script | Runs full flow via aether CLI binary. Harder setup/cleanup. | |
| Both integration test + runbook | Go test for CI, manual runbook for user verification. Most work. | |

**User's choice:** Go integration test (recommended)

### Test Coverage

| Option | Description | Selected |
|--------|-------------|----------|
| Full v1.13 flow | Init through seal, exercising all v1.13 components together. | ✓ |
| Phase 92 scope only | Worker lifecycle + context refresh + update integrity only. | |
| Split into 2 focused tests | One for worker lifecycle, one for gate recovery path. | |

**User's choice:** Full v1.13 flow (recommended)

---

## Update Integrity (VAL-02)

### Test Approach

| Option | Description | Selected |
|--------|-------------|----------|
| Update round-trip test | Create files, run update, verify content intact. Catches corruption. | ✓ |
| Post-update snapshot validation | Check files against expected content after update. Snapshot test. | |
| Extend existing install tests | Add integrity checks to existing install_cmd tests. | |

**User's choice:** Update round-trip test (recommended)

### Coverage Scope

| Option | Description | Selected |
|--------|-------------|----------|
| Both agents + commands | Agent definitions and command files across all platforms. | ✓ |
| Agents only | Most complex mirror chain, most likely to break. | |
| Full companion file set | Agents, commands, skills, docs, templates. Most thorough. | |

**User's choice:** Both agents + commands (recommended)

---

## Claude's Discretion

- Heartbeat writer choice (recommended prompt instruction, user agreed)
- Context freshness choice (recommended fresh per-spawn, user agreed)
- E2E test form (recommended Go integration test, user agreed)
- Update round-trip coverage (recommended both agents + commands, user agreed)

## Deferred Ideas

None — discussion stayed within phase scope.
