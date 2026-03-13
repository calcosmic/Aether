# Phase 8: Orchestrator Upgrade - Research

**Researched:** 2026-03-13
**Domain:** Convergence detection, diminishing returns, graceful interruption handling, JSON state validation and recovery in bash orchestrator scripts
**Confidence:** HIGH

## Summary

Phase 8 upgrades oracle.sh from a fixed-iteration loop with basic stop-file checking into an intelligent orchestrator that knows when research is done, detects when it is spinning its wheels, gracefully produces partial results on interruption, and recovers from malformed state. The core insight is that all the raw data for convergence detection already exists in the state files created by Phase 6 and the iteration mechanics built in Phase 7 -- Phase 8 computes derived metrics from that data and acts on them.

The current oracle.sh (319 lines) has: a `.stop` file check, basic jq validation warnings (no recovery), iteration counter increment, phase transitions based on confidence thresholds, and completion via `<oracle>COMPLETE</oracle>` from the AI. What it lacks: multi-signal convergence detection, diminishing returns detection, synthesis-on-interruption, signal handling (no `trap`), and JSON recovery when validation fails. The system exits with code 1 on max iterations with no useful output beyond whatever happens to be in synthesis.md.

The approach is to add three capabilities to oracle.sh: (1) a `compute_convergence` function that calculates gap resolution rate, novelty rate, and coverage completeness from plan.json history across iterations; (2) a `detect_diminishing_returns` function that tracks iteration-over-iteration changes and triggers strategy changes or synthesis after consecutive low-change iterations; (3) a `run_synthesis_pass` function that invokes the AI with a synthesis-specific prompt to produce a structured partial report from whatever state exists, triggered on stop signal, max iterations, or convergence. Additionally, validation is upgraded from warning-only to recovery-capable using the existing atomic-write backup/restore infrastructure.

**Primary recommendation:** Add convergence metrics as a new `convergence` object in state.json computed by oracle.sh after each iteration. Use a composite score of three structural signals. Trigger synthesis pass on any exit path (stop, max-iter, convergence). Use existing atomic-write utilities for JSON recovery.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
None explicitly locked -- all decisions delegated to Claude's discretion.

### Claude's Discretion
- **Convergence signals** -- Which combination of gap resolution rate, novelty rate, and coverage completeness to use; threshold values; how to weight multiple signals; whether convergence requires all signals or a weighted composite
- **Diminishing returns** -- How many low-change iterations trigger strategy change vs synthesis; what "strategy change" means in practice; how aggressive detection should be
- **Interruption handling** -- What the synthesis pass produces on stop signal or max-iterations; format and depth of the partial report; whether synthesis runs automatically or needs a flag
- **Error recovery** -- How to detect and recover from malformed JSON in state files; whether recovery is silent-fix or warn-and-continue; what validation runs after each iteration and what triggers recovery

### Deferred Ideas (OUT OF SCOPE)
None -- discussion stayed within phase scope.
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| LOOP-04 | Convergence detection uses structural metrics (gap resolution rate, novelty rate, coverage completeness) -- not self-assessed confidence alone | The `compute_convergence` function calculates three independent structural metrics from plan.json data: gap_resolution_rate (answered+partial vs total), novelty_rate (new key_findings added this iteration vs total), coverage_completeness (questions with iterations_touched > 0 vs total). These are computed by oracle.sh from actual state, not from AI self-assessment. |
| INTL-05 | Reflection loop detects diminishing returns and triggers strategy changes | The `detect_diminishing_returns` function tracks a rolling window of per-iteration changes (confidence delta, new findings count). When N consecutive iterations show minimal change, oracle.sh either forces a phase change (strategy change) or triggers early synthesis (if already in synthesize/verify phase). |
| OUTP-02 | On stop or max-iterations, oracle runs a synthesis pass producing useful partial results | The `run_synthesis_pass` function is called on every exit path (stop file, max iterations, convergence, SIGINT/SIGTERM). It invokes the AI CLI with a synthesis-specific prompt that reads current state files and produces a structured partial report written to synthesis.md and a final research-plan.md. |
</phase_requirements>

## Standard Stack

### Core
| Library/Tool | Version | Purpose | Why Standard |
|-------------|---------|---------|--------------|
| oracle.sh | Bash script | Main orchestrator being upgraded | Existing; all changes are additions to this file |
| jq | 1.6+ | All convergence metric computation, JSON validation, state manipulation | Project standard; already used throughout oracle.sh |
| aether-utils.sh | ~9,808 lines | validate-oracle-state subcommand, json_ok/json_err helpers | Already built for oracle validation in Phase 6 |
| atomic-write.sh | Utility | Backup/restore for JSON recovery | Already exists at .aether/utils/atomic-write.sh with create_backup, restore_backup |
| ava | ^6.0.0 | Unit tests for convergence functions | Project standard test runner |

