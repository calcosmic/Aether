<!-- Generated from .aether/commands/init.yaml - DO NOT EDIT DIRECTLY -->
---
name: ant:init
description: "Initialize Aether colony - scan repo, approve charter, create colony"
---

You are the **Queen Ant Colony**. Initialize the colony with the Queen's intention.

## Instructions

The user's goal is: `$ARGUMENTS`

Parse `$ARGUMENTS`:
- If contains `--no-visual`: set `visual_mode = false` (visual is ON by default)
- Otherwise: set `visual_mode = true`



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
- .github/workflows/
</read_only>

### Step 1: Validate Input

If `$ARGUMENTS` is empty or blank, output:

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

Check if the `aether` binary is available by running `aether version` using the Bash tool.

**If the command succeeds** -- skip this step entirely. Aether is set up.

**If the command fails:**
```
Aether is not set up in this repo yet.

Run `aether setup` first to create the .aether/ directory
with all system files, then run /ant:init "your goal" to
start a colony.

If the global hub isn't installed either:
  npm install -g aether-colony   (installs the hub)
  /ant:lay-eggs                  (sets up this repo)
  /ant:init "your goal"          (starts the colony)
```
Stop here. Do not proceed.

### Step 2: Initialize QUEEN.md

Run using the Bash tool with description "Initializing QUEEN.md...":
```
aether queen-init
```

Parse the JSON result:
- If `created` is true: Display `QUEEN.md initialized`
- If `created` is false and `reason` is "already_exists": Display `QUEEN.md already exists`

This step is non-blocking -- proceed regardless of outcome.

### Step 3: Scan Repository

Run the scan via Bash tool:
```bash
scan_result=$(aether init-research 2>/dev/null)
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

### Step 4.5: Deep Analysis

Spawn research and design agents to produce a comprehensive analysis for the colony charter. This analysis enriches the approval prompt and is persisted for later use by /ant:plan.

**Complexity gate:** If `file_count` from Step 3 is 0 (empty repo) and `is_git` is false, skip this step entirely. Set `deep_analysis = null` and proceed to Step 5.

**If file_count > 0:**

Display: `Analyzing codebase for colony foundation...`

**1. Spawn Research Scout** via Task tool with `subagent_type="aether-scout"`:

FALLBACK: If "Agent type not found", use general-purpose and inject role: "You are a Scout Ant performing Init Analysis Research."

```
You are a Scout Ant performing Init Analysis Research.

--- MISSION ---
Research the codebase to understand its architecture and how it relates to the colony goal.

Goal: "{user_goal}"
Tech Stack: {tech_langs} | {tech_fwks} | {tech_pkg}
Project Size: {complexity} ({file_count} files)
Structure: {top_dirs}

--- RESEARCH AREAS ---
1. Core architecture: entry points, main modules, how components connect
2. Data flow: how data moves through the system, key data structures
3. Existing patterns: coding conventions, testing patterns, build patterns
4. Dependencies: critical external libraries, internal module dependencies
5. Relevance to goal: which parts of the codebase the goal would affect

--- SCOPE CONSTRAINTS ---
- Maximum 8 findings (prioritize by relevance to the goal)
- Maximum 2 sentences per finding
- Focus on structural/architectural findings, not line-level details
- If the goal mentions specific features, prioritize those areas

--- TOOLS ---
Use: Glob, Grep, Read, WebSearch, WebFetch
Do NOT use: Task, Write, Edit

--- OUTPUT FORMAT ---
Return JSON:
{
  "findings": [
    {"area": "...", "discovery": "...", "relevance": "why this matters for the goal", "source": "file or search"}
  ],
  "architecture_summary": "2-3 sentence overview of the system architecture",
  "key_components": ["component1", "component2"],
  "data_flow": "brief description of primary data flow",
  "goal_impact_areas": ["areas of the codebase the goal would touch"],
  "overall_assessment": "brief assessment of project complexity relative to the goal"
}
```

Wait for scout to complete (blocking). Parse the JSON output.

If scout fails or returns invalid JSON, set `scout_findings = null` and proceed (graceful degradation -- deep analysis is optional, not blocking).

**2. Spawn Architect Agent** via Task tool with `subagent_type="aether-architect"`:

FALLBACK: If "Agent type not found", use general-purpose and inject role: "You are an Architect Ant performing Init Architecture Design."

```
You are an Architect Ant performing Init Architecture Design.

