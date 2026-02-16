# Worker/Agent System Analysis

## Executive Summary

The Aether colony implements a sophisticated multi-caste worker system with 22 distinct castes, each specializing in different aspects of software development. The system uses a biological metaphor (ants, colonies, castes) to organize work, with structured spawn trees, depth-based delegation limits, and a (currently non-functional) model routing system.

---

## Caste Catalog (22 Total)

### Core Castes (6)

#### 1. Queen 👑🐜
- **Role:** Colony orchestrator and coordinator
- **Files:** `.aether/agents/aether-queen.md`, `.opencode/agents/aether-queen.md`
- **Purpose:** Sets colony intention, manages state, spawns specialized workers, synthesizes results, advances phases
- **Model Assignment:** None (orchestrator only, not a worker)
- **Spawn Limits:** Depth 0, max 4 direct spawns
- **Key Behaviors:**
  - Controls phase boundaries
  - Uses pheromone signals (focus, redirect, feedback) to guide behavior
  - Enforces verification discipline (Iron Law: no completion without fresh evidence)

#### 2. Builder 🔨🐜
- **Role:** Implementation and code execution
- **Files:** `.aether/agents/aether-builder.md`, `.opencode/agents/aether-builder.md`, `.aether/workers.md`
- **Purpose:** Implements code, executes commands, manipulates files to achieve concrete outcomes
- **Model Assignment:** kimi-k2.5 (configured but non-functional)
- **Spawn Limits:** Depth 1-2, max 4 at depth 1, 2 at depth 2
- **Key Behaviors:**
  - TDD-First workflow (RED → VERIFY RED → GREEN → VERIFY GREEN → REFACTOR)
  - Systematic debugging discipline (no fixes without root cause)
  - Spawns only for genuine surprise (3x complexity)
- **Name Prefixes:** Chip, Hammer, Forge, Mason, Brick, Anvil, Weld, Bolt

#### 3. Watcher 👁️🐜
- **Role:** Validation, testing, quality assurance
- **Files:** `.aether/agents/aether-watcher.md`, `.opencode/agents/aether-watcher.md`, `.aether/workers.md`
- **Purpose:** Validates implementation, runs tests, ensures quality, guards phase boundaries
- **Model Assignment:** kimi-k2.5 (configured but non-functional)
- **Spawn Limits:** Depth 1-2, max 4 at depth 1, 2 at depth 2
- **Key Behaviors:**
  - The Watcher's Iron Law: Evidence before approval, always
  - Mandatory execution verification (syntax, import, launch, test suite)
  - Cannot exceed 6/10 quality score if execution checks fail
  - Creates flags for verification failures
- **Name Prefixes:** Vigil, Sentinel, Guard, Keen, Sharp, Hawk, Watch, Alert

#### 4. Scout 🔍🐜
- **Role:** Research, documentation lookup, exploration
- **Files:** `.aether/agents/aether-scout.md`, `.opencode/agents/aether-scout.md`, `.aether/workers.md`
- **Purpose:** Gathers information, searches documentation, retrieves context
- **Model Assignment:** kimi-k2.5 (configured but non-functional)
- **Spawn Limits:** Depth 1-2, max 4 at depth 1, 2 at depth 2
- **Key Behaviors:**
  - Plans research approach with sources, keywords, validation strategy
  - Uses Grep, Glob, Read, WebSearch, WebFetch
  - May spawn another scout for parallel research domains
- **Name Prefixes:** Swift, Dash, Ranger, Track, Seek, Path, Roam, Quest

#### 5. Colonizer 🗺️🐜
- **Role:** Codebase exploration and mapping
- **Files:** Defined in `.aether/workers.md` and `.aether/agents/aether-queen.md` (no standalone agent file)
- **Purpose:** Explores and indexes codebase structure, builds semantic understanding, detects patterns
- **Model Assignment:** kimi-k2.5 (configured but non-functional)
- **Spawn Limits:** Depth 1-2
- **Key Behaviors:**
  - Uses Glob, Grep, Read for exploration
  - Detects architecture patterns, naming conventions, anti-patterns
  - Maps dependencies (imports, call chains, data flow)
