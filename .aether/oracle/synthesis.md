# Research Synthesis

## Topic
Aether Ant production readiness audit — full system architecture review for reliability, autonomy, verification, memory, safety, and operator experience.

## Findings by Question

### Q1: What are the main components and their responsibilities? (answered, 85%)

**1. aether-utils.sh — Core Utility Layer** [S1]
- 11,272-line monolithic bash script with ~125+ subcommands
- All deterministic colony operations: state management, pheromone I/O, learning/instinct pipeline, hive brain, midden tracking, session management, colony-prime context assembly, XML exchange, skills matching, autopilot, suggestions, data-clean
- JSON output to stdout, non-zero exit with JSON errors to stderr
- Sources 9 utility modules at startup: file-lock, atomic-write, error-handler, chamber-utils, xml-utils, semantic-cli, hive, midden, skills
- Uses `set -euo pipefail` with structured ERR trap

**2. CLI (bin/cli.js + bin/lib/) — Installation & Hub Management** [S2]
- Node.js CLI providing the `aether` command (npm global install)
- 16 library modules: errors, logger, update-transaction, init, state-sync, model-profiles, proxy-health, nestmate-loader, spawn-logger, telemetry, colors, caste-colors, event-types, file-lock, state-guard, model-verify
- Handles: npm install → hub setup, `aether update` transactions, model profile management, state synchronization
- Dependencies: commander, js-yaml, picocolors

**3. Slash Commands (.claude/commands/ant/) — User Interface** [S3]
- 43 markdown command definitions interpreted by Claude Code
- Categories: setup (lay-eggs, init, colonize, plan), signals (focus, redirect, feedback), status (status, phase, flags, history, watch), session (pause-colony, resume-colony, resume), lifecycle (patrol, seal, entomb, maturity, update), advanced (swarm, oracle, dream, interpret, chaos, archaeology, organize, council, skill-create), data (data-clean, export-signals, import-signals)
- High-length commands split into execution playbooks (.aether/docs/command-playbooks/)

**4. Agent Definitions (.claude/agents/ant/) — 22 Worker Types** [S4]
- 3 tiers: Core (builder, watcher), Orchestration (queen, scout, route-setter), Specialist/Niche (17 agents)
- Core agents: builder (TDD implementation), watcher (testing/validation)
- 4 surveyors: nest, disciplines, pathogens, provisions
- Quality gates: auditor, gatekeeper, probe, measurer
- Specialists: keeper, tracker, weaver, chaos, archaeologist, includer, sage, ambassador, chronicler

**5. Skills System (.aether/skills/) — Reusable Behavior Modules** [S5]
- 10 colony skills (behavioral patterns): build-discipline, colony-interaction, colony-lifecycle, colony-visuals, context-management, error-presentation, pheromone-protocol, pheromone-visibility, state-safety, worker-priming
- 18 domain skills (technical knowledge): django, docker, golang, graphql, html-css, nextjs, nodejs, postgresql, prisma, python, rails, react, rest-api, svelte, tailwind, testing, typescript, vue
- Matching via skill-match: worker role + pheromone signals + codebase detect patterns
- Own 12K character budget independent of colony-prime token budget

**6. Pheromone System — Signal-Based Communication** [S6]
- Three signal types: FOCUS (attract attention), REDIRECT (hard constraint), FEEDBACK (gentle adjustment)
- Stored in .aether/data/pheromones.json
- Colony-prime injects active signals into worker prompts
- Content dedup via SHA-256 hashing, prompt injection sanitization (XML/shell/LLM override rejection), 500 char cap
- Automatic suggestions via suggest-analyze during builds

**7. Colony State (.aether/data/) — Persistent State** [S7]
- COLONY_STATE.json: goal, phases, tasks, instincts, events
- Session files: session.json for cross-conversation recovery
- Supporting: pheromones.json, constraints.json, midden/, learning-observations.json

**8. Hub (~/.aether/) — Cross-Colony User-Level** [S8]
- QUEEN.md: wisdom + user preferences
- hive/wisdom.json: 200-entry cap cross-colony wisdom with LRU eviction
- eternal/: legacy high-value signal storage (fallback)
- registry/: colony tracking with domain tags

**9. Utility Scripts (.aether/utils/) — Extracted Modules** [S9]
- 22 scripts sourced by aether-utils.sh or run standalone
- Infrastructure: file-lock.sh, atomic-write.sh, error-handler.sh
- Domain: hive.sh, midden.sh, skills.sh, oracle/oracle.sh
- Visualization: swarm-display.sh, spawn-tree.sh, watch-spawn-tree.sh, colorize-log.sh
- XML: xml-core.sh, xml-compose.sh, xml-convert.sh, xml-query.sh, xml-utils.sh
- Other: chamber-utils.sh, chamber-compare.sh, semantic-cli.sh, state-loader.sh, spawn-with-model.sh

**10. Exchange System (.aether/exchange/) — XML Import/Export** [S10]
- pheromone-xml.sh, wisdom-xml.sh, registry-xml.sh
- Enables cross-colony signal sharing via XML serialization
- Colony archive export for entombed colonies

**11. Templates (.aether/templates/) — 12 Initialization Templates** [S11]
- colony-state.template.json, pheromones.template.json, constraints.template.json
- session.template.json, midden.template.json, learning-observations.template.json
- crowned-anthill.template.md, handoff templates, QUEEN.md.template
- JQ template for state reset

**12. Documentation (.aether/docs/) — Internal Docs** [S12]
- Command playbooks (9 split files for build and continue orchestration)
- System docs: caste-system.md, pheromones.md, context-continuity.md, error-codes.md
- Operational: known-issues.md, source-of-truth-map.md, xml-utilities.md
- Reference: queen-commands.md, QUEEN-SYSTEM.md

**13. Workers.md — Worker Role Definitions** [S13]
- Named ant personality system with caste-specific communication styles
- Spawn tracking protocol (spawn-log, spawn-complete)
- Model selection documentation (session-level, LiteLLM proxy support)
- Honest execution model definitions

**14. CLI vs. aether-utils.sh Responsibility Boundary** [S71, S2, S1]
- CLI (bin/cli.js + 18 bin/lib/ modules) handles plumbing: installation/hub distribution (install, update with transactional rollback via UpdateTransaction, uninstall, checkpoint), model profile CRUD (caste-models list/set/reset/verify via model-profiles.js), telemetry (spawn-log, spawn-tree, performance via telemetry.js + spawn-logger.js), and state synchronization (state-sync.js, state-guard.js)
- CLI NEVER invokes aether-utils.sh directly — slash commands (.claude/commands/ant/) bridge the two by calling `bash .aether/aether-utils.sh <subcommand>`
- Bash layer (178 subcommands across 16 categories) handles all domain logic: colony operations, pheromone signals, learning/wisdom pipeline, worker lifecycle, context assembly, skills matching, autopilot, data maintenance
- Only cross-link: CLI copies aether-utils.sh to hub during `npm install`
- No functional overlaps found — CLI owns packaging/distribution/config, bash owns colony runtime logic
- Both have independent file-lock implementations (bin/lib/file-lock.js vs utils/file-lock.sh) and error hierarchies

