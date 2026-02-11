# Tier 1: Safety-Only (Formalize Existing)

## Philosophy

Tier 1 is the most conservative option. It changes **zero user-facing behavior**. No new commit points, no prompts, no branch management, no opt-in features. The only goal is to make the safety mechanisms that already exist **less intrusive, more reliable, and better documented**.

The guiding principle: if Aether is going to touch git state without explicit user consent, it should do so with the lightest possible footprint and with verifiable success/failure reporting.

This tier directly addresses the tension identified in Research 1.4: the `build.md` checkpoint currently uses the most intrusive mechanism available (`git add -A && git commit`) when a less intrusive alternative (`git stash push --include-untracked`) achieves the same safety goal. Tier 1 corrects this.

---

## What Changes

### Change 1: Switch `build.md` checkpoint from commit to stash

**Current** (build.md Step 3, line 85):
```bash
# If git repo detected:
git add -A && git commit --allow-empty -m "aether-checkpoint: pre-phase-$PHASE_NUMBER"
```

**Proposed:**
```bash
# If git repo detected:
git stash push --include-untracked -m "aether-checkpoint: pre-phase-$PHASE_NUMBER"
```

**Rationale:**
- `git stash` does not pollute the user's commit history (Research 1.4: intrusiveness drops from "High" to "Low")
- `--include-untracked` captures new files that plain `git stash` would miss, matching the coverage that `git add -A` previously provided
- The stash mechanism is already proven in `swarm.md` / `aether-utils.sh` where it works reliably
- Eliminates the violation of the user's git rule "Do not commit unless explicitly asked" (Research 1.4, Section 4.2A)
- Removes the `--allow-empty` noise commits that create history entries even when nothing changed

**Edge case — clean working tree:** If `git status --porcelain` returns empty (no changes to stash), skip the stash entirely and record the current HEAD hash as the checkpoint reference. This matches the existing behavior in `autofix-checkpoint`.

**Edge case — stash failure:** If `git stash push` fails (e.g., merge conflicts in progress), fall back to recording the HEAD hash and emit a warning. Do NOT fall back to `git commit` — that would reintroduce the intrusiveness this change eliminates.

### Change 2: Standardize checkpoint message format

**Current formats (inconsistent):**
- build.md: `aether-checkpoint: pre-phase-$PHASE_NUMBER`
- aether-utils.sh: `aether-autofix-$(date +%s)`

**Proposed format (unified):**
- build.md stash: `aether-checkpoint: pre-phase-$PHASE_NUMBER`
- swarm.md stash: `aether-checkpoint: pre-swarm-$SWARM_ID`

This follows the `aether-checkpoint:` namespace prefix recommended in Research 1.6 (Option B). The prefix is already partially in use and provides clean `git stash list | grep "aether-checkpoint"` filtering.

**Implementation:** Update the `autofix-checkpoint` function in `aether-utils.sh` to accept an optional label parameter instead of generating an epoch-based name.

### Change 3: Add rollback verification after each checkpoint

**Current behavior:** Both `build.md` and `swarm.md` create checkpoints but do not verify that the checkpoint was actually created successfully before proceeding with destructive work.

**Proposed:** After creating a stash checkpoint, verify it exists:

```bash
# After stash push:
stash_check=$(git stash list 2>/dev/null | head -1 | grep "aether-checkpoint" || echo "")
if [[ -z "$stash_check" ]]; then
  # Stash failed silently — warn the user
  echo "WARNING: Git checkpoint could not be created. Proceeding without rollback safety net."
fi
```

For `build.md`, this verification runs between Step 3 (checkpoint) and Step 4 (worker spawning). The build proceeds regardless — the checkpoint is a safety net, not a gate — but the user is informed if the net has a hole.

For `swarm.md`, the existing `autofix-checkpoint` function already returns a JSON result with type/ref, but the calling code does not validate that `type != "none"` before proceeding. Add a warning display when type is "none".

### Change 4: Document the rollback procedure

