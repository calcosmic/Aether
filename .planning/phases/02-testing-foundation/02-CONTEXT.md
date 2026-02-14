# Phase 2: Testing Foundation - Context

**Gathered:** 2026-02-13
**Status:** Ready for planning

<domain>
## Phase Boundary

Fix the 2 remaining Oracle-discovered bugs (duplicate status key, event timestamp ordering) and add tests to verify fixes + ensure core system functionality works correctly. Focus on bug fixes first, then regression tests.

</domain>

<decisions>
## Implementation Decisions

### Oracle Bug Fixes (Priority)
- Fix duplicate "status" key in COLONY_STATE.json task 1.1
- Fix event timestamp ordering in events array (chronological order)

### Testing Approach
- Add tests that verify the specific Oracle bugs are fixed
- Test core system functionality: state loading, command execution, file operations
- Bash integration tests for aether-utils.sh critical paths
- Existing tests should continue to pass

### Test Framework
- AVA for Node.js utilities
- Bash tests using simple assertions (no external framework needed)
- Test location: `tests/` directory at project root

### Coverage Scope
- Critical paths: state management, command execution, file operations
- Target: Cover the Oracle bug scenarios + core workflows
- Out of scope for now: Full coverage of all 59 subcommands

### Claude's Discretion
- Exact test file organization
- Specific assertion style
- Test helper utilities
- CI integration approach

</decisions>

<specifics>
## Specific Ideas

Oracle found these specific issues:
1. Task 1.1 in COLONY_STATE.json has duplicate "status" keys
2. Events at lines 173-175 have timestamps before initialization (11:18:15Z, 11:20:00Z, 11:30:00Z vs 16:00:00Z)

Want tests that catch these specific problems.

</specifics>

<deferred>
## Deferred Ideas

- Full test coverage of all 59 aether-utils.sh subcommands — future phase
- Performance testing — not needed now
- Visual/regression testing — out of scope
- Mock-based unit tests for external dependencies — future if needed

</deferred>

---

*Phase: 02-testing-foundation*
*Context gathered: 2026-02-13*
