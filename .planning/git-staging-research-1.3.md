## 3. Git Worktrees for Multi-Agent AI Workflows

### 3.1 What Git Worktrees Are

A **git worktree** is a linked working directory that shares the same `.git` repository database as the main checkout. Each worktree has its own:
- Working directory (separate filesystem path)
- `HEAD` reference (can be on a different branch)
- Index / staging area (per-worktree `index` file)
- Uncommitted changes (completely isolated from other worktrees)

What worktrees **share** across all linked trees:
- Object database (commits, blobs, trees)
- Remote configuration and fetch state
- Branch refs (under `refs/`)
- Hooks, config, and `.gitattributes`

**Key command surface:**
```bash
git worktree add ../feature-branch feature-branch   # Create linked worktree
git worktree list                                     # Show all worktrees
git worktree remove ../feature-branch                 # Clean up
git worktree prune                                    # Remove stale entries
```

Unlike cloning a repository multiple times, worktrees share the object store -- no duplicate downloads, no sync issues, and commits made in any worktree are immediately visible to all others (once pushed to a shared ref).

### 3.2 Critical Limitations and Gotchas

| Limitation | Detail |
|-----------|--------|
| **Same branch restriction** | Cannot check out the same branch in two worktrees simultaneously. `--force` overrides this but risks corruption. |
| **Shared refs contention** | Operations that modify shared refs (`git fetch`, `git push`, `git gc`) can contend across worktrees. Running `git fetch` in two worktrees at the same moment risks lock file conflicts. |
| **index.lock files** | Each worktree has its own index lock (`.git/worktrees/<name>/index.lock`), but interrupted operations leave stale locks that block further work. |
| **Submodule restrictions** | Worktrees containing submodules cannot be moved. |
| **Pruning hazard** | Manually deleting a worktree directory leaves stale metadata; must run `git worktree prune`. |
| **Folder overhead** | Each worktree is a full checkout of the project files on disk. For large monorepos this multiplies disk usage. |
| **No main worktree removal** | The original worktree (bare or initial clone) cannot be removed. |

### 3.3 Current Industry Usage in AI Coding (2025-2026)

Git worktrees have become the **de facto standard** for running multiple AI coding agents in parallel. The pattern emerged organically in late 2024, gained traction through 2025, and by early 2026 has been adopted into official tooling.

#### The Standard Pattern

1. Developer creates a worktree per task/feature branch
2. Each worktree gets its own AI agent instance (Claude Code, Cursor, Aider, etc.)
3. Agents work in complete filesystem isolation
4. Human reviews and merges results from each worktree

**Practitioners report running 5-10 parallel agents** -- some locally on a MacBook using separate worktree directories, others via cloud sandboxes (E2B) or remote sessions.

#### Tool Ecosystem

| Tool | Description |
|------|-------------|
| **Worktrunk** | CLI for git worktree management designed for AI agent workflows. Three core commands make worktrees as easy as branches. |
| **parallel-cc** | Coordinates parallel Claude Code sessions using git worktrees + E2B cloud sandboxes. |
| **Crystal** | Desktop app to run multiple Claude Code / Codex sessions in isolated worktrees. |
| **git-worktree-runner (CodeRabbit)** | Bash-based manager with editor + AI tool integration (Cursor, VS Code, Zed, Aider, Claude Code). |
| **opencode-worktree** | Zero-friction plugin for OpenCode; spawns tmux windows per worktree. |
| **ccswarm** | Rust-based multi-agent orchestration with git worktree isolation, specialized agent roles, and TUI monitoring. |

#### Cursor 2.0 Parallel Agent Mode

Cursor 2.0 natively supports **Parallel Agent Mode**: up to 8 agents, each in its own git worktree, working on separate branches simultaneously. This is the first major IDE to ship worktree-based agent isolation as a first-class feature.

#### Claude Code Swarm Mode (2026)

Anthropic's official Claude Code Swarm Mode (released early 2026 alongside Sonnet 5) uses git worktrees as its isolation mechanism. Each agent in a swarm operates in an independent working directory. Changes are only merged into the main branch after passing tests. This validates worktrees as the industry-standard approach.

