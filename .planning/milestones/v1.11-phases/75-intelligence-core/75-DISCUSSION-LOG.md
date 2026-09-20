# Phase 75: Intelligence Core - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-04-29
**Phase:** 75-intelligence-core
**Areas discussed:** Trust scoring integration, Circuit breaker design, Build ceremony learning flow, Circuit breaker scope

---

## Trust Scoring Integration

| Option | Description | Selected |
|--------|-------------|----------|
| Extend memory-capture | Add --source-type and --evidence-type to memory-capture so playbooks can pass them. Simplest change. | ✓ |
| Switch to learning-observe | Switch playbooks to use learning-observe which already has the flags. | |
| New unified command | Create a new command combining simplicity of memory-capture with trust flags of learning-observe. | |

**User's choice:** Extend memory-capture
**Notes:** Keep existing defaults (observation/anecdotal) so unflagged callers still work. Playbooks pass explicit flags.

### Default Score Behavior

| Option | Description | Selected |
|--------|-------------|----------|
| Keep low defaults | Keep observation/anecdotal as defaults so existing calls still work. | ✓ |
| Raise the defaults | Change defaults to something higher like success_pattern/single_phase. | |

**User's choice:** Keep low defaults

### Source/Evidence Type Authority

| Option | Description | Selected |
|--------|-------------|----------|
| Playbook-driven types | Playbooks explicitly pass --source-type and --evidence-type. | ✓ |
| Auto-detect from context | Auto-detect from colony state (e.g., if in build, assume build learning). | |

**User's choice:** Playbook-driven types

---

## Circuit Breaker Design

### Trigger Condition

| Option | Description | Selected |
|--------|-------------|----------|
| Consecutive failure count | Worker fails N times consecutively. Simple, predictable. | ✓ |
| Failure rate over window | Worker fails N times out of last M attempts. | |
| Both conditions | Most protective but more complex. | |

**User's choice:** Consecutive failure count

### Trip Behavior

| Option | Description | Selected |
|--------|-------------|----------|
| Redistribute to peers | Tasks go to other workers of same caste. No tasks lost. | ✓ |
| Skip and log | Tasks skipped and logged as blocked. | |
| Halt the build | Entire build halts and prompts user. | |

**User's choice:** Redistribute to peers

### Reset Strategy

| Option | Description | Selected |
|--------|-------------|----------|
| Per-wave reset | Breaker resets at start of each new wave. | ✓ |
| Cooldown timer | Reset after a timer (e.g., 60 seconds). | |
| Manual reset only | Only user can reset. | |

**User's choice:** Per-wave reset

---

## Circuit Breaker Scope

### Granularity

| Option | Description | Selected |
|--------|-------------|----------|
| Per-worker instance | Each worker instance has its own breaker. Most granular. | ✓ |
| Per-caste | All workers of same caste share a breaker. Too aggressive. | |
| Per-task-type | Tasks of same type share a breaker. | |

**User's choice:** Per-worker instance

### Parallel Mode Coverage

| Option | Description | Selected |
|--------|-------------|----------|
| Both modes | Circuit breaker in both in-repo and worktree modes. | ✓ |
| Worktree mode only | Only in worktree mode. | |

**User's choice:** Both modes

---

## Build Ceremony Learning Flow

### Build Ceremony Learning Capture

| Option | Description | Selected |
|--------|-------------|----------|
| Continue only | Build doesn't capture learnings — that stays in continue. Cleaner separation. | ✓ |
| Build + continue | Build ceremony also captures learnings mid-build. | |

**User's choice:** Continue only

### Continue Ceremony Source/Evidence Types

| Option | Description | Selected |
|--------|-------------|----------|
| success_pattern/multi_phase | Continue learnings get higher scores reflecting verified patterns. | ✓ |
| observation/single_phase | More conservative — treats continue learnings same as any observation. | |
| Planner decides | More nuanced but adds decision-making to ceremony. | |

**User's choice:** success_pattern/multi_phase

### Continue Ceremony Scope

| Option | Description | Selected |
|--------|-------------|----------|
| Capture with scoring | Continue captures learnings with trust scoring via extended memory-capture. | ✓ |
| Capture + decay refresh | Also runs trust-score-decay on existing observations. | |

**User's choice:** Capture with scoring

---

## Claude's Discretion

- Exact consecutive failure threshold (default 3)
- Circuit breaker state storage approach
- Visual rendering of breaker events
- Whether to log tripped workers at wave end
- Test coverage approach

## Deferred Ideas

None — discussion stayed within phase scope.
