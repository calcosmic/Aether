<!-- Generated from .aether/commands/seal.yaml - DO NOT EDIT DIRECTLY -->
---
name: ant:seal
description: "🏺🐜🏺 Seal the colony with Crowned Anthill milestone"
---

### Step -1: Normalize Arguments

Run: `normalized_args=$(bash .aether/aether-utils.sh normalize-args "$@")`

This ensures arguments work correctly in both Claude Code and OpenCode. Use `$normalized_args` throughout this command.

You are the **Queen**. Seal the colony when all work is complete.

## Instructions

Parse `$normalized_args`:
- If contains `--no-visual`: set `visual_mode = false` (visual is ON by default)
- Otherwise: set `visual_mode = true`

### Step 1: Read State

Read `.aether/data/COLONY_STATE.json`.

If file missing or `goal: null`:
```
No colony initialized. Run /ant:init first.
```
Stop here.

### Step 2: Validate Colony Is Complete

Extract: `goal`, `current_phase`, `plan.phases`, `milestone`, `state`.

**Precondition 1: All phases must be completed**

Check if all phases in `plan.phases` have `status: "completed"`:
```
all_completed = all(phase.status == "completed" for phase in plan.phases)
```

If NOT all completed:
```
Cannot archive colony with incomplete phases.

Completed phases: X of Y
Remaining: {list of incomplete phase names}

Run /ant:continue to complete remaining phases first.
```
Stop here.

**Precondition 2: State must not be EXECUTING**

If `state == "EXECUTING"`:
```
Colony is still executing. Run /ant:continue to reconcile first.
```
Stop here.

### Step 3: Check Milestone Eligibility

The full milestone progression is:
- **First Mound** — Phase 1 complete (first runnable)
- **Open Chambers** — Feature work underway (2+ phases complete)
- **Brood Stable** — Tests consistently green
- **Ventilated Nest** — Perf/latency acceptable (build + lint clean)
- **Sealed Chambers** — All phases complete (interfaces frozen)
- **Crowned Anthill** — Release-ready (user confirms via /ant:seal)

**If current milestone is "Crowned Anthill":**
```
Colony is already at Crowned Anthill milestone.
No further archiving needed.

Use /ant:status to view colony state.
```
Stop here.

**If current milestone is "Sealed Chambers":**
- Proceed to Step 4 (will upgrade to Crowned Anthill)

**If current milestone is "First Mound", "Open Chambers", "Brood Stable", "Ventilated Nest", or any intermediate milestone:**
- Since all phases are complete, the colony qualifies for both Sealed Chambers and Crowned Anthill
- The current logic allows proceeding to Step 4 (seal as Crowned Anthill)
- If user wants to explicitly achieve Sealed Chambers first, they can manually update milestone via COLONY_STATE.json

**If milestone is unrecognized (not in the 6 known stages):**
```
Unknown milestone: {milestone}

The milestone "{milestone}" is not recognized.
Known milestones: First Mound, Open Chambers, Brood Stable, Ventilated Nest, Sealed Chambers, Crowned Anthill

Run /ant:status to check colony state.
```
Stop here.

### Step 4: Archive Colony State

Create archive directory:
```
archive_dir=".aether/data/archive/session_$(date -u +%s)_archive"
mkdir -p "$archive_dir"
```

Copy the following files to the archive directory:
1. `.aether/data/COLONY_STATE.json` → `$archive_dir/COLONY_STATE.json`
2. `.aether/data/activity.log` → `$archive_dir/activity.log`
3. `.aether/data/spawn-tree.txt` → `$archive_dir/spawn-tree.txt`
4. `.aether/data/flags.json` → `$archive_dir/flags.json` (if exists)
5. `.aether/data/constraints.json` → `$archive_dir/constraints.json` (if exists)

Create archive manifest file `$archive_dir/manifest.json`:
```json
{
  "archived_at": "<ISO-8601 timestamp>",
  "goal": "<colony goal>",
  "total_phases": <number>,
  "milestone": "Crowned Anthill",
  "files": [
    "COLONY_STATE.json",
    "activity.log",
    "spawn-tree.txt",
    "flags.json",
    "constraints.json"
  ]
}
```

