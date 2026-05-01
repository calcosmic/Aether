# Feature Landscape: Aether v1.13 Recovery Hardening & Hive Learning

**Domain:** Recovery hardening, gate resilience, worker lifecycle, hive learning system
**Researched:** 2026-05-01
**Confidence:** HIGH (all findings verified against source code in `cmd/`, `pkg/colony/`, `pkg/memory/`)

## Executive Summary

v1.13 adds two major capability layers to Aether. The first is **recovery hardening**: making build/continue gates bulletproof against phantom advancement (builds that claim success but produced nothing), adding confidence-targeted Oracle planning with approval gates, converting raw init prompts into approval-ready briefs, and replacing STOP-wall gate failures with fix/unblock paths plus a new Fixer caste. The second is **hive learning**: a repo-scoped colony memory store with SQLite-backed FTS recall, pheromone skills (auto-created procedural memory from verified difficult tasks), Keeper curator with usage tracking and stale detection, and evidence-gated learning triggers that only promote verified successful work into durable wisdom.

The recovery hardening features extend existing infrastructure that is already well-proven: the gate system (`gate.go` with 8 gates, `shouldSkipGate` logic, `gateResultsWrite`/`Read`), the Oracle RALF loop (`compatibility_cmds.go`), the build claims system (`codex_build.go` with `last-build-claims.json`), and the recovery scanner (`recover_scanner.go` with 7 stuck-state detectors). The hive learning features are more architecturally novel -- they introduce SQLite (the first non-stdlib dependency in the Go runtime) and a full procedural memory pipeline. However, the existing memory infrastructure (`pkg/memory/` with observation capture, trust scoring, instinct management, queen promotion, hive promotion) provides the provenance and evidence framework that the new system builds on.

## Feature Categories

### Category A: Recovery Hardening

#### A1. Build Provenance Validation (AAC-001, AAC-002)

**Current state:** Build complete (`codex_build_finalize.go`) records worker outcomes and writes `last-build-claims.json` with file lists and task claims. Continue gates (`codex_continue.go:2361`) check `manifest_present`, `verification_steps_passed`, `implementation_evidence`, `operational_evidence`, and `no_critical_flags`. However, the continue `implementation_evidence` gate only checks whether evidence was *recorded*, not whether it *matches* what the build claimed. A build can report 5 files created, but if only 2 actually exist in the working tree, continue still advances.

**What's missing:**
- No filesystem verification that claimed files actually exist after build
- No reconciliation between `last-build-claims.json` and actual git diff
- Build-complete does not reject zero-modification or all-failed builds
- Continue does not validate provenance (that claims trace to actual worker results)
- Phantom advancement: a build can report "completed" with all dispatches marked `completed` but no actual file changes

**Expected behavior:**
1. At build-complete, verify that claimed files exist in the working tree
2. At build-complete, reject builds where all dispatches failed or no files were modified
3. At continue, reconcile claims against actual git diff (files added/modified/deleted)
4. Surface mismatches as blocking gate failures with specific file-level detail
5. Store provenance data for audit trail

**Complexity:** MEDIUM. The claim verification logic already partially exists in `codex_build_finalize.go:404-457` (filesystem fallback discovery). Extending it to validate rather than just discover requires checking claimed paths against `os.Stat` and `git diff --name-only`.

**Dependency:** Extends `codex_build_finalize.go` and `runCodexContinueGates()`.

#### A2. Confidence-Targeted Iterative Planning -- Oracle (AAC-003)

**Current state:** Oracle runs as a RALF (Research, Analyze, Learn, Formulate) loop via `compatibility_cmds.go:52`. It accepts a topic, runs a research depth (`quick`, `balanced`, `deep`, `exhaustive`), and produces a result. The loop runs to completion without confidence targets or approval gates. Oracle state is stored in `oracle/state.json` and `oracle/plan.json`.

**What's missing:**
- No user-settable confidence target (e.g., "research until 85% confidence")
- No iterative loop that re-researches based on confidence gaps
- No approval gate between research iterations (Oracle runs to completion)
- No confidence scoring of research outputs

**Expected behavior:**
1. User sets a confidence target: `aether oracle --confidence 0.85 "topic"`
2. Oracle runs first research iteration
3. Each iteration produces a confidence score (based on source quality, evidence count, cross-verification)
4. If confidence < target, Oracle identifies gaps and runs targeted re-research
5. Loop continues until target met, max iterations reached, or user approves early
6. Each iteration checkpoint is persisted for resume capability

