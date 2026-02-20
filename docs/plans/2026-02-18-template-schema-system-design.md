# Aether Template and Schema System — Complete Design Plan

## Executive Summary

The core problem is straightforward: LLMs are being asked to reconstruct complex data structures from memory, inside 300-line command files, under context pressure. The fix is equally straightforward — put the exact structure in a file the agent can read, and tell it to read that file before writing anything.

This plan covers eight areas: inventory, template format, schema system, loading patterns, variable substitution, versioning, distribution integration, and migration. Every recommendation is grounded in what the codebase actually does today.

---

## 1. Template Inventory and Priority Tiers

Priority is determined by two factors: how often the structure is written (frequency), and how catastrophically things break when it is wrong (blast radius).

### Tier 1 — Critical (Build These First)

These are written on every colony lifecycle operation. Getting them wrong silently corrupts state.

**COLONY_STATE.json** — Written by init, read by every command. The v3.0 structure is currently embedded as a 30-line JSON block in init.md (lines 184-213) and reconstructed from prose in build.md's auto-upgrade path. This is the highest-priority template in the system.

**constraints.json** — Written by init, modified by focus/redirect/feedback, read by build and resume. Currently a 6-line inline block in init.md (Step 4). Small structure but read on every build.

**session.json** — Written by init (via `session-init` in aether-utils.sh), read by resume. The resume command's entire flow depends on the exact field names `colony_goal`, `current_phase`, `last_command`, `suggested_next`, `baseline_commit`, `session_id`. If these drift from what `session-init` actually writes, resume breaks silently.

**flags.json** — Written by `flag-add` in aether-utils.sh. Read by build (Step 1.5), continue, and flags commands. The schema for individual flag entries (id, type, title, description, source, phase, status, created_at) needs to be standardized.

**manifest.json** — Written during entomb (via `chamber-create`). The two real examples in the codebase show inconsistent structure — the v1-1 chamber has 7 fields, the phase0 chamber has a completely different shape with nested arrays. This inconsistency is exactly the drift that templates prevent.

### Tier 2 — High Priority

These are written less frequently but their failure causes visible, user-facing breakage.

**CROWNED-ANTHILL.md** — Written by seal via heredoc (lines 209-232 of seal.md). The heredoc approach is already close to a template — it just needs to be extracted into a proper file so the LLM reads the structure rather than reconstructing it.

**HANDOFF.md** — Written by build (Step 5.9 error path, Step 6.5), pause-colony, and entomb (Step 11). Currently three separate heredocs with different content shapes. A single template with optional sections would unify these.

**worker-result.json** — The JSON that builders, watchers, and chaos ants are instructed to return (see build.md Steps 5.1, 5.4, 5.6). Currently defined inline in worker prompts. Workers reconstruct it from prose every time.

**phase-plan.md** — Written by plan command. The structure (tasks with depends_on, success_criteria, hints) needs to be consistent because build.md parses it.

### Tier 3 — Medium Priority

These matter for consistency and future tooling.

**completion-report.md** — Read by init Step 2.6 to extract inherited learnings. The parsing logic expects specific section headers (`## Colony Instincts`, `## Colony Learnings (Validated)`) and a specific line format (`N. [confidence] domain: description`). If the format drifts, inheritance silently fails.

**verification-report.md** — The 6-phase quality gate output from build's watcher. Currently described in prose in the watcher prompt.

**HANDOFF.md** (pause-colony variant) — Separate from the build-error variant above.

**caste-metrics.json** — Proposed, not yet implemented.

**phase-scratch.json** — Proposed, not yet implemented.

### Tier 4 — Configuration (One-Time)

**model-profiles.yaml** — Already exists as a file. Needs a template so that `aether update` can distribute it without overwriting user customizations.

**checkpoint-allowlist.json** — Already well-structured. Needs the `templates/` and `schemas/` directories added to `system_files` once templates exist.

---

## 2. Template Format Standard

Every template in this system should follow one format. The question was whether templates should be pure structure or annotated. The answer is: annotated for LLM consumption, pure for shell script consumption.

### The Annotated Template Format (for LLM-written files)

