# Architecture

**Analysis Date:** 2026-03-19

## Pattern Overview

**Overall:** Multi-agent colony orchestration system with distributed command execution, pheromone-based signaling, and stateful colony lifecycle management.

**Key Characteristics:**
- Modular playbook-based orchestration (commands split into sequential stages)
- Pheromone signal system (FOCUS/REDIRECT/FEEDBACK) for inter-agent guidance
- Transactional state management with file locking for consistency
- Multi-provider agent support (Claude Code + OpenCode with shared logic)
- XML exchange layer for inter-colony knowledge transfer
- TDD-first implementation discipline across all worker castes

## Layers

**User/Command Layer:**
- Purpose: CLI entry points via slash commands (Claude Code) and CLI interface
- Location: `.claude/commands/ant/` (Claude Code), `.opencode/commands/ant/` (OpenCode), `bin/cli.js` (Node CLI)
- Contains: Command definitions, routing logic, playbook orchestration
- Depends on: Orchestration layer (Queen logic, playbooks)
- Used by: Users and AI agents invoking `/ant:*` commands

**Orchestration Layer:**
- Purpose: Phase execution planning and worker spawning coordination
- Location: `.aether/docs/command-playbooks/` (build/continue stages), `bin/lib/` (Node orchestrators)
- Contains: Build wave executors, verification gates, worker priming, state synchronization
- Depends on: State management, utilities, worker definitions
- Used by: Command layer for executing multi-stage processes

**Worker/Agent Layer:**
- Purpose: Distributed work execution — specialized agents (builder, watcher, scout, etc.)
- Location: `.claude/agents/ant/` (Claude agents), `.opencode/agents/` (OpenCode agents)
- Contains: 22 caste definitions with role specifications, execution constraints, output formats
- Depends on: Pheromone signals, state context, utilities
- Used by: Queen for phase execution

**State Management Layer:**
- Purpose: Persistent colony state, event tracking, learning/instinct management
- Location: `.aether/data/` (runtime), `bin/lib/state-*.js` (state logic)
- Contains: COLONY_STATE.json (primary), pheromones.json, constraints.json, midden (failures), session tracking
- Depends on: File locking, atomic writes
- Used by: All layers for context, decision-making, and handoff

**Utility/Infrastructure Layer:**
- Purpose: Shared primitives — file operations, XML processing, logging, state validation
- Location: `.aether/utils/` (bash), `bin/lib/` (Node)
- Contains: file-lock.sh, atomic-write.sh, xml-*.sh, state-loader.sh, spawn-tree.sh
- Depends on: System utilities (bash, jq, xmllint if available)
- Used by: All layers for reliability and consistency

**Exchange/Memory Layer:**
- Purpose: Inter-colony knowledge transfer via XML — pheromones, wisdom, registry
- Location: `.aether/exchange/` (XML modules), `.aether/schemas/` (XSD validation)
- Contains: pheromone-xml.sh (signal export/import), wisdom-xml.sh (philosophy promotion), registry-xml.sh (lineage)
- Depends on: XML utilities, XSD schemas
- Used by: Pause/resume/seal lifecycle for eternal memory

**Test/Verification Layer:**
- Purpose: Quality assurance — unit, integration, bash, and e2e tests
- Location: `tests/unit/`, `tests/integration/`, `tests/bash/`, `tests/e2e/`
- Contains: 35+ test files, test fixtures, mocks
- Depends on: AVA test runner, sinon mocks, custom helpers
- Used by: CI/CD and local development workflows

## Data Flow

**Colony Initialization (init):**

1. User invokes `/ant:init "<goal>"`
2. Command validates input and checks Aether setup
3. Node CLI initializes COLONY_STATE.json with version, goal, timestamp
4. Node CLI creates session.json with session_id
5. Node CLI calls `aether-utils.sh queen-init` to initialize QUEEN.md wisdom document
6. Colony enters "READY" state
7. User can now run `/ant:plan` or `/ant:build`

**Phase Planning (plan):**

1. User invokes `/ant:plan`
2. Route-Setter agent analyzes goal and generates phase breakdown
3. Plan written to COLONY_STATE.json `.plan.phases[]` with tasks and dependencies
4. Confidence score attached to plan
5. Colony ready for `/ant:build 1`

