---
name: ant:init
description: "Initialize Aether colony - scan repo, approve charter, create colony"
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

Run using the Bash tool:
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

If `scan_result` is empty or `jq` fails, set all fields to fallback values and proceed (graceful degradation -- never stop init because scan fails).

### Step 4: Detect Re-Init Mode

Use Read tool to check `.aether/data/COLONY_STATE.json`.

- If file exists AND has a non-null `goal` field: set `reinit_mode = true`, store `existing_goal`
- Otherwise: set `reinit_mode = false`

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
------------------------------------------------------
   A E T H E R   C O L O N Y   I N I T
------------------------------------------------------
```

If re-init mode, display:
```
Re-init mode detected (existing goal: "{existing_goal}")
Charter will be updated. All colony state, wisdom, instincts, and progress will be preserved.
```

Then display the approval prompt as formatted Markdown with three sections:

**Section 1: Charter**
```markdown
## Charter

**Intent:** {user's goal from $normalized_args, or existing_intent if re-init}
**Vision:** {derived from user's goal by Claude, or existing_vision if re-init}
**Governance:** {blank for fresh init, or existing_governance if re-init}
**Goals:** {blank for fresh init, or existing_goals if re-init}
```

For fresh init, Claude should derive a brief Vision from the user's goal (1-2 sentences). Governance and Goals start blank. The user fills them in if desired.

For re-init, pre-populate all fields from existing QUEEN.md charter entries.

**Section 2: Context**
```markdown
## Context

**Tech Stack:** {tech_langs} | {tech_fwks} | {tech_pkg}
**Project Size:** {complexity} ({file_count} files)
**Structure:** {top_dirs}
**Git:** {commit_count} commits
{if survey_suggestion: **Note:** {survey_suggestion}}
```

**Section 3: Pheromones**
```markdown
## Pheromones

No pheromone suggestions yet -- use /ant:focus and /ant:redirect to guide the colony.
```

End with clear instructions:
```
--------------------------------------------------
Review the prompt above. You can:
  - Edit any section (just describe your changes)
  - Say "approve" or "looks good" to proceed
  - Say "cancel" to abort

If you don't respond, the colony will not be initialized.
--------------------------------------------------
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

2. Optionally update the goal field in COLONY_STATE.json in-place:
```bash
jq --arg new_goal "{approved_intent}" '.goal = $new_goal' .aether/data/COLONY_STATE.json > .aether/data/COLONY_STATE.json.tmp && mv .aether/data/COLONY_STATE.json.tmp .aether/data/COLONY_STATE.json
```

3. Run `bash .aether/aether-utils.sh session-init "$(jq -r '.session_id' .aether/data/COLONY_STATE.json)" "{approved_intent}"`

4. Skip to Step 8 (display result). Do NOT write COLONY_STATE.json from template, do NOT write constraints.json, do NOT write pheromones.json.

**If fresh init:**

1. Initialize QUEEN.md (already done in Step 2)
2. Write charter content via charter-write (same command as above)
3. Write COLONY_STATE.json from template:
   - Generate a session ID in the format `session_{unix_timestamp}_{random}` and an ISO-8601 UTC timestamp
   - Resolve template: check `~/.aether/system/templates/colony-state.template.json` first, then `.aether/templates/colony-state.template.json`
   - If no template found: output "Template missing: colony-state.template.json. Run aether update to fix." and stop
   - Read the template file. Follow its `_instructions` field
   - Replace placeholders: `__GOAL__` with approved intent, `__SESSION_ID__` with generated session ID, `__ISO8601_TIMESTAMP__` with current timestamp, `__PHASE_LEARNINGS__` with `[]`, `__INSTINCTS__` with `[]`
   - Remove ALL keys starting with underscore
   - Write the resulting JSON to `.aether/data/COLONY_STATE.json` using the Write tool

4. Write constraints.json from template:
   - Resolve template: check `~/.aether/system/templates/constraints.template.json` first, then `.aether/templates/constraints.template.json`
   - If no template found: output "Template missing: constraints.template.json. Run aether update to fix." and stop
   - Read template, follow `_instructions`, remove `_` prefixed keys, write to `.aether/data/constraints.json`

5. Initialize runtime files from templates (non-blocking):
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

6. Run `bash .aether/aether-utils.sh context-update init "{approved_intent}"`
7. Run `bash .aether/aether-utils.sh validate-state colony`
8. Register repo (silent on failure):
```bash
domain_tags=$(bash .aether/aether-utils.sh domain-detect 2>/dev/null | jq -r '.result.tags // ""' || echo "")
bash .aether/aether-utils.sh registry-add "$(pwd)" "$(jq -r '.version // "unknown"' ~/.aether/version.json 2>/dev/null || echo 'unknown')" --goal "{approved_intent}" --active true --tags "$domain_tags" 2>/dev/null || true
cp ~/.aether/version.json .aether/version.json 2>/dev/null || true
```
9. Seed QUEEN.md from hive (non-blocking):
```bash
domain_tags=$(jq -r --arg repo "$(pwd)" \
  '[.repos[] | select(.path == $repo) | .domain_tags // []] | .[0] // [] | join(",")' \
  "$HOME/.aether/registry.json" 2>/dev/null || echo "")
seed_args="queen-seed-from-hive --limit 5"
[[ -n "$domain_tags" ]] && seed_args="$seed_args --domain $domain_tags"
seed_result=$(bash .aether/aether-utils.sh $seed_args 2>/dev/null || echo '{}')
seeded_count=$(echo "$seed_result" | jq -r '.result.seeded // 0' 2>/dev/null || echo "0")
```
10. Run `bash .aether/aether-utils.sh session-init "{session_id}" "{approved_intent}"`

### Step 8: Display Result

Display the success header and result block:

```
------------------------------------------------------
   A E T H E R   C O L O N Y
------------------------------------------------------

Queen has set the colony's intention

   "{approved_intent}"

   Colony Status: READY

{If re-init: "   Mode: Re-init (charter updated, state preserved)"}
{If fresh and seeded_count > 0: "   Hive wisdom: {seeded_count} cross-colony pattern(s) seeded into QUEEN.md"}

State persisted -- safe to /clear, then run /ant:plan

--------------------------------------------------
   Next Up
--------------------------------------------------
   /ant:plan                 Generate execution plan
   /ant:status               Check colony state
   /ant:focus                Set initial focus
```
