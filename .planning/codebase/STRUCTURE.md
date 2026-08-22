---
last_mapped_commit: 92252d01
---

# Codebase Structure

**Analysis Date:** 2026-08-22

## Directory Layout

```
Aether/
├── cmd/                         # Go runtime commands (80+ files)
│   ├── aether/main.go          # CLI entry point
│   ├── root.go                 # Cobra root command, store init
│   ├── codex_build.go          # Build orchestration & dispatch
│   ├── codex_continue.go       # Continue verification & gating
│   ├── codex_continue_plan.go  # Continue manifest generation
│   ├── codex_continue_finalize.go # Result absorption
│   ├── codex_build_finalize.go # Build result absorption
│   ├── skills.go               # Worker brief assembly
│   ├── criterion_evidence.go   # Success criteria verification
│   ├── consolidation_lifecycle.go # Learning consolidation
│   ├── init_cmd.go             # `/ant-init` command
│   ├── plan_cmd.go             # `/ant-plan` command
│   ├── seal_cmd.go             # `/ant-seal` command
│   ├── colonize_cmd.go         # `/ant-colonize` command
│   ├── pheromone_*.go          # Signal management (focus, redirect, feedback)
│   ├── command_guide.go        # `/command-guide` reference documentation
│   ├── *_test.go               # Unit tests (mirrored file names)
│   └── testdata/               # Test fixtures
│
├── pkg/                         # Shared Go packages
│   ├── agent/                  # Agent pool & spawn orchestration
│   │   ├── agent.go            # Agent abstraction
│   │   ├── pool.go             # Concurrent agent pool
│   │   ├── spawn_tree.go       # Task graph & wave planning
│   │   ├── stream_manager.go   # Output streaming
│   │   └── task_router.go      # Task assignment
│   │
│   ├── colony/                 # Colony state & phase definitions
│   │   ├── colony.go           # Colony metadata
│   │   ├── phase.go            # Phase definition & tasks
│   │   ├── instincts.go        # Learned patterns
│   │   ├── pheromones.go       # Signal definitions
│   │   ├── constraints.go      # User constraints
│   │   ├── flags.go            # Phase flags (discovery, production, etc.)
│   │   ├── midden.go           # Failure tracking
│   │   └── parallel_mode.go    # In-repo vs worktree execution
│   │
│   ├── codex/                  # Runtime state machines & contracts
│   │   ├── building.go         # Build verification contracts
│   │   ├── continuing.go       # Continue contracts
│   │   ├── execution_binding.go # Worktree execution
│   │   └── permission_profile.go # Spawned worker permissions
│   │
│   ├── memory/                 # Learning pipeline
│   │   ├── pipeline.go         # Observation → Instinct promotion
│   │   ├── observe.go          # Capture learning observations
│   │   ├── promote.go          # Promote to instinct/QUEEN.md
│   │   ├── consolidate.go      # Seal-time 9-ant curation
│   │   ├── trust.go            # Trust scoring (60-day half-life)
│   │   └── queen.go            # QUEEN.md wisdom loading
│   │
│   ├── storage/                # Persistence layer
│   │   ├── store.go            # JSON store & atomic updates
│   │   ├── lock.go             # File locking
│   │   └── data_dir.go         # `.aether/data/` resolver
│   │
│   ├── events/                 # Event bus with TTL
│   │   ├── bus.go              # Pub/sub event system
│   │   └── event.go            # Event definitions
│   │
│   ├── graph/                  # Knowledge graph (instinct relationships)
│   │   ├── graph.go            # Instinct relationship graph
│   │   └── jq_layer.go         # jq-based queries
│   │
│   ├── exchange/               # XML import/export
│   │   ├── pheromone_xml.go    # Signal exchange format
│   │   ├── wisdom_xml.go       # Hive wisdom exchange
│   │   └── colony_archive_xml.go # Full colony export
│   │
│   ├── learn/                  # Structural learning stack (9-ant curation)
│   │   ├── curation_ants.go    # Orchestrator, archivist, critic, etc.
│   │   ├── archivist.go        # Historical observation archival
│   │   ├── critic.go           # Quality evaluation
│   │   ├── herald.go           # Hive promotion
│   │   ├── janitor.go          # TTL cleanup
│   │   ├── librarian.go        # Relationship indexing
│   │   ├── nurse.go            # Low-confidence healing
│   │   ├── scribe.go           # Audit trail
│   │   └── sentinel.go         # Corruption detection
│   │
│   ├── llm/                    # LLM provider routing
│   │   ├── routes.go           # Model selection logic
│   │   └── adapters/           # Provider-specific adapters
│   │
│   ├── trace/                  # Audit logging
│   │   └── tracer.go           # JSONL event logging
│   │
│   └── downloader/             # Binary download + extraction
│       └── downloader.go       # Release binary management
│
├── .aether/                     # Source definitions (published to hub)
│   ├── commands/               # YAML command source of truth
│   │   ├── build.yaml          # Build orchestration contract
│   │   ├── continue.yaml       # Continue orchestration contract
│   │   ├── init.yaml           # Colony initialization
│   │   ├── plan.yaml           # Phase planning
│   │   ├── discuss.yaml        # Intent clarification
│   │   ├── focus.yaml          # `/ant-focus` signal
│   │   ├── redirect.yaml       # `/ant-redirect` signal
│   │   ├── feedback.yaml       # `/ant-feedback` signal
│   │   └── *.yaml              # (60+ total command definitions)
│   │
│   ├── skills/                 # Skill definitions (55 colony + 31 domain)
│   │   ├── colony/             # Behavioral patterns
│   │   └── domain/             # Technical domain knowledge
│   │
│   ├── docs/                   # Distributed documentation
│   │   ├── wrapper-host-contract.md # Wrapper-runtime contract
│   │   ├── command-playbooks/  # Legacy reference docs (not loaded)
│   │   ├── structural-learning-stack.md # Learning pipeline docs
│   │   ├── learning-system-authority.md # Hive decision record
│   │   └── *.md                # Architecture, patterns, guides
│   │
│   ├── templates/              # JSON template generators
│   │   ├── colony_state.md     # COLONY_STATE.json template
│   │   ├── pheromones.md       # Pheromones template
│   │   └── *.md                # (12 total templates)
│   │
│   ├── exchange/               # XML schema modules
│   │   ├── pheromone-xml/      # Signal exchange schema
│   │   ├── wisdom-xml/         # Hive wisdom schema
│   │   └── colony-archive-xml/ # Full colony export schema
│   │
│   ├── utils/                  # Runtime utilities
│   │   └── oracle/oracle.md    # Oracle loop instructions
│   │
│   ├── data/                   # LOCAL ONLY (gitignored)
│   │   ├── COLONY_STATE.json   # Colony state + phases + instincts
│   │   ├── session.json        # Session bookkeeping
│   │   ├── pheromones.json     # Active signals
│   │   ├── pending-decisions.json # Unanswered decisions
│   │   ├── assumptions.json    # Plan assumptions
│   │   ├── behavior-observations.jsonl # Learning observations
│   │   ├── handoffs/           # Worker-to-worker relay notes
│   │   ├── midden/             # Failure tracking
│   │   ├── survey/             # Territory survey results
│   │   ├── phase-research/     # Scout-written research (D2.5)
│   │   ├── worker-debug/       # Worker debug artifacts (D2.5)
│   │   └── events/             # JSONL event bus (TTL cleanup)
│   │
│   ├── dreams/                 # LOCAL ONLY (session notes, not distributed)
│   ├── oracle/                 # LOCAL ONLY (deep research artifacts)
│   └── QUEEN.md                # Wisdom + user preferences + instincts
│
├── .claude/                     # Claude Code wrappers & agents
│   ├── commands/ant/           # 60 slash commands
│   │   ├── build.md            # `/ant-build` ceremony
│   │   ├── continue.md         # `/ant-continue` ceremony
│   │   ├── init.md             # `/ant-init` ceremony
│   │   ├── plan.md             # `/ant-plan` ceremony
│   │   ├── discuss.md          # `/ant-discuss` intent clarification
│   │   ├── focus.md            # `/ant-focus` signal
│   │   ├── redirect.md         # `/ant-redirect` signal
│   │   ├── seal.md             # `/ant-seal` finalization
│   │   └── *.md                # (60+ total commands)
│   │
│   ├── agents/ant/             # 27 worker agent definitions
│   │   ├── aether-builder.md   # Implementation worker
│   │   ├── aether-watcher.md   # Verification worker
│   │   ├── aether-scout.md     # Research worker
│   │   ├── aether-queen.md     # Orchestration (decision making)
│   │   ├── aether-probe.md     # Coverage analysis
│   │   ├── aether-gatekeeper.md # Security review
│   │   ├── aether-auditor.md   # Quality review
│   │   └── *.md                # (27 total agents)
│   │
│   ├── rules/                  # Development rules
│   │   └── aether-colony.md    # Colony system guide (auto-distributed)
│   │
│   ├── skills/                 # Repo-local custom skills
│   └── settings.json           # Claude Code project settings
│
├── .opencode/                   # OpenCode wrappers & agents (mirrors .claude/)
│   ├── commands/ant/           # Same 60 commands (OpenCode variants)
│   ├── agents/                 # 27 agent definitions (OpenCode variants)
│   └── OPENCODE.md             # OpenCode-specific rules
│
├── .codex/                      # Codex CLI native definitions
│   ├── agents/                 # 27 agent definitions (TOML format)
│   │   ├── aether-builder.toml
│   │   ├── aether-watcher.toml
│   │   └── *.toml              # (27 total agents)
│   │
│   └── CODEX.md                # Codex native commands & rules
│
├── .planning/                   # Phase documentation & codebase maps
│   ├── v1.27-*/                # Milestone phase folders
│   │   ├── PLAN.md             # Phase breakdown
│   │   ├── CONTEXT.md          # Decision context
│   │   ├── SUMMARY.md          # Phase completion summary
│   │   └── VERIFICATION.md     # Gate verification
│   │
│   └── codebase/               # GSD codebase analysis (this folder)
│       ├── ARCHITECTURE.md     # System design & layers
│       ├── STRUCTURE.md        # Directory layout & organization
│       ├── CONVENTIONS.md      # Coding style & patterns (quality focus)
│       ├── TESTING.md          # Test organization & frameworks (quality focus)
│       ├── STACK.md            # Technology choices (tech focus)
│       ├── INTEGRATIONS.md     # External services (tech focus)
│       └── CONCERNS.md         # Technical debt & issues (concerns focus)
│
├── go.mod                       # Go module definition
├── go.sum                       # Dependency checksums
├── CLAUDE.md                    # Project development guide (v1.0.63)
├── CHANGELOG.md                 # Release notes
└── README.md                    # Project overview
```

