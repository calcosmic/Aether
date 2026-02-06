---
name: ant:continue
description: Queen approves phase completion and clears check-in for colony to proceed
---

You are the **Queen Ant Colony**. Advance to the next phase.

## Instructions

### Step 0: Parse Arguments

Check if `$ARGUMENTS` contains `--all` or `all`.

- If `--all` or `all` is present: set `auto_mode = true`
- Otherwise: set `auto_mode = false`

Proceed to Step 1.

### Step 1: Read State

Use the Read tool to read `.aether/data/COLONY_STATE.json`.

Extract:
- `goal`, `state`, `current_phase`, `mode` from top level
- `plan.phases` for phase data
- `signals` for pheromone guidance
- `errors.records` and `errors.flagged_patterns` for error context
- `memory.phase_learnings` and `memory.decisions` for learnings
- `events` for activity history

If `COLONY_STATE.json` has `goal: null`, output `No colony initialized. Run /ant:init first.` and stop.

If `plan.phases` is empty, output `No project plan. Run /ant:plan first.` and stop.

### Step 1.5: Auto-Continue Loop (only if auto_mode is true)

If `auto_mode` is false, skip this step entirely and proceed to Step 2 (normal single-phase flow).

If `auto_mode` is true:

Use `plan.phases` from COLONY_STATE.json to determine how many phases remain. Calculate `remaining_phases` as all phases with status NOT equal to `"completed"`.

If no phases remain, output "All phases already complete." and proceed to Step 8.

Display:
```
+=====================================================+
|  AUTO-CONTINUE                                       |
+=====================================================+

Running all remaining phases automatically.
Halt conditions: watcher score < 4, or 2 consecutive failures.
Phases remaining: <count>
```

Initialize: `consecutive_failures = 0`, `phase_results = []`

**For each remaining phase (in order):**

1. Display: `"\n--- Auto-Continue: Phase <N>/<total> - <phase_name> ---\n"`

2. **Build the phase** using the **Task tool** with `subagent_type="general-purpose"`:
   ```
   You are executing /ant:build <phase_number> as part of an auto-continue run.

   Read the file .claude/commands/ant/build.md using the Read tool.
   Follow ALL instructions from Step 1 through Step 7e exactly,
   with $ARGUMENTS = "<phase_number>".

   IMPORTANT: In auto-continue mode, auto-approve the Phase Lead's plan
   in Step 5b WITHOUT asking the user. Skip the "Proceed with this plan?"
   prompt and proceed directly to Step 5c.

   Execute all steps. Return your result including:
   - The watcher quality_score (number 1-10)
   - The watcher recommendation ("approve" or "request_changes")
   - A one-line summary of what was built
   ```

3. **Check halt conditions** after build returns:
   - Extract `quality_score` from the Task result
   - If `quality_score < 4`:
     - Display: `"AUTO-CONTINUE HALTED: Phase <N> scored <score>/10. Manual review needed."`
     - Display: `"Run /ant:status to inspect, then /ant:build <N> to retry manually."`
     - Break out of the loop and proceed to the cumulative display
   - If build reported failure (no quality_score returned or task errored):
     - Increment `consecutive_failures`
     - If `consecutive_failures >= 2`:
       - Display: `"AUTO-CONTINUE HALTED: 2 consecutive phase failures."`
       - Break out of the loop
     - Otherwise: continue to next phase
   - If `quality_score >= 4`:
     - Reset `consecutive_failures = 0`

4. **Run continue logic** for this phase: Execute Steps 3-7 of this command (phase completion summary, extract learnings, auto-emit pheromones, clean pheromones, write events, update colony state) for the just-completed phase.

5. **Record phase result:**
   - Append to `phase_results`: `{phase: <N>, name: "<name>", score: <quality_score>, status: "COMPLETE" or "FAILED"}`
   - Display condensed summary:
     ```
     Phase <N>: <name> -- <COMPLETE|FAILED> (quality: <score>/10)
     ```

