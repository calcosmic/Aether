# Feature Research: Aether v2.5 Smart Init System

**Domain:** Multi-agent colony orchestration -- intelligent initialization with repo scanning, prompt generation, approval loops, and Queen file governance
**Researched:** 2026-03-27
**Confidence:** HIGH (based on direct codebase analysis of init.md, colonize.md, council.md, queen.sh, QUEEN.md template, and all related commands; web search was rate-limited but codebase provides comprehensive evidence)

---

## Context

The current `/ant:init` is purely mechanical: it takes a raw user goal string, creates COLONY_STATE.json from a template, initializes QUEEN.md as an empty wisdom document, and registers the repo. No research, no prompt refinement, no approval, no intelligence. Users forget to run `/ant:colonize` before planning. Re-running init warns about overwriting but offers no graceful update path.

The smart init milestone transforms `/ant:init` from a file-creation script into an intelligent first step: research the repo, generate a structured colony initialization prompt, show it for approval, and manage QUEEN.md as a living colony charter with intent, vision, and governance sections.

### Current Pain Points (from PROJECT.md user feedback)

1. `/ant:init` is mechanical -- takes raw goal string, sets up files, no research or intelligence
2. Users forget to run `/ant:colonize` before building
3. Subsequent inits reset everything (destructive -- warns but still overwrites)
4. QUEEN.md is initialized as empty wisdom doc with no project context
5. No structured colony prompt generated from natural language

### Existing Infrastructure That Smart Init Builds On

| Existing System | What It Does | How Smart Init Uses It |
|----------------|-------------|----------------------|
| `/ant:colonize` (257 lines) | 4 parallel scout survey producing 7 documents | Smart init needs a lightweight version of this, not the full deep survey |
| `/ant:council` (295 lines) | Multi-choice intent clarification via pheromone injection | Approval loop pattern is similar -- gather input, translate to structured output |
| `queen-init` subcommand | Creates QUEEN.md from template | Still needed for fresh colonies, but enhanced with charter sections |
| `queen-read` / `queen-promote` | QUEEN.md manipulation | Charter update must coexist with wisdom sections |
| `domain-detect` subcommand | Auto-detects domain tags from file presence | Init research can reuse this for tech stack detection |
| `session-verify-fresh` | Detects stale state files | Already used in init Step 2 for freshness checking |
| `suggest-analyze` / `suggest.sh` | Codebase pattern analysis | Research scan can reuse pattern detection logic |

---

## Table Stakes

Missing any of these = smart init still feels dumb.

| # | Feature | Why Expected | Complexity | Dependencies | Notes |
|---|---------|--------------|------------|--------------|-------|
| T1 | **Lightweight repo scan before initialization** | Users expect the system to know what it is initializing. Running init on a 500-file TypeScript repo should produce different context than running it on an empty directory. Currently init has zero awareness of the codebase. | MEDIUM | `init-research` bash subcommand, existing `domain-detect` | Fast surface scan: key config files, directory structure, git history, prior colony data. Target: <2 seconds. NOT a full colonize (which takes 30-60 seconds with 4 parallel agents). This is the single most important table-stakes feature -- without it, init remains blind. |
| T2 | **Auto-generate structured colony prompt from natural language** | The user types `/ant:init "Build a REST API"`. The system should expand this into a structured prompt with context, suggested focus areas, and charter fields. Currently the raw string becomes the colony goal with zero refinement. | MEDIUM | T1 (research data feeds prompt generation), `init-generate-prompt` bash subcommand | Bash + jq string assembly, not LLM generation. Deterministic and testable. Takes user goal + research JSON and produces structured prompt text with charter fields and suggested pheromones. |
| T3 | **User approval loop before proceeding** | Users must see and approve what the system is about to do before any files are created or modified. This is the core of "smart" -- the system proposes, the user disposes. Currently init creates files immediately with no preview. | LOW | T2 (need generated prompt to display) | LLM-mediated: display formatted prompt, wait for user confirmation, handle edits, loop until approved. No TUI library needed -- the LLM IS the UI (same pattern as build-wave plan confirmation). |
| T4 | **QUEEN.md charter sections on first init** | QUEEN.md should be a colony charter (intent, vision, governance, goals, architecture) plus wisdom -- not just an empty wisdom doc. First init should populate the charter sections. Currently QUEEN.md is initialized with placeholder text in 4 wisdom sections. | MEDIUM | `queen-charter-init` bash subcommand, updated QUEEN.md template | Adds Colony Charter section above existing wisdom sections. Charter fields (intent, vision, governance, architecture notes) are populated from the approved prompt. |
| T5 | **Subsequent inits update charter without resetting colony state** | Re-running `/ant:init` with a new goal should update the charter (intent, vision) but NEVER destroy accumulated wisdom, instincts, pheromones, or phase progress. Currently re-init warns about overwriting but still resets COLONY_STATE.json. | MEDIUM | T4, `queen-charter-update` bash subcommand | `_queen_charter_update` finds charter section by awk line-number and replaces specific fields. Wisdom sections completely untouched. COLONY_STATE.json goal updated, plan and phase progress preserved. |
| T6 | **Intelligent colonize suggestion** | When no recent territory survey exists or the codebase has changed significantly since the last survey, init should suggest running `/ant:colonize`. Users forget to run it, which means `/ant:plan` operates without survey context. | LOW | T1 (research detects stale/missing survey) | Simple boolean: if no `.aether/data/survey/` directory or survey files are stale (checked via `session-verify-fresh --command survey`), display a suggestion. Non-blocking -- does not auto-run colonize. |
| T7 | **Prior colony knowledge inheritance** | Already partially implemented in current init.md Step 2.6 (reads `completion-report.md`). Smart init should also inherit from QUEEN.md charter of prior colonies in chambers, not just completion reports. | LOW | T1 (research detects prior colonies), existing Step 2.6 logic | Read `.aether/QUEEN.md` charter section if it exists. Display inherited context in the approval prompt. This makes re-init feel like a continuation, not a reset. |

