# Colony-Prime CI Context Assembly Design

> Task 1.4: Design the pr-context subcommand for CI agent consumption
> Author: Weld-53 (Builder)
> Date: 2026-03-30
> Depends on: Task 1.1 (Branch-Local State Contract), Task 1.2 (Pheromone Propagation)
> Verified against: pheromone.sh (_colony_prime lines 737-1553), aether-utils.sh (context-capsule lines 4172-4368)

---

## 1. Problem Statement

In the PR-based workflow, each colony task creates its own branch with its own
worktree. Agents spawned on these branches need colony context (wisdom, signals,
learnings, blockers) to do their work. Today, colony-prime assembles this
context interactively -- it is called by the build orchestrator in the CLI
session, produces a `prompt_section` string, and that string is injected into
the worker agent's prompt.

The CI workflow changes the consumption model. When a PR is submitted and an
automated agent (Watcher, Gatekeeper, Auditor, Probe, Measurer) runs as a CI
check, it needs the same colony context but:

1. **No interactive session exists.** The CI agent cannot call `colony-prime`
   interactively -- it needs a machine-readable JSON output it can parse.
2. **Sources may be incomplete.** The branch-local state on a PR branch may not
   have all the files that a main-branch build would have (no session.json, no
   rolling-summary.log from previous phases).
3. **Caching matters.** Some context sources (QUEEN.md, hive wisdom) are
   cacheable across CI runs. Others (pheromones, flags) are volatile and must
   be read fresh each run.
4. **Token budget is tighter in CI.** CI agents have limited context windows.
   A `--compact` mode with aggressive trimming is needed.

This document defines the `pr-context` subcommand that produces structured JSON
for CI agent consumption, with explicit caching semantics and a defined fallback
chain for missing sources.

---

## 2. Current System Reference

### 2.1 colony-prime (pheromone.sh lines 737-1553)

colony-prime assembles worker context from 9 sections:

| # | Section | Source | Cacheable? |
|---|---------|--------|------------|
| 1 | QUEEN wisdom (global) | `~/.aether/QUEEN.md` | YES (changes rarely) |
| 2 | QUEEN wisdom (local) | `.aether/QUEEN.md` | YES (changes rarely) |
| 3 | User preferences | QUEEN.md sections | YES (changes rarely) |
| 4 | Hive wisdom | `~/.aether/hive/wisdom.json` | YES (changes rarely) |
| 5 | Context capsule | `context-capsule` subcommand | NO (branch-local state) |
| 6 | Phase learnings | `COLONY_STATE.json` | NO (branch-local) |
| 7 | Key decisions | `.aether/CONTEXT.md` | NO (branch-local) |
| 8 | Blockers | `flags.json` | NO (branch-local) |
| 9 | Rolling summary | `rolling-summary.log` | NO (branch-local) |
| 10 | Pheromone signals | `pheromones.json` | NO (branch-local) |

Budget: 8,000 chars normal / 4,000 chars compact.

Trim order (first removed = lowest retention priority):
1. rolling-summary
2. phase-learnings
3. key-decisions
4. hive-wisdom
5. context-capsule
6. user-prefs
7. queen-wisdom-global
8. queen-wisdom-local
9. pheromone-signals (REDIRECTs preserved even when section trimmed)
10. Blockers: NEVER trimmed

### 2.2 context-capsule (aether-utils.sh lines 4172-4368)

Reads COLONY_STATE.json, flags.json, pheromones.json, rolling-summary.log.
Produces a bounded snapshot (220 words max in compact mode) with goal, state,
phase, next_action, signals, decisions, risks, recent narrative.

### 2.3 Existing Error Handling

colony-prime has two error tiers:
- **HARD FAIL:** No QUEEN.md found anywhere (line 924-929) -- exits with error
- **SOFT FAIL:** pheromones.json missing (line 973-986) -- warns but continues
- **SOFT FAIL:** context-capsule returns empty (line 1177) -- continues with empty
- **SOFT FAIL:** hive-read returns nothing (line 1106-1117) -- falls back to eternal memory

---

## 3. Design: pr-context Subcommand

### 3.1 Interface

```
Usage: pr-context [--compact] [--json] [--branch <name>] [--ci-run-id <id>]

Flags:
  --compact       Tighter token budget (3,000 chars vs 6,000 default)
  --json          Output structured JSON (default: true for CI)
  --branch <name> Explicit branch name (default: git rev-parse --abbrev-ref HEAD)
  --ci-run-id <id> CI run identifier for cache keying

Output: JSON via json_ok()
```

