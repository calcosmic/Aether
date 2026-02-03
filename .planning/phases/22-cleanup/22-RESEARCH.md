# Phase 22: Cleanup - Research

**Researched:** 2026-02-03
**Domain:** Shell command cleanup -- wiring orphaned subcommands and removing dead code
**Confidence:** HIGH

## Summary

Phase 22 is a pure refactoring phase. Four command files (`plan.md`, `pause-colony.md`, `resume-colony.md`, `colonize.md`) contain inline pheromone decay formulas that must be replaced with `pheromone-batch` calls. Two command files (`continue.md`, `build.md`) contain manual error counting/categorization logic that must be replaced with `error-summary` and `error-pattern-check` calls. One command file (`continue.md`) contains manual array truncation that must be replaced with a `memory-compress` call. Four subcommands (`pheromone-combine`, `memory-token-count`, `memory-search`, `error-dedup`) must be removed as they have zero consumers.

The existing `build.md` and `status.md` commands already demonstrate the correct `pheromone-batch` call pattern. This is a copy-and-adapt operation for the other four commands. All utility subcommand interfaces are stable and documented in `aether-utils.sh`.

**Primary recommendation:** Execute as five atomic tasks matching CLEAN-01 through CLEAN-05. Each task is a direct text replacement in markdown prompt files -- no new code creation required.

## Standard Stack

No new libraries or tools are needed. All work is editing existing markdown command files and a single shell script.

### Core
| Tool | Version | Purpose | Why Standard |
|------|---------|---------|--------------|
| aether-utils.sh | 0.1.0 | Central utility layer | Already exists, provides all replacement subcommands |
| jq | system | JSON processing within shell | Used by aether-utils.sh internally |

### Supporting
None required -- this is pure edit work on existing files.

### Alternatives Considered
None -- the approach is fully specified in the requirements.

## Architecture Patterns

### Pattern 1: pheromone-batch Call Pattern (from build.md, lines 43-50)

**What:** Replace inline decay formula with a `pheromone-batch` Bash call
**When to use:** Any command that needs to compute current pheromone strengths

**Reference implementation (already working in build.md Step 3, lines 43-50):**
```markdown
### Step 3: Compute Active Pheromones

Use the Bash tool to run:
\`\`\`
bash .aether/aether-utils.sh pheromone-batch
\`\`\`

This returns JSON: `{"ok":true,"result":[...signals with current_strength...]}`. Parse the `result` array. Filter out signals where `current_strength < 0.05`.

If the command fails, treat as "no active pheromones."
```

**Also working in status.md Step 2, lines 41-48.**

### Pattern 2: error-summary Call Pattern (to be wired)

**What:** Replace manual error counting by severity/category with `error-summary` call
**Interface:** `bash .aether/aether-utils.sh error-summary`
**Returns:** `{"ok":true,"result":{"total":N,"by_category":{...},"by_severity":{...}}}`

### Pattern 3: error-pattern-check Call Pattern (to be wired)

**What:** Replace manual "count errors by category, flag if >= 3" logic with `error-pattern-check` call
**Interface:** `bash .aether/aether-utils.sh error-pattern-check`
**Returns:** `{"ok":true,"result":[{"category":"...","count":N,"first_seen":"...","last_seen":"..."},...]}`

### Pattern 4: memory-compress Call Pattern (to be wired)

**What:** Replace manual array truncation logic with `memory-compress` call
**Interface:** `bash .aether/aether-utils.sh memory-compress [threshold]`
**Returns:** `{"ok":true,"result":{"compressed":true,"tokens":N}}`
**Default threshold:** 10000 tokens