## Directory Purposes

**cmd/ (Go Runtime):**
- Purpose: All CLI command implementations and orchestration logic
- Contains: Command handlers (Cobra), manifests, verification logic, learning consolidation
- Key files: `root.go` (entry), `codex_build.go` (build), `codex_continue.go` (continue), `skills.go` (brief assembly)

**pkg/ (Shared Go Packages):**
- Purpose: Reusable types, algorithms, persistence, and infrastructure
- Contains: Data models, business logic, storage adapters, learning pipeline
- Key packages: `colony/` (state), `memory/` (learning), `agent/` (concurrency), `storage/` (persistence)

**.aether/ (Source of Truth):**
- Purpose: Canonical command definitions, skill library, and distributed documentation
- Contains: YAML command specs, skill definitions, architecture docs
- Published to: `~/.aether/` hub directory by `aether publish`
- Key files: `commands/*.yaml` (command contracts), `QUEEN.md` (learned wisdom)

**.claude/ (Claude Code Wrappers):**
- Purpose: User ceremony and worker spawning for Claude Code platform
- Contains: Markdown command files, agent definitions
- Never mutates: State files (COLONY_STATE.json, pheromones.json)
- Updated by: `aether publish` (syncs from `.aether/`)

**.codex/ (Codex CLI):**
- Purpose: Codex platform native command definitions and agents
- Contains: TOML agent definitions, Codex-specific rules
- Relationship: Mirrors Claude/OpenCode agent specs in TOML format for Codex runtime