```json
{
  "_template": "colony-state",
  "_version": "3.0",
  "_instructions": "Write this file to .aether/data/COLONY_STATE.json. Replace all __PLACEHOLDER__ values with real data. Remove all _template, _version, _instructions, and _comment_* keys before writing.",
  "version": "3.0",
  "goal": "__GOAL__",
  "state": "READY",
  "current_phase": 0,
  "session_id": "__SESSION_ID__",
  "initialized_at": "__ISO8601_TIMESTAMP__",
  "build_started_at": null,
  "plan": {
    "_comment_plan": "Generated by /ant:plan. Leave null until plan is created.",
    "generated_at": null,
    "confidence": null,
    "phases": []
  },
  "memory": {
    "_comment_memory": "Seeded from completion-report.md if prior colony exists. Otherwise empty arrays.",
    "phase_learnings": [],
    "decisions": [],
    "instincts": []
  },
  "errors": {
    "records": [],
    "flagged_patterns": []
  },
  "signals": [],
  "graveyards": [],
  "events": []
}
```

The `_comment_*` keys are stripped during the "remove metadata" step. This is better than inline JSON comments (which are invalid JSON) and better than a separate annotation file (which the LLM may not read).

### Why This Format Works

It solves three problems simultaneously. First, the LLM sees the exact structure it needs to produce — no reconstruction from memory. Second, the `_instructions` field tells the LLM what to do with the template in plain language, which appears directly in the LLM's context when it reads the file. Third, the `__PLACEHOLDER__` convention (double underscores) is visually distinct from real values and unambiguous — it cannot be confused with an empty string or null.

### The Pure Template Format (for shell script consumption)

A small number of templates are read and written by shell scripts rather than LLMs. These use `null` placeholders and no annotation metadata, because `jq` processes them directly:

```json
{
  "version": "1.0",
  "chamber_id": null,
  "created_at": null,
  "goal": null,
  "milestone": null,
  "phases_completed": 0,
  "total_phases": 0,
  "entombed_at": null
}
```

Shell scripts use `jq --arg` substitution rather than reading annotation comments.

### Markdown Template Format

For markdown documents like CROWNED-ANTHILL.md and HANDOFF.md, use placeholder comments:

```markdown
# Crowned Anthill — {{GOAL}}

**Sealed:** {{SEAL_DATE}}
**Milestone:** Crowned Anthill
**Version:** {{VERSION}}

## Colony Stats
- Total Phases: {{TOTAL_PHASES}}
- Phases Completed: {{PHASES_COMPLETED}} of {{TOTAL_PHASES}}
- Colony Age: {{COLONY_AGE_DAYS}} days
- Wisdom Promoted: {{PROMOTIONS_MADE}} entries

## Phase Recap
{{PHASE_RECAP}}

## Pheromone Legacy
- Instincts and validated learnings promoted to QUEEN.md
- {{PROMOTIONS_MADE}} total entries promoted

## The Work
{{GOAL}}
```

Double-brace `{{VARIABLE}}` for markdown is readable, unambiguous, and will not be confused with JSON syntax.

---

## 3. Schema System Design

The system already has `.aether/schemas/` with XSD files for XML structures. JSON Schema should live alongside these, in the same directory, with a clear naming convention.

### Directory Layout

```
.aether/schemas/
  aether-types.xsd          (existing — shared XML types)
  pheromone.xsd             (existing)
  queen-wisdom.xsd          (existing)
  colony-registry.xsd       (existing)
  worker-priming.xsd        (existing)
  prompt.xsd                (existing)
  json/
    colony-state.schema.json
    constraints.schema.json
    session.schema.json
    flags.schema.json
    manifest.schema.json
    worker-result.schema.json
    phase-plan.schema.json
```

### JSON Schema Design Principles

Schemas serve two roles: documentation (what fields exist and what they mean) and validation (does `validate-state` catch drift). They should not be overly strict — the colony is a living system and the schema needs to accommodate growth without breaking validation.

Use `additionalProperties: true` at the top level so new fields added during development do not fail validation. Require only the fields that every command absolutely depends on.

