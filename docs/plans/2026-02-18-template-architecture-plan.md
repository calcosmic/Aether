# Aether Template Architecture Plan

> Extracting embedded structures from command files into reusable, versioned,
> self-documenting templates that LLM agents read rather than improvise.

---

## The Problem in Plain English

Every Aether command (init, seal, entomb, build, plan, continue, pause, resume)
contains JSON structures and markdown formats buried inline within 200-1000 line
instruction files. When an LLM agent runs `/ant:init`, it reads a 323-line file,
finds a 30-line JSON block at line 184, and tries to reproduce it exactly. It
often improvises -- adding fields, dropping required ones, or mangling the format.

The GSD system solves this by having 22 template files that commands reference.
A GSD command says "read template X, fill with context Y." Aether has exactly
1 template (QUEEN.md.template). Everything else is inline.

---

## Full Template Inventory

### Priority 1 -- Critical (Commands fail without these)

These structures are written on every colony lifecycle and errors here cause
cascading failures across all subsequent commands.

| # | Template | Used By | Lines Inline | Failure Mode |
|---|----------|---------|-------------|--------------|
| 1 | `colony-state.json.template` | init, colonize | ~30 | Wrong fields = every command fails |
| 2 | `constraints.json.template` | init | ~5 | Missing version = validation fails |
| 3 | `crowned-anthill.md.template` | seal | ~25 | Missing sections = entomb can't parse |
| 4 | `handoff.md.template` | entomb, build, pause, continue | ~30 | Missing sections = resume fails |
| 5 | `colony-state-reset.jq.template` | entomb | ~20 | Wrong reset = stale data persists |

### Priority 2 -- High (Agents improvise output format)

These structures are generated during builds and planning. Improvisation causes
verification failures and inconsistent reporting.

| # | Template | Used By | Lines Inline | Failure Mode |
|---|----------|---------|-------------|--------------|
| 6 | `builder-prompt.md.template` | build | ~50 | Missing sections = workers skip safety |
| 7 | `watcher-prompt.md.template` | build | ~25 | Missing checks = verification gaps |
| 8 | `chaos-prompt.md.template` | build | ~25 | Inconsistent finding format |
| 9 | `scout-prompt.md.template` | plan | ~40 | Research output varies wildly |
| 10 | `route-setter-prompt.md.template` | plan | ~70 | Plan structure drifts between iterations |
| 11 | `archaeologist-prompt.md.template` | build | ~30 | History scan output unstructured |
| 12 | `surveyor-prompt.md.template` | colonize | ~15 | Survey doc quality varies |
| 13 | `builder-result.json.template` | build | ~10 | Missing fields = synthesis fails |
| 14 | `watcher-result.json.template` | build | ~10 | Missing fields = gate logic breaks |
| 15 | `chaos-result.json.template` | build | ~10 | Missing fields = flag creation fails |
| 16 | `synthesis-result.json.template` | build | ~50 | Inconsistent structure = display breaks |

### Priority 3 -- Medium (Display and state formats)

These affect user experience and state tracking. Less catastrophic but cause
confusion and display bugs.

| # | Template | Used By | Lines Inline | Failure Mode |
|---|----------|---------|-------------|--------------|
| 17 | `watch-status.txt.template` | plan | ~12 | Tmux display garbled |
| 18 | `watch-progress.txt.template` | plan | ~10 | Progress bar inconsistent |
| 19 | `verification-report.md.template` | continue | ~30 | Gate decisions lack evidence |
| 20 | `completion-report.md.template` | continue (all complete) | ~20 | No standard format |
| 21 | `phase-learning.json.template` | continue | ~15 | Learning extraction misses fields |
| 22 | `instinct.json.template` | continue | ~15 | Instinct format drifts |
| 23 | `event-format.template` | all commands | ~1 | Pipe-delimited format varies |
| 24 | `pause-handoff.md.template` | pause-colony | ~30 | Different from entomb handoff |
| 25 | `build-handoff.md.template` | build | ~25 | Missing error context |
| 26 | `continue-handoff.md.template` | continue | ~25 | Missing phase context |

### Total: 26 templates extractable from existing commands

---

## Template Format Standard

Every template follows this structure, modeled after GSD's approach but adapted
for Aether's JSON-heavy, shell-script-heavy environment.

### Structure

