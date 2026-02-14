# Issues Identified in Dream Session (2026-02-14)

## Issue 1: Split Brain Between Planning and Runtime State

**Problem:**
- `.planning/STATE.md` tracks v1.1 milestone, Phase 6 complete, ready for Phase 7
- `.aether/data/COLONY_STATE.json` says "COMPLETED" with goal "Fix loop bugs" and zero phases

**Impact:**
- Human reads STATE.md: "Phase 7 is next"
- Code reads COLONY_STATE.json: "nothing to do, we're done"
- The colony runtime doesn't know about the v1.1 work

**Root Cause:**
- STATE.md is for Aether developers (meta-project tracking)
- COLONY_STATE.json is for colony runtime (user goal tracking)
- They serve different purposes but the colony should sync them

**Fix:**
Add a plan (07-07) to create state synchronization:
1. Update COLONY_STATE.json with current v1.1 goal
2. Add phases from ROADMAP to COLONY_STATE.json plan.phases
3. Create utility to sync STATE.md → COLONY_STATE.json
4. Ensure status commands read from both sources

---

## Issue 2: Model Routing Configuration vs Execution

**Problem:**
- `model-profiles.yaml` has sophisticated routing configuration
- `build.md` documents setting ANTHROPIC_MODEL per caste
- But no verification that it actually happens

**Evidence:**
```bash
# build.md says to do this:
export ANTHROPIC_MODEL="$model"  # From model-profile get
```

But the Task tool inherits environment from parent Claude Code process - the exports in build.md are just documentation, not actual execution.

**Impact:**
- All workers may use default model regardless of task complexity
- Wasted API costs on complex tasks with simple models
- Suboptimal results on simple tasks with overkill models

**Fix:**
Add verification to plan 07-06:
1. Check that ANTHROPIC_MODEL is actually set before Task spawns
2. Add logging to confirm model assignment
3. Create test that verifies model routing is active

---

## Issue 3: Command Duplication Debt

**Problem:**
- 28 commands in `.claude/commands/ant/`
- 28 commands in `.opencode/commands/ant/`
- Manually duplicated, already drifted (diff shows differences)

**Evidence:**
```bash
$ diff -r .claude/commands/ant/ .opencode/commands/ant/
# Shows differences in build.md (steps 0.5, 0.6 missing in opencode)
```

**Impact:**
- Changes must be manually copied to both locations
- Risk of divergence (already happening)
- Technical debt that grows with each command update

**Fix:**
This is architectural debt, not a Phase 7 bug fix. Recommend:
- Add to v1.2 roadmap (feature: "YAML-based command generation")
- Short-term: create sync verification in CI
- Not blocking Phase 7

---

## Issue 4: Error Class Hierarchy Without Consumers

**Problem:**
- `bin/lib/errors.js` maps error codes to sysexits.h codes
- Beautiful Unix convention from 1983
- But nothing actually consumes these exit codes

**Impact:**
- Speaking precise language but no one is listening
- Exit codes are effectively documentation, not functional

**Fix:**
Low priority - this is refinement, not a bug. The errors work, just the exit codes aren't consumed. Could be enhanced in future.

---

## Issue 5: Checkpoint Allowlist (Working Well)

**Observation:**
- Phase 6 checkpoint system has explicit allowlist
- 91 system files safe to capture
- User data explicitly excluded
- This is working correctly

**Status:** ✅ No fix needed - this is the success story

---

# Recommended Actions for Phase 7

## Option A: Fix Issues Inline (Recommended)

Add these tasks to existing plans:

**07-06 (Initialization & Integration):**
- Update COLONY_STATE.json with v1.1 goal and phases
- Create state sync utility
- Verify model routing is actually happening

## Option B: Add New Plan (07-07)

Create a new plan specifically for:
- State synchronization system
- Model routing verification
- State reconciliation between .planning/ and .aether/data/

## Option C: Defer Non-Critical Issues

- Command duplication → v1.2 (feature, not bug)
- Exit code consumers → Future enhancement
- Focus Phase 7 on critical bugs only

---

# Decision Needed

Which option should we take? The split brain is the most critical - the colony runtime doesn't know about the v1.1 work, so commands like `/ant:status` and `/ant:build` will give wrong information.

My recommendation: **Option A** - add state sync tasks to 07-06.
