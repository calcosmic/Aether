# Research Synthesis

## Topic
PR-based, branch/worktree coding workflow for Aether's multi-agent colony system — implementation plan

## Aether Context
- 24 specialized agents (builder, watcher, gatekeeper, auditor, probe, queen, etc.)
- 7 mandatory quality gates in continue-gates.md
- Midden failure tracking with category/source/timestamp
- Pheromone signaling system (FOCUS/REDIRECT/FEEDBACK)
- Colony phases with task tracking and milestone system
- Existing worktree support: .claude/worktrees/agent-* directories (10 active)
- Slash commands: /ant:build, /ant:continue, /ant:run (autopilot)

## Findings by Question

### Q1: Git Worktree Workflows for AI Agents (confidence: 40%, status: partial)

**Claude Code Native Support:**
Claude Code has built-in `--worktree` (`-w`) flag that creates isolated worktrees at `<repo>/.claude/worktrees/<name>`, branching from `origin/HEAD`. Subagents use `isolation: worktree` frontmatter for automatic worktree creation. Auto-cleanup removes worktrees with no changes; worktrees with changes prompt user to keep/remove. `.worktreeinclude` file copies gitignored files (like `.env`) to new worktrees. [S1][S5]

**Aether's Existing Pattern:**
The Aether repo already has 10 active agent worktrees (agent-a09afdfa through agent-aea49f19), each on separate `worktree-agent-*` branches. These were created by Claude Code's subagent isolation system. [S6]

**Industry Convergence:**
All major AI coding tools converged on git worktrees for parallel agent isolation. Cursor supports up to 8 parallel agents (Feb 2026), Windsurf supports 5 parallel Cascade agents (Wave 13). [S1]

**Parallel Limits:**
No hard git limit on worktree count. Practical limits: ~5-6 with active builds on 32GB RAM, 10+ on 64GB. Developers find 2-4 manually trackable. Filesystem inode limits matter on ext4; XFS/ZFS preferred for high counts. [S4]

**Conflict Avoidance Patterns:**
(1) Each agent works in isolation on its own branch — conflicts handled at merge time. (2) Restrict agents to specific directories. (3) Prohibit force pushes and cross-branch mods. (4) Use `.claude/rules.md` configuration guards per worktree. (5) Agent-created worktrees show "slightly less consistent results" than human-established boundaries. [S2]

**Open-Source Orchestrator (ComposioHQ):**
MIT-licensed `agent-orchestrator`: each agent gets own worktree + branch + PR. Plugin architecture for runtime (tmux/Docker/K8s), agent (Claude Code/Codex/Aider), workspace (worktree/clone). CI failures auto-fixed with retry logic (default 2 attempts). Review comments routed back to agents. Dashboard on port 3000. [S3]

### Q2: PR-based AI Coding Workflows (confidence: 35%, status: partial)

**Claude Code GitHub Actions:**
Official `anthropics/claude-code-action@v1` provides full CI/CD PR workflow. Triggered by `@claude` mentions in issues/PRs. Auto-creates branches and PRs, supports skills integration, respects CLAUDE.md for coding standards. Configurable via `claude_args` (--max-turns, --model, --allowedTools). Supports direct API, AWS Bedrock, and Google Vertex AI. Auto-detects interactive vs automation mode. [S7][S14]

**GitHub Copilot Coding Agent:**
Automates branch creation (`copilot/<description>` naming), commit writing, PR opening, and PR description writing. Works asynchronously in ephemeral GitHub Actions environments. Single repo, single PR per task. Review comments route back to agent for iteration. 40-60% success rate on well-scoped issues. Cannot bypass branch protection rules (e.g., signed commits). [S8][S9]

**Devin Review:**
Auto-reviews trigger on PR open, commit push, draft-to-ready transitions, or reviewer enrollment. Confidence-based bug labeling: "Bugs" (severe/non-severe) and "Flags" (investigate/informational). Auto-fix generates suggested code changes in the diff view. Groups changes logically (not alphabetically), detects code moves/copies. CLI available via `npx devin-review {pr-url}` using git worktrees. Free for regular GitHub repos. [S10][S16]

**OpenAI Codex:**
`openai/codex-action@v1` GitHub Action: installs codex-cli, starts Responses API proxy, runs `codex exec`. Supports `@codex review` comments on PRs, automatic review on every PR when enabled. Jira-to-GitHub PR automation pipeline available. GPT-5.2-Codex recommended for code review accuracy. Works with both GitHub Actions and Jenkins for on-premises deployments. [S11][S15]

