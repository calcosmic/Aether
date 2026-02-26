---
name: ant:init
description: "🌱🐜🆕🐜🌱 Initialize Aether colony - Queen sets intention, colony mobilizes"
---

You are the **Queen Ant Colony**. Initialize the colony with the Queen's intention.

## Instructions

The user's goal is: `$ARGUMENTS`

Parse `$ARGUMENTS`:
- If contains `--no-visual`: set `visual_mode = false` (visual is ON by default)
- Otherwise: set `visual_mode = true`

<failure_modes>
### Colony State Overwrite
If COLONY_STATE.json already exists with an active colony:
- STOP before overwriting
- Warn: "Active colony detected with goal: [goal]. Overwriting will lose this data."
- Options: (1) Archive first with /ant:seal, (2) Continue and overwrite, (3) Cancel

### Write Failure Mid-Init
If writing COLONY_STATE.json fails partway:
- Remove the incomplete file (partial state is worse than no state)
- Report the error
- Recovery: user can run /ant:init again safely
</failure_modes>

<success_criteria>
Command is complete when:
- COLONY_STATE.json exists and is valid JSON
- Colony goal, milestone, and timestamp are set
- Session file is written
- User sees confirmation of colony creation
</success_criteria>

<read_only>
Do not touch during init:
- .aether/dreams/ (user notes)
- .aether/chambers/ (archived colonies)
- .env* files
- .claude/settings.json
- .github/workflows/
</read_only>

### Step 1: Validate Input

If `$ARGUMENTS` is empty or blank, output:

```
Aether Colony

  Initialize the colony with a goal. This creates the colony state,
  initializes constraints, and logs the init event.

  Usage: /ant:init "<your goal here>"

  Examples:
    /ant:init "Build a REST API with authentication"
    /ant:init "Create a soothing sound application"
    /ant:init "Design a calculator CLI tool"
```

Stop here. Do not proceed.

### Step 1.5: Verify Aether Setup

Check if `.aether/aether-utils.sh` exists using the Read tool.

**If the file already exists** — skip this step entirely. Aether is set up.

**If the file does NOT exist:**
```
Aether is not set up in this repo yet.

Run /ant:lay-eggs first to create the .aether/ directory
with all system files, then run /ant:init "your goal" to
start a colony.

If the global hub isn't installed either:
  npm install -g aether-colony   (installs the hub)
  /ant:lay-eggs                  (sets up this repo)
  /ant:init "your goal"          (starts the colony)
```
Stop here. Do not proceed.

### Step 1.6: Initialize QUEEN.md Wisdom Document

After bootstrap completes (or if system files already existed), run using the Bash tool:
```
bash .aether/aether-utils.sh queen-init
```

Parse the JSON result:
- If `created` is true: Display `QUEEN.md initialized`
- If `created` is false and `reason` is "already_exists": Display `QUEEN.md already exists`

This step is non-blocking — proceed regardless of outcome.

### Step 2: Read Current State with Freshness Check

Capture session start time:
```bash
INIT_START=$(date +%s)
```

Use the Read tool to read `.aether/data/COLONY_STATE.json`.

Check freshness of existing state:
```bash
fresh_check=$(bash .aether/aether-utils.sh session-verify-fresh --command init "" "$INIT_START")
is_stale=$(echo "$fresh_check" | jq -r '.stale | length')
freshness_status=$([[ "$is_stale" -gt 0 ]] && echo "stale" || echo "fresh")
```

If the `goal` field is not null:
- If state is stale (old session): Warn user but proceed
- If state is fresh (active session): Strongly recommend continuation

```
Colony already initialized with goal: "{existing_goal}"

State freshness: {freshness_status}
Initialized: {initialized_at}

To reinitialize with a new goal, the current state will be reset.
Proceeding with new goal: "{new_goal}"
```

**Note:** Init never auto-clears COLONY_STATE.json. Reinitialization is an explicit user choice.

### Step 2.6: Load Prior Colony Knowledge (Optional)

Check if `.aether/data/completion-report.md` exists using the Read tool.

