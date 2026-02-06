# Phase 37: Command Trim & Utilities - Context

**Gathered:** 2026-02-06
**Status:** Ready for planning

<domain>
## Phase Boundary

Shrink remaining commands and reduce aether-utils.sh to reach the ~1,800 line target. This is a reduction/consolidation phase — trimming existing code, not adding features.

Targets from roadmap:
- colonize.md: 538 → ~150 lines
- status.md: 303 → ~80 lines
- Signal commands (focus, redirect, feedback): → ~40 lines each
- aether-utils.sh: 372 → ~80 lines

</domain>

<decisions>
## Implementation Decisions

### Colonize reduction
- Surface scan only: file tree + key files (package.json, README, entry points), ~20 files max
- Output to `.planning/CODEBASE.md` file (no terminal summary needed)
- Moderate structure info: tech stack, entry points, key directories, file counts per directory, dependency list, test location (~50 lines max output)
- Colonize is for new projects only — existing colonies use /ant:resume-colony

### Status command output
- Quick glance purpose: answer "where are we?" in ~5 lines
- Show signal count only ("3 active signals"), not details
- Show worker count only ("2 workers active"), not names
- Structured sections format: headers with dividers for phase, tasks, signals sections

### Signal commands
- Keep three separate commands: /ant:focus, /ant:redirect, /ant:feedback
- One-line confirmation output: "✓ FOCUS signal emitted: [message preview]"
- Keep content length validation (~500 char max)

### Claude's Discretion
- Whether signal commands share a common template/structure (goal: ~40 lines each)
- Which logging helpers to keep (fit within ~80 lines total for utils)
- Whether aether-utils.sh stays bash or converts to markdown
- Per-utility decision: inline into callers vs delete entirely

### Utility functions
- Keep: validate-state, error-add
- Keep: some logging helpers (Claude decides which fit the budget)
- Removed utilities should be inlined where used, not just deleted

</decisions>

<specifics>
## Specific Ideas

No specific requirements — open to standard approaches for achieving line count targets.

</specifics>

<deferred>
## Deferred Ideas

- Integrate CDS-style .planning/ folder pattern into Aether — suggested during colonize discussion, would be its own phase

</deferred>

---

*Phase: 37-command-trim-utilities*
*Context gathered: 2026-02-06*
