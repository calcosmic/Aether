# Aether v2.0.0 System Integrity Audit — Final Report

**Oracle Research | 11 iterations | Converged at 80% confidence**
**Date:** 2026-03-22 | **Scope:** Codebase static analysis

---

## 1. Executive Summary

Aether v2.0.0 is a multi-agent colony system with 43 slash commands, 22 agent definitions, and an 11,221-line utility core (`aether-utils.sh`). The system orchestrates AI workers through a pheromone-signal-driven architecture where commands chain together via a centralized state machine (`print-next-up`), workers receive behavioral signals through prompt injection (`colony-prime`), and state is tracked in a single JSON file (`COLONY_STATE.json`).

**Key finding:** The system is architecturally coherent and functionally complete, with strong command chaining (30/43 commands use centralized routing [S2, S52]), verified v1.1.11 feature preservation (all 6 features traced, 542 tests passing [S20-S26]), and rich visual elements (333 visual matches across commands [S10, S48]). The v1-to-v2 refactoring introduced no regressions [S11, S18-S24]. However, the audit reveals a consistent pattern: **the system is optimized for the autonomous happy path and underserves edge cases, error states, and user visibility of internal operations.** This manifests as: routing that breaks after seal [S56], pheromone operations invisible during continue [S43], users unable to shape planning [S60], errors displayed as plain text while successes get rich formatting [S48], and safety mechanisms applied to only ~30% of write paths [S27, S29-S33].

**Critical risks:** One confirmed bug (post-seal state routing [S56]), one documentation inaccuracy (session-verify-fresh coverage claim [S45]), and a systemic architectural trait where safety mechanisms (atomic writes, session freshness, visual consistency, pheromone protocols) exist but cover only a fraction of the paths that need them. The most significant design tension is between the autonomous-first philosophy and the stated user persona of a non-technical founder who "tests like a user" — the system plans and builds without user input, then asks for validation after the fact.

**Evidence ceiling:** This audit has reached ~80% confidence through static code analysis. The remaining ~20% requires runtime testing (concurrent write behavior, crash recovery, non-TTY rendering) and design intent clarification (whether autonomous-first is deliberate, whether per-command visual theming is intentional).

---

## 2. Component Map

### High confidence (80%+)

| Component | Responsibility | Key Files | Boundary |
|-----------|---------------|-----------|----------|
| **State Machine** | Centralized command routing via 4-state model (IDLE/READY/EXECUTING/PLANNING) | `aether-utils.sh:201-235` [S2] | Used by 30/43 commands; 11 others have explicit routing; 2 dead-ends [S52, S58, S59] |
| **Colony Prime** | Worker context assembly — combines QUEEN.md, pheromones, phase learnings, hive wisdom into budget-capped prompt_section | `aether-utils.sh:7855-8510` [S13] | 8-section assembly with priority-ordered trimming, 8K char default budget [S35] |
| **Pheromone System** | User-colony signal communication (FOCUS/REDIRECT/FEEDBACK) with decay, dedup, and prompt injection | `aether-utils.sh:7177-7853` [S39, S40] | Signals sorted by priority, capped at 8, injected into 3 agents with formal protocol [S36-S38] |
| **Build Pipeline** | 5-playbook orchestration: prep → context → wave → verify → complete | `.aether/docs/command-playbooks/build-*.md` [S3, S7, S8, S41, S42] | Spawns parallel workers per wave, refreshes colony-prime between waves [S41] |
| **Continue Pipeline** | 4-playbook verification: verify → gates → advance → finalize | `.aether/docs/command-playbooks/continue-*.md` [S4, S55, S62, S63] | Mandatory runtime verification gate, optional quality/security gates [S62] |
| **Autopilot** | Automated build-verify-advance loop with 10 pause conditions | `.claude/commands/ant/run.md` [S12, S53] | Chains all 9 playbooks; converts interactive gate to auto-PAUSE [S61] |

### Medium confidence (50-79%)