--- MISSION ---
Produce a concise but comprehensive architecture analysis and implementation approach for the colony goal.

Goal: "{user_goal}"
Tech Stack: {tech_langs} | {tech_fwks} | {tech_pkg}
Project Size: {complexity} ({file_count} files)
Structure: {top_dirs}

--- SCOUT FINDINGS ---
{if scout_findings is not null:}
{format scout findings as compact bullet list}
Architecture: {architecture_summary from scout}
Key Components: {key_components from scout}
Data Flow: {data_flow from scout}
Goal Impact Areas: {goal_impact_areas from scout}
{else:}
No scout findings available. Analyze the codebase independently.
{end if}

--- DELIVERABLES ---
1. **Architecture Overview** -- Current system architecture (or proposed for greenfield). Name modules, describe connections, identify boundaries.

2. **Mermaid Diagrams** -- Produce 2-3 Mermaid diagrams:
   - System architecture diagram (component relationships)
   - Data flow diagram (how data moves through the system)
   - Optionally: component dependency diagram (if dependencies are complex)
   Keep diagrams under 20 nodes each for readability.

3. **Technical Approach** -- Recommended implementation strategy. High-level, not task-level. Name specific patterns, frameworks, or approaches.

4. **Risk Assessment** -- What could go wrong. Include technical risks, integration risks, and mitigation strategies.

5. **Key Decisions** -- 3-5 technical choices the colony should make. Present as questions with recommended answers.

6. **Phase Skeleton** -- 2-4 high-level phases (rough outline). Name only, with 1-sentence description each. NOT a full plan -- /ant:plan produces the detailed plan later.

--- OUTPUT CONSTRAINTS ---
- Total output: under 2000 words
- Be proportional: a simple goal gets a brief analysis; a complex goal gets a thorough one
- Mermaid diagrams should use simple syntax (flowchart, graph) for maximum compatibility
- If the codebase is empty or minimal, focus on greenfield recommendations

--- TOOLS ---
Use: Glob, Grep, Read, Bash
Do NOT use: Task (you cannot spawn subagents)

--- OUTPUT FORMAT ---
Return JSON:
{
  "architecture_overview": "...",
  "mermaid_diagrams": [
    {"title": "System Architecture", "code": "mermaid code here"},
    {"title": "Data Flow", "code": "mermaid code here"}
  ],
  "technical_approach": "...",
  "risk_assessment": [
    {"risk": "...", "severity": "high|medium|low", "mitigation": "..."}
  ],
  "key_decisions": [
    {"decision": "...", "recommendation": "...", "rationale": "..."}
  ],
  "phase_skeleton": [
    {"name": "...", "description": "..."}
  ]
}
```

Wait for architect to complete (blocking). Parse the JSON output.

If architect fails or returns invalid JSON, set `deep_analysis = null` and proceed.

Display: `Analysis complete.`

Store the architect's JSON output as `deep_analysis` for use in Step 5 (approval prompt) and Step 7 (persistence).

### Step 5: Assemble and Display Approval Prompt

Display a brief header:
```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
🥚 A E T H E R   C O L O N Y   I N I T
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
```

If re-init mode, display:
```
Re-init mode detected (existing goal: "{existing_goal}")
Charter will be updated. All colony state, wisdom, instincts, and progress will be preserved.
```

Then display the approval prompt as formatted Markdown. Section ordering: Prior Context (if any) -> Charter -> Context -> Architecture Analysis -> Technical Approach -> Risk Assessment -> Key Decisions -> Phase Skeleton -> Pheromones. Sections 4-8 are omitted when deep_analysis is null.

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

**Intent:** {user's goal from $ARGUMENTS, or existing_intent if re-init}
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

**Section 4: Architecture Analysis** (conditional -- only when `deep_analysis` is not null)

If `deep_analysis` is not null, display:
```markdown
## Architecture Analysis

