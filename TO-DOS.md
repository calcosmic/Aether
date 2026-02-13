# TO-DOS

This file tracks pending work items. Each todo is self-contained with full context for future reference.

---

## Priority 0: Urgent

### BUG: Build checkpoint stashes user data -- nearly lost Oracle spec - 2026-02-13

- **The build checkpoint system uses `git stash` on ALL dirty files, including user work that has nothing to do with the phase.** During the repo-local migration (phase 5), the checkpoint stashed 1145 lines of uncommitted TO-DOS.md content (Oracle spec, 10 advanced colony ideas, multi-ant vision) and never popped it back. User nearly lost hours of work -- only recovered by manually searching git stashes. **Root cause:** `git stash` is a blunt instrument. The checkpoint system doesn't distinguish between "system files I'm about to modify" and "user's unrelated work in progress." **Fix:** The build/update system must ONLY modify files on an explicit allowlist of system files. Never stash, checkpoint, or touch anything outside that list. If system files are dirty, warn the user -- but leave their work alone. **System files (safe to modify):** `.aether/*.md`, `.aether/aether-utils.sh`, `.aether/docs/`, `.claude/commands/ant/`, `.opencode/commands/ant/`, `runtime/`, `bin/cli.js`. **User data (NEVER touch):** `.aether/data/`, `.aether/dreams/`, `.aether/oracle/`, `TO-DOS.md`, COLONY_STATE.json, flags, learnings, constraints, project files. **The boundary is simple: system files are the tool, user data is their work. Updates touch the tool, never the work.**

### Remove run_in_background from build.md worker spawns - 2026-02-12

- **Delayed task-notification banners make build summaries look premature** - build.md spawns workers with `run_in_background: true` then collects results via `TaskOutput`. The data is correct but Claude Code fires `task-notification` banners asynchronously after the summary is already displayed, making it look like the summary was written before agents finished. **Fix:** Remove `run_in_background: true` from all Task calls in build.md Steps 5.1, 5.4, and 5.4.2. Multiple Task calls in a single message already run in parallel without the background flag — they just block and return results directly. Then remove Steps 5.2 and 5.4.1 (TaskOutput collection) since results come back from the Task calls themselves. Apply same change to OpenCode mirror. **Files:** `.claude/commands/ant/build.md`, `.opencode/commands/ant/ant:build.md`. **Scope:** modest — remove flag + delete ~20 lines of TaskOutput instructions.

### Deprecate old 2.x npm versions - 2026-02-12

- **npm registry has stale 2.x pre-release versions that could confuse users** - Versions 2.0.0 through 2.4.2 exist on npm from pre-stable development. The `latest` dist-tag correctly points to 1.0.0, so `npm install` works fine. But the 2.x versions are visible on the npm page and could confuse people into thinking they're newer. **Fix:** Run `npm deprecate aether-colony@">=2.0.0" "Pre-release versions. Install 1.0.0 for the stable release."` to mark them deprecated. **Scope:** one command. **Urgency:** high — public-facing confusion on npm.

---

## Priority 0.5: High Priority

### Implement Anthill Milestone System - 2026-02-13

- **Integrate biologically-grounded milestone naming into the colony lifecycle** - Aether currently has no formal milestone/versioning concept in its biological vocabulary. A full naming taxonomy and milestone system has been researched and saved to `.aether/docs/biological-reference.md`. The milestone names map real ant biology to project stages: **First Mound** (first runnable), **Open Chambers** (feature work underway), **Brood Stable** (tests green), **Ventilated Nest** (perf acceptable), **Sealed Chambers** (interfaces frozen), **Crowned Anthill** (release), **New Nest Founded** (next major). **Implementation:** (1) Add milestone tracking to COLONY_STATE.json (current milestone name + criteria), (2) Update `/ant:continue` to detect milestone transitions and announce them, (3) Update `/ant:status` to show current milestone, (4) Consider adopting the expanded caste/command taxonomy (12 roles, 100+ commands) from the reference doc as a roadmap for future commands. **Reference:** `.aether/docs/biological-reference.md` (40+ research sources, full command taxonomy, milestone definitions). **Scope:** medium for core milestone tracking, large for full taxonomy adoption.

---

## Priority 1: Core UX Fixes

### ~~Investigate: Output Appears Before Agents Finish - 2026-02-10~~ FIXED

- **RESOLVED:** Updated `build.md` to enforce blocking behavior. Steps 5.2, 5.4.1, and 5.6 now explicitly require waiting for ALL TaskOutput calls to return before proceeding. Next Steps are now conditional based on actual verification results. If verification fails, `/ant:continue` is not suggested.

### Build summary displays before task-notification banners arrive - 2026-02-12

- **Phase summary appears before all background agent notifications are shown to the user** - During `/ant:build`, workers are spawned with `run_in_background: true` and then collected via `TaskOutput` with `block: true`. The Queen synthesizes results and displays the summary based on the TaskOutput data (which IS the real completed output). However, Claude Code's `task-notification` banners for each agent arrive asynchronously AFTER the summary is already displayed, making it look like the summary was written before agents finished. This is confusing — the user sees "Phase 3 complete" and then gets 3 "Agent completed" notifications afterward. **Problem:** Even though the data is correct (TaskOutput blocks until completion), the visual ordering undermines trust in the summary. **Possible fixes:** (1) Don't use `run_in_background` — use foreground Task calls so there are no delayed notifications, (2) Add a brief "Waiting for notifications to clear..." step after TaskOutput collection, (3) Accept the UX quirk and document it. **Scope:** Investigate whether foreground Task calls can still be parallelized, or whether background is required for parallel spawning.

### ~~Progressive Disclosure UI - 2026-02-10~~ FIXED

- **RESOLVED:** Implemented compact-by-default output with `--verbose` flag for full details. Created format specification at `.aether/docs/progressive-disclosure.md`. Updated `status.md` (8-10 lines default) and `build.md` (12 lines default) with compact/verbose modes. Bracket counts like `[3 blockers]` indicate expandable sections.

### Auto-Load Context on Colony Commands - 2026-02-10

- **Colony commands should automatically load relevant context** - When running `/ant:init` or `/ant:plan` (especially after `/clear`), the system should automatically check and load: (1) TO-DOs.md for relevant pending work, (2) Colony state from previous sessions, (3) Any initialized goals/intentions. **Problem:** Natural discussions happen (brainstorming, adding to TO-DOs, research) but when colony commands run, they don't automatically pull in this context. User has to manually reference things. After context clear, commands should restore from persistent state. **Files:** `ant:init`, `ant:plan`, potentially all colony commands. **Solution:** At the start of key commands, automatically read TO-DOs.md and colony state files, surface relevant items, and incorporate into the command's context. `/ant:init` should ask "I see these pending TO-DOs - want to work on any of these?" `/ant:plan` after clear should read initialized state and continue from there.

### ~~Command Suggestions Must Be Actual Commands - 2026-02-10~~ FIXED

