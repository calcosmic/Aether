### Stage Audit Gate (Pre-Synthesis Check)

**This gate runs before Step 5.9. All 4 prior playbook stages must have completed with "ok" status.**

Check the cross-stage state record for each of the following stages:

| Stage | Playbook | Required Status |
|-------|----------|----------------|
| 1 | build-prep | ok |
| 2 | build-context | ok |
| 3 | build-wave | ok |
| 4 | build-verify | ok |

To verify, read the cross-stage state (typically `.aether/data/build-stage-state.json` or the in-memory record accumulated during the build run) and confirm each stage entry has `"status": "ok"`.

**If any stage is missing or did not complete with "ok" status:**
- HALT — do not proceed to Step 5.9
- Display:
  ```
  Stage Audit FAILED — cannot synthesize results.

  Missing or failed stages:
    {list each stage name with its recorded status, or "not recorded" if absent}

  Recovery options:
    1. Re-run the missing stage(s) manually, then retry /ant:build
    2. Run /ant:flags to review blockers
    3. Run /ant:swarm to auto-repair failed tasks
  ```
- Return `{"status": "failed", "summary": "Stage audit failed — stages {names} did not complete successfully"}` and stop.

**If all 4 stages passed:** proceed to Step 5.9.

### Step 5.9: Synthesize Results

**This step runs after all worker tasks have completed (Builders, Watcher, Chaos).**

Collect all worker outputs and create phase summary:

```json
{
  "status": "completed" | "failed" | "blocked",
  "summary": "...",
  "tasks_completed": [...],
  "tasks_failed": [...],
  "files_created": [...],
  "files_modified": [...],
  "spawn_metrics": {
    "spawn_count": {total workers spawned, including archaeologist if Step 4.5 fired, measurer if Step 5.5.1 fired, ambassador if Step 5.1.1 fired},
    "builder_count": {N},
    "watcher_count": 1,
    "chaos_count": 1,
    "archaeologist_count": {0 or 1, conditional on Step 4.5},
    "measurer_count": {0 or 1, conditional on Step 5.5.1},
    "ambassador_count": {0 or 1, conditional on Step 5.1.1},
    "parallel_batches": {number of waves}
  },
  "spawn_tree": {
    "{Archaeologist-Name}": {"caste": "archaeologist", "task": "pre-build history scan", "status": "completed"},
    "{Ambassador-Name}": {"caste": "ambassador", "task": "external integration design", "status": "completed"},
    "{Builder-Name}": {"caste": "builder", "task": "...", "status": "completed"},
    "{Watcher-Name}": {"caste": "watcher", "task": "verify", "status": "completed"},
    "{Measurer-Name}": {"caste": "measurer", "task": "performance baseline", "status": "completed"},
    "{Chaos-Name}": {"caste": "chaos", "task": "resilience testing", "status": "completed"}
  },
  "verification": {from Watcher output},
  "performance": {from Measurer output, or null if Step 5.5.1 was skipped},
  "resilience": {from Chaos Ant output},
  "archaeology": {from Archaeologist output, or null if Step 4.5 was skipped},
  "quality_notes": "..."
}
```

**Graveyard Recording:**
For each worker that returned `status: "failed"`:
  For each file in that worker's `files_modified` or `files_created`:
Run using the Bash tool with description "Recording failure grave...": `bash .aether/aether-utils.sh grave-add "{file}" "{ant_name}" "{task_id}" {phase} "{first blocker or summary}" && bash .aether/aether-utils.sh activity-log "GRAVE" "Queen" "Grave marker placed at {file} — {ant_name} failed: {summary}"`
  Then display a user-visible confirmation line:
  `⚰️ Grave recorded: {file} — {ant_name} failed ({summary})`

**Success capture: pattern synthesis (MEM-01):**

If `learning.patterns_observed` array in the synthesis JSON is non-empty, capture up to 2 patterns as success events:

Initialize a counter: `success_capture_count=0`

