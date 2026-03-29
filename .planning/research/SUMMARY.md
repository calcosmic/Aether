# Project Research Summary

**Project:** Aether v2.6 Bugfix & Hardening
**Domain:** Bash/jq shell script hardening -- multi-agent colony orchestration system
**Researched:** 2026-03-29
**Confidence:** HIGH

## Executive Summary

Aether v2.6 is a hardening milestone for a mature 5,200+ line bash/jq colony orchestration system. The research reveals no need for new dependencies -- all fixes use idiomatic bash (`grep -F --`, `jq --arg`) and established patterns already present in parts of the codebase. The core work is standardizing inconsistent patterns across ~30 grep call sites and ~25 JSON construction sites that evolved over 2+ years of incremental development.

The v2.6 scope breaks into four distinct work streams with clear dependencies: (1) input escaping and JSON hardening (the foundation -- everything else depends on safe variable handling), (2) cross-colony isolation fixes (LOCK_DIR mutation and QUEEN.md scoping), (3) colony depth selector (a new feature that gates Oracle/Architect spawns by effort level), and (4) YAML command generator (eliminates duplication between Claude Code and OpenCode command directories). Streams 1-2 are pure hardening with zero behavioral change for current inputs. Stream 3 is the only feature addition. Stream 4 is the highest-risk, lowest-urgency item and is a strong candidate for deferral to v2.7.

The key risk is the grep double-escape trap: the codebase has three different escaping conventions (sed-style, grep -F, and raw interpolation) used across different modules. A naive global fix would break more than it fixes. The research prescribes a per-call-site audit categorized by context (literal string vs BRE regex vs ERE regex) followed by targeted fixes using `grep -F` for all user-derived variables.

## Key Findings

### Recommended Stack

No new dependencies. All fixes use existing tools with established patterns already partially in use within the codebase.

**Core patterns to adopt:**
- `grep -F --` for all user-derived variable searches -- eliminates regex escaping entirely for literal string matching, POSIX-standard, zero maintenance
- `jq -n --arg` for all JSON string construction -- handles quotes, backslashes, newlines, control characters, and unicode correctly; replaces ~25 instances of fragile string interpolation
- `acquire_lock` with optional `--lock-dir` parameter -- replaces fragile global `LOCK_DIR` save/restore pattern in hive.sh with explicit scope passing
- ShellCheck 0.11.0 at `warning` severity -- currently only 6 of 29+ scripts are linted at `error` severity; expanding coverage catches the unescaped variable class of bugs

**Version requirements:** bash 3.2+ (macOS default), jq 1.6+ (currently 1.8.1), shellcheck 0.10+ (currently 0.11.0). All met.

### Expected Features

**Must have (table stakes -- the core depth selector):**
- Named depth levels matching existing plan.md convention (`minimal`/`standard`/`deep`/`full`) -- consistent UX
- Default to `standard` (current behavior minus Oracle/Architect) -- saves 2 opus-tier spawns per build by default
- Per-build CLI flag override (`--depth <level>`) -- primary use case: "this one phase is simple"
- Spawn plan header reflects active depth -- visibility into what runs
- Colony-level depth preference in COLONY_STATE.json -- set once with `/ant:depth`, use always
- No behavior change for existing colonies that don't use the flag

**Should have (competitive -- depth selector polish):**
- Phase-type auto-suggestion -- Queen analyzes phase name and suggests depth
- Continue flow respects build depth -- Gatekeeper/Auditor always run (safety nets), Probe gated by depth
- Autopilot depth consistency -- `/ant:run` uses same depth across all phases
- Cost/time estimate per depth level -- "Standard: ~5 spawns. Full: ~8 spawns (+3 opus-tier)"

**Defer (v2+):**
- YAML command generator (v2.7 candidate) -- highest risk, lowest urgency; partial generation is worse than none
- Per-caste toggles -- explodes API surface; named depth levels are the right abstraction
- Auto-detect optimal depth -- misclassification is costly; suggestion-only is the right balance

### Architecture Approach

The codebase operates across two storage scopes: **colony-local** (`.aether/` per repo) and **hub-shared** (`~/.aether/` across all colonies). The v2.6 fixes enforce this two-scope model by eliminating three architectural bugs that undermine cross-colony isolation.

**Major components affected:**