**15. Colony-Prime Orchestration Flow** [S72, S29, S1]
- Colony-prime is a read-only aggregator (never mutates state) assembling 9 sections from 8 data sources in order:
  1. QUEEN.md wisdom — extracts 6 sections (Philosophies, Patterns, Redirects, Stack Wisdom, Decrees, User Preferences) from global (~/.aether/QUEEN.md) + local (.aether/QUEEN.md), local extends global; FAILS HARD with E_FILE_NOT_FOUND if neither exists
  2. Pheromone-prime (delegated subprocess) — reads pheromones.json with type-specific decay (FOCUS 30d, REDIRECT 60d, FEEDBACK 90d), deactivates below 0.1 strength; extracts instincts from COLONY_STATE.json (confidence >= 0.5, not disproven); warns but continues if pheromones.json missing
  3. Hive-read — domain-scoped retrieval using registry.json tags, confidence threshold, sorted by confidence then validated_count; updates LRU access tracking on read; falls back to eternal memory
  4. Context-capsule (subprocess) — 220-word bounded state snapshot
  5. Phase-learnings — validated learnings from COLONY_STATE.json, grouped by phase, only from previous phases
  6. Key-decisions — parsed from CONTEXT.md Recent Decisions table (last 5 normal, 3 compact)
  7. Blocker warnings — from flags.json, unresolved blockers for current phase (NEVER trimmed)
  8. Rolling-summary — last 5 entries from 15-entry bounded log
- Budget enforcement: 8K normal / 4K compact characters. Trim order (first removed): rolling-summary → phase-learnings → key-decisions → hive-wisdom → context-capsule → user-prefs → queen-wisdom → pheromone-signals. REDIRECT signals preserved even when pheromone section trimmed
- Output: JSON with `prompt_section` (formatted markdown for worker injection), `log_line` (single-line status), metadata, signal/instinct counts
- Per-wave refresh during builds ensures workers get latest signals — not just one-time at phase start

**16. Subcommand Criticality Tiers** [S73, S1, S3]
- 178 total subcommands, only 43 (24%) are critical path for build/continue lifecycle
- Top 7 subcommands account for ~50% of all invocations across slash commands and playbooks: activity-log (49 calls), print-next-up (38), spawn-log (37), midden-write (34), normalize-args (33), generate-ant-name (32), context-update (31)
- These top 7 are single points of failure — a bug in any one cascades across the entire system
- 76 subcommands (43%) are never invoked by any command or playbook — dead code categories include: semantic search engine (6), swarm display/timing (10), view state management (6), learning display/selection (8), spawning diagnostics (5), suggest advanced (4), error/security advanced (5)
- Niche commands (dream, archaeology, chaos, organize, council) use only ~5-10 core subcommands plus isolated utilities
- Removing unused categories could reduce aether-utils.sh by 15-20% without breaking any active functionality

**17. Agent Interaction Model** [S4, S15, S73]
- Strict one-way coupling with 4 phases: (1) Playbooks spawn agents via Task tool with subagent_type. (2) Colony-prime prompt_section injected into agent prompts alongside archaeology context, integration plans, midden context, and skill sections. (3) Agents explicitly warned not to modify .aether/data/ or aether-utils.sh — they never call bash subcommands directly. (4) Results flow back as JSON to orchestrating slash command, which handles all state persistence
- Worker prompt composition (per build-wave.md): archaeology_context + integration_plan + grave_context + midden_context + colony-prime prompt_section + skill_section
- This is the healthiest coupling pattern in the system — agents are stateless workers with injected context, cleanly separated from state management

**18. SYNTHESIS: Resolved Unknowns** [S1, S71, S73]
- Dead-code indirect callers: aether-utils.sh uses a case statement for subcommand dispatch (not eval/dynamic dispatch), so the 76 dead subcommands have no indirect invocation mechanism. Grep across all .md files covers all static callers.
- CLI/bash hub race: CLI writes hub only during npm install/update [S71], bash during colony operations [S1]. Temporally disjoint, no concurrent access path in normal workflow.
- No actionable unknowns remain from static analysis.

### Q2: What are the dependency relationships between components, and where are coupling risks? (answered, 82%)

**1. COLONY_STATE.json is the central coupling nexus** [S14, S1]
- 219 direct references across 38 of 43 slash commands
- 153 references inside aether-utils.sh itself
- Dual access pattern: slash commands read/write via direct jq calls AND via aether-utils.sh subcommands
- This creates two parallel paths to the same state file — potential consistency risk if a slash command's inline jq and an aether-utils.sh subcommand race or apply different schemas

**2. Utility module source chain — silent degradation** [S1, S16]
- aether-utils.sh sources 9 modules at startup with conditional `[[ -f ]] && source` guards
- Dependency chain: file-lock.sh ← atomic-write.sh; xml-core.sh ← xml-query.sh, xml-convert.sh, xml-compose.sh ← xml-utils.sh ← exchange modules
- If any module is missing, the system silently degrades — functions become undefined, but `set -euo pipefail` only catches usage errors at call time, not at source time
- Fallback patterns exist for some functions (e.g., `atomic_write` has inline fallback), but not all

**3. CLI (Node.js) and aether-utils.sh (Bash) are near-parallel systems** [S2, S1]
- CLI handles hub installation, updates, model profiles in Node.js
- Slash commands handle all colony operations via bash subprocess calls to aether-utils.sh
- Only cross-link: CLI copies aether-utils.sh to hub during `npm install`
- Both systems have their own file-lock implementations (bin/lib/file-lock.js vs utils/file-lock.sh)
- Both systems have their own error hierarchies
- Risk: divergent state handling between two runtimes that share the same data files

**4. Agent coupling is one-way and well-bounded** [S4, S15]
- Agents are spawned by playbooks via Task tool with subagent_type
- Colony-prime assembles context → injected into agent prompts
- Agents explicitly warned not to modify aether-utils.sh or .aether/data/
- Agent results flow back to the orchestrating slash command, which handles state persistence
- This is the healthiest coupling pattern in the system

**5. Complete build/continue state mutation flow** [S44, S45, S15, S46, S47, S48, S49, S50]
- Traced through 9 playbooks (build-prep, build-context, build-wave, build-verify, build-complete, continue-verify, continue-gates, continue-advance, continue-finalize)
- State mutation sequence:
  1. **build-prep:** COLONY_STATE.json state→"EXECUTING" (with bash file lock)
  2. **build-context:** Reads COLONY_STATE.json (NO lock held) for colony-prime context + pheromone display
  3. **build-wave:** Spawns parallel builders → each writes to activity.log (no lock), spawn-tree.txt (no lock), midden.json (lock with fallback), pheromones.json (locked)
  4. **build-complete:** Writes HANDOFF.md, last-build-result.json, CONTEXT.md, session.json
  5. **continue-verify:** Reads COLONY_STATE.json (load-state acquires lock → unload-state releases)
  6. **continue-gates:** 7 mandatory gates (all read-only checks)
  7. **continue-advance:** Writes COLONY_STATE.json (LLM Write tool — NO bash-level lock), auto-emits pheromones (locked)
  8. **continue-finalize:** Writes HANDOFF.md, CHANGELOG.md, session.json, CONTEXT.md

