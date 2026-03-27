---
name: colony-lifecycle
description: Use when producing command output, handling state transitions, or routing users to the next action in the colony workflow
type: colony
domains: [lifecycle, routing, workflow, state]
agent_roles: [builder, watcher, route_setter, architect]
priority: normal
version: "1.0"
---

# Colony Lifecycle

## Purpose

Every command must leave the user with a clear next action. No dead ends. The colony has a defined state machine, and every output must orient the user within it.

## State Machine

The colony progresses through these states in order:

```
IDLE -> READY -> PLANNING -> EXECUTING -> SEALED -> ENTOMBED -> IDLE
```

| State | Meaning | Entered By | Next Action |
|-------|---------|------------|-------------|
| IDLE | No active colony | Default / after entomb | `/ant:init` |
| READY | Colony initialized, no plan | `/ant:init` | `/ant:plan` |
| PLANNING | Plan being generated | `/ant:plan` | `/ant:build 1` |
| EXECUTING | Phases being built | `/ant:build` | `/ant:continue` |
| SEALED | Colony marked complete | `/ant:seal` | `/ant:entomb` |
| ENTOMBED | Colony archived | `/ant:entomb` | `/ant:init` (new goal) |

## Next Up Block

Every command output must end with a "Next Up" block. This block tells the user exactly what to do next based on the current state.

### Format

```
━━ N E X T   U P ━━
Run /ant:continue to verify work and advance to the next phase.
```

### Rules

- Always include the exact command to run, with any arguments.
- If multiple valid next actions exist, list the primary one first, then alternatives.
- Match the Next Up to the current state -- never suggest a command that is invalid for the current state.
- After seal, suggest entomb. After entomb, suggest init with a new goal.
- Never output a command result without a Next Up block.

### State-Specific Next Up Examples

| Current State | Primary Next Up | Alternatives |
|---------------|-----------------|--------------|
| READY | `/ant:plan` | `/ant:colonize` (if existing code) |
| PLANNING | `/ant:build 1` | `/ant:focus` / `/ant:redirect` (to set signals first) |
| EXECUTING (just built) | `/ant:continue` | `/ant:status` (to review) |
| EXECUTING (just verified) | `/ant:build N+1` | `/ant:seal` (if last phase) |
| SEALED | `/ant:entomb` | -- |
| ENTOMBED | `/ant:init "new goal"` | -- |

## Command Chaining Awareness

Commands feed into each other. When producing output, be aware of what the previous command was and what the next one expects:

- `init` creates state that `plan` reads.
- `plan` creates phases that `build` executes.
- `build` creates artifacts that `continue` verifies.
- `continue` advances state that the next `build` reads.

If a command detects that prerequisite state is missing (e.g., `build` called with no plan), display a clear error explaining what to run first, not a cryptic failure message.

## Dead End Prevention

Before finalizing any command output, check: "Does this output end with a Next Up block?" If not, add one. There are zero valid cases where a command should leave the user without guidance on what to do next.
