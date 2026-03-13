# Aether: Oracle Deep Research Engine

## What This Is

Aether is a self-managing development assistant that uses ant colony metaphor to orchestrate AI workers across coding sessions. It has 36 commands, 22 agents, and ~10,000 lines of shell infrastructure. The colony is fully self-improving with complete learning pipelines (v1.0). The oracle (`/ant:oracle`) is a deep research engine that produces thorough, source-verified, actionable research using a structured RALF-loop pattern with gap-driven iteration, convergence detection, and configurable strategy (v1.1).

## Core Value

The oracle produces research you can act on — verified against official sources, iteratively deepened, and structured for the specific topic. It runs autonomously for as long as needed, drilling deeper each iteration rather than just covering more surface area.

## Current State

**Shipped:** v1.1 Oracle Deep Research (2026-03-13)
**Oracle core:** 1,188 LOC (oracle.sh + oracle.md)
**Oracle tests:** 168 tests (87 Ava + 81 bash)
**Total tests:** 512+ passing
**Architecture:** v4.0 (runtime/ eliminated, direct packaging)

## Requirements

### Validated

- Colony command infrastructure (36 commands, all functional)
- Pheromone signal system (FOCUS/REDIRECT/FEEDBACK emit and display)
- colony-prime injection (pheromones reach builders via prompt_section)
- Midden failure tracking (recent failures shown to builders)
- Graveyard file cautions (unstable files flagged to builders)
- Survey territory intelligence (codebase patterns fed to builders)
- State persistence across sessions (COLONY_STATE.json, CONTEXT.md)
- Memory-capture pipeline (learning-observe, observation counting)
- Instinct infrastructure (instinct-create, instinct-read exist)
- QUEEN.md infrastructure (queen-init, queen-read, queen-promote exist)
- Suggest-analyze/approve pipeline (pheromone suggestions exist)
- Phase learnings auto-inject into future builder prompts -- v1.0
- Key decisions auto-convert to FEEDBACK pheromones -- v1.0
- Recurring error patterns auto-emit REDIRECT pheromones -- v1.0
- Learning observations auto-promote to QUEEN.md when thresholds met -- v1.0
- Escalated flags inject as warnings into next phase builders -- v1.0
- colony-prime reads CONTEXT.md decisions for builder injection -- v1.0
- instinct-create called during continue flow with confidence >= 0.7 -- v1.0
- instinct-read results included in colony-prime output (domain-grouped) -- v1.0
- queen-promote called during seal and continue flows -- v1.0
- Success criteria patterns create instincts on recurrence -- v1.0
- Oracle uses structured state files (state.json, plan.json, gaps.md, synthesis.md) to bridge context -- v1.1
- Gap-driven iteration targeting highest-priority knowledge gaps -- v1.1
- Phase-aware prompts (survey/investigate/synthesize/verify) change behavior by lifecycle stage -- v1.1
- Multi-signal convergence detection (gap resolution, novelty, coverage) -- v1.1
- Topic decomposition into 3-8 tracked sub-questions with confidence scoring -- v1.1
- Per-question confidence scoring drives research priority -- v1.1
- Research plan visible as research-plan.md -- v1.1
- Diminishing returns detection triggers strategy changes -- v1.1
- Structured synthesis report with executive summary on every exit path -- v1.1
- Output structure adapts to research topic (5 template types) -- v1.1
- Source tracking with inline citations and trust scoring -- v1.1
- Single-source claims flagged as low confidence -- v1.1
- Sources collected in dedicated section with inline citations -- v1.1
- Mid-session steering via pheromone signals between iterations -- v1.1
- Configurable search strategy (breadth-first, depth-first, adaptive) -- v1.1
- Configurable focus areas via wizard and pheromone emission -- v1.1
- High-confidence findings promote to colony instincts/learnings -- v1.1
- Pre-built research strategy templates (tech-eval, architecture-review, bug-investigation, best-practices) -- v1.1

### Active

(None — define next milestone with `/gsd:new-milestone`)

### Out of Scope

- Cross-colony wisdom sharing -- solve single-colony learning first
- Model routing verification -- separate concern
- XML migration -- do gradually as files are touched
- Knowledge graph construction during research -- future
- Parallel sub-question research (spawn multiple AI instances) -- future
- Source credibility scoring (domain authority, recency) -- future
- Real-time web scraping / browser automation -- WebSearch/WebFetch handles 95%
- Academic database integration -- dev tool, not academic tool
- Multi-user collaboration -- single-developer CLI
- Autonomous scope expansion -- runaway scope is #1 failure mode
- Persistent cross-session research memory -- colony integration captures durable knowledge

## Context

Aether v1.1.0 with oracle deep research shipped. 512+ tests passing. The oracle is a complete RALF-loop research engine with structured state, gap-driven iteration, convergence detection, source verification, steering integration, and colony knowledge promotion. 5 research templates (tech-eval, architecture-review, bug-investigation, best-practices, custom) produce topic-appropriate output.

## Constraints

- **Backward compatible** -- existing colonies and commands must not break
- **Must work in Claude Code** -- all output via unicode/emoji, no ANSI
- **Bash 3.2 compatible** -- macOS ships bash 3.2
- **Test coverage** -- new behavior needs tests
- **Works on any repo** -- oracle research is for user projects, not just Aether
- **Colony branding** -- ant colony metaphor throughout

## Key Decisions

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| Connect, don't add | System has all pieces, just disconnected | Good -- v1.0 |
| colony-prime is the integration point | Single function that assembles all context | Good -- v1.0 |
| Upgrade existing oracle, not new system | Oracle infrastructure exists, just needs depth | Good -- v1.1 |
| Ralph loop rebranded as RALF | Colony theme, research-adapted | Good -- v1.1 |
| Research quality over research speed | Each iteration must add real depth | Good -- v1.1 |
| Structured state over flat append | state.json + plan.json replace progress.md | Good -- v1.1 |
| Phase-aware prompts | Survey/investigate/synthesize/verify lifecycle | Good -- v1.1 |
| Structural convergence metrics | Gap resolution + coverage + novelty, not self-assessed confidence | Good -- v1.1 |
| Source tracking via prompt + schema | AI records sources, oracle.sh counts structurally | Good -- v1.1 |
| Strategy as emphasis modifier | Phase system retains structural control | Good -- v1.1 |
| Wizard calls colony APIs directly | Avoids sourcing oracle.sh main loop | Good -- v1.1 |
| Templates drive both questions and output | Same template shapes decomposition and synthesis | Good -- v1.1 |

---
*Last updated: 2026-03-13 after v1.1 milestone*
