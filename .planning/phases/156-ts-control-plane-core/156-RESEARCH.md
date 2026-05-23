# Phase 156: TS Control Plane Core — Research

**Phase:** 156
**Date:** 2026-05-23
**Source:** ROADMAP.md, REQUIREMENTS.md, control-ts/ existing state

---

## Scope

Build the TypeScript control plane that can load colony assets and execute phases.

Requirements: CONTROL-02, CONTROL-03, CONTROL-04, CONTROL-05, CONTROL-06, CONTROL-09, CONTROL-10

---

## Existing State (from Phase 153)

```
control-ts/
├── package.json          # Has zod, yaml deps
├── tsconfig.json
├── vitest.config.ts
├── src/
│   ├── index.ts          # Almost empty — just version export
│   ├── loaders/          # EMPTY — created but no files
│   ├── schemas/          # AgentSchema, PhaseSchema, EventSchema, PolicySchema
│   ├── types/            # index.ts with version only
│   └── utils/            # projectRoot.ts
└── tests/
    ├── schemas/          # schema tests
    ├── fixtures/         # agents, phases, policies fixtures
    └── helpers.ts
```

Schemas already validate against colony/ assets from Phase 154.

---

## What Needs to Be Built

### 1. Agent Loader (CONTROL-02)
**File:** `control-ts/src/agents/loadAgents.ts`
- Read `colony/agents/*.yaml`
- Parse YAML
- Validate each with `AgentSchema`
- Return typed `Agent[]` array
- Handle file-not-found gracefully

### 2. Phase Loader (CONTROL-03)
**File:** `control-ts/src/phases/loadPhases.ts`
- Read `colony/phases/*.yaml`
- Parse YAML
- Validate each with `PhaseSchema`
- Return typed `Phase[]` array

### 3. Prompt Assembler (CONTROL-04)
**File:** `control-ts/src/prompts/assemblePrompt.ts`
- Read `colony/prompts/{role}.md`
- Load agent YAML to get `prompt_file` reference
- Return assembled prompt string
- Support optional context injection (skills, pheromones)

### 4. Phase Runner (CONTROL-05)
**File:** `control-ts/src/orchestrator/runPhase.ts`
- Accept phase ID
- Load phase YAML
- Select entry agent from `entry_agent` field
- Emit events to NDJSON stream
- Return execution result

### 5. Plan Executor (CONTROL-06)
**File:** `control-ts/src/orchestrator/executePlan.ts`
- Accept plan sequence (array of phase IDs)
- Run phases in order: init → plan → build → verify → seal
- Handle phase dependencies and failure policies
- Emit events for each step
- Return final result

### 6. Skills Loader (CONTROL-09)
**File:** `control-ts/src/skills/loadSkills.ts`
- Read skills from `.aether/skills/` and `~/.aether/skills/`
- Parse SKILL.md frontmatter
- Match skills to workers by role + task + pheromones
- Return matched skills for injection

### 7. Memory Read/Write (CONTROL-10)
**File:** `control-ts/src/memory/store.ts`
- Read/write JSON to `.aether/data/COLONY_STATE.json`
- Provide typed accessors for colony state fields
- Use file locking or atomic writes

---

## Directory Structure Target

```
control-ts/src/
├── index.ts                    # Re-export public API
├── agents/
│   └── loadAgents.ts           # CONTROL-02
├── phases/
│   └── loadPhases.ts           # CONTROL-03
├── prompts/
│   └── assemblePrompt.ts       # CONTROL-04
├── orchestrator/
│   ├── runPhase.ts             # CONTROL-05
│   └── executePlan.ts          # CONTROL-06
├── skills/
│   └── loadSkills.ts           # CONTROL-09
├── memory/
│   └── store.ts                # CONTROL-10
├── schemas/                    # Already exists
│   ├── agent.schema.ts
│   ├── phase.schema.ts
│   ├── event.schema.ts
│   └── policy.schema.ts
├── types/
│   └── index.ts                # Shared types
└── utils/
    └── projectRoot.ts          # Already exists
```

---

## Testing Strategy

Each module gets its own test file:
- `tests/agents/loadAgents.test.ts` — load all 27 agents, validate schema, check prompt_file resolution
- `tests/phases/loadPhases.test.ts` — load all 9 phases, validate schema
- `tests/prompts/assemblePrompt.test.ts` — assemble builder prompt, verify content
- `tests/orchestrator/runPhase.test.ts` — run init phase, verify event emission
- `tests/orchestrator/executePlan.test.ts` — run mini sequence, verify order
- `tests/skills/loadSkills.test.ts` — load and match skills
- `tests/memory/store.test.ts` — read/write colony state

All tests use existing fixtures under `tests/fixtures/`.

---

## Key Design Decisions

1. **NDJSON event stream:** Use append-only NDJSON to `.aether/events/current.ndjson` for event emission. This is the shared interface with Go.
2. **File paths:** All paths relative to repo root (resolved via `projectRoot.ts`).
3. **Error handling:** Validate early, fail fast. If a YAML file fails schema validation, throw with file path and Zod error details.
4. **No external API calls in this phase:** Adapters are Phase 157. Phase 156 is purely file I/O and orchestration.

---

## Integration Points

- `loadAgents.ts` reads `colony/agents/*.yaml` (created in Phase 154)
- `loadPhases.ts` reads `colony/phases/*.yaml` (created in Phase 154)
- `assemblePrompt.ts` reads `colony/prompts/*.md` (created in Phase 154)
- `executePlan.ts` will eventually use adapters from Phase 157

---

## Verification

- `cd control-ts && npm run typecheck` — must pass
- `cd control-ts && npm test` — must pass
- `cd control-ts && npx tsx src/agents/loadAgents.ts` — must print 27 agents
- `cd control-ts && npx tsx src/phases/loadPhases.ts` — must print 9 phases
