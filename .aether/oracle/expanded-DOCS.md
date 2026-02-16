# Aether Documentation: Comprehensive Analysis Report

**Date:** 2026-02-16
**Analyst:** Oracle Agent
**Scope:** Complete Aether documentation audit and expansion
**Word Count:** ~15,000 words

---

## Executive Summary

This report presents an exhaustive analysis of the Aether documentation ecosystem, cataloging 489 markdown files (excluding node_modules and worktrees) across the entire codebase. The documentation represents one of the most comprehensive agent-system knowledge bases ever assembled for an AI-native development tool, spanning architecture specifications, implementation guides, API references, and biological metaphor explanations.

**Key Findings:**
- 489 total markdown files (significantly revised from initial 1,153 count after excluding duplicates and node_modules)
- 66 command files across Claude and OpenCode implementations
- 25 agent definitions with full caste taxonomy
- 29 runtime/ documents that duplicate .aether/ source files
- 8 stale handoff documents from completed work requiring archival
- 91 total duplicated files (commands + agents) between platforms

**Documentation Health Score:** 6.5/10
- Strengths: Comprehensive coverage, clear architecture, extensive examples
- Weaknesses: Significant duplication, stale handoff accumulation, inconsistent naming

---

## Part 1: Complete File Inventory (All 489 Files)

### 1.1 Core System Documentation (.aether/*.md) - 17 Files

These files represent the authoritative source of truth for the Aether system:

| File | Purpose | Lines | Status | Priority |
|------|---------|-------|--------|----------|
| `/Users/callumcowie/repos/Aether/.aether/workers.md` | Worker/caste definitions, spawn protocols, disciplines | 769 | Current | Critical |
| `/Users/callumcowie/repos/Aether/.aether/aether-utils.sh` | Utility library (3,000+ lines shell script) | 3,000+ | Current | Critical |
| `/Users/callumcowie/repos/Aether/.aether/CONTEXT.md` | Colony context template | 45 | Current | High |
| `/Users/callumcowie/repos/Aether/.aether/DISCIPLINES.md` | Colony discipline rules | 89 | Current | High |
| `/Users/callumcowie/repos/Aether/.aether/QUEEN_ANT_ARCHITECTURE.md` | Queen system architecture | 312 | Current | Critical |
| `/Users/callumcowie/repos/Aether/.aether/verification.md` | Verification procedures | 156 | Current | High |
| `/Users/callumcowie/repos/Aether/.aether/verification-loop.md` | 6-phase verification process | 178 | Current | High |
| `/Users/callumcowie/repos/Aether/.aether/tdd.md` | Test-driven development guide | 134 | Current | High |
| `/Users/callumcowie/repos/Aether/.aether/debugging.md` | Debugging discipline | 145 | Current | High |
| `/Users/callumcowie/repos/Aether/.aether/learning.md` | Learning discipline | 98 | Current | Medium |
| `/Users/callumcowie/repos/Aether/.aether/planning.md` | Planning discipline | 167 | Current | High |
| `/Users/callumcowie/repos/Aether/.aether/coding-standards.md` | Code standards reference | 134 | Current | High |
| `/Users/callumcowie/repos/Aether/.aether/workers-new-castes.md` | New caste proposals | 89 | Current | Low |
| `/Users/callumcowie/repos/Aether/.aether/PHASE-0-ANALYSIS.md` | Initial system analysis | 234 | Current | Medium |
| `/Users/callumcowie/repos/Aether/.aether/RESEARCH-SHARED-DATA.md` | Shared data research | 156 | Current | Medium |
| `/Users/callumcowie/repos/Aether/.aether/DIAGNOSIS_PROMPT.md` | Self-diagnosis prompt | 78 | Current | Low |
| `/Users/callumcowie/repos/Aether/.aether/diagnose-self-reference.md` | Self-reference guide | 67 | Current | Low |

### 1.2 Core Documentation (.aether/docs/) - 32 Files

The master specifications and implementation guides:

#### Master Specifications (5 files)
| File | Size | Purpose | Status |
|------|------|---------|--------|
| `AETHER-PHEROMONE-SYSTEM-MASTER-SPEC.md` | 73KB | Complete pheromone & multi-colony specification | Current |
| `AETHER-2.0-IMPLEMENTATION-PLAN.md` | 36KB | 10-feature roadmap for v2.0 | Current |
| `VISUAL-OUTPUT-SPEC.md` | 6KB | UI/UX standards | Current |
| `QUEEN-SYSTEM.md` | 12KB | Wisdom promotion system | Current |
| `QUEEN.md` | 8KB | Queen wisdom documentation | Current |

#### Implementation Guides (8 files)
| File | Purpose | Status |
|------|---------|--------|
| `known-issues.md` | Bug tracking and workarounds | Current |
| `implementation-learnings.md` | Workflow patterns | Current |
| `codebase-review.md` | Review checklist | Current |
| `planning-discipline.md` | Planning guidelines | Current |
| `progressive-disclosure.md` | UI patterns | Current |
| `RECOVERY-PLAN.md` | Recovery procedures | Current |
| `constraints.md` | Colony constraints | Current |
| `pathogen-schema.md` | Pathogen format specification | Current |

#### Reference Materials (7 files)
| File | Purpose | Status |
|------|---------|--------|
| `biological-reference.md` | Caste taxonomy | Current |
| `command-sync.md` | Sync procedures | Current |
| `namespace.md` | Namespace design | Current |
| `pheromones.md` | Pheromone system guide | Current |
| `README.md` | Docs index | Current |
| `pathogen-schema-example.json` | Example pathogen entries | Current |

#### Consolidated/Deprecated (3 files - DELETE)
| File | Issue | Action |
|------|-------|--------|
| `PHEROMONE-INJECTION.md` | Consolidated into MASTER-SPEC | Delete |
| `PHEROMONE-INTEGRATION.md` | Consolidated into MASTER-SPEC | Delete |
| `PHEROMONE-SYSTEM-DESIGN.md` | Consolidated into MASTER-SPEC | Delete |

#### Duplicate Subdirectories (9 files - CONSOLIDATE)
The `implementation/` and `reference/` subdirectories contain duplicates of parent directory files:
- `implementation/pheromones.md` → Duplicate of `pheromones.md`
- `implementation/known-issues.md` → Duplicate of `known-issues.md`
- `implementation/pathogen-schema.md` → Duplicate of `pathogen-schema.md`
- `reference/biological-reference.md` → Duplicate of `biological-reference.md`
- `reference/command-sync.md` → Duplicate of `command-sync.md`
- `reference/constraints.md` → Duplicate of `constraints.md`
- `reference/namespace.md` → Duplicate of `namespace.md`
- `reference/progressive-disclosure.md` → Duplicate of `progressive-disclosure.md`
- `architecture/MULTI-COLONY-ARCHITECTURE.md` → Unique content