- **Name Prefixes:** Pioneer, Map, Chart, Venture, Explore, Compass, Atlas, Trek

#### 6. Architect 🏛️🐜
- **Role:** Knowledge synthesis and documentation coordination
- **Files:** `.aether/agents/aether-architect.md`, `.opencode/agents/aether-architect.md`, `.aether/workers.md`
- **Purpose:** Synthesizes knowledge, extracts patterns, coordinates documentation
- **Model Assignment:** glm-5 (configured but non-functional)
- **Spawn Limits:** Depth 1-2, rarely spawns (synthesis work is usually atomic)
- **Key Behaviors:**
  - Analyzes input for knowledge organization needs
  - Extracts success patterns, failure patterns, preferences, constraints
  - Creates coherent structures with actionable summaries
- **Name Prefixes:** Blueprint, Draft, Design, Plan, Schema, Frame, Sketch, Model

#### 7. Route-Setter 📋🐜
- **Role:** Planning and task decomposition
- **Files:** `.aether/agents/aether-route-setter.md`, `.opencode/agents/aether-route-setter.md`, `.aether/workers.md`
- **Purpose:** Creates structured phase plans, breaks down goals into achievable tasks
- **Model Assignment:** kimi-k2.5 (configured but non-functional)
- **Spawn Limits:** Depth 1-2
- **Key Behaviors:**
  - Bite-sized tasks (2-5 minutes each)
  - Exact file paths (no ambiguity)
  - Complete code (not "add appropriate code")
  - TDD flow in planning
- **Name Prefixes:** Route, Plan, Chart, Path

---

### Development Cluster - Weaver Ants (4)

#### 8. Weaver 🔄🐜
- **Role:** Code refactoring and restructuring
- **Files:** `.aether/agents/aether-weaver.md`, `.opencode/agents/aether-weaver.md`
- **Purpose:** Transforms tangled code into clean patterns without changing behavior
- **Model Assignment:** None specified (inherits default)
- **Spawn Limits:** Depth 1-2, max 4 at depth 1, 2 at depth 2
- **Key Behaviors:**
  - Never changes behavior during refactoring
  - Maintains test coverage (80%+ target)
  - Small, incremental changes
  - Techniques: Extract Method/Class, Inline, Rename, Move, Replace Conditional with Polymorphism
- **Name Prefixes:** Weave, Knit, Spin, Twine, Transform, Mend

#### 9. Probe 🧪🐜
- **Role:** Test generation and coverage analysis
- **Files:** `.aether/agents/aether-probe.md`, `.opencode/agents/aether-probe.md`
- **Purpose:** Digs deep to expose hidden bugs and untested paths
- **Model Assignment:** None specified (inherits default)
- **Spawn Limits:** Depth 1-2, max 4 at depth 1, 2 at depth 2
- **Key Behaviors:**
  - Scans for untested paths
  - Generates test cases
  - Runs mutation testing
  - Coverage targets: Lines 80%+, Branches 75%+, Functions 90%+, Critical paths 100%
- **Name Prefixes:** Test, Probe, Excavate, Uncover, Edge, Mutant, Trial, Check

#### 10. Ambassador 🔌🐜
- **Role:** Third-party API integration
- **Files:** `.aether/agents/aether-ambassador.md`, `.opencode/agents/aether-ambassador.md`
- **Purpose:** Bridges internal systems with external services
- **Model Assignment:** None specified (inherits default)
- **Spawn Limits:** Depth 1-2, max 4 at depth 1, 2 at depth 2
- **Key Behaviors:**
  - Researches external APIs thoroughly
  - Designs integration patterns (Client Wrapper, Circuit Breaker, Retry with Backoff)
  - Tests error scenarios
  - Security: API keys in env vars, HTTPS always, no secrets in logs
