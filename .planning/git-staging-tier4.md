> **STATUS: NOT IMPLEMENTED** — Research artifact from Phase 2 (git staging strategy). Tier 2 was chosen for implementation instead. This document describes a design that was evaluated but never built. See `git-staging-proposal.md` for the decision rationale.

# Tier 4: Branch-Aware Colony

## Philosophy

The colony is git-aware at the branch level. It knows where it is in the repository's branch topology, creates feature branches for safety when working on `main`, and can optionally create pull requests at project completion. This is the full git-native workflow: the colony treats branches as first-class citizens throughout its lifecycle, and GitHub integration is layered on top as an optional enhancement that never gates core functionality.

Tier 4 includes everything from Tiers 1, 2, and 3 (safety stashes, milestone commits at POST-ADVANCE, progress commit opt-in) and adds branch management, environment detection, and forge integration.

---

## What Changes

### Inherited from Tiers 1-3
- **Tier 1:** Safety stashes at PRE-BUILD and PRE-SWARM
- **Tier 2:** Milestone commit prompts at POST-ADVANCE and PROJECT-COMPLETE
- **Tier 3:** Opt-in progress commits at POST-VERIFY and POST-SWARM-FIX

### New in Tier 4

| Change | Lifecycle Point | Type |
|--------|----------------|------|
| **Environment detection** | `/ant:init` | Git-native |
| **Branch detection** | `/ant:init` | Git-native |
| **Auto-create feature branch** | `/ant:init` (if on default branch) | Git-native |
| **Branch context in status output** | `/ant:status` | Git-native |
| **Commit to feature branch** | Throughout colony lifecycle | Git-native |
| **PR creation offer** | PROJECT-COMPLETE | GitHub-specific |
| **PR body auto-generation** | PROJECT-COMPLETE | GitHub-specific |
| **Branch cleanup suggestions** | PROJECT-COMPLETE | Git-native |
| **Cross-session worktree guidance** | Documentation only | N/A |

---

## Branch Strategy

### Naming Convention

Colony branches follow the pattern:

```
aether/<goal-slug>
```

The goal slug is derived from the colony goal by:
1. Lowercasing the goal string
2. Replacing spaces and non-alphanumeric characters with hyphens
3. Collapsing consecutive hyphens
4. Truncating to 50 characters (git branch names should stay reasonable)
5. Stripping trailing hyphens

Examples:
| Goal | Branch Name |
|------|-------------|
| "Build a REST API with authentication" | `aether/build-a-rest-api-with-authentication` |
| "Fix CSS layout bugs in dashboard" | `aether/fix-css-layout-bugs-in-dashboard` |
| "Create a soothing sound application" | `aether/create-a-soothing-sound-application` |

If the branch already exists (e.g., from a prior colony session with the same goal), append a numeric suffix: `aether/build-a-rest-api-2`.

### When to Create vs. Reuse Branches

| Scenario | Action |
|----------|--------|
| User is on `main`/`master` | Prompt: "Create feature branch `aether/<slug>` for this work?" |
| User is on an existing `aether/*` branch | Reuse it. Log: "Continuing on existing colony branch." |
| User is on a non-aether feature branch | Respect it. Log: "Working on branch: `<name>`. Colony will commit here." |
| User is in detached HEAD state | Warn: "Detached HEAD detected. Branch management disabled for this session." |
| Branch name already taken | Create with numeric suffix (`aether/<slug>-2`) or prompt user for a name |

### Handling User on an Existing Feature Branch

If the user is already on a feature branch (e.g., `feature/auth-system`) when they run `/ant:init`, the colony respects that choice entirely. The rationale: the user made a deliberate decision to be on that branch. The colony should not second-guess it.

Behavior:
1. Record the branch name in `git_context.current_branch`
2. Set `git_context.colony_created_branch` to `false`
3. All commits happen on the user's branch
4. At PROJECT-COMPLETE, PR creation is still offered (targeting the detected default branch)

### Branch Switching During Colony Work

If the user switches branches mid-colony (detected at the next `/ant:build` or `/ant:continue`), the colony should:

1. **Detect the change:** Compare `git branch --show-current` against `git_context.current_branch` stored in COLONY_STATE.json
2. **Warn clearly:**
   ```
   Branch changed: was `aether/build-api`, now on `feature/hotfix`.
   Colony state may not match current branch.

   Options:
     1. Continue on `feature/hotfix` (update colony context)
     2. Switch back to `aether/build-api`
     3. Abort — I'll sort this out manually
   ```