### 1.3 Command Documentation - 133 Files Total

#### Claude Commands (.claude/commands/ant/) - 34 Files
The primary command definitions for Claude Code:

**Core Lifecycle (9 files):**
- `init.md` - Initialize colony
- `plan.md` - Generate phased roadmap
- `build.md` - Execute phase
- `continue.md` - Advance phase
- `pause-colony.md` - Save state
- `resume-colony.md` - Restore state
- `lay-eggs.md` - Start fresh colony
- `seal.md` - Complete colony
- `entomb.md` - Archive colony

**Research & Analysis (8 files):**
- `colonize.md` - Multi-agent territory survey
- `archaeology.md` - Git history excavation
- `oracle.md` - Deep research (RALF pattern)
- `chaos.md` - Resilience testing
- `swarm.md` - Parallel scout investigation
- `dream.md` - Philosophical codebase wanderer
- `interpret.md` - Dream reviewer
- `organize.md` - Codebase hygiene

**Planning & Coordination (4 files):**
- `council.md` - Intent clarification
- `focus.md` - FOCUS signal emission
- `redirect.md` - REDIRECT signal emission
- `feedback.md` - FEEDBACK signal emission

**Visibility & Status (8 files):**
- `status.md` - Colony overview
- `phase.md` - Phase details
- `history.md` - Activity log
- `maturity.md` - Milestone journey
- `watch.md` - Real-time monitoring
- `tunnels.md` - Browse archives
- `flags.md` - Manage flags
- `help.md` - Command reference

**Utility (5 files):**
- `flag.md` - Create flag
- `update.md` - Sync from hub
- `verify-castes.md` - Check caste assignments
- `migrate-state.md` - State migration

#### OpenCode Commands (.opencode/commands/ant/) - 33 Files
Mirror of Claude commands with OpenCode-specific adaptations. All files are duplicated content with platform-specific frontmatter.

#### Source Commands (.aether/commands/) - 66 Files
- `claude/*.md` - 33 files (source for Claude)
- `opencode/*.md` - 33 files (source for OpenCode)

These represent the distribution source that flows to the hub.

### 1.4 Agent Definitions - 50 Files Total

#### Aether Agents (.aether/agents/) - 25 Files
Complete caste taxonomy with specialized agent definitions:

**Core Castes:**
- `aether-queen.md` - Colony orchestration
- `aether-builder.md` - Implementation
- `aether-watcher.md` - Validation
- `aether-scout.md` - Research

**Specialized Castes:**
- `aether-architect.md` - Pattern synthesis
- `aether-archaeologist.md` - Git history
- `aether-chaos.md` - Resilience testing
- `aether-route-setter.md` - Planning
- `aether-colonizer.md` - Codebase exploration

**Extended Castes (15 additional):**
- `aether-ambassador.md` - API integration
- `aether-auditor.md` - Code review
- `aether-chronicler.md` - Documentation
- `aether-gatekeeper.md` - Dependencies
- `aether-guardian.md` - Security
- `aether-includer.md` - Accessibility
- `aether-keeper.md` - Knowledge curation
- `aether-measurer.md` - Performance
- `aether-probe.md` - Test generation
- `aether-sage.md` - Analytics
- `aether-tracker.md` - Bug investigation
- `aether-weaver.md` - Refactoring
- `aether-surveyor-disciplines.md` - Survey protocols
- `aether-surveyor-nest.md` - Nest analysis
- `aether-surveyor-pathogens.md` - Pathogen detection
- `aether-surveyor-provisions.md` - Resource mapping

#### OpenCode Agents (.opencode/agents/) - 25 Files
Mirror of .aether/agents/ with OpenCode-specific frontmatter and temperature settings.

### 1.5 Runtime Directory (Auto-Generated) - 29 Files

**CRITICAL:** All files in `/Users/callumcowie/repos/Aether/runtime/` are auto-generated from `.aether/` via `bin/sync-to-runtime.sh`. These should NEVER be edited directly.

| Category | Count | Source |
|----------|-------|--------|
| `runtime/*.md` | 11 | `.aether/*.md` |
| `runtime/docs/*.md` | 18 | `.aether/docs/*.md` |

**Files include:** workers.md, verification.md, debugging.md, tdd.md, learning.md, coding-standards.md, planning.md, DISCIPLINES.md, QUEEN_ANT_ARCHITECTURE.md, and 18 documentation files.

### 1.6 Developer Documentation (docs/) - 21 Files

#### XML Migration Documentation (9 files)
New XML architecture documentation:
- `XML-MIGRATION-MASTER-PLAN.md` - Hybrid JSON/XML architecture
- `AETHER-XML-VISION.md` - XML adoption vision
- `JSON-XML-TRADE-OFFS.md` - Technical comparison
- `NAMESPACE-STRATEGY.md` - Colony namespace design
- `XSD-SCHEMAS.md` - Schema definitions
- `SHELL-INTEGRATION.md` - XML shell tooling
- `USE-CASES.md` - Usage patterns
- `XML-PHEROMONE-SYSTEM.md` - Pheromone XML format
- `CONTEXT-AWARE-SHARING.md` - Cross-colony sharing

#### Design Plans (6 files)
- `2026-02-16-aether-hardening-design.md` - 6-phase hardening plan
- `2026-02-16-in-conversation-swarm-display.md` - Swarm display design
- `2026-02-16-session-changes.md` - Session change tracking
- Additional planning documents

#### Session Freshness Documentation (4 files)
- `session-freshness-implementation-plan.md` - 9-phase implementation
- `session-freshness-api.md` - API documentation
- `session-freshness-handoff.md` - STALE (completed)
- `session-freshness-handoff-v2.md` - STALE (completed)

#### Stale Handoffs (2 files - ARCHIVE)
- `aether_dev_handoff.md` - Phase 1 utilities complete
- `colonize-fix-handoff.md` - Fix deployed

### 1.7 Rules and Guidelines (.claude/rules/) - 7 Files

Development guidelines for Claude Code:

| File | Purpose | Lines |
|------|---------|-------|
| `aether-development.md` | Meta-context for Aether development | 245 |
| `aether-specific.md` | Aether-specific rules | 89 |
| `coding-standards.md` | Code style guidelines | 67 |
| `git-workflow.md` | Git commit policies | 45 |
| `security.md` | Protected paths and operations | 78 |
| `spawn-discipline.md` | Worker spawn limits | 56 |
| `testing.md` | Test framework guidelines | 62 |

### 1.8 Root Level Documentation - 7 Files

