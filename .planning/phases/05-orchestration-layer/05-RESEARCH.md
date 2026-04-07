# Phase 5: Orchestration Layer - Research

**Researched:** 2026-04-07
**Domain:** Go-based agent orchestration, task decomposition, runtime caste assignment
**Confidence:** HIGH

## Summary

The phase orchestrator extends the existing agent system with a centralized coordinator that decomposes phases into tasks, assigns specialist agents at runtime via the event bus, and validates outputs before marking phases complete. The curation orchestrator in `pkg/agent/curation/orchestrator.go` is the proven template -- the phase orchestrator follows the same Agent interface pattern but swaps sequential step execution for hybrid concurrent dispatch (plan upfront, execute via event bus).

The implementation adds new types to `pkg/agent/` (PhaseOrchestrator, TaskGraph, TaskRouter), extends `pkg/colony/colony.go` with orchestrator state fields, creates three new cobra commands, and migrates the autopilot from its separate `autopilot/state.json` to unified `COLONY_STATE.json`.

**Primary recommendation:** Model PhaseOrchestrator on the curation orchestrator's Agent interface implementation, use `errgroup` (already a dependency via pool.go) for concurrent task dispatch with dependency ordering, and implement TaskRouter as a keyword-matching heuristic that reuses the existing Caste enum.

## User Constraints (from 05-CONTEXT.md)

### Locked Decisions (D-01 through D-11)
- **D-01:** Hybrid model -- orchestrator plans tasks upfront (imperative decomposition), then dispatches them via the event bus for concurrent execution
- **D-02:** PhaseOrchestrator is a new type in `pkg/agent/` alongside Pool, Registry. Implements Agent interface
- **D-03:** Task decomposition happens at phase start -- orchestrator reads phase definition, creates task graph, dispatches as dependencies resolve
- **D-04:** Runtime matching -- caste assignment at dispatch time, not plan time
- **D-05:** Routing should reuse or extend existing skill-match scoring where possible
- **D-06:** Task descriptions include type hints (e.g., `[test]`, `[implement]`, `[research]`) that guide routing
- **D-07:** Context boundaries -- each agent runs in a goroutine with scoped context, cancellation via ctx.Done()
- **D-08:** Agents receive only their assigned task scope (task description, relevant files, success criteria). No sibling tasks or full phase plan
- **D-09:** Unified state -- orchestrator state integrates into COLONY_STATE.json. Autopilot's separate state.json migrated
- **D-10:** Three cobra commands: orchestrator-decompose, orchestrator-assign, orchestrator-status
- **D-11:** All commands produce JSON output via `outputOK()`

### Claude's Discretion
- Exact task graph data structure (slice vs map vs custom graph)
- Whether to add an `orchestrator-run` command that triggers a full phase execution
- How task type hints are specified in plan files (struct tags, comment prefixes, separate field)
- Whether to add orchestration events to the event bus topics
- Exact JSON schema for orchestrator-status output

### Deferred Ideas (OUT OF SCOPE)
- Branch/worktree integration with orchestrator -- Phase 6 scope
- Dynamic agent spawning based on task complexity (ORCH-08, v2)
- Graph-based task routing with dependency resolution (ORCH-09, v2)

## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| ORCH-01 | PhaseOrchestrator decomposes each phase into tasks and assigns specialist agents | TaskGraph + TaskRouter patterns documented below |
| ORCH-02 | TaskRouter maps task descriptions to agent castes | Keyword-matching heuristic + type hint parsing |
| ORCH-03 | Agents isolated per task -- no agent acts outside assigned scope | Goroutine + scoped context pattern (D-07/D-08) |
| ORCH-04 | All agent outputs handed back to Orchestrator before next phase | errgroup.Wait() + result channel pattern |
| ORCH-05 | Orchestrator validates outputs against success criteria | Validation pattern from task success_criteria field |
| ORCH-06 | Agent-role contracts are explicit, versioned, reusable | TaskContract struct with version field |
| ORCH-07 | Orchestrator maintains full visibility of system state | orchestrator-status command + state in COLONY_STATE.json |

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| golang.org/x/sync/errgroup | v0.16.0 | Concurrent goroutine dispatch with error propagation | Already used in pool.go; SetLimit for bounded concurrency |
| encoding/json | stdlib | Task graph serialization to COLONY_STATE.json | No external dependency needed |
| context | stdlib | Goroutine cancellation and scoped isolation | Already used throughout agent package |
| github.com/spf13/cobra | v1.10.2 | CLI commands for decompose/assign/status | Already integrated |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| github.com/jedib0t/go-pretty/v6 | v6.7.8 | Table rendering for orchestrator-status | Human-readable status display |
| github.com/tidwall/gjson | v1.18.0 | JSON path queries for task hint parsing | If complex task description parsing needed |