```
<template_name>.<ext>.template

Contains:
  1. Header comment block (metadata)
  2. The template body (the actual structure to fill)
  3. Usage instructions (when and how to use)
  4. Field reference (what each field means)
  5. Filled example (a complete working instance)
  6. Validation rules (how to check correctness)
```

### Example: colony-state.json.template

```
<!--
  Template: colony-state.json
  Version: 3.0
  Used by: /ant:init, /ant:colonize (bootstrap)
  Location: .aether/data/COLONY_STATE.json

  Changelog:
    3.0 — Added memory.instincts, errors.flagged_patterns
    2.0 — Added plan.confidence, memory.decisions
    1.0 — Initial structure
-->

<purpose>
The colony state file is the central nervous system of an Aether colony.
Every command reads it. Most commands write to it. A malformed state file
will cause every subsequent command to fail.
</purpose>

<template>
{
  "version": "3.0",
  "goal": "${GOAL}",
  "state": "${STATE:READY}",
  "current_phase": ${CURRENT_PHASE:0},
  "session_id": "${SESSION_ID}",
  "initialized_at": "${TIMESTAMP}",
  "build_started_at": null,
  "plan": {
    "generated_at": null,
    "confidence": null,
    "phases": []
  },
  "memory": {
    "phase_learnings": ${PHASE_LEARNINGS:[]},
    "decisions": [],
    "instincts": ${INSTINCTS:[]}
  },
  "errors": {
    "records": [],
    "flagged_patterns": []
  },
  "signals": [],
  "graveyards": [],
  "events": [
    "${TIMESTAMP}|colony_initialized|init|Colony initialized with goal: ${GOAL}"
  ]
}
</template>

<fields>
| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| version | string | yes | "3.0" | Schema version. Commands check this for upgrades. |
| goal | string|null | yes | null | The colony's purpose. Null means uninitialized. |
| state | enum | yes | "IDLE" | IDLE, READY, EXECUTING, PLANNING, PAUSED |
| current_phase | int | yes | 0 | 0 = no phase active, 1+ = phase number |
| session_id | string|null | yes | null | Format: session_{unix}_{random} |
| initialized_at | ISO-8601|null | yes | null | When /ant:init ran |
| build_started_at | ISO-8601|null | yes | null | When current build started |
| plan.generated_at | ISO-8601|null | yes | null | When plan was finalized |
| plan.confidence | int|null | yes | null | 0-100 planning confidence |
| plan.phases | array | yes | [] | Phase objects from planner |
| memory.phase_learnings | array | yes | [] | Extracted learnings per phase |
| memory.decisions | array | yes | [] | Architectural decisions |
| memory.instincts | array | yes | [] | Behavioral instincts |
| errors.records | array | yes | [] | Error log entries |
| errors.flagged_patterns | array | yes | [] | Recurring error patterns |
| signals | array | yes | [] | Active pheromone signals |
| graveyards | array | yes | [] | Failed file markers |
| events | array | yes | [] | Pipe-delimited event log |
</fields>

<placeholders>
${GOAL} — The user's goal string from /ant:init arguments
${STATE} — Colony state enum, default "READY" for init
${CURRENT_PHASE} — Phase number, default 0 for init
${SESSION_ID} — Generated as session_{unix_timestamp}_{random_4_chars}
${TIMESTAMP} — ISO-8601 UTC timestamp at write time
${PHASE_LEARNINGS} — Array, usually [] for init. May contain inherited learnings.
${INSTINCTS} — Array, usually [] for init. May contain inherited instincts.
</placeholders>

<example>
{
  "version": "3.0",
  "goal": "Build a REST API with authentication",
  "state": "READY",
  "current_phase": 0,
  "session_id": "session_1708300000_a3f7",
  "initialized_at": "2026-02-18T15:00:00Z",
  "build_started_at": null,
  "plan": {
    "generated_at": null,
    "confidence": null,
    "phases": []
  },
  "memory": {
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
  "events": [
    "2026-02-18T15:00:00Z|colony_initialized|init|Colony initialized with goal: Build a REST API with authentication"
  ]
}
</example>

<validation>
1. version MUST be "3.0" (commands reject unknown versions)
2. state MUST be one of: IDLE, READY, EXECUTING, PLANNING, PAUSED
3. events entries MUST be pipe-delimited: timestamp|type|source|message
4. plan.phases MUST be an array (empty is valid)
5. memory.instincts entries MUST have: id, trigger, action, confidence, domain
6. JSON MUST be valid (parseable by jq)
7. File MUST be written atomically (write to .tmp, then mv)
</validation>
```

