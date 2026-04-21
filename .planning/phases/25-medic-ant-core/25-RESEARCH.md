# Phase 25: Medic Ant Core - Research

**Gathered:** 2026-04-21
**Status:** Complete

## Research Summary

Comprehensive analysis of Aether colony data files, existing health check patterns, trace format, wrapper structure, and version detection. The Medic will be the colony's first true structural health checker — existing patrol/status commands are metric dashboards only.

## Data File Inventory

### Primary Data Files (with Go structs)

| File | Go Struct | Source Location |
|------|-----------|-----------------|
| COLONY_STATE.json | `colony.ColonyState` | `pkg/colony/colony.go:151` |
| pheromones.json | `colony.PheromoneFile` | `pkg/colony/pheromones.go:39` |
| session.json | `colony.SessionFile` | `pkg/colony/session.go:4` |
| midden/midden.json | `colony.MiddenFile` | `pkg/colony/midden.go:48` |
| instincts.json | `colony.InstinctsFile` | `pkg/colony/instincts.go:31` |
| learning-observations.json | `colony.LearningFile` | `pkg/colony/learning.go:19` |
| assumptions.json | `colony.AssumptionsFile` | `pkg/colony/assumptions.go:40` |
| pending-decisions.json | `colony.FlagsFile` | `pkg/colony/flags.go:18` |
| constraints.json | `colony.ConstraintsFile` (empty) | `pkg/colony/constraints.go:5` |

### Secondary Data Files (no Go structs — raw JSON)

| File | Notes |
|------|-------|
| workers.json | Template only, no Go struct |
| spawn-runs.json | Handled via `cmd/spawn_runs.go` |
| last-build-result.json | No struct definition found |
| colony-registry.json | Hub-level, no struct |
| instinct-graph.json | Edges array, no struct |
| queen-wisdom.json | No struct |
| cost-ledger.json | No struct |
| pheromone-branch-export.json | No struct |
| last-build-claims.json | No struct |
| pr-context-cache.json | No struct |

### JSONL Files

| File | Go Struct | Format |
|------|-----------|--------|
| trace.jsonl | `trace.TraceEntry` | JSONL, 50MB rotation |
| event-bus.jsonl | `events.Event` | JSONL, TTL-based cleanup |
| spawn-tree.txt | `agent.SpawnEntry` | TSV format |

### Cache Files (expected, not corruption)

- `.cache_COLONY_STATE.json`
- `.cache_instincts.json`

## Schema Details

### ColonyState (pkg/colony/colony.go:151-178)

```
version: string ("3.0")
goal: *string (required)
scope: ColonyScope ("project"|"meta")
colony_name: *string
colony_version: int (starts at 1)
state: State (IDLE|READY|EXECUTING|BUILT|COMPLETED)
current_phase: int
session_id: *string
initialized_at: *time.Time
build_started_at: *time.Time
plan: Plan
memory: Memory
errors: Errors
signals: []Signal (DEPRECATED — migrated to pheromones.json)
graveyards: []Graveyard
events: []string (pipe-delimited: "timestamp|type|source|description")
parallel_mode: ParallelMode ("in-repo"|"worktree")
worktrees: []WorktreeEntry
milestone: string
paused: bool
```

State machine transitions in `pkg/colony/state_machine.go:490-496`.

### PheromoneSignal (pkg/colony/pheromones.go:20-36)

```
id: string
type: PheromoneType (FOCUS|REDIRECT|FEEDBACK)
priority: string
source: string
created_at: string
expires_at: *string (optional)
active: bool
strength: *float64 (optional)
reason: *string (optional)
content: json.RawMessage (supports {"text": "..."} nested objects)
content_hash: *string (SHA-256 dedup)
reinforcement_count: *int
archived_at: *string
tags: []PheromoneTag (optional)
scope: *PheromoneScope (optional)
```

Content sanitized via `pkg/colony/sanitize.go` (500-char max, XML tag rejection, prompt injection rejection, shell injection rejection, angle bracket escaping).

### SessionFile (pkg/colony/session.go:4-18)

```
session_id: string
started_at: string
last_command: string
last_command_at: string
colony_goal: string
current_phase: int
current_milestone: string
suggested_next: string
context_cleared: bool
baseline_commit: string
resumed_at: *string (optional)
active_todos: []string
summary: string
```

### TraceEntry (pkg/trace/trace.go:29-37)

```
id: string (trc_{unix}_{hex})
run_id: string
timestamp: string (RFC3339)
level: TraceLevel (state|phase|pheromone|error|recovery|intervention|token|artifact)
topic: string (pattern: "state.transition", "phase.{status}", "error.add", etc.)
payload: map[string]interface{} (optional)
source: string
```

### MiddenEntry (pkg/colony/midden.go)

```
id, timestamp, category, source, message
reviewed: bool
acknowledged: bool
acknowledged_at, acknowledge_reason
tags: []string
```

Lives at `.aether/data/midden/midden.json` (subdirectory — different from all other data files).

## Existing Health Check Patterns

### Patrol (colony-vital-signs) — cmd/memory_health.go:53-164

