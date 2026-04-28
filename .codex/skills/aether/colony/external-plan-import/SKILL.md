---
name: external-plan-import
description: Use when an external PRD, roadmap, spec, or plan must be imported into colony planning safely
type: colony
domains: [planning, import, conflict-detection]
agent_roles: [architect, route_setter]
workflow_triggers: [plan]
task_keywords: [import, external plan, prd, roadmap, spec]
priority: normal
version: "1.0"
---

# External Plan Import

## Purpose

Imports external plans -- from documents, other tools, or previous projects -- into the colony's planning structure. Detects conflicts with existing colony decisions before writing anything, so the colony's integrity is never compromised by external input.

## When to Use

- User says "import this plan" or "use this spec"
- User provides a PRD, spec, or design document from outside
- Migrating plans from another tool (Jira, Linear, Notion)
- User has a pre-written ROADMAP they want the colony to adopt
- Merging plans from a different colony

## Instructions

### 1. Source Detection

```
Detect plan format:
  - Markdown document (PRD, spec, design doc)
  - JSON/YAML structured plan
  - Plain text task list
  - External tool export (detect format headers)
```

### 2. Plan Parsing

```
Extract from the source:
  1. Goals and objectives
  2. Tasks/phases with descriptions
  3. Dependencies between tasks
  4. Priority ordering
  5. Technical specifications
  6. Constraints and assumptions
  7. Success criteria / acceptance tests
```

### 3. Conflict Detection

Before importing, check against existing colony context:

```
CONFLICTS TO CHECK:
  1. Architecture decisions: Does the plan assume a different stack?
  2. Naming conventions: Does it use conflicting names?
  3. Dependency conflicts: Does it require packages not in the project?
  4. Phase overlap: Does it duplicate existing planned work?
  5. Constraint violations: Does it violate known project constraints?
  6. Priority conflicts: Does it reprioritize existing work?
```

### 4. Conflict Report

```
 CONFLICT REPORT -- {source_name}
   
    Compatible: {count} items (no conflicts)
    Partial:    {count} items (adjustable conflicts)
    Conflicts:  {count} items (hard conflicts requiring decision)
   
   Hard Conflicts:
   1. Plan assumes MongoDB, colony uses PostgreSQL
   2. Plan creates phase "Auth", existing phase 3 already covers auth
   3. Plan targets Node 18, colony uses Node 20
   
   Resolution Options:
   [A] Adapt plan to colony decisions (recommended)
   [B] Override colony decisions with plan
   [C] Import only non-conflicting items
   [D] Interactive: resolve each conflict individually
```

### 5. Import Process

```
After conflict resolution:
  1. Translate plan sections into colony ROADMAP phases
  2. Map tasks to phase waves
  3. Preserve original plan as reference document
  4. Add imported phases to ROADMAP.md
  5. Create phase directories with imported context
  6. Update dependency graph
  7. Emit import-complete pheromone
```

### 6. Import Metadata

```
Every imported plan gets metadata:
  - Source: Where it came from
  - Import date: When it was imported
  - Conflicts resolved: What was adjusted
  - Original preserved: Path to unmodified source
  - Mapping: How sections map to colony phases
```

## Key Patterns

- **Detect before write**: Never import without checking for conflicts first.
- **Preserve the original**: The imported plan is kept as-is alongside the adapted version.
- **Colony decisions win by default**: When in doubt, trust existing colony context.
- **Explicit resolution**: Every conflict gets a clear resolution, not a silent override.

## Output Format

```
 IMPORT -- {source_name}
   Phases extracted: {count}
   Conflicts found: {count} ({resolved} resolved)
   Items imported: {count}
   Items skipped: {count} ({reason})
   ROADMAP updated: phases {list} added
   Original: .aether/data/imports/{source_name}
```

## Examples

**Clean import:**
> "Imported 'Q2 roadmap' -- 5 phases extracted, no conflicts. Phases 4-8 added to ROADMAP. Ready to plan phase 4."

**Conflict resolution:**
> "Imported 'auth redesign plan' -- 3 conflicts detected. Resolved: adapted MongoDB refs to PostgreSQL, merged auth phase with existing phase 3, updated Node version to 20. 2 items imported, 1 merged."
