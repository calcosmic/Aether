# Phase 27: Bug Fixes & Safety Foundation - Research

**Researched:** 2026-02-04
**Domain:** Aether colony system bug fixes -- pheromone decay math, activity log persistence, error attribution, decision logging, same-file conflict prevention
**Confidence:** HIGH

## Summary

Phase 27 addresses five known defects from Aether's first real-world field test, all with verified root causes and known fixes. This is the foundation phase -- every subsequent v4.4 feature depends on correct pheromone signals, persistent activity logs, phase-attributed errors, wired decision logging, and conflict-free parallel execution.

The five requirements are:
1. **BUG-01**: Pheromone decay math -- FOCUS strength growing instead of decaying (root cause: negative elapsed time produces exponential growth when guards are missing)
2. **BUG-02**: Activity log append -- `activity-log-init` overwrites previous phases instead of archiving and preserving cross-phase entries
3. **BUG-03**: Error phase attribution -- `error-add` utility inserts `phase: null` because build.md never passes the current phase number
4. **BUG-04**: Decision logging during execution -- `build.md` has no decision-recording step, so only pheromone commands (`/ant:focus`, `/ant:redirect`, `/ant:feedback`, `/ant:colonize`) record decisions
5. **INT-02**: Same-file conflict prevention -- Phase Lead must group tasks touching the same file to the same worker

All five fixes stay within the existing bash+jq stack. No new dependencies are needed. The most complex change is the Phase Lead's conflict prevention rule (INT-02), which is a prompt-level addition, not a code change.

**Primary recommendation:** Fix all five issues with defensive, minimal changes to existing files. Do not redesign systems -- patch the specific failure points.

## Standard Stack

The established libraries/tools for this domain:

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| bash | 4.0+ | All utility functions, activity logging | Already the utility layer language. No change. |
| jq | 1.6+ | JSON manipulation, `exp` for decay math, `fromdate` for timestamps | Already used. IEEE754 double precision sufficient for decay calculations. |
| POSIX utilities | Standard | `date`, `mv`, `echo`, `>>` | Already used. `>>` append mode is the fix for BUG-02. |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `aether-utils.sh` | Extended (5 modifications) | Core utility layer | All five fixes modify or add subcommands in this file |
| `atomic-write.sh` | Existing | Safe JSON file writes | Used by error-add and pheromone utilities |
| `file-lock.sh` | Existing | File-level locking | Already provides byte-level safety for parallel writes |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| jq `exp()` for decay | `bc -l` arbitrary precision | jq already loaded; IEEE754 doubles are more than sufficient for signal strengths. Adding bc is unnecessary. |
| Prompt-level conflict prevention | Git worktrees per worker | Massive overkill. Workers run sequentially within waves. Aether is CLI-native, not git-workflow-native. |
| Phase field in error-add | Separate error-add-phased subcommand | Cleaner to modify existing error-add to accept optional phase param than to add a new subcommand. |

**Installation:**
```bash
# No new dependencies. All fixes modify existing files.
```

## Architecture Patterns

### Recommended File Changes
```
.aether/
  aether-utils.sh                # MODIFY: pheromone-decay, pheromone-batch, pheromone-cleanup, error-add, activity-log-init
.claude/commands/ant/
  build.md                       # MODIFY: Step 5a (conflict prevention), Step 5c (pass phase to error-add), add decision logging step
```

### Pattern 1: Defensive Decay Math Guards
**What:** Three guards added to pheromone-decay, pheromone-batch, and pheromone-cleanup to prevent strength growth.
**When to use:** Always -- these guards are unconditional.
**Example:**
```bash
# Source: Stack research STACK.md, verified against aether-utils.sh line 44-48
pheromone-decay)
  [[ $# -ge 3 ]] || json_err "Usage: pheromone-decay <strength> <elapsed_seconds> <half_life>"
  json_ok "$(jq -n --arg s "$1" --arg e "$2" --arg h "$3" '
    ($s|tonumber) as $strength |
    ([$e|tonumber, 0] | max) as $elapsed |     # GUARD 1: clamp elapsed >= 0
    ($h|tonumber) as $half_life |
    if $elapsed > ($half_life * 10) then
      {strength: 0}                             # GUARD 2: skip computation, effectively zero
    else
      ($strength * ((-0.693147180559945 * $elapsed / $half_life) | exp)) as $decayed |
      {strength: ([$decayed, $strength] | min | . * 1000000 | round / 1000000)}  # GUARD 3: cap at initial
    end
  ')"
  ;;
```