---

## Differentiators

Features that make Aether's init feel genuinely intelligent rather than just "prompt + approve."

| # | Feature | Value Proposition | Complexity | Dependencies | Notes |
|---|---------|-------------------|------------|--------------|-------|
| D1 | **Research-aware charter suggestions** | The system infers vision and governance from what it finds. A repo with 47 TypeScript files and a `jest.config.js` should auto-suggest "Follow existing TDD patterns" as governance. A greenfield project should default to "Establish testing patterns in Phase 1." | MEDIUM | T1 (research data), T2 (prompt generation) | Pattern: read test config -> suggest TDD governance. Read `.env.example` -> suggest env-var governance. Read existing `CONTRIBUTING.md` -> suggest its rules as governance. This is what makes init feel like it "understands" the project. |
| D2 | **Suggested pheromones from research** | Init generates suggested FOCUS and REDIRECT signals based on the research scan. User sees these in the approval prompt and can accept, reject, or modify them before they are injected. | LOW | T1 (research data), T2 (prompt generation) | Examples: research finds 12 `TODO` comments -> suggest FOCUS "technical debt cleanup". Research finds no test files -> suggest REDIRECT "skip tests". These are suggestions only -- user must approve. |
| D3 | **Colony complexity estimation** | Research produces a complexity estimate (small/medium/large) that informs the planning depth suggestion. A small repo gets `--fast` plan suggestion, a large repo gets `--deep`. | LOW | T1 (research data) | Based on: file count, directory depth, dependency count, git history length. Display in approval prompt: "Complexity: medium (47 files, 12 dependencies) -- suggest balanced planning depth." |
| D4 | **Chambers/tunnels context in approval prompt** | When prior archived colonies exist in `.aether/chambers/`, the approval prompt shows a summary: "2 prior colonies found. Last goal: 'Ship Aether v2'. Key instincts available for inheritance." | LOW | T1 (research detects chambers) | Read chamber names and goals from `.aether/chambers/*/COLONY_STATE.json` (if accessible). Display compact summary. Makes the user aware of institutional knowledge without overwhelming them. |
| D5 | **Goals section auto-populates from /ant:plan** | After `/ant:plan` generates phases, the charter Goals section updates automatically with the phase names and success criteria. The charter becomes a living document that evolves with the colony. | LOW | T4 (charter must exist), `/ant:plan` command | Add a step to plan.md that calls `queen-charter-update` with the generated phase goals. The charter's Goals section becomes a checklist that tracks plan progress. |
| D6 | **Architecture Notes from research** | Research extracts key architecture patterns (framework detected, module structure, entry points) and populates the Architecture Notes charter section. This gives colony-prime concrete architecture context from the first build. | LOW | T1 (research data), T4 (charter init) | Examples: "Express + TypeScript, modular structure under src/. Entry: src/index.ts. ORM: Prisma." This context flows into worker prompts via colony-prime. |