**Build Execution (build):**

1. User invokes `/ant:build <phase_number>`
2. Command loads build-prep playbook → context confirmation, state validation
3. Loads build-context playbook → fetches state, analyzes pheromones, extracts constraints
4. Loads build-wave playbook → spawns workers in waves (respecting dependencies)
5. Each worker spawned with context: goal, phase, task, pheromones, instincts
6. Workers self-organize (TDD cycles, file operations, command execution)
7. Loads build-verify playbook → collects worker outputs, validates file artifacts, runs tests
8. Loads build-complete playbook → synthesizes learnings, updates COLONY_STATE.json, suggests pheromones
9. Colony advances to next phase
10. Output: build summary, next action routing

**Continue/Verification (continue):**

1. User invokes `/ant:continue` after build completes
2. Watcher agent verifies build outputs (files exist, tests pass)
3. Queen extracts learnings from worker outputs and midden records
4. Instincts are promoted from learnings if confidence >= 0.8 (stored in COLONY_STATE.json)
5. Pheromones auto-emit based on patterns detected (FOCUS on complex areas, REDIRECT from known pitfalls)
6. Failure analysis updates midden (failure records with category, severity, root cause)
7. Advance to next phase or declare colony complete
8. Output: verification status, learnings extracted, suggested instincts

**Pheromone Signal Emission:**

1. User invokes `/ant:focus "<area>"`, `/ant:redirect "<pattern>"`, or `/ant:feedback "<note>"`
2. Signal written to pheromones.json with timestamp, priority, decay parameters
3. Signal broadcast to all workers via context injection
4. During build, workers read pheromones and adjust behavior (FOCUS attracts, REDIRECT repels)
5. Signals expire at phase end or decay based on timestamp

**State Synchronization:**

1. After any state change (build complete, phase advance, learning extracted), state-sync runs
2. Locks COLONY_STATE.json with exclusive file lock
3. Validates JSON schema and structure
4. Prunes events array to prevent unbounded growth (default 100 max)
5. Writes updated state atomically (temp file + mv)
6. Releases lock
7. Broadcasts state change event to activity log if available

**XML Exchange (pause/resume/seal):**

1. User invokes `/ant:pause-colony` → exports COLONY_STATE, pheromones, learnings to `.aether/exports/` as XML
2. User invokes `/ant:seal` → exports to `~/.aether/eternal/` (cross-repo eternal memory)
3. On `/ant:resume-colony` or `/ant:init`, system checks for exports and offers to reimport
4. Reimport via pheromone-xml.sh merges signals with deduplication
5. Reimport via wisdom-xml.sh promotes philosophies with namespace isolation

**Failure Tracking (midden):**

1. During build, if worker fails, failure logged to `.aether/data/midden/midden.json`
2. Failure record includes: id, category, severity, description, root_cause, phase, task_id, timestamp
3. Midden records used by `/ant:oracle` for deep research and pattern analysis
4. Midden reviewed during `/ant:continue` to extract lessons learned

## Key Abstractions

**Colony State:**
- Purpose: Persistent, transactional representation of colony progress
- Examples: `COLONY_STATE.json`, `session.json`, `pheromones.json`
- Pattern: Locked, validated, atomically written; schema enforced

**Phase:**
- Purpose: Discrete unit of work with tasks and dependencies
- Examples: Phase 1 (setup), Phase 2 (core features), Phase 3 (testing)
- Pattern: Defined by plan, tracked in state, workers collaborate per phase

**Worker/Caste:**
- Purpose: Specialized agent with role, constraints, and output format
- Examples: builder (implementation), watcher (verification), scout (research), oracle (deep analysis)
- Pattern: Spawned with context (goal, phase, task, pheromones), produces structured output

**Pheromone Signal:**
- Purpose: Non-blocking guidance from user or colony to workers
- Examples: FOCUS "authentication" (attract attention), REDIRECT "hardcoded secrets" (repel)
- Pattern: Priority-based, time-decaying, auto-suggested during builds

**Task Wave:**
- Purpose: Batch of independent tasks executed in parallel respecting dependency order
- Examples: Wave 1 (setup), Wave 2 (core logic + tests), Wave 3 (integration)
- Pattern: Dependency graph resolved, tasks spawned in order-respecting batches