**6. Concurrent write protection analysis across 5 shared files** [S54, S52, S53, S55, S51]
- **pheromones.json:** FULLY PROTECTED — acquire_lock at line 7044 before read-modify-write
- **midden.json:** PARTIALLY PROTECTED — lock attempted, but on failure falls through to lockless write (midden.sh lines 62-75). Two concurrent lockless writes both write to `midden.json.tmp` → same temp file → race condition where one entry is silently lost
- **activity.log:** UNPROTECTED — `echo >> append` relies on POSIX atomic append guarantee (safe for writes < PIPE_BUF, typically 4096 bytes; long task summaries could interleave)
- **spawn-tree.txt:** UNPROTECTED — same `echo >> append` pattern as activity.log
- **COLONY_STATE.json:** Mixed — bash subcommands (state-advance, spawn-complete) use acquire_lock; but the LLM Write tool call in continue-advance Step 2 holds no bash lock

**7. Midden graceful degradation creates a specific data loss race** [S52]
- When midden-write can't acquire the lock (contention during parallel builder failures), both lock-holding and lockless code paths use: `jq ... > midden.json.tmp && mv midden.json.tmp midden.json`
- All processes write to the SAME temp path (`$mw_midden_file.tmp`)
- Under simultaneous lockless writes: Process A writes .tmp → Process B overwrites .tmp → Process B mv's → Process A mv's (file gone) or loses its entry
- Impact: silent loss of failure records during high-concurrency failure scenarios

**8. Continue-advance has a latent COLONY_STATE.json race window** [S51, S55, S49]
- continue-verify acquires and releases the state lock (load-state → unload-state)
- continue-advance then instructs the LLM to "Write COLONY_STATE.json" via the Write tool — NOT a bash command, so no bash-level lock is held
- If a slow builder's spawn-complete fires between unload and the LLM's Write (spawn-complete acquires its own lock for failed-spawn event logging), the LLM's full-file overwrite destroys the spawn-complete event entry
- Mitigated in practice by sequential build→continue flow, but the window exists for long-running builder agents

**9. SYNTHESIS: Operational ceiling reached** [S62, S53, S1]
- Cross-validated with Q3 midden evidence — 17 real failures [S62] confirm coupling risks materialized (lock isolation, unbound variables)
- POSIX append safety for activity.log/spawn-tree.txt effectively resolved: spawn-log entries are short (<100 bytes), well under PIPE_BUF (4096) [S53]
- Remaining unknowns (concurrent write frequency, collision rate) are operational metrics requiring runtime monitoring — code analysis ceiling reached

### Q3: Where are the risk areas — execution reliability gaps, silent failures, fake autonomy, hallucination vectors, and verification weaknesses? (answered, 82%)

**1. Silent error suppression at industrial scale** [S1, S28]
- aether-utils.sh has ~338 instances of error-swallowing patterns (`2>/dev/null || true`, `2>/dev/null || echo`, etc.)
- The ERR trap is completely disabled for the suggest-analyze subcommand (lines 10236-10427), creating a ~200-line window where any error silently continues
- Combined with the conditional module sourcing, missing modules, failed writes, or corrupted data can propagate without detection

**2. Worker response validation is schema-only, not semantic** [S17, S20]
- `validate-worker-response` (line 2305) checks JSON structure — required fields exist and have correct types (string, boolean, number, array)
- It CANNOT detect a Builder that claims `status: completed` when tests actually failed, or a Watcher that reports `verification_passed: true` without actually running verification commands
- The gap between structural validation and truth validation is the system's primary hallucination vector

**3. Verification chain is prompt-enforced, not programmatically enforced** [S18, S19, S26, S27]
- Queen has "Verification Discipline Iron Law", Watcher has "Evidence Iron Law", Scout has "Never fabricate findings"
- These are instructions to LLM agents via markdown prompts — no hard programmatic check that tests were actually executed, build commands actually ran, or files reported as created actually exist
- The entire trust model rests on LLM compliance with prompt instructions
- However, the prompt-level discipline is thorough: Watcher requires 4 execution verification steps (syntax, imports, launch, tests), has a quality score ceiling rule, and requires fresh evidence per verification

**4. Agent fallback degrades dramatically** [S25, S4]
- Multiple playbooks contain FALLBACK comments: "If Agent type not found, use general-purpose and inject role"
- When fallback triggers, the rich agent definition (execution flow, critical rules, pheromone protocol, boundary declarations — often 200+ lines) is replaced with a single sentence role description
- A general-purpose agent with "You are a Chaos Ant - resilience tester" has none of the discipline, output format, or boundary constraints of the full aether-chaos.md definition

**5. Anti-pattern detection (check-antipattern) covers only 6 patterns** [S21]
- Detected: didSet recursion (Swift), `any` type (TS/JS), console.log (TS/JS), bare except (Python), exposed secrets (all langs), TODO/FIXME (all langs)
- Not detected: SQL injection, XSS, command injection, SSRF, unsafe deserialization, CORS misconfig, or any OWASP top 10 beyond hardcoded secrets
- The "Gatekeeper security gate" described in CLAUDE.md relies partly on this thin detection layer

**6. Dual file-lock implementations risk cross-runtime race** [S22, S23]
- Bash uses noclobber (`set -C; echo PID > lockfile`), Node.js uses `fs.openSync` with `'wx'` flag
- Both share same naming convention (basename.lock + .pid sidecar) and directory
- Under simultaneous access from both runtimes, the lock protocols may not provide mutual exclusion because noclobber and `'wx'` might not be atomic with respect to each other on all filesystems
- Mitigated by temporal separation: CLI operates only during install/update [S71], bash during colony operations [S1] — not concurrent in normal workflow

**7. String-typed confidence causes silent hive-read filtering** [S24]
- `hive-read` uses `jq --argjson min_conf` for confidence filtering
- If any hive wisdom entry has confidence stored as a string (`"0.8"` instead of `0.8`), the jq comparison `.confidence >= $min_conf` silently excludes it (jq string-to-number comparison returns false)
- The REDIRECT signal confirms this has occurred in practice
- Fix requires type coercion in the jq filter (`tonumber?`)

**8. Memory pipeline is universally fire-and-forget** [S56, S57, S58, S59, S68, S69]
- The `memory-capture` function (aether-utils.sh lines 5547-5648) orchestrates a 5-step pipeline: (1) learning-observe → learning-observations.json, (2) pheromone-write → pheromones.json, (3) learning-promote-auto → QUEEN.md + instincts, (4) activity-log → activity.log, (5) rolling-summary → rolling-summary.log
- EVERY caller of memory-capture across all playbooks uses `2>/dev/null || true`: build-wave.md line 383, build-verify.md lines 330/346/386, build-complete.md line 58, continue-advance.md line 66. This means the entire memory pipeline is fire-and-forget — any failure results in silent learning loss
- CRITICALLY: The pipeline is sequential with a kill-switch at step 1 — corrupted learning-observations.json triggers json_err, killing ALL 5 downstream steps (Q4 finding 12 corrected the earlier "independent per-step" characterization)
- The pipeline detects corruption (jq validation at line 5319) but does NOT recover (no file reset, no fallback write) — detection without remediation
- Three-layer error silence: callers suppress memory-capture, memory-capture suppresses sub-steps, sub-steps suppress internals — making diagnosis impossible

