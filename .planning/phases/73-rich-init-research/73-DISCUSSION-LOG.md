# Phase 73: Rich Init Research - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-04-28
**Phase:** 73-rich-init-research
**Areas discussed:** Tech stack depth, Directory structure patterns, Governance file details, Pheromone suggestions expansion

---

## Tech stack depth

| Option | Description | Selected |
|--------|-------------|----------|
| Parse dependency lists | Parse package.json dependencies, go.mod requires, Cargo.toml deps, etc. Report actual package names, version ranges, and dependency count. | ✓ |
| File-presence only (current) | Detect the marker file and report the language/framework name. Fast, simple, zero parsing risk. | |
| Count + language names only | Detect language + count dependencies but don't parse individual package names. Middle ground. | |

**User's choice:** Parse dependency lists

| Option | Description | Selected |
|--------|-------------|----------|
| All deps (full list) | Include all dependencies including devDependencies and indirect deps. Most accurate but noisy. | ✓ |
| Production deps only | Only production/runtime dependencies. Cleaner signal. | |
| Direct deps listed, total counted | Parse everything but only surface top-level dependencies in output. | |

**User's choice:** All deps (full list)

---

## Directory structure patterns

| Option | Description | Selected |
|--------|-------------|----------|
| Pattern classification | Detect common patterns: monorepo, microservices, standard app, library, unknown. Heuristic-based. | ✓ |
| Directory listing only | Just list top-level directories and let the user/LLM interpret. | |
| Classification + convention score | Classify + add a 'structure score' measuring conventionality. | |

**User's choice:** Pattern classification

| Option | Description | Selected |
|--------|-------------|----------|
| Type + detection signals | Report the primary type plus signals that led to it (e.g., 'monorepo — detected: packages/'). | ✓ |
| Primary type only | Only report the primary classification. Simple and clear. | |

**User's choice:** Type + detection signals

---

## Governance file details

| Option | Description | Selected |
|--------|-------------|----------|
| Parse config files | Parse config files to extract actual settings: ESLint rules, Prettier options, test config, CI steps. | ✓ |
| Detection only (current) | Keep current behavior: report which tools are present by name. | |
| Extract key values only | Detect tool + read file for key values (e.g., 'extends: recommended'). | |

**User's choice:** Parse config files

| Option | Description | Selected |
|--------|-------------|----------|
| All categories | Parse all 5 governance categories: linters, formatters, test frameworks, CI configs, build tools. | ✓ |
| Linters + CI only | Focus on highest-signal categories. Others stay at detection level. | |
| Linters + formatters + tests | Most actionable for workers. CI and build stay at detection. | |

**User's choice:** All categories

---

## Pheromone suggestions expansion

| Option | Description | Selected |
|--------|-------------|----------|
| Expand to ~25 patterns | Add more patterns covering monorepo, API, database, security, container, documentation, dependency health. | ✓ |
| Keep current 10 | They cover the basics. Phase 74 will add runtime detection. | |
| Expand to ~40+ patterns | Most comprehensive but more maintenance. | |

**User's choice:** Expand to ~25 patterns

| Option | Description | Selected |
|--------|-------------|----------|
| Hard-coded Go functions | Each pattern is a function. Simple, testable, deterministic. | |
| Data-driven pattern registry | Patterns defined as data (YAML/JSON) with condition-check types. Easier to add without code changes. | |
| You decide | Pick whichever approach fits established patterns. | ✓ |

**User's choice:** You decide

| Option | Description | Selected |
|--------|-------------|----------|
| Built-in only | No user config. Users can use pheromone commands directly for custom patterns. | ✓ |
| Extensible (user can add) | Support user-defined patterns from config file. | |

**User's choice:** Built-in only

---

## Claude's Discretion

- Implementation approach for pheromone pattern registry (hard-coded vs data-driven)
- Exact dependency parsing depth per file format
- Governance parsing error handling for malformed configs
- Exact pheromone patterns to add
- Colony context summary formatting

## Deferred Ideas

None.