**Aider:**
Tight git integration with auto-commits using Conventional Commits, "(aider)" attribution suffix on author/committer. In-chat `/diff`, `/undo`, `/commit`, `/git` commands. GitHub Action (`mirrajabi/aider-github-action`) for issue-to-PR workflows. No built-in branch management or merge automation — operates on current branch only. Codebase map for large project navigation. [S12][S17]

**Sweep AI:**
Originally a GitHub issue-to-PR bot (YC S23): describe task in issue, Sweep plans changes, writes code, creates PR with human review gate. Open-source. Now pivoted to JetBrains AI coding assistant while maintaining GitHub workflow capabilities. [S13]

**Industry Convergence Pattern:**
All major AI coding tools converged on: (1) task/issue as input → PR as output, (2) AI creates branch + commits + PR description automatically, (3) human review as mandatory gate before merge, (4) CI/CD integration via GitHub Actions, (5) review comments routed back to agents for iteration, (6) CLAUDE.md/REVIEW.md instruction files for customizing agent behavior per repo. [S7][S8][S10][S11]

### Q3: Aether Architecture Integration (confidence: 35%, status: partial)

**Quality Gates → PR Status Checks:**
Aether's 7 mandatory continue gates map to PR status checks: (1) Spawn Enforcement — verify parallelism, (2) Anti-Pattern scan, (3) Gatekeeper supply chain security (conditional), (4) Auditor quality scoring (block if <60 or critical), (5) TDD Evidence verification, (6) Runtime Verification (interactive — needs adaptation), (7) Flags gate (no unresolved blockers). Gates 1-5 and 7 are automatable; Gate 6 requires interactive user input. [S18]

**No Existing Worktree Isolation in Agents:**
No Aether agents use `isolation: worktree` in their definitions. All 24 agents run in the main repo context. Isolation enforced through role-based tool restrictions: Watcher/Gatekeeper/Auditor are read-only (no Write/Edit/Bash), Builder writes only assigned files, Probe writes only tests. Switching to worktree-per-agent would be a new pattern, not an enhancement. [S21][S22][S23][S24]

**Wave-Based Worker Orchestration → Parallel PRs:**
Build orchestration spawns workers in dependency-ordered waves. Wave 1 (no dependencies) runs in parallel, subsequent waves sequential. Conditional agents (Oracle, Architect, Ambassador) spawn pre-wave. This maps to "parallel PRs per wave, sequential merge between waves." [S19]

**Review Agent JSON Output → PR Comments:**
Review agents produce structured JSON ideal for PR comments: Watcher returns `{verification_passed, issues_found[{severity, file, line}], quality_score}`. Auditor returns `{findings{critical/high/medium/low}, issues[{file, line, severity}], overall_score}`. Gatekeeper returns `{security_findings, licenses, version_pinning_gaps}`. Probe returns `{tests_added, coverage{lines%, branches%}, edge_cases}`. [S21][S22][S23][S24]

**Colony-Prime in PR Context:**
Colony-prime assembles pheromones, QUEEN.md wisdom, instincts, research, and skills into worker prompts. In PR workflow: colony-prime runs once per PR branch for context, again at merge time. `--compact` mode (4K chars) exists for CI-constrained environments. [S19]

**Continue Flow → PR Lifecycle:**
continue-verify (build/types/lint/tests/coverage/secrets) → PR CI checks. continue-gates (7 gates) → PR required status checks. continue-advance (learning extraction, instinct creation, QUEEN.md promotion) → post-merge hooks. continue-finalize (changelog, handoff) → post-merge automation. Verification command resolution chain (CLAUDE.md → codebase.md → heuristics) would need CI config replication. [S18][S20]

**Failure Escalation in PR Model:**
Tiered escalation: total wave failure → halt PR; partial failure → Tier 3 (auto-push fix commits to PR branch); Tier 3 fails → Tier 4 (request human review with context). Midden auto-emits REDIRECT pheromones when error category recurs 3+ times. [S19]

### Q4: Review Automation Pipeline Design (confidence: 60%, status: partial)