**Example — colony-state.schema.json (core required fields only):**

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "$id": "https://aether.colony/schemas/json/colony-state",
  "title": "Colony State",
  "description": "Central colony state file. Written by /ant:init, read by all commands.",
  "type": "object",
  "required": ["version", "goal", "state", "current_phase", "plan", "memory", "errors", "events"],
  "additionalProperties": true,
  "properties": {
    "version": {
      "type": "string",
      "description": "Schema version. Must be 3.0 for current commands.",
      "enum": ["3.0"]
    },
    "goal": {
      "type": ["string", "null"],
      "description": "The colony's intention. Null when colony is IDLE."
    },
    "state": {
      "type": "string",
      "enum": ["IDLE", "READY", "PLANNING", "EXECUTING", "PAUSED"],
      "description": "Current colony lifecycle state."
    },
    "current_phase": {
      "type": "integer",
      "minimum": 0,
      "description": "0 = no phase active. 1+ = phase number currently executing."
    },
    "session_id": {
      "type": ["string", "null"],
      "description": "Format: session_{unix_timestamp}_{random}"
    },
    "plan": {
      "type": "object",
      "required": ["phases"],
      "properties": {
        "phases": {"type": "array"},
        "generated_at": {"type": ["string", "null"]},
        "confidence": {"type": ["number", "null"]}
      }
    },
    "memory": {
      "type": "object",
      "required": ["phase_learnings", "decisions", "instincts"],
      "properties": {
        "phase_learnings": {"type": "array"},
        "decisions": {"type": "array"},
        "instincts": {"type": "array"}
      }
    },
    "errors": {
      "type": "object",
      "required": ["records", "flagged_patterns"],
      "properties": {
        "records": {"type": "array"},
        "flagged_patterns": {"type": "array"}
      }
    },
    "events": {
      "type": "array",
      "description": "Event log. Max 100 entries (older entries pruned by build.md)."
    }
  }
}
```

### Validation Integration

The `validate-state colony` call in aether-utils.sh currently exists but its implementation is not shown. It should be wired to check against `colony-state.schema.json`. The validation step in init.md Step 6 and build.md Step 2 already calls this — no command changes needed, just the schema backing the function.

For shell scripts that cannot use a JSON Schema validator natively, a lightweight check using `jq` against required field presence is sufficient. Full JSON Schema validation can be added as an optional enhancement when a Node.js validator is available.

---

## 4. Loading and Instantiation Patterns

This is the most operationally important section. The loading pattern determines whether templates actually get used.

### Pattern A — Read, Fill, Write (for LLM-written files)

This is the primary pattern for all Tier 1 templates. The command instruction becomes:

```
### Step 3: Write Colony State

Read `.aether/templates/json/colony-state.template.json` using the Read tool.

This file contains the exact structure you must write. Replace all __PLACEHOLDER__ values:
- __GOAL__ → the user's goal from $ARGUMENTS
- __SESSION_ID__ → generated as session_{unix_timestamp}_{random}
- __ISO8601_TIMESTAMP__ → current UTC timestamp in ISO-8601 format

If prior colony knowledge was found in Step 2.6, populate memory.instincts and
memory.phase_learnings from those findings. Otherwise, leave as empty arrays.

Remove all keys beginning with underscore (_template, _version, _instructions,
all _comment_* keys) before writing.

Write the resulting JSON to `.aether/data/COLONY_STATE.json`.
```

This is a critical change from the current approach. The LLM is no longer reading a JSON block embedded in a command file. It is reading a separate file that contains only the structure, then filling it in. The template is the ground truth — if the template is wrong, fixing it fixes every future init.

### Pattern B — Copy, Then Update (for shell script-written files)

Used when a shell script creates the file and an LLM later updates it. The shell script does:

```bash
cp .aether/templates/json/manifest.template.json .aether/chambers/$chamber_name/manifest.json
jq --arg id "$chamber_name" \
   --arg goal "$goal" \
   --arg ts "$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
   '.chamber_id = $id | .goal = $goal | .entombed_at = $ts' \
   .aether/templates/json/manifest.template.json > .aether/chambers/$chamber_name/manifest.json
