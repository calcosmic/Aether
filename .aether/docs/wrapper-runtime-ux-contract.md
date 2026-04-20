# Wrapper-Runtime UX Contract

Updated: 2026-04-19

This contract defines the ownership boundary between the Go runtime and the
platform wrapper layer (Claude Code, OpenCode). It ensures wrappers enhance
presentation without duplicating logic or drifting from runtime truth.

Use the Go `aether` CLI as the source of truth for runtime behavior. Wrappers
only add presentation and pacing on top of that runtime contract.

## Runtime Surface (Go — Authoritative)

The Go runtime owns ALL of the following. Wrappers must never replicate them:

### State Management
- Colony state transitions (READY → EXECUTING → BUILT → COMPLETED)
- Phase advancement and gating
- Session tracking and freshness detection
- File locking and concurrent access

### Build/Continue Workflow
- Build manifest generation (`cmd/codex_build.go`)
- Worker dispatch planning and wave allocation
- Verification step execution
- Gate evaluation (security, quality, coverage, performance)
- Claims collection and persistence
- Continue verification, signal housekeeping, and advancement

### Visual Rendering
- All banner, progress bar, and status formatting (`cmd/codex_visuals.go`)
- ANSI color handling and terminal capability detection
- Caste emoji/color maps and identity rendering
- Spawn plan visualization
- Phase/task status display

### Structured Output
The runtime exposes two output modes:

| Mode | Env Var | Consumer | Content |
|------|---------|----------|---------|
| JSON | `AETHER_OUTPUT_MODE=json` | Machines, tests | Structured data envelopes |
| Visual | `AETHER_OUTPUT_MODE=visual` | Terminals, humans | Formatted ANSI output |

JSON output includes: state, phases, tasks, dispatches, verification results,
gates, claims, housekeeping, blockers, and next-step suggestions.

Visual output is the JSON data rendered through `codex_visuals.go` functions.

## Wrapper Additions (Markdown — Enhancement)

Wrappers MAY add the following on top of runtime output:

### Pre-Build Context
- Colony atmosphere (Queen persona, ant metaphor)
- Current phase context (what we're building and why)
- Pheromone signal summary (what guidance is active)
- Historical context (what previous phases accomplished)

### During-Build Narration
- Explaining what workers are doing in plain language
- Status updates with colony framing
- Noting when workers encounter issues

### Post-Build Summary
- What was accomplished in colony terms
- Key decisions made during the phase
- What the verification found
- What the next phase will address

### Follow-Up Guidance
- Suggesting pheromone signals for steering
- Recommending focus areas for upcoming phases
- Highlighting risks or blockers the user should know about

## Wrapper Anti-Patterns (Prohibited)

Wrappers MUST NOT:

1. **Mutate state files** — Never write to COLONY_STATE.json, session.json,
   pheromones.json, or any file in `.aether/data/` directly.

2. **Replay orchestration logic** — Never duplicate build dispatch planning,
   verification sequencing, gate evaluation, or phase advancement logic.

3. **Parse visual text as truth** — Never scrape ANSI-formatted output to
   extract state information. Use JSON mode if programmatic data is needed.

4. **Duplicate verification** — Never re-implement test running, security
   scanning, coverage analysis, or quality gating.

5. **Override runtime routing** — Never contradict the runtime's next-step
   suggestions. Wrappers may suggest alternatives but must present the
   runtime's recommendation first.

6. **Add unsanctioned menus** — Never create option menus or recovery paths
   that don't come from the runtime itself.

## Codex Platform

Codex is a special case because it does NOT use wrapper markdown:

- Codex interacts directly with the Go CLI
- All Codex UX comes from the runtime visual renderer
- Codex agents are defined in `.codex/agents/*.toml`, not slash commands
- Improvements to Codex UX must target `cmd/codex_visuals.go`
- Codex should NOT simulate wrapper behavior in its agents

## Source Chain

```
.aether/commands/*.yaml          ← Source definitions (name, runtime command, guardrails)
    ↓ (generation)
.claude/commands/ant/*.md        ← Claude Code wrappers
.opencode/commands/ant/*.md      ← OpenCode wrappers
    ↓ (delegation)
cmd/codex_*.go                   ← Go runtime (authoritative execution)
cmd/codex_visuals.go             ← Visual renderer (authoritative presentation)
```

The current repo does not check in a wrapper generator. The maintained contract
is YAML-backed manual sync plus automated parity and provenance tests.

## Enforcement

- Tests in `cmd/codex_visuals_test.go` verify visual output correctness
- YAML source files in `.aether/commands/` define wrapper boundaries
- `source-of-truth-map.md` documents the ownership hierarchy
- CLAUDE.md and CODEX.md reference this contract

## References

- `.aether/docs/source-of-truth-map.md` — Authority hierarchy
- `cmd/codex_visuals.go` — Visual rendering implementation
- `cmd/codex_build.go` — Build workflow implementation
- `cmd/codex_continue.go` — Continue workflow implementation
- `.aether/commands/build.yaml` — Build wrapper source definition
- `.aether/commands/continue.yaml` — Continue wrapper source definition
