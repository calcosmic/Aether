<!-- Generated from .aether/commands/init.yaml - DO NOT EDIT DIRECTLY -->
---
name: ant:init
description: "Initialize Aether colony - scan repo, approve charter, create colony"
---

### Step -1: Normalize Arguments

Run: `normalized_args=$(bash .aether/aether-utils.sh normalize-args "$@")`

This ensures arguments work correctly in both Claude Code and OpenCode. Use `$normalized_args` throughout this command.

You are the **Queen Ant Colony**. Initialize the colony with the Queen's intention.

## Instructions

The user's goal is: `$normalized_args`

Parse `$normalized_args`:
- If contains `--no-visual`: set `visual_mode = false` (visual is ON by default)
- Otherwise: set `visual_mode = true`


Note: Use `$normalized_args` instead of `$ARGUMENTS` throughout this command.

### Step 0: Initialize Visual Mode (if enabled)

If `visual_mode` is true:


<failure_modes>
### Colony State Overwrite
Re-init mode detects existing COLONY_STATE.json and preserves all state. Charter content is updated in-place via charter-write. Colony state, wisdom, instincts, learnings, pheromones, and phase progress are never reset.

### Write Failure Mid-Init
If writing COLONY_STATE.json fails partway:
- Remove the incomplete file (partial state is worse than no state)
- Report the error
- Recovery: user can run /ant:init again safely
</failure_modes>

<success_criteria>
Command is complete when:
- User has approved the charter prompt (Charter, Context, Pheromones sections)
- Charter content is written to QUEEN.md via charter-write
- COLONY_STATE.json exists and is valid JSON (fresh init only)
- Session file is written
- User sees confirmation of colony creation or re-init
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

If `$normalized_args` is empty or blank, output:

```
Aether Colony

  Initialize the colony with a goal. This scans the repo, generates
  a charter for your approval, then creates colony files.

  Usage: /ant:init "<your goal here>"

  Examples:
    /ant:init "Build a REST API with authentication"
    /ant:init "Create a soothing sound application"
    /ant:init "Design a calculator CLI tool"
```

Stop here. Do not proceed.

### Step 1.5: Verify Aether Setup

Check if `.aether/aether-utils.sh` exists using the Read tool.

**If the file already exists** -- skip this step entirely. Aether is set up.

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

### Step 2: Initialize QUEEN.md

Run:
```
bash .aether/aether-utils.sh queen-init
```

Parse the JSON result:
- If `created` is true: Display `QUEEN.md initialized`
- If `created` is false and `reason` is "already_exists": Display `QUEEN.md already exists`

This step is non-blocking -- proceed regardless of outcome.

### Step 3: Scan Repository

Run the scan via Bash tool:
```bash
scan_result=$(bash .aether/aether-utils.sh init-research 2>/dev/null)
scan_data=$(echo "$scan_result" | jq '.result')
```

Extract fields with jq defaults for missing data:
- `tech_langs`: `.tech_stack.languages | if length > 0 then join(", ") else "not detected" end`
- `tech_fwks`: `.tech_stack.frameworks | if length > 0 then join(", ") else "none" end`
- `tech_pkg`: `.tech_stack.package_managers | join(", ")`
- `complexity`: `.complexity.size`
- `file_count`: `.complexity.metrics.file_count`
- `top_dirs`: `.directory_structure.top_level_dirs | if . and length > 0 then join(", ") else "flat" end`
- `commit_count`: `.git_history.commit_count // "unknown"`
- `is_git`: `.git_history.is_git_repo // false`
- `survey_suggestion`: `.survey_status.suggestion.reason // empty`
- `has_active`: `.prior_colonies.has_active_colony // false`
- `active_goal`: `.prior_colonies.active_goal // empty`