- **Name Prefixes:** Bridge, Connect, Link, Diplomat, Protocol, Network, Port, Socket

#### 11. Tracker 🐛🐜
- **Role:** Bug investigation and root cause analysis
- **Files:** `.aether/agents/aether-tracker.md`, `.opencode/agents/aether-tracker.md`
- **Purpose:** Follows error trails to their source
- **Model Assignment:** None specified (inherits default)
- **Spawn Limits:** Depth 1-2, max 4 at depth 1, 2 at depth 2
- **Key Behaviors:**
  - Gathers evidence (logs, traces, context)
  - Reproduces consistently
  - Traces execution path
  - The 3-Fix Rule: If 3 fixes fail, escalate with architectural concern
- **Name Prefixes:** Track, Trace, Debug, Hunt, Follow, Trail, Find, Seek

---

### Knowledge Cluster - Leafcutter Ants (4)

#### 12. Chronicler 📝🐜
- **Role:** Documentation generation
- **Files:** `.aether/agents/aether-chronicler.md`, `.opencode/agents/aether-chronicler.md`
- **Purpose:** Preserves knowledge in written form
- **Model Assignment:** None specified (inherits default)
- **Spawn Limits:** Depth 1-2, max 4 at depth 1, 2 at depth 2
- **Key Behaviors:**
  - Surveys codebase to understand
  - Identifies documentation gaps
  - Documents APIs, guides, changelogs
  - Writing principles: Start with "why", clear language, working examples
- **Name Prefixes:** Record, Write, Document, Chronicle, Scribe, Archive, Script, Ledger

#### 13. Keeper 📚🐜
- **Role:** Knowledge curation and pattern archiving
- **Files:** `.aether/agents/aether-keeper.md`, `.opencode/agents/aether-keeper.md`
- **Purpose:** Organizes patterns and preserves colony wisdom
- **Model Assignment:** None specified (inherits default)
- **Spawn Limits:** Depth 1-2, max 4 at depth 1, 2 at depth 2
- **Key Behaviors:**
  - Collects wisdom from patterns and lessons
  - Organizes by domain (patterns/, constraints/, learnings/)
  - Validates patterns work
  - Prunes outdated info
- **Name Prefixes:** Archive, Store, Curate, Preserve, Guard, Keep, Hold, Save

#### 14. Auditor 👥🐜
- **Role:** Code review with specialized lenses
- **Files:** `.aether/agents/aether-auditor.md`, `.opencode/agents/aether-auditor.md`
- **Purpose:** Examines code with expert eyes for security, performance, quality
- **Model Assignment:** None specified (inherits default)
- **Spawn Limits:** Depth 1-2, max 4 at depth 1, 2 at depth 2
- **Key Behaviors:**
  - Security Lens: Input validation, auth, SQL injection, XSS, secrets
  - Performance Lens: Algorithm complexity, query efficiency, memory, caching
  - Quality Lens: Readability, test coverage, error handling, documentation
  - Maintainability Lens: Coupling, technical debt, duplication
- **Name Prefixes:** Review, Inspect, Exam, Scrutin, Verify, Check, Audit, Assess

#### 15. Sage 📜🐜
- **Role:** Analytics and trend analysis
- **Files:** `.aether/agents/aether-sage.md`, `.opencode/agents/aether-sage.md`
- **Purpose:** Extracts trends from history to guide decisions
- **Model Assignment:** None specified (inherits default)
- **Spawn Limits:** Depth 1-2, max 4 at depth 1, 2 at depth 2
- **Key Behaviors:**
  - Development metrics: Velocity, cycle time, deployment frequency
  - Quality metrics: Bug density, test coverage trends, technical debt
  - Team metrics: Work distribution, collaboration patterns
  - Creates visualizations: Trend lines, heat maps, cumulative flow diagrams
- **Name Prefixes:** Sage, Wise, Oracle, Prophet, Analyst, Trend, Pattern, Insight

---

### Quality Cluster - Soldier Ants (4)

