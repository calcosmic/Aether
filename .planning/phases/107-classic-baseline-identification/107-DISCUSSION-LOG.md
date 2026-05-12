# Phase 107 Discussion Log

**Date:** 2026-05-12
**Phase:** Classic Baseline Identification

## Areas Discussed

### Comparison Depth
**Options presented:**
- Summary table (1-page comparison with key differences)
- Full behavioral checklist (3-4 pages, module-by-module)

**Selection:** Full behavioral checklist

**Follow-up — Classification:**
- Module-by-module with 4-category classification (Restore in TS / Keep in Go / Obsolete / Reject)
- Module-by-module without classification

**Selection:** Module-by-module with classification

**Follow-up — Version coverage:**
- All 3 versions (v5.3.0, v5.3.3, v5.4.0)
- v5.4.0 only

**Selection:** All 3 versions

### Smoke Test Scope
**Options presented:**
- Exit codes only
- Exit codes + output patterns
- Full lifecycle verification (exit codes + output patterns + state changes)

**Selection:** Full lifecycle verification

**Follow-up — Implementation format:**
- Bash script
- Go test

**Selection:** "You decide" — Claude chose Bash script for simplicity and CI portability

## Decisions Captured

| ID | Decision | Rationale |
|----|----------|-----------|
| D-01 | Full behavioral checklist comparing all 3 versions | Thorough anchor for hybrid runtime work |
| D-02 | Each module includes 4-category classification | Directly feeds Phase 109 TS host work |
| D-03 | Show what changed across versions | Explains why selected version is the bridge |
| D-04 | Full lifecycle verification smoke test | Catches silent failures and state issues |
| D-05 | Bash script (scripts/smoke-test-classic.sh) | Simple, CI-friendly, independent of Go runtime |

## Deferred Ideas

None