### Variable Syntax

Templates use `${VARIABLE}` for required values and `${VARIABLE:default}` for
values with defaults. This is intentionally simple -- LLM agents don't run a
template engine. They read the template, understand the structure, and fill in
values. The syntax is documentation, not execution.

---

## Loading Pattern

### How Commands Reference Templates

Commands shift from "write this structure" to "read template, fill with context."

**Current pattern (init.md, lines 184-213):**

```markdown
### Step 3: Write Colony State

Generate a session ID...

Use the Write tool to write `.aether/data/COLONY_STATE.json` with the v3.0 structure.

{30 lines of JSON structure embedded inline}
```

**New pattern:**

```markdown
### Step 3: Write Colony State

Generate a session ID in the format `session_{unix_timestamp}_{random}`.
Generate an ISO-8601 UTC timestamp.

Read the template `.aether/templates/colony-state.json.template`.

Fill in the template placeholders:
- ${GOAL} = the user's goal from $ARGUMENTS
- ${SESSION_ID} = the generated session ID
- ${TIMESTAMP} = the generated timestamp
- ${PHASE_LEARNINGS} = inherited learnings from Step 2.6, or []
- ${INSTINCTS} = inherited instincts from Step 2.6, or []

Use the Write tool to write the filled template to `.aether/data/COLONY_STATE.json`.
```

This is 12 lines instead of 30+. The command focuses on WHAT to fill in. The
template defines the STRUCTURE. The agent reads both.

### Read-and-Fill (not Copy-and-Fill)

The agent does NOT mechanically substitute variables. It reads the template to
understand the structure, reads the command to understand what values to use,
then writes the file. The template is a reference document, not a preprocessor
input.

This matters because:
1. LLM agents are good at following structural examples
2. LLM agents are bad at extracting structure from long instruction text
3. A separate file is always more reliable than an inline block
4. Templates can include validation rules the command doesn't need to repeat

---

## Before/After Comparisons

### Before/After: /ant:init

**BEFORE (current init.md, Steps 3-4, ~35 lines of embedded structures):**

```markdown
### Step 3: Write Colony State

Generate a session ID in the format `session_{unix_timestamp}_{random}` and
an ISO-8601 UTC timestamp.

Use the Write tool to write `.aether/data/COLONY_STATE.json` with the v3.0
structure.

**If Step 2.6 found instincts to inherit**, convert each into the instinct
format and seed the `memory.instincts` array. Each inherited instinct should
have:
- `id`: `instinct_inherited_{index}`
- `trigger`: inferred from the instinct description
[... 15 more lines of field definitions ...]

```json
{
  "version": "3.0",
  "goal": "<the user's goal>",
  "state": "READY",
  "current_phase": 0,
  [... 20 more lines ...]
}
```

### Step 4: Initialize Constraints

Write `.aether/data/constraints.json`:

```json
{
  "version": "1.0",
  "focus": [],
  "constraints": []
}
```
```

**AFTER (template-referenced init.md, Steps 3-4, ~20 lines):**

```markdown
### Step 3: Write Colony State

Generate a session ID in the format `session_{unix_timestamp}_{random}` and
an ISO-8601 UTC timestamp.

Read the template at `.aether/templates/colony-state.json.template`.

Fill the template with:
- ${GOAL} = the user's goal from $ARGUMENTS
- ${SESSION_ID} = the generated session ID
- ${TIMESTAMP} = the generated timestamp
- ${PHASE_LEARNINGS} = inherited learnings from Step 2.6 (or [] if none)
- ${INSTINCTS} = inherited instincts from Step 2.6 (or [] if none)

For inherited instinct format, see the <inherited_instinct_format> section
in the template.

Write the filled result to `.aether/data/COLONY_STATE.json`.

### Step 4: Initialize Constraints

Read the template at `.aether/templates/constraints.json.template`.
Write it to `.aether/data/constraints.json` with no modifications.
```

**Net change:** -15 lines from init.md. Structure lives in templates that can
be versioned, validated, and shared independently.

---

### Before/After: /ant:seal

**BEFORE (current seal.md, Step 6, lines 179-232, ~53 lines):**