---

## Anti-Features

Features that seem good but create problems in the init context.

| Feature | Why Requested | Why Problematic | Alternative |
|---------|---------------|-----------------|-------------|
| Full deep survey on every init | Thorough codebase analysis like `/ant:colonize` | `/ant:colonize` takes 30-60 seconds with 4 parallel agents. Running this on every init would make init feel slow and wasteful, especially for re-inits where the codebase has not changed. Users want init to be fast. | Lightweight research scan (<2 seconds) for init. Suggest colonize when survey is stale or missing (T6). |
| Auto-run colonize from init | Users forget to run colonize, so just do it for them | Colonize spawns 4 parallel agents that explore the entire codebase. This is expensive (token-heavy, time-consuming) and may not be needed if the codebase is small or hasn't changed. Auto-running it removes user control over when to spend resources. | Intelligent suggestion (T6) -- display "No recent survey. Consider running /ant:colonize after init." User decides. |
| LLM-generated prompt instead of bash assembly | LLM could produce richer, more creative prompts | LLM generation is non-deterministic and untestable. Two inits with the same goal and repo could produce different prompts. Bash + jq assembly is deterministic, testable, and fast. The LLM's role is to display and facilitate the approval loop, not to generate the prompt. | Bash `init-generate-prompt` subcommand assembles from research JSON. LLM handles display + approval. |
| Multiple approval rounds with different aspects | Approve charter separately from pheromones separately from plan | Increases user friction. Each approval round is a context switch. Users want to get to building fast. The approval loop should be ONE pass: see everything, edit anything, approve once. | Single unified approval prompt showing all aspects (charter + pheromones + context). One approval. |
| Interactive TUI for approval (inquirer.js, etc.) | Rich terminal UI with checkboxes, dropdowns | Claude Code and OpenCode are conversational AI tools. The user is already in a chat interface. Adding a TUI library creates a dependency, platform compatibility issues, and a fundamentally different interaction model from the rest of Aether. | LLM-mediated approval (display Markdown, wait for text response). The LLM IS the UI. |
| QUEEN.md as separate charter + wisdom files | Keep charter concerns separate from wisdom concerns | QUEEN.md is already the single source of truth loaded by colony-prime. Splitting into two files means colony-prime must load two files, and the user must maintain two files. The METADATA JSON block already supports arbitrary fields. | Single QUEEN.md with charter section above wisdom sections. One file, one source of truth. |
| Auto-reset COLONY_STATE.json on re-init | Clean slate for new colony goal | Destructive and surprising. Users expect re-init to update the goal, not destroy their phase progress, instincts, and learnings. The current init warns about this but still does it. | T5: charter update preserves state. Only goal changes. Phase progress, instincts, learnings, pheromones all preserved. |
| Init generates the full plan | Skip `/ant:plan`, generate phases during init | Plan generation requires a research loop (multiple scout + planner iterations). This makes init slow and conflates two distinct concerns: "what do we want" (init) vs "how do we get there" (plan). Users may want to run `/ant:focus` and `/ant:redirect` between init and plan. | Init sets up the colony with a charter. Plan is a separate step. Init suggests `/ant:plan` as the next step. |
| Prompt injection via user goal | "Ignore previous instructions and..." | The init command takes `$ARGUMENTS` as a raw goal string. If the user is malicious or the goal comes from an automated system, prompt injection is possible. | Content sanitization already exists in pheromone system (XML tags rejected, angle brackets escaped). Apply same sanitization to goal text before storing in COLONY_STATE.json and QUEEN.md. |

---

## Feature Dependencies

```
[T1: Lightweight repo scan]
    └──enables──> [T2: Auto-generate structured prompt]
                    └──enables──> [T3: User approval loop]
                                    └──enables──> [T4: QUEEN.md charter sections]
                                                    └──enables──> [T5: Subsequent inits update charter]
                                                    └──enables──> [D5: Goals auto-populate from plan]
                                                    └──enables──> [D6: Architecture Notes from research]

[T1: Lightweight repo scan]
    └──enables──> [T6: Intelligent colonize suggestion]
    └──enables──> [D4: Chambers/tunnels context]
    └──enables──> [D3: Colony complexity estimation]
    └──enables──> [D1: Research-aware charter suggestions]
    └──enables──> [D2: Suggested pheromones from research]

[T7: Prior colony knowledge inheritance]
    └──requires──> [T1: Research detects prior colonies]
    └──enhances──> [T2: Prompt generation includes inherited context]
```

