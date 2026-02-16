## 7. GitHub Integration Opportunities

### 7.1 Current State Assessment

Aether currently has **zero GitHub integration**. All git usage is strictly local:

| Current Capability | Where | What It Does |
|---|---|---|
| `git add -A && git commit` | `build.md` Step 3 | Pre-phase safety checkpoint |
| `git stash push` | `swarm.md` Step 3 | Pre-autofix checkpoint |
| `git diff --stat` | `continue.md` Step 1.5 | Read-only verification |
| `git rev-parse --git-dir` | `build.md` Step 3 | Detect if in a git repo |
| `git log`, `git diff` | `dream.md` Step 1 | Codebase awareness for Dreamer |

No commands reference `gh`, `git push`, `git branch`, `git remote`, GitHub URLs, or any forge-specific tooling. The colony operates as if git is a local-only version control system.

**Key design facts:**
- The `aether-utils.sh` utility script contains zero git operations.
- The colony state lives in `.aether/data/COLONY_STATE.json` (local file, not synced to any remote service).
- No branch creation, switching, or management occurs anywhere in the system.
- The `~/.claude/rules/git.md` user rule explicitly states: "Do not push unless explicitly asked."
- Aether supports two host environments: **Claude Code** (has `gh` CLI access) and **OpenCode** (may not have `gh`).

### 7.2 Environment Capabilities

#### GitHub CLI (`gh`) Availability in Claude Code

The `gh` CLI (v2.86.0) is available and authenticated in this Claude Code environment. Key commands relevant to Aether:

| Command | Purpose | Aether Relevance |
|---|---|---|
| `gh pr create` | Create pull request | Post-project PR creation |
| `gh pr list` / `gh pr view` | View PRs | Status awareness |
| `gh issue create` | Create GitHub issue | Flag-to-issue sync |
| `gh issue list` | List issues | Import external issues as flags |
| `gh release create` | Create release with tag | Post-milestone tagging |
| `gh repo view` | Repository info | Detect if GitHub remote exists |
| `gh api` | Raw API access | Any custom integration |

**Critical constraint:** `gh` requires authentication (`gh auth login`) and a GitHub remote. If the user's repo has no GitHub remote, all `gh` commands will fail.

#### OpenCode Limitations

OpenCode may not have `gh` installed or authenticated. Any GitHub integration must be designed as an **optional enhancement** that degrades gracefully. OpenCode users would still get full colony functionality via git-native operations only.

### 7.3 How Other AI Tools Handle GitHub Integration

#### Claude Code (Native)

Claude Code's built-in system prompt instructs it to use `gh` for PR creation and management. Claude Code GitHub Actions (v1.0, GA September 2025) enables:
- `@claude` mentions in PRs/issues trigger automated responses
- PR creation from issues with `ai-pr` labels
- Automated code review on PR submission
- Branch creation and code editing driven by issue descriptions
- Configuration via `CLAUDE.md` in repository root

**Aether takeaway:** Claude Code already knows how to use `gh`. Aether does not need to teach it new tools; it needs to orchestrate the right moments to invoke them.

#### Aider

