# Pheromone Cross-Branch Propagation Protocol

> Task 1.2: Design pheromone injection and merge-back for PR-based workflow
> Author: Weld-83 (Builder)
> Date: 2026-03-30
> Depends on: Task 1.1 -- Branch-Local State Contract Design
> Verified against: colony-prime (_pheromone_write, _pheromone_read, _pheromone_prime, _pheromone_expire)

---

## 1. Problem Statement

In the PR-based workflow, each colony task creates its own git branch with an
isolated `.aether/data/pheromones.json`. Since `.aether/data/` is gitignored,
pheromone signals created on a feature branch are invisible on main and on other
feature branches. This isolation is intentional for most state, but pheromones
are different: a REDIRECT signal emitted by the user should constrain ALL agents
across ALL branches, and a FOCUS signal set before a build should apply to the
work regardless of which branch the agent is on.

The state contract (Task 1.1) established that "pheromone signals do NOT cross
branches" (Rule 4), which is correct for the current single-branch workflow.
This document defines the protocol to selectively propagate signals across
branches when the PR workflow is active, while preserving branch isolation where
it matters.

### Design Goals

1. **User intent is never lost.** A REDIRECT from the user must reach every
   agent, regardless of which branch they are working on.
2. **Branch-local signals stay branch-local.** A FEEDBACK signal generated
   during a build on `feature/phase-3` should not pollute `main`'s signal set
   unless the user explicitly intended it.
3. **No signal conflicts go unnoticed.** When main and a branch both have
   signals of the same type with overlapping intent, the protocol must detect
   and resolve this deterministically.
4. **The existing `_pheromone_write` dedup via `content_hash` is reused.**
   This protocol builds on the existing dedup mechanism, not a new one.
5. **Merge-back is non-blocking.** Signal propagation failures must never
   block a PR merge or break the colony.

---

## 2. Current System Reference

### 2.1 Signal Data Model

Each signal in `pheromones.json` has the following structure (from
`_pheromone_write`, colony-prime line 242):

```
{
  "id": "sig_focus_1743300000_1234",
  "type": "FOCUS | REDIRECT | FEEDBACK",
  "priority": "high | normal | low",
  "source": "user | worker:builder | system",
  "created_at": "2026-03-30T12:00:00Z",
  "expires_at": "2026-03-30T18:00:00Z | phase_end",
  "active": true,
  "strength": 0.8,
  "reason": "User emitted via /ant:focus",
  "content": { "text": "security" },
  "content_hash": "sha256:abc123...",
  "reinforcement_count": 0
}
```

Key fields for propagation:
- `content_hash`: SHA-256 of the raw content text. Used for dedup -- writing a
  signal with the same type and content_hash reinforces (not duplicates) the
  existing signal.
- `source`: Origin of the signal. User signals have higher propagation priority
  than worker or system signals.
- `strength`: 0.0-1.0. Dedup takes the `max` of existing and new strength.
- `reinforcement_count`: Incremented on dedup reinforce. Useful for detecting
  how important a signal has become.

### 2.2 Existing Dedup Behavior

From `_pheromone_write` (colony-prime lines 194-233):

1. Compute `content_hash` = SHA-256 of content text
2. Check for active signal with same `type` AND `content_hash`
3. If match found: reinforce (max strength, reset created_at, increment count)
4. If no match: create new signal

This dedup is per-file -- it only operates within a single `pheromones.json`.

### 2.3 Precedence Rules

From the pheromone system design:
- REDIRECT (high priority) > FOCUS (normal) > FEEDBACK (low)
- When multiple signals of the same type exist, higher strength wins for
  prompt injection ordering

---

## 3. Protocol Overview

The protocol has two phases:

1. **Injection at Branch Creation** -- snapshot canonical signals from main
   into the new branch so agents start with the correct context.
2. **Merge-Back After PR** -- flow branch-local signal changes back to main
   so the colony's knowledge is preserved.

```
                    INJECTION AT CREATION                MERGE-BACK AFTER PR

  main branch              feature branch                 main branch
  pheromones.json   --->   pheromones.json                pheromones.json
  (canonical)              (branch-local copy)             (updated)

  [user REDIRECTs]         [injected REDIRECTs]           [merged signals]
  [user FOCUSes]    ===>   [injected FOCUSes]      ===>   [new branch signals]
  [worker FEEDBACKs]       [NOT injected]                 [selected FEEDBACKs]
                           [branch-created FEEDBACKs]     [deconflicted]
```

