# Phase 120: Platform Dispatch Correctness - Research

**Gathered:** 2026-05-14
**Status:** Ready for planning

## Issues Identified

### 1. Codex prompt not passed (DSP-01)
**File:** `.aether/ts-host/src/platform-dispatcher.ts:250-265`

The `buildArgs()` function for `codex` builds:
```
codex exec --json --ephemeral --skip-git-repo-check --output-schema /tmp/schema.json
```

It never passes `config.prompt`. The Codex CLI `exec` subcommand expects the prompt via stdin or as a positional argument. The assembled prompt from `prompt-assembler.ts` is silently dropped.

**Fix:** Pass prompt as final positional arg or via stdin.

### 2. No platform-specific arg tests (DSP-02, DSP-03)
**File:** `.aether/ts-host/test/platform-dispatcher.test.ts`

The `spawnWorker` test overrides `AETHER_CODEX_PATH` to `node`, which ignores Codex-specific flags. There are no tests verifying:
- Claude gets `-p`, `--output-format json`, `--json-schema`, `--agent`, `--permission-mode`, and the prompt
- OpenCode gets `run`, `--agent`, `--format json`, and the prompt
- Codex gets `exec`, `--json`, `--ephemeral`, `--output-schema`, and the prompt

**Fix:** Export `buildArgs` and test each platform's arg array.

### 3. Simulation fallback masks broken dispatch (DSP-04)
**File:** `.aether/ts-host/src/worker-dispatch.ts:142`

```typescript
const simulate = opts.simulateWorkers !== false; // default true
```

This silently simulates when `simulateWorkers` is undefined. There's no log message, no warning, no way to tell if real dispatch would have worked. If a user expects real workers but forgets `simulateWorkers: false`, they get fake results with no indication.

**Fix:** Log a clear warning on simulation, and add an explicit `simulateWorkers: true` requirement (fail if undefined and no platforms available).

### 4. Spawn-log completeness (DSP-05)
**File:** `.aether/ts-host/src/worker-dispatch.ts:111-200`

`dispatchSingleWorker` already calls `spawn-log` before dispatch and `spawn-complete` after. It records manifest workers via name/caste/task. No internal/system workers are dispatched through this path. **Likely already correct** — verification needed.

## Files to Modify
- `src/platform-dispatcher.ts` — fix Codex args, export `buildArgs`
- `src/worker-dispatch.ts` — explicit simulation fallback warning
- `test/platform-dispatcher.test.ts` — test arg construction per platform

## Verification Target
- `npm test` passes
- `npm run typecheck` passes
- New tests cover all 3 platform arg patterns