#### 16. Guardian 🛡️🐜
- **Role:** Security audits and vulnerability scanning
- **Files:** `.aether/agents/aether-guardian.md`, `.opencode/agents/aether-guardian.md`
- **Purpose:** Patrols for security vulnerabilities and defends against threats
- **Model Assignment:** None specified (inherits default)
- **Spawn Limits:** Depth 1-2, max 4 at depth 1, 2 at depth 2
- **Key Behaviors:**
  - Scans for OWASP Top 10 vulnerabilities
  - Checks dependencies for CVEs
  - Security domains: Auth/AuthZ, Input Validation, Data Protection, Infrastructure
- **Name Prefixes:** Defend, Patrol, Watch, Vigil, Shield, Guard, Armor, Fort

#### 17. Measurer ⚡🐜
- **Role:** Performance profiling and optimization
- **Files:** `.aether/agents/aether-measurer.md`, `.opencode/agents/aether-measurer.md`
- **Purpose:** Benchmarks and optimizes system performance
- **Model Assignment:** None specified (inherits default)
- **Spawn Limits:** Depth 1-2, max 4 at depth 1, 2 at depth 2
- **Key Behaviors:**
  - Establishes performance baselines
  - Benchmarks under load
  - Profiles code paths
  - Identifies bottlenecks
- **Name Prefixes:** Metric, Gauge, Scale, Measure, Benchmark, Track, Count, Meter

#### 18. Includer ♿🐜
- **Role:** Accessibility audits and WCAG compliance
- **Files:** `.aether/agents/aether-includer.md`, `.opencode/agents/aether-includer.md`
- **Purpose:** Ensures all users can access the application
- **Model Assignment:** None specified (inherits default)
- **Spawn Limits:** Depth 1-2, max 4 at depth 1, 2 at depth 2
- **Key Behaviors:**
  - Runs automated accessibility scans
  - Manual testing (keyboard, screen reader)
  - Reviews code for semantic HTML and ARIA
  - WCAG compliance levels: A (minimum), AA (standard), AAA (enhanced)
- **Name Prefixes:** Access, Include, Open, Welcome, Reach, Universal, Equal, A11y

#### 19. Gatekeeper 📦🐜
- **Role:** Dependency management and supply chain security
- **Files:** `.aether/agents/aether-gatekeeper.md`, `.opencode/agents/aether-gatekeeper.md`
- **Purpose:** Guards what enters the codebase
- **Model Assignment:** None specified (inherits default)
- **Spawn Limits:** Depth 1-2, max 4 at depth 1, 2 at depth 2
- **Key Behaviors:**
  - Inventories all dependencies
  - Scans for security vulnerabilities (CVE database)
  - Audits licenses for compliance (Permissive, Weak Copyleft, Strong Copyleft, Proprietary)
  - Assesses dependency health
- **Name Prefixes:** Guard, Protect, Secure, Shield, Defend, Bar, Gate, Checkpoint

---

### Special Castes (3)

#### 20. Archaeologist 🏺🐜
- **Role:** Git history excavation
- **Files:** `.aether/agents/aether-archaeologist.md`, `.opencode/agents/aether-archaeologist.md`
- **Purpose:** Excavates why code exists through git history
- **Model Assignment:** glm-5 (configured but non-functional)
- **Spawn Limits:** Depth 1-2
- **Key Behaviors:**
  - Read-only: NEVER modifies code or colony state
  - Uses git log, git blame, git show, git log --follow
  - Identifies stability map, knowledge concentration, incident archaeology
- **Name Prefixes:** Relic, Fossil, Dig, Shard, Epoch, Strata, Lore, Glyph

#### 21. Oracle 🔮🐜
- **Role:** Deep research (RALF loop)
- **Files:** Defined in `.aether/workers.md` (no standalone agent file)
- **Purpose:** Performs deep research using the RALF (Research-Analyze-Learn-Findings) loop
- **Model Assignment:** minimax-2.5 (configured but non-functional)
- **Spawn Limits:** Depth 1-2
- **Key Behaviors:**
  - Deep research specialist
  - Used by `/ant:oracle` command
  - Not fully documented in agent files