**After the loop completes (or halts), display cumulative results:**

```
+=====================================================+
|  AUTO-CONTINUE COMPLETE                              |
+=====================================================+

Phases processed: <count>
  {for each phase_result:}
  {pass or fail} Phase <N>: <name> (quality: <score>/10)

{if halted:}
Halted at: Phase <N> -- <reason>
{/if}
```

Then proceed to Step 8 to display the normal result (showing the NEXT unbuilt phase, or "all complete" if done).

### Step 2: Determine Next Phase

Look at `current_phase` in `COLONY_STATE.json`. The next phase is `current_phase + 1`.

If there is no next phase (current is the last phase), proceed to Step 2.5 to generate a tech debt report, then output the completion message. Do NOT stop yet.

If there IS a next phase, skip Step 2.5 entirely and proceed to Step 3.

### Step 2.5: Generate Tech Debt Report (Project Completion)

This step runs ONLY when Step 2 detects all phases are complete (no next phase). It MUST NOT run on normal mid-project continue calls.

**1. Gather data** (all reads can be parallel):
- Use `errors.records` and `errors.flagged_patterns` from COLONY_STATE.json (already in memory from Step 1)
- Run: `bash ~/.aether/aether-utils.sh error-summary`
- Run: `bash ~/.aether/aether-utils.sh error-pattern-check`
- Use `memory.phase_learnings` from COLONY_STATE.json (already in memory from Step 1)
- Read `.aether/data/activity.log` (for cross-phase activity patterns)

**2. Synthesize the report:**

Display:
```
TECH DEBT REPORT
================

Project: {goal from COLONY_STATE.json}
Phases Completed: {count of completed phases}
Total Build Time: {estimate from activity log timestamps -- first to last entry}

Persistent Issues:
{for each entry in errors.flagged_patterns from COLONY_STATE.json:}
  {category} ({count} occurrences, phases {first_seen} - {last_seen}):
    {description}

{if no flagged_patterns:}
  None -- no recurring error patterns detected.

Error Summary:
  Total: {total from error-summary}
  By Severity: Critical: {n}, High: {n}, Medium: {n}, Low: {n}
  By Category: {category}: {n}, ...

{if total == 0:}
  No errors recorded during project execution.

Unresolved High-Severity Items:
  {errors from errors.records with severity "critical" or "high" that were never followed by a corresponding fix or resolution}

{if none:}
  None -- all high-severity items were addressed.

Phase Quality Trend:
  {for each phase_learning in memory.phase_learnings:}
  Phase {phase}: {phase_name} -- {errors_encountered} errors
  {extract watcher quality scores from events array if available}

Recommendations:
  {1-3 actionable items synthesized from the patterns above}
  {e.g., "The {category} error pattern persisted across {N} phases -- consider adding automated {category} checks to your CI pipeline."}
```

**3. Persist the report:**
Write the full report to `.aether/data/tech-debt-report.md` (both display AND file persistence).

### Step 2.5b: Promote Learnings to Global Tier

After generating the tech debt report, offer learning promotion.

**Auto-continue guard:** If `auto_mode` is true (set in Step 0), skip this entire step. Display:
  "Global learning promotion available. Run /ant:continue (without --all) to promote learnings."
Proceed to Step 2.5c.

**If auto_mode is false:**

1. Use `memory.phase_learnings` array from COLONY_STATE.json (already in memory from Step 1)
2. If phase_learnings is empty: display "No project learnings to promote." and skip to Step 2.5c

3. Analyze each learning and categorize:
   - **Promotion candidates:** Learnings that reference specific tech, patterns, or practices applicable across projects
     - Example good candidate: "bcrypt with 12 rounds causes 800ms delay -- use 10 rounds"
     - Example good candidate: "Integration tests caught missing error handlers that unit tests missed"
   - **Project-specific:** Learnings too tied to this project's details
     - Example: "Phase 3 had 2 errors"
     - Example: "Auth routes needed 3 retries"

