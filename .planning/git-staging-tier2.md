# Tier 2: Gate-Based Commit Suggestions

## Philosophy

Suggest commits at natural, verified boundaries. The user always decides. Opt-in, not opt-out.

Aether never commits without explicit user consent. Tier 2 adds intelligent commit **suggestions** at the moments where the research (1.5) identified the highest-value, lowest-disruption commit points. The colony generates a ready-to-use commit message from phase context, shows the user what would be staged, and lets them accept, customize, or skip with a single choice.

This is the "save your game at the save point" model: the prompt appears where the user already expects to pause, never mid-flow.

---

## Tier 1 Baseline (Included in Tier 2)

Tier 2 includes all of Tier 1 as its foundation. Tier 1 is defined as:

- **No behavioral changes to git operations.** The existing checkpoint in `build.md` and stash in `swarm.md` remain as-is.
- **Document all git touchpoints** in a developer-facing reference (from research 1.1).
- **Classify existing commits** using the Safety / Progress / Milestone taxonomy (from research 1.4).
- **Adopt the commit message convention** from research 1.6: namespaced prefix for operational commits (`aether-checkpoint:`, `aether-milestone:`), standard Conventional Commits for substantive work.
- **Preserve the "never auto-push" invariant** as an absolute rule.

Tier 1 is documentation-only. It changes no runtime behavior.

---

## What Changes in Tier 2

Everything from Tier 1, plus:

### 2.1 Commit Suggestion After POST-ADVANCE (Primary)

**Trigger:** `/ant:continue` passes ALL verification gates (Steps 1.5 through 1.10) and the phase is successfully advanced (Step 2 completes).

**Location in continue.md:** Between the end of Step 2.4 (Update Changelog) and the beginning of Step 3 (Display Result). A new **Step 2.6: Commit Suggestion** is inserted.

**What happens:**
1. Colony reads the completed phase metadata (id, name, description, tasks completed, files changed from `git diff --stat`)
2. Colony calls `generate-commit-message` to produce a suggested message
3. Colony displays the commit suggestion UX (see section below)
4. If user accepts, colony stages and commits. If user skips, flow continues to Step 3 unchanged.

**Why this point:** Research 1.5 scored POST-ADVANCE at 25/25 composite (maximum value, maximum risk, minimal disruption, maximum clarity). The user has already confirmed "Yes, tested and working" at the runtime verification gate. All automated gates passed. The phase is marked complete. The colony explicitly tells the user "safe to /clear" -- this is the natural save point.

### 2.2 Commit Suggestion After `/ant:pause-colony` (Secondary)

**Trigger:** `/ant:pause-colony` completes Step 4 (Write Handoff).

**Location in pause-colony.md:** Between Step 4 (Write Handoff) and Step 5 (Display Confirmation). A new **Step 4.5: Commit Suggestion** is inserted.

**What happens:**
1. Colony checks `git status --porcelain` to see if there are uncommitted changes
2. If changes exist, colony calls `generate-commit-message` with pause context
3. Colony displays the commit suggestion UX
4. If user accepts, colony stages and commits. If user skips, flow continues to Step 5.
5. If no uncommitted changes, skip silently.

**Why this point:** Research 1.5 scored SESSION-PAUSE at 15/25 composite. The user explicitly signaled "I'm leaving." Uncommitted changes may be lost if the user doesn't return to this machine or session. The prompt is natural: "Commit your work before you go?"

### 2.3 `generate-commit-message` Function in `aether-utils.sh`

A new subcommand added to `aether-utils.sh`:

```bash
generate-commit-message)
  # Generate an intelligent commit message from colony context
  # Usage: generate-commit-message <type> <phase_id> <phase_name> [summary]
  # Types: "milestone" | "pause" | "fix"
  # Returns: {"message": "...", "body": "...", "files_changed": N}

  msg_type="${1:-milestone}"
  phase_id="${2:-0}"
  phase_name="${3:-unknown}"
  summary="${4:-}"

  # Count changed files
  files_changed=0
  if git rev-parse --git-dir >/dev/null 2>&1; then
    files_changed=$(git diff --stat --cached HEAD 2>/dev/null | tail -1 | grep -oE '[0-9]+ file' | grep -oE '[0-9]+' || echo "0")
    if [[ "$files_changed" == "0" ]]; then
      files_changed=$(git status --porcelain 2>/dev/null | wc -l | tr -d ' ')
    fi
  fi

  case "$msg_type" in
    milestone)
      # Format: aether-milestone: phase N complete -- <name>
      # If summary provided, use it; otherwise use phase name
      if [[ -n "$summary" ]]; then
        message="aether-milestone: phase ${phase_id} complete -- ${summary}"
      else
        message="aether-milestone: phase ${phase_id} complete -- ${phase_name}"
      fi
      body="All verification gates passed. User confirmed runtime behavior."
      ;;
    pause)
      message="aether-checkpoint: session pause -- phase ${phase_id} in progress"
      body="Colony paused mid-session. Handoff document saved."
      ;;
    fix)
      if [[ -n "$summary" ]]; then
        message="fix: ${summary}"
      else
        message="fix: resolve issue in phase ${phase_id}"
      fi
      body="Swarm-verified fix applied and tested."
      ;;
    *)
      message="aether-checkpoint: phase ${phase_id}"
      body=""
      ;;
  esac

  # Enforce 72-char limit on subject line (truncate if needed)
  if [[ ${#message} -gt 72 ]]; then
    message="${message:0:69}..."
  fi

  json_ok "{\"message\":\"$message\",\"body\":\"$body\",\"files_changed\":$files_changed}"
  ;;
```

