### Step 2: Update State

Find current phase in `plan.phases`.
Determine next phase (`current_phase + 1`).

**If no next phase (all complete):** Skip to Step 2.4 (commit suggestion), then Step 2.7 (completion).

Update COLONY_STATE.json:

1. **Mark current phase completed:**
   - Set `plan.phases[current].status` to `"completed"`
   - Set all tasks in phase to `"completed"`

2. **Extract learnings (with validation status):**

   **CRITICAL: Learnings start as HYPOTHESES until verified.**

   A learning is only "validated" if:
   - The code was actually run and tested
   - The feature works in practice, not just in theory
   - User has confirmed the behavior

   Append to `memory.phase_learnings`:
   ```json
   {
     "id": "learning_<unix_timestamp>",
     "phase": <phase_number>,
     "phase_name": "<name>",
     "learnings": [
       {
         "claim": "<specific actionable learning>",
         "status": "hypothesis",
         "tested": false,
         "evidence": "<what observation led to this>",
         "disproven_by": null
       }
     ],
     "timestamp": "<ISO-8601>"
   }
   ```

   **Status values:**
   - `hypothesis` - Recorded but not verified (DEFAULT)
   - `validated` - Tested and confirmed working
   - `disproven` - Found to be incorrect

   **Do NOT record a learning if:**
   - It wasn't actually tested
   - It's stating the obvious
   - There's no evidence it works

2.5. **Capture learnings through memory pipeline:**

   For each learning extracted, run the memory pipeline (observation + auto-pheromone + auto-promotion check).

   Run using the Bash tool with description "Recording learning observations...":
   ```bash
   colony_name=$(jq -r '.session_id | split("_")[1] // "unknown"' .aether/data/COLONY_STATE.json 2>/dev/null || echo "unknown")

   # Get learnings from the current phase
   current_phase_learnings=$(jq -r --argjson phase "$current_phase" '.memory.phase_learnings[] | select(.phase == $phase)' .aether/data/COLONY_STATE.json 2>/dev/null || echo "")

   if [[ -n "$current_phase_learnings" ]]; then
     echo "$current_phase_learnings" | jq -r '.learnings[]?.claim // empty' 2>/dev/null | while read -r claim; do
       if [[ -n "$claim" ]]; then
         bash .aether/aether-utils.sh memory-capture "learning" "$claim" "pattern" "worker:continue" 2>/dev/null || true
       fi
     done
     echo "Recorded observations for threshold tracking"
   else
     echo "No learnings to record"
   fi
   ```

   This records each learning in `learning-observations.json` with:
   - Content hash for deduplication (same claim across phases increments count)
   - Observation count (increments if seen before)
   - Colony name for cross-colony tracking

   Memory capture also auto-emits a FEEDBACK pheromone and attempts auto-promotion when recurrence policy is met.

3. **Extract instincts from patterns:**

   Read activity.log for patterns from this phase's build.

   For each pattern observed (success, error_resolution, user_feedback):

   **If pattern matches existing instinct:**
   - Update confidence: +0.1 for success outcome, -0.1 for failure
   - Increment applications count
   - Update last_applied timestamp

   **If new pattern:**
   - Create new instinct with initial confidence:
     - success: 0.4
     - error_resolution: 0.5
     - user_feedback: 0.7

   Append to `memory.instincts`:
   ```json
   {
     "id": "instinct_<unix_timestamp>",
     "trigger": "<when X>",
     "action": "<do Y>",
     "confidence": 0.5,
     "status": "hypothesis",
     "domain": "<testing|architecture|code-style|debugging|workflow>",
     "source": "phase-<id>",
     "evidence": ["<specific observation that led to this>"],
     "tested": false,
     "created_at": "<ISO-8601>",
     "last_applied": null,
     "applications": 0,
     "successes": 0,
     "failures": 0
   }
   ```

   **Instinct confidence updates:**
   - Success when applied: +0.1, increment `successes`
   - Failure when applied: -0.15, increment `failures`
   - If `failures` >= 2 and `successes` == 0: mark `status: "disproven"`
   - If `successes` >= 2 and tested: mark `status: "validated"`

   Cap: Keep max 30 instincts (remove lowest confidence when exceeded).

4. **Advance state:**
   - Set `current_phase` to next phase number
   - Set `state` to `"READY"`
   - Set `build_started_at` to null
   - Append event: `"<timestamp>|phase_advanced|continue|Completed Phase <id>, advancing to Phase <next>"`

5. **Cap enforcement:**
   - Keep max 20 phase_learnings
   - Keep max 30 decisions
   - Keep max 30 instincts (remove lowest confidence)
   - Keep max 100 events

Write COLONY_STATE.json.

Validate the state file:
Run using the Bash tool with description "Validating colony state...": `bash .aether/aether-utils.sh validate-state colony`

### Step 2.1: Auto-Emit Phase Pheromones (SILENT)