**9. Autopilot has no mechanism for subtly bad work or dual-state desync** [S38, S60, S61]
- The autopilot (run.md) has 10 pause conditions, but ALL are binary pass/fail gates. Subtly incorrect work that passes tests and gates contaminates the codebase
- The Execution Contract states "Hard failure (state corruption) = halt immediately, no recovery attempt" — but there is NO detection for "subtly bad but gate-passing work"
- Autopilot tracks state in run-state.json (separate from COLONY_STATE.json). If autopilot-update succeeds but COLONY_STATE.json write fails, the two files desync with no reconciliation mechanism

**10. Historical evidence: 17 midden entries confirm 5+ risks materialized** [S62]
- The midden contains 17 real failure entries from colony operations:
  - **Lock isolation (confirmed):** "Chaos finding: Cross-repo lock isolation mismatch (high)" — validates finding #6
  - **String-typed confidence (confirmed):** "Chaos finding: String-typed confidence causes silent hive-read failure (medium)" — validates finding #7
  - **Unbound variable crashes (3 instances):** validates finding #1 about error suppression masking real bugs
  - **Unsanitized output:** validation gap where user-controlled category values flow into JSON output without escaping
  - **Fix ratio trend:** Sage analytics found fix ratio rising from 33.8% to 45.8% — error surface growing faster than repairs

**11. SYNTHESIS: Risk assessment complete** [S4, S62, S71, S1]
- 11 risk areas documented with deep findings; all characterized with severity
- 9/10 have dedicated Q5 recommendations; 2 low-priority risks mitigated by design (temporal separation for dual locks, documentation fix for check-antipattern scope)
- Remaining unknowns (suggest-analyze ERR trap failure rate, worker fallback %, json_ok injection frequency) are operational metrics at static analysis ceiling

### Q4: How does the memory and context management system handle state persistence, session resume, context rot, and cross-colony knowledge sharing? (answered, 82%)

**1. Dual-file state persistence** [S32, S33]
- session.json: lightweight session metadata (session_id, colony_goal, baseline_commit, last_command, suggested_next, last_command_at, started_at)
- COLONY_STATE.json: authoritative state (goal, state, current_phase, plan with phases array, memory with decisions/phase_learnings/instincts, events)
- Resume reads both files, but COLONY_STATE.json is always authoritative — session.json provides supplementary data for drift detection and session tracking
- If COLONY_STATE.json is missing or corrupted, resume blocks and asks user to start fresh or recover — no silent fallback

**2. Three-tier session resume** [S33, S34]
- Brief (action commands: /build, /plan, /continue): 1-line "Resuming: Phase X - Name" — minimal orientation
- Extended (/status): Brief + last activity timestamp for temporal context
- Full (/resume-colony): Complete header, goal, state, session ID, phase, active pheromones with strength bars, survey context, phase progress, HANDOFF.md context summary, next action routing
- This tiered approach prevents over-loading simple commands with recovery overhead

**3. Colony-prime budget enforcement with 8-level trimming** [S29]
- Character budget: 8K normal, 4K compact
- 9 sections assembled: queen-wisdom, user-prefs, hive-wisdom, context-capsule, phase-learnings, key-decisions, blockers, rolling-summary, pheromone-signals
- Trim order (first removed → last): rolling-summary → phase-learnings → key-decisions → hive-wisdom → context-capsule → user-prefs → queen-wisdom → pheromone-signals
- Blockers are NEVER trimmed (REDIRECT-priority). Pheromone REDIRECTs preserved even when signals section is trimmed.
- CRITICAL GAP: No feedback mechanism when trimming occurs — workers receive silently truncated context with no awareness of what was removed
- NOTE: CLAUDE.md documentation says "Rolling summary (highest priority — never trimmed first)" but code trims it FIRST — documentation is inverted

**4. Rolling summary — bounded narrative log** [S30, S29]
- Stored in .aether/data/rolling-summary.log, pipe-delimited (timestamp|event|source|summary)
- Bounded to 15 entries max (tail -n 15 after each write)
- Summary text capped at 180 chars, events at 24 chars
- Auto-recorded via memory-capture (line 5646 in aether-utils.sh)
- Colony-prime injects last 5 entries as "RECENT ACTIVITY (Colony Narrative)"
- First section trimmed under budget pressure — workers lose narrative continuity silently

**5. Context capsule — compact state snapshot** [S31, S29]
- Generates max 220-word snapshot from COLONY_STATE.json: goal, state, current phase, recent decisions (last 3), blocker flags, pheromone summaries, next action
- 5th priority in trimming order — survives after rolling-summary, learnings, decisions, and hive-wisdom are trimmed
- Serves as "low-token continuity" — compact enough to survive aggressive trimming but provides basic orientation

**6. Hive brain cross-colony retrieval chain** [S29, S36, S35]
- Flow: registry.json (domain tags for current repo) → hive-read (filtered by domain + confidence threshold) → fallback to eternal memory (~/.aether/eternal/memory.json)
- hive-read: sorts by confidence then validated_count, returns up to 5 entries (3 in compact)
- Access tracking: every read increments access_count and updates last_accessed, feeding LRU eviction
- Domain-scoped: wisdom only surfaces if domain_tags overlap with current repo's tags — prevents cross-domain pollution
- Hub-level locking: hive operations temporarily mutate LOCK_DIR to ~/.aether/hive/ for cross-repo mutual exclusion

**7. Hive-store deduplication and confidence boosting** [S35, S36, S24]
- Content hash (SHA-256 of sanitized text) for deduplication
- Same-repo re-promotion: skipped (no duplicate entries)
- Cross-repo match: merge with validated_count increment + multi-repo confidence tier (2 repos = 0.70, 3 = 0.85, 4+ = 0.95)
- Confidence uses max() — never downgraded
- 200-entry cap with LRU eviction (oldest by last_accessed)
- Known bug: string-typed confidence causes silent exclusion from hive-read (REDIRECT signal)

**8. Staleness and drift detection** [S32, S33]
- Staleness: session-read checks if last_command_at > 24 hours, sets is_stale flag
- Drift: resume compares baseline_commit with current HEAD, counts commits and changed files
- Both are informational only — never block restoration
- Deliberate design choice: "Restore identically regardless of time elapsed" — availability over caution

**9. CONTEXT.md — comprehensive crash-recovery document with 11 action types** [S63, S64, S65, S66]
- Managed by `_cmd_context_update` (lines 242-590) with lock protection (LOCK-04 via acquire_lock)
- 11 actions: init, update-phase, activity, constraint, decision, safe-to-clear, build-start, worker-spawn, worker-complete, build-progress, build-complete
- The `init` action creates a full template with 10 sections: System Status, Current Goal, What's In Progress, Active Constraints (REDIRECT), Active Pheromones (FOCUS), Recent Decisions, Recent Activity, Next Steps, If Context Collapses (recovery guide), Colony Health
- Updated at 4 lifecycle points: (1) init.md → context-update init, (2) build-complete.md → activity + build-complete, (3) continue-advance.md → reads Recent Decisions for pheromone auto-emission, (4) continue-finalize.md → activity + update-phase + decision
- The `decision` action (lines 480-518) auto-emits a FEEDBACK pheromone (`[decision] <text>`, source `auto:decision`, strength 0.6) at write time
- Worker spawn tracking: build-start, worker-spawn, worker-complete actions provide per-worker visibility during builds