### 3.2 Output Schema

```json
{
  "schema": "pr-context-v1",
  "generated_at": "2026-03-30T12:00:00Z",
  "branch": "feature/phase-3",
  "ci_run_id": "run-12345",

  "cache_status": {
    "queen_global": {"cached": true, "age_seconds": 3600, "source": "cache"},
    "queen_local": {"cached": false, "source": "fresh"},
    "hive_wisdom": {"cached": true, "age_seconds": 7200, "source": "cache"},
    "pheromones": {"cached": false, "source": "fresh"},
    "colony_state": {"cached": false, "source": "fresh"},
    "flags": {"cached": false, "source": "fresh"}
  },

  "queen": {
    "global": {"user_prefs": "...", "codebase_patterns": "...", "instincts": "..."},
    "local": {"user_prefs": "...", "codebase_patterns": "...", "build_learnings": "...", "instincts": "..."},
    "combined_prefs": ["[global] prefer simple solutions", "[local] use TDD"]
  },

  "signals": {
    "count": 5,
    "redirects": [{"type": "REDIRECT", "content": "...", "strength": 0.9, "source": "user"}],
    "focus": [{"type": "FOCUS", "content": "...", "strength": 0.8, "source": "user"}],
    "feedback": [{"type": "FEEDBACK", "content": "...", "strength": 0.7, "source": "worker:builder"}],
    "instincts": [{"category": "Code-style", "context": "...", "action": "...", "confidence": 0.9}]
  },

  "hive": {
    "source": "hive",
    "count": 3,
    "entries": ["...", "...", "..."]
  },

  "colony_state": {
    "exists": true,
    "goal": "Build feature X",
    "state": "EXECUTING",
    "current_phase": 3,
    "total_phases": 8,
    "phase_name": "Authentication"
  },

  "blockers": {
    "count": 1,
    "items": [{"title": "...", "description": "...", "source": "..."}]
  },

  "decisions": {
    "count": 3,
    "items": ["decision 1", "decision 2", "decision 3"]
  },

  "prompt_section": "--- formatted text for direct agent injection ---",
  "prompt_section_json": "\"...escaped JSON string...\"",
  "char_count": 4523,
  "budget": 6000,
  "trimmed_sections": [],
  "warnings": ["pheromones.json was empty -- no signals injected"],
  "fallbacks_used": ["hive_wisdom: eternal fallback", "context_capsule: COLONY_STATE.json missing"]
}
```

### 3.3 Key Differences from colony-prime

| Aspect | colony-prime | pr-context |
|--------|-------------|------------|
| Output format | JSON with prompt_section string | JSON with structured sections + prompt_section |
| Consumption | Injected into worker prompt by orchestrator | Parsed by CI agent script |
| Error on no QUEEN.md | HARD FAIL (exit 1) | SOFT FAIL (warn, return empty wisdom) |
| Cache awareness | None | Tracks which sources were cached |
| Fallback logging | Silent | Explicit fallbacks_used array |
| Source metadata | None | cache_status per source |
| Token budget | 8K/4K | 6K/3K (tighter for CI) |
| Branch awareness | Implicit (cwd) | Explicit (--branch flag) |
| Structured signals | No (formatted text only) | Yes (typed arrays) |

The critical difference: pr-context NEVER hard-fails. Every source has a
fallback. This is essential for CI where partial context is better than no
context.

---

## 4. Source Classification: Cacheable vs Volatile

### 4.1 Cacheable Sources

These change rarely (user edits QUEEN.md, hive-wisdom is promoted on seal).
They can be cached across CI runs within the same branch.

| Source | File | Cache TTL | Invalidation |
|--------|------|-----------|-------------|
| QUEEN.md (global) | `~/.aether/QUEEN.md` | 1 hour | File mtime change |
| QUEEN.md (local) | `.aether/QUEEN.md` | 1 hour | File mtime change |
| Hive wisdom | `~/.aether/hive/wisdom.json` | 2 hours | File mtime change |
| Eternal memory | `~/.aether/eternal/memory.json` | 2 hours | File mtime change |
| Registry | `~/.aether/registry.json` | 1 hour | File mtime change |

Cache key: `{source_path}:{mtime_seconds}`. If mtime matches cached entry,
use cached JSON. Otherwise, re-read and update cache.

