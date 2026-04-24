---
name: safe-rollback
description: Use when reverting phase or implementation work requires dependency-aware rollback and recovery planning
type: colony
domains: [git-operations, risk-management, version-control]
agent_roles: [keeper, watcher, medic]
workflow_triggers: [medic, build]
task_keywords: [rollback, revert, undo, back out, recovery]
priority: normal
version: "1.0"
---

# Safe Rollback

## Purpose

Safe git revert with dependency awareness. Uses the phase manifest to identify exact commits, checks downstream impact before reverting, and supports granular rollback targets.

## When to Use

- A phase was implemented incorrectly and needs to be redone
- A plan within a phase introduced a breaking change
- User says "revert", "rollback", "undo phase", or "go back to before..."
- Post-deployment issues require reverting specific work
- Before re-implementing a phase that had fundamental design flaws

## Instructions

### Pre-flight Checks

1. Read `.aether/data/phase-manifest.json` to understand the commit mapping:
   ```json
   {
     "phases": {
       "1": { "commits": ["abc1234", "def5678"], "status": "complete", "depends_on": [] },
       "2": { "commits": ["ghi9012"], "status": "complete", "depends_on": ["1"] },
       "3": { "commits": ["jkl3456", "mno7890"], "status": "in-progress", "depends_on": ["1", "2"] }
     }
   }
   ```

2. Identify what the user wants to revert:
   - `--last N`: Revert the last N commits
   - `--phase NN`: Revert all commits for phase NN
   - `--plan NN-MM`: Revert commits for plan steps MM within phase NN

3. **Dependency check**: For each commit being reverted, scan all subsequent phases for files that import, reference, or depend on code introduced by those commits. This uses git diff-tree to find changed files per commit, then grep for imports/references.

4. **Impact report**: Before reverting, present:
   - Commits to be reverted (with messages)
   - Files that will change (with line counts)
   - Downstream phases affected (with specific dependencies)
   - Risk level: `safe` (no downstream), `caution` (downstream exists but unaffected), `dangerous` (downstream will break)

5. **Confirmation gate**: If risk is `dangerous`, require explicit user confirmation. If `safe`, proceed automatically. If `caution`, warn but proceed.

### Execute Rollback

1. Create a rollback branch: `rollback/phase-{N}-{timestamp}`
2. Execute `git revert --no-commit` for each commit in reverse chronological order
3. If conflicts arise:
   - Do NOT auto-resolve
   - List conflicting files
   - Pause and ask user for guidance
4. Commit with message:
   ```
   revert: phase {N} -- {reason}

   Reverts: {commit hashes}
   Reason: {user-provided reason}
   ```
5. Update `.aether/data/phase-manifest.json` to mark the phase as `reverted`
6. Create `.aether/data/rollback-log.jsonl` entry with full details

### Post-rollback

1. Verify the project builds/lints/tests pass after revert
2. Notify of any downstream phases that need attention
3. Suggest next steps: re-plan, re-implement, or adjust downstream phases

## Key Patterns

- **Never hard reset**: Always use `git revert` to create new commits that undo changes, preserving history
- **Reverse order**: Revert commits in reverse chronological order to minimize conflicts
- **Atomic per phase**: Each phase rollback is a single commit containing all reverts for that phase
- **Manifest consistency**: Always update the phase manifest after rollback to reflect reality
- **Branch safety**: Never rollback on main/master directly; always use a branch that can be reviewed

## Output Format

```
Safe Rollback -- Phase 3 (Dashboard UI)
  Commits to revert: 2 (jkl3456, mno7890)
  Files affected: 7 files, -842/+12 lines
  Downstream impact: Phase 4 depends on chart component
  Risk level: DANGEROUS -- Phase 4 imports chart.ts from Phase 3

  Proceed? [y/N]:
```

## Examples

```
# Revert entire phase
> safe-rollback --phase 3

# Revert last 2 commits
> safe-rollback --last 2

# Revert specific plan steps
> safe-rollback --plan 3-4

# Dry run (show impact without reverting)
> safe-rollback --phase 2 --dry-run
```