---

## 4. Injection-at-Creation Protocol

### 4.1 When

When a new PR branch is created (via `git worktree add` or `git checkout -b`),
the pheromone injection step runs as part of branch setup.

### 4.2 What Gets Injected

NOT all signals propagate. The injection rule depends on signal source and type:

| Signal Type | Source | Injected? | Rationale |
|-------------|--------|-----------|-----------|
| REDIRECT | user | YES (always) | User constraints must reach all agents |
| REDIRECT | worker:* | YES (always) | Worker-discovered constraints are colony-wide knowledge |
| REDIRECT | system | YES (always) | System-generated constraints (e.g., midden threshold) are global |
| FOCUS | user | YES (always) | User attention directives apply across branches |
| FOCUS | worker:* | NO | Worker focus is task-specific to the originating branch |
| FOCUS | system | NO | System focus is task-specific |
| FEEDBACK | user | YES (always) | User calibration should apply everywhere |
| FEEDBACK | worker:* | NO | Worker feedback is specific to the work that generated it |
| FEEDBACK | system | NO | System feedback is branch-specific |

**Rule: User and REDIRECT signals always propagate. Worker FOCUS/FEEDBACK do not.**

### 4.3 How Injection Works

```
Step 1: Read main's pheromones.json
        pheromone-read --all
        -> active_signals[]

Step 2: Filter injectable signals
        Filter: type == REDIRECT (any source)
             OR (source == "user" AND type IN (FOCUS, FEEDBACK))

Step 3: For each injectable signal, write to branch pheromones.json
        For each signal in filtered list:
          pheromone-write <type> <content.text>
            --strength <signal.strength>
            --ttl <computed_ttl_from_expires_at>
            --source <signal.source>
            --reason "Injected from main branch (snapshot)"

        NOTE: This naturally reuses the existing content_hash dedup.
        If the branch already has a signal with the same type+content_hash,
        it will be reinforced (max strength) rather than duplicated.

Step 4: Write snapshot metadata
        Write .aether/data/pheromone-snapshot.json:
        {
          "snapshot_from_branch": "main",
          "snapshot_from_commit": "<main HEAD SHA>",
          "snapshot_at": "<ISO timestamp>",
          "injected_count": N,
          "injected_ids": ["sig_redirect_...", ...],
          "skipped_count": M,
          "skipped_reasons": ["worker:focus signals excluded"]
        }
```

### 4.4 TTL Handling During Injection

When injecting a signal from main into a branch, the TTL must be recalculated:

