### Step 2.0.7: Generate Review Context (pr-context)

After verification passes, generate structured colony context for the review and learning steps that follow.

Run using the Bash tool with description "Generating review context...":
```bash
REVIEW_CONTEXT=$(bash "$AETHER_UTILS" pr-context 2>/dev/null) || true
```

- If REVIEW_CONTEXT is empty or fails, continue without it (non-blocking)
- This provides structured colony context for review/learning steps
- Output is available for subsequent steps but does not gate them

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

2.4. **Deterministic fallback extraction (when AI learnings are empty):**

   If the builder produced no learnings (empty or missing `learning.patterns_observed` in synthesis JSON),
   fire the git-diff-based fallback to extract learnings deterministically.

   ```bash
   # Check if learnings were extracted from synthesis
   patterns_count=$(jq '[.memory.phase_learnings[-1].learnings // []] | add | length' .aether/data/COLONY_STATE.json 2>/dev/null || echo "0")
   fallback_count=0

   if [[ "$patterns_count" -eq 0 ]]; then
     # Builder skipped learning output -- fire deterministic fallback
     fallback_result=$(aether learning-extract-fallback 2>/dev/null || echo '{"learnings":[],"count":0}')
     fallback_count=$(echo "$fallback_result" | jq '.result.count // 0')
   fi

   # Echo for cross-stage capture (same pattern as hive_promoted_count)
   echo "fallback_count=$fallback_count"
   ```

   This is NON-BLOCKING -- if learning-extract-fallback fails, fallback_count stays 0 and no learnings are lost (they just aren't recorded this time).

   **Cross-stage variable passing:** The `fallback_count` value is output via echo for capture in continue-finalize.md's wisdom summary.

2.5. **Capture learnings through memory pipeline:**

   For each learning extracted, run the memory pipeline (observation + auto-pheromone + auto-promotion check).

   Run using the Bash tool with description "Recording learning observations...":
   ```bash
   colony_name=$(aether colony-name 2>/dev/null | jq -r '.result.name // ""')
   [[ -z "$colony_name" ]] && colony_name="unknown"

   # Get learnings from the current phase
   current_phase_learnings=$(jq -r --argjson phase "$current_phase" '.memory.phase_learnings[] | select(.phase == $phase)' .aether/data/COLONY_STATE.json 2>/dev/null || echo "")

   if [[ -n "$current_phase_learnings" ]]; then
     echo "$current_phase_learnings" | jq -r '.learnings[]?.claim // empty' 2>/dev/null | while read -r claim; do
       if [[ -n "$claim" ]]; then
         aether memory-capture "learning" "$claim" "pattern" "worker:continue" 2>/dev/null || true
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

3. **Extract instincts from phase patterns:**

   Review the completed phase for repeating patterns. For each pattern observed:

   Run using the Bash tool with description "Creating instinct from pattern...":
   ```bash
   aether instinct-create \
     --trigger "<when this situation arises>" \
     --action "<what worked or should be done>" \
     --confidence <0.7-0.9 based on evidence strength> \
     --domain "<testing|architecture|code-style|debugging|workflow>" \
     --source "phase-{id}" \
     --evidence "<specific observation>" 2>/dev/null || true
   ```

   Confidence guidelines:
   - Base 0.7 for observation_count=1, increasing by 0.05 per additional observation, cap 0.9
   - 0.8: error_resolution (fixed a recurring problem, or observation_count=3)
   - 0.9: user_feedback (explicit user guidance, or observation_count=5+)
   - When observation count is available from learning-observations.json, use formula: min(0.7 + (count-1)*0.05, 0.9)

   If pattern matches existing instinct, confidence will be boosted automatically.
   Cap: max 30 instincts enforced by `instinct-create` (lowest confidence evicted).

3a. **Extract instincts from recurring error patterns (midden):**

   Query the midden for recent failures and create instincts from recurring patterns:

   Run using the Bash tool with description "Checking midden for error patterns...":
   ```bash
   midden_result=$(aether midden-recent-failures 10 2>/dev/null || echo '{"count":0,"failures":[]}')
   midden_count=$(echo "$midden_result" | jq '.count // 0')
   ```

   If `midden_count` > 0, review the failure entries for recurring patterns (same category or similar message appearing 2+ times). For each recurring error pattern found:

   Run using the Bash tool with description "Creating instinct from error pattern...":
   ```bash
   aether instinct-create \
     --trigger "<when this error condition arises>" \
     --action "<how to avoid or handle this error>" \
     --confidence 0.8 \
     --domain "<testing|architecture|debugging>" \
     --source "midden-phase-{id}" \
     --evidence "<failure message and recurrence count>" 2>/dev/null || true
   ```

   Error pattern confidence is 0.8 (higher than success patterns) because recurring failures are strong negative signals.
   If no recurring patterns found, skip silently.

3b. **Extract instincts from success patterns:**

   Review the completed phase for approaches that succeeded on the first attempt or produced notably clean results. For each success pattern:

   Run using the Bash tool with description "Creating instinct from success pattern...":
   ```bash
   aether instinct-create \
     --trigger "<when this type of task arises>" \
     --action "<the approach that worked well>" \
     --confidence 0.7 \  # Base value; increase if observation_count > 1 per formula
     --domain "<testing|architecture|code-style|workflow>" \
     --source "success-phase-{id}" \
     --evidence "<what succeeded and why>" 2>/dev/null || true
   ```

   Success pattern confidence is 0.7 (base; calibrate with observation count if available). Only create success instincts for genuinely noteworthy approaches, not routine completions.
   Cap: limit to 2 success instincts per phase to avoid noise.

3c. **Promote high-confidence instincts to QUEEN.md (QUEEN-02):**

   After instinct creation (Steps 3/3a/3b), sweep all instincts for confidence >= 0.8
   and promote to the QUEEN.md Instincts section. This runs every /ant:continue to catch
   newly boosted instincts.

   Run using the Bash tool with description "Promoting high-confidence instincts to QUEEN.md...":
   ```bash
   instinct_data=$(jq -r '.memory.instincts // []' .aether/data/COLONY_STATE.json 2>/dev/null || echo '[]')
   promoted_instinct_count=0

   for encoded in $(echo "$instinct_data" | jq -r '.[] | select(.confidence >= 0.8) | @base64'); do
       trigger=$(echo "$encoded" | base64 -d | jq -r '.trigger')
       action=$(echo "$encoded" | base64 -d | jq -r '.action')
       confidence=$(echo "$encoded" | base64 -d | jq -r '.confidence')
       domain=$(echo "$encoded" | base64 -d | jq -r '.domain // "workflow"')

       # queen-promote-instinct handles dedup internally (skips if already in QUEEN.md)
       result=$(aether queen-promote-instinct \
           "$trigger" "$action" "$confidence" "$domain" 2>/dev/null || echo '{"ok":false}')

       was_promoted=$(echo "$result" | jq -r '.result.promoted // false' 2>/dev/null || echo "false")
       if [[ "$was_promoted" == "true" ]]; then
           promoted_instinct_count=$((promoted_instinct_count + 1))
       fi
   done

   if [[ $promoted_instinct_count -gt 0 ]]; then
       echo "Promoted $promoted_instinct_count instinct(s) to QUEEN.md"
   fi
   ```

   This step is NON-BLOCKING -- if queen-promote-instinct fails for any entry, log and continue.
   The dedup check inside queen-promote-instinct ensures idempotency (safe to run repeatedly).

3d. **Hive Promotion (NON-BLOCKING):**

   After QUEEN.md promotion, promote abstracted instincts to the cross-colony hive.

   Run using the Bash tool with description "Promoting high-confidence instincts to hive...":
   ```bash
   # Get instincts with confidence >= 0.8
   high_conf_instincts=$(jq -r '.memory.instincts[] | select(.confidence >= 0.8) | @base64' .aether/data/COLONY_STATE.json 2>/dev/null || echo "")

   # Derive source repo name from current directory
   source_repo="$(pwd)"

   # Read domain tags from registry (NOT from instinct.domain which is a category, not a repo domain)
   repo_domain_tags=$(jq -r --arg repo "$(pwd)" \
     '[.repos[] | select(.path == $repo) | .domain_tags // []] | .[0] // [] | join(",")' \
     "$HOME/.aether/registry.json" 2>/dev/null || echo "")

   hive_promoted_count=0
   hive_errors=0
   for encoded in $high_conf_instincts; do
     [[ -z "$encoded" ]] && continue

     # Extract trigger and action fields from the instinct object
     trigger=$(echo "$encoded" | base64 -d | jq -r '.trigger // empty')
     action=$(echo "$encoded" | base64 -d | jq -r '.action // empty')
     confidence=$(echo "$encoded" | base64 -d | jq -r '.confidence // 0.7')

     [[ -z "$trigger" || -z "$action" ]] && continue

     # Strip leading "When " or "when " from trigger to avoid "When When..." stutter
     trigger_clean=$(echo "$trigger" | sed 's/^[Ww]hen //')

     # Build the promotion text in "When {trigger}: {action}" format
     promote_text="When ${trigger_clean}: ${action}"

     # Build hive-promote args with --text and --source-repo (required)
     promote_args=(hive-promote --text "$promote_text" --source-repo "$source_repo" --confidence "$confidence")
     [[ -n "$repo_domain_tags" ]] && promote_args+=(--domain "$repo_domain_tags")

     # Call hive-promote which orchestrates abstract + store
     result=$(aether "${promote_args[@]}" 2>/dev/null || echo '{}')
     was_promoted=$(echo "$result" | jq -r '.result.action // "skipped"' 2>/dev/null || echo "skipped")

     if [[ "$was_promoted" == "promoted" || "$was_promoted" == "merged" ]]; then
       hive_promoted_count=$((hive_promoted_count + 1))
     else
       hive_errors=$((hive_errors + 1))
     fi
   done

   if [[ "$hive_promoted_count" -gt 0 ]]; then
     echo "Hive promotion: $hive_promoted_count instinct(s) promoted to cross-colony hive"
   fi

   # Store results for cross-stage summary line
   echo "hive_promoted_count=$hive_promoted_count"
   hive_error=$([[ $hive_errors -gt 0 ]] && echo 'true' || echo 'false')
   echo "hive_error=$hive_error"
   ```

   This step promotes high-confidence instincts to the cross-colony hive. It runs every /ant:continue and is idempotent -- hive-store deduplicates via SHA-256, so re-promoting existing instincts is safe.

   NON-BLOCKING: hive promotion failures never stop the continue flow.

   **Cross-stage variable passing:** The `hive_promoted_count` and `hive_error` values are output via echo for cross-stage capture. **IMPORTANT:** When executing continue-finalize.md after this step, Claude must capture these echoed values from the advance output and pass them as variables when running the finalize wisdom summary code. Shell variables do not persist between Bash tool invocations, so this explicit capture-and-forward step is required.

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

Write the updated state through targeted `state-mutate` calls. Each call acquires a lock, creates a backup, applies the jq expression, validates, and writes atomically. Do NOT use the Write tool or `state-write` with full JSON content — always use `state-mutate` to avoid stale-context corruption.

Run using the Bash tool with description "Advancing colony state...":
```bash
# Mark current phase completed
aether state-mutate --argjson pid "$current_phase" \
  '.plan.phases |= map(if .id == $pid then .status = "completed" else . end)'

# Append learning (if any — skip if no learnings were extracted in Step 2)
# aether state-mutate --argjson learning "$learning_json" \
#   '.memory.phase_learnings += [$learning]'

# Advance to next phase
aether state-mutate \
  --argjson pid "$current_phase" \
  --argjson next "$next_phase" \
  --arg timestamp "$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
  '.current_phase = $next | .state = "READY" | .build_started_at = null | .events += [$timestamp + "|phase_advanced|continue|Completed Phase " + ($pid|tostring) + ", advancing to Phase " + ($next|tostring)]'
```

Run using the Bash tool with description "Enforcing memory caps...":
```bash
# Cap enforcement — keep arrays bounded
aether state-mutate \
  '.memory.phase_learnings = (.memory.phase_learnings[-20:]) | .memory.decisions = (.memory.decisions[-30:]) | .memory.instincts = (.memory.instincts | sort_by(.confidence) | .[-30:]) | .events = (.events[-100:])'
```

Validate the state file:
Run using the Bash tool with description "Validating colony state...": `aether validate-state colony`

### Step 2.0.4: Worktree Merge-Back (DEPRECATED, NON-BLOCKING)

After state update, check for any completed worktree branches from the build wave and merge them back to the target branch.

> **DEPRECATED**: The `worktree-merge` command has been deprecated and will be removed
> in a future version. It returns `ok:true` with `deprecated:true` for backward
> compatibility. This step now always skips gracefully. Use `git merge` directly instead.

Run using the Bash tool with description "Checking for worktree branches to merge...":
```bash
# Check if worktree-merge is deprecated (returns ok:true with deprecated:true)
merge_check=$(aether worktree-merge --branch "check" 2>/dev/null || echo '{"ok":false}')
merge_deprecated=$(echo "$merge_check" | jq -r '.result.deprecated // false')

if [[ "$merge_deprecated" == "true" ]]; then
    # Command is deprecated — skip worktree merge-back entirely.
    # Set empty variables so downstream steps (2.0.5, 2.0.6) skip gracefully.
    last_merged_branch=""
    last_merge_sha=""
    merged_count=0
else
    # Legacy path: for older aether versions that still have real worktree-merge
    branches=$(git -C "$AETHER_ROOT" worktree list --porcelain 2>/dev/null \
        | grep "worktree-agent-\|worktree-" \
        | awk '{print $NF}' || echo "")

    last_merged_branch=""
    last_merge_sha=""
    merged_count=0

    for branch in $branches; do
        [[ -z "$branch" ]] && continue
        result=$(aether worktree-merge --branch "$branch" 2>/dev/null || echo '{"ok":false}')
        ok=$(echo "$result" | jq -r '.ok // false')
        deprecated=$(echo "$result" | jq -r '.result.deprecated // false')
        if [[ "$ok" == "true" && "$deprecated" != "true" ]]; then
            last_merged_branch="$branch"
            last_merge_sha=$(echo "$result" | jq -r '.result.sha // ""')
            merged_count=$((merged_count + 1))
        fi
    done

    if [[ "$merged_count" -gt 0 ]]; then
        echo "Merged $merged_count worktree branch(es). Last: $last_merged_branch ($last_merge_sha)"
    fi
fi
```

This step sets `$last_merged_branch` and `$last_merge_sha` which activates the existing dead code paths in Steps 2.0.5 and 2.0.6.

### Step 2.0.5: Pheromone Merge-Back (SILENT, NON-BLOCKING)

If a `pheromone-branch-export.json` exists in `.aether/exchange/` (written by seal ceremony on a PR branch and merged to main), run merge-back to collect branch-discovered signals into main's pheromone store. This entire step is silent and non-blocking -- continue proceeds even if merge-back fails.

Run using the Bash tool with description "Checking for pheromone merge-back file...":
```bash
# Check if a branch pheromone export was merged into main
export_file=".aether/exchange/pheromone-branch-export.json"
if [[ -f "$export_file" ]]; then
  merge_result=$(aether pheromone-merge-back --export-file "$export_file" 2>/dev/null || echo '{"ok":false}')
  merge_ok=$(echo "$merge_result" | jq -r '.ok // false' 2>/dev/null)
  if [[ "$merge_ok" == "true" ]]; then
    new_count=$(echo "$merge_result" | jq -r '.result.new_signals_written // 0' 2>/dev/null)
    skipped=$(echo "$merge_result" | jq -r '.result.skipped_count // 0' 2>/dev/null)
    conflicts=$(echo "$merge_result" | jq -r '.result.conflicts_resolved // [] | length' 2>/dev/null)
    if [[ "$new_count" -gt 0 || "$conflicts" -gt 0 ]]; then
      echo "Pheromone merge-back: $new_count new, $conflicts conflicts resolved, $skipped skipped (from merged branch)"
    fi
    # Clean up export file after successful merge
    rm -f "$export_file" 2>/dev/null || true
  else
    echo "Pheromone merge-back: failed (non-blocking)"
  fi
fi
```

### Step 2.0.6: Midden Collection (NON-BLOCKING)

After pheromone merge-back, collect failure records from any recently merged branch worktrees. This step is silent and non-blocking -- continue proceeds even if collection fails.

**Per D-04: Wire midden-collect into /ant:continue flow.**

If the colony uses a PR-based workflow and a merge just happened, attempt to collect the branch's midden entries:

Run using the Bash tool with description "Collecting branch midden entries...":
```bash
# Check if there's a recently merged branch to collect from
# The merge info comes from git log or COLONY_STATE context
last_merge_branch="${last_merged_branch:-}"
last_merge_sha="${last_merge_sha:-}"

if [[ -n "$last_merge_branch" && -n "$last_merge_sha" ]]; then
  collect_result=$(aether midden-collect \
    --branch "$last_merge_branch" --merge-sha "$last_merge_sha" \
    2>/dev/null || echo '{"ok":false}')
  collect_ok=$(echo "$collect_result" | jq -r '.ok // false' 2>/dev/null)
  if [[ "$collect_ok" == "true" ]]; then
    collect_status=$(echo "$collect_result" | jq -r '.result.status // "unknown"' 2>/dev/null)
    if [[ "$collect_status" == "collected" ]]; then
      new_entries=$(echo "$collect_result" | jq -r '.result.entries_collected // 0' 2>/dev/null)
      echo "Midden: collected $new_entries failure entries from branch $last_merge_branch"
    fi
  fi
fi
```

This step is NON-BLOCKING -- continue proceeds regardless of collection outcome. If `last_merge_branch` and `last_merge_sha` are not set (e.g., no recent merge), this step is silently skipped.

### Step 2.0.7: Cross-PR Midden Analysis (NON-BLOCKING)

After midden collection (Step 2.0.6), run cross-PR analysis to detect systemic failure patterns across multiple merged branches. Auto-emits REDIRECT pheromones to hub if systemic patterns found.

**Per D-05: Wire midden-cross-pr-analysis into /ant:continue flow.**

Run using the Bash tool with description "Running cross-PR midden analysis...":
```bash
analysis_result=$(aether midden-cross-pr-analysis --window 14 \
  2>/dev/null || echo '{"ok":false}')
analysis_ok=$(echo "$analysis_result" | jq -r '.ok // false' 2>/dev/null)
if [[ "$analysis_ok" == "true" ]]; then
  systemic=$(echo "$analysis_result" | jq -r '.result.systemic_categories // [] | length' 2>/dev/null || echo "0")
  if [[ "$systemic" -gt 0 ]]; then
    categories=$(echo "$analysis_result" | jq -r '.result.systemic_categories | join(", ")' 2>/dev/null)
    echo "Midden: cross-PR systemic pattern detected in: $categories (REDIRECT emitted)"
  fi
fi
```

This step is NON-BLOCKING -- advance proceeds regardless of analysis outcome.

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

aether pheromone-write FEEDBACK "$phase_feedback" \
  --strength 0.6 \
  --source "worker:continue" \
  --reason "Auto-emitted on phase advance: captures what worked and what was learned" \
  --ttl "30d" 2>/dev/null || true
```

The strength is 0.6 (auto-emitted = lower than user-emitted 0.7). Source is "worker:continue" to distinguish from user-emitted feedback. TTL is 30d so it survives phase transitions and can guide subsequent work.

#### 2.1b: Auto-emit FEEDBACK for phase decisions (PHER-01)

Extract recent decisions from CONTEXT.md "Recent Decisions" table and emit FEEDBACK pheromones for each. This ensures key decisions propagate as signals to guide future phases.

```bash
decisions=$(awk '
  /^## .*Recent Decisions/ { in_section=1; next }
  in_section && /^\| Date / { next }
  in_section && /^\|[-]+/ { next }
  in_section && /^---/ { exit }
  in_section && /^\| [0-9]{4}-[0-9]{2}/ {
    split($0, fields, "|")
    decision = fields[3]
    gsub(/^[[:space:]]+|[[:space:]]+$/, "", decision)
    if (decision != "") print decision
  }
' .aether/CONTEXT.md 2>/dev/null || echo "")

if [[ -n "$decisions" ]]; then
  emit_count=0
  while IFS= read -r dec && [[ $emit_count -lt 3 ]]; do
    [[ -z "$dec" ]] && continue
    # Deduplication: check if auto:decision or system:decision pheromone with this text already exists
    existing=$(jq -r --arg text "$dec" '
      [.signals[] | select(.active == true and (.source == "auto:decision" or .source == "system:decision") and (.content.text | contains($text)))] | length
    ' .aether/data/pheromones.json 2>/dev/null || echo "0")
    if [[ "$existing" == "0" ]]; then
      aether pheromone-write FEEDBACK \
        "[decision] $dec" \
        --strength 0.6 \
        --source "auto:decision" \
        --reason "Auto-emitted from phase decision during continue" \
        --ttl "30d" 2>/dev/null || true
      emit_count=$((emit_count + 1))
    fi
  done <<< "$decisions"
fi
```

Strength is 0.6 (auto-emitted = lower than user-emitted). Source is `"auto:decision"` to distinguish from manual pheromones. Cap: max 3 decision pheromones per continue run. Both `context-update decision` and Step 2.1b now use the same format (`[decision] ...`, source `auto:decision`, strength 0.6), so the dedup `contains()` check reliably catches signals emitted by either path. The dedup query also checks `system:decision` for backward compatibility with any pre-existing signals from before the format alignment.

#### 2.1c: Auto-emit REDIRECT for midden error patterns (PHER-02)

Query the actual failure store (`midden.json`) for recurring error categories. Categories with 3+ occurrences indicate persistent issues that should steer workers away from known failure modes.

```bash
midden_result=$(aether midden-recent-failures 50 2>/dev/null || echo '{"count":0,"failures":[]}')
midden_count=$(echo "$midden_result" | jq '.count // 0')

if [[ "$midden_count" -gt 0 ]]; then
  # Group by category, find categories with 3+ occurrences
  recurring_categories=$(echo "$midden_result" | jq -r '
    [.failures[] | .category]
    | group_by(.)
    | map(select(length >= 3))
    | map({category: .[0], count: length})
    | .[]
    | @base64
  ' 2>/dev/null || echo "")

  emit_count=0
  for encoded in $recurring_categories; do
    [[ $emit_count -ge 3 ]] && break
    [[ -z "$encoded" ]] && continue
    category=$(echo "$encoded" | base64 -d | jq -r '.category')
    count=$(echo "$encoded" | base64 -d | jq -r '.count')

    # Deduplication check
    existing=$(jq -r --arg cat "$category" '
      [.signals[] | select(.active == true and .source == "auto:error" and (.content.text | contains($cat)))] | length
    ' .aether/data/pheromones.json 2>/dev/null || echo "0")

    if [[ "$existing" == "0" ]]; then
      aether pheromone-write REDIRECT \
        "[error-pattern] Category \"$category\" recurring ($count occurrences)" \
        --strength 0.7 \
        --source "auto:error" \
        --reason "Auto-emitted: midden error pattern recurred 3+ times" \
        --ttl "30d" 2>/dev/null || true
      emit_count=$((emit_count + 1))

      # Capture as resolution candidate for promotion tracking
      aether memory-capture \
        "resolution" \
        "Recurring error pattern: $category ($count occurrences)" \
        "pattern" \
        "worker:continue" 2>/dev/null || true
    fi
  done
fi
```

REDIRECT strength is 0.7 (higher than auto FEEDBACK 0.6 — anti-patterns produce stronger signals). Source is `"auto:error"`. Cap: max 3 error pattern pheromones per continue run. Uses `midden-recent-failures` subcommand (actual failure store) instead of `errors.flagged_patterns` (which may be empty). Threshold is 3+ occurrences for high confidence in recurrence.

#### 2.1d: Auto-emit FEEDBACK for recurring success criteria (PHER-03)

Compare success criteria text across all completed phases. Criteria appearing in 2+ completed phases indicate recurring quality patterns worth reinforcing as signals.

```bash
recurring_criteria=$(jq -r '
  [.plan.phases[]
   | select(.status == "completed")
   | .id as $phase_id
   | (
       (.success_criteria // [])[] ,
       (.tasks // [] | .[].success_criteria // [])[]
     )
   | {phase: $phase_id, text: (. | ascii_downcase | gsub("^\\s+|\\s+$"; ""))}
  ]
  | group_by(.text)
  | map(select(length >= 2))
  | map({text: .[0].text, phases: [.[].phase] | unique, count: length})
  | .[:2]
  | .[]
  | @base64
' .aether/data/COLONY_STATE.json 2>/dev/null || echo "")

for encoded in $recurring_criteria; do
  [[ -z "$encoded" ]] && continue
  text=$(echo "$encoded" | base64 -d | jq -r '.text')
  count=$(echo "$encoded" | base64 -d | jq -r '.count')
  phases=$(echo "$encoded" | base64 -d | jq -r '.phases | join(", ")')

  # Deduplication check
  existing=$(jq -r --arg text "$text" '
    [.signals[] | select(.active == true and .source == "auto:success" and (.content.text | ascii_downcase | contains($text)))] | length
  ' .aether/data/pheromones.json 2>/dev/null || echo "0")

  if [[ "$existing" == "0" ]]; then
    aether pheromone-write FEEDBACK \
      "[success-pattern] \"$text\" recurs across phases $phases" \
      --strength 0.6 \
      --source "auto:success" \
      --reason "Auto-emitted: success criteria pattern recurred across $count phases" \
      --ttl "30d" 2>/dev/null || true
  fi
done
```

Strength is 0.6 (auto-emitted). Source is `"auto:success"`. Cap: max 2 success criteria pheromones per continue run (enforced by `.[:2]` in the jq query). Extracts from both phase-level `.success_criteria` and task-level `.tasks[].success_criteria` across all completed phases. Normalizes text with `ascii_downcase` and whitespace trimming for reliable matching.

#### 2.1e: Expire phase_end signals and archive to midden

After auto-emission, expire all signals with `expires_at == "phase_end"`. The FEEDBACK from 2.1a uses a 30d TTL and is not affected by this step.

Run using the Bash tool with description "Maintaining pheromone memory...": `aether pheromone-expire --phase-end-only 2>/dev/null && aether eternal-init 2>/dev/null`

This is idempotent — runs every time continue fires but only creates the directory/file once.

### Step 2.1.5: Check for Promotion Proposals (PHER-EVOL-02)

After extracting learnings, check for observations that have met promotion thresholds and present the tick-to-approve UX.

**Check for --deferred flag:**

If `$ARGUMENTS` contains `--deferred`:
```bash
if [[ "$ARGUMENTS" == *"--deferred"* ]] && [[ -f .aether/data/learning-deferred.json ]]; then
  echo "📦 Reviewing deferred proposals..."
  aether learning-approve-proposals --deferred ${verbose:+--verbose}
fi
```

**Normal proposal flow (MEM-01: Silent skip if empty):**

1. **Check for proposals:**
   ```bash
   proposals=$(aether learning-check-promotion 2>/dev/null || echo '{"proposals":[]}')
   proposal_count=$(echo "$proposals" | jq '.proposals | length')
   ```

2. **If proposals exist, invoke the approval workflow:**

   Only show the approval UI when there are actual proposals to review:

   ```bash
   if [[ "$proposal_count" -gt 0 ]]; then
     verbose_flag=""
     [[ "$ARGUMENTS" == *"--verbose"* ]] && verbose_flag="--verbose"
     aether learning-approve-proposals $verbose_flag
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
