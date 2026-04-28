---
name: documentation-ingestion
description: Use when existing project documentation must be discovered and converted into colony context
type: colony
domains: [documentation, ingestion, onboarding]
agent_roles: [scout, chronicler, keeper, architect]
workflow_triggers: [colonize, plan]
task_keywords: [documentation, docs, readme, adr, ingest]
priority: normal
version: "1.0"
---

# Documentation Ingestion

## Purpose

Scans a repository for any existing documentation -- ADRs, PRDs, specs, docs, READMEs, wikis -- and classifies, consolidates, and bootstraps colony context from them. Turns a scattered doc landscape into structured colony knowledge.

## When to Use

- User says "ingest our docs" or "scan for documentation"
- Colony initializing on a project with existing documentation
- Before planning to ensure all prior decisions are captured
- User wants to consolidate scattered docs into colony structure

## Instructions

### 1. Discovery Scan

```
Search for documentation files:
  *.md, *.rst, *.txt, *.adoc               -- Standard doc formats
  docs/, documentation/, wiki/, .wiki/      -- Doc directories
  ADR-*, DECISION-*, RFC-*                  -- Architecture records
  PRD-*, SPEC-*, REQUIREMENTS-*             -- Product specs
  CHANGELOG*, HISTORY*, ROADMAP*            -- Project history
  CONTRIBUTING*, CODE_OF_CONDUCT*           -- Community docs
  .github/ISSUE_TEMPLATE/*, .github/PULL_REQUEST_TEMPLATE/* -- Templates
```

### 2. Classification

```
For each document found:
  ADR:       Architecture Decision Record -- technical decision with context
  PRD:       Product Requirements Document -- what to build and why
  SPEC:      Technical Specification -- how to build it
  DOC:       General documentation -- guides, README, tutorials
  TEMPLATE:  Issue/PR templates
  HISTORY:   Changelogs, release notes
  CONFIG:    Configuration documentation
  UNKNOWN:   Doesn't fit categories -- flag for manual review
```

### 3. Content Extraction

```
From each document:
  1. Extract title and summary
  2. Identify key decisions and their rationale
  3. Extract requirements and acceptance criteria
  4. Identify stakeholders and ownership
  5. Detect references to other documents
  6. Note the document's age and last update
```

### 4. Conflict Detection

```
When multiple documents overlap:
  1. Detect contradictions (e.g., two ADRs with different tech choices)
  2. Detect duplicates (same information in multiple places)
  3. Detect outdated information (contradicts current code)
  4. Detect gaps (important areas with no documentation)
```

### 5. Consolidation

```
Create colony-ready documents:
  
  .aether/data/ingested/
     catalog.json           # All found docs with metadata
     decisions.md           # Consolidated decisions from ADRs
     requirements.md        # Consolidated requirements from PRDs
     specifications.md      # Consolidated specs
     conflicts.md           # Detected contradictions
     gaps.md                # Undocumented areas
     originals/             # Symlinks/copies of source docs
```

### 6. Colony Bootstrap

```
From ingested docs, populate colony context:
  1. Feed decisions into colony knowledge base
  2. Feed requirements into ROADMAP planning
  3. Feed specs into phase specifications
  4. Flag conflicts for user resolution
  5. Recommend documentation updates for gaps
```

## Key Patterns

- **Ingest, don't replace**: Original documents stay where they are; colony creates references.
- **Classify automatically**: Most docs can be classified by naming patterns and content.
- **Surface contradictions**: Conflicting docs are more valuable than agreeing ones -- they reveal ambiguity.
- **Gap identification**: Knowing what's NOT documented is as important as what is.

## Output Format

```
 INGESTION -- {repo_name}
   Documents found: {count}
   Classification:
     ADRs: {count} | PRDs: {count} | Specs: {count}
     Docs: {count} | Templates: {count} | Other: {count}
   
   Conflicts: {count} contradictions detected
   Gaps: {count} undocumented areas identified
   
   Colony context updated:
    {count} decisions imported
    {count} requirements captured
    {count} specifications cataloged
   
   Catalog: .aether/data/ingested/catalog.json
```

## Examples

**Full ingestion:**
> "Found 23 documents: 4 ADRs, 2 PRDs, 3 specs, 14 docs. 1 conflict: ADR-003 says 'use Redis' but PRD-v2 says 'use Memcached'. 5 gaps: no docs for auth, payments, deployment, monitoring, or error handling."

**Quick scan:**
> "Found 5 docs. README.md (general), CONTRIBUTING.md (template), 3 ADRs. No PRDs or specs found. Colony context populated with 3 architecture decisions."