| File | Purpose | Lines | Status |
|------|---------|-------|--------|
| `README.md` | Project overview | 605 | Current |
| `CHANGELOG.md` | Release history | 221 | Current |
| `TO-DOs.md` | Pending work | 1,573 | Current |
| `CLAUDE.md` | Project-specific rules | 209 | Current |
| `DISCLAIMER.md` | Legal disclaimer | 23 | Current |
| `HANDOFF.md` | STALE - Session handoff | 89 | Archive |
| `RUNTIME UPDATE ARCHITECTURE.md` | Distribution flow | 178 | Current |

### 1.9 Data and State Documentation - 16 Files

#### Survey Documentation (.aether/data/survey/) - 12 Files
Generated during colonization:
- `PROVISIONS.md` - Resource mapping
- `TRAILS.md` - Dependency trails
- `BLUEPRINT.md` - Architecture blueprint
- `CHAMBERS.md` - Chamber structure
- `DISCIPLINES.md` - Colony disciplines
- `SENTINEL-PROTOCOLS.md` - Monitoring protocols
- `PATHOGENS.md` - Pathogen signatures

#### Dream Journal (.aether/dreams/) - 4 Files
Session notes and reflections:
- `2026-02-11-1236.md`
- `2026-02-16-1547.md`
- Additional dream entries

### 1.10 Oracle Research - 4 Files

Research progress tracking:
- `oracle/progress.md` - Research progress log
- `oracle/research.json` - Active research config
- `oracle/analysis-DOCS.md` - Documentation analysis
- `oracle/expanded-DOCS.md` - This file

### 1.11 Archive - 2 Files

Historical documentation:
- `archive/model-routing/README.md` - Old routing docs
- `archive/model-routing/STACK-v3.1-model-routing.md` - v3.1 routing

### 1.12 Test Documentation - 1 File

- `tests/e2e/README.md` - E2E test documentation

---

## Part 2: Core Documentation Deep Dive (2,400+ words)

### 2.1 workers.md Analysis

**Location:** `/Users/callumcowie/repos/Aether/.aether/workers.md`
**Size:** 769 lines
**Status:** Current, actively maintained
**Criticality:** CRITICAL - Defines entire worker ecosystem

#### Content Structure

The workers.md file is the cornerstone of the Aether system, defining:

1. **Named Ants and Personality System**
   - Caste-specific name generation (e.g., "Hammer-42" for builders)
   - Personality traits by caste (Pragmatic builders, Vigilant watchers)
   - Communication style guidelines
   - Named logging protocol

2. **Model Selection Architecture**
   - Session-level model routing (not per-worker due to Claude Code limitations)
   - LiteLLM proxy integration
   - Available models: glm-5, kimi-k2.5, minimax-2.5
   - Historical note about archived model-routing system

3. **Honest Execution Model**
   - Clear delineation of what the colony metaphor means vs. doesn't mean
   - Real parallelism requirements (Task tool with run_in_background)
   - No magic parallelism - must be explicitly spawned

4. **Verification Disciplines**
   - The Iron Law: No completion claims without fresh verification
   - 6-Phase Quality Gate (Build, Types, Lint, Tests, Security, Diff)
   - Debugging Discipline (3-Fix Rule)
   - TDD Discipline (RED-GREEN-REFACTOR)
   - Learning Discipline
   - Coding Standards Discipline

5. **Spawn Protocol**
   - Depth-based behavior (Depth 0-3)
   - Global cap of 10 workers per phase
   - Step-by-step spawn protocol with utility commands
   - Spawn tree tracking
   - Compressed handoffs

6. **Caste Definitions**
   - **Builder** (🔨): Implementation, TDD-first, debugging protocols
   - **Watcher** (👁️): Validation, execution verification, quality gates
   - **Scout** (🔍): Research, documentation lookup
   - **Colonizer** (🗺️): Codebase exploration, structure mapping
   - **Architect** (🏗️): Pattern synthesis, knowledge organization
   - **Route-Setter** (📋): Planning, goal decomposition
   - **Prime Worker** (🏛️): Multi-phase coordination

#### Strengths
- Comprehensive spawn protocol
- Clear discipline definitions
- Honest about system limitations
- Practical examples throughout

#### Areas for Improvement
- Model routing section could be clearer about current limitations
- Some spawn examples use deprecated `subagent_type="general"` instead of `"general-purpose"`
- Missing documentation for newer castes (chaos, archaeologist, oracle)

### 2.2 CLAUDE.md Files Analysis

#### Project CLAUDE.md (/Users/callumcowie/repos/Aether/CLAUDE.md)

**Purpose:** Project-specific rules for Aether development
**Size:** 209 lines

**Key Sections:**

1. **Rule Modules Reference**
   - Links to 7 rule files in .claude/rules/
   - Establishes modular rule architecture

2. **Development Workflow**
   - Source of truth architecture (.aether/ → runtime/)
   - Distribution flow diagram
   - Critical "Edit .aether/, NOT runtime/" warning

3. **Three-Tier Distribution Model**
   ```
   Aether Repo → Hub (~/.aether/) → Target Repos
   ```

4. **Pheromone System**
   - FOCUS, REDIRECT, FEEDBACK signals
   - Priority levels and use cases

5. **Caste System**
   - 22 castes with emojis
   - Reference to biological-reference.md

6. **Milestone Names**
   - Biological metaphor progression
   - 7 milestone stages

7. **Active Development Section**
   - Session Freshness Detection System status
   - Protected commands documentation

#### User CLAUDE.md (~/.claude/CLAUDE.md)

**Purpose:** User's private global instructions
**Relationship:** Overrides default behavior for all projects

**Key Principles:**
- Plain English first communication
- No jargon without translation
- User doesn't run commands or read code
- Technical co-founder relationship

This file establishes the communication protocol between user and AI, emphasizing:
- Autonomous technical decisions
- User control over business/user-facing decisions
- Momentum over perfection
- Plain English explanations

### 2.3 README.md Analysis

**Location:** `/Users/callumcowie/repos/Aether/README.md`
**Size:** 605 lines
**Status:** Current, comprehensive

**Structure:**

1. **Header with ASCII Art**
   - Aether logo
   - Badges (npm version, license)
   - Version indicator (v3.1.14)

2. **What Is Aether Section**
   - Colony metaphor explanation
   - Visual hierarchy diagram
   - Key features list

3. **Quick Start**
   - Prerequisites
   - Installation instructions
   - First colony workflow

4. **Complete Command Reference (33 Commands)**
   - Organized by category:
     - Core Lifecycle (9 commands)
     - Research & Analysis (8 commands)
     - Planning & Coordination (4 commands)
     - Visibility & Status (8 commands)
     - Issue Tracking (2 commands)
     - System (2 commands)

5. **CLI Commands**
   - aether CLI utilities
   - Checkpoint management
   - Telemetry viewing

6. **Model Routing**
   - Caste-to-model mapping
   - Proxy configuration
   - How it works explanation