**Current state:** `continue.md` line 43 references "rollback to git checkpoint" but does not specify the mechanism. The user or agent must know to run the right git commands.

**Proposed:** Add explicit rollback instructions to the `build.md` output header and to `continue.md`'s stale-state handler.

For build.md, after the checkpoint output line, add:
```
Rollback: git stash pop  (if checkpoint type is "stash")
```

For continue.md stale-state handler (line 42-43), specify:
```
To rollback: Run `git stash list` to find the checkpoint, then `git stash pop <ref>`
```

This does not change behavior — it documents what already exists so users (and future agents) can act on it.

---

## What Stays the Same

- **No new commit points.** POST-ADVANCE, POST-VERIFY, SESSION-PAUSE — none of these get commit behavior. That is Tier 2+ territory.
- **No user prompts.** The system does not ask the user "Commit this?" at any point. All git operations remain silent infrastructure.
- **No branch management.** No feature branches, no `aether/phase-N` branches, no worktrees.
- **No opt-in configuration.** No settings, no flags, no user preferences for git behavior.
- **`git diff --stat` in continue.md stays as-is.** Read-only, zero tension with user rules.
- **All read-only git operations unchanged.** `git log`, `git blame`, `git diff` in dream.md and swarm.md scouts remain exactly as they are.
- **`autofix-rollback` logic unchanged.** The stash pop and reset --hard paths in aether-utils.sh continue to work as before (with the minor improvement that checkpoint names are now consistent).
- **Never auto-push.** This absolute invariant is preserved.

---

## Implementation Details

### File: `.claude/commands/ant/build.md` (and `.opencode/commands/ant/build.md` mirror)

**Lines 77-88 — Replace Step 3:**

Current:
```markdown
### Step 3: Git Checkpoint

Create a git checkpoint for rollback capability.

\```bash
git rev-parse --git-dir 2>/dev/null
\```

- **If succeeds** (is a git repo): `git add -A && git commit --allow-empty -m "aether-checkpoint: pre-phase-$PHASE_NUMBER"`
  Store the commit hash.
- **If fails** (not a git repo): Set checkpoint hash to `"(not a git repo)"`.
```

Proposed:
```markdown
### Step 3: Git Checkpoint

Create a git checkpoint for rollback capability.

\```bash
git rev-parse --git-dir 2>/dev/null
\```

- **If succeeds** (is a git repo):
  1. Check for changes: `git status --porcelain`
  2. **If changes exist**: `git stash push --include-untracked -m "aether-checkpoint: pre-phase-$PHASE_NUMBER"`
     - Verify: `git stash list | head -1 | grep "aether-checkpoint"` — warn if empty
     - Store checkpoint as `{type: "stash", ref: "aether-checkpoint: pre-phase-$PHASE_NUMBER"}`
  3. **If clean working tree**: Record `HEAD` hash via `git rev-parse HEAD`
     - Store checkpoint as `{type: "commit", ref: "$HEAD_HASH"}`
- **If fails** (not a git repo): Set checkpoint to `{type: "none", ref: "(not a git repo)"}`.

Rollback procedure: `git stash pop` (if type is "stash") or `git reset --hard $ref` (if type is "commit").
```

### File: `.aether/aether-utils.sh`

**Lines 786-808 — Update `autofix-checkpoint` function:**

Change the stash name from epoch-based to label-based:

Current (line 793):
```bash
stash_name="aether-autofix-$(date +%s)"
```

Proposed:
```bash
label="${1:-autofix-$(date +%s)}"
stash_name="aether-checkpoint: $label"
```

This allows callers to pass a descriptive label:
- `build.md` equivalent: `bash ~/.aether/aether-utils.sh autofix-checkpoint "pre-phase-3"`
- `swarm.md`: `bash ~/.aether/aether-utils.sh autofix-checkpoint "pre-swarm-$SWARM_ID"`
- Default (no argument): falls back to `aether-checkpoint: autofix-1738000000` (preserving current behavior)

**Lines 811-848 — No changes to `autofix-rollback`:**