- **Name Prefixes:** Sage, Seer, Vision, Augur, Mystic, Sibyl, Delph, Pythia

#### 22. Chaos 🎲🐜
- **Role:** Edge case testing and resilience probing
- **Files:** `.aether/agents/aether-chaos.md`, `.opencode/agents/aether-chaos.md`, `.aether/workers.md`
- **Purpose:** Probes edge cases, boundary conditions, and unexpected inputs
- **Model Assignment:** kimi-k2.5 (configured but non-functional)
- **Spawn Limits:** Depth 1-2
- **Key Behaviors:**
  - Read-only: NEVER modifies code or fixes what is found
  - Investigates exactly 5 scenarios: Edge Cases, Boundary Conditions, Error Handling, State Corruption, Unexpected Inputs
  - Documents findings with reproduction steps
- **Name Prefixes:** Probe, Stress, Shake, Twist, Snap, Breach, Surge, Jolt

---

### Surveyor Sub-Castes (4 specialized surveyors)

The Surveyor caste has 4 specialized variants that write to `.aether/data/survey/`:

#### Surveyor-Disciplines 📊🐜
- **Files:** `.aether/agents/aether-surveyor-disciplines.md`, `.opencode/agents/aether-surveyor-disciplines.md`
- **Purpose:** Maps coding conventions and testing patterns
- **Outputs:** `DISCIPLINES.md`, `SENTINEL-PROTOCOLS.md`

#### Surveyor-Nest 📊🐜
- **Files:** `.aether/agents/aether-surveyor-nest.md`, `.opencode/agents/aether-surveyor-nest.md`
- **Purpose:** Maps architecture and directory structure
- **Outputs:** `BLUEPRINT.md`, `CHAMBERS.md`

#### Surveyor-Pathogens 📊🐜
- **Files:** `.aether/agents/aether-surveyor-pathogens.md`, `.opencode/agents/aether-surveyor-pathogens.md`
- **Purpose:** Identifies technical debt, bugs, and concerns
- **Outputs:** `PATHOGENS.md`

#### Surveyor-Provisions 📊🐜
- **Files:** `.aether/agents/aether-surveyor-provisions.md`, `.opencode/agents/aether-surveyor-provisions.md`
- **Purpose:** Maps technology stack and external integrations
- **Outputs:** `PROVISIONS.md`, `TRAILS.md`

---

## Spawn System

### Mechanism

Workers are spawned using Claude Code's Task tool with `subagent_type="general-purpose"`. The spawn process follows a strict protocol:

1. **Check spawn allowance:**
   ```bash
   bash .aether/aether-utils.sh spawn-can-spawn {depth}
   # Returns: {"can_spawn": true/false, "depth": N, "max_spawns": N, "current_total": N}
   ```

2. **Generate child name:**
   ```bash
   bash .aether/aether-utils.sh generate-ant-name "{caste}"
   # Returns: "Hammer-42", "Vigil-17", etc.
   ```

3. **Log the spawn:**
   ```bash
   bash .aether/aether-utils.sh spawn-log "{parent}" "{caste}" "{child}" "{task}"
   ```

4. **Use Task tool** with structured prompt including:
   - Worker spec reference (read `.aether/workers.md`)
   - Constraints from constraints.json
   - Parent context
   - Specific task
   - Spawn capability notice (depth-based)

5. **Log completion:**
   ```bash
   bash .aether/aether-utils.sh spawn-complete "{child}" "{status}" "{summary}"
   ```

### Depth Limiting (Max 3)

| Depth | Role | Can Spawn? | Max Sub-Spawns | Behavior |
|-------|------|------------|----------------|----------|
| 0 | Queen | Yes | 4 | Dispatch initial workers |
| 1 | Prime Worker | Yes | 4 | Orchestrate phase, spawn specialists |
| 2 | Specialist | Yes (if surprised) | 2 | Focused work, spawn only for unexpected complexity |
| 3 | Deep Specialist | No | 0 | Complete work inline, no further delegation |

