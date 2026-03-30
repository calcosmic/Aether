# Phase 41: Midden Collection - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-03-31
**Phase:** 41-midden-collection
**Areas discussed:** API shape, cross-PR auto-emit, wiring points, pruning

---

## API Shape Mismatch

| Option | Description | Selected |
|--------|-------------|----------|
| Design doc API (worktree-path) | Use --worktree-path directly since midden is gitignored | |
| Smart wrapper (Recommended) | Keep --branch --merge-sha as user-facing API, internally resolve worktree path | ✓ |
| You decide | | → Selected |

**User's choice:** "You decide" — delegated to Claude. Smart wrapper chosen for user-friendliness.
**Notes:** ROADMAP promises `--branch --merge-sha` but midden data is gitignored so git show won't work. Smart wrapper resolves worktree path internally.

---

## Cross-PR Auto-Emit

| Option | Description | Selected |
|--------|-------------|----------|
| Auto-emit REDIRECT (Recommended) | System automatically tells all colonies to avoid problematic pattern | ✓ |
| Report only | System reports finding but waits for user to decide | |
| Tiered approach | Auto-emit for critical (4+ PRs), report for mild (2-3 PRs) | |

**User's choice:** Auto-emit REDIRECT
**Notes:** Like a smoke detector — works even when the user isn't actively monitoring.

---

## Wiring Points

| Option | Description | Selected |
|--------|-------------|----------|
| Continue + Run (Recommended) | Wire into /ant:continue and /ant:run | ✓ |
| Continue + Run + Status | Wire into all three for maximum coverage | |
| Git hook only | Post-merge hook fires automatically | |

**User's choice:** Continue + Run
**Notes:** These are the existing workflows where merges happen. Simpler than git hooks.

---

## Pruning Commands

| Option | Description | Selected |
|--------|-------------|----------|
| Defer pruning (Recommended) | Build core 3 subcommands now, add pruning later | |
| Include pruning | Build all commands together for completeness | ✓ |

**User's choice:** Include pruning
**Notes:** User wants the system complete from day one.

---

## Claude's Discretion

- API shape: smart wrapper approach (delegated by user)
- Exact worktree path resolution logic
- Cross-PR score formula tuning
- Output format details
- --dry-run flag

## Deferred Ideas

- `/ant:status` integration for cross-PR analysis
- Closed-without-merge worktree tracking
- Re-revert handling