### Dependency Notes

- **T1 is the foundation.** Without repo scanning, every other feature is operating blind. The research scan produces the JSON data that feeds prompt generation, charter suggestions, pheromone suggestions, and complexity estimation.
- **T2 and T3 form the "smart" core.** Research (T1) feeds prompt generation (T2), which feeds the approval loop (T3). This is the minimum viable intelligence chain.
- **T4 and T5 are the charter management pair.** T4 creates the charter on first init. T5 updates it on subsequent inits. They must preserve wisdom sections.
- **D1-D6 are enhancements that use T1 data.** They do not require the full T1-T5 chain to work, but they produce their best results when the full chain is in place.
- **T7 is independent of T4/T5.** It reads prior colony data (from completion-report.md or QUEEN.md) and injects it into the prompt. It does not modify QUEEN.md itself.
- **D5 requires plan.md changes.** The charter Goals section auto-populates from plan output. This is a cross-command dependency (init -> plan).

---

## MVP Definition

### Launch With (Phase 1 of milestone)

Minimum to make init feel intelligent.

- [ ] **T1** -- Lightweight repo scan (init-research bash subcommand, <2 seconds)
- [ ] **T2** -- Auto-generate structured colony prompt (init-generate-prompt bash subcommand)
- [ ] **T3** -- User approval loop (LLM-mediated display + confirm/edit)
- [ ] **T4** -- QUEEN.md charter sections on first init (queen-charter-init)
- [ ] **T5** -- Subsequent inits update charter without resetting (queen-charter-update)

Rationale: T1-T5 form the complete intelligence chain: scan -> generate -> approve -> charter. After these, a user running `/ant:init "Build a REST API"` sees a structured proposal based on their repo, can edit it, and gets a populated QUEEN.md charter. Re-running init updates the charter without destroying accumulated wisdom.

### Add After Validation (Phase 2 of milestone)

- [ ] **T6** -- Intelligent colonize suggestion (detect stale/missing survey)
- [ ] **T7** -- Prior colony knowledge inheritance (chambers/tunnels context)
- [ ] **D2** -- Suggested pheromones from research (FOCUS/REDIRECT suggestions in approval)
- [ ] **D3** -- Colony complexity estimation (informs planning depth suggestion)

### Future Consideration (Phase 3 or defer)

- [ ] **D1** -- Research-aware charter suggestions (infer governance from codebase patterns)
- [ ] **D4** -- Chambers/tunnels context in approval prompt (read archived colony summaries)
- [ ] **D5** -- Goals section auto-populates from /ant:plan (cross-command integration)
- [ ] **D6** -- Architecture Notes from research (populate charter architecture section)

---

## Feature Prioritization Matrix

| Feature | User Value | Implementation Cost | Priority |
|---------|------------|---------------------|----------|
| T1: Lightweight repo scan | HIGH -- init knows what it is initializing | MEDIUM -- new bash subcommand, ~200 lines | P1 |
| T2: Auto-generate structured prompt | HIGH -- user sees intelligent proposal | MEDIUM -- bash + jq assembly, ~150 lines | P1 |
| T3: User approval loop | HIGH -- user has control before files change | LOW -- LLM-mediated, no new code | P1 |
| T4: QUEEN.md charter sections | HIGH -- QUEEN.md becomes meaningful on first init | MEDIUM -- queen.sh functions + template, ~200 lines | P1 |
| T5: Subsequent inits update charter | HIGH -- re-init is safe, not destructive | MEDIUM -- queen-charter-update, ~150 lines | P1 |
| T6: Intelligent colonize suggestion | MEDIUM -- fixes "forgot to colonize" problem | LOW -- stale detection + display message | P2 |
| D2: Suggested pheromones | MEDIUM -- saves user from manual /ant:focus step | LOW -- derive from research patterns | P2 |
| T7: Prior colony inheritance | MEDIUM -- continuity across colony lifecycles | LOW -- read existing data, display in prompt | P2 |
| D3: Complexity estimation | MEDIUM -- smarter planning depth suggestion | LOW -- heuristic from file count + deps | P2 |
| D1: Research-aware charter suggestions | MEDIUM -- init infers governance from codebase | MEDIUM -- pattern matching on research data | P3 |
| D4: Chambers context | LOW -- nice awareness of history | LOW -- read chamber names | P3 |
| D5: Goals auto-populate from plan | MEDIUM -- charter evolves with colony | MEDIUM -- plan.md integration | P3 |
| D6: Architecture Notes from research | LOW -- useful context but not essential | LOW -- extract from research data | P3 |

