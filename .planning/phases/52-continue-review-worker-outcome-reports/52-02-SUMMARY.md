---
phase: 52-continue-review-worker-outcome-reports
plan: 02
status: complete
started: "2026-04-26T10:44:00Z"
completed: "2026-04-26T11:00:00Z"
---

## Plan 52-02: Wrapper Completion Packet Documentation

### What Changed

Updated both Claude and OpenCode continue command wrappers to document the new `report` field in the worker completion packet.

### Changes Made

1. **Worker return requirement** (line 59 in both files): Added `report` to the list of required structured result fields
2. **Completion packet JSON example**: Added `"report"` field with example markdown content showing how review worker findings should be formatted
3. **Guidance prose**: Added paragraph explaining the `report` field is optional but strongly recommended for review workers (Watcher, Gatekeeper, Auditor, Probe), and what happens when omitted

### Files Modified

- `.claude/commands/ant/continue.md` — 3 changes (requirement list, JSON example, guidance prose)
- `.opencode/commands/ant/continue.md` — 3 identical changes (platform parity)

### Acceptance Criteria

- [x] Both files contain `"report":` in the completion packet JSON example
- [x] Both files list `report` in the worker return requirement
- [x] Both files include the guidance paragraph about the report field
- [x] Both files are structurally aligned (same changes in both)

### Deviations

None — all changes applied as specified in the plan.

### key-files

- modified: .claude/commands/ant/continue.md
- modified: .opencode/commands/ant/continue.md
