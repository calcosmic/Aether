# Phase 15: Infrastructure State - Research

**Researched:** 2026-02-03
**Domain:** JSON state file design and Claude-native prompt enrichment for error tracking, colony memory, and event logging
**Confidence:** HIGH

## Summary

Phase 15 establishes the data layer that all subsequent phases (Worker Knowledge, Integration & Dashboard) depend on. It creates three new JSON state files (`errors.json`, `memory.json`, `events.json`) in `.aether/data/` and enriches existing command prompts to read/write these files. The key constraint: **no new Python, no new bash scripts, no new commands** -- all implementation is through JSON file creation and enriched markdown prompt files (`.claude/commands/ant/*.md`).

The research analyzed: (1) the current command prompts to understand exactly where state-writing instructions need to be inserted, (2) the v2 Python code (`error_prevention.py`, `triple_layer_memory.py`, `event-bus.sh`) to extract the right schema fields, and (3) the established patterns from Phase 14 for how to enrich command prompts with new Read/Write steps without breaking existing flow.

The established pattern is clear: commands already read `COLONY_STATE.json`, `pheromones.json`, and `PROJECT_PLAN.json` using the Read tool at their "Read State" step. Phase 15 adds `errors.json`, `memory.json`, and `events.json` to those reads, then adds new steps for writing to those files at appropriate points in the command flow.

**Primary recommendation:** Design minimal JSON schemas (flat arrays, simple objects), add file creation to init.md's Step 3, add error/event writing to build.md's Steps 5-6, add memory extraction to continue.md between Steps 3-4, and add decision logging to focus.md/redirect.md/feedback.md's Step 3.

## Standard Stack

### Core

This phase has no library dependencies. The "stack" is JSON schema design and Claude prompt engineering.

| Component | Purpose | Why Standard |
|-----------|---------|--------------|
| JSON files in `.aether/data/` | Persistent state storage | Established pattern from v3-rebuild (COLONY_STATE.json, pheromones.json, PROJECT_PLAN.json) |
| Read/Write tools in command prompts | State manipulation | Claude-native pattern -- no bash/jq/Python needed |
| ISO-8601 timestamps | Time-based filtering and ordering | Already used throughout the codebase (pheromones, colony state) |
| UUID-style IDs | Record identification | Pattern: `{type}_{unix_timestamp}_{random}` already used for pheromone IDs |

### Existing State Files (Reference)

| File | Schema | Read By | Written By |
|------|--------|---------|------------|
| `COLONY_STATE.json` | `{goal, state, current_phase, session_id, initialized_at, workers}` | All commands | init.md, build.md, continue.md, colonize.md |
| `pheromones.json` | `{signals: [{id, type, content, strength, half_life_seconds, created_at}]}` | status.md, build.md, resume-colony.md, pause-colony.md | init.md, focus.md, redirect.md, feedback.md, continue.md |
| `PROJECT_PLAN.json` | `{goal, generated_at, phases: [{id, name, description, status, tasks, success_criteria}]}` | Most commands | plan.md, build.md |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Flat JSON arrays | Nested category-indexed objects | Flat arrays are simpler for Claude to manipulate with Read/Write; nesting adds complexity for no benefit |
| Single events.json file | Separate files per event type | Single file keeps reads simple; multiple files require multiple Read calls |
| Inline pattern detection | Separate patterns.json | Keep it simple -- pattern detection is a count operation on the errors array |

## Architecture Patterns

### Key Constraint: All Implementation Is Prompt Enrichment

This is the same architectural pattern as Phase 14. Command files are **instruction documents that Claude reads and follows**. "Adding error logging" means adding a new step to `build.md` that says "If the phase encountered failures, use the Write tool to append error records to errors.json." Claude then follows those instructions at runtime.

There is NO bash, NO jq, NO Python to write. The prompt tells Claude what to do.

### Pattern 1: State File Initialization in init.md

**What:** init.md gains a new step to create the three state files with empty/initial schemas.
**When:** Every `/ant:init` invocation.
**Where in command flow:** After Step 3 (Write Colony State), before Step 4 (Emit INIT Pheromone). The new step creates all three files. This is the natural insertion point because colony state is already being written.