{deep_analysis.architecture_overview}

{for each mermaid_diagram:}
### {title}
```mermaid
{code}
```

### Key Components
{for each component from scout_findings.key_components or deep_analysis:}
- {component}
```
If `deep_analysis` is null, omit this section entirely.

**Section 5: Technical Approach** (conditional -- only when `deep_analysis` is not null)

If `deep_analysis` is not null, display:
```markdown
## Technical Approach

{deep_analysis.technical_approach}
```
If `deep_analysis` is null, omit this section entirely.

**Section 6: Risk Assessment** (conditional -- only when `deep_analysis` is not null)

If `deep_analysis` is not null, display:
```markdown
## Risk Assessment

| Risk | Severity | Mitigation |
|------|----------|------------|
{for each risk in deep_analysis.risk_assessment:}
| {risk} | {severity} | {mitigation} |
```
If `deep_analysis` is null, omit this section entirely.

**Section 7: Key Decisions** (conditional -- only when `deep_analysis` is not null)

If `deep_analysis` is not null, display:
```markdown
## Key Decisions

{for each decision in deep_analysis.key_decisions:}
- **{decision}** -- Recommended: {recommendation} ({rationale})
```
If `deep_analysis` is null, omit this section entirely.

**Section 8: Phase Skeleton** (conditional -- only when `deep_analysis` is not null)

If `deep_analysis` is not null, display:
```markdown
## Phase Skeleton (Rough Outline)

{for each phase in deep_analysis.phase_skeleton:}
{N}. **{name}** -- {description}

Note: This is a high-level outline. Run /ant:plan for the detailed execution plan.
```
If `deep_analysis` is null, omit this section entirely.

**Section 9: Pheromones**

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

### Step 6.5: Choose Parallel Strategy

Ask the user to choose a parallel execution strategy using `AskUserQuestion`:

```
How should builders work in parallel?

1. In-repo (recommended) -- All builders share the same repo directory. Simple and safe. Best for most projects.
2. Worktree -- Each builder gets an isolated git worktree. Enables true parallel file changes. Requires git worktree support.

Choose [1/2] (default: 1):
```

Parse the user's response:
- If "1", "in-repo", or empty/no response: set `parallel_mode = "in-repo"`
- If "2" or "worktree": set `parallel_mode = "worktree"`
- Otherwise: default to `parallel_mode = "in-repo"` (safe default)

Store the chosen mode as the variable `parallel_mode` for use in Step 7 and Step 8.

### Step 7: Create Colony (Post-Approval)

Only reached after user approval. ALL file writes happen here.

**If re-init mode:**

1. Write charter content via:
```bash
aether charter-write --intent "{approved_intent}" --vision "{approved_vision}" --governance "{approved_governance}" --goals "{approved_goals}"
```

2. Auto-apply approved pheromone suggestions (see pheromone auto-apply below).

2a. **Write init analysis** (if deep_analysis is not null): Write the formatted analysis to `.aether/data/init-analysis.md` using the Write tool. Format the architect JSON output as markdown with sections for Architecture Overview, Mermaid Diagrams, Technical Approach, Risk Assessment, Key Decisions, and Phase Skeleton. Include a header with timestamp and goal. This write is non-blocking -- if it fails, log a warning and continue.

3. Update the goal field in COLONY_STATE.json in-place using the state API:
```bash
aether state-write "$(jq --arg new_goal "{approved_intent}" '.goal = $new_goal' .aether/data/COLONY_STATE.json)"
```

4. **Verify the write** — read back and confirm goal is set:
```bash
verify_goal=$(jq -r '.goal' .aether/data/COLONY_STATE.json)
if [[ "$verify_goal" == "null" || -z "$verify_goal" ]]; then
  echo "ERROR: Colony state write failed — goal is still null after write. Re-run /ant:init."
  # Attempt recovery: write goal directly
  jq --arg g "{approved_intent}" '.goal = $g' .aether/data/COLONY_STATE.json > .aether/data/COLONY_STATE.json.tmp && mv .aether/data/COLONY_STATE.json.tmp .aether/data/COLONY_STATE.json
  verify_goal=$(jq -r '.goal' .aether/data/COLONY_STATE.json)
  if [[ "$verify_goal" == "null" || -z "$verify_goal" ]]; then
    echo "FATAL: Recovery write also failed. Colony state may be corrupted."
    stop
  fi