**Guard rationale:**
- **Guard 1** (`max(elapsed, 0)`): Prevents negative elapsed time, which was the root cause of the 8.005 bug. A negative elapsed flips the exponent sign, causing exponential growth.
- **Guard 2** (`elapsed > half_life * 10` -> 0): After 10 half-lives, strength is below 0.001 (2^-10 = 0.00098). Skip the `exp()` computation entirely to avoid floating-point edge cases.
- **Guard 3** (`min(decayed, strength)`): Clamp the result to never exceed the initial strength. Belt-and-suspenders -- if any upstream computation is wrong, the signal cannot grow.

### Pattern 2: Append-Mode Activity Logging
**What:** `activity-log-init` archives the current log and starts a new one, but the current log header writes with `>` (truncate). Additionally, activity entries from all phases should remain accessible.
**When to use:** At phase boundaries when `activity-log-init` is called.
**Example:**
```bash
# Current (broken): mv + truncate loses cross-phase accessibility
# Fix: mv + append header to new file, but also ensure a combined log exists

activity-log-init)
  phase_num="${1:-}"
  phase_name="${2:-}"
  [[ -z "$phase_num" ]] && json_err "Usage: activity-log-init <phase_num> [phase_name]"
  log_file="$DATA_DIR/activity.log"
  archive_file="$DATA_DIR/activity-phase-${phase_num}.log"
  # Copy current log to archive (not move -- keep combined log intact)
  if [ -f "$log_file" ] && [ -s "$log_file" ]; then
    cp "$log_file" "$archive_file"
  fi
  # Append phase header to the EXISTING combined log (not truncate)
  ts=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
  echo "" >> "$log_file"
  echo "# Phase $phase_num: ${phase_name:-unnamed} -- $ts" >> "$log_file"
  archived_flag="false"
  [ -f "$archive_file" ] && archived_flag="true"
  json_ok "{\"archived\":$archived_flag}"
  ;;
```

**Key insight:** The current code uses `mv` to move the log then `>` to create a fresh one. This means `activity.log` only ever contains the current phase. The fix changes this to `cp` (preserve archive) + `>>` (append phase header to combined log). After 3 phases, `activity.log` contains entries from all 3 phases.

### Pattern 3: Phase-Attributed Errors
**What:** `error-add` already has `phase: null` in its schema. The fix is twofold: (a) accept an optional phase parameter, and (b) have build.md pass the current phase number.
**When to use:** Every call to `error-add` during a build phase.
**Example:**
```bash
# Modified error-add: accepts optional 4th arg for phase number
error-add)
  [[ $# -ge 3 ]] || json_err "Usage: error-add <category> <severity> <description> [phase]"
  [[ -f "$DATA_DIR/errors.json" ]] || json_err "errors.json not found"
  id="err_$(date -u +%s)_$(head -c 2 /dev/urandom | od -An -tx1 | tr -d ' ')"
  ts=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
  phase_val="${4:-null}"
  # If phase_val is a number, use it; otherwise null
  if [[ "$phase_val" =~ ^[0-9]+$ ]]; then
    phase_jq="$phase_val"
  else
    phase_jq="null"
  fi
  updated=$(jq --arg id "$id" --arg cat "$1" --arg sev "$2" --arg desc "$3" --argjson phase "$phase_jq" --arg ts "$ts" '
    .errors += [{id:$id, category:$cat, severity:$sev, description:$desc, root_cause:null, phase:$phase, task_id:null, timestamp:$ts}] |
    if (.errors|length) > 50 then .errors = .errors[-50:] else . end
  ' "$DATA_DIR/errors.json") || json_err "Failed to update errors.json"
  atomic_write "$DATA_DIR/errors.json" "$updated"
  json_ok "\"$id\""
  ;;
```

**build.md changes:** Every `error-add` call in Step 6 becomes:
```
bash .aether/aether-utils.sh error-add "<category>" "<severity>" "<description>" <phase_number>
```