3. **Never silently continue** on a different branch. Branch divergence is a potential data integrity issue.
4. **Update `git_context`** if the user chooses to continue on the new branch.

---

## Environment Detection

At `/ant:init`, before any branch operations, run a detection chain and cache the results in `COLONY_STATE.json`. This detection runs once and informs all subsequent git operations.

### Detection Chain

```bash
# Step 1: Is this a git repo?
git rev-parse --git-dir 2>/dev/null || echo "NOT_GIT"

# Step 2: Does a remote exist?
git remote get-url origin 2>/dev/null || echo "NO_REMOTE"

# Step 3: Is the remote GitHub?
git remote get-url origin 2>/dev/null | grep -q "github.com" && echo "GITHUB" || echo "NOT_GITHUB"

# Step 4: Is gh CLI available and authenticated?
gh auth status 2>/dev/null && echo "GH_READY" || echo "GH_NOT_READY"

# Step 5: What branch are we on?
git branch --show-current 2>/dev/null || echo "DETACHED"

# Step 6: What is the default branch?
git symbolic-ref refs/remotes/origin/HEAD 2>/dev/null | sed 's@.*/@@' || echo "main"
```

### Cached State in COLONY_STATE.json

```json
{
  "git_context": {
    "is_git_repo": true,
    "has_remote": true,
    "remote_url": "https://github.com/user/repo.git",
    "forge": "github",
    "gh_available": true,
    "current_branch": "aether/build-rest-api",
    "default_branch": "main",
    "on_default_branch": false,
    "colony_created_branch": true,
    "detected_at": "2026-02-11T12:00:00Z"
  }
}
```

The `forge` field supports future extensibility:
- `"github"` — remote URL contains `github.com`
- `"gitlab"` — remote URL contains `gitlab.com`
- `"bitbucket"` — remote URL contains `bitbucket.org`
- `"other"` — has a remote but forge is unrecognized
- `null` — no remote configured

---

## GitHub Integration (Optional Layer)

All GitHub features are gated behind three conditions:
1. `git_context.forge == "github"`
2. `git_context.gh_available == true`
3. `git_context.on_default_branch == false` (must be on a feature branch)

If any condition fails, the GitHub layer is silently disabled. The colony continues with full git-native functionality.

### PR Creation at PROJECT-COMPLETE

When all phases are complete (the final POST-ADVANCE or a dedicated `/ant:complete` command):

1. **Check eligibility:** All three GitHub conditions met
2. **Prompt the user:**
   ```
   All phases complete. Create a pull request?
     1. Yes — auto-generate PR
     2. Yes — let me customize title/body
     3. No — I'll handle it manually
   ```
3. **If yes, generate PR content from colony state:**

#### PR Title
Pattern: `<goal>` (truncated to 70 chars)
Example: `Build a REST API with authentication`

#### PR Body (auto-generated template)

```markdown
## Summary

<goal from COLONY_STATE.json>

## Phases Completed

| # | Phase | Status |
|---|-------|--------|
| 1 | Project Setup | Completed |
| 2 | Authentication | Completed |
| 3 | API Endpoints | Completed |

## Key Changes

<git diff --stat against default branch>

## Colony Learnings

- <extracted from memory.phase_learnings>
- <key decisions from memory.decisions>

## Verification

- All phases passed automated verification (build, types, lint, tests)
- Runtime verification confirmed by user at each phase gate

---

Built with [Aether Colony](https://github.com/callumcowie/Aether)
```

4. **Execute:**
   ```bash
   gh pr create --title "<title>" --body "<body>" --base <default_branch>
   ```

5. **Report result:** Display the PR URL on success.

### Graceful Fallback for Non-GitHub Users

| Scenario | Fallback |
|----------|----------|
| No remote | Skip PR offer. Display: "Project complete. No remote configured." |
| Non-GitHub remote (GitLab, Bitbucket) | Skip `gh`. Display: "Project complete on branch `<name>`. Ready for merge/PR on your platform." |
| `gh` not installed or not authenticated | Skip PR creation. Display branch name and suggest manual PR. |
| User declines PR | Skip silently. Display branch name for manual workflow. |

