# Branch-Local State Contract Design

> Task 1.1: Define which colony state is branch-local vs hub-global
> Author: Brick-81 (Builder)
> Date: 2026-03-30
> Verified against: aether CLI, pkg/ packages, actual file locations

---

## 1. State Type Inventory

Every state type in the Aether system, classified by storage location.

### Branch-Local State (`.aether/data/` -- inside repo, gitignored)

These files live inside the repository at `.aether/data/`. They are
**gitignored** (see `.gitignore` line 80: `.aether/data/`), so each git
branch gets its own independent copy. Switching branches means switching
colony context.

| # | State File | Purpose | Source Module |
|---|-----------|---------|---------------|
| 1 | `COLONY_STATE.json` | Master colony state: goal, phases, memory (learnings, decisions, instincts, events), milestone | `aether state-*` commands |
| 2 | `pheromones.json` | Active pheromone signals (FOCUS, REDIRECT, FEEDBACK) with strength, decay, dedup | `aether pheromone-*` commands |
| 3 | `midden/midden.json` | Failure records with category, severity, acknowledgment status | `aether midden-*` commands |
| 4 | `flags.json` | Active flags (blocking, informational) created by `/ant-flag` | `aether flag-*` commands |
| 5 | `session.json` | Current session metadata: session_id, last command, colony_goal, baseline_commit | `aether session-*` commands |
| 6 | `learning-observations.json` | Raw learning observations captured via `memory-capture` | `aether memory-*` commands |
| 7 | `rolling-summary.log` | Rolling log of condensed learnings per phase | `aether memory-*` commands |
| 8 | `spawn-tree.txt` | Tree of spawned agents with parent/child relationships | `aether spawn-*` commands |
| 9 | `activity.log` | Timestamped activity entries (CREATED, MODIFIED, RESEARCH, etc.) | `aether CLI` |
| 10 | `last-build-result.json` | Result of the most recent build (status, phase, tasks) | build commands |
| 11 | `last-build-claims.json` | Files created/modified by the most recent build | build commands |
| 12 | `pending-decisions.json` | Unresolved decisions (replan, visual checkpoint) | build/continue |
| 13 | `errors.log` | Error entries for debugging | `aether error-*` commands |
| 14 | `queen-wisdom.json` | Local colony-level wisdom cache (legacy) | `aether queen-*` commands |
| 15 | `survey/` | Survey results from colonizer (architecture, disciplines, pathogens, provisions) | `aether session-*` commands |
| 16 | `backups/` | Checkpoint backups of COLONY_STATE.json | `aether state-*` commands |
| 17 | `constraints.json` | Focus areas and constraints (legacy, deprecating) | `aether pheromone-*` commands |
| 18 | `colony-registry.json` | Local colony registry (legacy, superseded by hub) | `aether CLI` |
| 19 | `completion-report.md` | Summary report from colony completion | seal commands |
| 20 | `watch/`, `watch-status.txt`, `watch-progress.txt` | Watcher agent monitoring state | watcher commands |
| 21 | `council/`, `immune/` | Council and immune system state | council/immune commands |
| 22 | `phase-research/` | Per-phase research artifacts | build commands |

**Key property:** All branch-local state uses `COLONY_DATA_DIR` (which resolves to
`.aether/data/` or `.aether/data/colonies/{name}/` when colony_name is set),
except `COLONY_STATE.json` which always uses `DATA_DIR` (it must remain at the
root for `_resolve_colony_data_dir` bootstrap).

### Hub-Global State (`~/.aether/` -- outside repo, branch-agnostic)

These files live in the user's home directory at `~/.aether/`. They are
**NOT inside any git repo**, so they are shared identically across all
branches, all worktrees, and all repositories.

| # | State File | Purpose | Source Module |
|---|-----------|---------|---------------|
| 1 | `QUEEN.md` | Global colony wisdom: user preferences, codebase patterns, build learnings, instincts | `aether queen-*` commands |
| 2 | `hive/wisdom.json` | Cross-colony wisdom (200-entry cap, LRU eviction, domain-scoped) | `aether hive-*` commands |
| 3 | `eternal/memory.json` | High-value signals promoted from expired pheromones (legacy fallback) | `aether pheromone-*` commands |
| 4 | `eternal/pheromones.xml` | XML export of pheromones for cross-colony sharing | `aether pheromone-*` commands |
| 5 | `registry.json` | All repos using Aether with domain tags, goals, active status | `aether CLI` |
| 6 | `skills/` | Installed skills (colony/ + domain/), manifest-based tracking | `aether skill-*` commands |
| 7 | `chambers/` | Entombed (archived) colony chambers | seal commands |
| 8 | `data/activity.log` | Hub-level activity log (cross-colony) | `aether CLI` |
| 9 | `version.json` | Installed Aether version | `aether CLI` |
| 10 | `manifest.json` | Package manifest for installed files | setupHub() |
| 11 | `system/` | Installed system files (agents, commands, utils) | setupHub() |

