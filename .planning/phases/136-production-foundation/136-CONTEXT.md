# Phase 136: Production Foundation - Context

**Gathered:** 2026-05-18
**Status:** Ready for planning

<domain>
## Phase Boundary

The TypeScript host dispatches real platform workers (Claude Code, OpenCode, Codex) for build, plan, continue, and oracle workflows — not simulation. Ceremony renders with real caste visuals, skill sections inject correctly, dry-run previews work, and missing platforms produce clear diagnostics.

This is the activation phase: the dispatch code already exists (`platform-dispatcher.ts`, `worker-dispatch.ts`, `wave-orchestrator.ts`), it just runs in simulation mode by default. Phase 136 flips the default to real dispatch and proves it works end-to-end.

</domain>

<decisions>
## Implementation Decisions

### Failure Handling
- **D-01:** Auth and configuration errors (missing CLI, expired credentials, wrong version) halt the build immediately with a plain English message — the user needs to act before continuing
- **D-02:** Timeouts and transient failures do NOT halt the build — the failed worker is marked as failed and remaining workers continue; a summary appears at wave end (e.g., "2 succeeded, 1 failed (Mason-67: timeout)")
- **D-03:** Error messages are plain English only — no error codes, file paths, or jargon. Example: "Claude Code is not installed. Install it from claude.ai/code and try again." (Go runtime's `platform-diagnostic` is used behind the scenes, but the user never sees raw diagnostic output)

### Real Dispatch UX
- **D-04:** Ceremony output is identical to simulation — same banners, same caste names (emoji + ANSI colors), same stage markers. The user does NOT see a "THIS IS REAL" banner or any indication that dispatch changed. The only difference is that files actually change and tests actually run
- **D-05:** The existing swarm/ceremony display (`swarm-display.ts`, `ceremony-adapter.ts`) feeds from real dispatch the same way it feeds from simulation — caste colors, emoji, spinners, per-worker progress. No new display code for real dispatch

### Dry-run Output
- **D-06:** `--dry-run` shows full ceremony preview (same caste names, wave structure, stage markers) with a visible "DRY RUN" badge/header so the user knows it's a preview. No workers are spawned. The user sees exactly what the real build would look like

### Skill Visibility
- **D-07:** One-line summary at build start when skills are injected: "Injecting N skills into worker prompts." No per-worker skill count, no ceremony noise. Skills are an internal optimization — worker output speaks for itself

### Claude's Discretion
- Technical implementation of simulation guard removal (`simulateWorkers=false` default)
- How to structure the `--dry-run` ceremony rendering (reuse existing ceremony adapter vs. new path)
- Claims parsing for real platform output (error classification for auth failures, rate limits, timeouts)
- How to wire real dispatch results into the existing Go finalizer pipeline
- Test strategy for real dispatch (what can be unit tested vs. integration tested vs. smoke tested)
- Ceremony "DRY RUN" badge rendering style and placement

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### TypeScript Host (Primary Codebase)
- `.aether/ts-host/src/worker-dispatch.ts` — Worker dispatch orchestration, simulation vs. real paths
- `.aether/ts-host/src/platform-dispatcher.ts` — Platform CLI detection and worker subprocess spawning
- `.aether/ts-host/src/wave-orchestrator.ts` — Parallel wave execution with dependency ordering
- `.aether/ts-host/src/prompt-assembler.ts` — Worker prompt construction and skill injection
- `.aether/ts-host/src/claims-parser.ts` — Parsing worker output claims from real platform output
- `.aether/ts-host/src/ceremony-adapter.ts` — Ceremony rendering (caste emoji, ANSI colors, stage markers)
- `.aether/ts-host/src/oracle-lifecycle.ts` — Oracle lifecycle with confidence-driven loop
- `.aether/ts-host/src/host.ts` — TS host entry point and command routing
- `.aether/ts-host/src/types.ts` — Shared type definitions (BuildDispatch, WorkerResult, etc.)
- `.aether/ts-host/src/lifecycle.ts` — Lifecycle smoke test harness

### Go Runtime (State Authority)
- `cmd/colony_prime_context.go` — Colony-prime context builder (prompt assembly, skill section generation)
- `cmd/skills.go` — Skill system commands (skill-index, skill-match, skill-inject)
- `cmd/spawn.go` — Spawn tracking (spawn-log, spawn-complete)
- `cmd/build_flow_cmds.go` — Build workflow commands (build --plan-only manifest generation)
- `cmd/codex_visuals.go` — Caste identity rendering (casteEmojiMap, casteColorMap, casteLabelMap)

### Wrapper Contracts
- `.aether/docs/wrapper-runtime-ux-contract.md` — Wrapper/runtime boundary rules
- `.aether/docs/host-command-reference.md` — Host command specifications
- `.aether/docs/wrapper-host-contract.md` — Host-assisted orchestrator contract

### Requirements
- `.planning/REQUIREMENTS.md` — HOST-01 through HOST-09, SKILL-01 through SKILL-03
- `.planning/ROADMAP.md` — Phase 136 goal and success criteria

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `platform-dispatcher.ts` already has real dispatch code using `spawn()` for subprocess invocation — it detects Claude/OpenCode/Codex CLIs and spawns with correct arguments per platform. The code exists and has been tested; it just isn't called by default.
- `ceremony-adapter.ts` renders ceremony events (spawn-plan, wave-start, worker-complete, closeout) with caste emoji and ANSI colors — already works for simulation, feeds from the same event stream for real dispatch.
- `swarm-display.ts` provides live terminal dashboard with animated spinners and per-worker progress — the display the user mentioned wanting for real workers.
- `claims-parser.ts` parses worker output into structured claims (files_modified, files_created, summary) — already handles the real platform output format.
- `wave-orchestrator.ts` manages parallel wave execution with dependency ordering — dispatches workers within each wave in parallel.

### Established Patterns
- Go manifest pattern: `aether build N --plan-only` returns JSON manifest with dispatch array, each entry has caste, task, agent, prompt content. TS host iterates this manifest.
- Go finalizer pattern: after workers complete, TS host calls `aether build N --finalize` with results. Go owns the state mutation.
- Ceremony event stream: Go emits JSONL events, TS host renders them. Real dispatch reuses the same rendering pipeline.

### Integration Points
- `simulateWorkers` flag in `worker-dispatch.ts` controls real vs. simulated path — flipping the default activates real dispatch
- `prompt-assembler.ts` gets `skill_section` from Go manifest — skill injection happens during prompt assembly
- Go `platform-diagnostic` command provides installation/auth/version checks — TS host calls it for HOST-08 error messages

</code_context>

<specifics>
## Specific Ideas

- User expects to see workers "working in the terminal window with the caste colors" — the existing swarm/ceremony display is the right UI, just needs to be wired to real dispatch output
- User chose "silent upgrade" for real dispatch UX — no banners or notices announcing the change from simulation

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope.

</deferred>

---

*Phase: 136-production-foundation*
*Context gathered: 2026-05-18*
