# Phase 30: Automation & New Capabilities - Research

**Researched:** 2026-02-05
**Domain:** Auto-spawned review/debug agents, pheromone recommendation surfacing, ANSI visual output, tech debt aggregation
**Confidence:** HIGH

## Summary

Phase 30 adds six capabilities to the Aether colony: auto-spawned reviewer after each builder wave (AUTO-01), auto-spawned debugger on test failure (AUTO-02), pheromone recommendations surfaced to users (AUTO-03), ANSI-colored build progress indicators (AUTO-04), colonizer visual output (AUTO-05), and tech debt report at project completion (INT-06). All six are prompt-level changes to existing markdown command files and worker specs, plus one potential new utility subcommand. No new libraries, no new scripts beyond potential aether-utils.sh additions.

The core challenge is that every feature integrates into the existing build.md execution loop (Steps 5c, 5.5, 7) or colonize.md display (Step 6). The reviewer auto-spawn happens after each wave in Step 5c (not just once at the end). The debugger auto-spawn is a conditional branch within the existing worker failure retry logic in Step 5c.f. The pheromone recommendations are a new display section in Step 7e. The ANSI visual output is achieved via bash echo/printf commands within the Queen's display sections. The tech debt report is a new step triggered when all phases are complete, in continue.md Step 2.