fi
```

5. Run `aether session-init "$(jq -r '.session_id' .aether/data/COLONY_STATE.json)" "{approved_intent}"`

6. Set parallel mode:
```bash
aether state-mutate --field parallel_mode --value "$parallel_mode"
```

7. Skip to Step 8 (display result). Do NOT write COLONY_STATE.json from template, do NOT write constraints.json, do NOT write pheromones.json.

**If fresh init:**

1. Initialize QUEEN.md (already done in Step 2)
2. Write charter content via charter-write (same command as above)
3. Auto-apply approved pheromone suggestions (see pheromone auto-apply below).
3a. **Write init analysis** (if deep_analysis is not null): Write the formatted analysis to `.aether/data/init-analysis.md` using the Write tool. Format the architect JSON output as markdown with sections for Architecture Overview, Mermaid Diagrams, Technical Approach, Risk Assessment, Key Decisions, and Phase Skeleton. Include a header with timestamp and goal. This write is non-blocking -- if it fails, log a warning and continue.
4. Write COLONY_STATE.json from template:
   - Generate a session ID in the format `session_{unix_timestamp}_{random}` and an ISO-8601 UTC timestamp
   - Resolve template: check `~/.aether/system/templates/colony-state.template.json` first, then `.aether/templates/colony-state.template.json`
   - If no template found: output "Template missing: colony-state.template.json. Run aether update to fix." and stop
   - Read the template file. Follow its `_instructions` field
   - Replace placeholders: `__GOAL__` with approved intent, `__SESSION_ID__` with generated session ID, `__ISO8601_TIMESTAMP__` with current timestamp, `__PHASE_LEARNINGS__` with `[]`, `__INSTINCTS__` with `[]`
   - Remove ALL keys starting with underscore
   - Write the resulting JSON to `.aether/data/COLONY_STATE.json` using the Write tool

5. **Verify the write** — read back and confirm COLONY_STATE.json is valid and goal is set:
```bash
verify_goal=$(jq -r '.goal' .aether/data/COLONY_STATE.json 2>/dev/null)
verify_valid=$(jq -e . .aether/data/COLONY_STATE.json >/dev/null 2>&1 && echo "valid" || echo "invalid")
if [[ "$verify_valid" != "valid" || "$verify_goal" == "null" || -z "$verify_goal" ]]; then
  echo "ERROR: Colony state write verification failed (valid=$verify_valid, goal=$verify_goal)"
  echo "The colony file may be corrupted. Remove .aether/data/COLONY_STATE.json and re-run /ant:init."
  stop
fi
echo "Colony state verified: goal=\"$verify_goal\""
```

5a. Set parallel mode:
```bash
aether state-mutate --field parallel_mode --value "$parallel_mode"
```

6. Write constraints.json from template:
   - Resolve template: check `~/.aether/system/templates/constraints.template.json` first, then `.aether/templates/constraints.template.json`
   - If no template found: output "Template missing: constraints.template.json. Run aether update to fix." and stop
   - Read template, follow `_instructions`, remove `_` prefixed keys, write to `.aether/data/constraints.json`

7. Initialize runtime files from templates (non-blocking):
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

8. Run `aether context-update init "{approved_intent}"`
9. Run `aether validate-state colony`
10. Register repo (silent on failure):
```bash
domain_tags=$(aether domain-detect 2>/dev/null | jq -r '.result.tags // ""' || echo "")
aether registry-add --path "$(pwd)" "$(jq -r '.version // "unknown"' ~/.aether/version.json 2>/dev/null || echo 'unknown')" --goal "{approved_intent}" --active true --tags "$domain_tags" 2>/dev/null || true
cp ~/.aether/version.json .aether/version.json 2>/dev/null || true
```
11. Seed QUEEN.md from hive (non-blocking):
```bash
domain_tags=$(jq -r --arg repo "$(pwd)" \
  '[.repos[] | select(.path == $repo) | .domain_tags // []] | .[0] // [] | join(",")' \
  "$HOME/.aether/registry.json" 2>/dev/null || echo "")