### Anti-Patterns to Avoid
- **Partial replacement:** Do not leave inline formulas alongside utility calls. The entire inline formula block must be replaced, not supplemented.
- **Changing display format:** The output format seen by users must remain identical after wiring. Only the computation method changes.
- **Over-scoping:** Do not refactor display logic, step numbering, or other unrelated parts of the command files.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Pheromone decay calculation | Inline `strength * e^(-0.693 * elapsed / half_life)` | `aether-utils.sh pheromone-batch` | Centralized, tested, handles edge cases (null half_life, timestamp parsing) |
| Error counting by severity | Manual filter/count in prompt instructions | `aether-utils.sh error-summary` | Already returns grouped counts by category and severity |
| Error pattern detection | Manual "count by category, flag if >= 3" | `aether-utils.sh error-pattern-check` | Returns categories with 3+ occurrences, includes timestamps |
| Memory array trimming | Manual "if exceeds N, keep only N" | `aether-utils.sh memory-compress` | Handles both retention limits and token budgets atomically |

## Common Pitfalls

### Pitfall 1: Losing the Display Format Contract
**What goes wrong:** Replacing the computation changes the output format displayed to the user.
**Why it happens:** The inline formulas include formatting instructions interleaved with computation. When replacing with a utility call, the formatting instructions might get lost.
**How to avoid:** Keep ALL display formatting instructions. Only replace the computation section. The `pheromone-batch` result array contains `current_strength` on each signal object, which maps directly to the existing display format.
**Warning signs:** Step instructions that say "Format:" or output template blocks get deleted.

### Pitfall 2: Not Handling Utility Failure Gracefully
**What goes wrong:** If `aether-utils.sh` call fails, the command crashes with no output.
**Why it happens:** The inline formula can't fail (it's just prompt instructions), but a Bash call can fail.
**How to avoid:** Include fallback text: "If the command fails, treat as [reasonable default]." build.md already does this: "If the command fails, treat as 'no active pheromones.'"
**Warning signs:** No error handling instructions after the Bash command.

### Pitfall 3: Forgetting to Update the help Command in aether-utils.sh
**What goes wrong:** After removing 4 subcommands, the `help` output still lists them.
**Why it happens:** The help text is a hardcoded JSON string on line 38.
**How to avoid:** CLEAN-05 must update line 38 to remove `pheromone-combine`, `memory-token-count`, `memory-search`, `error-dedup` from the commands array.
**Warning signs:** `bash .aether/aether-utils.sh help` lists commands that no longer exist.

### Pitfall 4: Misunderstanding continue.md Scope for memory-compress
**What goes wrong:** Replacing the wrong truncation logic. continue.md has MULTIPLE truncation instructions -- some for events.json (keep 100), some for memory.json (keep 20/30).
**Why it happens:** `memory-compress` only handles memory.json. Events.json truncation is a separate concern.
**How to avoid:** Only replace the memory.json truncation logic (phase_learnings > 20, decisions > 30). Leave events.json truncation instructions as-is.
**Warning signs:** Events.json truncation logic disappears from the command.

### Pitfall 5: Misunderstanding build.md Scope for error-pattern-check
**What goes wrong:** Removing the entire error-handling section rather than just the pattern-check portion.
**Why it happens:** build.md Step 6 has a complex error workflow: error-add (already wired), pattern flagging (to be replaced with error-pattern-check), and error summary display (to be replaced with error-summary). These are interleaved.
**How to avoid:** Replace ONLY the "Count errors by category, flag if >= 3" section with an `error-pattern-check` call. Leave `error-add` calls intact.
**Warning signs:** The `error-add` Bash calls get deleted.

## Detailed Inline Code Locations

### CLEAN-01: Inline Decay Formulas (4 files)

**plan.md** -- Lines 29-36 (Step 3: Compute Active Pheromones)
```markdown
### Step 3: Compute Active Pheromones

For each signal in `pheromones.json`:

1. If `half_life_seconds` is null, persists at original strength
2. Otherwise: `current_strength = strength * e^(-0.693 * elapsed_seconds / half_life_seconds)`
3. Filter out signals where `current_strength < 0.05`
```
**Replace with:** pheromone-batch call pattern (copy from build.md lines 43-50)
**Keep:** The "Format:" block below (lines 37-42) stays, but change it to reference `result` array from pheromone-batch output instead of manual computation.