Research confirms: (1) no new worker spec files are needed -- reviewer uses the existing watcher-ant.md spec in advisory mode, debugger uses builder-ant.md spec with diagnostic constraints; (2) ANSI escape codes work in Claude Code bash output -- use printf/echo with \e[XXm codes via Bash tool calls; (3) the Queen (build.md) cannot produce colored text directly in its markdown output, so colored sections must be rendered via bash echo commands; (4) tech debt aggregation pulls from errors.json flagged_patterns and activity.log, both already available.

**Primary recommendation:** Implement as 3 plans: Plan 1 covers AUTO-01 (reviewer) + AUTO-02 (debugger) since both modify build.md Step 5c execution loop. Plan 2 covers AUTO-03 (pheromone recommendations) + INT-06 (tech debt report) since both add post-build intelligence sections. Plan 3 covers AUTO-04 (ANSI visual output) + AUTO-05 (colonizer visual output) since both are visual formatting changes.

## Standard Stack

### Core

No new libraries. All changes are to existing markdown prompt files.

| File | Current Location | Purpose | Change Type |
|------|-----------------|---------|-------------|
| build.md | `.claude/commands/ant/build.md` | Build orchestration | Add reviewer after each wave, debugger on retry failure, pheromone recommendations, ANSI color output |
| continue.md | `.claude/commands/ant/continue.md` | Phase advancement | Add tech debt report at project completion |
| colonize.md | `.claude/commands/ant/colonize.md` | Codebase analysis | Add visual output with emojis and progress markers |
| watcher-ant.md | `.aether/workers/watcher-ant.md` | Quality validation | Used as reviewer spec (advisory mode) |
| builder-ant.md | `.aether/workers/builder-ant.md` | Code implementation | Used as debugger spec (diagnostic mode) |
| aether-utils.sh | `.aether/aether-utils.sh` | Utility functions | Potentially add color-echo or progress-bar subcommand |

### Supporting

| Tool | Purpose | Already Exists |
|------|---------|---------------|
| `aether-utils.sh error-summary` | Aggregate errors for tech debt report | Yes |
| `aether-utils.sh error-pattern-check` | Detect recurring error patterns | Yes |
| `aether-utils.sh activity-log-read` | Read activity log for tech debt aggregation | Yes |
| Task tool (subagent_type="general-purpose") | Spawn reviewer/debugger agents | Yes |
| Bash tool printf/echo | ANSI colored output | Yes -- verified working in Claude Code |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Reusing watcher-ant.md for reviewer | New reviewer-ant.md spec file | Adding a new worker spec (7th caste) is architecturally clean but conflicts with the project constraint "No New Commands" and the six-caste design. Reusing watcher-ant.md with an advisory mode override in the spawn prompt is simpler and maintains the existing architecture. |
| Reusing builder-ant.md for debugger | New debugger-ant.md spec file | Same reasoning. The debugger is a builder with diagnostic constraints, not a new caste. The spawn prompt constrains it to "patch, don't rewrite." |
| aether-utils.sh progress-bar subcommand | Inline printf in prompts | A utility subcommand is cleaner and reusable, but adds shell complexity. Given the ROADMAP says "ANSI progress bars are solved" and the research/STACK.md already has the pattern, adding a simple subcommand is justified. |
| Tech debt report as separate command | Integrated into continue.md completion | A separate `/ant:tech-debt` command would be more discoverable but violates "No New Commands" constraint. Triggering at project completion in continue.md is the right place. |

## Architecture Patterns

### Integration Points in build.md

The current build.md flow is:

```
Step 1: Validate
Step 2: Read State
Step 3: Compute Active Pheromones
Step 4: Update State
Step 4.5: Git Checkpoint
Step 5a: Spawn Phase Lead as Planner
Step 5b: Plan Checkpoint (auto-approve logic)
Step 5b-post: Record Plan Decisions
Step 5c: Execute Plan (wave loop with worker spawns)
  - Step 5c.4: Per-wave loop
    - Steps a-g: Per-worker spawn/retry logic
    - Step h: Post-wave conflict check
Step 5.5: Watcher Verification (mandatory)
Step 6: Record Outcome
Step 7a-e: Extract Learnings, Emit Pheromones, Display Results
Step 7f: Persistence Confirmation
```

**AUTO-01 (Reviewer) inserts after Step 5c.h (post-wave conflict check):**
After each wave completes and conflict check passes, spawn a reviewer (watcher in advisory mode). This is a new Step 5c.i within the per-wave loop.

**AUTO-02 (Debugger) modifies Step 5c.f-g (worker retry logic):**
Currently: worker fails -> retry count < 2 -> spawn NEW worker (same caste, same task). After retry also fails -> "Task failed after 2 retries." The debugger intercepts between the worker's first retry failure and the "give up" state. Flow becomes:
- Worker fails (first attempt) -> worker gets one retry (existing Step 5c.f)
- Worker retry fails (second attempt) -> spawn debugger-ant (NEW)
- Debugger attempts to diagnose AND fix -> if fix works, continue; if fix fails, THEN "Task failed"

**AUTO-03 (Pheromone recommendations) adds to Step 7e (Display Results):**
After the existing display, add a "Pheromone Recommendations" section with max 3 suggestions.

**AUTO-04 (ANSI visual output) modifies Step 5c display and Step 7e:**
Replace plain text worker status lines with bash echo calls using ANSI color codes per caste.

**INT-06 (Tech debt report) adds to continue.md Step 2:**
When continue.md detects "All phases complete" (currently just displays a message and stops), generate a tech debt report before stopping.

### Pattern 1: Advisory Reviewer (AUTO-01)

**What:** After each wave completes in Step 5c, spawn a watcher-ant in advisory mode. Findings are displayed inline but do NOT block progress. Only CRITICAL severity findings trigger a rebuild.

**When to use:** After Step 5c.h (post-wave conflict check) completes for each wave.

**Design:**

The reviewer is NOT a new caste. It uses the existing watcher-ant.md spec with a constrained prompt:

```
--- WORKER SPEC ---
{full contents of .aether/workers/watcher-ant.md}

--- ACTIVE PHEROMONES ---
{pheromone block from Step 3}

--- TASK ---
You are being spawned as a post-wave ADVISORY REVIEWER.

Phase {id}: {phase_name}
Wave {N} of {total_waves} just completed.

Workers in this wave:
{for each worker: caste, task, result summary}

Your mission:
1. Read the files modified by workers in this wave
2. Run Execution Verification (syntax, import, launch checks)
3. Run Quality mode checks at minimum
4. Produce findings with severity levels (CRITICAL, HIGH, MEDIUM, LOW)

IMPORTANT: You are in ADVISORY mode.
- Your findings will be DISPLAYED to the user but will NOT block progress
- Only CRITICAL severity findings will trigger a rebuild
- Focus on catching issues EARLY before they cascade to later waves
- Be concise -- this runs after every wave, not just at the end

Produce a structured report with:
- findings: array of {severity, description, location, recommendation}
- critical_count: number of CRITICAL findings
- summary: one-line summary of wave quality
```

**Rebuild logic (Queen handles this, not the reviewer):**

```
After reviewer returns:
  Parse critical_count from reviewer report.

  If critical_count > 0 AND rebuild_iterations < 2:
    Display: "CRITICAL issue found. Triggering rebuild of wave {N}."
    Increment rebuild_iterations for this wave.
    Re-run the wave's workers with reviewer findings as context.
    (Append to worker prompt: "Previous attempt had CRITICAL issues: {findings}. Fix these.")

  If critical_count > 0 AND rebuild_iterations >= 2:
    Display: "CRITICAL issues persist after 2 rebuilds. Continuing."
    Display findings for user review.

  If critical_count == 0:
    Display reviewer summary inline.
    Continue to next wave.
```

**CRITICAL vs WARNING severity boundary (Claude's discretion per CONTEXT.md):**

Recommended boundary definition for the reviewer prompt:
- CRITICAL: Code does not run (syntax errors, import failures, launch crashes), security vulnerabilities (exposed secrets, SQL injection), data corruption risk
- HIGH: Tests fail, missing major requirements, breaking existing functionality
- MEDIUM: Code quality issues, missing edge cases, convention violations
- LOW: Style issues, minor improvements, documentation gaps

Only CRITICAL triggers rebuild. HIGH/MEDIUM/LOW are displayed but do not block.

### Pattern 2: Auto-Spawned Debugger (AUTO-02)

**What:** When a worker's own retry fails (second attempt), spawn a debugger-ant that attempts to diagnose AND fix the failure. The debugger patches existing code rather than rewriting.

**When to use:** Replaces the current "give up after 2 retries" in Step 5c.g.

**Design:**

The debugger uses builder-ant.md spec with diagnostic constraints:

```
--- WORKER SPEC ---
{full contents of .aether/workers/builder-ant.md}

--- ACTIVE PHEROMONES ---
{pheromone block from Step 3}

--- TASK ---
You are being spawned as a DEBUGGER ANT.

A worker failed its task twice. Your job: diagnose the failure and fix it.

Failed task: {task_description}
Failed caste: {caste}
First attempt error: {error_from_attempt_1}
Second attempt error: {error_from_attempt_2}
Files involved: {files_from_worker_report}

Your mission:
1. Read the files involved in the failure
2. DIAGNOSE the root cause -- understand WHY it failed, not just WHAT failed
3. Identify the MINIMAL patch to fix the issue
4. Apply the fix using Write/Edit tools
5. Run verification (syntax check, import check, tests if available)
6. Report what you found and what you changed

CONSTRAINTS:
- PATCH the existing code. Do NOT rewrite from scratch.
- Preserve the original worker's approach and intent.
- If the failure is in a test, fix the code to pass the test (not the other way around).
- If you cannot diagnose the issue, report "UNDIAGNOSABLE" with your analysis.

Produce a structured report with:
- diagnosis: root cause description
- fix_applied: boolean
- files_modified: array of paths
- verification: pass/fail with details
```

**Queen-side logic for debugger spawn:**

Current Step 5c.f-g flow:
```
f. Worker fails, retry < 2 -> spawn retry
g. Worker fails, retry >= 2 -> "Task failed after 2 retries"
```

New flow:
```
f. Worker fails (attempt 1) -> spawn retry (same as current, worker gets one retry)
f2. Worker retry fails (attempt 2) -> spawn debugger-ant (NEW)
g. If debugger fix_applied == true:
     Mark task as completed.
     Display: "Debugger fixed: {diagnosis}"
   If debugger fix_applied == false or "UNDIAGNOSABLE":
     Based on task criticality (Claude's discretion per CONTEXT.md):
       - If task is critical to success criteria: halt and ask user
       - If task is non-critical: skip and continue
     Display: "Debugger could not fix: {diagnosis}. Task skipped."
```

### Pattern 3: Pheromone Recommendations (AUTO-03)

**What:** After build completes, surface natural language recommendations to the user based on build outcomes.

**When to use:** Added to Step 7e display output.

**Design per CONTEXT.md decisions:**
- Natural language descriptive guidance, NOT copy-paste commands
- Triggered by ALL build outcomes (failures AND successes)
- Can appear between waves when urgent patterns emerge
- Maximum 3 suggestions per build

**Recommendation generation logic (Queen synthesizes from available data):**

Sources for recommendations:
1. Worker results from Step 5c (which tasks succeeded/failed, error patterns)
2. Watcher report from Step 5.5 (quality score, issues found)
3. Errors.json flagged_patterns (recurring cross-phase issues)
4. Activity log patterns (which castes struggled, which files had issues)

**Trigger patterns:**

| Signal | Trigger | Example Recommendation |
|--------|---------|----------------------|
| Repeated test failures in same module | error cluster detection | "The authentication module had repeated test failures across multiple workers. Consider focusing the colony on stabilizing auth before expanding." |
| Quality score < 6 | low quality signal | "This phase scored below average. The colony might benefit from a redirect signal to avoid {specific anti-pattern found}." |
| Multiple workers touching related files | convergence signal | "Several workers modified files in the API layer. A focus signal on API consistency could help maintain coherence." |
| Clean build, high quality | positive signal | "Strong build quality on the data layer. The patterns established here could guide similar modules." |
| Flagged error patterns from errors.json | persistent issue | "Build errors have recurred {N} times. Consider a redirect to avoid {pattern}." |

**Between-wave urgent recommendations:**

If the reviewer (AUTO-01) detects CRITICAL issues in a wave, display a recommendation immediately:
```
Recommendation: The colony detected critical issues in {area}. Consider pausing after this build to investigate.
```

**Display format in Step 7e:**

```
Pheromone Recommendations:
  Based on this build's outcomes:

  1. {natural language recommendation}
     Signal: {observation that triggered it}

  2. {natural language recommendation}
     Signal: {observation}

  These are suggestions, not commands. Use /ant:focus or /ant:redirect to act on them.
```

### Pattern 4: ANSI Visual Output (AUTO-04)

**What:** Build output uses ANSI-colored progress indicators with caste-specific colors.

**Critical constraint:** Claude's text output (what the LLM types directly) is plain markdown text and does NOT render ANSI escape codes. Colored output MUST be produced via Bash tool calls (printf/echo). The Queen must use Bash tool to display colored status lines.

**Color scheme (from CONTEXT.md, extended):**

```bash
# Full caste color scheme
COLOR_QUEEN="\e[1;33m"        # Bold yellow
COLOR_COLONIZER="\e[36m"      # Cyan
COLOR_ROUTESETTER="\e[33m"    # Yellow
COLOR_BUILDER="\e[32m"        # Green
COLOR_WATCHER="\e[35m"        # Magenta
COLOR_SCOUT="\e[34m"          # Blue
COLOR_ARCHITECT="\e[37m"      # White/bright
COLOR_DEBUGGER="\e[31m"       # Red
COLOR_REVIEWER="\e[34m"       # Blue (same as scout -- reviewer is watcher variant)
COLOR_RESET="\e[0m"           # Reset
```

**Implementation approach:**

The Queen (build.md) currently displays worker status as plain text:
```
{caste_emoji} {caste}-ant: {task_description}
  Result: {COMPLETE or ERROR}
  Files: {count}
```

Change to use Bash tool for colored output:
```
bash -c 'printf "\e[32m[BUILDER]\e[0m auth-routes ... \e[32mCOMPLETE\e[0m\n"'
```

**Where to add colored output:**
1. Wave header: `--- Wave {N}/{total} ---` -> colored via bash echo
2. Worker spawn announcements: `Spawning {caste}-ant...` -> caste-colored
3. Worker results: status line with caste color
4. Progress bar: `[########............] 3/5` -> caste-colored bar fill
5. Reviewer summary: blue-colored inline findings
6. Final summary header: yellow (Queen color)

**Pattern for colored sections in build.md prompts:**

```
After worker returns, instead of outputting plain text, use:

bash -c 'printf "\e[32m%-12s\e[0m %s ... \e[32m%s\e[0m\n" "[BUILDER]" "auth-routes" "COMPLETE"'
```

This approach uses the Bash tool to render the ANSI codes, which the terminal will display with colors. The LLM text around it remains plain.

### Pattern 5: Colonizer Visual Output (AUTO-05)

**What:** The colonize command gains emoji and progress marker output matching the build command's visual style.

**Current state:** colonize.md Step 6 displays results as plain text. The colonizer workers already have emoji in their Visual Identity sections (added in v4.2), but the Queen's colonize command display lacks the box-drawing headers, step progress, and visual formatting that build.md has.

**Changes needed:**
1. Add box-drawing header at start (like build.md has)
2. Add step progress display (checkmarks for each completed step)
3. Add emoji and progress markers for each colonizer spawn
4. Add colonizer progress (1/3, 2/3, 3/3 for STANDARD/FULL mode)
5. Add synthesis step visual indicator
6. ANSI color for colonizer output (cyan)

### Pattern 6: Tech Debt Report (INT-06)

**What:** When all phases are complete, generate an aggregated tech debt report from errors.json and activity log.

**Where:** continue.md Step 2 currently detects "no next phase" and outputs:
```
All phases complete. Colony has finished the project plan.
```

Add tech debt report generation before this output.

**Data sources:**
1. `errors.json` -- `flagged_patterns` array (recurring error categories with counts, first/last seen)
2. `errors.json` -- full `errors` array (individual errors with category, severity, phase, description)
3. Activity log -- entries across all phases (via `activity-log-read` with no filter)
4. `memory.json` -- `phase_learnings` array (accumulated learnings)

**Report format:**

```
TECH DEBT REPORT
================

Project: {goal}
Phases Completed: {count}

Persistent Issues (from flagged patterns):
  {category}: {count} occurrences ({first_seen} - {last_seen})
    {description}

Error Distribution:
  By Severity: {critical}: N, {high}: N, {medium}: N, {low}: N
  By Category: {category}: N, ...

Unresolved Items:
  {errors with severity HIGH or CRITICAL that were never resolved}

Phase Quality Trends:
  Phase 1: {quality from watcher score if available}
  Phase 2: ...

Recommendations:
  {1-3 actionable items derived from the data}
```

**Storage:** Per CONTEXT.md, the tech debt report lives within Aether's state, not .planning/. Write to `.aether/data/tech-debt-report.md` (or display only -- Claude's discretion).

### Anti-Patterns to Avoid

- **New caste creation for reviewer/debugger:** Do NOT create reviewer-ant.md or debugger-ant.md files. They are existing castes (watcher, builder) with constrained prompts. The six-caste architecture is intentional.
- **Blocking reviewer:** The reviewer MUST be advisory only. Its findings display but do not halt unless CRITICAL. Non-critical findings should never prevent wave progression.
- **Colored markdown text:** Claude's own text output is plain markdown. Do NOT put ANSI escape codes in the LLM's text response -- they will render as literal characters. Use Bash tool printf/echo for colored output.
- **Live spinners/animations:** Task tool is synchronous. Sub-agents buffer all output. No streaming. Stick to static progress bars and status lines displayed between worker returns.
- **Debugger rewriting code:** Per CONTEXT.md, the debugger patches existing code. It must NOT start fresh or rewrite the worker's approach.
- **Too many pheromone suggestions:** Max 3 per build. Force prioritization.

## Don't Hand-Roll

Problems with existing solutions in the codebase:

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Reviewer quality assessment | New review logic | Existing watcher-ant.md scoring rubric | The rubric from Phase 29 (Correctness/Completeness/Quality/Safety/Integration with weights) is exactly what the reviewer needs. Spawning a watcher in advisory mode inherits all of this. |
| Error aggregation for tech debt | Custom error counting | `aether-utils.sh error-summary` + `error-pattern-check` | These subcommands already aggregate by category/severity and detect recurring patterns. |
| Progress bar rendering | Custom bar logic | Pattern from research/STACK.md | The `printf '%*s' "$filled" '' | tr ' ' '#'` pattern is already documented and tested. |
| Activity log reading | Custom log parsing | `aether-utils.sh activity-log-read` | Already exists, supports caste filtering, returns last 20 entries. |
| Pheromone validation | Custom validation | `aether-utils.sh pheromone-validate` | Already validates content length (>20 chars minimum). |

**Key insight:** Phase 30 features are orchestration-level additions to build.md and continue.md. They compose existing capabilities (watcher scoring, builder fixing, error tracking, activity logging) rather than building new ones.

## Common Pitfalls

### Pitfall 1: Reviewer Blocking Progress

**What goes wrong:** Reviewer findings treated as mandatory gates, turning advisory mode into blocking mode. Build stalls waiting for issue resolution.
**Why it happens:** The watcher-ant.md spec has a "request_changes" recommendation that implies blocking. The Phase Lead or Queen may interpret non-CRITICAL findings as requiring action.
**How to avoid:** The reviewer spawn prompt MUST explicitly state "ADVISORY mode -- findings displayed but do not block." The Queen's post-reviewer logic MUST only check `critical_count`, not `recommendation`.
**Warning signs:** Build pausing after reviewer returns with only HIGH/MEDIUM issues.

### Pitfall 2: Debugger Rewriting Instead of Patching

**What goes wrong:** Debugger spawned with builder-ant.md spec rewrites the entire file instead of applying a minimal fix. The original worker's approach is lost.
**Why it happens:** builder-ant.md's default workflow is "implement code" which naturally leads to full implementations. Without explicit constraints, the builder instinct is to write, not patch.
**How to avoid:** Debugger spawn prompt MUST include explicit constraints: "PATCH the existing code. Do NOT rewrite from scratch. Preserve the original worker's approach."
**Warning signs:** Debugger output showing large file rewrites instead of targeted changes.

### Pitfall 3: ANSI Colors in LLM Text Output

**What goes wrong:** ANSI escape codes placed in the Queen's direct text output render as literal `\e[32m` characters instead of green text.
**Why it happens:** Claude's text output is rendered as markdown, not terminal escape sequences. Only Bash tool output passes through the terminal emulator.
**How to avoid:** ALL colored output must go through Bash tool calls: `bash -c 'printf "\e[32mtext\e[0m\n"'`. The Queen's own text between bash calls remains plain.
**Warning signs:** Literal escape code characters visible in output.

### Pitfall 4: Reviewer Running After Every Wave in LIGHTWEIGHT Mode

**What goes wrong:** LIGHTWEIGHT projects get reviewer overhead after every wave, negating the "lean mode for small projects" intent.
**Why it happens:** AUTO-01 adds reviewer to the per-wave loop without mode checking.
**How to avoid:** Add mode check: skip reviewer in LIGHTWEIGHT mode (similar to how Step 5.5 already skips watcher in LIGHTWEIGHT). STANDARD and FULL modes get the reviewer.
**Warning signs:** LIGHTWEIGHT builds taking significantly longer than before Phase 30.

### Pitfall 5: Debugger Spawning Immediately Without Worker Retry

**What goes wrong:** Worker fails once, debugger spawns immediately without giving the worker a retry chance.
**Why it happens:** CONTEXT.md decision says "worker gets one retry attempt first, debugger triggers on second failure" but the implementation might miss this ordering.
**How to avoid:** The debugger spawn MUST be in Step 5c.f2 (after retry fails), NOT in Step 5c.f (the retry itself). The existing retry logic in Step 5c.f stays unchanged.
**Warning signs:** Debugger spawning on first worker failure.

### Pitfall 6: Tech Debt Report on Every Continue

**What goes wrong:** Tech debt report generates on every `/ant:continue` call, not just at project completion.
**Why it happens:** Misplacing the trigger -- it should only fire when "no next phase" is detected.
**How to avoid:** Tech debt report generation is conditional on `current_phase` being the last phase. Only triggers in the "All phases complete" branch of Step 2.
**Warning signs:** Tech debt report appearing mid-project.

### Pitfall 7: Pheromone Recommendations as Commands

**What goes wrong:** Recommendations formatted as copy-paste commands like "Run: /ant:focus 'auth module'" instead of natural language guidance.
**Why it happens:** Default LLM behavior to suggest actionable commands.
**How to avoid:** Per CONTEXT.md, recommendations are "natural language descriptive guidance, not copy-paste commands." The prompt must explicitly instruct: "Use descriptive language like a senior engineer's observation."
**Warning signs:** Recommendations starting with "Run:" or "/ant:".

## Code Examples

### Example 1: Advisory Reviewer Spawn in Wave Loop

This integrates into build.md Step 5c, after the post-wave conflict check (Step 5c.h):

```markdown
**i. Post-Wave Advisory Review:**

**Mode Check:** Read COLONY_STATE.json mode field.
If mode is "LIGHTWEIGHT": Skip reviewer. Continue to next wave.

Otherwise:

1. Read `.aether/workers/watcher-ant.md`
2. Spawn reviewer via Task tool with `subagent_type="general-purpose"`:

--- WORKER SPEC ---
{full contents of .aether/workers/watcher-ant.md}

--- ACTIVE PHEROMONES ---
{pheromone block from Step 3}

--- TASK ---
You are being spawned as a post-wave ADVISORY REVIEWER.
{... see Pattern 1 above for full prompt ...}

3. After reviewer returns:
   Parse critical_count from the reviewer's findings.

   Display reviewer summary inline:
   Use Bash tool:
   bash -c 'printf "\e[34m[REVIEWER]\e[0m Wave {N}: {summary} ({issue_count} findings)\n"'

   If findings exist, display each:
   bash -c 'printf "  \e[34m%s\e[0m: %s\n" "{severity}" "{description}"'

   If critical_count > 0 AND wave_rebuild_count < 2:
     Display: "CRITICAL issue detected. Rebuilding wave {N}..."
     Increment wave_rebuild_count.
     Re-run this wave's workers with findings appended to their prompts.

   If critical_count > 0 AND wave_rebuild_count >= 2:
     Display: "CRITICAL issues persist after 2 rebuilds. Continuing to next wave."

   If critical_count == 0:
     Continue to next wave.
```

### Example 2: Debugger Spawn on Retry Failure

This modifies build.md Step 5c.f and adds Step 5c.f2:

```markdown
f. **If worker failed and retry count < 1:**
   (Worker gets one retry attempt -- unchanged from current behavior)
   Log retry, spawn new worker with failure context.
   Increment retry counter.

f2. **If worker retry also failed (retry count >= 1):**
   Log debugger spawn:
   bash .aether/aether-utils.sh activity-log "SPAWN" "queen" "debugger-ant for: {task_description}"

   Read `.aether/workers/builder-ant.md`
   Spawn debugger via Task tool:

   --- WORKER SPEC ---
   {full contents of .aether/workers/builder-ant.md}

   --- TASK ---
   You are being spawned as a DEBUGGER ANT.
   {... see Pattern 2 above for full prompt ...}

   After debugger returns:
   If debugger reports fix_applied == true:
     Mark task as completed.
     Log: bash .aether/aether-utils.sh activity-log "COMPLETE" "debugger-ant" "Fixed: {diagnosis}"
     Display: "Debugger fixed: {diagnosis}"
   Else:
     Log: bash .aether/aether-utils.sh activity-log "ERROR" "debugger-ant" "Could not fix: {diagnosis}"
     Display: "Debugger could not fix: {diagnosis}."
     Mark task as failed.
     Continue to next worker.

g. (Remove the old "retry count >= 2" give-up logic -- replaced by debugger)
```

### Example 3: ANSI Colored Worker Status

Replaces the plain text worker status in build.md Step 5c.e:

```markdown
e. **After worker returns:**
   ...
   Display worker result with ANSI color via Bash tool:

   For builder:
   bash -c 'printf "\e[32m%-12s\e[0m %s ... \e[32m%s\e[0m\n" "[BUILDER]" "{task_description}" "COMPLETE"'

   For watcher:
   bash -c 'printf "\e[35m%-12s\e[0m %s ... \e[35m%s\e[0m\n" "[WATCHER]" "{task_description}" "COMPLETE"'

   For error:
   bash -c 'printf "\e[31m%-12s\e[0m %s ... \e[31m%s\e[0m\n" "[{CASTE}]" "{task_description}" "ERROR"'

   Color mapping:
     builder  -> \e[32m (green)
     watcher  -> \e[35m (magenta)
     colonizer -> \e[36m (cyan)
     scout    -> \e[34m (blue)
     architect -> \e[37m (white)
     route-setter -> \e[33m (yellow)
     debugger -> \e[31m (red)
     reviewer -> \e[34m (blue)

   Progress bar (caste-colored):
   bash -c 'printf "\e[{color_code}m[%s%s]\e[0m %d/%d workers complete\n" "{filled}" "{empty}" {completed} {total}'
```

### Example 4: Pheromone Recommendations Display

Added to build.md Step 7e after existing display:

```markdown
**Pheromone Recommendations:**

Analyze the build outcomes to generate max 3 recommendations:

Sources:
- worker_results: which tasks succeeded/failed, error summaries
- watcher_report: quality score, issue list
- errors.json: flagged_patterns for recurring issues
- Per-wave reviewer findings (from Step 5c.i)

For each recommendation:
- Must reference a SPECIFIC observation from this build
- Must be natural language guidance, not a command
- Must suggest a DIRECTION, not a specific action

Display (max 3):
```
Pheromone Recommendations:
  Based on this build's outcomes:

  1. {recommendation}
     Signal: {what triggered this}

  2. {recommendation}
     Signal: {what triggered this}

  These are suggestions, not commands. Use /ant:focus or /ant:redirect to act on them.
```

If no meaningful patterns emerge, display:
```
  No specific recommendations -- build was clean.
```
```

### Example 5: Tech Debt Report at Project Completion

Added to continue.md Step 2, in the "no next phase" branch:

```markdown
### Step 2.5: Generate Tech Debt Report (Project Completion)

This step runs only when Step 2 detects all phases are complete.

1. Read `.aether/data/errors.json`
2. Run: bash .aether/aether-utils.sh error-summary
3. Run: bash .aether/aether-utils.sh error-pattern-check
4. Read `.aether/data/memory.json` (phase_learnings)
5. Read `.aether/data/activity.log`

Synthesize a tech debt report:

Display:
```
TECH DEBT REPORT
================
Project: {goal}
Phases Completed: {count}
Total Build Time: {estimate from activity log timestamps}

Persistent Issues:
{for each flagged_pattern:}
  {category} ({count} occurrences, {first_seen} - {last_seen}):
    {description}

Error Summary:
  Total: {total}
  By Severity: Critical: {n}, High: {n}, Medium: {n}, Low: {n}
  By Category: {category}: {n}, ...

Unresolved High-Severity Items:
  {errors with severity "critical" or "high" -- these are potential tech debt}

Phase Quality Trend:
  {from phase_learnings, show error counts per phase}

Recommendations:
  {1-3 actionable items based on patterns}
```

Also write the report to `.aether/data/tech-debt-report.md` for persistence.
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Single watcher at end of build | Watcher after all workers + reviewer after each wave | Phase 30 (new) | Issues caught per-wave instead of at the end, reducing cascading problems |
| Worker retry, then give up | Worker retry, then debugger diagnose+fix, then give up | Phase 30 (new) | Failed tasks get intelligent diagnosis before being abandoned |
| User manually interprets build outcomes | Pheromone recommendations surfaced automatically | Phase 30 (new) | Colony guidance becomes proactive, not reactive |
| Plain text build output | ANSI-colored caste-specific output | Phase 30 (new) | Visual distinction between castes, instant pattern recognition |
| No project completion report | Tech debt report at completion | Phase 30 (new) | Accumulated issues surfaced at project end for user awareness |

**Existing patterns that remain unchanged:**
- Watcher verification (Step 5.5) remains as the final quality gate -- reviewer is an additional per-wave advisory check
- Activity log, error tracking, pheromone system all continue as-is
- Worker spawn pattern (Task tool with full spec injection) identical for reviewer/debugger

## Open Questions

### 1. Reviewer Cost vs Value for Simple Waves

**What we know:** Each reviewer spawn is a Task tool call with the full watcher-ant.md spec (~560 lines). For a 2-task phase with 1 wave, this adds a reviewer spawn that may be unnecessary.
**What's unclear:** Whether the cost (time + tokens) of per-wave review is worth it for simple phases.
**Recommendation:** Skip reviewer for waves with only 1 worker (no cross-worker interaction to catch). Only spawn reviewer for waves with 2+ workers. This keeps LIGHTWEIGHT mode lean and avoids unnecessary overhead for single-worker waves.

### 2. ANSI Color Rendering Across Terminals

**What we know:** ANSI escape codes work in Claude Code's bash output (verified via GitHub issue #18728). The research/STACK.md documents the color scheme.
**What's unclear:** Whether all users' terminal configurations render 8-color ANSI codes correctly. Some terminals with custom themes may have poor contrast for certain colors.
**Recommendation:** Use the basic 8-color ANSI codes (30-37) which are universally supported. Avoid 256-color or truecolor codes. This is what the existing research recommends.

### 3. Tech Debt Report Persistence Format

**What we know:** CONTEXT.md says "Tech debt report lives within Aether's state, not .planning/" and format is Claude's discretion.
**What's unclear:** Whether to write a file (`.aether/data/tech-debt-report.md`) or only display in terminal.
**Recommendation:** Both -- display in terminal AND write to `.aether/data/tech-debt-report.md`. The display ensures the user sees it; the file ensures persistence across sessions.

### 4. Debugger Failure Behavior

**What we know:** CONTEXT.md says debugger failure behavior is Claude's discretion "based on task criticality."
**What's unclear:** How to determine "task criticality" at runtime -- the PROJECT_PLAN.json task schema doesn't have a criticality field.
**Recommendation:** Infer criticality from the success criteria. If the failed task directly maps to a phase success criterion, treat it as critical (halt and notify user). If it's a supporting task, skip and continue. The Queen already has the success criteria in memory from Step 2.

## Sources

### Primary (HIGH confidence)

- `.claude/commands/ant/build.md` -- Full build orchestration flow, 819 lines. Contains the exact integration points for reviewer (Step 5c.h), debugger (Step 5c.f-g), visual output (Step 5c.e, Step 7e), and pheromone recommendations (Step 7e).
- `.aether/workers/watcher-ant.md` -- Full watcher spec, 564 lines. Contains scoring rubric (Correctness 0.30, Completeness 0.25, Quality 0.20, Safety 0.15, Integration 0.10), specialist modes, execution verification. This is the reviewer's base spec.
- `.aether/workers/builder-ant.md` -- Full builder spec, 354 lines. Contains implementation workflow and spawn patterns. This is the debugger's base spec.
- `.claude/commands/ant/continue.md` -- Full continue flow, 440 lines. Step 2 "no next phase" is where tech debt report triggers.
- `.claude/commands/ant/colonize.md` -- Full colonize flow, 396 lines. Step 6 display is where visual output (AUTO-05) is added.
- `.aether/aether-utils.sh` -- Utility layer, 302 lines. 16 subcommands. error-summary, error-pattern-check, activity-log-read are key for tech debt aggregation.
- `.planning/research/STACK.md` -- Prior research on ANSI color codes, progress bars, and CLI animation patterns for Aether. Confirmed approach.

### Secondary (MEDIUM confidence)

- [GitHub Issue #18728](https://github.com/anthropics/claude-code/issues/18728) -- ANSI color codes confirmed working in Claude Code bash output (closed as resolved, Jan 2026).
- `.planning/v5-FIELD-NOTES.md` -- Field note 8 originally proposed auto-spawned reviewer/debugger concept.
- `.planning/phases/30-automation/30-CONTEXT.md` -- User decisions constraining this phase's implementation.

### Tertiary (LOW confidence)

- None. All findings verified against primary codebase sources.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH -- All changes are to existing files, no new dependencies
- Architecture (reviewer/debugger integration): HIGH -- Exact insertion points identified in build.md
- Architecture (ANSI visual output): HIGH -- Verified ANSI works in Claude Code bash, pattern documented in research/STACK.md
- Architecture (tech debt report): HIGH -- Data sources (errors.json, activity.log) and aggregation utilities already exist
- Pitfalls: HIGH -- Based on direct analysis of existing code patterns and CONTEXT.md constraints

**Research date:** 2026-02-05
**Valid until:** 2026-03-05 (stable -- all changes are prompt-level, no external dependencies to change)