### Step 4.5: Increment Colony Version

Before writing the Crowned Anthill milestone, increment `colony_version` in COLONY_STATE.json.

```bash
# Read current colony_version (default to 0 for backward compat with older colonies)
current_colony_version=$(jq -r '.colony_version // 0' .aether/data/COLONY_STATE.json 2>/dev/null || echo 0)
# Guard against non-integer values (floats, strings)
[[ "$current_colony_version" =~ ^[0-9]+$ ]] || current_colony_version=0
new_colony_version=$(( current_colony_version + 1 ))

# Write incremented value back via state-mutate (atomic, locked, validated)
bash .aether/aether-utils.sh state-mutate --argjson v "$new_colony_version" '.colony_version = $v' || {
  echo "Warning: state-mutate failed — colony_version defaults to 1, state file unchanged"
  new_colony_version=1
}
```

Use `new_colony_version` as `{colony_version}` throughout the rest of the seal ceremony (e.g., display as "Crowned Anthill v{colony_version}").

**Error handling:** If jq fails or produces empty/short output, COLONY_STATE.json is NOT overwritten and `new_colony_version` defaults to 1. Never let version increment failures block the seal.

### Step 5: Update Milestone to Crowned Anthill

Update COLONY_STATE.json:
1. Set `milestone` to `"Crowned Anthill"`
2. Set `milestone_updated_at` to current ISO-8601 timestamp
3. Append event: `"<timestamp>|milestone_reached|archive|Achieved Crowned Anthill milestone - colony archived"`

### Step 5.1: Update Changelog

**MANDATORY: Record the seal in the project changelog. This step is never skipped.**

If no `CHANGELOG.md` exists, `changelog-append` creates one automatically.

Build a summary of what the colony accomplished across all phases:
- Collect completed phase names from COLONY_STATE.json
- Summarize the goal and key outcomes in one line

```bash
bash .aether/aether-utils.sh changelog-append \
  "$(date +%Y-%m-%d)" \
  "seal-crowned-anthill" \
  "00" \
  "{key_files_csv}" \
  "Colony sealed at Crowned Anthill;{goal}" \
  "{phases_completed} phases completed;Colony wisdom promoted to QUEEN.md" \
  ""
```

- `{key_files_csv}` — list the most significant files created or modified across the colony's lifetime (derive from phase plans or git log)
- `{goal}` — the colony goal from COLONY_STATE.json

**Error handling:** If `changelog-append` fails, log to midden and continue — changelog failure never blocks sealing.

### Step 5.2: Update Registry (Silent)

Mark the colony as inactive in the global registry. This is silent on failure — registry is not required for the colony to work.

Run using the Bash tool (ignore errors):
```bash
bash .aether/aether-utils.sh registry-add "$(pwd)" "$(jq -r '.version // "unknown"' ~/.aether/version.json 2>/dev/null || echo 'unknown')" --active false 2>/dev/null || true
```

If the command fails, proceed silently. This is optional bookkeeping.

### Step 5.25: Hive Promotion (NON-BLOCKING)

After wisdom promotion, promote abstracted instincts to the cross-colony hive.

**Extract high-confidence instincts for hive promotion:**
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
  result=$(bash .aether/aether-utils.sh "${promote_args[@]}" 2>/dev/null || echo '{}')
  was_promoted=$(echo "$result" | jq -r '.result.action // "skipped"' 2>/dev/null || echo "skipped")

  if [[ "$was_promoted" == "promoted" || "$was_promoted" == "merged" ]]; then
    hive_promoted_count=$((hive_promoted_count + 1))
  fi
done

if [[ "$hive_promoted_count" -gt 0 ]]; then
  echo "Hive promotion: $hive_promoted_count instinct(s) promoted to cross-colony hive"
