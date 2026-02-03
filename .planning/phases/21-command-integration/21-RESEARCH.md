# Phase 21: Command Integration - Research

**Researched:** 2026-02-03
**Domain:** Claude command prompt files (.claude/commands/ant/*.md) + worker spec files (.aether/workers/*.md) integration with aether-utils.sh
**Confidence:** HIGH

## Summary

This phase replaces inline LLM computation in command prompt files with deterministic shell calls to `aether-utils.sh`. The research involved reading all 4 command files (status.md, build.md, continue.md, init.md), all 6 worker spec files, and the complete aether-utils.sh script to map exactly where inline computation occurs and what utility subcommands replace them.

The integration pattern is consistent: where prompts currently instruct the LLM to perform math or JSON construction inline, they should instead instruct the LLM to use the Bash tool to run `bash .aether/aether-utils.sh <subcommand> [args]`, then parse the JSON result. The wrapper script outputs `{"ok":true,"result":...}` on success and `{"ok":false,"error":"..."}` to stderr on failure.

**Primary recommendation:** Replace inline computation blocks surgically -- keep surrounding prompt structure identical, only swap the computation method from "Claude does math" to "Claude runs Bash tool and reads JSON output."

## Standard Stack

This phase involves no new libraries. It modifies existing markdown command prompts and worker spec markdown files.

### Core

| Component | Path | Purpose | Modified By |
|-----------|------|---------|-------------|
| aether-utils.sh | `.aether/aether-utils.sh` | Deterministic ops wrapper | NOT modified (Phase 20 deliverable) |
| status.md | `.claude/commands/ant/status.md` | Colony status display command | INT-01 |
| build.md | `.claude/commands/ant/build.md` | Phase build command | INT-02 |
| continue.md | `.claude/commands/ant/continue.md` | Phase advance command | INT-03 |
| init.md | `.claude/commands/ant/init.md` | Colony initialization command | INT-05 |
| 6 worker specs | `.aether/workers/*.md` | Ant caste behavior specs | INT-04 |

### Subcommands Used

| Subcommand | Invocation | Output Shape | Used By |
|------------|------------|-------------|---------|
| pheromone-batch | `bash .aether/aether-utils.sh pheromone-batch` | `{"ok":true,"result":[{...signal + current_strength}]}` | status.md, build.md |
| pheromone-cleanup | `bash .aether/aether-utils.sh pheromone-cleanup` | `{"ok":true,"result":{"removed":N,"remaining":N}}` | status.md, continue.md |
| pheromone-effective | `bash .aether/aether-utils.sh pheromone-effective <sensitivity> <strength>` | `{"ok":true,"result":{"effective_signal":N}}` | all 6 worker specs |
| error-add | `bash .aether/aether-utils.sh error-add <category> <severity> <description>` | `{"ok":true,"result":"err_<id>"}` | build.md |
| validate-state | `bash .aether/aether-utils.sh validate-state all` | `{"ok":true,"result":{"pass":bool,"files":[...]}}` | init.md |

## Architecture Patterns

### Pattern 1: Bash Tool Invocation in Command Prompts

**What:** Instruct the LLM to use the Bash tool to run aether-utils.sh, then parse JSON output for display/decisions.
**When to use:** Everywhere a command prompt currently asks the LLM to compute pheromone decay, construct error JSON, or validate state.

**Before (inline computation in status.md Step 2):**
```markdown
### Step 2: Compute Pheromone Decay

For each signal in `pheromones.json`, compute current strength:

1. If `half_life_seconds` is null -> signal persists at original strength
2. Otherwise: `current_strength = strength * e^(-0.693 * elapsed_seconds / half_life_seconds)`
3. Mark signals below 0.05 strength as expired
```

**After (delegated to aether-utils.sh):**
```markdown
### Step 2: Compute Pheromone Decay

Use the Bash tool to run:
```
bash .aether/aether-utils.sh pheromone-batch
```

This returns JSON with each signal's `current_strength` pre-computed. Parse the `.result` array. Signals with `current_strength < 0.05` are expired.
```

### Pattern 2: Error Handling for Shell Calls

**What:** All aether-utils.sh calls return `{"ok":true,...}` on success or write `{"ok":false,"error":"..."}` to stderr with exit code 1 on failure.
**When to use:** Every Bash tool invocation of aether-utils.sh should check the `ok` field.

**Template for command prompts:**
```markdown
Use the Bash tool to run: `bash .aether/aether-utils.sh <subcommand> [args]`

If the command fails (non-zero exit or `ok: false`), output an error message and continue with fallback behavior.
```

### Pattern 3: Worker Spec Pheromone-Effective Integration

**What:** Replace the inline `effective_signal = sensitivity * signal_strength` math in worker specs with a Bash tool call to `pheromone-effective`.
**When to use:** In the "Pheromone Math" section of each worker spec.

**Key consideration:** Worker ants are spawned as Task tool subagents. They DO have access to the Bash tool. The instruction should tell them to call aether-utils.sh for each signal they need to evaluate.

### Recommended Edit Structure

For each file, the changes are surgical -- replace specific computation blocks while keeping surrounding prompt structure intact:

```
status.md:
  - Step 2 (lines 39-46): Replace inline decay formula with pheromone-batch call
  - Step 2.5 (lines 47-51): Replace inline cleanup with pheromone-cleanup call

build.md:
  - Step 3 (lines 43-55): Replace inline decay formula with pheromone-batch call
  - Step 6 error logging (lines 243-258): Replace manual JSON construction with error-add calls

continue.md:
  - Step 5 (lines 171-178): Replace inline cleanup with pheromone-cleanup call

init.md:
  - After Step 6, before Step 7: Add new step calling validate-state all

worker specs (all 6):
  - "Pheromone Math" section: Replace inline formula with pheromone-effective call
```

### Anti-Patterns to Avoid

- **Rewriting entire command files:** Changes should be surgical replacements of specific computation blocks. The surrounding prompt text, display format, and flow remain identical.
- **Removing the explanation of what the math does:** Keep a brief description of what pheromone decay/effective signal means conceptually. Just change HOW it's computed from "you calculate" to "run this command."
- **Making the aether-utils call optional/fallback:** The whole point is deterministic results. The shell call IS the primary path, not a fallback.

## Don't Hand-Roll

Problems that look simple but have existing solutions:

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Pheromone decay computation | LLM doing e^(-0.693 * t / half_life) inline | `pheromone-batch` subcommand | LLM math is unreliable; shell uses jq with exact formula |
| Effective signal calculation | LLM multiplying sensitivity * strength | `pheromone-effective` subcommand | Eliminates rounding errors and inconsistent thresholds |
| Error JSON construction | LLM manually building error JSON objects | `error-add` subcommand | Handles ID generation, timestamp, 50-entry cap, atomic write |
| Expired pheromone removal | LLM filtering and rewriting pheromones.json | `pheromone-cleanup` subcommand | Atomic write, correct math, threshold consistency |
| State file validation | LLM eyeballing JSON structure | `validate-state all` subcommand | Type-checks every field systematically |

**Key insight:** The LLM is unreliable at math (exponential decay), unreliable at generating unique IDs, and unreliable at maintaining JSON structure constraints (array caps, required fields). These are exactly the operations aether-utils.sh was built to handle.

## Common Pitfalls

### Pitfall 1: Forgetting pheromone-batch replaces BOTH decay AND filtering

**What goes wrong:** status.md Step 2 currently computes decay AND marks expired signals. The `pheromone-batch` command computes decay but does NOT remove expired signals -- it just shows `current_strength`. The cleanup step (Step 2.5) still needs `pheromone-cleanup` separately.
**Why it happens:** Assuming one command replaces the entire block.
**How to avoid:** Step 2 calls `pheromone-batch` (read-only, gets current strengths). Step 2.5 calls `pheromone-cleanup` (mutates file, removes expired). Two separate calls.
**Warning signs:** If the plan only has one Bash call for Steps 2+2.5 combined.

### Pitfall 2: build.md error-add only handles SOME errors

**What goes wrong:** build.md Step 6 logs errors from two sources: (a) ant's failure report, and (b) watcher issues with HIGH/CRITICAL severity. The `error-add` subcommand handles individual error additions. Pattern checking is a separate step.
**Why it happens:** Trying to replace the entire Step 6 error block with a single call.
**How to avoid:** Each error logged should be a separate `error-add` call. Pattern checking can use `error-pattern-check` afterward, but the current prompt manually checks patterns -- the plan should decide whether to also delegate that.
**Warning signs:** If plan replaces ALL of Step 6 error handling with a single call.

### Pitfall 3: Worker specs are read by spawned subagents, not the Queen

**What goes wrong:** Worker specs are embedded in Task tool prompts when ants are spawned. The "Pheromone Math" section is read and followed by subagent LLMs, not the orchestrating command. The Bash tool instruction must work in the subagent context.
**Why it happens:** Not considering the execution context of worker spec instructions.
**How to avoid:** The pheromone-effective instruction must explicitly say "Use the Bash tool to run..." since the subagent has Bash tool access. The path `.aether/aether-utils.sh` must be relative (which it is -- same as all other file references in the specs).
**Warning signs:** If plan changes worker specs without considering subagent context.

### Pitfall 4: init.md validate-state timing

**What goes wrong:** Calling `validate-state all` BEFORE all state files are created will fail (files don't exist yet).
**Why it happens:** Inserting the validation call too early in the init flow.
**How to avoid:** The validate-state call must come AFTER Step 5 (Emit INIT Pheromone) -- all 5 state files exist by then: COLONY_STATE.json, errors.json, memory.json, events.json, pheromones.json. Insert as a new Step 6.5 or renumber.
**Warning signs:** If validation step is placed before all files are written.

### Pitfall 5: error-add doesn't handle pattern flagging

**What goes wrong:** The `error-add` subcommand adds an error and caps at 50 entries. But build.md Step 6 also checks for pattern flagging (categories with 3+ errors). `error-add` does NOT do pattern flagging -- that's a separate concern.
**Why it happens:** Assuming error-add handles the full error pipeline.
**How to avoid:** After calling `error-add` for each error, the pattern flagging logic either stays inline (Claude checks manually) or uses `error-pattern-check` subcommand. The `error-pattern-check` output can inform what to add to `flagged_patterns`, but the actual flagged_patterns write is still the LLM's job since it requires constructing specific JSON and writing to errors.json.
**Warning signs:** If plan removes ALL error handling from build.md, including pattern flagging.

### Pitfall 6: Pheromone bar rendering still needs LLM

**What goes wrong:** Trying to have aether-utils.sh render the ASCII decay bars.
**Why it happens:** Over-delegating -- the bars are a display concern, not a computation concern.
**How to avoid:** `pheromone-batch` provides `current_strength` values. The LLM still renders the bars (`=` characters) from those values. The delegation boundary is: shell does math, LLM does display formatting.
**Warning signs:** If plan tries to add bar rendering to aether-utils.sh.

## Detailed File Analysis

### status.md (263 lines)

**Current inline computation locations:**
- **Step 2 (lines 39-46):** Decay formula `current_strength = strength * e^(-0.693 * elapsed_seconds / half_life_seconds)`. Replace with `pheromone-batch`.
- **Step 2.5 (lines 47-51):** Remove expired signals and rewrite pheromones.json. Replace with `pheromone-cleanup`.

**What stays the same:**
- Step 1 (Read State) -- unchanged
- Step 3 (Display Status) -- unchanged, but now uses data from pheromone-batch output instead of inline calculation
- All display formatting (bars, headers, sections) -- unchanged

**Exact replacement for Step 2:**
```markdown
### Step 2: Compute Pheromone Decay

Use the Bash tool to run:
```
bash .aether/aether-utils.sh pheromone-batch
```

This returns JSON: `{"ok":true,"result":[...signals with current_strength...]}`. Parse the `result` array. Each signal object includes a `current_strength` field with the decayed value. Signals with `current_strength < 0.05` are effectively expired.

If the command fails (file not found, invalid JSON), treat as "no active pheromones."
```

**Exact replacement for Step 2.5:**
```markdown
### Step 2.5: Clean Expired Pheromones

If any signals from Step 2 had `current_strength < 0.05`, use the Bash tool to run:
```
bash .aether/aether-utils.sh pheromone-cleanup
```

This removes expired signals from `pheromones.json` and returns `{"ok":true,"result":{"removed":N,"remaining":N}}`.

If no signals are expired, skip this step.
```

### build.md (386 lines)

**Current inline computation locations:**
- **Step 3 (lines 43-55):** Same decay formula as status.md. Replace with `pheromone-batch`.
- **Step 6 error logging (lines 243-295):** Manual JSON construction for error records. Partially replace with `error-add`.

**What stays the same:**
- Steps 1, 2, 4, 4.5, 5, 5.5 -- completely unchanged
- Step 6 state updates (PROJECT_PLAN.json, COLONY_STATE.json) -- unchanged
- Step 6 event writing -- unchanged
- Step 6 pattern flagging -- stays inline OR uses error-pattern-check
- Step 6 spawn outcomes -- unchanged
- Step 7 display -- unchanged

**Exact replacement for Step 3:**
Same pattern as status.md Step 2. Use `pheromone-batch`, render bars from returned `current_strength` values.

**Replacement for Step 6 error logging:**
Where the prompt currently says "For each failure, append an error record..." and "For each issue in the watcher report with severity HIGH or CRITICAL, append an error record...", replace with:
```markdown
For each failure reported by the ant, use the Bash tool to run:
```
bash .aether/aether-utils.sh error-add "<category>" "<severity>" "<description>"
```

For each HIGH or CRITICAL issue from the watcher report, use the Bash tool to run:
```
bash .aether/aether-utils.sh error-add "verification" "<severity>" "<description>"
```

Each call returns `{"ok":true,"result":"<error_id>"}`. Use the returned error IDs when logging error_logged events.
```

**Note:** The `error-add` subcommand handles: ID generation, timestamp, appending to errors array, 50-entry cap, and atomic write. It does NOT handle: root_cause, phase, or task_id fields (those are set to null). The phase/task_id gap is a tradeoff -- the utility keeps things simple at the cost of not recording phase context in the error. This may be acceptable since errors can be correlated with phases via timestamps.

### continue.md (267 lines)

**Current inline computation locations:**
- **Step 5 (lines 171-178):** Decay formula + removal of expired signals + rewrite. Replace with `pheromone-cleanup`.

**What stays the same:**
- Steps 1-4, 4.5, 6-8 -- completely unchanged

**Exact replacement for Step 5:**
```markdown
### Step 5: Clean Expired Pheromones

Use the Bash tool to run:
```
bash .aether/aether-utils.sh pheromone-cleanup
```

This removes signals with current_strength below 0.05 from `pheromones.json` and returns `{"ok":true,"result":{"removed":N,"remaining":N}}`. The cleanup result (removed count) can be mentioned in the display output.
```

### init.md (185 lines)

**Current state:** No inline computation -- but no validation either. Requirement INT-05 adds a NEW step.

**What changes:**
- Add a new step between Step 6 (Write Init Event) and Step 7 (Display Result). This could be "Step 6.5" or the steps could be renumbered.

**New step to add:**
```markdown
### Step 6.5: Validate State Files

Use the Bash tool to run:
```
bash .aether/aether-utils.sh validate-state all
```

This validates all state files (COLONY_STATE.json, pheromones.json, errors.json, memory.json, events.json) and returns `{"ok":true,"result":{"pass":true|false,"files":[...]}}`.

If `pass` is false, output a warning identifying which file(s) failed validation. This catches initialization bugs immediately.
```

**Step 7 display update:** Add validation result to the step progress display:
```
  ✓ Step 6.5: Validate State Files
```

### Worker Specs (6 files, all identical structure for Pheromone Math)

**Files affected:**
1. `.aether/workers/architect-ant.md` (lines 18-44)
2. `.aether/workers/builder-ant.md` (lines 18-44)
3. `.aether/workers/colonizer-ant.md` (lines 18-44)
4. `.aether/workers/route-setter-ant.md` (lines 18-44)
5. `.aether/workers/scout-ant.md` (lines 18-44)
6. `.aether/workers/watcher-ant.md` (lines 18-44)

**Current inline computation (identical in all 6):**
```markdown
## Pheromone Math

Calculate effective signal strength to determine action priority:

```
effective_signal = sensitivity * signal_strength
```

Where signal_strength is the pheromone's current decay value (0.0 to 1.0).
```

**Replacement pattern (same for all 6):**
```markdown
## Pheromone Math

To compute effective signal strength for each active pheromone, use the Bash tool:

```
bash .aether/aether-utils.sh pheromone-effective <sensitivity> <strength>
```

This returns `{"ok":true,"result":{"effective_signal":N}}`. Use the `effective_signal` value to determine action priority.
```

**What stays the same in each worker spec:**
- Sensitivity table -- unchanged
- Threshold interpretation (>0.5 PRIORITIZE, 0.3-0.5 NOTE, <0.3 IGNORE) -- unchanged
- Worked examples -- update to show the Bash call but keep the same example values and interpretation
- Combination Effects section -- unchanged
- Everything else -- unchanged

**Worked example update pattern:**
```markdown
**Worked example:**
```
Example: FOCUS signal at strength 0.8, REDIRECT signal at strength 0.4

Run: bash .aether/aether-utils.sh pheromone-effective 0.9 0.8
Result: {"ok":true,"result":{"effective_signal":0.72}}  -> PRIORITIZE

Run: bash .aether/aether-utils.sh pheromone-effective 0.9 0.4
Result: {"ok":true,"result":{"effective_signal":0.36}}  -> NOTE

Action: Strongly prioritize focused area. Note the redirect but don't
fully avoid -- the signal is fading.
```
```

## Code Examples

### Verified aether-utils.sh invocations

**pheromone-batch (no arguments, reads pheromones.json):**
```bash
bash .aether/aether-utils.sh pheromone-batch
# Returns: {"ok":true,"result":[{"id":"init_1706...","type":"INIT","content":"...","strength":1.0,"half_life_seconds":null,"created_at":"...","current_strength":1.0}, ...]}
```

**pheromone-cleanup (no arguments, mutates pheromones.json):**
```bash
bash .aether/aether-utils.sh pheromone-cleanup
# Returns: {"ok":true,"result":{"removed":2,"remaining":3}}
```

**pheromone-effective (two arguments):**
```bash
bash .aether/aether-utils.sh pheromone-effective 0.9 0.8
# Returns: {"ok":true,"result":{"effective_signal":0.72}}
```

**error-add (three arguments):**
```bash
bash .aether/aether-utils.sh error-add "verification" "high" "Missing input validation on auth endpoint"
# Returns: {"ok":true,"result":"err_1706893200_a3f2"}
```

**validate-state all (one argument):**
```bash
bash .aether/aether-utils.sh validate-state all
# Returns: {"ok":true,"result":{"pass":true,"files":[{"file":"COLONY_STATE.json","checks":["pass","pass","pass","pass","pass"],"pass":true}, ...]}}
```

### Error case output

```bash
bash .aether/aether-utils.sh pheromone-batch
# When pheromones.json missing:
# stderr: {"ok":false,"error":"pheromones.json not found"}
# exit code: 1
```

## State of the Art

| Old Approach (current) | New Approach (this phase) | Impact |
|------------------------|--------------------------|--------|
| LLM computes e^(-0.693 * t / h) inline | `pheromone-batch` via Bash tool | Deterministic, consistent results |
| LLM constructs error JSON manually | `error-add` via Bash tool | Correct IDs, timestamps, cap enforcement |
| LLM filters and rewrites pheromones.json | `pheromone-cleanup` via Bash tool | Atomic write, correct thresholds |
| LLM multiplies sensitivity * strength | `pheromone-effective` via Bash tool | Exact arithmetic, no rounding drift |
| No post-init validation | `validate-state all` via Bash tool | Catches initialization bugs |

## Open Questions

1. **error-add missing phase/task_id fields**
   - What we know: `error-add` sets `root_cause`, `phase`, and `task_id` to null. build.md currently instructs setting these fields.
   - What's unclear: Is losing phase/task_id context acceptable?
   - Recommendation: Accept the tradeoff. The error timestamp + sequential phase execution makes correlation straightforward. Alternatively, errors could be enriched after creation via a separate Read-modify-Write step, but that defeats the simplicity goal.

2. **build.md pattern flagging after error-add**
   - What we know: `error-add` adds one error at a time. Pattern flagging (checking if 3+ errors share a category) happens after all errors are logged. `error-pattern-check` returns categories with 3+ errors but doesn't write to `flagged_patterns`.
   - What's unclear: Should pattern flagging remain inline (LLM reads errors.json and writes flagged_patterns) or should a new utility be added?
   - Recommendation: Keep pattern flagging inline for now. `error-pattern-check` provides the data, but the LLM still needs to construct and write flagged_pattern entries. This is a display/decision concern, not a computation concern.

3. **Multiple pheromone-effective calls per worker**
   - What we know: A worker may need to evaluate 3-4 signals. That's 3-4 separate Bash tool calls.
   - What's unclear: Is the overhead of multiple Bash calls acceptable vs. one batch call?
   - Recommendation: Accept the overhead. The `pheromone-effective` calls are fast (<100ms each). A batch variant could be added later if needed, but the current subcommand interface is simple and clear.

## Sources

### Primary (HIGH confidence)
- `.aether/aether-utils.sh` -- read directly, all 242 lines, all subcommand signatures verified
- `.claude/commands/ant/status.md` -- read directly, 263 lines, inline computation at lines 39-51 identified
- `.claude/commands/ant/build.md` -- read directly, 386 lines, inline computation at lines 43-55 and 243-295 identified
- `.claude/commands/ant/continue.md` -- read directly, 267 lines, inline computation at lines 171-178 identified
- `.claude/commands/ant/init.md` -- read directly, 185 lines, no current inline computation, insertion point after Step 6 identified
- `.aether/workers/architect-ant.md` -- read directly, Pheromone Math at lines 18-44
- `.aether/workers/builder-ant.md` -- read directly, Pheromone Math at lines 18-44
- `.aether/workers/colonizer-ant.md` -- read directly, Pheromone Math at lines 18-44
- `.aether/workers/route-setter-ant.md` -- read directly, Pheromone Math at lines 18-44
- `.aether/workers/scout-ant.md` -- read directly, Pheromone Math at lines 18-44
- `.aether/workers/watcher-ant.md` -- read directly, Pheromone Math at lines 18-44
- `.aether/utils/atomic-write.sh` -- read directly, confirms atomic write pattern used by cleanup/error-add
- `.aether/utils/file-lock.sh` -- read directly, confirms file locking infrastructure
- `bash .aether/aether-utils.sh help` -- executed, confirmed subcommand list
- `bash .aether/aether-utils.sh version` -- executed, confirmed v0.1.0

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH -- all files read directly, no external dependencies
- Architecture: HIGH -- invocation patterns verified by running aether-utils.sh
- Pitfalls: HIGH -- all edge cases identified from direct file analysis

**Research date:** 2026-02-03
**Valid until:** 2026-03-03 (stable -- these are project-internal files, not external dependencies)