**Intelligence fields (new):**
- `colony_context_colonies`: `.colony_context.prior_colonies // []` -- array of prior colony summaries (each has goal, phases, outcome, summary)
- `colony_context_charter`: `.colony_context.existing_charter // {}` -- existing charter content from QUEEN.md
- `governance_rules`: `.governance.rules // []` -- array of governance rule objects (each has rule, source, strength)
- `pheromone_suggestions`: `.pheromone_suggestions // []` -- array of suggestion objects (each has type, content, reason, priority)

If `scan_result` is empty or `jq` fails, set all fields to fallback values (empty arrays/objects for intelligence fields) and proceed (graceful degradation -- never stop init because scan fails).

### Step 4: Detect Re-Init Mode

Use Read tool to check `.aether/data/COLONY_STATE.json`.

- If file exists AND has a non-null `goal` field:
  - Check the `milestone` field. If `milestone == "Crowned Anthill"`:
    - This is a **sealed colony**. Treat as **fresh init**, NOT re-init.
    - Set `reinit_mode = false`
    - Display: `Previous colony was sealed. Starting fresh colony.`
    - The old COLONY_STATE.json will be overwritten in Step 7 (fresh init path).
  - Otherwise (colony exists but is NOT sealed): set `reinit_mode = true`, store `existing_goal`
- If file does not exist or `goal` is null: set `reinit_mode = false`

If re-init mode, read existing charter entries from `.aether/QUEEN.md`:
```bash
existing_intent=$(grep '\[charter\] \*\*Intent\*\*:' .aether/QUEEN.md 2>/dev/null | sed 's/.*\*\*Intent\*\*: //' | sed 's/ (Colony:.*//' || true)
existing_vision=$(grep '\[charter\] \*\*Vision\*\*:' .aether/QUEEN.md 2>/dev/null | sed 's/.*\*\*Vision\*\*: //' | sed 's/ (Colony:.*//' || true)
existing_governance=$(grep '\[charter\] \*\*Governance\*\*:' .aether/QUEEN.md 2>/dev/null | sed 's/.*\*\*Governance\*\*: //' | sed 's/ (Colony:.*//' || true)
existing_goals=$(grep '\[charter\] \*\*Goal\*\*:' .aether/QUEEN.md 2>/dev/null | sed 's/.*\*\*Goal\*\*: //' | sed 's/ (Colony:.*//' || true)
```

Strip `(Colony: ...)` suffixes using sed. If grep finds nothing, variables remain empty.

### Step 5: Assemble and Display Approval Prompt

Display a brief header:
```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
   A E T H E R   C O L O N Y   I N I T
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
```

If re-init mode, display:
```
Re-init mode detected (existing goal: "{existing_goal}")
Charter will be updated. All colony state, wisdom, instincts, and progress will be preserved.
```

Then display the approval prompt as formatted Markdown. Section ordering: Prior Context (if any) -> Charter -> Context -> Pheromones.

**Section 1: Prior Context (conditional -- only when prior colonies exist)**

If `colony_context_colonies` has entries (length > 0), display:
```markdown
## Prior Context

Previous colonies in this repo:

{For each colony (max 3, most recent first):}
- **{goal}** -- {outcome} ({phases} phases){if summary is non-empty: ". {summary}"}
```

Per locked decision: when no prior colonies exist, omit this section entirely. No placeholder, no header.

Keep each colony to 1-2 lines. Show goal, outcome (milestone), phase count, and summary from CROWNED-ANTHILL.md if available.

**Section 2: Charter**
```markdown
## Charter

**Intent:** {user's goal from $normalized_args, or existing_intent if re-init}
**Vision:** {derived from user's goal by Claude, or existing_vision if re-init}
**Governance:** {see governance logic below}
**Goals:** {blank for fresh init, or existing_goals if re-init}
```

For fresh init, Claude should derive a brief Vision from the user's goal (1-2 sentences). Goals start blank. The user fills them in if desired.

