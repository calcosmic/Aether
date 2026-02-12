> **STATUS: NOT IMPLEMENTED** — Research artifact from Phase 2 (git staging strategy). Tier 2 was chosen for implementation instead. This document describes a design that was evaluated but never built. See `git-staging-proposal.md` for the decision rationale.

# Tier 3: Hooks-Based Automation

## Philosophy

Configurable commit automation. Users set their preference once, and the colony respects it throughout the session and across sessions. This is the "set it and forget it" tier — like Aider's auto-commit approach but with granular control over *when* commits happen.

The core insight from Phase 1 research: the industry has converged on a spectrum from fully automatic (Aider) to fully manual (Cursor/Copilot), but **no tool lets users choose where on the spectrum they want to sit**. Aider's users complain about surprise commits; Cursor's users complain about forgotten commits. Tier 3 solves this by making the commit behavior a first-class user preference rather than a hardcoded system decision.

Tier 3 includes everything from Tiers 1 and 2. The config system *controls* when Tier 2's commit suggestions become automatic.

---

## What Changes

### Everything from Tiers 1 + 2

- Tier 1: Safety-only checkpoints (stash-based, standardized messages, rollback verification)
- Tier 2: Gate-based commit suggestions at POST-ADVANCE (25/25 composite score), PROJECT-COMPLETE, and SESSION-PAUSE

### New: Git Configuration System

Add a `git` section to `.aether/data/constraints.json` (reusing the existing config file rather than creating a new one):

```json
{
  "version": "1.0",
  "focus": [],
  "constraints": [],
  "git": {
    "auto_commit": "never",
    "commit_format": "namespace",
    "hooks": {
      "on_before_commit": null,
      "on_after_commit": null
    }
  }
}
```

### New: Four Auto-Commit Modes

| Mode | Behavior | Equivalent To | Who It's For |
|------|----------|---------------|-------------|
| `"never"` | No automatic commits. Colony suggests but never acts. User runs `git commit` manually. | Tier 1 + Tier 2 suggestions displayed but not executed | Users with strict `~/.claude/rules/git.md` ("do not commit unless explicitly asked"). Default mode. |
| `"safety"` | Only creates safety checkpoints (stash-based). No code commits. Identical to Tier 1. | Tier 1 | Users who want rollback protection but full control over their git history |
| `"verified"` | Auto-commits after verified gates pass (POST-ADVANCE, PROJECT-COMPLETE). User is still prompted at SESSION-PAUSE. | Tier 2 points, but automatic instead of prompted | Users who trust the verification loop and want milestones in git without manual action |
| `"aggressive"` | Commits after every phase build (POST-BUILD) + continue (POST-ADVANCE) + swarm fix (POST-SWARM-FIX). Like Aider but with colony semantics. | Every meaningful state change | Users who want Aider-style full history, or teams auditing AI contributions |

### New: Pre/Post Commit Hooks

Configurable shell commands that run before and after any colony-initiated commit:

```json
{
  "git": {
    "hooks": {
      "on_before_commit": "npm run lint-staged",
      "on_after_commit": "echo 'Committed phase $AETHER_PHASE' >> /tmp/aether-audit.log"
    }
  }
}
```

---

## Configuration UX

### How the User Sets Their Preference

**Primary: Config file edit.** The user edits `.aether/data/constraints.json` directly. This is already a file the colony reads on every build (Step 4 of `build.md`). No new file to discover.

```bash
# Example: set to verified mode
jq '.git.auto_commit = "verified"' .aether/data/constraints.json > tmp && mv tmp .aether/data/constraints.json
```

**Secondary: aether-utils.sh subcommand.** Add a `git-config` subcommand for convenience:

```bash
bash .aether/aether-utils.sh git-config auto_commit verified
# Output: {"ok":true,"result":{"auto_commit":"verified"}}
```

**Not recommended: First-run prompt.** A prompt during `/ant:init` was considered but rejected. Reasons:
1. Init already has cognitive overhead (goal definition, constraint setting)
2. New users don't know enough about the system to make an informed choice
3. The default (`"never"`) is the safest and aligns with the user's existing git rules
4. Users who want auto-commits are power users who can find the config

### How to Override Per-Session

Two mechanisms, depending on the host environment:

**In Claude Code:** Use `CLAUDE.md` or session-level instructions:
```
For this session: auto-commit after verified phases.
```
The colony command files (`build.md`, `continue.md`) already read `CLAUDE.md` instructions. The git config in `constraints.json` is the persistent default; session instructions override it.

