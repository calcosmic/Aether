---
name: ant:colonize
description: "📊🐜🗺️🐜📊 Survey territory with 4 parallel scouts for comprehensive colony intelligence"
---

You are the **Queen**. Dispatch Surveyor Ants to map the territory.

The arguments are: `$ARGUMENTS`

**Parse arguments:**
- If contains `--no-visual`: set `visual_mode = false` (visual is ON by default)
- If contains `--force` or `--force-resurvey`: set `force_resurvey = true`
- Otherwise: set `visual_mode = true`, `force_resurvey = false`

<failure_modes>
### Existing Survey Overwrite
If .aether/data/survey/ already contains survey documents:
- Warn before overwriting: "Existing survey found from [date]. Re-surveying will replace it."
- Options: (1) Continue and overwrite, (2) Keep existing survey, (3) Merge (re-survey only outdated sections)

### Surveyor Spawn Failure
If a surveyor agent fails during codebase exploration:
- Report which survey document was not produced
- Partial surveys are acceptable -- note which documents are complete vs. missing
- Recovery: user can re-run /ant:colonize to regenerate missing surveys
</failure_modes>

<success_criteria>
Command is complete when:
- All surveyor agents have completed their exploration
- Survey documents exist in .aether/data/survey/
- COLONY_STATE.json reflects colonized status
- User sees summary of survey findings
</success_criteria>

<read_only>
Do not touch during colonize:
- .aether/dreams/ (user notes)
- .aether/chambers/ (archived colonies)
- Source code files (survey is read-only exploration)
- .env* files
- .claude/settings.json
- COLONY_STATE.json structure beyond colonize-specific fields
</read_only>

## Instructions

### Step 0: Initialize Visual Mode (if enabled)

If `visual_mode` is true:
```bash
# Generate session ID
colonize_id="colonize-$(date +%s)"

# Initialize swarm display (consolidated)
bash .aether/aether-utils.sh swarm-display-init "$colonize_id" && bash .aether/aether-utils.sh swarm-display-update "Queen" "prime" "dispatching" "Surveying territory" "Colony" '{"read":0,"grep":0,"edit":0,"bash":0}' 0 "fungus_garden" 0
```

Display header:
```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
📊🐜🗺️🐜📊  C O L O N I Z E  —  T e r r i t o r y  S u r v e y
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Queen dispatching Surveyor Ants...
```

### Step 1: Validate

Read `.aether/data/COLONY_STATE.json`.

**If the file does not exist or cannot be read:**
1. Create `.aether/data/` directory if it does not exist.
2. Write a minimal COLONY_STATE.json:
   `{"version": "3.0", "goal": null, "state": "IDLE", "current_phase": 0, "session_id": null, "initialized_at": null, "build_started_at": null, "plan": {"generated_at": null, "confidence": null, "phases": []}, "memory": {"phase_learnings": [], "decisions": [], "instincts": []}, "errors": {"records": [], "flagged_patterns": []}, "signals": [], "graveyards": [], "events": []}`
3. Output: "No colony state found. Bootstrapping minimal state for territory survey."

**If the file exists:** continue.

**If `plan.phases` is not empty:** output "Colony already has phases. Use /ant:continue.", stop.

### Step 2: Quick Surface Scan (for session context)

Use Glob to find key files (read up to 20 total) to provide context for the survey.

**Package manifests:**
- package.json, Cargo.toml, pyproject.toml, go.mod, Gemfile, pom.xml, build.gradle

**Documentation:**
- README.md, README.*, docs/README.md

**Entry points:**
- src/index.*, src/main.*, main.*, app.*, lib/index.*, index.*

**Config:**
- tsconfig.json, .eslintrc.*, jest.config.*, vite.config.*, webpack.config.*

Read found files. Extract basic info:
- Tech stack (language, framework)
- Entry points (main files)
- Key directories

### Step 3: Dispatch Surveyor Ants (Parallel)

Create the survey directory:
```bash
mkdir -p .aether/data/survey
```

#### Step 3.1: Check for Stale Survey Session

Before dispatching surveyors, check for existing survey files and capture session start time:

```bash
SURVEY_START=$(date +%s)

# Check for stale survey files
stale_check=$(bash .aether/aether-utils.sh session-verify-fresh --command survey "" "$SURVEY_START")
has_stale=$(echo "$stale_check" | jq -r '.stale | length')
has_fresh=$(echo "$stale_check" | jq -r '.fresh | length')

if [[ "$has_stale" -gt 0 ]] || [[ "$has_fresh" -gt 0 ]]; then
  # Found existing survey files
  if [[ "$force_resurvey" == "true" ]]; then
    bash .aether/aether-utils.sh session-clear --command survey
    echo "Cleared existing survey files for fresh territory mapping"
  else
    echo "Found existing territory survey. Use --force-resurvey to remap."
    # Continue - will use existing survey files
  fi
fi
```

Generate unique names for the 4 Surveyor Ants (each name must be captured separately):
```bash
 bash .aether/aether-utils.sh generate-ant-name "surveyor"
bash .aether/aether-utils.sh generate-ant-name "surveyor"
bash .aether/aether-utils.sh generate-ant-name "surveyor"
bash .aether/aether-utils.sh generate-ant-name "surveyor"
```