**10. CONTEXT.md as decision-to-pheromone bridge** [S66, S63]
- Continue-advance Step 2.1b (lines 194-233) extracts decisions from CONTEXT.md's "Recent Decisions" table using awk
- Auto-emits FEEDBACK pheromones for up to 3 decisions per continue run
- Deduplication via jq `.contains()` check against existing `auto:decision` or `system:decision` pheromones
- Potential double-emit path: `context-update decision` emits one pheromone at write time, then continue-advance reads same decision from CONTEXT.md → but dedup check catches it because both paths use identical format
- The `2>/dev/null || true` on the pheromone-write call means failed auto-emission is silent

**11. Context capsule resilience to corrupted/missing state** [S67]
- Missing COLONY_STATE.json: returns `{exists:false, prompt_section:"", word_count:0}` immediately (line 9371-9373) — clean early exit
- Corrupted COLONY_STATE.json: each jq field extraction uses `2>/dev/null || echo <default>` — degrades to defaults (goal → "No goal set", state → "IDLE", current_phase → 0, total_phases → 0)
- Missing auxiliary files (flags.json, pheromones.json, rolling-summary.log): each produces empty sections, no crash
- Compact mode: progressively strips "Recent narrative" then "Open risks" sections to stay under word budget
- Verdict: one of the most defensive functions in the codebase — will never crash colony-prime regardless of state file condition

**12. Memory-capture pipeline: sequential with hard checkpoint at step 1** [S68, S69, S56]
- learning-observe (step 1) validates learning-observations.json structure at line 5319: `jq -e . "$observations_file"`
- If corrupted → `json_err "$E_JSON_INVALID"` (non-zero exit) → memory-capture checks at line 5576-5578 and exits with its own json_err
- This kills ALL remaining steps: pheromone-write (step 2), learning-promote-auto (step 3), activity-log (step 4), rolling-summary (step 5) — none execute
- Callers wrap with `2>/dev/null || true` → total pipeline death is invisible to the orchestrator
- Correction to Q3 finding #8: the pipeline is not "independently fire-and-forget per step" — it's a sequential chain where step 1 failure is a kill switch for all downstream memory operations
- learning-observe DETECTS corruption (jq validation) but does NOT RECOVER (no file reset, no fallback write) — detection without remediation

**13. Budget trimming visibility gap: information exists but not routed** [S70, S29]
- When colony-prime trims sections, it records which sections in `cp_budget_trimmed_list` (lines 8405-8485)
- Appended to `cp_log_line`: e.g., "truncated: rolling-summary,phase-learnings (budget: 8000)"
- This log line appears in JSON output as the `log_line` field — visible to the orchestrating slash command
- However, the `prompt_section` field (what workers see) does NOT include trimming information
- The gap is not "missing data" but "missing routing" — the orchestrator knows, the workers don't

**14. SYNTHESIS: Memory system complete** [S63, S67, S29]
- Memory system fully traced from input to output across all 13 findings
- Remaining unknowns (budget trimming frequency, memory-capture corruption frequency) are operational metrics requiring runtime data
- Context capsule resilience (finding 11) is one of the strongest defense patterns, confirmed independently by Q1 colony-prime analysis and Q3 risk assessment
- All contradictions resolved: decision pheromone double-emission caught by dedup, staleness-without-action is deliberate design, memory-capture is sequential not independent

### Q5: What would an expert change about this architecture? (answered, 80%)

**RECOMMENDATION 1: Per-phase COLONY_STATE.json checkpointing** [S38, S39, S43, S42]

The state-safety skill prescribes "Before making significant state changes, create a backup" but this is only implemented during entomb (colony archival). Normal builds, continues, and autopilot runs do NOT create COLONY_STATE.json backups. The autopilot explicitly states "Hard failure (state corruption) = halt immediately, no recovery attempt." A mid-autopilot corruption is total loss.

**Fix:** Add `cp COLONY_STATE.json COLONY_STATE.json.phase-N.bak` before each build-wave. Near-zero cost, provides rollback capability. Keep last 3 phase backups to bound disk usage.

**RECOMMENDATION 2: Extend the evidence trail to close semantic verification gaps** [S41, S17, S20, S38]

The system already has 8 excellent programmatic gates (spawn gate, watcher gate, TDD evidence gate, anti-pattern gate, flags gate, security gate, quality gate, runtime verification gate). The gap is between structural validation and truth validation — a Builder can claim "completed" while tests actually fail.

**Fix:** Two targeted additions following the existing TDD evidence gate pattern:
- (a) Capture test runner exit code during build-verify and cross-reference against Watcher's `verification_passed` claim. Exit code != 0 but verification_passed == true → flag as fabricated.
- (b) Verify Builder's `files_created` list against actual filesystem. If claimed files don't exist, reject the response.

These follow the pattern already proven in Step 1.10 (TDD evidence gate in continue-gates.md).

**RECOMMENDATION 3: Triage error suppression instances** [S1, S28]

Not all ~338 error suppression instances are bad. Categorize into: (a) Correct suppression (optional/fallback paths — keep), (b) Lazy suppression (hiding real errors — fix), (c) Dangerous suppression (data-writing operations — critical fix).

**Priority targets:**
- The suggest-analyze ERR trap gap (lines 10236-10427) — 200 lines without error trapping during builds
- Any `2>/dev/null` on `atomic_write` or state mutation calls
- The conditional module sourcing pattern (missing module → silently undefined functions)

**RECOMMENDATION 4: Type coercion in hive jq filters** [S24, S36, S35]

The string-typed confidence bug silently excludes valid wisdom from hive-read results, causing colonies to re-learn patterns they should already know.

**Fix:** Add `(tonumber? // 0)` before numeric comparisons in hive-read's jq filter. Audit all `jq --argjson` usage across aether-utils.sh for the same pattern. This is a ~5-line fix with outsized reliability impact on cross-colony knowledge transfer.

**RECOMMENDATION 5: Context trimming notification** [S29, S30]

When colony-prime trims sections to fit the token budget, workers receive silently truncated context with no awareness of what was removed.

**Fix:** Add a single line to trimmed output: `NOTICE: Context trimmed. Sections removed: {list}. Query colony-prime for full context if needed.` Cost: ~30 characters. Also fix the CLAUDE.md documentation inversion — it claims "Rolling summary (highest priority — never trimmed first)" but the code trims it first.

**RECOMMENDATION 6: Agent fallback degradation logging** [S25, S4]

When a specialized agent falls back to general-purpose, the 200+ lines of discipline, output format, and boundary constraints are replaced with a single sentence. This happens silently.

**Fix:** (a) Log fallback as a midden entry, (b) include WARNING in build synthesis, (c) optionally emit a FEEDBACK pheromone. This gives the operator visibility into when the system is running at reduced capability.

**RECOMMENDATION 7: Git checkpoint branches for autopilot** [S38, S42, S1]

The autopilot chains build→verify→continue→advance with no rollback mechanism. Subtly bad work that passes all gates contaminates subsequent phases.

