# Requirements — v1.12 Safe Colony

## Loop Safety

### LOOP-01: Continue watcher loop prevention
When `/ant-continue` spawns a watcher that fails (not times out), the system must detect repeated failures and auto-skip the watcher after N consecutive failures instead of re-spawning it each time the user runs continue.

### LOOP-02: Continue recovery command loop prevention
When `/ant-continue` is blocked and suggests a recovery command, the recovery command must not loop back to `/ant-continue` with the same state, creating an infinite block-suggest-retry cycle.

### LOOP-03: Build wave retry loop prevention
When `/ant-build` dispatches workers and a worker fails, the wave execution must not re-dispatch the same failed worker indefinitely. Failed workers must be tracked and escalated (not silently retried).

### LOOP-04: Plan circular dependency prevention
When `/ant-plan` generates a phase plan, the plan must not contain circular phase dependencies (phase A depends on B, B depends on A). The planner must detect and reject circular dependency chains.

### LOOP-05: Lifecycle command retry safety
When lifecycle commands (`/ant-seal`, `/ant-entomb`, `/ant-status`, `/ant-resume`) encounter errors, they must provide a clear next step that is different from the command that just failed. They must never suggest re-running themselves as the only recovery option.

### LOOP-06: Loop detection telemetry
All loop-breaking events must be logged to the colony event bus with the loop type, detection signal, and action taken. `/ant-status` must surface recent loop-break events.

## Depth Controls

### DEPTH-01: Independent planning depth
The planning system (`/ant-plan`) must support a planning depth setting (light/standard/deep) that controls how thoroughly tasks are decomposed. Light = minimal tasks, standard = normal breakdown, deep = granular subtasks with edge cases.

### DEPTH-02: Independent verification depth
The verification system (`/ant-continue`) must support a verification depth setting (light/standard/heavy) that controls how thorough the review is. This is separate from planning depth and already partially exists (light/heavy) — extend to three levels.

### DEPTH-03: Smart depth defaults
The system must auto-select both planning depth and verification depth based on two signals combined:
1. **Phase position** — final phases in a milestone get heavier treatment
2. **Code change risk** — phases touching security-critical paths, core runtime, or high-blast-radius files get heavier treatment

### DEPTH-04: User depth selection at plan time
When `/ant-plan` runs, the user must be presented with the smart default for both planning depth and verification depth, and can accept the defaults or override either one before the plan is generated.

### DEPTH-05: Depth persistence across continue
The verification depth selected at plan time must be stored in the build packet and honored by `/ant-continue` without requiring the user to re-specify it.

## Traceability

| REQ-ID | Phase | Status |
|--------|-------|--------|
| LOOP-01 | Phase 80 | Complete |
| LOOP-02 | Phase 80 | Complete |
| LOOP-03 | Phase 80 | Complete |
| LOOP-04 | Phase 81 | Complete |
| LOOP-05 | Phase 81 | Complete |
| LOOP-06 | Phase 82 | Complete |
| DEPTH-01 | Phase 83 | Complete |
| DEPTH-02 | Phase 84 | Pending |
| DEPTH-03 | Phase 85 | Pending |
| DEPTH-04 | Phase 86 | Pending |
| DEPTH-05 | Phase 86 | Pending |

## Out of Scope

| Feature | Reason |
|---------|--------|
| Depth controls for build waves | Build wave parallelism is orthogonal to planning/verification depth |
| Depth controls for research/oracle | These are already scoped by their own mechanisms |
| Cross-colony depth preferences | Depth is colony-scoped, not user-global |
| Visual depth indicator in status | Can be added later; not essential for v1.12 |