**.planning/ (Phase Documentation):**
- Purpose: Milestone planning, phase context, and codebase analysis
- Contains: Phase PLAN.md, CONTEXT.md, SUMMARY.md, and GSD codebase maps
- Key subdirs: `v1.27-*` (phase folders), `codebase/` (this analysis)

## Key File Locations

**Entry Points:**
- `cmd/aether/main.go`: Binary entry point
- `cmd/root.go:Execute()`: Cobra root command and store initialization
- `.aether/commands/*.yaml`: Command source definitions (build, continue, init, plan, etc.)

**Configuration:**
- `.aether/version.json`: Version metadata
- `.aether/QUEEN.md`: Wisdom + user preferences + learned instincts
- `.aether/manifest.json`: Command registry and skill index
- `~/.aether/` (hub): Installed commands, skills, docs (shared across all repos)

**Core Logic:**
- `cmd/codex_build.go`: Build manifest generation and dispatch
- `cmd/codex_continue.go`: Verification, gating, learning
- `cmd/codex_continue_finalize.go`: Result absorption and state update
- `cmd/skills.go`: Worker brief assembly and skill injection
- `pkg/memory/pipeline.go`: Learning observation → instinct → promotion pipeline
- `pkg/agent/pool.go`: Concurrent agent execution

**Testing:**
- `cmd/*_test.go`: Unit tests (mirrored file names)
- `cmd/testdata/`: Test fixtures and snapshots
- `bench/`: Benchmark suite (acceptance tests, harness)
- `test/__snapshots__/`: Jest snapshot tests

