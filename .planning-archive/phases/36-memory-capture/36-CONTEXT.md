# Phase 36: Memory Capture - Context

**Gathered:** 2026-02-21
**Status:** Ready for planning

<domain>
## Phase Boundary

Wire the existing memory systems so they actually capture and store learnings. This phase enhances `/ant:continue` and `/ant:build` to capture learnings and failures automatically. Creating new memory systems or pheromone types is out of scope.

</domain>

<decisions>
## Implementation Decisions

### Learning Capture UX

- **Automatic observation** — Colony observes patterns during builds without user input; no manual prompting for "what did you learn"
- **Checkbox approval at continue** — Reuse Phase 34's approval flow pattern; user selects which captured learnings to promote
- **Silent skip if empty** — If no learnings were captured, skip the prompt entirely without notice

### Failure Logging Scope

- **Build failures** — Worker errors, timeouts, unhandled exceptions
- **Approach changes** — Worker self-reports when trying X doesn't work and switching to Y; requires agent convention for logging
- **All test failures** — Including TDD red-green cycle; captures test failures during development, not just final builds
- **NOT user redirects** — REDIRECT signals are intentional guidance, not failures

### Midden Structure

- **One file per type** — `midden/build-failures.md`, `midden/test-failures.md`, `midden/approach-changes.md`
- **Structured YAML/JSON entries** — Each entry includes: timestamp, phase, what failed, why, what worked instead
- **Parseable format** — Tools can read and aggregate failures across phases

### Promotion Threshold

- **1 observation + user approval** — Lowered from 5; user approval is the quality gate
- **Rationale** — 5-observation threshold is why QUEEN.md stays empty; if something is worth capturing once and user approves, it's valid wisdom

### Claude's Discretion

- Exact YAML/JSON schema for midden entries
- How to integrate failure logging into existing worker patterns
- Whether to append or prepend entries in midden files
- Error handling when midden write fails

</decisions>

<specifics>
## Specific Ideas

- "This should be an automatic thing" — user expects colony to observe and log without being asked
- Phase 34 checkbox pattern worked well — reuse for learning approval

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope.

</deferred>

---

*Phase: 36-memory-capture*
*Context gathered: 2026-02-21*