**Governance field logic:**
- For fresh init with `governance_rules` available (length > 0): pre-populate with semicolon-separated rule text from the detected rules. Format: `"TDD required; ESLint enforced; Follow CONTRIBUTING.md"`. These are editable by the user.
- For fresh init with no governance_rules: leave blank.
- For re-init with existing_governance non-empty: pre-populate from existing QUEEN.md charter entries.
- For re-init with existing_governance empty but governance_rules available: pre-populate from governance_rules.

For re-init, pre-populate Intent, Vision, and Goals from existing QUEEN.md charter entries.

**Section 3: Context**
```markdown
## Context

**Tech Stack:** {tech_langs} | {tech_fwks} | {tech_pkg}
**Project Size:** {complexity} ({file_count} files)
**Structure:** {top_dirs}
**Git:** {commit_count} commits
{if survey_suggestion: **Note:** {survey_suggestion}}
```

**Section 4: Pheromones**

If `pheromone_suggestions` has entries (length > 0), display:
```markdown
## Pheromones

Suggested signals based on repo analysis:

1. [FOCUS] Testing infrastructure present (47 test files) -- maintain TDD discipline
2. [REDIRECT] Environment files detected -- never commit secrets or .env files
3. [FOCUS] Code quality tools configured -- follow existing lint/format rules

Edit, remove, or add signals as needed. Approved signals will be auto-applied.
```

The numbered list uses the actual type and content from `pheromone_suggestions`. Each line format: `{N}. [{type}] {content}`.

Per locked decision: suggestions are fully editable. User can reword, remove, or add their own.
Per locked decision: all sections look the same -- no visual distinction between auto-generated and user-written content.

If no pheromone suggestions available (empty array), display the existing default:
```markdown
## Pheromones

No pheromone suggestions yet -- use /ant:focus and /ant:redirect to guide the colony.
```

End with clear instructions:
```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Review the prompt above. You can:
  - Edit any section (just describe your changes)
  - Say "approve" or "looks good" to proceed
  - Say "cancel" to abort

If you don't respond, the colony will not be initialized.
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
```

**STOP HERE.** Wait for the user's response. Do NOT proceed to Step 6 until the user responds.

### Step 6: Handle User Response

Parse the user's response:
- If the user approves (says "approve", "looks good", "yes", "ok", or similar): proceed to Step 7
- If the user provides edits: apply the edits to the relevant section(s), re-display the full prompt, increment a revision counter, and wait again
- If the user cancels: display "Colony initialization cancelled." and stop
- Max 2 revision rounds. After 2 rejections/edits, display: "Maximum revisions reached. Approve current version, or cancel init?" and wait for final decision

When applying edits, Claude updates the section content in memory (not files) and re-displays the full prompt. Each re-display includes a revision counter: "(Revision {N}/2)"

### Step 7: Create Colony (Post-Approval)

Only reached after user approval. ALL file writes happen here.

**If re-init mode:**

1. Write charter content via:
```bash
bash .aether/aether-utils.sh charter-write --intent "{approved_intent}" --vision "{approved_vision}" --governance "{approved_governance}" --goals "{approved_goals}"
```

2. Auto-apply approved pheromone suggestions (see pheromone auto-apply below).

3. Optionally update the goal field in COLONY_STATE.json in-place:
```bash
jq --arg new_goal "{approved_intent}" '.goal = $new_goal' .aether/data/COLONY_STATE.json > .aether/data/COLONY_STATE.json.tmp && mv .aether/data/COLONY_STATE.json.tmp .aether/data/COLONY_STATE.json
```

4. Run `bash .aether/aether-utils.sh session-init "$(jq -r '.session_id' .aether/data/COLONY_STATE.json)" "{approved_intent}"`

5. Skip to Step 8 (display result). Do NOT write COLONY_STATE.json from template, do NOT write constraints.json, do NOT write pheromones.json.

**If fresh init:**

