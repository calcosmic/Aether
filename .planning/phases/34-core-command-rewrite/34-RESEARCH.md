# Phase 34: Core Command Rewrite - Research

**Researched:** 2026-02-06
**Domain:** Command file simplification with state-boundary-safe patterns
**Confidence:** HIGH

## Summary

Phase 34 rewrites `build.md` (1,080 lines) and `continue.md` (534 lines) to implement state updates at start-of-next-command rather than end-of-current-command. This architectural change solves the "orphaned EXECUTING status" problem identified in the M4L-AnalogWave postmortem where state written at command end was lost at context boundaries.

The core pattern change: `build` writes minimal EXECUTING state before spawning workers, `continue` detects completed output files and reconciles state. This follows the SIMP-02 requirement that state updates happen at the start of the next command, not the end of the current one.

Target line counts (aspirational given preserved functionality):
- `build.md`: 1,080 -> ~300 lines (72% reduction)
- `continue.md`: 534 -> ~120 lines (78% reduction)

**Primary recommendation:** Preserve visual identity (banners, colors, pheromone bars) while removing verbose display templates, redundant state validation, Bayesian spawn tracking, and step-by-step progress templates. Detection mechanism: use existence of `SUMMARY.md` files in phase directories as completion signal.

## Standard Stack

No external libraries needed. This is pure command file refactoring.

### Core
| Component | Purpose | Why Standard |
|-----------|---------|--------------|
| COLONY_STATE.json | State storage | Phase 33 established single-file pattern |
| Claude Code commands | Command definitions | Existing architecture |
| Task tool | Worker spawning | Existing spawning pattern |
| Bash tool | ANSI output | Color and banner display |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| aether-utils.sh | N/A | Utility functions | Pheromone decay, validation |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| SUMMARY.md existence | State field marker | State field requires write; file existence is passive |
| Minimal state write | Full state write | Full write risks loss at boundary |

## Architecture Patterns

### Current build.md Structure (1,080 lines)

| Section | Lines | Purpose | Verdict |
|---------|-------|---------|---------|
| Step 1: Validate | 13 | Argument validation | KEEP (simplified) |
| Step 2: Read State | 20 | Load COLONY_STATE.json | KEEP (simplified) |
| Step 3: Compute Pheromones | 50 | Decay calc, sensitivity matrix | DEFER (Phase 36) |
| Step 4: Update State | 15 | Set EXECUTING, phase status | REWRITE (minimal) |
| Step 4.5: Git Checkpoint | 20 | Pre-build commit | KEEP |
| Step 5a: Phase Lead Planning | 120 | Task assignment plan | KEEP (core logic) |
| Step 5b: Plan Checkpoint | 30 | Plan approval flow | SIMPLIFY (auto-approve more) |
| Step 5b-post: Record Decisions | 25 | Memory.decisions write | KEEP |
| Step 5c: Execute Plan | 330 | Worker spawning loop | KEEP (core logic) |
| Step 5.5: Watcher Verification | 50 | Mandatory watcher | KEEP |
| Step 6: Record Outcome | 100 | State write, error logging | MOVE to continue |
| Step 7a: Extract Learnings | 40 | Phase learnings | MOVE to continue |
| Step 7b: Emit Pheromones | 30 | Auto-emit FEEDBACK | MOVE to continue |
| Step 7c-e: Display Results | 200 | Step progress, delegation tree, pheromone recommendations | SIMPLIFY |
| Step 7f: Persistence Confirm | 20 | State validation | REMOVE (redundant) |

**Total removable/movable:** ~450 lines (Steps 6, 7a-7f minus essential display)

### Current continue.md Structure (534 lines)

