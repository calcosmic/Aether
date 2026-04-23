# Midden Cross-Branch Collection Strategy

> Task 1.3: Design post-merge ingestion, idempotency, revert handling,
> and cross-PR failure pattern detection for the midden system.
> Author: Weld-19 (Builder)
> Date: 2026-03-30
> Depends on: Task 1.1 state-contract-design.md
> Verified against: `cmd/` midden commands and the Go `aether` runtime

---

## 1. Problem Statement

In the PR-based workflow (Task 1.0), agents build on isolated feature branches.
Each branch has its own `.aether/data/midden/midden.json` (gitignored, branch-local).
When a PR merges, the branch's midden entries are left behind -- they never reach main.

This means:
- Failure patterns discovered on branch `feature/phase-3` are invisible on `main`
- The auto-REDIRECT threshold (3+ same-category entries) never fires across branches
- A reverted PR's failure entries persist silently on the abandoned branch
- Cross-PR systemic issues (same category failing on multiple PRs) go undetected

This design defines how branch-local midden entries flow into a shared,
merge-aware failure record.

---

## 2. Current State Summary

### 2.1 Midden Entry Schema

Each entry created by `_midden_write` (in `aether midden-*` commands) has this shape:

```json
{
  "id": "midden_1711785600_12345",
  "timestamp": "2026-03-30T12:00:00Z",
  "category": "resilience",
  "source": "gatekeeper",
  "message": "Test flaked 3 times in CI pipeline",
  "reviewed": false
}
```

Additional fields added by other operations:
- `tags: []` -- added by `midden-tag`
- `acknowledged: true`, `acknowledged_at`, `acknowledge_reason` -- added by `midden-acknowledge`

### 2.2 Current Auto-REDIRECT Mechanism

The `error-pattern-check` command (aether CLI line 1707) checks
`COLONY_STATE.json` `errors.records` for categories with `count >= 3`.
This operates on branch-local state only -- no cross-branch visibility.

### 2.3 Existing Ingestion Pattern

`_midden_ingest_errors` (midden Go commands line 188) demonstrates the ingestion
pattern: read from a source file, create midden entries, move the source
file to `.ingested`. This is the precedent for post-merge ingestion.

---

## 3. Design: Post-Merge Collection Protocol

### 3.1 New Subcommand: `midden-collect`

A new subcommand `_midden_collect` that runs on `main` after a PR merges.
It reads midden entries from the merged branch and ingests them into main's
midden with provenance tracking.

**Usage:**
```
midden-collect --source-branch <branch> --merge-commit <sha> [--dry-run]
```

### 3.2 Where Branch Midden Lives

When a PR branch is merged:
1. The branch's working tree still exists (git does not delete it immediately)
2. The branch ref still exists until garbage collected
3. We can checkout the branch's `.aether/data/midden/midden.json` via:
   ```
   git show <branch>:.aether/data/midden/midden.json
   ```
   Note: `.aether/data/` is gitignored, so this will NOT work.

**Resolution:** Since midden data is gitignored, we cannot use `git show`.
Instead, we use the worktree path. When agents work in worktrees (per the
state contract), the worktree directory persists until explicitly removed.

The collection protocol requires the **worktree path** as input:

```
midden-collect --worktree-path /path/to/repo-worktree-1 \
               --branch-name feature/phase-3 \
               --merge-commit abc123 \
               [--dry-run]
```

### 3.3 Collection Flow

```
                   POST-MERGE COLLECTION FLOW
                   ==========================

  +------------------+         +------------------+
  | Feature Branch   |         | Main Branch      |
  | (worktree)       |         | (repo root)      |
  |                  |         |                  |
  | midden.json      |         | midden.json      |
  |   [entry_A]      |         |   [entry_X]      |
  |   [entry_B]      |         |   [entry_Y]      |
  |   [entry_C]      |         |                  |
  +--------+---------+         +--------+---------+
           |                            ^
           |  midden-collect             |
           |  --worktree-path ...        |
           +----------------------------+
                                        |
                                        v
                               +------------------+
                               | Collection Log   |
                               | (main's data/)   |
                               |                  |
                               | collected-       |
                               |   merges.json    |
                               +------------------+
```

