# Architecture Research: Smart Init System for Aether v2.5

**Domain:** Aether colony initialization system (slash commands, shell utils, QUEEN.md governance, prompt generation)
**Researched:** 2026-03-27
**Confidence:** HIGH (based on direct codebase analysis of init.md, plan.md, colonize.md, queen.sh, pheromone.sh colony-prime, session.sh, state-api.sh, and all template files)

---

## Executive Summary

The current `/ant:init` command (388 lines in `.claude/commands/ant/init.md`) is a mechanical file-setup operation: parse the goal string, write COLONY_STATE.json from a template, initialize constraints, pheromones, midden, learning-observations, create a session file, and register the repo. It does zero research about the repo being initialized, generates no structured prompt, and offers no approval step.

The smart init milestone needs to transform this into an intelligent first step that (1) scans the repo to understand its structure, (2) generates a structured colony initialization prompt combining the user's natural language goal with repo context, (3) presents the prompt for approval before proceeding, and (4) manages QUEEN.md as a living colony charter with new governance sections (intent, vision, goals, architecture).

The key architectural insight: **init.md is a Markdown prompt executed by Claude Code's LLM**. It does not run as a traditional program. Every step is an instruction the LLM follows, calling tools (Bash, Read, Write, Glob) at each step. This means "approval loops" and "prompt generation" must work within Claude Code's execution model -- the LLM can present text to the user and wait for a response, then continue.

---

## Part 1: System Overview

### Current Init Architecture

```
User types: /ant:init "Build a REST API with authentication"
    |
    v
init.md (388 lines of Markdown instructions for Claude Code LLM)
    |
    +-- Step 1: Validate input (goal string not empty)
    +-- Step 1.5: Verify aether-utils.sh exists
    +-- Step 1.6: queen-init (create QUEEN.md from template)
    +-- Step 2: Read COLONY_STATE.json (check existing state)
    +-- Step 2.6: Load prior colony knowledge (completion-report.md)
    +-- Step 3: Write COLONY_STATE.json (from template with substitutions)
    +-- Step 4: Initialize constraints.json (from template)
    +-- Step 4.5: Initialize runtime files (pheromones, midden, learning-obs)
    +-- Step 5: context-update init "$ARGUMENTS" (creates CONTEXT.md)
    +-- Step 6: validate-state colony
    +-- Step 6.5: Detect nestmates
    +-- Step 6.6: Register repo (silent, non-blocking)
    +-- Step 6.7: Seed QUEEN.md from hive (non-blocking)
    +-- Step 7: Display result
    +-- Step 8: session-init
```

### Proposed Smart Init Architecture

```
User types: /ant:init "Build a REST API with authentication"
    |
    v
init.md (refactored, modular)
    |
    +-- Step 1: Validate input [EXISTING]
    +-- Step 1.5: Verify aether-utils.sh exists [EXISTING]
    |
    +-- Step 2: REPO SCAN (NEW)
    |   |-- Lightweight scan: package manifests, entry points, config
    |   |-- Check survey freshness: survey files exist and recent?
    |   |-- Check chambers: prior colonies exist?
    |   |-- Domain detection: what kind of project is this?
    |   +-- Output: scan_summary JSON (tech, size, survey status)
    |
    +-- Step 3: GENERATE COLONY PROMPT (NEW)
    |   |-- Combine user goal + scan_summary + hive wisdom
    |   |-- Produce structured colony initialization document:
    |   |   - Intent (what the user wants)
    |   |   - Vision (what success looks like)
    |   |   - Governance (rules, constraints)
    |   |   - Goals (measurable outcomes)
    |   |   - Architecture notes (if survey available)
    |   +-- Output: colony_prompt (Markdown text)
    |
    +-- Step 4: APPROVAL LOOP (NEW)
    |   |-- Display generated prompt to user
    |   |-- Wait for user response (approve / edit / cancel)
    |   |-- If approved: proceed to write files
    |   |-- If edited: regenerate from user modifications
    |   |-- If cancelled: stop, no files written
    |
    +-- Step 5: QUEEN.MD GOVERNANCE UPDATE (NEW)
    |   |-- If QUEEN.md exists and has governance sections: UPDATE
    |   |-- If QUEEN.md exists but no governance: ADD new sections
    |   |-- If QUEEN.md doesn't exist: CREATE with governance sections
    |   +-- New sections: ## Intent, ## Vision, ## Governance, ## Goals, ## Architecture
    |
    +-- Step 6: Write COLONY_STATE.json [EXISTING - Step 3]
    +-- Step 7: Initialize constraints [EXISTING - Step 4]
    +-- Step 8: Initialize runtime files [EXISTING - Step 4.5]
    +-- Step 9: context-update [EXISTING - Step 5]
    +-- Step 10: Validate state [EXISTING - Step 6]
    +-- Step 11: Register + seed [EXISTING - Steps 6.5-6.7]
    +-- Step 12: Display result [EXISTING - Step 7]
    +-- Step 13: session-init [EXISTING - Step 8]
```