**In any environment:** Edit `constraints.json` before the build:
```bash
jq '.git.auto_commit = "aggressive"' .aether/data/constraints.json > tmp && mv tmp .aether/data/constraints.json
```
Then revert after the session. This is clunky but explicit.

### Default Setting

**`"never"`** — no automatic commits.

Rationale:
1. **Aligns with the user's existing `~/.claude/rules/git.md`:** "Do not commit unless explicitly asked." The default should not violate the user's own rules out of the box.
2. **Matches industry consensus:** Claude Code, Cursor, Windsurf, and Copilot all default to no auto-commit. Only Aider auto-commits by default, and its users report this as the #1 source of complaints.
3. **Progressive disclosure:** Users discover auto-commit when they're ready. Tier 2 suggestions educate them about when commits are valuable; Tier 3 config lets them automate what they've learned to appreciate.
4. **Harm avoidance:** An auto-commit accident (committing secrets, committing broken code to main) is worse than a missed commit. The default should minimize the worse outcome.

---

## Implementation Details

### Config Schema

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "properties": {
    "git": {
      "type": "object",
      "properties": {
        "auto_commit": {
          "type": "string",
          "enum": ["never", "safety", "verified", "aggressive"],
          "default": "never",
          "description": "Controls when the colony automatically creates git commits"
        },
        "commit_format": {
          "type": "string",
          "enum": ["namespace", "conventional", "hybrid"],
          "default": "namespace",
          "description": "Commit message format for colony-generated commits"
        },
        "hooks": {
          "type": "object",
          "properties": {
            "on_before_commit": {
              "type": ["string", "null"],
              "default": null,
              "description": "Shell command to run before each colony commit. Non-zero exit aborts the commit."
            },
            "on_after_commit": {
              "type": ["string", "null"],
              "default": null,
              "description": "Shell command to run after each colony commit. Failure is logged but does not revert."
            }
          }
        }
      }
    }
  }
}
```

### Where the Config Is Read

The config must be checked at each commit decision point. These are the command files that need modification:

| File | When Config Is Checked | What Happens |
|------|----------------------|--------------|
| `build.md` Step 3 | PRE-BUILD checkpoint | `safety` or higher: create stash checkpoint. `never`: skip checkpoint entirely. |
| `build.md` Step 7 | POST-BUILD (end of build) | `aggressive`: auto-commit with `aether-progress:` message. Otherwise: no action. |
| `continue.md` Step 2 | POST-ADVANCE (phase complete) | `verified` or `aggressive`: auto-commit with `aether-milestone:` message. `never` or `safety`: display Tier 2 suggestion only. |
| `continue.md` final step | PROJECT-COMPLETE | `verified` or `aggressive`: auto-commit. Others: suggest. |
| `pause-colony.md` | SESSION-PAUSE | All modes except `never`: prompt for commit (this is always interactive because the user explicitly invoked pause). |
| `swarm.md` Step 3 | PRE-SWARM checkpoint | `safety` or higher: create stash. `never`: skip. |
| `swarm.md` final step | POST-SWARM-FIX | `aggressive`: auto-commit with `aether-fix:` message. Others: no action. |

**Reading the config:** Each command file already reads `constraints.json` (or should). The git config is extracted alongside focus/constraints:

```bash
# In build.md Step 4 (Load Constraints), add:
# Extract git config
auto_commit=$(jq -r '.git.auto_commit // "never"' .aether/data/constraints.json 2>/dev/null || echo "never")
```

Since command files are markdown instructions interpreted by the LLM (not executed as scripts), the actual implementation is:

> Read `.aether/data/constraints.json`. Extract `git.auto_commit` (default: `"never"`) and `git.hooks` (default: both null). Use these values to determine commit behavior at each decision point in this command.

### Hook Execution Model

Hooks are **inline shell commands** executed via `bash -c`. They are simple, transparent, and require no additional infrastructure.

**Pre-commit hook:**
```bash
# If on_before_commit is set, run it before committing
pre_hook=$(jq -r '.git.hooks.on_before_commit // empty' .aether/data/constraints.json 2>/dev/null)
if [ -n "$pre_hook" ]; then
    # Export context variables for the hook
    export AETHER_PHASE="$PHASE_NUMBER"
    export AETHER_COMMIT_TYPE="milestone"  # or "progress", "checkpoint", "fix"
    export AETHER_COMMIT_MSG="$commit_message"

    if ! bash -c "$pre_hook"; then
        echo "Pre-commit hook failed. Commit aborted."
        # Fall back to Tier 2 behavior: show suggestion, let user decide
        exit 0
    fi
