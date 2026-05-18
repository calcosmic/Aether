# Phase 140: Hardening and Validation - Context

**Gathered:** 2026-05-18
**Status:** Ready for planning

<domain>
## Phase Boundary

End-to-end validation that the complete TypeScript host + Go runtime pipeline works correctly across all 4 prior phases (136-139), wrapper markdown contracts are honored, and the v1.21 milestone is ready to ship. This is a gap-filling and cross-cutting validation phase — no new production features.

The phase covers: wrapper-runtime UX alignment tests across all commands, cross-phase integration verification, milestone audit confirming all requirements addressed, and regression testing ensuring nothing broke across the milestone.

</domain>

<decisions>
## Implementation Decisions

### Wrapper Alignment Testing
- **D-01:** Snapshot tests verify that ceremony sections (spawn-plan, wave-start, worker-complete, closeout) appear in wrapper output — section presence, not exact text match. Wrappers add Queen narration and pheromone context around the runtime ceremony; tests confirm the ceremony markers exist inside wrapper output, not that wrapper output matches runtime output exactly.
- **D-02:** Wrapper alignment tests cover ALL commands — build, plan, continue, oracle (the 4 dispatched commands with ceremony) plus init, status, seal, resume, and other read-only commands. The 4 dispatched commands have the richest ceremony output; read-only commands have simpler output but still need wrapper-runtime consistency.
- **D-03:** Substring containment for ceremony markers: tests check that runtime ceremony markers (stage separators, caste names, iteration markers) appear inside wrapper output. This is tolerant of wrapper-added framing while catching missing ceremony sections.

### Claude's Discretion
- How to structure the snapshot test files (one file per command, or grouped by workflow)
- Whether to test Claude wrappers, OpenCode wrappers, or both
- How to generate test fixtures (run host commands with mocked platforms, capture output)
- Milestone audit format and what constitutes "all requirements addressed"
- Whether to add Go-level integration tests for the Go→TS→Go bridge
- Regression test scope (full TS suite + full Go suite, or targeted subsets)

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Wrapper Contracts
- `.aether/docs/wrapper-runtime-ux-contract.md` — Defines what wrappers must render, ceremony contract, and the wrapper-runtime boundary
- `.aether/docs/wrapper-host-contract.md` — Defines how wrappers call the host and process output
- `.claude/commands/ant/build.md` — Claude Code build wrapper (orchestrator that delegates to host)
- `.claude/commands/ant/continue.md` — Claude Code continue wrapper
- `.opencode/commands/ant/build.md` — OpenCode build wrapper
- `.opencode/commands/ant/continue.md` — OpenCode continue wrapper

### Runtime Ceremony (Source of Truth)
- `.aether/ts-host/src/ceremony-adapter.ts` — Ceremony rendering with caste emoji, ANSI colors, stage markers
- `.aether/ts-host/src/host.ts` — Host command dispatch with iteration loop, ceremony output
- `cmd/codex_visuals.go` — Go runtime ceremony renderer (ANSI banners, progress bars, caste identity)

### Prior Phase Outputs (What Must Work Together)
- `.aether/ts-host/src/confidence-loop.ts` — Iteration engine (Phase 139)
- `.aether/ts-host/src/confidence-evaluator.ts` — Scoring from worker claims (Phase 139)
- `.aether/ts-host/src/hive-injector.ts` — Hive wisdom injection (Phase 137)
- `.aether/ts-host/src/spawn-orchestrator.ts` — Worker spawning with budget (Phase 138)
- `.aether/ts-host/src/worker-dispatch.ts` — Real dispatch pipeline (Phase 136)
- `.aether/ts-host/src/platform-dispatcher.ts` — Platform CLI detection (Phase 136)

### Requirements
- `.planning/REQUIREMENTS.md` — v1.21 requirements with traceability
- `.planning/ROADMAP.md` — Phase 140 definition and milestone status

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `.aether/ts-host/test/host-integration.test.ts` — 35 integration tests with mock patterns (__setCallGoJSON, __setDispatchWorkers, __setDetectAvailablePlatforms). The wrapper alignment tests can reuse these mocking patterns.
- `.aether/ts-host/test/confidence-loop-e2e.test.ts` — E2E test patterns for the iteration lifecycle
- `.aether/ts-host/test/classic-command-parity.test.ts` — Existing parity tests between wrapper and host

### Established Patterns
- Test style: node:test with assert/strict (all TS host tests follow this)
- Mocking: bridge-level mocking via __setCallGoJSON for Go subprocess calls
- Ceremony sections are delimited by `──` ANSI stage markers

### Integration Points
- Wrapper commands call `aether host <command>` via the platform's shell
- Host produces ceremony output on stderr (ANSI-formatted)
- Wrappers render the ceremony inside their colony framing (Queen persona, pheromones)
- The classic-command-parity tests already compare some output patterns

</code_context>

<specifics>
## Specific Ideas

- The existing `classic-command-parity.test.ts` file already tests some wrapper-host parity — extend this rather than creating an entirely new test file
- Ceremony markers use the `── Stage Name ──` format — these are the section presence anchors

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope.

</deferred>

---

*Phase: 140-hardening-and-validation*
*Context gathered: 2026-05-18*