For each pattern in `learning.patterns_observed`:
- If `success_capture_count >= 2`, stop (cap reached)
- Run using the Bash tool with description "Capturing synthesis pattern success...":
```bash
bash .aether/aether-utils.sh memory-capture \
  "success" \
  "${pattern.trigger}: ${pattern.action} (evidence: ${pattern.evidence})" \
  "pattern" \
  "worker:builder" 2>/dev/null || true
```
- Increment `success_capture_count`

The cap of 2 prevents observation count inflation when builds produce many patterns. Each captured pattern enters learning-observations.json with a content hash for deduplication across builds.

### Step 5.9.1: Persist Builder Claims (MANDATORY)

Write builder file claims to `.aether/data/last-build-claims.json` for verification during /ant:continue.

Collect from each builder worker's output:
- files_created: [...all files_created from all builders...]
- files_modified: [...all files_modified from all builders...]

Run using the Bash tool with description "Persisting builder claims...":
```bash
echo '{"files_created":[...], "files_modified":[...], "build_phase": N, "timestamp": "ISO8601"}' > .aether/data/last-build-claims.json
```

Replace the `[...]` placeholders with the actual arrays collected from builder outputs, `N` with the current phase number, and `ISO8601` with the current timestamp.

This file is consumed by verify-claims during /ant:continue.

**Error Handoff Update:**
If workers failed, update handoff with error context for recovery:

Resolve the build error handoff template path:
  Check ~/.aether/system/templates/handoff-build-error.template.md first,
  then .aether/templates/handoff-build-error.template.md.

If no template found: output "Template missing: handoff-build-error.template.md. Run aether update to fix." and stop.

Read the template file. Fill all {{PLACEHOLDER}} values:
  - {{PHASE_NUMBER}} → current phase number
  - {{PHASE_NAME}} → current phase name
  - {{BUILD_TIMESTAMP}} → current ISO-8601 UTC timestamp
  - {{FAILED_WORKERS}} → formatted list of failed workers (one "- {ant_name}: {failure_summary}" per line)
  - {{GRAVE_MARKERS}} → formatted list of grave markers (one "- {file}: {caution_level} caution" per line)

Remove the HTML comment lines at the top of the template.
Write the result to .aether/HANDOFF.md using the Write tool.

Only fires when workers fail. Zero impact on successful builds.

--- SPAWN TRACKING ---

The spawn tree will be visible in `/ant:watch` because each spawn is logged.

--- OUTPUT FORMAT ---

Return JSON:
{
  "status": "completed" | "failed" | "blocked",
  "summary": "What the phase accomplished",
  "tasks_completed": ["1.1", "1.2"],
  "tasks_failed": [],
  "files_created": ["path1", "path2"],
  "files_modified": ["path3"],
  "spawn_metrics": {
    "spawn_count": 7,
    "watcher_count": 1,
    "chaos_count": 1,
    "archaeologist_count": 1,
    "measurer_count": 1,
    "ambassador_count": 1,
    "builder_count": 3,
    "parallel_batches": 2,
    "sequential_tasks": 1
  },
  "spawn_tree": {
    "Relic-8": {"caste": "archaeologist", "task": "pre-build history scan", "status": "completed", "children": {}},
    "Diplomat-7": {"caste": "ambassador", "task": "external integration design", "status": "completed", "children": {}},
    "Hammer-42": {"caste": "builder", "task": "...", "status": "completed", "children": {}},
    "Vigil-17": {"caste": "watcher", "task": "...", "status": "completed", "children": {}},
    "Benchmark-3": {"caste": "measurer", "task": "performance baseline", "status": "completed", "children": {}},
    "Entropy-9": {"caste": "chaos", "task": "resilience testing", "status": "completed", "children": {}}
  },
  "verification": {
    "build": {"command": "npm run build", "exit_code": 0, "passed": true},
    "tests": {"command": "npm test", "passed": 24, "failed": 0, "total": 24},
    "success_criteria": [
      {"criterion": "API endpoint exists", "evidence": "GET /api/users returns 200", "passed": true},
      {"criterion": "Tests cover happy path", "evidence": "3 tests in users.test.ts", "passed": true}
    ]
  },
  "debugging": {
    "issues_encountered": 0,
    "issues_resolved": 0,
    "fix_attempts": 0,
    "architectural_concerns": []
  },
  "tdd": {
    "cycles_completed": 5,
    "tests_added": 5,
    "tests_total": 47,
    "coverage_percent": 85,
    "all_passing": true
  },
  "learning": {
    "patterns_observed": [
      {
        "type": "success",
        "trigger": "when implementing API endpoints",
        "action": "use repository pattern with DI",
        "evidence": "All tests passed first try"
      }
    ],
    "instincts_applied": ["instinct_123"],
    "instinct_outcomes": [
      {"id": "instinct_123", "success": true}
    ]
  },
  "quality_notes": "Any concerns or recommendations",
  "ui_touched": true | false
}
```

### Step 6: Visual Checkpoint (if UI touched)

Parse synthesis result. If `ui_touched` is true:

```
━━━ 🖼️🐜 V I S U A L   C H E C K P O I N T ━━━

