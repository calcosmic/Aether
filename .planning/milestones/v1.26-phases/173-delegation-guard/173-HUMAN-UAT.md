---
status: partial
phase: 173-delegation-guard
source: [173-VERIFICATION.md]
started: 2026-08-13T14:30:00Z
updated: 2026-08-13T14:30:00Z
audit_acknowledged:
  milestone: v1.26
  at: 2026-08-22
  gap_snapshot: "partial::scenarios=1"
---

## Current Test

[awaiting human testing]

## Tests

### 1. Budget refusal is visible in the normal build narration

When a helper spawn is refused during a real build (depth cap, whole-run
budget ceiling, cycle refusal, or corrupted-ledger refusal), the refusal
should be explained in the plain-English on-screen narration the operator
actually reads — not only in a technical stderr line or JSON payload they
would have to dig for.

expected: During a real `/ant-build` or `/ant-continue` run that hits a
spawn refusal, the narration shows a readable explanation of what was
refused and why (e.g. "the colony's helper budget for this run is used
up"), without the operator needing to inspect raw command output. No code
in phase 173 touched the wrapper/narration files, so this has never been
observed — it needs a real run that actually triggers a refusal.
result: [pending]

## Summary

total: 1
passed: 0
issues: 0
pending: 1
skipped: 0
blocked: 0

## Gaps