**If the file does NOT exist**, skip to Step 3 — this is a fresh colony with no prior history.

**If the file exists**, read it and extract:
1. **Instincts** — look for the `## Colony Instincts` section. Each line has format: `N. [confidence] domain: description`. Keep only instincts with confidence >= 0.5.
2. **Learnings** — look for the `## Colony Learnings (Validated)` section. Keep all numbered items.

Store the extracted instincts and learnings for use in Step 3 (Write Colony State). Display a brief note:

```
🧠 Prior colony knowledge found:
   {N} instinct(s) inherited (confidence >= 0.5)
   {N} validated learning(s) carried forward
```

If no instincts meet the threshold, display:
```
🧠 Prior colony knowledge found but no high-confidence instincts to inherit.
```

**Important:** This step is read-only and non-blocking. If the file is malformed or unreadable, skip silently and proceed to Step 3 with empty memory.

### Step 3: Write Colony State

Generate a session ID in the format `session_{unix_timestamp}_{random}` and an ISO-8601 UTC timestamp.

Resolve the colony-state template path:
  Check `~/.aether/system/templates/colony-state.template.json` first,
  then `.aether/templates/colony-state.template.json`.

If no template found: output "Template missing: colony-state.template.json. Run aether update to fix." and stop.

Read the template file. Follow its `_instructions` field.
Replace all `__PLACEHOLDER__` values:
  - `__GOAL__` → the user's goal from $ARGUMENTS
  - `__SESSION_ID__` → the generated session ID (format: `session_{unix_timestamp}_{random}`)
  - `__ISO8601_TIMESTAMP__` → the current ISO-8601 UTC timestamp (used in both `initialized_at` and the events entry)
  - `__PHASE_LEARNINGS__` → JSON array from Step 2.6, or `[]` if none
  - `__INSTINCTS__` → JSON array from Step 2.6, or `[]` if none

IMPORTANT: `__PHASE_LEARNINGS__` and `__INSTINCTS__` must be JSON array values (e.g., `[]` not `"[]"`).

**If Step 2.6 found instincts to inherit**, convert each into the instinct format for the `__INSTINCTS__` array. Each inherited instinct should have:
- `id`: `instinct_inherited_{index}`
- `trigger`: inferred from the instinct description
- `action`: the instinct description
- `confidence`: the original confidence value (from the completion report)
- `domain`: the original domain (from the completion report)
- `source`: `"inherited:completion-report"`
- `evidence`: `["Validated in prior colony session"]`
- `created_at`: current ISO-8601 timestamp
- `last_applied`: null
- `applications`: 0
- `successes`: 0

**If Step 2.6 found validated learnings**, seed the `__PHASE_LEARNINGS__` array with each as:
- `phase`: `"inherited"`
- `learning`: the learning text
- `status`: `"validated"`
- `source`: `"inherited:completion-report"`

**If Step 2.6 was skipped or found nothing**, use empty arrays `[]` for both `__PHASE_LEARNINGS__` and `__INSTINCTS__`.

Remove ALL keys starting with underscore (`_template`, `_version`, `_instructions`, `_comment_*`).
Write the resulting JSON to `.aether/data/COLONY_STATE.json` using the Write tool.

### Step 4: Initialize Constraints

Resolve the constraints template path:
  Check `~/.aether/system/templates/constraints.template.json` first,
  then `.aether/templates/constraints.template.json`.

If no template found: output "Template missing: constraints.template.json. Run aether update to fix." and stop.

Read the template file. Follow its `_instructions` field.
No placeholder substitution needed — the data keys are written as-is.
Remove ALL keys starting with underscore (`_template`, `_version`, `_instructions`, `_comment_*`).
Write the resulting JSON to `.aether/data/constraints.json` using the Write tool.

### Step 4.5: Initialize Runtime Files from Templates

Initialize runtime files that support colony operations. Each file is created from its template if it doesn't already exist.

**For each template, check both hub and local paths:**
- `~/.aether/system/templates/{template}` first
- `.aether/templates/{template}` second

