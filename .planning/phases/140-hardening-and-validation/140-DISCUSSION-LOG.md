# Phase 140: Hardening and Validation - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-05-18
**Phase:** 140-hardening-and-validation
**Areas discussed:** Wrapper alignment

---

## Wrapper Alignment

| Option | Description | Selected |
|--------|-------------|----------|
| Snapshot testing | Automated tests comparing wrapper and host output for consistency | ✓ |
| Manual comparison checklist | Manual run-through of each command | |
| Structured contract verification | Parse output into structured data, then diff sections | |

**User's choice:** Snapshot testing

| Option | Description | Selected |
|--------|-------------|----------|
| Section presence only | Test ceremony sections appear in both, don't test exact text | ✓ |
| Exact output matching | Test exact ANSI output matches | |
| Substring containment | Test runtime ceremony markers appear inside wrapper output | |

**User's choice:** Section presence only

| Option | Description | Selected |
|--------|-------------|----------|
| All 4 commands | Test build, plan, continue, oracle | |
| Build + continue only | Test the two most complex commands | |
| All commands | Test every command including init, status, seal, etc. | ✓ |

**User's choice:** All commands

**Notes:** Wrapper alignment is the primary concern for Phase 140. User chose snapshot tests with section presence verification covering all commands. Wrappers add Queen narration around runtime ceremony — tests confirm ceremony markers exist, not exact text match.

---

## Claude's Discretion

- Test file structure (per-command or grouped)
- Whether to test Claude wrappers, OpenCode wrappers, or both
- Fixture generation approach
- Milestone audit format
- Go-level integration test scope
- Regression test scope (full vs targeted)

## Deferred Ideas

None — discussion stayed within phase scope.