### 4.2 Volatile Sources

These change with every build or may be absent on a fresh PR branch.
They are ALWAYS read fresh.

| Source | File | Why Volatile |
|--------|------|-------------|
| Pheromone signals | `.aether/data/pheromones.json` | Created/modified every build |
| Colony state | `.aether/data/COLONY_STATE.json` | Updated every build/continue |
| Flags | `.aether/data/flags.json` | Created/resolved during builds |
| Rolling summary | `.aether/data/rolling-summary.log` | Appended every continue |
| Context decisions | `.aether/CONTEXT.md` | Updated by orchestrator |
| Context capsule | (computed from above) | Derived from volatile sources |
| Session | `.aether/data/session.json` | Per-conversation |

### 4.3 Cache Storage

Cache file: `.aether/data/pr-context-cache.json` (gitignored, branch-local).

```json
{
  "schema": "pr-context-cache-v1",
  "entries": {
    "queen_global": {
      "path": "/Users/user/.aether/QUEEN.md",
      "mtime": 1743300000,
      "data": { "user_prefs": "...", "codebase_patterns": "...", "instincts": "..." },
      "cached_at": "2026-03-30T12:00:00Z"
    },
    "hive_wisdom": {
      "path": "/Users/user/.aether/hive/wisdom.json",
      "mtime": 1743290000,
      "data": { "source": "hive", "entries": ["..."] },
      "cached_at": "2026-03-30T11:00:00Z"
    }
  }
}
```

Cache eviction: Remove entries older than their TTL on each pr-context call.
Cache is per-branch (lives in `.aether/data/`), so each branch gets its own
cache for local QUEEN.md but shares the global cache concept (global QUEEN.md
path is the same across branches).

---

## 5. Fallback Chain

When a source is missing, corrupt, or unreadable, pr-context follows this
chain. Every fallback is logged in the `fallbacks_used` output array.

### 5.1 QUEEN.md Fallback Chain

```
1. Read ~/.aether/QUEEN.md (global)
   FAIL -> warn, continue with empty global wisdom

2. Read .aether/QUEEN.md (local)
   FAIL -> warn, continue with empty local wisdom

3. Both missing -> continue with empty wisdom
   (colony-prime would HARD FAIL here; pr-context does not)

NOTE: colony-prime line 924-929 exits with error if neither exists.
      pr-context soft-fails instead, returning empty wisdom sections.
```

### 5.2 Hive Wisdom Fallback Chain

```
1. hive-read --domain <tags> --limit N --format json
   (reads ~/.aether/hive/wisdom.json with domain scoping)
   FAIL or empty -> step 2

2. Read ~/.aether/eternal/memory.json high_value_signals
   FAIL or empty -> step 3

3. Return empty hive section
   (This matches colony-prime's existing behavior, pheromone.sh lines 1096-1171)
```

### 5.3 Pheromone Signals Fallback Chain

```
1. Read .aether/data/pheromones.json
   MISSING -> step 2
   EMPTY (no active signals) -> step 2

2. Return empty signals section with warning
   (This matches colony-prime's existing behavior, pheromone.sh lines 973-986)

NOTE: On a fresh PR branch, pheromones.json may not exist.
      The pheromone-snapshot-inject protocol (Task 1.2, Section 4) should
      have created it during branch setup. If it did not, pr-context
      gracefully continues without signals.
```

### 5.4 Colony State Fallback Chain

```
1. Read .aether/data/COLONY_STATE.json
   MISSING -> step 2
   EMPTY or invalid JSON -> step 2

2. Return exists=false with defaults:
   {
     "exists": false,
     "goal": "No goal set",
     "state": "UNKNOWN",
     "current_phase": 0,
     "total_phases": 0,
     "phase_name": ""
   }

3. context-capsule subcommand already handles this:
   Returns {"exists":false,"prompt_section":"","word_count":0}
   (verified at aether-utils.sh line 4210-4213)
```

### 5.5 Flags/Blockers Fallback Chain

```
1. Read .aether/data/flags.json
   MISSING -> return count=0, items=[]

2. Invalid JSON -> warn, return count=0, items=[]
```

### 5.6 Context Decisions Fallback Chain

```
1. Read .aether/CONTEXT.md
   MISSING -> return count=0, items=[]

2. No "Recent Decisions" section -> return count=0, items=[]
```

### 5.7 Complete Fallback Priority

When multiple sources fail, the fallback order ensures the most critical
information is preserved:

```
CRITICAL (must have, soft-fail only):
  QUEEN.md wisdom     -> empty wisdom (agents work without it)
  Pheromone REDIRECTs -> empty signals (agents miss constraints)

IMPORTANT (should have):
  Hive wisdom         -> eternal fallback -> empty
  Colony state        -> defaults (goal="No goal set")
  Blockers            -> empty (agents miss warnings)

NICE TO HAVE:
  Phase learnings     -> empty (agents miss history)
  Key decisions       -> empty (agents miss context)
  Rolling summary     -> empty (agents miss narrative)
  User preferences    -> empty (agents miss calibration)
  Context capsule     -> empty (agents miss snapshot)
```

---

## 6. Token Budget Management

### 6.1 Budget Levels

| Mode | Budget | Use Case |
|------|--------|----------|
| Default | 6,000 chars | CI agents with standard context |
| Compact (--compact) | 3,000 chars | Tight CI agents, review pipeline tiers 1-2 |

These are tighter than colony-prime's 8K/4K because CI agents also consume
the PR diff, test output, and review criteria in their context window.

### 6.2 Section Budget Allocation

In default mode (6,000 chars), approximate allocation:

| Section | Max Chars | Priority |
|---------|-----------|----------|
| QUEEN wisdom (global + local) | 1,500 | HIGH |
| Pheromone signals (all) | 1,000 | HIGH |
| Hive wisdom | 500 | MEDIUM |
| Context capsule | 400 | MEDIUM |
| Phase learnings | 800 | MEDIUM |
| Key decisions | 400 | MEDIUM |
| Blockers | 500 | HIGH (never trimmed) |
| User preferences | 400 | LOW |
| Rolling summary | 300 | LOW |

### 6.3 Trim Order (same as colony-prime, with tighter thresholds)

When over budget, sections are removed in this order (first = trimmed first):

1. rolling-summary
2. phase-learnings
3. key-decisions
4. hive-wisdom
5. context-capsule
6. user-prefs
7. queen-wisdom-global
8. queen-wisdom-local
9. pheromone-signals (REDIRECTs preserved)
10. blockers: NEVER trimmed

This matches colony-prime exactly (pheromone.sh lines 1376-1492) so agents
receive consistent context regardless of whether they run in interactive
or CI mode.

### 6.4 Budget Enforcement Algorithm

Reuse colony-prime's existing algorithm (pheromone.sh lines 1388-1492) with
`cp_max_chars` set to 6,000 or 3,000 instead of 8,000/4,000.

---

## 7. Integration with Existing colony-prime

### 7.1 Relationship Diagram

```
                    EXISTING SYSTEM                      NEW: pr-context
                    ===============                      ================

  +------------------+                                  +------------------+
  | /ant:build       |                                  | CI Pipeline      |
  | (interactive)    |                                  | (automated)      |
  +--------+---------+                                  +--------+---------+
           |                                                     |
           | calls colony-prime --compact                        | calls pr-context --compact
           |                                                     |
           v                                                     v
  +------------------+                                  +------------------+
  | colony-prime     |                                  | pr-context       |
  | (pheromone.sh)   |                                  | (pheromone.sh)   |
  |                  |                                  |                  |
  | Assembles:       |                                  | Assembles:       |
  |  - QUEEN wisdom  |                                  |  - QUEEN wisdom  |
  |  - signals       |                                  |  - signals       |
  |  - hive          |                                  |  - hive          |
  |  - capsule       |                                  |  - capsule       |
  |  - learnings     |                                  |  - learnings     |
  |  - decisions     |                                  |  - decisions     |
  |  - blockers      |                                  |  - blockers      |
  |  - rolling       |                                  |  - rolling       |
  |                  |                                  |  + CACHE LAYER   |
  | Output:          |                                  |  + STRUCTURED    |
  |  prompt_section  |                                  |    SIGNAL ARRAYS |
  |  (text string)   |                                  |  + FALLBACK LOG  |
  |                  |                                  |                  |
  | Budget: 8K/4K    |                                  | Output:          |
  | HARD FAIL: no    |                                  |  structured JSON |
  |  QUEEN.md        |                                  |  + prompt_section|
  +------------------+                                  |  + cache_status  |
           |                                             |  + fallbacks     |
           |                                             |                  |
           v                                             | Budget: 6K/3K   |
  +------------------+                                  | SOFT FAIL: all   |
  | Worker Agent     |                                  |  sources         |
  | (injected via    |                                  +--------+---------+
  |  prompt)         |                                           |
  +------------------+                                           v
                                                        +------------------+
                                                        | CI Agent        |
                                                        | (parses JSON,   |
                                                        |  uses structured|
                                                        |  sections)      |
                                                        +------------------+
```

