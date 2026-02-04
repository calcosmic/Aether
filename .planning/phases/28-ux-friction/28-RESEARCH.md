# Phase 28: UX & Friction Reduction - Research

**Researched:** 2026-02-04
**Domain:** Claude Code prompt engineering, UX flow changes to existing .claude/commands/ant/ prompt files
**Confidence:** HIGH

## Summary

Phase 28 requires modifications to three existing Claude Code command prompts (build.md, continue.md, colonize.md) and no new libraries, infrastructure, or utility scripts. All three requirements (UX-01, UX-02, FLOW-01) are achievable through prompt text additions and control flow changes within existing command files.

The codebase already persists all state to JSON files (COLONY_STATE.json, PROJECT_PLAN.json, errors.json, memory.json, events.json, pheromones.json) before commands complete -- the "safe to /clear" message is a display-only addition at the end of existing output templates. Auto-continue requires adding a `--all` argument parser and a loop wrapper around the existing continue.md logic. Pheromone-first flow requires adding contextual suggestions to colonize.md's Step 6 display.

**Primary recommendation:** Implement as 2 plans: Plan 1 covers UX-01 (safe-to-clear in build.md, continue.md, colonize.md) + FLOW-01 (colonize.md pheromone suggestions), Plan 2 covers UX-02 (auto-continue --all loop in continue.md).

## Standard Stack

### Core

No new libraries. All changes are to existing markdown prompt files.

| File | Current Location | Purpose | Change Type |
|------|-----------------|---------|-------------|
| build.md | `.claude/commands/ant/build.md` | Phase execution | Add "safe to /clear" to Step 7e display |
| continue.md | `.claude/commands/ant/continue.md` | Phase advancement | Add --all loop + "safe to /clear" to Step 8 |
| colonize.md | `.claude/commands/ant/colonize.md` | Codebase analysis | Add pheromone suggestions to Step 6 + "safe to /clear" |

### Supporting

| Tool | Purpose | Already Exists |
|------|---------|---------------|
| `aether-utils.sh validate-state all` | Verify state persistence before showing "safe to /clear" | Yes -- runs during /ant:init Step 6.5 |
| `aether-utils.sh pheromone-batch` | Check active pheromones for colonize suggestions | Yes -- used in build.md Step 3 |

### Alternatives Considered

None. This is prompt text changes only. No architectural alternatives exist.

## Architecture Patterns

### Current Command Output Structure

All three target commands end with a display step that shows results + "Next Steps" suggestions. The pattern is consistent:

**build.md Step 7e** ends with:
```
Next:
  /ant:continue            Advance to next phase
  /ant:feedback "<note>"   Give feedback first
  /ant:status              View full colony status
```

**continue.md Step 8** ends with:
```
Next Steps:
  /ant:build <next>      Start building Phase <next>
  /ant:phase <next>      Review phase details first
  /ant:focus "<area>"    Guide colony attention before building
  /ant:redirect "<pat>"  Set constraints before building
```

**colonize.md Step 6** ends with:
```
Next:
  /ant:plan              Generate project plan
  /ant:focus "<area>"    Focus on specific area
  /ant:redirect "<pat>"  Warn against patterns found
```

### Pattern 1: Safe-to-Clear Message (UX-01)

**What:** Add a persistence confirmation message AFTER the "Next Steps" block in each command's final display step.

**When to use:** After any command that completes meaningful work and has already written all state files.

**Format (consistent across all commands):**
```
---
All state persisted. Safe to /clear context if needed.
  State: .aether/data/COLONY_STATE.json
  Plan:  .aether/data/PROJECT_PLAN.json
  Run /ant:resume-colony in a new session to restore.
```

**Verification approach:** Before displaying the message, run `bash .aether/aether-utils.sh validate-state all` and only show the "safe to /clear" message if `pass: true`. If validation fails, show a warning instead:
```
WARNING: State validation failed. Do NOT /clear until resolved.
  Run /ant:status to check state integrity.
```