---

## Part 2: Question 1 -- Where Does Lightweight Repo Scanning Live?

### Options Considered

| Option | Location | Pros | Cons |
|--------|----------|------|------|
| A) New utils module | `.aether/utils/scan.sh` | Clean separation, testable | New module to maintain, 10th domain module |
| B) Extend existing domain-detect | `queen.sh:_domain_detect()` | Already exists, just expand | queen.sh is 1242 lines, already heavy |
| C) Inline in init.md | Steps within init.md | No new code | Not testable, not reusable, init.md already 388 lines |
| D) New subcommands in existing module | `state-api.sh` or `session.sh` | Uses existing infrastructure | Wrong domain concern |

### Recommendation: Option A -- New `scan.sh` utils module

**Rationale:**

1. **Clean domain boundary.** Repo scanning is a distinct concern from state management, session tracking, or queen wisdom. It deserves its own module.

2. **Already have precedent.** The modularization in v2.1 (Phase 13) extracted 9 domain modules from the monolith. The pattern is established: each module has a single responsibility, is sourced by aether-utils.sh, and exposes functions with `_module_function` naming.

3. **Testable in isolation.** Shell functions in `.aether/utils/scan.sh` can be tested independently via the bash test infrastructure (tests/bash/).

4. **Reusable by other commands.** Both `/ant:init` and `/ant:colonize` need repo scanning. Currently, colonize.md (Step 2) does inline scanning with Glob/Read tool calls. A shared `scan.sh` module means both commands call the same functions.

5. **Keeps init.md manageable.** init.md is already 388 lines. Adding scanning logic inline would push it past 500 lines, which is the established threshold for splitting into playbooks.

### Module Structure

```
.aether/utils/scan.sh
  _scan_repo()          -- Full repo scan, returns JSON with tech/size/survey status
  _scan_quick()         -- Lightweight scan (package manifests + entry points only)
  _scan_survey_status() -- Check if territory survey exists and is fresh
  _scan_chambers()      -- Check for prior colony chambers
  _scan_domain()        -- Enhanced domain detection (extends existing _domain_detect)
```

### Integration with aether-utils.sh

Add to the source block (after line 42):
```bash
[[ -f "$SCRIPT_DIR/utils/scan.sh" ]] && source "$SCRIPT_DIR/utils/scan.sh"
```

Add dispatch cases:
```bash
scan-repo) _scan_repo "$@" ;;
scan-quick) _scan_quick "$@" ;;
scan-survey-status) _scan_survey_status "$@" ;;
scan-chambers) _scan_chambers "$@" ;;
```

### What _scan_repo Returns