**Global Cap:** Maximum 10 workers per phase to prevent runaway spawning.

**Spawn Decision Criteria (Depth 2+):**
Only spawn if genuine surprise:
- Task is 3x larger than expected
- Discovered a sub-domain requiring different expertise
- Found blocking dependency that needs parallel investigation

**DO NOT spawn for:**
- Tasks completable in < 10 tool calls
- Tedious but straightforward work
- Slight scope expansion within expertise

### Tree Tracking

All spawns are logged to `.aether/data/spawn-tree.txt` in pipe-delimited format:
```
2024-01-15T10:30:00Z|Queen|builder|Hammer-42|implement auth module|default|spawned
```

Format: `timestamp|parent_id|child_caste|child_name|task_summary|model|status`

The spawn tree is visible in `/ant:watch` command output and can be visualized as:
```
QUEEN (depth 0)
├── builder-1 (depth 1)
│   └── watcher-1 (depth 2)
└── scout-1 (depth 1)
```

### Compressed Handoffs

- Each level returns ONLY a summary, not full context
- Parent synthesizes child results, does not pass through
- Prevents context rot across spawn depths

---

## Model Routing

### Configuration

Model assignments are defined in `.aether/model-profiles.yaml`:

```yaml
worker_models:
  prime: glm-5
  archaeologist: glm-5
  architect: glm-5
  oracle: minimax-2.5
  route_setter: kimi-k2.5
  builder: kimi-k2.5
  watcher: kimi-k2.5
  scout: kimi-k2.5
  chaos: kimi-k2.5
  colonizer: kimi-k2.5

task_routing:
  default_model: kimi-k2.5
  complexity_indicators:
    complex:
      keywords: [design, architecture, plan, coordinate, synthesize, strategize, optimize]
      model: glm-5
    simple:
      keywords: [implement, code, refactor, write, create]
      model: kimi-k2.5
    validate:
      keywords: [test, validate, verify, check, review, audit]
      model: minimax-2.5
```

### Available Models

| Model | Provider | Context | Best For |
|-------|----------|---------|----------|
| glm-5 | Z_AI | 200K | Planning, coordination, complex reasoning |
| kimi-k2.5 | Moonshot | 256K | Code generation, visual coding, validation |
| minimax-2.5 | MiniMax | 200K | Research, architecture, task decomposition |

### Status: NON-FUNCTIONAL

**The model-per-caste routing system is aspirational only.**

From `.aether/workers.md`:
> "A model-per-caste routing system was designed and implemented (archived in `.aether/archive/model-routing/`) but cannot function due to Claude Code Task tool limitations. The archive is preserved for future use if the platform adds environment variable support for subagents."

### Blockers

1. **Claude Code Task Tool Limitation:** The Task tool does not support passing environment variables to spawned workers. All workers inherit the parent session's model configuration.

2. **No Environment Variable Inheritance:** ANTHROPIC_MODEL set in parent is not inherited by spawned workers through Task tool.

3. **Session-Level Model Selection:** Model selection happens at the session level, not per-worker. To use a specific model, user must:
   ```bash
   export ANTHROPIC_BASE_URL=http://localhost:4000
   export ANTHROPIC_AUTH_TOKEN=sk-litellm-local
   export ANTHROPIC_MODEL=kimi-k2.5
   claude
   ```

### Historical Note

The complete model routing configuration was archived. See `git show model-routing-v1-archived` for the complete configuration.

---

## Worker Priming System

### Agent Definition Files

Each caste has a dedicated agent definition file:
- `.aether/agents/aether-{caste}.md` (Claude Code)
- `.opencode/agents/aether-{caste}.md` (OpenCode)

### Agent File Structure