### 7.2 Code Reuse Strategy

pr-context should reuse colony-prime's existing functions rather than
duplicating logic:

| Function | Reuse Method |
|----------|-------------|
| `_extract_wisdom()` | Call directly (already defined in pheromone.sh) |
| `_filter_wisdom_entries()` | Call directly |
| `context-capsule` | Call via subcommand invocation |
| `pheromone-prime` | Call via subcommand invocation |
| `hive-read` | Call via subcommand invocation |
| Budget trimming | Extract to shared `_budget_enforce()` function |

The budget trimming logic (pheromone.sh lines 1388-1492) should be extracted
into a shared function `_budget_enforce()` that both `colony-prime` and
`pr-context` call with different `max_chars` values.

### 7.3 Subcommand Registration

Add to aether-utils.sh case statement (near line 3905):

```
pr-context) _pr_context "$@" ;;
```

### 7.4 When pr-context Is Called

| Caller | When | Mode |
|--------|------|------|
| CI pipeline (GitHub Actions) | On PR push/open, before agent review | `--compact` |
| `/ant:continue` Step 4 | After verification, for review context | default |
| `/ant:run` autopilot | Before each phase's review cycle | `--compact` |
| Watcher agent | When running as CI check | `--compact` |

---

## 8. Architecture Diagram

```
                     pr-context SUBCOMMAND ARCHITECTURE
                     ================================

     INPUT SOURCES                          CACHE LAYER
     ============                          ==========

     CACHEABLE (TTL-based):                .aether/data/
     +-------------------------+           pr-context-cache.json
     | ~/.aether/QUEEN.md     |--------+            ^
     | ~/.aether/hive/        |        |           |
     |   wisdom.json          |        +-- mtime  |
     | ~/.aether/eternal/     |        |   check   |
     |   memory.json          |        |           |
     +-------------------------+        +-----+-----+
                                             |
     VOLATILE (always fresh):                 | miss
     +-------------------------+              |
     | .aether/data/           |              v
     |   pheromones.json      |--------+
     | .aether/data/           |        |   FRESH READ
     |   COLONY_STATE.json    |        |
     | .aether/data/           |        |
     |   flags.json           |        |
     | .aether/data/           |        |
     |   rolling-summary.log  |        |
     | .aether/CONTEXT.md     |        |
     +-------------------------+        |
                                         |
                                         v
                              +----------------------+
                              |   ASSEMBLY ENGINE    |
                              |                      |
                              | 1. Load QUEEN wisdom |
                              |    (global + local)   |
                              | 2. Load pheromones   |
                              |    via pheromone-prime|
                              | 3. Load hive wisdom  |
                              |    via hive-read     |
                              | 4. Load colony state |
                              |    via context-capsule|
                              | 5. Load phase        |
                              |    learnings         |
                              | 6. Load decisions    |
                              | 7. Load blockers     |
                              | 8. Load rolling      |
                              |    summary           |
                              +----------+-----------+
                                         |
                                         v
                              +----------------------+
                              |   BUDGET ENFORCER    |
                              |   (_budget_enforce)  |
                              |                      |
                              | 6,000 chars default  |
                              | 3,000 chars compact  |
                              |                      |
                              | Trim order:          |
                              |  1. rolling-summary  |
                              |  2. phase-learnings  |
                              |  3. key-decisions    |
                              |  4. hive-wisdom      |
                              |  5. context-capsule  |
                              |  6. user-prefs       |
                              |  7. queen-global     |
                              |  8. queen-local      |
                              |  9. signals          |
                              |     (keep REDIRECTs) |
                              | 10. blockers: NEVER  |
                              +----------+-----------+
                                         |
                                         v
                              +----------------------+
                              |   OUTPUT FORMATTER   |
                              |                      |
                              | Structured JSON:     |
                              |  - queen (typed)     |
                              |  - signals (typed)   |
                              |  - hive (typed)      |
                              |  - colony_state      |
                              |  - blockers          |
                              |  - decisions         |
                              |  - prompt_section    |
                              |  - cache_status      |
                              |  - fallbacks_used    |
                              |  - warnings          |
                              |  - trimmed_sections  |
                              +----------+-----------+
                                         |
                                         v
                              +----------------------+
                              |   CONSUMERS          |
                              |                      |
                              | CI Agent (Watcher)   |
                              | CI Agent (Gatekeeper)|
                              | CI Agent (Auditor)   |
                              | CI Agent (Probe)     |
                              | CI Agent (Measurer)  |
                              +----------------------+


     FALLBACK CHAIN (per source)
     ===========================

     QUEEN.md:     global -> local -> empty wisdom (WARN)
     Hive:         hive-read -> eternal -> empty (WARN)
     Pheromones:   pheromones.json -> empty signals (WARN)
     Colony state: COLONY_STATE.json -> defaults (WARN)
     Flags:        flags.json -> empty blockers (WARN)
     Decisions:    CONTEXT.md -> empty (no warn)
     Rolling:      rolling-summary.log -> empty (no warn)
     Capsule:      context-capsule -> empty (no warn)
```