### Supporting
| Tool | Version | Purpose | When to Use |
|------|---------|---------|-------------|
| bash trap builtin | N/A | Signal handling for SIGINT/SIGTERM | Added to oracle.sh for graceful interruption |
| date -u | N/A | ISO-8601 timestamps for convergence history | Already used in oracle.sh |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Composite convergence score in bash/jq | External convergence detector script | Extra file; the jq queries are simple enough to inline |
| Rolling window in state.json | Separate convergence-history.json | Extra file to manage; state.json already has the iteration data; adding a small convergence object is cleaner |
| trap SIGINT for signal handling | Continue using .stop file only | .stop file requires the user to run `/ant:oracle stop`; trap catches Ctrl+C directly in tmux sessions where users instinctively hit Ctrl+C |

## Architecture Patterns

### Recommended Project Structure

No new files created -- all changes modify existing files:

```
.aether/oracle/
  oracle.sh           # MODIFY: add convergence, diminishing returns, synthesis pass, trap, recovery
  oracle.md           # MINOR MODIFY: add synthesis-pass directive (new phase-like behavior)
  state.json          # SCHEMA EXTEND: add convergence object with iteration history

tests/
  unit/oracle-convergence.test.js    # NEW: ava tests for convergence metrics
  bash/test-oracle-convergence.sh    # NEW: bash integration tests for convergence and recovery
```

### Pattern 1: Multi-Signal Convergence Detection

**What:** A function that computes three structural metrics from plan.json and state.json, returning a composite convergence score.

**When to use:** After each iteration, before checking for completion.

**Design:**

The three signals measure different dimensions of research progress:

1. **Gap Resolution Rate** -- What fraction of questions are no longer "open"? Computed as: `(answered_count + partial_with_high_confidence_count) / total_questions`. This measures breadth of coverage.

2. **Novelty Rate** -- Are iterations still producing new findings? Computed by comparing the total key_findings count this iteration vs the previous iteration. If the delta is 0 or near-0 for consecutive iterations, novelty has dried up. This measures whether the research is still learning.

3. **Coverage Completeness** -- What fraction of questions have been touched at all? Computed as: `questions_with_nonempty_iterations_touched / total_questions`. This measures whether the research has attempted all areas.

**Composite scoring:** Use a weighted average with: gap_resolution 40%, novelty_rate 30%, coverage_completeness 30%. Convergence is declared when the composite score exceeds a threshold (recommend starting at 0.85) AND the novelty rate has been below a threshold for 2+ consecutive iterations (research has stopped producing new findings at a high coverage level).

**Why not require all signals:** A single holdout question at 40% confidence should not block convergence if 7/8 questions are answered at 90%+. The weighted composite handles this gracefully.

```bash
# Example: compute_convergence function
compute_convergence() {
  local plan_file="$1"
  local state_file="$2"

  local total answered partial_high coverage novelty_delta

  total=$(jq '[.questions[]] | length' "$plan_file")
  answered=$(jq '[.questions[] | select(.status == "answered")] | length' "$plan_file")
  partial_high=$(jq '[.questions[] | select(.status == "partial" and .confidence >= 70)] | length' "$plan_file")

  # Gap resolution: fraction of questions substantively addressed
  local gap_resolution
  if [ "$total" -eq 0 ]; then
    gap_resolution=100
  else
    gap_resolution=$(( (answered + partial_high) * 100 / total ))
  fi

  # Coverage: fraction of questions touched at all
  local touched
  touched=$(jq '[.questions[] | select((.iterations_touched // []) | length > 0)] | length' "$plan_file")
  local coverage=$(( touched * 100 / total ))

  # Novelty: compare total findings count to previous iteration's count
  local current_findings prev_findings
  current_findings=$(jq '[.questions[].key_findings | length] | add // 0' "$plan_file")
  prev_findings=$(jq '.convergence.prev_findings_count // 0' "$state_file")
  local novelty_delta=$(( current_findings - prev_findings ))

  # Output as JSON for oracle.sh to consume
  jq -n --argjson gap "$gap_resolution" --argjson cov "$coverage" \
        --argjson novelty "$novelty_delta" --argjson findings "$current_findings" \
    '{gap_resolution_pct: $gap, coverage_pct: $cov, novelty_delta: $novelty, total_findings: $findings}'
}
```

### Pattern 2: Diminishing Returns Detection with Strategy Change

**What:** Track per-iteration deltas and trigger action when consecutive iterations show minimal progress.

**When to use:** After convergence metrics are computed each iteration.

**Design:**