### 3.4 Entry Enrichment

When an entry is collected from a branch, it gets enriched with provenance
metadata before being written to main's midden:

```
ORIGINAL ENTRY (branch):          ENRICHED ENTRY (main):
========================          ========================
{                                  {
  "id": "midden_17117_123",          "id": "midden_17117_123",
  "timestamp": "...",                "timestamp": "...",
  "category": "resilience",          "category": "resilience",
  "source": "gatekeeper",            "source": "gatekeeper",
  "message": "...",                  "message": "...",
  "reviewed": false                  "reviewed": false,
}                                    "collected_from": "feature/phase-3",
                                     "collected_at": "2026-03-30T14:00:00Z",
                                     "merge_commit": "abc123def",
                                     "original_entry_id": "midden_17117_123"
                                   }
```

**Key design choice:** The original `id` is PRESERVED, not regenerated.
This is the foundation of idempotency (see Section 4).

---

## 4. Idempotency Guarantees

### 4.1 The Idempotency Problem

Post-merge hooks or CI can re-run. If `midden-collect` runs twice for the
same merge commit, it must not create duplicate entries.

### 4.2 Idempotency Strategy: Merge Commit Fingerprint

```
IDEMPOTENCY CHECK FLOW
======================

  midden-collect --merge-commit abc123 --branch-name feature/phase-3
        |
        v
  [1] Read collected-merges.json from main's .aether/data/
        |
        v
  [2] Search for fingerprint: merge_commit + branch_name
        |
        +-- FOUND  -->  Skip collection. Return {"status":"already_collected",
        |               "merge_commit":"abc123", "entries_collected": N}
        |
        +-- NOT FOUND  -->  Proceed with collection (Section 3.4)
                                |
                                v
                           [3] For each entry in branch midden.json:
                                |
                                +-- Check if entry id already exists in
                                |   main's midden.json
                                |
                                +-- If NOT present: enrich + append
                                |
                                +-- If present: skip (already collected
                                |   by prior run or duplicate id)
                                |
                                v
                           [4] Write fingerprint to collected-merges.json:
                                {
                                  "merge_commit": "abc123",
                                  "branch_name": "feature/phase-3",
                                  "collected_at": "2026-03-30T14:00:00Z",
                                  "entries_collected": 3,
                                  "entries_skipped": 0,
                                  "fingerprint": "sha256(branch+commit+count)"
                                }
                                |
                                v
                           [5] Return result JSON
```

### 4.3 Fingerprint Format

```json
{
  "merge_commit": "abc123def456",
  "branch_name": "feature/phase-3",
  "collected_at": "2026-03-30T14:00:00Z",
  "entries_collected": 3,
  "entries_skipped_dup": 0,
  "fingerprint": "sha256(feature/phase-3|abc123def456|3)"
}
```

The fingerprint is `sha256(branch_name + "|" + merge_commit + "|" + entry_count)`.
This catches the case where additional entries were added to the branch's
midden after the first collection attempt (rare but possible if collection
runs while the worktree is still active).

### 4.4 Collected Merges Storage

```
.aether/data/midden/collected-merges.json
```

Schema:
```json
{
  "version": "1.0.0",
  "merges": [
    {
      "merge_commit": "abc123def456",
      "branch_name": "feature/phase-3",
      "collected_at": "2026-03-30T14:00:00Z",
      "entries_collected": 3,
      "entries_skipped_dup": 0,
      "fingerprint": "sha256hash"
    }
  ]
}
```

### 4.5 Dual-Layer Idempotency