```json
{
  "ok": true,
  "result": {
    "tech_stack": {
      "language": "typescript",
      "framework": "express",
      "runtime": "node"
    },
    "entry_points": ["src/index.ts", "src/server.ts"],
    "project_type": "api",
    "has_tests": true,
    "has_ci": true,
    "file_count": 47,
    "directory_count": 12,
    "domain_tags": ["node", "typescript"],
    "survey": {
      "exists": true,
      "last_surveyed": "2026-03-20T14:00:00Z",
      "is_fresh": false,
      "stale_documents": ["PROVISIONS.md"]
    },
    "chambers": {
      "count": 2,
      "last_colony_goal": "Add authentication"
    },
    "prior_knowledge": {
      "has_completion_report": true,
      "instinct_count": 3,
      "learning_count": 5
    }
  }
}
```

### Relationship to Existing domain-detect

The existing `_domain_detect()` in queen.sh (lines 1078-1094) does simple file-presence detection (checks for package.json, Cargo.toml, go.mod, etc.). The new `_scan_domain()` should call `_domain_detect()` first, then enrich with additional signals:

- Parse package.json for framework detection (check dependencies)
- Check for tsconfig.json to distinguish TypeScript from JavaScript
- Check for test directories (test/, tests/, __tests__/) to detect testing patterns
- Count files in src/ vs lib/ to estimate project size

This means `_domain_detect()` stays in queen.sh (it is also called by init.md Step 6.6 for registry), and `_scan_domain()` in scan.sh wraps it with enrichment.

---

## Part 3: Question 2 -- How Does the Prompt Generator Work?

### The Core Idea

The prompt generator transforms a user's natural language goal + repo scan results + hive wisdom into a structured colony initialization document. This document becomes the "colony charter" -- it defines what the colony is trying to achieve and how it should operate.

### Where It Lives

**The prompt generator should NOT be a shell subcommand.** It is an LLM operation that requires synthesis and judgment. Shell scripts are for deterministic operations (file I/O, JSON manipulation, state transitions). The prompt generator needs the LLM's ability to interpret natural language, reason about project structure, and produce coherent prose.

**Implementation: A step within init.md** (Steps 2-3 of the proposed architecture). The step instructs the Claude Code LLM to:

1. Read the scan results from `_scan_repo`
2. Read hive wisdom via `hive-read`
3. Read prior completion-report.md if it exists
4. Synthesize these into a structured Markdown document

### Structured Colony Prompt Format

```markdown
# Colony Charter: {goal_summary}

## Intent
{1-2 paragraphs: what the user wants to achieve, interpreted from their goal string}

## Vision
{1-2 paragraphs: what success looks like for this colony}

## Governance
### Constraints
- {rules from REDIRECT pheromones if any}
- {constraints inferred from scan (e.g., "must maintain existing API compatibility")}

### Quality Standards
- {testing expectations from scan}
- {code style from DISCIPLINES.md if survey available}

### Communication
- {user preferences from QUEEN.md or hub}

## Goals
1. {primary goal from user input}
2. {inferred goals from scan}
3. {goals from prior colony knowledge if applicable}

## Architecture Notes
{if survey available: key patterns from BLUEPRINT.md, tech stack from PROVISIONS.md}
{if no survey: "Run /ant:colonize for deeper architectural context"}

## Prior Knowledge
{if completion-report.md exists: inherited instincts and learnings}
{if hive wisdom available: relevant cross-colony patterns}
```

### Why This Works in Claude Code's Execution Model

In Claude Code, a slash command is a Markdown prompt. The LLM reads it and follows the instructions step by step. Between steps, the LLM can:

- Call Bash to run shell commands
- Call Read to read files
- Call Write to create files
- Present text to the user and wait for response

So the flow is:

```
Step 2 (in init.md): Run _scan_repo via Bash tool
    |
    v
LLM has scan results in context
    |
Step 3 (in init.md): LLM synthesizes prompt from:
    - scan results (JSON from Step 2)
    - user goal ($ARGUMENTS)
    - hive wisdom (from hive-read via Bash)
    - prior knowledge (from completion-report.md via Read)
    |
    v
LLM generates colony_prompt (Markdown text)
    |
Step 4 (in init.md): LLM presents colony_prompt to user
    |
    v
User responds: "looks good" / "change X" / "cancel"
    |
    v
LLM proceeds or adjusts based on response
```

### No New Shell Subcommand Needed

The prompt generator is purely an LLM operation within init.md. No new `prompt-generate` subcommand is needed. The scan results provide the data; the LLM does the synthesis.

However, the scan results themselves DO need to be available as a shell subcommand so the LLM can call them via Bash:

```
bash .aether/aether-utils.sh scan-repo
```

---

## Part 4: Question 3 -- How Does the Approval Loop Work?

### Claude Code's Execution Model

Claude Code executes slash commands as sequential instructions in a Markdown prompt. The LLM processes the prompt step by step. **Between steps, the LLM can pause and wait for user input.**

This is how existing commands already work. For example, in init.md Step 2, if state already exists, the LLM presents a warning and the user responds. The LLM then continues based on the response.

### Proposed Approval Loop Pattern

In init.md, after generating the colony prompt (Step 3), add a step that:

```
### Step 4: Present Colony Charter for Approval

Display the generated colony charter:

{colony_prompt}

Then ask:

```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
   C O L O N Y   C H A R T E R
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Review the colony charter above. Options:
  1. Approve -- proceed with initialization
  2. Edit -- tell me what to change
  3. Cancel -- do not initialize

Waiting for your decision.
```

Wait for the user's response.

- If the user approves: proceed to Step 5 (write files)
- If the user requests edits: modify the colony_prompt based on their feedback, re-present, and wait again
- If the user cancels: output "Colony initialization cancelled. No files were written." and STOP
```

### Why This Works

This is identical to how the existing init.md Step 2 works when detecting existing state. The LLM presents information and waits for user response. No special mechanism is needed -- this is Claude Code's normal execution model.

### Potential Pitfall: Context Window Pressure

The colony prompt + approval display + subsequent file writes all happen in a single conversation turn. If the scan results or colony prompt are very large, they consume context that could be needed for later steps.

**Mitigation:** Keep the colony prompt under 2000 characters. The scan results should be summarized, not raw JSON. The approval display should show the prompt, not the full scan data.

---

## Part 5: Question 4 -- How Does QUEEN.md Governance Update Work?

### Current QUEEN.md Structure (v2 format)

```markdown
# QUEEN.md -- Colony Wisdom

> Last evolved: {TIMESTAMP}
> Wisdom version: 2.0.0

---

## User Preferences
{user preferences}

---

## Codebase Patterns
{validated patterns and anti-patterns}

---

## Build Learnings
{what worked and what failed}

---

## Instincts
{high-confidence behavioral patterns}

---

## Evolution Log
{table of changes}

---

<!-- METADATA {json} -->
```

### Proposed QUEEN.md Structure (v3 format)

```markdown
# QUEEN.md -- Colony Charter & Wisdom

> Last evolved: {TIMESTAMP}
> Wisdom version: 3.0.0
> Colony goal: {goal}

---

## Intent
{what this colony is trying to achieve}

---

## Vision
{what success looks like}

---

## Governance
### Constraints
{hard rules}

### Quality Standards
{testing, code style, review expectations}

---

## Goals
{measurable outcomes}

---

## Architecture Notes
{tech stack, patterns, key decisions}

---

## User Preferences
{communication style, expertise level, decision patterns}

---

## Codebase Patterns
{validated approaches and anti-patterns}

---

## Build Learnings
{what worked and what failed}

---

## Instincts
{high-confidence behavioral patterns}

---

## Evolution Log
{table of changes}

---

<!-- METADATA {json with v3.0 version} -->
```

### Where New Sections Are Written

**Option A: New queen-governance subcommand in queen.sh**