In all fallback cases, output the equivalent `git push` and manual PR URL if determinable:
```
To create a PR manually:
  git push -u origin aether/build-rest-api
  Then open: https://github.com/<user>/<repo>/compare/main...aether/build-rest-api
```

---

## Implementation Details

### Changes to `init.md`

Insert after Step 3 (Write Colony State), before Step 4 (Write Constraints):

**New Step 3.5: Git Context Detection + Branch Setup**

```
1. Run environment detection chain (6 commands from Detection Chain section)
2. Write `git_context` field into COLONY_STATE.json
3. If `is_git_repo == false`:
   - Log: "No git repo detected. Git features disabled."
   - Set all git_context fields to null/false
   - Continue to Step 4

4. If `on_default_branch == true`:
   - Generate branch slug from goal
   - Prompt user:
     "You're on `<default_branch>`. Create feature branch `aether/<slug>`?"
       1. Yes (recommended)
       2. No — stay on <default_branch>
   - If yes:
     - Run: git checkout -b aether/<slug>
     - Update git_context.current_branch
     - Set git_context.colony_created_branch = true
     - Log: "Created branch: aether/<slug>"

5. If `on_default_branch == false`:
   - Log: "Working on branch: <current_branch>"
   - Set git_context.colony_created_branch = false
```

### Changes to `build.md`

Insert at the beginning of Step 3 (the existing checkpoint step):

**Branch consistency check:**
```
1. Read git_context from COLONY_STATE.json
2. If git_context.is_git_repo:
   - current = $(git branch --show-current)
   - If current != git_context.current_branch:
     - Display branch-switch warning (see Branch Switching section)
     - Block build until user resolves
```

### Changes to `continue.md`

**At PROJECT-COMPLETE (after all phases done):**

After the existing completion celebration output, add:

```
1. If git_context.is_git_repo && git_context.colony_created_branch:
   - Display: "Colony work is on branch: aether/<slug>"

2. If git_context.forge == "github" && git_context.gh_available:
   - Offer PR creation (see PR Creation section)

3. Display branch cleanup suggestion:
   "When you're done with this branch:
     git checkout <default_branch>
     git merge aether/<slug>    # or merge via PR
     git branch -d aether/<slug>"
```

This could live in a new **Step 2.6** in `continue.md` (after the existing Step 2.5 completion output), or in a new `/ant:complete` command if the project completion flow grows complex enough to warrant separation.

### New `aether-utils.sh` Functions

Three new subcommands added to the utility layer:

#### `detect-git-context`

```bash
# Usage: detect-git-context
# Returns: Full git_context JSON object
# Pure detection — no side effects, no branch creation
```

Runs the 6-step detection chain and outputs JSON. Called once at `/ant:init`.

#### `create-colony-branch`

```bash
# Usage: create-colony-branch <goal_string>
# Returns: {branch: "aether/<slug>", created: true|false, error: null|"..."}
# Side effect: creates and checks out a new branch
```

Handles slug generation, collision detection (appending `-2`, `-3`, etc.), and the actual `git checkout -b`. Returns the final branch name.

#### `create-pr`

```bash
# Usage: create-pr <title> <body> <base_branch>
# Returns: {url: "https://...", number: N, created: true|false, error: null|"..."}
# Requires: gh CLI authenticated, GitHub remote
# Side effect: pushes current branch, creates PR
```

Handles the push + PR creation in one call. If the branch has not been pushed, runs `git push -u origin <branch>` first. Wraps `gh pr create` with error handling.

### Git-Native vs. GitHub-Specific (Clear Separation)

| Feature | Type | Requires | Works in OpenCode |
|---------|------|----------|-------------------|
| Branch detection | Git-native | `git` | Yes |
| Feature branch creation | Git-native | `git` | Yes |
| Branch consistency check | Git-native | `git` | Yes |
| Branch cleanup suggestions | Git-native | `git` | Yes |
| `detect-git-context` util | Git-native | `git` | Yes |
| `create-colony-branch` util | Git-native | `git` | Yes |
| PR creation offer | GitHub-specific | `gh` + GitHub remote | No (graceful skip) |
| PR body auto-generation | GitHub-specific | `gh` + GitHub remote | No (graceful skip) |
| `create-pr` util | GitHub-specific | `gh` + GitHub remote | No (graceful skip) |

