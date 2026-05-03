# Phase 94: Recovery Data Model - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-05-03
**Phase:** 94-Recovery Data Model
**Areas discussed:** Retry boundary, Recovery visibility, Error classification rules

---

## Retry Boundary

| Option | Description | Selected |
|--------|-------------|----------|
| Try once then tell me | Colony tries once. If it works, great. If not, it stops and tells you. | ✓ |
| Try once then skip | Colony tries once, logs failure but continues with remaining workers. | |
| Try twice then tell me | Colony tries twice, then stops and tells you. | |

**Question:** For "requires-attempt" failures, what should the colony do?
**User's choice:** Try once then tell me

| Option | Description | Selected |
|--------|-------------|----------|
| Retry 3 times, then tell me | Colony retries up to 3 times automatically, then stops and tells you. | ✓ |
| Retry once, then tell me | Colony retries once. If it fails again, stops immediately. | |
| Keep retrying until budget runs out | Colony keeps retrying until manual stop or phase budget. | |

**Question:** For fully "recoverable" failures, how many automatic retries before giving up?
**User's choice:** Retry 3 times, then tell me

---

## Recovery Visibility

| Option | Description | Selected |
|--------|-------------|----------|
| Summary in /ant-status | Compact summary after each phase. | |
| Real-time retry messages | See each retry attempt as it happens in build output. | ✓ |
| Log file only | Recovery activity only in the phase-scoped recovery log file. | |

**Question:** When the colony auto-recovers a failed worker, how much should you see?
**User's choice:** Real-time retry messages

| Option | Description | Selected |
|--------|-------------|----------|
| One-line retry message | Concise: 'Worker Builder-67 timed out — retrying (attempt 2/3)' | ✓ |
| Retry message + error detail | Retry message plus the original error output. | |

**Question:** For the real-time messages, how much detail per retry?
**User's choice:** One-line retry message

---

## Error Classification Rules

| Option | Description | Selected |
|--------|-------------|----------|
| Yes, that's right | Timeouts, context overflow, resource limits = transient (auto-retry) | ✓ |
| Need to discuss more | Discuss edge cases or add more transient types. | |

**Question:** Transient (auto-retry) errors — timeouts, context overflow, resource limits. Sound right?
**User's choice:** Yes, that's right

| Option | Description | Selected |
|--------|-------------|----------|
| Yes, that's right | Bad task spec, missing dependency, invalid path, structural error = systemic (immediate escalate) | ✓ |
| Need to discuss more | Discuss edge cases or add more systemic types. | |

**Question:** Systemic (immediate escalate) errors — bad task spec, missing dependency, invalid path, structural error. Sound right?
**User's choice:** Yes, that's right

| Option | Description | Selected |
|--------|-------------|----------|
| Yes, that's right | Partial completion, garbled output = requires-attempt (try once then tell you) | ✓ |
| Need to discuss more | Discuss edge cases or add more requires-attempt types. | |

**Question:** Requires-attempt (try once then tell you) errors — partial completion, garbled/unparseable output. Sound right?
**User's choice:** Yes, that's right

---

## Claude's Discretion

- Exact struct field names and Go types for FailureRecord and RecoveryLogEntry
- Error pattern matching implementation (string matching, exit codes, or both)
- Recovery log file naming convention
- How recovery log relates to existing midden system
- CLI commands for inspecting recovery logs
- Pause duration between retries
- Partial completion detection approach

## Deferred Ideas

None — discussion stayed within phase scope.
