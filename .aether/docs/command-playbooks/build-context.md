### Step 4: Load Colony Context (colony-prime)

Call `colony-prime --compact` to get unified worker context (wisdom + context capsule + signals + instincts):

Run using the Bash tool with description "Loading colony context...":
```bash
prime_result=$(aether colony-prime --compact 2>/dev/null)
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
aether pheromone-display
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
aether survey-load "{phase_name}" 2>/dev/null
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

### Step 4.0.5: Load Phase Research

Load domain research generated during `/ant:plan` for injection into worker prompts:

Run using the Bash tool with description "Loading phase research...":
```bash
phase_id="{phase_number}"
research_file=".aether/data/phase-research/phase-${phase_id}-research.md"

if [[ -f "$research_file" ]]; then
  research_content=$(cat "$research_file")
  research_word_count=$(wc -w < "$research_file" | tr -d ' ')

  # Apply depth-based character budget (per D-03/D-04)
  budget_result=$(aether context-budget --depth "$colony_depth" 2>/dev/null || echo '{"ok":true,"result":{"context":8000}}')
  research_budget=$(echo "$budget_result" | jq -r '.result.context // 8000')
  if [[ ${#research_content} -gt $research_budget ]]; then
    research_content="${research_content:0:$research_budget}"
    echo "[research] trimmed to ${research_budget} chars (depth: $colony_depth)" >&2
  fi

  research_context="--- PHASE RESEARCH (Domain Knowledge) ---
${research_content}
--- END PHASE RESEARCH ---"

  echo "Research loaded: phase-${phase_id}-research.md (${research_word_count} words)"
else
  research_context=""
  echo "No phase research found -- plan was generated before research feature"
fi
```

**Parse the result:**
- If file exists: `research_context` contains the wrapped research content, ready for injection
- If file does NOT exist: `research_context` is empty, build continues without research (backward compatibility)

**Display:**
```
Research loaded: phase-{phase_id}-research.md ({research_word_count} words)
```
Or if no research file:
```
No phase research found -- plan was generated before research feature
```

**Store for worker injection:** The `research_context` variable is now available for build-wave.md and build-verify.md to inject into worker prompts. This depth-based budget scales with colony depth level via `aether context-budget`.

### Step 4.1: Archaeologist Pre-Build Scan

**Conditional step — only fires when the phase modifies existing files.**

**DEPTH CHECK: Also skip at "light" depth regardless of file modification.**

If `colony_depth` is "light": Skip this step silently, proceed to Step 4.2.
Otherwise: Apply existing file-modification conditional below.

1. **Detect existing-file modification:**
   Examine each task in the phase. Look at task descriptions, constraints, and hints for signals:
   - Keywords: "update", "modify", "add to", "integrate into", "extend", "change", "refactor", "fix"
   - References to existing file paths (files that already exist in the repo)
   - Task type: if a task is purely "create new file X" with no references to existing code, it is new-file-only

   **If ALL tasks are new-file-only** (no existing files will be modified):
   - Skip this step silently — produce no output, no spawn
   - Proceed directly to Step 4.2

### Step 4.2: Suggest Pheromones (DEPRECATED)

**Conditional step — skipped if `--no-suggest` flag is passed.**

> **DEPRECATED**: The `suggest-*` commands have been deprecated and will be removed
> in a future version. They return `ok:true` with `deprecated:true` for backward
> compatibility. This step now always skips gracefully.

Run using the Bash tool with description "Checking suggest deprecation status...":
```bash
suggest_result=$(aether suggest-approve --dry-run 2>/dev/null)
suggest_deprecated=$(echo "$suggest_result" | jq -r '.result.deprecated // false')

if [[ "$suggest_deprecated" == "true" ]]; then
    # Command is deprecated — skip silently, continue to Step 4.3
    :
elif [[ -z "$suggest_result" ]]; then
    # Command failed entirely — skip silently
    :
else
    # Legacy path: parse suggestion_count (for older aether versions)
    suggestion_count=$(echo "$suggest_result" | jq -r '.result.suggestion_count // 0')
    if [[ "$suggestion_count" -gt 0 ]]; then
        echo "$suggestion_count pheromone suggestion(s) detected from code analysis"
        aether suggest-approve 2>/dev/null || true
    fi
fi
```

**Non-blocking**: This step never stops the build.

**Error handling**:
- If suggest-approve returns error: Skip silently, continue
- If suggest-approve returns deprecated: Skip silently, continue
- Never let suggestion failures block the build

### Step 4.3: Skill Detection

**Non-blocking step — failures are logged and skipped.**

Build the skills index and detect which domain skills match the current codebase.

**4.3.1 — Build/read the skills index:**

Run using the Bash tool with description "Building skills index...":
```bash
skill_index_result=$(aether skill-index 2>/dev/null)
```

**Parse the JSON response:**
- If `.ok` is false or command fails: Set `skill_index_count = 0`, log warning, skip to next step
- If successful: Extract `.result.skill_count` as `skill_index_count`

**4.3.2 — Detect domain skills matching this codebase:**

Run using the Bash tool with description "Detecting codebase skills...":
```bash
skill_detect_result=$(aether skill-detect "$(pwd)" 2>/dev/null)
```

**Parse the JSON response:**
- If `.ok` is false or command fails: Set `skill_detections = "[]"`, log warning, continue
- If successful: Extract `.result.detections` as `skill_detections` (JSON array)
- Count entries in `skill_detections` as `skill_detection_count`

**4.3.3 — Store cross-stage state and display:**

Store the following variables for use by build-wave.md:
- `skill_index_count` — total number of skills in the index
- `skill_detections` — JSON array of matched skills with scores (e.g., `[{"name": "react", "score": 70}]`)

Display to user:
```
🧠 Skills: {skill_index_count} indexed, {skill_detection_count} matched to codebase
```

**Error handling:**
- If `skill-index` fails: Log `⚠️ Skill index unavailable — continuing without skills`, set defaults, continue
- If `skill-detect` fails: Log `⚠️ Skill detection failed — continuing without matches`, set defaults, continue
- Never let skill failures block the build

2. **If existing code modification detected — spawn Archaeologist Scout:**

   Generate archaeologist name and dispatch:
   Run using the Bash tool with description "Naming archaeologist...": `aether generate-ant-name "archaeologist"` (store as `{archaeologist_name}`)
   Run using the Bash tool with description "Dispatching archaeologist...": `aether spawn-log "Queen" "scout" "{archaeologist_name}" "Pre-build archaeology scan"`

   Display:
   ```
   ━━━ 🏺🐜 A R C H A E O L O G I S T ━━━
   ──── 🏺🐜 Spawning {archaeologist_name} — Pre-build history scan ────
   ```

   > **Platform note**: In Claude Code, use `Task tool with subagent_type`. In OpenCode, use the equivalent agent spawning mechanism for your platform (e.g., invoke the agent definition from `.opencode/agents/`).

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

   Log activity: aether activity-log "READ" "{Ant-Name}" "description"

   Report (plain text):
   - WHY key code sections exist (from commits)
   - Known workarounds/hacks to preserve
   - Key architectural decisions
   - Areas of caution (high churn, reverts, emergencies)
   - Stable bedrock vs volatile sand sections
   ```

   **Wait for results** (blocking — use TaskOutput with `block: true`).

   Log completion:
   Run using the Bash tool with description "Recording archaeologist findings...": `aether spawn-complete "{archaeologist_name}" "completed" "Pre-build archaeology scan"`

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