---

## 9. Data Flow in CI Pipeline

```
                    PR WORKFLOW: pr-context DATA FLOW
                    ================================

    Developer pushes to PR branch
              |
              v
    +--------------------+
    | CI Pipeline Start  |
    +----+-----------+---+
         |           |
         |           |
    +----v----+  +---v-----------+
    | Tier 1: |  | Tier 2:       |
    | CI      |  | Agent Reviews |
    | Checks  |  |               |
    |         |  | +-----------+ |
    | tests,  |  | | Watcher   | |
    | lint,   |  | +-----------+ |
    | build   |  |      |        |
    |         |  |      | pr-context --compact
    | (no     |  |      v        |
    |  colony |  | +-----------+ |
    |  context|  | | pr-context| |
    |  needed)|  | | output    | |
    +----+----+  | +-----------+ |
         |       |      |        |
         |       | +---v-----------+
         |       | | Gatekeeper    |
         |       | +-----------+   |
         |       |      | pr-context --compact
         |       |      v        |
         |       | +-----------+ |
         |       | | Auditor    | |
         |       | +-----------+ |
         |       +---+-----------+
         |           |
         v           v
    +--------------------+
    | Tier 3: Aggregator |
    | (merges review     |
    |  outputs, checks   |
    |  for conflicts)    |
    +----+-----------+---+
         |           |
         v           v
    +--------------------+
    | Tier 4: Human Gate |
    | (PR review UI)     |
    +----+-----------+---+
         |           |
         v           v
    +--------------------+
    | Tier 5: Post-Merge |
    | (state sync to     |
    |  main, pheromone   |
    |  merge-back)       |
    +--------------------+
```

---

## 10. Edge Cases

### 10.1 No Colony Initialized on Branch

A PR branch may not have `.aether/data/COLONY_STATE.json` (e.g., a developer
creates a branch without running `/ant:init`). In this case:

- pr-context returns `colony_state.exists = false` with defaults
- context-capsule returns `{"exists":false,...}` (verified at aether-utils.sh 4210-4213)
- All other sources are read independently
- The CI agent receives wisdom and signals (if pheromones exist) but no colony
  state. This is acceptable -- the agent can still perform code review.

### 10.2 Worktree Isolation

When agents work in separate worktrees (as designed in the PR workflow), each
worktree has its own `.aether/data/` directory. pr-context reads from the
current working directory's data, which is the worktree-local data. Hub state
(`~/.aether/`) is shared across all worktrees.

```
main worktree:   /repo/                    -> .aether/data/ (main state)
agent worktree:  /repo-worktree-agent-1/   -> .aether/data/ (agent-1 state)
hub:             ~/.aether/                -> shared across all
```

### 10.3 Concurrent CI Runs

When two PRs trigger CI simultaneously, both may call pr-context. Since
branch-local state is isolated per worktree/branch, there is no conflict for
volatile sources. Hub state reads are safe (read-only). The cache file
(`.aether/data/pr-context-cache.json`) is branch-local, so concurrent writes
to different branches do not conflict. Concurrent writes to the same branch's
cache file use the existing `acquire_lock` mechanism from `file-lock.sh`.

### 10.4 Stale Cache After Merge

After a PR is merged to main, main's `.aether/data/` is updated by the
post-merge sync (Task 1.1, Section 5). The cache on main may be stale (pointing
to pre-merge QUEEN.md content). pr-context's mtime-based cache invalidation
handles this: if QUEEN.md was updated during post-merge sync, the mtime changes,
and the cache entry is invalidated on the next pr-context call.