Aider's approach to git is deeply integrated but stays git-native:
- Every edit auto-commits with descriptive messages (similar to Aether's checkpoint pattern)
- `/undo` reverts the last commit instantly
- Branch-aware: adapts context when the user switches branches
- GitHub integration is via a separate GitHub Action (`aider-github-action`), not built into the CLI tool itself
- AiderDesk adds git worktree support for parallel isolated development

**Aether takeaway:** Aider keeps git-native by default and pushes GitHub-specific features to external workflows. This is a good model: the core tool should work without GitHub.

#### Cursor

Cursor's approach leans heavily into platform integration:
- BugBot: automated PR code review directly in GitHub
- Background Agents can work on multiple tasks in parallel (analogous to Aether's spawn model)
- Linear integration lets agents pick up issues directly
- Plans to deepen JIRA, GitHub Issues, and CI/CD integrations in 2026

**Aether takeaway:** Cursor's deep integration makes sense for a commercial product with a large team. Aether should be more cautious since it targets individual developers who may not want or need tight platform coupling.

### 7.4 Integration Opportunities Analysis

---

#### Opportunity A: PR Creation After Project Completion

**Description:** When all phases complete (Step 2.5 in `continue.md`), offer to create a GitHub PR summarizing the entire project. The PR body would be auto-generated from `COLONY_STATE.json` (goal, phases, learnings, completion report).

**How it would work:**
1. At project completion, detect if a GitHub remote exists: `git remote get-url origin 2>/dev/null`
2. Detect if on a feature branch (not `main`/`master`): `git branch --show-current`
3. If on a feature branch with a GitHub remote, offer: "Create a PR for this work?"
4. Generate PR body from colony state: goal, phases completed, key files, learnings
5. Execute: `gh pr create --title "..." --body "..." --base main`

**Assessment:**

| Dimension | Score | Notes |
|---|---|---|
| Feasibility | 5/5 | `gh pr create` is well-documented, simple to invoke |
| Effort | 2/5 | Low effort: template the PR body, one command to create |
| Value | 4/5 | High value for teams using feature-branch workflow |
| Git-native? | No | GitHub-specific (`gh pr create`) |

**Fallbacks:**
- GitLab: `glab mr create` (GitLab CLI has near-identical syntax)
- Bitbucket: No standard CLI; could output a URL template for the user to open
- Local-only: Skip silently; display "Project complete" without PR prompt
- Universal: Generate a `PULL_REQUEST.md` file with the PR body text, let user copy-paste

**Recommendation:** Implement as opt-in at project completion. Detection-based: only offer if GitHub remote is detected AND user is on a non-default branch AND `gh` is available.

---

#### Opportunity B: Branch Management (Feature Branch at Init)

**Description:** When `/ant:init` runs, optionally create a feature branch (`aether/<goal-slug>`) so all colony work happens off `main`. At project completion, the branch is ready for PR/merge.

**How it would work:**
1. At `/ant:init`, detect current branch
2. If on `main`/`master`, offer: "Create a feature branch for this work?"
3. If yes: `git checkout -b aether/<slugified-goal>`
4. All subsequent checkpoint commits happen on this branch
5. At project completion, branch is ready for merge/PR

**Assessment:**

| Dimension | Score | Notes |
|---|---|---|
| Feasibility | 5/5 | `git checkout -b` is universal git |
| Effort | 2/5 | Simple branch creation + slug generation |
| Value | 5/5 | Protects `main` from WIP; enables clean PR workflow |
| Git-native? | **Yes** | Pure git; works everywhere |

**Fallbacks:**
- Works identically on all platforms (git-native)
- Users already on a feature branch: skip (detect and respect existing branch)
- Non-git repos: skip (already handled by `git rev-parse --git-dir` check)

**Recommendation:** Strongly recommended. This is the highest-value, lowest-effort opportunity and is fully git-native. It should be the first integration implemented. The key design question is whether to create the branch automatically or prompt the user. Given the "do not commit unless explicitly asked" user rule, prompting is safer.

---

#### Opportunity C: Issue Sync (Colony Flags to GitHub Issues)

**Description:** Sync `/ant:flag` blockers to GitHub Issues, and optionally import GitHub Issues as colony flags. Bidirectional sync: when a blocker flag is created, create a corresponding GitHub Issue; when the flag is resolved, close the issue.

**How it would work:**
1. When `flag-add` creates a blocker, optionally run: `gh issue create --title "..." --body "..." --label "aether-blocker"`
2. Store the GitHub issue number in the flag's metadata
3. When `flag-resolve` resolves a flag, run: `gh issue close <number> --comment "Resolved: <message>"`
4. Reverse sync: `gh issue list --label "aether-flag"` to import external issues as colony flags

**Assessment:**

| Dimension | Score | Notes |
|---|---|---|
| Feasibility | 3/5 | Bidirectional sync is complex; one-way (flags to issues) is simple |
| Effort | 4/5 | Moderate: requires metadata storage, error handling, sync state |
| Value | 3/5 | Useful for teams, but solo devs (Aether's primary audience) rarely need this |
| Git-native? | No | GitHub-specific (`gh issue create/close`) |

**Fallbacks:**
- GitLab: `glab issue create` (similar CLI)
- Bitbucket: No standard CLI for issues
- Local-only: Flags remain local (current behavior, which is fine)
- Universal: Export flags to a `FLAGS.md` file that can be committed and shared

**Recommendation:** Low priority. Implement one-way (flags to issues) as an optional enhancement only if user explicitly enables it via a config setting. Bidirectional sync adds too much complexity for the solo-developer use case. A simpler alternative: at project completion, include unresolved flags in the PR body.

---

#### Opportunity D: Branch Detection (Adjust Behavior by Branch Context)

**Description:** Detect whether the user is on `main`/`master` vs. a feature branch and adjust colony behavior accordingly. For example: warn if building directly on `main`, adjust checkpoint strategy based on branch, suggest branch creation if on default branch.

**How it would work:**
1. At `/ant:build` or `/ant:init`, run: `git branch --show-current`
2. Compare against default branch: `git symbolic-ref refs/remotes/origin/HEAD 2>/dev/null | sed 's@.*/@@'`
3. If on default branch:
   - Warning: "You're building on main. Consider creating a feature branch."
   - Offer: `git checkout -b aether/<goal-slug>`
4. If on feature branch:
   - Note in status: "Working on branch: <name>"
   - Enable PR creation at completion

**Assessment:**

| Dimension | Score | Notes |
|---|---|---|
| Feasibility | 5/5 | Simple git commands, no external dependencies |
| Effort | 1/5 | Minimal: a few lines of shell in the build/init commands |
| Value | 4/5 | Prevents accidental main pollution; enables smarter workflows |
| Git-native? | **Yes** | Pure git; works everywhere |

**Fallbacks:**
- Works identically on all platforms (git-native)
- If no git repo: already handled; no detection needed
- If no remote: skip default-branch detection, use heuristic (main or master)

**Recommendation:** Strongly recommended. Extremely low effort, high safety value. Should be implemented alongside Opportunity B (branch management). Together they form a coherent "branch-aware colony" feature.

---

#### Opportunity E: Release Tagging After Milestone Completion

**Description:** After all phases complete, optionally create a git tag or GitHub Release marking the milestone. The tag could follow semver or use a custom scheme (e.g., `aether/v1-<goal-slug>`).

**How it would work:**
1. At project completion (Step 2.5), detect if tagging is desired
2. Git-native tag: `git tag -a "v<version>" -m "Aether colony: <goal>"`
3. GitHub Release (if remote exists): `gh release create "v<version>" --title "<goal>" --notes "<completion-report>"`

**Assessment:**

| Dimension | Score | Notes |
|---|---|---|
| Feasibility | 4/5 | Simple, but version numbering requires a convention or user input |
| Effort | 2/5 | Low effort for basic tagging; moderate for smart versioning |
| Value | 2/5 | Nice-to-have; most solo devs don't use formal releases for WIP |
| Git-native? | **Partially** | `git tag` is native; `gh release create` is GitHub-specific |

**Fallbacks:**
- GitLab: `glab release create` (similar)
- Bitbucket: Manual release creation
- Local-only: `git tag` works without any remote
- Universal: Offer `git tag` as default; `gh release` only if GitHub remote detected

**Recommendation:** Low priority. Implement the git-native `git tag` version as a simple opt-in at project completion. GitHub Releases are overkill for the typical Aether use case. Could be useful if Aether evolves to support multi-project orchestration or CI/CD triggers.

---

### 7.5 Additional Opportunity: Remote Push at Milestone

An implicit sixth opportunity is worth noting: offering `git push` at natural milestone points (phase completion or project completion). This is currently forbidden by user rules ("Do not push unless explicitly asked"), but the user could explicitly opt in during the runtime verification gate.

| Dimension | Score | Notes |
|---|---|---|
| Feasibility | 5/5 | `git push` is trivial |
| Effort | 1/5 | One line of code |
| Value | 3/5 | Prevents work loss if local machine fails; enables remote collaboration |
| Git-native? | **Yes** | Pure git |

**Recommendation:** Do NOT implement as auto-push. However, listing "push to remote" as a suggestion in the project completion output is reasonable. The user can copy-paste `git push` if they want.

### 7.6 Compatibility Matrix

| Opportunity | Claude Code | OpenCode | GitHub | GitLab | Bitbucket | Local-only |
|---|---|---|---|---|---|---|
| A. PR creation | Full | Degraded* | Full | Via `glab` | Manual | N/A |
| B. Branch management | Full | Full | Full | Full | Full | Full |
| C. Issue sync | Full | Degraded* | Full | Via `glab` | Limited | N/A |
| D. Branch detection | Full | Full | Full | Full | Full | Full |
| E. Release tagging | Full | Full | Full (gh release) | Via `glab` | Manual | `git tag` only |

*OpenCode "degraded" means the feature is unavailable if `gh` CLI is not installed, but the colony continues to function without it.

**Key principle:** All git-native features (B, D) work everywhere. GitHub-specific features (A, C, E-partial) are strictly additive enhancements that never block colony functionality.

### 7.7 Detection Strategy

Before invoking any GitHub-specific command, Aether should run a detection chain:

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

This chain produces an environment profile that determines which features are available. The detection should run once at `/ant:init` and be cached in `COLONY_STATE.json` under a new `git_context` field:

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
    "on_default_branch": false
  }
}
```

### 7.8 Prioritized Recommendations

Listed in order of implementation priority (factoring in value, effort, and git-nativeness):

| Priority | Opportunity | Type | Effort | Value | Rationale |
|---|---|---|---|---|---|
| **1** | D. Branch detection | Git-native | 1/5 | 4/5 | Near-zero effort, prevents accidents, prerequisite for B |
| **2** | B. Branch management | Git-native | 2/5 | 5/5 | Highest value; protects main; enables PR workflow |
| **3** | A. PR creation | GitHub-specific | 2/5 | 4/5 | Natural complement to B; high value for teams |
| **4** | E. Release tagging (git tag only) | Git-native | 2/5 | 2/5 | Low effort, nice milestone marker |
| **5** | C. Issue sync (one-way) | GitHub-specific | 4/5 | 3/5 | Moderate complexity; only valuable for team workflows |
| **6** | E. GitHub Releases | GitHub-specific | 3/5 | 2/5 | Overkill for most users; defer |
| **7** | C. Issue sync (bidirectional) | GitHub-specific | 5/5 | 3/5 | High complexity; defer to future version |

**The recommended implementation path:**
1. **Phase 1 (git-native foundation):** Implement D (branch detection) and B (branch management) together. These require zero external dependencies and work for 100% of users. The detection chain from Section 7.7 provides the foundation for all future integrations.
2. **Phase 2 (GitHub enhancement):** Implement A (PR creation) as an opt-in feature at project completion. Only activate when the detection chain confirms GitHub remote + `gh` availability.
3. **Phase 3 (polish):** Add E (git tagging) and consider C (one-way flag-to-issue sync) based on user demand.

### 7.9 Design Principles for Implementation

1. **Git-native first, forge-specific second.** Every feature should have a git-native fallback. GitHub is an enhancement layer, never a requirement.

2. **Detection, not assumption.** Never assume the user has GitHub, `gh`, a remote, or even git. Always detect and degrade gracefully.

3. **Opt-in, not opt-out.** Given the user's explicit rule "Do not push unless explicitly asked," all remote-affecting operations must be prompted. Local git operations (branch, tag) can be offered at natural decision points.

4. **Respect existing branch.** If the user is already on a feature branch when running `/ant:init`, respect it. Do not create a new branch or switch branches without asking.

5. **Portable across hosts.** Features must work in both Claude Code and OpenCode. If a feature requires `gh`, it must be clearly marked as Claude Code-only or require the user to install `gh` manually.

6. **Composable with existing commit classification.** Branch management and PR creation should integrate with the commit classification system proposed in Section 4.4 (Safety / Progress / Milestone commits). Milestone commits on a feature branch naturally become PR-ready.
