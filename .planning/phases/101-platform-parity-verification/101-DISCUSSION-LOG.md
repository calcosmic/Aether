# Phase 101: Platform Parity Verification - Discussion Log

**Date:** 2026-05-07
**Mode:** Default (interactive)

## Areas Discussed

### 1. Parity gap severity

**Q1: How should parity mismatches be classified?**
- Options: Binary pass/fail, Three-tier (Critical/Warning/Info), Functional only
- **Selected:** Three-tier (Recommended) — matches Phase 100's code review pattern

**Q2: Should the parity report include fix suggestions?**
- Options: Counts only, Counts + fix hints
- **Selected:** Counts only (Recommended) — researcher/planner decide fixes

**Q3: Should wrappers referencing phantom commands be Critical?**
- Options: Yes phantom = Critical, No phantoms allowed
- **Selected:** Yes, phantom = Critical (Recommended)

### 2. Codex coverage scope

**Q4: How should the 33 missing Codex entries be handled?**
- Options: Flag all 33 as Info gaps, Verify 27 only, Severity by command type
- **Selected:** Flag all 33 as Info gaps (Recommended) — establishes full parity picture

**Q5: Should any missing Codex entries be higher severity than Info?**
- Options: No Info only, Lifecycle commands = Warning
- **Selected:** Lifecycle commands = Warning

### 3. Test freeze approach

**Q6: Freeze current state or enforce ideal parity?**
- Options: Freeze current state, Enforce ideal parity
- **Selected:** Freeze current state (Recommended) — tests pass today, Phase 105 resolves gaps

**Q7: What should the parity golden test freeze?**
- Options: Snapshot names only, Snapshot names + flags, Snapshot everything
- **Selected:** Snapshot names only (Recommended) — maintainable, catches most drift

**Q8: Single combined test or per-surface-pair tests?**
- Options: Single combined test, Per-surface-pair tests
- **Selected:** Single combined test (Recommended) — simpler, report identifies which surface

## Summary

8 decisions captured across 3 areas. No deferred ideas.