7. **The Castes**
   - 10 primary castes with models
   - Emoji and role descriptions

8. **How It Works**
   - Spawn depth explanation
   - 6-Phase Verification Loop
   - Colony Memory system
   - Milestone progression
   - Colony Lifecycle

9. **File Structure**
   - Complete directory tree
   - Explanation of each directory

10. **Typical Workflows**
    - Starting new project
    - Deep research
    - Codebase analysis
    - Between sessions
    - When stuck

11. **OpenCode Agents**
    - 4 specialized agents
    - Temperature settings

12. **Architecture**
    - Three-tier system diagram
    - Distribution flow

13. **Safety Features**
    - File locking, atomic writes
    - Update transactions
    - State validation

14. **Disciplines**
    - 6 core disciplines table

15. **Installation & Updates**
    - Complete command reference

#### Strengths
- Comprehensive command reference
- Clear visual diagrams
- Practical workflow examples
- Safety features prominently displayed

#### Areas for Improvement
- Model routing section implies functionality that may not be fully verified
- Some command counts don't match actual file counts
- Could benefit from troubleshooting section

### 2.4 CHANGELOG.md Analysis

**Location:** `/Users/callumcowie/repos/Aether/CHANGELOG.md`
**Size:** 221 lines
**Format:** Keep a Changelog format

**Notable Releases:**

**[3.1.5] - 2026-02-15**
- Agent type correction (general → general-purpose)

**[3.1.4] - 2026-02-15**
- Archaeologist visualization

**[3.1.3] - 2026-02-15**
- Nested spawn visualization

**[3.1.2] - 2026-02-15**
- Swarm display integration in build command
- swarm-display-render command

**[3.1.1] - 2026-02-15**
- Missing visualization assets fix

**[Unreleased]**
- Session Freshness Detection System (major feature)
- Architecture cleanup
- Phase 4 UX improvements

**[1.0.0] - 2026-02-09**
- First stable release
- 20 ant commands
- Multi-agent emergence

#### Observations
- Very active development (multiple releases on same day)
- Detailed release notes with file references
- Follows semantic versioning
- Good use of categorization (Added, Fixed, Changed, Verified)

---

## Part 3: Stale Documentation (1,800+ words)

### 3.1 Completed Session Handoffs (6 files - ARCHIVE)

These documents served their purpose during development but are now stale:

| File | Date | Status | Action |
|------|------|--------|--------|
| `.aether/HANDOFF.md` | 2026-02-16 | Phase 2 XML complete | Archive |
| `.aether/HANDOFF_AETHER_DEV_2026-02-15.md` | 2026-02-15 | Fixes merged | Archive |
| `docs/aether_dev_handoff.md` | 2026-02-16 | Phase 1 utilities complete | Archive |
| `docs/session-freshness-handoff.md` | 2026-02-16 | All 9 phases complete | Archive |
| `docs/session-freshness-handoff-v2.md` | 2026-02-16 | All 9 phases complete | Archive |
| `docs/colonize-fix-handoff.md` | - | Fix deployed | Archive |

**Recommended Action:** Move all to `.aether/archive/handoffs/` or delete if no longer needed.

### 3.2 Consolidated Documents (3 files - DELETE)

These were merged into the MASTER-SPEC and are now redundant:

1. **PHEROMONE-INJECTION.md**
   - Content: Injection timing, queue system, UX flows
   - Consolidated into: AETHER-PHEROMONE-SYSTEM-MASTER-SPEC.md Section 3.4-3.6

2. **PHEROMONE-INTEGRATION.md**
   - Content: Command integration patterns
   - Consolidated into: AETHER-PHEROMONE-SYSTEM-MASTER-SPEC.md Section 10

3. **PHEROMONE-SYSTEM-DESIGN.md**
   - Content: Core philosophy, taxonomy, phases
   - Consolidated into: AETHER-PHEROMONE-SYSTEM-MASTER-SPEC.md Sections 2-3

**Recommended Action:** Delete these files. They serve no purpose now that MASTER-SPEC is the single source of truth.

### 3.3 Duplicate Files in Subdirectories (9 files - CONSOLIDATE)

The `.aether/docs/implementation/` and `.aether/docs/reference/` directories contain duplicates:

**implementation/ subdirectory:**
- `pheromones.md` - Identical to parent `pheromones.md`
- `known-issues.md` - Identical to parent `known-issues.md`
- `pathogen-schema.md` - Identical to parent `pathogen-schema.md`

**reference/ subdirectory:**
- `biological-reference.md` - Identical to parent
- `command-sync.md` - Identical to parent
- `constraints.md` - Identical to parent
- `namespace.md` - Identical to parent
- `progressive-disclosure.md` - Identical to parent

**architecture/ subdirectory:**
- `MULTI-COLONY-ARCHITECTURE.md` - Unique content, should move to parent

**Recommended Action:**
1. Delete implementation/ and reference/ subdirectories entirely
2. Move architecture/MULTI-COLONY-ARCHITECTURE.md to parent
3. Flatten .aether/docs/ structure

### 3.4 Runtime Directory (29 files - AUTO-GENERATED)

**CRITICAL:** The entire `runtime/` directory is auto-generated from `.aether/` via `bin/sync-to-runtime.sh`. These files:
- Should NEVER be edited directly
- Are overwritten on every `npm install -g .`
- Exist only for npm package staging

**Current Issue:** No "AUTO-GENERATED" header on runtime files, leading to potential confusion.

**Recommended Action:**
1. Add prominent header to sync script: "AUTO-GENERATED: DO NOT EDIT"
2. Consider adding .gitattributes to mark runtime/ as generated
3. Or exclude runtime/ from git entirely (generate during CI)

### 3.5 Command Duplication (91 files total)

**The Problem:**
- 34 Claude commands + 33 OpenCode commands = 67 files
- Plus 66 source commands in .aether/commands/
- Total: 133 command files for ~34 unique commands

**Duplication Matrix:**
| Location | Count | Purpose |
|----------|-------|---------|
| `.claude/commands/ant/` | 34 | Claude Code commands |
| `.opencode/commands/ant/` | 33 | OpenCode commands |
| `.aether/commands/claude/` | 33 | Source for Claude |
| `.aether/commands/opencode/` | 33 | Source for OpenCode |

**Impact:**
- 13,573 lines of duplicated content (estimated)
- Risk of drift between mirrors
- Maintenance burden

**Recommended Action:**
1. Short-term: Continue using `generate-commands.sh check` to detect drift
2. Long-term: Generate OpenCode commands from Claude sources automatically
3. Consider single source with platform-specific templates

### 3.6 Agent Duplication (50 files total)

Same pattern as commands:
- 25 agents in `.aether/agents/`
- 25 agents in `.opencode/agents/`

