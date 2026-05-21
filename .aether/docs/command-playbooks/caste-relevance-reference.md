# Caste Relevance Reference

> Source of truth: `cmd/caste_relevance.go` and `cmd/queen_spawn_budget.go`

## Caste Registry

| Caste | Base Score | Keywords (partial) |
|-------|-----------|-------------------|
| builder | 20 | implement, build, create, add, write, fix, code, deploy |
| watcher | 20 | verify, test, validate, check, review, quality |
| scout | 20 | research, investigate, survey, analyze, document, explore |
| route_setter | 15 | plan, route, decompose, structure, organize |
| architect | 20 | design, schema, architecture, interface, boundary, evaluate |
| oracle | 20 | research, spike, evaluate, unknown, deep dive (discovery only) |
| chaos | 15 | resilience, failure, robustness, crash, stress test |
| archaeologist | 20 | legacy, migration, modernize, rewrite, history, refactor old |
| gatekeeper | 20 | auth, crypto, security, token, secrets, permissions, compliance |
| auditor | 20 | compliance, audit, production, release, quality gate (production only) |
| probe | 15 | test coverage, edge case, validation, missing tests, coverage gap |
| measurer | 20 | performance, optimize, latency, scale, benchmark, memory, cpu |
| ambassador | 20 | api, sdk, oauth, external service, integration, webhook, third-party |
| tracker | 20 | bug, fix, regression, investigate failure, root cause, issue |
| weaver | 20 | refactor, cleanup, modernize, extract, simplify, restructure |
| keeper | 10 | knowledge, pattern, convention, standard, document, preserve, wisdom |
| chronicler | 15 | documentation, docs, guide, readme, changelog, manual |
| includer | 15 | accessibility, a11y, wcag, screen reader, aria, inclusive |
| surveyor-provisions | 10 | dependency, dependencies, provisions, external, stack, package |
| surveyor-nest | 10 | architecture, structure, layout, map, chamber, directory |
| surveyor-disciplines | 10 | convention, discipline, testing, pattern, practice, standard |
| surveyor-pathogens | 10 | pathogen, debt, fragile, risk, bug, health, failure |
| medic | 10 | health, diagnose, repair, heal, fix state |
| fixer | 10 | auto-fix, repair, patch, remediate, self-heal |
| porter | 10 | deploy, deliver, ship, publish, release, package |
| sage | 10 | wisdom, synthesize, learn, pattern, retrospective |

Scoring: `baseScore + keywordMatches*10 + conditionBonus`. Auto-include rules in `applySpecialRules` set score to 100 for builder (implementation tasks), architect (high risk), oracle (discovery), gatekeeper (high risk), and auditor (production).

## Spawn Thresholds

| Flow Type | Threshold | Notes |
|-----------|-----------|-------|
| build, continue | 30 | Lower bar; Queen filters later via budget |
| continue (heavy depth) | 25 | Even lower for heavy verification |
| plan | 40 | Higher bar; planning is selective |
| colonize, swarm | 35 | Territory and bug work are focused |
| seal | 50 | Highest bar; only strongly relevant castes |

Source: `spawnThreshold`

## Always-Required Castes

### Build (`queenBuildSafetyRequiredCastes` + `isAlwaysRequired`)
- **All builds**: `probe`, `watcher`
- **Non-discovery builds**: + `builder`
- **High-risk / production / review-required builds**: + `auditor`, `gatekeeper`

### Continue
| Depth | Required Castes |
|-------|----------------|
| light | `watcher` |
| standard | `watcher`, `probe` |
| heavy | `watcher`, `gatekeeper`, `auditor`, `probe` |

### Plan
- Always: `scout`, `route_setter`

### Colonize
- Always: `surveyor-provisions`, `surveyor-nest`, `surveyor-disciplines`, `surveyor-pathogens`

### Swarm
- Always: `tracker`, `scout`, `archaeologist`, `builder`, `watcher`
- High-risk phases: + `gatekeeper`

