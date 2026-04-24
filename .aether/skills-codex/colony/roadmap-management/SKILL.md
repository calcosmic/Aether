---
name: roadmap-management
description: Use when the colony roadmap needs phases added, removed, reordered, inserted, or validated
type: colony
domains: [planning, project-management, roadmap]
agent_roles: [route_setter, architect, queen]
workflow_triggers: [plan]
task_keywords: [roadmap, reorder, insert, phase, backlog]
priority: normal
version: "1.0"
---

# Roadmap Management

## Purpose
Manages the colony's roadmap -- the ordered list of phases that must be completed. Handles adding new phases, removing cancelled ones, reordering priorities, inserting urgent work as decimal phases, and maintaining the backlog for future work.

## When to Use
- `aether roadmap add` -- add a new phase to the roadmap
- `aether roadmap remove` -- remove a phase and renumber
- `aether roadmap reorder` -- change phase ordering
- `aether roadmap insert` -- insert urgent work between existing phases
- `aether roadmap backlog` -- manage backlog items (999.x numbering)
- The queen decides to reprioritize the colony's work
- Scope changes require adding or removing planned phases

## Instructions

### Step 1 -- Read Current Roadmap
Read `.aether/roadmap.md` and parse into a structured phase list:

```markdown
# Colony Roadmap

## Active Phases
### Phase 1: {Title} -- {Status}
- Goal: {description}
- Depends on: {none | phase numbers}
- Estimated complexity: {S/M/L/XL}

### Phase 2: {Title} -- {Status}
...

## Backlog (999.x)
### Phase 999.1: {Title}
- Idea: {description}
- Priority: {low | medium | high}
- Proposed by: {source}
```

### Step 2 -- Execute Requested Operation

#### Add Phase (to end of active phases)
1. Determine the next phase number (last active phase number + 1)
2. Create the phase entry with the provided title, goal, and dependencies
3. If no dependency specified, default to depending on the previous phase
4. Write the updated roadmap
5. Create the phase directory: `.aether/phases/{N}/` with `state.md` (status: pending)

#### Remove Phase
1. Identify the phase to remove
2. Check for dependent phases -- any phase that lists this phase as a dependency
3. If dependents exist: warn and ask for confirmation (removing will orphan them)
4. Remove the phase entry from the roadmap
5. Renumber all subsequent phases (e.g., removing phase 3 makes old 4->3, 5->4)
6. Update all dependency references in remaining phases to reflect new numbering
7. Archive removed phase data to `.aether/phases/_archived/{original_number}-{title}/`
8. Write updated roadmap

#### Reorder Phases
1. Accept the new ordering as a list of phase numbers
2. Validate: all phases accounted for, no duplicates
3. Validate: new ordering respects hard dependencies (a phase cannot come before its dependency)
4. If invalid: report the conflict and suggest alternatives
5. Renumber phases according to new order
6. Update dependency references
7. Write updated roadmap

#### Insert Phase (Decimal Numbering)
1. Accept the insertion point (between phases N and N+1)
2. Assign decimal number: N.1 (or N.2 if N.1 exists, etc.)
3. The inserted phase depends on phase N by default
4. Phase N+1 now depends on the inserted phase (if it previously depended on N)
5. Write the phase entry with its decimal number
6. Create `.aether/phases/{N}.1/state.md`
7. Write updated roadmap

Note: Decimal phases are temporary. After insertion, the roadmap shows them inline. During a future renumber operation, they can be collapsed into integer numbering.

#### Backlog Management
1. Backlog items use 999.x numbering (999.1, 999.2, etc.)
2. To add: append to backlog section with idea description and priority
3. To promote: move from 999.x to the next active phase number, renumber
4. To deprioritize: move an active phase to backlog with a 999.x number
5. Write updated roadmap

### Step 3 -- Validate Roadmap Integrity
After any modification, verify:

1. **No orphan dependencies**: Every dependency reference points to an existing phase
2. **No circular dependencies**: Phase A cannot depend on B if B depends on A (directly or transitively)
3. **Sequential validity**: Dependencies always reference earlier phases (or same-level decimal parents)
4. **Status consistency**: Completed phases are not listed as dependencies with pending status
5. **Backlog isolation**: 999.x phases have no active dependencies and no active dependents

### Step 4 -- Update Colony State
Record the roadmap change in `.aether/data/roadmap-history.json`:
```json
{
  "timestamp": "ISO",
  "operation": "add|remove|reorder|insert|backlog",
  "description": "what changed",
  "affected_phases": [N, ...],
  "previous_state": "snapshot of affected entries before change"
}
```

## Key Patterns

### Renumbering Algorithm
When renumbering after removal:
1. Sort phases by current number
2. Assign new sequential numbers starting from 1
3. Build a mapping: {old: new}
4. Apply mapping to all dependency fields
5. Rename phase directories in `.aether/phases/`

### Dependency Chain Validation
Walk the dependency graph depth-first. If you revisit a node, there is a cycle. Report the cycle path and refuse the operation.

### Decimal Phase Conventions
- Use N.1, N.2, etc. for insertions between N and N+1
- Decimal phases should be resolved (collapsed to integers) before the colony seals
- Maximum one level of decimal nesting (no 3.1.1)

## Output Format
- `.aether/roadmap.md` -- updated roadmap
- `.aether/phases/{N}/state.md` -- new phase state files
- `.aether/data/roadmap-history.json` -- change log

## Examples

### Example 1 -- Adding a Phase
Current roadmap has phases 1-4. Queen adds "Add caching layer" after phase 3.

Result: Phase 5 created ("Add caching layer"), depends on phase 3 (or 4, based on queen input). `.aether/phases/5/` created with state: pending.

### Example 2 -- Removing with Renumbering
Phase 3 cancelled. Phases 4 and 5 depend on phase 3.

After confirmation: Phase 3 archived. Old phase 4 becomes phase 3 (dependency updated: "was phase 3, now phase 2" or removed if phase 3 was the dependency). Old phase 5 becomes phase 4. All dependency references updated.

### Example 3 -- Urgent Decimal Insertion
Between phase 2 (completed) and phase 3 (pending), a critical security fix is needed.

Inserted as phase 2.1 ("Security patch for auth bypass"). Phase 3 now depends on 2.1 instead of (or in addition to) 2. Phase 2.1 depends on phase 2.

### Example 4 -- Backlog Promotion
Phase 999.2 ("Add dark mode") promoted to active work. Currently 5 active phases.

Phase 999.2 becomes phase 6. Backlog renumbers: 999.3->999.2, etc. Phase 6 gets full `.aether/phases/6/` directory.
