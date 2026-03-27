---
name: ant:seal
description: "🏺🐜🏺 Seal the colony with Crowned Anthill milestone"
---

You are the **Queen**. Seal the colony with a ceremony — no archiving.

## Instructions

Parse `$ARGUMENTS`:
- If contains `--no-visual`: set `visual_mode = false` (visual is ON by default)
- Otherwise: set `visual_mode = true`

<failure_modes>
### Crowned Anthill Write Failure
If writing the Crowned Anthill milestone document fails:
- Do not mark the colony as sealed in state
- Report the error -- sealing is incomplete
- Recovery: user can re-run /ant:seal after fixing the issue

### State Update Failure After Seal
If COLONY_STATE.json update fails after seal document is written:
- The seal document exists but state doesn't reflect it
- Report the inconsistency
- Options: (1) Retry state update only, (2) Manual state fix, (3) Re-run /ant:seal
</failure_modes>

<success_criteria>
Command is complete when:
- Crowned Anthill milestone document is written
- COLONY_STATE.json reflects sealed status
- All phase evidence is summarized in the seal document
- User sees confirmation of successful seal
</success_criteria>

<read_only>
Do not touch during seal:
- .aether/dreams/ (user notes)
- .aether/chambers/ (archived colonies -- seal does NOT archive)
- Source code files
- .env* files
- .claude/settings.json
</read_only>

### Step 0: Initialize Visual Mode (if enabled)

If `visual_mode` is true:
### Step 1: Read State

Read `.aether/data/COLONY_STATE.json`.

If file missing or `goal: null`:
```
No colony initialized. Run /ant:init first.
```
Stop here.

Extract: `goal`, `state`, `current_phase`, `plan.phases`, `milestone`, `version`, `initialized_at`.

### Step 2: Maturity Gate

Run `bash .aether/aether-utils.sh milestone-detect` to get `milestone`, `phases_completed`, `total_phases`.

**If milestone is already "Crowned Anthill":**
```
Colony already sealed at Crowned Anthill.
Run /ant:entomb to archive this colony to chambers.
```
Stop here.

**If state is "EXECUTING":**
```
Colony is still executing. Run /ant:continue first.
```
Stop here.

**If all phases complete** (phases_completed == total_phases, or milestone is "Sealed Chambers"):
- Set `incomplete_warning = ""` (no warning needed)
- Proceed to Step 3.

**If phases are incomplete** (any other milestone — First Mound, Open Chambers, Brood Stable, Ventilated Nest, etc.):
- Set `incomplete_warning = "WARNING: {phases_completed} of {total_phases} phases complete. Sealing now will mark incomplete work as the final state."`
- Proceed to Step 3 (warn but DO NOT block).

### Step 3: Confirmation

Display what will be sealed:
```
SEAL COLONY

Goal: {goal}
Phases: {phases_completed} of {total_phases} completed
Current Milestone: {milestone}

{If incomplete_warning is not empty, display it here}

This will:
  - Award the Crowned Anthill milestone
  - Write CROWNED-ANTHILL.md ceremony record
  - Promote colony wisdom to QUEEN.md

Seal this colony? (yes/no)
```

Use `AskUserQuestion with yes/no options`.

If not "yes":
```
Sealing cancelled. Colony remains active.
```
Stop here.

### Step 3.5: Analytics Review

Before wisdom approval, spawn Sage to analyze colony trends and provide data-driven insights.

**Check phase threshold and spawn Sage:**
```bash
# Check if colony has enough history for meaningful analytics
phases_completed=$(jq '[.plan.phases[] | select(.status == "completed")] | length' .aether/data/COLONY_STATE.json 2>/dev/null || echo "0")

if [[ "$phases_completed" -ge 3 ]]; then
  # Generate Sage name and dispatch
  sage_name=$(bash .aether/aether-utils.sh generate-ant-name "sage")
  bash .aether/aether-utils.sh spawn-log "Queen" "sage" "$sage_name" "Colony analytics review"

  # Display spawn notification
  echo ""
  echo "📜🐜 Sage $sage_name spawning — Analyzing colony trends and patterns..."
fi
```

**Spawn Sage using Task tool when threshold is met:**
If phases_completed >= 3, spawn the Sage agent using Task tool with `subagent_type="aether-sage"`:

```xml
<task>
  <description>📜 Sage {sage_name}: Colony analytics review</description>
  <prompt>
You are {sage_name}, a 📜 Sage Ant.

Mission: Analyze colony history and provide data-driven insights for wisdom promotion

Data Sources:
- COLONY_STATE.json: Colony configuration, phases, learnings, instincts
- activity.log: Activity history with timestamps
- midden.json: Failure patterns and flagged issues

Analysis Areas:
1. Velocity Trends: Phase completion rate over time
2. Bug Density: Error frequency and patterns from midden
3. Review Turnaround: Time from build to continue

Work:
1. Read COLONY_STATE.json for phase history and learnings
2. Read activity.log for velocity patterns
3. Read midden.json for bug density analysis
4. Calculate metrics and identify trends
5. Provide recommendations for wisdom promotion

**IMPORTANT:** You are strictly read-only. Do not modify any files.

Log activity: bash .aether/aether-utils.sh activity-log "ANALYZING" "{sage_name}" "description"

Return ONLY this JSON (no other text):
{{
  "ant_name": "{sage_name}",
  "caste": "sage",
  "status": "completed" | "failed" | "blocked",
  "summary": "What you analyzed and found",
  "key_findings": [],
  "trends": {{
    "velocity": "increasing|stable|decreasing",
    "bug_density": "high|medium|low",
    "review_turnaround": "fast|moderate|slow"
  }},
  "metrics_analyzed": [],
  "predictions": [],
  "recommendations": [
    {{"priority": 1, "action": "", "expected_impact": ""}}
  ],
  "next_steps": [],
  "blockers": []
}}
  </prompt>
</task>
```

**FALLBACK:** If "Agent type not found", use general-purpose agent and inject role: "You are a Sage Ant - analytics specialist that extracts trends from history to guide decisions."

**Parse Sage JSON output:**
After spawn, extract from response: `key_findings`, `trends`, `recommendations`

Log completion and update swarm display:
```bash
bash .aether/aether-utils.sh spawn-complete "$sage_name" "completed" "Analytics review complete"
```

**Display Sage completion line:**
```
📜 {sage_name}: Colony analytics review complete ✓
   Velocity: {trend} | Bug density: {level} | Review turnaround: {speed}
```

**Log high-priority recommendations to midden (non-blocking):**
For each recommendation with priority <= 2:
```bash
bash .aether/aether-utils.sh midden-write "analytics" "Sage recommendation (P{priority}): {action}" "sage"
```

**Display insights summary:**
```
📜 Sage Insights:
   Key Findings: {count}
   Top Recommendation: {first recommendation action}
```

**Continue to Step 3.6 (non-blocking):**
Proceed to Step 3.6 regardless of Sage findings — Sage is strictly non-blocking.

**If phases_completed < 3:**
Skip silently (no output) — proceed directly to Step 3.6.

### Step 3.6: Wisdom Approval

Before sealing, review wisdom proposals accumulated during this colony's lifecycle.

```bash
# --- Batch auto-promotion for auto-threshold observations (QUEEN-02) ---
# Auto-promote observations meeting higher recurrence thresholds
# before presenting the interactive review for lower-threshold proposals.

obs_file=".aether/data/learning-observations.json"
auto_promoted_count=0

if [[ -f "$obs_file" ]]; then
  for encoded in $(jq -r '.observations[] | @base64' "$obs_file" 2>/dev/null); do
    content=$(echo "$encoded" | base64 -d | jq -r '.content // empty')
    wisdom_type=$(echo "$encoded" | base64 -d | jq -r '.wisdom_type // "pattern"')
    colony=$(echo "$encoded" | base64 -d | jq -r '.colonies[0] // "unknown"')
    [[ -z "$content" ]] && continue

    result=$(bash .aether/aether-utils.sh learning-promote-auto "$wisdom_type" "$content" "$colony" "learning" 2>/dev/null || echo '{}')
    was_promoted=$(echo "$result" | jq -r '.result.promoted // false' 2>/dev/null || echo "false")
    if [[ "$was_promoted" == "true" ]]; then
      auto_promoted_count=$((auto_promoted_count + 1))
    fi
  done
fi

if [[ "$auto_promoted_count" -gt 0 ]]; then
  echo "Auto-promoted $auto_promoted_count observation(s) to QUEEN.md (met recurrence thresholds)"
  echo ""
fi
# --- END Batch auto-promotion ---

# Check for pending proposals
proposals=$(bash .aether/aether-utils.sh learning-check-promotion 2>/dev/null || echo '{"proposals":[]}')
proposal_count=$(echo "$proposals" | jq '.proposals | length')

if [[ "$proposal_count" -gt 0 ]]; then
  echo ""
  echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
  echo "   🧠 WISDOM REVIEW"
  echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
  echo ""
  echo "Review wisdom proposals before sealing this colony."
  echo "Approved proposals will be promoted to QUEEN.md."
  echo ""

  # Run approval workflow (blocking)
  bash .aether/aether-utils.sh learning-approve-proposals

  echo ""
  echo "Wisdom review complete. Proceeding with sealing ceremony..."
  echo ""
else
  echo "No wisdom proposals to review."
fi
```