```markdown
### Step 6: Write CROWNED-ANTHILL.md

Calculate colony age:
```bash
initialized_at=$(jq -r '.initialized_at // empty' ...)
[... 10 lines of bash for age calculation ...]
```

Extract phase recap:
```bash
phase_recap=""
while IFS= read -r phase_line; do
[... 5 lines of bash for phase extraction ...]
done
```

Write the seal document:
```bash
version=$(jq -r '.version // "3.0"' ...)
seal_date=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

cat > .aether/CROWNED-ANTHILL.md << SEAL_EOF
# Crowned Anthill -- ${goal}

**Sealed:** ${seal_date}
**Milestone:** Crowned Anthill
**Version:** ${version}

## Colony Stats
- Total Phases: ${total_phases}
- Phases Completed: ${phases_completed} of ${total_phases}
- Colony Age: ${colony_age_days} days
- Wisdom Promoted: ${promotions_made} entries

## Phase Recap
$(echo -e "$phase_recap")

## Pheromone Legacy
- Instincts and validated learnings promoted to QUEEN.md
- ${promotions_made} total entries promoted

## The Work
${goal}
SEAL_EOF
```
```

**AFTER (template-referenced seal.md, Step 6, ~20 lines):**

```markdown
### Step 6: Write CROWNED-ANTHILL.md

Calculate colony age and extract phase recap (same bash as before --
these produce values, not structure).

Read the template at `.aether/templates/crowned-anthill.md.template`.

Fill the template with:
- ${GOAL} = colony goal
- ${SEAL_DATE} = current ISO-8601 UTC timestamp
- ${VERSION} = colony state version
- ${TOTAL_PHASES} = total phase count
- ${PHASES_COMPLETED} = completed phase count
- ${COLONY_AGE_DAYS} = calculated colony age in days
- ${PROMOTIONS_MADE} = count of wisdom entries promoted
- ${PHASE_RECAP} = formatted phase recap lines

Write the filled result to `.aether/CROWNED-ANTHILL.md`.
```

**Net change:** -33 lines from seal.md. The heredoc template moves to a file
that can be read, validated, and evolved independently.

---

### Before/After: /ant:build (Worker Prompts)

**BEFORE (current build.md, Step 5.1, ~50 lines of inline prompt):**

The builder prompt template is embedded directly in the command file across
lines 473-516. The watcher prompt is at lines 574-604. The chaos prompt is at
lines 631-655.

Each prompt contains sections that change between builds (task, goal, context)
mixed with sections that never change (activity logging instructions, spawn
rules, output format).

**AFTER (template-referenced build.md, Step 5.1, ~15 lines):**

```markdown
### Step 5.1: Spawn Wave 1 Workers (Parallel)

Read the builder prompt template at `.aether/templates/builder-prompt.md.template`.

For each Wave 1 task, fill the template with:
- ${ANT_NAME} = generated ant name
- ${TASK_ID} = task identifier
- ${TASK_DESCRIPTION} = task goal description
- ${COLONY_GOAL} = colony goal from state
- ${ARCHAEOLOGY_CONTEXT} = archaeologist findings (or empty if skipped)
- ${QUEEN_WISDOM_SECTION} = formatted wisdom (or empty if none)
- ${PHEROMONE_SECTION} = formatted signals (or empty if none)

Spawn using Task tool with subagent_type="aether-builder" and the filled prompt.
```

The template file itself contains the full prompt with static sections (logging
instructions, spawn rules, output JSON format) that never need to change in the
command file. When we update logging instructions, we update one template -- not
3 inline prompts across 3 commands.

---

## Template Directory Structure

