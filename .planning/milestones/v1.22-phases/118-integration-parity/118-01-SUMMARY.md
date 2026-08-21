---
phase: 118-integration-parity
plan: 01
status: complete
completed: "2026-05-14"
---

# Plan 118-01 Summary: Golden Workflow + Ceremony Snapshot Tests

## What Was Built

Captured baseline ceremony snapshots and created deterministic snapshot tests for all ceremony template outputs. These tests prove the restored system matches Classic v5.4 ceremony rendering.

### Changes

1. **Ceremony snapshot tests** (`.aether/ts-host/test/ceremony-snapshots.test.ts`)
   - 10 snapshot tests covering: banner (BUILD, CROWNED ANTHILL), spawn frame (builder, oracle), stage separators (Build, Continue), boxen output (build-summary, closeout-ritual)
   - Plain-text snapshot strategy (no external library): `loadSnapshot`/`saveSnapshot` helpers
   - `AETHER_UPDATE_SNAPSHOTS=1` env var for intentional baseline regeneration
   - Additional assertions: markdownRenderer strips ANSI, jsonRenderer returns empty strings

2. **Golden workflow test** (`.aether/ts-host/test/golden-workflow.test.ts`)
   - End-to-end lifecycle test: plan → build → continue with `simulateWorkers: true`
   - Captures all ceremony stdout output during lifecycle run
   - Normalizes output before snapshot comparison: timestamps → `<TIMESTAMP>`, tmpdir → `<TMPDIR>`, PIDs → `<PID>`, durations → `<DURATION>`
   - Stores baseline in `test/__snapshots__/golden-workflow-spawn-tree.txt`

3. **Snapshot files** (`.aether/ts-host/test/__snapshots__/`)
   - 9 `.txt` snapshot files for deterministic renderer output comparison

4. **Test documentation** (`.aether/ts-host/test/README.md`)
   - Documents snapshot test purpose, running instructions, update procedure, and normalization rules

5. **npm script** (`.aether/ts-host/package.json`)
   - Added `test:update-snapshots`: `AETHER_UPDATE_SNAPSHOTS=1 tsx --test test/*.test.ts`

### Test Results

- `npx tsx --test test/ceremony-snapshots.test.ts` — **10/10 PASS**
- `npx tsx --test test/golden-workflow.test.ts` — **1/1 PASS**

### Key Decisions

- Plain text snapshots over jest-snapshot: simpler, no extra dependency, readable in git diffs
- Normalization rules make golden workflow stable across machines and runs
- Auto-create missing snapshots on first run (with warning) to reduce friction