**Files to initialize:**

1. **pheromones.json** - Signal tracking for colony guidance
   - Template: `pheromones.template.json`
   - Target: `.aether/data/pheromones.json`
   - If missing: copy template, remove `_` prefixed keys

2. **midden.json** - Failure signal tracking
   - Template: `midden.template.json`
   - Target: `.aether/data/midden/midden.json`
   - If missing: copy template, remove `_` prefixed keys

3. **learning-observations.json** - Pattern observation tracking
   - Template: `learning-observations.template.json`
   - Target: `.aether/data/learning-observations.json`
   - If missing: copy template, remove `_` prefixed keys

Run using Bash tool:
```bash
for template in pheromones midden learning-observations; do
  if [[ "$template" == "midden" ]]; then
    target=".aether/data/midden/midden.json"
  else
    target=".aether/data/${template}.json"
  fi
  if [[ ! -f "$target" ]]; then
    template_file=""
    for path in ~/.aether/system/templates/${template}.template.json .aether/templates/${template}.template.json; do
      if [[ -f "$path" ]]; then
        template_file="$path"
        break
      fi
    done
    if [[ -n "$template_file" ]]; then
      jq 'with_entries(select(.key | startswith("_") | not))' "$template_file" > "$target" 2>/dev/null || true
    fi
  fi
done
```

This step is non-blocking — proceed regardless of outcome.

### Step 5: Initialize Context Document

Run using Bash tool:
```
bash .aether/aether-utils.sh context-update init "$ARGUMENTS"
```

This creates `.aether/CONTEXT.md` — the colony's persistent memory. If context collapses, this file tells the next session what we were doing.

### Step 6: Validate State File

Use the Bash tool to run:
```
bash .aether/aether-utils.sh validate-state colony
```

This validates COLONY_STATE.json structure. If validation fails, output a warning.

### Step 6.5: Detect Nestmates

Run using Bash tool: `node -e "const nl = require('./bin/lib/nestmate-loader'); console.log(JSON.stringify(nl.findNestmates(process.cwd())))"`

If nestmates are found:
1. Display: `Nestmates found: N related colonies`
2. List each nestmate with name and truncated goal
3. Check for shared TO-DOs or cross-project dependencies

### Step 6.6: Register Repo (Silent)

Attempt to register this repo in the global hub. Both steps are silent on failure — registry is not required for the colony to work.

Run using the Bash tool (ignore errors):
```
bash .aether/aether-utils.sh registry-add "$(pwd)" "$(jq -r '.version // "unknown"' ~/.aether/version.json 2>/dev/null || echo 'unknown')" 2>/dev/null || true
```

Then attempt to write `.aether/version.json` with the hub version:
```
cp ~/.aether/version.json .aether/version.json 2>/dev/null || true
```

If either command fails, proceed silently. These are optional bookkeeping.

### Step 7: Display Result

Output this header:

```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
   A E T H E R   C O L O N Y
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
```

Then output the result:

```
👑 Queen has set the colony's intention

   "{goal}"

🏠 Colony Status: READY

{If instincts or learnings were inherited from Step 2.6:}
🧠 Inherited from prior colony:
   {N} instinct(s) | {N} learning(s)
{End if}

{If nestmates found in Step 5.5:}
🏘️ Nest Context: {N} sibling colonies detected
   Context from related projects will be automatically considered
   during planning and execution.
{End if}

💾 State persisted — safe to /clear, then run /ant:plan

📋 Context document created at `.aether/CONTEXT.md` — read this if session resets

──────────────────────────────────────────────────
🐜 Next Up
──────────────────────────────────────────────────
   /ant:plan                 📋 Create execution plan
   /ant:status               📊 Check colony state
   /ant:focus                🎯 Set initial focus
```

### Step 8: Initialize Session

Initialize session tracking to enable `/ant:resume` after context clear:

```bash
bash .aether/aether-utils.sh session-init "$(jq -r '.session_id' .aether/data/COLONY_STATE.json)" "$ARGUMENTS"
```