- **RESOLVED:** Updated `status.md`, `continue.md`, `plan.md`, and `phase.md` to calculate actual phase numbers before display. Each file now has explicit instructions to substitute real values (e.g., `/ant:build 3`) instead of template placeholders. `build.md` was already correct.

### Question: What is the Point of /ant:status? - 2026-02-11

- **Evaluate whether /ant:status is actually useful** - The colony already tells you everything as you go along. After `/ant:build`, it shows what was done. After `/ant:continue`, it shows what's next. The auto-recovery headers now show context after `/clear`. So when would a user think "I need to see the status"? **Possible answers:** (1) It's redundant and should be removed. (2) It's useful for a "dashboard" view showing flags, instincts, all phases at once. (3) It's useful when you're unsure what state the colony is in. **Current behavior:** Shows goal, phase progress, tasks, constraints, flags, instincts, suggested next command. **Question to answer:** Is there a scenario where this is actually useful that isn't already covered by the commands themselves? If not, consider deprecating. If yes, make the use case clearer in docs.

### ~~Command Suggestions Need Phase Context - 2026-02-11~~ DONE

- **RESOLVED:** Updated Next Steps output in `status.md`, `plan.md`, and `phase.md` (both `.claude` and `.opencode` variants) to include phase name inline with command suggestions. E.g., `/ant:build 3   Phase 3: Add Authentication`. `continue.md` already had this. Also fixed `.opencode/plan.md` which was hardcoded to Phase 1 — now dynamically calculates first incomplete phase. Synced to global `~/.claude/commands/ant/`.

### Codebase Ant Pre-Flight Check - 2026-02-11 ⭐ HIGH IMPACT

- **Automatic plan validation against current codebase before each phase executes** - Plans are made with imperfect knowledge; by execution time, the codebase may have changed or the planner may have missed existing patterns. A "Codebase Ant" should validate each phase's tasks against the actual codebase before workers spawn. **Problem:** Planning ant researches but can miss things. Codebase evolves between planning and execution. Tasks may reference files that don't exist, miss better patterns, or conflict with recent changes. **When:** Automatically in `/ant:build` after reading state (Step 4.5) but before spawning workers (Step 5). **What Codebase Ant does:** (1) Read phase tasks, (2) For each task: verify referenced files/paths exist, find existing patterns task should follow, check for recent changes that conflict, identify simpler approaches. (3) Output validation result with suggestions. **Output format:** `🗺️ PRE-BUILD VALIDATION` showing per-task checks with ✅/⚠️/💡 indicators. **Constraints:** Fast (<30 seconds) using only Glob/Grep/Read - no deep research. Non-blocking if validation passes. Can auto-inject discoveries into task hints. Surfaces warnings but only halts on critical issues (e.g., referenced file missing). **Implementation:** Add Step 4.5 to `build.md` that spawns a Scout with codebase validation prompt, collects result, enriches task hints, then proceeds to spawn builders. **Why P1:** Catches plan/reality mismatches before wasted work. Improves worker success rate. Makes the colony smarter about its own codebase.

### ~~Pass Learnings and Instincts to Spawned Workers - 2026-02-10~~ DONE

- **RESOLVED:** Added `--- COLONY KNOWLEDGE ---` section to Builder Worker Prompt Template in `build.md` (both `.claude` and `.opencode` variants). Workers now receive: (1) Top 5 instincts by confidence (>= 0.5), (2) Recent validated learnings from last 3 phases, (3) Flagged error patterns to avoid. Section is omitted entirely if no relevant knowledge exists. Synced to global `~/.claude/commands/ant/build.md`.

---

## Priority 2: Context Management Infrastructure

### Session Continuity Marker - 2026-02-10

- **Track last activity for seamless resume** - Store lightweight session state for instant context recovery. **Implementation:** `.aether/data/session.json` with: `last_command`, `last_command_at`, `context_cleared`, `suggested_next`, `active_todos`. On any command start: check session.json, if recent activity show "Continuing from {last_command}". **Token consideration:** Single small JSON file, read once at command start.

### Pre-Command Context Check (All Commands) - 2026-02-10

- **Universal context loading before any /ant:* command** - Every command should start with same context awareness. **Implementation:** Before execution: (1) Load COLONY_STATE.json (goal, phase, state), (2) Load constraints.json, (3) Load session.json, (4) Check TO-DOS.md for matches. Build 3-line context summary injected into command. **Token consideration:** Summary only, not full file contents. ~100 tokens max.

### Background CONTEXT.md File - 2026-02-10

- **Auto-generated ambient context file updated after each phase** - Single file that captures current state in human-readable format. **Implementation:** `.aether/data/CONTEXT.md` auto-updated by `/ant:continue`: Current Focus, Recent Decisions, Top Instincts, Known Issues. Commands read first 50 lines for instant orientation. **Token consideration:** Capped at 50 lines, replaces reading multiple files.

### Chamber Specialization (Code Zones) - 2026-02-10

- **Categorize codebase into behavioral zones during colonization** - During colonization, categorize code areas into zones that affect worker behavior: **Fungus Garden (core)**: Critical paths, high test coverage areas - workers use extra caution, more testing, slower changes. **Nursery (new)**: Recently created features - okay to iterate fast, more experimental. **Refuse Pile (deprecated)**: Legacy/dead code - workers avoid touching unless explicit, quarantine behavior. **Implementation:** Add `zones` object to colony state during colonization. Workers check zone before starting work and adjust behavior accordingly. **Biological basis:** Leafcutter ants maintain separate chambers for fungus gardens vs waste to prevent cross-contamination. **Why foundational:** Panic vs Aggressive Alarm depends on knowing which zone code is in to determine response severity.

---

## Priority 3: Quick Wins (Simple, High Value)

### ~~Ant Graveyards Feature - 2026-02-10~~ DONE

- **RESOLVED:** Implemented `grave-add` and `grave-check` commands in `aether-utils.sh`. Grave markers are stored in `graveyards` array in COLONY_STATE.json (capped at 30 entries). Builder prompts in `build.md` now check for nearby graves before modifying files (`caution_level: "high"/"low"/"none"`). Failed workers automatically get grave markers recorded in Step 5.6. Both `.claude` and `.opencode` command copies updated. Existing colonies work via `// []` fallback; new colonies get `"graveyards": []` in init template.

### Panic vs Aggressive Alarm - 2026-02-10

- **Different error types trigger different colony responses** - Currently all errors treated similarly. Implement tiered alarm system: **AGGRESSIVE (security vulnerability near core)**: Halt everything, swarm to fix immediately. **PANIC (test failure in peripheral code)**: Stop current work, reassess before continuing. **NOTE (type error, lint warning)**: Log and continue. Response determined by error severity + proximity to core code. **Implementation:** Categorize errors by type and location. Different categories trigger different colony responses (halt vs continue vs swarm). **Biological basis:** Ant alarm responses depend on nest proximity - flight when far from nest, aggression when near. **Depends on:** Chamber Specialization (needs to know which zone code is in).

