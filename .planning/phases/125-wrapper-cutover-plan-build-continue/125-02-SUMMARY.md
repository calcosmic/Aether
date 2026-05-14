# Plan 125-02 Summary: Claude/OpenCode Wrapper Cutover

**Phase:** 125 — Wrapper Cutover (Plan/Build/Continue)  
**Plan:** 125-02  
**Status:** Complete  
**Commit:** 868d9a42

## What Was Done

Rewrote all six wrapper markdown files to delegate orchestration to `aether host plan|build|continue` instead of manually performing plan-only/finalizer/spawn-log steps.

### Files Modified

| File | Before | After | Reduction |
|------|--------|-------|-----------|
| `.claude/commands/ant/plan.md` | 241 | 106 | 56% |
| `.claude/commands/ant/build.md` | 182 | 110 | 40% |
| `.claude/commands/ant/continue.md` | 127 | 89 | 30% |
| `.opencode/commands/ant/plan.md` | 241 | 157 | 35% |
| `.opencode/commands/ant/build.md` | 182 | 111 | 39% |
| `.opencode/commands/ant/continue.md` | 127 | 89 | 30% |
| **Total** | **1100** | **662** | **40%** |

### Key Changes

1. **Primary orchestration:** All wrappers now call `aether host plan|build|continue` as the primary manifest-fetch path
2. **Fallback:** Each wrapper documents the direct Go CLI fallback when TS host is unavailable
3. **Finalizers remain direct Go CLI:** `aether plan-finalize`, `aether build-finalize`, `aether continue-finalize` are still called directly (TS host does not handle finalizers per boundary contract)
4. **Removed:** Inline multi-step CLI command chains, spawn-log logic, manual ceremony rendering instructions
5. **Preserved:** Generated-from-YAML headers, guardrails, cross-platform drift guards, wrapper-specific concerns (depth ceremony, signal presentation, error handling)

### Verification

- All six wrappers contain `aether host` in their execution section
- All six wrappers contain their respective finalizer command
- No inline multi-step Go CLI command chains remain
- All wrappers retain the "Generated from" header
- Each individual wrapper achieved ≥30% line reduction
- Total reduction: 40% (438 lines removed)

## Deviation from Plan

None. All tasks executed as specified.
