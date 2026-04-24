---
name: comprehensive-codebase-map
description: Use when the colony needs a complete multi-perspective map of a complex codebase
type: colony
domains: [codebase, analysis, mapping]
agent_roles: [surveyor-provisions, surveyor-nest, surveyor-disciplines, surveyor-pathogens, architect, scout]
workflow_triggers: [colonize]
task_keywords: [map, architecture, layout, disciplines, provisions, pathogens]
priority: normal
version: "1.0"
---

# Comprehensive Codebase Map

## Purpose

Comprehensive codebase analysis using 4 parallel specialist agents that each produce focused documents. Together, these 7 documents give the colony complete understanding of the codebase -- its structure, flows, surfaces, risks, and conventions.

## When to Use

- User says "map the codebase" or "full analysis"
- Colony needs deep understanding before major planning
- Onboarding to a complex existing codebase
- After significant changes to verify understanding is current
- Pre-milestone planning on large projects

## Instructions

### 1. Specialist Agent Dispatch

Launch 4 parallel analysis agents:

**Agent A -- Architect Analyst:**
```
Produces:
  1. ARCHITECTURE.md -- System design, module boundaries, dependency graph
  2. DATA-FLOWS.md   -- How data moves through the system
```

**Agent B -- API Cataloger:**
```
Produces:
  3. API-SURFACES.md  -- All public APIs, endpoints, interfaces
  4. INTEGRATIONS.md  -- External services, third-party connections
```

**Agent C -- Quality Scout:**
```
Produces:
  5. CONVENTIONS.md   -- Code style, naming patterns, architectural decisions
  6. TECH-DEBT.md     -- Prioritized debt inventory with severity
```

**Agent D -- Risk Assessor:**
```
Produces:
  7. RISK-MAP.md      -- Security, performance, and maintainability risks
```

### 2. Mapping Protocol

Each agent follows this protocol:

```
PHASE 1 -- Reconnaissance:
  Scan directory structure, identify file types, count modules
  
PHASE 2 -- Deep Read:
  Read source files relevant to their specialty
  Extract patterns, relationships, and anomalies
  
PHASE 3 -- Analysis:
  Cross-reference findings with other agents' domains
  Identify gaps and contradictions
  
PHASE 4 -- Document:
  Write structured findings to .aether/data/codebase/
```

### 3. Convergence

After all agents complete:

```
1. Merge agent findings into unified understanding
2. Resolve any contradictions between agent reports
3. Generate cross-references between documents
4. Produce CODEBASE-SUMMARY.md -- executive overview
5. Update codebase intelligence index
6. Emit mapping-complete pheromone
```

### 4. Document Formats

Each document follows:
```
# {Document Title}

## Overview
{One paragraph summary}

## Detailed Findings
{Structured findings}

## Cross-References
{Links to related documents}

## Recommendations
{Action items based on findings}

## Metadata
{Scan date, file count, agent ID}
```

### 5. Incremental Mapping

```
If prior mapping exists:
  1. Read previous documents
  2. Detect files changed since last mapping
  3. Only re-analyze changed files and affected modules
  4. Merge updated findings with existing documents
  5. Highlight deltas in each document
```

## Key Patterns

- **Parallel, not sequential**: All 4 agents run simultaneously.
- **Specialists, not generalists**: Each agent focuses on its domain deeply.
- **Cross-referenced**: Documents link to each other for navigation.
- **Incremental**: Re-mapping only touches what changed.

## Output Format

```
 MAPPING COMPLETE -- {project_name}
   Agents dispatched: 4
   Documents produced: 7 + 1 summary
   Files analyzed: {count}
   Duration: {time}
   
   Documents:
    ARCHITECTURE.md  -- {module_count} modules mapped
    DATA-FLOWS.md    -- {flow_count} data flows traced
    API-SURFACES.md  -- {api_count} APIs cataloged
    INTEGRATIONS.md  -- {integration_count} integrations found
    CONVENTIONS.md   -- {pattern_count} patterns identified
    TECH-DEBT.md     -- {debt_count} items ({critical} critical)
    RISK-MAP.md      -- {risk_count} risks assessed
   
   Location: .aether/data/codebase/
```

## Examples

**Full mapping:**
> "4 agents completed in 45 seconds. 7 documents produced. 142 files analyzed. Key finding: circular dependency between auth and user modules. Risk: 2 critical security items in API surfaces."

**Incremental update:**
> "Re-mapping after phase 3 changes. 12 files changed. Updated ARCHITECTURE.md and API-SURFACES.md. 3 new APIs cataloged, 2 deprecated APIs removed."
