---
name: ant:colonize
description: "📊🐜🗺️🐜📊 Survey territory with 4 parallel scouts for comprehensive colony intelligence"
---

You are the **Queen**. Dispatch Surveyor Ants to map the territory.

The arguments are: `$normalized_args`

**Parse arguments:**
- If contains `--no-visual`: set `visual_mode = false` (visual is ON by default)
- Otherwise: set `visual_mode = true`

## Instructions

### Step -1: Normalize Arguments

Run: `normalized_args=$(bash .aether/aether-utils.sh normalize-args "$@")`

This ensures arguments work correctly in both Claude Code and OpenCode. Use `$normalized_args` throughout this command.

### Step 0: Display Header

Display header:
```
📊🐜🗺️🐜📊 ═══════════════════════════════════════════════
         C O L O N I Z E  —  T e r r i t o r y  S u r v e y
═══════════════════════════════════════════════ 📊🐜🗺️🐜📊

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

Generate unique names for the 4 Surveyor Ants and log their dispatch:
```bash
bash .aether/aether-utils.sh generate-ant-name "surveyor"
bash .aether/aether-utils.sh generate-ant-name "surveyor"
bash .aether/aether-utils.sh generate-ant-name "surveyor"
bash .aether/aether-utils.sh generate-ant-name "surveyor"
```

Log the dispatch:
```bash
bash .aether/aether-utils.sh spawn-log "Queen" "surveyor" "{provisions_name}" "Mapping provisions and trails"
bash .aether/aether-utils.sh spawn-log "Queen" "surveyor" "{nest_name}" "Mapping nest structure"
bash .aether/aether-utils.sh spawn-log "Queen" "surveyor" "{disciplines_name}" "Mapping disciplines and sentinels"
bash .aether/aether-utils.sh spawn-log "Queen" "surveyor" "{pathogens_name}" "Identifying pathogens"
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

Check that all 7 documents were created:
```bash
ls .aether/data/survey/PROVISIONS.md 2>/dev/null && echo "PROVISIONS: OK" || echo "PROVISIONS: MISSING"
ls .aether/data/survey/TRAILS.md 2>/dev/null && echo "TRAILS: OK" || echo "TRAILS: MISSING"
ls .aether/data/survey/BLUEPRINT.md 2>/dev/null && echo "BLUEPRINT: OK" || echo "BLUEPRINT: MISSING"
ls .aether/data/survey/CHAMBERS.md 2>/dev/null && echo "CHAMBERS: OK" || echo "CHAMBERS: MISSING"
ls .aether/data/survey/DISCIPLINES.md 2>/dev/null && echo "DISCIPLINES: OK" || echo "DISCIPLINES: MISSING"
ls .aether/data/survey/SENTINEL-PROTOCOLS.md 2>/dev/null && echo "SENTINEL: OK" || echo "SENTINEL: MISSING"
ls .aether/data/survey/PATHOGENS.md 2>/dev/null && echo "PATHOGENS: OK" || echo "PATHOGENS: MISSING"
```

If any documents are missing, note which ones in the output.

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
📊🐜🗺️🐜📊 ═══════════════════════════════════════════════════
   T E R R I T O R Y   S U R V E Y   C O M P L E T E
═══════════════════════════════════════════════════ 📊🐜🗺️🐜📊
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