## Naming Conventions

**Files:**
- `*_cmd.go`: Command handler files (e.g., `init_cmd.go`, `plan_cmd.go`, `seal_cmd.go`)
- `codex_*.go`: Runtime orchestration logic (e.g., `codex_build.go`, `codex_continue.go`)
- `*_test.go`: Unit test files (mirror source file names)
- `COLONY_STATE.json`: Single source of truth for colony state
- `*.yaml`: Command definitions in `.aether/commands/`
- `*.md`: Markdown wrappers and documentation

**Directories:**
- `cmd/`: Go runtime commands
- `pkg/agent/`, `pkg/colony/`, etc.: Package organization by domain
- `.aether/`: Source of truth for distributed content
- `.claude/commands/ant/`: Claude Code command wrappers
- `.aether/data/`: Local-only state (gitignored)
- `.planning/v1.XX-*/`: Phase documentation
- `.planning/codebase/`: GSD codebase analysis

**Go Types:**
- `codexBuildManifest`: Build manifest (cmd/codex_build.go)
- `codexContinueManifest`: Continue manifest (cmd/codex_continue.go)
- `codexBuildDispatch`: Worker assignment (cmd/codex_build.go)
- `ColonyState`: Full colony state (pkg/colony/colony.go)
- `Phase`: Phase definition (pkg/colony/phase.go)
- `Instinct`: Learned pattern (pkg/colony/instincts.go)

## Where to Add New Code