| Component | Responsibility | Key Files | Notes |
|-----------|---------------|-----------|-------|
| **Session Recovery** | Two-tier: resume (lightweight, CONTEXT.md) and resume-colony (full, HANDOFF.md) | `resume.md` [S5], `resume-colony.md` [S15] | No fallback for missing HANDOFF.md [S44]; staleness computed but ignored [S45] |
| **Atomic Write** | Safe JSON mutation via temp-file-and-rename pattern | `.aether/utils/atomic-write.sh` [S27] | Covers only 4 of 10+ write paths; backup exists but never auto-restored [S27, S29-S33] |
| **File Locking** | noclobber-based locking with PID tracking and stale detection | `.aether/utils/file-lock.sh` [S28] | Single-lock-per-process limitation; basename-only keys (single source) |
| **Hive Brain** | Cross-colony wisdom with domain-scoped retrieval and LRU eviction | `.aether/utils/hive.sh` [S19] | 200-entry cap; multi-repo confidence boosting [CLAUDE.md] |

---

## 3. Dependency Analysis

### Internal Dependencies

**COLONY_STATE.json is the single point of failure** [S2, S13]. Every stateful command reads or writes this file. The state machine (`print-next-up`), colony-prime context assembly, session recovery, and build/continue pipelines all depend on it. Corruption propagates to every downstream operation.

**Coupling assessment:**
- **Tight coupling (high risk):** `aether-utils.sh` ↔ `COLONY_STATE.json` — 125 subcommands in one file mutating one state file [S2, S13]
- **Loose coupling (healthy):** Slash commands ↔ playbooks — clean delegation pattern [S3, S4, S54, S55]
- **Disconnected (problematic):** `state` field ↔ `milestone` field in COLONY_STATE.json — no coordination; seal updates one but not the other [S56]. CONTEXT.md ↔ HANDOFF.md — parallel systems with no cross-referencing [S5, S14, S15]. Pheromone injection ↔ pheromone consumption — colony-prime injects to all agents but only 3/22 have formal handling [S13, S36-S38].

### External Dependencies

- **Claude Code runtime** — slash commands and agent definitions depend on Claude Code's tool calling (AskUserQuestion, Read/Write, Agent spawning)
- **jq** — JSON processing throughout aether-utils.sh; graceful degradation when missing [S34]
- **bash 4+** — associative arrays, process substitution
- **npm** — packaging and distribution (`npm install -g .`)
- **tmux** — optional, for `/ant:watch` [S59]

### LLM/Bash Coordination Gap

Slash commands mutate COLONY_STATE.json through Claude's Read/Write tools. Bash subcommands mutate the same file through `jq` + `echo` or `atomic_write`. **There is no locking coordination between these two write paths** [S1, S3, S4, S28]. During parallel builds with multiple workers, this creates a last-write-wins race condition.

---

## 4. Risk Assessment

### High confidence (80%+)

**Post-seal state routing bug** [S56, S2]
- `seal.md` sets `milestone` to "Crowned Anthill" but does NOT update `state`
- State remains "READY", causing `print-next-up` to suggest `/ant:build {next_phase}` pointing to non-existent phases
- `entomb.md` properly resets to IDLE [S57], but the gap between seal and entomb is a broken state
- **Impact:** Every sealed colony gets confusing routing suggestions
- **Fix:** Add SEALED/COMPLETED to state machine, or have seal update the state field

**Incomplete safety mechanism coverage** [S27, S29-S33, S45, S46, S48, S50]
- `atomic_write`: 4 of 10+ JSON write paths (40%)
- `session-verify-fresh`: 5 of 43 commands (12%)
- `CONTEXT.md` updates: 5 commands + playbooks (12%)
- `--no-visual`: 18 of 43 commands (42%)
- `pheromone_protocol`: 3 of 22 agents (14%)
- `print-standard-banner`: 0 of 43 commands (0%)
- Six independent instances across four research questions confirm this as a systemic trait, not isolated oversights

### Medium confidence (50-79%)