Track a rolling window of the last N iterations' novelty deltas in `state.json.convergence.history` (array of objects: `{iteration, novelty_delta, confidence_delta, phase}`). When 3 consecutive iterations have `novelty_delta <= 1` (at most 1 new finding), trigger action:

- **If in survey or investigate phase:** Force phase transition to synthesize. This is a "strategy change" -- stop looking for new info, start consolidating what exists.
- **If in synthesize or verify phase:** Trigger early synthesis pass and declare convergence. The research has plateaued at its current depth.

The threshold of 3 consecutive low-change iterations balances patience (giving research time to find things) against efficiency (not wasting iterations). This is deliberately conservative -- err toward doing more research rather than stopping early, per the CONTEXT.md guidance.

```bash
# Example: detect_diminishing_returns
detect_diminishing_returns() {
  local state_file="$1"

  # Read last 3 novelty deltas from convergence history
  local low_change_count
  low_change_count=$(jq '
    [.convergence.history[-3:][] | select(.novelty_delta <= 1)] | length
  ' "$state_file" 2>/dev/null || echo "0")

  local current_phase
  current_phase=$(jq -r '.phase' "$state_file")

  if [ "$low_change_count" -ge 3 ]; then
    case "$current_phase" in
      survey|investigate)
        echo "strategy_change"  # Force advance to synthesize
        ;;
      synthesize|verify)
        echo "synthesize_now"   # Trigger synthesis and stop
        ;;
    esac
  else
    echo "continue"
  fi
}
```

### Pattern 3: Synthesis Pass on Any Exit

**What:** A function that invokes the AI with a synthesis-specific prompt to produce a structured partial report.

**When to use:** On every exit path -- stop file, max iterations, convergence, SIGINT/SIGTERM.

**Design:**

The synthesis pass is a single additional AI invocation with a special prompt:

```
## SYNTHESIS PASS (Final)

This is the final iteration. Produce a structured research report from the current state.

Read these files:
- .aether/oracle/state.json
- .aether/oracle/plan.json
- .aether/oracle/synthesis.md
- .aether/oracle/gaps.md

Write a FINAL version of synthesis.md with this structure:
1. Executive Summary (2-3 paragraphs)
2. Findings by Question (organized by sub-question, with confidence levels)
3. Open Questions (remaining gaps, clearly labeled with confidence)
4. Methodology Notes (how many iterations, which phases completed)

Also update research-plan.md to reflect the final state.
Set state.json status to "complete" or "stopped" as appropriate.
```

The synthesis pass runs even if the research completed only 1 iteration. It reads whatever state exists and produces the best possible report. The function signature:

```bash
run_synthesis_pass() {
  local reason="$1"  # "converged" | "stopped" | "max_iterations" | "interrupted"
  local state_file="$2"
  local oracle_md="$3"

  # Update state.json status
  local new_status
  case "$reason" in
    converged) new_status="complete" ;;
    *) new_status="stopped" ;;
  esac

  jq --arg status "$new_status" --arg reason "$reason" \
    '.status = $status | .stop_reason = $reason' "$state_file" > "$state_file.tmp" && mv "$state_file.tmp" "$state_file"

  # Build synthesis prompt and invoke AI
  build_synthesis_prompt "$state_file" | $AI_CMD 2>&1 | tee /dev/stderr

  # Regenerate research-plan.md one final time
  generate_research_plan
}
```

### Pattern 4: Signal Handling with trap

**What:** Bash `trap` handler for SIGINT and SIGTERM that triggers synthesis before exit.

**When to use:** Set up at script start, before the main loop.

**Design:**

```bash
# Near top of oracle.sh, after variable declarations
INTERRUPTED=false

cleanup_and_synthesize() {
  if [ "$INTERRUPTED" = true ]; then
    return  # Prevent re-entrant calls
  fi
  INTERRUPTED=true
  echo ""
  echo "Oracle interrupted. Running synthesis pass..."
  run_synthesis_pass "interrupted" "$STATE_FILE" "$SCRIPT_DIR/oracle.md"
  echo "Partial results saved to synthesis.md and research-plan.md"
  exit 130  # Standard SIGINT exit code
}

trap cleanup_and_synthesize SIGINT SIGTERM
```

**Important:** The trap fires after the current child process (the AI CLI invocation) completes, because bash waits for the foreground process. This means:
- If the user hits Ctrl+C during an AI iteration, bash waits for the AI to finish that iteration (or the AI process catches SIGINT itself and exits)
- Then the trap fires, running the synthesis pass
- This is acceptable behavior -- we get the latest state from the just-completed iteration

The existing `.stop` file mechanism is preserved as a complementary approach. The stop file is checked between iterations (cooperative), while trap handles interrupts during iterations.