```

The template never changes during this operation — `jq` reads from template, writes to destination.

### Pattern C — Template Reference for Worker Output (for spawned agents)

Builder, watcher, and chaos ant output JSON is currently defined inline in worker prompts. Move the definition to template files and reference them:

```
Return ONLY the JSON structure defined in `.aether/templates/json/worker-result.template.json`.
Read that file now. Fill in each field from your work results.
Remove the _instructions key before returning.
```

This is the pattern that will have the highest immediate reliability impact on builds, because workers currently reconstruct the output JSON schema from memory on every spawn.

### Pattern D — Heredoc Replacement (for markdown documents)

Replace the heredoc constructs in seal.md and entomb.md:

Current approach (seal.md lines 209-232):
```bash
cat > .aether/CROWNED-ANTHILL.md << SEAL_EOF
# Crowned Anthill — ${goal}
...inline template...
SEAL_EOF
```

New approach:
```bash
sed \
  -e "s/{{GOAL}}/$goal/g" \
  -e "s/{{SEAL_DATE}}/$seal_date/g" \
  -e "s/{{VERSION}}/$version/g" \
  -e "s/{{TOTAL_PHASES}}/$total_phases/g" \
  -e "s/{{PHASES_COMPLETED}}/$phases_completed/g" \
  -e "s/{{COLONY_AGE_DAYS}}/$colony_age_days/g" \
  -e "s/{{PROMOTIONS_MADE}}/$promotions_made/g" \
  .aether/templates/md/crowned-anthill.template.md > .aether/CROWNED-ANTHILL.md
```

Or, since LLMs write this file rather than shell scripts in most cases, Pattern A applies: read the template, fill the placeholders, write the output.

---

## 5. Variable Substitution Approach

Three placeholder conventions exist in the codebase right now. The new system should standardize on two — one per consumer type.

### For JSON templates consumed by LLMs: `__DOUBLE_UNDERSCORE__`

Rationale: visually loud, cannot be confused with valid JSON values, cannot be confused with markdown syntax. An LLM reading `"goal": "__GOAL__"` immediately understands this is a placeholder.

Reserved placeholders (system-wide):
- `__GOAL__` — the colony's goal string
- `__SESSION_ID__` — session_{unix_timestamp}_{random}
- `__ISO8601_TIMESTAMP__` — current UTC time
- `__UNIX_TIMESTAMP__` — current time as epoch integer
- `__NULL__` — explicitly null (LLM writes `null` not the string)

### For markdown templates consumed by shell scripts or LLMs: `{{DOUBLE_BRACE}}`

Rationale: standard template convention, readable, easy to `sed` substitute, does not conflict with JSON or shell syntax.

### Existing convention to deprecate: `{SINGLE_BRACE}`

The QUEEN.md template currently uses `{TIMESTAMP}`. This conflicts with common shell variable syntax and JSON format strings. Migrate to `{{TIMESTAMP}}` when the template is next touched.

### Strip Logic

Every template consumer must strip metadata before writing. This is a one-line `jq` call for JSON:

```bash
jq 'del(.._template, .._version, .._instructions) | del(.. | objects | with_entries(select(.key | startswith("_comment_"))))' template.json
```

Or more simply, since the LLM is writing the file directly, the instruction "remove all underscore-prefixed keys" is sufficient — LLMs follow this reliably when it is an explicit instruction.

---

## 6. Version Management

This is where the design requires the most discipline, because it touches the update pipeline.

### Template Versioning Within Files

Every JSON template carries its own version:

```json
{
  "_template": "colony-state",
  "_version": "3.0",
  "_min_aether_version": "1.1.0",
  ...
}
```

The `_min_aether_version` field lets the system detect when a distributed template is too new for the local Aether installation.

### The Core Versioning Constraint

Template version must match the file version the commands expect. COLONY_STATE.json is at v3.0. The template must produce v3.0 output. When the format evolves to v3.1, both the template and the commands that read the file must be updated together, in the same commit.

This is already how the system works for command files — the sync script keeps commands and system files in lockstep. Templates are now system files and get the same treatment.

### Upgrade Path When Templates Evolve

When `aether update` delivers a new template version to a target repo:

1. The new template arrives at `.aether/templates/json/colony-state.template.json`
2. The running colony's `.aether/data/COLONY_STATE.json` still uses the old format
3. The existing auto-upgrade logic in build.md and continue.md handles the conversion

No new migration mechanism is needed. The existing "auto-upgrade old state" pattern (checking for version "1.0" or "2.0" in the `version` field) is the right model. Extend it: if version is less than the template version, run the upgrade path.

### Template Registry

Add a `templates/REGISTRY.json` file that maps template names to their current versions:

```json
{
  "_registry_version": "1.0",
  "_description": "Maps template names to versions. Read by aether update to detect template drift.",
  "templates": {
    "colony-state": "3.0",
    "constraints": "1.0",
    "session": "1.0",
    "flags": "1.0",
    "manifest": "1.0",
    "worker-result": "1.0",
    "crowned-anthill": "1.0",
    "handoff": "1.0"
  }
}
```

The update command can compare this registry against the hub version to tell users when templates have changed.

---

## 7. Distribution Integration

Templates must flow through the existing sync pipeline. The sync script at `bin/sync-to-runtime.sh` is the right place to add them — it already handles schemas (lines 74-79). The template directory needs the same treatment.

### Sync Script Addition

Add to the `SYSTEM_FILES` array in sync-to-runtime.sh:

```bash
"templates/REGISTRY.json"
"templates/json/colony-state.template.json"
"templates/json/constraints.template.json"
"templates/json/session.template.json"
"templates/json/flags.template.json"
"templates/json/manifest.template.json"
"templates/json/worker-result.template.json"
"templates/md/crowned-anthill.template.md"
"templates/md/handoff.template.md"
"templates/md/completion-report.template.md"
```

This is the only change needed to the distribution pipeline. Once added to SYSTEM_FILES, templates flow automatically:

```
.aether/templates/ (source of truth)
  -> runtime/templates/ (staging, via sync-to-runtime.sh on npm install)
  -> ~/.aether/system/templates/ (hub, packaged in npm)
  -> target-repo/.aether/templates/ (working copy, via aether update)
