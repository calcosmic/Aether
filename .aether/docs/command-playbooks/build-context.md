### Step 4: Load Colony Context (colony-prime)

Call `colony-prime --compact` to get unified worker context (wisdom + context capsule + signals + instincts):

Run using the Bash tool with description "Loading colony context...":
```bash
prime_result=$(bash .aether/aether-utils.sh colony-prime --compact 2>/dev/null)
```

**Parse the JSON response:**
- If `.ok` is false: This is a FAIL HARD error - display the error message and stop the build
- If successful: Extract from `.result`:
  - `signal_count` - number of active pheromone signals
  - `instinct_count` - number of filtered instincts
  - `prompt_section` - the formatted markdown to inject into worker prompts
  - `log_line` - status message for display

Display after constraints:
```
{log_line from colony-prime}
```

Then display the active pheromones table by running:
```bash
bash .aether/aether-utils.sh pheromone-display
```

This shows the user exactly what signals are guiding the colony:
- 🎯 FOCUS signals (what to pay attention to)
- 🚫 REDIRECT signals (what to avoid - hard constraints)
- 💬 FEEDBACK signals (guidance to consider)

**Store for worker injection:** The `prompt_section` variable contains compact formatted context (QUEEN wisdom + context capsule + pheromone signals) ready for injection.

### Step 4.0: Load Territory Survey

Check if territory survey exists and load relevant documents:

Run using the Bash tool with description "Loading territory survey...":
```bash
bash .aether/aether-utils.sh survey-load "{phase_name}" 2>/dev/null
```

**Parse the JSON response:**
- If `.ok` is false: Set `survey_docs = null` and skip survey loading
- If successful: Extract `.docs` (comma-separated list) and `.dir`

**Determine phase type from phase name:**
| Phase Contains | Documents to Load |
|----------------|-------------------|
| UI, frontend, component, button, page | DISCIPLINES.md, CHAMBERS.md |
| API, endpoint, backend, route | BLUEPRINT.md, DISCIPLINES.md |
| database, schema, model, migration | BLUEPRINT.md, PROVISIONS.md |
| test, spec, coverage | SENTINEL-PROTOCOLS.md, DISCIPLINES.md |
| integration, external, client | TRAILS.md, PROVISIONS.md |
| refactor, cleanup, debt | PATHOGENS.md, BLUEPRINT.md |
| setup, config, initialize | PROVISIONS.md, CHAMBERS.md |
| *default* | PROVISIONS.md, BLUEPRINT.md |

**Read the relevant survey documents** from `.aether/data/survey/`:
- Extract key patterns to follow
- Note file locations for new code
- Identify known concerns to avoid

**Display summary:**
```
━━━ 🗺️🐜 S U R V E Y   L O A D E D ━━━
{for each doc loaded}
  {emoji} {filename} — {brief description}
{/for}

{if no survey}
  (No territory survey — run /ant:colonize for deeper context)
{/if}
```

**Store for builder injection:**
- `survey_patterns` — patterns to follow
- `survey_locations` — where to place files
- `survey_concerns` — concerns to avoid

### Step 4.1: Archaeologist Pre-Build Scan

**Conditional step — only fires when the phase modifies existing files.**

1. **Detect existing-file modification:**
   Examine each task in the phase. Look at task descriptions, constraints, and hints for signals:
   - Keywords: "update", "modify", "add to", "integrate into", "extend", "change", "refactor", "fix"
   - References to existing file paths (files that already exist in the repo)
   - Task type: if a task is purely "create new file X" with no references to existing code, it is new-file-only

   **If ALL tasks are new-file-only** (no existing files will be modified):
   - Skip this step silently — produce no output, no spawn
   - Proceed directly to Step 4.2

### Step 4.2: Suggest Pheromones

**Conditional step — skipped if `--no-suggest` flag is passed.**

Analyze codebase and suggest pheromone signals based on detected patterns.

