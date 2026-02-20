---
name: ant:init
description: "🌱🐜🆕🐜🌱 Initialize Aether colony - Queen sets intention, colony mobilizes"
---

You are the **Queen Ant Colony**. Initialize the colony with the Queen's intention.

## Instructions

### Step -1: Normalize Arguments

Run: `normalized_args=$(bash .aether/aether-utils.sh normalize-args "$@")`

This ensures arguments work correctly in both Claude Code and OpenCode.

The user's goal is: `$normalized_args`

Parse `$normalized_args`:
- If contains `--no-visual`: set `visual_mode = false` (visual is ON by default)
- Otherwise: set `visual_mode = true`

Note: Use `$normalized_args` instead of `$ARGUMENTS` throughout this command.

### Step 0: Initialize Visual Mode (if enabled)

If `visual_mode` is true:
```bash
# Generate session ID
init_id="init-$(date +%s)"

# Initialize swarm display
bash .aether/aether-utils.sh swarm-display-init "$init_id"
bash .aether/aether-utils.sh swarm-display-update "Queen" "prime" "excavating" "Colony initialization" "Colony" '{"read":0,"grep":0,"edit":0,"bash":0}' 0 "fungus_garden" 0
```

### Step 0.5: Version Check (Non-blocking)

Run using the Bash tool: `bash .aether/aether-utils.sh version-check 2>/dev/null || true`

If the command succeeds and the JSON result contains a non-empty string, display it as a one-line notice. Proceed regardless of outcome.

### Step 1: Validate Input

If `$normalized_args` is empty or blank, output:

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

### Step 1.5: Bootstrap System Files (Conditional)

Check if `.aether/aether-utils.sh` exists using the Read tool.

**If the file already exists** — skip this step entirely. System files are present.

**If the file does NOT exist:**
- Check if `~/.aether/system/aether-utils.sh` exists (expand `~` to the user's home directory)
- **If the hub exists:** Run using the Bash tool:
  ```bash
  mkdir -p .aether/docs .aether/utils .aether/templates .aether/schemas .aether/exchange .claude/rules && \
  cp -f ~/.aether/system/aether-utils.sh .aether/ && \
  cp -f ~/.aether/system/workers.md .aether/ 2>/dev/null || true && \
  cp -f ~/.aether/system/CONTEXT.md .aether/ 2>/dev/null || true && \
  cp -f ~/.aether/system/model-profiles.yaml .aether/ 2>/dev/null || true && \
  cp -Rf ~/.aether/system/docs/* .aether/docs/ 2>/dev/null || true && \
  cp -Rf ~/.aether/system/utils/* .aether/utils/ 2>/dev/null || true && \
  cp -Rf ~/.aether/system/templates/* .aether/templates/ 2>/dev/null || true && \
  cp -Rf ~/.aether/system/schemas/* .aether/schemas/ 2>/dev/null || true && \
  cp -Rf ~/.aether/system/exchange/* .aether/exchange/ 2>/dev/null || true && \
  cp -Rf ~/.aether/system/rules/* .claude/rules/ 2>/dev/null || true && \
  chmod +x .aether/aether-utils.sh
  ```
  This copies system files from the global hub into `.aether/` and rules into `.claude/rules/`. Display:
  ```
  Bootstrapped system files from global hub.
  ```
- **If the hub does NOT exist:** Output:
  ```
  No Aether system files found locally or in ~/.aether/system/.
  Run `aether install` or `npx aether-colony install` to set up the global hub first.
  ```
  Stop here. Do not proceed.

### Step 2: Read Current State

Use the Read tool to read `.aether/data/COLONY_STATE.json`.

If the `goal` field is not null, output:

```
Colony already initialized with goal: "{existing_goal}"

To reinitialize with a new goal, the current state will be reset.
Proceeding with new goal: "{new_goal}"
```

### Step 2.5: Load Prior Colony Knowledge (Optional)

Check if `.aether/data/completion-report.md` exists using the Read tool.

**If the file does NOT exist**, skip to Step 3 — this is a fresh colony with no prior history.

**If the file exists**, read it and extract:
1. **Instincts** — look for the `## Colony Instincts` section. Each line has format: `N. [confidence] domain: description`. Keep only instincts with confidence >= 0.5.
2. **Learnings** — look for the `## Colony Learnings (Validated)` section. Keep all numbered items.

Store the extracted instincts and learnings for use in Step 3. Display a brief note:

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
  - `__GOAL__` → the user's goal from $normalized_args
  - `__SESSION_ID__` → the generated session ID (format: `session_{unix_timestamp}_{random}`)
  - `__ISO8601_TIMESTAMP__` → the current ISO-8601 UTC timestamp (used in both `initialized_at` and the events entry)
  - `__PHASE_LEARNINGS__` → JSON array from Step 2.5, or `[]` if none
  - `__INSTINCTS__` → JSON array from Step 2.5, or `[]` if none

IMPORTANT: `__PHASE_LEARNINGS__` and `__INSTINCTS__` must be JSON array values (e.g., `[]` not `"[]"`).

**If Step 2.5 found instincts to inherit**, convert each into the instinct format for the `__INSTINCTS__` array. Each inherited instinct should have:
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

**If Step 2.5 found validated learnings**, seed the `__PHASE_LEARNINGS__` array with each as:
- `phase`: `"inherited"`
- `learning`: the learning text
- `status`: `"validated"`
- `source`: `"inherited:completion-report"`

**If Step 2.5 was skipped or found nothing**, use empty arrays `[]` for both `__PHASE_LEARNINGS__` and `__INSTINCTS__`.

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

### Step 5: Validate State File

Use the Bash tool to run:
```
bash .aether/aether-utils.sh validate-state colony
```

This validates COLONY_STATE.json structure. If validation fails, output a warning.

### Step 5.5: Detect Nestmates

Run using Bash tool: `node -e "const nl = require('./bin/lib/nestmate-loader'); console.log(JSON.stringify(nl.findNestmates(process.cwd())))"`

If nestmates are found:
1. Display: `Nestmates found: N related colonies`
2. List each nestmate with name and truncated goal
3. Check for shared TO-DOs or cross-project dependencies

### Step 5.6: Register Repo (Silent)

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

### Step 6: Display Result

**If visual_mode is true, render final swarm display:**
```bash
bash .aether/aether-utils.sh swarm-display-update "Queen" "prime" "completed" "Colony initialized" "Colony" '{"read":5,"grep":2,"edit":3,"bash":2}' 100 "fungus_garden" 100
bash .aether/aether-utils.sh swarm-display-render "$init_id"
```

Output this header:

```
🌱🐜🆕🐜🌱 ═══════════════════════════════════════════════════
   A E T H E R   C O L O N Y
═══════════════════════════════════════════════════ 🌱🐜🆕🐜🌱
```

Then output the result:

```
👑 Queen has set the colony's intention

   "{goal}"

🏠 Colony Status: READY
📋 Session: <session_id>

{If instincts or learnings were inherited from Step 2.5:}
🧠 Inherited from prior colony:
   {N} instinct(s) | {N} learning(s)
{End if}

{If nestmates found in Step 5.5:}
🏘️ Nest Context: {N} sibling colonies detected
   Context from related projects will be automatically considered
   during planning and execution.
{End if}

🐜 The colony awaits your command:

   /ant:plan      📋 Generate project plan
   /ant:colonize  🗺️  Analyze existing codebase first
   /ant:watch     👁️  Set up live visibility

💾 State persisted — safe to /clear, then run /ant:plan
```
