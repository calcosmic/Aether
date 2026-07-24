---
phase: 153-ts-scaffold-and-schemas
plan: 01
subsystem: control-plane
tags: [typescript, zod, yaml, vitest, node, esm]

requires:
  - phase: 152-boundary-parity
    provides: Architecture boundary decisions (Go stays spine, TS owns orchestration)
provides:
  - Runnable Node library project at control-ts/ with npm install, tsc, and vitest
  - Strict TypeScript configuration (ES2022, NodeNext, strict true, declaration emit)
  - Sample YAML/NDJSON fixtures for all four colony asset types (agent, phase, policy, event)
  - Shared test helper resolving fixture paths via import.meta.url (cwd-safe)
affects:
  - 153-02 (Zod schemas will validate these fixtures)
  - 156 (TS control plane core will import from this scaffold)
  - 158 (Event stream will use the NDJSON fixture pattern)

tech-stack:
  added: [typescript 6.0.3, zod 4.4.3, yaml 2.9.0, vitest 4.1.7, tsx 4.22.3, @types/node 25.9.1]
  patterns:
    - ESM-only ("type": "module") with NodeNext module resolution
    - Flat src/ structure organised by domain (schemas/, types/, loaders/, utils/)
    - Fixture-driven testing with cwd-safe path resolution
    - Minimal toolchain: tsc + tsx + vitest, no web framework baggage

key-files:
  created:
    - control-ts/package.json
    - control-ts/tsconfig.json
    - control-ts/vitest.config.ts
    - control-ts/src/index.ts
    - control-ts/src/types/index.ts
    - control-ts/tests/helpers.ts
    - control-ts/tests/fixtures/agents/queen.yaml
    - control-ts/tests/fixtures/agents/builder.yaml
    - control-ts/tests/fixtures/phases/init.yaml
    - control-ts/tests/fixtures/phases/plan.yaml
    - control-ts/tests/fixtures/policies/model-routing.yaml
    - control-ts/tests/fixtures/events/sample.ndjson
  modified: []

key-decisions:
  - "Adjusted dependency versions to match npm registry availability (vitest 4.1.7, typescript 6.0.3, zod 4.4.3, @types/node 25.9.1)"
  - "Kept ESM-only with NodeNext to align with modern Node library best practices"
  - "Used import.meta.url-based fixture resolution to avoid cwd-relative path pitfalls"

patterns-established:
  - "Fixture-driven contract tests: hand-written YAML/NDJSON fixtures in tests/fixtures/ serve as schema validation contracts"
  - "Cwd-safe path resolution: all fixture paths resolve from import.meta.url, never process.cwd()"
  - "Minimal toolchain: no Vite/Next.js/web framework — pure Node library"

requirements-completed:
  - CONTROL-01
---

# Phase 153 Plan 01: TypeScript Scaffold & Fixtures Summary

**Runnable TypeScript control plane scaffold with strict ESM config, Vitest test runner, and sample colony asset fixtures for all four schema types.**

## Performance

- **Duration:** ~4 min
- **Started:** 2026-05-22T22:18:47Z
- **Completed:** 2026-05-22T22:23:00Z
- **Tasks:** 3
- **Files modified:** 12

## Accomplishments
- Created control-ts/ project with package.json, tsconfig.json, and vitest.config.ts
- Installed all dependencies successfully (zod, yaml, typescript, vitest, tsx, @types/node)
- TypeScript compilation (tsc --noEmit) passes with zero errors
- Built complete directory structure: src/{schemas,types,loaders,utils}, tests/{fixtures,schemas}
- Wrote six syntactically valid fixture files covering agents, phases, policies, and events
- Added cwd-safe test helper (fixturePath) using import.meta.url resolution

## Task Commits

Each task was committed atomically:

1. **Task 1: Create project manifest and TypeScript configuration** - `3ac7ba3c` (feat)
2. **Task 2: Create public API entry point, types barrel, and test helpers** - `f96c596a` (feat)
3. **Task 3: Write sample YAML and NDJSON test fixtures** - `cf9a9d06` (test)

## Files Created/Modified
- `control-ts/package.json` - Project manifest with ESM scripts and dependencies
- `control-ts/tsconfig.json` - Strict TS config targeting ES2022 with NodeNext module resolution
- `control-ts/vitest.config.ts` - Vitest config with node environment and tests/**/*.test.ts include
- `control-ts/src/index.ts` - Public API entry point exporting AETHER_CONTROL_VERSION
- `control-ts/src/types/index.ts` - Types barrel (placeholder for z.infer re-exports)
- `control-ts/tests/helpers.ts` - Shared test utilities with projectRoot and fixturePath
- `control-ts/tests/fixtures/agents/queen.yaml` - Sample queen agent fixture
- `control-ts/tests/fixtures/agents/builder.yaml` - Sample builder agent fixture
- `control-ts/tests/fixtures/phases/init.yaml` - Sample init phase fixture
- `control-ts/tests/fixtures/phases/plan.yaml` - Sample plan phase fixture
- `control-ts/tests/fixtures/policies/model-routing.yaml` - Sample policy fixture
- `control-ts/tests/fixtures/events/sample.ndjson` - Sample NDJSON event lines

## Decisions Made
- Adjusted dependency versions to match currently available npm registry versions (vitest 4.1.7 instead of 3.2.7, typescript 6.0.3 instead of 5.8.3, zod 4.4.3 instead of 3.25.2, @types/node 25.9.1 instead of 22.0.0)
- Kept ESM-only with NodeNext module resolution to align with modern Node library patterns
- Used import.meta.url-based fixture resolution to avoid cwd-relative path pitfalls documented in 153-RESEARCH.md

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Updated dependency versions to match npm registry availability**
- **Found during:** Task 1 (npm install)
- **Issue:** vitest@3.2.7, typescript@5.8.3, zod@3.25.2, and @types/node@22.0.0 did not exist in the npm registry, causing `npm install` to fail with ETARGET
- **Fix:** Queried npm registry for latest stable versions and updated package.json to vitest@^4.1.7, typescript@^6.0.3, zod@^4.4.3, @types/node@^25.9.1
- **Files modified:** control-ts/package.json
- **Verification:** npm install completed successfully with 0 vulnerabilities
- **Committed in:** 3ac7ba3c (Task 1 commit)

---

**Total deviations:** 1 auto-fixed (1 blocking)
**Impact on plan:** Version adjustment necessary for the project to build at all. No scope creep.

## Issues Encountered
- Vitest v4 exits with code 1 when no test files are found (expected — test files will be created in Plan 02). The runner itself executes without configuration errors.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Scaffold is complete and building; ready for Plan 02 (Zod schema implementation)
- Fixtures exist and are syntactically valid; ready to be consumed by schema tests
- No blockers

## Self-Check: PASSED

- All 13 created files verified on disk
- All 5 commits verified in git log
- npm install completes with 0 vulnerabilities
- npx tsc --noEmit exits with code 0
- All YAML and NDJSON fixtures parse successfully

---
*Phase: 153-ts-scaffold-and-schemas*
*Completed: 2026-05-22*