Run using the Bash tool with description "Analyzing codebase for suggestions...":
```bash
bash .aether/aether-utils.sh suggest-approve --dry-run 2>/dev/null
```

Parse the JSON result to get `suggestion_count`.

If `suggestion_count` > 0:
- Display: "💡 {count} pheromone suggestion(s) detected from code analysis"
- Run: `bash .aether/aether-utils.sh suggest-approve`
- Parse result for approved/rejected/skipped counts
- If approved > 0: Display "✓ {approved} FOCUS signal(s) added"

If `suggestion_count` == 0:
- Skip silently (no output)

**Non-blocking**: This step never stops the build. Even if suggest-approve fails,
log a warning and continue to Step 5.

**Error handling**:
- If suggest-analyze returns error: Log warning, continue
- If suggest-approve returns error: Log warning, continue
- Never let suggestion failures block the build

2. **If existing code modification detected — spawn Archaeologist Scout:**

   Generate archaeologist name and dispatch:
   Run using the Bash tool with description "Naming archaeologist...": `bash .aether/aether-utils.sh generate-ant-name "archaeologist"` (store as `{archaeologist_name}`)
   Run using the Bash tool with description "Dispatching archaeologist...": `bash .aether/aether-utils.sh spawn-log "Queen" "scout" "{archaeologist_name}" "Pre-build archaeology scan" && bash .aether/aether-utils.sh swarm-display-update "{archaeologist_name}" "scout" "excavating" "Pre-build archaeology scan" "Queen" '{"read":0,"grep":0,"edit":0,"bash":0}' 0 "fungus_garden" 15`

   Display:
   ```
   🏺🐜 Archaeologist {archaeologist_name} spawning
       Scanning history of files to be modified...
   ```

   Spawn a Scout (using Task tool with `subagent_type="aether-archaeologist"`, include `description: "🏺 Archaeologist {archaeologist_name}: Pre-build history scan"`) with this prompt:
   # FALLBACK: If "Agent type not found", use general-purpose and inject role: "You are an Archaeologist Ant - git historian that excavates why code exists."

   ```
   You are {Archaeologist-Name}, a 🏺🐜 Archaeologist Ant.

   Mission: Pre-build archaeology scan

   Files: {list of existing files that will be modified}

   Work:
   1. Read each file to understand current state
   2. Run: git log --oneline -15 -- "{file_path}" for history
   3. Run: git log --all --grep="fix\|bug\|workaround\|hack\|revert" --oneline -- "{file_path}" for incident history
   4. Run: git blame "{file_path}" | head -40 for authorship
   5. Note TODO/FIXME/HACK markers

   Log activity: bash .aether/aether-utils.sh activity-log "READ" "{Ant-Name}" "description"

   Report (plain text):
   - WHY key code sections exist (from commits)
   - Known workarounds/hacks to preserve
   - Key architectural decisions
   - Areas of caution (high churn, reverts, emergencies)
   - Stable bedrock vs volatile sand sections
   ```

   **Wait for results** (blocking — use TaskOutput with `block: true`).

   Log completion and update swarm display:
   Run using the Bash tool with description "Recording archaeologist findings...": `bash .aether/aether-utils.sh spawn-complete "{archaeologist_name}" "completed" "Pre-build archaeology scan" && bash .aether/aether-utils.sh swarm-display-update "{archaeologist_name}" "scout" "completed" "Pre-build archaeology scan" "Queen" '{"read":8,"grep":5,"edit":0,"bash":2}' 100 "fungus_garden" 100`

3. **Store and display findings:**

   Store the archaeologist's output as `archaeology_context`.

   Display summary:
   ```
   ━━━ 🏺🐜 A R C H A E O L O G Y ━━━
   {summary of findings from archaeologist}
   ```

4. **Injection into builder prompts:**
   The `archaeology_context` will be injected into builder prompts in Step 5.1 (see below).
   If this step was skipped (no existing files modified), the archaeology section is omitted from builder prompts.

---