fi
```

**Post-commit hook:**
```bash
# If on_after_commit is set, run it after committing
post_hook=$(jq -r '.git.hooks.on_after_commit // empty' .aether/data/constraints.json 2>/dev/null)
if [ -n "$post_hook" ]; then
    export AETHER_PHASE="$PHASE_NUMBER"
    export AETHER_COMMIT_TYPE="milestone"
    export AETHER_COMMIT_HASH="$commit_hash"

    bash -c "$post_hook" || echo "Post-commit hook failed (non-fatal)."
fi
```

**Environment variables available to hooks:**

| Variable | Description | Example |
|----------|-------------|---------|
| `AETHER_PHASE` | Current phase number | `3` |
| `AETHER_COMMIT_TYPE` | One of: `checkpoint`, `progress`, `milestone`, `fix` | `milestone` |
| `AETHER_COMMIT_MSG` | The commit message that will be / was used | `aether-milestone: phase 3 complete` |
| `AETHER_COMMIT_HASH` | The commit hash (post-commit only) | `abc1234` |
| `AETHER_GOAL` | The colony goal | `Build authentication system` |

**Why not use Claude Code's hooks system?** Claude Code's hooks (`PreToolUse`, `PostToolUse`, etc.) are defined in `settings.json` and run on tool invocations, not on colony lifecycle events. They cannot distinguish between a `git commit` from the colony vs. a `git commit` from the user. Aether's hooks are colony-aware (they know the phase, commit type, and colony state). Claude Code hooks remain useful as a secondary enforcement layer but are orthogonal to Aether's hook system.

### Interaction with `~/.claude/rules/git.md`

The user's rule says: "Do not commit unless explicitly asked."

This rule applies to the **LLM's behavior**, not to scripts. The colony's git config in `constraints.json` constitutes "explicit asking" — the user configured `auto_commit: "verified"`, which is an explicit, persistent instruction to commit at verified boundaries.

**Resolution matrix:**

| git.md Rule | constraints.json Setting | Behavior |
|------------|-------------------------|----------|
| "Do not commit" | `auto_commit: "never"` | No conflict. No auto-commits. |
| "Do not commit" | `auto_commit: "safety"` | No conflict. Stash-only, no commits. |
| "Do not commit" | `auto_commit: "verified"` | Config overrides for colony milestones. The config IS the explicit ask. |
| "Do not commit" | `auto_commit: "aggressive"` | Config overrides for all colony commits. User has made a deliberate choice. |

**Important nuance:** The `auto_commit` config only authorizes *colony-initiated* commits. It does not override the git.md rule for arbitrary `git commit` commands the LLM might attempt outside colony operations. The rule "do not commit unless explicitly asked" remains in force for non-colony git activity.

---

## What the Commit Flow Looks Like

### Mode: `"verified"` (Recommended Power-User Setting)

```
User runs /ant:build 3
  -> Colony reads constraints.json: auto_commit = "verified"
  -> PRE-BUILD: git stash --include-untracked (safety checkpoint, always)
  -> Workers build Phase 3...
  -> POST-BUILD: No commit (verified mode only commits after verification)
  -> Build complete, output displayed

User runs /ant:continue
  -> Verification loop runs (build, types, lint, tests, security, diff)
  -> All gates pass
  -> Runtime verification: user confirms "Yes, tested and working"
  -> POST-ADVANCE:
    -> Run on_before_commit hook (if set)
    -> If hook passes: git add <modified files> && git commit -m "aether-milestone: phase 3 complete -- signal unification"
    -> Run on_after_commit hook (if set)
    -> Display: "Committed: aether-milestone: phase 3 complete (abc1234)"
  -> Phase advanced, ready for next build
```

### Mode: `"aggressive"` (Aider-Like)

```
User runs /ant:build 3
  -> Colony reads constraints.json: auto_commit = "aggressive"
  -> PRE-BUILD: git stash --include-untracked
  -> Workers build Phase 3...
  -> POST-BUILD: git add <modified files> && git commit -m "aether-progress: phase 3 -- implement signal schema"
  -> Build complete

User runs /ant:continue
  -> Verification loop runs
  -> POST-ADVANCE: git add <modified files> && git commit -m "aether-milestone: phase 3 complete -- signal unification"
  -> Phase advanced