Current init.md flow:
```
Step 1: Validate Input
Step 2: Read Current State
Step 3: Write Colony State (COLONY_STATE.json)
Step 4: Emit INIT Pheromone (pheromones.json)
Step 5: Display Result
```

Phase 15 flow:
```
Step 1: Validate Input
Step 2: Read Current State
Step 3: Write Colony State (COLONY_STATE.json)
Step 4: Create State Files (errors.json, memory.json, events.json) <-- NEW
Step 5: Emit INIT Pheromone (pheromones.json)
Step 6: Write Init Event (events.json) <-- NEW
Step 7: Display Result
```

### Pattern 2: Error Logging in build.md

**What:** build.md gains error-writing logic in Step 6 (Record Outcome).
**When:** After the spawned ant returns and reports results.
**Where in command flow:** Step 6 already updates PROJECT_PLAN.json and COLONY_STATE.json. Add error logging here.

Key insight: The spawned ant's report includes an "Issues" section. build.md should parse that report, identify failures, and write error records to errors.json. It should also check for pattern flagging (3+ occurrences of same category).

Current build.md Step 6:
```
After the ant returns:
- Mark tasks as completed/failed in PROJECT_PLAN.json
- Set phase status
- Update COLONY_STATE.json
```

Phase 15 Step 6:
```
After the ant returns:
- Mark tasks as completed/failed in PROJECT_PLAN.json
- Set phase status
- Update COLONY_STATE.json
- Read errors.json
- For each failure/issue in the ant's report, append an error record
- Check pattern flagging: count errors by category, flag if >= 3
- Write updated errors.json
- Write phase_complete or phase_failed event to events.json
```

### Pattern 3: Event Writing as a Cross-Cutting Concern

**What:** Commands write event records to events.json when state changes occur.
**When:** On state-changing operations (init, build start, build complete, continue, pheromone emit).
**Where:** Each command gets an "Append Event" step near the end of its flow, before the Display step.

Event-writing commands and their event types:
| Command | Event Type | When |
|---------|------------|------|
| init.md | `colony_initialized` | After state files created |
| build.md | `phase_started` | After state updated to EXECUTING |
| build.md | `phase_completed` / `phase_failed` | After recording outcome |
| build.md | `error_logged` | When an error is written to errors.json |
| continue.md | `phase_advanced` | After advancing to next phase |
| focus.md | `pheromone_emitted` | After writing FOCUS signal |
| redirect.md | `pheromone_emitted` | After writing REDIRECT signal |
| feedback.md | `pheromone_emitted` | After writing FEEDBACK signal |

### Pattern 4: Memory Extraction at Phase Boundaries

**What:** continue.md extracts learnings from the completed phase before advancing.
**When:** At phase boundary, before updating colony state.
**Where in command flow:** New step between "Determine Next Phase" and "Clean Expired Pheromones."

Current continue.md flow:
```
Step 1: Read State
Step 2: Determine Next Phase
Step 3: Clean Expired Pheromones
Step 4: Update Colony State
Step 5: Display Result
```

Phase 15 flow:
```
Step 1: Read State (add errors.json, memory.json, events.json to reads)
Step 2: Determine Next Phase
Step 3: Extract Phase Learnings <-- NEW
Step 4: Clean Expired Pheromones
Step 5: Write Phase Advanced Event <-- NEW
Step 6: Update Colony State
Step 7: Display Result (show learnings extracted)
```

### Pattern 5: Decision Logging in Pheromone Commands

**What:** focus.md, redirect.md, feedback.md log decisions to memory.json.
**When:** After writing the pheromone signal.
**Where:** New step after "Append Signal" and before "Display Result."

These are "significant decisions" -- the user is deliberately guiding the colony. Log:
- What decision was made (the pheromone content)
- Why (the pheromone type indicates intent: FOCUS = priority, REDIRECT = constraint, FEEDBACK = adjustment)
- When (timestamp)
- Context (current phase, colony state)

### Anti-Patterns to Avoid