1. **file-lock.sh** -- Add optional `lock_dir_override` parameter to `acquire_lock` so hive.sh can lock hub resources without mutating the global `LOCK_DIR`. This is the linchpin fix for cross-colony isolation.
2. **hive.sh** -- Replace 6 separate LOCK_DIR save/restore blocks with a `_hive_acquire_lock` helper that passes the lock directory as a parameter. Create `~/.aether/hive/locks/` as a dedicated lock directory (currently locks sit alongside data files).
3. **spawn.sh / spawn-tree.sh / swarm.sh** -- Convert 10 grep calls with `ant_name` interpolation from regex mode to `grep -F` (fixed-string). Add input validation at spawn-tree entry point as defense in depth.
4. **aether-utils.sh / error-handler.sh** -- Migrate ~25 `json_ok`/`json_err` call sites from string interpolation to `jq -n --arg`. Refactor `json_err` to use jq internally for correct escaping of all error fields.
5. **queen.sh** -- Add `--scope local|hub` parameter to `queen-promote` so high-confidence wisdom can be written to the hub QUEEN.md (currently always writes local).

**Key patterns to follow:**
- Parameterized lock acquisition (pass scope as argument, never mutate globals)
- Fixed-string grep for user data (`grep -F --`)
- jq-native JSON construction (never string interpolation)
- Explicit scope for cross-colony writes (colony-local vs hub)

### Critical Pitfalls

1. **The Double-Escape Trap** -- The codebase has three grep escaping conventions (sed-style, grep -F, raw interpolation) used across different modules. Adding a global escape function without per-call-site audit will double-escape strings already escaped upstream (e.g., learning.sh line 1248 sed-escaping followed by grep). Prevention: audit and categorize every grep call before writing any fix. Prefer `grep -F` which eliminates escaping entirely for literal searches.

2. **JSON Construction With printf** -- At least 6 call sites construct JSON by interpolating bash variables into printf format strings. If the variable contains quotes, backslashes, or newlines, the JSON becomes invalid and downstream jq parsing fails silently. Prevention: migrate all user-derived variables to `jq -n --arg`. Do NOT change the `json_ok` function signature -- fix the callers.

3. **LOCK_DIR Mutation Leak** -- hive.sh temporarily mutates the global `LOCK_DIR` for hub-level locking. If a signal handler, trap, or concurrent operation fires between save and restore, colony-local locks land in the hub directory, breaking mutual exclusion. Prevention: add `lock_dir_override` parameter to `acquire_lock` instead of mutating a global. Use subshell isolation for any remaining save/restore patterns.

4. **Depth Selector Breaks Build Assumptions** -- Adding a depth selector that reduces spawn capacity can break build playbooks that assume depth 1 always allows 4 spawns. Oracle's RALF loop spawns sub-agents at depth 2 that would be blocked in `minimal` mode. Prevention: "standard" mode must match current hardcoded limits exactly. Swarm spawns should be exempt from depth limits. Oracle needs special handling to allow at least 1 sub-agent at depth 2.

5. **YAML Generator Synchronization Debt** -- Generating only a subset of commands is worse than generating none (developers must remember which commands are YAML-sourced and which are hand-maintained). Prevention: generate ALL 44 commands or defer entirely. Add generated-file headers. Consider deferring to v2.7.

## Implications for Roadmap

Based on research, suggested phase structure:

### Phase 1: Input Escaping & JSON Hardening
**Rationale:** This is the foundation. Every subsequent phase touches grep patterns or JSON construction. Getting escaping right first prevents cascading breakage. Zero behavioral change for current inputs -- purely defensive.
**Delivers:** Safe grep patterns across all modules, safe JSON construction across all helpers, expanded shellcheck coverage, bash 3.2 compatibility fixes
**Addresses:** FEATURES table stakes T6 (backward compatibility -- no behavior change), STACK patterns 1 and 2
**Avoids:** Pitfalls 1 (double-escape), 2 (JSON injection), 6 (awk regex injection), 7 (timing.log prefix match), 8 (bash 3.2 `((depth++))`), 10 (missing `--` separator), 11 (changelog JSON concatenation), 13 (jq -Rs trailing newline), 12 (duplicated depth logic)