**LLM/bash write coordination gap** [S1, S3, S4, S28]
- Two independent write paths to COLONY_STATE.json with no shared locking
- Practical risk depends on concurrent write frequency during parallel builds, which requires runtime measurement
- **Impact:** Potential state corruption during parallel worker execution

**Single-lock architecture limitation** [S28] (single source)
- `file-lock.sh` can hold only ONE lock per process
- Nested acquisition silently overwrites tracking
- Lock keys are basename-only, creating same-name collision risk
- **Impact:** Complex operations requiring multiple locks cannot be safely coordinated

**HANDOFF.md missing fallback** [S44, S15]
- `resume-colony.md` has no fallback when HANDOFF.md doesn't exist
- Crash scenarios that don't trigger HANDOFF generation leave users unable to fully resume
- **Impact:** Session recovery fails after certain crash types

### Low confidence (<50%)

**Concurrent write frequency** — Unknown how often LLM and bash processes write simultaneously during typical builds. Requires runtime instrumentation to assess practical risk. (No source — identified as gap)

**Non-TTY visual rendering** — Unknown how visual elements (ASCII art, box-drawing, emoji) render in non-terminal contexts (CI pipelines, log files). Requires runtime testing. (No source — identified as gap)

---

## 5. Scalability Analysis

### Current Capacity

The system is designed for **single-user, single-colony-per-repo operation**:
- One COLONY_STATE.json per `.aether/data/` directory [S2]
- Single-lock-per-process architecture [S28]
- Colony-prime assembles context within 8K char budget to avoid prompt bloat [S35]
- Hive Brain caps at 200 wisdom entries with LRU eviction [S19]
- Pheromone-prime caps at 8 active signals per injection [S39]

### Growth Limitations

**High confidence (80%+):**
- **aether-utils.sh monolith** (11,221 lines, 125 subcommands) [S13] — Adding features increases cognitive load and parse time. The v2.0.0 extraction of hive/midden to separate files [S19, S20] was a step in the right direction but only moved ~9 functions out.
- **Budget-based context trimming** [S35] — As wisdom, learnings, and pheromones grow, lower-priority sections get trimmed. Users won't see a failure — just silently reduced context fidelity.

**Medium confidence (50-79%):**
- **42 commands + playbooks as markdown** — Each command is a Claude Code prompt file. Scaling to 60+ commands increases command discovery friction (no categories, search, or grouping).
- **JSON state files without schema** — No schema validation for COLONY_STATE.json, pheromones.json, etc. As structure evolves, backward compatibility depends on careful defaults and jq error handling [S34, S35].

### Scaling Strategy

The current architecture works well for its intended use case (individual developers running colonies on local repos). Scaling to multi-user or multi-colony scenarios would require: shared state with proper locking, schema versioning for state files, and splitting aether-utils.sh into domain-specific modules.

---

## 6. Improvement Recommendations

Prioritized by user impact and implementation effort. Synthesized from all 8 research questions and 4 cross-question patterns.

### Critical (State/Data Integrity)

| # | Issue | Source | Fix |
|---|-------|--------|-----|
| 1 | **Post-seal state routing** — seal sets milestone but not state, causing broken routing | [S56, S2] | Add SEALED state to print-next-up, or have seal.md update state field |
| 2 | **CLAUDE.md session-verify-fresh claim** — docs say "all stateful commands" but only 5/43 implement it | [S45] | Correct documentation or expand implementation |

### High (User Experience)

| # | Issue | Source | Fix |
|---|-------|--------|-----|
| 3 | **Plan approval touchpoint** — plans auto-commit phases without user approval | [S60] | Add summary + confirmation before committing to COLONY_STATE.json |
| 4 | **Autopilot positive checkpoint** — all 10 pause conditions are negative; no "proceed?" gate | [S61, S53] | Add optional phase-completion checkpoint (disable with `--no-confirm`) |
| 5 | **Continue pheromone visibility** — all pheromone operations silent during continue | [S43] | Add brief signal change summary to continue output |
| 6 | **Pheromone protocol expansion** — 19/22 agents lack formal signal handling | [S36-S38] | Add pheromone_protocol to at least the 5 agents that mention pheromones incidentally |