4. Display learnings with promotion suggestions:
   ```
   PROJECT LEARNINGS:

   Candidates for global promotion (applicable across projects):
     [1] "<learning text>" (Phase <N>)
     [2] "<learning text>" (Phase <N>)

   Project-specific (not recommended for promotion):
     [-] "<learning text>" (Phase <N>)

   Which learnings would you like to promote? (numbers, "all candidates", or "none")
   ```

5. Wait for user selection.

6. For each selected learning:
   - Use `goal` field from COLONY_STATE.json (already in memory from Step 1)
   - Infer tags from the colonization decision in `memory.decisions` (look for the `type: "colonization"` decision entry -- extract tech stack keywords like language names, framework names, domain keywords)
   - Run:
     ```
     bash ~/.aether/aether-utils.sh learning-promote "<learning_content>" "<goal>" <phase_number> "<comma_separated_tags>"
     ```
   - If result contains `"promoted":true`: note success
   - If result contains `"reason":"cap_reached"`:
     - Display: "Global learnings at capacity (50/50). Remove an existing learning to make room."
     - Read `~/.aether/learnings.json` and display existing learnings with indices
     - Ask user which to remove (by index)
     - Remove the selected entry from the file using jq, then retry the promotion

7. Display promotion result:
   ```
   Promoted {N} learnings to global tier.
     ~/.aether/learnings.json ({count}/50 entries)
   ```

### Step 2.5c: Display completion message and stop

```
All phases complete. Colony has finished the project plan.
  Tech debt report: .aether/data/tech-debt-report.md
  Global learnings: ~/.aether/learnings.json ({count}/50 entries, or "none promoted" if skipped)

  /ant:status   View final colony status
  /ant:plan     Generate a new plan (will replace current)
```

Stop here.

### Step 3: Phase Completion Summary

Before advancing, display a summary of the completed phase using data from the state files read in Step 1.

Output:

```
PHASE <N> REVIEW: <phase_name>
───────────────────────────────
  Tasks:
    ✅ <task_id>: <description>
    ✅ <task_id>: <description>
    ❌ <task_id>: <description> (deferred)
    ...
    Completed: <N>/<total>

  Errors:
    <count> errors encountered
    (list severity counts: N critical, N high, N medium, N low)

  Decisions:
    <count> decisions logged during this phase
    (list last 3 decisions from memory.decisions array: "<content>")

---------------------------------------------------
```

Get task data from `plan.phases` in COLONY_STATE.json -- look at the current phase's `tasks` array. Show `[x]` for completed, `[ ]` for incomplete.

Use the Bash tool to run:
```
bash ~/.aether/aether-utils.sh error-summary
```

This returns JSON: `{"ok":true,"result":{"total":N,"by_category":{...},"by_severity":{...}}}`. Use the `by_severity` counts for the display above.

For phase-specific error counts, also filter `errors.records` entries where `phase` matches the current phase number.

If the command fails, fall back to showing "Errors: Unable to compute".

Get decision data from `memory.decisions` -- count the array entries. Show last 3 decisions.

If no errors were encountered during this phase:
```
  Errors: None
```

If no decisions were logged:
```
  Decisions: None
```

This step is DISPLAY ONLY -- it reads state but does not write anything. The purpose is to give the user a retrospective before the phase advances.

### Step 4: Extract Phase Learnings

**Duplicate Detection:** Before extracting learnings, check `events` array (already in memory from Step 1) for an event where:
- Event contains `"auto_learnings_extracted"` (pipe-delimited format)
- Event content contains `"Phase <current_phase_number>:"`

If such an event is found AND `$ARGUMENTS` does NOT contain "force" or "--force":
- Output: `"📚 Learnings already captured during build (auto-extracted at <event_timestamp>) -- skipping extraction."`
- Skip the rest of Step 4 AND Step 4.5 (pheromone emission)
- Proceed directly to Step 5