Add a function `_queen_write_governance()` that:
1. Reads the current QUEEN.md
2. Checks if Intent/Vision/Governance/Goals/Architecture sections exist
3. If not: inserts them after the header, before User Preferences
4. If yes: updates the content (preserving existing wisdom sections below)
5. Updates METADATA to version 3.0.0

This follows the exact pattern of `_queen_write_learnings()` and `_queen_promote_instinct()` -- both find a section, manipulate it with awk/sed, and do atomic writes.

**Option B: New utils module for governance**

Create `.aether/utils/governance.sh` with `_governance_write()`.

**Recommendation: Option A** -- queen.sh already owns all QUEEN.md manipulation. Adding governance functions there keeps the "one file owns one document" principle. The `_queen_write_governance()` function is naturally a sibling of `_queen_promote()`, `_queen_write_learnings()`, and `_queen_promote_instinct()`.

### How Re-init Works (Update Without Reset)

The PROJECT.md states: "Re-running init should update Queen file, not destroy colony state."

This means:

1. **COLONY_STATE.json**: Do NOT overwrite if it already exists with an active goal. The existing Step 2 already handles this (warns, asks confirmation).

2. **QUEEN.md governance sections**: Update the Intent, Vision, Goals sections with the new goal. Do NOT touch User Preferences, Codebase Patterns, Build Learnings, or Instincts -- those accumulate through colony work.

3. **Runtime files** (pheromones, midden, learning-observations): Do NOT overwrite if they already exist. Step 4.5 already handles this (only creates if missing).

4. **Session**: Update session-init with the new goal but preserve phase state.

### Template Changes

The QUEEN.md template (`.aether/templates/QUEEN.md.template`) needs to be updated to v3 format with the new sections. However, existing colonies with v2 QUEEN.md files must be handled:

- `_queen_init()` creates from template only if QUEEN.md doesn't exist
- `_queen_write_governance()` adds new sections to existing v2 files
- A migration path (like the existing v1->v2 migration in `_queen_migrate()`) could add empty governance sections to v2 files

### Impact on colony-prime (Worker Context Assembly)

The `_colony_prime()` function in pheromone.sh (lines 735-1284) extracts wisdom from QUEEN.md using `_extract_wisdom()`. Currently it looks for 4 sections: User Preferences, Codebase Patterns, Build Learnings, Instincts.

**New sections must be injected into worker context.** The Intent and Goals sections are particularly valuable -- workers should know what the colony is trying to achieve. Architecture Notes from the survey provide critical context for builders.

The `_extract_wisdom()` function needs a v3 path that also extracts:
- `intent` -- the colony's purpose
- `goals` -- measurable outcomes
- `architecture_notes` -- tech stack and patterns from survey

These get added to the `prompt_section` that colony-prime assembles. They should be injected with HIGH retention priority (trimmed late in the budget), since they are fundamental context for every worker.

**Token budget impact:** The current budget is 8,000 chars (4,000 compact). Adding intent + goals + architecture adds roughly 500-1500 chars. This is manageable but requires the trim order to be updated.

### Updated Trim Order

```
1. Rolling summary (trimmed first -- lowest retention priority)
2. Phase learnings
3. Key decisions
4. Hive wisdom
5. Context capsule
6. Build learnings
7. User preferences
8. Codebase patterns
9. Pheromone signals
10. Instincts
11. Intent + Goals (trimmed last -- highest retention priority)
12. Blockers (NEVER trimmed)
```

---

## Part 6: Component Boundaries

### New Components