**pause-colony.md** -- Lines 19-24 (Step 2: Compute Pheromone Decay)
```markdown
### Step 2: Compute Pheromone Decay

For each signal in `pheromones.json`, compute current strength:
1. If `half_life_seconds` is null -> persists at original strength
2. Otherwise: `current_strength = strength * e^(-0.693 * elapsed_seconds / half_life_seconds)`
3. Note which signals are still active (strength >= 0.05)
```
**Replace with:** pheromone-batch call pattern, parse result for active/expired status.

**resume-colony.md** -- Lines 29-33 (Step 2: Compute Pheromone Decay)
```markdown
### Step 2: Compute Pheromone Decay

For each signal in `pheromones.json`, compute current strength:
1. If `half_life_seconds` is null -> persists at original strength
2. Otherwise: `current_strength = strength * e^(-0.693 * elapsed_seconds / half_life_seconds)`
3. Note which signals are still active (strength >= 0.05)
```
**Replace with:** pheromone-batch call pattern, parse result for active/expired status.

**colonize.md** -- Lines 28-33 (Step 2: Compute Active Pheromones)
```markdown
### Step 2: Compute Active Pheromones

For each signal in `pheromones.json`:
1. If `half_life_seconds` is null, persists at original strength
2. Otherwise: `current_strength = strength * e^(-0.693 * elapsed_seconds / half_life_seconds)`
3. Filter out signals where `current_strength < 0.05`
```
**Replace with:** pheromone-batch call pattern (copy from build.md lines 43-50).

### CLEAN-02: Manual Array Truncation in continue.md

**continue.md** -- Lines 108-113 (inside Step 4: Extract Phase Learnings)
```markdown
If the `phase_learnings` array exceeds 20 entries, remove the oldest entries to keep only 20.

Use the Write tool to write the updated memory.json.
```
**Replace with:** `bash .aether/aether-utils.sh memory-compress` call after the Write tool step.
**Note:** The `memory-compress` subcommand already enforces: phase_learnings <= 20, decisions <= 30, with aggressive halving if over token threshold.
**Keep:** The events.json truncation on line 169 ("If the `events` array exceeds 100 entries...") -- that is NOT handled by memory-compress.

### CLEAN-03: Manual Error Categorization in build.md

**build.md** -- Lines 274-288 (inside Step 6, "Check Pattern Flagging" section)
```markdown
**Check Pattern Flagging:** Count errors in the `errors` array by `category`. If any category has 3 or more errors and is not already in `flagged_patterns`, add:

{JSON block for flagged pattern}

If the category already exists in `flagged_patterns`, update its `count`, `last_seen`, and `description`.
```
**Replace with:** `bash .aether/aether-utils.sh error-pattern-check` call. Parse the returned array of categories with 3+ errors. Use the result to update `flagged_patterns` in errors.json.
**Keep:** The `error-add` calls above this section (lines 249-260) remain unchanged -- they are already wired.

### CLEAN-04: Manual Error Counting in continue.md and build.md

**continue.md** -- Lines 56-70 (inside Step 3: Phase Completion Summary)
```markdown
  Errors:
    <count> errors encountered
    (list severity counts: N critical, N high, N medium, N low)
...
Get error data from `errors.json` -- filter the `errors` array by `phase` field matching the current phase number. Count by severity level.
```
**Replace with:** `bash .aether/aether-utils.sh error-summary` call, then filter/display from its result.
**Note:** error-summary returns `by_severity` and `by_category` counts. The phase-specific filtering may still need to happen manually OR the display can use the global summary -- depends on how the output should look.

**build.md** -- The error counting is embedded in Step 6 display, but build.md doesn't currently have an explicit manual count step -- it delegates to error-add and pattern flagging. The error-summary call would supplement the display in Step 7 (lines 370-376) where watcher report shows issue counts.

### CLEAN-05: Remove 4 Subcommands from aether-utils.sh