UI changes detected. Verify appearance before continuing.

Files touched:
{list files from files_created + files_modified that match UI patterns}

Options:
  1. Approve - UI looks correct
  2. Reject - needs changes (describe issues)
  3. Skip - defer visual review
```

Use AskUserQuestion to get approval. Record in events:
- If approved: `"<timestamp>|visual_approved|build|Phase {id} UI approved"`
- If rejected: `"<timestamp>|visual_rejected|build|Phase {id} UI rejected: {reason}"`

### Step 6.5: Update Handoff Document

After synthesis is complete, update the handoff document with current state for session recovery:

```bash
# Update handoff with build results
jq -n \
  --arg timestamp "$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
  --arg goal "$(jq -r '.goal' .aether/data/COLONY_STATE.json)" \
  --arg phase "$(jq -r '.current_phase' .aether/data/COLONY_STATE.json)" \
  --arg phase_name "{phase_name}" \
  --arg status "{synthesis.status}" \
  --arg summary "{synthesis.summary}" \
  --argjson tasks_completed '{synthesis.tasks_completed | length}' \
  --argjson tasks_failed '{synthesis.tasks_failed | length}' \
  --arg next_action "{if synthesis.status == "completed" then "/ant:continue" else "/ant:flags" end}" \
  '{
    "last_updated": $timestamp,
    "goal": $goal,
    "current_phase": $phase,
    "phase_name": $phase_name,
    "build_status": $status,
    "summary": $summary,
    "tasks_completed": $tasks_completed,
    "tasks_failed": $tasks_failed,
    "next_recommended_action": $next_action,
    "can_resume": true,
    "note": "Phase build completed. Run /ant:continue to advance if verification passed."
  }' > .aether/data/last-build-result.json