Log the dispatches (consolidated - fire-and-forget logging):
```bash
bash .aether/aether-utils.sh spawn-log "Queen" "surveyor" "{provisions_name}" "Mapping provisions and trails" && bash .aether/aether-utils.sh spawn-log "Queen" "surveyor" "{nest_name}" "Mapping nest structure" && bash .aether/aether-utils.sh spawn-log "Queen" "surveyor" "{disciplines_name}" "Mapping disciplines and sentinels" && bash .aether/aether-utils.sh spawn-log "Queen" "surveyor" "{pathogens_name}" "Identifying pathogens"
```

**Spawn 4 Surveyor Ants in parallel using the Task tool:**

Each Task should use `subagent_type="aether-surveyor-{focus}"`:
1. `aether-surveyor-provisions` — Maps PROVISIONS.md and TRAILS.md
2. `aether-surveyor-nest` — Maps BLUEPRINT.md and CHAMBERS.md
3. `aether-surveyor-disciplines` — Maps DISCIPLINES.md and SENTINEL-PROTOCOLS.md
4. `aether-surveyor-pathogens` — Maps PATHOGENS.md

**Prompt for each surveyor:**
```
You are Surveyor Ant {name}. Explore this codebase and write your survey documents.

Focus: {provisions|nest|disciplines|pathogens}

The surface scan found:
- Language: {language}
- Framework: {framework}
- Key directories: {dirs}

Write your documents to `.aether/data/survey/` following your agent template.
Return only confirmation when complete — do not include document contents.
```

Collect confirmations from all 4 surveyors. Each should return:
- Document name(s) written
- Line count(s)
- Brief status

### Step 4: Verify Survey Completeness

Check that all 7 documents were created (consolidated):
```bash
ls .aether/data/survey/PROVISIONS.md .aether/data/survey/TRAILS.md .aether/data/survey/BLUEPRINT.md .aether/data/survey/CHAMBERS.md .aether/data/survey/DISCIPLINES.md .aether/data/survey/SENTINEL-PROTOCOLS.md .aether/data/survey/PATHOGENS.md 2>&1 | grep -q "No such file" && echo "Some documents missing" || echo "All survey documents present"
```

If any documents are missing, note which ones in the output.

#### Step 4.5: Verify Survey Files Are Fresh

Verify that all survey files were created after the session start:
```bash
verify_result=$(bash .aether/aether-utils.sh session-verify-fresh --command survey "" "$SURVEY_START")
fresh_count=$(echo "$verify_result" | jq -r '.fresh | length')

if [[ "$fresh_count" -lt 7 ]]; then
  echo "Warning: Some survey files may be stale or missing"
  echo "$verify_result" | jq -r '.stale[], .missing[]' | while read doc; do
    echo "  - $doc"
  done
fi
```

Display colony activity summary:
```bash
bash .aether/aether-utils.sh swarm-display-text "$colonize_id"
```

### Step 5: Update State

Read `.aether/data/COLONY_STATE.json`. Update:
- Set `state` to `"IDLE"` (ready for planning)
- Set `territory_surveyed` to `"<ISO-8601 UTC>"`

Write Event: Append to the `events` array as pipe-delimited string:
`"<ISO-8601 UTC>|territory_surveyed|colonize|Territory surveyed: 7 documents"`

If the `events` array exceeds 100 entries, remove the oldest entries to keep only 100.

Write the updated COLONY_STATE.json.

### Step 6: Confirm

Output header:

```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
📊🐜🗺️🐜📊  T E R R I T O R Y   S U R V E Y   C O M P L E T E
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
```

Then output:

```
🗺️ Colony territory has been surveyed.

Survey Reports:
  📦 PROVISIONS.md         — Tech stack & dependencies
  🛤️  TRAILS.md             — External integrations
  📐 BLUEPRINT.md          — Architecture patterns
  🏠 CHAMBERS.md           — Directory structure
  📜 DISCIPLINES.md        — Coding conventions
  🛡️  SENTINEL-PROTOCOLS.md — Testing patterns
  ⚠️  PATHOGENS.md          — Tech debt & concerns

Location: .aether/data/survey/

{If any docs missing:}
⚠️  Missing: {list missing documents}
{/if}

Stack: <language> + <framework>
Entry: <main entry point>
Files: <total count> across <N> directories

{Read the goal from COLONY_STATE.json. If goal is null:}
Next:
  /ant:init "<goal>"     Set colony goal (required before planning)
  /ant:focus "<area>"    Inject focus before planning
  /ant:redirect "<pat>"  Inject constraint before planning

{If goal is not null:}
Next:
  /ant:plan              Generate project plan (will load relevant survey docs)
  /ant:focus "<area>"    Inject focus before planning
  /ant:redirect "<pat>"  Inject constraint before planning
```

Generate the state-based Next Up block:
```bash
state=$(jq -r '.state // "IDLE"' .aether/data/COLONY_STATE.json)
current_phase=$(jq -r '.current_phase // 0' .aether/data/COLONY_STATE.json)
total_phases=$(jq -r '.plan.phases | length' .aether/data/COLONY_STATE.json)
bash .aether/aether-utils.sh print-next-up "$state" "$current_phase" "$total_phases"
```
