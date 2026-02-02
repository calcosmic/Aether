# Command Consolidation Review

**Captured:** 2026-02-01
**Source:** User feedback during Phase 5 completion

## Issue

There are currently **19 ant commands** total. User suggested reviewing whether some commands could be consolidated to reduce complexity.

## Current Commands (19 total)

1. `/ant:` - Main command index
2. `/ant:init` - Initialize colony with intention
3. `/ant:status` - Show colony state and status
4. `/ant:phase` - Show phase details and check-in status
5. `/ant:execute` - Execute a phase
6. `/ant:focus` - Emit FOCUS pheromone
7. `/ant:redirect` - Emit REDIRECT pheromone
8. `/ant:feedback` - Emit FEEDBACK pheromone
9. `/ant:continue` - Approve phase continuation at check-in
10. `/ant:adjust` - Adjust pheromones during check-in
11. `/ant:recover` - Recover colony from checkpoint
12. `/ant:memory` - Query and manage colony memory
13. `/ant:colonize` - Colonize codebase (Colonizer Ant)
14. `/ant:plan` - Plan phase structure (Route-setter Ant)
15. `/ant:build` - Build and implement code (Builder Ant)
16. `/ant:review` - Review quality (Watcher Ant)
17. `/ant:errors` - View error ledger
18. `/ant:pause-colony` - Pause colony work and create handoff
19. `/ant:resume-colony` - Resume colony from saved session

## Potential Consolidations

### Pheromone Commands (focus, redirect, feedback)
- **Current:** 3 separate commands
- **Consolidation:** `/ant:pheromone {type} {args}` or `/ant:signal {type}`
- **Pros:** Reduces command count, unified interface
- **Cons:** Less discoverable, requires remembering types
- **Recommendation:** Keep separate for discoverability (3 commands is reasonable)

### Colony Control Commands (pause-colony, resume-colony)
- **Current:** 2 separate commands
- **Consolidation:** `/ant:pause` with auto-resume via `/clear` (implicit)
- **Pros:** Simpler, .continue-here.md already created
- **Cons:** Less explicit control
- **Recommendation:** Consider consolidating to `/ant:pause` (creates handoff), `/clear` resumes

### Worker Ant Commands (colonize, plan, build, review)
- **Current:** 4 explicit caste commands (Colonizer, Route-setter, Builder, Watcher)
- **Note:** These are explicit Worker Ant caste invocations, not general-purpose
- **Consolidation:** `/ant:worker {caste}` or `/ant:caste {type}`
- **Pros:** Unified interface for Worker Ant invocation
- **Cons:** Less discoverable, requires knowing caste names
- **Recommendation:** Keep separate for caste clarity and discoverability

### Checkpoint/Recovery Commands (recover, memory)
- **Current:** Separate commands
- **Note:** Different purposes (recover from crash vs query memory)
- **Consolidation:** `/ant:state {action}` (recover, status, memory)
- **Pros:** Groups state-related operations
- **Cons:** Breaks existing command patterns
- **Recommendation:** Keep separate (different concerns)

## Summary

**Current count:** 19 commands
**After consolidations:** 16-17 commands

**Recommended actions:**
1. Keep pheromone commands separate (discoverability)
2. Consolidate `pause-colony`/`resume-colony` to just `/ant:pause` (resume via `/clear`)
3. Keep Worker Ant commands separate (caste clarity)
4. Keep checkpoint/memory commands separate (different concerns)

**Net reduction:** 1 command (18 total)

## Decision Needed

Should we proceed with consolidating pause-colony/resume-colony? Or are there other consolidation opportunities?