**Key insight:** State is ALREADY persisted before the display step in all three commands. build.md writes state in Step 6, continue.md in Step 7, colonize.md in Step 5+7. The message is purely informational -- it does not trigger any new persistence logic.

### Pattern 2: Auto-Continue Loop (UX-02)

**What:** When `/ant:continue --all` is invoked, execute all remaining phases sequentially without requiring user approval at each phase boundary.

**Current continue.md flow:**
1. Read state -> 2. Determine next phase -> 3. Phase summary -> 4. Extract learnings -> 4.5. Auto-emit pheromones -> 5. Clean pheromones -> 6. Write events -> 7. Update colony state -> 8. Display result

**Modified flow for --all:**
1. Parse `$ARGUMENTS` for `--all` flag
2. If `--all` is present, wrap Steps 2-8 in a loop that:
   a. Advances to next phase
   b. Automatically triggers `/ant:build <next_phase>` for each phase
   c. After build completes, runs the continue logic (Steps 3-7) to advance
   d. Repeats until no more phases remain
   e. Displays cumulative results at the end

**Critical design decision:** Auto-continue must still include the build step. Simply "continuing" without building is meaningless. The flow should be:

```
For each remaining phase:
  1. Display: "Auto-continuing: Phase <N> - <name>"
  2. Execute build logic (the full Step 5a-5c-5.5-6-7 from build.md)
  3. Execute continue logic (Steps 3-7 from continue.md)
  4. Display condensed phase summary
  5. Check for critical failures -- if watcher score < 4/10, halt auto-continue
```

**Implementation options (two approaches):**

**Option A -- continue.md calls build inline:** continue.md's --all mode reads the build.md spec and inlines its logic. This is complex and creates massive prompt duplication.

**Option B -- continue.md instructions say "run /ant:build then /ant:continue for each phase":** The command prompt tells Claude to execute the build and continue sequence for each remaining phase. Since Claude Code executes commands as prompt instructions (not function calls), the prompt can simply say "execute the build flow for phase N" and reference build.md's logic.

**Recommended: Option B with Task tool delegation.** For each remaining phase, spawn a Task that executes the build, then the continue logic runs normally. This keeps continue.md manageable and reuses build.md's existing logic via the Task tool.

**However**, there is a simpler option: The `--all` flag on continue.md makes it advance through phases, and at each step it spawns the build as a sub-task (using the Task tool). After the build sub-task returns, continue.md performs its normal continue logic (learnings, state update, advance). The display shows condensed output per phase.

**Halt conditions for auto-continue:**
- Watcher quality_score < 4 (critical failure)
- More than 2 consecutive phase failures
- All phases complete (normal termination)

### Pattern 3: Pheromone-First Flow (FLOW-01)

**What:** After colonize completes, the output suggests specific pheromone injections based on the colonization findings before proceeding to `/ant:plan`.

**Current colonize.md Step 6 output:**
```
Next:
  /ant:plan              Generate project plan
  /ant:focus "<area>"    Focus on specific area
  /ant:redirect "<pat>"  Warn against patterns found
```

**Modified output:** The colonizer ant's report contains specific findings about the codebase (tech stack, patterns, conventions, potential issues). The Queen should analyze these findings and suggest CONCRETE pheromone injections:

```
Suggested Pheromone Injections:
  Based on the colonization findings, consider these signals before planning:

  /ant:focus "<specific area from findings>"
    Reason: <why this area deserves attention based on findings>

  /ant:redirect "<specific pattern to avoid>"
    Reason: <why this should be avoided based on findings>

  These are suggestions. Proceed directly to /ant:plan if no guidance needed.

Next:
  /ant:plan              Generate project plan (after optional pheromone injection)
```

**Key insight:** The colonizer ant already reports its findings in Step 4. The Queen (in Step 6) already has those findings in context. The modification is to the Step 6 display instructions -- tell the Queen to analyze the ant's report and generate specific, concrete pheromone suggestions rather than generic command references.

### Anti-Patterns to Avoid