### Pattern 5: JSON Validation with Recovery

**What:** After each iteration, validate state.json and plan.json. If malformed, recover from backup.

**When to use:** After the AI writes state files, before oracle.sh reads them for convergence/transition logic.

**Design:**

The current oracle.sh has warning-only validation:
```bash
if ! jq -e . "$STATE_FILE" >/dev/null 2>&1; then
  echo "WARNING: state.json is invalid JSON after iteration $i"
fi
```

Upgrade to recovery-capable validation:

```bash
validate_and_recover() {
  local file="$1"
  local file_name=$(basename "$file")

  if jq -e . "$file" >/dev/null 2>&1; then
    return 0  # Valid
  fi

  echo "WARNING: $file_name is invalid JSON after iteration $i. Attempting recovery..."

  # Try to restore from the pre-iteration backup
  if [ -f "$file.pre-iteration" ]; then
    cp "$file.pre-iteration" "$file"
    echo "  Recovered $file_name from pre-iteration backup"
    return 0
  fi

  # Fall back to atomic-write backup system
  source "$AETHER_ROOT/.aether/utils/atomic-write.sh"
  if restore_backup "$file"; then
    echo "  Recovered $file_name from atomic-write backup"
    return 0
  fi

  echo "  FATAL: Cannot recover $file_name. Triggering synthesis with last known good state."
  return 1
}
```

**Pre-iteration backup pattern:** Before each AI invocation, copy state.json and plan.json to `.pre-iteration` files. After the iteration, if validation passes, these become the new "last known good" state. If validation fails, restore from them.

```bash
# Before AI invocation in the loop
cp "$STATE_FILE" "$STATE_FILE.pre-iteration"
cp "$PLAN_FILE" "$PLAN_FILE.pre-iteration"

# After AI invocation
if ! validate_and_recover "$STATE_FILE" || ! validate_and_recover "$PLAN_FILE"; then
  echo "Unrecoverable state corruption. Running synthesis with last valid state..."
  run_synthesis_pass "corruption" "$STATE_FILE" "$SCRIPT_DIR/oracle.md"
  exit 1
fi
```

### Pattern 6: State Schema Extension for Convergence History

**What:** Add a `convergence` object to state.json to track iteration-over-iteration metrics.

**When to use:** Updated by oracle.sh after each iteration.

```json
{
  "version": "1.0",
  "topic": "...",
  "convergence": {
    "prev_findings_count": 0,
    "prev_overall_confidence": 0,
    "history": [
      {
        "iteration": 1,
        "novelty_delta": 5,
        "confidence_delta": 15,
        "gap_resolution_pct": 25,
        "coverage_pct": 60,
        "phase": "survey"
      }
    ],
    "composite_score": 0,
    "converged": false
  }
}
```

The `convergence` field is optional in state.json -- oracle.sh creates it on first use (backward compatible with Phase 6/7 state files). The validate-oracle-state subcommand in aether-utils.sh should be updated to accept but not require this field.

### Anti-Patterns to Avoid

- **AI-computed convergence:** Do NOT ask the AI to assess whether research is converging. The AI has incentives to both over-report (to finish early) and under-report (to keep researching). oracle.sh computes convergence from structural data.
- **Complex convergence formulas:** The three signals are simple ratios computable with basic jq. Do not introduce statistical methods, moving averages, or ML-based convergence detection. The system operates on 5-50 iterations -- statistical sophistication is wasted.
- **Blocking synthesis on interruption:** The synthesis pass should have a timeout (e.g., 120 seconds). If the AI hangs during synthesis, the script must still exit cleanly with whatever state exists.
- **Silent recovery without logging:** When JSON recovery occurs, always log what happened and what was recovered. Silent fixes hide bugs in the oracle.md prompt that cause malformed output.
- **Convergence history unbounded growth:** Cap the history array at max_iterations entries. For a 50-iteration session, 50 small objects is fine. Do not let it grow beyond the session.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| JSON backup/restore | Custom backup in oracle.sh | atomic-write.sh (create_backup, restore_backup) | Already built with rotation, temp files, and atomic rename |
| State file validation | New validation in oracle.sh | validate-oracle-state in aether-utils.sh | Already validates schema, types, enum values |
| JSON validity check | Custom parser | `jq -e . "$file" >/dev/null 2>&1` | Standard jq pattern; returns exit code 1 on invalid JSON |
| ISO-8601 timestamps | Custom formatting | `date -u +"%Y-%m-%dT%H:%M:%SZ"` | Already used throughout oracle.sh |
| Signal handling | Custom process management | bash `trap` builtin | Standard, well-documented, handles SIGINT/SIGTERM |
| Research plan generation | New function | Existing generate_research_plan in oracle.sh | Already regenerates from state.json + plan.json |