| Section | Lines | Purpose | Verdict |
|---------|-------|---------|---------|
| Step 0: Parse Arguments | 10 | --all flag | KEEP |
| Step 1: Read State | 20 | Load COLONY_STATE.json | KEEP |
| Step 1.5: Auto-Continue Loop | 80 | --all mode loop | KEEP (--all decision is Claude's discretion) |
| Step 2: Determine Next Phase | 10 | Find next phase | KEEP |
| Step 2.5: Tech Debt Report | 70 | Project completion | KEEP (runs once) |
| Step 2.5b: Promote Learnings | 60 | Global learning promotion | KEEP |
| Step 2.5c: Completion Message | 15 | Final display | KEEP |
| Step 3: Phase Completion Summary | 50 | Retrospective display | SIMPLIFY |
| Step 4: Extract Phase Learnings | 60 | Duplicate detection, learnings | REWRITE (detect from output) |
| Step 4.5: Auto-Emit Pheromones | 60 | FEEDBACK/REDIRECT emission | KEEP |
| Step 5: Clean Pheromones | 15 | Cleanup utility call | KEEP |
| Step 6: Write Events | 15 | Event logging | SIMPLIFY |
| Step 7: Update Colony State | 10 | Advance current_phase | KEEP |
| Step 8: Display Result | 50 | Result output | KEEP |
| Step 9: Persistence Confirm | 15 | State validation | REMOVE (redundant) |

**Total removable:** ~80 lines (Steps 3 simplification, 4 rewrite, 9 removal)

### New build.md Pattern (Target ~300 lines)

```markdown
### Step 1: Validate + Read State
- Validate arguments
- Read COLONY_STATE.json
- Extract: goal, current_phase, plan.phases, signals, mode

### Step 2: Update State (Minimal)
- Set state="EXECUTING"
- Set current_phase=N
- Set workers.builder="active"
- Write COLONY_STATE.json (ONLY these fields)
- Do NOT update task status, learnings, or pheromones

### Step 3: Git Checkpoint
- Create pre-build commit (existing logic)

### Step 4: Spawn Phase Lead + Execute Workers
- Phase Lead planning (existing logic, simplified)
- Worker execution loop (existing logic)
- Worker retry/debugger logic (existing)

### Step 5: Watcher Verification
- Spawn watcher (existing logic)
- Display results

### Step 6: Final Output
- Display banner, worker results, delegation tree
- Display "Phase build complete. Run /ant:continue to advance."
- Do NOT write final state - continue handles reconciliation
```

### New continue.md Pattern (Target ~120 lines)

```markdown
### Step 1: Read State + Detect Completion
- Read COLONY_STATE.json
- Check: does SUMMARY.md exist for current phase?
- Check: what output files exist?
- Reconcile: update task statuses based on file existence

### Step 2: Update State (Full)
- Mark tasks completed/failed based on detection
- Extract learnings from build output
- Log errors if any detected
- Update spawn_outcomes
- Emit auto-pheromones (FEEDBACK, REDIRECT if needed)
- Advance current_phase
- Set state="READY"
- Write COLONY_STATE.json

### Step 3: Display Result
- Phase completion summary
- Next phase preview
- Commands available
```

### Detection Mechanism for Completed Work

**SIMP-07 Pattern:** "Output-as-state" - file existence indicates completion

| Signal | Detection Method |
|--------|------------------|
| Phase complete | `.planning/phase-N/SUMMARY.md` exists |
| Task complete | File path mentioned in task exists |
| Build ran | `state=="EXECUTING"` in COLONY_STATE.json |
| Build failed | EXECUTING but no new outputs since timestamp |

**Orphan State Handling:**
When `continue` detects `state=="EXECUTING"` but no completion signals:
1. Check activity.log for last worker activity timestamp
2. If stale (>30 min): offer rollback to git checkpoint
3. If recent: assume build is still running (warn user)

### Anti-Patterns to Avoid

- **End-of-command state writes for critical data:** EXECUTING status, learnings, and error logs should survive context boundaries. Move to start-of-next-command.
- **Verbose step-by-step progress lists:** The 17-line step progress in build.md adds noise. Replace with single "Build complete" message.
- **Redundant persistence confirmation:** Step 7f runs validate-state after every build. Unnecessary with single-file state.
- **Per-caste sensitivity tables in output:** The sensitivity matrix display (Step 7e) is noise. Keep pheromone bar, remove matrix.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Completion detection | Custom polling | File existence check | Simple, reliable |
| Pheromone decay | Custom math | aether-utils.sh pheromone-batch | Already tested |
| State validation | Custom validator | aether-utils.sh validate-state | Already tested |

**Key insight:** Detection mechanism should be passive (check file existence) not active (poll state repeatedly).

## Common Pitfalls

### Pitfall 1: Losing Build State at Context Boundary
**What goes wrong:** Build writes EXECUTING but context clears before completion state written
**Why it happens:** State update at end-of-command pattern
**How to avoid:** Build writes ONLY minimal state before workers; continue reconciles
**Warning signs:** Orphaned EXECUTING status that never clears

### Pitfall 2: Duplicate Learnings
**What goes wrong:** Build extracts learnings, continue extracts again
**Why it happens:** Both commands have learning extraction logic
**How to avoid:** Build does NOT extract learnings; continue handles all post-build state
**Warning signs:** Duplicate entries in memory.phase_learnings

### Pitfall 3: Task Status Mismatch
**What goes wrong:** Task marked complete in state but output file missing
**Why it happens:** State written before output actually created
**How to avoid:** continue detects completion from output existence, not from pre-written state
**Warning signs:** Task shows complete but verification fails

### Pitfall 4: Removing Too Much Visual Identity
**What goes wrong:** Colony loses its personality, output feels generic
**Why it happens:** Over-aggressive simplification targeting line counts
**How to avoid:** Per CONTEXT.md: keep banners, colors, pheromone bars, spawn output
**Warning signs:** User feedback that colony feels "dead" or "generic"

### Pitfall 5: Breaking --all Mode
**What goes wrong:** Auto-continue stops working after changes
**Why it happens:** --all mode spawns build via Task tool; changes break that interface
**How to avoid:** Test --all mode explicitly after changes
**Warning signs:** --all mode fails to progress between phases

## Code Examples

### Minimal State Write (build Step 2)
```markdown
### Step 2: Update State (Minimal)

Use Write tool to update `.aether/data/COLONY_STATE.json`:
- Set `state` to `"EXECUTING"`
- Set `current_phase` to the phase number
- Set `workers.builder` to `"active"`
- Add timestamp to track when build started: `build_started_at: "<ISO-8601>"`

Write the file. Do NOT update:
- Task statuses (continue handles this)
- Learnings (continue handles this)
- Pheromones (continue handles this)
- Spawn outcomes (continue handles this)
```

### Detection Pattern (continue Step 1)
```markdown
### Step 1: Read State + Detect Completion

Use the Read tool to read `.aether/data/COLONY_STATE.json`.

**Detect build completion:**
1. Check `state` field. If not "EXECUTING", no build ran - proceed to normal continue.
2. If "EXECUTING", build ran. Detect what completed:
   - Check if `.planning/phases/{phase}/SUMMARY.md` exists
   - Check activity.log for worker completions
   - Check git log for commits since `build_started_at`

**Reconcile state:**
For each task in the phase:
- If output file exists: mark task "completed"
- If output file missing but worker ran: mark task "failed"
- If worker didn't run: mark task "pending" (incomplete build)

Update task statuses in plan.phases[N].tasks.
```

### Simplified Build Output (replace Step 7e)
```markdown
### Step 6: Display Results

Display banner using Bash tool:
```
bash -c 'printf "\n\e[1;33m+=====================================================+\e[0m\n"'
bash -c 'printf "\e[1;33m|  BUILD COMPLETE                                     |\e[0m\n"'
bash -c 'printf "\e[1;33m+=====================================================+\e[0m\n\n"'
```

Display:
```
Phase {id}: {name}

Git Checkpoint: {commit_hash}

Workers:
  {Per-worker: "[CASTE] task ... COMPLETE/ERROR"}

Next:
  /ant:continue            Advance to next phase
  /ant:feedback "<note>"   Give feedback first
```

Do NOT display:
- Step progress list (17 lines)
- Caste sensitivity table
- Persistence confirmation
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| State update at end-of-command | State update at start-of-next-command | v5.1 (Phase 34) | State survives context boundaries |
| Explicit task completion | Output-as-state detection | v5.1 (Phase 34) | No state loss on boundary |
| 17-step verbose output | Condensed result display | v5.1 (Phase 34) | Cleaner output |
| Persistence confirmation step | Removed (single file = reliable) | v5.1 (Phase 34) | Simpler commands |

**Deprecated/outdated:**
- Step 7f: Persistence Confirmation - redundant with single-file state
- Step-by-step progress checklist - replaced with condensed output
- Per-caste sensitivity table in output - noise, removed

## Open Questions

1. **Exact line targets vs preserved functionality:**
   - CONTEXT.md says "targets are goals not mandates"
   - With preserved banners/colors/spawn output, 300 lines for build may be optimistic
   - Recommendation: Aim for ~400 lines build, ~150 lines continue as realistic targets

2. **--all mode preservation:**
   - CONTEXT.md marks as "Claude's discretion"
   - Recommendation: KEEP --all mode. It's useful for unattended runs.
   - Implementation: No changes needed to --all logic itself

3. **Halt condition thresholds:**
   - CONTEXT.md marks as "Claude's discretion"
   - Current: watcher score < 4 OR 2 consecutive failures
   - Recommendation: Keep current thresholds. They're reasonable.

4. **Detection mechanism edge cases:**
   - What if SUMMARY.md exists but is empty?
   - What if build crashed mid-file-write?
   - Recommendation: Check file non-empty AND contains expected markers

## Sources

### Primary (HIGH confidence)
- Current command files: `.claude/commands/ant/build.md` (1,080 lines), `continue.md` (534 lines)
- COLONY_STATE.json schema: `.aether/data/COLONY_STATE.json`
- Phase 33 research: `.planning/phases/33-state-foundation/33-RESEARCH.md`
- CONTEXT.md decisions: `.planning/phases/34-core-command-rewrite/34-CONTEXT.md`

### Secondary (MEDIUM confidence)
- Requirements: SIMP-02, SIMP-05, SIMP-07 from ROADMAP/REQUIREMENTS
- v5 Field Notes: context boundary issues documented

### Tertiary (LOW confidence)
- None - all findings from direct codebase inspection

## Metadata

**Confidence breakdown:**
- Current structure analysis: HIGH - direct file inspection
- Removal recommendations: HIGH - mapped against CONTEXT.md decisions
- Detection mechanism: MEDIUM - logical but untested
- Line count targets: MEDIUM - realistic given preserved functionality

**Research date:** 2026-02-06
**Valid until:** No expiration - internal refactoring
