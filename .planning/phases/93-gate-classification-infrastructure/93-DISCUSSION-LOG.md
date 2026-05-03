# Phase 93: Gate Classification Infrastructure - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-05-03
**Phase:** 93-Gate Classification Infrastructure
**Areas discussed:** Gate classification mapping, audit trail format, command surface

---

## Gate Classification Mapping

| Option | Description | Selected |
|--------|-------------|----------|
| hard_block | gatekeeper, watcher_veto, flags — never auto-resolved | ✓ |
| soft_block | auditor, complexity, tdd_evidence, anti_pattern, verification_loop, spawn_gate — auto-resolve when verified non-critical | ✓ |
| advisory | medic, runtime — log only, never block | ✓ |

**User's choice:** "you decide" — Claude classified all gates
**Notes:** GATE-02 locks gatekeeper and watcher_veto as hard_block. Flags added as hard_block because they represent explicit human signals. Pre-check gates (tests_pass, no_critical_flags) classified separately as hard_block.

---

## Audit Trail Format

| Option | Description | Selected |
|--------|-------------|----------|
| Extend GateCheckResult | Add queen_annotation field to existing struct | ✓ |
| New queen-decisions JSON | Separate file for queen annotations | |
| Annotate inline | Modify existing fields in place | |

**User's choice:** "you decide" — Claude chose extend GateCheckResult
**Notes:** Extending the existing struct keeps everything in one file (gate-results-{N}.json). Original finding text, fix hint, and recovery options are never modified — queen's decision is appended alongside.

---

## Command Surface

| Option | Description | Selected |
|--------|-------------|----------|
| Table + JSON | Human-readable table default, --json flag for agents | ✓ |
| Table only | Just human-readable output | |
| JSON only | Just machine-readable output | |

**User's choice:** "you decide" — Claude chose dual output (table + JSON)
**Notes:** Follows existing OutputWorkflow pattern. Classification data stored as Go map constant, not config file.

---

## Claude's Discretion

- All three areas were deferred to Claude ("you decide")
- Gate classification rationale: hard_block for security/human signals, soft_block for quality/code issues, advisory for diagnostics
- Audit trail: queen_annotation struct with decision, rationale, timestamp, queen_version
- Command: `aether gate-classify` with table default and `--json` flag

## Deferred Ideas

None — discussion stayed within phase scope.