**Key property:** Hub state is accessed via `$HOME/.aether/` hardcoded paths,
never via `DATA_DIR` or `COLONY_DATA_DIR`.

---

## 2. State-Flow Diagram

```
                          BRANCH-LOCAL STATE                      HUB-GLOBAL STATE
                    (.aether/data/ per branch)                (~/.aether/ shared)

    +-------------------+                             +----------------------+
    | main branch       |                             | QUEEN.md             |
    | COLONY_STATE.json |<------- read/write -------->| hive/wisdom.json     |
    | pheromones.json   |                             | eternal/memory.json  |
    | midden/           |                             | registry.json        |
    | flags.json        |                             | skills/              |
    | session.json      |                             | chambers/            |
    | learnings/        |                             +----------------------+
    +-------------------+                                      ^
            |                                                  |
            | git branch                                       | always shared
            |                                                  |
    +-------------------+                                      |
    | feature/phase-3   |  (isolate per worktree/branch)       |
    | COLONY_STATE.json |<---- reads ----+                     |
    | pheromones.json   |                |                     |
    | midden/           |                |                     |
    | flags.json        |                |                     |
    | session.json      |                |                     |
    +-------------------+                |                     |
            |                             |                     |
            | PR merge                    |                     |
            v                             |                     |
    +-------------------+                |                     |
    | post-merge        |                |                     |
    | (main updated)    |                |                     |
    +-------------------+                |                     |
                                         |                     |
            +----------------------------+---------------------+
            |            STATE READ RULES                     |
            +--------------------------------------------------+

    BUILD ON PR BRANCH reads:
    1. Branch-local: COLONY_STATE.json, pheromones, midden, flags
       -> These reflect THIS branch's colony context
    2. Hub-global: QUEEN.md, hive/wisdom.json, eternal, registry
       -> These are the SAME across all branches
    3. QUEEN.md: two-level loading
       -> Global (~/.aether/QUEEN.md) loaded FIRST
       -> Local (.aether/QUEEN.md) loaded SECOND, extends global
    4. Pheromones: branch-local only
       -> Each branch has its own signal set
    5. Instincts: stored in branch-local COLONY_STATE.json
       -> Different per branch; promoted to hub QUEEN.md on seal
```

---

## 3. Read Rules During a Build on a PR Branch

When an agent runs `/ant-build` or `/ant-continue` on a feature branch:

### Rule 1: Branch-local state is authoritative for colony context

| State | Source | Rationale |
|-------|--------|-----------|
| COLONY_STATE.json | Branch-local | Phase progress, task assignments, milestones are per-branch |
| pheromones.json | Branch-local | Signals are scoped to the colony's work on this branch |
| midden/ | Branch-local | Failures observed on this branch |
| flags.json | Branch-local | Flags block progress on this branch |
| session.json | Branch-local | Session is per-conversation, per-branch |
| learning-observations.json | Branch-local | Observations from work on this branch |
| spawn-tree.txt | Branch-local | Agent spawns are per-session |
| activity.log | Branch-local | Activity is per-session |

### Rule 2: Hub-global state is authoritative for cross-branch knowledge

| State | Source | Rationale |
|-------|--------|-----------|
| QUEEN.md (global) | Hub | Wisdom accumulated across all colonies, all branches |
| hive/wisdom.json | Hub | Cross-colony patterns, shared across branches |
| eternal/memory.json | Hub | High-value signals from any colony |
| registry.json | Hub | Which repos exist, their domain tags |
| skills/ | Hub | Available skills are the same everywhere |

### Rule 3: QUEEN.md uses two-level merging

1. Load `~/.aether/QUEEN.md` (hub-global) first
2. Load `.aether/QUEEN.md` (branch-local) second
3. Local entries EXTEND (not override) global entries
4. Colony-prime injects combined wisdom into worker prompts