fi
```

**Continue to Step 5.5 (non-blocking):**
Proceed to Step 5.5 regardless of hive promotion results — hive promotion is strictly non-blocking.

### Step 5.5: Write Final Handoff

After archiving, write the final handoff documenting the completed colony:

Resolve the handoff template path:
  Check ~/.aether/system/templates/handoff.template.md first,
  then .aether/templates/handoff.template.md.

If no template found: output "Template missing: handoff.template.md. Run aether update to fix." and stop.

Read the template file. Fill all {{PLACEHOLDER}} values:
  - {{CHAMBER_NAME}} → archive directory name
  - {{GOAL}} → goal
  - {{PHASES_COMPLETED}} → total_phases (OpenCode seal archives completed colonies)
  - {{TOTAL_PHASES}} → total_phases
  - {{MILESTONE}} → "Crowned Anthill"
  - {{ENTOMB_TIMESTAMP}} → seal timestamp

Remove the HTML comment lines at the top of the template.
Write the result to .aether/HANDOFF.md using the Write tool.

This handoff serves as the final record of the completed colony.

### Step 5.75: Export XML Archive (best-effort)

Export colony data as a combined XML archive and a standalone pheromones.xml. Both are best-effort — seal proceeds even if XML export fails.

```bash
# Check if xmllint is available
if command -v xmllint >/dev/null 2>&1; then
  xml_result=$(bash .aether/aether-utils.sh colony-archive-xml ".aether/exchange/colony-archive.xml" 2>&1)
  xml_ok=$(echo "$xml_result" | jq -r '.ok // false' 2>/dev/null)
  if [[ "$xml_ok" == "true" ]]; then
    xml_pheromone_count=$(echo "$xml_result" | jq -r '.result.pheromone_count // 0' 2>/dev/null)
    xml_export_line="XML Archive: colony-archive.xml (${xml_pheromone_count} active signals)"
  else
    xml_export_line="XML Archive: export failed (non-blocking)"
  fi

  # Also export standalone pheromones.xml for cross-colony sharing
  pher_result=$(bash .aether/aether-utils.sh pheromone-export-xml ".aether/exchange/pheromones.xml" 2>&1)
  pher_ok=$(echo "$pher_result" | jq -r '.ok // false' 2>/dev/null)
  if [[ "$pher_ok" == "true" ]]; then
    pher_signal_count=$(jq '[.signals[] | select(.active != false)] | length' .aether/data/pheromones.json 2>/dev/null || echo "0")
    pher_export_line="Signal Export: pheromones.xml (${pher_signal_count} signals, importable by other colonies)"
  else
    pher_export_line="Signal Export: failed (non-blocking)"
  fi

  # Export standalone queen-wisdom.xml for cross-colony wisdom sharing
  wisdom_result=$(bash .aether/aether-utils.sh wisdom-export-xml ".aether/exchange/queen-wisdom.xml" 2>&1)
  wisdom_ok=$(echo "$wisdom_result" | jq -r '.ok // false' 2>/dev/null)
  if [[ "$wisdom_ok" == "true" ]]; then
    wisdom_count=$(echo "$wisdom_result" | jq -r '.result.entries // 0' 2>/dev/null)
    wisdom_export_line="Wisdom Export: queen-wisdom.xml (${wisdom_count} entries)"
  else
    wisdom_export_line="Wisdom Export: failed (non-blocking)"
  fi

  # Export standalone colony-registry.xml for lineage tracking
  registry_result=$(bash .aether/aether-utils.sh registry-export-xml ".aether/exchange/colony-registry.xml" 2>&1)
  registry_ok=$(echo "$registry_result" | jq -r '.ok // false' 2>/dev/null)
  if [[ "$registry_ok" == "true" ]]; then
    registry_count=$(echo "$registry_result" | jq -r '.result.colonies // 0' 2>/dev/null)
    registry_export_line="Registry Export: colony-registry.xml (${registry_count} colonies)"
  else
    registry_export_line="Registry Export: failed (non-blocking)"
  fi
else
  xml_export_line="XML Archive: skipped (xmllint not available)"
  pher_export_line="Signal Export: skipped (xmllint not available)"
  wisdom_export_line="Wisdom Export: skipped (xmllint not available)"
  registry_export_line="Registry Export: skipped (xmllint not available)"
fi
```

### Step 6: Display Result

Output:
```
🏺 ════════════════════════════════════════════════════
   C R O W N E D   A N T H I L L   v{colony_version}