```

### Bootstrap Integration

The init.md bootstrap step (Step 1.5, lines 63-75) already copies templates from hub on first install:

```bash
cp -Rf ~/.aether/system/templates/* .aether/templates/ 2>/dev/null || true
```

This line already exists in the command. Once templates are in the hub, they arrive in new repos automatically. No change needed to the bootstrap step.

### Checkpoint Allowlist Addition

The `checkpoint-allowlist.json` currently lists `system_files` that are safe to stash. Templates are system files and should be added:

```json
"system_files": [
  ".aether/aether-utils.sh",
  ".aether/workers.md",
  ".aether/docs/**/*.md",
  ".aether/templates/**/*",
  ".aether/schemas/**/*",
  ".claude/commands/ant/**/*.md",
  ...
]
```

---

## 8. Migration Plan

The migration has a strict requirement: no existing commands break during the transition. The approach is additive-first — templates exist alongside embedded structures, then embedded structures are removed once templates are validated.

### Phase 1: Create Templates Without Touching Commands (Week 1)

Create all template files in `.aether/templates/`. Add them to sync script. Distribute via `npm install -g .`. At this point, templates exist in all repos but nothing reads them yet. Zero risk.

File creation order (highest impact first):
1. `templates/json/colony-state.template.json`
2. `templates/json/constraints.template.json`
3. `templates/json/session.template.json`
4. `templates/json/flags.template.json`
5. `templates/json/manifest.template.json`
6. `templates/json/worker-result.template.json`
7. `templates/md/crowned-anthill.template.md`
8. `templates/md/handoff.template.md`

### Phase 2: Update Commands to Read Templates (Week 2)

Start with init.md — it is the most critical and the safest to change because it writes a fresh file every time. The change is surgical: replace the inline JSON block (lines 184-213) with a "read the template, fill placeholders, write the file" instruction.

**init.md Step 3 — before:**
```
Write `.aether/data/COLONY_STATE.json` with the v3.0 structure.
[30-line inline JSON block]
```

**init.md Step 3 — after:**
```
Read `.aether/templates/json/colony-state.template.json` using the Read tool.
Replace all __PLACEHOLDER__ values as documented in the template's _instructions field.
Remove all underscore-prefixed metadata keys.
Write the result to `.aether/data/COLONY_STATE.json`.
```

Then update in order: constraints (init.md Step 4), worker-result (build.md Steps 5.1, 5.4, 5.6), CROWNED-ANTHILL.md (seal.md Step 6), HANDOFF.md (build.md Step 5.9, entomb.md Step 11), manifest.json (entomb.md Step 6 via chamber-create).

### Phase 3: Wire Schema Validation (Week 2-3)

Add JSON Schema files to `.aether/schemas/json/`. Update `validate-state colony` in aether-utils.sh to check `colony-state.schema.json`. The validation calls already exist in init.md Step 6 and build.md Step 2 — the schema just needs to be there for the validator to use.

### Phase 4: Remove Embedded Structures (Week 3)

Once templates have been in production for at least one full colony lifecycle (init -> plan -> build -> seal -> entomb), remove the inline JSON blocks from the command files. By this point, the commands reference templates and the inline blocks are dead code. Removing them makes the command files shorter and clearer.

### Phase 5: Agent Template References (Week 4)

Update agent definition files in `.aether/agents/` to reference templates for their expected output format. Each agent that returns structured JSON (builders, watchers, chaos ants, scouts) gets a line in its definition:

```
## Output Format