| Layer | Mechanism | Catches |
|-------|-----------|---------|
| 1. Merge fingerprint | Check `collected-merges.json` for same merge_commit + branch_name | Same merge collected twice |
| 2. Entry ID dedup | Skip entries whose `id` already exists in main's midden | Partial collection, orphaned entries |

Layer 1 is the fast path (single file lookup).
Layer 2 is the safety net (handles edge cases where Layer 1 fails or data is
partially written).

---

## 5. Reverted PR Handling

### 5.1 The Revert Problem

When a PR is reverted, the code changes are undone, but the midden entries
that were collected during the original merge remain in main's midden.
These entries may describe failures that no longer apply.

### 5.2 Revert Detection

```
REVERT DETECTION FLOW
=====================

  git log --merges --ancestry-path --grep="Revert" main
        |
        v
  [1] Parse revert commit message for original merge commit SHA
     Standard format: "Revert Merge pull request #42 from user/feature"
        |
        v
  [2] Look up original merge in collected-merges.json
        |
        +-- FOUND  -->  Mark merge as "reverted" (see 5.3)
        |
        +-- NOT FOUND  -->  No action (pre-collection revert or external)
```

### 5.3 Revert Actions

When a revert is detected for a previously collected merge:

```
REVERT HANDLING ACTIONS
=======================

  Original merge: abc123 (feature/phase-3, 3 entries collected)
  Revert commit: def456

  [1] UPDATE collected-merges.json entry:
      {
        "merge_commit": "abc123",
        "reverted_by": "def456",
        "reverted_at": "2026-03-30T16:00:00Z",
        "status": "reverted"
      }

  [2] TAG (not delete) collected entries in main's midden:
      For each entry with collected_from == "feature/phase-3"
      AND merge_commit == "abc123":
        - Add tag: "reverted:def456"
        - Set: "reviewed": false (force re-review)
        - Do NOT delete -- retention policy (see 5.4)

  [3] Return summary:
      {
        "revert_commit": "def456",
        "original_merge": "abc123",
        "entries_tagged": 3,
        "entries_deleted": 0
      }
```

**New subcommand:** `midden-handle-revert --revert-commit <sha>`

### 5.4 Why Tag, Not Delete

| Approach | Pro | Con |
|----------|-----|-----|
| Delete entries | Clean midden | Loses history; pattern may recur |
| Tag entries | Full audit trail; searchable | Requires query filter for active analysis |
| Soft-delete flag | Both worlds | More complex queries |

**Decision: Tag-based approach.** Entries tagged with `reverted:<sha>` are:
- Excluded from auto-REDIRECT threshold counting (by filtering on tags)
- Still visible in `midden-review --include-reverted` for audit
- Still searchable via `midden-search` for pattern analysis
- Automatically acknowledged after 30 days via `midden-prune`

### 5.5 Closed-Without-Merge Handling

When a PR is closed without merging (abandoned work):

```
CLOSED PR HANDLING
==================

  Scenario: feature/phase-3 PR #42 closed without merge

  [1] NO collection happens (midden-collect only runs on merge)
  [2] Branch midden entries stay on the branch (harmless -- branch-local)
  [3] If worktree is cleaned up, entries are lost (acceptable -- abandoned work)
  [4] No action required in main's midden
```

This is the simplest case: closed PRs are a no-op for midden collection.
The branch-local midden entries represent work that was not accepted and
should not pollute main's failure tracking.

---

## 6. Cross-PR Failure Pattern Detection

### 6.1 The Cross-PR Detection Problem

The existing auto-REDIRECT fires when 3+ entries share the same category
within a single branch's midden. With the PR workflow, failures of the same
category may be spread across multiple PRs, each below threshold individually
but collectively indicating a systemic issue.

```
EXAMPLE: Systemic "resilience" failures across 4 PRs

  PR #38 (feature/auth):   2 resilience entries  (below threshold)
  PR #39 (feature/api):    1 resilience entry    (below threshold)
  PR #40 (feature/ui):     2 resilience entries  (below threshold)
  PR #41 (feature/db):     1 resilience entry    (below threshold)
                           --------
  TOTAL across PRs:        6 resilience entries  (ABOVE threshold!)
```