1. Initialize QUEEN.md (already done in Step 2)
2. Write charter content via charter-write (same command as above)
3. Auto-apply approved pheromone suggestions (see pheromone auto-apply below).
4. Write COLONY_STATE.json from template:
   - Generate a session ID in the format `session_{unix_timestamp}_{random}` and an ISO-8601 UTC timestamp
   - Resolve template: check `~/.aether/system/templates/colony-state.template.json` first, then `.aether/templates/colony-state.template.json`
   - If no template found: output "Template missing: colony-state.template.json. Run aether update to fix." and stop
   - Read the template file. Follow its `_instructions` field
   - Replace placeholders: `__GOAL__` with approved intent, `__SESSION_ID__` with generated session ID, `__ISO8601_TIMESTAMP__` with current timestamp, `__PHASE_LEARNINGS__` with `[]`, `__INSTINCTS__` with `[]`
   - Remove ALL keys starting with underscore
   - Write the resulting JSON to `.aether/data/COLONY_STATE.json` using the Write tool

5. Write constraints.json from template:
   - Resolve template: check `~/.aether/system/templates/constraints.template.json` first, then `.aether/templates/constraints.template.json`
   - If no template found: output "Template missing: constraints.template.json. Run aether update to fix." and stop
   - Read template, follow `_instructions`, remove `_` prefixed keys, write to `.aether/data/constraints.json`

6. Initialize runtime files from templates (non-blocking):
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

7. Run `bash .aether/aether-utils.sh context-update init "{approved_intent}"`
8. Run `bash .aether/aether-utils.sh validate-state colony`
9. Register repo (silent on failure):
```bash
domain_tags=$(bash .aether/aether-utils.sh domain-detect 2>/dev/null | jq -r '.result.tags // ""' || echo "")
bash .aether/aether-utils.sh registry-add "$(pwd)" "$(jq -r '.version // "unknown"' ~/.aether/version.json 2>/dev/null || echo 'unknown')" --goal "{approved_intent}" --active true --tags "$domain_tags" 2>/dev/null || true
cp ~/.aether/version.json .aether/version.json 2>/dev/null || true
```
10. Seed QUEEN.md from hive (non-blocking):
```bash
domain_tags=$(jq -r --arg repo "$(pwd)" \
  '[.repos[] | select(.path == $repo) | .domain_tags // []] | .[0] // [] | join(",")' \
  "$HOME/.aether/registry.json" 2>/dev/null || echo "")
seed_args="queen-seed-from-hive --limit 5"
[[ -n "$domain_tags" ]] && seed_args="$seed_args --domain $domain_tags"
seed_result=$(bash .aether/aether-utils.sh $seed_args 2>/dev/null || echo '{}')
seeded_count=$(echo "$seed_result" | jq -r '.result.seeded // 0' 2>/dev/null || echo "0")
```
11. Run `bash .aether/aether-utils.sh session-init "{session_id}" "{approved_intent}"`

**Pheromone auto-apply (referenced by both re-init and fresh init paths above):**