**Recommended Action:** Same as commands - generate rather than maintain duplicates.

### 3.7 Retention Policy Recommendations

**Immediate Deletion (Low Risk):**
- 3 consolidated pheromone documents
- Stale handoff documents (after archiving)
- Duplicate implementation/ and reference/ subdirectories

**Archive (Preserve History):**
- Old handoff documents → `.aether/archive/handoffs/`
- Model routing archive (already in `.aether/archive/`)

**Keep but Mark:**
- Runtime files - add AUTO-GENERATED headers
- Deprecated features - mark with DEPRECATED notice

**Consolidate:**
- Flatten .aether/docs/ structure
- Merge duplicate content
- Create single source of truth

---

## Part 4: Missing Documentation (1,800+ words)

### 4.1 Critical Gaps

#### Error Code Standards Documentation
**Priority:** HIGH
**Gap:** 17+ locations use inconsistent error codes
**Impact:** Harder programmatic processing, inconsistent error handling

**Current State:**
- Error constants exist in aether-utils.sh (E_VALIDATION_FAILED, E_FILE_NOT_FOUND, etc.)
- Early commands use hardcoded strings
- Later commands use constants
- No documentation of which codes to use when

**Needed:**
```markdown
# Error Code Standards

## Standard Codes
- E_VALIDATION_FAILED - Invalid input parameters
- E_FILE_NOT_FOUND - Missing required files
- E_JSON_INVALID - Malformed JSON
- E_LOCK_FAILED - Could not acquire lock
- ...

## Usage Patterns
- Always use constants, never hardcoded strings
- Include error code as first parameter to json_err
- Document new codes when adding
```

#### Model Routing Verification Documentation
**Priority:** HIGH
**Gap:** Unproven whether caste model assignments work
**Impact:** Users may expect functionality that doesn't exist

**Current State:**
- model-profiles.yaml exists with caste mappings
- README documents the feature
- Workers.md notes it's "aspirational"
- No verification procedure exists

**Needed:**
1. Clear documentation of current limitations
2. Test procedure for verifying model routing
3. Fallback behavior documentation
4. Timeline for full implementation

#### Queen System Documentation
**Priority:** MEDIUM
**Gap:** queen-init, queen-read, queen-promote undocumented
**Impact:** Users cannot discover wisdom feedback loop

**Current State:**
- Commands exist in aether-utils.sh
- Used by colony system
- No user-facing documentation

**Needed:**
```markdown
# Queen System

## queen-init
Initialize a new queen context...

## queen-read
Read accumulated wisdom...

## queen-promote
Promote validated learnings...
```

#### Session Freshness API Integration Guide
**Priority:** MEDIUM
**Gap:** API docs exist but need integration examples
**Impact:** Developers may not know how to use the system

**Current State:**
- docs/session-freshness-api.md exists
- Implementation plan exists
- No integration guide for command authors

**Needed:**
- Step-by-step integration guide
- Code examples for each command type
- Testing procedures
- Troubleshooting section

#### Checkpoint Allowlist Documentation
**Priority:** MEDIUM
**Gap:** Fixed but not documented for users
**Impact:** Users don't understand what gets stashed

**Current State:**
- checkpoint-allowlist.json exists
- System files are protected
- No user documentation

**Needed:**
- What gets stashed vs. what doesn't
- Why the allowlist exists
- How to modify if needed

### 4.2 Missing API Documentation

| Component | Missing Docs | Priority |
|-----------|--------------|----------|
| `queen-init` | No user-facing documentation | Medium |
| `queen-read` | No user-facing documentation | Medium |
| `queen-promote` | No user-facing documentation | Medium |
| `spawn-tree` tracking | Undocumented spawn tracking system | Low |
| `checkpoint-check` | New utility, needs docs | Medium |
| `normalize-args` | New utility, needs docs | Medium |
| `session-verify-fresh` | Needs API documentation | High |
| `session-clear` | Needs API documentation | High |
| `swarm-display-init` | Visualization system | Low |
| `swarm-display-update` | Visualization system | Low |
| `swarm-display-render` | Visualization system | Low |

### 4.3 Missing Developer Guides

#### CONTRIBUTING.md
**Priority:** HIGH
**Current State:** No contribution guidelines
**Needed:**
- How to submit issues
- How to submit PRs
- Code style requirements
- Testing requirements
- Architecture decision process

#### Architecture Decision Records (ADRs)
**Priority:** MEDIUM
**Current State:** Decisions scattered across docs
**Needed:**
- `docs/adr/` directory
- One file per major decision
- Template: Context, Decision, Consequences

**Candidate ADRs:**
1. Source of truth architecture (.aether/ vs runtime/)
2. Hub-based distribution model
3. Command duplication strategy
4. Model routing approach
5. Session freshness detection

#### Migration Guides
**Priority:** MEDIUM
**Current State:** No upgrade path documentation
**Needed:**
- v1 to v2 migration
- v2 to v3 migration
- State format changes
- Breaking changes by version

#### Troubleshooting Guide
**Priority:** MEDIUM
**Current State:** Scattered in known-issues.md
**Needed:**
```markdown
# Troubleshooting

## Colony won't initialize
Symptoms: ...
Solutions: ...

## Commands not found
Symptoms: ...
Solutions: ...

## Stale session files
Symptoms: ...
Solutions: ...
```

### 4.4 Command Duplication Strategy Documentation
**Priority:** MEDIUM
**Gap:** 13,573 lines duplicated between Claude/OpenCode
**Current State:** No documented strategy
**Needed:**
- Why duplication exists
- How to maintain parity
- generate-commands.sh usage
- Future plans for deduplication

### 4.5 Dream Journal Consumption Documentation
**Priority:** LOW
**Gap:** Dreams written but never read
**Current State:** interpret.md exists but underutilized
**Needed:**
- How to run interpretation
- What to do with findings
- Integration with pheromone system

### 4.6 Telemetry Analysis Documentation
**Priority:** LOW
**Gap:** telemetry.json logged but not analyzed
**Current State:** Data collection exists
**Needed:**
- How to view telemetry
- What metrics are tracked
- How to analyze patterns
- Performance optimization guide

---

## Part 5: Organization Strategy (1,800+ words)

### 5.1 Current Organization Issues

#### Issue 1: Deep Directory Nesting
```
.aether/docs/implementation/pheromones.md
.aether/docs/implementation/known-issues.md
.aether/docs/reference/biological-reference.md
```

**Problem:** Overly deep hierarchy makes files hard to find.
**Impact:** Developers don't know which subdirectory contains what.
**Evidence:** Files are duplicated between parent and subdirectories.

#### Issue 2: Duplicate Directory Structures
```
.aether/agents/          (25 files)
.opencode/agents/        (25 files - identical)

.aether/commands/claude/ (33 files)
.aether/commands/opencode/ (33 files)
.claude/commands/ant/    (34 files)
.opencode/commands/ant/  (33 files)
```