### Medium (Consistency & Polish)

| # | Issue | Source | Fix |
|---|-------|--------|-----|
| 7 | **Banner standardization** — 4+ styles; print-standard-banner exists but unused | [S48-S50] | Adopt one variant across all commands |
| 8 | **--no-visual parity** — only 42% of commands support it | [S48, S49, S51] | Extend to all visual-heavy commands |
| 9 | **Progress bar unification** — two implementations with different formats | [S50, S51] | Consolidate to one implementation |
| 10 | **Error state visual treatment** — errors as plain text, successes richly formatted | [S48] | Create standard error display template |

### Low (Technical Debt)

| # | Issue | Source | Fix |
|---|-------|--------|-----|
| 11 | **resume-colony HANDOFF.md fallback** | [S44, S15] | Fall back to COLONY_STATE.json + CONTEXT.md |
| 12 | **Non-atomic write paths** — 6 JSON mutations use unsafe echo writes | [S30-S33] | Migrate to atomic_write |
| 13 | **Dead-end command routing** — preferences and watch have no next-step | [S58, S59] | Add minimal routing (arguably intentional) |
| 14 | **session-verify-fresh coverage** — 5/43 commands | [S45] | Add to core commands |
| 15 | **CONTEXT.md update coverage** — 30+ commands never update it | [S46, S47] | Add to lifecycle commands |

---

## 7. Cross-Question Patterns

Four systemic patterns emerged from cross-validating 8 research questions:

### Pattern 1: Autonomous-First, Validate-After
The system does work autonomously and asks questions afterward. Plan auto-commits phases [S60]. Build runs workers autonomously [S3]. Continue gates verification AFTER work is done [S62]. Autopilot converts the one mandatory gate from interactive to auto-PAUSE [S61]. **Validated by 5 independent question tracks** (Q1, Q2, Q3, Q4, Q7) — the strongest finding in this audit.

### Pattern 2: Happy-Path Richness, Error-State Poverty
Success states get ASCII art, progress bars, box-drawing tables, and emoji [S48, S49]. Error states get plain text [S48]. The system celebrates progress but mutes problems. Q6's silent failure modes (corrupted JSON returning defaults [S35]) extend this to the data layer.

### Pattern 3: Mechanisms Exist but Coverage is Incomplete
Six independent instances: atomic_write (4/10+ paths [S27, S29-S33]), session-verify-fresh (5/43 commands [S45]), CONTEXT.md updates (5 commands [S46, S47]), print-standard-banner (0 commands [S50]), pheromone protocol (3/22 agents [S36-S38]), --no-visual (18/43 commands [S48-S51]). **The most pervasive architectural characteristic.**

### Pattern 4: State Dimensions Are Disconnected
`state` and `milestone` fields have no coordination — seal updates one but not the other [S56]. CONTEXT.md and HANDOFF.md run in parallel with no cross-referencing [S5, S14, S15]. Pheromone injection and consumption evolved independently (colony-prime injects to all, but only 3 agents have protocol [S13, S36-S38]).

---

## 8. Resolved Contradictions

| Contradiction | Resolution | Evidence |
|---|---|---|
| Pheromone "lowest priority" label vs. actual persistence | Naming confusion — "priority" means trim order, not importance. Lowest trim = highest persistence. | [S35] |
| CLAUDE.md session-verify-fresh claim vs. 5/43 reality | Documentation bug. | [S45, S5] |
| Plan autonomy vs. user profile ("I test like a user") | Genuine design tension — autonomous-first was built before user persona was defined. | [S60] |
| workers.md as "central definition" vs. pheromone protocol in agents | Evolved separation — workers.md = "what", agent definitions = "how". Acceptable architecture. | [S16, S36-S38] |
| atomic_write (safe) coexisting with echo>file (unsafe) | Incomplete migration — atomic_write added for critical paths, secondary files never migrated. | [S27, S29-S33] |
| seal sets milestone but not state | Bug. State machine has no terminal state. | [S56, S2] |
| session-read computes staleness but resume ignores it | Over-engineering — staleness was built but consumer was designed to restore regardless. | [S45, S5] |
| print-standard-banner exists but 0 commands use it | Abandoned standardization attempt. | [S50] |