- **Do not add new state files:** The safe-to-clear message uses existing validate-state, not a new "persistence status" file.
- **Do not change state schemas:** No new fields in COLONY_STATE.json or any other state file.
- **Do not create new commands:** UX-02 (auto-continue) is a flag on the existing `/ant:continue` command, not a new `/ant:auto-build` command.
- **Do not duplicate build.md logic in continue.md:** Auto-continue should delegate to build via Task tool, not copy-paste 700 lines of build logic.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| State validation | New validation logic | `bash .aether/aether-utils.sh validate-state all` | Already exists, validates all 5 JSON files |
| Pheromone status check | Manual file reads | `bash .aether/aether-utils.sh pheromone-batch` | Already computes decay, returns current strengths |
| Build execution in auto-continue | Inline build logic | Task tool with build instructions | Keeps continue.md manageable, reuses build.md |

**Key insight:** This phase is entirely about prompt text changes. There are zero utility functions to build, zero new infrastructure pieces, zero new state schemas.

## Common Pitfalls

### Pitfall 1: Safe-to-Clear Message Shown Before State is Actually Written

**What goes wrong:** The "safe to /clear" message is added to the display template but placed before the state-writing steps complete.
**Why it happens:** Prompt instructions execute sequentially but the LLM might reorder or skip steps.
**How to avoid:** The message MUST be in the final display step (Step 7e for build, Step 8 for continue, Step 6/7 for colonize). Add explicit instruction: "Display this message ONLY after all Write tool operations in prior steps have completed."
**Warning signs:** State files missing data after user /clears based on the message.

### Pitfall 2: Auto-Continue Runs Without Building

**What goes wrong:** `/ant:continue --all` advances the phase counter without actually building each phase -- it just marks phases as "completed" without doing work.
**Why it happens:** continue.md's current logic only advances the counter. Building is a separate command.
**How to avoid:** The `--all` mode MUST include build execution for each phase. The prompt must explicitly say "for each remaining phase: first build it, then continue."
**Warning signs:** Phases marked "completed" but no code changes, no worker spawns, no watcher verification.

### Pitfall 3: Auto-Continue Never Stops on Failure

**What goes wrong:** Auto-continue runs through all phases even when phases are failing catastrophically.
**Why it happens:** No halt condition defined in the loop.
**How to avoid:** Define explicit halt conditions: watcher score below threshold, consecutive failures, critical errors.
**Warning signs:** Multiple failed phases in a row, accumulating errors with no intervention.

### Pitfall 4: Colonize Pheromone Suggestions Are Generic

**What goes wrong:** The pheromone suggestions after colonize are generic boilerplate ("consider focusing on important areas") rather than specific to the actual findings.
**Why it happens:** The prompt instruction says "suggest pheromones" but doesn't say "base suggestions on the specific findings the ant reported."
**How to avoid:** The instruction must explicitly say: "Analyze the colonizer ant's report. Identify 1-2 specific focus areas and 0-1 specific redirect patterns based on the ACTUAL findings. Do NOT give generic suggestions."
**Warning signs:** Every colonize run suggests the same pheromones regardless of codebase.

### Pitfall 5: Prompt Size Explosion in continue.md

**What goes wrong:** Adding auto-continue loop logic to continue.md makes the prompt too long (currently ~320 lines, could grow to 500+).
**Why it happens:** Inlining build logic instead of delegating.
**How to avoid:** Use Task tool delegation for the build step. Keep continue.md's --all additions to ~50 lines of loop logic + Task spawn.
**Warning signs:** continue.md exceeds 400 lines.

### Pitfall 6: Validate-State Fails and Blocks "Safe to Clear"

**What goes wrong:** validate-state returns false due to a benign issue (e.g., empty errors array counted as "no errors" rather than valid), blocking the safe-to-clear message.
**Why it happens:** validate-state checks schema structure, not content completeness.
**How to avoid:** If validate-state fails, show the warning but don't crash. The message is informational, not a gate. Also: test validate-state against the actual state files that exist after a build/continue/colonize to confirm it passes.
**Warning signs:** Users never see the "safe to /clear" message because validation always fails.