```yaml
---
name: aether-{caste}
description: "{description}"
---

You are **{Emoji} {Caste} Ant** in the Aether Colony. {Role description}

## Aether Integration

This agent operates as a **{specialist/orchestrator}** within the Aether Colony system. You:
- Report to the Queen/Prime worker who spawns you
- Log activity using Aether utilities
- Follow depth-based spawning rules
- Output structured JSON reports

## Activity Logging

Log progress as you work:
```bash
bash .aether/aether-utils.sh activity-log "ACTION" "{your_name} ({Caste})" "description"
```

## Your Role

As {Caste}, you:
1. {Responsibility 1}
2. {Responsibility 2}
...

## Depth-Based Behavior

| Depth | Role | Can Spawn? |
|-------|------|------------|
| 1 | Prime {Caste} | Yes (max 4) |
| 2 | Specialist | Only if surprised |
| 3 | Deep Specialist | No |

## Output Format

```json
{
  "ant_name": "{your name}",
  "caste": "{caste}",
  "status": "completed" | "failed" | "blocked",
  "summary": "What you accomplished",
  ...
}
```
```

### Priming Process

When a worker is spawned via Task tool, it receives:
1. **Worker Spec:** Reference to read `.aether/workers.md` for caste discipline
2. **Constraints:** From constraints.json (pheromone signals)
3. **Parent Context:** Task description, why spawning, parent identity
4. **Specific Task:** The sub-task to complete
5. **Spawn Capability:** Depth-based spawn permissions

---

## Dependencies Between Workers

### Typical Spawn Chains

**Build Phase:**
```
Queen (depth 0)
└── Prime Builder (depth 1)
    ├── Builder A (depth 2) - file 1
    ├── Builder B (depth 2) - file 2
    └── Watcher (depth 2) - verification
```

**Research Phase:**
```
Queen (depth 0)
└── Prime Scout (depth 1)
    ├── Scout A (depth 2) - docs
    └── Scout B (depth 2) - code
```

**Planning Phase:**
```
Queen (depth 0)
└── Route-Setter (depth 1)
    └── Colonizer (depth 2) - codebase mapping
```

### Caste Collaboration Patterns

| Primary | Spawns | For |
|---------|--------|-----|
| Builder | Watcher | Verification after implementation |
| Builder | Scout | Research unfamiliar patterns |
| Watcher | Scout | Investigate unfamiliar code |
| Route-Setter | Colonizer | Understand codebase before planning |
| Prime | Any | Based on task analysis |

---

## Issues Found

### Critical

1. **Model Routing Non-Functional (P0.5)**
   - Configuration exists but cannot be executed
   - All workers use parent's model regardless of caste assignment
   - Blocked by Claude Code Task tool limitations

2. **BUG-005/BUG-011: Lock Deadlock in flag-auto-resolve**
   - Location: `.aether/aether-utils.sh:1022`
   - If jq fails, lock never released -> deadlock
   - Workaround: Restart colony session if commands hang on flags

### Medium

3. **Caste Count Inconsistency**
   - CLAUDE.md claims 22 castes but lists different counts in different places
   - Some castes lack standalone agent files (Colonizer, Oracle)
   - Surveyor has 4 sub-variants but is counted as one caste

4. **Error Code Inconsistency (BUG-007)**
   - 17+ locations use hardcoded strings instead of `$E_*` constants
   - Pattern: early commands use strings, later commands use constants

### Minor

5. **Model Assignment Documentation Gap**
   - Some agent files specify model (Architect: glm-5, Route-Setter: kimi-k2.5)
   - Others don't specify, inherit default
   - Inconsistent documentation of intended model assignments

6. **Spawn Tree Format Versioning**
   - Comment in aether-utils.sh mentions "NEW FORMAT: includes model field"
   - Suggests format evolution without migration strategy

---

## Improvement Opportunities

### High Priority

1. **Implement True Model Routing**
   - Options:
     a) Wait for Claude Code to support env vars in Task tool
     b) Use LiteLLM proxy with routing logic
     c) Implement model-specific agent endpoints
   - Value: Optimize cost/performance by using cheaper models for simple tasks