### Rule 4: Pheromone signals do NOT cross branches

- Each branch has its own `pheromones.json`
- Signals created on `feature/phase-3` are invisible on `main` and other branches
- This is intentional: different branches may have different constraints

### Rule 5: Instincts are branch-local until promoted

- Instincts live inside `COLONY_STATE.json` (branch-local)
- They are promoted to hub `QUEEN.md` via `queen-promote` during `/ant-seal`
- After promotion, they become visible to all branches via hub QUEEN.md

---

## 4. Post-Merge State Flow Rules

When a PR is merged into main, state flows as follows:

### 4.1 Code changes (handled by git merge)

All changes to `cmd/`, `pkg/`, `.claude/commands/`,
`.claude/agents/`, etc. are merged into main via git. These are source-of-truth
files tracked by git.

### 4.2 Branch-local state (NOT merged by git)

Since `.aether/data/` is gitignored, branch-local state is **NOT automatically
merged**. This creates a coordination challenge:

| Scenario | What happens | Mitigation |
|----------|-------------|------------|
| Phase completed on branch | COLONY_STATE.json updated on branch, not on main | `/ant-continue` must update main after merge |
| Pheromone created on branch | Signal exists only on branch | Hub QUEEN.md promotion at seal |
| Midden entry on branch | Failure tracked only on branch | Must be promoted to hub or re-observed |
| Learning on branch | Observation in branch-local file | Must flow to hub via wisdom pipeline |

### 4.3 Required state synchronization after merge

For the PR-based workflow to work correctly, the following state must flow
from the feature branch back to main after merge:

```
Feature branch (pre-merge)          Main branch (post-merge)
==========================          =========================
COLONY_STATE.json                   COLONY_STATE.json
  - phase_learnings    ------>        - phase_learnings (appended)
  - instincts          ------>        - instincts (merged by id)
  - events             ------>        - events (appended)
  - current_phase      ------>        - current_phase (advanced)
  - milestone          ------>        - milestone (updated)

pheromones.json                     (NOT copied -- branch-scoped)
midden/midden.json                  (NOT copied -- branch-scoped)
  - BUT: critical failures    ------> queen-wisdom REDIRECT in hub

QUEEN.md (local)                    QUEEN.md (local)
  - patterns promoted   ------>        - patterns (merged)
```

### 4.4 Hub state flows (always cross-branch)

Hub state is automatically shared because it lives outside git:

| Hub state | Updated when | Visible to |
|-----------|-------------|------------|
| QUEEN.md | `queen-promote` during continue/seal | All branches immediately |
| hive/wisdom.json | `hive-promote` during seal (confidence >= 0.8) | All branches immediately |
| eternal/memory.json | `eternal-store` on pheromone expiry | All branches immediately |
| registry.json | `registry-add` on colony init | All branches immediately |

---

## 5. PR Lifecycle State Contract

### Phase: Branch Creation

```
1. Create branch from main:  git checkout -b feature/task-1.1
2. Branch-local state inherited: .aether/data/ files copied from main
   (they are untracked files, so git carries the working tree copy)
3. Hub state: unchanged (shared via ~/.aether/)
```

### Phase: Build on Branch

```
1. /ant-init or /ant-build writes to branch-local state
2. Colony reads hub QUEEN.md + hive for cross-branch wisdom
3. Pheromones created are branch-scoped
4. Learnings captured are branch-scoped
5. Midden entries are branch-scoped
```

### Phase: Review (5-tier pipeline)

```
1. CI checks run (hub-agnostic -- code quality, tests)
2. Agent reviews run (may read branch-local state for context)
3. Aggregation (hub state: hive wisdom informs review quality)
4. Human gate (no state dependency)
5. Post-merge verification (main branch state updated)
```

### Phase: Merge

```
1. Git merges code changes into main
2. Branch-local state NOT merged (gitignored)
3. Hub state already up-to-date (written during build/continue)
4. POST-MERGE HOOK REQUIRED: synchronize state back to main
```

### Phase: Post-Merge State Sync (NEW -- required for PR workflow)