**Design decisions:**
- The function is deterministic shell, not LLM-generated. This avoids token cost (research 1.2 flagged Claude Code's token expense for diff-based messages) and ensures consistency.
- The message format follows research 1.6's recommendation: `aether-milestone:` prefix for operational milestones, standard `fix:` for substantive work.
- The 72-character limit enforces the user's stated git rule ("under 72 characters").
- Imperative mood is baked into the format ("complete", "resolve", not "completed", "resolved").

---

## Commit Suggestion UX

### Display Format

When a commit suggestion is triggered, display:

```
──────────────────────────────────────────────────
Commit Suggestion
──────────────────────────────────────────────────

  Message:  aether-milestone: phase 2 complete -- authentication
  Files:    12 files changed
  Staged:   All modified and new files (git add -A)

──────────────────────────────────────────────────
```

### User Prompt

Use `AskUserQuestion` with these options:

```
Commit this milestone?

1. Yes, commit with this message
2. Yes, but let me write the message
3. No, I'll commit later
```

### Behavior for Each Option

**Option 1: "Yes, commit with this message"**

```bash
git add -A && git commit -m "<generated_message>"
```

Display confirmation:
```
Committed: aether-milestone: phase 2 complete -- authentication (12 files)
```

Then continue to the next step in the command flow.

**Option 2: "Yes, but let me write the message"**

Use `AskUserQuestion` again:
```
Enter your commit message (or press Enter to use the suggested one):
```

The user types their message. Then:
```bash
git add -A && git commit -m "<user_message>"
```

Display confirmation and continue.

**Option 3: "No, I'll commit later"**

Display:
```
Skipped. Your changes are saved on disk but not committed.
```

Continue to the next step. No state change. No judgment.

### UX Principles

1. **Never block progress.** The suggestion is a 3-second detour, not a gate. Skipping has zero consequences.
2. **Default to skip-friendly.** If AskUserQuestion somehow fails or times out, the behavior is "skip" (no commit).
3. **No repeated nagging.** If the user skips at POST-ADVANCE, do NOT ask again at pause-colony for the same phase's work. Track a `last_commit_suggestion_phase` in colony state to prevent double-prompting.
4. **Respect the user's git rules.** The user said "do not commit unless explicitly asked." The AskUserQuestion prompt IS explicitly asking. Selecting "Yes" IS explicitly consenting.

---

## Implementation Details

### Changes to `continue.md`

Insert a new **Step 2.6** between Step 2.4 (Update Changelog) and Step 2.5 (Project Completion):

```markdown
### Step 2.6: Commit Suggestion (Optional)

**This step is non-blocking. Skipping does not affect phase advancement.**

After the phase is advanced and changelog updated, suggest a commit:

1. Generate the commit message:
\`\`\`bash
bash ~/.aether/aether-utils.sh generate-commit-message "milestone" {phase_id} "{phase_name}" "{one_line_summary}"
\`\`\`

2. Check how many files changed:
\`\`\`bash
git diff --stat HEAD 2>/dev/null | tail -5
\`\`\`

3. Display the suggestion:
\`\`\`
──────────────────────────────────────────────────
Commit Suggestion
──────────────────────────────────────────────────

  Message:  {generated_message}
  Files:    {files_changed} files changed
  Preview:  {first 5 lines of git diff --stat}

──────────────────────────────────────────────────
\`\`\`

4. Use AskUserQuestion:
\`\`\`
Commit this milestone?

1. Yes, commit with this message
2. Yes, but let me write the message
3. No, I'll commit later
\`\`\`

5. If option 1:
\`\`\`bash
git add -A && git commit -m "{generated_message}"
\`\`\`
   Display: `Committed: {message} ({files_changed} files)`

6. If option 2:
   Use AskUserQuestion to get custom message, then:
\`\`\`bash
git add -A && git commit -m "{custom_message}"
\`\`\`
   Display: `Committed: {custom_message} ({files_changed} files)`

7. If option 3:
   Display: `Skipped. Changes saved on disk but not committed.`

8. Record the suggestion in colony state to prevent double-prompting:
   Set `last_commit_suggestion_phase` to `{phase_id}` in COLONY_STATE.json.

Continue to Step 2.5 (Project Completion) or Step 3 (Display Result).
```

**Note on step numbering:** Step 2.6 is inserted AFTER 2.4 (changelog) because the changelog entry should be included in the commit. It is BEFORE 2.5 (project completion) because 2.5 only runs when all phases complete, and its completion report should also be included in the final commit.

### Changes to `pause-colony.md`

Insert a new **Step 4.5** between Step 4 (Write Handoff) and Step 5 (Display Confirmation):

```markdown
### Step 4.5: Commit Suggestion (Optional)

**This step is non-blocking. Skipping does not affect the pause.**

Check if there are uncommitted changes:
\`\`\`bash
git status --porcelain 2>/dev/null
\`\`\`

If output is empty (nothing to commit), skip this step silently.

If there are uncommitted changes:

1. Read current phase from COLONY_STATE.json (already loaded in Step 1).

2. Check if this phase was already prompted at POST-ADVANCE:
   If `last_commit_suggestion_phase` == current phase, skip this step.

3. Generate the commit message:
\`\`\`bash
bash ~/.aether/aether-utils.sh generate-commit-message "pause" {current_phase} "{phase_name}"
\`\`\`

4. Display and prompt (same UX as continue.md Step 2.6).

5. Execute user's choice (same logic as continue.md Step 2.6).

Continue to Step 5.
```

### New Function in `aether-utils.sh`

Add the `generate-commit-message` case to the main `case "$1" in` block, in a new section:

```bash
# ============================================
# GIT COMMIT UTILITIES
# ============================================

generate-commit-message)
  # (full implementation as shown in section 2.3 above)
  ;;
```

Place this section after the existing `SWARM UTILITIES` block and before the final `*)` catch-all case.

### Commit Message Format (from Research 1.6)

| Context | Format | Example |
|---------|--------|---------|
| Phase milestone | `aether-milestone: phase N complete -- <name>` | `aether-milestone: phase 3 complete -- auth system` |
| Session pause | `aether-checkpoint: session pause -- phase N in progress` | `aether-checkpoint: session pause -- phase 2 in progress` |
| Swarm fix | `fix: <description>` | `fix: resolve JWT token refresh race condition` |
| Project complete | `aether-milestone: project complete -- <goal_summary>` | `aether-milestone: project complete -- task management API` |

All messages:
- Use imperative mood (per user's git.md rule)
- Stay under 72 characters (per user's git.md rule)
- Focus on "why" via the phase name / goal context (per user's git.md rule)
- Use the `aether-` prefix for colony operational commits (per research 1.6 recommendation)
- Use standard Conventional Commits prefixes for substantive work like fixes (per research 1.6 hybrid approach)

---

## Effort Estimate

**Medium-Low**

| Component | Effort | Rationale |
|-----------|--------|-----------|
| `generate-commit-message` in aether-utils.sh | Low | ~50 lines of deterministic bash. No external dependencies. Pattern follows existing subcommands. |
| Step 2.6 in continue.md | Low | ~30 lines of markdown instructions. Inserts cleanly between existing steps. Uses existing `AskUserQuestion` pattern. |
| Step 4.5 in pause-colony.md | Low | ~20 lines of markdown instructions. Simpler variant of the continue.md logic. |
| Mirror to .opencode/ | Low | Copy the same changes to `.opencode/commands/ant/continue.md` and `.opencode/commands/ant/pause-colony.md`. |
| `last_commit_suggestion_phase` state tracking | Low | One field addition to COLONY_STATE.json. |
| Testing | Medium | Need to verify: (a) AskUserQuestion works correctly at these points, (b) git operations succeed/fail gracefully in non-git repos, (c) double-prompt prevention works, (d) 72-char truncation handles edge cases. |

**Total: ~2-3 hours of implementation work.** The majority is testing, not coding.

---

## Risk Assessment

### R1: `git add -A` Stages Sensitive Files

**Risk:** The commit suggestion uses `git add -A`, which stages everything including `.env`, secrets, large binaries, and work-in-progress files the user deliberately chose not to track.

**Mitigation:** Research 1.4 identified this as the most concerning aspect of the current checkpoint system. For Tier 2, the risk is partially mitigated because:
- The user sees the file list before confirming
- The user can choose "No, I'll commit later" and stage selectively themselves
- `.gitignore` still applies (git add -A respects it)

**Residual risk:** Users who quickly select "Yes" may not review the file list carefully. This is acceptable for Tier 2 because the user explicitly consented, but Tier 3 could introduce selective staging.

### R2: AskUserQuestion Disrupts Flow

**Risk:** The prompt adds friction to the continue flow. Users who never want to commit via Aether will find it annoying.

**Mitigation:**
- The prompt is a single question with 3 clear options. It takes ~2 seconds.
- Option 3 ("No, skip") is always available and has zero consequences.
- The double-prompt prevention (`last_commit_suggestion_phase`) ensures users are never asked twice for the same work.
- Future Tier 3 could add a config flag to disable suggestions entirely.

### R3: Generated Message Is Wrong or Truncated

**Risk:** The `generate-commit-message` function uses phase metadata which may be stale, missing, or produce a truncated message at the 72-char limit.

**Mitigation:**
- The function gracefully falls back to phase name if summary is empty.
- Truncation adds "..." to indicate the message was cut.
- Option 2 ("let me write the message") always lets the user override.
- The function is deterministic (no LLM involved), so output is predictable and testable.

### R4: Git Operations Fail

**Risk:** `git add -A && git commit` could fail (not a git repo, merge conflicts, pre-commit hooks reject, disk full).

**Mitigation:**
- The function checks `git rev-parse --git-dir` before any git operations (following existing pattern from build.md line 82).
- If `git commit` fails, display the error and continue the command flow. The commit suggestion is non-blocking; failure to commit does NOT block phase advancement or pause.
- Pre-commit hooks are respected (no `--no-verify` flag). If hooks fail, the user sees the hook output and can fix + commit manually.

### R5: State File Corruption

**Risk:** Adding `last_commit_suggestion_phase` to COLONY_STATE.json could conflict with concurrent writes or corrupt the file.

**Mitigation:** The field is written at the same time as other state updates (phase advancement in Step 2), not as a separate write. This piggybacks on the existing atomic write pattern.

---

## User Impact

### What Changes for the User

1. **After `/ant:continue` succeeds:** The user sees a commit suggestion box with a pre-written message. They choose Yes/Custom/No. If they choose Yes, their work is committed. If No, nothing changes from today's behavior.

2. **After `/ant:pause-colony`:** Same pattern, but only if there are uncommitted changes and the user wasn't already prompted at POST-ADVANCE.

3. **Nothing else changes.** The build checkpoint, swarm stash, verification loop, and all other behaviors remain identical.

### Workflow Comparison

**Today (no Tier 2):**
```
/ant:build 2 -> /ant:continue -> [advancement displayed] -> user manually runs git add + git commit
```

**With Tier 2:**
```
/ant:build 2 -> /ant:continue -> [advancement displayed] -> "Commit?" -> Yes -> [committed] -> [next steps displayed]
                                                          -> No  -> [next steps displayed, same as today]
```

The "No" path is identical to today. Tier 2 adds exactly one optional interaction at a natural pause point.

### Who Benefits Most

- **Users who forget to commit:** The prompt reminds them at the right moment, with a ready-to-use message.
- **Users who want clean git history:** Each phase becomes one commit with a consistent, descriptive message. `git log --oneline` tells the story of the project.
- **Users who are new to Aether:** The prompt teaches them that "this is a good time to commit" without being prescriptive.

### Who Is Least Affected

- **Users who always commit manually:** They select "No" every time. After a few phases, muscle memory makes this a ~1 second interaction.
- **Users with complex git workflows:** They can skip the suggestion and manage branches, staging, and commits themselves.

---

## Trade-offs: Tier 2 vs Tier 1

| Dimension | Tier 1 (Baseline) | Tier 2 (Gate-Based Suggestions) |
|-----------|-------------------|--------------------------------|
| **User friction** | Zero (documentation only) | Minimal (+1 prompt per phase advance, skippable) |
| **Git history quality** | Unchanged (user's existing habits) | Improved (consistent milestone commits with descriptive messages) |
| **Risk of data loss** | Unchanged (user must remember to commit) | Reduced (prompt reminds user at verified boundaries) |
| **Implementation cost** | Near-zero | Medium-low (~2-3 hours) |
| **Maintenance burden** | Near-zero | Low (one utility function, two step insertions) |
| **Respects user git rules** | Fully (no behavior change) | Fully (user explicitly consents via AskUserQuestion) |
| **Commit message quality** | N/A | High (generated from phase context, follows conventions) |
| **Colony-git integration** | None (git and colony are separate concerns) | Light (colony suggests, user decides) |
| **Reversibility** | N/A | Full (remove the steps, revert to Tier 1) |

### What Tier 2 Gains Over Tier 1

1. **Reduced cognitive load.** The user doesn't have to remember to commit, compose a message, or figure out what to stage. The colony handles the "when" and "what" questions; the user just says "yes" or "no."

2. **Consistent project narrative.** With Tier 2, `git log --oneline` for an Aether project reads as a phase-by-phase story. Without it, the log depends entirely on the user's discipline.

3. **Safety at session boundaries.** The pause-colony prompt catches the case where a user says "I'm done for now" but forgets to commit first.

### What Tier 2 Costs Over Tier 1

1. **One additional interaction per phase.** The AskUserQuestion adds ~2-3 seconds to the continue flow. Over a 5-phase project, that is ~15 seconds total.

2. **Slightly more complex command files.** `continue.md` gains ~30 lines. `pause-colony.md` gains ~20 lines. `aether-utils.sh` gains ~50 lines. None of these are architecturally complex.

3. **A new state field.** `last_commit_suggestion_phase` is one integer added to COLONY_STATE.json. Negligible complexity.

---

## Open Questions for Tier 3

The following are deliberately NOT addressed in Tier 2, but flagged for potential Tier 3 design:

1. **Selective staging instead of `git add -A`.** Should the colony only stage files it modified, excluding user's unrelated work-in-progress?
2. **Config flag to disable suggestions.** Should there be an `aether.git.suggest_commits: false` setting?
3. **Commit body with verification evidence.** Should the commit include test counts, gate results, and success criteria in the body?
4. **Branch-based isolation.** Should the colony work on `aether/phase-N` branches instead of the user's current branch?
5. **Downgrading build.md checkpoint from commit to stash.** Research 1.4 recommended this; it is a Tier 1 improvement but was deferred to keep Tier 1 documentation-only.