Metric dashboard only. Computes health score 0-100:
- instinctCount > 0: +10
- signalCount > 0: +10
- errorCount == 0: +15
- completedPhases > 0: +15

Labels: Critical (<20), Struggling (20-39), Stable (40-59), Healthy (60-79), Thriving (80+).

**Does NOT validate JSON structure, file integrity, or schema correctness.**

### Status Dashboard — cmd/status.go:56-293

Loads COLONY_STATE.json, computes progress bars, loads memory health, pheromone summary, flag counts.
**Silently returns empty on errors** — never reports file problems.

### State Loading — cmd/state_load.go:14-35

`loadActiveColonyState()` normalizes legacy states and repairs missing plans from artifacts.
Key normalization: PAUSED→READY+paused, PLANNED→READY, SEALED→COMPLETED, ENTOMBED→IDLE.

### Memory Health — cmd/memory_health.go:25-85

Aggregates from 3 files: instincts.json, learning-observations.json, midden.json.
Counts active/archived instincts, pending promotions, recent failures.

## Validation Patterns to Leverage

### Existing Valid() Methods

| Type | Method | Valid Values |
|------|--------|-------------|
| PlanGranularity | `Valid()` | sprint, milestone, quarter, major |
| ParallelMode | `Valid()` | in-repo, worktree |
| ColonyScope | `Valid()` | project, meta |
| AssumptionConfidence | `Valid()` | confident, likely, unclear |

### Storage Layer

`pkg/storage/storage.go`:
- `AtomicWrite` — validates JSON before writing via temp file + rename
- `ReadJSONL` — skips malformed lines, logs to stderr
- File locking via `pkg/storage/lock.go`

### Content Sanitization

`pkg/colony/sanitize.go`:
- `SanitizeSignalContent()` — 500-char max, XML rejection, injection detection
- `DetectPromptIntegrityFindings()` — prompt injection patterns

## Wrapper File Counts

| Surface | Count | Path |
|---------|-------|------|
| YAML commands | 49 | `.aether/commands/*.yaml` |
| Claude commands | 49 | `.claude/commands/ant/*.md` |
| OpenCode commands | 49 | `.opencode/commands/ant/*.md` |
| Codex agents (TOML) | 24 | `.codex/agents/*.toml` |
| Claude agents | 24 | `.claude/agents/ant/*.md` |
| OpenCode agents | 24 | `.opencode/agents/*.md` |
| Claude mirror | 24 | `.aether/agents-claude/*.md` |
| Codex mirror (TOML) | 24 | `.aether/agents-codex/*.toml` |
| Colony skills | 10 | `.aether/skills/colony/` |
| Domain skills | 18 | `.aether/skills/domain/` |
| **Total skills** | **28** | |

All counts verified. Agent mirrors match 1:1. Command wrappers match 1:1 across Claude/OpenCode/YAML.

## Version Detection

- Version stored in `cmd/version.go` + `cmd/root.go` via ldflags (defaults to "0.0.0-dev")
- `resolveVersion()` tries git describe --tags --abbrev=0 as fallback
- No `.aether/version.json` file — version is purely git tags / ldflags
- COLONY_STATE.json `version` field is "3.0" (string, not semver)
- No version migration framework — only legacy state name normalization

## Gotchas

1. **Dual instinct schemas** — Instinct (simple, in ColonyState.Memory.Instincts) and InstinctEntry (rich, in standalone instincts.json). Medic must check both.
2. **Constraints is a ghost** — Go struct is empty `{}`, template has content, `countConstraints()` returns hardcoded 0. Flag if file has content Go never reads.
3. **Deprecated signals in COLONY_STATE** — Old colonies may have signals[] that should have migrated to pheromones.json.
4. **Session template outdated** — Template has fewer fields than Go struct.
5. **Events are pipe-delimited strings** — Not JSON objects. Parse with `strings.Split(entry, "|")`.
6. **Midden subdirectory** — Lives at `.aether/data/midden/midden.json`, not `.aether/data/midden.json`.
7. **Pheromone content is json.RawMessage** — Not a plain string. Supports nested objects like `{"text": "..."}`.
8. **No file-level health checking exists** — The Medic is the first true integrity checker.

## Recommended Health Check Categories

1. **File existence + JSON parseability** — All ~15 data files + JSONL files
2. **Schema validation** — Required fields present, valid enum values
3. **Cross-file consistency** — session.json.current_phase matches COLONY_STATE.json.current_phase, etc.
4. **Structural integrity** — Valid state transitions, valid phase status values, no orphaned worktrees
5. **Version compatibility** — Detect old format strings, deprecated fields
6. **Wrapper parity** — Command counts match, agent counts match across surfaces

## Recommended Scanner Implementation Order

1. **State scanner** — COLONY_STATE.json (highest impact, most complex)
2. **Session scanner** — session.json (cross-ref with state)
3. **Pheromone scanner** — pheromones.json (signal integrity)
4. **Data file scanner** — All remaining JSON files (parseability + basic checks)
5. **JSONL scanner** — trace.jsonl + event-bus.jsonl (line integrity)
6. **Wrapper parity scanner** — Cross-surface count validation

---

*Phase: 25-medic-ant-core*
*Research completed: 2026-04-21*
