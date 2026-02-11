## 4. User Git Rules vs Colony Safety: Tension Analysis

### 4.1 Current User Git Rules (Verbatim)

From `~/.claude/rules/git.md`:

> - Do not commit unless explicitly asked.
> - Do not push unless explicitly asked.
> - Use concise commit messages: imperative mood, under 72 characters, focused on "why" not "what".

From `~/.claude/CLAUDE.md` (Safety section):

> - Never run destructive commands (disk wipes, rm -rf home, etc.) unless explicitly typed by user.
> - After completing a task, run available checks (tests, linters, builds) and report results.

From `DISCLAIMER.md` (Best Practices):

> 1. Use git with frequent commits
> 2. Review the colony's plan before building
> 3. Read generated code before committing
> 4. Run tests after each phase
> 5. Keep backups of important data

### 4.2 Where Aether Already Violates or Bends These Rules

Aether currently auto-commits in at least three distinct locations, none of which involve explicit user consent at the moment of commit:

#### A. `build.md` Step 3 — Pre-Phase Git Checkpoint

```bash
git add -A && git commit --allow-empty -m "aether-checkpoint: pre-phase-$PHASE_NUMBER"
```

- **Nature:** Safety checkpoint before any build work begins.
- **Purpose:** Enables rollback if the build phase causes damage.
- **Uses `git add -A`:** Stages everything, including potentially sensitive files.
- **Uses `--allow-empty`:** Will create a commit even if there are no changes.
- **Conflict with rules:** Directly violates "Do not commit unless explicitly asked." The user invoked `/ant:build`, not "please commit." The commit is an implicit side-effect.
- **Conflict with DISCLAIMER:** Contradicts "Read generated code before committing" since the checkpoint is created automatically before any review.

#### B. `swarm.md` Step 3 — Autofix Checkpoint

```bash
bash ~/.aether/aether-utils.sh autofix-checkpoint
```

- **Nature:** Creates either a git stash or records the current HEAD hash before the swarm applies fixes.
- **Purpose:** Enables rollback if the swarm's fix fails verification.
- **Mechanism:** Uses `git stash push` if there are uncommitted changes; otherwise records the commit hash. Does NOT create a new commit in most cases (stash is preferred).
- **Conflict with rules:** Milder violation. `git stash` is not a commit, but it does modify git state without explicit user consent. The rollback path (`autofix-rollback`) can invoke `git stash pop` or `git checkout`, which are state-altering operations.

#### C. `continue.md` Step 1.5 Phase 6 — Diff Review

```bash
git diff --stat
```

- **Nature:** Read-only operation. No violation.
- **Purpose:** Verification of changes before phase advancement.
- **Conflict with rules:** None. This is a read operation.

#### D. Implicit: No `git push` anywhere

Notably, Aether never auto-pushes. This respects the second rule ("Do not push unless explicitly asked"). The damage radius of auto-commits is limited to local state.

### 4.3 Analysis of the Tension

The tension is structural, not accidental. It arises from two legitimate but opposing design goals:

**Goal 1 (User Rule):** "I want full control over what enters my git history. Commits represent my conscious decisions about what code I endorse."

**Goal 2 (Colony Safety):** "Autonomous agents modifying dozens of files need rollback points. Without checkpoints, a failed build phase could leave the codebase in an unrecoverable state."

These goals are not fully reconcilable. Any automated safety system must modify state to create safety nets, but the user's rules say "don't modify state without my permission."

#### The Spectrum of Severity

Not all auto-commits are equal. There is a clear hierarchy of intrusiveness:

| Mechanism | Intrusiveness | Reversibility | Safety Value |
|-----------|--------------|---------------|-------------|
| `git diff --stat` | None (read-only) | N/A | Informational |
| `git stash push` | Low (hidden state) | Easy (`stash pop`) | High |
| `git commit --allow-empty` | Medium (visible in log) | Moderate (`reset`) | High |
| `git add -A && git commit` | High (stages everything) | Moderate (`reset`) | High |
| `git push` | Very High (public) | Difficult | N/A (never done) |

The current build.md implementation jumps straight to "High" intrusiveness (add everything + commit) without exhausting lower-intrusiveness alternatives first.

#### The `git add -A` Problem

`git add -A` is particularly concerning because it stages ALL files, including:
- `.env` files with secrets
- Large binary files
- Work-in-progress files in other branches
- Files the user deliberately chose not to track

This violates the spirit of the user's CLAUDE.md safety section, even if it doesn't violate the letter.

### 4.4 Proposed Commit Classification System

Commits in the colony context serve fundamentally different purposes. A classification system can resolve the tension by applying different consent rules to different commit types.

#### Class 1: Safety Commits (Rollback Capability)

- **Purpose:** Create a restore point before destructive operations.
- **Examples:** Pre-build checkpoint, pre-swarm checkpoint, pre-autofix checkpoint.
- **Characteristics:**
  - Created BEFORE changes, not after
  - Represent the "known good" state
  - Are intended to be temporary / potentially reset
  - The user's existing work is what's being preserved, not new AI-generated code