### 6.2 Cross-PR Aggregation

**New subcommand:** `midden-cross-pr-analysis [--category <cat>] [--window <days>]`

```
CROSS-PR ANALYSIS FLOW
======================

  midden-cross-pr-analysis --window 14
        |
        v
  [1] Query main's midden for all entries in the time window:
      WHERE timestamp >= NOW() - 14 days
      AND reviewed != true
      AND tags DO NOT contain "reverted:*"
        |
        v
  [2] Group entries by category:
      {
        "resilience": [
          {"id": "...", "collected_from": "feature/auth", "message": "..."},
          {"id": "...", "collected_from": "feature/api", "message": "..."},
          ...
        ],
        "quality": [...],
        ...
      }
        |
        v
  [3] For each category, compute cross-PR metrics:
      {
        "category": "resilience",
        "total_entries": 6,
        "unique_prs": 4,         <-- KEY METRIC: spread across PRs
        "entries_per_pr": {
          "feature/auth": 2,
          "feature/api": 1,
          "feature/ui": 2,
          "feature/db": 1
        },
        "time_span_days": 12,
        "cross_pr_score": 0.85   <-- confidence this is systemic
      }
        |
        v
  [4] Threshold check:
      IF unique_prs >= 2 AND total_entries >= 3:
        --> Flag as "cross-pr-systemic"
        --> Auto-emit REDIRECT pheromone to hub
      IF unique_prs >= 3 AND total_entries >= 5:
        --> Flag as "cross-pr-critical"
        --> Escalate to human notification
        |
        v
  [5] Return analysis result
```

### 6.3 Cross-PR Score Formula

```
cross_pr_score = (unique_prs / max_prs) * 0.6
               + (total_entries / max_entries) * 0.4

where:
  max_prs = 5        (ceiling for PR diversity)
  max_entries = 10   (ceiling for entry volume)
```

| unique_prs | total_entries | score | classification |
|------------|---------------|-------|---------------|
| 1 | 3 | 0.40 | single-PR (local issue) |
| 2 | 3 | 0.52 | cross-pr-systemic |
| 3 | 5 | 0.70 | cross-pr-systemic |
| 4 | 6 | 0.84 | cross-pr-critical |
| 5 | 10 | 1.00 | cross-pr-critical |

### 6.4 Integration with Auto-REDIRECT

The cross-PR analysis augments (does not replace) the existing per-branch
auto-REDIRECT mechanism:

```
FAILURE PATTERN DETECTION (TWO TIERS)
=====================================

  TIER 1: Per-Branch (existing)
  =============================
  _midden_write checks category count within current branch
  IF category_count >= 3:
    --> Emit REDIRECT to branch-local pheromones.json
    --> Strength: 0.7
    --> Scoped to current branch only

  TIER 2: Cross-PR (new)
  ======================
  midden-cross-pr-analysis runs periodically (on merge, on /ant-status)
  IF cross_pr_score >= 0.5 AND unique_prs >= 2:
    --> Emit REDIRECT to HUB pheromones (or QUEEN.md wisdom)
    --> Strength: proportional to score (0.5-1.0)
    --> Visible to ALL branches
    --> Includes context: "Pattern detected across N PRs in M days"

  TRIGGER POINTS:
  - After every midden-collect (post-merge)
  - On /ant-status (periodic check)
  - On /ant-continue (review phase)
```

### 6.5 Analysis Output Schema

