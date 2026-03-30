# Knowledge Gaps

## Open Questions
- Q1 (40%): Git worktree patterns — good coverage of Claude Code, Cursor, Windsurf, ComposioHQ patterns. Need: deeper conflict resolution strategies beyond single-writer, WorktreeCreate hook patterns, shared resource locking mechanisms, Devin/Copilot Workspace worktree specifics.
- Q2 (35%): PR-based AI coding workflows — covered 6 major tools. Need: deeper PR description formatting standards, test evidence inclusion patterns, merge automation specifics (no tool auto-merges), success rate factors beyond Copilot's 40-60%.
- Q3 (35%): Aether architecture mapping — initial mapping done: 7 gates → status checks, wave orchestration → parallel PRs, review agents → PR comments, continue flow → PR lifecycle. Need: concrete design for phase-as-PR vs task-as-PR granularity, pheromone propagation across branches, midden tracking across PRs, colony-prime caching strategy for CI, interactive gate (Runtime Verification) adaptation.
- Q4 (60%): Review automation pipeline — SIGNIFICANTLY IMPROVED. Now have: gate aggregation via composite job pattern, auto-fix patterns (direct commit vs stacked PR), deterministic readiness detection (Good To Go), LLM-ready thread management (gh-pr-review), 5-tier pipeline composition mapped to Aether. Remaining gaps: concrete GitHub Actions YAML for Aether's pipeline, how Aether's midden failure tracking feeds back into PR review comments, Tier 2 agent review timing (how long do agent reviews take in CI?), cost estimation for running 5 agent reviews per PR.
- Q5 (25%): Branch naming and safeguards — naming conventions documented, GitHub Ruleset safeguards identified. Need: rollback strategy design, partial failure handling (batch of PRs where some pass/fail), long-lived branch management, lock file merge driver patterns.

## Contradictions
- Nick Mitchinson notes agent-created worktrees show "slightly less consistent results" vs human-established boundaries, but Claude Code's native worktree support is designed specifically for automated agent creation. Need to investigate whether Claude Code's implementation has solved the consistency issue.
- No AI coding tool currently auto-merges PRs — all require human review as a gate. This may conflict with Aether's autopilot (/ant:run) goal of fully autonomous build-verify-advance cycles. Design decision needed: should Aether's PR workflow allow auto-merge after all gates pass, or always require human approval?
- Aether's Gate 6 (Runtime Verification) is interactive (user confirms app runs). In a CI/PR pipeline, this either becomes a smoke test action or a blocking PR review comment requiring user response. The current 4-option interactive prompt doesn't map to async PR workflows.
- Good To Go's AMBIGUOUS classification (needs human judgment) creates a tension with Aether's autopilot mode — what happens when a PR has AMBIGUOUS comments and /ant:run is in autonomous mode?

## Discovered Unknowns
- How does `.worktreeinclude` interact with Aether's `.aether/data/` directory (which is local-only)?
- Do the 10 existing agent worktrees in Aether represent successful parallel work or abandoned/stale state?
- How should worktree-per-phase vs worktree-per-agent vs worktree-per-task be decided?
- What happens when two agents modify the same file in different worktrees — Clash solves detection, but what's the resolution strategy?
- Copilot Coding Agent has 40-60% success rate — what determines success vs failure?
- How do instruction files (CLAUDE.md, REVIEW.md) propagate into worktree-based agent workflows?
- Should Aether's quality gates run as GitHub Actions checks or as local pre-merge agents?
- How should colony state (COLONY_STATE.json, pheromones.json) be synchronized across parallel worktree branches?
- Can colony-prime context be cached in CI to avoid re-computation per PR check?
- How should the midden failure tracker work when failures happen in different PR branches?
- What branch naming convention should Aether adopt: tool-native (worktree-<name>), colony-aware (ant/<phase>/<task>), or generic (ai/<agent>/<task>)?
- CodeRabbit's .coderabbit.yaml overrides all UI settings — does this create a maintenance burden alongside Aether's existing CLAUDE.md/workers.md config?
- Sourcery's static rules are Python/JS only — how does this limit its utility for polyglot Aether projects?
- How should the 5-tier pipeline handle Tier 2 agent reviews that timeout or exceed cost limits?
- What is the latency cost of running 5 parallel agent reviews (Watcher, Gatekeeper, Auditor, Probe, Chaos) in CI?
- Should Good To Go or a custom Aether equivalent be the readiness aggregator?

## Last Updated
Iteration 3 -- 2026-03-30T21:45:00Z