The rollback function searches stash by name (`grep "$ref"`), which will work with the new naming format. No code changes needed here.

### File: `.claude/commands/ant/swarm.md` (and `.opencode/commands/ant/swarm.md` mirror)

**Line 59 — Update checkpoint call to pass label:**

Current:
```bash
bash ~/.aether/aether-utils.sh autofix-checkpoint
```

Proposed:
```bash
bash ~/.aether/aether-utils.sh autofix-checkpoint "pre-swarm-$SWARM_ID"
```

**Lines 306-309 — No changes to rollback call.** The rollback mechanism is unchanged.

### File: `.claude/commands/ant/continue.md` (and `.opencode/commands/ant/continue.md` mirror)

**Lines 42-43 — Add explicit rollback instructions:**

Current:
```markdown
   - Offer: continue anyway or rollback to git checkpoint
```

Proposed:
```markdown
   - Offer: continue anyway or rollback to git checkpoint
   - Rollback procedure: `git stash list | grep "aether-checkpoint"` to find ref, then `git stash pop <ref>` to restore
```

### Summary of files touched:

| File | Change Type | Lines Affected |
|------|------------|----------------|
| `.claude/commands/ant/build.md` | Modify Step 3 | ~10 lines replaced |
| `.opencode/commands/ant/build.md` | Mirror of above | ~10 lines replaced |
| `.aether/aether-utils.sh` | Update `autofix-checkpoint` stash name | ~3 lines changed |
| `.claude/commands/ant/swarm.md` | Pass label to checkpoint call | 1 line changed |
| `.opencode/commands/ant/swarm.md` | Mirror of above | 1 line changed |
| `.claude/commands/ant/continue.md` | Add rollback doc to stale handler | 1 line added |
| `.opencode/commands/ant/continue.md` | Mirror of above | 1 line added |

**Total: 7 files, ~30 lines changed.**

---

## Effort Estimate

**Low.**

Rationale:
- All changes are within existing code paths — no new functions, no new files, no new control flow
- The stash mechanism is already implemented and proven in `aether-utils.sh`
- build.md changes are swapping one git command for another in a markdown instruction template
- The `.opencode/` mirrors are exact copies, so changes are mechanical duplication
- No tests exist for the checkpoint system, so no test updates needed (this is a pre-existing gap, not introduced by Tier 1)
- No configuration system to build or maintain
- Estimated implementation time: 1-2 hours including testing

---

## Risk Assessment

### Risk 1: Stash conflicts on pop (Low probability, Medium impact)

**Scenario:** The user manually stashes something between checkpoint creation and rollback. When Aether tries to `git stash pop`, it pops the wrong stash or encounters conflicts.

**Mitigation:** The named stash search (`grep "aether-checkpoint"`) already targets the correct stash by name rather than relying on stack position. This risk exists today with the swarm checkpoint and has not caused reported issues.

### Risk 2: `--include-untracked` captures unwanted files (Low probability, Low impact)

**Scenario:** The stash captures large binary files, build artifacts, or `.env` files that the user had in their working tree but didn't intend to track.

**Mitigation:** This is the same file set that `git add -A` was previously staging and committing. Stashing them is strictly less harmful than committing them — stash contents are hidden and ephemeral, while committed files enter permanent history. The risk is not new; it is inherited from the current design and reduced by this change.

### Risk 3: Stash is accidentally dropped or cleared (Low probability, High impact)

**Scenario:** The user runs `git stash clear` or `git stash drop` between checkpoint creation and potential rollback, destroying the safety net.

**Mitigation:** This risk also exists today for swarm checkpoints and is inherent to stash-based strategies. The rollback verification step (Change 3) warns when the checkpoint is missing. Additionally, stash loss only matters if the build phase actually needs to be rolled back — in the happy path (vast majority of cases), the stash is never needed and can be safely lost.

### Risk 4: Working tree is modified between stash and build (Negligible probability)

**Scenario:** Between creating the stash checkpoint and the build phase running, something modifies the working tree.

