# Phase 26: Auto-Repair - Context

**Gathered:** 2026-04-21
**Status:** Ready for planning
**Source:** Phase 25 context + R040 requirement

<domain>
## Phase Boundary

Implement the `--fix` repair logic for the Medic. When `aether medic --fix` is run, the Medic attempts to repair all fixable issues found by the Phase 25 scanners. Every repair is logged to trace.jsonl with before/after state. Destructive repairs require `--force` in addition to `--fix`.

**Scope:** Repair implementation only. The Medic skill (healthy state spec) is Phase 27. Ceremony integrity checks are Phase 28.
</domain>

<decisions>
## Implementation Decisions

### Repair Execution Model
- **D-01:** Repairs run after the scan completes. The Medic first scans (read-only), then repairs fixable issues in priority order: critical first, then warnings, then info.
- **D-02:** Each repair is an atomic operation. If a repair fails, it's logged and the Medic continues to the next repair. Partial success is reported.
- **D-03:** `--fix` enables repair mode. `--fix --force` enables destructive repairs (truncating corrupted files, removing entries that lose data).

### Backup Before Repair
- **D-04:** Before any repairs, snapshot all colony data files to `.aether/backups/medic-{timestamp}/`. This is a full copy of `.aether/data/` plus any files being modified.
- **D-05:** Backup happens once at the start of repair mode, not per-repair. This avoids redundant copies.
- **D-06:** Backup cleanup is the user's responsibility. The Medic keeps the latest 3 backups, removes older ones.

### Trace Logging
- **D-07:** Every repair writes a `trace.jsonl` entry with: level="intervention", topic="medic.repair", payload containing {action, file, before (snippet), after (snippet), success, error}.
- **D-08:** Failed repairs also get logged with error details.
- **D-09:** A final trace entry summarizes: total repairs attempted, succeeded, failed.

### Specific Repairs
- **D-10:** Fix stale spawn state: clear `spawn-runs.json` current_run_id if no active build, reset run status from "running" to "failed" or "completed" if stale.
- **D-11:** Remove orphaned worktree entries: filter worktrees array in COLONY_STATE.json to remove entries with status "orphaned". Verify git worktree doesn't actually exist before removing the entry.
- **D-12:** Rebuild missing indexes: regenerate `.cache_COLONY_STATE.json` and `.cache_instincts.json` if they reference stale data.
- **D-13:** Fix corrupted JSON structures: attempt to parse file, if fails, try to recover valid JSON (truncate at last valid closing brace/bracket), write recovered version. Requires `--force` since data may be lost.
- **D-14:** Clear expired pheromone signals: set `active: false` on signals past their `expires_at` timestamp.
- **D-15:** Normalize legacy state values: apply `normalizeLegacyColonyState()` fixes if old state strings detected (PAUSED, PLANNED, etc.).
- **D-16:** Clear deprecated signals field: if COLONY_STATE.json has non-empty `signals` array, move entries to pheromones.json (if valid) and clear the array.

### Repair Safety
- **D-17:** Never repair a file that can't be parsed at all (completely corrupted). Instead, report it as critical and suggest manual restoration from backup.
- **D-18:** Never delete files. Only modify file contents in place.
- **D-19:** After all repairs, re-run the health scan to verify the repairs worked. Report any remaining issues.

### Claude's Discretion
- Order of repairs when multiple critical issues exist
- Specific recovery heuristics for corrupted JSON (how aggressive the truncation)
- Whether to rebuild indexes proactively or only when stale
</decisions>

<canonical_refs>
## Canonical References

### Phase 25 Implementation
- `cmd/medic_cmd.go` — Medic command, MedicOptions, HealthIssue structs
- `cmd/medic_scanner.go` — performHealthScan, all scanners, fileChecker, issue helpers
- `cmd/medic_wrapper.go` — Wrapper parity scanner
- `.planning/phases/25-medic-ant-core/25-RESEARCH.md` — All colony data schemas
- `.planning/phases/25-medic-ant-core/25-CONTEXT.md` — Repair philosophy decisions

### Trace System
- `cmd/trace_cmds.go` — Existing trace commands
- `pkg/trace/trace.go` — TraceEntry, Tracer, Log helpers
- `cmd/medic_cmd.go` — performHealthScan integration

### State Loading & Repair
- `cmd/state_load.go` — normalizeLegacyColonyState, repairMissingPlanFromArtifacts
- `pkg/storage/storage.go` — AtomicWrite, SaveJSON (for safe file writes)
- `pkg/storage/lock.go` — FileLocker for concurrent access safety
</canonical_refs>

<deferred>
## Deferred Ideas

- **Wrapper/runtime drift reconciliation** — Complex, needs deep ceremony knowledge. Phase 28 scope.
- **Cross-colony repair** — Repairing other repos' colonies. Phase 29 scope.
- **Auto-spawn on health issues** — Phase 30 scope.
</deferred>

---

*Phase: 26-auto-repair*
*Context gathered: 2026-04-21*
