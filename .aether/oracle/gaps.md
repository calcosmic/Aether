# Knowledge Gaps

## Open Questions
- Q1 (40%): Git worktree patterns — good coverage of Claude Code, Cursor, Windsurf, ComposioHQ patterns. Need: deeper conflict resolution strategies beyond single-writer, WorktreeCreate hook patterns, shared resource locking mechanisms, Devin/Copilot Workspace worktree specifics.
- Q2 (60%): PR-based AI coding workflows — SIGNIFICANTLY IMPROVED. Now have: deep empirical success data (Microsoft 878-PR study with monthly progression, task type breakdowns, scope analysis), evidence-based PR template pattern (4 mandatory sections), instruction file format as key lever (copilot-instructions.md structure), GitHub CLI auto-merge mechanism (gh pr merge --auto), Claude Code Action configuration specifics, review bottleneck economics. Remaining: exact PR body templates from each tool (what does a Copilot-generated description look like vs Claude Code), success metrics from tools other than Copilot (Claude Code, Codex success rates unreported), how test coverage metrics are automatically embedded in PR bodies (vs just linked), iteration cycle depth (how many review-feedback loops per PR on average), whether Aether's PR description should follow a standard or custom template.
- Q3 (65%): Aether architecture mapping — strong coverage. Now have: task-as-PR granularity decision, pheromone cross-branch propagation analysis, midden cross-PR tracking design, colony-prime CI caching split, Runtime Verification PR adaptation. Remaining: concrete GitHub Actions YAML implementing the task-as-PR workflow, COLONY_STATE.json phase tracking across PR branches, testing the pheromone main-branch-canonical pattern in practice.
- Q4 (60%): Review automation pipeline — strong coverage. Now have: gate aggregation, auto-fix patterns, Good To Go readiness, gh-pr-review thread management, 5-tier pipeline composition. Remaining: concrete GitHub Actions YAML for Aether, midden→PR review comment feedback loop, Tier 2 agent review timing/cost, whether to build custom readiness aggregator vs use Good To Go.
- Q5 (65%): Branch naming and safeguards — strong coverage. Now have: merge queue bisection, GitHub merge queue mechanics, rollback strategies, lock file merge drivers, short-lived branch consensus, sequential merge pattern, CI auto-fix recovery. Remaining: Aether-specific branch naming convention decision (ant/<phase>/<task> vs worktree-agent-*), colony state sync across merge queue temp branches.

## Contradictions
- Nick Mitchinson notes agent-created worktrees show "slightly less consistent results" vs human-established boundaries, but Claude Code's native worktree support is designed specifically for automated agent creation. Need to investigate whether Claude Code's implementation has solved the consistency issue.
- No AI coding tool currently auto-merges PRs — all require human review as a gate. This may conflict with Aether's autopilot (/ant:run) goal of fully autonomous build-verify-advance cycles. Design decision needed: should Aether's PR workflow allow auto-merge after all gates pass, or always require human approval? The `gh pr merge --auto` CLI mechanism exists for conditional auto-merge.
- Aether's Gate 6 (Runtime Verification) is interactive. **PARTIALLY RESOLVED (Iteration 5):** Mapped all 4 options to PR equivalents.
- Good To Go's AMBIGUOUS classification creates tension with Aether's autopilot mode.
- ComposioHQ reports 84.6% CI success rate with auto-fix, but cross-agent merge conflicts remain unsolved (human responsibility).
- Mergify bisection vs GitHub merge queue — different strategies for batch failure.
- Pheromone signal delay proportional to merge latency (Iteration 5).
- **NEW (Iteration 6):** Microsoft's dotnet/runtime study shows 67.9% success rate with instruction refinement, but this is for a well-instrumented, heavily-documented large repo. Aether targets diverse repos with varying documentation quality — success rates may differ significantly. The "instruction quality > model quality" finding may not generalize if target repos lack the engineering hygiene to write good instruction files.
- **NEW (Iteration 6):** The review bottleneck economics — one person generating PRs faster than a team can review — directly conflicts with Aether's multi-agent parallel PR generation model. If 9 human-initiated Copilot PRs created 5-9 hours of review work, Aether's multi-agent approach could generate 10-20 PRs per phase, creating an even larger review queue. This tension needs a design resolution: either limit PR generation rate, invest heavily in agent-based review to reduce human load, or accept that human review is the throughput bottleneck.

## Discovered Unknowns
- How does `.worktreeinclude` interact with Aether's `.aether/data/` directory (which is local-only)?
- Do the 10 existing agent worktrees in Aether represent successful parallel work or abandoned/stale state?
- How should worktree-per-phase vs worktree-per-agent vs worktree-per-task be decided? **PARTIALLY RESOLVED (Iteration 5):** Task-as-PR is the right granularity.
- What happens when two agents modify the same file in different worktrees?
- How do instruction files (CLAUDE.md, REVIEW.md) propagate into worktree-based agent workflows?
- Should Aether's quality gates run as GitHub Actions checks or as local pre-merge agents?
- How should colony state (COLONY_STATE.json, pheromones.json) be synchronized across parallel worktree branches? **PARTIALLY RESOLVED (Iteration 5).**
- Can colony-prime context be cached in CI? **PARTIALLY RESOLVED (Iteration 5).**
- How should the midden failure tracker work across PR branches? **RESOLVED (Iteration 5).**
- What branch naming convention should Aether adopt?
- How should the 5-tier pipeline handle Tier 2 agent reviews that timeout or exceed cost limits?
- What is the latency cost of running 5 parallel agent reviews in CI?
- Should Good To Go or a custom Aether equivalent be the readiness aggregator?
- Should merge queue use Mergify bisection or GitHub native queue?
- How does npm-merge-driver's archived status affect long-term viability?
- How should COLONY_STATE.json track task completion across PR branches? (Iteration 5)
- Should Aether create a "verification PR" per phase or run gates per-task-PR? (Iteration 5)
- **NEW (Iteration 6):** What is the right PR description template for Aether? Should it follow the evidence-based 4-section pattern (outcome+risk, test evidence, security scans, rollback plan) or a simpler task-focused template?
- **NEW (Iteration 6):** How should Aether balance the review bottleneck — should the 5-tier pipeline's agent reviews (Tier 2) be designed to reduce human review time enough that human review (Tier 4) becomes a quick confirmation rather than a deep dive?
- **NEW (Iteration 6):** Should Aether's instruction file (CLAUDE.md) include a PR-specific section with template format, test evidence requirements, and scope guidance (matching the dotnet/runtime copilot-instructions.md pattern)?

## Last Updated
Iteration 6 -- 2026-03-31T00:30:00Z