The line is clean: everything above the PR creation layer is universal git. Everything below requires GitHub. OpenCode users get full branch management but no PR creation (unless they install `gh` independently).

---

## Cross-Session Worktree Guidance

Per research finding 1.3: git worktrees are NOT applicable within a single Aether colony session but ARE useful for running multiple independent colony sessions in parallel.

Tier 4 does not implement worktree support. Instead, it documents the pattern for users who want parallel colonies:

### Recommended User Workflow (documentation only)

```bash
# Create a worktree for a second colony
git worktree add ../my-repo-auth aether/auth-system

# In terminal 1: existing colony in the main checkout
cd my-repo
/ant:init "Build API endpoints"

# In terminal 2: second colony in the worktree
cd ../my-repo-auth
/ant:init "Implement authentication"

# Each colony works independently on its own branch
# Merge results when both complete
```

This guidance would appear in project documentation or in a help command, not in the colony's active workflow. It is informational, not automated.

---

## Effort Estimate

**Medium**

Rationale:
- **Low-effort components** (60% of work):
  - `detect-git-context` utility: ~30 lines of bash, straightforward detection chain
  - Branch detection in init.md: ~15 lines of prompt logic
  - Branch consistency check in build.md: ~10 lines
  - Branch cleanup suggestions: Static text output
  - `create-colony-branch` utility: ~40 lines including slug generation and collision handling

- **Medium-effort components** (40% of work):
  - PR body auto-generation: Requires reading COLONY_STATE.json, extracting phase data, formatting markdown, handling edge cases (empty phases, no learnings)
  - `create-pr` utility: Push + PR creation + error handling + fallback messaging
  - Branch-switch detection and user prompting in build.md/continue.md
  - Testing the full flow across: git repo, non-git repo, GitHub remote, non-GitHub remote, `gh` available, `gh` unavailable

- **No high-effort components.** All individual pieces are well-understood git/gh operations. The complexity is in the combinatorial testing across environments, not in the code itself.

- **Estimated implementation time:** 2-3 colony phases (one for git-native branch management, one for GitHub PR integration, one for testing and edge cases).

---

## Risk Assessment

### Branch Conflicts

**Risk: Low-Medium.** If the colony creates a branch at init and the user manually creates commits or branches in another terminal, merge conflicts could arise when they try to merge the colony's branch.

**Mitigation:** The colony only commits at well-defined points (Tier 2 milestone commits). Between those points, files are uncommitted in the working tree. The risk is equivalent to a human developer working on a feature branch -- standard git workflow.

### User Confusion

**Risk: Medium.** Users unfamiliar with branch workflows may not understand why the colony is offering to create a branch, or may be surprised to find themselves on a different branch after `/ant:init`.

**Mitigation:**
- Always prompt before creating a branch (never auto-create silently)
- Display the current branch in `/ant:status` output
- Include the branch name in the completion output
- Provide explicit cleanup instructions at project end

### Force-Push Risks

**Risk: Low.** The colony never runs `git push --force`. The `create-pr` utility uses only `git push -u origin <branch>` (normal push with tracking). If the push fails due to divergent history, the colony reports the error and suggests manual resolution.

**Mitigation:** No force-push commands exist anywhere in the implementation. This is a hard constraint.

### Stale `git_context`

**Risk: Low.** The git context is detected once at `/ant:init` and cached. If the user changes remotes, renames branches, or modifies their git config during a colony session, the cached context becomes stale.

**Mitigation:** The branch consistency check at each `/ant:build` catches the most critical staleness case (branch switching). For other changes (remote URL, default branch), staleness is unlikely during a single colony session and the consequences are minor (PR targets wrong base branch -- easily corrected).

### `gh` Authentication Expiry

**Risk: Low.** If `gh auth status` passes at init but the token expires during a long colony session, the PR creation at project completion will fail.

**Mitigation:** The `create-pr` utility handles `gh` failures gracefully: display the error, provide the manual fallback (push command + compare URL), and let the user resolve auth independently. The colony does not retry auth.

### Accidental Commits to Default Branch

**Risk: Low (Tier 4 specifically reduces this risk).** If the user declines branch creation at init, they remain on `main` and all commits go to `main`. This is the current behavior (Tiers 1-3) and is an explicit user choice.