## Code Examples

### UX-01: Safe-to-Clear Addition to build.md Step 7e

Add after the existing "Next:" block at the end of Step 7e:

```markdown
After the "Next:" block, add a state persistence confirmation:

Run `bash .aether/aether-utils.sh validate-state all` using the Bash tool.

If the result has `pass: true`:
```
---
All state persisted. Safe to /clear context if needed.
  State: .aether/data/ (6 files validated)
  Resume: /ant:resume-colony
```

If the result has `pass: false`:
```
---
WARNING: State validation issue detected. Check /ant:status before clearing.
```
```

### UX-01: Safe-to-Clear Addition to continue.md Step 8

Add after the existing "Next Steps:" block:

```markdown
---
All state persisted. Safe to /clear context if needed.
  State: .aether/data/ (6 files validated)
  Resume: /ant:resume-colony
```

### FLOW-01: Pheromone Suggestions in colonize.md Step 6

Replace the current Step 6 display with an enhanced version:

```markdown
### Step 6: Display Results

Display the ant's findings, then analyze the findings to suggest specific pheromones:

```
👑 CODEBASE COLONIZED

  Goal: "{goal}"

{ant's report}

  Findings saved to memory.json

Suggested Pheromone Injections:
  Based on colonization findings:
  {analyze the ant's report and suggest 1-2 specific pheromones}

  /ant:focus "<specific area from ant's findings>"
    Why: <concrete reason from analysis>

  {if the ant identified problematic patterns or risks:}
  /ant:redirect "<specific pattern to avoid>"
    Why: <concrete reason from analysis>

  Skip these if you want the colony to plan without constraints.

Next:
  /ant:plan              Generate project plan
  /ant:focus "<area>"    Inject focus before planning
  /ant:redirect "<pat>"  Inject constraint before planning
```

The pheromone suggestions MUST be derived from the actual colonizer report,
not generic boilerplate. Reference specific findings from the ant's analysis.
```

### UX-02: Auto-Continue --all in continue.md

Add argument parsing at the beginning of continue.md:

```markdown
### Step 0: Parse Arguments

Check if `$ARGUMENTS` contains `--all` or `all`.

If `--all` is present, set `auto_mode = true`. Proceed to Step 1.
If `--all` is not present, set `auto_mode = false`. Proceed to Step 1 (normal flow).
```

Then add the loop wrapper after Step 1:

```markdown
### Step 1.5: Auto-Continue Loop (only if auto_mode is true)

If `auto_mode` is false, skip to Step 2 (normal flow).

If `auto_mode` is true, execute the following loop:

For each remaining phase (from current_phase + 1 to the last phase):

  1. Display: "--- Auto-Continue: Phase <N>/<total> ---"

  2. Build the phase: Use the **Task tool** with subagent_type="general-purpose":
     ```
     You are executing /ant:build <phase_number> as part of an auto-continue run.

     Read and follow the instructions in .claude/commands/ant/build.md exactly,
     with $ARGUMENTS = "<phase_number>".

     Execute all steps from Step 1 through Step 7e.
     Return the watcher quality_score and recommendation.
     ```

  3. After build returns, check the watcher result:
     - If quality_score < 4: HALT auto-continue. Display:
       "Auto-continue halted: Phase <N> scored <score>/10. Manual review needed."
       Then proceed to Step 8 display for the current state.
     - If quality_score >= 4: Continue with normal continue logic (Steps 3-7)

  4. Run Steps 3-7 of continue.md for this phase (learnings, pheromones, state update)

  5. Display condensed summary:
     "Phase <N>: <name> -- <COMPLETE|FAILED> (quality: <score>/10)"

After the loop completes (or halts), proceed to Step 8 with cumulative display.
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| No persistence message | No persistence message | - | Users lose work by /clearing at wrong time |
| Manual build+continue cycle | Manual build+continue cycle | - | Tedious for multi-phase projects |
| Generic "Next:" suggestions after colonize | Generic "Next:" suggestions | - | Users skip pheromone injection |

**After this phase:**
| Old Approach | New Approach | Impact |
|--------------|--------------|--------|
| No persistence message | "Safe to /clear" with validation | Users know when state is saved |
| Manual build+continue cycle | `--all` flag runs full pipeline | Multi-phase builds unattended |
| Generic colonize suggestions | Specific pheromone suggestions | Users more likely to inject guidance |

## Open Questions

### 1. Auto-Continue Build Delegation Mechanism

**What we know:** The Task tool can spawn subagents that read and follow prompt files. build.md is a prompt file at `.claude/commands/ant/build.md`.

**What's unclear:** Whether a Task-spawned agent can effectively execute the full build.md flow (which itself spawns Task agents for Phase Lead + workers). This is nested Task spawning at depth 2+. Claude Code's Task tool may have depth limitations.

**Recommendation:** If Task delegation fails due to depth limits, fall back to having continue.md inline a simplified build flow for auto-mode. This is less elegant but guaranteed to work. The plan should include a verification step that tests Task nesting before committing to the delegation approach.

### 2. Auto-Continue Phase Approval Checkpoint

**What we know:** The current build.md Step 5b asks the user "Proceed with this plan?" before executing. In auto-continue mode, this would block.

**What's unclear:** Should auto-continue skip the plan approval step entirely, or auto-approve?

**Recommendation:** Auto-continue should auto-approve (skip Step 5b user prompt). The Phase Lead still plans, but approval is automatic. This aligns with the success criteria: "without requiring user approval at each phase boundary." Add a note in the build delegation prompt: "In auto-continue mode, auto-approve the Phase Lead's plan without asking the user."

### 3. Safe-to-Clear vs Pause-Colony Overlap

**What we know:** `/ant:pause-colony` already creates a HANDOFF.md and suggests `/ant:resume-colony`. The new "safe to /clear" message suggests `/ant:resume-colony` too.

**What's unclear:** Whether the safe-to-clear message should also trigger pause-colony's HANDOFF.md creation.

**Recommendation:** No. Keep them separate. "Safe to /clear" is a lightweight informational message. `/ant:pause-colony` is a deliberate session handoff with rich context. The safe-to-clear message just confirms state files are written. Users who want a full handoff should still use `/ant:pause-colony`.

## Sources

### Primary (HIGH confidence)
- Direct file reads of all 13 command files in `.claude/commands/ant/`
- Direct file read of `.aether/aether-utils.sh` (full 302-line utility script)
- Direct file reads of all 6 JSON state files in `.aether/data/`
- Direct file reads of `.planning/PROJECT.md`, `.planning/ROADMAP.md`

### Secondary (MEDIUM confidence)
- None needed. All information sourced from codebase inspection.

### Tertiary (LOW confidence)
- Task tool nesting depth limits -- not verified against Claude Code platform documentation. Based on existing build.md behavior (which nests Phase Lead -> Worker spawns at depth 1-2).

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH -- all files inspected directly, changes are prompt-text-only
- Architecture: HIGH -- current command structure fully mapped, modification points identified
- Pitfalls: HIGH -- based on direct analysis of existing command flows and their interactions
- Auto-continue mechanism: MEDIUM -- Task nesting depth is uncertain, may need fallback

**Research date:** 2026-02-04
**Valid until:** No expiry -- this is prompt engineering on a stable internal system, not external library research

## Appendix: Current File Sizes and Modification Scope

| File | Current Lines | Estimated Change | Change Description |
|------|--------------|------------------|--------------------|
| build.md | 730 | +20 lines | Add validate-state call + safe-to-clear message to Step 7e |
| continue.md | 319 | +80 lines | Add Step 0 (arg parse) + Step 1.5 (auto loop) + safe-to-clear to Step 8 |
| colonize.md | 170 | +30 lines | Enhance Step 6 with pheromone suggestions + safe-to-clear |

**Total estimated change:** ~130 lines of prompt text additions across 3 files.