```json
{
  "analysis_timestamp": "2026-03-30T14:00:00Z",
  "window_days": 14,
  "total_entries_scanned": 24,
  "categories": {
    "resilience": {
      "total_entries": 6,
      "unique_prs": 4,
      "entries_per_pr": {
        "feature/auth": 2,
        "feature/api": 1,
        "feature/ui": 2,
        "feature/db": 1
      },
      "time_span_days": 12,
      "cross_pr_score": 0.84,
      "classification": "cross-pr-critical",
      "auto_redirect_emitted": true,
      "redirect_content": "resilience failures across 4 PRs in 12 days"
    },
    "quality": {
      "total_entries": 2,
      "unique_prs": 1,
      "entries_per_pr": {"feature/auth": 2},
      "time_span_days": 1,
      "cross_pr_score": 0.28,
      "classification": "single-pr",
      "auto_redirect_emitted": false
    }
  },
  "systemic_categories": ["resilience"],
  "recommendation": "Investigate resilience pattern across feature/auth, feature/api, feature/ui, feature/db"
}
```

---

## 7. Complete Flow Diagram

```
COMPLETE MIDDEN CROSS-BRANCH COLLECTION LIFECYCLE
=================================================

  BRANCH CREATION          BUILD ON BRANCH           POST-MERGE
  ================         ===============           ==========

  git checkout -b          Agent runs build          PR merged to main
  feature/phase-3          |                         |
       |                   |                         v
       v                   v                   +----+----+
  .aether/data/         _midden_write           | CI runs |
  midden.json           creates entries         |         |
  (empty, inherited     in branch-local         +----+----+
  from main's copy)     midden.json                 |
       |                   |                         v
       |                   v                   midden-collect
       |              +----+----+              --worktree-path ...
       |              | Branch  |              --merge-commit ...
       |              | midden  |                   |
       |              | .json   |                   v
       |              |         |              +----+----+
       |              | entry_1 |              | Enrich  |
       |              | entry_2 |              | entries |
       |              | entry_3 |              | with    |
       |              +---------+              | proven- |
       |                                       | ance    |
       |                                       +----+----+
       |                                            |
       |                                            v
       |                                       +----+----+
       |                                       | Check   |
       |                                       | idem-   |
       |                                       | potency |
       |                                       +----+----+
       |                                            |
       |                    +-----------------------+-----------------+
       |                    |                                         |
       |                    v                                         v
       |              Already                               Append to main's
       |              collected                             midden.json
       |              (skip)                                      |
       |                    |                                         v
       |                    v                                    +----+----+
       |              Return                               midden-cross-pr-
       |              {"status":                            analysis
       |               "already_                                   |
       |               collected"}                                v
       |                                                     +----+----+
       |                                                     | Check   |
       |                                                     | cross-  |
       |                                                     | PR      |
       |                                                     | patterns |
       |                                                     +----+----+
       |                                                          |
       |                            +-----------------------------+-----+
       |                            |                                   |
       |                            v                                   v
       |                     systemic                        no systemic
       |                     detected                        pattern
       |                            |                                   |
       |                            v                                   v
       |                     emit REDIRECT                     return
       |                     to HUB                            clean
       |                     (all branches)                    report
       |
       |
       |  REVERT SCENARIO (separate flow)
       |  ===============================
       |
       v
  git revert -m 1 <merge_sha>
       |
       v
  midden-handle-revert
  --revert-commit <sha>
       |
       v
  [1] Find original merge in collected-merges.json
       |
       v
  [2] Tag entries: "reverted:<revert_sha>"
       |
       v
  [3] Mark merge as "reverted" status
       |
       v
  [4] Re-run cross-pr analysis
       (excluded reverted entries)
```

---

## 8. Subcommand Summary

| Subcommand | Purpose | Input | Output |
|------------|---------|-------|--------|
| `midden-collect` | Ingest branch midden into main post-merge | `--worktree-path`, `--branch-name`, `--merge-commit` | Collected count, skipped count |
| `midden-handle-revert` | Tag entries from a reverted merge | `--revert-commit <sha>` | Tagged count, merge status |
| `midden-cross-pr-analysis` | Detect systemic patterns across PRs | `--category`, `--window <days>` | Category analysis, scores, recommendations |