**Complexity:** MEDIUM-HIGH. Requires designing a confidence scoring model for research outputs. The loop infrastructure exists (Oracle already loops), but the scoring and gap identification logic is new.

**Dependency:** Extends `compatibility_cmds.go` Oracle flow. Requires new confidence model.

#### A3. Init Synthesis (AAC-004)

**Current state:** `init-research` (`init_research.go`, 598 lines) scans the codebase for tech stack, governance, git history, pheromone suggestions (10 patterns), and charter data. The init wrapper (`.claude/commands/ant/init.md`) calls `init-research` and presents charter for approval. However, the output is raw research data, not a synthesized brief. The user sees 10 separate data points and must mentally synthesize them into a launch decision.

**What's missing:**
- No synthesis of raw research data into an approval-ready brief
- No risk assessment or complexity estimate
- No recommended phase structure based on codebase analysis
- No "ready to launch" / "needs discussion" signal

**Expected behavior:**
1. Init-research produces raw data (existing)
2. New synthesis step converts raw data into a structured brief: goal alignment, scope assessment, risk flags, recommended approach, estimated complexity
3. Brief includes a launch-readiness signal (green/yellow/red)
4. User reviews brief and approves/adjusts before colony creation
5. Approved brief is stored as charter metadata for downstream reference

**Complexity:** MEDIUM. The data collection exists. The synthesis is new but bounded (it transforms known inputs into a structured output format).

**Dependency:** Extends `init_research.go` output. Minor changes to `init_cmd.go` for brief storage.

#### A4. Recoverable Gate Failures with /ant-unblock (AAC-006 through AAC-011)

**Current state:** When continue gates fail, the output includes `blocking_issues` and per-gate recovery templates (`gate.go:490-521`). The recovery templates provide text instructions ("Run `go test ./...` to see failures"). However, the failure is a STOP wall -- the user must manually investigate, fix, and re-run `/ant-continue`. There is no structured unblock path, no state persistence for gate failures, and no automatic retry with selective re-checking.

**What's missing:**
- No `/ant-unblock` command that reads gate failure context and offers structured fix paths
- No `gate-results.json` file that persists gate failure state across sessions
- No smart retry that re-runs only failed gates (currently `shouldSkipGate` handles this, but only for gates that previously passed)
- No Fixer caste agent that can investigate and fix gate failures
- Gate failure banners are text-only with no actionable CLI integration

**Expected behavior:**
1. When gates fail, persist detailed failure state to `gate-results.json` (per-phase)
2. Display structured failure banner with: which gates failed, why, and what to do
3. `/ant-unblock` reads failure state, identifies which gates can be auto-retried vs need manual fix
4. Smart retry re-runs only failed gates (tests_pass always re-runs per existing rule)
5. Fixer caste agent reads gate context, investigates root cause, applies fix, verifies, reports

**Complexity:** MEDIUM. Gate failure persistence and smart retry build on existing `shouldSkipGate` and `gateResultsWrite`/`Read`. The Fixer caste is the largest new piece.

**Dependency:** Extends `gate.go`, `codex_continue.go`. New agent caste (Fixer).

#### A5. Smart Gate Retry with State Persistence (gate-results.json)

**Current state:** Gate results are stored in `COLONY_STATE.json` via `gateResultsWrite()` (`gate.go:548-571`). Results are merged by name key (upsert). `gateResultsRead()` returns all results. The `shouldSkipGate()` function skips previously-passed gates (except `tests_pass` which always re-runs).

**What's missing:**
- Gate results are only in COLONY_STATE.json -- they are not in a standalone file that survives colony state mutations
- No per-phase gate result history (results are global, not scoped to phase)
- No retry count tracking (how many times a gate has been retried)
- No exponential backoff or cooldown for repeated gate failures
- No distinction between "soft fail" (retryable) and "hard fail" (needs manual fix)

**Expected behavior:**
1. Gate results persist to a standalone `gate-results.json` per phase
2. Each result tracks: gate name, pass/fail, detail, retry count, last retried at, cooldown until
3. Soft-fail gates auto-retry after cooldown (e.g., tests_pass after 30s)
4. Hard-fail gates require manual intervention or Fixer agent
5. `/ant-status` shows gate result summary for current phase

