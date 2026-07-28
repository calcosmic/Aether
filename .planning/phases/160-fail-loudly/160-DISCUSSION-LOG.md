# Phase 160: Fail Loudly - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-07-27
**Phase:** 160-fail-loudly
**Areas discussed:** Failure behavior, Unblock guidance, Fold today's findings

---

## Area selection

| Option | Description | Selected |
|--------|-------------|----------|
| Failure behavior | Halt vs continue when a load-bearing CLI call fails | ✓ |
| Unblock guidance | Build /ant-unblock wrapper vs repoint the message | ✓ |
| Fold today's findings | Where the 2026-07-27 deferred findings land | ✓ |
| Pull visuals earlier | Move Phase 168 work earlier in milestone order | |

---

## Failure behavior

| Option | Description | Selected |
|--------|-------------|----------|
| Depends on the call (Recommended) | Gates halt; enrichment continues with loud warning | ✓ |
| Always halt | Any failed call stops the run | |
| Always continue loudly | Visible error, nothing stops | |

**User's choice:** Depends on the call
**Notes:** Plan must classify each audited call as gate vs enrichment.

---

## Unblock guidance

| Option | Description | Selected |
|--------|-------------|----------|
| Build the wrapper (Recommended) | Create /ant-unblock on Claude Code + OpenCode | ✓ |
| Repoint the message | Name the real CLI command instead | |
| You decide | Claude picks based on wiring cost | |

**User's choice:** Build the wrapper

---

## Fold today's findings

| Option | Description | Selected |
|--------|-------------|----------|
| Split by theme (Recommended) | Debug-file trio → 160; contract items → 163 | ✓ |
| All into Phase 160 | One sweep now | |
| All deferred to 163 | Keep 160 exactly as drafted | |

**User's choice:** Split by theme

---

## Claude's Discretion

- Ledger format/location (RETIRE-04)
- Drift-audit mechanics (LOUD-04/05)
- Gate-vs-enrichment classification details
- /ant-unblock wrapper wording
- CLAUDE.md:838 one-line fix folded into LOUD-07

## Deferred Ideas

- Route-setter/scout permission contradictions → Phase 163
- `artifacts` sub-schema unusable escape hatch → Phase 163
- Pull Phase 168 visual work earlier — raised, not selected; order stands
- AETHER_PREFLIGHT_TIMEOUT env knob; delete dead TS preflight
- Stale worktree copies under `cmd/.aether/worktrees/` grep pollution