```

Resolve the build success handoff template path:
  Check ~/.aether/system/templates/handoff-build-success.template.md first,
  then .aether/templates/handoff-build-success.template.md.

If no template found: output "Template missing: handoff-build-success.template.md. Run aether update to fix." and stop.

Read the template file. Fill all {{PLACEHOLDER}} values:
  - {{GOAL}} → colony goal (from COLONY_STATE.json)
  - {{PHASE_NUMBER}} → current phase number
  - {{PHASE_NAME}} → current phase name
  - {{BUILD_STATUS}} → synthesis.status
  - {{BUILD_TIMESTAMP}} → current ISO-8601 UTC timestamp
  - {{BUILD_SUMMARY}} → synthesis summary
  - {{TASKS_COMPLETED}} → count of completed tasks
  - {{TASKS_FAILED}} → count of failed tasks
  - {{FILES_CREATED}} → count of created files
  - {{FILES_MODIFIED}} → count of modified files
  - {{SESSION_NOTE}} → "Build succeeded — ready to advance." if status is completed, else "Build completed with issues — review before continuing."

Remove the HTML comment lines at the top of the template.
Write the result to .aether/HANDOFF.md using the Write tool.

This ensures the handoff always reflects the latest build state, even if the session crashes before explicit pause.

### Step 6.6: Update Context Document

Log this build activity to `.aether/CONTEXT.md`:

Run using the Bash tool with description "Updating build context...": `bash .aether/aether-utils.sh context-update activity "build {phase_id}" "{synthesis.status}" "{files_created_count + files_modified_count}" && bash .aether/aether-utils.sh context-update build-complete "{synthesis.status}" "{synthesis.status == 'completed' ? 'success' : 'failed'}"`

Also update safe-to-clear status:
- If build completed successfully: `context-update safe-to-clear "YES" "Build complete, ready to continue"`
- If build failed: `context-update safe-to-clear "NO" "Build failed — run /ant:swarm or /ant:flags"`

### Step 6.7: Check for Promotion Proposals

After build completion (success or failure), check if any observations have met promotion thresholds.

Run using the Bash tool with description "Checking for wisdom promotions...":
```bash
proposals=$(bash .aether/aether-utils.sh learning-check-promotion 2>/dev/null || echo '{"proposals":[]}')
proposal_count=$(echo "$proposals" | jq '.proposals | length')
echo "{\"proposal_count\": $proposal_count}"
```

Parse the result. If proposal_count > 0:
- Display: "📚 $proposal_count wisdom proposal(s) ready for review"
- Run: `bash .aether/aether-utils.sh learning-approve-proposals`
- This presents the one-at-a-time UI for user review

If proposal_count == 0:
- Silently continue (no output needed per user decision)

Note: This runs regardless of build success/failure. Failed builds may have recorded failure observations that are ready for promotion.

### Step 7: Display Results

**This step runs ONLY after synthesis is complete. All values come from actual worker results.**

**Display BUILD SUMMARY (always shown, replaces compact/verbose split):**

Calculate `total_tools` by summing `tool_count` from all worker return JSONs (builders + watcher + chaos).
Calculate `elapsed` using `build_started_at_epoch` (epoch integer captured at Step 5 start by Plan 01): `$(( $(date +%s) - build_started_at_epoch ))` formatted as Xm Ys.

```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
🔨 B U I L D   S U M M A R Y
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
📍 Phase {id}: {name}
🎲 Pattern:  {selected_pattern}

🐜 Workers:  {pass_count} passed  {fail_count} failed  ({total} total)
🛠️ Tools:    {total_tools} calls across all workers
⏱️ Duration: {elapsed}

{if measurer_ran:}
📊 Measurer: {baseline_count} baselines established, {bottleneck_count} bottlenecks identified
{end if}

{if ambassador_ran:}
🔌 Ambassador: Integration plan for {integration_plan.service_name} ready
{end if}

{if fail_count > 0:}
Failed:
  {for each failed worker:}
  {caste_emoji} {Ant-Name}: {task_description} ✗ ({failure_reason} after {tool_count} tools)
  {end for}

Retry: /ant:swarm to auto-repair failed tasks, or /ant:flags to review blockers
{end if}
```

**If verbose_mode is true**, additionally show the spawn tree and TDD details after the BUILD SUMMARY block (keep the existing verbose-only sections: Colony Work Tree, Tasks Completed, TDD, Patterns Learned, Debugging, Model Routing). Prepend with:
```
━━ Details (--verbose) ━━
```

After displaying the BUILD SUMMARY (and optional verbose details), call the Next Up helper by running using the Bash tool with description "Displaying next steps...":
```bash
state=$(jq -r '.state // "IDLE"' .aether/data/COLONY_STATE.json 2>/dev/null || echo "IDLE")
current_phase=$(jq -r '.current_phase // 0' .aether/data/COLONY_STATE.json 2>/dev/null || echo "0")
total_phases=$(jq -r '.plan.phases | length' .aether/data/COLONY_STATE.json 2>/dev/null || echo "0")
bash .aether/aether-utils.sh print-next-up "$state" "$current_phase" "$total_phases"
```

**Routing Note:** The state-based Next Up block above routes based on colony state. If verification failed or blockers exist, review `/ant:flags` before continuing.

**IMPORTANT:** Build does NOT update task statuses or advance state. Run `/ant:continue` to:
- Mark tasks as completed
- Extract learnings
- Advance to next phase

### Step 8: Update Session

Update the session tracking file to enable `/ant:resume` after context clear:

Run using the Bash tool with description "Saving build session...": `bash .aether/aether-utils.sh session-update "/ant:build {phase_id}" "/ant:continue" "Phase {phase_id} build completed: {synthesis.status}"`
