# Phase 83: Planning Depth System - Context

**Gathered:** 2026-04-30
**Status:** Ready for planning

<domain>
## Phase Boundary

`/ant-plan` supports a `--planning-depth` setting (light/standard/deep) that controls how thoroughly tasks are decomposed within each plan. Light = minimal tasks, standard = normal breakdown, deep = granular subtasks with edge cases.

This phase covers DEPTH-01 from REQUIREMENTS.md.

**What this phase delivers:**
- A `--planning-depth` flag accepted by the Go runtime `aether plan` command
- Three depth levels: light, standard, deep
- The depth value passed into plan prompt context so wrapper agents can adjust task decomposition
- Updated wrapper markdown (.claude/commands/ant/plan.md, .opencode/commands/ant/plan.md) exposing the flag in help text

**What this phase does NOT deliver:**
- Verification depth extension (DEPTH-02, Phase 84)
- Smart depth defaults (DEPTH-03, Phase 85)
- Depth selection UI or persistence (DEPTH-04/05, Phase 86)
- Changes to the existing PlanGranularity system (sprint/milestone/quarter/major) which controls phase count — that's a separate concept

**Key distinction:** Planning depth (DEPTH-01) controls task decomposition detail within a plan. PlanGranularity (existing) controls how many phases are generated. These are orthogonal — a phase can have any combination of granularity and planning depth.

</domain>

<decisions>
## Implementation Decisions

### Naming
- **D-01:** Use the ROADMAP's naming: light/standard/deep. Do NOT rename the existing PlanGranularity depth names (fast/balanced/deep/exhaustive) — they refer to a different concept (phase count scope, not task detail).

### Task decomposition levels
- **D-02 (Claude's Discretion):** Light mode task count — let the planner determine based on phase context. Target range: 1-3 tasks per plan with objective-level descriptions, no implementation steps.
- **D-03 (Claude's Discretion):** Deep mode task count — let the planner determine based on phase context. Target range: 5-8 tasks per plan including edge cases, error handling, and test coverage as separate tasks.
- **D-04:** Standard mode is the baseline — normal task breakdown as currently produced. No change to existing behavior when depth is unspecified.

### Integration approach
- **D-05:** Apply planning depth at both the runtime AND wrapper level. The Go binary accepts `--planning-depth` and passes it into the plan prompt context. The wrapper markdown files (.claude/commands/ant/plan.md, .opencode/commands/ant/plan.md) are updated to document the flag and pass it through.

### Relationship to existing system
- **D-06:** Planning depth is independent of PlanGranularity. The `--planning-depth` flag does not affect how many phases are generated. It only affects task detail within each plan.
- **D-07:** The `colony_depth` field on ColonyState already exists but is unused for planning. This phase does NOT repurpose it — depth selection UI and persistence is Phase 86's scope.

### Default behavior
- **D-08:** When no `--planning-depth` is specified, default to "standard" (no change to current behavior).

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Requirements
- `.planning/REQUIREMENTS.md` — DEPTH-01 definition (light/standard/deep task decomposition)

### Roadmap
- `.planning/ROADMAP.md` — Phase 83 goal, success criteria, and dependency chain (DEPTH-01 through DEPTH-05)

### Existing depth system (different concept)
- `pkg/colony/colony.go` — PlanGranularity type definition (sprint/milestone/quarter/major) and Valid() method
- `pkg/colony/granularity.go` — GranularityRange() function mapping granularity to phase count ranges
- `cmd/codex_plan.go` — normalizedGranularity() and planningDepthForGranularity() functions (existing depth naming: fast/balanced/deep/exhaustive)

### Phase dependencies
- `.planning/phases/81-plan-and-lifecycle-loop-safety/81-CONTEXT.md` — Phase 81 context (dependency: plan must be loop-safe before adding depth features)

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `ColonyState.PlanGranularity` — existing field on colony state that could serve as a reference pattern for how to add a new depth concept
- `normalizedGranularity()` in `cmd/codex_plan.go` — existing function that maps granularity values to depth names; useful as a template for the new planning depth normalization

### Established Patterns
- The plan command builds a prompt context map (`wrapper_contract`) that wrapper agents consume — adding `planning_depth` to this map is the natural integration point
- Plan YAML source definitions live in `.aether/commands/` and generate to `.claude/commands/ant/` and `.opencode/commands/ant/`

### Integration Points
- `cmd/codex_plan.go` — main plan command implementation, where the `--planning-depth` flag gets parsed and passed into the prompt context
- `.aether/commands/plan.yaml` — YAML source definition for the plan command, where the flag should be declared
- `.claude/commands/ant/plan.md` — Claude Code wrapper for `/ant-plan`
- `.opencode/commands/ant/plan.md` — OpenCode wrapper for `/ant-plan`

</code_context>

<specifics>
## Specific Ideas

No specific requirements — open to standard approaches. The planner should use reasonable defaults for task counts at each depth level.

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope.

</deferred>

---

*Phase: 83-planning-depth-system*
*Context gathered: 2026-04-30*