- **Consent model:** IMPLICIT consent is acceptable. The user invoked `/ant:build`, which is documented as creating a checkpoint. The checkpoint protects their work.
- **Preferred mechanism:** `git stash` (lower intrusiveness than commit). Only escalate to commit if stash is not viable (e.g., new untracked files that stash won't capture).
- **Message convention:** `aether-checkpoint: pre-<operation>` (already in use)

#### Class 2: Progress Commits (Recording Work Done)

- **Purpose:** Snapshot the state after a worker completes a task, for incremental rollback.
- **Examples:** After Builder completes a task, after Watcher verifies a component.
- **Characteristics:**
  - Created AFTER changes
  - Represent AI-generated code
  - The user has NOT reviewed this code yet
  - Multiple progress commits may accumulate during a build phase
- **Consent model:** OPT-IN required. These commits record unreviewed AI code into git history. The user's rule "do not commit unless explicitly asked" applies most strongly here.
- **Alternative:** Use a staging area (branch, stash, or diff file) instead of committing to the working branch. The user can review and commit after the phase.
- **If implemented:** Message convention: `aether-progress: <task-id> <description>`

#### Class 3: Milestone Commits (Phase Complete)

- **Purpose:** Mark a verified, user-approved phase as complete.
- **Examples:** After `/ant:continue` passes all gates and the user approves runtime verification.
- **Characteristics:**
  - Created AFTER verification passes
  - User has explicitly approved via runtime verification gate
  - All tests pass, all gates pass
  - Represents a meaningful project milestone
- **Consent model:** EXPLICIT consent, but could be prompted. Since the user already confirmed "Yes, tested and working" in the runtime verification gate, a follow-up "Commit this milestone?" is natural and low-friction.
- **Message convention:** `Phase <id>: <phase-name> — <one-line summary>`
- **Follows user's rule:** Imperative mood, under 72 characters, focused on "why."

### 4.5 Interaction Matrix

| Commit Class | User Rule Violation? | Acceptable? | Recommended Mechanism |
|-------------|---------------------|-------------|----------------------|
| Safety (pre-build) | Technically yes | Yes, with documentation | `git stash` preferred; commit only if stash insufficient |
| Progress (mid-build) | Yes, strongly | No, unless opted in | Branch or stash; do not commit to working branch |
| Milestone (post-verify) | No, if prompted | Yes | Prompt user; commit with descriptive message |

### 4.6 The DISCLAIMER.md Paradox

The DISCLAIMER says both:
- "Use git with frequent commits" (encourages auto-commit)
- "Read generated code before committing" (discourages auto-commit)

These are directed at different audiences:
- The first is advice to the user about their own workflow
- The second is about AI-generated code specifically

The resolution: Aether should create restore points (safety commits), but should NOT auto-commit AI-generated code to the user's branch without review. The DISCLAIMER's advice is consistent when read this way: "Commit YOUR work frequently, but review AI work before committing it."

### 4.7 Recommendations

#### R1: Downgrade safety checkpoints from commits to stashes where possible

Replace in `build.md` Step 3:
```bash
# Current (high intrusiveness)
git add -A && git commit --allow-empty -m "aether-checkpoint: pre-phase-$PHASE_NUMBER"

# Proposed (lower intrusiveness)
git stash push -m "aether-checkpoint: pre-phase-$PHASE_NUMBER" --include-untracked
```

Fallback to commit only when `git stash` is not available or fails. This preserves rollback capability while reducing git log pollution.

**Trade-off:** `git stash` is a stack, so nested stashes can be confusing. A named branch might be cleaner for multi-phase projects.

#### R2: Never auto-commit AI-generated code to the user's working branch

Progress commits (Class 2) should not happen automatically. Instead:
- Use a detached branch or temporary branch (e.g., `aether/phase-N`) for intermediate work
- Present a diff summary after the phase completes
- Let the user decide whether to commit (respecting their git.md rule)

#### R3: Prompt for milestone commits at natural consent points

The runtime verification gate in `continue.md` (Step 1.9) already has the user's attention and explicit approval. Add an optional commit prompt here:

```
Commit this milestone? (Phase N: {name})
  1. Yes — commit with standard message
  2. Yes — let me write the message
  3. No — I'll commit later
```

This respects "do not commit unless explicitly asked" because the user explicitly says "yes."

#### R4: Never use `git add -A` in automated contexts

Replace with targeted staging:
- Stage only files within `.aether/` for state commits
- Stage only files modified by the build phase for progress commits (if opted in)
- Let the user run their own `git add` for milestone commits

#### R5: Document the commit classification in the colony system

Users should understand what Aether does to their git state. The `build.md` and `swarm.md` should explicitly state what git operations they perform and why, so invoking the command constitutes informed consent.

#### R6: Preserve the "never auto-push" invariant

This is already respected and should remain an absolute rule. Auto-commits are recoverable; auto-pushes are not (practically speaking). This is the one line Aether should never cross.

### 4.8 Summary Table

| Current Behavior | Rule Tension | Proposed Change | Priority |
|-----------------|-------------|-----------------|----------|
| `build.md` auto-commits with `git add -A` | HIGH — violates "don't commit" and risks staging secrets | Switch to `git stash --include-untracked`; fallback to targeted commit | High |
| `swarm.md` uses `git stash` | LOW — stash is non-intrusive | Keep as-is; already the right approach | None |
| `continue.md` uses `git diff` | NONE — read-only | Keep as-is | None |
| No progress commits exist | N/A | Keep it this way unless user opts in | None |
| No milestone commits exist | MISSED OPPORTUNITY | Add opt-in prompt at runtime verification gate | Medium |
| No auto-push anywhere | NONE — correctly respects rule | Keep as absolute invariant | None |