### Pattern 4: Decision Logging During Execution
**What:** build.md currently has NO step that records decisions to `memory.json`. Decisions like "assign all README tasks to one worker" are made by the Phase Lead but never written to the decisions array. Only pheromone commands (`/ant:focus`, `/ant:redirect`, `/ant:feedback`) and `/ant:colonize` write decisions.
**When to use:** After the Phase Lead returns a plan (Step 5b) and after watcher verification (Step 5.5).
**Example:**
```markdown
# New step in build.md after Step 5b (Plan Checkpoint):

**Record Plan Decisions:** Read `.aether/data/memory.json`. For each significant planning
decision in the Phase Lead's plan (e.g., caste assignment, wave grouping, task ordering),
append a decision record to the `decisions` array:

{
  "id": "dec_<unix_timestamp>_<4_random_hex>",
  "type": "plan",
  "content": "<decision summary -- e.g. 'Grouped tasks 3.1 and 3.2 to same builder because both touch routes/index.ts'>",
  "context": "Phase <id> -- Phase Lead plan",
  "phase": <current_phase>,
  "timestamp": "<ISO-8601 UTC>"
}

If the `decisions` array exceeds 30 entries, remove the oldest entries to keep only 30.
Write the updated memory.json.
```

**Key insight:** The gap is that build.md has no decision-recording logic at all. Pheromone commands each have a "Step 4: Log Decision" but build.md skips this entirely. The fix adds decision logging at two points: (1) after plan approval (recording the Phase Lead's strategic decisions), and (2) after watcher verification (recording quality-related decisions like "approved despite medium severity issues").

### Pattern 5: Same-File Conflict Prevention in Phase Lead
**What:** Add a conflict prevention rule to the Phase Lead's planning prompt so it groups tasks touching the same file to the same worker.
**When to use:** During Step 5a when the Phase Lead prompt is constructed.
**Example:**
```markdown
# Addition to Phase Lead prompt in build.md Step 5a:

--- CONFLICT PREVENTION RULE ---
CRITICAL: Tasks that modify the SAME FILE must be assigned to the SAME WORKER in the
same wave. Before assigning tasks to workers:
1. Identify which files each task will likely create or modify (from task descriptions)
2. If two or more tasks reference the same file, assign them to the same worker
3. This prevents parallel write conflicts where one builder overwrites another's changes

Example:
  Task 3.1: Add auth routes to src/routes/index.ts
  Task 3.2: Add API routes to src/routes/index.ts
  -> Both touch src/routes/index.ts -> assign to SAME builder-ant

  Task 3.3: Create middleware at src/middleware/auth.ts
  -> Different file -> can go to a different builder-ant

If uncertain about file overlap, default to assigning tasks to the same worker.
```

**Queen-side backup (build.md Step 5c):** After receiving the plan, before executing workers, the Queen scans task descriptions for file path overlaps. If overlap is detected between tasks assigned to different workers in the same wave, merge those tasks into a single worker assignment.

### Anti-Patterns to Avoid
- **LLM-computed decay math**: NEVER fall back to manual `exp()` calculation. If `aether-utils.sh` is unavailable, treat pheromones at their raw initial strength (fail-open). The LLM getting the sign wrong was the root cause of BUG-01.
- **Full activity log truncation**: NEVER use `>` (truncate) on `activity.log` during phase init. Always `>>` (append) for the combined log.
- **Git worktrees for conflict prevention**: Overkill. Task grouping at planning time is simpler, lighter, and sufficient.
- **Separate subcommand for phased errors**: Modifying the existing `error-add` to accept an optional phase parameter is cleaner than adding a new `error-add-phased` subcommand. Backward compatible -- existing calls with 3 args still work (phase defaults to null).

## Don't Hand-Roll

Problems that look simple but have existing solutions:

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Pheromone decay | Custom LLM math | jq `exp()` via aether-utils.sh | LLMs cannot reliably compute transcendental functions. Field test proved this: strength grew from 0.7 to 8.005. |
| File conflict detection | Complex dependency graph analyzer | Prompt-level rule in Phase Lead + simple text scan by Queen | Same-file detection from task descriptions is sufficient. The colony already learned this lesson (field notes 10, 13, 30). |
| Atomic JSON writes | Manual file locking + retry | `atomic-write.sh` (already exists) | Already handles backup, write, verify. Used by all state file updates. |
| Timestamp computation | Shell `date` arithmetic | jq `now`, `fromdate` | jq handles ISO-8601 parsing and epoch conversion. Already used throughout pheromone-batch. |

**Key insight:** Every fix in this phase uses existing infrastructure. No new utilities or patterns are needed -- only corrections to existing logic and additions to existing prompts.

## Common Pitfalls

### Pitfall 1: Negative Elapsed Time in Decay Math
**What goes wrong:** If `created_at` timestamp is in the future relative to `now` (due to timezone issues, clock drift, or string parsing errors), elapsed becomes negative. A negative elapsed flips the exponential decay into exponential growth: `e^(-(-x)) = e^x`.
**Why it happens:** jq's `fromdate` has known issues with DST and fractional seconds. The existing code strips fractional seconds via regex `sub("\\.[0-9]+Z$";"Z")` but does not guard against the resulting epoch being greater than `now`.
**How to avoid:** Guard 1 in the defensive decay math pattern: `([$e|tonumber, 0] | max)`. This clamps elapsed to >= 0 unconditionally.
**Warning signs:** Any pheromone `current_strength` greater than its initial `strength` value.

### Pitfall 2: Activity Log Phase Header Collision
**What goes wrong:** If the same phase number is built twice (retry scenario), `activity-log-init` is called twice with the same phase number. The archive file `activity-phase-N.log` gets overwritten on the second call.
**Why it happens:** The archive filename is deterministic based on phase number only, not timestamp.
**How to avoid:** Check if the archive file already exists before copying. If it does, append a numeric suffix: `activity-phase-3.log`, `activity-phase-3-2.log`, etc. Or use timestamp-based archive names: `activity-phase-3-20260204T140000Z.log`.
**Warning signs:** Archive file size is smaller than expected (overwritten by a retry's smaller log).

### Pitfall 3: Decision Array Bloat from Over-Logging
**What goes wrong:** If every Phase Lead decision is logged as a separate entry, a complex phase with 8 tasks could generate 8+ decision entries per build. The 30-entry cap means decisions from just 3-4 phases fill the array, evicting older strategic decisions.
**Why it happens:** The temptation is to log granular decisions ("assigned task 3.1 to builder", "assigned task 3.2 to builder"). These are low-value individually.
**How to avoid:** Log decisions at the STRATEGIC level, not the task level. One decision per plan approval ("Grouped tasks by file ownership: 3.1+3.2 to builder-A on routes/index.ts, 3.3 to builder-B on middleware/auth.ts"). One decision per watcher verdict ("Approved Phase 3 with quality 7/10, deferred 2 medium issues to tech debt"). Target 2-3 decisions per phase maximum.
**Warning signs:** `decisions` array at 30 entries after just 5 phases.

### Pitfall 4: Error-Add Phase Parameter Parsing
**What goes wrong:** The phase number is passed as a positional argument to a bash function. If the description (arg 3) contains spaces and is not properly quoted, word splitting could push extra words into arg 4, corrupting the phase value.
**Why it happens:** Shell word splitting on unquoted variables. The existing `error-add` calls in build.md pass the description as a quoted string, but copy-paste errors could lose quotes.
**How to avoid:** Use `--argjson phase` in jq to parse the phase as a number, with a fallback to null if it's not a valid number. The pattern shown in Code Examples handles this.
**Warning signs:** `phase` field in errors.json contains a string like "module" instead of a number.

### Pitfall 5: Phase Lead Ignoring Conflict Prevention Rule
**What goes wrong:** The Phase Lead is an LLM agent. Adding a rule to its prompt does not guarantee it will follow the rule every time. The Phase Lead might assign overlapping tasks to different workers if the file overlap is not obvious from task descriptions.
**Why it happens:** LLM instruction following is probabilistic. Long prompts with many rules see reduced compliance on later rules.
**How to avoid:** Two-layer defense: (1) rule in Phase Lead prompt (primary), (2) Queen-side validation in build.md before executing the plan (backup). The Queen scans the plan output for file references and detects overlaps. If overlaps are found between different workers, merge those tasks before spawning.
**Warning signs:** Watcher reports that previously verified work has reverted. Activity log shows two builders modifying the same file in the same wave.

## Code Examples

Verified patterns from direct codebase analysis:

### Defensive Pheromone Batch Processing
```bash
# Source: aether-utils.sh pheromone-batch (line 54-62) -- MODIFIED with guards
pheromone-batch)
  [[ -f "$DATA_DIR/pheromones.json" ]] || json_err "pheromones.json not found"
  now=$(date -u +%s)
  json_ok "$(jq --arg now "$now" '.signals | map(. + {
    current_strength: (
      if .half_life_seconds == null then .strength
      else
        (($now|tonumber) - (.created_at | sub("\\.[0-9]+Z$";"Z") | fromdate)) as $elapsed |
        if $elapsed < 0 then .strength                    # GUARD: future timestamp -> use raw strength
        elif $elapsed > (.half_life_seconds * 10) then 0  # GUARD: very old -> zero
        else
          (.strength * ((-0.693147180559945 * $elapsed / .half_life_seconds) | exp)) as $d |
          if $d > .strength then .strength else $d end    # GUARD: never exceed initial
        end
      end | . * 1000 | round / 1000)
  })' "$DATA_DIR/pheromones.json")" || json_err "Failed to read pheromones.json"
  ;;
```

### Error-Add with Optional Phase
```bash
# Source: aether-utils.sh error-add (line 180-191) -- MODIFIED with optional phase param
error-add)
  [[ $# -ge 3 ]] || json_err "Usage: error-add <category> <severity> <description> [phase]"
  [[ -f "$DATA_DIR/errors.json" ]] || json_err "errors.json not found"
  id="err_$(date -u +%s)_$(head -c 2 /dev/urandom | od -An -tx1 | tr -d ' ')"
  ts=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
  phase_val="${4:-null}"
  if [[ "$phase_val" =~ ^[0-9]+$ ]]; then
    phase_jq="$phase_val"
  else
    phase_jq="null"
  fi
  updated=$(jq --arg id "$id" --arg cat "$1" --arg sev "$2" --arg desc "$3" --argjson phase "$phase_jq" --arg ts "$ts" '
    .errors += [{id:$id, category:$cat, severity:$sev, description:$desc, root_cause:null, phase:$phase, task_id:null, timestamp:$ts}] |
    if (.errors|length) > 50 then .errors = .errors[-50:] else . end
  ' "$DATA_DIR/errors.json") || json_err "Failed to update errors.json"
  atomic_write "$DATA_DIR/errors.json" "$updated"
  json_ok "\"$id\""
  ;;
```

### Build.md Decision Logging Addition
```markdown
# New content for build.md -- inserted after Step 5b Plan Checkpoint, before Step 5c

### Step 5b-post: Record Plan Decisions

Read `.aether/data/memory.json`. Synthesize 2-3 strategic decisions from the approved plan.
Append each to the `decisions` array:

{
  "id": "dec_<unix_timestamp>_<4_random_hex>",
  "type": "plan",
  "content": "<strategic decision -- e.g. 'Grouped file-overlapping tasks 3.1+3.2 to single builder to prevent conflicts'>",
  "context": "Phase <id> plan -- <brief plan summary>",
  "phase": <current_phase_number>,
  "timestamp": "<ISO-8601 UTC>"
}

Cap at 30 entries (remove oldest if exceeded). Write updated memory.json.

Also after Step 5.5 (watcher verification), record the quality decision:

{
  "id": "dec_<unix_timestamp>_<4_random_hex>",
  "type": "quality",
  "content": "<watcher verdict -- e.g. 'Phase 3 approved at 7/10, deferred 2 medium issues'>",
  "context": "Phase <id> watcher verification",
  "phase": <current_phase_number>,
  "timestamp": "<ISO-8601 UTC>"
}
```

### Build.md Error-Add with Phase
```markdown
# Modified error-add calls in build.md Step 6 -- add phase number

# Before (current):
bash .aether/aether-utils.sh error-add "<category>" "<severity>" "<description>"

# After (fixed):
bash .aether/aether-utils.sh error-add "<category>" "<severity>" "<description>" <phase_number>
```

### Phase Lead Conflict Prevention Prompt Addition
```markdown
# Added to build.md Step 5a Phase Lead prompt, after "Available worker castes:"

--- CONFLICT PREVENTION RULE ---
CRITICAL: Tasks that modify the SAME FILE must be assigned to the SAME WORKER.

Before creating your plan:
1. For each task, identify which files it will likely create or modify
2. If two tasks reference the same file path, they MUST go to the same worker
3. If unsure about file overlap, group tasks conservatively (same worker)

This prevents parallel write conflicts where one builder overwrites another's work.
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| No decay guards | Three-guard defensive decay | This phase | Prevents pheromone strength growth bug |
| Log truncation at phase boundary | Append-mode with phase archives | This phase | Cross-phase activity history preserved |
| Phase-less error tracking | Phase-attributed errors | This phase | Errors traceable to source phase |
| Decisions only from pheromone commands | Decisions from build + pheromone commands | This phase | Execution decisions recorded |
| No file conflict awareness | Phase Lead groups by file | This phase | Prevents last-write-wins in parallel workers |

**Deprecated/outdated:**
- LLM fallback math: Worker specs contain "fall back to manual multiplication" for pheromone-effective. This is acceptable for simple multiplication. But for decay (transcendental `exp()`), LLM fallback MUST be eliminated. Workers should treat unavailable pheromone-batch as "all pheromones active at initial strength" (fail-open), never attempt manual `exp()` computation.

## Open Questions

Things that couldn't be fully resolved:

1. **Activity log combined vs archive-only**
   - What we know: Current code moves the log (destroying combined view). The fix preserves both combined and per-phase archives.
   - What's unclear: Should `activity-log-read` read the combined log or per-phase archives? How does this interact with the 100-event cap in events.json?
   - Recommendation: Combined log as primary (what `activity-log-read` returns). Per-phase archives as backup. Combined log has no size cap but individual phases are naturally bounded by worker count.

2. **Decision logging granularity**
   - What we know: 2-3 strategic decisions per phase is the target. 30-entry cap with oldest eviction.
   - What's unclear: Exactly which Phase Lead decisions are "strategic" enough to log. Risk of either over-logging (fills cap fast) or under-logging (decisions array stays empty).
   - Recommendation: Start with two fixed logging points: post-plan-approval and post-watcher-verification. Review after 3 phases and adjust.

3. **Phase Lead compliance with conflict prevention rule**
   - What we know: LLMs follow prompt rules probabilistically. The rule must be prominent and clear.
   - What's unclear: How reliably the Phase Lead will detect file overlaps from task descriptions alone. Some overlaps may not be obvious (e.g., "add error handling" might touch multiple files).
   - Recommendation: Two-layer defense -- Phase Lead prompt rule + Queen-side validation scan. Accept that prompt-level compliance won't be 100% and rely on the Queen's backup check.

## Sources

### Primary (HIGH confidence)
- `/Users/callumcowie/repos/Aether/.aether/aether-utils.sh` -- 265 lines, all 16 subcommands analyzed. Lines 44-48 (pheromone-decay), 54-62 (pheromone-batch), 64-74 (pheromone-cleanup), 180-191 (error-add), 226-250 (activity-log, activity-log-init) directly examined.
- `/Users/callumcowie/repos/Aether/.claude/commands/ant/build.md` -- 663 lines. No decision logging found. Error-add calls at Step 6 lack phase parameter. Phase Lead prompt at Step 5a lacks conflict prevention rule.
- `/Users/callumcowie/repos/Aether/.claude/commands/ant/continue.md` -- 319 lines. Step 3 attempts to show decisions and filter errors by phase, confirming both features are expected but currently produce empty results.
- `/Users/callumcowie/repos/Aether/.claude/commands/ant/focus.md` -- Decision logging pattern at Step 4 used as reference for build.md decision logging format.
- `/Users/callumcowie/repos/Aether/.aether/data/errors.json` -- Existing errors all have `phase: null`, confirming BUG-03.
- `/Users/callumcowie/repos/Aether/.aether/data/memory.json` -- Empty `decisions: []` array, confirming BUG-04.
- `/Users/callumcowie/repos/Aether/.planning/v5-FIELD-NOTES.md` -- Notes 10, 13, 17, 18, 19, 20, 30 directly describe these bugs.

### Secondary (MEDIUM confidence)
- `/Users/callumcowie/repos/Aether/.planning/research/STACK.md` -- Decay math root cause analysis, defensive guard pattern, jq `exp()` verification.
- `/Users/callumcowie/repos/Aether/.planning/research/PITFALLS.md` -- CP-4 (same-file conflicts), CP-1 (platform constraints for context).
- `/Users/callumcowie/repos/Aether/.planning/research/ARCHITECTURE.md` -- Component boundaries, same-file conflict prevention design (Section 5).
- `/Users/callumcowie/repos/Aether/.planning/research/FEATURES.md` -- Conflict prevention competitor analysis.

### Tertiary (LOW confidence)
- None. All findings are based on direct codebase analysis and verified prior research.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH -- no new dependencies, all modifications to existing files
- Architecture: HIGH -- minimal changes, well-understood failure modes with verified root causes
- Pitfalls: HIGH -- all five pitfalls grounded in actual field test data (notes 10, 13, 17, 18, 19, 20)
- Code examples: HIGH -- derived from direct reading of current source code, modifications are minimal

**Research date:** 2026-02-04
**Valid until:** Indefinite for bug fixes (these are deterministic corrections, not trend-dependent)