If no matching event is found, OR if `$ARGUMENTS` contains "force" or "--force":
- Proceed with the existing extraction logic below

---

Review the completed phase by analyzing:
- Tasks completed in this phase (from `plan.phases` -- look at the current phase's tasks)
- Errors encountered during this phase (from `errors.records` -- filter by `phase` field matching current phase)
- Events that occurred (from `events` array -- recent events related to this phase)
- Flagged patterns (from `errors.flagged_patterns` array)

Append a phase learning entry to `memory.phase_learnings` array in COLONY_STATE.json:

```json
{
  "id": "learn_<unix_timestamp>_<4_random_hex>",
  "phase": <current_phase_number>,
  "phase_name": "<phase name from plan.phases>",
  "learnings": [
    "<specific thing learned -- what worked, what didn't, what to remember>",
    "<another specific learning>"
  ],
  "errors_encountered": <count of errors with this phase number>,
  "timestamp": "<ISO-8601 UTC>"
}
```

Learnings must be SPECIFIC and ACTIONABLE. Good: "TypeScript strict mode caught 12 type errors early." Bad: "Phase completed successfully." Draw from actual task outcomes, errors, and events -- not boilerplate.

Write the updated COLONY_STATE.json.

Then use the Bash tool to run:
```
bash ~/.aether/aether-utils.sh memory-compress
```

This enforces retention limits (phase_learnings <= 20, decisions <= 30) and compresses if over token threshold. Returns `{"ok":true,"result":{"compressed":true,"tokens":N}}`.

If the command fails, the Write tool already saved the data -- no action needed.

**Update Spawn Outcomes:** Review the `events` array for events containing `phase_completed` or `phase_failed` related to the current phase. If the phase completed successfully, look at events for spawn-related entries or the build report to identify which castes contributed. For each identified caste, increment `alpha` and `successes` in `spawn_outcomes`. If the phase failed, increment `beta` and `failures` for identified castes. Increment `total_spawns` regardless. Include in the COLONY_STATE.json write.

If no castes can be identified from events, skip this step.

### Step 4.5: Auto-Emit Pheromones

If Step 4 was skipped due to auto-extraction detection, skip this step as well and proceed to Step 5.

After extracting learnings, automatically emit pheromones based on phase outcomes.

**Always emit a FEEDBACK pheromone** summarizing what worked and what didn't from the phase learnings.

Append to `signals` array in COLONY_STATE.json:

```json
{
  "id": "auto_<unix_timestamp>_<4_random_hex>",
  "type": "FEEDBACK",
  "content": "<summary of what worked and what didn't from the phase learnings — be specific, reference actual task outcomes>",
  "strength": 0.5,
  "half_life_seconds": 21600,
  "created_at": "<ISO-8601 UTC>",
  "source": "auto:continue",
  "auto": true
}
```

**Conditionally emit a REDIRECT pheromone** if `errors.flagged_patterns` has any entries related to this phase (check if any flagged pattern's errors occurred during this phase):

```json
{
  "id": "auto_<unix_timestamp>_<4_random_hex>",
  "type": "REDIRECT",
  "content": "Avoid repeating: <description of the flagged pattern and its root causes>",
  "strength": 0.9,
  "half_life_seconds": 86400,
  "created_at": "<ISO-8601 UTC>",
  "source": "auto:continue",
  "auto": true
}
```

Before appending each pheromone, validate its content. Use the Bash tool to run:
```
bash ~/.aether/aether-utils.sh pheromone-validate "<the pheromone content string>"
```

This returns JSON: `{"ok":true,"result":{"pass":true|false,...}}`.

**If `pass` is false:** Do not append this pheromone. Instead, append a rejection event to `events` array as pipe-delimited string:
`"<timestamp>|pheromone_rejected|continue|Auto-pheromone rejected: <reason> (length: <N>, min: 20)"`

**If `pass` is true:** Proceed to append the pheromone to `signals` array.

If the command fails (non-zero exit), skip validation and append the pheromone anyway (fail-open for auto-emitted pheromones).

**Log Events:** For each auto-emitted pheromone, append to `events` array as pipe-delimited string:
`"<timestamp>|pheromone_auto_emitted|continue|<TYPE> pheromone auto-emitted: <first 80 chars of content>"`

If the `events` array exceeds 100 entries, remove the oldest entries to keep only 100. Write the updated COLONY_STATE.json.

### Step 5: Clean Expired Pheromones

Use the Bash tool to run:
```
bash ~/.aether/aether-utils.sh pheromone-cleanup
```

This removes signals with `current_strength` below 0.05 from the `signals` array in COLONY_STATE.json and returns `{"ok":true,"result":{"removed":N,"remaining":N}}`. The cleanup result (removed count) can be mentioned in the display output.

### Step 6: Write Events

**Only if Step 4 actually ran** (learnings were extracted, not skipped due to auto-extraction detection), append a `learnings_extracted` event to `events` array as pipe-delimited string:
`"<timestamp>|learnings_extracted|continue|Extracted <N> learnings from Phase <id>: <name>"`

If Step 4 was skipped, do NOT write this event (it was already written during the build).

**Always** append a `phase_advanced` event as pipe-delimited string:
`"<timestamp>|phase_advanced|continue|Advanced from Phase <current> to Phase <next>"`

If the `events` array exceeds 100 entries, remove the oldest entries to keep only 100.

Include in the COLONY_STATE.json write.

### Step 7: Update Colony State

Use the Write tool to update `.aether/data/COLONY_STATE.json`:
- Set `current_phase` to the next phase number
- Set `state` to `"READY"`
- Set all workers to `"idle"`

This write should include all updates accumulated from Steps 4, 4.5, and 6 (learnings, signals, events, spawn_outcomes).

### Step 8: Display Result

Output this header at the start of your response:

```
+=====================================================+
|  👑 AETHER COLONY :: CONTINUE                        |
+=====================================================+
```

Then show step progress.

If `auto_mode` was true, show:

```
  ✓ Step 0: Parse Arguments (--all mode)
  ✓ Step 1: Read State
  ✓ Step 1.5: Auto-Continue Loop (<N> phases)
  ✓ Step 8: Display Result
```

(Steps 2-7 are executed per-phase inside the loop, not shown individually.)

If `auto_mode` was false, show the normal progress:

```
  ✓ Step 0: Parse Arguments
  ✓ Step 1: Read State
  ✓ Step 2: Determine Next Phase
  ✓ Step 3: Phase Completion Summary
  ✓ Step 4: Extract Phase Learnings
  ✓ Step 4.5: Auto-Emit Pheromones
  ✓ Step 5: Clean Expired Pheromones
  ✓ Step 6: Write Events
  ✓ Step 7: Update Colony State
  ✓ Step 8: Display Result
```

Then output a divider and the result:

```
---

Phase <current> approved. Advancing to Phase <next>.

  Phase <next>: <name>
  <description>

  Tasks: <count>
  State: READY

  Learnings Extracted:
    {if Step 4 ran:}
    - <learning 1>
    - <learning 2>
    {if Step 4 was skipped:}
    (already captured during build -- see auto-extraction event)

  🧪 Auto-Emitted Pheromones:
    FEEDBACK (0.5, 6h): "<first 80 chars of content>"
    {if REDIRECT was emitted:}
    REDIRECT (0.9, 24h): "<first 80 chars of content>"

Next Steps:
  /ant:build <next>      Start building Phase <next>
  /ant:phase <next>      Review phase details first
  /ant:focus "<area>"    Guide colony attention before building
  /ant:redirect "<pat>"  Set constraints before building
```

### Step 9: Persistence Confirmation

After displaying the result above, add a state persistence confirmation:

```
---
All state persisted. Safe to /clear context if needed.
  State: .aether/data/ (6 files validated)
  Resume: /ant:resume-colony
```