- **Over-engineering schemas:** The v2 ErrorRecord had 20+ fields. v3 needs 7 fields max. Keep it minimal.
- **Deep nesting:** Don't create `{errors: {by_category: {syntax: [...], runtime: [...]}}}`. Use flat arrays and count at read time.
- **Separate pattern tracking:** Don't create a separate `patterns.json`. Pattern detection is `count errors where category == X`. Do it inline when reading errors.json.
- **Event delivery tracking:** events.json is a LOG, not a queue. No subscriptions, no delivery tracking, no acknowledgments. Workers filter by timestamp.
- **Memory layers:** Don't recreate triple-layer memory. One flat `memory.json` with three arrays: `phase_learnings`, `decisions`, `patterns`.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Pattern detection | Complex pattern matching algorithm | Simple array count by category field | 3+ occurrences of same category is just a filter+count |
| Event pub/sub | Subscription/delivery system | Flat event log + timestamp filter | Workers read events.json and filter by `timestamp > last_check`. No delivery tracking needed. |
| Memory compression | DAST compression algorithm | Claude's natural summarization | When extracting learnings, Claude naturally summarizes. No algorithm needed. |
| Schema validation | JSON Schema validator | Trust Claude's Write tool output | Claude follows the schema template in the prompt. No runtime validation needed. |
| ID generation | UUID library | `{type}_{unix_timestamp}_{random_hex}` | Already established pattern for pheromone IDs. Claude generates these inline. |

**Key insight:** Every "system" from v2 that required Python/bash code is replaced by a prompt instruction that Claude follows. The "error tracking system" is: "Read errors.json, append a record, count by category, flag if >= 3, write back." No code needed.

## Common Pitfalls

### Pitfall 1: Bloating Command Prompts

**What goes wrong:** Adding error logging, event writing, memory extraction, and decision logging makes command prompts grow from ~90-200 lines to 500+ lines, becoming unwieldy and confusing for Claude to follow.
**Why it happens:** Each new state file adds Read + Process + Write steps to every command.
**How to avoid:** Keep new steps concise. Use inline JSON templates (not verbose explanations). Group related operations (e.g., "Read errors.json and events.json" in one Read call, not two). Target: no command exceeds ~250 lines.
**Warning signs:** If a command prompt has more than 10 numbered steps, it's too complex.

### Pitfall 2: Read-Modify-Write Race Conditions

