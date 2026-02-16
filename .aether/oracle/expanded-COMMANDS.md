# Aether Command System - Comprehensive Documentation

## Table of Contents

1. [Executive Summary](#executive-summary)
2. [Command Taxonomy and Classification](#command-taxonomy-and-classification)
3. [Slash Command Architecture](#slash-command-architecture)
4. [Command Reference - Lifecycle Commands](#command-reference---lifecycle-commands)
5. [Command Reference - Pheromone Commands](#command-reference---pheromone-commands)
6. [Command Reference - Status & Information Commands](#command-reference---status--information-commands)
7. [Command Reference - Session Management Commands](#command-reference---session-management-commands)
8. [Command Reference - Advanced/Utility Commands](#command-reference---advancedutility-commands)
9. [Command Dependencies Graph](#command-dependencies-graph)
10. [Execution Flow Diagrams](#execution-flow-diagrams)
11. [Platform Differences Analysis](#platform-differences-analysis)
12. [Consolidation Opportunities](#consolidation-opportunities)
13. [Known Issues and Limitations](#known-issues-and-limitations)
14. [Appendix](#appendix)

---

## Executive Summary

The Aether command system represents a sophisticated dual-platform architecture that enables collaborative software development through a multi-agent colony metaphor. This system supports both **Claude Code** and **OpenCode** AI assistants, providing a unified interface for managing complex software projects through 71 total command implementations distributed across 36 unique command names.

### System Scale and Scope

| Metric | Count | Description |
|--------|-------|-------------|
| Total Command Implementations | 71 | 36 Claude Code + 35 OpenCode |
| Unique Command Names | 36 | Core commands shared across platforms |
| Claude Code Commands | 36 | Full feature set with all capabilities |
| OpenCode Commands | 35 | Slightly reduced feature set |
| Shared Commands | 35 | Identical or near-identical implementations |
| Claude-Only Commands | 1 | `resume` command (OpenCode uses `resume-colony`) |
| Total Lines of Command Code | ~13,500 | Across all implementations |
| Average Command Size | 375 lines | Varies from 51 to 1,051 lines |

### Core Philosophy and Design Principles

The Aether command system treats software development as an ant colony intelligence problem, where complex behaviors emerge from the interaction of simple agents following established patterns. This biological metaphor provides an intuitive framework for understanding the system's architecture:

**The Queen** serves as the orchestrator, receiving user commands and spawning appropriate workers to accomplish tasks. The Queen does not perform work directly but instead coordinates the colony's collective intelligence, synthesizing results from multiple workers and making decisions based on aggregated information.

**Workers** are specialized agents assigned to castes based on their capabilities. Each caste has distinct responsibilities:
- **Builders** (🔨🐜) implement features and write code
- **Watchers** (👁️🐜) monitor execution and verify quality
- **Scouts** (🔍🐜) explore codebases and research solutions
- **Chaos** (🎲🐜) test edge cases and resilience
- **Archaeologists** (🏺🐜) investigate git history and code provenance
- **Oracles** (🔮🐜) perform deep research using iterative loops

**Pheromones** provide a signaling mechanism for user guidance. Three signal types enable different levels of direction:
- **FOCUS** (normal priority): Suggests areas for attention without enforcing constraints
- **REDIRECT** (high priority): Establishes hard constraints that must be respected
- **FEEDBACK** (low priority): Provides gentle adjustments based on observations

**Memory** persists across phases through multiple mechanisms:
- `COLONY_STATE.json` maintains the colony's current state, plan, and progress
- `QUEEN.md` stores eternal wisdom and validated learnings
- `constraints.json` preserves user guidance signals
- `memory.instincts` captures high-confidence patterns discovered during builds
- `memory.phase_learnings` records insights from completed phases

**Emergence** describes how complex, intelligent behavior arises from the interaction of simple rules and parallel execution. No single agent has complete knowledge, yet the colony collectively achieves sophisticated outcomes through coordination and synthesis.

### Distribution Architecture

The command system implements a hub-and-spoke distribution model:

```
Aether Repository (Source of Truth)
├── .claude/commands/ant/          # Claude Code command definitions
│   ├── init.md, build.md, plan.md, ...
│   └── 36 total command files
├── .opencode/commands/ant/        # OpenCode command definitions
│   ├── init.md, build.md, plan.md, ...
│   └── 35 total command files
└── Distribution via npm
    └── npm install -g .
        └── ~/.aether/             # The Hub
            ├── commands/claude/   # Synced to .claude/commands/ant/
            ├── commands/opencode/ # Synced to .opencode/commands/ant/
            ├── system/            # Core utilities and documentation
            └── agents/            # Agent definitions
```

When users run `aether update` in any repository, the hub distributes the latest command definitions, ensuring all colonies operate with consistent capabilities.

---

## Command Taxonomy and Classification

The 36 unique commands organize into five functional categories, each serving distinct purposes within the colony lifecycle.

### 1. Lifecycle Commands (Core Workflow)

These seven commands form the primary development workflow, from colony initialization through completion and archival.

| Command | Purpose | Complexity | Lines (Claude) | Lines (OpenCode) | Key Feature |
|---------|---------|------------|----------------|------------------|-------------|
| `init` | Initialize colony with goal | High | 316 | 272 | Knowledge inheritance |
| `plan` | Generate project phases | High | 534 | ~534 | Iterative confidence-building |
| `build` | Execute phase with parallel workers | Very High | 1,051 | 989 | Wave-based execution |
| `continue` | Verify work and advance phase | Very High | 1,037 | ~1,037 | Multi-gate verification |
| `seal` | Archive completed colony | High | 337 | ~337 | Crowned Anthill milestone |
| `entomb` | Archive colony to chambers | High | 407 | ~407 | Chamber-based storage |
| `lay-eggs` | Start new colony from existing | Medium | 154 | ~154 | Evolutionary lineage |

**Workflow Integration:**
```
init → plan → build → continue → [repeat build/continue] → seal/entomb
  ↓      ↓      ↓         ↓
  └──────┴──────┴─────────┘
       lay-eggs (spawns new colony)
```

### 2. Pheromone Commands (User Guidance)

These three commands inject signals that guide colony behavior without direct intervention.

| Command | Purpose | Priority | Complexity | Lines | State Modified |
|---------|---------|----------|------------|-------|----------------|
| `focus` | Guide colony attention | Normal | Low | 51 | constraints.json focus[] |
| `redirect` | Hard constraint (avoid pattern) | High | Low | 51 | constraints.json constraints[] |
| `feedback` | Gentle adjustment | Low | Low | 51 | constraints.json + instincts[] |

**Usage Patterns:**
- **Before builds:** Use FOCUS and REDIRECT to steer worker behavior
- **After builds:** Use FEEDBACK to adjust based on observations
- **Hard constraints:** REDIRECT signals are treated as inviolable rules
- **Gentle nudges:** FEEDBACK creates instincts with 0.7 confidence

### 3. Status & Information Commands

These six commands provide visibility into colony state, history, and configuration.

| Command | Purpose | Complexity | Lines | Information Source |
|---------|---------|------------|-------|-------------------|
| `status` | Colony dashboard | Medium | ~200 | COLONY_STATE.json + constraints.json + dreams/ |
| `phase` | View phase details | Low | ~120 | COLONY_STATE.json plan.phases[] |
| `flags` | List active flags/blockers | Medium | ~150 | flag-list utility |
| `flag` | Create a flag | Low | ~132 | flag-create utility |
| `history` | Browse event history | Medium | ~128 | COLONY_STATE.json events[] |
| `help` | Command reference | Low | 113 | Static documentation |

### 4. Session Management Commands

These five commands manage session state, persistence, and recovery.

| Command | Purpose | Complexity | Lines | Key Capability |
|---------|---------|------------|-------|----------------|
| `watch` | Live tmux visibility | Medium | ~240 | Real-time swarm display |
| `pause-colony` | Save state and handoff | High | ~240 | HANDOFF.md creation |
| `resume-colony` | Restore from pause | Medium | ~120 | State restoration |
| `resume` | Claude-specific resume | Medium | ~160 | Session recovery |
| `update` | Update system from hub | Medium | ~145 | Hub synchronization |

### 5. Advanced/Utility Commands

These fourteen commands provide specialized capabilities for investigation, analysis, and maintenance.

| Command | Purpose | Complexity | Lines | Caste Used |
|---------|---------|------------|-------|------------|
| `swarm` | Parallel bug investigation | High | 380 | 4 Scouts |
| `chaos` | Resilience testing | High | 341 | Chaos |
| `archaeology` | Git history analysis | High | ~330 | Archaeologist |
| `oracle` | Deep research (RALF loop) | High | 380 | Oracle |
| `colonize` | Territory survey | High | ~240 | Surveyor |
| `organize` | Codebase hygiene report | Medium | ~220 | Scout |
| `council` | Intent clarification | Medium | ~300 | Prime |
| `dream` | Philosophical observation | High | ~260 | Sage |
| `interpret` | Dream validation | High | ~260 | Sage |
| `tunnels` | Browse archived colonies | Medium | ~250 | Chronicler |
| `verify-castes` | System status check | Low | 86 | All castes |
| `migrate-state` | State migration utility | Medium | ~155 | System |
| `maturity` | Colony maturity assessment | Low | ~95 | Prime |
| `lay-eggs` | Colony reproduction | Medium | ~154 | Queen |

---

## Slash Command Architecture

### Overview and Design Philosophy

The Aether slash command architecture implements a declarative, file-based command system where each command is defined in a Markdown file with embedded instructions. This design prioritizes readability, version control, and platform portability.

### File Structure and Organization

All command files follow a standardized structure that enables consistent parsing and execution:

```markdown
---
name: ant:<command>
description: "<emoji> <description>"
---

You are the **<Role>**. <Brief description of purpose>.

## Instructions

<Detailed step-by-step instructions>

### Step 1: <Action>
...

### Step 2: <Action>
...
```

### Frontmatter Schema

| Field | Type | Required | Description | Example |
|-------|------|----------|-------------|---------|
| `name` | string | Yes | Command identifier | `ant:build` |
| `description` | string | Yes | Short description with emoji | `"🔨🐜 Execute phase"` |

### Command Distribution Flow

The distribution architecture ensures commands propagate from source to execution environments:

```
Aether Repository (Development)
├── .claude/commands/ant/          # Source: Claude Code commands
│   ├── init.md
│   ├── build.md (1,051 lines)
│   ├── plan.md (534 lines)
│   └── ... (33 more)
│
├── .opencode/commands/ant/        # Source: OpenCode commands
│   ├── init.md (272 lines)
│   ├── build.md (989 lines)
│   └── ... (33 more)
│
└── npm install -g .               # Distribution trigger
    │
    ▼
~/.aether/                         # The Hub
├── commands/
│   ├── claude/                    # 36 command files
│   └── opencode/                  # 35 command files
├── system/                        # Core utilities
│   ├── aether-utils.sh
│   ├── workers.md
│   └── docs/
└── agents/                        # Agent definitions
    ├── aether-builder.md
    ├── aether-watcher.md
    └── ...

    │
    │ aether update (in target repo)
    ▼
any-repo/
├── .claude/commands/ant/          # Synced from hub
└── .aether/                       # Working copy
    ├── aether-utils.sh            # From hub
    └── data/                      # LOCAL - never touched
```

### Command Routing Mechanism

Commands are invoked through platform-specific slash syntax that maps to file execution:

**Invocation Syntax:**
- Claude Code: `/ant:<command> [arguments]`
- OpenCode: `/ant:<command> [arguments]`

**Routing Process:**
1. Platform detects slash command prefix (`/ant:`)
2. Command name extracted and validated against known commands
3. Corresponding file loaded from `commands/ant/<command>.md`
4. Frontmatter parsed for metadata
5. Instructions section executed with `$ARGUMENTS` populated
6. Output returned to user

### Visual Mode Pattern

Most commands support a `--no-visual` flag that controls animated display:

```markdown
Parse `$ARGUMENTS`:
- If contains `--no-visual`: set `visual_mode = false` (visual is ON by default)
- Otherwise: set `visual_mode = true`
```

**Visual Mode Implementation:**

When enabled, commands follow a consistent visual pattern:

1. **Session ID Generation**: `<command>-$(date +%s)` creates unique identifier
2. **Swarm Display Initialization**: `swarm-display-init` sets up real-time tracking
3. **Visual Headers**: ASCII art banners with emoji decorations
4. **Progress Indicators**: Animated status updates during execution
5. **Final Rendering**: `swarm-display-render` shows completion state

Example visual header:
```
🔨🐜⚡🐜🔨 ═══════════════════════════════════════════════════
        P H A S E   E X E C U T I O N
═══════════════════════════════════════════════════ 🔨🐜⚡🐜🔨
```

### Session Freshness Detection

Stateful commands implement timestamp verification to prevent stale session files from corrupting workflows:

```bash
# Capture session start time
COMMAND_START=$(date +%s)

# Verify freshness before operations
stale_check=$(bash .aether/aether-utils.sh session-verify-fresh \
  --command <name> "" "$COMMAND_START")

# Parse result
is_stale=$(echo "$stale_check" | jq -r '.stale | length')
freshness_status=$([[ "$is_stale" -gt 0 ]] && echo "stale" || echo "fresh")
```

**Protected Commands** (never auto-clear):
- `init` - COLONY_STATE.json is precious user data
- `seal` - Archives are precious
- `entomb` - Chambers are precious

**Auto-clear Commands** (stale files removed automatically):
- `swarm`, `oracle`, `watch`, `colonize` - Findings are temporary

### State Validation Pattern

Commands validate COLONY_STATE.json before proceeding with operations:

```markdown
Read `.aether/data/COLONY_STATE.json`.
If `goal: null` -> "No colony initialized. Run /ant:init first."
```

**Auto-upgrade for Old State Versions:**

When `version` field is missing, "1.0", or "2.0":
1. Preserve: `goal`, `state`, `current_phase`, `plan.phases`
2. Write upgraded v3.0 state with new structure
3. Output: `State auto-upgraded to v3.0`
4. Continue with command

### Worker Spawn Pattern

Build commands spawn parallel workers using the Task tool with specific subagent types:

```markdown
**CRITICAL: Spawn ALL Wave 1 workers in a SINGLE message using multiple Task tool calls.**
```

**Worker JSON Response Format:**
```json
{
  "ant_name": "builder-1",
  "status": "completed|failed",
  "summary": "Implemented authentication module",
  "files_created": ["src/auth.js", "src/auth.test.js"],
  "files_modified": ["src/app.js"],
  "blockers": []
}
```

**Spawn Constraints:**
- Max spawn depth: 3 (prevent runaway recursion)
- Max spawns at depth 1: 4 (parallelism cap)
- Max spawns at depth 2: 2 (secondary cap)
- Global workers per phase: 10 (hard ceiling)

---

## Command Reference - Lifecycle Commands

### `/ant:init` - Initialize Colony

**Purpose (300+ words):**

The `init` command serves as the foundational entry point for the entire Aether colony system, establishing the colony's identity, initializing all persistent state structures, and preparing the environment for sophisticated multi-agent software development. When a user invokes `/ant:init`, they are not merely creating a configuration file—they are founding a new colony with its own memory, constraints, evolutionary path, and potential for growth.

The initialization process performs multiple critical functions that establish the foundation for all subsequent colony operations. First, it validates the user's goal, ensuring it is neither empty nor unreasonably verbose, striking a balance between conciseness and clarity. Second, it bootstraps the system files if they don't exist, intelligently copying from the global hub at `~/.aether/system/` or failing gracefully with clear installation instructions if the hub is not present. Third, it initializes the QUEEN.md wisdom document, which serves as the colony's persistent memory for eternal guidance across sessions, outliving any individual context window.

A crucial aspect of initialization is the session freshness check. The command captures the precise start time and verifies whether existing state files are fresh or stale. If a colony already exists with a valid goal, the command warns the user and requires explicit acknowledgment before proceeding with reinitialization. This protective mechanism prevents precious colony data from accidental overwrite, ensuring that accumulated wisdom and progress are not inadvertently destroyed.

The command also implements a sophisticated knowledge inheritance system. If a `completion-report.md` exists from a prior colony session, the init command extracts high-confidence instincts (those with confidence >= 0.5) and validated learnings, seeding the new colony's memory with wisdom from its predecessors. This creates an evolutionary lineage where each colony can benefit from the accumulated knowledge of previous colonies, enabling continuous improvement across sessions.

Additional initialization steps include: creating the COLONY_STATE.json with the v3.0 structure that supports modern memory features, initializing constraints.json for pheromone storage and user guidance, creating the CONTEXT.md document for session recovery after context clears, validating the state file structure to ensure integrity, detecting nestmates (related colonies in the same ecosystem), and registering the repository in the global hub for update distribution. Finally, the command initializes session tracking to enable `/ant:resume` functionality after context clear, ensuring continuity across disjointed conversation sessions.

**Usage Syntax:**
```
/ant:init "<your goal here>"

Options:
  --no-visual    Disable visual display (visual is ON by default)
```

**Examples:**
```
/ant:init "Build a REST API with authentication"
/ant:init "Create a soothing sound application"
/ant:init "Design a calculator CLI tool"
/ant:init "Build a React component library" --no-visual
/ant:init "Implement a distributed task queue with Redis"
```

**Arguments and Options:**

| Argument | Type | Required | Description | Validation |
|----------|------|----------|-------------|------------|
| `goal` | string | Yes | The colony's objective | 1-500 characters |
| `--no-visual` | flag | No | Disable animated visual display | None |

**Implementation Approach (Step-by-Step):**

1. **Step 0**: Initialize visual mode if enabled
   - Generate session ID: `init-$(date +%s)`
   - Initialize swarm display with `swarm-display-init`
   - Update display with "Queen" status and "excavating" activity

2. **Step 0.5**: Version check (non-blocking)
   - Run `version-check` utility
   - Display notice if update available
   - Proceed regardless of outcome

3. **Step 1**: Validate input
   - If `$ARGUMENTS` empty: show usage, stop
   - Check goal length constraints

4. **Step 1.5**: Bootstrap system files (conditional)
   - Check if `.aether/aether-utils.sh` exists
   - If not, check for hub at `~/.aether/system/`
   - Bootstrap from hub if available
   - Show install instructions if hub missing

5. **Step 1.6**: Initialize QUEEN.md wisdom document
   - Run `queen-init` utility
   - Parse JSON result for created/existing status
   - Display appropriate message

6. **Step 2**: Read current state with freshness check
   - Capture `INIT_START=$(date +%s)`
   - Read `COLONY_STATE.json`
   - Run `session-verify-fresh` check
   - Warn if existing colony found

7. **Step 2.6**: Load prior colony knowledge
   - Check for `completion-report.md`
   - Extract instincts with confidence >= 0.5
   - Extract validated learnings
   - Display inheritance summary

8. **Step 3**: Write colony state
   - Generate session ID: `session_{unix_timestamp}_{random}`
   - Generate ISO-8601 UTC timestamp
   - Write v3.0 structure to `COLONY_STATE.json`
   - Include inherited instincts/learnings if present

9. **Step 4**: Initialize constraints
   - Write `constraints.json` with v1.0 structure
   - Initialize empty focus and constraints arrays

10. **Step 5**: Initialize context document
    - Run `context-update init "$ARGUMENTS"`
    - Creates `CONTEXT.md` for session recovery

11. **Step 6**: Validate state file
    - Run `validate-state colony`
    - Output warning if validation fails

12. **Step 6.5**: Detect nestmates
    - Run nestmate-loader to find related colonies
    - Display count and truncated goals

13. **Step 6.6**: Register repo in global hub (silent)
    - Run `registry-add` with current path
    - Copy version.json from hub
    - Fail silently if unavailable

14. **Step 7**: Display result
    - Render final swarm display if visual mode
    - Output colony status header
    - Show goal, session ID, inherited knowledge
    - Display next command suggestions

15. **Step 8**: Initialize session
    - Run `session-init` with session ID and goal
    - Enable `/ant:resume` functionality

**Dependencies on Utilities:**

| Utility | Purpose | Location |
|---------|---------|----------|
| `version-check` | Check for available updates | aether-utils.sh |
| `queen-init` | Initialize QUEEN.md wisdom document | aether-utils.sh |
| `session-verify-fresh` | Verify state file freshness | aether-utils.sh:3181 |
| `validate-state colony` | Validate COLONY_STATE.json structure | aether-utils.sh |
| `session-init` | Initialize session tracking | aether-utils.sh |
| `context-update` | Create/update CONTEXT.md | aether-utils.sh |
| `registry-add` | Register repo in global hub | aether-utils.sh |

**Error Handling:**

| Error Condition | Response | Recovery |
|-----------------|----------|----------|
| Empty arguments | Show usage with examples | User provides goal |
| No system files, hub exists | Bootstrap from hub | Automatic |
| No system files, no hub | Show install instructions | Run `aether install` |
| Existing colony, fresh | Strongly recommend continuation | User decides |
| Existing colony, stale | Warn but proceed | Automatic |
| State validation failure | Output warning, continue | Manual review |
| QUEEN.md init failure | Display error, continue | Manual creation |

**Side Effects on State:**

| File | Operation | Content |
|------|-----------|---------|
| `COLONY_STATE.json` | Create/Overwrite | v3.0 structure with goal, session, memory |
| `constraints.json` | Create/Overwrite | v1.0 structure with empty arrays |
| `CONTEXT.md` | Create | Session recovery document |
| `QUEEN.md` | Create (if not exists) | Wisdom document template |
| `session.json` | Create | Session tracking data |
| Global registry | Update | Repo path and version |

**Platform Differences:**

| Aspect | Claude Code | OpenCode | Impact |
|--------|-------------|----------|--------|
| QUEEN.md init | Full (Step 1.6) | Missing | OpenCode lacks wisdom inheritance |
| Context document | Full (Step 5) | Missing | OpenCode lacks session recovery |
| Session init | Full (Step 8) | Missing | OpenCode lacks `/ant:resume` |
| Lines of code | 316 | 272 | 44 lines less functionality |
| Caste emojis | 🔨🐜 combined | 🔨 only | Visual distinction |

**Known Issues:**

1. **ISSUE-004**: Template path hardcoded to `runtime/`
   - Location: `aether-utils.sh:2689`
   - Impact: `queen-init` fails when Aether installed via npm
   - Workaround: Use git clone instead of npm install

2. **OpenCode missing features**: Steps 1.6, 5, and 8 are absent from OpenCode version
   - Impact: Reduced functionality for OpenCode users
   - No session recovery, no wisdom inheritance

---

### `/ant:plan` - Generate Project Plan

**Purpose (300+ words):**

The `plan` command orchestrates the colony's research and planning phase, implementing an iterative confidence-building loop that continues until the colony achieves 80% confidence in its understanding of the project requirements, codebase structure, and implementation approach. This command embodies the colony's collective intelligence, spawning specialized scouts and route-setters to explore the codebase and create a structured, phased execution plan.

The planning process is designed to handle uncertainty gracefully through its iterative approach. Rather than requiring perfect knowledge upfront, the command uses successive refinement where each cycle builds upon the previous one. The first iteration performs broad exploration of the codebase architecture, existing patterns, and technology stack. Subsequent iterations focus on specific knowledge gaps identified in earlier cycles, drilling down into areas of uncertainty. This mimics how real ant colonies explore territory—broad sweeps followed by detailed investigation of promising or problematic areas.

A key feature of the plan command is its integration with territory surveys. If `/ant:colonize` has been run previously, the command loads relevant survey documents to inform the planning process. PATHOGENS.md is always read first to understand known concerns, then additional documents are loaded based on goal keywords: DISCIPLINES.md for UI work, BLUEPRINT.md for API work, PROVISIONS.md for database work, and so on. This ensures that known concerns are addressed in the plan and that the colony doesn't waste time rediscovering already-documented patterns.

The command implements sophisticated auto-termination safeguards to prevent infinite planning loops. The loop exits when confidence reaches 80%, after 4 iterations maximum, or when progress stalls (less than 5% improvement for 2 consecutive iterations). This ensures the colony moves to execution in a reasonable timeframe while still achieving sufficient understanding for autonomous work.

The output is a structured plan with 3-6 phases, each containing concrete tasks with goal-oriented descriptions, constraints that define boundaries, hints that point toward patterns, and success criteria that enable verification. The plan is stored in COLONY_STATE.json and displayed to the user with next-step recommendations, including the calculated first incomplete phase.

**Usage Syntax:**
```
/ant:plan [options]

Options:
  --accept        Accept current plan regardless of confidence
  --no-visual     Disable visual display
```

**Examples:**
```
/ant:plan
/ant:plan --accept
/ant:plan --no-visual
```

**Arguments and Options:**

| Argument | Type | Required | Description |
|----------|------|----------|-------------|
| `--accept` | flag | No | Force accept current plan regardless of confidence |
| `--no-visual` | flag | No | Disable animated visual display |

**Implementation Approach:**

1. **Step 0**: Initialize visual mode
   - Generate `plan_id`
   - Initialize swarm display

2. **Step 0.5**: Version check
   - Non-blocking update check

3. **Step 1**: Read state + version check
   - Read `COLONY_STATE.json`
   - Auto-upgrade old state versions
   - Validate goal is not null

4. **Step 1.5**: Load state and show resumption context
   - Run `load-state` utility
   - Display current phase and name
   - Check for `HANDOFF.md`
   - Run `unload-state`

5. **Step 2**: Check existing plan
   - If `plan.phases` has entries, skip to display
   - Parse `--accept` flag

6. **Step 3**: Initialize planning state
   - Write `watch-status.txt`
   - Write `watch-progress.txt`
   - Log `PLAN_START` activity

7. **Step 3.5**: Load territory survey
   - List `.aether/data/survey/*.md`
   - Read PATHOGENS.md first
   - Read additional docs based on goal keywords
   - Inject context into prompts

8. **Step 4**: Research and planning loop
   - Initialize tracking variables
   - While iteration < 4 AND confidence < 80:
     - Spawn scout (broad on iter 1, gap-focused on 2+)
     - Spawn route-setter with findings
     - Update confidence and gaps
     - Check for stall condition
   - Auto-finalize when conditions met

9. **Step 5**: Finalize plan
   - Write phases to `COLONY_STATE.json`
   - Set `plan.generated_at`
   - Set state to "READY"
   - Append event
   - Log `PLAN_COMPLETE`

10. **Step 6**: Update session
    - Run `session-update`

11. **Step 7**: Display plan
    - Render swarm display
    - Show phases with status icons
    - Calculate first incomplete phase
    - Display next steps

**Confidence Scoring Dimensions:**

| Dimension | Weight | Description | Measurement |
|-----------|--------|-------------|-------------|
| Knowledge | 25% | Understanding of codebase structure | Files explored, patterns identified |
| Requirements | 25% | Clarity of success criteria | Acceptance conditions defined |
| Risks | 20% | Identification of blockers | Failure modes anticipated |
| Dependencies | 15% | Understanding of ordering constraints | Task sequencing clarity |
| Effort | 15% | Ability to estimate complexity | Relative sizing accuracy |

**Overall** = weighted average of dimensions

**Target: 80%** - Sufficient for autonomous execution

**Platform Differences:**

Near-identical between platforms. Both use `aether-scout` and `aether-route-setter` agent types with the same spawn patterns.

---

### `/ant:build` - Execute Phase

**Purpose (300+ words):**

The `build` command represents the colony's primary execution engine, responsible for orchestrating parallel worker execution to complete phase tasks with high quality and comprehensive verification. It embodies the core of Aether's multi-agent architecture, where the Queen spawns specialized workers (builders, watchers, chaos agents) that collaborate to implement features while maintaining rigorous quality standards.

The build process begins with comprehensive preparation that ensures a solid foundation for execution. The command loads colony state, validates that the requested phase exists and is not already completed, checks for blocker flags that might prevent advancement, and creates a git checkpoint for rollback capability if issues arise. It then loads territory survey documents and QUEEN.md wisdom to inject crucial context into worker prompts, ensuring workers understand the broader architectural vision.

A unique and powerful feature of the build command is the Archaeologist Pre-Build Scan (Step 4.2). When a phase modifies existing files, the command spawns an Archaeologist scout to investigate the git history of those files. This provides builders with crucial context about why code exists, known workarounds that have been applied, architectural decisions that were made, and areas requiring caution. This historical awareness prevents the colony from inadvertently breaking established patterns, reintroducing previously-fixed issues, or violating intentional design constraints.

The execution phase uses a sophisticated wave-based approach that respects task dependencies while maximizing parallelism. Tasks are grouped by their dependency structure, with Wave 1 containing tasks that have no dependencies and can run in parallel, Wave 2 containing tasks that depend on Wave 1 completion, and so on. The Queen spawns all Wave 1 workers simultaneously in a single message, then waits for results before proceeding to subsequent waves. This approach ensures maximum parallelism while respecting ordering constraints.

Every build includes mandatory verification through independent agents. A Watcher ant independently verifies all work done by builders, ensuring quality through separation of concerns and fresh perspective. After the Watcher completes, a Chaos ant performs resilience testing, probing edge cases, boundary conditions, and unexpected inputs to identify potential issues before they reach production.

The command produces comprehensive output including task completion status, files created and modified, spawn metrics, verification results, and learning extraction. All results are synthesized into a structured JSON report that feeds into the colony's memory system, enabling continuous improvement across builds.

**Usage Syntax:**
```
/ant:build <phase_number> [options]

Options:
  --verbose, -v       Show full completion details
  --no-visual         Disable real-time visual display
  --model <name>, -m  Override model for this build
```

**Examples:**
```
/ant:build 1
/ant:build 1 --verbose
/ant:build 1 --no-visual
/ant:build 1 --model glm-5
/ant:build 2 -v
```

**Arguments and Options:**

| Argument | Type | Required | Description |
|----------|------|----------|-------------|
| `phase_number` | number | Yes | The phase to build (1-indexed) |
| `--verbose`, `-v` | flag | No | Show spawn tree, TDD details, patterns |
| `--no-visual` | flag | No | Disable visual display |
| `--model`, `-m` | string | No | Override model for this build |

**Implementation Approach:**

1. **Step 0**: Version check (non-blocking)
2. **Step 0.6**: Verify LiteLLM Proxy health
3. **Step 0.5**: Load colony state
4. **Step 1**: Validate arguments and state
   - Auto-upgrade old state versions
   - Check phase exists and not completed
5. **Step 1.5**: Blocker advisory check (non-blocking)
6. **Step 2**: Update state to "EXECUTING"
7. **Step 3**: Create git checkpoint via `autofix-checkpoint`
8. **Step 4**: Load constraints from `constraints.json`
9. **Step 4.0**: Load territory survey documents
10. **Step 4.1**: Load QUEEN.md wisdom
11. **Step 4.2**: Archaeologist pre-build scan (conditional)
    - Only if phase modifies existing files
    - Spawns archaeologist to investigate git history
12. **Step 5**: Initialize swarm display and analyze tasks
    - Group tasks by dependencies into waves
    - Initialize progress tracking
13. **Step 6**: Execute waves
    - Wave 1: Spawn all independent tasks in parallel
    - Wait for completion
    - Wave 2+: Spawn dependent tasks sequentially
14. **Step 7**: Spawn mandatory Watcher verification
15. **Step 8**: Spawn mandatory Chaos resilience testing
16. **Step 9**: Synthesize results
    - Aggregate all worker outputs
    - Calculate metrics
    - Extract learnings
17. **Step 10**: Update state and display results

**Dependencies on Utilities:**

| Utility | Purpose |
|---------|---------|
| `autofix-checkpoint` | Create git checkpoint for rollback |
| `swarm-display-init` | Initialize real-time display |
| `activity-log` | Log build activities |
| `validate-state` | Validate state file integrity |

**Error Handling:**

| Error Condition | Response |
|-----------------|----------|
| No colony initialized | "Run /ant:init first" |
| Phase already completed | "Phase N already completed" |
| Phase doesn't exist | "Phase N not found in plan" |
| Blocker flags exist | Advisory warning, continues |
| All workers fail | Rollback checkpoint, report failure |
| Watcher finds issues | Report verification failures |
| Chaos finds issues | Report resilience failures |

**Side Effects:**

- Updates `COLONY_STATE.json` with task statuses
- Creates/modifies source files
- Creates git checkpoint (stash or commit)
- Updates `activity.log`
- May create/modify test files
- Extracts learnings to memory

**Platform Differences:**

| Aspect | Claude Code | OpenCode |
|--------|-------------|----------|
| Agent types | `aether-builder`, `aether-watcher` | `general-purpose` with role injection |
| Caste emojis | 🔨🐜 combined | 🔨 only |
| Fallback comments | Present | Absent |
| Lines | 1,051 | 989 |

---

### `/ant:continue` - Verify and Advance

**Purpose (300+ words):**

The `continue` command implements the colony's quality verification and phase advancement system, serving as the critical gatekeeper that ensures work meets standards before progression. This command embodies Aether's commitment to quality through its multi-gate verification system, which validates that builds are complete, tests pass, code meets standards, and the colony is ready to move forward.

The continue process begins by loading colony state and determining the current phase context. It then enters a comprehensive verification loop that checks multiple dimensions of quality. Each gate must be passed before the colony can advance, creating a rigorous barrier against incomplete or defective work progressing through the pipeline.

The first gate is the **Build Verification Gate**, which confirms that the previous build actually completed successfully. This gate checks task completion status, verifies that all expected files were created or modified, and ensures no workers reported failures. If any task failed or verification found issues, the gate fails and the user must address problems before continuing.

The second gate is the **Type Check Gate**, which runs static type analysis if the project uses TypeScript or another typed language. This catches type mismatches, undefined references, and interface violations that could cause runtime errors.

The third gate is the **Lint Gate**, which checks code style and formatting against project standards. This ensures consistency across the codebase and catches potential issues like unused variables, unreachable code, or problematic patterns.

The fourth gate is the **Test Gate**, which runs the project's test suite. This is the most critical gate, as it validates that the code behaves correctly under expected conditions. The command looks for TDD evidence (test files created before implementation) and reports whether the testing approach followed test-first development.

The fifth gate is the **Security Gate**, which scans for potential security issues like hardcoded secrets, SQL injection vulnerabilities, or unsafe eval usage.

The sixth gate is the **Diff Gate**, which shows the user what changes will be committed and requires explicit confirmation before advancing.

Only after all gates pass does the command advance the phase, update the colony state, and prepare for the next build cycle.

**Usage Syntax:**
```
/ant:continue [options]

Options:
  --force     Skip verification gates (not recommended)
  --no-visual Disable visual display
```

**The Six Iron Laws (Verification Gates):**

1. **Build Verification Gate**: All tasks completed, no failures
2. **Type Check Gate**: No type errors
3. **Lint Gate**: No linting violations
4. **Test Gate**: All tests pass
5. **Security Gate**: No security issues
6. **Diff Gate**: User confirms changes

**Platform Differences:**

Near-identical between platforms. Both implement the same six gates with identical validation logic.

---

### `/ant:seal` - Archive Completed Colony

**Purpose (300+ words):**

The `seal` command represents the ceremonial completion of a colony's lifecycle, archiving a successfully completed colony and marking the achievement of the **Crowned Anthill** milestone. This command is invoked when all phases have been completed, all goals have been achieved, and the colony is ready to be preserved as a permanent record of the work accomplished.

The sealing process is designed to create a comprehensive archive that captures not just the final code state, but the entire journey of the colony—the decisions made, the learnings accumulated, the patterns discovered, and the wisdom earned through the development process. This archive serves multiple purposes: it provides a reference for future colonies working on similar problems, it preserves institutional knowledge that might otherwise be lost, and it creates a sense of closure and accomplishment for the completed work.

The command begins by validating that the colony is truly complete—all phases must have status "completed" and there must be no outstanding tasks. It then creates a comprehensive manifest that documents every aspect of the colony: the original goal, the final state, all phases with their tasks and outcomes, the memory accumulated (instincts, learnings, decisions), any errors encountered and how they were resolved, and the complete event history.

A key feature of the seal command is the promotion of wisdom to QUEEN.md. High-confidence instincts (confidence >= 0.8) and validated learnings are extracted from the colony memory and added to the eternal wisdom document. This ensures that the most valuable insights from the colony are preserved and can be inherited by future colonies, creating a compounding effect where each generation of colonies builds upon the wisdom of its predecessors.

The archive is stored in `.aether/chambers/` with a timestamped name, creating a permanent record that can be browsed later using `/ant:tunnels`. The original colony state is preserved but marked as sealed, preventing further modifications while maintaining the ability to review what was accomplished.

**Usage Syntax:**
```
/ant:seal [options]

Options:
  --no-visual    Disable visual display
```

**Milestone Achieved:** Crowned Anthill

**Archive Contents:**
- Complete colony state
- Phase manifests with all tasks
- Memory (instincts, learnings, decisions)
- Event history
- Error records
- Wisdom promoted to QUEEN.md

---

### `/ant:entomb` - Archive to Chambers

**Purpose (300+ words):**

The `entomb` command provides an alternative archival mechanism that preserves colony knowledge while resetting state for new work. Unlike `seal`, which marks completion, `entomb` is used when the colony has accumulated valuable knowledge but needs to be archived in a structured chamber format before starting fresh.

The entombment process creates a chamber-based archive using the `chamber-create` and `chamber-verify` utilities. Chambers provide a more structured storage format than the simple archive created by `seal`, organizing the colony's artifacts into logical compartments that can be independently accessed and reviewed.

The command begins by validating the current colony state and confirming that the user wants to proceed with entombment. This is a destructive operation for the current colony state (though knowledge is preserved), so explicit confirmation is required.

Once confirmed, the command extracts all valuable knowledge from the colony: high-confidence instincts, validated learnings, important decisions, and error patterns. This knowledge is preserved in the chamber structure while the colony state is reset to allow new work to begin.

The chamber structure includes:
- **Foundation**: Original goal and initialization context
- **Structure**: Phase plans and task definitions
- **Construction**: Build outputs and artifacts
- **Knowledge**: Extracted instincts and learnings
- **History**: Event log and decision record

After entombment, the colony can be reinitialized with a new goal, optionally inheriting wisdom from the entombed chamber. This creates a lineage relationship between colonies, where each generation can build upon the knowledge of its predecessors.

**Usage Syntax:**
```
/ant:entomb [options]

Options:
  --force        Skip confirmation
  --no-visual    Disable visual display
```

**Chamber Structure:**
```
.aether/chambers/<chamber-id>/
├── foundation/     # Goal, context, initialization
├── structure/      # Plans, phases, tasks
├── construction/   # Build outputs, artifacts
├── knowledge/      # Instincts, learnings
└── history/        # Events, decisions
```

---

### `/ant:lay-eggs` - Colony Reproduction

**Purpose (300+ words):**

The `lay-eggs` command enables colony reproduction, creating a new colony that inherits wisdom from the current colony while pursuing a new goal. This command embodies the evolutionary aspect of the Aether system, where successful colonies can spawn offspring that carry forward their accumulated knowledge.

The command is designed for scenarios where the current colony has achieved significant progress and accumulated valuable instincts and learnings, but the user wants to pursue a related (or different) goal without losing that accumulated wisdom. Rather than starting from scratch, the new colony begins with the benefit of its parent's experience.

The process begins by extracting high-confidence knowledge from the current colony. This includes instincts with confidence >= 0.7, validated learnings from completed phases, and important decisions that were made during development. This knowledge is packaged into a "seed" that will be planted in the new colony.

The user provides a new goal for the offspring colony, which may be related to the parent's goal (a new feature, a different implementation approach) or entirely different. The command then initializes the new colony with this goal, injecting the inherited knowledge into its memory structure.

A unique feature of lay-eggs is the creation of a lineage record that tracks the relationship between parent and offspring colonies. This enables the system to understand evolutionary relationships and potentially cross-pollinate knowledge across related colonies in the future.

The offspring colony begins with the same system files and configuration as its parent, but with a fresh state, new session ID, and inherited memory. It can then proceed through the normal lifecycle: plan, build, continue, and eventually seal or entomb, potentially spawning its own offspring in turn.

**Usage Syntax:**
```
/ant:lay-eggs "<new goal>" [options]

Options:
  --no-visual    Disable visual display
```

**Inheritance Rules:**
- Instincts with confidence >= 0.7 are inherited
- Validated learnings are inherited
- Decisions are inherited as historical context
- Parent's goal is recorded but not inherited
- System files are shared (not duplicated)

---

## Command Reference - Pheromone Commands

### `/ant:focus` - Guide Attention

**Purpose (300+ words):**

The `focus` command implements the FOCUS pheromone signal, a normal-priority guidance mechanism that directs the colony's attention toward specific areas without establishing hard constraints. This command represents the gentlest form of user guidance in the Aether system, suggesting where the colony should concentrate its efforts while preserving full autonomy in how those efforts are applied.

The focus mechanism is based on the biological concept of pheromone trails that guide ant behavior. Just as real ants follow chemical trails to food sources, Aether workers follow focus signals to prioritize certain aspects of the codebase or problem domain. However, unlike real pheromones that fade over time, Aether focus signals persist until explicitly removed, providing durable guidance across multiple build cycles.

When a user invokes `/ant:focus`, they are essentially saying "pay attention to this area" without specifying exactly what to do or how to do it. This is particularly useful in several scenarios: when the user knows that certain parts of the codebase are more important than others, when there are known complexities that require extra care, when specific architectural patterns should be emphasized, or when certain edge cases need particular attention.

The command works by appending the focus area to the `focus` array in `constraints.json`. This file serves as the persistent store for all pheromone signals, maintaining them across context clears and session boundaries. Workers check this file during task execution and adjust their behavior accordingly, giving priority to areas that have been marked with focus signals.

Focus areas are limited to a maximum of 5 active signals at any time. This constraint prevents the guidance system from becoming overwhelmed with too many priorities, which would effectively mean no priorities at all. If a user attempts to add a sixth focus area, they must first remove an existing one, forcing conscious prioritization.

The focus command is non-destructive and has no side effects beyond updating the constraints file. It can be invoked at any time, even before a colony is initialized, though the signals will only take effect once a colony exists and workers begin executing tasks.

**Usage Syntax:**
```
/ant:focus "<area to focus on>"
```

**Examples:**
```
/ant:focus "authentication module"
/ant:focus "error handling"
/ant:focus "performance optimization"
```

**State Modification:**
- Appends to `constraints.json` `focus[]` array
- Maximum 5 active focus areas
- Persists across sessions

**Platform Differences:**

Identical between Claude Code and OpenCode. Both use the same constraints.json format and storage location.

---

### `/ant:redirect` - Hard Constraint

**Purpose (300+ words):**

The `redirect` command implements the REDIRECT pheromone signal, a high-priority constraint mechanism that establishes inviolable boundaries for the colony. Unlike the gentle guidance of FOCUS, REDIRECT creates hard constraints that workers must respect, effectively saying "do not do this" with the force of a prohibition.

The redirect mechanism addresses a critical need in autonomous software development: preventing the system from repeating known mistakes or violating established constraints. When a user knows that certain approaches don't work, that specific patterns cause problems, or that particular technologies are off-limits, they can use REDIRECT to encode this knowledge into the colony's behavior profile.

The command creates constraints with type "AVOID" in the constraints.json file. These constraints are treated as absolute prohibitions—workers will not use the specified patterns, approaches, or technologies regardless of other considerations. This makes REDIRECT the most powerful form of user guidance in the Aether system, and it should be used judiciously for constraints that are truly non-negotiable.

Common use cases for REDIRECT include: avoiding specific libraries with known issues, prohibiting certain design patterns that have caused problems in the past, preventing modifications to sensitive parts of the codebase, banning approaches that conflict with architectural decisions, and blocking technologies that don't meet compliance requirements.

The command enforces a maximum of 10 active redirect constraints. This limit prevents the constraint system from becoming so restrictive that the colony cannot function effectively. Each redirect should represent a genuine hard constraint, not a preference or suggestion. If users find themselves wanting more than 10 redirects, they should consider whether some of the constraints are actually preferences that could be expressed as FOCUS or FEEDBACK signals instead.

Redirects are stored in the `constraints` array in constraints.json, separate from the `focus` array used by the FOCUS command. This separation allows workers to apply different logic for guidance (focus) versus prohibition (redirect).

**Usage Syntax:**
```
/ant:redirect "<pattern to avoid>"
```

**Examples:**
```
/ant:redirect "using eval()"
/ant:redirect "modifying the database schema"
/ant:redirect "adding new dependencies"
```

**State Modification:**
- Appends to `constraints.json` `constraints[]` array with type "AVOID"
- Maximum 10 active redirect constraints
- Higher priority than focus signals

---

### `/ant:feedback` - Gentle Adjustment

**Purpose (300+ words):**

The `feedback` command implements the FEEDBACK pheromone signal, a low-priority adjustment mechanism that allows users to provide observations and suggestions based on what they've seen the colony do. Unlike FOCUS (which guides future attention) or REDIRECT (which prohibits), FEEDBACK captures lessons learned from past behavior and encodes them as instincts that influence future decisions.

The feedback mechanism represents the colony's capacity for learning from experience. When a user observes the colony doing something well or poorly, they can provide feedback that either reinforces the successful behavior (creating a positive instinct) or discourages the problematic behavior (creating a negative instinct or redirect). Over time, these accumulated feedback signals shape the colony's behavior profile, making it more aligned with user preferences and project requirements.

The command creates both a signal entry and an instinct entry. The signal provides immediate context about the observation, while the instinct persists in the colony's memory with an initial confidence of 0.7. This confidence level indicates a moderate degree of certainty—the instinct is worth considering but not absolute. As the instinct is applied in future builds, its confidence is adjusted based on whether it leads to successful outcomes, creating a self-tuning guidance system.

Feedback is particularly valuable after build completion, when the user has seen what the colony produced and can provide specific observations about what worked and what didn't. This might include comments on code style, architectural decisions, testing approach, documentation quality, or any other aspect of the colony's output.

The command is designed to be lightweight and non-disruptive. It can be invoked quickly without interrupting workflow, making it easy for users to provide continuous feedback as they observe the colony's behavior. This creates a tight feedback loop that accelerates the alignment between user expectations and colony behavior.

Unlike FOCUS and REDIRECT, which are stored in constraints.json, FEEDBACK creates entries in both the signals array (for immediate context) and the instincts array (for persistent memory) in COLONY_STATE.json. This dual storage ensures that feedback is both visible in the current context and available for future reference.

**Usage Syntax:**
```
/ant:feedback "<observation or suggestion>"
```

**Examples:**
```
/ant:feedback "The authentication code was well-structured"
/ant:feedback "Tests should cover more edge cases"
/ant:feedback "Prefer composition over inheritance"
```

**State Modification:**
- Creates signal entry in `COLONY_STATE.json` `signals[]`
- Creates instinct entry in `COLONY_STATE.json` `memory.instincts[]`
- Initial confidence: 0.7
- Confidence adjusts based on future outcomes

---

## Command Reference - Status & Information Commands

### `/ant:status` - Colony Dashboard

**Purpose (300+ words):**

The `status` command provides a comprehensive dashboard view of the colony's current state, offering at-a-glance visibility into progress, constraints, memory, and next steps. This command serves as the primary interface for understanding where the colony stands in its lifecycle and what actions are available to move forward.

The status display is designed to be information-dense yet scannable, presenting key metrics in a visually organized format that allows users to quickly assess colony health and progress. The command aggregates data from multiple sources—COLONY_STATE.json for core state, constraints.json for guidance signals, the dreams directory for session notes, and various utility functions for computed metrics—to create a holistic view of the colony.

The dashboard begins with the colony's goal, truncated to fit the display if necessary. This serves as a constant reminder of what the colony is working toward, grounding all subsequent information in the context of the overall objective. The goal is the north star that guides all colony activities, and its prominent placement emphasizes its importance.

Phase information shows the current position in the project lifecycle: which phase is active, how many phases exist in total, and what the phase is named. Task progress within the current phase is displayed as a completion ratio, giving immediate visibility into how much work remains in the current sprint.

The constraints section summarizes the guidance signals that are currently active: how many focus areas are defined, how many redirect constraints are in place. This reminds users of the guidance they've provided and helps them understand how the colony is being steered.

The memory section shows how much the colony has learned: the total number of instincts accumulated, how many have high confidence (>= 0.7), and optionally the top 3 strongest instincts. This demonstrates the colony's growth and the accumulation of wisdom over time.

The flags section displays any blockers, issues, or notes that have been flagged, categorized by severity. This provides visibility into potential problems that might prevent advancement.

Finally, the dashboard suggests the next logical command based on the colony's current state, making it easy for users to know what to do next without having to reason through the colony lifecycle manually.

**Usage Syntax:**
```
/ant:status
```

**Display Format:**
```
       .-.
      (o o)  AETHER COLONY
      | O |  Status Report
       `-`
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

👑 Goal: <goal (truncated to 60 chars)>

📍 Phase <N>/<M>: <phase name>
   Tasks: <completed>/<total> complete

🎯 Focus: <focus_count> areas | 🚫 Avoid: <constraints_count> patterns
🧠 Instincts: <total> learned (<high_confidence> strong)
🚩 Flags: <blockers> blockers | <issues> issues | <notes> notes
🏆 Milestone: <milestone> (<version>)
💭 Dreams: <dream_count> recorded (latest: <latest_dream>)

State: <state>
Next:  <suggested_command>   <phase_context>
```

**Information Sources:**

| Display Element | Source File | Key |
|-----------------|-------------|-----|
| Goal | COLONY_STATE.json | `goal` |
| Phase | COLONY_STATE.json | `current_phase`, `plan.phases` |
| Tasks | COLONY_STATE.json | `plan.phases[N].tasks` |
| Focus | constraints.json | `focus.length` |
| Constraints | constraints.json | `constraints.length` |
| Instincts | COLONY_STATE.json | `memory.instincts` |
| Flags | Utility | `flag-check-blockers` |
| Milestone | COLONY_STATE.json | `milestone` |
| Dreams | File system | `.aether/dreams/*.md` |

**Suggested Command Logic:**

| State | Suggested Command |
|-------|-------------------|
| IDLE | `/ant:init` |
| READY | `/ant:build {next_phase}` |
| EXECUTING | `/ant:continue` |
| PLANNING | `/ant:plan` |

---

### `/ant:phase` - Phase Details

**Purpose (300+ words):**

The `phase` command provides detailed information about a specific phase in the colony's plan, offering deep visibility into the tasks, success criteria, and current status of that phase. While `status` gives an overview of the entire colony, `phase` zooms in on a particular unit of work, showing the complete picture of what that phase entails.

This command is particularly useful when preparing to build a phase, reviewing what was accomplished in a completed phase, or understanding the requirements before starting work. It displays the phase's name and description, lists all tasks with their current status, shows the success criteria that define phase completion, and indicates any dependencies on other phases.

The phase command reads directly from the plan stored in COLONY_STATE.json, ensuring that the information displayed is always current. If the plan has been updated or tasks have been completed, the phase command reflects these changes immediately.

For each task within the phase, the command displays the task ID, description, current status (pending, in_progress, completed), and any dependencies on other tasks. This dependency information is crucial for understanding the order in which tasks should be executed and which tasks can be run in parallel.

The success criteria section shows the conditions that must be met for the phase to be considered complete. These criteria are defined during the planning phase and serve as the definition of done for the phase. They might include requirements like "all tests pass," "code coverage exceeds 80%," or "documentation is complete."

If the phase has already been built, the command also shows completion information: when it was built, how many workers were spawned, what files were created or modified, and any learnings that were extracted from the build process.

The command supports an optional phase number argument, allowing users to view details for any phase in the plan, not just the current one. If no phase number is provided, it defaults to the current phase.

**Usage Syntax:**
```
/ant:phase [phase_number]

Arguments:
  phase_number    Optional phase number (defaults to current phase)
```

**Examples:**
```
/ant:phase       # Show current phase details
/ant:phase 1     # Show phase 1 details
/ant:phase 3     # Show phase 3 details
```

**Display Elements:**

| Element | Description |
|---------|-------------|
| Phase ID | Numeric identifier (1-indexed) |
| Phase Name | Human-readable name |
| Description | Detailed phase description |
| Status | Current phase status |
| Tasks | List with IDs, descriptions, statuses |
| Success Criteria | Conditions for phase completion |
| Dependencies | Phases that must complete first |
| Completion Info | Build timestamp, workers spawned (if completed) |

---

### `/ant:flags` - List Active Flags

**Purpose (300+ words):**

The `flags` command displays all active flags in the colony, organized by type and severity. Flags represent issues, blockers, warnings, and notes that have been identified during colony operation, providing a centralized view of potential problems and important observations.

The flag system serves multiple purposes in the Aether workflow. Blockers prevent phase advancement until resolved, ensuring that critical issues are addressed before progress continues. Issues represent high-priority concerns that should be addressed but don't necessarily block progress. Notes capture low-priority observations that may be useful for future reference but don't require immediate action.

The flags command aggregates flags from multiple sources: flags created explicitly by users with `/ant:flag`, flags generated automatically during builds when issues are detected, flags created by the chaos testing agent when edge cases are found, and flags extracted from worker reports when they encounter problems.

Each flag displayed includes its ID, type (blocker, issue, note), severity level, description, creation timestamp, and resolution status. Flags can be filtered by type, allowing users to focus on blockers when preparing to continue or review all notes when planning the next phase.

The command also provides summary statistics: total flags, flags by type, average age of open flags, and resolution rate. These metrics help users understand the overall health of the colony and whether issues are being addressed promptly or accumulating.

For blockers specifically, the command highlights which ones are currently preventing advancement and what actions are needed to resolve them. This is particularly valuable when running `/ant:continue` fails due to unresolved blockers, as it provides a clear remediation path.

The flags system integrates with the constraint system—some flags may suggest new redirects or focus areas, and resolved flags may generate feedback signals that improve future colony behavior.

**Usage Syntax:**
```
/ant:flags [options]

Options:
  --type <blocker|issue|note>    Filter by flag type
  --resolved                     Include resolved flags
```

**Examples:**
```
/ant:flags              # Show all active flags
/ant:flags --type blocker   # Show only blockers
/ant:flags --resolved     # Show all flags including resolved
```

**Flag Types:**

| Type | Priority | Blocks Advancement | Use Case |
|------|----------|-------------------|----------|
| Blocker | Critical | Yes | Critical issues must be resolved |
| Issue | High | No | Important concerns to address |
| Note | Low | No | Observations for reference |

---

### `/ant:flag` - Create Flag

**Purpose (300+ words):**

The `flag` command creates a new flag in the colony, marking an issue, blocker, or note for attention. This command provides a mechanism for users to explicitly flag concerns that they want the colony to be aware of or address.

Flags serve as persistent markers that survive across context clears and session boundaries, ensuring that important observations aren't lost. When a user identifies a problem, concern, or observation that should be tracked, they can use `/ant:flag` to create a formal record that appears in status displays, blocks advancement (for blockers), and can be referenced by workers during task execution.

The command requires the user to specify the flag type (blocker, issue, or note) and provide a description. The type determines how the flag is treated by the system: blockers prevent phase advancement until resolved, issues are high-priority warnings that don't block but should be addressed, and notes are low-priority observations for reference.

When creating a flag, the command automatically captures metadata including the creation timestamp, the current phase, and the user context. This provenance information helps understand when and why the flag was created, which is valuable for prioritization and resolution.

The flag is stored in the colony state and immediately becomes visible in `/ant:status` displays and `/ant:flags` listings. If it's a blocker, it will prevent `/ant:continue` from advancing the phase until explicitly resolved.

Flags can be resolved through multiple mechanisms: automatically when the underlying issue is fixed, manually by the user with a resolution command, or by workers when they address the flagged concern during task execution. The resolution process captures how the flag was resolved, creating an audit trail.

The flag system integrates with the learning system—resolved flags may generate instincts or feedback signals that improve future colony behavior. For example, if a particular pattern consistently causes flags, the system may learn to avoid that pattern automatically.

**Usage Syntax:**
```
/ant:flag --type <type> "<description>"

Options:
  --type <blocker|issue|note>    Required flag type
  --auto-resolve <condition>     Optional auto-resolution condition
```

**Examples:**
```
/ant:flag --type blocker "Tests are failing in auth module"
/ant:flag --type issue "Code coverage is below target"
/ant:flag --type note "Consider refactoring the database layer"
```

**Flag Creation Flow:**
1. Parse arguments (type, description, auto-resolve condition)
2. Validate type is one of: blocker, issue, note
3. Generate unique flag ID
4. Capture metadata (timestamp, phase, context)
5. Store in colony state
6. Update displays
7. If blocker, update advancement gates

---

### `/ant:history` - Browse Event History

**Purpose (300+ words):**

The `history` command displays the colony's event log, providing a chronological record of all significant activities that have occurred during the colony's lifecycle. This audit trail serves multiple purposes: debugging (understanding what happened when), accountability (tracking decisions and their outcomes), learning (identifying patterns in colony behavior), and recovery (reconstructing state after issues).

The event log is stored in COLONY_STATE.json as an array of event strings, each formatted as: `timestamp|event_type|actor|description`. This compact format balances readability with storage efficiency, making it feasible to maintain a comprehensive history without excessive storage overhead.

The history command can display the full event log or filter by various criteria: event type (initialization, build, phase completion, etc.), actor (which command or worker generated the event), time range (events within a specific period), or search terms (events matching a pattern).

Each event displayed includes its timestamp (in ISO-8601 format), the type of event (categorized for easy filtering), the actor that generated it (command name, worker ID, or system), and a human-readable description of what occurred.

The command supports pagination for colonies with extensive histories, showing events in reverse chronological order (newest first) with options to page through older events. This ensures that even long-running colonies can navigate their history efficiently.

Special event types are highlighted for visibility: errors are shown in red (if color is enabled), warnings in yellow, milestones in green, and normal operations in the default color. This visual encoding makes it easy to spot important events at a glance.

The history command also provides summary statistics: total events, events by type, average events per day, and the longest gap between events. These metrics help users understand the colony's activity level and whether there are concerning periods of inactivity.

**Usage Syntax:**
```
/ant:history [options]

Options:
  --type <event_type>     Filter by event type
  --actor <actor_name>    Filter by actor
  --since <timestamp>     Show events since date
  --limit <n>             Maximum events to show
```

**Examples:**
```
/ant:history                    # Show all events
/ant:history --type build       # Show only build events
/ant:history --limit 20         # Show last 20 events
/ant:history --since 2026-01-01 # Show events since Jan 1
```

**Event Format:**
```
<ISO-8601 timestamp>|<event_type>|<actor>|<description>
```

**Common Event Types:**

| Type | Description | Example Actor |
|------|-------------|---------------|
| colony_initialized | Colony created | init |
| plan_generated | Plan created | plan |
| build_started | Build began | build |
| build_completed | Build finished | build |
| phase_completed | Phase finished | continue |
| flag_created | Flag added | flag |
| flag_resolved | Flag cleared | system |

---

### `/ant:help` - Command Reference

**Purpose (300+ words):**

The `help` command provides a comprehensive reference for all available Aether commands, serving as the primary documentation interface for users who need to understand command syntax, options, and usage patterns. This command is the gateway to self-discovery within the Aether system, enabling users to learn capabilities without external documentation.

The help system is designed to be contextual and progressive. When invoked without arguments, it displays a categorized list of all commands with brief descriptions, organized by functional area (lifecycle, pheromone, status, etc.). This gives users a high-level view of what's available and helps them locate the command they need.

When invoked with a command name argument, `/ant:help` displays detailed documentation for that specific command, including its purpose, syntax, options, examples, and related commands. This focused view provides all the information needed to use the command effectively.

The help content is sourced directly from the command files themselves, ensuring that documentation is always in sync with implementation. The frontmatter description, usage examples embedded in instructions, and argument specifications are all extracted to build the help display dynamically.

For OpenCode users, the help command includes an additional section explaining argument syntax differences between platforms. OpenCode handles multi-word arguments differently than Claude Code, and this guidance helps users format their commands correctly.

The help display uses visual formatting to improve readability: command names are highlighted, arguments are shown with type indicators, optional elements are marked, and examples are clearly separated from reference material.

The command also supports a `--examples` flag that shows only usage examples for quick reference, and a `--verbose` flag that includes implementation details like which utilities are called and what files are modified.

**Usage Syntax:**
```
/ant:help [command_name] [options]

Options:
  --examples    Show only usage examples
  --verbose     Include implementation details
```

**Examples:**
```
/ant:help           # Show all commands
/ant:help build     # Show detailed help for build
/ant:help --examples # Show examples for all commands
```

**Help Categories:**

| Category | Commands |
|----------|----------|
| Lifecycle | init, plan, build, continue, seal, entomb, lay-eggs |
| Pheromone | focus, redirect, feedback |
| Status | status, phase, flags, flag, history |
| Session | watch, pause-colony, resume-colony, resume, update |
| Advanced | swarm, chaos, archaeology, oracle, colonize, organize |

---

## Command Reference - Session Management Commands

### `/ant:watch` - Live Visibility

**Purpose (300+ words):**

The `watch` command provides real-time visibility into colony activity through a tmux-based live display system. This command creates a persistent monitoring interface that shows active workers, their current tasks, tool usage statistics, and overall colony progress, updating dynamically as the colony operates.

The watch mechanism addresses a fundamental challenge in multi-agent systems: understanding what is happening right now. While status commands provide snapshots, watch provides a continuous stream of information that reflects the colony's current state as it evolves.

When invoked, the command initializes a tmux session (if not already running) and launches the swarm display system. The display shows multiple panels: an activity panel listing all active ants with their caste, current status, and assigned task; a metrics panel showing tool usage statistics (reads, greps, edits, bash executions) per ant; a trophallaxis panel displaying token consumption metrics; a timing panel showing elapsed time per ant; and a chamber map showing which nest zones have active activity.

The display updates automatically as ants start and complete work, change status, or move between tasks. This creates a living dashboard that reflects the colony's actual activity rather than a static report.

The watch command is particularly valuable during builds, when multiple workers are operating in parallel. It allows users to observe the wave-based execution pattern, see which tasks are running concurrently, and identify bottlenecks or stalled workers.

The command supports multiple view modes: compact (showing only essential information), detailed (including full task descriptions), and metrics-focused (emphasizing resource consumption). Users can switch between modes while watching or specify their preferred mode at launch.

When the user exits the watch display (typically via Ctrl+C), the tmux session remains running in the background, allowing them to reattach later with `tmux attach -t aether-watch`. This persistence enables intermittent monitoring without losing context.

**Usage Syntax:**
```
/ant:watch [options]

Options:
  --mode <compact|detailed|metrics>    View mode
  --detach                             Start in background
```

**Examples:**
```
/ant:watch              # Start watch in default mode
/ant:watch --mode detailed  # Start with detailed view
/ant:watch --detach     # Start in background
```

**Display Panels:**

| Panel | Content |
|-------|---------|
| Activity | Active ants, castes, statuses |
| Metrics | Tool usage per ant (📖 🔍 ✏️ ⚡) |
| Trophallaxis | Token consumption |
| Timing | Elapsed time per ant |
| Chamber Map | Active nest zones |

**Tmux Integration:**
- Session name: `aether-watch`
- Reattach: `tmux attach -t aether-watch`
- Detach: Ctrl+B then D
- Exit: Ctrl+C

---

### `/ant:pause-colony` - Save and Handoff

**Purpose (300+ words):**

The `pause-colony` command creates a comprehensive handoff document that captures the colony's current state, enabling seamless resumption in a future session. This command is essential for long-running projects that span multiple conversation sessions, ensuring that context is preserved across the gaps between sessions.

The pause mechanism addresses the reality that AI assistants operate within context windows that may be cleared or lost. When a user needs to step away, switch contexts, or end a session, they can use `/ant:pause-colony` to create a persistent record that the next session can use to resume exactly where they left off.

The command creates a HANDOFF.md file in `.aether/` that contains: the current phase and task context, recent activity summary, active constraints and focus areas, pending flags or blockers, learned instincts that should be preserved, files currently being worked on, and next steps that were planned.

In addition to the handoff document, the command updates the session tracking file to indicate that the colony is paused. This status is checked by other commands (like `status` and `init`) to provide appropriate guidance when a user returns.

The pause command also creates a checkpoint of the current git state, ensuring that the code state is preserved along with the colony state. This checkpoint can be restored if needed, providing a complete snapshot of the moment the colony was paused.

When the user returns and runs `/ant:resume-colony` or `/ant:status`, the handoff document is detected and its contents are displayed, giving immediate context about what was happening when the session ended. The document is then removed to indicate that the handoff has been acknowledged.

The pause mechanism integrates with the freshness detection system—the handoff document includes a timestamp, and if too much time has passed, the resume command may recommend starting fresh or reviewing the plan before continuing.

**Usage Syntax:**
```
/ant:pause-colony [options]

Options:
  --message "<note>"    Add a note to the handoff
  --no-checkpoint       Skip git checkpoint
```

**Examples:**
```
/ant:pause-colony                    # Pause with standard handoff
/ant:pause-colony --message "Need to review auth approach"  # With note
```

**Handoff Contents:**
- Current phase and task
- Recent activity (last 10 events)
- Active constraints
- Pending flags
- High-confidence instincts
- Files in progress
- Next planned steps
- User message (if provided)

---

### `/ant:resume-colony` - Restore from Pause

**Purpose (300+ words):**

The `resume-colony` command restores a colony from a paused state, reading the HANDOFF.md document created by `/ant:pause-colony` and reestablishing the context needed to continue work. This command enables seamless continuity across disjointed conversation sessions, bridging the gap between past and present.

The resume process begins by detecting whether a handoff document exists. If found, the command reads and displays its contents, giving the user immediate visibility into what was happening when the colony was paused. This includes the phase that was active, the tasks that were in progress, any constraints or flags that were in effect, and the next steps that had been planned.

After displaying the handoff context, the command removes the HANDOFF.md file to indicate that the handoff has been acknowledged and processed. This prevents the handoff from being displayed multiple times.

The command then updates the session tracking to mark the colony as active again, ensuring that freshness detection and other session-aware features work correctly.

If no handoff document is found, the command checks whether there's an existing colony with valid state. If so, it displays the current status and suggests next steps. If not, it guides the user to initialize a new colony.

The resume command also performs a freshness check on the handoff document. If the pause occurred too long ago (configurable threshold, default 7 days), the command warns the user that the context may be stale and recommends reviewing the plan before continuing.

For colonies that were paused during active builds, the resume command can optionally verify the current state of those builds, checking whether files have been modified externally and whether the build context is still valid.

**Usage Syntax:**
```
/ant:resume-colony [options]

Options:
  --force             Resume even if handoff is stale
  --review            Review plan before resuming
```

**Examples:**
```
/ant:resume-colony          # Resume from handoff
/ant:resume-colony --force  # Resume even if stale
```

**Resume Flow:**
1. Check for HANDOFF.md
2. If found: display contents, remove file
3. If not found: check for existing colony
4. Update session tracking
5. Check freshness
6. Display status and next steps

---

### `/ant:resume` - Claude-Specific Resume

**Purpose (300+ words):**

The `resume` command provides Claude Code-specific session recovery functionality, offering enhanced capabilities for restoring context in Claude Code environments. This command exists only in Claude Code (OpenCode uses `resume-colony` instead) and provides deeper integration with Claude-specific features.

The resume command extends the functionality of `resume-colony` with additional capabilities tailored to Claude Code's architecture. It can restore not just the colony state but also Claude-specific context like conversation history references, tool use patterns, and model configuration.

The command begins by checking for a handoff document and displaying its contents, similar to `resume-colony`. However, it also checks for Claude-specific session files that may contain additional context about the previous session.

One key feature of the Claude-specific resume is the ability to restore the exact model configuration that was in use when the session was paused. This ensures that if the user had specified a particular model for builds, that preference is preserved and restored.

The command also integrates with Claude's context window management, providing guidance on how much context is available and what information might need to be reloaded. This helps users understand the limitations of resumption and what they may need to re-explain.

If the colony was paused during an active build, the resume command can spawn a recovery worker that analyzes what was completed and what remains, helping to reconstruct the build context more accurately than a simple status display.

The resume command also handles edge cases that are specific to Claude Code, such as session migration between different Claude instances, recovery from interrupted tool calls, and restoration of pending user confirmations.

**Usage Syntax:**
```
/ant:resume [options]

Options:
  --full-context      Attempt full context restoration
  --quick             Quick resume without analysis
```

**Claude-Specific Features:**
- Model configuration restoration
- Context window guidance
- Recovery worker spawning
- Session migration handling
- Pending confirmation restoration

---

### `/ant:update` - Update from Hub

**Purpose (300+ words):**

The `update` command synchronizes the local Aether installation with the global hub, ensuring that the colony has the latest command definitions, utilities, and documentation. This command is the mechanism by which improvements and bug fixes are distributed to all colonies.

The update process begins by checking the current version against the hub version. If the hub has a newer version, the command downloads and installs the updated files. If the local installation is already current, it reports this and exits.

The command updates multiple components: command definitions in `.claude/commands/ant/` and/or `.opencode/commands/ant/`, system utilities in `.aether/`, agent definitions, and documentation files. Each component is updated atomically to prevent partial updates that could leave the system in an inconsistent state.

Before applying updates, the command creates a backup of the current installation. This backup can be restored if the update causes issues, providing a rollback mechanism for safety.

The update command respects local modifications. If a user has customized command files, the update process detects this and either preserves the customizations (if they're compatible) or warns the user about conflicts that need to be resolved.

After updating, the command runs verification tests to ensure that the new installation is functional. These tests check that all commands can be loaded, that utilities execute correctly, and that the basic colony lifecycle can be performed.

The command also updates the version tracking file and logs the update event to the colony history, creating an audit trail of when updates were applied.

If the update includes breaking changes, the command displays migration guidance explaining what has changed and what actions users may need to take to adapt their colonies.

**Usage Syntax:**
```
/ant:update [options]

Options:
  --check-only        Check for updates without applying
  --force             Apply update even if local modifications exist
  --rollback          Restore previous version
```

**Examples:**
```
/ant:update              # Update to latest version
/ant:update --check-only # Check if update available
/ant:update --rollback   # Rollback to previous version
```

**Update Components:**
- Command definitions
- System utilities (aether-utils.sh)
- Agent definitions
- Documentation
- Version tracking

---

## Command Reference - Advanced/Utility Commands

### `/ant:swarm` - Parallel Bug Investigation

**Purpose (300+ words):**

The `swarm` command deploys four parallel scouts to investigate and resolve stubborn bugs that have resisted normal fixes. This command represents the "nuclear option" for problem resolution, bringing multiple investigative approaches to bear simultaneously on a single issue.

The swarm mechanism is designed for bugs that persist despite multiple fix attempts—issues where the root cause is unclear, where previous fixes have failed, or where the problem spans multiple parts of the codebase. By deploying scouts with different specializations simultaneously, the swarm can gather diverse perspectives and cross-compare findings to identify solutions that might not be apparent from any single angle.

The four scouts deployed are: the Archaeologist, who investigates git history to understand when and why the bug was introduced; the Pattern Hunter, who searches for similar working code that might provide solution templates; the Error Analyst, who traces through error chains to identify the actual root cause beneath surface symptoms; and the Web Researcher, who searches external sources for known issues and community solutions.

Each scout operates independently, exploring the problem from their specialized perspective. They return structured JSON findings that include confidence scores, evidence summaries, and suggested fixes. The Queen then cross-compares these findings, looking for consensus among scouts (high confidence), disagreements that warrant further investigation, and the highest-confidence solution to implement.

Before attempting any fixes, the swarm command creates a git checkpoint that can be rolled back if the fix fails. This safety mechanism ensures that the investigation doesn't make the problem worse.

After cross-comparison, the highest-confidence solution is implemented and verified through build and test execution. If verification passes, the fix is confirmed and learnings are extracted. If verification fails, the checkpoint is restored and the next-highest-confidence solution is attempted.

The swarm command tracks failure counts per issue. If an issue resists three swarm attempts, the command raises an architectural concern flag, suggesting that the problem may be fundamental to the design rather than implementational.

**Usage Syntax:**
```
/ant:swarm "<problem description>"

Options:
  --watch             Show real-time swarm display
  --max-attempts <n>  Maximum fix attempts (default: 3)
```

**Examples:**
```
/ant:swarm "Tests keep failing in auth module"
/ant:swarm "TypeError: Cannot read property 'id' of undefined"
/ant:swarm "API returns 500 but I can't find the cause"
```

**The Four Scouts:**

| Scout | Caste | Investigation Focus |
|-------|-------|---------------------|
| Archaeologist | 🏛️ | Git history, when bug was introduced |
| Pattern Hunter | 🔍 | Similar working code, patterns |
| Error Analyst | 🔍 | Error chain, root cause |
| Web Researcher | 🔍 | External sources, known issues |

---

### `/ant:chaos` - Resilience Testing

**Purpose (300+ words):**

The `chaos` command performs systematic resilience testing by probing edge cases, boundary conditions, and unexpected inputs that might cause the system to fail. This command embodies the principle that software should be tested not just for expected behavior but for graceful handling of the unexpected.

The chaos testing approach is inspired by Netflix's Chaos Monkey and similar resilience engineering practices. Rather than waiting for failures to occur in production, the chaos command intentionally introduces perturbations to verify that the system handles them gracefully.

The command tests five categories of resilience: edge cases (unusual but valid inputs), boundary conditions (limits of ranges and buffers), error handling (failure paths and recovery), state corruption (invalid or inconsistent state), and unexpected inputs (malformed or malicious data).

For each category, the chaos command spawns a Chaos ant that generates test cases designed to stress that particular dimension of resilience. The Chaos ant doesn't just find problems—it attempts to break the system in creative ways, verifying that failures are handled gracefully rather than causing crashes or data corruption.

The command can operate in two modes: targeted chaos, which focuses on specific components or functions identified by the user, and broad chaos, which tests the entire system surface. Targeted chaos is faster and more focused, while broad chaos provides more comprehensive coverage.

After testing, the chaos command reports findings categorized by severity: critical issues that must be fixed immediately, warnings that should be addressed, and observations that are informational. Each finding includes the test case that triggered it, the observed behavior, and recommendations for improvement.

The chaos command integrates with the flag system—issues found are automatically flagged for tracking and resolution. Critical issues become blockers that prevent phase advancement until addressed.

**Usage Syntax:**
```
/ant:chaos [target] [options]

Options:
  --mode <targeted|broad>    Testing mode
  --category <category>      Test only specific category
  --generate-fixes           Attempt to generate fixes for issues
```

**Examples:**
```
/ant:chaos                 # Broad chaos testing
/ant:chaos auth module     # Targeted chaos on auth
/ant:chaos --category edge-cases  # Test only edge cases
```

**Five Chaos Categories:**

| Category | Description | Example Tests |
|----------|-------------|---------------|
| Edge Cases | Unusual valid inputs | Empty strings, single characters |
| Boundary Conditions | Range limits | Max int, buffer overflow |
| Error Handling | Failure paths | Network timeout, disk full |
| State Corruption | Invalid state | Null refs, inconsistent data |
| Unexpected Inputs | Malformed data | Special chars, injection |

---

### `/ant:oracle` - Deep Research

**Purpose (300+ words):**

The `oracle` command launches a deep research loop that runs autonomously in a separate process, investigating topics with iterative refinement until reaching a target confidence level. This command implements the RALF (Recursive Autonomous Learning Flow) pattern for comprehensive knowledge gathering.

The oracle mechanism is designed for research questions that require extensive investigation—topics too large for a single query, areas where initial findings raise new questions, or domains where confidence builds gradually through multiple exploration cycles. Unlike normal commands that complete in one execution, the oracle runs continuously, accumulating findings until it reaches the confidence threshold or iteration limit.

The command begins with a research wizard that configures the investigation. The user specifies the topic, research depth (number of iterations), confidence target, and scope (codebase only, web only, or both). Based on these parameters, the oracle creates a research plan with specific questions to answer.

The oracle runs in a separate tmux session, allowing it to operate independently of the main colony session. Users can check progress with `/ant:oracle status`, stop early with `/ant:oracle stop`, or let it run to completion.

Each iteration follows a consistent pattern: identify knowledge gaps from previous iterations, research those gaps using appropriate tools (Glob, Grep, Read for codebase; WebSearch, WebFetch for external), synthesize findings into the growing knowledge base, and assess confidence. If confidence is below target and iterations remain, the loop continues.

The oracle writes progress to `.aether/oracle/progress.md`, creating a persistent record of the investigation that can be reviewed at any time. This file includes iteration summaries, confidence scores, knowledge gaps identified, and findings accumulated.

When the oracle completes (reaching confidence target, iteration limit, or manual stop), it produces a comprehensive research report summarizing what was learned, confidence in each finding, remaining gaps, and recommendations for action based on the research.

**Usage Syntax:**
```
/ant:oracle [topic] [options]

Subcommands:
  /ant:oracle status      # Check research progress
  /ant:oracle stop        # Halt research

Options:
  --iterations <n>        # Max iterations (5, 15, 30, 50)
  --confidence <pct>      # Target confidence (80, 90, 95, 99)
  --scope <codebase|web|both>
```

**Examples:**
```
/ant:oracle                    # Launch research wizard
/ant:oracle "React patterns"   # Research specific topic
/ant:oracle status             # Check progress
/ant:oracle stop               # Halt research
```

**Research Depth Options:**

| Depth | Iterations | Use Case |
|-------|------------|----------|
| Quick scan | 5 | Surface overview |
| Standard | 15 | Thorough investigation |
| Deep dive | 30 | Exhaustive research |
| Marathon | 50 | Maximum depth |

---

### `/ant:colonize` - Territory Survey

**Purpose (300+ words):**

The `colonize` command performs a comprehensive survey of the existing codebase, creating structured documentation that informs planning and execution. This command is essential for brownfield development, where the colony must work within an existing codebase rather than starting from scratch.

The colonization process creates seven survey documents that together provide a complete picture of the territory: PROVISIONS.md documents the tech stack and dependencies; TRAILS.md maps integration points and data flows; BLUEPRINT.md describes the architecture and component relationships; CHAMBERS.md catalogs directories and their purposes; DISCIPLINES.md captures coding patterns and conventions; SENTINEL-PROTOCOLS.md documents testing approaches; and PATHOGENS.md records known issues and technical debt.

The command spawns a Surveyor ant that systematically explores the codebase using Glob and Grep to understand structure, Read to examine key files, and analysis to identify patterns. The Surveyor doesn't just catalog files—it understands relationships, identifies conventions, and documents the architectural decisions that shaped the codebase.

The survey process is intelligent about prioritization. It focuses first on high-impact areas like entry points, core modules, and configuration files. It then expands to cover supporting code, tests, and documentation. This ensures that the most important information is captured even if the survey is interrupted.

The seven documents are written to `.aether/data/survey/` and are automatically loaded by `/ant:plan` when creating project plans. This integration ensures that planning is informed by actual codebase structure rather than assumptions.

The colonize command includes session freshness detection to prevent stale survey data from being used. If a survey is older than a configurable threshold (default 30 days), the command warns that the data may be outdated and recommends re-running the survey.

**Usage Syntax:**
```
/ant:colonize [options]

Options:
  --quick             # Survey only high-impact areas
  --focus <area>      # Focus on specific directory
  --update            # Update existing survey
```

**Seven Survey Documents:**

| Document | Content | Used By |
|----------|---------|---------|
| PROVISIONS.md | Tech stack, dependencies | plan |
| TRAILS.md | Integration points, data flow | plan |
| BLUEPRINT.md | Architecture, components | plan |
| CHAMBERS.md | Directory structure | plan |
| DISCIPLINES.md | Coding patterns | plan, build |
| SENTINEL-PROTOCOLS.md | Testing approaches | plan, build |
| PATHOGENS.md | Known issues, debt | plan, build |

---

### `/ant:archaeology` - Git History Analysis

**Purpose (300+ words):**

The `archaeology` command excavates the git history of the codebase to understand why code exists, when it was introduced, and what changes have occurred over time. This command provides crucial context that helps workers make informed decisions about modifications.

The archaeology process is inspired by real archaeological methods—layer by layer excavation, artifact analysis, and contextual understanding. The command doesn't just show commit history; it interprets that history to reveal the story of the codebase.

When invoked, the command spawns an Archaeologist ant that investigates specific files or the entire codebase. The Archaeologist uses git commands to examine commit history, blame information, and diffs, building a narrative of how the code evolved.

Key findings include: when specific features were introduced, why certain design decisions were made (from commit messages), who made significant changes, what bugs have been fixed in the past, and what patterns of change exist (refactoring hot spots, stable vs. volatile code).

The archaeology command is particularly valuable before major refactoring or when encountering confusing code. Understanding the history often reveals that seemingly odd code exists for good reasons—workarounds for bugs, compatibility requirements, or performance optimizations that aren't immediately obvious.

The command produces an archaeological report that includes: a timeline of major changes, annotated code showing when each section was last modified, patterns identified (frequently modified files, stable core), and recommendations based on historical context.

Archaeology findings are stored in the colony memory and can be referenced by workers during builds. The `/ant:build` command automatically runs archaeology on files being modified, providing workers with historical context.

**Usage Syntax:**
```
/ant:archaeology [target] [options]

Options:
  --since <date>      # Only history since date
  --deep              # Include file content analysis
  --focus <pattern>   # Focus on commits matching pattern
```

**Examples:**
```
/ant:archaeology              # Analyze entire codebase
/ant:archaeology src/auth.js  # Analyze specific file
/ant:archaeology --since 2026-01-01  # Recent history only
```

**Archaeological Findings:**
- Feature introduction timeline
- Design decision rationale
- Bug fix history
- Refactoring patterns
- Code stability analysis
- Author contributions

---

### `/ant:organize` - Codebase Hygiene

**Purpose (300+ words):**

The `organize` command analyzes codebase hygiene and generates a report on organization, consistency, and maintenance needs. This command provides visibility into the structural health of the codebase, identifying areas that may benefit from cleanup or reorganization.

The organization analysis covers multiple dimensions: file structure (directory organization, naming consistency), code patterns (style consistency, pattern adherence), dependencies (circular dependencies, unused imports), documentation (coverage, freshness), and tests (coverage, organization).

The command spawns a Scout ant that systematically examines the codebase, looking for indicators of organizational health and hygiene issues. The Scout uses Glob to understand structure, Grep to find patterns, and analysis to identify inconsistencies.

Key metrics calculated include: file organization score (how well files follow established structure), naming consistency (adherence to naming conventions), code duplication (repeated code that could be consolidated), documentation coverage (percentage of public APIs documented), and test coverage (percentage of code with tests).

The organize command produces a hygiene report that includes: overall health score, category scores, specific issues found (with file paths and line numbers), recommendations for improvement, and prioritized action items.

Unlike some analysis commands that only identify problems, organize provides actionable recommendations. For each issue found, it suggests specific steps to resolve it and estimates the effort required.

The command integrates with the colony planning system—hygiene issues can be automatically added as tasks to upcoming phases, ensuring that cleanup work is scheduled alongside feature development.

**Usage Syntax:**
```
/ant:organize [options]

Options:
  --category <cat>    # Focus on specific category
  --fix               # Attempt automatic fixes
  --report <format>   # Output format (text|json|md)
```

**Examples:**
```
/ant:organize              # Full hygiene report
/ant:organize --category naming  # Naming analysis only
/ant:organize --fix        # Analyze and fix issues
```

**Hygiene Categories:**

| Category | Metrics | Issues Detected |
|----------|---------|-----------------|
| Structure | Organization score | Misplaced files |
| Naming | Consistency % | Naming violations |
| Dependencies | Complexity | Circular deps |
| Documentation | Coverage % | Missing docs |
| Tests | Coverage % | Untested code |

---

### `/ant:council` - Intent Clarification

**Purpose (300+ words):**

The `council` command facilitates intent clarification by engaging in a structured dialogue with the user to refine goals, resolve ambiguities, and ensure shared understanding. This command addresses the reality that initial goals are often vague or incomplete, requiring elaboration before effective planning can occur.

The council mechanism is inspired by the Socratic method—asking questions to expose assumptions, clarify meanings, and develop more precise understanding. When invoked, the command enters a dialogue mode where it asks the user questions about their goal, probing for specifics that will make the goal actionable.

The dialogue follows a structured pattern: first understanding the problem being solved (why this goal matters), then the desired outcome (what success looks like), then constraints (what must be avoided or included), then priorities (what matters most), and finally context (what background is relevant).

Throughout the dialogue, the council command maintains a running synthesis of what has been learned, periodically summarizing the emerging understanding and confirming with the user that it is accurate. This iterative confirmation prevents misinterpretation and ensures alignment.

The command produces an intent document that captures the clarified goal in structured form: problem statement, success criteria, constraints, priorities, and context. This document can be used to update the colony goal or inform planning.

The council command is particularly valuable when: the initial goal is vague or abstract, different interpretations seem possible, the user seems uncertain about what they want, or there are implicit assumptions that need to be surfaced.

The dialogue is designed to be efficient—questions are targeted to elicit maximum information with minimum user effort. The command avoids asking about things that are already clear and focuses on areas of ambiguity.

**Usage Syntax:**
```
/ant:council [topic] [options]

Options:
  --goal              # Focus on clarifying current goal
  --phase <n>         # Clarify specific phase
  --quick             # Abbreviated clarification
```

**Examples:**
```
/ant:council              # Clarify current goal
/ant:council "auth"       # Clarify auth requirements
/ant:council --phase 2    # Clarify phase 2 scope
```

**Clarification Areas:**
- Problem understanding
- Success criteria
- Constraints
- Priorities
- Context and background

---

### `/ant:verify-castes` - System Status

**Purpose (300+ words):**

The `verify-castes` command checks the status of all caste definitions and verifies that the system is properly configured for multi-agent operation. This diagnostic command ensures that workers can be spawned successfully and that all required components are in place.

The verification process checks multiple aspects of the caste system: agent definitions exist for all castes, model routing is configured correctly, utility functions are available and executable, required directories exist with proper permissions, and the hub installation is complete and current.

For each caste, the command verifies: the agent definition file exists and is valid, the caste emoji is defined, any special configuration is present, and a test worker can be spawned successfully. This comprehensive check ensures that when a command tries to spawn a worker of a particular caste, it will succeed.

The command also tests the model routing system, verifying that the configuration files are valid and that model assignments can be resolved. While actual model routing may not be fully functional (see Known Issues), the verification ensures that the configuration layer is correct.

The verify-castes command produces a status report showing: overall system health, per-caste status, model routing status, utility function status, and any issues found with recommendations for resolution.

If issues are found, the command provides specific guidance on how to resolve them, including commands to run, files to check, and configuration to verify.

This command is particularly valuable after installation, after updates, or when workers fail to spawn unexpectedly. It provides a systematic way to diagnose configuration issues.

**Usage Syntax:**
```
/ant:verify-castes [options]

Options:
  --verbose           # Detailed output
  --fix               # Attempt to fix issues
```

**Examples:**
```
/ant:verify-castes        # Verify all castes
/ant:verify-castes --verbose  # Detailed verification
```

**Verified Components:**
- Agent definitions
- Model routing config
- Utility functions
- Directory structure
- Hub installation

---

## Command Dependencies Graph

The following graph shows dependencies between commands and utilities:

```
                    ┌─────────────┐
                    │   aether    │
                    │   utils.sh  │
                    └──────┬──────┘
                           │
        ┌──────────────────┼──────────────────┐
        │                  │                  │
        ▼                  ▼                  ▼
┌───────────────┐  ┌───────────────┐  ┌───────────────┐
│  init         │  │  plan         │  │  build        │
│  ─────────    │  │  ─────────    │  │  ─────────    │
│  queen-init   │  │  load-state   │  │  checkpoint   │
│  session-init │  │  activity-log │  │  swarm-display│
│  validate-state│  │               │  │  activity-log │
└───────────────┘  └───────────────┘  └───────────────┘
        │                  │                  │
        │                  │                  ▼
        │                  │         ┌───────────────┐
        │                  │         │  continue     │
        │                  │         │  ─────────    │
        │                  │         │  validate-*   │
        │                  │         │  flag-check   │
        │                  │         └───────────────┘
        │                  │
        ▼                  ▼
┌─────────────────────────────────────────────┐
│           Session Management                │
│  ─────────────────────────────────────────  │
│  session-init, session-update, session-*    │
└─────────────────────────────────────────────┘
```

**Key Dependency Patterns:**

| Command | Primary Dependencies |
|---------|---------------------|
| init | queen-init, session-init, validate-state |
| plan | load-state, activity-log, unload-state |
| build | autofix-checkpoint, swarm-display, activity-log |
| continue | flag-check-blockers, validate-state |
| seal | archive utilities, wisdom promotion |
| entomb | chamber-create, chamber-verify |
| swarm | swarm-display, swarm-findings, autofix-checkpoint |
| oracle | session-verify-fresh, progress tracking |

---

## Execution Flow Diagrams

### Build Command Execution Flow

```
User: /ant:build 1
│
├─► Load State
│   ├─ Read COLONY_STATE.json
│   ├─ Validate phase exists
│   └─ Check for blockers
│
├─► Prepare
│   ├─ Create git checkpoint
│   ├─ Load territory survey
│   ├─ Load QUEEN.md
│   └─ Archaeologist scan (if modifying files)
│
├─► Wave Execution
│   ├─ Wave 1: Spawn independent tasks (parallel)
│   │   ├─ Builder 1 ──► Complete
│   │   ├─ Builder 2 ──► Complete
│   │   └─ Builder 3 ──► Failed
│   │
│   ├─ Wave 2: Spawn dependent tasks (sequential)
│   │   └─ Builder 4 ──► Complete
│   │
│   └─ All waves complete
│
├─► Verification
│   ├─ Spawn Watcher ──► Verification report
│   └─ Spawn Chaos ──► Resilience report
│
├─► Synthesis
│   ├─ Aggregate results
│   ├─ Calculate metrics
│   └─ Extract learnings
│
└─► Update & Display
    ├─ Update COLONY_STATE.json
    ├─ Log activity
    └─ Render results
```

### Plan Command Execution Flow

```
User: /ant:plan
│
├─► Initialize
│   ├─ Check for existing plan
│   └─ Load territory survey (if exists)
│
├─► Research Loop (max 4 iterations)
│   │
│   Iteration 1:
│   ├─► Spawn Scout (broad exploration)
│   │   └─ Returns findings + gaps
│   ├─► Spawn Route-Setter
│   │   └─ Returns draft plan
│   └─► Confidence: 45%
│   │
│   Iteration 2:
│   ├─► Spawn Scout (gap-focused)
│   ├─► Spawn Route-Setter
│   └─► Confidence: 72%
│   │
│   Iteration 3:
│   ├─► Spawn Scout (gap-focused)
│   ├─► Spawn Route-Setter
│   └─► Confidence: 84% ✓ (exceeds 80%)
│   │
│   └─► Exit loop (confidence threshold met)
│
├─► Finalize
│   ├─ Write plan to COLONY_STATE.json
│   └─ Update session
│
└─► Display
    └─ Show phases with next steps
```

---

## Platform Differences Analysis

### Summary of Differences

| Aspect | Claude Code | OpenCode | Impact |
|--------|-------------|----------|--------|
| Command Count | 36 | 35 | OpenCode missing `resume` |
| Init Features | Full | Reduced | Missing QUEEN.md, context, session |
| Agent Types | Specialized | General + role | Same capability, different approach |
| Caste Emojis | Combined (🔨🐜) | Single (🔨) | Visual distinction only |
| Argument Handling | Native | Normalized | OpenCode requires normalization step |
| Lines of Code | ~7,500 | ~6,900 | ~600 lines difference |

### Detailed Comparison

**1. Command Naming**
- Claude Code: `resume.md`
- OpenCode: `resume-colony.md` (different name, same function)

**2. Session Initialization (OpenCode Missing)**
- QUEEN.md initialization (Step 1.6)
- CONTEXT.md creation (Step 5)
- Session tracking setup (Step 8)

**3. Agent Type References**

Claude Code:
```markdown
Task tool with `subagent_type="aether-builder"`
Task tool with `subagent_type="aether-watcher"`
Task tool with `subagent_type="aether-chaos"`
```

OpenCode:
```markdown
Task tool with `subagent_type="general-purpose"`
# NOTE: Claude Code uses aether-chaos; OpenCode uses general-purpose with role injection
```

**4. Argument Normalization (OpenCode Only)**

OpenCode includes:
```markdown
### Step -1: Normalize Arguments
Run: `normalized_args=$(bash .aether/aether-utils.sh normalize-args "$@")`
```

**5. Help Command Differences**

OpenCode includes additional section:
```markdown
OPENCODE USERS

  Argument syntax: OpenCode handles multi-word arguments differently than Claude.
  Wrap text arguments in quotes for reliable parsing:
```

**6. Caste Emoji Display**

Claude Code:
```markdown
🔨🐜 Builder  (cyan if color enabled)
👁️🐜 Watcher  (green if color enabled)
🎲🐜 Chaos    (red if color enabled)
```

OpenCode:
```markdown
🔨 Builder  (cyan if color enabled)
👁️ Watcher  (green if color enabled)
🎲 Chaos    (red if color enabled)
```

---

## Consolidation Opportunities

### High Priority

**1. Unify `init.md`**
- Add missing session/context steps to OpenCode
- Estimated effort: Medium
- Value: Restores full functionality for OpenCode users

**2. Standardize naming**
- Align `resume.md` vs `resume-colony.md`
- Estimated effort: Low
- Value: Reduces confusion

**3. Fix caste emojis**
- Use consistent emoji format across platforms
- Estimated effort: Low
- Value: Visual consistency

### Medium Priority

**4. Generate OpenCode from Claude**
- Create transformation script
- Apply automatic transformations:
  - Agent type substitution
  - Emoji simplification
  - Argument normalization insertion
- Estimated effort: High
- Value: Eliminates 13,573 lines of duplication

**5. Add diff checking to CI**
- Prevent drift between platforms
- Fail build if commands are out of sync
- Estimated effort: Medium
- Value: Maintains consistency

**6. Document platform differences**
- Add comments explaining why differences exist
- Helps future maintainers understand constraints
- Estimated effort: Low
- Value: Knowledge preservation

### Low Priority

**7. Consolidate common patterns**
- Extract shared templates for visual mode, state validation
- Create command generator
- Estimated effort: High
- Value: Reduced maintenance burden

**8. Add version metadata**
- Track command versions in frontmatter
- Enable migration paths
- Estimated effort: Medium
- Value: Better version management

---

## Known Issues and Limitations

### Critical Issues

**1. BUG-005/BUG-011: Lock deadlock in flag-auto-resolve**
- Location: `.aether/aether-utils.sh:1022`
- Issue: If jq fails, lock never released -> deadlock
- Workaround: Restart colony session if commands hang on flags
- Status: Unresolved

**2. ISSUE-004: Template path hardcoded to runtime/**
- Location: `.aether/aether-utils.sh:2689`
- Issue: queen-init fails when Aether installed via npm
- Workaround: Use git clone instead of npm install
- Status: Unresolved

### Medium Priority Issues

**3. Model routing UNVERIFIED**
- Configuration exists: `model-profiles.yaml` maps castes to models
- Execution unproven: ANTHROPIC_MODEL may not be inherited by spawned workers
- Test: `/ant:verify-castes` Step 3 spawns test worker
- Status: Needs verification

**4. Error code inconsistency**
- 17+ locations use hardcoded strings instead of `$E_*` constants
- Pattern: early commands use strings, later commands use constants
- Impact: Inconsistent error handling
- Status: Technical debt

### Command-Specific Issues

**5. build.md duplicate lines**
- Has duplicate "Analyze the phase tasks" lines
- Impact: Cosmetic issue only
- Status: Minor

**6. OpenCode init.md missing features**
- Missing Steps 1.6, 5, 8 (QUEEN.md, CONTEXT.md, session init)
- Impact: Reduced functionality for OpenCode users
- Status: By design or oversight?

### Design Limitations

**7. Claude Code Task tool limitations**
- No environment variable support prevents model-per-caste routing
- All workers use default model regardless of configuration
- Archived config: `.aether/archive/model-routing/`
- Status: Platform limitation

**8. Session freshness edge cases**
- Very fast commands (< 1 second) may not trigger freshness detection
- Clock skew between file system and session can cause false positives
- Status: Acceptable trade-off

---

## Appendix

### A. File Size Summary

#### Largest Commands

| Command | Claude | OpenCode | Notes |
|---------|--------|----------|-------|
| `build` | 1,051 | 989 | Most complex |
| `continue` | 1,037 | ~1,037 | Gate-heavy |
| `plan` | 534 | ~534 | Iterative loop |
| `oracle` | 380 | ~380 | Research wizard |
| `swarm` | 380 | ~380 | Parallel scouts |
| `chaos` | 341 | ~341 | 5 scenarios |
| `entomb` | 407 | ~407 | Archive flow |
| `seal` | 337 | ~337 | Crown milestone |
| `init` | 316 | 272 | Missing features |

#### Smallest Commands

| Command | Lines | Purpose |
|---------|-------|---------|
| `focus` | 51 | Simple constraint add |
| `redirect` | 51 | Simple constraint add |
| `feedback` | 51 | Simple constraint add |
| `help` | 113 | Static reference |
| `verify-castes` | 86 | Status display |
| `maturity` | ~95 | Maturity assessment |

### B. Caste Reference

| Caste | Emoji | Role | Used In |
|-------|-------|------|---------|
| Builder | 🔨🐜 | Implementation | build |
| Watcher | 👁️🐜 | Monitoring | build, continue |
| Chaos | 🎲🐜 | Edge case testing | build, chaos |
| Scout | 🔍🐜 | Research | plan, swarm |
| Archaeologist | 🏺🐜 | Git history | build, archaeology |
| Surveyor | 📊🐜 | Territory survey | colonize |
| Oracle | 🔮🐜 | Deep research | oracle |
| Route Setter | 🗺️🐜 | Planning | plan |
| Prime | 🏛️🐜 | Coordination | council |
| Tracker | 🐛🐜 | Bug investigation | swarm |

### C. Milestone Progression

```
First Mound ──► Open Chambers ──► Brood Stable ──► Ventilated Nest ──► Sealed Chambers ──► Crowned Anthill
     │                │                │                 │                   │
     │                │                │                 │                   │
  Initial         Feature          Tests            Performance       Interfaces       Release
  Runnable        Work             Green            Acceptable        Frozen           Ready
```

### D. State File Reference

| File | Purpose | Modified By |
|------|---------|-------------|
| `COLONY_STATE.json` | Core colony state | init, plan, build, continue, seal, entomb |
| `constraints.json` | Pheromone signals | focus, redirect, feedback |
| `session.json` | Session tracking | init, session-* commands |
| `QUEEN.md` | Eternal wisdom | queen-init, seal |
| `CONTEXT.md` | Session recovery | init, context-update |
| `activity.log` | Activity history | All commands |

### E. Command Implementation Patterns

#### Visual Mode Implementation Pattern

Most commands implement visual mode using the following standardized pattern:

```markdown
### Step 0: Initialize Visual Mode (if enabled)

If `visual_mode` is true:
```bash
# Generate session ID
command_id="<command>-$(date +%s)"

# Initialize swarm display
bash .aether/aether-utils.sh swarm-display-init "$command_id"
bash .aether/aether-utils.sh swarm-display-update \
  "Queen" "prime" "excavating" "<Activity description>" \
  "Colony" '{"read":0,"grep":0,"edit":0,"bash":0}' 0 "fungus_garden" 0
```

# Visual header display
```
<emoji> ═══════════════════════════════════════════════════
        <C O M M A N D   N A M E>
═══════════════════════════════════════════════════ <emoji>
```

#### State Validation Pattern

Commands that require colony state follow this validation pattern:

```markdown
### Step 1: Read and Validate State

Read `.aether/data/COLONY_STATE.json`.

**Auto-upgrade old state:**
If `version` field is missing, "1.0", or "2.0":
1. Preserve: `goal`, `state`, `current_phase`, `plan.phases`
2. Write upgraded v3.0 state
3. Output: `State auto-upgraded to v3.0`
4. Continue with command

**Validate:** If `goal: null`:
```
No colony initialized. Run /ant:init "<goal>" first.
```
Stop here.
```

#### Worker Spawn Pattern

Build commands use parallel worker spawning:

```markdown
### Step N: Spawn Workers

**CRITICAL: Spawn ALL Wave 1 workers in a SINGLE message using multiple Task tool calls.**

Spawn Builder via Task tool with `subagent_type="aether-builder"`:
```
You are <ant_name>, a 🔨🐜 Builder Ant.

Task: <task description>

Return JSON:
{
  "ant_name": "<name>",
  "status": "completed|failed",
  "summary": "...",
  "files_created": [],
  "files_modified": [],
  "blockers": []
}
```

Wait for all workers to complete.
```

### F. Utility Function Reference

#### Core Utilities (aether-utils.sh)

| Function | Purpose | Usage |
|----------|---------|-------|
| `version-check` | Check for updates | `bash .aether/aether-utils.sh version-check` |
| `queen-init` | Initialize QUEEN.md | `bash .aether/aether-utils.sh queen-init` |
| `session-init` | Initialize session | `bash .aether/aether-utils.sh session-init <id> <goal>` |
| `session-update` | Update session | `bash .aether/aether-utils.sh session-update <last_cmd> <next_cmd> <context>` |
| `session-verify-fresh` | Check freshness | `bash .aether/aether-utils.sh session-verify-fresh --command <name>` |
| `session-clear` | Clear stale files | `bash .aether/aether-utils.sh session-clear --command <name>` |
| `validate-state` | Validate state file | `bash .aether/aether-utils.sh validate-state colony` |
| `load-state` | Load with locking | `bash .aether/aether-utils.sh load-state` |
| `unload-state` | Release lock | `bash .aether/aether-utils.sh unload-state` |
| `activity-log` | Log activity | `bash .aether/aether-utils.sh activity-log <type> <actor> <message>` |
| `autofix-checkpoint` | Git checkpoint | `bash .aether/aether-utils.sh autofix-checkpoint <name>` |
| `autofix-rollback` | Rollback checkpoint | `bash .aether/aether-utils.sh autofix-rollback <type> <ref>` |
| `context-update` | Update CONTEXT.md | `bash .aether/aether-utils.sh context-update <action> <data>` |
| `flag-create` | Create flag | `bash .aether/aether-utils.sh flag-create <type> <message>` |
| `flag-list` | List flags | `bash .aether/aether-utils.sh flag-list --type <type>` |
| `flag-check-blockers` | Check blockers | `bash .aether/aether-utils.sh flag-check-blockers` |
| `milestone-detect` | Detect milestone | `bash .aether/aether-utils.sh milestone-detect` |
| `swarm-display-init` | Init display | `bash .aether/aether-utils.sh swarm-display-init <id>` |
| `swarm-display-update` | Update display | `bash .aether/aether-utils.sh swarm-display-update <name> <caste> <status> <task> <parent> <tools> <time> <chamber> <progress>` |
| `swarm-display-render` | Render display | `bash .aether/aether-utils.sh swarm-display-render <id>` |
| `swarm-findings-init` | Init findings | `bash .aether/aether-utils.sh swarm-findings-init <id>` |
| `swarm-findings-add` | Add finding | `bash .aether/aether-utils.sh swarm-findings-add <id> <type> <confidence> <data>` |
| `swarm-solution-set` | Set solution | `bash .aether/aether-utils.sh swarm-solution-set <id> <solution>` |
| `swarm-cleanup` | Cleanup swarm | `bash .aether/aether-utils.sh swarm-cleanup <id> --archive` |
| `chamber-create` | Create chamber | `bash .aether/aether-utils.sh chamber-create <id>` |
| `chamber-verify` | Verify chamber | `bash .aether/aether-utils.sh chamber-verify <id>` |
| `registry-add` | Register repo | `bash .aether/aether-utils.sh registry-add <path> <version>` |

### G. JSON Output Formats

#### Worker Response Format

All workers return standardized JSON:

```json
{
  "ant_name": "builder-1",
  "status": "completed|failed|partial",
  "summary": "Implemented user authentication with JWT tokens",
  "files_created": [
    "src/auth.js",
    "src/auth.test.js",
    "src/middleware/auth.js"
  ],
  "files_modified": [
    "src/app.js",
    "src/routes/index.js"
  ],
  "blockers": [],
  "learnings": [
    "Express middleware pattern works well for auth"
  ],
  "time_seconds": 145,
  "tools_used": {
    "read": 12,
    "grep": 8,
    "edit": 5,
    "bash": 3
  }
}
```

#### Scout Research Format

Scouts return research findings in this format:

```json
{
  "findings": [
    {
      "area": "authentication",
      "discovery": "JWT tokens are used for session management",
      "source": "src/auth.js"
    }
  ],
  "gaps_remaining": [
    {
      "id": "gap_1",
      "description": "How are refresh tokens handled?"
    }
  ],
  "gaps_resolved": ["gap_0"],
  "overall_knowledge_confidence": 65
}
```

#### Route-Setter Plan Format

Route-setters return plans in this format:

```json
{
  "plan": {
    "phases": [
      {
        "id": 1,
        "name": "Setup Authentication",
        "description": "Implement user login and registration",
        "tasks": [
          {
            "id": "1.1",
            "goal": "Create user model with password hashing",
            "constraints": ["Use bcrypt", "Store only hashed passwords"],
            "hints": ["See src/models/user.js pattern"],
            "success_criteria": ["User can be created", "Password is hashed"],
            "depends_on": []
          }
        ],
        "success_criteria": ["Users can register", "Users can login"]
      }
    ]
  },
  "confidence": {
    "knowledge": 75,
    "requirements": 80,
    "risks": 70,
    "dependencies": 85,
    "effort": 60,
    "overall": 78
  },
  "delta_reasoning": "Added password hashing constraint based on security best practices",
  "unresolved_gaps": ["OAuth integration approach"]
}
```

### H. Error Code Reference

| Code | Name | Description | Used In |
|------|------|-------------|---------|
| E_SUCCESS | 0 | Operation completed successfully | All commands |
| E_GENERAL | 1 | General error | All commands |
| E_INVALID_ARGS | 2 | Invalid arguments | init, build, plan |
| E_FILE_NOT_FOUND | 3 | Required file not found | All stateful commands |
| E_STATE_INVALID | 4 | State file invalid | init, plan, build |
| E_NO_COLONY | 5 | No colony initialized | plan, build, continue |
| E_PHASE_NOT_FOUND | 6 | Phase doesn't exist | build |
| E_PHASE_COMPLETED | 7 | Phase already completed | build |
| E_BLOCKERS_EXIST | 8 | Blockers prevent advancement | continue |
| E_VERIFICATION_FAILED | 9 | Verification gate failed | continue |
| E_SPAWN_FAILED | 10 | Worker spawn failed | build, swarm |
| E_CHECKPOINT_FAILED | 11 | Git checkpoint failed | build |
| E_UTILITY_ERROR | 12 | Utility function error | All commands |
| E_NETWORK_ERROR | 13 | Network operation failed | oracle, swarm |
| E_TIMEOUT | 14 | Operation timed out | oracle, watch |
| E_PERMISSION_DENIED | 15 | Permission denied | entomb, seal |
| E_ALREADY_EXISTS | 16 | Resource already exists | init |
| E_NOT_IMPLEMENTED | 17 | Feature not implemented | Various |

### I. Event Types Reference

| Event Type | Description | Generated By |
|------------|-------------|--------------|
| colony_initialized | Colony created | init |
| state_upgraded | State migrated to new version | plan, build, continue |
| plan_generated | Plan created | plan |
| plan_accepted | Plan accepted by user | plan |
| build_started | Build began | build |
| build_completed | Build finished | build |
| build_failed | Build failed | build |
| phase_completed | Phase finished | continue |
| phase_advanced | Moved to next phase | continue |
| flag_created | Flag added | flag |
| flag_resolved | Flag cleared | flag-auto-resolve |
| constraint_added | Focus/redirect added | focus, redirect |
| feedback_added | Feedback provided | feedback |
| instinct_learned | New instinct created | build (learning extraction) |
| milestone_reached | Milestone achieved | continue, seal |
| colony_sealed | Colony archived | seal |
| colony_entombed | Colony entombed | entomb |
| session_paused | Session paused | pause-colony |
| session_resumed | Session resumed | resume-colony, resume |
| swarm_deployed | Swarm initiated | swarm |
| swarm_success | Swarm fixed issue | swarm |
| swarm_failed | Swarm fix failed | swarm |
| oracle_started | Research started | oracle |
| oracle_completed | Research finished | oracle |
| oracle_stopped | Research halted | oracle stop |
| error_occurred | Error recorded | Various |

### J. Configuration File Schemas

#### COLONY_STATE.json (v3.0)

```json
{
  "version": "3.0",
  "goal": "Build a REST API with authentication",
  "state": "READY",
  "current_phase": 2,
  "session_id": "session_1708000000_abc123",
  "initialized_at": "2026-02-16T10:00:00Z",
  "build_started_at": "2026-02-16T11:00:00Z",
  "milestone": "Open Chambers",
  "milestone_updated_at": "2026-02-16T10:30:00Z",
  "plan": {
    "generated_at": "2026-02-16T10:15:00Z",
    "confidence": 85,
    "phases": [
      {
        "id": 1,
        "name": "Setup Project",
        "description": "Initialize project structure",
        "status": "completed",
        "tasks": [...],
        "success_criteria": [...]
      }
    ]
  },
  "memory": {
    "phase_learnings": [...],
    "decisions": [...],
    "instincts": [...]
  },
  "errors": {
    "records": [],
    "flagged_patterns": []
  },
  "signals": [],
  "graveyards": [],
  "events": [
    "2026-02-16T10:00:00Z|colony_initialized|init|Colony initialized..."
  ]
}
```

#### constraints.json (v1.0)

```json
{
  "version": "1.0",
  "focus": [
    "authentication module",
    "error handling"
  ],
  "constraints": [
    {
      "type": "AVOID",
      "description": "using eval()",
      "priority": "high",
      "created_at": "2026-02-16T10:05:00Z"
    }
  ]
}
```

#### session.json

```json
{
  "session_id": "session_1708000000_abc123",
  "initialized_at": "2026-02-16T10:00:00Z",
  "last_command": "/ant:build 1",
  "next_recommended": "/ant:continue",
  "context_summary": "Building authentication system",
  "freshness": {
    "last_activity": "2026-02-16T11:30:00Z",
    "activity_count": 15
  }
}
```

### K. Command Quick Reference Card

```
┌─────────────────────────────────────────────────────────────────┐
│                    AETHER COMMAND QUICK CARD                     │
├─────────────────────────────────────────────────────────────────┤
│ LIFECYCLE                                                        │
│   init "goal"        Initialize colony with goal                │
│   plan               Generate project plan                       │
│   build N            Execute phase N                             │
│   continue           Verify and advance phase                    │
│   seal               Archive completed colony                    │
│   entomb             Archive to chambers                         │
│   lay-eggs "goal"    Start new colony from existing             │
├─────────────────────────────────────────────────────────────────┤
│ PHEROMONES                                                       │
│   focus "area"       Guide colony attention                      │
│   redirect "avoid"   Hard constraint                             │
│   feedback "note"    Gentle adjustment                           │
├─────────────────────────────────────────────────────────────────┤
│ STATUS                                                           │
│   status             Colony dashboard                            │
│   phase [N]          Phase details                               │
│   flags              List active flags                           │
│   flag --type T "msg" Create flag                               │
│   history            Event history                               │
│   help [cmd]         Command reference                           │
├─────────────────────────────────────────────────────────────────┤
│ SESSION                                                          │
│   watch              Live visibility                             │
│   pause-colony       Save and handoff                            │
│   resume-colony      Restore from pause                          │
│   resume             Claude-specific resume                      │
│   update             Update from hub                             │
├─────────────────────────────────────────────────────────────────┤
│ ADVANCED                                                         │
│   swarm "problem"    Parallel bug investigation                 │
│   chaos              Resilience testing                          │
│   oracle [topic]     Deep research                               │
│   colonize           Territory survey                            │
│   archaeology        Git history analysis                        │
│   organize           Codebase hygiene                            │
│   verify-castes      System status                               │
└─────────────────────────────────────────────────────────────────┘
```

---

## Additional Detailed Command Analyses

### Deep Dive: Build Command Wave System

The build command's wave system represents one of the most sophisticated aspects of the Aether command architecture. This system enables parallel execution while respecting task dependencies, maximizing throughput without violating ordering constraints.

**Wave Construction Algorithm:**

1. **Dependency Graph Analysis**: The command analyzes all tasks in the phase to build a dependency graph. Each task has a `depends_on` array listing task IDs that must complete before it can start.

2. **Topological Sort**: The algorithm performs a topological sort on the dependency graph, assigning each task to a wave based on its depth in the dependency chain.

3. **Wave Assignment**:
   - Wave 1: Tasks with no dependencies (depth 0)
   - Wave 2: Tasks that depend only on Wave 1 (depth 1)
   - Wave N: Tasks that depend on tasks in waves 1 through N-1

4. **Parallel Execution**: All tasks in a wave are spawned simultaneously using multiple Task tool calls in a single message. This maximizes parallelism while respecting the dependency structure.

5. **Sequential Waves**: The command waits for all tasks in the current wave to complete before starting the next wave, ensuring that dependencies are satisfied.

**Example Wave Structure:**

```
Phase: Implement Authentication

Tasks:
  1.1 Create user model (no deps)
  1.2 Setup database schema (no deps)
  1.3 Install auth libraries (no deps)
  2.1 Implement registration (depends on 1.1, 1.2)
  2.2 Implement login (depends on 1.1, 1.3)
  3.1 Create auth middleware (depends on 2.2)
  3.2 Write tests (depends on 2.1, 2.2)

Wave 1 (Parallel): [1.1, 1.2, 1.3]
Wave 2 (Parallel): [2.1, 2.2]
Wave 3 (Parallel): [3.1, 3.2]
```

**Benefits of Wave System:**

- **Maximum Parallelism**: Independent tasks run simultaneously
- **Dependency Safety**: Ordering constraints are always respected
- **Resource Efficiency**: No idle waiting once dependencies are satisfied
- **Scalability**: Handles complex dependency graphs efficiently
- **Observability**: Clear visibility into execution order and progress

### Deep Dive: Session Freshness Detection

Session freshness detection is a critical reliability feature that prevents stale session files from silently breaking workflows. This system uses timestamp-based verification to detect when files may be outdated.

**Freshness Detection Algorithm:**

1. **Session Start Capture**: When a command begins, it captures the current timestamp: `COMMAND_START=$(date +%s)`

2. **File Timestamp Check**: The system checks the modification time of relevant files (COLONY_STATE.json, session files, etc.)

3. **Threshold Comparison**: File timestamps are compared against the session start time with a configurable threshold (default 5 seconds). Files modified before the threshold are considered stale.

4. **Stale File Handling**: Based on the command type:
   - **Protected commands** (init, seal, entomb): Warn user but never auto-clear
   - **Auto-clear commands** (swarm, oracle, watch): Automatically remove stale files
   - **Prompt commands**: Ask user whether to clear or preserve

5. **Verification**: After clearing or proceeding, the system verifies that files are now fresh.

**Freshness Status Categories:**

| Status | Definition | Action |
|--------|------------|--------|
| Fresh | File modified within threshold | Proceed normally |
| Stale | File modified before threshold | Clear or warn |
| Missing | File doesn't exist | Initialize new |
| Invalid | File exists but unreadable | Error |

**Protected vs. Auto-Clear Commands:**

Protected commands never auto-clear because they manage precious data:
- `init`: COLONY_STATE.json contains user goal and progress
- `seal`: Archives are permanent records
- `entomb`: Chambers preserve knowledge

Auto-clear commands remove stale files automatically:
- `swarm`: Findings are temporary investigation results
- `oracle`: Research progress can be restarted
- `watch`: Display state is ephemeral
- `colonize`: Survey data can be refreshed

### Deep Dive: Pheromone Signal System

The pheromone signal system provides a biologically-inspired mechanism for user guidance. Three signal types enable different levels of direction, from gentle suggestions to hard constraints.

**FOCUS Signal (Normal Priority):**

Focus signals guide colony attention toward specific areas without enforcing constraints. They are stored in the `focus` array in constraints.json.

- **Purpose**: Suggest areas for attention
- **Storage**: `constraints.json` `focus[]`
- **Limit**: Maximum 5 active focus areas
- **Lifetime**: Persists until explicitly removed
- **Worker Behavior**: Workers prioritize focus areas but can work elsewhere

**REDIRECT Signal (High Priority):**

Redirect signals establish hard constraints that workers must respect. They are stored in the `constraints` array with type "AVOID".

- **Purpose**: Prohibit specific patterns or approaches
- **Storage**: `constraints.json` `constraints[]`
- **Limit**: Maximum 10 active redirect constraints
- **Lifetime**: Persists until explicitly removed
- **Worker Behavior**: Workers must not violate redirects

**FEEDBACK Signal (Low Priority):**

Feedback signals provide gentle adjustments based on observations. They create both immediate signals and persistent instincts.

- **Purpose**: Adjust behavior based on observations
- **Storage**: `COLONY_STATE.json` `signals[]` and `memory.instincts[]`
- **Limit**: No explicit limit
- **Lifetime**: Signals persist; instincts evolve
- **Worker Behavior**: Instincts influence decisions with confidence weighting

**Signal Integration in Worker Prompts:**

Workers receive pheromone signals in their task prompts:

```
--- PHEROMONE SIGNALS ---

FOCUS (pay attention to):
  - authentication module
  - error handling

REDIRECT (must avoid):
  - using eval()
  - modifying the database schema

INSTINCTS (lessons learned):
  - [0.8] Always validate input before processing
  - [0.7] Prefer async/await over callbacks
```

**Confidence Evolution:**

Instincts created from feedback have initial confidence of 0.7. This confidence evolves based on outcomes:
- **Success**: Confidence increases (up to 1.0)
- **Failure**: Confidence decreases (down to 0.0)
- **No application**: Confidence slowly decays

Instincts with confidence below 0.3 are automatically archived, preventing the system from accumulating low-value guidance.

### Deep Dive: The Oracle RALF Loop

The Oracle command implements the RALF (Recursive Autonomous Learning Flow) pattern, a sophisticated iterative research mechanism designed to achieve deep understanding through successive refinement cycles.

**RALF Loop Architecture:**

1. **Research Phase**: The Oracle explores the topic using available tools (Glob, Grep, Read for codebase research; WebSearch, WebFetch for external research). It identifies key findings and knowledge gaps.

2. **Analysis Phase**: Findings are synthesized and assessed for completeness. The Oracle determines what is known, what remains unknown, and which gaps are most critical to address.

3. **Learning Phase**: Based on identified gaps, the Oracle formulates specific research questions for the next iteration. These questions target the most impactful unknowns.

4. **Feedback Phase**: Confidence is assessed across multiple dimensions. If confidence is below target and iterations remain, the loop continues with the new questions.

**Confidence Dimensions:**

The Oracle assesses confidence across five dimensions, similar to the plan command:
- **Knowledge**: Understanding of the topic's scope and key concepts
- **Requirements**: Clarity of what needs to be learned
- **Risks**: Awareness of potential pitfalls and edge cases
- **Dependencies**: Understanding of relationships to other topics
- **Effort**: Ability to estimate research depth needed

**Iteration Strategies:**

- **Early iterations** (1-5): Broad exploration to map the territory
- **Middle iterations** (6-15): Focused investigation of key gaps
- **Late iterations** (16+): Deep dives into remaining unknowns

**Auto-Termination Conditions:**

The RALF loop terminates when:
- Confidence target is reached (default 95%)
- Maximum iterations completed (configurable: 5, 15, 30, 50)
- Progress stalls (< 5% improvement for 3 consecutive iterations)
- User manually stops (`/ant:oracle stop`)

**Research Artifacts:**

The Oracle produces several artifacts:
- `progress.md`: Iteration-by-iteration log of findings and confidence
- `research.json`: Structured research configuration and metadata
- `discoveries/`: Directory containing detailed findings by topic
- `archive/`: Historical research sessions for reference

### Deep Dive: Swarm Cross-Comparison Algorithm

The swarm command's solution ranking algorithm uses multi-scout consensus to identify the most reliable fix for stubborn bugs.

**Cross-Comparison Process:**

1. **Finding Collection**: As each scout completes, their findings are collected with confidence scores.

2. **Agreement Detection**: The algorithm identifies areas where scouts agree. High agreement across multiple scouts indicates high-confidence findings.

3. **Disagreement Resolution**: Where scouts disagree, the algorithm weights findings by:
   - Scout confidence score
   - Evidence quality (concrete vs. speculative)
   - Scout specialization relevance to the issue

4. **Solution Ranking**: Potential solutions are ranked by:
   - Number of supporting scouts
   - Average confidence of supporting scouts
   - Evidence strength
   - Implementation feasibility

5. **Selection**: The highest-ranked solution is selected for implementation.

**Scout Specializations:**

| Scout | Strengths | Best For |
|-------|-----------|----------|
| Archaeologist | Historical context | Regression bugs, recent changes |
| Pattern Hunter | Code patterns | Inconsistent implementations |
| Error Analyst | Root cause analysis | Complex error chains |
| Web Researcher | External knowledge | Known issues, library bugs |

**Verification Strategy:**

After implementing a solution, the swarm verifies:
- Build passes
- Tests pass
- Original issue is resolved
- No regressions introduced

If verification fails, the swarm rolls back and attempts the next-ranked solution.

### Deep Dive: Continue Command Iron Laws

The continue command's six verification gates (Iron Laws) ensure quality before phase advancement.

**Gate 1: Build Verification**
- Checks that all tasks in the phase are marked complete
- Verifies no workers reported failure
- Validates expected files were created/modified

**Gate 2: Type Check**
- Runs TypeScript compiler or equivalent
- Checks for type errors, undefined references
- Validates interface implementations

**Gate 3: Lint**
- Runs ESLint, Prettier, or project-specific linters
- Checks code style consistency
- Identifies anti-patterns and problematic code

**Gate 4: Test**
- Runs test suite (unit, integration, e2e)
- Validates test coverage meets thresholds
- Checks for TDD evidence (tests before implementation)

**Gate 5: Security**
- Scans for hardcoded secrets
- Checks for injection vulnerabilities
- Validates input sanitization

**Gate 6: Diff**
- Shows user all changes that will be committed
- Requires explicit confirmation
- Provides final opportunity to catch issues

**Gate Enforcement:**

- All gates must pass for phase advancement
- Failed gates block continuation
- `--force` flag can skip gates (not recommended)
- Gate results are logged for audit trail

---

## Summary and Conclusions

The Aether command system represents a sophisticated approach to AI-assisted software development, combining biological metaphors with rigorous engineering practices. The 36 unique commands (71 total implementations across Claude Code and OpenCode) provide comprehensive coverage of the software development lifecycle.

**Key Architectural Strengths:**

1. **Dual-Platform Support**: Near-identical functionality across Claude Code and OpenCode maximizes accessibility

2. **Multi-Agent Coordination**: The caste system enables specialized workers to collaborate effectively

3. **Persistent Memory**: State management across sessions ensures continuity and learning

4. **User Guidance**: Pheromone signals provide intuitive mechanisms for steering behavior

5. **Quality Assurance**: Multiple verification gates ensure code quality

6. **Extensibility**: The command structure allows for easy addition of new capabilities

**Areas for Improvement:**

1. **Platform Unification**: OpenCode version lacks some features present in Claude Code
2. **Model Routing**: Per-caste model assignment needs verification
3. **Error Handling**: Standardize error codes across all commands
4. **Documentation**: Continue expanding command documentation

**Future Directions:**

- Enhanced learning mechanisms for instinct evolution
- Cross-colony knowledge sharing
- Automated test generation
- Integration with CI/CD pipelines
- Visual workflow designer

---

*Documentation generated: 2026-02-16*
*Total commands documented: 36 unique commands, 71 implementations*
*Sections: 13 major sections, 34 command references, 11 appendices*
*Final word count: 20,000+ words*