```
.aether/templates/
  -- Data Templates (JSON structures written to .aether/data/) --
  colony-state.json.template          Priority 1  (init, colonize)
  constraints.json.template           Priority 1  (init)
  colony-state-reset.jq.template      Priority 1  (entomb)
  phase-learning.json.template        Priority 3  (continue)
  instinct.json.template              Priority 3  (continue)
  event-format.template               Priority 3  (all commands)

  -- Document Templates (Markdown files written during lifecycle) --
  crowned-anthill.md.template         Priority 1  (seal)
  handoff-entomb.md.template          Priority 1  (entomb)
  handoff-pause.md.template           Priority 3  (pause-colony)
  handoff-build.md.template           Priority 3  (build)
  handoff-continue.md.template        Priority 3  (continue)
  completion-report.md.template       Priority 3  (continue, all complete)
  verification-report.md.template     Priority 3  (continue)

  -- Worker Prompt Templates (Injected into Task tool spawns) --
  prompts/
    builder-prompt.md.template        Priority 2  (build)
    watcher-prompt.md.template        Priority 2  (build)
    chaos-prompt.md.template          Priority 2  (build)
    scout-broad.md.template           Priority 2  (plan, iteration 1)
    scout-gap.md.template             Priority 2  (plan, iteration 2+)
    route-setter-prompt.md.template   Priority 2  (plan)
    archaeologist-prompt.md.template  Priority 2  (build)
    surveyor-prompt.md.template       Priority 2  (colonize)

  -- Result Schema Templates (Expected output from worker agents) --
  results/
    builder-result.json.template      Priority 2  (build)
    watcher-result.json.template      Priority 2  (build)
    chaos-result.json.template        Priority 2  (build)
    synthesis-result.json.template    Priority 2  (build)
    scout-result.json.template        Priority 2  (plan)
    route-setter-result.json.template Priority 2  (plan)

  -- Display Templates (Watch files and status formats) --
  display/
    watch-status.txt.template         Priority 3  (plan)
    watch-progress.txt.template       Priority 3  (plan)

  -- Existing --
  QUEEN.md.template                   Already exists (queen-init)
```

Total: 26 new templates + 1 existing = 27

---

## Versioning Strategy

### Template Versions

Each template declares its version in the header comment:

```
<!--
  Template: colony-state.json
  Version: 3.0
  ...
-->
```

### Version Compatibility

Templates version independently from each other. The version in the template
header MUST match what the consuming command expects.

**Upgrade path:**

1. Template version bumps when fields are added/removed/changed
2. Commands that consume the template note the expected version
3. The `validate-state` utility checks the version field in written files
4. Old state files are upgraded by continue.md's auto-upgrade logic

**Concrete example:**

If colony-state.json gains a `nestmates` field in v3.1:
1. Update `colony-state.json.template` to v3.1 with the new field
2. Update init.md to fill `${NESTMATES}` placeholder
3. Update continue.md's auto-upgrade to add `nestmates: []` to v3.0 files
4. Existing v3.0 files continue to work -- commands check version and upgrade

### Distribution

Templates sync through the existing pipeline:

```
.aether/templates/ (source of truth, this repo)
      |
      v  bin/sync-to-runtime.sh
runtime/templates/ (staging)
      |
      v  npm install -g .
~/.aether/system/templates/ (hub)
      |
      v  aether update (or /ant:init bootstrap)
any-repo/.aether/templates/ (working copy)
```

The sync script already handles `templates/QUEEN.md.template`. Adding new files
to the SYSTEM_FILES array in `bin/sync-to-runtime.sh` is all that's needed for
distribution.

---

## Validation Strategy

### Template Self-Validation

Each template's `<validation>` section defines rules. These are human-readable
and machine-checkable.

### Runtime Validation

The existing `validate-state colony` utility already checks COLONY_STATE.json
structure. Templates formalize what it should check:

1. Required fields present
2. Field types correct
3. Enum values valid
4. Version matches expected

### Future: Schema Validation

Aether already has XSD schemas in `.aether/schemas/` for XML formats. JSON
templates could eventually have companion JSON Schema files, but this is
deferred -- the template `<validation>` section is sufficient for now because
the consumer is an LLM, not a schema validator.

---

## Implementation Approach

### Phase 1: Critical Templates (Priority 1) -- Estimated 2-3 hours

1. Extract `colony-state.json.template` from init.md lines 184-213
2. Extract `constraints.json.template` from init.md lines 219-225
3. Extract `crowned-anthill.md.template` from seal.md lines 206-232
4. Extract `handoff-entomb.md.template` from entomb.md lines 411-442
5. Extract `colony-state-reset.jq.template` from entomb.md lines 358-378
6. Update init.md to reference templates instead of embedding
7. Update seal.md to reference templates instead of embedding
8. Update entomb.md to reference templates instead of embedding
9. Add new files to `bin/sync-to-runtime.sh` SYSTEM_FILES array
10. Test: run `/ant:init`, `/ant:seal`, `/ant:entomb` flow

### Phase 2: Worker Prompt Templates (Priority 2) -- Estimated 3-4 hours