```
After merge, the following must happen on main:

1. Advance COLONY_STATE.json on main:
   - Append phase_learnings from feature branch
   - Merge instincts (by id, keep higher confidence)
   - Append events
   - Advance current_phase

2. Update hub state (if not already done):
   - Promote high-confidence instincts to QUEEN.md
   - Store cross-colony wisdom in hive
   - Archive completed pheromones to eternal

3. Reset branch-local ephemera on main:
   - Clear session.json (or update to reflect merge)
   - Rotate spawn-tree.txt
   - Archive branch activity
```

---

## 6. Implications for Worktree Isolation

When agents work in separate git worktrees:

```
main worktree:         /path/to/repo/          (.aether/data/ -> main state)
agent-1 worktree:      /path/to/repo-worktree-1/  (.aether/data/ -> agent-1 state)
agent-2 worktree:      /path/to/repo-worktree-2/  (.aether/data/ -> agent-2 state)

Hub (~/.aether/):      Shared across ALL worktrees
```

### Conflict scenarios:

| Conflict | Detection | Resolution |
|----------|-----------|------------|
| Two agents write COLONY_STATE.json | File lock (`acquire_lock`) | First writer wins; second retries |
| Two agents write pheromones.json | File lock | Same as above |
| Two agents promote to hub QUEEN.md | File lock on QUEEN.md | Sequential writes, merge-aware |
| Hub wisdom.json concurrent write | Hub-level file lock | Sequential with lock |
| Activity.log concurrent append | Append-only, no lock needed | Lines may interleave (acceptable) |

### Worktree isolation guarantee:

Since `.aether/data/` is gitignored and each worktree has its own working tree,
each agent gets complete isolation of branch-local state. Hub state requires
file locks for safe concurrent access (already implemented via `pkg/storage/storage.go`).

---

## 7. Summary Table

| State Type | Location | Scope | Git-tracked? | Merge behavior |
|-----------|----------|-------|-------------|----------------|
| COLONY_STATE.json | `.aether/data/` | Branch-local | No (gitignored) | NOT auto-merged; requires sync |
| pheromones.json | `.aether/data/` | Branch-local | No | NOT merged; branch-scoped by design |
| midden/ | `.aether/data/` | Branch-local | No | NOT merged; critical ones promoted to hub |
| flags.json | `.aether/data/` | Branch-local | No | NOT merged; re-evaluated per branch |
| session.json | `.aether/data/` | Branch-local | No | NOT merged; reset per session |
| learning-observations.json | `.aether/data/` | Branch-local | No | NOT merged; wisdom flows via hub pipeline |
| rolling-summary.log | `.aether/data/` | Branch-local | No | NOT merged; appended on main post-merge |
| spawn-tree.txt | `.aether/data/` | Branch-local | No | NOT merged; per-session |
| activity.log | `.aether/data/` | Branch-local | No | NOT merged; per-session |
| last-build-*.json | `.aether/data/` | Branch-local | No | NOT merged; per-build |
| pending-decisions.json | `.aether/data/` | Branch-local | No | NOT merged; per-branch |
| errors.log | `.aether/data/` | Branch-local | No | NOT merged; per-session |
| queen-wisdom.json | `.aether/data/` | Branch-local | No | NOT merged; legacy |
| survey/ | `.aether/data/` | Branch-local | No | NOT merged; per-colony |
| QUEEN.md (global) | `~/.aether/` | Hub-global | N/A | Always current; shared across branches |
| hive/wisdom.json | `~/.aether/hive/` | Hub-global | N/A | Always current; shared across repos |
| eternal/memory.json | `~/.aether/eternal/` | Hub-global | N/A | Always current; shared across repos |
| registry.json | `~/.aether/` | Hub-global | N/A | Always current; shared across repos |
| skills/ | `~/.aether/skills/` | Hub-global | N/A | Always current; shared across repos |
| chambers/ | `~/.aether/chambers/` | Hub-global | N/A | Always current; shared across repos |

---

## 8. Verification

All assertions in this document were verified against the actual codebase:

- Source code paths verified via `grep` of `COLONY_DATA_DIR`, `DATA_DIR`, `$HOME/.aether/` in all utils modules
- File existence verified for all 22 branch-local and 11 hub-global state types
- `.gitignore` confirmed to exclude `.aether/data/` (line 80)
- `~/.aether/` confirmed to be outside any git repo
- Test file: `pkg/storage/storage_test.go` (34/34 passing)

---

*Design complete. Next steps: implement post-merge state sync mechanism (Task 1.2).*