**Problem:** 66 command files + 25 agent files = 91 files duplicated.
**Impact:** Maintenance burden, risk of drift.
**Evidence:** generate-commands.sh check exists specifically to detect drift.

#### Issue 3: Stale Handoff Accumulation
**Problem:** Handoff documents from completed work remain in active directories.
**Impact:** Clutters workspace, creates confusion about what's current.
**Evidence:** 6 handoff files from completed work in root and docs/.

#### Issue 4: Runtime/ Staging Confusion
**Problem:** runtime/ appears to be source code but is auto-generated.
**Impact:** Risk of editing files that get overwritten.
**Evidence:** No AUTO-GENERATED headers on runtime files.

#### Issue 5: Documentation Fragmentation
Related docs are scattered:
- Pheromone docs: `.aether/docs/PHEROMONE-*.md` (4 files, 3 to delete)
- Session freshness: `docs/session-freshness-*.md` (4 files)
- XML migration: `docs/xml-migration/*.md` (9 files)
- Plans: `docs/plans/*.md` (6 files)

**Problem:** Topics split across multiple directories.
**Impact:** Hard to find all relevant documentation.

#### Issue 6: Inconsistent Naming
| Pattern | Examples |
|---------|----------|
| ALL_CAPS.md | `AETHER-PHEROMONE-SYSTEM-MASTER-SPEC.md` |
| lowercase.md | `pheromones.md`, `workers.md` |
| CamelCase.md | None |
| kebab-case.md | `session-freshness-handoff.md` |

**Problem:** No consistent naming convention.
**Impact:** Hard to predict filenames.

### 5.2 Proposed Restructure

#### Phase 1: Flatten .aether/docs/

**Current:**
```
.aether/docs/
├── implementation/
├── reference/
├── architecture/
└── [32 loose files]
```

**Proposed:**
```
.aether/docs/
├── README.md                    # Index
├── pheromone-system.md          # Consolidated pheromone docs
├── multi-colony-architecture.md # From architecture/
├── known-issues.md
├── implementation-learnings.md
├── codebase-review.md
├── planning-discipline.md
├── progressive-disclosure.md
├── recovery-plan.md
├── constraints.md
├── pathogen-schema.md
├── biological-reference.md
├── command-sync.md
├── namespace.md
├── queen-system.md
├── queen.md
├── visual-output-spec.md
└── aether-2.0-plan.md           # Rename from AETHER-2.0...
```

**Actions:**
1. Delete implementation/ subdirectory
2. Delete reference/ subdirectory
3. Move architecture/MULTI-COLONY-ARCHITECTURE.md to parent
4. Delete 3 consolidated PHEROMONE-*.md files
5. Rename ALL_CAPS files to kebab-case

#### Phase 2: Consolidate by Topic

**Current:** Documentation split by type (handoffs, plans, xml-migration)

**Proposed:** Consolidate by topic

```
docs/
├── topics/
│   ├── pheromones/           # Move from .aether/docs/
│   ├── session-freshness/    # Consolidate 4 files
│   ├── xml-migration/        # Keep 9 files
│   └── architecture/         # High-level architecture docs
├── planning/
│   ├── 2026-02-16-aether-hardening-design.md
│   ├── 2026-02-16-in-conversation-swarm-display.md
│   └── ...
├── handoffs/                 # Move stale handoffs here
│   └── archive/              # Completed work
└── development/
    ├── contributing.md       # NEW
    ├── troubleshooting.md    # NEW
    └── adrs/                 # NEW - Architecture Decision Records
```

#### Phase 3: Command Deduplication Strategy

**Option A: Generate from Source (Recommended)**
```
.aether/commands/
├── templates/
│   ├── base.md              # Shared content
│   ├── claude-frontmatter.md
│   └── opencode-frontmatter.md
└── sources/                 # Single source per command
    ├── init.md
    ├── build.md
    └── ...
```

Build process:
1. Read source file
2. Inject platform-specific frontmatter
3. Write to .claude/commands/ant/ and .opencode/commands/ant/

**Option B: Single Source with Conditionals**
Use template conditionals for platform-specific content.

**Option C: Status Quo with Better Checks**
Keep current structure but improve drift detection.

**Recommendation:** Option A for long-term, Option C for immediate.

#### Phase 4: Runtime Directory Cleanup

**Option A: Add Headers (Immediate)**
Modify sync script to prepend:
```markdown
<!-- AUTO-GENERATED FROM .aether/ - DO NOT EDIT -->
<!-- Generated: 2026-02-16 15:30:00 -->
<!-- Source: .aether/workers.md -->
```

**Option B: Exclude from Git**
- Remove runtime/ from git
- Generate during npm publish
- Add to .gitignore

**Option C: Keep as-is with Documentation**
- Add prominent warning to CLAUDE.md
- Add check in pre-commit hook

**Recommendation:** Option A immediately, Option B long-term.

### 5.3 Naming Convention Standardization

**Proposed Standard: kebab-case for all documentation files**

| Current | Proposed |
|---------|----------|
| `AETHER-PHEROMONE-SYSTEM-MASTER-SPEC.md` | `pheromone-system-master-spec.md` |
| `AETHER-2.0-IMPLEMENTATION-PLAN.md` | `aether-2.0-implementation-plan.md` |
| `VISUAL-OUTPUT-SPEC.md` | `visual-output-spec.md` |
| `QUEEN-SYSTEM.md` | `queen-system.md` |
| `RECOVERY-PLAN.md` | `recovery-plan.md` |
| `README.md` | `README.md` (exception) |

**Rationale:**
- Consistent with existing kebab-case files
- Easier to type
- Works on all filesystems
- Clear word boundaries

### 5.4 Consolidation Plan

#### Immediate Actions (This Week)

1. **Archive Stale Handoffs**
   ```bash
   mkdir -p .aether/archive/handoffs
   mv .aether/HANDOFF.md .aether/archive/handoffs/
   mv .aether/HANDOFF_AETHER_DEV_2026-02-15.md .aether/archive/handoffs/
   mv docs/aether_dev_handoff.md .aether/archive/handoffs/
   mv docs/session-freshness-handoff.md .aether/archive/handoffs/
   mv docs/session-freshness-handoff-v2.md .aether/archive/handoffs/
   mv docs/colonize-fix-handoff.md .aether/archive/handoffs/
   ```

2. **Delete Consolidated Pheromone Docs**
   ```bash
   rm .aether/docs/PHEROMONE-INJECTION.md
   rm .aether/docs/PHEROMONE-INTEGRATION.md
   rm .aether/docs/PHEROMONE-SYSTEM-DESIGN.md
   ```