**Key insight:** Phase 8 adds LOGIC on top of existing INFRASTRUCTURE. The state files, validation, backups, and plan generation already exist. Phase 8 computes new metrics from existing data and adds new control flow paths (convergence exit, synthesis pass, recovery).

## Common Pitfalls

### Pitfall 1: Convergence Thresholds Too Aggressive
**What goes wrong:** Research stops too early because the composite score hits the threshold while significant gaps remain.
**Why it happens:** Thresholds tuned for ideal scenarios; real research is messier.
**How to avoid:** Start with a high composite threshold (0.85) and require novelty rate to have been low for 2+ iterations. This means convergence requires BOTH high coverage AND evidence that research has stopped producing new findings. Err toward conservative (do more research).
**Warning signs:** Oracle converges in 3-4 iterations with synthesis.md containing thin findings.

### Pitfall 2: Synthesis Pass Invokes Broken AI Session
**What goes wrong:** The synthesis pass AI invocation fails because the state files are corrupted (the reason we're doing synthesis might BE corruption).
**Why it happens:** If the exit path is "corruption recovery," the files may not be in a valid state for the AI to read.
**How to avoid:** The synthesis pass should be defensive -- if state files are unreadable, produce a minimal report from whatever is in synthesis.md (which is markdown, harder to corrupt). The synthesis prompt should include: "If any state file is unreadable, skip it and work with what you have."
**Warning signs:** Synthesis pass itself fails, leaving no useful output.

### Pitfall 3: trap Causes Re-Entrant Synthesis
**What goes wrong:** User hits Ctrl+C during the synthesis pass (which was triggered by a previous Ctrl+C), causing recursive cleanup.
**Why it happens:** The trap fires again while the synthesis pass is running.
**How to avoid:** Use an `INTERRUPTED` flag. On first interrupt, set the flag and run synthesis. On second interrupt during synthesis, exit immediately without re-running synthesis. Example: `if [ "$INTERRUPTED" = true ]; then exit 130; fi`.
**Warning signs:** Duplicate synthesis output or infinite loop on interruption.

### Pitfall 4: Convergence History Breaks Existing State Schema
**What goes wrong:** Adding the `convergence` object to state.json causes validate-oracle-state to fail.
**Why it happens:** The existing validator checks for exact fields; new fields may trigger failures.
**How to avoid:** The jq validation in validate-oracle-state uses `has(f)` checks, not "no extra fields" checks. Adding new fields does NOT break existing validation. Verify by running existing tests after adding the convergence field.
**Warning signs:** validate-oracle-state returns false after convergence data is added to state.json.

### Pitfall 5: Diminishing Returns Detection Too Sensitive
**What goes wrong:** Oracle triggers strategy change after 3 iterations where the AI legitimately found only 1 new finding per iteration (e.g., deep investigation phase).
**Why it happens:** The threshold of "novelty_delta <= 1" may be too strict during investigate phase where deep dives on one question naturally produce fewer but more valuable findings.
**How to avoid:** Weight the threshold by phase. In investigate phase, a single new finding IS progress (use threshold of 0). In survey phase, finding fewer than 2 new things per iteration signals stalling. Use `novelty_delta <= 0` for investigate, `novelty_delta <= 1` for others.
**Warning signs:** Oracle skips investigation and jumps to synthesis too early on topics requiring deep dives.

### Pitfall 6: Max Iterations Exit Loses Work
**What goes wrong:** Oracle reaches max iterations and exits with code 1, and the user misses the results.
**Why it happens:** Current code: `echo "Max iterations reached"; exit 1`. No synthesis, no clear pointer to results.
**How to avoid:** Replace the max-iterations exit with a synthesis pass. The exit code should be 0 (synthesis succeeded) not 1 (error). Print clear instructions pointing to synthesis.md and research-plan.md.
**Warning signs:** Users report oracle "failed" when it actually completed useful research but hit the iteration cap.

## Code Examples

### Example 1: Main Loop Structure After Phase 8 Changes

```bash
# Source: Updated oracle.sh main loop structure
# Conceptual -- shows the flow, not exact implementation

# Setup
INTERRUPTED=false
trap cleanup_and_synthesize SIGINT SIGTERM

for i in $(seq 1 "$MAX_ITERATIONS"); do
  # Check stop file (cooperative stop)
  if [ -f "$STOP_FILE" ]; then
    rm -f "$STOP_FILE"
    echo "Oracle stopped by user at iteration $i"
    run_synthesis_pass "stopped" "$STATE_FILE" "$SCRIPT_DIR/oracle.md"
    exit 0
  fi

  # Pre-iteration backup for recovery
  cp "$STATE_FILE" "$STATE_FILE.pre-iteration"
  cp "$PLAN_FILE" "$PLAN_FILE.pre-iteration"

  # Run AI iteration (existing)
  OUTPUT=$(build_oracle_prompt "$STATE_FILE" "$SCRIPT_DIR/oracle.md" | $AI_CMD 2>&1 | tee /dev/stderr) || true

  # Validate and recover (Phase 8 upgrade)
  if ! validate_and_recover "$STATE_FILE" || ! validate_and_recover "$PLAN_FILE"; then
    run_synthesis_pass "corruption" "$STATE_FILE" "$SCRIPT_DIR/oracle.md"
    exit 1
  fi

  # Increment iteration (existing Phase 7)
  ITER_TS=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
  jq --arg ts "$ITER_TS" '.iteration += 1 | .last_updated = $ts' "$STATE_FILE" > "$STATE_FILE.tmp" && mv "$STATE_FILE.tmp" "$STATE_FILE"

  # Phase transition (existing Phase 7)
  NEW_PHASE=$(determine_phase "$STATE_FILE" "$PLAN_FILE")
  CURRENT_PHASE=$(jq -r '.phase' "$STATE_FILE")
  if [ "$NEW_PHASE" != "$CURRENT_PHASE" ]; then
    echo "  Phase transition: $CURRENT_PHASE -> $NEW_PHASE"
    jq --arg phase "$NEW_PHASE" '.phase = $phase' "$STATE_FILE" > "$STATE_FILE.tmp" && mv "$STATE_FILE.tmp" "$STATE_FILE"
  fi

  # Compute convergence metrics (Phase 8 new)
  update_convergence_metrics "$STATE_FILE" "$PLAN_FILE"

  # Check for diminishing returns (Phase 8 new)
  DR_RESULT=$(detect_diminishing_returns "$STATE_FILE")
  case "$DR_RESULT" in
    strategy_change)
      echo "  Diminishing returns detected. Advancing to synthesize phase."
      jq '.phase = "synthesize"' "$STATE_FILE" > "$STATE_FILE.tmp" && mv "$STATE_FILE.tmp" "$STATE_FILE"
      ;;
    synthesize_now)
      echo "  Research plateaued. Running synthesis."
      run_synthesis_pass "converged" "$STATE_FILE" "$SCRIPT_DIR/oracle.md"
      exit 0
      ;;
  esac

  # Check for convergence (Phase 8 new)
  if check_convergence "$STATE_FILE"; then
    echo "  Research converged."
    run_synthesis_pass "converged" "$STATE_FILE" "$SCRIPT_DIR/oracle.md"
    exit 0
  fi

  # Regenerate research-plan.md (existing)
  generate_research_plan

  # Check for AI completion signal (existing)
  if echo "$OUTPUT" | grep -q "<oracle>COMPLETE</oracle>"; then
    run_synthesis_pass "converged" "$STATE_FILE" "$SCRIPT_DIR/oracle.md"
    exit 0
  fi

  sleep 2
done

# Max iterations -- still run synthesis (Phase 8 change)
echo "Max iterations ($MAX_ITERATIONS) reached."
run_synthesis_pass "max_iterations" "$STATE_FILE" "$SCRIPT_DIR/oracle.md"
exit 0
```

### Example 2: Convergence Metric Computation with jq

```bash
# Source: jq queries for convergence metrics
# These compute structural metrics from plan.json without AI involvement

# Gap resolution: fraction of questions substantively resolved
jq '
  (.questions | length) as $total |
  ([.questions[] | select(.status == "answered")] | length) as $answered |
  ([.questions[] | select(.status == "partial" and .confidence >= 70)] | length) as $strong_partial |
  if $total == 0 then 100
  else (($answered + $strong_partial) * 100 / $total) | floor
  end
' plan.json

# Novelty: total findings count (compare iteration-over-iteration)
jq '[.questions[].key_findings | length] | add // 0' plan.json

# Coverage completeness: fraction of questions with any research
jq '
  (.questions | length) as $total |
  ([.questions[] | select((.iterations_touched // []) | length > 0)] | length) as $touched |
  if $total == 0 then 100
  else ($touched * 100 / $total) | floor
  end
' plan.json
```

### Example 3: Synthesis Prompt Construction

```bash
# Source: Synthesis-specific prompt for the final AI pass
build_synthesis_prompt() {
  local state_file="$1"
  local reason="$2"

  cat <<SYNTHESIS_DIRECTIVE
## SYNTHESIS PASS (Final Report)

This is the final pass. The oracle loop has ended ($reason).
Produce the best possible research report from the current state.

Read ALL of these files:
- .aether/oracle/state.json -- session metadata
- .aether/oracle/plan.json -- questions, findings, confidence
- .aether/oracle/synthesis.md -- accumulated findings
- .aether/oracle/gaps.md -- remaining unknowns

Then REWRITE synthesis.md as a structured final report:

### Required Sections:
1. **Executive Summary** -- 2-3 paragraphs summarizing what was found
2. **Findings by Question** -- organized by sub-question, with confidence %
3. **Open Questions** -- remaining gaps with explanation of what is unknown and why
4. **Confidence Assessment** -- overall confidence and per-question breakdown

Write the COMPLETE updated synthesis.md. Do NOT add new research.
Consolidate and organize what already exists.

Also update state.json: set status to "complete" if findings are substantive,
or "stopped" if research was interrupted early.

SYNTHESIS_DIRECTIVE

  # Append the base oracle.md for tool access and rules
  cat "$SCRIPT_DIR/oracle.md"
}
```

### Example 4: Pre-Iteration Backup and Validation Recovery

```bash
# Source: Validation with recovery using existing infrastructure

validate_and_recover() {
  local file="$1"
  local backup_file="${file}.pre-iteration"

  # Step 1: Check if file is valid JSON
  if jq -e . "$file" >/dev/null 2>&1; then
    return 0  # Valid
  fi

  echo "WARNING: $(basename "$file") is invalid JSON after iteration. Attempting recovery..."

  # Step 2: Try pre-iteration backup (always available within the loop)
  if [ -f "$backup_file" ] && jq -e . "$backup_file" >/dev/null 2>&1; then
    cp "$backup_file" "$file"
    echo "  Recovered from pre-iteration backup."
    return 0
  fi

  # Step 3: Try atomic-write backup system
  if source "$AETHER_ROOT/.aether/utils/atomic-write.sh" 2>/dev/null; then
    if restore_backup "$file" 2>/dev/null; then
      echo "  Recovered from atomic-write backup."
      return 0
    fi
  fi

  echo "  FATAL: Cannot recover $(basename "$file")."
  return 1
}
```

## State of the Art

| Old Approach (Phase 7) | Phase 8 Approach | Impact |
|------------------------|------------------|--------|
| Completion only via AI `<oracle>COMPLETE</oracle>` signal | Multi-signal structural convergence detection | Oracle stops based on measured progress, not AI self-assessment |
| No diminishing returns detection | Rolling window tracks per-iteration changes, triggers strategy change | Prevents wasting iterations when research plateaus |
| Exit with code 1 on max iterations, no synthesis | Synthesis pass on every exit path (stop, max-iter, convergence, interrupt) | Every oracle run produces a useful structured report |
| Warning-only JSON validation | Validation with pre-iteration backup recovery | Malformed state triggers recovery, not silent corruption |
| No signal handling (no trap) | trap SIGINT SIGTERM with synthesis-before-exit | Ctrl+C produces results instead of losing work |
| .stop file only for cooperative stop | .stop file + trap for both cooperative and interrupt | Covers both user-initiated stop and terminal interrupts |
| state.json has no convergence data | state.json includes convergence object with iteration history | oracle.sh makes data-driven decisions about when to stop |

**Deprecated/outdated after this phase:**
- `exit 1` on max iterations -- replaced by synthesis pass + `exit 0`
- Warning-only jq validation -- replaced by validate-and-recover
- Sole reliance on `<oracle>COMPLETE</oracle>` for completion -- convergence detection provides an independent completion signal

## Discretion Recommendations

Based on research into the codebase, convergence detection patterns, and bash signal handling:

| Area | Recommendation | Rationale |
|------|---------------|-----------|
| Convergence signals | Weighted composite of 3 metrics: gap resolution (40%), novelty rate (30%), coverage (30%) | Three independent dimensions cover breadth, depth, and productivity; weighted composite prevents one outlier from dominating |
| Convergence threshold | Composite >= 85% AND novelty rate low for 2+ iterations | High bar + productivity check prevents premature convergence |
| Diminishing returns trigger | 3 consecutive iterations with novelty_delta at or below phase-adjusted threshold | Conservative (not aggressive) -- per CONTEXT.md guidance |
| Phase-adjusted novelty threshold | investigate: 0 (any finding is progress), survey/synthesize/verify: 1 | Deep dives naturally produce fewer but more valuable findings |
| Strategy change action | Force advance to synthesize phase | Concrete, testable action; better than vague "broaden scope" |
| Synthesis output format | Rewrite synthesis.md with executive summary + per-question findings + open questions | Structured, useful for reading; extends existing synthesis.md format |
| Synthesis trigger | Automatic on every exit path (no flag needed) | User always gets useful output; no configuration to forget |
| Error recovery approach | Warn-and-recover (not silent) | Log the recovery so users know the AI produced bad JSON; helps debug prompt issues |
| Pre-iteration backup | Copy state.json and plan.json to .pre-iteration files before each AI call | Simple, reliable; complements existing atomic-write infrastructure |
| Convergence history storage | In state.json under `convergence` key (not a separate file) | Keeps all state in one place; backward compatible (field is optional) |

## Open Questions

1. **Empirical convergence threshold tuning**
   - What we know: 85% composite threshold and 3-iteration diminishing returns window are informed estimates
   - What's unclear: Whether these produce good results on real research topics
   - Recommendation: Ship with these values, add environment variable overrides (ORACLE_CONVERGENCE_THRESHOLD, ORACLE_DR_WINDOW) for easy tuning without code changes

2. **Synthesis pass duration/timeout**
   - What we know: The synthesis pass is a single AI invocation that should complete in 30-120 seconds
   - What's unclear: Whether to add a hard timeout, and what happens if synthesis itself is interrupted
   - Recommendation: Add a 180-second timeout to the synthesis AI invocation using `timeout 180 $AI_CMD ...`. If synthesis times out, log a warning but preserve whatever state files exist.

3. **Whether validate-oracle-state needs updating for the convergence field**
   - What we know: The existing jq validation uses `has(f)` checks for required fields and does not reject extra fields
   - What's unclear: Whether the validator should actively check the convergence object structure
   - Recommendation: Leave validate-oracle-state unchanged for now. The convergence field is internal to oracle.sh and does not need schema validation -- it is written by oracle.sh itself (trusted code), not by the AI.

## Sources

### Primary (HIGH confidence)
- `.aether/oracle/oracle.sh` (319 lines) -- Current orchestrator with all Phase 7 additions
- `.aether/oracle/oracle.md` (126 lines) -- Current phase-aware prompt
- `.aether/aether-utils.sh` lines 1203-1274 -- validate-oracle-state implementation
- `.aether/utils/atomic-write.sh` (227 lines) -- Backup/restore infrastructure
- `tests/unit/oracle-phase-transitions.test.js` -- Existing phase transition tests
- `tests/unit/oracle-state.test.js` -- Existing state validation tests
- `tests/bash/test-oracle-phase.sh` -- Existing bash integration tests
- `.planning/ROADMAP.md` -- Phase 8 success criteria
- `.planning/REQUIREMENTS.md` -- LOOP-04, INTL-05, OUTP-02 definitions
- `.planning/phases/07-iteration-prompt-engineering/07-RESEARCH.md` -- Phase 7 research (convergence threshold tuning noted as Phase 8 concern)

### Secondary (MEDIUM confidence)
- [Bash Reference Manual: Signals](https://www.gnu.org/software/bash/manual/html_node/Signals.html) -- Official docs for trap behavior
- [Baeldung: Handling Signals in Bash Script](https://www.baeldung.com/linux/bash-signal-handling) -- Practical trap patterns
- [Iterative review-fix loops remove LLM hallucinations (DEV Community)](https://dev.to/yannick555/iterative-review-fix-loops-remove-llm-hallucinations-and-there-is-a-formula-for-it-4ee8) -- Convergence patterns in AI loops: "Round 1 catches half the errors, Round 2 catches half of what remains"
- [Ralph (snarktank/ralph)](https://github.com/snarktank/ralph) -- RALF pattern origin; moved from AI-signaled completion to structural state checking (prd.json `passes: true`) after AI prematurely signaled COMPLETE
- [Ralph Playbook](https://claytonfarr.github.io/ralph-playbook/) -- Manual convergence detection; no automated diminishing returns; our approach is more sophisticated

### Tertiary (LOW confidence)
- Convergence threshold values (85% composite, 3-iteration window) -- Informed by research patterns but not empirically validated on Aether's oracle; flagged for tuning
- Phase-adjusted novelty thresholds (0 for investigate, 1 for others) -- Logical reasoning about what constitutes "progress" per phase, not validated with real sessions

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH -- no new dependencies; all tools already exist in the project
- Architecture (convergence metrics): HIGH -- uses simple jq queries on existing data structures; well-defined
- Architecture (diminishing returns): MEDIUM -- threshold values are educated defaults, need empirical validation
- Architecture (synthesis pass): HIGH -- straightforward AI invocation with a specific prompt
- Architecture (signal handling): HIGH -- standard bash trap pattern, well-documented
- Architecture (JSON recovery): HIGH -- leverages existing atomic-write infrastructure
- Pitfalls: HIGH -- identified from reading actual code and understanding failure modes

**Research date:** 2026-03-13
**Valid until:** 2026-04-13 (stable domain; bash patterns and jq usage don't change)