---

## 9. Open Questions

These gaps cannot be resolved through static code analysis:

### Runtime Testing Required
- Non-TTY visual rendering behavior (ASCII art, box-drawing, emoji in CI/log contexts)
- Practical frequency of concurrent LLM + bash writes to COLONY_STATE.json
- Corrupted-JSON degradation path testing (unit-level)
- Claude behavior when HANDOFF.md is missing during resume-colony
- CONTEXT.md staleness impact during real non-build workflows

### Design Intent Clarification Required
- Are dead-end commands (preferences, watch) intentionally terminal? (Likely yes)
- Is the asymmetric touchpoint distribution deliberate? (Evidence suggests yes — 5 questions converge)
- Is per-command visual theming intentional or accidental divergence? (Evidence suggests accidental — abandoned print-standard-banner [S50])

---

## 10. Sources

All 66 sources are codebase files examined during 11 research iterations.

| ID | Path | Description |
|---|---|---|
| S1 | `.claude/commands/ant/init.md` | Init command — colony initialization flow |
| S2 | `.aether/aether-utils.sh:201-235` | `print-next-up()` — state-based routing |
| S3 | `.claude/commands/ant/build.md` | Build command — playbook orchestrator |
| S4 | `.claude/commands/ant/continue.md` | Continue command — verify and advance |
| S5 | `.claude/commands/ant/resume.md` | Resume command — 10-step session recovery |
| S6 | `.claude/commands/ant/plan.md` | Plan command — iterative research and planning |
| S7 | `.aether/docs/command-playbooks/build-complete.md` | Build completion playbook |
| S8 | `.aether/docs/command-playbooks/build-context.md` | Build context — colony-prime and pheromone injection |
| S9 | `.claude/commands/ant/council.md` | Council command — multi-choice intent clarification |
| S10 | `.claude/commands/ant/status.md` | Status command — colony dashboard with visual elements |
| S11 | `CHANGELOG.md` | Changelog — v1.1.11 to v2.0.0 changes |
| S12 | `.claude/commands/ant/run.md` | Autopilot — build-verify-advance loop |
| S13 | `.aether/aether-utils.sh:7855-8510` | `colony-prime` — unified worker priming payload |
| S14 | `.claude/commands/ant/pause-colony.md` | Pause — session handoff with state preservation |
| S15 | `.claude/commands/ant/resume-colony.md` | Resume-colony — full state restoration |
| S16 | `.aether/workers.md` | Worker definitions — roles, model selection |
| S17 | `.claude/commands/ant/seal.md` | Seal — Crowned Anthill milestone ceremony |
| S18 | `.aether/aether-utils.sh:26-33` | Sourcing chain — utility imports at startup |
| S19 | `.aether/utils/hive.sh` | Extracted hive functions |
| S20 | `.aether/utils/midden.sh` | Extracted midden functions |
| S21 | `.aether/utils/file-lock.sh:20` | LOCK_DIR definition |
| S22 | `.aether/aether-utils.sh:462-477` | `context-update` constraint handler |
| S23 | `.aether/aether-utils.sh:509-510` | Decision auto-emit FEEDBACK pheromone |
| S24 | `.aether/docs/command-playbooks/build-wave.md:262-264` | Midden injection per wave |
| S25 | `tests/bash/test-hive-wisdom.sh` | Hive wisdom bash tests |
| S26 | `npm test` output | Test suite — 542 tests passed |
| S27 | `.aether/utils/atomic-write.sh` | Atomic write — temp+rename with JSON validation |
| S28 | `.aether/utils/file-lock.sh` | File lock — noclobber-based with PID tracking |
| S29 | `.aether/aether-utils.sh:1315-1348` | `error-add` — atomic_write path |
| S30 | `.aether/aether-utils.sh:1470-1480` | `learning-promote-global` — non-atomic |
| S31 | `.aether/aether-utils.sh:1740-1763` | `error-patterns-record` — non-atomic |
| S32 | `.aether/aether-utils.sh:2530-2540` | `swarm-findings-add` — non-atomic |
| S33 | `.aether/aether-utils.sh:2935-2946` | `registry-register` — non-atomic |
| S34 | `.aether/aether-utils.sh:80-110` | Feature detection and graceful degradation |
| S35 | `.aether/aether-utils.sh:8150-8484` | Colony-prime budget enforcement |
| S36 | `.claude/agents/ant/aether-builder.md:76-110` | Builder pheromone_protocol |
| S37 | `.claude/agents/ant/aether-watcher.md:93-126` | Watcher pheromone_protocol |
| S38 | `.claude/agents/ant/aether-scout.md:47-81` | Scout pheromone_protocol |
| S39 | `.aether/aether-utils.sh:7676-7853` | `pheromone-prime` — signal decay and assembly |
| S40 | `.aether/aether-utils.sh:7177-7291` | `pheromone-display` — formatted signal table |
| S41 | `.aether/docs/command-playbooks/build-wave.md:277-319` | Builder Worker Prompt template |
| S42 | `.aether/docs/command-playbooks/build-verify.md:17-27` | Watcher Worker Prompt template |
| S43 | `.aether/docs/command-playbooks/continue-advance.md:169` | Silent pheromone operations in continue |
| S44 | `.aether/utils/state-loader.sh` | State loader with HANDOFF.md detection |
| S45 | `.aether/aether-utils.sh:9626-9654` | `session-read` with staleness calculation |
| S46 | `.aether/docs/command-playbooks/build-wave.md:257-289` | Build wave context-update calls |
| S47 | `.aether/docs/command-playbooks/continue-finalize.md:259-270` | Continue finalize context-update |
| S48 | `.claude/commands/ant/status.md:224-249` | Status ASCII art + dashboard |
| S49 | `.claude/commands/ant/maturity.md:32-87` | Maturity ASCII art + journey |
| S50 | `.aether/aether-utils.sh:186-196` | `print-standard-banner` (unused) |
| S51 | `.aether/utils/swarm-display.sh:94-111` | `render_progress_bar` (percentage format) |
| S52 | `.claude/commands/ant/` (all 43 files) | print-next-up grep results |
| S53 | `.claude/commands/ant/run.md:60-186` | Autopilot loop + 10 pause conditions |
| S54 | `.aether/docs/command-playbooks/build-complete.md:313-321` | Build-complete routing |
| S55 | `.aether/docs/command-playbooks/continue-finalize.md:39-72` | Continue-finalize routing |
| S56 | `.claude/commands/ant/seal.md:341-348` | Seal milestone without state update |
| S57 | `.claude/commands/ant/entomb.md:346-370` | Entomb state reset |
| S58 | `.claude/commands/ant/preferences.md` | Preferences — dead-end routing |
| S59 | `.claude/commands/ant/watch.md` | Watch — dead-end routing |
| S60 | `.claude/commands/ant/plan.md:410-421` | Plan auto-finalize |
| S61 | `.claude/commands/ant/run.md:112-114` | Autopilot runtime gate override |
| S62 | `.aether/docs/command-playbooks/continue-gates.md:554-621` | Runtime Verification Gate |
| S63 | `.aether/docs/command-playbooks/continue-finalize.md:179-244` | Commit + context clear prompts |
| S64 | `.claude/commands/ant/init.md:16-21` | Init overwrite protection |
| S65 | `.aether/docs/command-playbooks/build-complete.md:165-183` | Visual Checkpoint |
| S66 | `.aether/docs/command-playbooks/continue-verify.md:325-331` | Verification failure confirmation |

---

*Oracle research complete. 11 iterations, 54 findings, 66 sources, 8 questions, 4 cross-question patterns, 8 resolved contradictions. Overall confidence: 80%. Evidence ceiling: ~84% (static analysis limit).*
