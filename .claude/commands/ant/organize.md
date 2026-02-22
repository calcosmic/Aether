---
name: ant:organize
description: "🧹🐜🏛️🐜🧹 Run codebase hygiene report - archivist ant scans for stale files, dead code, orphaned configs (report-only)"
---

You are the **Queen Ant Colony**. Spawn an archivist to analyze codebase hygiene.

> **Note:** `$ARGUMENTS` is unused. Future extensions could accept a path scope argument.

## Instructions

Parse `$ARGUMENTS`:
- If contains `--no-visual`: set `visual_mode = false` (visual is ON by default)
- Otherwise: set `visual_mode = true`

### Step 0: Initialize Visual Mode (if enabled)

If `visual_mode` is true, run using the Bash tool with description "Initializing organize display...":
```bash
organize_id="organize-$(date +%s)"
bash .aether/aether-utils.sh swarm-display-init "$organize_id"
bash .aether/aether-utils.sh swarm-display-update "Queen" "prime" "excavating" "Organize hygiene scan" "Colony" '{"read":0,"grep":0,"edit":0,"bash":0}' 0 "fungus_garden" 0
bash .aether/aether-utils.sh swarm-display-text "$organize_id"
```

### Step 1: Read State

Use the Read tool to read these files (in parallel):
- `.aether/data/COLONY_STATE.json`
- `.aether/data/activity.log`

From COLONY_STATE.json, extract:
- `goal` from top level
- `plan.phases` for phase data
- `errors.records` for error patterns
- `memory` for decisions/learnings
- `events` for activity

**Validate:** If `COLONY_STATE.json` has `goal: null`, output `No colony initialized. Run /ant:init first.` and stop.

### Step 2: Compute Active Pheromones

Run using the Bash tool with description "Loading active pheromones...":
```bash
bash .aether/aether-utils.sh pheromone-read
```

Use `.result.signals` as the active signal list (already decay-filtered by runtime logic).

Format as the standard ACTIVE PHEROMONES block:
```
ACTIVE PHEROMONES:
  {TYPE padded to 10 chars}: "{content}"
```

If no active signals after filtering:
```
  (no active pheromones)
```

### Step 3: Spawn Archivist (Keeper-Ant)

Read `.aether/workers.md` and extract the `## Keeper` section.

If `visual_mode` is true, run using the Bash tool with description "Updating organize display...":
```bash
bash .aether/aether-utils.sh swarm-display-update "Archivist" "keeper" "observing" "Codebase hygiene analysis" "Queen" '{"read":0,"grep":0,"edit":0,"bash":0}' 0 "fungus_garden" 30
```

Spawn via **Task tool** with `subagent_type="aether-keeper"`:
# FALLBACK: If "Agent type not found", use general-purpose and inject role: "You are a Keeper Ant - curates knowledge and synthesizes patterns."