```

---

## Effort Estimate: Medium

**Rationale:**

| Component | Effort | Notes |
|-----------|--------|-------|
| Config schema in constraints.json | Low | Add 10 lines of JSON; no migration needed (missing `git` key defaults to `"never"`) |
| `git-config` subcommand in aether-utils.sh | Low | ~30 lines of bash, straightforward jq manipulation |
| Modify `build.md` to read config + act | Medium | Step 3 (checkpoint) and Step 7 (post-build) need conditional logic |
| Modify `continue.md` to read config + act | Medium | Step 2 (post-advance) needs conditional commit logic |
| Modify `pause-colony.md` | Low | Add commit prompt with config check |
| Modify `swarm.md` | Low | Post-fix commit if aggressive |
| Hook execution infrastructure | Low | ~20 lines of bash; env var export + bash -c |
| Mirror changes to `.opencode/` commands | Medium | All changes must be replicated to 4-5 OpenCode command files |
| Testing all 4 modes end-to-end | Medium | Each mode needs manual verification across build/continue/swarm flows |
| Documentation | Low | Update README or add to help command |

**Total: Medium.** The config system is simple, the hook execution is simple, but the changes touch 5-7 command files (plus OpenCode mirrors) and each file needs conditional logic based on the config value. The testing surface is 4 modes x 3-4 commit points = 12-16 behavioral paths.

---

## Risk Assessment

### R1: Auto-Commit Accidents (Severity: High, Likelihood: Low)

**What could go wrong:** User sets `auto_commit: "aggressive"`, and the colony commits broken code, secrets in `.env`, or large binary files to the working branch.

**Mitigations:**
- Default is `"never"` — accidents require deliberate opt-in.
- Even in `"aggressive"` mode, Tier 1's `git add <specific files>` (not `git add -A`) prevents staging unrelated files. Only files modified by the colony's workers are staged.
- The pre-commit hook (`on_before_commit`) can run lint-staged, secret detection, or any gatekeeper the user wants.
- Colony never auto-pushes. All commits are local-only. The blast radius is limited to the local repo.
- `"verified"` mode only commits after the full 6-phase verification loop passes, including security scan (Phase 5 checks for exposed secrets).

### R2: Config Drift Across Sessions (Severity: Low, Likelihood: Medium)

**What could go wrong:** User sets aggressive mode for a test project, forgets about it, and it applies to a production project.

**Mitigations:**
- Config is per-project (in `.aether/data/constraints.json`, which lives in the project directory), not global. Different projects have independent settings.
- `/ant:status` should display the current git config as part of the colony status output. This makes the setting visible.

### R3: Hook Execution Failures (Severity: Medium, Likelihood: Low)

**What could go wrong:** A pre-commit hook fails (e.g., linter finds issues), and the user doesn't understand why the commit didn't happen.

**Mitigations:**
- Clear error output: "Pre-commit hook failed. Commit aborted. Hook output: [...]"
- Fallback to Tier 2 behavior: show the commit suggestion so the user can commit manually after fixing the hook issue.
- Post-commit hook failures are logged but non-fatal (the commit already happened, rolling back would be worse).

### R4: Interaction with IDE Git Extensions (Severity: Low, Likelihood: Low)

**What could go wrong:** Colony auto-commits while the user has VS Code's Source Control panel open, causing confusion about staged/unstaged state.

**Mitigations:**
- This is inherent to any tool that modifies git state. Aider, Claude Code checkpoints, and Cursor all face this.
- Colony commits are clearly labeled with `aether-` prefixes, making them identifiable in git log.

### R5: The `git add -A` Trap (Severity: High, Likelihood: Medium)

**What could go wrong:** If the implementation uses `git add -A` (as the current `build.md` does), auto-commits could stage files the user never intended to track.

**Mitigation: This is a Tier 1 fix, not a Tier 3 concern.** Tier 1 should already replace `git add -A` with targeted staging. Tier 3 inherits that fix. If Tier 1 is not implemented, Tier 3 must NOT be implemented — the auto-commit modes would amplify the `git add -A` problem.

---

## User Impact

### Workflow Change by Mode

| Mode | User Experience Change | Mental Model |
|------|----------------------|--------------|
| `"never"` | No change from Tier 2. Suggestions appear, user acts on them or ignores them. | "Colony advises, I decide." |
| `"safety"` | Stash checkpoints happen silently. User barely notices. | "Colony protects my work." |
| `"verified"` | After `/ant:continue` passes all gates, a commit appears in git log automatically. User sees "Committed: ..." in output. | "Colony saves my milestones." |
| `"aggressive"` | After every build and continue, commits appear. Git log fills with `aether-progress:` and `aether-milestone:` entries. | "Colony records everything." |

### Who Benefits Most

- **Solo developers on personal projects:** `"verified"` mode eliminates the "I forgot to commit after the build" problem without cluttering history.
- **Teams auditing AI contributions:** `"aggressive"` mode creates a complete audit trail of every AI-generated change, with phase numbers and commit types for traceability.
- **Cautious enterprise developers:** `"never"` mode with pre-commit hooks gives them full control plus enforcement (e.g., require all commits to pass lint-staged).

### Who Is Hurt

- **Users who don't read docs:** If someone sets `"aggressive"` without understanding it, they may be surprised by commits they didn't expect. Mitigated by the `"never"` default.
- **Users with complex git workflows (rebasing, squashing):** Auto-commits create more commits to manage during rebase/squash. Mitigated by using `"verified"` (fewer commits) or `"never"`.

---

## Trade-offs: Power vs. Complexity

### Is This Over-Engineering?

**Arguments that it IS over-engineering:**

1. **Four modes is a lot.** Most users will use `"never"` (the default) or `"verified"`. The `"safety"` and `"aggressive"` modes serve niche use cases. Three modes (`never`/`verified`/`aggressive`) might suffice.

2. **Hooks add complexity for rare use cases.** How many Aether users will configure `on_before_commit` hooks? Probably fewer than 5%. The hooks are power-user features that most people won't touch.

3. **Config in constraints.json feels overloaded.** The file currently holds `focus` and `constraints` (arrays of strings). Adding a structured `git` object changes the file's character from "simple project constraints" to "multi-purpose config."

4. **Testing surface is large.** 4 modes x 7 commit points x 2 environments (Claude Code + OpenCode) = 56 behavioral paths. Most won't be tested.

**Arguments that it is NOT over-engineering:**

1. **The config schema is 15 lines of JSON.** The implementation cost is low. The `auto_commit` field is one string value checked at 4-5 decision points. This is not a plugin system or an event bus — it's a simple conditional.

2. **The industry data shows this is exactly what users want.** Aider's #1 complaint is "too aggressive." Claude Code's #1 gap is "no auto-commit option." Tier 3 gives users the choice that no existing tool provides.

3. **The hooks are opt-in and optional.** If no one uses them, they cost nothing (a null check in the commit path). If someone does use them, they enable powerful workflows (lint-staged, audit logging, Slack notifications) without Aether needing to implement each one.

4. **The modes map cleanly to the commit classification from Phase 1.** `"safety"` = Class 1 only. `"verified"` = Class 1 + Class 3. `"aggressive"` = Class 1 + Class 2 + Class 3. The abstraction is not arbitrary — it mirrors the research.

### Recommendation

**Implement Tier 3 with three modes initially: `"never"`, `"verified"`, `"aggressive"`.** Drop `"safety"` because it's identical to Tier 1 behavior (no additional code needed — if the user wants Tier 1 behavior, they leave the config at `"never"` and Tier 1's stash checkpoints still operate).

**Implement hooks as a stretch goal.** The config schema should include the `hooks` field from day one (future-proofing), but the hook execution logic can be deferred to a later phase. The config field having a `null` default means it's invisible until needed.

This reduces the initial implementation to:
- 1 config field (`auto_commit`) with 3 values
- 4-5 command file modifications (add conditional commit at each decision point)
- 0 new bash infrastructure (the commit logic is inline in the command file instructions)

**Revised effort: Low-Medium.**

---

## Comparison with Industry Approaches

| Feature | Aider | Claude Code | Aether Tier 3 |
|---------|-------|-------------|---------------|
| Auto-commit default | ON | OFF | OFF (`"never"`) |
| User can disable | Yes (`--no-auto-commits`) | N/A (no auto-commit exists) | Yes (default is off) |
| User can enable | N/A (always on) | No built-in option | Yes (`"verified"` or `"aggressive"`) |
| Granularity of control | Binary (on/off) | N/A | 3 modes (never/verified/aggressive) |
| Commit message format | Conventional Commits | User-defined | Configurable (namespace/conventional/hybrid) |
| Pre-commit hooks | `--git-commit-verify` (enable/disable) | `PreToolUse` on `Bash(git commit:*)` | `on_before_commit` (arbitrary shell command) |
| Post-commit hooks | None | `PostToolUse` on `Bash(git commit:*)` | `on_after_commit` (arbitrary shell command) |
| Auto-push | No | No | No (absolute invariant) |
| Attribution | `(aider)` in author metadata | `Co-Authored-By` trailer | `aether-` prefix in commit message |

Tier 3 provides the most granular user control of any AI coding tool's git integration, while defaulting to the most conservative behavior.