**aether-utils.sh** -- Remove these case blocks:
| Subcommand | Lines | Confirmed Zero Consumers |
|------------|-------|--------------------------|
| `pheromone-combine` | 76-83 | No command or worker spec calls it |
| `memory-token-count` | 160-163 | No command or worker spec calls it |
| `memory-search` | 181-192 | No command or worker spec calls it (old .aether/utils/memory-search.sh is a different file, already deleted) |
| `error-dedup` | 222-237 | No command or worker spec calls it |

**Also update:** Line 38 help text -- remove these 4 from the commands array.

**Post-removal subcommand list (11 remaining):**
help, version, pheromone-decay, pheromone-effective, pheromone-batch, pheromone-cleanup, validate-state, memory-compress, error-add, error-pattern-check, error-summary

## Code Examples

### Example 1: Replacing plan.md Step 3 (CLEAN-01)

**Before (plan.md lines 29-42):**
```markdown
### Step 3: Compute Active Pheromones

For each signal in `pheromones.json`:

1. If `half_life_seconds` is null, persists at original strength
2. Otherwise: `current_strength = strength * e^(-0.693 * elapsed_seconds / half_life_seconds)`
3. Filter out signals where `current_strength < 0.05`

Format:

\`\`\`
ACTIVE PHEROMONES:
- {TYPE} (strength {current_strength:.2f}): "{content}"
\`\`\`
```

**After:**
```markdown
### Step 3: Compute Active Pheromones

Use the Bash tool to run:
\`\`\`
bash .aether/aether-utils.sh pheromone-batch
\`\`\`

This returns JSON: `{"ok":true,"result":[...signals with current_strength...]}`. Parse the `result` array. Filter out signals where `current_strength < 0.05`.

If the command fails, treat as "no active pheromones."

Format:

\`\`\`
ACTIVE PHEROMONES:
- {TYPE} (strength {current_strength:.2f}): "{content}"
\`\`\`
```

### Example 2: Replacing continue.md Memory Truncation (CLEAN-02)

**Before (continue.md lines 108-113):**
```markdown
If the `phase_learnings` array exceeds 20 entries, remove the oldest entries to keep only 20.

Use the Write tool to write the updated memory.json.
```

**After:**
```markdown
Use the Write tool to write the updated memory.json.

Then use the Bash tool to run:
\`\`\`
bash .aether/aether-utils.sh memory-compress
\`\`\`

This enforces retention limits (phase_learnings <= 20, decisions <= 30) and returns `{"ok":true,"result":{"compressed":true,"tokens":N}}`.
```

### Example 3: Replacing build.md Pattern Flagging (CLEAN-03)

**Before (build.md lines 274-288):**
```markdown
**Check Pattern Flagging:** Count errors in the `errors` array by `category`. If any category has 3 or more errors and is not already in `flagged_patterns`, add:
...
```

**After:**
```markdown
**Check Pattern Flagging:** Use the Bash tool to run:
\`\`\`
bash .aether/aether-utils.sh error-pattern-check
\`\`\`

This returns JSON: `{"ok":true,"result":[{"category":"...","count":N,"first_seen":"...","last_seen":"..."},...]}`

For each category in the result that is not already in `flagged_patterns`, add:
...
```

### Example 4: Removing a Subcommand (CLEAN-05)

**Remove entire case block (e.g., pheromone-combine lines 76-83):**
```bash
  pheromone-combine)
    [[ $# -ge 2 ]] || json_err "Usage: pheromone-combine <signal1_strength> <signal2_strength>"
    json_ok "$(jq -n --arg s1 "$1" --arg s2 "$2" '{
      net_effect: ((($s1|tonumber) - ($s2|tonumber)) | if . < 0 then 0 else . end | . * 1000 | round / 1000),
      dominant: (if ($s1|tonumber) >= ($s2|tonumber) then "signal1" else "signal2" end),
      ratio: (if ($s2|tonumber) == 0 then null else (($s1|tonumber) / ($s2|tonumber)) | . * 1000 | round / 1000 end)
    }')"
    ;;
```

