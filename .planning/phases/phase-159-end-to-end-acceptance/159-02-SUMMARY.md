# Plan 02 Summary: End-to-End Acceptance — Demo CLI & Audits

**Status:** Complete
**Date:** 2026-05-24

## What Was Built

1. **Demo CLI entry point** (`control-ts/src/cli.ts`)
   - Parses `--task` argument with a default task
   - Calls `executePlan(["init", "plan", "build"])` with stub adapters
   - Reads and parses NDJSON events from `.aether/events/current.ndjson`
   - Reports plan status, phases executed, event count, and event types

2. **Hardcoded behaviour audit script** (`scripts/audit-hardcoded.sh`)
   - Checks for `"You are"` directive strings in non-test Go files
   - Checks for ceremony/ritual/playbook strings in non-test Go files
   - Checks for `"Your task is to"` directive strings in non-test Go files
   - Allows fallback and error text per the boundary contract

3. **Extraction audit script** (`scripts/audit-extraction.sh`)
   - Verifies agent YAML files exist in `colony/agents/` (>= 8)
   - Verifies prompt Markdown files exist in `colony/prompts/` (>= 8)
   - Verifies phase YAML files exist in `colony/phases/` (>= 6)
   - Checks for agent personality strings only in Go
   - Validates the extraction audit doc from Phase 152 has >= 90 classified symbols

## Commits

- `49c7c46c` feat(159-02): add demo CLI entry point for end-to-end acceptance
- `7e7d8390` feat(159-02): add hardcoded behaviour audit script
- `7393cb7b` feat(159-02): add extraction audit script
- `2201fcb0` fix(159-02): extract oracle directives to colony asset and refine audit

## Verification

- `npm run aether:control -- --task "create a small test file and verify it"` completes with events
- `bash scripts/audit-hardcoded.sh` passes all checks
- `bash scripts/audit-extraction.sh` passes all checks

## Issues Encountered

- Agent got stuck in a loop checking git log repeatedly. Work was complete; SUMMARY.md was written manually.
