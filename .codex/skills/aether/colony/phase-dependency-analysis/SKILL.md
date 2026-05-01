---
name: phase-dependency-analysis
description: Use when phase or task ordering needs dependency, overlap, or blast-radius analysis
type: colony
domains: [analysis, dependency-management, planning]
agent_roles: [route_setter, architect, scout]
workflow_triggers: [plan]
task_keywords: [dependency, dependencies, ordering, parallel, blast radius]
priority: normal
version: "1.0"
---

# Phase Dependency Analysis

## Purpose
Analyzes phase dependencies through three lenses -- file overlap, semantic dependencies, and data flow -- to auto-suggest optimal phase ordering. Produces a dependency graph that the roadmap-manager can use to validate or restructure the colony's execution plan.

## When to Use
- `aether plan` is run and the roadmap has phases without explicit dependencies
- The architect suspects phases may conflict or could be parallelized
- A phase failed and the queen needs to understand blast radius
- Before reordering the roadmap to ensure valid dependency chains
- When adding a new phase to determine where it fits in the dependency graph
- During colony init to generate the initial dependency relationships

## Instructions

### Step 1 -- Collect Phase Data
For each phase in the roadmap:

1. Read the phase description, goal, and scope from `.aether/roadmap.md`
2. If PLAN.md exists, read it for task-level file paths
3. If SPEC.md exists, read it for interface contracts
4. If CONTEXT.md exists, read it for codebase context
5. Extract: expected files to create/modify, APIs to consume/expose, data models involved

Record per-phase data:
```
phase: {N}
title: {title}
files_expected: [{paths or glob patterns}]
apis_produced: [{interface names}]
apis_consumed: [{interface names}]
data_models: [{model names}]
shared_state: [{global state, config, env vars}]
```

### Step 2 -- File Overlap Analysis
Compare every pair of phases for file overlap:

```
overlap(A, B) = |files_expected(A)  files_expected(B)| / |files_expected(A)  files_expected(B)|
```

Classification:
- **High overlap (>0.3)**: Phases likely conflict. Must be sequential, or tasks must be split to avoid simultaneous file access.
- **Medium overlap (0.1-0.3)**: Possible conflict. Review for read vs write patterns.
- **Low overlap (<0.1)**: Unlikely to conflict. Good candidates for parallel execution.

Record overlaps:
```markdown
## File Overlap Matrix

| | P1 | P2 | P3 | P4 |
|---|---|---|---|---|
| P1 | -- | H | L | M |
| P2 | H | -- | L | L |
| P3 | L | L | -- | M |
| P4 | M | L | M | -- |
```

### Step 3 -- Semantic Dependency Analysis
Determine logical dependencies between phases:

1. **Producer-Consumer**: Phase A creates an API or module that Phase B uses -> A must precede B
2. **Foundation-Extension**: Phase A establishes a pattern or infrastructure that Phase B extends -> A must precede B
3. **Independent**: Neither phase references the other's outputs -> can be parallel
4. **Mutually Exclusive**: Phases implement alternative approaches to the same problem -> pick one

For each phase pair, classify the relationship and assign a dependency type:
```
pair: ({A}, {B})
relationship: {producer-consumer | foundation-extension | independent | mutually-exclusive}
direction: {A->B | B->A | none | conflict}
confidence: {high | medium | low}
reasoning: {why this relationship exists}
```

### Step 4 -- Data Flow Analysis
Trace data dependencies between phases:

1. Identify data models each phase creates, reads, or modifies
2. Identify shared state (databases, config files, environment variables)
3. Map data flow: who creates data, who reads it, who transforms it

```
data_flow:
  - model: {User}
    created_by: [Phase 1]
    read_by: [Phase 2, Phase 4]
    modified_by: [Phase 3]
    implies: Phase 1 -> Phase 2, Phase 1 -> Phase 3, Phase 1 -> Phase 4
```

### Step 5 -- Synthesize Dependency Graph
Combine all three analyses into a unified dependency graph:

