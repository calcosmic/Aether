# Phase 120: Platform Dispatch Correctness - Context

**Gathered:** 2026-05-14
**Status:** Ready for planning

## Phase Boundary

Fix platform-specific dispatch bugs, add tests for argument construction, and make simulation fallback explicit. No new features — stabilization of existing v1.17 dispatch code.

## Implementation Decisions

- Export `buildArgs` from `platform-dispatcher.ts` for unit testing (currently private)
- Codex prompt: pass as final positional argument to `codex exec`
- Simulation warning: write to stderr so it appears in ceremony logs
- Keep existing `simulateWorkers` default behavior but add visible warning

## Threat Model

| Threat | Mitigation |
|--------|-----------|
| Codex exec receives no prompt | Pass `config.prompt` as final positional arg |
| Silent simulation masks broken dispatch | Log `⚠️  No platforms available; simulating worker` |
| Arg construction drifts undetected | Unit test each platform's args array |
| Exported `buildArgs` becomes public API | Mark as `@internal` in JSDoc |

## Test Strategy
- Unit tests for `buildArgs` per platform (Claude, OpenCode, Codex)
- Integration test: simulate fallback logs warning when no platforms available
- Existing tests must continue passing
