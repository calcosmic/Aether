# Plan 156-01 Summary — TS Control Plane Core Asset Loaders

## Deliverables

| File | Status |
|------|--------|
| `control-ts/src/agents/loadAgents.ts` | Created — exports `loadAgents()`, `loadAgentById()` |
| `control-ts/src/phases/loadPhases.ts` | Created — exports `loadPhases()`, `loadPhaseById()` |
| `control-ts/src/prompts/assemblePrompt.ts` | Created — exports `assemblePrompt()`, `assemblePromptForAgent()` |
| `control-ts/tests/agents/loadAgents.test.ts` | Created — 5 tests passing |
| `control-ts/tests/phases/loadPhases.test.ts` | Created — 5 tests passing |
| `control-ts/tests/prompts/assemblePrompt.test.ts` | Created — 8 tests passing |
| `control-ts/src/types/index.ts` | Updated — exports Agent, Phase, ColonyEvent, Policy |
| `control-ts/src/index.ts` | Updated — barrel re-exports all public APIs |

## Test Results

```
Test Files  7 passed (7)
     Tests  43 passed (43)
```

All existing schema tests continue to pass. New loader tests verify:
- 27 agents loaded and validated against AgentSchema
- 9 phases loaded and validated against PhaseSchema
- Prompt assembly for builder and queen roles
- Context injection (skills + pheromones) appends correctly
- Missing files throw clear errors
- `loadAgentById` / `loadPhaseById` return correct objects or null

## Build Results

```
npm run typecheck  -> 0 errors, 0 warnings
npm test           -> 43 tests passing, 0 failing
```

## Notes

- Colony assets (agents, phases, prompts) are read from the repo root `colony/` directory (one level above `control-ts/`).
- Prompt files from repo root `colony/prompts/` were copied into `control-ts/colony/prompts/` so that `AgentSchema.superRefine` (which resolves `prompt_file` relative to `projectRoot` = `control-ts/`) continues to pass validation.
- A pre-existing Zod v4 API incompatibility in `policy.schema.ts` (`z.record` with single arg + `.default`) was fixed by explicitly passing key and value types to `z.record`.
- Spot-checks confirm: `loadAgents()` returns 27, `loadPhases()` returns 9.