**Complexity:** LOW-MEDIUM. Extends existing gate result storage with richer metadata.

**Dependency:** Extends `gate.go`. Read by `/ant-status`.

#### A6. Fixer Caste (27th Agent)

**Current state:** Aether has 26 agent castes. The Medic caste handles colony health diagnosis and repair. The Tracker caste investigates bugs. Neither is designed specifically for gate failure investigation and resolution. Gate failures currently require manual user intervention.

**What's missing:**
- No agent caste whose primary role is reading gate failure context, investigating root cause, applying a fix, verifying the fix, and reporting
- No gate-specific investigation skills or patterns
- No integration between gate failure state and agent dispatch

**Expected behavior:**
1. Fixer caste receives gate failure context (which gate failed, detail, phase, manifest)
2. Fixer investigates: reads relevant files, checks test output, examines build claims
3. Fixer applies targeted fix (e.g., fixes failing test, removes broken import)
4. Fixer verifies: re-runs the specific gate check
5. Fixer reports: what was wrong, what was fixed, verification result
6. If Fixer cannot resolve, reports with diagnosis for user

**Complexity:** MEDIUM. New agent definition following established patterns. Gate context assembly is the main new logic.

**Dependency:** Requires A4 (gate failure persistence) to provide context. New agent files across 4 surfaces.

#### A7. Worker Process Lifecycle (AAC-014 through AAC-017)

**Current state:** Workers are dispatched as external processes via `codex.NewWorkerInvoker` (`codex_build.go:109`). The build pipeline tracks dispatch status but does not track process lifecycle. `cleanupStaleWorkersBeforeDispatch()` (`codex_worker_cleanup.go`) terminates stale workers before new dispatches. The Oracle loop has process tree termination (`oracle_process_unix.go`). But there is no heartbeat system, no PID tracking in colony state, and no process group management for build workers.

**What's missing:**
- No worker heartbeats (periodic liveness check)
- No PID tracking in colony state (cannot correlate a stuck worker with its OS process)
- No process group management for build workers (Oracle has this, build does not)
- No stale worker detection during execution (only cleanup before next dispatch)
- No worker timeout with graceful degradation (current timeout is wall-clock only)

**Expected behavior:**
1. Each dispatched worker records its PID in colony state
2. Workers emit periodic heartbeats (timestamp file or state update)
3. Colony runtime monitors heartbeats and detects stalled workers
4. Stalled workers are terminated via process group (SIGTERM, then SIGKILL)
5. Worker cleanup runs periodically during execution, not just before next dispatch
6. `/ant-status` shows worker process health (PID, heartbeat age, status)

**Complexity:** MEDIUM-HIGH. Process lifecycle management across platforms (Unix/Windows) is non-trivial. The Oracle process tree code provides a template, but build workers may run in different process contexts.

**Dependency:** New subsystem in `cmd/`. Extends `codex_build.go` dispatch flow.

### Category B: Hive Learning System

#### B1. Colony Memory Store (AAC-019 through AAC-021)

**Current state:** Colony memory is distributed across multiple JSON files: `learning-observations.json` (observations), `instincts.json` (instincts), `pheromones.json` (signals), `midden/midden.json` (failures), `reviews/{domain}/ledger.json` (findings). Each file has its own read/write patterns via `pkg/storage.Store`. The memory health system (`memory_health.go`) aggregates metrics from all files. There is no unified query interface -- each file must be read separately.

**What's missing:**
- No unified colony memory store (currently fragmented across 7+ JSON files)
- No query capability (cannot search across memory types)
- No budgeting (memory can grow unbounded)
- No session-injectable memory context (colony-prime reads individual files, not a unified memory view)

**Expected behavior:**
1. Colony memory store provides a unified API for reading/writing all memory types
2. Memory is budgeted (total size cap, per-type caps)
3. Colony-prime injects a memory summary section into worker context
4. Memory queries can cross-reference (e.g., "what instincts relate to recent failures?")
5. Memory persists across sessions and survives `/clear`

**Complexity:** HIGH. This is the foundational piece for the entire hive learning system. Unifying fragmented JSON stores into a coherent API while maintaining backward compatibility is the core challenge.