**Version verification:** All dependencies are already in go.mod. No new packages required.

## Architecture Patterns

### Recommended Project Structure
```
pkg/agent/
├── agent.go              # Existing: Agent interface, Caste enum, Registry
├── pool.go               # Existing: event bus dispatch with bounded concurrency
├── spawn_tree.go         # Existing: agent lifecycle tracking
├── builder.go            # Existing: BuilderAgent example
├── orchestrator.go       # NEW: PhaseOrchestrator, TaskGraph, TaskRouter
├── orchestrator_test.go  # NEW: tests
├── task_graph.go         # NEW (optional): if task graph logic is complex enough to warrant separate file
└── curation/             # Existing: curation orchestrator template
    └── orchestrator.go

cmd/
├── orchestrator.go       # NEW: orchestrator-decompose, orchestrator-assign, orchestrator-status commands
├── orchestrator_test.go  # NEW: command tests
└── autopilot.go          # MODIFY: migrate state reads from autopilot/state.json to COLONY_STATE.json

pkg/colony/
└── colony.go             # MODIFY: add OrchestratorState field to ColonyState
```

### Pattern 1: PhaseOrchestrator (Agent Interface)
**What:** A type implementing the Agent interface (Name, Caste, Triggers, Execute) that manages task decomposition and dispatch. Follows the curation orchestrator template exactly.
**When to use:** Always -- this is the core orchestration entry point.
**Key design:** The PhaseOrchestrator's Execute() method triggers a full phase run: decompose phase into tasks, build dependency graph, dispatch tasks in dependency order via the event bus, collect results, validate against success criteria.

```go
// Source: pkg/agent/curation/orchestrator.go (proven template)
type PhaseOrchestrator struct {
    store   *storage.Store
    bus     *events.Bus
    mu      sync.Mutex
    // Task graph and results populated at execution time
    graph   *TaskGraph
    results map[string]*TaskResult
}

func (o *PhaseOrchestrator) Name() string { return "phase-orchestrator" }
func (o *PhaseOrchestrator) Caste() Caste { return CasteRouteSetter } // or new Caste?
func (o *PhaseOrchestrator) Triggers() []Trigger {
    return []Trigger{{Topic: "phase.start"}, {Topic: "orchestrator.run"}}
}
func (o *PhaseOrchestrator) Execute(ctx context.Context, event events.Event) error {
    // 1. Extract phase from event payload
    // 2. Build task graph
    // 3. Dispatch tasks in dependency order
    // 4. Collect and validate results
    // 5. Update COLONY_STATE.json
    return o.Run(ctx, phase)
}
```

**Difference from curation orchestrator:**
- Curation: sequential, fixed 8 steps, sentinel abort
- Phase: concurrent dispatch via errgroup, dependency-based ordering, result validation

### Pattern 2: TaskGraph (Dependency-Aware Dispatch)
**What:** Data structure representing tasks and their dependencies. Tasks with resolved dependencies are dispatched first; dependent tasks wait.
**When to use:** Every phase execution -- this is the core scheduling structure.
**Design:** Use a map-based adjacency list. Track in-degree (unresolved dependency count) per task. Tasks with in-degree 0 are ready to dispatch.