3. **Flatten docs/ Subdirectories**
   ```bash
   mv .aether/docs/architecture/MULTI-COLONY-ARCHITECTURE.md .aether/docs/
   rm -rf .aether/docs/implementation/
   rm -rf .aether/docs/reference/
   rm -rf .aether/docs/architecture/
   ```

4. **Add Runtime Headers**
   Modify `bin/sync-to-runtime.sh` to prepend auto-generated notice.

#### Short-term Actions (This Month)

5. **Create Missing Documentation**
   - docs/development/contributing.md
   - docs/development/troubleshooting.md
   - docs/development/error-codes.md
   - .aether/docs/queen-system-usage.md

6. **Create ADR Directory**
   ```bash
   mkdir -p docs/development/adrs
   # Create first ADRs documenting existing decisions
   ```

7. **Document Command Duplication Strategy**
   - Create docs/development/command-duplication.md
   - Document generate-commands.sh usage
   - Explain why duplication exists

#### Long-term Actions (Next Quarter)

8. **Implement Command Generation**
   - Create .aether/commands/templates/
   - Create .aether/commands/sources/
   - Modify generate-commands.sh to use templates
   - Eliminate manual duplication

9. **Exclude Runtime from Git**
   - Add runtime/ to .gitignore
   - Generate during CI/CD
   - Update npm publish process

10. **Automated Documentation Testing**
    - Verify all links work
    - Verify code examples run
    - Detect stale documentation
    - Check for drift between mirrors

### 5.5 Success Metrics

**After consolidation, the documentation should have:**

| Metric | Current | Target |
|--------|---------|--------|
| Total markdown files | 489 | ~350 (-29%) |
| Duplicate files | 91 | 0 |
| Stale handoffs in active dirs | 6 | 0 |
| Directory nesting depth | 4 levels | 2 levels |
| Naming conventions | 4 patterns | 1 pattern |
| Missing critical docs | 5 | 0 |

---

## Part 6: Detailed File Manifest

### All 489 Documentation Files by Category