**Fix:** Before each autopilot build, create a lightweight git tag (e.g., `aether/pre-phase-3`). Combined with Rec 1 (state backup), this provides full state + code rollback. The existing `autofix-checkpoint` subcommand (line 2409) already implements git stashing for Aether files — extend it to per-phase checkpointing.

**RECOMMENDATION 8: Memory pipeline circuit breaker with file recovery** [S56, S68, S69]

The memory-capture pipeline (Q4 finding 12) has a sequential kill-switch at step 1: corrupted learning-observations.json blocks ALL 5 downstream steps silently. The pipeline detects corruption (jq validation at line 5319) but doesn't recover. Combined with 3-layer error suppression (Q3 finding 8), this creates an invisible failure mode where a single corrupted file permanently disables the colony's learning capability.

**Fix:** Add a recovery path to learning-observe: if `jq -e .` fails on the observations file, reset it to the template (`learning-observations.template.json`), log a midden entry for the corruption event, and retry. Cost: ~15 lines. This transforms a permanent silent failure into a recoverable event with an audit trail. This is the only recommendation that addresses a confirmed "detection without remediation" gap.

**RECOMMENDATION 9: Autopilot run-state.json ↔ COLONY_STATE.json reconciliation** [S60, S42, S38]

Autopilot tracks state in run-state.json separately from COLONY_STATE.json (Q3 finding 9). If autopilot-update succeeds but the LLM Write tool call for COLONY_STATE.json fails, the two files desync with no detection. Resume reads COLONY_STATE.json as authoritative but doesn't check run-state.json.

**Fix:** Add a reconciliation check at autopilot loop start: compare `run-state.json.current_phase` with `COLONY_STATE.json.current_phase`. If they disagree, pause and present both values to the operator. Cost: ~10 lines in the autopilot loop. Combined with Rec 1 (state backup) and Rec 7 (git checkpoints), this provides detection + rollback for the desync scenario.

### Cross-Question Synthesis: Priority, Feasibility, and Interdependency Analysis

**Recommendation Priority Matrix** (cross-referenced with findings across Q1-Q4):

| Rec | Impact | Effort | Urgency | Depends On |
|-----|--------|--------|---------|------------|
| 4 (type coercion) | High | ~5 lines | CONFIRMED BUG — REDIRECT signal + midden evidence [S62, S24] | None |
| 1 (state backup) | High | ~3 lines per call site | Prevents total loss in autopilot [S38, S43] | None |
| 8 (memory circuit breaker) | High | ~15 lines | Silent permanent failure [S68, S69] | None |
| 5 (trim notification) | Medium | ~30 chars added | Workers operate blind to context loss [S70, S29] | None |
| 2 (evidence trail) | High | ~20 lines per check | Primary hallucination vector [S17, S20] | None |
| 9 (state reconciliation) | Medium | ~10 lines | Prevents desync cascade [S60, S42] | Rec 1 |
| 6 (fallback logging) | Low-Medium | ~10 lines | Silent degradation [S25] | None |
| 7 (git checkpoints) | Medium | ~5 lines | No code rollback exists [S38, S42] | Rec 1 |
| 3 (error triage) | High (long-term) | Large (audit 338 instances) | Growing error surface [S62] | None |

**Key insight:** Recs 1, 4, 5, and 8 are all independent, low-effort, high-impact fixes that can be done in parallel. Recs 7 and 9 depend on Rec 1. Rec 3 is the largest effort but addresses the root cause behind multiple other findings.

**Feasibility validation from cross-question evidence:**
- Rec 1 is trivially feasible: Q2 finding 5 [S44] shows build-prep already acquires file lock and writes to COLONY_STATE.json — adding `cp` before the write is one line
- Rec 2 follows proven patterns: Q3 finding 3 [S41] confirms the TDD evidence gate already checks test file existence — extending to exit code and file existence checks is the same pattern
- Rec 4 is a confirmed 5-line fix: Q3 finding 7 + Q4 finding 7 [S24, S36] pinpoint the exact jq filter line
- Rec 7 reuses existing infrastructure: Q1 finding 16 [S1] confirms autofix-checkpoint already implements git stashing for Aether files

## Cross-Question Analysis

### Pattern 1: Documentation Accuracy Problem

Six instances where system labels don't match behavior, identified across Q1-Q4:

1. **"Rolling summary highest priority"** (Q4 finding 3) [S29]: CLAUDE.md says "never trimmed first" — code trims it FIRST. **Resolution: code is authoritative.** Documentation must be corrected.
2. **"Graceful degradation"** for midden (Q2 finding 7) [S52]: Label masks a data loss race condition (shared temp file path). **Resolution: rename to "lockless fallback" in code comments.**
3. **"Fire-and-forget per step"** (Q3 finding 8 vs Q4 finding 12) [S56, S68]: Pipeline is sequential with hard checkpoint at step 1. **Resolution: Q4's deeper analysis is correct.** Memory-capture is a sequential kill-switch, not independent fire-and-forget.
4. **"Security gate"** (Q3 finding 5) [S21]: check-antipattern covers 6 patterns, Gatekeeper is an LLM prompt. **Resolution: label oversells.** The gate provides basic detection, not comprehensive security scanning.
5. **"125 subcommands"** (Q1 finding 16) [S73]: Actual count is 178, with 76 (43%) dead code. **Resolution: CLAUDE.md should state 178 total, 102 actively used.**
6. **"State-safety skill prescribes backups"** (gaps.md) [S39, S43]: Skill says backup before changes, but only entomb implements it. **Resolution: Q5 Rec 1 addresses this gap.**

**Cross-question pattern:** The system's documentation and naming consistently describes aspirational behavior rather than implemented behavior. This creates a trust gap for operators who rely on documentation to understand system guarantees.

### Pattern 2: COLONY_STATE.json Vulnerability Chain

Three separate question domains converge on the same core vulnerability:

- **Q2 finding 1** [S14]: 219 references, dual-access pattern (jq + subcommands)
- **Q2 finding 8** [S51, S55, S49]: continue-advance writes via LLM Write tool with no bash lock
- **Q3 finding 9** [S38, S60]: autopilot dual-state desync between run-state.json and COLONY_STATE.json

These are facets of one problem: COLONY_STATE.json is the highest-value file with the most inconsistent protection. Q5 Recs 1, 7, and 9 form a layered defense: backup before mutation (Rec 1), code rollback via git (Rec 7), and desync detection (Rec 9).

### Pattern 3: Three-Layer Error Silence

Q3 findings 1, 8, 10 and Q4 finding 12 reveal a systematic error suppression pattern:

- **Layer 1 (callers):** `2>/dev/null || true` on memory-capture, pheromone operations [S57, S58]
- **Layer 2 (orchestrator):** memory-capture checks only step 1 exit, suppresses internal failures [S56]
- **Layer 3 (functions):** Individual functions use `|| true` on sub-operations [S59]

The midden evidence (Q3 finding 10) [S62] proves this causes real bugs: 5+ risks have materialized, the fix ratio is rising (33.8% → 45.8%), indicating the error surface grows faster than repairs. Q5 Rec 3 (error triage) addresses the root cause, while Rec 8 (memory circuit breaker) provides targeted recovery for the most impactful pipeline.

### Pattern 4: Healthy Architecture Strengths