**Concrete work items:**
- Audit and categorize all ~30 grep call sites with variable interpolation (literal vs BRE vs ERE)
- Convert user-derived grep calls to `grep -F --`
- Migrate ~25 `json_ok` call sites from string interpolation to `jq -n --arg`
- Refactor `json_err`/`json_warn` to use jq internally
- Add `.shellcheckrc`, expand lint targets to all utils/*.sh files, lower severity to `warning`
- Fix `((depth++))` to `$((depth + 1))` in spawn-tree.sh
- Consolidate duplicated depth logic between spawn.sh and spawn-tree.sh

### Phase 2: Cross-Colony Isolation
**Rationale:** Depends on Phase 1 because the grep fixes touch spawn.sh and spawn-tree.sh which the isolation work also modifies. Isolation fixes must come before the depth selector because the depth selector changes spawn behavior that isolation correctness depends on.
**Delivers:** Parameterized lock acquisition, hive.sh LOCK_DIR refactor, QUEEN.md scope parameter, validate-package.sh content integrity checks, CLAUDE.md documentation correction
**Addresses:** STACK pattern 3 (LOCK_DIR parameter), ARCHITECTURE bugs 1-2 (LOCK_DIR mutation, QUEEN.md scoping), ARCHITECTURE bug 4 (atomic-write contract documentation)
**Avoids:** Pitfall 3 (LOCK_DIR leak), Pitfall 9 (validate-package.sh content gaps)

**Concrete work items:**
- Add `lock_dir_override` parameter to `acquire_lock` in file-lock.sh
- Create `_hive_acquire_lock` helper, replace all 6 save/restore blocks in hive.sh
- Create `~/.aether/hive/locks/` directory
- Add `--scope local|hub` to `queen-promote` in queen.sh
- Add content integrity checks to validate-package.sh (function existence, shell syntax, JSON parse)
- Fix CLAUDE.md LOCK_DIR path documentation (`.aether/locks/` not `.aether/data/locks/`)
- Add CONTRACT comment to atomic-write.sh

### Phase 3: Colony Depth Selector
**Rationale:** Feature addition that depends on safe grep/JSON handling (Phase 1) and correct spawn isolation (Phase 2). This is the only user-facing feature change in v2.6.
**Delivers:** Four depth levels (minimal/standard/deep/full), `--depth` CLI flag, `/ant:depth` preference command, spawn plan header updates, build summary depth field, one-time migration notice
**Addresses:** FEATURES all P1 items (T1-T6, D1), ARCHITECTURE pattern for spawn gating
**Avoids:** Pitfall 4 (depth selector breaks build assumptions), Pitfall 8 (bash 3.2 compatibility)

**Concrete work items:**
- Define depth-to-caste mapping table (minimal/standard/deep/full)
- Add `--depth` argument parsing to build.md and build-prep.md
- Gate Oracle spawn on depth >= deep in build-wave.md Step 5.0.1
- Gate Architect spawn on depth >= deep in build-wave.md Step 5.0.2
- Gate Chaos spawn on depth >= standard in build-wave.md Step 5.1
- Add `build_depth` field to COLONY_STATE.json schema
- Create `/ant:depth` command for colony-level preference
- Update spawn plan header and BUILD SUMMARY to reflect depth
- Add one-time migration notice for existing colonies
- Pass depth through to `/ant:run` autopilot
- Mirror all changes to OpenCode commands

### Phase 4: YAML Command Generator (Conditional)
**Rationale:** Lowest urgency, highest risk. Should only proceed if Phases 1-3 complete with capacity to spare. Deferral to v2.7 is a valid and likely outcome.
**Delivers:** YAML source-of-truth for all 44 commands, generator script, generated-file headers, CI check for staleness
**Addresses:** FEATURES deferred item (YAML generator)
**Avoids:** Pitfall 5 (synchronization debt)

**Concrete work items (only if proceeding):**
- Design YAML schema for command definitions
- Create `.aether/commands/` source directory
- Build generator that produces both `.claude/commands/ant/` and `.opencode/commands/ant/`
- Generate ALL 44 commands (partial generation is unacceptable)
- Add generated-file headers (`<!-- GENERATED: do not edit manually -->`)
- Add `npm run generate:commands` script
- Update validate-package.sh for new file structure
- Decide: commit generated files or gitignore + postinstall hook

### Phase Ordering Rationale

- Phase 1 first because escaping fixes are the foundation -- every subsequent phase touches the same grep/JSON patterns, and getting escaping wrong in Phase 1 would cascade into Phase 2-3 changes
- Phase 2 second because isolation fixes modify file-lock.sh and hive.sh (the locking infrastructure that the depth selector depends on for correct spawn behavior)
- Phase 3 third because it is the feature addition that builds on hardened infrastructure -- it modifies spawn.sh, build playbooks, and COLONY_STATE.json, all of which are touched by earlier phases
- Phase 4 last (or deferred) because it is orthogonal to the hardening work and has the highest risk of introducing synchronization problems

### Research Flags

Phases likely needing deeper research during planning:
- **Phase 3:** The depth selector has nuanced interactions with Oracle's RALF sub-agent spawning, swarm's independent cap system, and the continue flow's Gatekeeper/Auditor/Probe agents. The boundary between "build effort" and "post-build safety" needs design decisions during phase planning.
- **Phase 4:** YAML generator has an unresolved decision on whether to commit generated files or gitignore them. Each approach has tradeoffs that need stakeholder input.

Phases with standard patterns (skip research-phase):
- **Phase 1:** Well-documented bash hardening patterns. The research has already identified every call site that needs fixing with specific before/after code.
- **Phase 2:** Lock parameterization and scope-based writes are established architectural patterns. All code locations are identified.

## Confidence Assessment

| Area | Confidence | Notes |
|------|------------|-------|
| Stack | HIGH | All recommendations are standard bash/jq idioms already partially in use. No new dependencies. Verified tool versions on target machine. |
| Features | HIGH | Depth selector design is grounded in direct analysis of existing build playbooks, plan.md precedent, and all 8 caste definitions. Edge cases thoroughly explored. |
| Architecture | HIGH | All four architectural bugs identified with specific line numbers, concrete failure modes, and tested fix patterns. Direct codebase inspection of all affected files. |
| Pitfalls | HIGH | 13 pitfalls identified from direct codebase inspection. Each has concrete evidence (file, line number), failure mode description, and prevention strategy. macOS bash 3.2 compatibility matrix included. |

**Overall confidence:** HIGH

### Gaps to Address

- **bash version constraint ambiguity:** PROJECT.md states "bash 4+" but CLAUDE.md and emoji-audit.sh reference macOS bash 3.2 compatibility. The research assumed bash 3.2 is required. This should be explicitly resolved during Phase 1 planning -- if bash 4+ is truly required, several fixes become simpler (associative arrays, `mapfile`, etc.).

- **"standard" default user impact:** The depth selector's default (`standard` = current behavior minus Oracle/Architect) is a behavioral change for existing users. Research identified a one-time migration notice as mitigation, but real-world validation is needed to confirm this is sufficient.

- **YAML generator file strategy:** Whether to commit generated command files or gitignore them is unresolved. Committing creates large PR diffs; gitignoring adds a required setup step after cloning. This decision should be made before Phase 4 planning begins.

- **Web search rate limiting:** Stack and pitfalls research note that web search was rate-limited, so some findings are based on training data rather than live sources. This is low-risk because the recommendations are standard shell scripting practices that have not fundamentally changed, but it means no current-year edge cases were discovered from external sources.

## Sources

### Primary (HIGH confidence)
- Direct codebase inspection of `aether-utils.sh` (5,200+ lines, 40+ json_ok sites, 30+ grep instances)
- Direct codebase inspection of all 10 domain modules (spawn.sh, spawn-tree.sh, hive.sh, queen.sh, learning.sh, file-lock.sh, atomic-write.sh, state-api.sh, swarm.sh, suggest.sh)
- Direct codebase inspection of all build playbooks (build-prep, build-wave, build-verify, build-complete, continue-verify, continue-gates)
- Direct codebase inspection of agent definitions (workers.md), command definitions (plan.md depth levels), and COLONY_STATE.json schema
- Verified tool versions: bash 3.2.57, jq 1.8.1, shellcheck 0.11.0 on target macOS machine
- PROJECT.md v2.6 scope definition (lines 43-55)

### Secondary (MEDIUM confidence)
- ShellCheck documentation on SC2086 (double quote to prevent globbing) and SC2061 (quote grep pattern) -- training data
- `jq --arg` safety properties and `grep -F` POSIX specification -- well-established, verified on target
- CLI UX best practices for verbosity/depth controls (-v/-vv, named levels) -- established domain knowledge
- Existing plan.md depth system (--fast/--balanced/--deep/--exhaustive) as design precedent

### Tertiary (LOW confidence)
- Multi-agent orchestration tool comparisons (CrewAI, AutoGen, LangGraph) -- web search rate-limited
- Whether "standard" default will satisfy most users -- needs real-world validation
- macOS bash 3.2 vs bash 4 edge case differences -- well-documented but not verified against all v2.6 code paths

---
*Research completed: 2026-03-29*
*Ready for roadmap: yes*