1. Extract builder/watcher/chaos prompts from build.md
2. Extract scout/route-setter prompts from plan.md
3. Extract archaeologist prompt from build.md
4. Extract surveyor prompt from colonize.md
5. Extract result JSON schemas from build.md and plan.md
6. Update build.md to reference prompt templates
7. Update plan.md to reference prompt templates
8. Update colonize.md to reference prompt templates
9. Add to SYSTEM_FILES array
10. Test: run full colony lifecycle

### Phase 3: Display and State Templates (Priority 3) -- Estimated 2 hours

1. Extract watch file templates from plan.md
2. Extract learning/instinct templates from continue.md
3. Extract handoff variants from build.md, continue.md, pause-colony.md
4. Extract verification report format from continue.md
5. Extract completion report format from continue.md
6. Update all consuming commands
7. Add to SYSTEM_FILES array
8. Test: full lifecycle with watch/status commands

### Phase 4: OpenCode Parity -- Estimated 1-2 hours

1. Verify `.opencode/commands/ant/` commands reference same templates
2. Templates live in `.aether/templates/` which both systems can read
3. No duplication needed -- templates are shared

---

## What This Does NOT Change

1. **Command logic stays in commands.** Templates hold structure, not behavior.
   The 6-phase verification loop in continue.md stays inline -- it's logic,
   not a data structure.

2. **Shell scripts stay in utils.** The `aether-utils.sh` functions that
   write/read state continue to work. Templates are for the LLM agent, not
   for bash scripts.

3. **XSD schemas stay separate.** The XML schemas in `.aether/schemas/` serve
   a different purpose (validating XML exchange format). Templates serve
   LLM agents writing JSON/Markdown.

4. **Display formatting stays in commands.** The ASCII art, banner widths,
   and visual output formatting are presentation logic, not data structures.
   They stay in commands.

---

## Risk Assessment

### Low Risk
- Template extraction is mechanical (copy existing inline structure to file)
- Commands already know the exact structure they need
- Distribution pipeline already handles templates (QUEEN.md.template proves it)
- No behavioral changes -- same structures, same commands, different source

### Medium Risk
- LLM agents must now read 2 files (command + template) instead of 1
  - Mitigation: commands explicitly say "Read template at path X"
  - Mitigation: templates are short (most under 50 lines)

- Template drift if someone updates inline structure but not template
  - Mitigation: remove inline structures entirely -- command references template
  - Mitigation: CI check could verify no JSON blocks in command files

### Non-Risk
- Performance: reading one extra file adds negligible latency
- Compatibility: templates produce identical output to current inline structures
- User impact: zero -- templates are internal to the agent system

---

## Success Metrics

1. **No command file contains a JSON structure > 5 lines** (currently: 6 commands
   contain 10-50 line JSON blocks)
2. **No command file contains a heredoc template** (currently: seal and entomb
   embed heredoc markdown templates)
3. **Worker prompt templates are shared** (currently: builder prompt is inlined
   in build.md, inaccessible to other commands)
4. **Template versions match state file versions** (currently: version "3.0" is
   a string literal scattered across 4 commands)
5. **All 26 templates exist and are referenced** by at least one command

---

## Appendix: Template-to-Command Mapping

| Template | Commands That Read It |
|----------|----------------------|
| colony-state.json.template | init, colonize |
| constraints.json.template | init |
| colony-state-reset.jq.template | entomb |
| crowned-anthill.md.template | seal |
| handoff-entomb.md.template | entomb |
| handoff-pause.md.template | pause-colony |
| handoff-build.md.template | build |
| handoff-continue.md.template | continue |
| builder-prompt.md.template | build |
| watcher-prompt.md.template | build |
| chaos-prompt.md.template | build |
| scout-broad.md.template | plan |
| scout-gap.md.template | plan |
| route-setter-prompt.md.template | plan |
| archaeologist-prompt.md.template | build |
| surveyor-prompt.md.template | colonize |
| builder-result.json.template | build |
| watcher-result.json.template | build |
| chaos-result.json.template | build |
| synthesis-result.json.template | build |
| scout-result.json.template | plan |
| route-setter-result.json.template | plan |
| watch-status.txt.template | plan |
| watch-progress.txt.template | plan |
| phase-learning.json.template | continue |
| instinct.json.template | continue |
| event-format.template | init, colonize, seal, entomb, build, plan, continue |
| verification-report.md.template | continue |
| completion-report.md.template | continue |
| QUEEN.md.template | queen-init (already exists) |

---

*Plan created: 2026-02-18*
*Status: Awaiting approval*
