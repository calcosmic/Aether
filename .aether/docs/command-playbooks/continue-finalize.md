### Step 2.2: Update Handoff Document

After advancing the phase, update the handoff document with the new current state:

```bash
# Determine if there's a next phase
next_phase_id=$((current_phase + 1))
has_next_phase=$(jq --arg next "$next_phase_id" '.plan.phases | map(select(.id == ($next | tonumber))) | length' .aether/data/COLONY_STATE.json)

# Write updated handoff
cat > .aether/HANDOFF.md << 'HANDOFF_EOF'
# Colony Session — Phase Advanced

## Quick Resume
Run `/ant:build {next_phase_id}` to start working on the current phase.

## State at Advancement
- Goal: "$(jq -r '.goal' .aether/data/COLONY_STATE.json)"
- Completed Phase: {completed_phase_id} — {completed_phase_name}
- Current Phase: {next_phase_id} — {next_phase_name}
- State: READY
- Updated: $(date -u +%Y-%m-%dT%H:%M:%SZ)

## What Was Completed
- Phase {completed_phase_id} marked as completed
- Learnings extracted: {learning_count}
- Instincts updated: {instinct_count}
- Wisdom promoted to QUEEN.md: {promoted_count}

## Current Phase Tasks
$(jq -r '.plan.phases[] | select(.id == next_phase_id) | .tasks[] | "- [ ] \(.id): \(.description)"' .aether/data/COLONY_STATE.json)

## Next Steps
- Build current phase: `/ant:build {next_phase_id}`
- Review phase details: `/ant:phase {next_phase_id}`
- Pause colony: `/ant:pause-colony`

## Session Note
Phase advanced successfully. Colony is READY to build Phase {next_phase_id}.
HANDOFF_EOF
```

This handoff reflects the post-advancement state, allowing seamless resumption even if the session is lost.

### Step 2.3: Update Changelog

**MANDATORY: Append a changelog entry for the completed phase. This step is never skipped.**

If no `CHANGELOG.md` exists, `changelog-append` creates one automatically.

**Step 2.3.1: Collect plan data**

```bash
bash .aether/aether-utils.sh changelog-collect-plan-data "{phase_identifier}" "{plan_number}"
```

Parse the returned JSON to extract `files`, `decisions`, `worked`, and `requirements` arrays.

- `{phase_identifier}` is the full phase name (e.g., `36-memory-capture`)
- `{plan_number}` is the plan number (e.g., `01`)

If the command fails (e.g., no plan file found), fall back to collecting data manually:
- Files: from `git diff --stat` of the completed phase
- Decisions: from COLONY_STATE.json `memory.decisions` (last 5)
- Worked/requirements: leave empty

**Step 2.3.2: Append changelog entry**

```bash
bash .aether/aether-utils.sh changelog-append \
  "$(date +%Y-%m-%d)" \
  "{phase_identifier}" \
  "{plan_number}" \
  "{files_csv}" \
  "{decisions_semicolon_separated}" \
  "{worked_semicolon_separated}" \
  "{requirements_csv}"
```

This atomically writes the entry. If the project already has a Keep a Changelog format, it adds a "Colony Work Log" separator section to keep both formats clean.

**Error handling:** If `changelog-append` fails, log to midden and continue — changelog failure never blocks phase advancement.

### Step 2.4: Commit Suggestion (Optional)

**This step is non-blocking. Skipping does not affect phase advancement or any subsequent steps. Failure to commit has zero consequences.**

After the phase is advanced and changelog updated, suggest a commit to preserve the milestone.

#### Step 2.4.1: Capture AI Description

**As the AI, briefly describe what was accomplished in this phase.**

Look at:
1. The phase PLAN.md `<objective>` section (what we set out to do)
2. Tasks that were marked complete
3. Files that were modified (from git diff --stat)
4. Any patterns or decisions recorded

**Provide a brief, memorable description** (10-15 words, imperative mood):
- Good: "Implement task-based model routing with keyword detection and precedence chain"
- Good: "Fix build timing by removing background execution from worker spawns"
- Bad: "Phase complete" (too vague)
- Bad: "Modified files in bin/lib" (too mechanical)

Store this as `ai_description` for the commit message.

#### Step 2.4.2: Generate Enhanced Commit Message

```bash
bash .aether/aether-utils.sh generate-commit-message "contextual" {phase_id} "{phase_name}" "{ai_description}" {plan_number}
```

Parse the returned JSON to extract:
- `message` - the commit subject line
- `body` - structured metadata (Scope, Files)
- `files_changed` - file count
- `subsystem` - derived subsystem name
- `scope` - phase.plan format

**Check files changed:**
```bash
git diff --stat HEAD 2>/dev/null | tail -5
```
If not in a git repo or no changes detected, skip this step silently.

**Display the enhanced suggestion:**
```
──────────────────────────────────────────────────
Commit Suggestion
──────────────────────────────────────────────────

  AI Description: {ai_description}

  Formatted Message:
  {message}

  Metadata:
  Scope: {scope}
  Files: {files_changed} files changed
  Preview: {first 5 lines of git diff --stat}

──────────────────────────────────────────────────
```

