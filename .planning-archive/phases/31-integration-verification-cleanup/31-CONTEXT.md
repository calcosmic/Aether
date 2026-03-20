# Phase 31: Integration Verification + Cleanup - Context

**Gathered:** 2026-02-20
**Status:** Ready for planning

<domain>
## Phase Boundary

Verify the full colony agent system works end-to-end (agent resolution, state updates, output compatibility), then clean house: trim .aether/docs/, fix the bash line wrapping bug, create repo-structure.md, update README for v2.0, and mark v2.0 as shipped with version bump + git tag.

</domain>

<decisions>
## Implementation Decisions

### Docs curation
- Priority: developer reference docs (architecture, known issues, error codes) over user guides
- Audience: the project owner re-orienting after a break — not new contributors
- Cut docs go to `.aether/docs/archive/` (not deleted)
- Claude audits all docs and recommends which 8-10 to keep — no must-keep list locked

### Repo documentation
- repo-structure.md lives in repo root (next to README.md)
- High-level overview only — top-level directories with one-line descriptions
- README updated to feature v2.0 agents as a key capability
- README tone: action-oriented — show what commands do, what agents exist, make it feel powerful

### Verification scope
- Verify the path from agent return → slash command → COLONY_STATE.json actually works (don't just trust existing code)
- Bash line wrapping bug: Claude investigates, identifies, fixes, and adds a test case
- Claude decides verification depth (wiring-only vs real invocation) and test approach (automated vs manual) based on risk

### Cleanup boundaries
- .planning/ directory: keep everything as-is (project history)
- Light tidy only: fix obviously misplaced files or stale state, don't reorganize
- Phase 31 includes marking v2.0 as shipped (update ROADMAP.md + STATE.md)
- v2.0 "done" = version bump to 2.0.0 in package.json + git tag + npm publish

### Claude's Discretion
- Which specific docs survive the 8-10 trim (audit and recommend)
- Verification depth: wiring-only vs one real agent invocation
- Whether INT-02 needs automated tests or manual spot-checks
- Exact structure and content of repo-structure.md
- How to find and fix the bash line wrapping bug

</decisions>

<specifics>
## Specific Ideas

- README should feel action-oriented: "run /ant:build to spawn a real builder agent that writes code"
- Docs audience is "me coming back after a break" — quick re-orientation, not onboarding guide
- Archive folder preserves cut docs without cluttering the main docs directory

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 31-integration-verification-cleanup*
*Context gathered: 2026-02-20*
