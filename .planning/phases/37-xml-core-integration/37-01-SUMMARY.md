---
phase: 37-xml-core-integration
plan: 01
status: complete
started: 2026-03-29
completed: 2026-03-29
---

## Plan 37-01: Seal/Entomb XML Exchange

### Objective
Add wisdom and registry XML export to seal's Step 6.5, and add exchange/ XML archiving to entomb's chamber archiving step.

### What Was Built
- **seal.yaml**: Added `wisdom-export-xml` and `registry-export-xml` dispatcher calls to both body_claude (Step 6.5) and body_opencode (Step 5.75), exporting queen-wisdom.xml and colony-registry.xml to .aether/exchange/. All exports follow best-effort, non-blocking pattern per D-01/D-02.
- **entomb.yaml**: Added exchange/ XML file copy to chamber directory in body_claude (Step 7) and body_opencode (Step 6.5), with exchange cleanup during colony reset (body_claude Step 10, body_opencode Step 8). Per D-04/D-05/D-06.

### Tasks Completed
| # | Task | Status |
|---|------|--------|
| 1 | Add wisdom + registry XML export to seal.yaml | Done |
| 2 | Add exchange/ XML archiving to entomb.yaml | Done |

### Key Decisions
- Followed existing best-effort pattern for all XML exports (non-blocking per D-02)
- Exchange XML cleanup happens during colony reset, after archiving to chamber
- Display lines show export counts for wisdom entries, registry colonies, and exchange files

### Files Modified
- `.aether/commands/seal.yaml` — wisdom + registry XML export in Step 6.5
- `.aether/commands/entomb.yaml` — exchange/ XML archiving + cleanup

### Issues
None.

## Self-Check: PASSED