Read `.aether/templates/json/worker-result.template.json` before returning results.
Your response MUST match this structure exactly.
```

This is the final step because it touches the most files (25 agent definitions) and has the least risk — agent output format was already described in prose, the template just formalizes it.

---

## Implementation File Locations

All files to create live under `.aether/templates/` (source of truth) and sync to `runtime/templates/` automatically.

**JSON templates:**
- `.aether/templates/json/colony-state.template.json`
- `.aether/templates/json/constraints.template.json`
- `.aether/templates/json/session.template.json`
- `.aether/templates/json/flags.template.json`
- `.aether/templates/json/manifest.template.json`
- `.aether/templates/json/worker-result.template.json`
- `.aether/templates/REGISTRY.json`

**Markdown templates:**
- `.aether/templates/md/crowned-anthill.template.md`
- `.aether/templates/md/handoff.template.md`
- `.aether/templates/md/completion-report.template.md`

**JSON Schemas:**
- `.aether/schemas/json/colony-state.schema.json`
- `.aether/schemas/json/constraints.schema.json`
- `.aether/schemas/json/flags.schema.json`
- `.aether/schemas/json/manifest.schema.json`

**Sync script:**
- `bin/sync-to-runtime.sh` — add template entries to SYSTEM_FILES

---

## Key Design Decisions Summary

**Why annotated templates rather than pure structure.** The LLM reads the template file at runtime. If the template contains instructions about what to do with it, those instructions are in the LLM's context at exactly the moment they are needed. A pure structure file forces the instructions to live in the command file, which is the current problem.

**Why `__DOUBLE_UNDERSCORE__` for JSON placeholders.** The system must not break if an LLM fails to substitute a placeholder. `__GOAL__` is invalid as a JSON string value if the resulting file is then passed through `jq`, which catches the error. `null` placeholders (the alternative) would produce silently-wrong valid JSON.

**Why not a template engine.** Introducing a Handlebars or Jinja-style template engine adds a dependency and a processing step. The system's current approach — LLMs read files directly — is already the right pattern. Templates work with that pattern rather than replacing it.

**Why keep templates and schemas separate.** Templates tell producers what to write. Schemas tell validators what to check. They serve different consumers (LLMs vs. shell scripts/validators) and should not be combined. The existing XSD schemas follow this separation correctly; the new JSON templates extend it.

**Why the additive migration rather than a big-bang replacement.** The system is actively used. Two existing chamber archives show real colony work. A migration that breaks the init command breaks every new project. The additive-first approach means templates can be deployed and validated in one colony before commands are updated to depend on them.

---

*Plan created: 2026-02-18*
*Status: Awaiting approval*