```go
// Source: based on D-01 (hybrid model) and D-03 (decomposition at phase start)
type TaskGraph struct {
    tasks    map[string]*TaskNode       // task ID -> node
    edges    map[string][]string         // task ID -> list of dependent task IDs
    inDegree map[string]int             // task ID -> unresolved dependency count
}

type TaskNode struct {
    ID          string
    Goal        string
    Caste       Caste        // resolved at dispatch time by TaskRouter
    Status      string       // pending, in_progress, completed, failed
    DependsOn   []string
    Criteria    []string     // success criteria for validation
    TypeHint    string       // parsed from task description, e.g., "implement", "test", "research"
}

type TaskResult struct {
    TaskID    string
    AgentName string
    Caste     Caste
    Success   bool
    Output    string
    Error     string
    Duration  time.Duration
}
```

**Dispatch algorithm (Kahn's algorithm variant):**
1. Build in-degree map from DependsOn fields
2. Enqueue all tasks with in-degree 0
3. Dispatch ready tasks concurrently via errgroup (bounded by pool maxG)
4. When a task completes, decrement dependents' in-degree; enqueue newly-ready tasks
5. Repeat until all tasks complete or a failure blocks remaining tasks

### Pattern 3: TaskRouter (Runtime Caste Assignment)
**What:** Maps task descriptions to agent castes at dispatch time using keyword matching and type hints.
**When to use:** Every task before dispatch -- the router decides which caste executes it.
**Design:** Two-pass approach -- first check explicit type hints (`[test]`, `[implement]`), then fall back to keyword matching.

```go
// Source: D-04 (runtime matching), D-05 (extend skill-match), D-06 (type hints)
func RouteTask(task TaskNode) Caste {
    // Pass 1: explicit type hint
    if hint := parseTypeHint(task.Goal); hint != "" {
        return hintToCaste(hint)
    }
    // Pass 2: keyword matching
    return keywordMatch(task.Goal)
}

// Type hint patterns: [implement] -> CasteBuilder, [test] -> CasteWatcher,
//                     [research] -> CasteScout, [validate] -> CasteWatcher,
//                     [analyze] -> CasteScout, [chaos] -> CasteChaos (if needed)

func hintToCaste(hint string) Caste {
    switch strings.ToLower(hint) {
    case "implement", "build", "code", "fix":
        return CasteBuilder
    case "test", "verify", "validate", "check":
        return CasteWatcher
    case "research", "investigate", "explore", "analyze":
        return CasteScout
    case "design", "architect", "plan":
        return CasteArchitect
    case "chaos", "stress", "resilience":
        return CasteChaos  // if chaos caste exists, otherwise CasteWatcher
    default:
        return CasteBuilder // safe default
    }
}
```

**Heuristic fallback (keyword matching):**
- "test", "verify", "assert" -> CasteWatcher
- "research", "investigate", "find", "discover" -> CasteScout
- "implement", "create", "build", "add", "write", "fix" -> CasteBuilder
- "design", "architect", "structure" -> CasteArchitect
- Default: CasteBuilder

[VERIFIED: pkg/agent/agent.go] -- All 9 castes defined; CasteChaos is not currently in the enum. Chaos is handled as a niche agent. The router should only use castes that exist: Builder, Watcher, Scout, Oracle, Curator, Architect, RouteSetter, Colonizer, Archaeologist.

**Note on missing CasteChaos:** The enum at `pkg/agent/agent.go:17-27` does not include a CasteChaos constant. The router should map "chaos" type hints to CasteWatcher (closest role for resilience/edge-case testing). This is a Claude's discretion item for the planner.

### Pattern 4: Agent Isolation via Context Boundaries
**What:** Each dispatched agent runs in its own goroutine with a scoped context. The orchestrator passes only the task-relevant data, not the full phase plan.
**When to use:** Every task dispatch -- this is the isolation model from D-07 and D-08.
**Design:** Follow pool.go's errgroup pattern but with per-task context scoping.

```go
// Source: pkg/agent/pool.go dispatch pattern + D-07 (context boundaries)
g, ctx := errgroup.WithContext(ctx)
g.SetLimit(maxG)

for _, task := range readyTasks {
    task := task // capture
    g.Go(func() error {
        taskCtx, cancel := context.WithTimeout(ctx, taskTimeout)
        defer cancel()

        // Scoped context: only task description, relevant files, success criteria
        scopedEvent := buildScopedEvent(task)

        // Route to appropriate agent via registry
        agents := registry.Match(taskEvent.Topic)
        if len(agents) == 0 {
            return fmt.Errorf("no agent matched for task %s (caste: %s)", task.ID, task.Caste)
        }

        err := agents[0].Execute(taskCtx, scopedEvent)
        o.recordResult(task.ID, agents[0].Name(), task.Caste, err)
        return err
    })
}
```

### Anti-Patterns to Avoid
- **Sequential dispatch of independent tasks:** Use errgroup for concurrent execution. Independent tasks MUST run in parallel.
- **Passing full phase plan to agents:** Violates D-08. Each agent sees only its own task scope.
- **Hardcoding caste names in task descriptions:** Use the TaskRouter's keyword matching. Task descriptions should describe what to do, not who does it.
- **Mutable shared state without mutex:** The orchestrator's results map is written from multiple goroutines. Use sync.Mutex or sync.Map.
- **Ignoring context cancellation:** Every goroutine must check ctx.Done() regularly.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Concurrent goroutine dispatch | Manual goroutine + WaitGroup | `errgroup.WithContext` + `SetLimit` | Error propagation, bounded concurrency, context cancellation -- all built in |
| Dependency graph traversal | Custom topological sort from scratch | Kahn's algorithm on map-based adjacency list | Simple, well-understood, O(V+E). No library needed for ~10 tasks per phase |
| JSON serialization of state | Custom marshal/unmarshal | `encoding/json` struct tags | Already used throughout the codebase; COLONY_STATE.json is JSON |
| CLI output formatting | fmt.Println + manual padding | `outputOK()` for JSON, `go-pretty` for tables | Established patterns in cmd/status.go and other commands |
| Task routing | ML/heuristic scoring | Keyword matching + type hints | ~20 keywords cover 95% of tasks; no training data needed |

**Key insight:** The dependency graph for phase tasks is small (typically 3-10 tasks per phase). Kahn's algorithm on a map is sufficient -- no need for a graph library.

## Common Pitfalls

### Pitfall 1: Deadlock from Circular Dependencies
**What goes wrong:** Task A depends on B, B depends on A. Neither can dispatch. Orchestrator hangs forever.
**Why it happens:** Plan files with circular task dependencies (route-setter error or user mistake).
**How to avoid:** Validate the task graph for cycles before dispatch. Add a cycle detection pass using DFS with three-color marking (white/gray/black) or simply check that all tasks eventually reach in-degree 0 during Kahn's algorithm. If the queue empties before all tasks are processed, a cycle exists.
**Warning signs:** orchestrator-status shows tasks stuck in "pending" with no active agents.

### Pitfall 2: Goroutine Leak from Uncancelled Contexts
**What goes wrong:** If a parent context is cancelled but child goroutines don't check ctx.Done(), they leak.
**Why it happens:** Agent Execute() methods that do blocking I/O without context checks.
**How to avoid:** Always use context.WithTimeout for per-task contexts. Ensure every agent implementation checks ctx.Done() between work phases (see builder.go pattern at lines 49, 79, 91, 104, 118).
**Warning signs:** Rising goroutine count visible in runtime metrics; orchestrator-status shows agents as "active" long after phase completes.

### Pitfall 3: Race Conditions in Result Collection
**What goes wrong:** Multiple goroutines write to the orchestrator's results map concurrently, causing data corruption.
**Why it happens:** Go maps are not safe for concurrent writes.
**How to avoid:** Use sync.Mutex around all writes to the results map. The curation orchestrator uses this pattern (mu sync.Mutex at line 69 of orchestrator.go). Alternatively, use a channel to collect results from a single reader goroutine.
**Warning signs:** Flaky test failures, "concurrent map writes" panics.

### Pitfall 4: Dual State Drift (Autopilot Migration)
**What goes wrong:** After migrating autopilot state to COLONY_STATE.json, the old autopilot/state.json still exists. Both files get out of sync.
**Why it happens:** Migration doesn't remove or redirect the old file. Some code paths still read from autopilot/state.json.
**How to avoid:** (a) Remove all references to autopilotStatePath constant. (b) Add orchestrator fields directly to ColonyState. (c) Add a migration check: if autopilot/state.json exists but COLONY_STATE.json has orchestrator fields, warn and suggest deleting the old file. (d) The autopilot commands should read from COLONY_STATE.json exclusively.
**Warning signs:** Two different "current phase" values depending on which command is queried.

### Pitfall 5: TaskRouter Defaults to Wrong Caste
**What goes wrong:** A task like "review the test coverage" routes to CasteWatcher (keyword: "test") when the user intended CasteScout (keyword: "review").
**Why it happens:** Keyword matching is order-dependent and ambiguous.
**How to avoid:** (a) Task type hints in plan files (`[review]` -> scout) are the primary mechanism. (b) Keyword matching is a fallback only. (c) Log the routing decision so the user can see why a caste was chosen. (d) Allow override via task-level caste hints in the plan.
**Warning signs:** Wrong specialist type executing tasks; user reports "why did the builder do the research task?"

## Code Examples

### Implementing Agent Interface (from curation orchestrator)
```go
// Source: pkg/agent/curation/orchestrator.go:48-70, 107-125
type PhaseOrchestrator struct {
    store   *storage.Store
    bus     *events.Bus
    mu      sync.Mutex
}

func (o *PhaseOrchestrator) Name() string { return "phase-orchestrator" }
func (o *PhaseOrchestrator) Caste() Caste { return CasteRouteSetter }
func (o *PhaseOrchestrator) Triggers() []Trigger {
    return []Trigger{
        {Topic: "phase.start"},
        {Topic: "orchestrator.run"},
    }
}
func (o *PhaseOrchestrator) Execute(ctx context.Context, event events.Event) error {
    // Extract phase from event, run orchestration
    return o.Run(ctx, phase)
}
```

### errgroup Concurrent Dispatch (from pool.go)
```go
// Source: pkg/agent/pool.go:230-285
g, ctx := errgroup.WithContext(ctx)
g.SetLimit(maxG)

for _, task := range readyTasks {
    task := task // capture loop variable
    g.Go(func() error {
        return o.dispatchTask(ctx, task)
    })
}
return g.Wait()
```

### State Persistence Pattern (from autopilot.go)
```go
// Source: cmd/autopilot.go:56-66
if err := store.SaveJSON(statePath, state); err != nil {
    outputError(2, fmt.Sprintf("failed to save: %v", err), nil)
    return nil
}
outputOK(map[string]interface{}{...})
```

### Task Graph Building
```go
// Source: D-03 (decomposition at phase start)
func BuildTaskGraph(phase colony.Phase) (*TaskGraph, error) {
    g := &TaskGraph{
        tasks:    make(map[string]*TaskNode),
        edges:    make(map[string][]string),
        inDegree: make(map[string]int),
    }
    for _, task := range phase.Tasks {
        id := taskID(task)
        node := &TaskNode{
            ID:        id,
            Goal:      task.Goal,
            DependsOn: task.DependsOn,
            Criteria:  task.SuccessCriteria,
            Status:    TaskPending,
        }
        // Parse type hint from goal text
        node.TypeHint = parseTypeHint(task.Goal)
        node.Caste = RouteTask(node)

        g.tasks[id] = node
        g.inDegree[id] = len(task.DependsOn)
    }
    // Build edges (dependency -> dependents)
    for _, node := range g.tasks {
        for _, dep := range node.DependsOn {
            g.edges[dep] = append(g.edges[dep], node.ID)
        }
    }
    // Validate no cycles
    if hasCycle(g) {
        return nil, fmt.Errorf("task graph has circular dependencies")
    }
    return g, nil
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Shell-based orchestration scripts | Go-based PhaseOrchestrator type | This phase | Type-safe, testable, concurrent |
| Autopilot separate state.json | Unified COLONY_STATE.json | This phase | Single source of truth for all state |
| Manual agent assignment | Runtime TaskRouter with keyword matching | This phase | Automated caste selection |
| Sequential task execution | errgroup concurrent dispatch with dependency ordering | This phase | Parallel execution of independent tasks |

**Deprecated/outdated:**
- `autopilot/state.json` — migrates to COLONY_STATE.json. Old file should be cleaned up or warned about.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | CasteChaos constant does not exist in the Caste enum | Architecture: Pattern 3 | If chaos testing is needed, the router must map to an existing caste. Verified: `pkg/agent/agent.go:17-27` has no CasteChaos. |
| A2 | Phase tasks from the route-setter already include `DependsOn` and `SuccessCriteria` fields | Architecture: Pattern 2 | ColonyState Task struct has both fields (`pkg/colony/colony.go:133-141`). Verified. |
| A3 | The route-setter generates tasks with sufficient description text for keyword matching | Architecture: Pattern 3 | Risk: if tasks have generic descriptions like "do step 2", keyword matching fails. Type hints mitigate this. |
| A4 | Autopilot state migration is backward-compatible (old autopilot/state.json can be read once for migration) | Pitfall 4 | Need to verify: no code in cmd/autopilot.go writes to the path during the migration window. |
| A5 | The event bus topic "phase.start" is not yet used by other agents | Architecture: Pattern 1 | Need to verify: grep for existing topic subscriptions. If occupied, use "orchestrator.phase.start" instead. |

## Open Questions

1. **Should PhaseOrchestrator use a new Caste or existing CasteRouteSetter?**
   - What we know: The orchestrator is a type of route-setter (planning + dispatch). CasteRouteSetter is already defined.
   - What's unclear: Whether adding a new `CasteOrchestrator` constant makes the role clearer in status output.
   - Recommendation: Use CasteRouteSetter for now. The orchestrator IS a route-setter that also executes. A new caste can be added later without breaking anything.

2. **What happens when an agent task fails mid-phase?**
   - What we know: Curation orchestrator continues after non-sentinel failures (line 188: `continue`).
   - What's unclear: Should phase orchestrator fail-fast on any task failure, or continue sibling tasks and report partial completion?
   - Recommendation: Continue siblings (they may be independent). Mark failed tasks. The orchestrator reports partial success with failed task details. The autopilot or user decides whether to retry.

3. **How does orchestrator interact with the existing Pool?**
   - What we know: Pool dispatches events to matching agents. Orchestrator also dispatches agents.
   - What's unclear: Does the orchestrator bypass the Pool (direct agent execution) or publish events that the Pool dispatches?
   - Recommendation: Direct execution via registry. The orchestrator IS the dispatcher for phase tasks. The Pool handles ambient event dispatch (consolidation, learning events). Two dispatch paths coexist -- this matches the D-01 hybrid model.

4. **Whether orchestrator-run command is needed beyond the existing /ant:build**
   - What we know: /ant:build triggers phase execution via shell commands. orchestrator-run would be a Go-level equivalent.
   - What's unclear: Whether this replaces /ant:build or complements it.
   - Recommendation: Defer orchestrator-run. The three commands (decompose, assign, status) provide visibility. Actual execution is triggered by existing build flow. Add orchestrator-run in a follow-up if needed.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go | All implementation | ✓ | 1.26.1 | — |
| errgroup | Concurrent dispatch | ✓ | v0.16.0 (go.mod) | — |
| cobra | CLI commands | ✓ | v1.10.2 (go.mod) | — |
| go-pretty | Status table rendering | ✓ | v6.7.8 (go.mod) | — |
| storage.Store | State persistence | ✓ | in pkg/storage | — |
| events.Bus | Event dispatch | ✓ | in pkg/events | — |

**Missing dependencies with no fallback:** None -- all required packages are already in go.mod.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go testing (stdlib) |
| Config file | none -- Go convention |
| Quick run command | `go test ./pkg/agent/ -run TestPhaseOrch -v` |
| Full suite command | `go test ./... -race` |

### Phase Requirements to Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| ORCH-01 | PhaseOrchestrator decomposes phase into tasks | unit | `go test ./pkg/agent/ -run TestBuildTaskGraph` | ❌ Wave 0 |
| ORCH-02 | TaskRouter maps descriptions to castes | unit | `go test ./pkg/agent/ -run TestRouteTask` | ❌ Wave 0 |
| ORCH-03 | Agents isolated per task (context scoping) | unit | `go test ./pkg/agent/ -run TestTaskIsolation` | ❌ Wave 0 |
| ORCH-04 | All outputs collected before phase advance | unit | `go test ./pkg/agent/ -run TestCollectResults` | ❌ Wave 0 |
| ORCH-05 | Output validated against success criteria | unit | `go test ./pkg/agent/ -run TestValidateOutput` | ❌ Wave 0 |
| ORCH-06 | TaskContract explicit and versioned | unit | `go test ./pkg/agent/ -run TestTaskContract` | ❌ Wave 0 |
| ORCH-07 | orchestrator-status shows full state | unit | `go test ./cmd/ -run TestOrchestratorStatus` | ❌ Wave 0 |

### Sampling Rate
- **Per task commit:** `go test ./pkg/agent/ -run TestPhaseOrch -count=1`
- **Per wave merge:** `go test ./... -race`
- **Phase gate:** `go test ./... -race` green before verification

### Wave 0 Gaps
- [ ] `pkg/agent/orchestrator_test.go` -- covers ORCH-01, ORCH-03, ORCH-04
- [ ] `pkg/agent/task_router_test.go` -- covers ORCH-02
- [ ] `cmd/orchestrator_test.go` -- covers ORCH-07
- [ ] `pkg/colony/colony_test.go` -- extend for orchestrator state fields

*(None of these test files exist yet -- they must be created in Wave 0 or alongside implementation)*

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | no | CLI tool, not web-facing |
| V3 Session Management | no | No sessions -- stateless commands |
| V4 Access Control | no | Single-user CLI tool |
| V5 Input Validation | yes | Task descriptions parsed for type hints; reject injection patterns in task goals |
| V6 Cryptography | no | No cryptographic operations |

### Known Threat Patterns for Go CLI Orchestration

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Task description injection (malicious task goals) | Tampering | Sanitize type hint parsing; treat task goals as opaque strings for routing |
| State file corruption via concurrent writes | Tampering | AtomicWrite pattern already in storage.Store |
| Goroutine exhaustion (malicious task graph with 1000s of tasks) | Denial of Service | Bound task count in graph building; errgroup.SetLimit caps concurrency |

## Sources

### Primary (HIGH confidence)
- `pkg/agent/curation/orchestrator.go` -- Orchestrator struct, Agent interface, Run(), StepResult pattern
- `pkg/agent/agent.go` -- Agent interface, Caste enum (9 castes), Registry, Match()
- `pkg/agent/pool.go` -- errgroup dispatch, bounded concurrency, StreamManager integration
- `pkg/agent/spawn_tree.go` -- SpawnEntry tracking, lifecycle management
- `pkg/colony/colony.go` -- ColonyState struct, Phase/Task structs with DependsOn/SuccessCriteria
- `cmd/autopilot.go` -- autopilotState struct, state.json path, CRUD commands
- `cmd/status.go` -- Dashboard rendering, depth/granularity display patterns
- `pkg/events/bus.go` -- Publish/Subscribe, TopicMatch, JSONL persistence

### Secondary (MEDIUM confidence)
- `cmd/colony_cmds.go` -- colony-depth get/set command pair (template for orchestrator commands)
- `.planning/phases/04-planning-granularity-controls/04-CONTEXT.md` -- Enum pattern, persistence, autopilot integration

### Tertiary (LOW confidence)
- Kahn's algorithm for topological sort -- standard CS algorithm, no specific source needed

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH -- all dependencies already in go.mod, no new packages needed
- Architecture: HIGH -- curation orchestrator is a proven template in the same codebase
- Pitfalls: MEDIUM -- based on Go concurrency patterns (well-known) and project-specific dual-state issue (inferred from code)

**Research date:** 2026-04-07
**Valid until:** 2026-05-07 (stable phase scope, no fast-moving dependencies)