```
--- WORKER SPEC ---
{Architect section from .aether/workers.md}

--- ACTIVE PHEROMONES ---
{pheromone block from Step 2}

--- TASK ---
You are being spawned as an ARCHIVIST ANT (codebase hygiene analyzer).

Your mission: Produce a structured HYGIENE REPORT. You are REPORT-ONLY.
You MUST NOT delete, modify, move, or create any project files.
You may ONLY read files and produce a report.

Colony goal: "{goal from COLONY_STATE.json}"
Colony mode: {mode from COLONY_STATE.json}
Current phase: {current_phase from COLONY_STATE.json}

--- COLONY DATA ---

PROJECT PLAN:
{plan.phases from COLONY_STATE.json -- phases, tasks, their statuses}

ERROR HISTORY:
{errors.records and errors.flagged_patterns from COLONY_STATE.json}

MEMORY:
{memory.phase_learnings and memory.decisions from COLONY_STATE.json}

ACTIVITY LOG (last 50 lines):
{tail of activity.log}

--- SCAN INSTRUCTIONS ---

Analyze the codebase for hygiene issues in three categories. For each finding,
assign a confidence level: HIGH (strong evidence), MEDIUM (likely but uncertain),
or LOW (speculative). Only HIGH confidence items should be presented as actionable.

**Category 1: Stale Files**
Check for files that may no longer be needed:
- Files referenced in completed tasks that might have been scaffolding/temporary
- Look at the project structure (use Glob tool to scan key directories)
- Check for TODO/FIXME/HACK comments that reference completed phases
- Check for test fixtures or mock data that reference completed features
- Look for empty files or stub implementations

**Category 2: Dead Code Patterns**
Use colony data to identify dead code signals:
- Recurring error patterns from COLONY_STATE.json errors.flagged_patterns (code that keeps breaking may be vestigial)
- Error categories with high counts concentrated in specific files
- Imports or dependencies referenced in errors but possibly no longer needed
- Read key source files and look for commented-out code blocks, unused exports, unreachable branches

**Category 3: Orphaned Configs**
Check for configuration that may not be connected:
- .aether/data/ files that have empty or default-only content
- Environment variables referenced in code but not in any .env example
- Config files that reference features/paths that don't exist
- Package.json scripts (if exists) that reference missing files

--- OUTPUT FORMAT ---

Produce your report in this exact structure:

CODEBASE HYGIENE REPORT
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Project: {goal}
Scanned: {timestamp}
Confidence threshold: HIGH findings are actionable, MEDIUM/LOW are informational

HIGH CONFIDENCE FINDINGS
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
{For each HIGH confidence finding:}
  [{category}] {description}
    Evidence: {what data/observation supports this}
    Location: {file path(s)}

{If no HIGH findings: "No high-confidence hygiene issues detected."}

MEDIUM CONFIDENCE OBSERVATIONS
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
{For each MEDIUM confidence finding:}
  [{category}] {description}
    Evidence: {what data/observation supports this}
    Location: {file path(s)}

{If no MEDIUM findings: "No medium-confidence observations."}

LOW CONFIDENCE NOTES
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
{For each LOW confidence finding:}
  [{category}] {description}

{If no LOW findings: "No low-confidence notes."}

SUMMARY
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  High: {count} actionable findings
  Medium: {count} observations
  Low: {count} notes

  Health: {CLEAN if 0 HIGH findings, MINOR ISSUES if 1-3 HIGH, NEEDS ATTENTION if 4+ HIGH}

CONSTRAINTS:
- Be CONSERVATIVE. When in doubt, classify as LOW confidence.
- Do NOT flag standard framework files (package.json, tsconfig.json, etc.) as orphaned.
- Do NOT flag .aether/ internal data files as stale (they are managed by the colony).
- Do NOT flag .claude/ command files as stale (they are the colony's brain).
- Aim for a useful report, not an exhaustive one. 5-15 findings is ideal.
```

### Step 4: Display Report

After the keeper-ant returns, display header:

```
🧹🐜🏛️🐜🧹 ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
   C O D E B A S E   H Y G I E N E   R E P O R T
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━ 🧹🐜🏛️🐜🧹
```

Then display using Bash tool with description "Displaying hygiene report header...":
```bash
bash -c 'printf "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n"'
bash -c 'printf "   C O D E B A S E   H Y G I E N E   R E P O R T\n"'
bash -c 'printf "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n\n"'
```

Then display the keeper-ant's full report verbatim.

If `visual_mode` is true, run using the Bash tool with description "Rendering organize display...":
```bash
bash .aether/aether-utils.sh swarm-display-update "Archivist" "keeper" "completed" "Hygiene report complete" "Queen" '{"read":8,"grep":4,"edit":0,"bash":2}' 100 "fungus_garden" 100
bash .aether/aether-utils.sh swarm-display-update "Queen" "prime" "completed" "Organize complete" "Colony" '{"read":2,"grep":1,"edit":1,"bash":1}' 100 "fungus_garden" 100
bash .aether/aether-utils.sh swarm-display-text "$organize_id"
```

### Step 5: Persist Report

Use the Write tool to write the full report to `.aether/data/hygiene-report.md` (overwriting any previous report).

Display:

```
---
Report saved: .aether/data/hygiene-report.md

This report is advisory only. No files were modified.

Next:
  /ant:status           View colony status
  /ant:build <phase>    Continue building
  /ant:focus "<area>"   Focus colony on a hygiene area
```

### Step 6: Log Activity

Use the Bash tool with description "Logging hygiene activity..." to run:
```
bash .aether/aether-utils.sh activity-log "COMPLETE" "queen" "Hygiene report generated"
```

Display persistence confirmation:

```
---
All state persisted. Safe to /clear context if needed.
  Report: .aether/data/hygiene-report.md
  Resume: /ant:resume-colony
```

### Step 7: Next Up

Generate the state-based Next Up block by running using the Bash tool with description "Generating Next Up suggestions...":
```bash
state=$(jq -r '.state // "IDLE"' .aether/data/COLONY_STATE.json)
current_phase=$(jq -r '.current_phase // 0' .aether/data/COLONY_STATE.json)
total_phases=$(jq -r '.plan.phases | length' .aether/data/COLONY_STATE.json)
bash .aether/aether-utils.sh print-next-up "$state" "$current_phase" "$total_phases"
```