**Priority key:**
- P1: Must have for launch -- the intelligence chain
- P2: Should have, adds significant value
- P3: Nice to have, polish

---

## Competitor / Analog Feature Analysis

| Feature | Cursor | Windsurf | Copilot Workspace | Aider | Aether (planned) |
|---------|--------|----------|-------------------|-------|------------------|
| Project setup / init | Manual `.cursorrules` | Manual `.windsurfrules` | Auto-scans repo on workspace open | Manual `CONVENTIONS.md` | Auto-research + generate + approve |
| Repo awareness on init | No (reads files as context but no init step) | No (Cascade reads files on demand) | Yes (indexes repo structure) | No (reads files on demand) | Yes (lightweight scan <2s) |
| Structured prompt generation | No (user writes rules manually) | No (user writes rules or asks Cascade) | No (task description only) | No (conventions are free-form) | Yes (goal + research -> structured prompt) |
| User approval before setup | No (instant apply) | No (instant apply) | No (instant apply) | No (instant apply) | Yes (display + approve/edit loop) |
| Governance document evolution | No (static file) | No (static file) | No (no governance concept) | No (static file) | Yes (charter updates on re-init, preserves wisdom) |
| Cross-session knowledge | No (each session starts fresh) | No (each session starts fresh) | No (each workspace starts fresh) | No (each session starts fresh) | Yes (prior colony knowledge inheritance) |
| Colonize suggestion | N/A | N/A | N/A | N/A | Yes (detect stale survey, suggest /ant:colonize) |

**Key insight:** No existing AI coding tool has a smart init flow that combines repo scanning, structured prompt generation, user approval, and evolving governance. Most tools rely on static instruction files that the user must create and maintain manually. Aether's approach of making init intelligent and QUEEN.md a living charter is genuinely differentiated.

**Confidence:** MEDIUM on competitor analysis -- web search was rate-limited, so competitor features are based on training data knowledge (pre-2025) rather than current documentation. The Aether analysis is HIGH confidence (direct codebase inspection).

---

## What the Smart Init Experience Should Look Like

### Fresh Colony (Empty or New Repo)

**Current experience:**
```
$ /ant:init "Build a REST API with authentication"

[388 lines of file creation steps execute silently]

AETHER COLONY
Queen has set the colony's intention
   "Build a REST API with authentication"
Colony Status: READY
Next Up: /ant:plan
```

**Smart init experience:**
```
$ /ant:init "Build a REST API with authentication"

Scanning repository...

PROPOSED COLONY INITIALIZATION
================================================================

Goal: Build a REST API with authentication

Repository Context:
  Language: TypeScript
  Framework: None detected (greenfield)
  Files: 3 (README, package.json, tsconfig.json)
  Tests: None detected
  Complexity: Small

Colony Charter:
  Intent: Build a REST API with authentication
  Vision: A production-ready REST API with JWT authentication,
          rate limiting, input validation, and comprehensive tests
  Governance:
    - Establish testing patterns in Phase 1
    - Follow existing TypeScript strict mode
    - All endpoints must have request validation
  Architecture Notes: Greenfield project. Suggested stack: Express + TypeScript.

Suggested Guidance:
  FOCUS: "authentication and authorization"
  FOCUS: "test coverage"
  REDIRECT: "skip validation"

Planning Suggestion: balanced (4-6 iterations, 90% confidence target)

================================================================

Review the proposed setup above.
- Edit any section by describing what to change
- Type "approved" to proceed
- Type "cancel" to abort

> [user types "approved"]

Colony initialized.
  QUEEN.md: Colony Charter created with 4 sections
  State persisted: .aether/data/COLONY_STATE.json
  Suggested pheromones: 2 FOCUS signals ready to inject

  Next: /ant:colonize (recommended for deeper analysis)
        /ant:plan    (generate execution plan)
        /ant:focus   (set additional focus areas)
```

### Re-Init on Active Colony

**Current experience:**
```
$ /ant:init "Add WebSocket support to the API"

Colony already initialized with goal: "Build a REST API with authentication"
State freshness: fresh
Proceeding with new goal: "Add WebSocket support to the API"

[ALL PREVIOUS STATE IS OVERWRITTEN]
```

