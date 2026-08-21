---
phase: 153-ts-scaffold-and-schemas
plan: 02
status: complete
completed: "2026-05-23"
---

# Plan 153-02 Summary: Zod Schemas & Validation Tests

## What Was Built

Implemented Zod schemas for all four colony asset types and wrote Vitest tests that validate each schema against the sample fixtures from Plan 01.

### Files Created

| File | Purpose |
|------|---------|
| `control-ts/src/schemas/agent.schema.ts` | `AgentSchema` with `superRefine` file-existence and path-traversal checks |
| `control-ts/src/schemas/phase.schema.ts` | `PhaseSchema` with required fields, defaults, and optional ceremony |
| `control-ts/src/schemas/event.schema.ts` | `EventSchema` with ISO datetime validation and `parseEventLine` helper |
| `control-ts/src/schemas/policy.schema.ts` | `PolicySchema` with nested defaults for model_routing, memory_rules, skill_creation, safety_gates |
| `control-ts/src/utils/projectRoot.ts` | CWD-safe `projectRoot` resolver using `import.meta.url` |
| `control-ts/tests/schemas/agent.schema.test.ts` | 5 tests: valid fixtures, missing prompt_file, path traversal, empty id |
| `control-ts/tests/schemas/phase.schema.test.ts` | 4 tests: valid fixtures, missing required_agents, missing success_criteria |
| `control-ts/tests/schemas/event.schema.test.ts` | 4 tests: valid NDJSON lines, empty line, invalid JSON, missing timestamp |
| `control-ts/tests/schemas/policy.schema.test.ts` | 4 tests: valid fixture, empty object defaults, empty provider rejection |
| `control-ts/src/types/index.ts` | Re-exports `Agent`, `Phase`, `ColonyEvent`, `Policy` inferred types |
| `control-ts/src/index.ts` | Public API barrel re-exporting all schemas, types, and `parseEventLine` |
| `control-ts/colony/prompts/queen.md` | Placeholder prompt file so fixture validation passes |
| `control-ts/colony/prompts/builder.md` | Placeholder prompt file so fixture validation passes |

### Key Decisions

- **Zod 4 vs Zod 3:** The scaffold installed Zod 4.4.3 (Zod 3.25.x did not exist in npm). Schemas were written with Zod 4 syntax — `z.string().datetime({ offset: true })`, `z.enum([...])`, etc. All patterns work identically to the planned Zod 3 approach.
- **Path traversal guard:** `agent.schema.ts` uses `superRefine` (not `refine`) so both the `..` check and the file-existence check can emit separate `ZodIssueCode.custom` issues.
- **projectRoot utility:** Created `src/utils/projectRoot.ts` to avoid `process.cwd()` pitfalls — resolves relative to the module file location via `import.meta.url`.

### Test Results

```
Test Files  4 passed (4)
Tests  17 passed (17)
Duration  277ms
```

All tests pass via `npm run test:schemas` in `control-ts/`.

### Security Mitigations

| Threat | Status |
|--------|--------|
| T-153-01 Path traversal in `prompt_file` | Mitigated — `superRefine` rejects `..` segments |
| T-153-02 YAML bomb / excessive anchors | Mitigated — fixtures are small; yaml package defaults used |
| T-153-03 Malformed NDJSON injection | Mitigated — `parseEventLine` wraps `JSON.parse` in try/catch with line context |

## Deviations

- None.