| Component | File | Responsibility | Dependencies |
|-----------|------|----------------|-------------|
| scan.sh | `.aether/utils/scan.sh` | Repo scanning functions | aether-utils.sh infrastructure (json_ok, jq) |
| scan-repo subcommand | dispatch in aether-utils.sh | Full repo scan entry point | scan.sh sourced |
| scan-quick subcommand | dispatch in aether-utils.sh | Lightweight scan entry point | scan.sh sourced |
| queen-governance subcommand | dispatch in aether-utils.sh | Write governance sections | queen.sh sourced |
| QUEEN.md v3 template | `.aether/templates/QUEEN.md.template` | Template with governance sections | None |
| init.md (refactored) | `.claude/commands/ant/init.md` | Smart init command | scan.sh, queen-governance |
| OpenCode init.md | `.opencode/commands/ant/init.md` | OpenCode parallel | Same as above |

### Modified Components

| Component | File | Change | Risk |
|-----------|------|--------|------|
| queen.sh | `.aether/utils/queen.sh` | Add `_queen_write_governance()` function | Low -- additive, no changes to existing functions |
| aether-utils.sh | `.aether/aether-utils.sh` | Add source for scan.sh, add dispatch cases | Low -- follows established pattern |
| pheromone.sh | `.aether/utils/pheromone.sh` | Update `_extract_wisdom()` for v3 QUEEN.md format, update trim order in `_colony_prime()` | Medium -- must not break v2 format reading |
| colony-state template | `.aether/templates/colony-state.template.json` | Possibly add governance fields to state | Low -- additive JSON fields |
| CLAUDE.md | `CLAUDE.md` | Document new QUEEN.md sections, update QUEEN.md structure description | Low -- documentation only |

### NOT Modified

| Component | Why |
|-----------|-----|
| session.sh | No changes to session management |
| state-api.sh | No changes to state read/write |
| hive.sh | No changes to hive brain |
| learning.sh | No changes to learning pipeline |
| pheromone write/display | No changes to signal management |
| build/continue playbooks | No changes to build flow |
| Agent definitions | No changes to worker agents |
| OpenCode agent definitions | No changes |

---

## Part 7: Data Flow

### Smart Init Data Flow

```
/ant:init "Build a REST API"
    |
    v
[1] _scan_repo (Bash call)
    |-- Reads: package.json, tsconfig.json, etc.
    |-- Checks: .aether/data/survey/ freshness
    |-- Checks: .aether/chambers/ for prior colonies
    |-- Calls: _domain_detect (existing in queen.sh)
    |-- Returns: scan_summary JSON
    |
    v
[2] hive-read (Bash call)
    |-- Reads: ~/.aether/hive/wisdom.json
    |-- Filters: by domain tags from scan
    |-- Returns: relevant cross-colony wisdom
    |
    v
[3] LLM Synthesis (in init.md)
    |-- Input: user goal + scan_summary + hive wisdom
    |-- Output: colony_prompt (structured Markdown)
    |
    v
[4] User Approval (interactive)
    |-- Display: colony_prompt
    |-- Wait: for user response
    |-- Branch: approve / edit / cancel
    |
    v
[5] _queen_write_governance (Bash call)
    |-- Input: approved intent, vision, governance, goals, architecture
    |-- Reads: existing .aether/QUEEN.md
    |-- Inserts: new governance sections (or updates existing)
    |-- Updates: METADATA version to 3.0.0
    |
    v
[6] Write COLONY_STATE.json (existing flow)
    |-- Template substitution
    |-- Prior knowledge inheritance
    |
    v
[7] Initialize runtime files (existing flow)
    |-- pheromones.json (if missing)
    |-- midden.json (if missing)
    |-- learning-observations.json (if missing)
    |
    v
[8] session-init (existing flow)
    |-- Create session.json
    |-- Baseline commit capture
```

### How Governance Flows to Workers