### Step 3.7: Hive Promotion (NON-BLOCKING)

After QUEEN.md promotion, promote abstracted instincts to the cross-colony hive.

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

**Continue to Step 4 (non-blocking):**
Proceed to Step 4 regardless of hive promotion results — hive promotion is strictly non-blocking.

### Step 4: Log Seal Activity

Log the seal ceremony to activity log:
```bash
bash .aether/aether-utils.sh activity-log "MODIFIED" "Queen" "Colony sealed - wisdom review completed"
```

### Step 5: Update Milestone to Crowned Anthill

Update COLONY_STATE.json:
1. Set `milestone` to `"Crowned Anthill"`
2. Set `milestone_updated_at` to current ISO-8601 timestamp
3. Append event: `"<timestamp>|milestone_reached|seal|Achieved Crowned Anthill milestone"`

Run `bash .aether/aether-utils.sh validate-state colony` after write.

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

### Step 5.5: Documentation Coverage Audit

Before writing the seal document, spawn a Chronicler to survey documentation coverage.

**Generate Chronicler name and dispatch:**
```bash
# Generate unique chronicler name
chronicler_name=$(bash .aether/aether-utils.sh generate-ant-name "chronicler")

# Log spawn and update swarm display
bash .aether/aether-utils.sh spawn-log "Queen" "chronicler" "$chronicler_name" "Documentation coverage audit"
```

**Display:**
```
━━━ 📝🐜 C H R O N I C L E R ━━━
──── 📝🐜 Spawning {chronicler_name} — documentation coverage audit ────
📝 Chronicler {chronicler_name} spawning — Surveying documentation coverage...
```

**Spawn Chronicler using Task tool:**
Spawn the Chronicler using Task tool with `subagent_type="aether-chronicler"`:

```xml
<task>
  <description>📝 Chronicler {chronicler_name}: Documentation coverage audit</description>
  <prompt>
You are {chronicler_name}, a 📝 Chronicler Ant.

Mission: Documentation coverage audit before seal ceremony

Survey the following documentation types:
- README.md (project overview, quick start)
- API documentation (endpoints, parameters, responses)
- Guides (tutorials, how-tos, best practices)
- Changelogs (version history, release notes)
- Code comments (JSDoc, TSDoc inline documentation)
- Architecture docs (system design, decisions)

Work:
1. Check if README.md exists and covers: installation, usage, examples
2. Look for docs/ directory and survey guide coverage
3. Check for API documentation (OpenAPI, README sections, etc.)
4. Verify CHANGELOG.md exists and has recent entries
5. Sample source files for inline documentation coverage
6. Identify documentation gaps (missing, outdated, incomplete)

**IMPORTANT:** You are strictly read-only. Do not modify any files.

Log activity: bash .aether/aether-utils.sh activity-log "SURVEYING" "{chronicler_name}" "description"

Return ONLY this JSON (no other text):
{
  "ant_name": "{chronicler_name}",
  "caste": "chronicler",
  "status": "completed" | "failed" | "blocked",
  "summary": "What you surveyed and found",
  "documentation_created": [],
  "documentation_updated": [],
  "pages_documented": 0,
  "code_examples_verified": [],
  "coverage_percent": 0,
  "gaps_identified": [
    {"type": "README|API|Guide|Changelog|Comments|Architecture", "severity": "high|medium|low", "description": "...", "location": "..."}
  ],
  "blockers": []
}
  </prompt>
</task>
```

**FALLBACK:** If "Agent type not found", use general-purpose agent and inject role: "You are a Chronicler Ant - documentation specialist that surveys and identifies documentation gaps."

**Parse Chronicler JSON output:**
Extract from response: `coverage_percent`, `gaps_identified`, `pages_documented`

Log completion and update swarm display:
```bash
bash .aether/aether-utils.sh spawn-complete "$chronicler_name" "completed" "Documentation audit complete"
```

**Display Chronicler completion line:**
```
📝 {chronicler_name}: Documentation coverage audit ({pages_documented} pages, {coverage_percent}% coverage) ✓
```

