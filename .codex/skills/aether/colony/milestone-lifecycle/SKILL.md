---
name: milestone-lifecycle
description: Use when starting, completing, archiving, or transitioning milestones between project stages
type: colony
domains: [project-management, milestone-tracking, lifecycle]
agent_roles: [keeper, chronicler, queen]
workflow_triggers: [seal, init]
task_keywords: [milestone, archive, wrap up, new milestone, transition]
priority: normal
version: "1.0"
---

# Milestone Lifecycle

## Purpose

Full milestone lifecycle management: creation, completion, archival, and next milestone preparation. Tracks milestone state across phases and ensures clean transitions between project stages.

## When to Use

- Starting a new milestone after the previous one completes or project begins
- All phases in a milestone are complete and ready for final review
- User says "new milestone", "complete milestone", "archive", or "wrap up vN"
- Before archiving to ensure all phases are properly documented
- Transitioning from one version/iteration to the next

## Instructions

### Create Milestone

1. Verify no active milestone exists (check `.aether/data/milestone-state.json`)
2. If a previous milestone exists, prompt to complete or archive it first
3. Create milestone record:
   ```json
   {
     "id": "MS-{NN}",
     "name": "{user-provided name}",
     "version": "{semver if applicable}",
     "created": "ISO8601",
     "status": "active",
     "phases": [],
     "goals": ["{goal 1}", "{goal 2}"],
     "success_criteria": ["{criterion 1}"]
   }
   ```
4. Create directory: `.aether/milestones/{NN}/`
5. Write `.aether/milestones/{NN}/MILESTONE.md` with goals, scope, and success criteria
6. Update `.aether/data/milestone-state.json` to set this as the active milestone

### Track Progress

1. As phases are planned and executed, update the milestone's phase list
2. Track per-phase status: `planned`, `in-progress`, `complete`, `skipped`, `reverted`
3. Compute milestone completion percentage: `(complete / total) * 100`
4. Report progress via `aether status`

### Complete Milestone

1. Verify all phases are in terminal state (`complete` or `skipped`)
2. Run completion checklist:
   - [ ] All phases have LEARNINGS.md
   - [ ] All phase manifests are up to date
   - [ ] No open todos reference this milestone's phases
   - [ ] Tests pass on the final state
   - [ ] Documentation reflects the current state
3. Extract milestone-wide learnings (invoke `learning-extractor --milestone {NN}`)
4. Generate milestone summary in `.aether/milestones/{NN}/SUMMARY.md`
5. Update status to `complete` with completion timestamp

### Archive Milestone

1. Verify milestone is `complete`
2. Run `learning-extractor --milestone {NN}` if not already done
3. Compress all milestone artifacts into `.aether/archive/{NN}/`
4. Remove active phase directories (optional, controlled by `--keep-phases` flag)
5. Update `.aether/data/milestone-state.json` to clear the active milestone
6. Retain colony knowledge base entries (they are cumulative, not milestone-scoped)

### Prepare Next Milestone

1. Review previous milestone's lessons and surprises
2. Scan `.aether/data/todos.jsonl` for promoted but unimplemented items
3. Review `.aether/data/seeds.jsonl` for triggered seeds
4. Present a "carry forward" list: items from the previous milestone that should inform the next
5. Invoke `aether init` or prompt user for new milestone goal

## Key Patterns

- **One active milestone**: Only one milestone can be active at a time
- **Terminal states only**: A milestone can only be archived if all phases are terminal (not in-progress)
- **Knowledge preservation**: Archiving removes files but preserves colony knowledge base entries
- **Carry-forward**: Always surface unfinished items and lessons when transitioning milestones

## Output Format

```
Milestone MS-02: "User Authentication & Profiles" -- CREATED
  Goals: OAuth login, profile management, role-based access
  Success criteria: Users can sign up, log in, edit profile, see role-appropriate content
  Directory: .aether/milestones/02/

Phase 1/5 complete (20%) | Phase 2 in-progress (40%)

Milestone MS-02 progress: 2/5 phases (40%)
```

## Examples

```
# Create new milestone
> milestone-lifecycle create "User Authentication & Profiles"

# Check progress
> milestone-lifecycle progress

# Complete and archive
> milestone-lifecycle complete
> milestone-lifecycle archive
```