```
nodes: [{phase numbers}]
edges:
  - from: {A} to: {B}
    type: {hard | soft | none}
    sources: [{file_overlap | semantic | data_flow}]
    confidence: {high | medium | low}
```

**Edge types:**
- **Hard**: Must be sequential. Derived from producer-consumer or foundation-extension relationships.
- **Soft**: Should be sequential but could be parallel with coordination. Derived from medium file overlap or indirect data flow.
- **None**: No dependency. Can run in parallel.

### Step 6 -- Generate Ordering Suggestions
From the dependency graph, produce:

1. **Suggested ordering**: Topological sort of hard edges only
2. **Parallelization opportunities**: Groups of phases with no hard edges between them
3. **Risk areas**: Soft dependencies that could become hard if assumptions are wrong
4. **Cycle warnings**: Any circular dependencies detected

Output:
```markdown
## Suggested Phase Ordering

### Sequential Chain (hard dependencies)
Phase 1 -> Phase 3 -> Phase 5

### Parallel Group A (no dependencies between)
Phase 2, Phase 4

### Parallel Group B (no dependencies between)
Phase 6, Phase 7 (both depend on Phase 5)

### Risk: Soft Dependencies
- Phase 2 -> Phase 4 (data_flow: shared User model, could conflict on schema changes)
  Mitigation: Phase 4 should read User model as read-only, coordinate schema changes through Phase 3
```

### Step 7 -- Write Dependency Graph
Save the complete analysis:

```markdown
# Colony Dependency Graph

## Summary
{N} phases analyzed, {X} hard dependencies, {Y} soft dependencies, {Z} parallel groups.

## Phase Data
{per-phase data from Step 1}

## File Overlap Matrix
{from Step 2}

## Semantic Relationships
{from Step 3}

## Data Flow Map
{from Step 4}

## Dependency Edges
{from Step 5}

## Ordering Suggestions
{from Step 6}

## Confidence Assessment
| Edge | Confidence | What would change it |
|------|-----------|---------------------|
| {A->B} | {H/M/L} | {evidence needed} |
```

## Key Patterns

### Three-Lens Validation
Never rely on a single analysis type. File overlap alone misses semantic dependencies. Semantic analysis alone misses data conflicts. All three must agree before marking a dependency as "none."

### Confidence Escalation
Low-confidence edges should be escalated to the architect for manual review. The analyzer makes recommendations; the architect makes decisions.

### Incremental Updates
When a new phase is added, don't re-anze everything. Compare only the new phase against existing ones. Update the graph incrementally.

## Output Format
- `.aether/context/dependency-graph.md` -- full analysis
- `.aether/context/dependency-matrix.json` -- machine-readable overlap/edge data
- Updates to `.aether/roadmap.md` dependency fields (if queen approves suggestions)

## Examples

### Example 1 -- Simple Linear Project
4-phase project to build a CRUD app.

File overlap: low between all phases (different files). Semantic: Phase 1 (models) -> Phase 2 (API) -> Phase 3 (frontend) -> Phase 4 (tests). Data flow: models created in Phase 1 consumed by all others.

Result: linear chain. No parallelism possible. Graph confirms the obvious ordering.

### Example 2 -- Parallel Opportunity
6-phase project with frontend and backend tracks.

File overlap: zero between frontend and backend phases. Semantic: Phase 1 (DB schema) is foundation for both tracks. Data flow: shared data model, but read-only on frontend.

Result: Phase 1 first, then Phase 2+3 (backend) and Phase 4+5 (frontend) in parallel. Phase 6 (integration) depends on both tracks. Significant time savings.

### Example 3 -- Hidden Conflict
Phases 3 and 5 appear independent but both modify the User model schema.

File overlap: medium (shared migration files). Semantic: independent goals but shared data model. Data flow: both write to User table.

Result: soft dependency detected. Recommendation: sequential with Phase 5 reading the schema that Phase 3 establishes. Risk flagged if both try to run migrations simultaneously.