- If `expires_at == "phase_end"`: keep as `"phase_end"` (resets with the
  branch's phase lifecycle)
- If `expires_at` is an ISO timestamp: compute remaining TTL from now, write
  with that TTL. If already expired, skip injection.

### 4.5 Injection Flow Diagram

```
                           BRANCH CREATION
                           ================

  +-------------------+                              +-------------------+
  | MAIN BRANCH       |                              | NEW BRANCH        |
  | pheromones.json   |                              | pheromones.json   |
  |                   |                              | (empty or copy)   |
  | signals:          |                              |                   |
  |  [R] user:avoid   |     pheromone-snapshot       |                   |
  |  [R] sys:pattern  |     --inject-from main       |                   |
  |  [F] user:auth    |          |                    |                   |
  |  [F] wrk:db       |          v                    |                   |
  |  [B] user:clean   |   +-------------+            |                   |
  |  [B] wrk:fast     |   | FILTER      |            |                   |
  |                   |   |             |            |                   |
  |                   |   | INJECT:     |            |                   |
  |                   |   |  R:any      |            |                   |
  |                   |   |  F:user     |            |                   |
  |                   |   |  B:user     |            |                   |
  |                   |   |             |            |                   |
  |                   |   | SKIP:       |            |                   |
  |                   |   |  F:worker   |            |                   |
  |                   |   |  F:system   |            |                   |
  |                   |   |  B:worker   |            |                   |
  |                   |   |  B:system   |            |                   |
  |                   |   +------+------+            |                   |
  |                   |          |                    |                   |
  |                   |          | pheromone-write    |                   |
  |                   |          | (per signal)       |                   |
  |                   |          +------->------------>| signals:          |
  |                   |                               |  [R] user:avoid   |
  |                   |                               |  [R] sys:pattern  |
  |                   |                               |  [F] user:auth    |
  |                   |                               |  [B] user:clean   |
  |                   |                               |                   |
  |                   |                               | pheromone-        |
  |                   |                               | snapshot.json     |
  +-------------------+                               +-------------------+

  Legend: [R]=REDIRECT, [F]=FOCUS, [B]=FEEDBACK
```

---

## 5. Merge-Back-After-PR Protocol

### 5.1 When

After a PR is merged into main (via GitHub merge, git merge, or squash merge).
This runs as a post-merge step on the main branch.

### 5.2 What Gets Merged Back

Branch-local signals that should flow back to main:

| Signal Type | Source | Merged Back? | Rationale |
|-------------|--------|-------------|-----------|
| REDIRECT | user | NO | Already exists on main (was injected) |
| REDIRECT | worker:* | YES | New constraint discovered during work |
| REDIRECT | system | YES | New system constraint discovered |
| FOCUS | user | NO | Already on main |
| FOCUS | worker:* | NO | Task-specific, no longer relevant |
| FOCUS | system | NO | Task-specific |
| FEEDBACK | user | NO | Already on main |
| FEEDBACK | worker:* | CONDITIONAL | Only if reinforcement_count >= 2 |
| FEEDBACK | system | CONDITIONAL | Only if reinforcement_count >= 2 |

**Rule: Only new branch-discovered REDIRECTs and heavily-reinforced FEEDBACKs
merge back. User signals are already on main. Worker FOCUS is discarded.**

The `reinforcement_count >= 2` threshold for FEEDBACK means the signal was
emitted at least twice during the build, indicating it represents a recurring
observation rather than a one-off note.

### 5.3 How Merge-Back Works

```
Step 1: Read branch's pheromones.json (before merge, from the merged commit)
        NOTE: Since .aether/data/ is gitignored, we need to capture this
        BEFORE the merge, or pass it via a transient file.

        Solution: Before merge, run pheromone-export-branch
        -> writes .aether/data/pheromone-branch-export.json
        This file IS committed to the branch (not gitignored) as
        .aether/data/pheromone-branch-export.json

Step 2: Read main's pheromones.json
        pheromone-read --all
        -> main_signals[]

Step 3: For each branch signal eligible for merge-back:
        a. Check if main already has a signal with same type+content_hash
        b. If YES (conflict): apply Conflict Resolution (Section 6)
        c. If NO: write new signal to main via pheromone-write

Step 4: Write merge metadata
        Append to .aether/data/pheromone-merge-log.json:
        {
          "merged_from_branch": "feature/phase-3",
          "merged_from_commit": "<branch HEAD SHA>",
          "merged_at": "<ISO timestamp>",
          "new_signals": ["sig_redirect_...", ...],
          "conflicts_resolved": [
            {"signal_id": "...", "resolution": "reinforce|skip|replace"}
          ],
          "skipped_count": N
        }
```

### 5.4 The Export-Before-Merge Requirement

Since `.aether/data/pheromones.json` is gitignored, the branch's signal data
is lost when the branch is deleted after merge. The protocol requires a pre-merge
export step:

```
# As part of the PR submission (before merge):
pheromone-export-branch
  -> Reads branch's pheromones.json
  -> Writes .aether/data/pheromone-branch-export.json (TRACKED by git)
  -> This file contains the branch's signal snapshot at PR time

# The PR includes this file in the commit
git add .aether/data/pheromone-branch-export.json
```

The export file format (Section 7) is designed to be small and non-sensitive,
containing only signal metadata needed for merge-back.

### 5.5 Merge-Back Flow Diagram

```
                         POST-MERGE STATE SYNC
                         =====================

  +-------------------+    PRE-MERGE     +-------------------+
  | FEATURE BRANCH    |    EXPORT        | MERGED TO MAIN    |
  | pheromones.json   |                  |                   |
  | (gitignored)      |                  | pheromones.json   |
  |                   |   +----------+   | (canonical)       |
  | signals:          |   | EXPORT   |   |                   |
  |  [R] inj:avoid    |   | BEFORE   |   | signals:          |
  |  [R] NEW:csrf     |   | MERGE    |   |  [R] user:avoid   |
  |  [F] inj:auth     |   |          |   |  [R] sys:pattern  |
  |  [F] wrk:db       |   v          v   |  [F] user:auth    |
  |  [B] NEW:slow x3  | +----------------+|  [B] user:clean   |
  |  [B] wrk:fast x1  | | pheromone-     ||                   |
  +-------------------+ | branch-export  ||                   |
                         | .json          ||                   |
                         | (GIT-TRACKED)  ||                   |
                         +-------+--------++-------------------+
                                 |         |
                                 | READ    | READ main signals
                                 v         v
                         +---------------------------+
                         | MERGE-BACK ENGINE         |
                         |                           |
                         | FILTER mergeable:         |
                         |  [R] NEW:csrf   -> MERGE  |
                         |  [B] NEW:slow x3 -> MERGE |
                         |                           |
                         | SKIP (already on main):   |
                         |  [R] inj:avoid            |
                         |  [F] inj:auth             |
                         |                           |
                         | SKIP (worker FOCUS):      |
                         |  [F] wrk:db               |
                         |                           |
                         | SKIP (low reinforcement): |
                         |  [B] wrk:fast x1          |
                         +-------------+-------------+
                                       |
                                       | pheromone-write (to main)
                                       v
                         +---------------------------+
                         | MAIN (updated)            |
                         | pheromones.json           |
                         |  [R] user:avoid           |
                         |  [R] sys:pattern          |
                         |  [R] NEW:csrf  <---       |
                         |  [F] user:auth            |
                         |  [B] user:clean           |
                         |  [B] NEW:slow  <---       |
                         |                           |
                         | pheromone-merge-log.json   |
                         +---------------------------+
```

---

## 6. Conflict Resolution

A conflict occurs when main has a signal and the branch also has a signal of
the same type with the same `content_hash`. This means the signal was injected
from main and potentially modified on the branch (strength changed, reinforced,
or deactivated).

### 6.1 Conflict Detection

A conflict exists when, for a merge-back eligible signal:
- Main has an active signal with `type == branch_signal.type`
- AND `content_hash == branch_signal.content_hash`

### 6.2 Resolution by Signal Type

| Conflict Type | Resolution | Rationale |
|---------------|------------|-----------|
| REDIRECT (any source) | **REINFORCE** -- take max(strength_main, strength_branch), increment reinforcement_count | REDIRECTs are safety constraints. If a branch reinforced it, it means the constraint was relevant. More strength = more attention. |
| FOCUS (user source) | **SKIP** -- main's signal is authoritative. User set it on main, branch inherited it. | User signals should not be silently strengthened by worker activity. |
| FEEDBACK (user source) | **REINFORCE** -- take max strength, increment reinforcement_count | If a branch's worker also observed the same thing, the signal is validated. |
| FEEDBACK (worker source) | **REINFORCE** if branch reinforcement_count >= 2, else **SKIP** | Only promote well-established observations. |

### 6.3 Conflict Resolution Algorithm

```
function resolve_signal_conflict(main_signal, branch_signal):
    if main_signal.type == "REDIRECT":
        return REINFORCE(main_signal, branch_signal)

    if main_signal.source == "user" AND main_signal.type == "FOCUS":
        return SKIP  # user FOCUS is authoritative on main

    if main_signal.type == "FEEDBACK":
        if branch_signal.reinforcement_count >= 2:
            return REINFORCE(main_signal, branch_signal)
        else:
            return SKIP

    # Default: skip
    return SKIP

function REINFORCE(main_signal, branch_signal):
    new_strength = max(main_signal.strength, branch_signal.strength)
    new_reinforcement = main_signal.reinforcement_count + 1
    # Reset created_at to now (extends TTL from merge point)
    # Keep main's expires_at unless branch has later expiry
    new_expires = max(main_signal.expires_at, branch_signal.expires_at)
    return { signal with updated strength, reinforcement, expires }
```

### 6.4 Semantic Conflict (Same Type, Different Content)

When main has a REDIRECT for "avoid pattern X" and the branch created a new
REDIRECT for "avoid pattern Y" (different content_hash), this is NOT a conflict.
Both signals coexist. The dedup in `_pheromone_write` handles this naturally
-- different content means different signals.

### 6.5 Contradiction Detection (Advisory, Not Blocking)

A more subtle case: main has `FOCUS "use PostgreSQL"` and the branch has
`FOCUS "use SQLite"`. These are different signals (different content_hash), so
they both exist. This is a contradiction that should be surfaced to the user
but NOT block the merge.

```
CONTRADICTION RULE (advisory):
When merge-back introduces a FOCUS signal whose content contradicts an existing
FOCUS signal on main, log a warning in pheromone-merge-log.json:

{
  "warning": "contradictory_focus",
  "main_signal": {"id": "...", "content": "use PostgreSQL"},
  "branch_signal": {"id": "...", "content": "use SQLite"},
  "resolution": "both_signals_retained",
  "recommendation": "User should review and resolve via /ant:redirect or /ant:focus"
}
```

Contradiction detection is heuristic (keyword overlap, antonyms) and does not
block any merge step. It is purely advisory.

---

## 7. Data Formats

### 7.1 Pheromone Snapshot Metadata

File: `.aether/data/pheromone-snapshot.json` (gitignored, branch-local)

```json
{
  "schema": "pheromone-snapshot-v1",
  "snapshot_from_branch": "main",
  "snapshot_from_commit": "abc123def456",
  "snapshot_at": "2026-03-30T12:00:00Z",
  "injected": [
    {
      "original_id": "sig_redirect_1743300000_1234",
      "new_id": "sig_redirect_1743300100_5678",
      "type": "REDIRECT",
      "content_hash": "sha256:abc123...",
      "strength": 0.9,
      "source": "user",
      "action": "created"
    }
  ],
  "skipped": [
    {
      "original_id": "sig_focus_1743300000_9999",
      "type": "FOCUS",
      "source": "worker:builder",
      "reason": "worker-sourced FOCUS excluded from injection"
    }
  ],
  "injected_count": 4,
  "skipped_count": 3
}
```

### 7.2 Pheromone Branch Export (Pre-Merge)

File: `.aether/data/pheromone-branch-export.json` (git-tracked, committed with PR)

```json
{
  "schema": "pheromone-branch-export-v1",
  "exported_at": "2026-03-30T14:00:00Z",
  "branch_name": "feature/phase-3",
  "branch_commit": "def456abc789",
  "signals": [
    {
      "id": "sig_redirect_1743300500_1111",
      "type": "REDIRECT",
      "source": "worker:builder",
      "content_hash": "sha256:def456...",
      "content_text": "avoid raw SQL in migrations",
      "strength": 0.9,
      "created_at": "2026-03-30T13:00:00Z",
      "expires_at": "phase_end",
      "reinforcement_count": 1,
      "eligible_for_merge": true,
      "merge_reason": "new worker REDIRECT discovered on branch"
    },
    {
      "id": "sig_feedback_1743300600_2222",
      "type": "FEEDBACK",
      "source": "worker:watcher",
      "content_hash": "sha256:ghi789...",
      "content_text": "test coverage for auth module is thin",
      "strength": 0.7,
      "created_at": "2026-03-30T13:30:00Z",
      "expires_at": "phase_end",
      "reinforcement_count": 3,
      "eligible_for_merge": true,
      "merge_reason": "FEEDBACK with reinforcement_count >= 2"
    },
    {
      "id": "sig_focus_1743300700_3333",
      "type": "FOCUS",
      "source": "worker:builder",
      "content_hash": "sha256:jkl012...",
      "content_text": "database schema",
      "strength": 0.8,
      "created_at": "2026-03-30T13:45:00Z",
      "expires_at": "phase_end",
      "reinforcement_count": 0,
      "eligible_for_merge": false,
      "merge_reason": "worker-sourced FOCUS excluded from merge-back"
    }
  ],
  "total_signals": 8,
  "eligible_count": 2,
  "ineligible_count": 6
}
```

### 7.3 Merge Log

File: `.aether/data/pheromone-merge-log.json` (gitignored, appended on main)

```json
{
  "schema": "pheromone-merge-log-v1",
  "entries": [
    {
      "merged_from_branch": "feature/phase-3",
      "merged_from_commit": "def456abc789",
      "merged_at": "2026-03-30T15:00:00Z",
      "new_signals_written": [
        {
          "original_id": "sig_redirect_1743300500_1111",
          "new_id": "sig_redirect_1743301000_4444",
          "type": "REDIRECT",
          "content_hash": "sha256:def456..."
        }
      ],
      "conflicts_resolved": [
        {
          "content_hash": "sha256:abc123...",
          "type": "REDIRECT",
          "main_strength": 0.9,
          "branch_strength": 0.9,
          "resolution": "reinforced",
          "new_strength": 0.9,
          "new_reinforcement_count": 2
        }
      ],
      "warnings": [
        {
          "type": "contradictory_focus",
          "main_content": "use PostgreSQL",
          "branch_content": "use SQLite",
          "recommendation": "User should review"
        }
      ],
      "skipped_count": 5
    }
  ]
}
```

---

## 8. Integration Points

### 8.1 New Subcommands

| Subcommand | Purpose | When |
|------------|---------|------|
| `pheromone-snapshot-inject` | Inject main signals into current branch | Branch creation |
| `pheromone-export-branch` | Export branch signals for merge-back | Pre-merge (PR submission) |
| `pheromone-merge-back` | Merge branch signals into main | Post-merge on main |
| `pheromone-merge-log` | Read merge log entries | Debugging/auditing |

### 8.2 Command Integration

| Command | Integration Point | Behavior |
|---------|-------------------|----------|
| `/ant:build` | Before spawning workers | If on a non-main branch without snapshot, run `pheromone-snapshot-inject` automatically |
| `/ant:continue` | Step 4 (post-verify) | Promote eligible branch signals to hub QUEEN.md (existing behavior) |
| `/ant:seal` | Step 3.7 | Export branch pheromones for merge-back if PR workflow is active |
| `/ant:run` | After each phase merge | Run `pheromone-merge-back` on main after merge |

### 8.3 File Locking

All write operations use existing `acquire_lock`/`release_lock` from
`pkg/storage/storage.go`. The merge-back operation acquires a lock on main's
`pheromones.json` before writing, preventing concurrent merge conflicts
when multiple PRs merge in quick succession.

---

## 9. Edge Cases

### 9.1 Branch Created From Non-Main Branch

When a branch is created from another feature branch (not main), the injection
source should still be main. The rationale: main's pheromones.json is canonical,
and injecting from an intermediate branch would accumulate worker noise.

```
main -> feature/A (inject from main)
main -> feature/B (inject from main)
feature/A -> feature/A-sub (inject from main, NOT from feature/A)
```

### 9.2 Force Push / Branch Rewrite

If a branch is force-pushed, the pre-merge export may be stale. The merge-back
engine should detect this by comparing the export's `branch_commit` with the
actual merged commit SHA. If they differ, log a warning and skip merge-back
for that PR (signals may be lost, but this is safer than merging stale data).

### 9.3 Squash Merge

Squash merges lose individual commit history but the branch export file survives
(if included in the squash). The merge-back engine reads the export file, not
git history, so squash merges work correctly.

### 9.4 No Pheromones on Main

If main has no `pheromones.json` (fresh colony, no signals emitted), injection
is a no-op. The branch starts with an empty signal set, which is correct.

### 9.5 Branch Deleted Before Merge-Back

If a branch is deleted (e.g., closed without merge) before merge-back runs,
the branch export file (if committed) remains accessible via git reflog for a
grace period. If the export was never committed, signals are lost -- this is
acceptable since unmerged work is discarded by definition.

### 9.6 Concurrent PRs Merging to Main

When two PRs merge to main in quick succession, both merge-back operations may
try to write to main's `pheromones.json`. The existing file lock mechanism
handles this: the first writer acquires the lock, writes, releases. The second
writer acquires the lock, reads the now-updated main, and merges against the
latest state. Since `_pheromone_write` uses content_hash dedup, concurrent
writes of the same signal reinforce rather than duplicate.

---

## 10. Summary

| Aspect | Decision |
|--------|----------|
| Canonical pheromones | main branch `pheromones.json` |
| Injection source | Always main, never intermediate branches |
| Injected signals | All REDIRECTs (any source) + user FOCUS/FEEDBACK |
| Merged-back signals | New branch REDIRECTs + heavily reinforced FEEDBACKs |
| Conflict resolution | REDIRECT: reinforce; user FOCUS: skip; FEEDBACK: conditional reinforce |
| Pre-merge export | `pheromone-branch-export.json` (git-tracked) |
| Post-merge log | `pheromone-merge-log.json` (gitignored, main) |
| Blocking? | No -- merge-back failures are logged, never block PR merge |
| Existing mechanisms reused | `content_hash` dedup, `atomic_write`, `acquire_lock` |

---

## 11. Verification

All assertions in this document were verified against the actual codebase:

- `_pheromone_write` dedup via `content_hash`: verified at colony-prime lines 194-233
- Signal data model: verified at colony-prime lines 242-255
- `_pheromone_read` decay and filtering: verified at colony-prime lines 463-541
- `_pheromone_prime` prompt injection: verified at colony-prime lines 548-665
- `_pheromone_expire` archival: verified at colony-prime lines 1559-1638
- `.aether/data/` gitignored: verified in state-contract-design.md (Task 1.1)
- State contract branch-local rules: verified in state-contract-design.md Section 3-4

---

*Design complete. The `pheromone-snapshot-inject` and `pheromone-merge-back`
subcommands exist as deferred placeholders (cobra `Deprecated` field set).
Full implementation is a future task.*