seed_args="queen-seed-from-hive --limit 5"
[[ -n "$domain_tags" ]] && seed_args="$seed_args --domain $domain_tags"
seed_result=$(aether $seed_args 2>/dev/null || echo '{}')
seeded_count=$(echo "$seed_result" | jq -r '.result.seeded // 0' 2>/dev/null || echo "0")
```
12. Run `aether session-init "{session_id}" "{approved_intent}"`

**Pheromone auto-apply (referenced by both re-init and fresh init paths above):**

If approved pheromone suggestions exist (the user kept them in the prompt and didn't remove them during the approval loop):

For each approved pheromone suggestion, call:
```bash
aether pheromone-write "{type}" '{content}' --source "system:init" --reason '{reason}' --ttl "30d" 2>/dev/null || true
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
  pher_import=$(aether pheromone-import-xml --input "$latest_chamber/pheromones.xml" --colony "imported" 2>/dev/null || echo '{"ok":false}')
  pher_imported=$(echo "$pher_import" | jq -r '.result.imported // 0' 2>/dev/null || echo "0")
  echo "Pheromones: ${pher_imported} signal(s) imported"
fi

# Import wisdom to queen-wisdom.json (per D-09)
if [[ -f "$latest_chamber/queen-wisdom.xml" ]]; then
  wis_import=$(aether wisdom-import-xml "$latest_chamber/queen-wisdom.xml" ".aether/data/queen-wisdom.json" 2>/dev/null || echo '{"ok":false}')
  wis_imported=$(echo "$wis_import" | jq -r '.result.imported // 0' 2>/dev/null || echo "0")
  echo "Wisdom: ${wis_imported} entries(s) imported to queen-wisdom.json"
fi

# Import registry lineage (per D-09)
if [[ -f "$latest_chamber/colony-registry.xml" ]]; then
  reg_import=$(aether registry-import-xml "$latest_chamber/colony-registry.xml" 2>/dev/null || echo '{"ok":false}')
  reg_imported=$(echo "$reg_import" | jq -r '.result.imported // 0' 2>/dev/null || echo "0")
  echo "Registry: ${reg_imported} colon(ies) lineage imported"
fi
```

All imports are non-blocking -- log warning and continue if any fails.

**If user selects "no":**

Display "Import skipped. Starting fresh colony." and proceed to Step 8.

**If xml_import_available is false (no chambers, no XML, or no xmllint):**

Skip silently -- proceed directly to Step 8 without any mention of import (per D-11).
### Step 7.5: Install Clash Detection Hook

If the `aether clash-setup` command is available, run:

```bash
aether clash-setup --install 2>/dev/null || true
```

This installs the PreToolUse hook that prevents conflicting edits across worktrees.
Non-blocking — if it fails, init continues normally.

Also configure the merge driver for package-lock.json:

```bash
aether gitconfig merge-driver 2>/dev/null || true
```

### Step 8: Display Result

Display the success header and result block:

```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
🥚 A E T H E R   C O L O N Y
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

👑 Queen has set the colony's intention

   "{approved_intent}"

   🟢 Colony Status: READY
   🏗️  Parallel Strategy: {parallel_mode} ({if parallel_mode == "worktree": "builders work in isolated git worktrees" else: "builders share the same repo directory"})

{If re-init: "   🔄 Mode: Re-init (charter updated, state preserved)"}
{If fresh and seeded_count > 0: "   🧠 Hive wisdom: {seeded_count} cross-colony pattern(s) seeded into QUEEN.md"}
{If deep_analysis is not null: "   🔍 Deep analysis: .aether/data/init-analysis.md"}

💾 State persisted -- safe to /clear, then run /ant:plan

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
🐜 Next Up
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
   /ant:plan                 📊 Generate execution plan (with init analysis as foundation)
   /ant:status               📋 Check colony state
   /ant:focus                🎯 Set initial focus
```