2. **Complete Agent File Coverage**
   - Create standalone agent files for:
     - Colonizer (currently only in workers.md)
     - Oracle (currently only in workers.md)
   - Ensures consistency across the system

3. **Unify Error Code Usage**
   - Audit all error returns in aether-utils.sh
   - Replace hardcoded strings with `$E_*` constants
   - Add linting rule to prevent regression

### Medium Priority

4. **Enhanced Spawn Tree Visualization**
   - Current: Text file with pipe-delimited format
   - Improvement: ASCII tree visualization, web-based viewer
   - Value: Better understanding of colony work patterns

5. **Worker Performance Metrics**
   - Track completion rates by caste
   - Track spawn depth effectiveness
   - Identify which castes spawn most/least
   - Value: Optimize caste assignments and spawn strategies

6. **Caste-Specific Tool Access**
   - Some castes (Surveyor) specify allowed tools in frontmatter
   - Others don't specify, get default tool set
   - Standardize tool access by caste purpose

### Low Priority

7. **Dynamic Caste Creation**
   - Allow runtime definition of new castes
   - Use case: Project-specific specialist roles
   - Complexity: High (requires agent file generation)

8. **Cross-Repository Worker Migration**
   - Allow workers to migrate between repos with state
   - Use case: Multi-repo projects
   - Complexity: Medium (requires state serialization)

---

## File Inventory

### Agent Definition Files (47 total)

**`.aether/agents/` (24 files):**
- aether-ambassador.md
- aether-archaeologist.md
- aether-architect.md
- aether-auditor.md
- aether-builder.md
- aether-chaos.md
- aether-chronicler.md
- aether-gatekeeper.md
- aether-guardian.md
- aether-includer.md
- aether-keeper.md
- aether-measurer.md
- aether-probe.md
- aether-queen.md
- aether-route-setter.md
- aether-sage.md
- aether-scout.md
- aether-surveyor-disciplines.md
- aether-surveyor-nest.md
- aether-surveyor-pathogens.md
- aether-surveyor-provisions.md
- aether-tracker.md
- aether-watcher.md
- aether-weaver.md
- workers.md (reference)

**`.opencode/agents/` (23 files):**
- Same castes as .aether/agents/ (minus surveyor variants and workers.md)

### Key System Files

- `.aether/workers.md` - Main worker definitions and discipline
- `.aether/aether-utils.sh` - Spawn logging, name generation, depth checking
- `.aether/model-profiles.yaml` - Model assignments (non-functional)
- `.aether/data/spawn-tree.txt` - Spawn tree log (runtime)
- `.aether/data/activity.log` - Activity log (runtime)

---

## Summary Statistics

| Metric | Count |
|--------|-------|
| Total Castes | 22 |
| Core Castes | 7 (Queen, Builder, Watcher, Scout, Colonizer, Architect, Route-Setter) |
| Development Cluster | 4 (Weaver, Probe, Ambassador, Tracker) |
| Knowledge Cluster | 4 (Chronicler, Keeper, Auditor, Sage) |
| Quality Cluster | 4 (Guardian, Measurer, Includer, Gatekeeper) |
| Special Castes | 3 (Archaeologist, Oracle, Chaos) |
| Surveyor Sub-variants | 4 (Disciplines, Nest, Pathogens, Provisions) |
| Agent Definition Files | 47 (.aether: 24, .opencode: 23) |
| Max Spawn Depth | 3 |
| Max Workers Per Phase | 10 |
| Max Spawns at Depth 1 | 4 |
| Max Spawns at Depth 2 | 2 |
| Functional Model Routing | 0 (non-functional) |

---

*Analysis generated: 2026-02-16*
*Analyst: Oracle Caste*
*Source: Comprehensive review of .aether/workers.md, .aether/agents/*.md, .opencode/agents/*.md, .aether/aether-utils.sh, .aether/model-profiles.yaml*