### 3.4 Applicability to Aether's Architecture

This is the critical analysis. Aether's architecture has a specific constraint that distinguishes it from the tools above: **all workers (Builder, Watcher, Scout, etc.) operate within a single Claude Code session, sharing the same working directory.**

#### Current Architecture: Workers WITHIN a Single Session

**Verdict: Worktrees do NOT help within a single colony session.**

Why not:
1. **Shared working directory.** All Aether workers (spawned via the `Task` tool) inherit the parent's working directory. There is no mechanism to give a sub-agent a different `cwd`.
2. **Task tool limitations.** The `Task` tool spawns sub-agents within the same Claude Code process. These are not separate OS processes with independent filesystem views -- they are logical sub-conversations sharing the same shell environment.
3. **Single git checkout.** All workers read from and write to the same checked-out branch. A Builder modifying `src/api.ts` while another Builder modifies `src/auth.ts` works fine (different files), but two workers touching the same file would conflict regardless of worktrees.
4. **Git operations contend.** If Worker A runs `git add` while Worker B runs `git status`, they hit the same index. Worktrees don't help because all workers share the same worktree.

**The Aether colony model is analogous to multiple developers sitting at the same computer, taking turns using the keyboard.** Worktrees solve the problem of multiple developers on different computers; they don't solve the shared-keyboard problem.

#### Cross-Session Usage: Multiple Colony Instances

**Verdict: Worktrees WOULD help for running multiple independent Aether colonies in parallel.**

Scenario: A developer wants to run Colony A on "implement auth system" and Colony B on "refactor database layer" simultaneously.

```
repo/                          # Main checkout -- Colony A
repo-worktrees/
  auth-feature/                # Worktree -- Colony A works here
  db-refactor/                 # Worktree -- Colony B works here
```

Each colony gets its own worktree, its own branch, and its own Claude Code session. This is exactly the pattern the industry has standardized on. Aether doesn't need special support for this -- the user just creates worktrees manually (or uses Worktrunk/parallel-cc) and runs `/ant:init` in each.

#### Future Architecture: Workers with Isolated Worktrees

**Verdict: Worktrees COULD enable true parallel file isolation if Aether's architecture evolves.**

A future Aether version could potentially:

1. **Queen creates worktrees for each Builder.** Before spawning workers, the Prime Worker creates a worktree per parallelizable task. Each Builder gets instructions to `cd` into its worktree.
   - **Problem:** The `Task` tool sub-agents share the parent's working directory. Even if you `cd` in the sub-agent's bash, subsequent tool calls may not respect it.

2. **Workers use branch-per-task with merge coordination.** Each Builder works on a separate branch within the same worktree, staging changes independently.
   - **Problem:** Git only supports one checked-out branch per worktree. You cannot have two workers on different branches in the same directory.

3. **External orchestration layer.** Rather than sub-agents via `Task`, Aether spawns actual separate Claude Code processes (like ccswarm or parallel-cc do). Each process gets its own worktree.
   - **Tradeoff:** Loses the single-session simplicity. Requires an external runner (tmux, shell scripts, or a dedicated orchestrator).
   - **This is effectively what Claude Code Swarm Mode does natively.**

4. **Hybrid model.** Keep the current shared-directory model for coordination (Scout, Watcher, Route-Setter) but isolate Builders into worktrees when they touch different file domains.
   - **Most promising** for Aether's evolution, but requires Claude Code to support spawning sub-agents with different working directories, which it currently does not.

### 3.5 Comparison: Aether vs Worktree-Based Systems

| Dimension | Aether (Current) | Worktree-Based (ccswarm, Cursor 2.0, Swarm Mode) |
|-----------|-------------------|--------------------------------------------------|
| **Isolation model** | Shared directory, logical separation | Physical directory isolation per agent |
| **Conflict risk** | High for same-file edits | Zero (until merge) |
| **Git contention** | Single index, potential lock conflicts | Per-worktree index, no lock conflicts |
| **Communication** | Implicit (shared files, constraints.json) | Explicit (must merge/message) |
| **Setup cost** | Zero (just spawn) | Per-worktree creation, branch management |
| **Context sharing** | Instant (same files visible) | Delayed (must commit + merge/cherry-pick) |
| **Agent spawning** | Task tool (lightweight) | Separate processes (heavyweight) |
| **Merge burden** | None during work, conflicts at commit | Deferred merge conflicts post-work |

