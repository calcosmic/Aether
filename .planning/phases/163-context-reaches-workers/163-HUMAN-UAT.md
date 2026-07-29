---
status: partial
phase: 163-context-reaches-workers
source: [163-VERIFICATION.md]
started: 2026-07-29T18:30:00Z
updated: 2026-07-29T18:30:00Z
---

## Current Test

[awaiting human testing]

## Tests

### 1. `aether build <n> --print-brief` ten-second charter check on a real colony
expected: On a real, populated colony (not synthetic test fixtures), the checklist output lets a person answer "did the charter arrive?" within ten seconds — the Charter row shows PRESENT with a plausible character count when a charter is approved, and the manifest-level Context Capsule row is populated.
result: [pending]

### 2. Local COLONY_STATE.json suggestion pollution cleanup
expected: After cleanup (orchestrator already ran `aether suggest-approve --dismiss-all` on 2026-07-29 — all 136 junk entries now carry `dismissed: true`), one real `aether build` end-to-end shows a "Suggestions From This Build" closeout block that is small and relevant, not a wall of ~136 stale entries.
result: [pending — cleanup done, real-build confirmation outstanding]

### 3. CONTEXT-09 real-world cheap-model validation (deferred by design per D-08)
expected: A real build on an inexpensive model with this phase's context restored shows visibly improved worker behavior versus a build without it; documented informally through normal use, not a staged benchmark.
result: [pending]

## Summary

total: 3
passed: 0
issues: 0
pending: 3
skipped: 0
blocked: 0

## Gaps