### Seal
| Depth | Required Castes |
|-------|----------------|
| light | (none) |
| standard | `auditor`, `probe` |
| heavy | `gatekeeper`, `auditor`, `probe` |

Source: `isAlwaysRequired`

## Spawn Budget (Max Workers)

Source: `queenMaxWorkersForBudget`

### Build
| Condition | Max Workers | Reason |
|-----------|-------------|--------|
| maintenance + low risk + docs/maintenance look | 4 | low-risk documentation or maintenance build |
| high risk OR production mode | 8 | high-risk or production build |
| medium risk | 6 | medium-risk build |
| discovery mode | 5 | discovery build |
| default | 6 | standard build |

### Continue
| Depth | Max Workers | Reason |
|-------|-------------|--------|
| light | 3 | light verification |
| heavy | 6 | heavy verification |
| standard | 4 | standard verification |

### Plan
| Condition | Max Workers | Reason |
|-----------|-------------|--------|
| high risk | 6 | high-risk planning |
| default | 4 | standard planning |

### Colonize
- Always: 4 (territory survey)

### Swarm
- Always: 5 (focused swarm)

### Seal
| Condition | Max Workers | Reason |
|-----------|-------------|--------|
| heavy depth OR high risk | 5 | heavy seal review |
| default | 4 | standard seal review |

## Suppression Rules

Source: `isCasteSuppressed`

| Flow | Condition | Suppressed Castes |
|------|-----------|-------------------|
| build | discovery mode | `builder`, `weaver`, `fixer`, `porter` |
| seal | light depth | `gatekeeper`, `auditor`, `probe` |
| continue | (always) | `builder`, `weaver`, `tracker`, `archaeologist`, `ambassador` |
| seal | (always) | `builder`, `weaver`, `tracker`, `archaeologist`, `ambassador` |

## Flow Allowlists

Source: `casteAllowedForFlow`

| Flow | Allowed Castes |
|------|---------------|
| build | All except `surveyor-*` |
| continue | `watcher`, `gatekeeper`, `auditor`, `probe`, `measurer`, `chaos`, `includer`, `keeper`, `sage`, `medic`, `fixer` |
| plan | `scout`, `route_setter`, `architect`, `oracle`, `keeper`, `chronicler`, `includer`, `gatekeeper` |
| colonize | `surveyor-*` only |
| swarm | `tracker`, `scout`, `archaeologist`, `builder`, `watcher`, `gatekeeper`, `probe`, `weaver`, `medic`, `fixer` |
| seal | `gatekeeper`, `auditor`, `probe`, `porter`, `chronicler`, `keeper`, `sage`, `measurer`, `includer` |

## Examples

| Phase Name | Mode | Flow | Castes Spawned |
|------------|------|------|---------------|
| "Settings UI panel" | prototype | build | builder, watcher, probe |
| "Auth token rotation" | production | build | builder, watcher, gatekeeper, probe, architect, auditor |
| "Database migration" | production | build | builder, watcher, auditor, architect, probe |
| "Performance optimization" | prototype | build | builder, watcher, measurer, probe |
| "Refactor legacy parser" | maintenance | build | builder, watcher, weaver, archaeologist, probe |
| "Discovery spike on vector DB" | discovery | build | oracle, scout, architect (builder suppressed) |
| "Security hardening" | production | build | builder, watcher, probe, gatekeeper, auditor, architect |
| "Release candidate packaging" | production | build | builder, watcher, probe, gatekeeper, auditor |

## Key Functions

- `casteRelevanceScore` — computes 0-100 relevance score for a caste against a phase
- `queenOrchestrate` — main dispatch function; returns selected castes after budget
- `queenCandidateDispatches` — filters by threshold, allowlist, suppression, and always-required
- `applyQueenSpawnBudget` — enforces max-workers cap, sorts required first
- `queenSpawnBudgetForPhase` — assembles budget from flow + phase + state
- `queenMaxWorkersForBudget` — returns max workers and human-readable reason
- `queenBuildSafetyRequiredCastes` — build-specific required castes based on mode and risk
- `queenBuildSafetyReviewRequired` — triggers review for production mode or security/release keywords