**Dependency:** Foundation for B3-B8. Must come first.

#### B2. SQLite Colony State with FTS Recall (AAC-022 through AAC-024)

**Current state:** All colony state is stored as JSON files in `.aether/data/`. There is no database. Searching across memory requires loading individual JSON files and scanning in-memory. FTS (full-text search) does not exist.

**What's missing:**
- No SQLite database for structured queries
- No FTS5 for searching memory content (observations, instincts, findings, failures)
- No efficient recall of "all memories related to X"
- No temporal queries ("what happened in the last 5 phases?")

**Expected behavior:**
1. SQLite database at `.aether/data/colony.db` (or `.aether/data/memory.db`)
2. Tables for observations, instincts, pheromone signals, review findings, midden entries, gate results
3. FTS5 virtual table for full-text search across all memory content
4. CLI subcommands: `memory-search`, `memory-query`, `memory-stats`
5. Colony-prime uses FTS to find relevant memories for worker context injection
6. Database is a secondary index -- JSON files remain authoritative (zero migration risk)

**Complexity:** HIGH. First introduction of a non-stdlib dependency (`modernc.org/sqlite` or `mattn/go-sqlite3`). Schema design, migration strategy, and index maintenance are all new concerns. However, using SQLite as a secondary index (JSON files remain authoritative) eliminates migration risk.

**Dependency:** Requires B1 (unified memory API). Go dependency decision needed.

#### B3. Pheromone Skills (AAC-025 through AAC-027)

**Current state:** Skills are static markdown files in `.aether/skills/` with frontmatter metadata (name, category, detect patterns, roles). They are matched to workers via `skill-match` and injected via `skill-inject`. Skills are created manually (`/ant-skill-create`) or shipped with Aether. There is no automatic skill creation from colony work.

**What's missing:**
- No automatic skill creation from verified difficult tasks
- No "procedural memory" -- skills that capture how the colony solved a specific type of problem
- No usage tracking (which skills were actually helpful)
- No connection between skills and learning (instincts that become skills)

**Expected behavior:**
1. When a task is completed successfully after difficulty (multiple retries, gate failures resolved by Fixer, high complexity), the system proposes a procedural memory skill
2. Skill captures: what the problem was, how it was solved, what patterns to watch for, what actions to take
3. Skill is auto-created as a draft in `~/.aether/skills/domain/`
4. Skill is associated with the pheromone that triggered the learning
5. Future workers encountering similar patterns get the skill injected automatically
6. Skills have usage tracking: times injected, times applied, success rate

**Complexity:** MEDIUM. Skill creation from task context is bounded (transform known data into a markdown template). Usage tracking requires instrumenting `skill-inject`.

**Dependency:** Requires B1 (memory store for tracking). Extends existing skill system.

#### B4. Keeper Curator (AAC-028 through AAC-029)

**Current state:** There is a Keeper agent caste (listed in CLAUDE.md as "Preserves knowledge"). However, there is no `cmd/keeper*.go` file -- the Keeper caste has no runtime commands. The curation pipeline (`pkg/agent/curation/`) has 8 curation ants (archivist, critic, herald, janitor, librarian, nurse, scribe, sentinel) with CLI subcommands in `cmd/curation_cmds.go`, but the curation pipeline is not wired into the colony lifecycle (identified as a gap in v1.11 research).