```
QUEEN.md (v3) with governance sections
    |
    v
_colony_prime() (pheromone.sh)
    |-- _extract_wisdom() extracts intent, goals, architecture
    |-- Builds prompt_section with governance context
    |-- Injects into worker prompts via build-context.md Step 4
    |
    v
Worker (Builder/Watcher/Scout) receives:
    "--- COLONY INTENT ---
    Build a REST API with authentication for a SaaS product
    --- END COLONY INTENT ---

    --- COLONY GOALS ---
    1. Working API with CRUD endpoints
    2. JWT authentication middleware
    3. Test coverage > 80%
    --- END COLONY GOALS ---"

    --- COLONY ARCHITECTURE ---
    Stack: TypeScript + Express + PostgreSQL
    Pattern: Repository pattern for data access
    --- END COLONY ARCHITECTURE ---"
```

---

## Part 8: Anti-Patterns

### Anti-Pattern 1: Making scan.sh do too much

The scan should be FAST (under 2 seconds). It reads a few files and checks directory existence. It should NOT:
- Parse every source file (that's what /ant:colonize does)
- Run git log analysis (that's what /ant:archaeology does)
- Check test coverage (that's what /ant:chaos does)

Keep scan.sh to: file presence checks, manifest parsing, directory counting.

### Anti-Pattern 2: Writing governance sections with sed/awk directly in init.md

The init.md file is a Markdown prompt. It should call shell subcommands for file manipulation, not contain inline sed/awk. The pattern is:
```
Run using Bash tool:
bash .aether/aether-utils.sh queen-governance --intent "..." --vision "..." ...
```
NOT:
```
Use sed to insert sections into QUEEN.md...
```

### Anti-Pattern 3: Breaking v2 QUEEN.md backward compatibility

The `_extract_wisdom()` function in pheromone.sh must continue to read v2 format QUEEN.md files. New v3 files get additional sections extracted. The format detection (line 783: `grep -q '^## Build Learnings$'`) should be extended, not replaced. A v3 file can be detected by checking for `^## Intent$`.

### Anti-Pattern 4: Making the approval loop complex

The approval loop should be simple: show, wait, proceed or stop. Do NOT add:
- Multi-round negotiation
- Version history of the prompt
- Side-by-side comparison with prior colonies

The user can always re-run `/ant:init` if they want to change the charter.

### Anti-Pattern 5: Putting governance data in COLONY_STATE.json

Governance (intent, vision, goals, architecture) lives in QUEEN.md, not COLONY_STATE.json. COLONY_STATE.json tracks operational state (current phase, plan, events, instincts). QUEEN.md is the charter -- the "why" and "what". COLONY_STATE.json is the "where are we". Mixing them breaks the separation of concerns.

---

## Part 9: Build Order

### Phase 1: Scan Module (no dependencies)

1. Create `.aether/utils/scan.sh` with `_scan_repo()`, `_scan_quick()`, `_scan_survey_status()`, `_scan_chambers()`, `_scan_domain()`
2. Add source line to aether-utils.sh
3. Add dispatch cases to aether-utils.sh
4. Write tests for scan functions
5. Update QUEEN.md template to v3 format (add empty governance sections)

**Risk:** None -- purely additive, no existing code modified.

### Phase 2: Queen Governance (depends on Phase 1 for template)

1. Add `_queen_write_governance()` to queen.sh
2. Add `queen-governance` dispatch case to aether-utils.sh
3. Update `_queen_init()` to use v3 template
4. Write tests for governance write function

**Risk:** Medium -- modifying queen.sh requires careful section manipulation. Follow the pattern of `_queen_write_learnings()` exactly.

### Phase 3: Colony-Prime v3 Support (depends on Phase 2)

1. Update `_extract_wisdom()` in pheromone.sh to handle v3 format (add intent, goals, architecture extraction)
2. Update `_colony_prime()` trim order to include governance sections
3. Add governance sections to the prompt_section assembly
4. Write tests for v3 extraction

**Risk:** Medium -- must not break v2 format. Test both v2 and v3 files.

### Phase 4: Init.md Refactor (depends on Phases 1-3)

1. Refactor init.md to add scan step (before state write)
2. Add prompt generation step (LLM synthesis)
3. Add approval loop step
4. Add governance write step (after approval)
5. Update OpenCode init.md for parity
6. End-to-end test: run /ant:init, verify governance sections in QUEEN.md

**Risk:** Medium -- init.md is 388 lines and is the most-used command. Changes must be tested thoroughly.

### Phase 5: Documentation and Validation

1. Update CLAUDE.md with new QUEEN.md v3 structure
2. Update workers.md if needed
3. Full integration test: init -> plan -> build -> verify governance flows to workers
4. Run validate-package.sh

### Dependency Graph

```
Phase 1 (scan.sh)    Phase 2 (queen-governance)
    |                       |
    v                       v
    |-----> Phase 3 (colony-prime v3) <-----|
                  |
                  v
           Phase 4 (init.md refactor)
                  |
                  v
           Phase 5 (docs + validation)
```

Phases 1 and 2 can be built in parallel. Phase 3 depends on both (needs scan data to enrich governance, needs governance functions to exist). Phase 4 depends on Phase 3. Phase 5 is final.

---

## Part 10: Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| scan.sh slow on large repos | Medium | Low | Scan only reads file existence and manifest files, not source. Cap at 50 files checked. |
| QUEEN.md v3 breaks v2 consumers | Medium | High | `_extract_wisdom()` has format detection. Test v2 and v3 paths. Add v3 detection alongside existing v2 detection. |
| Token budget exceeded with governance | Low | Medium | Governance adds ~1000 chars. Trim order puts it at high retention priority. Compact mode already trims aggressively. |
| Approval loop confuses users | Medium | Low | Keep it simple: show, approve/edit/cancel. One interaction, not a negotiation. |
| init.md refactor breaks existing behavior | Medium | High | Test all existing init scenarios (fresh, re-init, prior knowledge) before and after refactor. |
| Governance sections empty on update | Low | Low | `_queen_write_governance()` only writes if content is non-empty. Existing v2 QUEEN.md files without governance sections get them added with placeholder text. |
| OpenCode parity broken | Low | Low | OpenCode init.md needs same changes. Review manually after Claude Code changes are stable. |

---

## Sources

- HIGH confidence: Direct analysis of `.claude/commands/ant/init.md` (388 lines) -- full step flow traced
- HIGH confidence: Direct analysis of `.claude/commands/ant/colonize.md` (257 lines) -- existing scanning patterns traced
- HIGH confidence: Direct analysis of `.claude/commands/ant/plan.md` (667 lines) -- planning context loading traced
- HIGH confidence: Direct analysis of `.aether/utils/queen.sh` (1242 lines) -- all queen functions traced, section manipulation patterns documented
- HIGH confidence: Direct analysis of `.aether/utils/pheromone.sh` `_colony_prime()` function (lines 735-1284) -- full context assembly traced, trim order documented
- HIGH confidence: Direct analysis of `.aether/utils/pheromone.sh` `_extract_wisdom()` function (lines 779-887) -- v2 format extraction documented
- HIGH confidence: Direct analysis of `.aether/utils/session.sh` (547 lines) -- session management patterns documented
- HIGH confidence: Direct analysis of `.aether/utils/state-api.sh` (200 lines) -- state read/write/mutate documented
- HIGH confidence: Direct analysis of `.aether/templates/QUEEN.md.template` (62 lines) -- v2 template structure documented
- HIGH confidence: Direct analysis of `.aether/templates/colony-state.template.json` (36 lines) -- state template structure documented
- HIGH confidence: Direct analysis of `.aether/aether-utils.sh` (first 100 lines) -- source loading and dispatch pattern documented
- HIGH confidence: Direct analysis of `.planning/PROJECT.md` -- v2.5 milestone scope and user feedback documented
- HIGH confidence: Pattern analysis of 9 existing utils modules -- modularization pattern confirmed

---

*Architecture research for: Aether v2.5 Smart Init*
*Researched: 2026-03-27*