Not all findings are negative. Cross-question evidence reveals strong design patterns:

- **Agent isolation** (Q1 finding 17, Q2 finding 4) [S4, S15]: One-way coupling, stateless workers, explicit boundary declarations — the healthiest coupling in the system
- **Context capsule resilience** (Q4 finding 11) [S67]: Cascading defaults handle every failure mode — will never crash colony-prime
- **Pheromone deduplication** (Q4 finding 10) [S66, S63]: Double-emission path exists but dedup catches it — working as designed
- **CLI/bash boundary** (Q1 finding 14) [S71, S2, S1]: Clean separation with no functional overlaps
- **Tiered session resume** (Q4 finding 2) [S33, S34]: Proportional recovery prevents over-loading simple commands
- **Budget enforcement with REDIRECT preservation** (Q4 finding 3) [S29]: Even under aggressive trimming, critical constraints survive

These strengths indicate the architecture is sound at the macro level. The risks are concentrated in specific implementation details (error handling, state protection, documentation accuracy) rather than fundamental design flaws.

### Pattern 5: Operational Ceiling — Static Analysis Limits

Cross-referencing all 5 questions reveals a consistent pattern: each question's remaining gaps require runtime data that code analysis cannot provide. This represents an **analytical ceiling**, not an evidence gap:

| Question | Remaining Unknown | Why Code Analysis Can't Resolve |
|----------|------------------|---------------------------------|
| Q2 | Concurrent write frequency | Build parallelism bounded by wave size (2-4 builders). Write pattern fully characterized [S44-S55]. Collision rate is runtime-only. |
| Q3 | suggest-analyze ERR trap failure rate | Disabled ERR trap [S28] creates theoretical risk. Actual frequency is runtime-only. |
| Q3 | Worker spawn fallback % | All 22 agents registered in Claude Code [S4]. Fallback triggers only in non-standard environments. |
| Q4 | Token budget trimming frequency | Depends on colony state size (instincts, pheromones, wisdom) — varies per colony. |
| Q5 | Recommendation effectiveness | Requires testing fixes in practice — can't be validated from code analysis. |

**Implication for confidence scoring:** Questions with only operational unknowns remaining are scored at 80-85% (good understanding with limitations known). The gap to 95%+ requires deployment monitoring, not further code analysis.

### Pattern 6: Recommendation Coverage

Cross-referencing the 9 recommendations against Q3's 11 risk findings:

- **9 risks have dedicated recommendations** (Recs 1-9)
- **2 risks mitigated by existing design:**
  1. Dual file-lock (Q3 finding 6): temporal separation [S71, S1] — CLI during install, bash during operations
  2. check-antipattern scope (Q3 finding 5): documentation accuracy issue, recommend renaming + external scanners

All critical and medium risks are covered. The recommendation set is complete.

## Resolved Contradictions (9 total)

1. **Decision pheromone double-emission**: Both paths emit identical format; dedup catches it via `.contains()` check [S66]. No actual duplicate.
2. **"Evidence before claims" vs schema-only validation**: Prompt-level discipline and schema validation serve different purposes — structural minimum vs aspirational behavior [S17, S18]. Gap is real but intentional.
3. **Staleness detected but never acted upon**: Deliberate design — "Restore identically regardless of time elapsed" [S33]. Detection serves informational display only.
4. **Memory-capture "fire-and-forget" vs sequential kill-switch**: Definitively resolved by Q4 finding 12 — step 1 failure kills all 5 downstream steps [S68, S69]. Pipeline is sequential, not independent.
5. **Dead-code indirect callers via eval/dynamic dispatch**: Resolved — aether-utils.sh uses case-statement dispatch, not eval [S1]. The 76 dead subcommands have no indirect invocation mechanism. Grep analysis [S73] covers all static callers.
6. **CLI/bash hub file race**: Resolved — CLI writes hub only during npm install/update [S71], bash during colony operations [S1]. Temporally disjoint, no concurrent access path.
7. **Dual file-lock implementations**: Resolved — cross-runtime mutual exclusion is theoretically unverified, but CLI and bash are temporally separated [S71, S1]. Not concurrent in normal workflow. If code fix needed: mkdir-based locking is atomic across all runtimes.
8. **"Security gate" vs actual detection scope**: Resolved — documentation accuracy issue (Pattern 1). check-antipattern covers 6 patterns [S21], Gatekeeper is an LLM agent. Label oversells; appropriate action is documentation correction, not code fix.
9. **State-safety skill prescribes backups that aren't implemented**: Resolved — confirmed gap between skill aspiration and implementation [S39, S43]. Q5 Rec 1 provides the fix. Gap is documented and addressed.