**What's missing:**
- No Keeper runtime commands ( caste exists in agent definitions but not in Go runtime)
- No usage tracking for instincts, skills, or pheromone signals
- No stale detection (instincts that haven't been applied in N days, skills never injected)
- No archival pipeline (moving stale memories to cold storage)
- No Keeper integration with the hive learning system

**Expected behavior:**
1. Keeper tracks usage of each instinct, skill, and signal (last used, use count, success rate)
2. Keeper detects stale memories (not used in 30+ days, low confidence, superseded by newer learning)
3. Keeper proposes archival for stale memories (move to `~/.aether/eternal/` or mark as archived)
4. Keeper runs periodically (at continue, at seal) as a maintenance step
5. `/ant-status` shows memory health metrics from Keeper

**Complexity:** MEDIUM. Usage tracking requires instrumenting existing read paths. Stale detection is a query over existing timestamps. Archival follows existing `eternal` patterns.

**Dependency:** Requires B1 (unified memory). Extends existing curation pipeline.

#### B5. Learning Triggers with Evidence Rules (AAC-030)

**Current state:** Learning observations are captured via `learning-observe` (`learning.go`) with a trust scoring system (`pkg/memory/trust.go`). The promotion pipeline (`pkg/memory/promote.go`) checks if observations are eligible for instinct promotion based on trust score and observation count. However, the trigger for learning capture is manual -- agents call `learning-observe` explicitly. There is no evidence rule that gates learning creation on verified success.

**What's missing:**
- No evidence rules that prevent unverified work from creating durable learning
- No automatic learning triggers tied to build/continue outcomes
- No distinction between "observed" learning (raw note) and "verified" learning (backed by successful implementation)
- No privacy gate that prevents sensitive content from being written to learning

**Expected behavior:**
1. Learning triggers fire automatically at build-complete and continue-complete
2. Evidence rules gate learning creation: only create durable learning when:
   - Build verification passed (tests green, gates passed)
   - Implementation evidence exists (files created/modified match claims)
   - The learning is corroborated (multiple workers or phases agree)
3. Privacy gate: scan learning content for secrets, API keys, credentials before writing
4. Learning quality tier: raw observation (low) -> verified observation (medium) -> instinct (high) -> hive wisdom (highest)
5. Unverified learning stays as transient observations, never promoted to instincts

**Complexity:** MEDIUM. Evidence rules are boolean checks on existing gate/claim data. Privacy gate reuses existing sanitization patterns (`pkg/colony/sanitize.go`).

**Dependency:** Requires A1 (provenance validation for evidence). Extends `pkg/memory/`.

#### B6. Privacy/Secret Gate for Learning Writes

**Current state:** Content sanitization exists in `pkg/colony/sanitize.go` for pheromone signals (XML tag rejection, angle bracket escaping, shell injection blocking, LLM instruction override detection). Content is capped at 500 characters. However, this sanitization is specific to pheromones and is not applied to learning observations or instincts.

**What's missing:**
- No secret/key/credential detection in learning content
- No sanitization of learning observations before persistence
- No privacy gate that prevents sensitive content from entering the learning pipeline
- No scanning of file content referenced by learning (e.g., a learning that says "use the API key in config.yaml")

**Expected behavior:**
1. All learning content passes through a privacy gate before writing
2. Privacy gate detects: API keys, passwords, tokens, private keys, secrets, PII patterns
3. Detected secrets are redacted (replaced with `[REDACTED]`) or the learning is rejected
4. File references in learning content are scanned for secrets
5. Privacy gate runs at `learning-observe`, `instinct-create`, `hive-store`, and `hive-promote`

**Complexity:** LOW-MEDIUM. Pattern matching for secrets is well-understood. The infrastructure exists (`sanitize.go`). Extending it to cover learning content is additive.

**Dependency:** Standalone. Can be built independently.

#### B7. Repo Isolation with Optional Hive Promotion

**Current state:** Hive Brain (`cmd/hive.go`) stores cross-colony wisdom at `~/.aether/hive/wisdom.json` with 200-entry LRU cap. Instincts are promoted to hive at seal if confidence >= 0.8. However, there is no explicit repo isolation boundary -- any colony can read any hive entry. There is no opt-in/opt-out for hive promotion at the colony level.

**What's missing:**
- No repo isolation boundary for colony memory (all memories in a colony are equally accessible)
- No user control over which learnings get promoted to hive
- No colony-level opt-out from hive promotion ("keep my learnings local")
- No sensitivity labeling (some learnings should never leave the repo)

**Expected behavior:**
1. Colony memory is repo-scoped by default (only accessible within the colony)
2. Hive promotion is opt-in per learning or per category
3. User can set `hive_promotion: false` in colony config to disable all hive promotion
4. Sensitive learnings (containing project-specific secrets, internal URLs, etc.) are automatically excluded from hive promotion
5. Hive-promoted wisdom is generalized (repo-specific details stripped) before storage

**Complexity:** LOW. Mostly configuration and filtering logic. The promotion pipeline already exists.

**Dependency:** Extends existing hive promotion at seal.

## Feature Dependencies

```
A1. Build Provenance Validation
    |
    +---> Evidence rules for learning triggers (B5)
    |
    +---> Build-complete rejects phantom claims

A2. Oracle Confidence-Targeted Planning
    |
    +---> Iterative loop with approval gates
    |
    +---> Confidence model for research outputs

A3. Init Synthesis
    |
    +---> Synthesized brief from raw research data
    |
    +---> Charter metadata for downstream reference

A4. Recoverable Gate Failures + /ant-unblock
    |
    +---> Gate failure persistence (A5)
    |         |
    |         v
    |     gate-results.json per phase
    |
    +---> Fixer caste context (A6)
    |         |
    |         v
    |     Fixer reads gate context, investigates, fixes, verifies
    |
    +---> Smart retry (A5)

A5. Smart Gate Retry + State Persistence
    |
    +---> Extends gate.go with richer metadata
    |
    +---> Per-phase gate results
    |
    +---> Retry counts, cooldowns, soft/hard fail distinction

A6. Fixer Caste (27th Agent)
    |
    +---> Requires A4 for gate failure context
    |
    +---> New agent definition across 4 surfaces

A7. Worker Process Lifecycle
    |
    +---> Heartbeats, PID tracking, process groups
    |
    +---> Stale worker detection during execution
    |
    +---> Independent of A1-A6

B1. Colony Memory Store
    |
    +---> Foundation for all B features
    |
    +---> Unified API, budgeting, session injection

B2. SQLite + FTS Recall
    |
    +---> Requires B1 (unified memory API)
    |
    +---> Secondary index, JSON remains authoritative
    |
    +---> New Go dependency

B3. Pheromone Skills
    |
    +---> Requires B1 (usage tracking)
    |
    +---> Auto-created from verified difficult tasks
    |
    +---> Procedural memory pipeline

B4. Keeper Curator
    |
    +---> Requires B1 (unified memory for stale detection)
    |
    +---> Usage tracking, stale detection, archival
    |
    +---> Wires existing curation pipeline into lifecycle

B5. Learning Triggers with Evidence Rules
    |
    +---> Requires A1 (provenance for evidence)
    |
    +---> Auto-triggers at build/continue complete
    |
    +---> Evidence-gated promotion pipeline

B6. Privacy/Secret Gate
    |
    +---> Standalone, extends sanitize.go
    |
    +---> Applied at all learning write points

B7. Repo Isolation + Hive Promotion
    |
    +---> Extends existing hive promotion
    |
    +---> Opt-in/opt-out, sensitivity labeling
    |
    +---> Generalization before hive storage
```

## Table Stakes

Features that users expect. Missing = product feels incomplete.

| Feature | Why Expected | Complexity | Notes |
|---------|--------------|------------|-------|
| A1: Build provenance validation | Without it, builds can claim success and advance phases with zero actual changes | Medium | Core trust issue -- phantom advancement undermines the entire build/continue contract |
| A4: Recoverable gate failures | Current STOP walls with text instructions are a poor UX for an otherwise automated system | Medium | Users expect structured recovery paths, not "go figure it out" messages |
| A5: Gate state persistence | Gate results already persist in COLONY_STATE.json, but lack per-phase scoping and retry metadata | Low-Medium | Users expect gate state to survive sessions and support smart retry |
| B5: Evidence-gated learning | Learning from unverified work creates noise in instincts and hive wisdom | Medium | Without evidence gates, the learning system degrades trust in instincts |
| B6: Privacy gate for learning | Writing secrets to learning files is a security risk | Low-MEDIUM | Users expect their secrets to never appear in colony memory or hive wisdom |

## Differentiators

Features that set Aether apart. Not expected, but valued.

| Feature | Value Proposition | Complexity | Notes |
|---------|-------------------|------------|-------|
| A2: Confidence-targeted Oracle | Iterative research that quantifies confidence and targets a threshold | Medium-High | No other AI colony framework has confidence-gated research loops |
| A6: Fixer caste | Self-healing gate failures -- the colony fixes its own blockers | Medium | Autonomous recovery from gate failures is a strong differentiator |
| A7: Worker process lifecycle | Heartbeats and PID tracking make worker health observable | Medium-HIGH | Enterprise-grade process management for AI workers |
| B2: SQLite with FTS recall | Searchable memory across the entire colony history | HIGH | Full-text search over colony memory is a novel capability |
| B3: Pheromone skills | Procedural memory that auto-creates skills from verified difficult tasks | Medium | Turns colony experience into reusable knowledge automatically |
| B4: Keeper curator | Automated memory hygiene -- stale detection, archival, usage tracking | Medium | Prevents memory bloat without manual maintenance |

## Anti-Features

Features to explicitly NOT build.

| Anti-Feature | Why Avoid | What to Do Instead |
|--------------|-----------|-------------------|
| Machine learning for confidence scoring | Requires training data Aether does not have. Adds unpredictable behavior. | Use deterministic confidence model based on evidence count, source quality, cross-verification (weighted scoring similar to existing trust scoring) |
| Real-time memory sync across concurrent agents | YAGNI -- agents write during build/continue, not concurrently. Would add massive complexity. | Sequential memory writes via file locking (existing pattern) |
| Auto-promotion of all learnings to hive | Would flood hive with low-quality, repo-specific observations | Evidence-gated promotion (B5) with user opt-in (B7) |
| Web UI for memory/learning management | CLI-only for now. A web dashboard is a future consideration (listed in existing out-of-scope). | CLI subcommands for memory search, query, stats |
| Automatic Fixer execution without user awareness | Fixer making changes without the user knowing is dangerous | Fixer proposes fix, user approves before application (or Fixer runs in a dry-run mode first) |
| SQLite as primary state store | Replacing JSON with SQLite for COLONY_STATE.json breaks backward compatibility and the existing wrapper-runtime contract | SQLite as secondary index only. JSON files remain authoritative |
| Worker heartbeat over network | Workers run as local processes, not network services. Network heartbeats add unnecessary complexity. | File-based heartbeat (timestamp file or state update) |
| Learning from failed work | Failed work teaches what not to do, but the midden system already handles failure patterns | Midden tracks failures. Learning pipeline only creates durable instincts from verified successes |

## Feature Prioritization Matrix

| Feature | User Value | Implementation Cost | Priority | Dependency |
|---------|------------|---------------------|----------|------------|
| A1: Build provenance validation | CRITICAL | MEDIUM | P0 | None |
| A4: Recoverable gate failures | HIGH | MEDIUM | P0 | None |
| A5: Gate state persistence | HIGH | LOW-MEDIUM | P0 | None |
| B6: Privacy gate for learning | HIGH | LOW-MEDIUM | P0 | None |
| B5: Evidence-gated learning | HIGH | MEDIUM | P1 | A1 |
| A6: Fixer caste | HIGH | MEDIUM | P1 | A4, A5 |
| A3: Init synthesis | MEDIUM | MEDIUM | P1 | None |
| B1: Colony memory store | MEDIUM | HIGH | P1 | None |
| B7: Repo isolation + hive opt | MEDIUM | LOW | P1 | None |
| A2: Oracle confidence target | MEDIUM | MEDIUM-HIGH | P2 | None |
| B4: Keeper curator | MEDIUM | MEDIUM | P2 | B1 |
| B3: Pheromone skills | MEDIUM | MEDIUM | P2 | B1 |
| A7: Worker process lifecycle | LOW-MEDIUM | MEDIUM-HIGH | P2 | None |
| B2: SQLite + FTS recall | LOW-MEDIUM | HIGH | P3 | B1 |

## MVP Recommendation

### Phase 1 (Must Have)
1. **A1: Build provenance validation** -- Prevents phantom advancement. Core trust issue.
2. **A5: Gate state persistence** -- Extends existing gate results with retry metadata.
3. **A4: Recoverable gate failures + /ant-unblock** -- Structured recovery paths.
4. **B6: Privacy gate for learning** -- Security baseline before any learning features.

### Phase 2 (Should Have)
5. **B5: Evidence-gated learning triggers** -- Requires A1 provenance for evidence.
6. **A6: Fixer caste** -- Requires A4/A5 for gate failure context.
7. **A3: Init synthesis** -- Independent, improves onboarding UX.
8. **B7: Repo isolation + hive opt** -- Low cost, extends existing hive promotion.

### Phase 3 (Add After Validation)
9. **B1: Colony memory store** -- Foundation for remaining B features.
10. **B4: Keeper curator** -- Requires B1.
11. **A2: Oracle confidence target** -- Independent but complex.
12. **B3: Pheromone skills** -- Requires B1.

### Defer
- **A7: Worker process lifecycle** -- High cost, moderate value. Heartbeats are nice-to-have until workers run long enough to stall. Current wall-clock timeout is sufficient for most use cases.
- **B2: SQLite + FTS recall** -- High cost, requires new dependency. JSON file scanning is adequate until memory volumes justify a database. Defer until B1 proves the memory API and volume justifies the index.

## Existing System Integration Points

| Feature | Integration Point | File | What Changes |
|---------|-------------------|------|-------------|
| A1: Provenance validation | `runCodexContinueGates()` | `cmd/codex_continue.go:2361` | Add provenance gate check that validates claims against filesystem |
| A1: Provenance validation | Build complete | `cmd/codex_build_finalize.go` | Add filesystem verification of claimed files, reject empty builds |
| A4: Recoverable failures | Gate failure output | `cmd/gate.go:490-531` | Add structured recovery actions, persist to `gate-results.json` |
| A4: /ant-unblock | New command | New `cmd/unblock.go` | Read gate failure state, offer retry/fix paths |
| A5: Gate persistence | `gateResultsWrite()` | `cmd/gate.go:548-571` | Add retry count, cooldown, soft/hard fail fields |
| A6: Fixer caste | Agent definitions | `.claude/agents/ant/aether-fixer.md` (new) | New agent across 4 surfaces |
| A3: Init synthesis | `init-research` output | `cmd/init_research.go` | Add synthesis step that produces approval-ready brief |
| B5: Learning triggers | Build/continue complete | `cmd/codex_build_finalize.go`, `cmd/codex_continue_finalize.go` | Add evidence-gated learning capture |
| B6: Privacy gate | `learning-observe`, `instinct-create` | `cmd/learning.go`, `cmd/instinct.go` | Add privacy scanning before write |
| B7: Hive opt | `hive-promote` at seal | `cmd/hive.go` | Add colony-level opt-out, sensitivity filtering |
| B1: Memory store | New unified API | New `cmd/memory_store.go` | Unified read/write/query for all memory types |
| B4: Keeper curator | New commands | New `cmd/keeper.go` | Usage tracking, stale detection, archival |
| B3: Pheromone skills | `skill-inject` | `cmd/skills.go` | Auto-create from verified difficult tasks, usage tracking |

## Sources

- `cmd/codex_build.go` -- Build dispatch, claims, manifest (lines 22-199)
- `cmd/codex_build_finalize.go` -- Build complete flow, filesystem fallback discovery (lines 194-457)
- `cmd/codex_continue.go` -- Continue gates, provenance check, gate report (lines 60-2429)
- `cmd/gate.go` -- Gate checking, recovery templates, skip logic, persistence (lines 1-700)
- `cmd/compatibility_cmds.go` -- Oracle RALF loop, autopilot (lines 52-140)
- `cmd/recover_scanner.go` -- 7 stuck-state detectors (lines 1-100)
- `cmd/codex_worker_cleanup.go` -- Stale worker termination (lines 1-32)
- `cmd/oracle_process_unix.go` -- Process tree termination (lines 1-63)
- `cmd/init_research.go` -- Codebase scanning, charter generation (598 lines)
- `cmd/init_cmd.go` -- Colony initialization (lines 1-80)
- `cmd/hive.go` -- Hive Brain wisdom storage, LRU cap, promotion (lines 1-80)
- `cmd/learning.go` -- Observation capture, trust scoring (lines 1-66)
- `cmd/instinct.go` -- Instinct CRUD, duplicate detection, confidence reinforcement (lines 1-80)
- `cmd/memory_health.go` -- Memory health aggregation from multiple JSON files (lines 1-80)
- `pkg/memory/` -- Observation, pipeline, promote, queen, trust, consolidate packages
- `pkg/colony/colony.go` -- ColonyState struct, GateResultEntry, Charter, PendingSuggestion (lines 220-302)
- `pkg/colony/instincts.go` -- InstinctEntry, InstinctsFile types (lines 1-35)
- `pkg/colony/sanitize.go` -- Content sanitization for pheromone signals
- `pkg/storage/storage.go` -- Atomic write, file locking, JSON persistence
- `.planning/PROJECT.md` -- v1.13 milestone requirements (AAC-001 through AAC-031, REC-LOOP-01)

---
*Feature research for: Aether v1.13 Recovery Hardening & Hive Learning*
*Researched: 2026-05-01*