**Playbook:**
- Purpose: Sequential stage definition for multi-stage commands
- Examples: build-prep, build-context, build-wave, build-verify, build-complete
- Pattern: Markdown document with steps, loaded by orchestrator, executed sequentially

**Instinct:**
- Purpose: Learned pattern with confidence score (0.0–1.0)
- Examples: "Always use atomic writes for state" (confidence: 0.95), "Cache file lists in loops" (0.78)
- Pattern: Extracted from phase learnings, stored in COLONY_STATE.json, reused in subsequent phases

## Entry Points

**User-Facing Commands:**
- Location: `.claude/commands/ant/` (definitions), served via Claude Code slash commands
- Triggers: User types `/ant:init`, `/ant:build`, `/ant:continue`, etc.
- Responsibilities: Parse arguments, load playbooks, orchestrate worker spawning, route results

**CLI Entry:**
- Location: `bin/cli.js`
- Triggers: `aether` command, `npm install -g aether-colony` setup
- Responsibilities: Global hub initialization, command sync, installation, validation

**Package Installation:**
- Location: `bin/cli.js` postinstall hook
- Triggers: `npm install aether-colony` in any repo
- Responsibilities: Call `setupHub()` to populate `~/.aether/` with system files

**Test Entry:**
- Location: `tests/unit/*.test.js`, `tests/bash/*.sh`, `tests/integration/*.test.js`
- Triggers: `npm test`, `npm run test:bash`
- Responsibilities: Verify state management, worker spawning, file operations, pheromone logic

## Error Handling

**Strategy:** Fail-fast with JSON error reporting; transactional state prevents corruption.

**Patterns:**

**Structured Errors:**
- All errors return JSON to stderr with `code` and `message`
- Error codes defined in `.aether/utils/error-handler.sh` and `bin/lib/errors.js`
- Examples: `E_LOCK_FAILED`, `E_JSON_INVALID`, `E_FILE_NOT_FOUND`, `E_VALIDATION_FAILED`

**File Lock Guarantees:**
- Before modifying COLONY_STATE.json, acquire exclusive lock via `file-lock.sh`
- Lock prevents concurrent writes from multiple agents
- Stale locks detected and cleaned up automatically (PID-based)
- Timeout after 30 seconds with clear error message

**Atomic Writes:**
- All state changes use atomic-write.sh (temp file + mv) not direct writes
- Prevents corruption from crashes mid-write
- Rollback via `git checkout` if user rejects partial changes

**Rollback:**
- Users can restore COLONY_STATE.json from git history with `git checkout`
- Archive system (chambers) preserves complete phase states before sealing
- Eternal memory (XML exports) preserved for cross-colony recovery

**Worker Failure Isolation:**
- If one worker fails, other wave tasks continue
- Failures logged to midden; don't cascade to unrelated tasks
- Build halts only on hard failures (state corruption, critical setup)

## Cross-Cutting Concerns

**Logging:**
- Bash: colorized output via colorize-log.sh (timestamps, severity levels)
- Node: structured logging via logger.js (JSON events to stdout, diagnostics to stderr)
- Activity log: written to `.aether/data/` if writable, includes all state changes

**Validation:**
- State: JSON schema validation via state-sync.js (required fields, type checking)
- Files: Artifact existence checks, test output parsing, git status detection
- Pheromones: Signal structure validated before emission, duplicates deduplicated

**Authentication:**
- Not handled by Aether — delegated to LiteLLM proxy for model routing
- CLI respects OPENAI_API_KEY, ANTHROPIC_API_KEY environment variables
- Workers inherit model context from parent spawn

**Concurrency:**
- File-based locking (not distributed) — safe within single machine
- State updates serialized via exclusive locks
- Multiple agents can read state concurrently; writes are serialized

**Testing Coverage:**
- Unit: State management, file operations, error handling (35+ tests)
- Integration: Pheromone emission, learning extraction, instinct pipeline (8+ tests)
- Bash: Lock lifecycle, XML round-trip, command sync (4+ tests)
- Target: 80%+ coverage for new code per TDD discipline

---

*Architecture analysis: 2026-03-19*
