# Git Staging Strategy Proposal

## Executive Summary

### Problem

Aether's colony system modifies files across multiple phases, spawns worker ants that touch code in parallel, and runs swarm-based autofix loops -- yet its git integration is minimal and inconsistent. The `build.md` checkpoint uses `git add -A && git commit`, which pollutes the user's commit history and violates their own git rules ("Do not commit unless explicitly asked"). The `swarm.md` checkpoint uses `git stash`, a different mechanism, for no clear reason. There is no commit suggestion at natural milestones, no configurable behavior, and no branch isolation. Users must remember to commit manually after each phase, compose their own messages, and manage branches themselves.

This gap means: messy git history, risk of forgotten commits, and a colony system that is git-adjacent rather than git-integrated.

### The Four Tiers

1. **Tier 1 -- Safety-Only:** Switch the build checkpoint from `git commit` to `git stash`, standardize checkpoint naming, add rollback verification. Zero user-facing behavior change. (~30 lines changed across 7 files.)

2. **Tier 2 -- Gate-Based Commit Suggestions:** Add optional commit prompts at POST-ADVANCE and SESSION-PAUSE. The colony generates a message, the user says Yes/Custom/No. No automatic commits. (~100 lines added across 4 files + 1 new function.)

3. **Tier 3 -- Configurable Automation:** Add an `auto_commit` config with modes (`never`/`verified`/`aggressive`). Users set their preference once; the colony respects it. Optional pre/post commit hooks. (~150 lines across 5-7 files + config schema.)

4. **Tier 4 -- Branch-Aware Colony:** Environment detection at init, feature branch creation when on `main`, branch consistency checks, and optional GitHub PR creation at project completion. (~200 lines across 4 files + 3 new functions.)

### Recommended Approach

Implement Tiers 1 and 2 together as an immediate improvement. They are low-risk, low-effort, and address the most pressing problems (history pollution, forgotten commits). Defer Tiers 3 and 4 until real-world usage of Tier 2 reveals whether users want automation and branch management, or whether the suggestion model is sufficient.

### Decision Needed

Choose an implementation scope from the options in the Decision Prompt section below.

---

## Comparison Matrix

