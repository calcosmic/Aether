# Architecture Research: v1.13 Recovery Hardening & Hive Learning

**Domain:** Build/continue gate hardening, confidence-targeted Oracle loop, init synthesis, worker lifecycle tracking, hive learning layer with SQLite
**Researched:** 2026-05-01
**Overall confidence:** HIGH (based on direct source code analysis of 316 cmd/*.go files, 12 pkg/ packages, and existing colony state types)

## Executive Summary

v1.13 adds three distinct feature streams that touch well-defined integration surfaces in the existing Go runtime. Recovery hardening modifies two critical paths (build-finalize and continue-finalize) to validate provenance before allowing phase advancement. Hive learning adds a new `pkg/hive/` package backed by SQLite with FTS5, connected to the existing `pkg/memory/` pipeline via event bus subscriptions. Worker lifecycle tracking extends the existing `pkg/codex/process_tracker.go` with heartbeat monitoring and stale cleanup.

The architecture is additive, not disruptive. No existing data structures need breaking changes -- all new fields use `omitempty`. The SQLite colony.db coexists with JSON files because it serves a different purpose: JSON files hold colony state (authoritative, human-readable, git-trackable), while SQLite holds learned procedural memory (searchable, accumulated, privacy-gated).

**Key risk:** The build-provenance validation (AAC-001, AAC-002) must reject phantom builds without breaking legitimate partial-success scenarios. The existing `assessCodexContinue()` function in `cmd/codex_continue.go` already distinguishes partial success from full failure -- provenance validation layers on top of this, not replacing it.

## Integration Architecture

### Component Map

```
EXISTING                                           NEW (v1.13)
========                                           ============

RECOVERY HARDENING
cmd/codex_build_finalize.go ........... cmd/provenance.go (new)
cmd/codex_continue_finalize.go ......... cmd/provenance.go (new)
cmd/codex_continue.go .................. cmd/provenance.go (new -- provenance gate)
cmd/gate.go ............................ cmd/gate.go (gate-results.json persistence)
cmd/circuit_breaker.go ................ cmd/circuit_breaker.go (REC-LOOP-01 inheritance)
pkg/colony/colony.go .................. pkg/colony/colony.go (BuildProvenance field)

ORACLE LOOP
cmd/oracle_loop.go .................... cmd/oracle_loop.go (confidence target param)
cmd/oracle_loop.go .................... cmd/oracle_loop.go (iterative refinement)
.aether/oracle/state.json .............. (already has TargetConfidence field)

INIT SYNTHESIS
cmd/init_cmd.go ....................... cmd/init_cmd.go (synthesis subcommand)
cmd/init_research.go .................. cmd/init_research.go (brief assembly)
cmd/colony_prime_context.go ........... (reads Charter from COLONY_STATE.json)

GATE RECOVERY
cmd/gate.go ............................ cmd/gate.go (gate-results.json)
cmd/codex_continue_finalize.go ......... cmd/unblock_cmd.go (new -- /ant-unblock)
cmd/codex_build.go .................... (Fixer caste dispatch)

WORKER LIFECYCLE
pkg/codex/process_tracker.go ........... pkg/codex/process_tracker.go (heartbeat fields)
pkg/codex/process_tracker.go ........... cmd/heartbeat_monitor.go (new)
cmd/worker_cleanup_signal_*.go ......... cmd/heartbeat_monitor.go (stale cleanup)

HIVE LEARNING
pkg/memory/pipeline.go ................. pkg/hive/ (new package)
pkg/events/bus.go ..................... pkg/hive/store.go (event subscriber)
cmd/hive.go ........................... pkg/hive/ (refactored from cmd/)
pkg/storage/storage.go ................. pkg/hive/store.go (SQLite alongside Store)
cmd/colony_prime_context.go ........... pkg/hive/recall.go (FTS5 retrieval)
cmd/codex_build.go .................... pkg/hive/hooks.go (learning triggers)
cmd/codex_continue_finalize.go ......... pkg/hive/hooks.go (learning triggers)
```

---

## Question 1: Build Provenance Validation in Build-Finalize and Continue-Verify

### Where It Hooks

**Build-finalize path** (`cmd/codex_build_finalize.go`):

The provenance gate inserts between line 203 (`applyCodexBuildState`) and line 220 (`buildCodexBuildManifest`), which is the existing state-mutation-to-manifest-write window. Currently this window has no validation that the build actually produced meaningful output. The hook point is:

```
runCodexBuildFinalize():
  1. Load completion data (existing)
  2. Merge external build results (existing)
  3. >>> NEW: validateBuildProvenance(dispatches, claims, phase) <<<
  4. applyCodexBuildState (existing)
  5. write manifest (existing)
  6. atomic commit (existing)
```

**Continue-verify path** (`cmd/codex_continue_finalize.go`):

The provenance gate inserts at line 163-166, between `assessCodexContinue()` and `runCodexContinueGates()`. Currently `assessCodexContinue()` produces a `codexContinueAssessment` that already has `PartialSuccess` and `Tasks` fields. The provenance validator enriches this assessment with filesystem-grounded evidence:

```
runCodexContinueFinalize():
  1. validateExternalContinueState (existing)
  2. runCodexContinueVerificationSnapshot (existing)
  3. assessCodexContinue (existing)
  4. >>> NEW: validateContinueProvenance(assessment, manifest, phase) <<<
  5. runCodexContinueGates (existing -- receives enriched assessment)
```

### New Component: `cmd/provenance.go`

```go
// BuildProvenance holds evidence that a build actually occurred.
type BuildProvenance struct {
    BuildPhase       int       `json:"build_phase"`
    DispatchesTotal  int       `json:"dispatches_total"`
    DispatchesPassed int       `json:"dispatches_passed"`
    FilesCreated     int       `json:"files_created"`
    FilesModified    int       `json:"files_modified"`
    TestsWritten     int       `json:"tests_written"`
    GitDiffFiles     int       `json:"git_diff_files"`
    ZeroModFlag      bool      `json:"zero_modification"`
    AllFailedFlag    bool      `json:"all_failed"`
    ValidatedAt      time.Time `json:"validated_at"`
}

// validateBuildProvenance checks that a build produced real output.
// Returns error if build should be rejected (AAC-001).
func validateBuildProvenance(dispatches []codexBuildDispatch, claims codexBuildClaims, phase colony.Phase) (BuildProvenance, error)

// validateContinueProvenance checks that continue claims match reality.
// Returns enriched assessment with provenance evidence (AAC-002).
func validateContinueProvenance(assessment codexContinueAssessment, manifest codexContinueManifest, phase colony.Phase) (BuildProvenance, error)
```

### Data Flow

```
Build-Finalize:
  codexExternalBuildCompletion
    -> mergeExternalBuildResults()
    -> validateBuildProvenance()  // NEW: rejects zero-mod or all-failed builds
    -> applyCodexBuildState()
    -> BuildProvenance stored in manifest metadata

Continue-Finalize:
  codexContinueAssessment (from assessCodexContinue)
    -> validateContinueProvenance()  // NEW: cross-checks claims vs filesystem
    -> codexContinueGateReport (gates receive provenance data)
    -> gate-results.json includes provenance_validation check
```

### State Changes

Add to `colony.ColonyState`:
```go
BuildProvenance *BuildProvenance `json:"build_provenance,omitempty"`
```

This field is populated during build-finalize and cleared during phase advance. The `omitempty` ensures backward compatibility with existing colonies.

---

## Question 2: Confidence-Targeted Oracle Loop

### Where It Integrates

The Oracle loop already has confidence tracking. `oracleStateFile` has `TargetConfidence` and `OverallConfidence` fields. `oracleReadyForCompletion()` already checks `state.OverallConfidence >= state.TargetConfidence`. The `oracleDepthLevels` map already maps depth names to `(MaxIterations, TargetConfidence)` pairs.

The integration point is in `startOracleCompatibility()` (line 298-371). Currently the depth is resolved from the `--depth` flag or defaults to "balanced" (85% target). The new feature adds:

1. **User-settable confidence target** via `--confidence-target` flag (e.g., `--confidence-target 95`)
2. **Iterative refinement** when confidence is below target but all questions are answered -- the loop re-opens questions for deeper investigation rather than stopping
3. **Phase-aware depth** -- Oracle respects colony `PlanningDepth` from COLONY_STATE.json

### Changes to `cmd/oracle_loop.go`

```go
// In startOracleCompatibility():
// NEW: --confidence-target flag overrides depth-based default
confidenceTarget, _ := cmd.Flags().GetInt("confidence-target")
if confidenceTarget > 0 {
    depthCfg.TargetConfidence = confidenceTarget
}

// In oracleReadyForCompletion():
// NEW: if all answered but below target, re-open lowest-confidence questions
func oracleReadyForCompletion(plan oraclePlanFile, state oracleStateFile) bool {
    if state.OverallConfidence >= state.TargetConfidence {
        return true
    }
    if oracleAllQuestionsAnswered(plan) && state.OverallConfidence >= state.TargetConfidence-10 {
        return true // close enough
    }
    return false
}

// NEW: oracleReopenForDeepening re-opens answered questions below target
func oracleReopenForDeepening(plan *oraclePlanFile, state oracleStateFile) int {
    reopened := 0
    for i := range plan.Questions {
        if plan.Questions[i].Status == "answered" && plan.Questions[i].Confidence < state.TargetConfidence-10 {
            plan.Questions[i].Status = "partial"
            plan.Questions[i].Confidence = plan.Questions[i].Confidence // preserve
            reopened++
        }
    }
    return reopened
}
```

### Where in the Loop

```
runOracleLoop():
  for state.Iteration < state.MaxIterations {
    ...existing iteration logic...
    oracleReadyForCompletion(plan, state)  // ENHANCED: lower threshold
    >>> NEW: if all answered but below target, oracleReopenForDeepening() <<<
    ...existing max_iterations_reached...
  }
```

### Integration with Colony State

Oracle depth selection reads from colony state:
```go
// In startOracleCompatibility():
state, _ := loadActiveColonyState()
if state.VerificationDepth != "" {
    switch colony.NormalizeVerificationDepth(state.VerificationDepth) {
    case colony.VerificationDepthHeavy:
        depthCfg.MaxIterations = 8
        depthCfg.TargetConfidence = 95
    case colony.VerificationDepthLight:
        depthCfg.MaxIterations = 2
        depthCfg.TargetConfidence = 60
    }
}
```

---

## Question 3: Init Synthesis

### Where It Fits

Init synthesis is a new subcommand `aether init-synthesize` that assembles an approval-ready launch brief from codebase scouting data. It fits **between** `aether init-research` and `aether init` in the ceremony flow:

```
Current flow:
  aether init-research --goal "..."   (scouting)
  aether init "..."                   (colony creation)

New flow:
  aether init-research --goal "..."   (scouting -- unchanged)
  aether init-synthesize --goal "..." (NEW: brief assembly)
  aether init "..." --charter-json "..." (colony creation -- already supports --charter-json)
```

### Relationship to Colony-Prime

Colony-prime (`cmd/colony_prime_context.go`) already reads the `Charter` field from `COLONY_STATE.json` and injects it into worker context as a section. The synthesis output feeds into the charter, which colony-prime already handles. No changes to colony-prime are needed.

### New Component

```go
// cmd/init_synthesize.go
type LaunchBrief struct {
    Goal            string            `json:"goal"`
    Vision          string            `json:"vision"`
    TechStack       string            `json:"tech_stack"`
    KeyRisks        []string          `json:"key_risks"`
    Constraints     []string          `json:"constraints"`
    SuggestedPhases int               `json:"suggested_phases"`
    Complexity      string            `json:"complexity"` // low/medium/high
    Charter         colony.Charter    `json:"charter"`
}
```

The synthesis reads from `init-research` output (which already produces tech stack, governance, pheromone suggestions, directory analysis) and assembles them into a structured charter.

---

## Question 4: gate-results.json and Flag/Blocker System

### Current State

Gate results are currently stored **inline** in `COLONY_STATE.json` as `GateResults []GateResultEntry`. The `gateResultsWrite()` function (cmd/gate.go:552) does an atomic upsert-merge into the state file. The `gateResultsRead()` function reads from state.

The PRD mentions `gate-results.json` as a separate file. This is a **separation concern** -- gate results are currently tightly coupled to colony state mutations.

### Recommended Architecture

Keep gate results in `COLONY_STATE.json` for atomic consistency (the continue-finalize path already does `store.UpdateJSONAtomically`), but add a **mirror write** to `gate-results.json` for independent inspection:

```go
// In runCodexContinueFinalize(), after gate run (line 166):
gates := runCodexContinueGates(...)

// Existing: write to COLONY_STATE.json (atomic)
_ = gateResultsWrite(gateResultEntries)

// NEW: write standalone copy for /ant-status and /ant-unblock
if err := store.SaveJSON("gate-results.json", gates); err != nil {
    // non-blocking -- gate results in state are authoritative
}
```

### Interaction with Loop Safety Circuit Breaker

The circuit breaker (`cmd/circuit_breaker.go`) already emits `emitLoopBreakEvent()` which calls `emitLifecycleCeremony()`. Gate results interact with the circuit breaker through `shouldSkipGate()` (line 536): previously passed gates are skipped on re-run.

For REC-LOOP-01 (all new gates inherit loop safety):
- Every new gate check function must call `shouldSkipGate(priorResults, gateName)` before executing
- Every gate failure must call `emitLoopBreakEvent()` if it is a repeated failure
- The `gateRecoveryTemplates` map (line 473) needs entries for new gate types

### /ant-unblock Command

```go
// cmd/unblock_cmd.go
var unblockCmd = &cobra.Command{
    Use:   "unblock [gate-name]",
    Short: "Acknowledge and clear a gate blocker",
}

// Reads gate-results.json, marks the named gate as acknowledged,
// clears the GateResultEntry from COLONY_STATE.json,
// and returns recovery instructions.
```

This interacts with the flag system (`pkg/colony/flags.go`) by creating a `FlagEntry` when a gate blocks, and resolving it when `/ant-unblock` runs.

---

## Question 5: Fixer Caste

### Where It Fits in the Caste System

The existing caste system has 26 castes (25 agents + Porter added in v1.10). The Fixer is the 27th caste. It fits into the dispatch manifest as a new caste value:

```go
// In pkg/codex/dispatch.go or equivalent:
// Existing castes: builder, watcher, scout, oracle, chaos, architect, ...
// NEW: "fixer"

// In cmd/codex_visuals.go casteColorMap:
"fixer": "\033[38;5;208m",  // orange -- distinct from builder yellow
```

### Dispatch Integration

The Fixer caste is dispatched by the continue-finalize path when gates fail. In `runCodexContinueFinalize()`, after gates fail (line 193-199):

```go
if !gates.Passed {
    // NEW: if recovery template suggests fixable issue, dispatch Fixer
    if fixableGates := identifyFixableGates(gates); len(fixableGates) > 0 {
        fixerDispatch := codexContinueWorkerFlowStep{
            Stage:   "recovery",
            Caste:   "fixer",
            Name:    deterministicAntName("fixer", phase.Name),
            Task:    fmt.Sprintf("Fix gate failures: %s", strings.Join(fixableGates, ", ")),
            Status:  "pending",
        }
        // Add to worker flow but do not auto-execute -- user runs /ant-continue again
    }
    // ...existing blocked continue path...
}
```

### Agent Definition

New files needed (mirroring existing pattern):
- `.claude/agents/ant/aether-fixer.md` -- Claude Code agent
- `.opencode/agents/aether-fixer.md` -- OpenCode agent
- `.codex/agents/aether-fixer.toml` -- Codex agent
- `.aether/agents-claude/aether-fixer.md` -- packaging mirror

---

## Question 6: Worker Heartbeat / Process Tracking

### Existing Foundation

`pkg/codex/process_tracker.go` already has:
- `TrackedProcess` struct with PID, WorkerName, Caste, Platform, Root, SpawnedAt
- `GlobalProcessTracker()` singleton
- `TrackProcess()` / `UntrackProcess()` / `KillProcess()` / `KillAll()`
- `DetectStaleWorkers()` -- finds processes still running but not tracked by current process
- `CleanupStaleWorkers()` -- detects and kills stale workers
- `isKnownWorkerProcess()` -- checks if a PID is a codex/claude/opencode process
- `workerProcessRegistryRel = ".aether/data/worker-processes.json"` -- persistent registry

### Integration Points

The heartbeat system extends `TrackedProcess`:

```go
type TrackedProcess struct {
    PID           int       `json:"pid"`
    WorkerName    string    `json:"worker_name,omitempty"`
    Caste         string    `json:"caste,omitempty"`
    Platform      string    `json:"platform,omitempty"`
    Root          string    `json:"root,omitempty"`
    SpawnedAt     time.Time `json:"spawned_at"`
    // NEW fields:
    LastHeartbeat time.Time `json:"last_heartbeat,omitempty"`
    TaskID        string    `json:"task_id,omitempty"`
    Phase         int       `json:"phase,omitempty"`
    Status        string    `json:"status,omitempty"` // "running", "completed", "failed", "timed_out"
}
```

### Heartbeat Monitor

```go
// cmd/heartbeat_monitor.go
type HeartbeatMonitor struct {
    tracker    *codex.ProcessTracker
    interval   time.Duration
    timeout    time.Duration // 2x interval for warning, 4x for stale
    onStale    func(process codex.TrackedProcess) // callback for stale detection
}
```

The monitor runs as a goroutine during build waves. It checks `LastHeartbeat` on each tracked process and emits events via the ceremony bus when workers go stale. The observer chain in `pkg/codex/worker.go` calls `TrackProcess()` at spawn and `UntrackProcess()` at completion -- the heartbeat fields are updated by the worker wrapper environment variables (`AETHER_WORKER_NAME`, `AETHER_WORKER_CASTE`) which already exist.

### Integration with Existing Worker Flow

```
cmd/codex_build.go:
  executeCodexBuildDispatches()
    -> for each dispatch:
      invoker.Invoke(ctx, config)
        -> TrackProcess(pid, process)  // EXISTING
        -> heartbeatMonitor.Watch(pid) // NEW: register for monitoring
        -> ...worker runs...
        -> UntrackProcess(pid)          // EXISTING
        -> heartbeatMonitor.Unwatch(pid) // NEW: stop monitoring
```

---

## Question 7: Hive Learning Layer (pkg/hive/)

### New Package: `pkg/hive/`

This is the largest new component. It introduces a `pkg/hive/` package with SQLite-backed storage, connected to the existing memory pipeline.

### Package Structure

```
pkg/hive/
  store.go       -- SQLite database management, schema, CRUD
  recall.go      -- FTS5 full-text search and recall
  hooks.go       -- Learning trigger points (phase-end, seal, difficulty detection)
  privacy.go     -- Privacy gate that intercepts all write paths
  skill.go       -- Auto-created skills from verified difficult tasks
  curator.go     -- Keeper curator logic for wisdom maintenance
```

### SQLite Schema

```sql
CREATE TABLE IF NOT EXISTS memories (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    repo_id     TEXT NOT NULL,          -- repo root hash for isolation
    domain      TEXT NOT NULL,          -- go, react, general, etc.
    trigger     TEXT NOT NULL,          -- what situation triggers this
    action      TEXT NOT NULL,          -- what to do
    evidence    TEXT,                   -- what evidence supports this
    confidence  REAL DEFAULT 0.5,
    source      TEXT,                   -- phase-N, seal, manual
    created_at  TEXT DEFAULT (datetime('now')),
    accessed_at TEXT DEFAULT (datetime('now')),
    access_count INTEGER DEFAULT 0,
    verified    INTEGER DEFAULT 0,      -- 1 = survived verification
    skill_id    TEXT                    -- link to auto-created skill
);

CREATE VIRTUAL TABLE IF NOT EXISTS memories_fts USING fts5(
    trigger, action, evidence, domain,
    content=memories,
    content_rowid=id
);

-- Triggers to keep FTS in sync
CREATE TRIGGER memories_ai AFTER INSERT ON memories BEGIN
    INSERT INTO memories_fts(rowid, trigger, action, evidence, domain)
    VALUES (new.id, new.trigger, new.action, new.evidence, new.domain);
END;

CREATE TRIGGER memories_ad AFTER DELETE ON memories BEGIN
    INSERT INTO memories_fts(memories_fts, rowid, trigger, action, evidence, domain)
    VALUES ('delete', old.id, old.trigger, old.action, old.evidence, old.domain);
END;

CREATE TRIGGER memories_au AFTER UPDATE ON memories BEGIN
    INSERT INTO memories_fts(memories_fts, rowid, trigger, action, evidence, domain)
    VALUES ('delete', old.id, old.trigger, old.action, old.evidence, old.domain);
    INSERT INTO memories_fts(rowid, trigger, action, evidence, domain)
    VALUES (new.id, new.trigger, new.action, new.evidence, new.domain);
END;
```

### Connection to Existing Memory Pipeline

The existing pipeline is in `pkg/memory/pipeline.go`:

```
Observation -> Trust Score -> Event Bus -> Auto-Promote -> Instinct -> QUEEN.md -> Hive
```

The hive learning layer connects at **two points**:

1. **Write path** (learning hooks): New trigger points in `pkg/hive/hooks.go` subscribe to event bus topics and write to SQLite when colony work is verified:
   ```
   events.Bus.Subscribe("phase.completed")    -> hive.StoreMemory()
   events.Bus.Subscribe("seal.completed")     -> hive.StoreMemory()
   events.Bus.Subscribe("gate.passed")        -> hive.StoreMemory()
   events.Bus.Subscribe("learning.observe")   -> hive.StoreMemory() (with privacy gate)
   ```

2. **Read path** (recall): Colony-prime context assembly (`cmd/colony_prime_context.go`) adds a new section that queries the hive via FTS5:
   ```
   colonyPrimeSections:
     ...existing sections...
     hive_wisdom: hive.Recall(domain, query, limit)  // NEW section, low priority
   ```

### Privacy Gate

The privacy gate intercepts all write paths before data reaches SQLite:

```go
// pkg/hive/privacy.go
type PrivacyGate struct {
    blockedPatterns []string  // e.g., file paths, API keys, user names
    maxEntrySize   int       // character limit per memory entry
}

func (g *PrivacyGate) Screen(memory Memory) (Memory, error) {
    // 1. Check for blocked patterns (file paths with usernames, API keys, etc.)
    // 2. Strip repo-specific identifiers
    // 3. Truncate oversized entries
    // 4. Sanitize prompt injection patterns (reuse pkg/colony/sanitize.go patterns)
    return memory, nil
}
```

All write methods in `pkg/hive/store.go` call `privacyGate.Screen()` before INSERT.

### Refactoring cmd/hive.go

The existing `cmd/hive.go` implements Hive Brain as a flat-file system (`~/.aether/hive/wisdom.json`). The new `pkg/hive/` package does NOT replace this -- it adds a repo-scoped layer alongside it:

| Layer | Location | Scope | Storage |
|-------|----------|-------|---------|
| Hive Brain (existing) | `~/.aether/hive/wisdom.json` | Cross-colony | JSON file, 200-entry cap |
| Hive Learning (new) | `.aether/data/colony.db` | Per-repo | SQLite with FTS5 |

The two layers interact: Hive Learning feeds into Hive Brain at seal time (confidence >= 0.8), same as the existing instinct promotion path.

---

## Question 8: SQLite colony.db Coexistence with JSON Files

### Architecture Decision

SQLite does NOT replace JSON files. They serve different purposes:

| Concern | JSON Files (.aether/data/) | SQLite (.aether/data/colony.db) |
|---------|---------------------------|--------------------------------|
| Colony state | COLONY_STATE.json | Not stored |
| Phase plan | COLONY_STATE.json (plan field) | Not stored |
| Pheromones | pheromones.json | Not stored |
| Gate results | COLONY_STATE.json (gate_results field) | Not stored |
| Session | session.json | Not stored |
| **Learned memories** | instincts (in COLONY_STATE.json) | memories table (searchable) |
| **FTS recall** | Not possible | memories_fts (FTS5) |
| **Auto-skills** | Not stored | skills table |

### Coexistence Pattern

```go
// pkg/hive/store.go
func Open(dbPath string) (*Store, error) {
    // dbPath = ".aether/data/colony.db"
    // Uses a SINGLE connection with WAL mode for concurrent reads
    db, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_busy_timeout=5000")
    if err != nil {
        return nil, err
    }
    return &Store{db: db, privacy: NewPrivacyGate()}, nil
}
```

The SQLite store uses `pkg/storage/Store` for file-level locking on the `.db` file itself, ensuring no concurrent writes from multiple Aether processes. The existing `FileLocker` in `pkg/storage/lock.go` handles this.

### File Locking Integration

```go
func (s *Store) WriteMemory(ctx context.Context, mem Memory) error {
    // Acquire file lock via storage.Store
    return s.fileStore.UpdateFile("colony.db.lock", func(existing []byte) ([]byte, error) {
        // Actual SQLite write happens inside the lock
        return s.insertMemory(ctx, mem)
    })
}
```

Wait -- this is wrong. SQLite has its own locking via WAL mode. The `FileLocker` from `pkg/storage/` is for JSON file atomicity. For SQLite, we rely on:

1. `PRAGMA journal_mode=WAL` -- allows concurrent reads during writes
2. `PRAGMA busy_timeout=5000` -- retries on lock contention
3. Single-writer pattern -- only the main Aether process writes; worker subprocesses never touch colony.db directly

---

## Question 9: Learning Hooks and Event Bus

### Phase Lifecycle Events

The event bus (`pkg/events/bus.go`) already supports topic-pattern subscriptions. Learning hooks subscribe to lifecycle topics:

```go
// pkg/hive/hooks.go
func (h *HookManager) Start(ctx context.Context, bus *events.Bus) error {
    // Phase completed -- extract learnings from phase work
    ch1, err := bus.Subscribe("lifecycle.phase_completed")
    // Seal completed -- promote high-confidence memories
    ch2, err := bus.Subscribe("lifecycle.seal_completed")
    // Gate passed -- record what worked
    ch3, err := bus.Subscribe("lifecycle.gate_passed")
    // Learning observation -- capture with privacy gate
    ch4, err := bus.Subscribe("learning.observe")
    // ...
}
```

### Where Events Are Emitted

Currently, lifecycle events are emitted via `emitLifecycleCeremony()` in `cmd/ceremony_emitter.go`. These events go to the ceremony event bus (separate from the `pkg/events.Bus`). The learning hooks need to connect to the **same bus** that the `pkg/memory/pipeline.go` uses.

The ceremony emitter uses `events.CeremonyTopic*` constants. The memory pipeline subscribes to `"learning.observe"` and `"consolidation.*"`. These are the same bus instance -- the ceremony topics and learning topics coexist.

### New Event Topics Needed

```go
// In pkg/events/ceremony.go (or a new events/topics.go)
const (
    TopicPhaseCompleted  = "lifecycle.phase_completed"
    TopicSealCompleted   = "lifecycle.seal_completed"
    TopicGatePassed      = "lifecycle.gate_passed"
    TopicGateFailed      = "lifecycle.gate_failed"
    TopicDifficultyDetected = "learning.difficulty_detected"
)
```

### Evidence Rules

Learning hooks only fire when evidence rules are met (AAC-024):

```go
type EvidenceRule struct {
    MinPhaseNumber    int     // Don't learn from phase 1 (too noisy)
    MinConfidence     float64 // 0.7 minimum for auto-capture
    RequireVerification bool  // Only learn from verified (gate-passed) phases
    RequireTests      bool    // Only learn if tests existed before the phase
    MaxEntriesPerPhase int    // Cap per-phase learning to prevent noise
}
```

---

## Question 10: Privacy Gate Interception

### All Write Paths

The privacy gate intercepts every write to `pkg/hive/store.go`:

```go
// pkg/hive/store.go
type Store struct {
    db       *sql.DB
    privacy  *PrivacyGate
    repoID   string  // repo root hash for isolation
}

func (s *Store) WriteMemory(ctx context.Context, mem Memory) (int64, error) {
    // 1. Apply privacy gate
    screened, err := s.privacy.Screen(mem)
    if err != nil {
        return 0, fmt.Errorf("privacy gate rejected: %w", err)
    }

    // 2. Set repo isolation
    screened.RepoID = s.repoID

    // 3. Write to SQLite
    result, err := s.db.ExecContext(ctx, insertMemorySQL, ...)
    return result.LastInsertId()
}

func (s *Store) BulkWriteMemories(ctx context.Context, mems []Memory) ([]int64, error) {
    var ids []int64
    for _, mem := range mems {
        id, err := s.WriteMemory(ctx, mem) // each goes through privacy gate
        if err != nil {
            continue // skip rejected entries, don't block batch
        }
        ids = append(ids, id)
    }
    return ids, nil
}
```

### Privacy Gate Rules

```go
// pkg/hive/privacy.go
type PrivacyGate struct {
    maxEntrySize    int      // default: 2000 chars
    blockedPatterns []string // regex patterns for sensitive content
}

func NewPrivacyGate() *PrivacyGate {
    return &PrivacyGate{
        maxEntrySize: 2000,
        blockedPatterns: []string{
            `api[_-]?key`,           // API keys
            `secret[_-]?key`,        // Secret keys
            `password`,              // Passwords
            `Bearer\s+[A-Za-z0-9]+`, // Bearer tokens
            `-----BEGIN.*PRIVATE`,   // Private keys
            `\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Z|a-z]{2,}\b`, // Email addresses
            `/home/[^/]+/`,         // Home directory paths with usernames
            `/Users/[^/]+/`,        // macOS user paths
        },
    }
}

func (g *PrivacyGate) Screen(mem Memory) (Memory, error) {
    // 1. Check each field against blocked patterns
    // 2. Replace home directory paths with generic equivalents
    // 3. Truncate fields exceeding maxEntrySize
    // 4. Reuse pkg/colony/sanitize.go patterns for prompt injection detection
    // 5. Return error if critical content is blocked (e.g., API key detected)
    return mem, nil
}
```

---

## Suggested Build Order

Based on dependency analysis of the integration points:

### Phase Group A: Recovery Hardening (AAC-001 through AAC-007)
**Order:** Provenance validation first, then gate recovery, then /ant-unblock

1. **Provenance validation** (`cmd/provenance.go`) -- no dependencies on other new features
2. **gate-results.json mirror** -- extends existing `gate.go`, no new deps
3. **/ant-unblock command** (`cmd/unblock_cmd.go`) -- depends on gate-results.json
4. **Fixer caste** -- depends on gate recovery templates being in place
5. **REC-LOOP-01 inheritance** -- applies loop safety to all new gate paths

### Phase Group B: Oracle & Init Enhancements (AAC-003, AAC-004)
**Order:** Independent of Group A, can build in parallel

1. **Confidence-targeted Oracle** -- modifies existing `cmd/oracle_loop.go`
2. **Init synthesis** (`cmd/init_synthesize.go`) -- independent new command

### Phase Group C: Worker Lifecycle (AAC-014 through AAC-017)
**Order:** Depends on nothing new, but should complete before Group D

1. **Heartbeat fields** -- extend `TrackedProcess` (backward compatible)
2. **Heartbeat monitor** (`cmd/heartbeat_monitor.go`) -- new goroutine
3. **Stale cleanup integration** -- extend existing `CleanupStaleWorkers()`

### Phase Group D: Hive Learning (AAC-019 through AAC-031)
**Order:** Largest feature group, builds on everything above

1. **pkg/hive/store.go** -- SQLite schema and CRUD (no external deps)
2. **pkg/hive/privacy.go** -- Privacy gate (needed before any writes)
3. **pkg/hive/recall.go** -- FTS5 recall (needed for colony-prime integration)
4. **pkg/hive/hooks.go** -- Event bus subscriptions (needs events from Group A)
5. **pkg/hive/skill.go** -- Auto-created skills (needs privacy gate)
6. **pkg/hive/curator.go** -- Keeper curator (needs recall and store)
7. **Colony-prime integration** -- add hive_wisdom section to context assembly

### Phase Group E: Full System Hardening (AAC-018)
**Order:** Last, validates everything works together

1. **E2E flow validation** -- test complete build-verify-advance with all new gates
2. **Cross-platform consistency** -- verify Claude, OpenCode, Codex all work

---

## Anti-Patterns to Avoid

### 1. Don't Put Colony State in SQLite
Colony state (COLONY_STATE.json) is authoritative, human-readable, and git-trackable. SQLite is for accumulated learning that benefits from search. Mixing them creates a nightmare for debugging and recovery.

### 2. Don't Block the Build Pipeline on Hive Writes
Hive learning is fire-and-forget. If SQLite is locked or the write fails, log and continue. Never let a learning write failure block phase advancement.

### 3. Don't Duplicate the Existing Gate Check Logic
`runCodexContinueGates()` already has a rich gate system. Provenance validation adds a NEW check, not a replacement. The provenance check is one more entry in the gate report, not a parallel gate system.

### 4. Don't Make the Privacy Gate Overly Aggressive
The privacy gate should catch API keys and credentials, not block legitimate learning about error patterns that happen to contain file paths. Use blocklist patterns, not allowlist.

### 5. Don't Break Existing Oracle State Format
`oracleStateFile` has a `Depth` field (added in v1.12). New fields must use `omitempty`. The existing `oracleReadyForCompletion()` function must remain the primary completion check -- confidence-targeted refinement is additive.

---

## Scalability Considerations

| Concern | Current (JSON files) | With SQLite |
|---------|---------------------|-------------|
| Learning entries | Limited by JSON file size / memory array | SQLite handles millions of rows |
| FTS search | Not possible (linear scan of JSON) | FTS5 with ranking in <10ms |
| Cross-repo learning | Hive Brain (200-entry cap) | Per-repo DB + Hive Brain promotion |
| Privacy enforcement | Manual (pheromone sanitization) | Automated privacy gate on all writes |
| Concurrent access | File locking (adequate for JSON) | WAL mode + busy timeout |

---

## Sources

- Direct source code analysis: 316 cmd/*.go files, 12 pkg/ packages (2026-05-01)
- `pkg/colony/colony.go` -- ColonyState struct, state machine, depth types
- `cmd/codex_build_finalize.go` -- Build-finalize path (lines 144-257)
- `cmd/codex_continue_finalize.go` -- Continue-finalize path (lines 115-226)
- `cmd/oracle_loop.go` -- Oracle RALF loop with confidence tracking
- `cmd/gate.go` -- Gate check system with recovery templates
- `cmd/circuit_breaker.go` -- Circuit breaker with per-worker tracking
- `pkg/events/bus.go` -- Event bus with pub/sub and JSONL persistence
- `pkg/memory/pipeline.go` -- Wisdom pipeline (observe -> promote -> queen)
- `pkg/codex/process_tracker.go` -- Worker process tracking and cleanup
- `cmd/colony_prime_context.go` -- Colony-prime context assembly
- `cmd/hive.go` -- Existing Hive Brain (cross-colony wisdom in JSON)
- `pkg/storage/storage.go` -- Atomic file operations and file locking
- `cmd/init_cmd.go` -- Colony initialization with charter support
- PROJECT.md -- v1.13 requirements (AAC-001 through AAC-031, REC-LOOP-01)