══════════════════════════════════════════════════ 🏺

✅ Colony archived successfully!

👑 Goal: {goal (truncated to 60 chars)}
📍 Phases: {total_phases} completed
🏆 Milestone: Crowned Anthill v{colony_version}

📦 Archive Location: {archive_dir}
   - COLONY_STATE.json
   - activity.log
   - spawn-tree.txt
   - flags.json (if existed)
   - constraints.json (if existed)
{xml_export_line}
{pher_export_line}
{wisdom_export_line}
{registry_export_line}

🐜 The colony has reached its final form.
   The anthill stands crowned and sealed.
   History is preserved. The colony rests.

💾 State persisted — safe to /clear

🐜 What would you like to do next?
   1. /ant:lay-eggs "<new goal>"  — Start a new colony
   2. /ant:tunnels                — Browse archived colonies
   3. /clear                      — Clear context and continue

Use AskUserQuestion with these three options.

If option 1 selected: proceed to run /ant:lay-eggs flow
If option 2 selected: run /ant:tunnels
If option 3 selected: display "Run /ant:lay-eggs to begin anew after clearing"
```

### Step 6.5: Commit Suggestion (Non-blocking)

After the ceremony, offer to commit all colony work.

**Gate check — skip silently if any fail:**
1. Not a git repo: `git rev-parse --git-dir 2>/dev/null` fails -> skip
2. Clean working tree: `git status --porcelain 2>/dev/null` is empty -> skip to Step 6.6

**If uncommitted changes exist:**

Generate a seal commit message:
```bash
seal_commit=$(bash .aether/aether-utils.sh generate-commit-message seal "$phases_completed" "$goal" "$colony_age_days" 2>/dev/null)
seal_message=$(echo "$seal_commit" | jq -r '.result.message // "aether-seal: colony sealed"')
seal_body=$(echo "$seal_commit" | jq -r '.result.body // ""')
```

Display the suggestion:
```
Commit suggestion:
  $seal_message
  $seal_body
```

Prompt with AskUserQuestion (3 options):
1. **Commit with this message** — Run `git add -A && git commit -m "$seal_message" -m "$seal_body"`
2. **Edit message** — Ask user for their preferred message, then commit with that
3. **Skip** — Do not commit

If the user chooses option 1 or 2 and the commit succeeds, set `seal_committed = true`.
If the commit fails or user skips, set `seal_committed = false`.

**Error handling:** If `generate-commit-message` fails, fall back to message `"aether-seal: colony sealed"`. Never let commit suggestion failures stop the seal flow.

### Step 6.6: Push Suggestion (Non-blocking)

Only show if a commit was just made in Step 6.5 (`seal_committed == true`) OR if there are unpushed commits.

**Gate check — skip silently if any fail:**
1. Not a git repo -> skip
2. No remote configured: `git remote -v 2>/dev/null` is empty -> skip

**Check for unpushed commits:**
```bash
unpushed=$(git log --oneline @{u}..HEAD 2>/dev/null | wc -l | tr -d ' ')
```
If `unpushed == 0` and `seal_committed == false` -> skip

**If there are commits to push:**

Detect current branch and upstream status:
```bash
current_branch=$(git branch --show-current)
has_upstream=$(git rev-parse --abbrev-ref @{u} 2>/dev/null && echo "yes" || echo "no")
```

Display: `Push {unpushed} commit(s) to remote? Branch: {current_branch}`

Prompt with AskUserQuestion (2 options):
1. **Push now** — If upstream exists: `git push`. If no upstream: `git push -u origin $current_branch`
2. **I'll push later** — Skip

Display result on success or skip.

**Error handling:** If push fails, display the error but do not stop the seal flow.

**Safety:** Never auto-push. Always require explicit user approval.

### Edge Cases

**If milestone is already "Sealed Chambers" but phases are complete:**
- Proceed with archiving and upgrade to Crowned Anthill

**If any archive files are missing:**
- Archive what exists, note in manifest which files were missing

**If archive directory already exists:**
- Append timestamp to make unique: `session_<ts>_archive_<random>`
