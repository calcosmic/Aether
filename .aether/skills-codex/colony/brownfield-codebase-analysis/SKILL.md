---
name: brownfield-codebase-analysis
description: Use during colonize to understand an existing codebase before planning work
type: colony
domains: [analysis, onboarding, codebase-intelligence]
agent_roles: [surveyor-provisions, surveyor-nest, surveyor-disciplines, surveyor-pathogens, scout, architect]
workflow_triggers: [colonize]
priority: normal
version: "1.0"
---

# Brownfield Codebase Analysis

## Purpose

When a colony lands on an existing codebase (brownfield project), it needs structured understanding before it can plan work. This skill performs deep analysis producing a colonization report -- the colony's orientation map.

## When to Use

- User says "analyze this codebase" or "colonize this repo"
- Colony is initialized on an existing project with code already present
- Before planning phases on a brownfield project
- When onboarding to a project the colony hasn't seen before

## Instructions

### 1. Reconnaissance Sweep

```
1. Scan top-level files: package.json, Cargo.toml, go.mod, pyproject.toml, etc.
2. Identify primary language, framework, and build system
3. Detect test framework and CI/CD configuration
4. Map directory structure (top 3 levels)
5. Identify entry points (main files, index files, config files)
```

### 2. Architecture Mapping

```
1. Trace import/dependency graph from entry points
2. Identify architectural patterns (MVC, microservices, monorepo, etc.)
3. Map module boundaries and their responsibilities
4. Detect data flow: where data enters, transforms, and persists
5. Catalog external service integrations
```

### 3. API Surface Catalog

```
1. Extract all public API endpoints (REST, GraphQL, RPC)
2. Map function signatures of exported/public modules
3. Identify CLI commands if applicable
4. Document authentication and authorization patterns
5. Note rate limiting, versioning, and deprecation markers
```

### 4. Data Layer Analysis

```
1. Identify database type and schema (SQL models, NoSQL documents)
2. Map ORM/ODM usage and migration patterns
3. Trace data flow from user input to persistence
4. Identify caching layers and strategies
5. Catalog background jobs and queue systems
```

### 5. Tech Debt Scan

```
1. Search for TODO, FIXME, HACK, XXX comments
2. Identify deprecated dependency usage
3. Detect dead code paths (unused exports, unreachable branches)
4. Flag inconsistent patterns (mixed state management, duplicate utilities)
5. Note security anti-patterns (hardcoded secrets, SQL injection risks)
```

### 6. Colony Report Generation

Write structured documents to `.aether/data/colonization/`:

- `tech-stack.md` -- Language, frameworks, tools, versions
- `architecture.md` -- System design, module map, data flows
- `api-catalog.md` -- All public interfaces
- `data-layer.md` -- Storage, schemas, migrations
- `tech-debt.md` -- Prioritized debt inventory
- `colony-orientation.md` -- Executive summary for colony onboarding

## Key Patterns

- **Shallow first, deep on demand**: Start with broad surface analysis, go deeper only where the colony will work.
- **Emit pheromones**: Tag high-complexity areas with warning pheromones so planners know to budget extra time.
- **Respect `.gitignore`**: Never analyze node_modules, vendor, or build artifacts.
- **Incremental updates**: If colonization data exists, diff against it rather than rebuilding from scratch.

## Output Format

```
 COLONIZATION REPORT -- {project_name}
   Language: {primary} ({version})
   Framework: {name} ({version})
   Architecture: {pattern}
   Modules: {count} identified
   APIs: {count} endpoints
   Data Stores: {list}
   Tech Debt Items: {count} ({critical} critical, {high} high)
   Colonization Score: {1-10} (10 = fully understood)

   Key Risks: {top 3 risk areas}
   Recommended First Phase: {suggestion}
```

## Examples

**Colonize a Node.js API:**
> "Colonizing... Detected Express.js API with PostgreSQL, Redis cache, and Bull queue system. 47 endpoints cataloged. 23 tech debt items found (3 critical: auth token rotation, SQL string concatenation in user search, expired SSL cert in config). Colony orientation complete."

**Re-colonize after changes:**
> "Incremental colonization... 3 new modules detected, 2 APIs changed. Tech debt reduced by 5 items. Colonization score improved from 6 to 8."