**Update help text on line 38 from:**
```
"commands":["help","version","pheromone-decay","pheromone-effective","pheromone-batch","pheromone-cleanup","pheromone-combine","validate-state","memory-token-count","memory-compress","memory-search","error-add","error-pattern-check","error-summary","error-dedup"]
```
**To:**
```
"commands":["help","version","pheromone-decay","pheromone-effective","pheromone-batch","pheromone-cleanup","validate-state","memory-compress","error-add","error-pattern-check","error-summary"]
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Inline decay formula in each command prompt | Centralized `pheromone-batch` in aether-utils.sh | Phase 20 (utility modules) | Consistency, single source of truth |
| Manual JSON array truncation in prompts | `memory-compress` subcommand | Phase 20 | Atomic writes, threshold-based compression |
| Manual error counting in prompts | `error-summary` subcommand | Phase 20 | Structured JSON output, reusable |

**Deprecated/outdated:**
- pheromone-combine: Never got consumers; net-effect calculation not needed by any command
- memory-token-count: Subsumed by memory-compress (which reports tokens after compression)
- memory-search: Replaced by the older `.aether/utils/memory-search.sh` (now deleted); the aether-utils.sh version was never wired
- error-dedup: No command needs deduplication -- error-add already caps at 50 entries

## Open Questions

1. **error-summary phase filtering**
   - What we know: `error-summary` returns global totals. continue.md Step 3 needs per-phase error counts.
   - What's unclear: Whether to add phase filtering to `error-summary` or keep manual filtering for phase-specific counts alongside the utility call.
   - Recommendation: Use `error-summary` for the global display and keep the manual "filter by phase" instruction for the phase-specific count in continue.md Step 3. This minimizes scope and avoids modifying aether-utils.sh beyond removals.

2. **build.md error-summary integration scope**
   - What we know: build.md's Step 7 display (lines 370-376) shows watcher report issue counts, not error-summary data. The requirement says "build.md calls error-summary instead of manual error counting."
   - What's unclear: Where exactly in build.md the manual error counting happens -- the counting seems embedded in the display template rather than as explicit computation steps.
   - Recommendation: Add an `error-summary` call at the end of Step 6 (after all error-add calls) to get structured counts for the Step 7 display. This replaces the implicit "count issues from watcher report" with explicit structured data.

## Sources

### Primary (HIGH confidence)
- `.aether/aether-utils.sh` -- direct reading, all subcommand interfaces verified
- `.claude/commands/ant/build.md` -- reference implementation for pheromone-batch call pattern
- `.claude/commands/ant/status.md` -- reference implementation for pheromone-batch call pattern
- `.claude/commands/ant/plan.md` -- inline formula locations verified at lines 29-36
- `.claude/commands/ant/pause-colony.md` -- inline formula locations verified at lines 19-24
- `.claude/commands/ant/resume-colony.md` -- inline formula locations verified at lines 29-33
- `.claude/commands/ant/colonize.md` -- inline formula locations verified at lines 28-33
- `.claude/commands/ant/continue.md` -- truncation at lines 108-113, error counting at lines 56-70
- `.claude/commands/ant/build.md` -- pattern flagging at lines 274-288

### Secondary (MEDIUM confidence)
- `.planning/milestones/v4.0-MILESTONE-AUDIT.md` -- confirms 8 orphaned subcommands (line 139)
- `.planning/ROADMAP.md` -- confirms Phase 22 scope (line 31)

### Tertiary (LOW confidence)
None -- all findings verified from primary sources.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH -- no new tools needed, all existing infrastructure
- Architecture: HIGH -- reference implementations already exist in build.md and status.md
- Pitfalls: HIGH -- all locations verified by line number from source files
- Code examples: HIGH -- before/after patterns derived from existing working code

**Research date:** 2026-02-03
**Valid until:** 2026-03-03 (stable -- these are internal project files, not external dependencies)