## Sources
- [S1] `.aether/aether-utils.sh` — Core utility script (codebase, 2026-03-23)
- [S2] `bin/cli.js` + `bin/lib/` — CLI entry point and library modules (codebase, 2026-03-23)
- [S3] `.claude/commands/ant/` — 43 slash command definitions (codebase, 2026-03-23)
- [S4] `.claude/agents/ant/` — 22 agent definition files (codebase, 2026-03-23)
- [S5] `.aether/skills/` — Skills directory with colony/ and domain/ subdirs (codebase, 2026-03-23)
- [S6] CLAUDE.md — Pheromone System section (codebase, 2026-03-23)
- [S7] `.aether/data/COLONY_STATE.json` — Colony state file (codebase, 2026-03-23)
- [S8] CLAUDE.md — Hub and Hive Brain sections (codebase, 2026-03-23)
- [S9] `.aether/utils/` — 22 utility scripts (codebase, 2026-03-23)
- [S10] `.aether/exchange/` — XML exchange modules (codebase, 2026-03-23)
- [S11] `.aether/templates/` — 12 template files (codebase, 2026-03-23)
- [S12] `.aether/docs/` — Documentation directory (codebase, 2026-03-23)
- [S13] `.aether/workers.md` — Worker role definitions (codebase, 2026-03-23)
- [S14] `.claude/commands/ant/` — COLONY_STATE.json reference count analysis (codebase, 2026-03-23)
- [S15] `.aether/docs/command-playbooks/build-wave.md` — Build wave orchestration playbook (codebase, 2026-03-23)
- [S16] `.aether/utils/` — Utility module source dependency chain analysis (codebase, 2026-03-23)
- [S17] `.aether/aether-utils.sh` (validate-worker-response, lines 2305-2400) — Schema-only validation (codebase, 2026-03-23)
- [S18] `.claude/agents/ant/aether-watcher.md` — Watcher agent: Evidence Iron Law (codebase, 2026-03-23)
- [S19] `.claude/agents/ant/aether-queen.md` — Queen agent: Verification Discipline Iron Law (codebase, 2026-03-23)
- [S20] `.aether/docs/command-playbooks/build-verify.md` — Watcher spawn and result processing (codebase, 2026-03-23)
- [S21] `.aether/aether-utils.sh` (check-antipattern, lines 1797-1860) — 6-pattern detection (codebase, 2026-03-23)
- [S22] `.aether/utils/file-lock.sh` — Bash PID-based locking with noclobber (codebase, 2026-03-23)
- [S23] `bin/lib/file-lock.js` — Node.js PID-based locking with exclusive open (codebase, 2026-03-23)
- [S24] `.aether/utils/hive.sh` (hive-read, lines 240-320) — Confidence filtering via argjson (codebase, 2026-03-23)
- [S25] `.aether/docs/command-playbooks/` — Agent FALLBACK comments across 4 playbooks (codebase, 2026-03-23)
- [S26] `.claude/agents/ant/aether-scout.md` — Anti-fabrication rules and source tracking (codebase, 2026-03-23)
- [S27] `.claude/agents/ant/aether-measurer.md` — Honest measurement vs fabrication rules (codebase, 2026-03-23)
- [S28] `.aether/aether-utils.sh` (suggest-analyze, lines 10235-10427) — ERR trap disabled region (codebase, 2026-03-23)
- [S29] `.aether/aether-utils.sh` (colony-prime, lines 7869-8460) — Context assembly with budget trimming (codebase, 2026-03-23)
- [S30] `.aether/aether-utils.sh` (rolling-summary, lines 9274-9332) — Bounded 15-entry log (codebase, 2026-03-23)
- [S31] `.aether/aether-utils.sh` (context-capsule, lines 9334-9400+) — 220-word compact snapshot (codebase, 2026-03-23)
- [S32] `.aether/aether-utils.sh` (session-read, lines 9640-9668) — 24-hour staleness detection (codebase, 2026-03-23)
- [S33] `.claude/commands/ant/resume.md` — 10-step session restore with drift detection (codebase, 2026-03-23)
- [S34] `.claude/commands/ant/resume-colony.md` — Full state restoration (codebase, 2026-03-23)
- [S35] `.aether/utils/hive.sh` (hive-store, lines 63-238) — Dedup, cross-repo merge, LRU cap (codebase, 2026-03-23)
- [S36] `.aether/utils/hive.sh` (hive-read, lines 240-375) — Domain-scoped filtering, access tracking (codebase, 2026-03-23)
- [S37] `.aether/utils/hive.sh` (hive-abstract, lines 377-399) — Text transformation (codebase, 2026-03-23)
- [S38] `.claude/commands/ant/run.md` — Autopilot: 10 pause conditions (codebase, 2026-03-23)
- [S39] `.aether/skills/colony/state-safety/SKILL.md` — State safety skill (codebase, 2026-03-23)
- [S40] `.aether/docs/error-codes.md` — 11 error codes with recovery (codebase, 2026-03-23)
- [S41] `.aether/docs/command-playbooks/continue-gates.md` — 7 mandatory gates (codebase, 2026-03-23)
- [S42] `.aether/aether-utils.sh` (autopilot-update, lines 11067-11124) — run-state.json tracking (codebase, 2026-03-23)
- [S43] `.claude/commands/ant/entomb.md` — Only COLONY_STATE.json backup location (codebase, 2026-03-23)
- [S44] `.aether/docs/command-playbooks/build-prep.md` — Build prep with locking (codebase, 2026-03-23)
- [S45] `.aether/docs/command-playbooks/build-context.md` — Colony-prime assembly (codebase, 2026-03-23)
- [S46] `.aether/docs/command-playbooks/build-complete.md` — Synthesis and handoff (codebase, 2026-03-23)
- [S47] `.aether/docs/command-playbooks/continue-verify.md` — 6-phase verification loop (codebase, 2026-03-23)
- [S48] `.aether/docs/command-playbooks/continue-gates.md` — 7 mandatory gates (codebase, 2026-03-23)
- [S49] `.aether/docs/command-playbooks/continue-advance.md` — State update and pheromone emission (codebase, 2026-03-23)
- [S50] `.aether/docs/command-playbooks/continue-finalize.md` — Handoff, changelog, session (codebase, 2026-03-23)
- [S51] `.aether/utils/state-loader.sh` — Load/unload with lock management (codebase, 2026-03-23)
- [S52] `.aether/utils/midden.sh` — Lock-protected with lockless fallback (codebase, 2026-03-23)
- [S53] `.aether/aether-utils.sh` (spawn-log, lines 1517-1537) — Append-only, no locking (codebase, 2026-03-23)
- [S54] `.aether/aether-utils.sh` (pheromone-write, lines 6919-7078) — Lock-protected, SHA-256 dedup (codebase, 2026-03-23)
- [S55] `.aether/aether-utils.sh` (spawn-complete, lines 1538-1573) — Locked write on failure (codebase, 2026-03-23)
- [S56] `.aether/aether-utils.sh` (memory-capture, lines 5547-5648) — 5-step pipeline (codebase, 2026-03-23)
- [S57] `.aether/docs/command-playbooks/build-wave.md` (line 379-383) — Fire-and-forget caller (codebase, 2026-03-23)
- [S58] `.aether/docs/command-playbooks/continue-advance.md` (line 66) — Fire-and-forget caller (codebase, 2026-03-23)
- [S59] `.aether/aether-utils.sh` (learning-promote-auto, lines 5469-5544) — Internal || true on instinct (codebase, 2026-03-23)
- [S60] `.aether/aether-utils.sh` (autopilot-update, lines 11067-11124) — Separate state file (codebase, 2026-03-23)
- [S61] `.claude/commands/ant/run.md` — Execution Contract: 10 binary pause conditions (codebase, 2026-03-23)
- [S62] `.aether/data/midden/midden.json` — 17 real failure entries (codebase, 2026-03-23)
- [S63] `.aether/aether-utils.sh` (_cmd_context_update, lines 242-590) — CONTEXT.md management (codebase, 2026-03-23)
- [S64] `.aether/docs/command-playbooks/build-complete.md` (lines 239-243) — CONTEXT.md update (codebase, 2026-03-23)
- [S65] `.aether/docs/command-playbooks/continue-finalize.md` (lines 253-271) — CONTEXT.md update (codebase, 2026-03-23)
- [S66] `.aether/docs/command-playbooks/continue-advance.md` (lines 194-233) — Decision-to-pheromone bridge (codebase, 2026-03-23)
- [S67] `.aether/aether-utils.sh` (context-capsule, lines 9334-9527) — Cascading defaults (codebase, 2026-03-23)
- [S68] `.aether/aether-utils.sh` (memory-capture, lines 5575-5578) — Step 1 kill-switch (codebase, 2026-03-23)
- [S69] `.aether/aether-utils.sh` (learning-observe, lines 5319-5321) — JSON validation (codebase, 2026-03-23)
- [S70] `.aether/aether-utils.sh` (colony-prime budget, lines 8490-8493) — Trimming log gap (codebase, 2026-03-23)
- [S71] `bin/cli.js` + `bin/lib/` (18 modules) — CLI responsibility analysis (codebase, 2026-03-23)
- [S72] `.aether/aether-utils.sh` (colony-prime + pheromone-prime) — Complete orchestration flow (codebase, 2026-03-23)
- [S73] `.claude/commands/ant/*.md` + `.aether/docs/command-playbooks/*.md` — Subcommand criticality analysis (codebase, 2026-03-23)

## Last Updated
Iteration 11 — Final synthesis pass: resolved all 9 contradictions (3 newly resolved: dual file-lock via temporal separation, security gate label as documentation issue, state-safety backup gap addressed by Rec 1), marked all questions as answered at analytical ceiling, confirmed 73 sources with 100% attribution coverage, 6 cross-question patterns validated