### TODO Relevance Scoring - 2026-02-10

- **Smart matching of TODOs to current context** - When surfacing TODOs, score by relevance not just priority. **Implementation:** Score = weighted sum of: priority (0.2), recency (0.2), keyword match to current goal (0.3), file proximity (0.3). Only surface TODOs with score > 0.5. Show top 3 max. **Token consideration:** Scoring happens before output, only relevant items sent to context.

---

## Priority 3.5: New Ant Types

### ~~🎲 Chaos Ant - 2026-02-11 ⭐ HIGH IMPACT~~ DONE

- **RESOLVED:** Implemented as `/ant:chaos` command (`chaos.md`). Adversarial testing agent spawned automatically during build verification (Step 5.4.2 in `build.md`) after the Watcher completes. Probes phase work for edge cases, boundary conditions, and resilience issues. Reports findings with severity levels; critical/high findings create blocker flags. Command files: `.claude/commands/ant/chaos.md` and `.opencode/commands/ant/ant:chaos.md`. Build pipeline integration at Step 5.4.2 (post-Watcher resilience testing). Utility support: `generate-ant-name "chaos"` and `get_caste_emoji` updated in `aether-utils.sh` with chaos-specific prefixes (Probe, Stress, Shake, etc.) and 🎲 emoji.

~~- **Adversarial testing agent that actively tries to break code** - Builders are optimistic ("it works!"), Watchers verify happy paths, but nobody actively tries to break things. Chaos Ant does. **What it tests:** Edge cases (empty strings, nulls, unicode, huge inputs), race conditions (what if two users do X simultaneously?), auth bypasses (what if admin but also deleted?), unexpected state combinations. **Implementation:** Spawned during verification phase after builders complete, alongside or before Watcher. Fed the code + success criteria, instructed to find ways to break them. Returns list of failures found with reproduction steps. **Output format:** `🎲 CHAOS REPORT: Found 3 breaks — (1) empty email crashes signup, (2) negative quantity accepted in cart, (3) deleted user can still access API`. **Biological basis:** Some ant species have "police" ants that attack colony members behaving abnormally — maintaining colony health through adversarial pressure. **Why high impact:** Catches the bugs that reach production. Proven value in chaos engineering.~~

### ~~🏺 Archaeologist Ant - 2026-02-11 ⭐ HIGH IMPACT~~ DONE

- **RESOLVED:** Implemented as `/ant:archaeology` command (`archaeology.md`). Git history analyst spawned automatically during build pre-flight (Step 4.5 in `build.md`) when phases modify existing files with significant history. Runs `git log`, `git blame`, analyzes commit context, and surfaces tribal knowledge. Findings injected into builder prompts as `archaeology_context` (Step 5.1). Also invokable manually: `/ant:archaeology src/path/`. Command files: `.claude/commands/ant/archaeology.md` and `.opencode/commands/ant/ant:archaeology.md`. Build pipeline integration at Step 4.5 (Archaeologist pre-build scan). Utility support: `generate-ant-name "archaeologist"` and `get_caste_emoji` updated in `aether-utils.sh` with archaeologist-specific prefixes (Relic, Fossil, Dig, etc.) and 🏺 emoji.

~~- **Git history analyst that explains why code exists** - On mature codebases, the *why* matters as much as the *what*. "Don't remove this null check — it was added after the 2021 production crash." Without historical context, workers make confident mistakes. **What it does:** Runs `git log`, `git blame`, analyzes commit messages and PR descriptions. Reasons about *why* code is structured this way. Surfaces tribal knowledge buried in history. **When triggered:** Auto-triggered when `/ant:build` touches files with significant history (>2 years old, high churn, or many authors). Can also be invoked manually: `/ant:archaeology src/legacy/`. **Output format:** `🏺 ARCHAEOLOGY REPORT: This file has 847 commits from 12 authors. Key findings: (1) Lines 45-60 are a workaround for iOS 12 bug #4521 — iOS 12 now unsupported, safe to remove. (2) The unusual caching pattern was added after a DDoS in 2022, do not simplify. (3) Author left TODO on line 203: "temporary fix" — it's been 3 years.` **Implementation:** Spawned as Scout with git analysis prompt. Injects findings into builder prompts as context. **Why high impact:** Prevents "removed dead code, broke everything" disasters. Saves hours of investigation.~~

### ~~💭 Dreamer Ant - 2026-02-11~~ DONE

- **RESOLVED:** Implemented as `/ant:dream` command (`dream.md`). Philosophical wanderer agent that reads codebase, git history, colony state, and TO-DOs, then performs 5-8 cycles of random exploration writing observations to `.aether/dreams/`. Each dream has a category (musing/observation/concern/emergence/archaeology/prophecy/undercurrent), a deep reflection, a plain-terms "for dummies" explanation, and optional pheromone suggestions with copy-paste commands and their own plain explanations. Never modifies code or colony state. Can run in dedicated terminal or same session.

### Surface Dreams in /ant:status - 2026-02-11

- **Show recent dream summary in colony status output** - Add a small section to `/ant:status` that reads `.aether/dreams/` directory, finds the most recent dream file, and displays a compact summary (e.g., `💭 3 dreams (last: 2h ago) | 1 concern`). Expandable with `--verbose` to show one-liners per dream. **Files:** `status.md`. **Why:** Dreams should surface where users already look, not require a separate command.

### Dreamer Build Integration - 2026-02-11

- **Optionally check relevant dreams before phase execution** - During `/ant:build`, after loading state but before spawning workers, check if any recent dreams are relevant to the phase about to be built. Surface relevant dreams as context. **Files:** `build.md`. **Why:** Makes dreams actionable without requiring manual review. **Deferred:** Implement after Dreamer has been tested and proven useful.

### ~~Mark Unbuilt .planning/ Designs with Status Headers - 2026-02-11~~ DONE

- **RESOLVED:** Added `> STATUS: NOT IMPLEMENTED — Research artifact from Phase 2` headers to `git-staging-tier3.md` and `git-staging-tier4.md`. Files kept in place as reference material with clear status markers.

