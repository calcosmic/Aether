# Phase 4: Planning Granularity Controls - Context

**Gathered:** 2026-04-07
**Status:** Ready for planning

<domain>
## Phase Boundary

Let users control how many phases the plan generates by selecting a granularity range (sprint, milestone, quarter, major). This phase covers: defining 4 granularity ranges with phase count bounds, adding a Go enum type (following the ColonyDepth pattern from Phase 3), wiring the `--granularity` flag into `/ant:plan`, validating plan output against the selected range, persisting the setting in COLONY_STATE.json, and ensuring the autopilot respects it. It does NOT cover build depth or orchestration — those are separate phases.

</domain>

<decisions>
## Implementation Decisions

### Range definitions
- **D-01:** 4 granularity presets: Sprint (1-3 phases), Milestone (4-7), Quarter (8-12), Major (13-20). Matches PLAN-01 exactly.
- **D-02:** Follow the same Go enum pattern as ColonyDepth — `type PlanGranularity string` with const declarations and a `Valid()` method.

### Default behavior
- **D-03:** `/ant:plan` always asks for granularity if none is set — no silent default. The user picks from the 4 presets each time (unless one is already persisted from a previous `/ant:plan` or `/ant:init` call).
- **D-04:** Once selected, granularity persists in COLONY_STATE.json. Subsequent `/ant:plan` calls use the persisted value unless overridden with `--granularity`.

### Out-of-range handling
- **D-05:** If the route-setter generates a plan outside the selected range, show a clear warning (actual count vs. chosen range) and let the user decide: accept as-is, adjust the range to fit, or replan. No silent auto-trimming or hard rejection.

### Granularity and depth interaction
- **D-06:** Granularity and depth are fully independent. Granularity controls phase count (breadth), depth controls build thoroughness per phase. No cross-influence or soft recommendations.

### Route-setter integration
- **D-07:** The `--granularity` bounds (min/max phases) must be injected into the route-setter prompt, replacing the current hardcoded "Maximum 6 phases" constraint. The plan command reads the persisted or selected granularity and passes min/max to the route-setter.
- **D-08:** The plan command's output constraint line (`Maximum 6 phases`) must be dynamically set based on the selected granularity range's max value.

### Autopilot integration
- **D-09:** `/ant:run` reads the persisted granularity from COLONY_STATE.json and respects the phase count. If the plan has more phases than the range allows, the autopilot warns but continues (the plan was already accepted by the user during `/ant:plan`).

### Claude's Discretion
- Exact enum implementation details (iota vs string constants)
- Whether to add `--granularity` flag to `/ant:init` (like depth has)
- How the "always ask" prompt appears in `/ant:plan` (inline question vs separate step)
- Exact warning message format for out-of-range plans
- Whether `state-mutate` should validate granularity values (like depth)
- Whether to add a `plan-granularity get/set` command pair (like `colony-depth`)

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Requirements
- `.planning/REQUIREMENTS.md` — PLAN-01 through PLAN-05 define the exact requirements for this phase

### Roadmap
- `.planning/ROADMAP.md` §Phase 4 — Success criteria, dependencies, and risk notes

### Colony state model
- `pkg/colony/colony.go:25-45` — `ColonyDepth` enum pattern to follow exactly for `PlanGranularity`
- `pkg/colony/colony.go:66-86` — `ColonyState` struct (add `PlanGranularity` field)

### Plan command (needs --granularity flag + range injection)
- `.claude/commands/ant/plan.md:72-87` — Planning depth presets (pattern to follow for granularity presets)
- `.claude/commands/ant/plan.md:477-487` — Route-setter output constraints (replace hardcoded "Maximum 6 phases")
- `.aether/commands/claude/plan.md:477-493` — Same in .aether source

### Route-setter agent (needs dynamic phase bounds)
- `.aether/agents/aether-route-setter.md:36` — "3-6 phases for most goals" (must become dynamic)
- `.aether/agents/aether-route-setter.md:106` — Phase count validation (must use granularity bounds)
- `.aether/agents-claude/aether-route-setter.md:55` — Same in agents-claude mirror

### Autopilot (needs to read persisted granularity)
- `.aether/commands/claude/run.md:67-69` — Step 0 reads COLONY_STATE.json (add granularity check)
- `.aether/commands/claude/run.md:174` — `--max-phases` cap (should interact with granularity bounds)

### Status display (needs to show granularity)
- `cmd/status.go:114-120` — Depth display in status output (add granularity line below it)

### Depth command pattern (follow for plan-granularity commands)
- `cmd/colony_cmds.go:65-143` — `colony-depth get/set` command pair (template for `plan-granularity get/set`)

### Phase 3 context (established patterns)
- `.planning/phases/03-build-depth-controls/03-CONTEXT.md` — Depth enum, persistence, validation patterns

### Phase 1 infrastructure (audit logging)
- `pkg/storage/storage.go` — `AppendJSONL` for audit trail of granularity changes

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `ColonyDepth` enum type in `pkg/colony/colony.go` — exact template for `PlanGranularity` enum
- `colony-depth get/set` commands — exact template for `plan-granularity get/set`
- Plan command already has a preset selection pattern (planning depth: fast/balanced/deep/exhaustive) — can add granularity presets alongside
- `/ant:status` already displays depth — add granularity on the same pattern
- Phase 1 audit infrastructure for tracking granularity mutations

### Established Patterns
- Enum types use `type X string` with const declarations and a `Valid()` method
- Commands use `outputOK()` / `outputError()` for consistent JSON output
- State mutations go through `store.SaveJSON()` with `FileLocker`
- Plan command reads state at start, passes constraints to route-setter prompt
- Agent definitions in `.aether/agents/` must be mirrored to `.aether/agents-claude/`

### Integration Points
- `pkg/colony/colony.go` — Add `PlanGranularity` field to `ColonyState` struct
- `.claude/commands/ant/plan.md` — Add granularity selection step + pass bounds to route-setter
- `.aether/agents/aether-route-setter.md` — Replace hardcoded phase limits with dynamic bounds
- `.aether/agents-claude/aether-route-setter.md` — Mirror changes
- `cmd/status.go` — Add granularity to status output
- `cmd/colony_cmds.go` — Add `plan-granularity get/set` commands (if created)
- `.aether/commands/claude/run.md` — Autopilot reads persisted granularity

</code_context>

<specifics>
## Specific Ideas

- The "always ask" behavior means first-time `/ant:plan` users see a clear choice upfront — no surprises about plan size
- Warn+choose for out-of-range respects the user's judgment — sometimes a 5-phase sprint is exactly what they wanted
- Keeping granularity and depth independent means the system has 4x4 = 16 combinations, all valid

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope.

</deferred>

---
*Phase: 04-planning-granularity-controls*
*Context gathered: 2026-04-07*