### 10.5 Compact Mode Exhaustion

In extreme cases, even compact mode (3,000 chars) may not fit all critical
sections. In this case, pr-context follows the same trim order as colony-prime
but with a stricter floor: blockers + REDIRECT signals are ALWAYS preserved,
even if they alone exceed the budget. If the budget is exceeded even after
trimming everything except blockers and REDIRECTs, pr-context returns the
output with a `warnings` entry noting the budget overflow.

---

## 11. Integration Points

### 11.1 New Subcommands

| Subcommand | Purpose | Location |
|------------|---------|----------|
| `pr-context` | Assemble CI-ready colony context | pheromone.sh (near colony-prime) |

### 11.2 New Helper Functions

| Function | Purpose | Reused By |
|----------|---------|-----------|
| `_pr_context()` | Main pr-context implementation | pr-context subcommand |
| `_budget_enforce()` | Shared budget trimming logic | colony-prime, pr-context |
| `_cache_read()` | Read from pr-context-cache.json | pr-context |
| `_cache_write()` | Write to pr-context-cache.json | pr-context |

### 11.3 Command Integration

| Command | Integration Point | Behavior |
|---------|-------------------|----------|
| `/ant:continue` | Step 4 (post-verify) | Call pr-context to generate review context |
| `/ant:run` | Before review cycle | Call pr-context --compact for each CI agent |
| CI workflow | PR push/open | Call pr-context --compact for Watcher, Gatekeeper, etc. |

### 11.4 File Locking

- Cache reads: No lock needed (read-only)
- Cache writes: Use `acquire_lock` on `.aether/data/pr-context-cache.json`
  (same pattern as other data files, via `file-lock.sh`)
- Volatile source reads: No lock needed (read-only, branch-local)
- Hub state reads: No lock needed (read-only)

---

## 12. Summary

| Aspect | Decision |
|--------|----------|
| Subcommand name | `pr-context` |
| Output format | Structured JSON (not just prompt_section) |
| Error policy | NEVER hard-fail; all sources have fallbacks |
| Cacheable sources | QUEEN.md (global/local), hive, eternal (TTL-based) |
| Volatile sources | pheromones, COLONY_STATE, flags, rolling-summary, CONTEXT.md |
| Token budget | 6,000 chars default / 3,000 chars compact |
| Trim order | Same as colony-prime (rolling-summary first, blockers never) |
| Fallback for QUEEN.md | Empty wisdom (warn) -- NOT hard-fail |
| Fallback for hive | eternal memory -> empty |
| Fallback for pheromones | Empty signals (warn) |
| Fallback for colony state | exists=false with defaults |
| Cache storage | `.aether/data/pr-context-cache.json` (gitignored) |
| Cache invalidation | mtime-based per source |
| Code reuse | Reuse _extract_wisdom, _filter_wisdom_entries; extract _budget_enforce |
| Integration | CI pipeline tiers 2-3, /ant:continue, /ant:run |

---

## 13. Verification

All assertions in this document were verified against the actual codebase:

- colony-prime function: verified at pheromone.sh lines 737-1553
- colony-prime 9 sections assembled: verified at pheromone.sh lines 1000-1012
- Budget trimming order: verified at pheromone.sh lines 1376-1492
- HARD FAIL on no QUEEN.md: verified at pheromone.sh lines 924-929
- SOFT FAIL on no pheromones.json: verified at pheromone.sh lines 973-986
- context-capsule COLONY_STATE.json missing: verified at aether-utils.sh lines 4210-4213
- context-capsule word budget: verified at aether-utils.sh lines 4344-4363
- context-capsule max defaults: verified at aether-utils.sh lines 4177-4181
- Hive fallback to eternal: verified at pheromone.sh lines 1139-1171
- Subcommand registration pattern: verified at aether-utils.sh lines 3904-3906
- File lock mechanism: verified in state-contract-design.md Section 6
- `.aether/data/` gitignored: verified in state-contract-design.md Section 1
- State contract branch-local rules: verified in state-contract-design.md Sections 3-4
- Pheromone propagation protocol: verified in pheromone-propagation-design.md Sections 4-5

---

*Design complete. Next steps: implement pr-context subcommand in pheromone.sh,
extract _budget_enforce() shared function, and add CI workflow integration
(implementation tasks).*