### 3.6 Recommendation

**For Aether's current architecture (v3.0), git worktrees are NOT applicable within a colony session.** The shared-directory, single-session model is a fundamental design choice that trades isolation for simplicity and instant context sharing. This is a valid tradeoff.

**What Aether SHOULD do instead for git safety within a shared directory:**
1. Use careful file-level task decomposition (already done via Route-Setter)
2. Implement sequential git operations with a coordination layer (e.g., only the Prime Worker or Queen commits)
3. Use git stash or checkpoint commits between phases (already partially done with `aether-checkpoint`)
4. Avoid parallel writes to the same file (task decomposition responsibility)

**Where worktrees DO matter for Aether:**
1. **Cross-colony parallelism.** Document the pattern of using worktrees to run multiple independent colony sessions. This is a user workflow, not an Aether feature.
2. **Future architecture consideration.** If Claude Code's `Task` tool ever supports per-sub-agent working directories, Aether could assign worktrees to Builders for true parallel file isolation. Monitor Claude Code Swarm Mode's approach as a reference implementation.
3. **Competition awareness.** Cursor 2.0's Parallel Agent Mode and Claude Code Swarm Mode both use worktrees. Aether's value proposition is the single-session colony metaphor with emergent organization -- this is orthogonal to worktree isolation, not in competition with it. They solve different problems.

**Bottom line:** Worktrees solve inter-agent filesystem isolation. Aether's current design intentionally chooses intra-session coordination over isolation. These are complementary strategies, not conflicting ones. The right question is not "should Aether use worktrees?" but "should Aether's commit strategy account for the fact that it does NOT have worktree isolation?" -- and the answer is yes, which is the focus of the broader git staging research.

### 3.7 Sources

- [Nx Blog: How Git Worktrees Changed My AI Agent Workflow](https://nx.dev/blog/git-worktrees-ai-agents)
- [Nick Mitchinson: Using Git Worktrees for Multi-Feature Development with AI Agents](https://www.nrmitchi.com/2025/10/using-git-worktrees-for-multi-feature-development-with-ai-agents/)
- [Dennis Somerville: Parallel Workflows with Git Worktrees and Multiple AI Agents](https://medium.com/@dennis.somerville/parallel-workflows-git-worktrees-and-the-art-of-managing-multiple-ai-agents-6fa3dc5eec1d)
- [Steve Kinney: Using Git Worktrees for Parallel AI Development](https://stevekinney.com/courses/ai-development/git-worktrees)
- [Mike Mason: AI Coding Agents in 2026](https://mikemason.ca/writing/ai-coding-agents-jan-2026/)
- [incident.io: Shipping Faster with Claude Code and Git Worktrees](https://incident.io/blog/shipping-faster-with-claude-code-and-git-worktrees)
- [Claude Code Official Docs: Common Workflows](https://code.claude.com/docs/en/common-workflows)
- [Worktrunk CLI](https://github.com/max-sixty/worktrunk)
- [ccswarm: Multi-Agent Orchestration](https://github.com/nwiizo/ccswarm)
- [Crystal: Parallel AI Sessions](https://github.com/stravu/crystal)
- [parallel-cc: Parallel Claude Code Management](https://github.com/frankbria/parallel-cc)
- [CodeRabbit git-worktree-runner](https://github.com/coderabbitai/git-worktree-runner)
- [paddo.dev: Claude Code's Hidden Multi-Agent System](https://paddo.dev/blog/claude-code-hidden-swarm/)
- [Cursor 2.0 Parallel Agent Mode](https://blog.meetneura.ai/parallel-agent-mode/)
- [Git Worktree Official Documentation](https://git-scm.com/docs/git-worktree)
- [GitHub spec-kit: Native Worktree Support for Concurrent Agent Execution](https://github.com/github/spec-kit/issues/1476)