**This entire step produces NO user-visible output.** All pheromone operations run silently — learnings are deposited in the background. If any pheromone call fails, log the error and continue. Phase advancement must never fail due to pheromone errors.

#### 2.1a: Auto-emit FEEDBACK pheromone for phase outcome

After learning extraction completes in Step 2, auto-emit a FEEDBACK signal summarizing the phase:

```bash
# phase_id and phase_name come from Step 2 state update
# Take the top 1-3 learnings by evidence strength from memory.phase_learnings
# Compress into a single summary sentence

# If learnings were extracted, build a brief summary from them (first 1-3 claims)
# Otherwise use the minimal fallback
phase_feedback="Phase $phase_id ($phase_name) completed. Key patterns: {brief summary of 1-3 learnings from Step 2}"
# Fallback if no learnings: "Phase $phase_id ($phase_name) completed without notable patterns."

bash .aether/aether-utils.sh pheromone-write FEEDBACK "$phase_feedback" \
  --strength 0.6 \
  --source "worker:continue" \
  --reason "Auto-emitted on phase advance: captures what worked and what was learned" \
  --ttl "30d" 2>/dev/null || true
```

The strength is 0.6 (auto-emitted = lower than user-emitted 0.7). Source is "worker:continue" to distinguish from user-emitted feedback. TTL is 30d so it survives phase transitions and can guide subsequent work.

#### 2.1b: Auto-emit REDIRECT for recurring error patterns

Check `errors.flagged_patterns[]` in COLONY_STATE.json for patterns that have appeared in 2+ phases:

```bash
flagged_patterns=$(jq -r '.errors.flagged_patterns[]? | select(.count >= 2) | .pattern' .aether/data/COLONY_STATE.json 2>/dev/null || true)
```

For each pattern returned by the above query, emit a REDIRECT signal:

```bash
bash .aether/aether-utils.sh pheromone-write REDIRECT "$pattern_text" \
  --strength 0.7 \
  --source "system" \
  --reason "Auto-emitted: error pattern recurred across 2+ phases" \
  --ttl "30d" 2>/dev/null || true
```

REDIRECT strength is 0.7 (higher than auto FEEDBACK 0.6 — anti-patterns produce stronger signals than successes). TTL is 30d (not phase_end) because recurring errors should persist across multiple phases.

Also capture each recurring pattern as a resolution candidate so the colony can promote "finally fixed" lessons over time:

```bash
bash .aether/aether-utils.sh memory-capture \
  "resolution" \
  "$pattern_text" \
  "pattern" \
  "worker:continue" 2>/dev/null || true
```

This writes a compact rolling summary entry, emits FEEDBACK guidance, and contributes to recurrence-based promotion in QUEEN wisdom.

If `errors.flagged_patterns` doesn't exist or is empty, skip silently.

#### 2.1c: Expire phase_end signals and archive to midden

After auto-emission, expire all signals with `expires_at == "phase_end"`. The FEEDBACK from 2.1a uses a 30d TTL and is not affected by this step.

Run using the Bash tool with description "Maintaining pheromone memory...": `bash .aether/aether-utils.sh pheromone-expire --phase-end-only 2>/dev/null && bash .aether/aether-utils.sh eternal-init 2>/dev/null`

This is idempotent — runs every time continue fires but only creates the directory/file once.

### Step 2.1.5: Check for Promotion Proposals (PHER-EVOL-02)

After extracting learnings, check for observations that have met promotion thresholds and present the tick-to-approve UX.

**Check for --deferred flag:**

If `$ARGUMENTS` contains `--deferred`:
```bash
if [[ "$ARGUMENTS" == *"--deferred"* ]] && [[ -f .aether/data/learning-deferred.json ]]; then
  echo "📦 Reviewing deferred proposals..."
  bash .aether/aether-utils.sh learning-approve-proposals --deferred ${verbose:+--verbose}
fi
```

**Normal proposal flow (MEM-01: Silent skip if empty):**

1. **Check for proposals:**
   ```bash
   proposals=$(bash .aether/aether-utils.sh learning-check-promotion 2>/dev/null || echo '{"proposals":[]}')
   proposal_count=$(echo "$proposals" | jq '.proposals | length')
   ```

2. **If proposals exist, invoke the approval workflow:**

   Only show the approval UI when there are actual proposals to review:

   ```bash
   if [[ "$proposal_count" -gt 0 ]]; then
     verbose_flag=""
     [[ "$ARGUMENTS" == *"--verbose"* ]] && verbose_flag="--verbose"
     bash .aether/aether-utils.sh learning-approve-proposals $verbose_flag
   fi
   # If no proposals, silently skip without notice (per user decision)
   ```

   The learning-approve-proposals function handles:
   - Displaying proposals with checkbox UI
   - Capturing user selection
   - Executing batch promotions via queen-promote
   - Deferring unselected proposals
   - Offering undo after successful promotions
   - Logging PROMOTED activity

**Skip conditions:**
- learning-check-promotion returns empty or fails
- No proposals to review (silent skip - no output)
- QUEEN.md does not exist