**Log gaps to midden (non-blocking):**
For each gap in `gaps_identified` with severity "high" or "medium":
```bash
bash .aether/aether-utils.sh midden-write "documentation" "Gap ({severity}): {description} at {location}" "chronicler"
```

**Display summary:**
```
📝 Chronicler complete — {coverage_percent}% coverage, {gap_count} gaps logged to midden
```

**Continue to Step 6 (non-blocking):**
Proceed to Step 6 regardless of Chronicler findings — Chronicler is strictly non-blocking.

### Step 6: Write CROWNED-ANTHILL.md

Calculate colony age:
```bash
initialized_at=$(jq -r '.initialized_at // empty' .aether/data/COLONY_STATE.json)
if [[ -n "$initialized_at" ]]; then
  init_epoch=$(date -j -f "%Y-%m-%dT%H:%M:%SZ" "$initialized_at" +%s 2>/dev/null || echo 0)
  now_epoch=$(date +%s)
  if [[ "$init_epoch" -gt 0 ]]; then
    colony_age_days=$(( (now_epoch - init_epoch) / 86400 ))
  else
    colony_age_days=0
  fi
else
  colony_age_days=0
fi
```

Extract phase recap:
```bash
phase_recap=""
while IFS= read -r phase_line; do
  phase_name=$(echo "$phase_line" | jq -r '.name')
  phase_status=$(echo "$phase_line" | jq -r '.status')
  phase_recap="${phase_recap}  - ${phase_name}: ${phase_status}\n"
done < <(jq -c '.plan.phases[]' .aether/data/COLONY_STATE.json 2>/dev/null)
```

Write the seal document:
```bash
version=$(jq -r '.version // "3.0"' .aether/data/COLONY_STATE.json)
seal_date=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
```

Resolve the crowned-anthill template path:
  Check ~/.aether/system/templates/crowned-anthill.template.md first,
  then .aether/templates/crowned-anthill.template.md.

If no template found: output "Template missing: crowned-anthill.template.md. Run aether update to fix." and stop.

Read the template file. Fill all {{PLACEHOLDER}} values:
  - {{GOAL}} → goal (from colony state)
  - {{SEAL_DATE}} → seal_date (ISO-8601 UTC timestamp)
  - {{VERSION}} → version (from colony state)
  - {{TOTAL_PHASES}} → total_phases
  - {{PHASES_COMPLETED}} → phases_completed
  - {{COLONY_AGE_DAYS}} → colony_age_days
  - {{PROMOTIONS_MADE}} → promotions_made
  - {{PHASE_RECAP}} → phase recap list (one entry per line, formatted from the bash loop above)

Remove the HTML comment lines at the top of the template (lines starting with <!--).
Write the result to .aether/CROWNED-ANTHILL.md using the Write tool.

### Step 6.5: Export XML Archive (best-effort)

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
else
  xml_export_line="XML Archive: skipped (xmllint not available)"
  pher_export_line="Signal Export: skipped (xmllint not available)"
fi
```

### Step 7: Display Ceremony


Display the ASCII art ceremony:
```
        .     .
       /|\   /|\
      / | \ / | \
     /  |  X  |  \
    /   | / \ |   \
   /    |/   \|    \
  /     /     \     \
 /____ /  ___  \ ____\
      / /   \ \
     / /     \ \
    /_/       \_\
     |  CROWNED |
     | ANTHILL  |
     |__________|
```

Below the ASCII art, display:
```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
   C R O W N E D   A N T H I L L
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Goal: {goal}
Phases: {phases_completed} of {total_phases} completed
{If incomplete_warning is not empty: display it}
Wisdom Promoted: {promotion_summary}

Seal Document: .aether/CROWNED-ANTHILL.md
{xml_export_line}
{pher_export_line}

The colony stands crowned and sealed.
Its wisdom lives on in QUEEN.md.
The anthill has reached its final form.

──────────────────────────────────────────────────
🐜 Next Up
──────────────────────────────────────────────────
   /ant:entomb              🏺 Archive colony to chambers
   /ant:lay-eggs            🥚 Start a new colony
   /ant:tunnels             🗄️  Browse archived chambers
```

### Edge Cases

**Colony already at Crowned Anthill:**
- Display message and guide to /ant:entomb. Do NOT re-seal.

**Phases incomplete:**
- Warn but allow. The seal proceeds after confirmation.

**Missing QUEEN.md:**
- queen-init creates it. If that fails, skip promotion (non-fatal).

**Missing initialized_at:**
- Colony age defaults to 0 days.

**Empty phases array:**
- Can seal a colony with 0 phases (rare but valid). phases_completed = 0, total_phases = 0.