**Use AskUserQuestion:**
```
Commit this milestone?

1. Yes, commit with this message
2. Yes, but let me edit the description
3. No, I'll commit later
```

**If option 1 ("Yes, commit with this message"):**
```bash
git add -A && git commit -m "{message}" -m "{body}"
```
Display: `Committed: {message} ({files_changed} files)`

**If option 2 ("Yes, but let me edit"):**
Use AskUserQuestion to get the user's custom description:
```
Enter your description (or press Enter to keep: '{ai_description}'):
```
Then regenerate the commit message with the new description and commit.

**If option 3 ("No, I'll commit later"):**
Display: `Skipped. Your changes are saved on disk but not committed.`

**Record the suggestion to prevent double-prompting:**
Set `last_commit_suggestion_phase` to `{phase_id}` in COLONY_STATE.json (add the field at the top level if it does not exist).

**Error handling:** If any git command fails (not a repo, merge conflict, pre-commit hook rejection), display the error output and continue to the next step. The commit suggestion is advisory only -- it never blocks the flow.

Continue to Step 2.5 (Context Clear Suggestion), then to Step 2.7 (Project Completion) or Step 3 (Display Result).

### Step 2.5: Context Clear Suggestion (Optional)

**This step is non-blocking. Skipping does not affect phase advancement.**

After committing (or skipping commit), suggest clearing context to refresh before the next phase.

1. **Display the suggestion:**
```
──────────────────────────────────────────────────
Context Refresh
──────────────────────────────────────────────────

State is fully persisted and committed.
Phase {next_id} is ready to build.

──────────────────────────────────────────────────
```

2. **Use AskUserQuestion:**
```
Clear context now?

1. Yes, clear context then run /ant:build {next_id}
2. No, continue in current context
```

3. **If option 1 ("Yes, clear context"):**

   **IMPORTANT:** Claude Code does not support programmatic /clear. Display instructions:
   ```
   Please type: /clear
   
   Then run: /ant:build {next_id}
   ```
   
   Record the suggestion: Set `context_clear_suggested` to `true` in COLONY_STATE.json.

4. **If option 2 ("No, continue in current context"):**
   Display: `Continuing in current context. State is saved.`

Continue to Step 2.7 (Project Completion) or Step 3 (Display Result).

### Step 2.6: Update Context Document

After phase advancement is complete, update `.aether/CONTEXT.md`:

**Log the activity:**
```bash
bash .aether/aether-utils.sh context-update activity "continue" "Phase {prev_id} completed, advanced to {next_id}" "—"
```

**Update the phase:**
```bash
bash .aether/aether-utils.sh context-update update-phase {next_id} "{next_phase_name}" "YES" "Phase advanced, ready to build"
```

**Log any decisions from this session:**
If any architectural decisions were made during verification, also run:
```bash
bash .aether/aether-utils.sh context-update decision "{decision_description}" "{rationale}" "Queen"
```

### Step 2.7: Project Completion

Runs ONLY when all phases complete.

1. Read activity.log and errors.records
2. Display tech debt report:

```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
   🎉 P R O J E C T   C O M P L E T E 🎉
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

👑 Goal Achieved: {goal}
📍 Phases Completed: {total}

{if flagged_patterns:}
⚠️ Persistent Issues:
{list any flagged_patterns}
{end if}

🧠 Colony Learnings:
{condensed learnings from memory.phase_learnings}

👑 Wisdom Added to QUEEN.md:
{count} patterns/redirects/philosophies promoted across all phases

🐜 The colony rests. Well done!
```

3. Write summary to `.aether/data/completion-report.md`
4. Display next commands and stop.

### Step 3: Display Result

Output:

```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
   P H A S E   A D V A N C E M E N T
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

✅ Phase {prev_id}: {prev_name} -- COMPLETED

🧠 Learnings Extracted:
{list learnings added}

👑 Wisdom Promoted to QUEEN.md:
{for each promoted learning:}
   [{type}] {brief claim}
{end for}

🐜 Instincts Updated:
{for each instinct created or updated:}
   [{confidence}] {domain}: {action}
{end for}

─────────────────────────────────────────────────────

➡️ Advancing to Phase {next_id}: {next_name}
   {next_description}
   📋 Tasks: {task_count}
   📊 State: READY

──────────────────────────────────────────────────
🐜 Next Up
──────────────────────────────────────────────────
   /ant:build {next_id}     🔨 Build next phase
   /ant:status              📊 Check progress

💾 State persisted — context clear suggested above

📋 Context document updated at `.aether/CONTEXT.md`
```

**IMPORTANT:** In the "Next Steps" section above, substitute the actual phase number for `{next_id}` (calculated in Step 2 as `current_phase + 1`). For example, if advancing to phase 4, output `/ant:build 4` not `/ant:build {next_id}`.

### Step 4: Update Session

Update the session tracking file to enable `/ant:resume` after context clear:

```bash
bash .aether/aether-utils.sh session-update "/ant:continue" "/ant:build {next_id}" "Phase {prev_id} completed, advanced to Phase {next_id}"
```

Run using the Bash tool with description "Saving session state...": `bash .aether/aether-utils.sh session-update "/ant:continue" "/ant:build {next_id}" "Phase {prev_id} completed, advanced to Phase {next_id}"`
