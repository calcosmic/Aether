---
source: shipped
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

## Literal CLI Commands

When the user message is already a literal `aether ...` command, treat it as an instruction to run that command directly.

- Do not inspect repo files first to infer what the command "might mean".
- Use `aether --help` or `aether <subcommand> --help` only to confirm availability or flags.
- Treat the installed `aether` binary as the source of truth if docs and runtime disagree.
- If the binary does not expose a documented command, say so plainly and follow the binary's actual command surface.
- When invoking lifecycle commands through Codex shell execution, prefer `AETHER_OUTPUT_MODE=visual aether ...` unless the user explicitly wants JSON.
- Do not prepend exploratory narration like "I'm checking the repo" or "I'm treating this as..." before a literal command.
- If the `aether` CLI already rendered the result, do not restate the same guidance in a second synthetic "Next Up" block.

## State Machine

The colony progresses through these states in order:

```
IDLE -> READY -> PLANNING -> EXECUTING -> SEALED -> ENTOMBED -> IDLE
```

The authoritative runtime values in `COLONY_STATE.json` are `IDLE`, `READY`,
`EXECUTING`, `BUILT`, and `COMPLETED`. Terms like "planning", "sealed", and
"entombed" describe lifecycle moments and next steps, not always literal
persisted state values.

| State | Meaning | Entered By | Next Action |
|-------|---------|------------|-------------|
| IDLE | No active colony | Default / after entomb | `/ant-init` or `aether init` |
| READY | Colony initialized, no plan | `/ant-init` or `aether init` | `/ant-plan` or `aether plan` |
| PLANNING | Plan being generated | `/ant-plan` or `aether plan` | `/ant-build 1` or `aether build 1` |
| EXECUTING | Phases being built | `/ant-build` or `aether build` | `/ant-continue` or `aether continue` |
| SEALED | Colony marked complete with active state retained for review | `/ant-seal` or `aether seal` | `/ant-status` or `aether status` |
| ENTOMBED | Colony archived | `/ant-entomb` or `aether entomb` | `/ant-init` or `aether init` (new goal) |

## Next Up Block

Every command output must end with a "Next Up" block. This block tells the user exactly what to do next based on the current state.

### Format

```
━━ N E X T   U P ━━
Run /ant-continue or `aether continue` to verify work and advance to the next phase.
```

### Rules

- Always include the exact command to run, with any arguments.
- If multiple valid next actions exist, list the primary one first, then alternatives.
- Match the Next Up to the current state -- never suggest a command that is invalid for the current state.
- After sealing, run `AETHER_OUTPUT_MODE=visual aether status` first to review the retained sealed state; `aether entomb` is a separate optional owner-confirmed archive-and-clear action.
- Never invoke entomb automatically. After a successful entomb, suggest init with a new goal.
- Never output a command result without a Next Up block.

### State-Specific Next Up Examples

| Current State | Primary Next Up | Alternatives |
|---------------|-----------------|--------------|
| READY | `/ant-plan` or `aether plan` | `/ant-colonize` or `aether colonize` (if existing code) |
| PLANNING | `/ant-build 1` or `aether build 1` | `/ant-focus` or `aether focus` / `/ant-redirect` or `aether redirect` (to set signals first) |
| EXECUTING (just built) | `/ant-continue` or `aether continue` | `/ant-status` or `aether status` (to review) |
| EXECUTING (just verified) | `/ant-build N+1` or `aether build N+1` | `/ant-seal` or `aether seal` (if last phase) |
| SEALED | `/ant-status` or `aether status` | `/ant-entomb` or `aether entomb` only when the owner separately chooses archive-and-clear |
| ENTOMBED | `/ant-init "new goal"` or `aether init "new goal"` | -- |

## Pause and Resume

Use `aether pause` and `aether resume` as the only public pause and return commands. Pause records one validated safe-boundary handoff; resume is the sole recovery front door and lets the Go runtime decide whether restoration is safe.

- Keep the runtime's provenance groups distinct: **Confirmed** means the handoff and durable evidence agree; **Reconstructed** means one honest point was derived from named durable evidence; **Conflicting** means durable sources disagree; **Unknown** means evidence is insufficient.
- **Conflicting** or **Unknown** evidence stops with **state effect none**. Relay the named evidence or owner decision and the runtime's exact next action.
- Codex and platform wrappers must not inspect, select, or edit recovery evidence or lifecycle state. They render the runtime result; they do not choose facts or perform repair.
- Do not invent a second public recovery command. Expert diagnosis remains read-only and returns to `aether resume` for any restoration.

## Routing and Autopilot Guardrails

When user intent is freeform, classify it before acting:

- Small, well-defined task: route to a quick build path with normal verification.
- Ambiguous idea: route to discuss, assumption surfacing, or spec refinement.
- Existing active phase with completed build evidence: route to continue.
- Failed or inconsistent lifecycle state: route to `aether resume`. It either restores from consistent evidence or stops without mutation and names the exact status review or owner decision required.
- Multi-phase autonomous work: continue only while the next step is deterministic and stop when a real user decision is needed.

Do not let "autopilot" bypass lifecycle gates. It may chain valid steps, but each step must still leave state, evidence, and Next Up output consistent.

## Command Chaining Awareness

Commands feed into each other. When producing output, be aware of what the previous command was and what the next one expects:

- `init` creates state that `plan` reads.
- `plan` creates phases that `build` executes.
- `build` creates artifacts that `continue` verifies.
- `continue` advances state that the next `build` reads.
- `pause` writes one validated handoff that `resume` alone may restore.
- `seal` retains the completed colony for `status` review; only a separately chosen `entomb` verifies the archive and clears active state.

If a command detects that prerequisite state is missing (e.g., `build` called with no plan), display a clear error explaining what to run first, not a cryptic failure message.

## Dead End Prevention

Before finalizing any command output, check: "Does this output end with a Next Up block?" If not, add one. There are zero valid cases where a command should leave the user without guidance on what to do next.