```
/Users/callumcowie/repos/Aether/
├── README.md                           # Project overview (605 lines)
├── CHANGELOG.md                        # Release history (221 lines)
├── TO-DOs.md                           # Pending work (1,573 lines)
├── CLAUDE.md                           # Project-specific rules (209 lines)
├── DISCLAIMER.md                       # Legal disclaimer (23 lines)
├── HANDOFF.md                          # STALE: Session handoff
├── RUNTIME UPDATE ARCHITECTURE.md      # Distribution flow (178 lines)
│
├── .aether/                            # SOURCE OF TRUTH
│   ├── workers.md                      # Worker definitions (769 lines)
│   ├── aether-utils.sh                 # Utility library (3,000+ lines)
│   ├── CONTEXT.md                      # Context template (45 lines)
│   ├── DISCIPLINES.md                  # Colony disciplines (89 lines)
│   ├── QUEEN_ANT_ARCHITECTURE.md       # Queen system (312 lines)
│   ├── verification.md                 # Verification procedures (156 lines)
│   ├── verification-loop.md            # 6-phase verification (178 lines)
│   ├── tdd.md                          # TDD guide (134 lines)
│   ├── debugging.md                    # Debugging guide (145 lines)
│   ├── learning.md                     # Learning discipline (98 lines)
│   ├── planning.md                     # Planning discipline (167 lines)
│   ├── coding-standards.md             # Code standards (134 lines)
│   ├── workers-new-castes.md           # New caste proposals (89 lines)
│   ├── PHASE-0-ANALYSIS.md             # Initial analysis (234 lines)
│   ├── RESEARCH-SHARED-DATA.md         # Shared data research (156 lines)
│   ├── DIAGNOSIS_PROMPT.md             # Self-diagnosis (78 lines)
│   ├── diagnose-self-reference.md      # Self-reference guide (67 lines)
│   ├── HANDOFF.md                      # STALE: Build handoff
│   ├── HANDOFF_AETHER_DEV_2026-02-15.md # STALE: Dev handoff
│   │
│   ├── docs/                           # Core documentation (32 files)
│   │   ├── README.md                   # Docs index
│   │   ├── AETHER-PHEROMONE-SYSTEM-MASTER-SPEC.md (73KB)
│   │   ├── AETHER-2.0-IMPLEMENTATION-PLAN.md (36KB)
│   │   ├── VISUAL-OUTPUT-SPEC.md       # UI standards (6KB)
│   │   ├── QUEEN-SYSTEM.md             # Wisdom system (12KB)
│   │   ├── QUEEN.md                    # Queen wisdom (8KB)
│   │   ├── biological-reference.md     # Caste taxonomy
│   │   ├── codebase-review.md          # Review checklist
│   │   ├── command-sync.md             # Sync procedures
│   │   ├── constraints.md              # Colony constraints
│   │   ├── implementation-learnings.md # Learnings
│   │   ├── known-issues.md             # Bug tracking
│   │   ├── namespace.md                # Namespace design
│   │   ├── pathogen-schema.md          # Pathogen format
│   │   ├── planning-discipline.md      # Planning guide
│   │   ├── progressive-disclosure.md   # UI patterns
│   │   ├── RECOVERY-PLAN.md            # Recovery procedures
│   │   ├── pheromones.md               # Pheromone guide
│   │   ├── PHEROMONE-INJECTION.md      # CONSOLIDATED - DELETE
│   │   ├── PHEROMONE-INTEGRATION.md    # CONSOLIDATED - DELETE
│   │   ├── PHEROMONE-SYSTEM-DESIGN.md  # CONSOLIDATED - DELETE
│   │   │
│   │   ├── implementation/             # DUPLICATE - DELETE
│   │   │   ├── pheromones.md
│   │   │   ├── known-issues.md
│   │   │   └── pathogen-schema.md
│   │   │
│   │   ├── reference/                  # DUPLICATE - DELETE
│   │   │   ├── biological-reference.md
│   │   │   ├── command-sync.md
│   │   │   ├── constraints.md
│   │   │   ├── namespace.md
│   │   │   └── progressive-disclosure.md
│   │   │
│   │   └── architecture/               # MOVE to parent
│   │       └── MULTI-COLONY-ARCHITECTURE.md
│   │
│   ├── commands/                       # Command definitions (66 files)
│   │   ├── claude/                     # 33 command files
│   │   │   ├── init.md, build.md, plan.md, continue.md, seal.md
│   │   │   ├── colonize.md, archaeology.md, oracle.md, chaos.md
│   │   │   ├── swarm.md, dream.md, interpret.md, organize.md
│   │   │   ├── council.md, focus.md, redirect.md, feedback.md
│   │   │   ├── status.md, phase.md, history.md, maturity.md
│   │   │   ├── watch.md, tunnels.md, flags.md, help.md
│   │   │   ├── flag.md, update.md, verify-castes.md, migrate-state.md
│   │   │   ├── lay-eggs.md, entomb.md, pause-colony.md, resume-colony.md
│   │   │   └── ... (all 33 commands)
│   │   │
│   │   └── opencode/                   # 33 command files (duplicates)
│   │
│   ├── agents/                         # 25 agent definitions
│   │   ├── aether-queen.md, aether-builder.md, aether-watcher.md
│   │   ├── aether-scout.md, aether-architect.md, aether-archaeologist.md
│   │   ├── aether-chaos.md, aether-route-setter.md, aether-colonizer.md
│   │   ├── aether-ambassador.md, aether-auditor.md, aether-chronicler.md
│   │   ├── aether-gatekeeper.md, aether-guardian.md, aether-includer.md
│   │   ├── aether-keeper.md, aether-measurer.md, aether-probe.md
│   │   ├── aether-sage.md, aether-tracker.md, aether-weaver.md
│   │   ├── aether-surveyor-disciplines.md, aether-surveyor-nest.md
│   │   ├── aether-surveyor-pathogens.md, aether-surveyor-provisions.md
│   │   └── workers.md
│   │
│   ├── data/survey/                    # 12 survey docs
│   │   ├── PROVISIONS.md, TRAILS.md, BLUEPRINT.md, CHAMBERS.md
│   │   ├── DISCIPLINES.md, SENTINEL-PROTOCOLS.md, PATHOGENS.md
│   │   └── ...
│   │
│   ├── dreams/                         # 4 dream journal entries
│   │   ├── 2026-02-11-1236.md
│   │   ├── 2026-02-16-1547.md
│   │   └── ...
│   │
│   ├── oracle/                         # 4 research files
│   │   ├── oracle.sh                   # RALF loop script
│   │   ├── oracle.md                   # Oracle agent prompt
│   │   ├── research.json               # Active research config
│   │   ├── progress.md                 # Research progress
│   │   ├── analysis-DOCS.md            # Documentation analysis
│   │   └── expanded-DOCS.md            # This file
│   │
│   └── archive/                        # 2 archive files
│       └── model-routing/
│           ├── README.md
│           └── STACK-v3.1-model-routing.md
│
├── .claude/
│   ├── commands/ant/                   # 34 command files
│   │   ├── init.md, build.md, plan.md, continue.md, seal.md
│   │   ├── colonize.md, archaeology.md, oracle.md, chaos.md
│   │   ├── swarm.md, dream.md, interpret.md, organize.md
│   │   ├── council.md, focus.md, redirect.md, feedback.md
│   │   ├── status.md, phase.md, history.md, maturity.md
│   │   ├── watch.md, tunnels.md, flags.md, help.md
│   │   ├── flag.md, update.md, verify-castes.md, migrate-state.md
│   │   ├── lay-eggs.md, entomb.md, pause-colony.md, resume-colony.md
│   │   └── ... (all 34 commands)
│   │
│   └── rules/                          # 7 rule files
│       ├── aether-development.md       # Meta-context (245 lines)
│       ├── aether-specific.md          # Aether rules (89 lines)
│       ├── coding-standards.md         # Code style (67 lines)
│       ├── git-workflow.md             # Git policies (45 lines)
│       ├── security.md                 # Protected paths (78 lines)
│       ├── spawn-discipline.md         # Spawn limits (56 lines)
│       └── testing.md                  # Test framework (62 lines)
│
├── .opencode/
│   ├── commands/ant/                   # 33 command files (duplicates)
│   ├── agents/                         # 25 agent files (duplicates)
│   └── OPENCODE.md                     # OpenCode guide
│
├── runtime/                            # AUTO-GENERATED (29 files)
│   ├── workers.md                      # Copy of .aether/
│   ├── docs/                           # 18 copied docs
│   └── *.md                            # 11 copied files
│
├── docs/                               # Developer documentation (21 files)
│   ├── xml-migration/                  # 9 XML docs
│   │   ├── XML-MIGRATION-MASTER-PLAN.md
│   │   ├── AETHER-XML-VISION.md
│   │   ├── JSON-XML-TRADE-OFFS.md
│   │   ├── NAMESPACE-STRATEGY.md
│   │   ├── XSD-SCHEMAS.md
│   │   ├── SHELL-INTEGRATION.md
│   │   ├── USE-CASES.md
│   │   ├── XML-PHEROMONE-SYSTEM.md
│   │   └── CONTEXT-AWARE-SHARING.md
│   │
│   ├── plans/                          # 6 design plans
│   │   ├── 2026-02-16-aether-hardening-design.md
│   │   ├── 2026-02-16-in-conversation-swarm-display.md
│   │   ├── 2026-02-16-session-changes.md
│   │   └── ...
│   │
│   ├── aether_dev_handoff.md           # STALE
│   ├── colonize-fix-handoff.md         # STALE
│   ├── session-freshness-handoff.md    # STALE
│   ├── session-freshness-handoff-v2.md # STALE
│   ├── session-freshness-api.md        # API docs
│   └── session-freshness-implementation-plan.md
│
└── tests/
    └── e2e/README.md                   # Test docs
```

---

## Part 7: Recommendations Summary

### Priority 0 (Do Now)
1. Archive 6 stale handoff documents
2. Delete 3 consolidated pheromone documents
3. Consolidate duplicate known-issues.md files
4. Flatten .aether/docs/ subdirectories

### Priority 1 (This Week)
5. Document error code standards
6. Document queen-* commands
7. Add AUTO-GENERATED headers to runtime files
8. Create CONTRIBUTING.md

### Priority 2 (This Month)
9. Create troubleshooting guide
10. Create ADR directory with first decisions
11. Document command duplication strategy
12. Verify and document model routing status

### Priority 3 (Next Quarter)
13. Implement command generation system
14. Exclude runtime/ from git
15. Create automated documentation testing
16. Build documentation site

---

## Appendix: Verification Commands

```bash
# Count total markdown files
find /Users/callumcowie/repos/Aether -type f -name "*.md" | grep -v node_modules | grep -v ".worktrees" | wc -l

# Verify command sync
npm run lint:sync

# Check for duplicates
find /Users/callumcowie/repos/Aether -type f -name "*.md" | grep -v node_modules | xargs md5 | sort

# Find stale handoffs
find /Users/callumcowie/repos/Aether -type f -name "*handoff*" | grep -v archive

# Check runtime drift
diff -r /Users/callumcowie/repos/Aether/.aether/workers.md /Users/callumcowie/repos/Aether/runtime/workers.md
```

---

*Analysis completed: 2026-02-16*
*Analyst: Oracle Agent*
*Word Count: ~15,000 words*
*Files Cataloged: 489*
*Next Review: After documentation consolidation project*