If approved pheromone suggestions exist (the user kept them in the prompt and didn't remove them during the approval loop):

For each approved pheromone suggestion, call:
```bash
bash .aether/aether-utils.sh pheromone-write "{type}" '{content}' --source "system:init" --reason '{reason}' --ttl "30d" 2>/dev/null || true
```

Implementation notes:
- Claude (the LLM executing init.md) tracks which pheromones the user kept, edited, or removed during the approval loop (Step 6). Only emit pheromones that survived approval.
- Use single quotes around pheromone content and reason to avoid shell metacharacter issues (per pitfall 4).
- Each `pheromone-write` call uses `2>/dev/null || true` to make it non-blocking -- a failed write should never stop colony creation.
- The `--source "system:init"` tag identifies these as init-derived pheromones.
- The `--ttl "30d"` gives suggestions a 30-day lifespan (project-level conventions, not phase-specific).
- `pheromone-write` handles deduplication via content hashing -- if a signal with the same content already exists, it will reinforce rather than duplicate.

### Step 7.5: Import Previous Colony Data (optional)

Check if previous colony chambers contain importable XML data:

```bash
# Find most recent chamber with XML files (per D-07)
latest_chamber=$(ls -d .aether/chambers/20* 2>/dev/null | sort -r | head -1)
xml_import_available=false
import_summary=""

if [[ -n "$latest_chamber" ]]; then
  xml_count=$(find "$latest_chamber" -maxdepth 1 -name "*.xml" ! -name "colony-archive.xml" 2>/dev/null | wc -l | tr -d ' ')
  if [[ "$xml_count" -gt 0 ]] && command -v xmllint >/dev/null 2>&1; then
    xml_import_available=true
    chamber_name=$(basename "$latest_chamber")
    # Count importable items for display
    signal_count=$(jq '.signals | length' "$latest_chamber/pheromones.json" 2>/dev/null || echo "0")
    import_summary="Found ${signal_count} signal(s) and ${xml_count} XML file(s) from colony '${chamber_name}'"
  fi
fi
```

**If xml_import_available is true:**

Display the import offer (per D-08):
```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
   PREVIOUS COLONY DATA FOUND
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

{import_summary}

Import signals and wisdom from this colony?
This will add to (not replace) your current colony data.

Import? (yes/no)
```

Use `AskUserQuestion` with yes/no options.

**If user selects "yes":**

Import ALL available data types (per D-09 -- no cherry-picking):

```bash
# Import pheromones (per D-09)
if [[ -f "$latest_chamber/pheromones.xml" ]]; then
  pher_import=$(bash .aether/aether-utils.sh pheromone-import-xml "$latest_chamber/pheromones.xml" "imported" 2>/dev/null || echo '{"ok":false}')
  pher_imported=$(echo "$pher_import" | jq -r '.result.imported // 0' 2>/dev/null || echo "0")
  echo "Pheromones: ${pher_imported} signal(s) imported"
fi

# Import wisdom to queen-wisdom.json (per D-09)
if [[ -f "$latest_chamber/queen-wisdom.xml" ]]; then
  wis_import=$(bash .aether/aether-utils.sh wisdom-import-xml "$latest_chamber/queen-wisdom.xml" ".aether/data/queen-wisdom.json" 2>/dev/null || echo '{"ok":false}')
  wis_imported=$(echo "$wis_import" | jq -r '.result.imported // 0' 2>/dev/null || echo "0")
  echo "Wisdom: ${wis_imported} entries(s) imported to queen-wisdom.json"
fi

# Import registry lineage (per D-09)
if [[ -f "$latest_chamber/colony-registry.xml" ]]; then
  reg_import=$(bash .aether/aether-utils.sh registry-import-xml "$latest_chamber/colony-registry.xml" 2>/dev/null || echo '{"ok":false}')
  reg_imported=$(echo "$reg_import" | jq -r '.result.imported // 0' 2>/dev/null || echo "0")
  echo "Registry: ${reg_imported} colon(ies) lineage imported"
fi
```

All imports are non-blocking -- log warning and continue if any fails.

**If user selects "no":**

Display "Import skipped. Starting fresh colony." and proceed to Step 8.

**If xml_import_available is false (no chambers, no XML, or no xmllint):**

Skip silently -- proceed directly to Step 8 without any mention of import (per D-11).
### Step 8: Display Result

Display the success header and result block:

```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
   A E T H E R   C O L O N Y
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Queen has set the colony's intention

   "{approved_intent}"

   Colony Status: READY

{If re-init: "   Mode: Re-init (charter updated, state preserved)"}
{If fresh and seeded_count > 0: "   Hive wisdom: {seeded_count} cross-colony pattern(s) seeded into QUEEN.md"}

State persisted -- safe to /clear, then run /ant:plan

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
   Next Up
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
   /ant:plan                 Generate execution plan
   /ant:status               Check colony state
   /ant:focus                Set initial focus
```