**Mitigation:** The prompt at init clearly explains the consequence: "No — stay on `<default_branch>`" makes it clear that commits will land on the default branch. The branch detection warning in `/ant:build` provides a secondary reminder.

---

## User Impact

**Significant workflow change.** Tier 4 introduces branch management into the colony lifecycle, which changes the user's mental model from "colony works in my current directory" to "colony works on a branch in my current directory."

### How to Introduce Gradually

1. **Tier 4 should be opt-in initially.** Do not enable branch management by default. Let users discover it or enable it via a config flag (e.g., `aether.git.branch_management: true` in constraints or a global config).

2. **Start with detection only.** Before enabling branch creation, ship just the `detect-git-context` detection at init + branch display in status. This is zero-impact: it adds information without changing behavior.

3. **Then add branch creation prompts.** Once detection is stable, add the "Create feature branch?" prompt at init. The prompt is opt-in by nature (user must say "yes").

4. **Then add PR creation.** Once branch management is stable and users are comfortable with the branch workflow, add the PR creation offer at project completion.

5. **Never make any of this mandatory.** A user who declines all prompts should experience identical behavior to Tier 3. Branch management is additive, not replacive.

### Rollout Phases

| Phase | What Ships | User Impact |
|-------|-----------|-------------|
| 4a: Detection | `detect-git-context` at init, branch name in status | Zero — informational only |
| 4b: Branching | Feature branch prompt at init, consistency check at build | Low — opt-in prompt, can always decline |
| 4c: GitHub | PR creation at project-complete | Low — opt-in prompt, GitHub users only |

---

## Trade-offs

### Full Git Integration vs. Complexity

| Benefit | Cost |
|---------|------|
| Protects `main` from WIP commits | Adds branch management concepts to a system designed for simplicity |
| Enables clean PR workflow | Users must understand branches to use this effectively |
| Auto-generated PRs save time | PR body generation adds template complexity and maintenance burden |
| Branch context prevents accidents | Detection chain adds ~1s to init time |
| Cross-platform fallbacks | More code paths to test (git, GitHub, GitLab, no-remote, no-git) |

### Completeness vs. User Surprise

The colony's metaphor is biological (ants, queens, workers). Introducing git branches and pull requests is a domain mismatch: ants don't do version control. The risk is that Tier 4 makes the colony feel like a git automation tool rather than an organic build system.

**Mitigation:** Keep git operations as invisible as possible. The branch creation prompt should feel like "preparing the workspace," not "entering git workflow mode." The PR creation should feel like "sharing your work," not "running gh commands." Language matters.

### GitHub Coupling vs. Universality

Adding GitHub-specific features creates a two-tier user experience: GitHub users get PR creation, non-GitHub users do not. This is acceptable because:
1. The feature gap is small (one prompt at project end)
2. The core colony experience is identical regardless of forge
3. Non-GitHub users can still push and create PRs manually with the provided instructions
4. The architecture is extensible to other forges (`glab` for GitLab, etc.) if demand warrants it

### Branch Isolation vs. Simplicity

Creating a feature branch at init adds safety but also adds a step the user must understand and eventually clean up. For short-lived colonies (single-phase fixes), branch creation may feel like overhead.

**Mitigation:** The prompt at init lets the user decline. For quick fixes, "No — stay on main" is one keystroke away. The colony should adapt its prompting based on project scale: single-phase plans could default to "stay on current branch" while multi-phase plans could default to "create feature branch."

---

## Summary

Tier 4 transforms the colony from git-unaware to branch-native, with GitHub as an optional enhancement layer. The key design principles are:

1. **Git-native first.** Branch detection and creation work everywhere git works.
2. **Detection, not assumption.** The environment detection chain runs once and informs all decisions.
3. **Prompt, never auto-act.** Branch creation and PR creation are always user-prompted.
4. **Graceful degradation.** Every GitHub feature has a git-native or manual fallback.
5. **Additive, not replacive.** Declining all Tier 4 prompts yields identical behavior to Tier 3.

The implementation surface is well-bounded: three new utility functions, modifications to two existing commands (init.md, continue.md), one new COLONY_STATE.json field, and a branch consistency check in build.md. The effort is Medium, the risk is Low-Medium, and the user impact is managed through a phased rollout strategy.