**Mitigation:** The stash and build happen in the same command invocation (`/ant:build`), so there is no practical window for external modification.

### Risk 5: `git stash push --include-untracked` fails (Low probability, Low impact)

**Scenario:** The stash command fails due to git state issues (e.g., rebase in progress, merge conflicts).

**Mitigation:** The proposed implementation records the HEAD hash as a fallback when stash fails (matching existing `autofix-checkpoint` behavior). The build proceeds with a warning. This is equivalent to the current situation where `git commit` might also fail in the same git states.

---

## User Impact

**Minimal to zero.**

- Users will **stop seeing** `aether-checkpoint: pre-phase-N` commits in their `git log`. This is a positive change — the most common complaint with the current system would be these noisy, auto-generated commits cluttering history.
- Users will see checkpoint entries in `git stash list` instead. Stash entries are hidden by default and only visible when explicitly requested.
- The rollback procedure changes from `git reset --hard <hash>` to `git stash pop`, which is actually safer (stash pop preserves working tree changes; reset --hard destroys them).
- Warning messages may appear if a checkpoint fails to create, which is new visibility but purely informational.
- No new prompts, no new decisions, no new configuration. The user's workflow is unchanged.

---

## Trade-offs

### What you gain (vs. doing nothing)

1. **Reduced git history pollution.** No more `aether-checkpoint: pre-phase-N` commits appearing in `git log --oneline`. Stashes are invisible in normal git workflows.
2. **Eliminated rule violation.** The `build.md` checkpoint no longer violates the user's "Do not commit unless explicitly asked" rule. Stash is a different category of git state modification — it preserves the user's changes without recording them as intentional commits.
3. **Consistent checkpoint mechanism.** Both `build.md` and `swarm.md` now use the same approach (stash-based), with the same naming convention, and the same rollback procedure. Currently they use different strategies for no clear reason.
4. **Verifiable checkpoints.** The verification step catches silent failures, which currently go undetected.
5. **Documented rollback.** Users and agents know exactly how to roll back, rather than guessing.
6. **Safer rollback path.** `git stash pop` is safer than `git reset --hard` — it applies changes rather than destroying them, and it can surface merge conflicts rather than silently overwriting.

### What you lose (vs. doing nothing)

1. **Checkpoint permanence.** Stashes are ephemeral; commits are permanent. If the stash is accidentally cleared, the safety net is gone. However, in practice, stashes are rarely cleared accidentally, and the checkpoint is only needed in the rare case of a build phase failure.
2. **Checkpoint discoverability.** `git log --grep="aether-checkpoint"` no longer works for build checkpoints (only stash list shows them). Users who relied on finding checkpoints in git history will need to look in `git stash list` instead. This is unlikely to affect anyone since the checkpoints were not designed to be user-discoverable.
3. **`--allow-empty` coverage.** The current system creates a checkpoint even when the working tree is clean. The proposed system skips checkpointing for clean trees (records HEAD hash instead). This is actually correct behavior — if the tree is clean, HEAD already IS the checkpoint — but it is a behavioral change.

### What you neither gain nor lose

- **Rollback capability:** Preserved. The mechanism changes but the capability does not.
- **Safety for the happy path:** No change. Most builds succeed and the checkpoint is never referenced.
- **Remote repository state:** No change. Neither current nor proposed system touches remote.

---

## Relationship to Higher Tiers

Tier 1 is designed to be a standalone improvement that requires no further work. However, it also serves as a clean foundation for higher tiers:

- **Tier 2** (Prompted Milestones) would add opt-in commit prompts at POST-ADVANCE. It builds on Tier 1's cleaned-up checkpoint system but does not require it.
- **Tier 3** (Full Strategy) would add branch management, progress tracking, and configurable behavior. Tier 1's consistent stash-based checkpoints make Tier 3 cleaner to implement since there's one checkpoint mechanism to integrate with rather than two.

Tier 1 can be implemented independently and delivers value even if Tier 2 and Tier 3 are never pursued.