**What goes wrong:** Two concurrent commands read errors.json, both append, and the second write overwrites the first's additions.
**Why it happens:** Claude commands are prompt-based and may theoretically overlap (though unlikely in practice since they're user-initiated).
**How to avoid:** This is NOT a real concern for this system. Commands are run one at a time by the user. The sequential Read/Write tool pattern is fine. Don't add locking or atomic write complexity.
**Warning signs:** If you're thinking about file locks, you're over-engineering.

### Pitfall 3: Unbounded JSON File Growth

**What goes wrong:** errors.json and events.json grow forever, eventually becoming too large for Claude to read efficiently.
**Why it happens:** Every error and event is appended but nothing is ever removed.
**How to avoid:** Add a retention policy in the schema design. Keep last 50 errors, last 100 events. When writing, if array exceeds limit, trim oldest entries. Add this as a simple instruction: "If the errors array has more than 50 entries, remove the oldest entries to keep only 50."
**Warning signs:** JSON files exceeding ~100 entries.

### Pitfall 4: Forgetting to Update Step Numbers

**What goes wrong:** Adding steps to a command changes step numbering. The step progress display at the end references old numbers.
**Why it happens:** Phase 14 added step progress indicators (`Step 1: Validate`, etc.) to init.md, build.md, continue.md. Adding new steps means renumbering.
**How to avoid:** When modifying a command, update BOTH the step instructions AND the step progress display at the end. The step progress display is in the last Display step of each command.
**Warning signs:** Step count mismatch between instructions and display.

### Pitfall 5: Inconsistent Event Schema

**What goes wrong:** Different commands write events with different field sets (one includes `phase`, another doesn't; one uses `type: "phase_start"`, another uses `type: "PHASE_START"`).
**Why it happens:** Event writing is added to 8 different command files independently.
**How to avoid:** Define the event schema ONCE in the research and reference it consistently. Use lowercase_snake_case for event types. Every event has exactly: `{id, type, source, content, timestamp}`. The `source` identifies which command wrote it.
**Warning signs:** Inconsistent casing or field presence across commands.

### Pitfall 6: Memory Extraction Producing Empty Learnings

**What goes wrong:** continue.md's learning extraction step produces generic boilerplate like "Phase completed successfully" instead of actual learnings.
**Why it happens:** The prompt instruction is too vague ("extract learnings from this phase").
**How to avoid:** Give Claude specific extraction prompts: "What patterns were used? What errors were encountered? What decisions were made? What should the colony remember for future phases?" Reference errors.json and events.json to ground the extraction in actual data.
**Warning signs:** All learning entries look the same regardless of what happened.

## Code Examples

### Schema: errors.json (Initial State)

```json
{
  "errors": [],
  "flagged_patterns": []
}
```

### Schema: errors.json (After Use)

```json
{
  "errors": [
    {
      "id": "err_1706900000_a1b2",
      "category": "runtime",
      "severity": "high",
      "description": "Test suite failed with 3 assertion errors in auth module",
      "root_cause": "Missing null check on user token before validation",
      "phase": 2,
      "task_id": "2.3",
      "timestamp": "2026-02-03T10:00:00Z"
    }
  ],
  "flagged_patterns": [
    {
      "category": "runtime",
      "count": 3,
      "first_seen": "2026-02-03T08:00:00Z",
      "last_seen": "2026-02-03T10:00:00Z",
      "flagged_at": "2026-02-03T10:00:00Z",
      "description": "Recurring runtime errors -- 3 occurrences detected"
    }
  ]
}
```

**Design rationale:** The `flagged_patterns` array is separate from `errors` to make it easy to display in status.md. When build.md writes a new error, it counts errors by category. If any category reaches 3+, it adds/updates an entry in `flagged_patterns`. This is simpler than inline pattern detection at read time.

### Schema: memory.json (Initial State)

```json
{
  "phase_learnings": [],
  "decisions": [],
  "patterns": []
}
```

### Schema: memory.json (After Use)

```json
{
  "phase_learnings": [
    {
      "id": "learn_1706900000_c3d4",
      "phase": 1,
      "phase_name": "Project Setup",
      "learnings": [
        "TypeScript strict mode caught 12 type errors early",
        "ESLint flat config is simpler than .eslintrc"
      ],
      "errors_encountered": 1,
      "timestamp": "2026-02-03T10:00:00Z"
    }
  ],
  "decisions": [
    {
      "id": "dec_1706900000_e5f6",
      "type": "focus",
      "content": "WebSocket security",
      "context": "Phase 2 in progress, building API layer",
      "phase": 2,
      "timestamp": "2026-02-03T11:00:00Z"
    }
  ],
  "patterns": []
}
```

**Design rationale:** Three arrays for different knowledge types. `phase_learnings` are extracted by continue.md at phase boundaries. `decisions` are logged by focus/redirect/feedback commands. `patterns` are extracted by the architect ant (Phase 16 concern, not Phase 15 -- include the array but leave empty). Each entry has an ID and timestamp for ordering and deduplication.

### Schema: events.json (Initial State)

```json
{
  "events": []
}
```

### Schema: events.json (After Use)

```json
{
  "events": [
    {
      "id": "evt_1706900000_a1b2",
      "type": "colony_initialized",
      "source": "init",
      "content": "Colony initialized with goal: Build a REST API",
      "timestamp": "2026-02-03T08:00:00Z"
    },
    {
      "id": "evt_1706900001_c3d4",
      "type": "phase_started",
      "source": "build",
      "content": "Phase 1: Project Setup started",
      "timestamp": "2026-02-03T08:01:00Z"
    },
    {
      "id": "evt_1706900100_e5f6",
      "type": "phase_completed",
      "source": "build",
      "content": "Phase 1: Project Setup completed (4/4 tasks done)",
      "timestamp": "2026-02-03T09:40:00Z"
    },
    {
      "id": "evt_1706900200_g7h8",
      "type": "pheromone_emitted",
      "source": "focus",
      "content": "FOCUS: WebSocket security (strength 0.7, half-life 1hr)",
      "timestamp": "2026-02-03T10:00:00Z"
    }
  ]
}
```

**Design rationale:** Simplest possible event log. Every event has exactly 5 fields. The `source` field identifies the command that wrote the event. The `content` field is a human-readable description. Workers can filter by `timestamp` and `type` to find relevant events. Maximum 100 events retained (trim oldest on write).

### Example: Error Logging Prompt Addition for build.md

This is what the prompt text looks like when added to build.md Step 6:

```markdown
### Step 6: Record Outcome

After the ant returns, use the Read tool to read `.aether/data/errors.json` and `.aether/data/events.json`.

**Update PROJECT_PLAN.json** (existing logic)...

**Log Errors:** If the ant reported any failures or issues:
For each failure, append an error record to the `errors` array in errors.json:

{
  "id": "err_<unix_timestamp>_<4_random_hex>",
  "category": "<one of: syntax, import, runtime, type, spawning, phase, verification, api, file, logic, performance, security>",
  "severity": "<one of: critical, high, medium, low>",
  "description": "<what went wrong>",
  "root_cause": "<why it happened, if apparent from the ant's report>",
  "phase": <phase_number>,
  "task_id": "<task_id if applicable>",
  "timestamp": "<ISO-8601 UTC>"
}

**Check Pattern Flagging:** Count errors in the `errors` array by `category`. If any category has 3 or more errors and is not already in `flagged_patterns`, add:

{
  "category": "<the category>",
  "count": <total count>,
  "first_seen": "<timestamp of earliest error in this category>",
  "last_seen": "<timestamp of latest error in this category>",
  "flagged_at": "<current ISO-8601 UTC>",
  "description": "Recurring <category> errors -- <count> occurrences detected"
}

If the category already exists in `flagged_patterns`, update its `count`, `last_seen`, and `description`.

If the `errors` array exceeds 50 entries, remove the oldest entries to keep only 50.

Use the Write tool to write the updated errors.json.

**Write Event:** Append to the `events` array in events.json:

{
  "id": "evt_<unix_timestamp>_<4_random_hex>",
  "type": "<phase_completed or phase_failed>",
  "source": "build",
  "content": "Phase <id>: <name> <completed|failed> (<completed_count>/<total_count> tasks done)",
  "timestamp": "<ISO-8601 UTC>"
}

If the `events` array exceeds 100 entries, remove the oldest entries to keep only 100.

Use the Write tool to write the updated events.json.
```

### Example: Memory Extraction Prompt Addition for continue.md

```markdown
### Step 3: Extract Phase Learnings

Read `.aether/data/errors.json`, `.aether/data/memory.json`, and `.aether/data/events.json`.

Review the completed phase. Extract learnings by analyzing:
- Tasks completed in this phase (from PROJECT_PLAN.json)
- Errors encountered during this phase (from errors.json, filter by `phase` field)
- Events that occurred (from events.json, filter by recent timestamps)
- Flagged patterns (from errors.json `flagged_patterns` array)

Create a phase learning entry and append to the `phase_learnings` array in memory.json:

{
  "id": "learn_<unix_timestamp>_<4_random_hex>",
  "phase": <phase_number>,
  "phase_name": "<phase name>",
  "learnings": [
    "<specific thing learned -- what worked, what didn't, what to remember>",
    "<another specific learning>"
  ],
  "errors_encountered": <count of errors with this phase number>,
  "timestamp": "<ISO-8601 UTC>"
}

Learnings should be SPECIFIC and ACTIONABLE, not generic. Good: "TypeScript strict mode caught 12 type errors early." Bad: "Phase completed successfully."

If the `phase_learnings` array exceeds 20 entries, remove the oldest entries to keep only 20.

Use the Write tool to write the updated memory.json.
```

### Example: Decision Logging Prompt Addition for focus.md

```markdown
### Step 4: Log Decision

Read `.aether/data/memory.json`.

Append a decision record to the `decisions` array:

{
  "id": "dec_<unix_timestamp>_<4_random_hex>",
  "type": "focus",
  "content": "<the focus area>",
  "context": "Phase <current_phase> -- <current state description>",
  "phase": <current_phase from COLONY_STATE.json>,
  "timestamp": "<ISO-8601 UTC>"
}

If the `decisions` array exceeds 30 entries, remove the oldest entries to keep only 30.

Use the Write tool to write the updated memory.json.
```

## State of the Art

| Old Approach (v2) | Current Approach (v3) | When Changed | Impact |
|---|---|---|---|
| `error_prevention.py` (686 lines, Python runtime) | Prompt instructions in build.md + errors.json | v3 rebuild (2026-02-03) | No Python needed; Claude follows prompt instructions to write JSON |
| `triple_layer_memory.py` (2,543 lines across 5 files) | Single memory.json with 3 arrays + prompt instructions | v3 rebuild | Working memory is Claude's native context; only persistent learnings stored |
| `event-bus.sh` (890 lines, bash pub/sub) | events.json log + timestamp filtering | v3 rebuild | No subscriptions/delivery; workers read and filter by timestamp |
| ErrorCategory enum (15 categories) | 12 string categories in prompt template | v3 rebuild | Claude picks from list; no enum needed |
| ErrorPattern class with Bayesian tracking | Simple count-by-category with threshold 3 | v3 rebuild | Pattern detection is just array filtering |

**Deprecated/outdated:**
- Python runtime for state management -- replaced by Claude Read/Write tools
- Bash event bus -- replaced by simple JSON log
- Triple-layer memory hierarchy -- replaced by flat JSON with 3 arrays
- Separate errors/ directory with individual error files -- replaced by single errors.json

## Recommended Error Categories

Simplified from v2's 15 categories to 12, removing ones that don't apply in the Claude-native context:

| Category | Description | Example |
|----------|-------------|---------|
| `syntax` | Code syntax errors | Missing bracket, invalid JSON |
| `import` | Import/module errors | Module not found, circular dependency |
| `runtime` | Runtime exceptions | Null reference, division by zero |
| `type` | Type errors | Wrong argument type, missing field |
| `spawning` | Agent spawning failures | Task tool failure, timeout |
| `phase` | Phase execution errors | Task blocked, dependency unmet |
| `verification` | Test/validation failures | Tests fail, lint errors |
| `api` | External API failures | HTTP errors, rate limits |
| `file` | File I/O errors | File not found, permission denied |
| `logic` | Logic bugs | Wrong output, incorrect behavior |
| `performance` | Performance issues | Timeout, excessive resource use |
| `security` | Security vulnerabilities | Exposed credentials, injection |

## Recommended Event Types

| Event Type | Source Command | When Emitted |
|------------|---------------|--------------|
| `colony_initialized` | init | After all state files created |
| `phase_started` | build | After state set to EXECUTING |
| `phase_completed` | build | After successful phase completion |
| `phase_failed` | build | After phase failure |
| `error_logged` | build | When error written to errors.json |
| `pattern_flagged` | build | When error category reaches 3+ |
| `phase_advanced` | continue | After advancing to next phase |
| `learnings_extracted` | continue | After writing to memory.json |
| `pheromone_emitted` | focus/redirect/feedback | After writing pheromone signal |

## Retention Limits

| File | Array | Max Entries | Rationale |
|------|-------|-------------|-----------|
| errors.json | `errors` | 50 | ~50 errors is enough history; older errors are less relevant |
| errors.json | `flagged_patterns` | 20 | Patterns don't expire, but 20 is plenty |
| events.json | `events` | 100 | Event log for recent context; workers filter by timestamp |
| memory.json | `phase_learnings` | 20 | One per phase; 20 phases is a long project |
| memory.json | `decisions` | 30 | Most recent decisions are most relevant |
| memory.json | `patterns` | 20 | Extracted patterns (Phase 16 concern) |

## Commands Modification Summary

### init.md (Currently 128 lines)

**Changes:**
- Step 3: Add creation of errors.json, memory.json, events.json with initial schemas (existing Write tool call expands)
- New step: Write `colony_initialized` event to events.json
- Update step numbering and step progress display

**Estimated size after:** ~160 lines

### build.md (Currently 198 lines)

**Changes:**
- Step 2: Add errors.json and events.json to Read calls
- Step 4 (Update State): Add `phase_started` event write to events.json
- Step 6 (Record Outcome): Add error logging, pattern flagging, `phase_completed`/`phase_failed` event write
- Update step numbering and step progress display

**Estimated size after:** ~260 lines (largest command -- has the most state writing)

### continue.md (Currently 90 lines)

**Changes:**
- Step 1: Add errors.json, memory.json, events.json to Read calls
- New Step 3: Extract Phase Learnings (between Determine Next Phase and Clean Pheromones)
- New step: Write `phase_advanced` and `learnings_extracted` events
- Update step numbering and step progress display

**Estimated size after:** ~150 lines

### focus.md (Currently 73 lines)

**Changes:**
- Step 2: Add COLONY_STATE.json read (already there) and memory.json, events.json reads
- New Step 4: Log decision to memory.json
- New Step 5: Write `pheromone_emitted` event to events.json
- Update step numbering

**Estimated size after:** ~105 lines

### redirect.md (Currently 75 lines)

**Changes:** Same pattern as focus.md
**Estimated size after:** ~107 lines

### feedback.md (Currently 76 lines)

**Changes:** Same pattern as focus.md
**Estimated size after:** ~108 lines

### status.md (Currently 151 lines)

**Note:** ERR-04 requires status.md to display errors and patterns. However, the phase description says "ERR-04, but that's Phase 17/DASH." The status.md dashboard enrichment is a Phase 17 concern. For Phase 15, status.md does NOT need modification. The state files just need to exist and be written to correctly. Phase 17 will add reading/displaying.

**Phase 15 changes to status.md:** None required. ERR-04 display is deferred to Phase 17.

## Open Questions

1. **Should build.md also write a `phase_started` event at Step 4?**
   - What we know: The requirements say "state-changing commands write event records." Starting a phase is a state change.
   - Recommendation: Yes, write both `phase_started` and `phase_completed`/`phase_failed` events. This gives workers context about what is happening, not just what completed.

2. **Should errors.json store the ant's full report or just extracted fields?**
   - What we know: The ant's report can be verbose (hundreds of lines). Error records should be concise.
   - Recommendation: Store extracted fields only (category, severity, description, root_cause). The full report is ephemeral -- Claude's context has it during the session.

3. **Should memory extraction happen automatically or spawn an architect ant?**
   - What we know: Phase 16 (Worker Knowledge) enriches worker specs. Phase 15 establishes the data layer.
   - Recommendation: In Phase 15, continue.md extracts learnings INLINE (Claude analyzes the phase data and writes learnings directly). No spawning needed. Phase 16 may add architect-ant-based pattern extraction later.

4. **ERR-04 (status.md displays errors) -- Phase 15 or Phase 17?**
   - What we know: The requirements list ERR-04 under Phase 15, but the phase context notes "ERR-04, but that's Phase 17/DASH."
   - Recommendation: Include minimal error display in status.md now (just show error count and any flagged patterns in the NEXT ACTIONS section). This satisfies ERR-04 without the full dashboard treatment that Phase 17 will provide. Alternatively, defer entirely if Phase 17 is confirmed to handle it.

## Sources

### Primary (HIGH confidence)
- Current command files in `.claude/commands/ant/*.md` -- Read directly, exact current state
- Current data files in `.aether/data/*.json` -- Read directly, exact current schemas
- V3_LOST_FEATURES.md -- Comprehensive mapping of what was lost and how to restore
- HANDOFF.md -- Complete v3 rebuild history and design decisions

### Secondary (MEDIUM confidence)
- `git show 020cecf:.aether/error_prevention.py` -- v2 error tracking schema (ErrorRecord, ErrorCategory, ErrorPattern)
- `git show c002111:.aether/data/events.json` -- v2 event bus schema
- `git show 3f50db9:.aether/data/memory.json` -- v2 memory schema
- Phase 14 RESEARCH.md and PLAN files -- Established patterns for prompt enrichment

### Tertiary (LOW confidence)
- None -- all findings are based on direct codebase analysis

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH -- no external dependencies; all patterns established in prior phases
- Architecture: HIGH -- exact command files read; insertion points identified precisely
- Pitfalls: HIGH -- based on actual codebase analysis and understanding of Claude's Read/Write patterns
- Schemas: HIGH -- derived from v2 Python code and simplified for v3 Claude-native approach

**Research date:** 2026-02-03
**Valid until:** 2026-03-03 (stable -- no external dependencies to go stale)