~~- **Add status markers to unimplemented research artifacts**~~ - `.planning/` contains 3,522 lines across 12 files. Tier 3 (`git-staging-tier3.md`, 441 lines) and Tier 4 (`git-staging-tier4.md`, 528 lines) are fully designed but were never built. They have no markers distinguishing them from implemented work — a newcomer reading them would assume Aether supports hooks-based auto-commits and GitHub PR integration. **Fix:** Add `> STATUS: NOT IMPLEMENTED — research artifact from Phase 2` header to each unbuilt file. Keep files in place (they're valuable reference for *why* Tier 2 was chosen). Optionally add a `.planning/README.md` index. **Source:** Dream session 2026-02-11, Dream 3: The Shadow of Unbuilt Futures. **Scope:** trivial, 2-3 files.

### ~~Colony Memory: Seed Instincts from Prior Sessions (Minimal Fix) - 2026-02-11~~ DONE

- **RESOLVED:** Added Step 2.5 to `init.md` that reads `completion-report.md` and seeds `memory.instincts` (confidence >= 0.7) and `memory.phase_learnings` (validated) into the new colony. Non-blocking and gracefully skips if no report exists. Both `.claude` and `.opencode` mirrors updated. Full cross-session memory system remains at Priority 5.

~~- **Make init.md load high-confidence instincts from previous completion reports**~~ - Each colony starts fresh with empty `memory.instincts`, `memory.phase_learnings`, and `memory.decisions`. `completion-report.md` is written by `/ant:continue` at project completion but **never read** by any command. The data exists, only the loading is missing. **Minimal fix:** Add a step to `init.md` that reads the most recent `.aether/data/completion-report.md` (if it exists) and seeds `memory.instincts` with any instinct at confidence >= 0.7. Gives the new colony a head start without importing everything blindly. **Files:** `init.md` + `.opencode` mirror. **Source:** Dream session 2026-02-11, Dream 5: The Eternal Present of a Colony Without Memory. **Scope:** modest, medium. **Note:** The full cross-session memory system (Priority 5: `.aether/projects/<hash>/`) remains the long-term goal — this is the 80% fix.

---

## Priority 4: Enhancements

### Smart Command Suggestion - 2026-02-10

- **Context-aware next command suggestions** - Instead of just "Next: /ant:build phase:2", analyze context for smarter suggestions. **Implementation:** If TO-DOS.md has P1 bug → suggest `/ant:swarm`. If last phase had low watcher score → suggest `/ant:focus "quality"`. If constraints empty → suggest adding focus/redirect. Decision tree based on colony state.

### Immune Memory (Pathogen Recognition) - 2026-02-10

- **Recognize recurring bug patterns and escalate response** - Track "pathogen signatures" (recurring error patterns, bug types). When a similar error appears, check against known signatures. If match: boost worker count, increase scrutiny, flag as "known pathogen - escalating response." Goes beyond instinct confidence - this is about recognition and escalation of known threats. **Implementation:** Add `pathogens` array to colony state with error signatures. Error handling checks for pattern matches and triggers escalated response (more workers, higher priority). **Biological basis:** Leafcutter ants remember pathogens for 30+ days and fight them more intensely on re-exposure.

### Conversation-to-Colony Bridge - 2026-02-10

- **Detect intent in natural discussion and suggest colony actions** - When in discussion mode, recognize patterns and offer transitions. **Implementation:** Detect "let's work on X" → suggest `/ant:init "X"`. Detect research findings → store for next `/ant:plan`. Detect decisions "let's use JWT" → offer to add constraint. **Token consideration:** Pattern matching on user input, no extra context needed.

### YAML Command Generator — Eliminate Manual Duplication - 2026-02-11

- **Build the YAML-based command generation system described in `src/commands/README.md`** - Currently 22 command files are manually duplicated across `.claude/commands/ant/` (~4,939 lines) and `.opencode/commands/ant/` (~4,926 lines). The infrastructure is half-built: `src/commands/_meta/tool-mapping.yaml` (54 lines) and `src/commands/_meta/template.yaml` (64 lines) exist, and the README (110 lines) describes the full system. But no individual command YAML definitions or generator script were ever created. **Implementation:** (1) Create YAML definitions for all 22 commands (canonical source of truth), (2) Build `./bin/generate-commands.sh` using tool-mapping.yaml for platform-specific translation, (3) Add CI/pre-commit check to verify generated output matches source. **Source:** Dream session 2026-02-11, Dream 4: The Architecture That Lives Only in Words. **Scope:** significant, large — probably a multi-phase colony project. **Risk:** medium — generator bugs could silently break commands. **Note:** Manual duplication works today; this is efficiency/maintenance improvement, not a fix.

### Iron Law Process Logging (Lightweight) - 2026-02-11

- **Add optional process logging for Iron Law compliance** - All 6 Iron Laws in `workers.md` are text-only instructions with no runtime enforcement. The Watcher validates results (code compiles, tests pass) but not process (was TDD actually followed?). A Builder that writes code first and adds tests after is indistinguishable from proper RED-GREEN-REFACTOR. **If needed:** Require builders to log each TDD step (RED: wrote failing test → GREEN: made it pass → REFACTOR: cleaned up) in their activity output. Watcher checks the log for step completeness. **Source:** Dream session 2026-02-11, Dream 6: Iron Laws and Trust. **Scope:** significant, large — process tracing across all worker types. **Risk:** high — over-engineering enforcement could slow workers. **Recommendation:** Defer until quality issues are traced to Iron Law violations. Accept text-based discipline as sufficient for LLM agents for now.

---

## Priority 5: Evaluate Later

### Care-Kill Dichotomy - 2026-02-10

- **Explicit infection scoring for refactor vs delete decisions** - Add quantified "infection score" to code areas that triggers automatic care (refactor, improve) vs kill (delete entirely) recommendations. Currently this happens implicitly during planning - making it explicit could help route-setter make clearer recommendations. **Implementation:** Heuristics for infection scoring (error frequency, test coverage, change frequency, age). Score above threshold = recommend deletion. **Biological basis:** Ant workers kill infected brood when necessary to prevent systemic disease spread - care-kill dichotomy. **Status:** Maybe - evaluate after other features are in. May add complexity for marginal gain.

### Cross-Session Memory Persistence - 2026-02-10

- **Project-level memory like Claude Code's Auto Memory** - Persistent memory across sessions, not just within colony lifecycle. **Implementation:** `.aether/projects/<project-hash>/` with: MEMORY.md (index, first 200 lines loaded), decisions/, patterns/, failures/ (graveyards). Survives colony reset. **Token consideration:** Only load index file by default, deep files on demand.

### Git-Aware Context - 2026-02-10

- **Colony state awareness of git operations** - On branch switch: check if different colony session exists. On commit: auto-snapshot colony state. On merge conflict: surface relevant instincts about those files. **Implementation:** Git hooks or detection in commands. **Token consideration:** Minimal - just state lookups.

---

## Priority 6: New Features (Non-Urgent)

### Detective Command - 2026-02-09

- **Create detective command** - Build a new command that leverages deep research skill for thorough codebase investigations. **Problem:** Need a specialized command for deep-dive codebase analysis and investigation work. **Files:** `commands/detective.md` (new). **Solution:** Design command that uses deep research capabilities for comprehensive codebase exploration and detective-style investigation tasks. **Why P6:** New capability, not fixing existing issues.

---

## Priority 7: Research

### Research Claude Code Plugins - 2026-02-10

- **Investigate packaging Aether as a Claude Code plugin** - Research the official plugin format and whether Aether could be distributed as a plugin. **Problem:** Currently Aether is a collection of commands/skills that must be manually set up. Plugins allow single-command installation and sharing. **Files:** Review plugin docs at code.claude.com/docs/en/plugins, anthropics/claude-code GitHub, and claude-plugins.dev registry. **Solution:** Determine if we can bundle ant colony system + CDS workflow as an installable plugin for broader distribution. **Why P7:** Research task, can happen in parallel with implementation work.

---

## Priority 8: Backlog (Future Exploration)

### Weird Colony Ideas to Explore - 2026-02-10

- **Review creative colony enhancement concepts** - Brainstormed ideas for making the colony feel more alive without adding commands: (1) Colony Mood - single emergent emoji/word indicator of colony health, (2) Dream State - idle-time speculation where colony notices things while you're away, (3) Worker Personality Variance - subtle personality modifiers (careful/bold/curious/skeptical) creating emergence from variation, (4) Memory Echoes - code-location-attached memories that surface when revisiting files, (5) Silent Watcher - passive immune system that quietly emits pheromones when it notices concerns, (6) Entropy Signal - fourth pheromone type representing chaos/tech debt that colony naturally cleans, (7) Colony Echoes Across Projects - cross-pollination of instincts between colonies on similar tech stacks. **Status:** Ideas only, needs review to select which to pursue.

---

## Review 2026-02-11 Action Items

### Quick Wins

### ~~Sync aether-utils.sh between .aether/ and runtime/~~ 2026-02-11

- **Copy swarm/grave functions to runtime/** - `.aether/aether-utils.sh` has 10 functions missing from `runtime/aether-utils.sh`: autofix-checkpoint, autofix-rollback, spawn-can-spawn-swarm, swarm-findings-init, swarm-findings-add, swarm-findings-read, swarm-solution-set, swarm-cleanup, grave-add, grave-check. `runtime/` has 1 function .aether/ lacks: generate-commit-message. **Problem:** swarm.md calls these functions and will FAIL when run against runtime/. **Fix:** Copy the 10 missing functions to runtime/aether-utils.sh. **Scope:** trivial, ~80 lines. **Source:** Review 2026-02-11, Regression Hunter agent finding.

### ~~Fix package.json path mismatch~~ 2026-02-11

- **package.json lists non-existent file** - package.json:8-13 lists `.opencode/opencode.json` but the actual file is `.opencode/opencode.json`. npm pack may fail or produce unexpected results. **Fix:** Update files field to `.opencode/opencode.json` (with dot prefix). **Scope:** trivial, 1 line. **Source:** Review 2026-02-11, Git Distribution agent finding.

### ~~Squash 13 checkpoint commits before push~~ 2026-02-11

- **93% of commits are noise** - 13 of 14 unpushed commits are "aether-checkpoint: pre-phase-X" messages with no semantic meaning. Average quality score: 1.2/5. **Problem:** Pollutes git history, makes bisect/debug difficult. **Fix:** Interactive rebase to squash all checkpoints into 1-2 meaningful commits. Keep feature commit "Update spawn output format: emoji adjacent to ant name" separate. **Scope:** 10 minutes. **Source:** Review 2026-02-11, Agent 4 finding.

### ~~Deprecate stale TODO entries~~ 2026-02-11

- **Auto-Load Context and Background CONTEXT.md may be duplicates** - Auto-Load Context on Colony Commands (P1, 2026-02-10) and Pre-Command Context Check (P2, 2026-02-10) describe similar functionality. Background CONTEXT.md File (P2, 2026-02-10) mentioned in completion-report learnings but not implemented. **Fix:** Review these 3 entries, mark duplicates as deprecated, clarify remaining work. **Scope:** 5 minutes. **Source:** Review 2026-02-11, Process Reviewer agent finding.

### Medium Effort

### ~~Standardize YAML quoting across command mirrors~~ 2026-02-11 DONE

- **RESOLVED:** Colony session `session_1770811383_f1x3r` Phase 1 standardized all 26 command files to use double-quoted YAML description values across both `.claude/` and `.opencode/` mirrors. Verified by independent Watcher.

### ~~Lower instinct confidence threshold from 0.7 to 0.5~~ 2026-02-11 DONE

- **RESOLVED:** Colony session `session_1770811383_f1x3r` Phase 1 lowered threshold from 0.7 to 0.5 in both `.claude/commands/ant/init.md` and `.opencode/commands/ant/init.md`. Verified zero residual 0.7 references.

### ~~Add lint/typecheck scripts to package.json~~ 2026-02-11 DONE

- **RESOLVED:** Colony session `session_1770811383_f1x3r` Phase 2 added `lint:shell` (shellcheck), `lint:json` (node -e JSON.parse), `lint:sync` (generate-commands.sh check), and top-level `lint` scripts. Zero new dependencies. All pass clean.

### Strategic

### Build YAML-based command generator 2026-02-11

- **Eliminate manual command duplication** - 20-21 files manually maintained across `.claude/` and `.opencode/` mirrors. Risk of drift increases with each edit. `src/commands/README.md` describes a YAML-based generation system that was never built. **Implementation:** (1) Create YAML definitions for all commands (single source of truth), (2) Build `./bin/generate-commands.sh sync` to propagate changes, (3) Add CI check to verify generated output matches source. **Scope:** significant, large — probably a multi-phase colony project. **Risk:** medium — generator bugs could silently break commands. **Source:** Review 2026-02-11, Dream 4 finding + regression analysis.

### Commit or archive .planning/ directory 2026-02-11

- **Research docs not committed** - `.planning/` contains 13 files (~2,800 lines) of git-staging research and tier specifications. These are uncommitted. **Options:** (a) Commit to repository as design documentation, (b) Move to separate docs repo, (c) Delete if no longer relevant. Tier 3 and Tier 4 are marked "NOT IMPLEMENTED" — are these still planned? **Scope:** medium, decision + action. **Source:** Review 2026-02-11, Agent 1 finding.

### Adopt feature-branch workflow for checkpoints 2026-02-11

- **13/14 commits are checkpoints** - Heavy reliance on local save-points rather than semantic commits. **Problem:** Noise in git history, difficult to debug or bisect. **Fix:** Replace "aether-checkpoint" pattern with meaningful commit messages when changes are pushed. Consider using `git stash` for local save-points (Tier 1 stash-based checkpoints already implemented). **Scope:** behavioral, ongoing. **Source:** Review 2026-02-11, Agent 4 recommendation.

---

## Priority 0: Urgent - Bug Fixes

### Per-repo update mechanism (`/ant:update` or `aether update`) - 2026-02-13

- **Repos need a way to pull the latest Aether system files without overwriting colony data** - After the repo-local migration (Phases 1-4), each repo has its own copy of `.aether/` system files (utils, docs, workers, commands). When the source Aether repo gets updated with new features or bug fixes, there's no way to propagate those changes to other repos that use the colony system. **Requirements:** (1) A command (e.g., `/ant:update` or `node bin/cli.js update`) that updates system files in the current repo to the latest version, (2) It must NOT overwrite per-repo colony data (`.aether/data/` -- COLONY_STATE.json, activity.log, spawn-tree.txt, flags, learnings, error-patterns, signatures), (3) A version check that can notify users when an update is available (e.g., "Aether v1.1.0 available, you're on v1.0.0"), (4) Ideally runs automatically as a check at the start of common commands (`/ant:status`, `/ant:build`) with a non-blocking notice. **Design considerations:** Need to define what counts as "system files" vs "colony data" -- system files are the tools/commands/docs that ship with Aether, colony data is what each colony produces. The `.aether/data/` directory is already gitignored, which is a natural boundary. Could use `package.json` version comparison against a known source (npm registry, git tag, or a local reference repo). **Scope:** Medium. **Files:** `bin/cli.js` (update subcommand), potentially a new `/ant:update` command, and a version-check hook in status/build commands.

### Fix Ant Command Parsing - 'ant [command] [text]' Doesn't Execute - 2026-02-13

- **When running 'ant plan' with additional text, the command doesn't execute properly** - If the user runs "ant plan" followed by any text (e.g., "ant plan work on the authentication"), the ant plan command doesn't run. Instead, it does a "plan" without actually executing the /ant:plan command, which means the planning doesn't happen properly. This works correctly in Claude (native), but not in OpenCode. **Why it's P0:** Breaks core functionality - users cannot provide context to ant commands. **Investigation needed:** Compare OpenCode command parsing vs Claude command parsing to find why text arguments cause the command to not execute. **Files to check:** `.opencode/commands/ant/` command files, any argument parsing logic.

### Multi-Ant Parallel Execution - Colony Can Run Multiple Ants Simultaneously - 2026-02-13

- **Enable the colony to run multiple ant commands/tasks in parallel without conflicts** - Currently, only one ant command can run at a time. The vision is for the colony to become a massive network where many ants can work on different tasks simultaneously. The Queen ant must intelligently coordinate them so they don't conflict with each other's work.

**Core Problems to Solve:**
1. **State conflicts** - Two ants modifying COLONY_STATE.json simultaneously
2. **File conflicts** - Two ants editing the same file
3. **Resource conflicts** - Two ants running the same tests/builds
4. **Coordination** - How does Queen know what's happening across all ants?

**Potential Approaches:**

*Approach A: Session-Based Isolation* - Each ant command spawns a unique session ID. State files include session ID for ownership. Queen tracks all active sessions and their claimed files. Before any write, ant checks if file is claimed by another session. If conflict: wait, reassign, or abort with suggestion.

*Approach B: Queue + Worker Pool* - All ant commands go into a queue instead of running immediately. Queen assigns tasks to worker ants based on availability and skills. Only N ants active at once (configurable). Provides natural serialization for state, parallel for execution.

*Approach C: Optimistic Locking with Conflict Resolution* - Each file/task has a version number. Ant reads current state, does work, tries to write. If version changed, conflict detected -> auto-retry or merge. Queen mediates conflicts based on priority.

*Approach D: Spatial Division (Chamber-based)* - Divide codebase into zones (chambers) - see Chamber Specialization. Each ant assigned to specific zone(s). Ants in different zones can run in parallel safely. Ants in same zone coordinate via Queen.

**Knowledge Hierarchy Vision:** Queen has overall overview -> Specialized Castes (scouts, workers, nurses) have domain expertise -> Information trickles down -> Ants communicate discoveries to each other. The colony becomes a "little world of workers that almost really exist."

**One-Year Vision:** Create something extremely unique - a world of agents that feel like they really exist, with genuine emergent behavior from the interactions.

**Status:** DO NOT IMPLEMENT - discuss approach before designing.

---

## Priority 3: New Research Tasks

### Properly Implement Graveyard Feature - 2026-02-13

- **Graveyards feature exists but not implemented properly** - The graveyard feature was marked as DONE in a previous session, but implementation is incomplete or not working as intended. Need to: (1) Verify `grave-add` and `grave-check` functions exist in `aether-utils.sh` and work correctly, (2) Verify graveyard markers are being added to COLONY_STATE.json when workers fail, (3) Verify Builder prompts check for nearby graves before modifying files with appropriate caution levels, (4) Test the full flow: worker fails -> grave marker added -> future build respects grave -> caution level applied. **Files:** `.aether/aether-utils.sh`, `.claude/commands/ant/build.md`, `.opencode/commands/ant/build.md`, COLONY_STATE.json structure.

### Research and Implement Pheromone System - 2026-02-13

- **Pheromone system needs research and proper implementation** - Pheromones are the colony's communication mechanism but the current implementation is incomplete. Current state: TTL-based model documented but may not be fully implemented. Need to: (1) Research biological pheromone systems (trail, alarm, brood, queen pheromones), (2) Map to colony communication needs (task delegation, error warning, success signals, coordination), (3) Implement properly in code (not just docs), (4) Integrate with existing commands. **Note:** As part of this, need to discuss creating a knowledge hub for ant research findings - a place where separate agents can do extensive research loops and store findings. Could be `.aether/research/` directory. **Depends on:** Chamber Specialization for zone-aware pheromone responses.

---

### Oracle Ant: RALF-Based Research System - IMPLEMENTATION SPEC - 2026-02-13

**STATUS:** READY TO IMPLEMENT - Full spec below

---

## OVERVIEW

Implement Oracle Ant command using the RALF (Recursive Agent Loop Framework) pattern from https://github.com/snarktank/ralph. Oracle Ant is a deep research agent that runs in an iterative loop, with fresh context each iteration, persisting knowledge via files.

**Key Principle:** Oracle works EXACTLY like Ralph - sequential loop, one agent at a time, fresh context per iteration. NOT parallel execution initially.

---

## FILE STRUCTURE TO CREATE

```
.aether/oracle/
├── oracle.sh              # Main bash loop script (~100 lines)
├── oracle.md              # Prompt for AI agent (~80 lines)
├── research.json          # Research topic/questions (generated by command)
├── progress.md            # Append-only research log (generated by loop)
├── .stop                  # Stop signal file (created by /oracle:stop)
├── archive/               # Previous research runs (auto-created)
└── discoveries/
    └── synthesized.md     # Final summary (generated at end)

.opencode/commands/ant/
└── oracle.md              # Command definition (~120 lines)

.claude/commands/ant/
└── oracle.md              # Exact mirror of .opencode version
```

---

## FILE 1: `.aether/oracle/oracle.sh`

**Purpose:** Main loop script - spawns fresh AI instances repeatedly until research complete

```bash
#!/bin/bash
# Oracle Ant - Deep research loop using RALF pattern
# Usage: ./oracle.sh [max_iterations]
# Based on: https://github.com/snarktank/ralph

set -e

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
AETHER_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
MAX_ITERATIONS=${1:-50}
TARGET_CONFIDENCE=95

# Files
RESEARCH_FILE="$SCRIPT_DIR/research.json"
PROGRESS_FILE="$SCRIPT_DIR/progress.md"
STOP_FILE="$SCRIPT_DIR/.stop"
ARCHIVE_DIR="$SCRIPT_DIR/archive"
DISCOVERIES_DIR="$SCRIPT_DIR/discoveries"

# Check research.json exists
if [ ! -f "$RESEARCH_FILE" ]; then
  echo "Error: No research.json found. Run /ant:oracle with a topic first."
  exit 1
fi

# Extract topic for archiving
CURRENT_TOPIC=$(jq -r '.topic // empty' "$RESEARCH_FILE" 2>/dev/null || echo "")
LAST_TOPIC_FILE="$SCRIPT_DIR/.last-topic"

# Archive previous run if topic changed
if [ -f "$LAST_TOPIC_FILE" ] && [ -f "$PROGRESS_FILE" ]; then
  LAST_TOPIC=$(cat "$LAST_TOPIC_FILE" 2>/dev/null || echo "")
  if [ -n "$CURRENT_TOPIC" ] && [ -n "$LAST_TOPIC" ] && [ "$CURRENT_TOPIC" != "$LAST_TOPIC" ]; then
    DATE=$(date +%Y-%m-%d)
    TOPIC_SLUG=$(echo "$LAST_TOPIC" | tr '[:upper:]' '[:lower:]' | sed 's/[^a-z0-9]/-/g' | sed 's/--*/-/g' | sed 's/^-\|-$//g')
    ARCHIVE_FOLDER="$ARCHIVE_DIR/$DATE-$TOPIC_SLUG"

    echo "Archiving previous research: $LAST_TOPIC"
    mkdir -p "$ARCHIVE_FOLDER"
    [ -f "$RESEARCH_FILE" ] && cp "$RESEARCH_FILE" "$ARCHIVE_FOLDER/"
    [ -f "$PROGRESS_FILE" ] && cp "$PROGRESS_FILE" "$ARCHIVE_FOLDER/"
    echo "   Archived to: $ARCHIVE_FOLDER"

    # Reset progress file
    echo "# Oracle Research Progress" > "$PROGRESS_FILE"
    echo "" >> "$PROGRESS_FILE"
  fi
fi

# Track current topic
if [ -n "$CURRENT_TOPIC" ]; then
  echo "$CURRENT_TOPIC" > "$LAST_TOPIC_FILE"
fi

# Initialize progress file if needed
if [ ! -f "$PROGRESS_FILE" ]; then
  echo "# Oracle Research Progress" > "$PROGRESS_FILE"
  echo "" >> "$PROGRESS_FILE"
fi

# Initialize discoveries directory
mkdir -p "$DISCOVERIES_DIR"

echo ""
echo "==============================================================="
echo "  ORACLE ANT - Deep Research Loop"
echo "==============================================================="
echo "Topic: $CURRENT_TOPIC"
echo "Max iterations: $MAX_ITERATIONS"
echo "Target confidence: $TARGET_CONFIDENCE%"
echo ""

# Main loop
for i in $(seq 1 $MAX_ITERATIONS); do
  # Check for stop signal
  if [ -f "$STOP_FILE" ]; then
    rm -f "$STOP_FILE"
    echo ""
    echo "Oracle stopped by user at iteration $i"
    break
  fi

  echo ""
  echo "---------------------------------------------------------------"
  echo "  Iteration $i of $MAX_ITERATIONS"
  echo "---------------------------------------------------------------"

  # Run AI with oracle.md prompt
  OUTPUT=$(claude --dangerously-skip-permissions --print < "$SCRIPT_DIR/oracle.md" 2>&1 | tee /dev/stderr) || true

  # Check for completion signal
  if echo "$OUTPUT" | grep -q "<oracle>COMPLETE</oracle>"; then
    echo ""
    echo "==============================================================="
    echo "  ORACLE RESEARCH COMPLETE!"
    echo "==============================================================="
    echo "Completed at iteration $i"
    exit 0
  fi

  echo ""
  echo "Iteration $i complete. Continuing..."
  sleep 2
done

echo ""
echo "==============================================================="
echo "  ORACLE REACHED MAX ITERATIONS"
echo "==============================================================="
echo "Max iterations ($MAX_ITERATIONS) reached without completion."
echo "Check $PROGRESS_FILE for current research status."
exit 1
```

---

## FILE 2: `.aether/oracle/oracle.md`

**Purpose:** Prompt given to AI each iteration (fresh context)

```markdown
You are an **Oracle Ant** - a deep research agent in the Aether Colony.

## Your Mission

Research a topic thoroughly and accumulate knowledge across iterations.

## Instructions

### Step 1: Read Research Topic
Read `.aether/oracle/research.json` to understand what you're researching.

### Step 2: Read Previous Progress
Read `.aether/oracle/progress.md` to see what previous iterations discovered.

### Step 3: Research
Research deeply using available tools (Glob, Grep, Read, WebFetch). Focus on filling knowledge gaps, answering unanswered questions, deepening understanding, finding patterns and connections.

### Step 4: Append Findings
APPEND to `.aether/oracle/progress.md` (never replace, always append).

### Step 5: Update Codebase Patterns
If you discovered a reusable pattern, add it to the `## Codebase Patterns` section at the TOP of progress.md.

### Step 6: Rate Confidence
Rate your overall confidence (0-100%) that the research is complete.

### Step 7: Check Completion
If confidence >= target_confidence OR all questions answered: Output `<oracle>COMPLETE</oracle>`. Otherwise end normally for another iteration.

## Important Rules
- Work on ONE focused area per iteration
- Always APPEND to progress.md, never replace
- Read previous iterations' findings before researching
- Do NOT modify any code files or colony state
- Only write to `.aether/oracle/` directory
```

---

## FILE 3: `.opencode/commands/ant/oracle.md` and `.claude/commands/ant/oracle.md`

**Purpose:** Command definition that users invoke with `/ant:oracle "topic"`

Handles: input validation, directory init, research.json creation, progress.md init, header display, loop execution, results display. Subcommands: `/oracle:stop`, `/oracle:status`.

**Non-Invasive Guarantee:** Oracle NEVER touches COLONY_STATE.json, constraints.json, activity.log, or any code files. Only writes to `.aether/oracle/`.

---

## IMPLEMENTATION ORDER

| Step | Action | Verify |
|------|--------|--------|
| 1 | Create `.aether/oracle/` directories | `ls .aether/oracle/` |
| 2 | Write `.aether/oracle/oracle.sh` | `cat .aether/oracle/oracle.sh` |
| 3 | `chmod +x .aether/oracle/oracle.sh` | `ls -la .aether/oracle/oracle.sh` |
| 4 | Write `.aether/oracle/oracle.md` | `cat .aether/oracle/oracle.md` |
| 5 | Write `.opencode/commands/ant/oracle.md` | Verify |
| 6 | Copy to `.claude/commands/ant/oracle.md` | `diff` both files |
| 7 | Test with `/ant:oracle "test topic"` | Should create research.json and run loop |

---

## REFERENCE: Ralph Repo

Source: https://github.com/snarktank/ralph

Key patterns: Each iteration = fresh AI instance. Memory via files (progress.txt, prd.json). Stop signal: `<promise>COMPLETE</promise>`. Archive on branch change. jq for JSON parsing.

---

## Priority: Future Vision - 10 Advanced Colony Implementations

*Research synthesis from multi-agent exploration of AI coding patterns, self-improving systems, memory architectures, emergent behavior, and future trends.*

---

### 1. COLONY CONSTITUTION - Self-Critique Principles

- **A written "Colony Constitution" -- a set of principles that all ants reference for self-critique before completing work.** Constitution stored in `.aether/constitution.md` with immutable principles (e.g., "No partial implementations", "All code must pass tests"). Before marking any task complete, workers run internal critique: "Does my action violate any constitutional principle?" Principles can evolve via user feedback but require explicit amendment process. Watchers verify constitutional compliance, not just functional correctness. **Inspired by:** Constitutional AI (Anthropic). **Files:** `.aether/constitution.md`, updates to `workers.md` and `build.md`.

### 2. EPISODIC MEMORY - Learning With Context

- **Store the full "story" of how patterns were discovered, not just the patterns themselves.** Every instinct includes its origin episode: phase, task, files, workers involved, what went wrong, what fixed it. Instincts become queryable: "Why does this instinct exist?" -> returns the full narrative. Episodes linked together across sessions. **Inspired by:** Letta/MemGPT episodic memory, Mem0 layered memory architecture. **Files:** Updates to `learning.md`, `.aether/data/episodes/`, instinct structure in COLONY_STATE.json.

### 3. PHEROMONE EVOLUTION - Signals That Strengthen/Decay

- **Pheromones don't just exist - they evolve based on outcomes.** Successful pheromones strengthen; unused ones fade. Each pheromone tracks: `times_applied`, `success_rate`, `last_used`. Weak pheromones (success < 50%, unused > 10 phases) auto-archive. Strong pheromones (success > 80%) auto-promote to instincts. User can "pin" pheromones to prevent decay. **Inspired by:** AlphaZero self-play reward signals, ACO pheromone evaporation. **Files:** Updates to `pheromones.md`, `aether-utils.sh`.

### 4. BOIDS COORDINATION - Three Rules for Worker Spawning

- **Flocking-style coordination for spawned workers using three simple rules.** Separation: Workers avoid files already being worked on. Alignment: Workers steer toward the colony goal. Cohesion: Workers cluster related changes. These three rules cause workers to self-organize around code regions naturally. **Inspired by:** Reynolds' Boids model. **Files:** Updates to `aether-utils.sh`, `workers.md`.

### 5. ADVERSARIAL CHAOS - Controlled Problem Injection

- **Chaos Ant intentionally introduces subtle bugs during builds to test colony resilience.** Watcher must catch injected problems. If caught: colony resilience metric increases. If missed: revealed to user as learning. Injection rate configurable (default 20%). Never injects security vulnerabilities or data loss risks. **Inspired by:** Red teaming, chaos engineering. **Files:** Updates to `chaos.md`, `build.md`.

### 6. COLONY SLEEP - Memory Consolidation During Pause

- **When the colony is paused (or after N phases), run a "sleep" consolidation process.** Cluster recent learnings, identify patterns, propose new instincts, decay unused signals, archive resolved blockers. Dreamer Ant runs during sleep. **Inspired by:** Letta "sleep-time compute", memory consolidation in neuroscience. **Files:** New `.aether/oracle/sleep.md`, updates to `continue.md`.

### 7. WORKER QUALITY SCORES - Reputation System

- **Each spawned worker earns a quality score based on output.** High-quality workers' instincts carry more weight. Score updates: +0.05 success, -0.10 failure, +0.02 exceptional. Workers below 0.3 flagged for review. Scores persisted across sessions. **Inspired by:** RLHF, reputation systems. **Files:** New `.aether/data/worker-registry.json`, updates to `build.md`.

### 8. QUORUM SENSING - Threshold-Based Commitment

- **Colony doesn't commit to an approach until enough workers signal agreement.** Spawn parallel Scouts to investigate alternatives. Quorum threshold: 2/3 must agree with confidence > 0.7. If no quorum: spawn more Scouts or surface disagreement to user. **Inspired by:** Ant quorum sensing (house hunting), consensus algorithms. **Files:** Updates to `plan.md`, `build.md`.

### 9. FEDERATED WISDOM - Cross-Colony Knowledge Sharing

- **Export/import high-confidence instincts between colonies as JSON packages.** Federation structure: `~/.aether/federation/instincts/{domain}.json`. Imported instincts tagged with `source: "federation"` at lower initial confidence (0.5). Trust scores track reliability. Privacy: colonies can mark instincts as "local-only". **Inspired by:** Transfer learning, model distillation. **Files:** New `federation.md`, `.aether/federation/` structure.

### 10. SELF-DRIVING COLONY MODE - Autonomous Building Sessions

- **Extended autonomous building sessions where Queen delegates entirely.** User activates: `/ant:self-driving --duration 4h --goal "build feature X"`. Workers in isolated git worktrees. Subplanners merge when segments complete. Can run overnight. **Inspired by:** Cursor "Self-Driving Codebases" research. **Files:** New `self-driving.md`, git worktree management.

---

### Summary: 10 Proposals by Category

| Category | Proposals |
|----------|-----------|
| **Self-Improvement** | Colony Constitution (#1), Worker Quality Scores (#7), Colony Sleep (#6) |
| **Memory & Learning** | Episodic Memory (#2), Pheromone Evolution (#3), Federated Wisdom (#9) |
| **Coordination** | Boids Coordination (#4), Quorum Sensing (#8) |
| **Resilience** | Adversarial Chaos (#5) |
| **Autonomy** | Self-Driving Colony Mode (#10) |

### Implementation Priority (Suggested)

| Priority | Proposal | Effort | Impact |
|----------|----------|--------|--------|
| **P1** | Colony Constitution | Low | High |
| **P1** | Pheromone Evolution | Medium | High |
| **P2** | Episodic Memory | Medium | High |
| **P2** | Worker Quality Scores | Medium | Medium |
| **P2** | Colony Sleep | Medium | High |
| **P3** | Boids Coordination | Medium | Medium |
| **P3** | Quorum Sensing | Medium | Medium |
| **P3** | Adversarial Chaos | Medium | Medium |
| **P4** | Federated Wisdom | High | High |
| **P4** | Self-Driving Mode | High | High |

---

## Token Efficiency Notes

All implementations should follow these principles:
- **Load summaries, not full files** - First N lines, not entire contents
- **Score before sending** - Filter irrelevant items before adding to context
- **Cap lists** - Max 3-5 items for any list (instincts, TODOs, learnings)
- **Progressive disclosure** - Show counts by default, details on demand
- **Single source of truth** - CONTEXT.md replaces reading 5 separate files