All three are additions to `aether midden-*` commands and dispatched from
`aether CLI`.

---

## 9. Integration Points

### 9.1 When Collection Runs

| Trigger | Command | Notes |
|---------|---------|-------|
| Post-merge hook | `midden-collect` | Requires worktree path; must run before worktree cleanup |
| `/ant-continue` (on main) | `midden-collect` | Fallback if no post-merge hook; reads from last merged branch ref |
| `/ant-status` | `midden-cross-pr-analysis` | Periodic health check |

### 9.2 When Revert Handling Runs

| Trigger | Command | Notes |
|---------|---------|-------|
| Post-merge hook detects revert commit message | `midden-handle-revert` | `git log --grep="Revert"` |
| `/ant-status` | Checks for unhandled reverts | Scans `collected-merges.json` for merges without revert status that have corresponding revert commits |

### 9.3 Data Flow Integration with State Contract (Task 1.1)

```
MIDDEN DATA FLOW (aligned with state-contract-design.md Section 4.3)
====================================================================

  Branch-local (NOT merged by git):
    midden/midden.json on branch --> collected via midden-collect
    pheromones.json on branch    --> NOT copied (branch-scoped)

  Main branch (post-merge):
    midden/midden.json           <-- receives collected entries
    midden/collected-merges.json <-- tracks collection fingerprints

  Hub-global (always cross-branch):
    QUEEN.md                     <-- receives REDIRECT if cross-pr-critical
    pheromone signals            <-- REDIRECT emitted to hub for systemic patterns

  Critical failures from branch midden:
    --> Promoted to QUEEN.md REDIRECT (via cross-pr analysis)
    --> NOT directly copied to main pheromones
```

---

## 10. Edge Cases and Failure Modes

| Edge Case | Handling |
|-----------|----------|
| Worktree deleted before collection | Entries lost (acceptable -- same as abandoned PR). Log warning. |
| midden.json missing on branch | No-op. Return `{"entries_collected": 0}`. |
| midden.json corrupt on branch | Log error. Skip collection for this merge. Do not poison main's midden. |
| Concurrent collection for same merge | File lock on `collected-merges.json`. Second writer sees fingerprint and skips. |
| Same entry ID from different branches | Unlikely (uses PID + timestamp). If it occurs, second collection skips via Layer 2 dedup. |
| Branch with 0 midden entries | Record merge fingerprint with `entries_collected: 0`. Still idempotent. |
| Revert of a revert (re-revert) | Remove `reverted:*` tag. Re-run cross-pr analysis. Set merge status back to `active`. |
| `collected-merges.json` deleted | Fallback to Layer 2 (entry ID dedup). Log warning. Rebuild from entry provenance. |

---

## 11. Retention and Pruning

| Data | Retention | Pruning Trigger |
|------|-----------|-----------------|
| Collected midden entries | Indefinite (or per colony lifecycle) | `/ant-seal` archives all midden |
| `collected-merges.json` entries | 90 days after merge | `midden-prune --stale-merges` |
| Reverted entry tags | 30 days after revert | `midden-prune --reverted --age 30` |
| Cross-PR analysis cache | Not persisted | Computed on demand |

---

## 12. Verification

Assertions in this design verified against:

- `aether midden-*` commands: Entry schema confirmed from `_midden_write` (line 57-63)
- `aether CLI` line 1707: Auto-REDIRECT threshold (`>= 3`) confirmed
- `state-contract-design.md` Section 4.3: Midden is branch-local, NOT merged by git
- `.gitignore` line 80: `.aether/data/` confirmed gitignored
- `midden.template.json`: Dual schema (`signals` + `entries`) noted for backward compat

All assertions are grounded in the actual codebase as of 2026-03-30.

---

*Design complete. Next steps: implement `midden-collect`, `midden-handle-revert`,
and `midden-cross-pr-analysis` in `aether midden-*` commands (implementation tasks).*