| Dimension | Tier 1: Safety-Only | Tier 2: Gate-Based Suggestions | Tier 3: Configurable Automation | Tier 4: Branch-Aware Colony |
|---|---|---|---|---|
| **Effort** | Low (~1-2 hrs) | Medium-Low (~2-3 hrs) | Medium (~4-6 hrs) | Medium (~6-8 hrs) |
| **Risk** | Low | Low | Medium | Medium |
| **User Impact** | Minimal (invisible improvement) | Low (+1 skippable prompt per phase) | Medium (new config to learn) | Significant (branch management concepts) |
| **Git History Impact** | Positive (removes noise commits) | Positive (consistent milestone commits) | Positive (configurable commit granularity) | Positive (work isolated to feature branches) |
| **Respects User Rules** | Fully (eliminates rule violation) | Fully (user explicitly consents) | Fully (user configures preference) | Fully (all branch ops are prompted) |
| **Requires Config** | No | No | Yes (constraints.json `git` section) | Yes (constraints.json + init prompts) |
| **Files Changed** | 7 | 4 (+mirrors) | 5-7 (+mirrors) | 4 (+mirrors) |
| **New Functions** | 0 | 1 (`generate-commit-message`) | 1-2 (`git-config`, hook runner) | 3 (`detect-git-context`, `create-colony-branch`, `create-pr`) |
| **Industry Precedent** | Standard (all tools do safety checkpoints) | Moderate (VS Code save reminders) | Strong (Aider's `--auto-commits`, but configurable) | Strong (GitHub CLI workflows, feature branch model) |
| **Standalone Value** | Yes | Yes (with Tier 1 included) | Yes (with Tiers 1+2 included) | Yes (with Tiers 1+2+3 included) |

---

## Tier Summaries

### Tier 1: Safety-Only

- **Philosophy:** Change zero user-facing behavior. Make existing safety mechanisms less intrusive and more reliable.
- **Key Changes:** Replace `git add -A && git commit` checkpoint with `git stash push --include-untracked`; standardize checkpoint naming under `aether-checkpoint:` prefix; add rollback verification after each checkpoint; document the rollback procedure explicitly.
- **Effort:** Low. All changes are within existing code paths -- no new functions, no new files. ~30 lines changed across 7 files.
- **Risk:** Low. The stash mechanism is already proven in `aether-utils.sh`. Edge cases (clean working tree, stash failure) are handled with graceful fallbacks.

Full details: [.planning/git-staging-tier1.md](./git-staging-tier1.md)

### Tier 2: Gate-Based Commit Suggestions

- **Philosophy:** Suggest commits at natural, verified boundaries. The user always decides. Opt-in, not opt-out.
- **Key Changes:** Insert a commit suggestion prompt after POST-ADVANCE (phase completion) and at SESSION-PAUSE; add a `generate-commit-message` function that produces consistent, convention-following messages from phase context; double-prompt prevention via state tracking.
- **Effort:** Medium-Low. One new bash function (~50 lines), two command file insertions (~50 lines combined), state field addition. ~2-3 hours including testing.
- **Risk:** Low. The prompt is non-blocking (skip has zero consequences). Git operations are gated behind user consent. Message generation is deterministic, not LLM-generated.

Full details: [.planning/git-staging-tier2.md](./git-staging-tier2.md)

### Tier 3: Configurable Automation

- **Philosophy:** Users set their commit preference once, the colony respects it. "Set it and forget it."
- **Key Changes:** Add `git.auto_commit` config field to `constraints.json` with modes `never`/`verified`/`aggressive`; add optional pre/post commit hooks with environment variables; modify 5-7 command files to check config at each commit decision point.
- **Effort:** Medium. The config system is simple but touches many files. Testing surface is 3 modes x 4-5 commit points. Hooks can be deferred as a stretch goal.
- **Risk:** Medium. Auto-commit accidents are possible if users opt into `aggressive` mode carelessly. Mitigated by `never` as the default and by never auto-pushing.

Full details: [.planning/git-staging-tier3.md](./git-staging-tier3.md)

### Tier 4: Branch-Aware Colony

- **Philosophy:** The colony is git-aware at the branch level. Feature branches for safety, environment detection for adaptation, optional PR creation for completion.
- **Key Changes:** 6-step environment detection chain at init; feature branch creation prompt when user is on `main`; branch consistency checks at each build; PR auto-generation at project completion (GitHub only, with graceful fallback for all other forges).
- **Effort:** Medium. Three new utility functions, modifications to 4 command files. The complexity is combinatorial (git/no-git, GitHub/not-GitHub, gh/no-gh) rather than algorithmic.
- **Risk:** Medium. Branch management changes the user's mental model. Mitigated by making every branch operation a prompted choice, never automatic.

Full details: [.planning/git-staging-tier4.md](./git-staging-tier4.md)

---

## Dependency Chain

```
Tier 1 (Safety-Only)
  |
  v
Tier 2 (Gate-Based Suggestions)  -- includes Tier 1
  |
  v
Tier 3 (Configurable Automation) -- includes Tiers 1 + 2
  |
  v
Tier 4 (Branch-Aware Colony)     -- includes Tiers 1 + 2 + 3
```

### Can Tiers Be Implemented Independently?

| Combination | Feasible? | Notes |
|---|---|---|
| Tier 1 alone | Yes | Standalone improvement. Delivers value immediately. |
| Tier 2 alone (without Tier 1) | Possible but not recommended | Tier 2's commit suggestions would coexist with the noisy checkpoint commits Tier 1 eliminates. The `git add -A` concern from Tier 1 would remain. |
| Tier 2 with Tier 1 | Yes (recommended) | Natural pairing. Tier 1 cleans up the foundation; Tier 2 adds user-facing value. |
| Tier 3 without Tier 2 | Not recommended | Tier 3's config modes control Tier 2's suggestion behavior. Without Tier 2, there is nothing to configure. |
| Tier 4 without Tier 3 | Possible | Branch management is conceptually independent. However, combining branch isolation with commit automation is the full-value proposition. |
| Tiers 1+2+3 without Tier 4 | Yes | This is the "commit-aware" colony without branch management. Fully coherent. |

### Minimum Viable Implementation Path

**Tier 1 + Tier 2** -- delivers the highest value-to-effort ratio. Eliminates history pollution (Tier 1) and adds guided commit suggestions at verified milestones (Tier 2). Total effort: ~4-5 hours. Total risk: Low.

---

## Recommendation

**Implement Tiers 1 and 2 together as a single phase of work.**

Reasoning:

1. **Tier 1 is a pure improvement with no tradeoffs.** It fixes an existing rule violation (`git commit` without consent), reduces noise in git history, and standardizes checkpoint naming. There is no argument against it.

2. **Tier 2 is the highest-value addition.** It addresses the most common gap (forgotten commits at milestones) with the lowest friction (one skippable prompt). The user explicitly consents to every commit, which aligns perfectly with the existing git rules.

3. **Tiers 3 and 4 should wait for usage data.** The config system (Tier 3) and branch management (Tier 4) are valuable features, but they add complexity that may not be needed yet. After running with Tier 2 for several projects, the user will have firsthand experience with the suggestion model and can make an informed decision about whether automation and branch isolation are worth the added complexity.

4. **The research-first approach paid off.** All four tiers are fully designed and documented. If the user later decides to implement Tier 3 or 4, the design work is done -- only implementation remains.

5. **Conservative default matches user profile.** The user's git rules emphasize manual control ("Do not commit unless explicitly asked"). Tier 2's suggestion model respects this exactly: the colony asks, the user decides. Jumping straight to Tier 3's auto-commit would be a philosophical shift that should be deliberate, not defaulted into.

---

## Decision Prompt

Choose one of the following implementation paths:

### Option A: Conservative -- Tier 1 Only
- Scope: Fix the checkpoint mechanism, standardize naming, add rollback docs
- Effort: ~1-2 hours
- Result: Cleaner git behavior, zero user-facing change
- Best if: You want the smallest possible change and are comfortable managing commits manually

### Option B: Recommended -- Tiers 1 + 2
- Scope: Everything in Option A, plus commit suggestions at POST-ADVANCE and SESSION-PAUSE
- Effort: ~4-5 hours
- Result: Clean checkpoints + guided commit workflow at natural milestones
- Best if: You want the colony to remind you to commit at the right moments with ready-to-use messages

### Option C: Ambitious -- Tiers 1 + 2 + 3
- Scope: Everything in Option B, plus configurable auto-commit modes and optional hooks
- Effort: ~8-11 hours
- Result: Full commit automation spectrum from manual to aggressive, per-project config
- Best if: You want different commit behaviors for different projects and are comfortable with a config system

### Option D: Full Vision -- All 4 Tiers
- Scope: Everything in Option C, plus branch management, environment detection, and GitHub PR creation
- Effort: ~14-19 hours
- Result: Complete git-native colony with branch isolation and forge integration
- Best if: You want the colony to manage the entire git lifecycle from init to PR

### Option E: None -- Keep Current Behavior
- Scope: No changes
- Effort: 0
- Result: Current behavior continues (checkpoint commits in history, manual commit workflow)
- Best if: The current system works well enough and you prefer to invest time elsewhere

---

*This proposal was synthesized from research conducted across 6 sub-topics (git touchpoint audit, industry survey, worktree analysis, commit classification, lifecycle mapping, and message conventions) and 4 detailed tier designs. The full research documents are available in `.planning/git-staging-*.md`.*