**CodeRabbit Pipeline:**
4-stage pipeline: (1) preprocess PR content, (2) LLM analysis, (3) post-process, (4) post review comments. Review profiles: 'chill' (bugs/security only) vs 'assertive' (full review). One-click commit suggestions + 'Fix with AI' for complex changes. Configured via `.coderabbit.yaml` with path_filters, file_path_instructions (per-glob custom rules), ignored_branch, ignored_titles. [S25][S31][S33]

**Sourcery Pipeline:**
Hybrid LLM + static analysis with multiple specialized reviewers. Validation pass reduces false positives before posting. Tracks quality, security, complexity, docs, testing. Per-function quality scores. Dashboard-based config (not file-based like CodeRabbit). Static rule comments limited to Python/JavaScript. [S26]

**Pre-Merge Conflict Detection (Clash):**
Clash (clash-sh/clash) detects conflicts between git worktrees via in-memory merge-tree simulation (Rust gix library). Non-destructive. Integrates as Claude Code PreToolUse hook: `clash check <file>` before Write/Edit. Also provides `clash status` (conflict matrix), `clash watch` (live TUI), JSON output for automation. [S28]

**GitHub Actions Gate Aggregation:**
Check suites aggregate the highest-priority conclusion from all check runs — any single failure means suite failure. The composite gate job pattern uses `needs: [lint, test, security, ...]` to create a single required check that depends on all sub-jobs. Three successful statuses: success, skipped, neutral. All required checks must pass against the latest commit SHA before merge. Path-filtered workflows that skip leave checks "Pending" and block merge unless handled. Parallelizing independent jobs (lint + test + security concurrently) is recommended for speed. [S34][S35][S42]

**Auto-Fix Commit Patterns:**
Two patterns emerged: (1) Direct commit — CodeRabbit's `@coderabbitai autofix` pushes fixes directly to the PR branch. (2) Stacked PR — `@coderabbitai autofix stacked pr` creates a separate branch/PR for isolated review. The 4-step autofix flow: trigger (PR comment/checkbox) → collection (scan unresolved threads, gather "Prompt for AI Agents" blocks) → generation+verification (agent applies fixes, runs build) → delivery (commit or stacked PR). Even if verification fails, changes are delivered for iteration. This represents the shift from "passive analysis to active remediation." [S38][S41]

**Deterministic PR Readiness (Good To Go):**
Good To Go provides deterministic readiness detection for AI agents. Classifies comments as ACTIONABLE (must fix), NON_ACTIONABLE (praise, nitpicks, resolved), or AMBIGUOUS (needs human judgment). Built-in parsers for CodeRabbit, Greptile, Claude Code, Cursor. Returns 5 statuses: READY, ACTION_REQUIRED, UNRESOLVED, CI_FAILING, ERROR. JSON output: `{status, action_items[], actionable_comments[], ci_status{}, threads{}}`. Deployable as GitHub Actions required status check. Supports `/rerun-gtg` comment command. Design philosophy: "determinism over heuristics" — every PR has exactly one status at any moment. [S36][S37]

**LLM-Ready PR Thread Management (gh-pr-review):**
CLI extension providing full inline PR review comment support. Single `review view` command returns entire assembled review structure as JSON (reviews → comments → thread_comments hierarchy with thread_id, path, line, is_resolved, is_outdated). Agent workflow: (1) fetch unresolved threads, (2) reply to thread with fix evidence, (3) resolve thread programmatically. Server-side filters (--reviewer, --states, --unresolved, --tail) reduce payload for LLM token efficiency. Omits null fields, uses stable field ordering for deterministic parsing. Registers as Vercel add-skill. [S39]

**5-Tier Pipeline Composition for Aether:**
Based on all research, the optimal pipeline maps Aether's existing structures:

- **Tier 1 — CI Checks (parallel, automated):** Build, type-check, lint, test suite, coverage, secrets scan, anti-pattern scan. Maps to Aether's continue-verify 6-phase loop.
- **Tier 2 — Agent Reviews (parallel, after Tier 1 passes):** Watcher verification, Gatekeeper security audit (conditional), Auditor quality gate, Probe coverage analysis (conditional), Chaos resilience testing (conditional). Maps to Aether's continue-gates agent spawns.
- **Tier 3 — Aggregation:** Composite gate job aggregates Tier 1+2 results. Good To Go readiness check evaluates comment/thread resolution state. Maps to Aether's gate decision logic.
- **Tier 4 — Human Gate:** Required human review approval + runtime verification confirmation (Aether's Gate 6 adapted). Maps to Aether's Runtime Verification gate.
- **Tier 5 — Post-Merge:** Learning extraction, instinct creation, QUEEN.md promotion, changelog update, pheromone updates. Maps to Aether's continue-advance and continue-finalize. [S18][S20][S34][S35][S36]

**Repository Readiness Assessment (Factory.ai):**
Factory.ai's Agent Readiness framework evaluates repo maturity across 8 dimensions (style/validation, build, testing, docs, dev environment, code quality, observability, security). Level 3 ("Standardized") is the threshold for "production-ready for agents" — requiring E2E tests, maintained docs, security scanning. Binary pass/fail criteria. Progression requires 80% of criteria at each level plus all previous. Useful as a pre-flight check before enabling Aether's PR workflow automation in a new repo. [S40]

### Q5: Branch Naming, Organization, and Safeguards (confidence: 25%, status: partial)

**AI Tool Branch Naming:**
Copilot: `copilot/<description>` (changed Oct 2025 from UUIDs). Codex: `codex/<description>`. Claude Code: `worktree-<name>` (community request for configurable prefix). Community recommendation: `ai/<agent-name>/<task>` (e.g. `ai/claude/add-auth`). No tool currently allows custom prefix configuration. [S29][S30]

**GitHub Ruleset Safeguards:**
(1) Required human approval — single most important safeguard. (2) Stale approval dismissal — new commits auto-revoke approval. (3) Force push prevention — disabled by default on protected branches. (4) Code scanning as required status check. Empirical: 0 unauthorized writes landed across 50 test runs. [S27]

**Conflict Hotspots:**
Shared files (route defs, config, barrel exports) are primary conflict sources. Mitigations: single-writer rules per file, additive-only changes, frequent merges to main. Lock file divergence (package-lock.json) is underappreciated when parallel agents install packages independently. [S28][S32]

**Token and Cleanup Safety:**
AI agent CI tokens: single-repo scope, 60-minute TTL. Stale worktree references accumulate — `git worktree prune` should run periodically. [S27]

## Sources
- [S1] [Claude Code Docs — Common Workflows](https://code.claude.com/docs/en/common-workflows) (documentation, 2026-03-30)
- [S2] [Using Git Worktrees for Multi-Feature Development with AI Agents](https://www.nrmitchi.com/2025/10/using-git-worktrees-for-multi-feature-development-with-ai-agents/) (blog, 2026-03-30)
- [S3] [ComposioHQ/agent-orchestrator](https://github.com/ComposioHQ/agent-orchestrator) (github, 2026-03-30)
- [S4] [Performance Optimization for Git Worktrees](https://gitcheatsheet.dev/docs/advanced/worktrees/performance/) (documentation, 2026-03-30)
- [S5] [Boris Cherny — Built-in git worktree support for Claude Code](https://www.threads.com/@boris_cherny/post/DVAAnexgRUj) (official, 2026-03-30)
- [S6] [Aether repo — agent worktree directories](.claude/worktrees/) (codebase, 2026-03-30)
- [S7] [Claude Code GitHub Actions — Official Docs](https://code.claude.com/docs/en/github-actions) (documentation, 2026-03-30)
- [S8] [About GitHub Copilot coding agent](https://docs.github.com/en/copilot/concepts/agents/coding-agent/about-coding-agent) (documentation, 2026-03-30)
- [S9] [GitHub Copilot coding agent 101](https://github.blog/ai-and-ml/github-copilot/github-copilot-coding-agent-101-getting-started-with-agentic-workflows-on-github/) (official, 2026-03-30)
- [S10] [Devin Review — Devin Docs](https://docs.devin.ai/work-with-devin/devin-review) (documentation, 2026-03-30)
- [S11] [Codex Workflows — OpenAI Developers](https://developers.openai.com/codex/workflows) (documentation, 2026-03-30)
- [S12] [Git Integration — Aider Documentation](https://aider.chat/docs/git.html) (documentation, 2026-03-30)
- [S13] [Sweep AI — GitHub Repository](https://github.com/sweepai/sweep) (github, 2026-03-30)
- [S14] [anthropics/claude-code-action](https://github.com/anthropics/claude-code-action) (github, 2026-03-30)
- [S15] [openai/codex-action](https://github.com/openai/codex-action) (github, 2026-03-30)
- [S16] [Devin 101: Automatic PR Reviews — Cognition](https://cognition.ai/blog/devin-101-automatic-pr-reviews-with-the-devin-api) (blog, 2026-03-30)
- [S17] [mirrajabi/aider-github-action](https://github.com/mirrajabi/aider-github-action) (github, 2026-03-30)
- [S18] [Aether continue-gates playbook](.aether/docs/command-playbooks/continue-gates.md) (codebase, 2026-03-30)
- [S19] [Aether build-wave playbook](.aether/docs/command-playbooks/build-wave.md) (codebase, 2026-03-30)
- [S20] [Aether continue-verify playbook](.aether/docs/command-playbooks/continue-verify.md) (codebase, 2026-03-30)
- [S21] [Aether Watcher agent definition](.claude/agents/ant/aether-watcher.md) (codebase, 2026-03-30)
- [S22] [Aether Gatekeeper agent definition](.claude/agents/ant/aether-gatekeeper.md) (codebase, 2026-03-30)
- [S23] [Aether Auditor agent definition](.claude/agents/ant/aether-auditor.md) (codebase, 2026-03-30)
- [S24] [Aether Probe agent definition](.claude/agents/ant/aether-probe.md) (codebase, 2026-03-30)
- [S25] [CodeRabbit — AI Code Review Integration](https://www.coderabbit.ai/blog/how-to-integrate-ai-code-review-into-your-devops-pipeline) (blog, 2026-03-30)
- [S26] [Sourcery Code Review Overview](https://docs.sourcery.ai/Code-Review/Overview/) (documentation, 2026-03-30)
- [S27] [AI Agents in CI/CD — Why GitHub Rulesets Matter](https://ancuta.org/posts/ai-agents-in-your-ci-cd-why-github-rulesets-matter/) (blog, 2026-03-30)
- [S28] [Clash — Pre-merge conflict detection](https://github.com/clash-sh/clash) (github, 2026-03-30)
- [S29] [AI Branch Naming Conventions](https://mike.bailey.net.au/notes/software/git/aidock/ai-branch-naming-conventions/) (blog, 2026-03-30)
- [S30] [Copilot Branch Naming Changelog](https://github.blog/changelog/2025-10-16-copilot-coding-agent-uses-better-branch-names-and-pull-request-titles/) (official, 2026-03-30)
- [S31] [CodeRabbit IDE Extension — One-click Fix](https://www.coderabbit.ai/blog/code-with-ai-review-with-coderabbits-ide-extension-apply-fixes-in-one-click) (blog, 2026-03-30)
- [S32] [Git Worktree Conflicts with AI Agents](https://www.termdock.com/en/blog/git-worktree-conflicts-ai-agents) (blog, 2026-03-30)
- [S33] [.coderabbit.yaml configuration reference](https://gist.github.com/bemijonathan/8bc892b1e12954e45a906e0704cff86d) (documentation, 2026-03-30)
- [S34] [About Status Checks — GitHub Docs](https://docs.github.com/articles/about-status-checks) (documentation, 2026-03-30)
- [S35] [Master GitHub Actions Status Checks — Pull Checklist](https://www.pullchecklist.com/posts/github-actions-status-checks) (blog, 2026-03-30)
- [S36] [Good To Go — Deterministic PR Readiness Detection](https://dsifry.github.io/goodtogo/) (documentation, 2026-03-30)
- [S37] [dsifry/goodtogo — GitHub Repository](https://github.com/dsifry/goodtogo) (github, 2026-03-30)
- [S38] [CodeRabbit Autofix — Official Documentation](https://docs.coderabbit.ai/finishing-touches/autofix) (documentation, 2026-03-30)
- [S39] [gh-pr-review — LLM-Ready PR Review CLI Extension](https://github.com/agynio/gh-pr-review) (github, 2026-03-30)
- [S40] [Factory.ai Agent Readiness — Repository Maturity Framework](https://factory.ai/news/agent-readiness) (official, 2026-03-30)
- [S41] [The State of AI Code Review in 2026](https://dev.to/rahulxsingh/the-state-of-ai-code-review-in-2026-trends-tools-and-whats-next-2gfh) (blog, 2026-03-30)
- [S42] [How to Configure Status Checks in GitHub Actions — OneUptime](https://oneuptime.com/blog/post/2026-01-26-status-checks-github-actions/view) (blog, 2026-03-30)

## Last Updated
Iteration 3 -- 2026-03-30T21:45:00Z