**Smart init experience:**
```
$ /ant:init "Add WebSocket support to the API"

Scanning repository...

PROPOSED CHARTER UPDATE
================================================================

Current Colony: "Build a REST API with authentication"
  Phase: 3 of 5 (in progress)
  Instincts: 4 | Learnings: 7 | Survey: complete

New Goal: Add WebSocket support to the API

Charter Changes:
  Intent: Build a REST API with WebSocket support for real-time features
  Vision: [UPDATED] A production-ready REST API with JWT auth,
          WebSocket connections, and comprehensive tests
  Governance: [PRESERVED] No changes
  Architecture Notes: [UPDATED] Express + TypeScript + ws library.
          Existing REST endpoints preserved.

What happens:
  - Colony goal updated (does NOT reset phase progress)
  - Charter sections updated (does NOT touch wisdom/instincts)
  - Phases 1-3 preserved, Phase 4+ may need re-planning

================================================================

Review the proposed changes.
- Edit any section
- Type "approved" to update charter
- Type "cancel" to keep current goal

> [user types "approved"]

Charter updated. Phase progress preserved.
Consider running /ant:plan to adjust remaining phases for new goal.

  Next: /ant:plan    (re-plan remaining phases)
        /ant:build 4 (continue with current plan)
```

### Init with Prior Colony Chambers

```
$ /ant:init "Refactor the auth module"

Scanning repository...

PROPOSED COLONY INITIALIZATION
================================================================

Goal: Refactor the auth module

Repository Context:
  Language: TypeScript
  Framework: Express
  Files: 47 | Tests: Yes (jest)
  Complexity: Medium

Prior Colony Knowledge:
  2 archived colonies found
  - "Ship Aether v2" (2026-03-21) -- 5 instincts available
  - "Implement skills layer" (2026-03-22) -- 3 instincts available

  Inherited instinct: [0.85] testing: always run full test suite
    after module extraction
  Inherited learning: JWT token validation must check expiry
    before verifying signature

Colony Charter:
  Intent: Refactor the auth module
  Vision: Cleaner auth architecture with separation of concerns,
          leveraging existing patterns from prior colonies
  Governance:
    - Follow existing codebase conventions (from survey)
    - Maintain backward compatibility
    - All existing tests must continue to pass
  Architecture Notes: Express + TypeScript, JWT auth module at
    src/auth/. Consider extracting to src/middleware/.

================================================================
```

---

## Sources

### Primary (HIGH confidence)
- Direct codebase analysis of `.claude/commands/ant/init.md` (388 lines, current mechanical init flow)
- Direct codebase analysis of `.claude/commands/ant/colonize.md` (257 lines, deep survey with 4 parallel agents)
- Direct codebase analysis of `.claude/commands/ant/council.md` (295 lines, intent clarification via pheromones)
- Direct codebase analysis of `.aether/utils/queen.sh` (1,242 lines, all QUEEN.md manipulation)
- Direct codebase analysis of `.aether/templates/QUEEN.md.template` (v2 format, 4 wisdom sections)
- Direct codebase analysis of `.aether/QUEEN.md` (actual data: 1 instinct, 6 patterns, 1 learning from ~20 phases)
- Direct codebase analysis of `.aether/chambers/` directory (2 archived colonies)
- `.planning/PROJECT.md` (v2.5 milestone requirements and user feedback)
- `.planning/research/STACK-SMART-INIT.md` (complementary stack research for this milestone)

### Secondary (MEDIUM confidence)
- User testing feedback from PROJECT.md: "init is purely mechanical", "users forget to run /ant:colonize", "subsequent inits reset everything"
- Existing completion-report.md inheritance logic in init.md Step 2.6 (partial prior colony knowledge)
- Existing `domain-detect` subcommand for auto-detecting tech stack
- Existing `session-verify-fresh` subcommand for staleness detection

### Tertiary (LOW confidence)
- Competitor feature analysis (Cursor, Windsurf, Copilot Workspace, Aider) -- web search was rate-limited; competitor features based on training data knowledge (pre-2025) rather than current docs
- Whether research-aware charter suggestions (D1) produce high-quality governance text (untested hypothesis)
- Whether the approval loop UX feels natural or cumbersome (needs user testing)

---
*Feature research for: Aether v2.5 Smart Init System*
*Researched: 2026-03-27*