**New Runtime Command:**
1. Implement handler: `cmd/new_cmd.go` (func `init()` registers with rootCmd)
2. Add YAML spec: `.aether/commands/new.yaml` (command contract)
3. Add wrapper: `.claude/commands/ant/new.md` and `.opencode/commands/ant/new.md`
4. Document: `.aether/docs/command-guide.md` or run `aether command-guide new`
5. Test: `cmd/new_cmd_test.go`

**New Shared Package:**
1. Create: `pkg/new_package/` directory
2. Implement: Type definitions in `pkg/new_package/types.go` or `pkg/new_package/package.go`
3. Test: `pkg/new_package/*_test.go`
4. Import: From `cmd/` commands as needed

**New Worker Agent:**
1. Define: `.claude/agents/ant/aether-newagent.md`
2. Mirror: `.opencode/agents/aether-newagent.md` (exact copy)
3. Define: `.codex/agents/aether-newagent.toml` (TOML variant)
4. Register: Update agent registry in `.aether/manifest.json`
5. Distribute: `aether publish` syncs to hub

**New Skill:**
1. Create: `.aether/skills/domain/<skill-name>.md` or `~/.aether/skills/domain/<skill-name>.md` (user)
2. Frontmatter: `name`, `category`, `detect_patterns`, `for_roles`
3. Content: Reusable instructions + examples
4. Distribution: User skills stay local; shipped skills get `aether publish`

**New Phase:**
1. Add to plan: Edit `.aether/data/COLONY_STATE.json` (or via `/ant-plan`)
2. Define in code: Phase is automatically loaded from state; no separate code file
3. Test: Add phase fixture to test data

**New State File (in .aether/data/):**
1. Must be JSON
2. Mutated only via `store.UpdateJSONAtomically()` (file locking)
3. Never read/written by wrappers directly
4. Initialize: Via `aether` command or state-mutate

## Special Directories

**`.aether/data/` (Local Colony State):**
- Purpose: Persistent colony state (gitignored)
- Generated: By runtime commands (`build`, `continue`, `seal`, etc.)
- Committed: Never (gitignore applies)
- Mutation: Only via Go runtime (via `storage.Store`)
- Read by: Go runtime only; wrappers read via `aether status` command output

**`.aether/data/phase-research/` (Scout Research):**
- Purpose: Phase domain research artifacts written by Scout
- Generated: During phase planning (by Scout via dispatch)
- Committed: No (under `.aether/data/`, which is gitignored)
- Format: JSON or markdown artifacts per phase

**`.aether/data/worker-debug/` (Worker Debug Artifacts):**
- Purpose: Debug files written by workers (e.g., profiling output)
- Generated: During build (if worker chooses to write debug files)
- Committed: No
- Format: Any (depends on worker)

**`.aether/dreams/` (Session Notes):**
- Purpose: User dream journal entries (thoughts, observations)
- Generated: Via `/ant-dream` command
- Committed: No (gitignored)
- Format: Markdown

**`.aether/oracle/` (Deep Research):**
- Purpose: Oracle loop research artifacts (RALF cycle)
- Generated: Via `/ant-oracle` command (research exploration)
- Committed: No (gitignored)
- Format: Markdown documents

**`.aether/chambers/` (Sealed Colonies):**
- Purpose: Archived completed colonies
- Generated: By `/ant-entomb` after `/ant-seal`
- Committed: Yes (historical record)
- Format: Directory per sealed colony with full state snapshot

**`~/.aether/` (Hub - Shared Installation):**
- Purpose: Cross-colony shared runtime, commands, skills, wisdom
- Installed by: `aether publish` (from Aether repo) or `aether install` (from local dev)
- Contents: Commands, skills, docs, agents (published from `.aether/`, `.claude/`, `.opencode/`, `.codex/`)
- Hive wisdom: `~/.aether/hive/wisdom.json` (cross-colony learning)
- Eternal memory: `~/.aether/eternal/` (legacy high-value signals)
- User preferences: `~/.aether/QUEEN.md` (global user settings)

---

*Structure analysis: 2026-08-22*
