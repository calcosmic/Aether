#!/bin/bash
# Oracle Ant - Deep research loop using RALF pattern
# Usage: ./oracle.sh [max_iterations_override]
# Based on: https://github.com/snarktank/ralph
#
# Configuration is read from state.json (written by /ant:oracle wizard).
# Command-line arg overrides max_iterations if provided.

set -e

# Unset CLAUDECODE to allow spawning Claude CLI from within Claude Code
unset CLAUDECODE 2>/dev/null || true

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
AETHER_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

# Files
STATE_FILE="$SCRIPT_DIR/state.json"
PLAN_FILE="$SCRIPT_DIR/plan.json"
GAPS_FILE="$SCRIPT_DIR/gaps.md"
SYNTHESIS_FILE="$SCRIPT_DIR/synthesis.md"
RESEARCH_PLAN_FILE="$SCRIPT_DIR/research-plan.md"
STOP_FILE="$SCRIPT_DIR/.stop"
ARCHIVE_DIR="$SCRIPT_DIR/archive"
DISCOVERIES_DIR="$SCRIPT_DIR/discoveries"

# Generate research-plan.md from state.json and plan.json
generate_research_plan() {
  local state_file="$STATE_FILE"
  local plan_file="$PLAN_FILE"
  local output_file="$RESEARCH_PLAN_FILE"

  # Bail if source files don't exist
  [ -f "$state_file" ] || return 0
  [ -f "$plan_file" ] || return 0

  local topic iteration max_iter confidence status
  topic=$(jq -r '.topic' "$state_file")
  iteration=$(jq -r '.iteration' "$state_file")
  max_iter=$(jq -r '.max_iterations' "$state_file")
  confidence=$(jq -r '.overall_confidence' "$state_file")
  status=$(jq -r '.status' "$state_file")

  {
    echo "# Research Plan"
    echo ""
    echo "**Topic:** $topic"
    echo "**Status:** $status | **Iteration:** $iteration of $max_iter"
    echo "**Overall Confidence:** ${confidence}%"
    echo ""
    echo "## Questions"
    echo "| # | Question | Status | Confidence |"
    echo "|---|----------|--------|------------|"
    jq -r '.questions[] | "| \(.id) | \(.text) | \(.status) | \(.confidence)% |"' "$plan_file"
    echo ""
    echo "## Next Steps"
    local next
    next=$(jq -r '[.questions[] | select(.status != "answered")] | sort_by(.confidence) | first | .text // "All questions answered"' "$plan_file")
    echo "Next investigation: $next"
    echo ""
    echo "---"
    echo "*Generated from plan.json -- do not edit directly*"
  } > "$output_file"
}

# Determine research phase based on structural metrics in state.json and plan.json
# Phases: survey -> investigate -> synthesize -> verify
determine_phase() {
  local state_file="$1"
  local plan_file="$2"

  # Bail to default if files missing
  [ -f "$state_file" ] || { echo "survey"; return 0; }
  [ -f "$plan_file" ] || { echo "survey"; return 0; }

  local total_questions touched_count avg_confidence below_50_count

  total_questions=$(jq '[.questions[]] | length' "$plan_file" 2>/dev/null || echo "0")
  if [ "$total_questions" -eq 0 ]; then
    echo "survey"
    return 0
  fi

  # Count questions with non-empty iterations_touched arrays
  touched_count=$(jq '[.questions[] | select((.iterations_touched // []) | length > 0)] | length' "$plan_file" 2>/dev/null || echo "0")

  # Average confidence across all questions
  avg_confidence=$(jq '[.questions[].confidence] | if length > 0 then (add / length) else 0 end | floor' "$plan_file" 2>/dev/null || echo "0")

  # Count questions below 50% confidence that are not answered
  below_50_count=$(jq '[.questions[] | select(.status != "answered" and .confidence < 50)] | length' "$plan_file" 2>/dev/null || echo "0")

  # verify: avg confidence >= 80%
  if [ "$avg_confidence" -ge 80 ]; then
    echo "verify"
    return 0
  fi

  # synthesize: avg confidence >= 60% OR fewer than 2 questions below 50%
  if [ "$avg_confidence" -ge 60 ] || [ "$below_50_count" -lt 2 ]; then
    echo "synthesize"
    return 0
  fi

  # investigate: all questions touched OR avg confidence >= 25%
  if [ "$touched_count" -ge "$total_questions" ] || [ "$avg_confidence" -ge 25 ]; then
    echo "investigate"
    return 0
  fi

  # Default: survey
  echo "survey"
}

# Build the complete prompt by prepending a phase-specific directive to oracle.md
build_oracle_prompt() {
  local state_file="$1"
  local oracle_md="$2"

  local current_phase
  current_phase=$(jq -r '.phase // "survey"' "$state_file" 2>/dev/null || echo "survey")

  # Emit phase-specific directive
  case "$current_phase" in
    survey)
      cat <<'DIRECTIVE'
## Current Phase: SURVEY

Cast a wide net -- get initial findings for every open question. Target untouched
questions first (those with empty iterations_touched arrays). Aim for 20-40%
confidence per question. List all discovered unknowns in gaps.md.

Do NOT go deep on any single question yet. Breadth over depth in this phase.
Your goal is to ensure every question has at least some initial findings before
the research moves to the investigation phase.

---

DIRECTIVE
      ;;
    investigate)
      cat <<'DIRECTIVE'
## Current Phase: INVESTIGATE

Target the lowest-confidence question and go DEEP. You MUST reference existing
findings in synthesis.md and ADD NEW information, not restate what is already there.
Aim to push confidence above 70% for your target question.

Update gaps.md with specific remaining unknowns. If you find contradictions with
existing findings, document them explicitly. One thoroughly investigated question
per iteration is better than shallow passes on many.

---

DIRECTIVE
      ;;
    synthesize)
      cat <<'DIRECTIVE'
## Current Phase: SYNTHESIZE

Read ALL findings in synthesis.md before doing anything. Identify connections,
patterns, and contradictions ACROSS questions. Consolidate redundant findings.
Resolve contradictions with evidence. Push overall confidence toward the target.

Your job is NOT to find new information -- it is to make sense of what has already
been found. Cross-reference answers between questions. Strengthen weak claims
with evidence from other questions. Remove speculation that lacks support.

---

DIRECTIVE
      ;;
    verify)
      cat <<'DIRECTIVE'
## Current Phase: VERIFY

Focus on claims in gaps.md contradictions section. Cross-reference key findings
with additional sources. Confirm or correct confidence scores. Mark well-supported
questions as answered with 90%+ confidence.

Final gaps.md should contain only genuinely unresolvable unknowns. If a contradiction
cannot be resolved, document both positions with evidence quality assessment.
This is the final quality pass before research completion.

---

DIRECTIVE
      ;;
    *)
      echo "## Current Phase: $current_phase"
      echo ""
      echo "---"
      echo ""
      ;;
  esac

  # Emit the base oracle.md prompt
  cat "$oracle_md"
}

# Compute convergence metrics from plan.json structural data
# Returns JSON object with gap_resolution_pct, coverage_pct, novelty_delta, total_findings
compute_convergence() {
  local plan_file="$1"
  local state_file="$2"

  local total answered partial_high

  total=$(jq '[.questions[]] | length' "$plan_file" 2>/dev/null || echo "0")
  answered=$(jq '[.questions[] | select(.status == "answered")] | length' "$plan_file" 2>/dev/null || echo "0")
  partial_high=$(jq '[.questions[] | select(.status == "partial" and .confidence >= 70)] | length' "$plan_file" 2>/dev/null || echo "0")

  # Gap resolution: fraction of questions substantively addressed
  local gap_resolution
  if [ "$total" -eq 0 ]; then
    gap_resolution=100
  else
    gap_resolution=$(( (answered + partial_high) * 100 / total ))
  fi

  # Coverage: fraction of questions with non-empty iterations_touched
  local touched coverage
  touched=$(jq '[.questions[] | select((.iterations_touched // []) | length > 0)] | length' "$plan_file" 2>/dev/null || echo "0")
  if [ "$total" -eq 0 ]; then
    coverage=100
  else
    coverage=$(( touched * 100 / total ))
  fi

  # Novelty: compare total findings count to previous iteration
  local current_findings prev_findings novelty_delta
  current_findings=$(jq '[.questions[].key_findings | length] | add // 0' "$plan_file" 2>/dev/null || echo "0")
  prev_findings=$(jq '.convergence.prev_findings_count // 0' "$state_file" 2>/dev/null || echo "0")
  novelty_delta=$(( current_findings - prev_findings ))

  jq -n --argjson gap "$gap_resolution" --argjson cov "$coverage" \
        --argjson novelty "$novelty_delta" --argjson findings "$current_findings" \
    '{gap_resolution_pct: $gap, coverage_pct: $cov, novelty_delta: $novelty, total_findings: $findings}'
}

# Update convergence metrics in state.json after each iteration
update_convergence_metrics() {
  local state_file="$1"
  local plan_file="$2"

  local metrics
  metrics=$(compute_convergence "$plan_file" "$state_file")

  local gap_pct coverage_pct novelty_delta total_findings
  gap_pct=$(echo "$metrics" | jq '.gap_resolution_pct')
  coverage_pct=$(echo "$metrics" | jq '.coverage_pct')
  novelty_delta=$(echo "$metrics" | jq '.novelty_delta')
  total_findings=$(echo "$metrics" | jq '.total_findings')

  local current_confidence prev_confidence confidence_delta current_iteration current_phase
  current_confidence=$(jq '.overall_confidence // 0' "$state_file" 2>/dev/null || echo "0")
  prev_confidence=$(jq '.convergence.prev_overall_confidence // 0' "$state_file" 2>/dev/null || echo "0")
  confidence_delta=$(( current_confidence - prev_confidence ))
  current_iteration=$(jq '.iteration // 0' "$state_file" 2>/dev/null || echo "0")
  current_phase=$(jq -r '.phase // "survey"' "$state_file" 2>/dev/null || echo "survey")

  # Compute composite score:
  # gap_resolution * 0.4 + coverage * 0.3 + (novelty_delta <= 1 ? 100 : 0) * 0.3
  # Using integer arithmetic scaled by 100
  local novelty_component composite_score converged
  if [ "$novelty_delta" -le 1 ]; then
    novelty_component=100
  else
    novelty_component=0
  fi
  composite_score=$(( gap_pct * 40 / 100 + coverage_pct * 30 / 100 + novelty_component * 30 / 100 ))

  local conv_threshold
  conv_threshold=${ORACLE_CONVERGENCE_THRESHOLD:-85}
  if [ "$composite_score" -ge "$conv_threshold" ]; then
    converged="true"
  else
    converged="false"
  fi

  # Update state.json with convergence data
  jq --argjson prev_findings "$total_findings" \
     --argjson prev_confidence "$current_confidence" \
     --argjson iteration "$current_iteration" \
     --argjson novelty "$novelty_delta" \
     --argjson conf_delta "$confidence_delta" \
     --argjson gap "$gap_pct" \
     --argjson cov "$coverage_pct" \
     --arg phase "$current_phase" \
     --argjson composite "$composite_score" \
     --argjson converged "$converged" \
     '
     .convergence = (.convergence // {}) |
     .convergence.prev_findings_count = $prev_findings |
     .convergence.prev_overall_confidence = $prev_confidence |
     .convergence.history = ((.convergence.history // []) + [{
       iteration: $iteration,
       novelty_delta: $novelty,
       confidence_delta: $conf_delta,
       gap_resolution_pct: $gap,
       coverage_pct: $cov,
       phase: $phase
     }]) |
     .convergence.composite_score = $composite |
     .convergence.converged = $converged
     ' "$state_file" > "$state_file.tmp" && mv "$state_file.tmp" "$state_file"
}

# Check if research has converged
# Returns 0 (true) if composite_score >= threshold AND last 2 history entries have low novelty
check_convergence() {
  local state_file="$1"

  local conv_threshold
  conv_threshold=${ORACLE_CONVERGENCE_THRESHOLD:-85}

  local composite_score
  composite_score=$(jq '.convergence.composite_score // 0' "$state_file" 2>/dev/null || echo "0")

  if [ "$composite_score" -lt "$conv_threshold" ]; then
    return 1
  fi

  # Check that at least 2 history entries exist and last 2 have novelty_delta <= 1
  local history_len
  history_len=$(jq '(.convergence.history // []) | length' "$state_file" 2>/dev/null || echo "0")
  if [ "$history_len" -lt 2 ]; then
    return 1
  fi

  local low_novelty_count
  low_novelty_count=$(jq '[(.convergence.history // [])[-2:][] | select(.novelty_delta <= 1)] | length' "$state_file" 2>/dev/null || echo "0")

  if [ "$low_novelty_count" -ge 2 ]; then
    return 0
  fi

  return 1
}

# Detect diminishing returns from convergence history
# Outputs: "strategy_change", "synthesize_now", or "continue"
detect_diminishing_returns() {
  local state_file="$1"

  local dr_window
  dr_window=${ORACLE_DR_WINDOW:-3}

  local history_len
  history_len=$(jq '(.convergence.history // []) | length' "$state_file" 2>/dev/null || echo "0")

  if [ "$history_len" -lt "$dr_window" ]; then
    echo "continue"
    return 0
  fi

  local current_phase
  current_phase=$(jq -r '.phase // "survey"' "$state_file" 2>/dev/null || echo "survey")

  # Phase-adjusted novelty threshold
  local novelty_threshold
  case "$current_phase" in
    investigate) novelty_threshold=0 ;;
    *) novelty_threshold=1 ;;
  esac

  # Count entries in the last dr_window with novelty at or below threshold
  local low_change_count
  low_change_count=$(jq --argjson window "$dr_window" --argjson threshold "$novelty_threshold" \
    '[(.convergence.history // [])[-$window:][] | select(.novelty_delta <= $threshold)] | length' \
    "$state_file" 2>/dev/null || echo "0")

  if [ "$low_change_count" -ge "$dr_window" ]; then
    case "$current_phase" in
      survey|investigate)
        echo "strategy_change"
        ;;
      synthesize|verify)
        echo "synthesize_now"
        ;;
      *)
        echo "continue"
        ;;
    esac
  else
    echo "continue"
  fi
}

# Validate a JSON file and recover from backup if invalid
validate_and_recover() {
  local file="$1"

  if jq -e . "$file" >/dev/null 2>&1; then
    return 0
  fi

  echo "WARNING: $(basename "$file") is invalid JSON. Attempting recovery..." >&2

  # Try pre-iteration backup
  if [ -f "${file}.pre-iteration" ] && jq -e . "${file}.pre-iteration" >/dev/null 2>&1; then
    cp "${file}.pre-iteration" "$file"
    echo "  Recovered $(basename "$file") from pre-iteration backup." >&2
    return 0
  fi

  # Fall back to atomic-write backup system
  if [ -f "$AETHER_ROOT/.aether/utils/atomic-write.sh" ]; then
    source "$AETHER_ROOT/.aether/utils/atomic-write.sh" 2>/dev/null || true
    if type restore_backup >/dev/null 2>&1 && restore_backup "$file" 2>/dev/null; then
      echo "  Recovered $(basename "$file") from atomic-write backup." >&2
      return 0
    fi
  fi

  echo "  FATAL: Cannot recover $(basename "$file")." >&2
  return 1
}

# Build the synthesis-specific prompt for the final AI pass
build_synthesis_prompt() {
  local reason="$1"

  cat <<SYNTHESIS_DIRECTIVE
## SYNTHESIS PASS (Final Report)

This is the final pass. The oracle loop has ended (reason: $reason).
Produce the best possible research report from the current state.

Read ALL of these files:
- .aether/oracle/state.json -- session metadata
- .aether/oracle/plan.json -- questions, findings, confidence
- .aether/oracle/synthesis.md -- accumulated findings
- .aether/oracle/gaps.md -- remaining unknowns

If any state file is unreadable, skip it and work with what you have.

Then REWRITE synthesis.md as a structured final report:

### Required Sections:
1. **Executive Summary** -- 2-3 paragraphs summarizing what was found
2. **Findings by Question** -- organized by sub-question, with confidence %
3. **Open Questions** -- remaining gaps with explanation of what is unknown and why
4. **Methodology Notes** -- how many iterations, which phases completed

Also update state.json: set status to "complete" if reason is "converged",
or "stopped" otherwise.

SYNTHESIS_DIRECTIVE

  # Append the base oracle.md for tool access and rules
  cat "$SCRIPT_DIR/oracle.md"
}

# Run the synthesis pass: update state, invoke AI, regenerate research plan
run_synthesis_pass() {
  local reason="$1"

  # Update state.json status and stop_reason before AI call
  local new_status
  case "$reason" in
    converged) new_status="complete" ;;
    *) new_status="stopped" ;;
  esac

  if [ -f "$STATE_FILE" ] && jq -e . "$STATE_FILE" >/dev/null 2>&1; then
    jq --arg status "$new_status" --arg reason "$reason" \
      '.status = $status | .stop_reason = $reason' "$STATE_FILE" > "$STATE_FILE.tmp" && mv "$STATE_FILE.tmp" "$STATE_FILE"
  fi

  echo ""
  echo "==============================================================="
  echo "  SYNTHESIS PASS ($reason)"
  echo "==============================================================="

  # Invoke AI with synthesis prompt
  if command -v timeout >/dev/null 2>&1; then
    build_synthesis_prompt "$reason" | timeout 180 $AI_CMD 2>&1 | tee /dev/stderr || true
  else
    build_synthesis_prompt "$reason" | $AI_CMD 2>&1 | tee /dev/stderr || true
  fi

  # Regenerate research-plan.md one final time
  generate_research_plan

  echo ""
  echo "Results saved to:"
  echo "  $SYNTHESIS_FILE"
  echo "  $RESEARCH_PLAN_FILE"
}

# Trap handler: synthesize before exit on SIGINT/SIGTERM
cleanup_and_synthesize() {
  if [ "$INTERRUPTED" = true ]; then
    exit 130
  fi
  INTERRUPTED=true
  echo ""
  echo "Oracle interrupted. Running synthesis pass..."
  run_synthesis_pass "interrupted"
  exit 130
}

# Check state.json exists (wizard must create it before launching oracle.sh)
if [ ! -f "$STATE_FILE" ]; then
  echo "Error: No state.json found. Run /ant:oracle to configure research first."
  exit 1
fi

# Read config from state.json (wizard writes these)
CURRENT_TOPIC=$(jq -r '.topic // empty' "$STATE_FILE" 2>/dev/null || echo "")
TARGET_CONFIDENCE=$(jq -r '.target_confidence // 95' "$STATE_FILE" 2>/dev/null || echo "95")
JSON_MAX_ITER=$(jq -r '.max_iterations // 50' "$STATE_FILE" 2>/dev/null || echo "50")

# Command-line arg overrides state.json
MAX_ITERATIONS=${1:-$JSON_MAX_ITER}

# Detect AI CLI (claude or opencode)
if command -v claude &>/dev/null; then
  AI_CMD="claude --dangerously-skip-permissions --print"
elif command -v opencode &>/dev/null; then
  AI_CMD="opencode --dangerously-skip-permissions --print"
else
  echo "Error: Neither 'claude' nor 'opencode' CLI found on PATH."
  exit 1
fi

# Archive previous run if topic changed
if [ -f "$STATE_FILE" ]; then
  LAST_TOPIC=$(jq -r '.topic // empty' "$STATE_FILE" 2>/dev/null || echo "")
  # If the wizard passed a new topic via environment, compare
  if [ -n "${ORACLE_NEW_TOPIC:-}" ] && [ -n "$LAST_TOPIC" ] && [ "$ORACLE_NEW_TOPIC" != "$LAST_TOPIC" ]; then
    ARCHIVE_FOLDER="$ARCHIVE_DIR/$(date +%Y-%m-%d-%H%M%S)"

    echo "Archiving previous research: $LAST_TOPIC"
    mkdir -p "$ARCHIVE_FOLDER"
    for f in state.json plan.json gaps.md synthesis.md research-plan.md; do
      [ -f "$SCRIPT_DIR/$f" ] && cp "$SCRIPT_DIR/$f" "$ARCHIVE_FOLDER/"
    done
    echo "   Archived to: $ARCHIVE_FOLDER"
    # Do NOT create empty files -- the wizard handles initial file creation
  fi
fi

# Initialize discoveries directory
mkdir -p "$DISCOVERIES_DIR"

echo ""
echo "==============================================================="
echo "  ORACLE ANT - Deep Research Loop"
echo "==============================================================="
echo "Topic:       $CURRENT_TOPIC"
echo "Iterations:  $MAX_ITERATIONS"
echo "Confidence:  $TARGET_CONFIDENCE%"
echo "CLI:         $AI_CMD"
echo ""

# Signal handling setup
INTERRUPTED=false
trap cleanup_and_synthesize SIGINT SIGTERM

# Main loop
for i in $(seq 1 "$MAX_ITERATIONS"); do
  # Check for stop signal (cooperative stop)
  if [ -f "$STOP_FILE" ]; then
    rm -f "$STOP_FILE"
    echo ""
    echo "Oracle stopped by user at iteration $i"
    run_synthesis_pass "stopped"
    exit 0
  fi

  echo ""
  echo "---------------------------------------------------------------"
  echo "  Iteration $i of $MAX_ITERATIONS"
  echo "---------------------------------------------------------------"

  # Pre-iteration backup for recovery
  cp "$STATE_FILE" "$STATE_FILE.pre-iteration"
  cp "$PLAN_FILE" "$PLAN_FILE.pre-iteration"

  # Run AI with phase-aware prompt (directive + oracle.md)
  OUTPUT=$(build_oracle_prompt "$STATE_FILE" "$SCRIPT_DIR/oracle.md" | $AI_CMD 2>&1 | tee /dev/stderr) || true

  # Validate and recover from malformed JSON
  if ! validate_and_recover "$STATE_FILE" || ! validate_and_recover "$PLAN_FILE"; then
    run_synthesis_pass "corruption"
    exit 1
  fi

  # Increment iteration counter
  ITER_TS=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
  jq --arg ts "$ITER_TS" '.iteration += 1 | .last_updated = $ts' "$STATE_FILE" > "$STATE_FILE.tmp" && mv "$STATE_FILE.tmp" "$STATE_FILE"

  # Check for phase transition
  NEW_PHASE=$(determine_phase "$STATE_FILE" "$PLAN_FILE")
  CURRENT_PHASE=$(jq -r '.phase' "$STATE_FILE")
  if [ "$NEW_PHASE" != "$CURRENT_PHASE" ]; then
    echo "  Phase transition: $CURRENT_PHASE -> $NEW_PHASE"
    jq --arg phase "$NEW_PHASE" '.phase = $phase' "$STATE_FILE" > "$STATE_FILE.tmp" && mv "$STATE_FILE.tmp" "$STATE_FILE"
  fi

  # Update convergence metrics
  update_convergence_metrics "$STATE_FILE" "$PLAN_FILE"

  # Check for diminishing returns
  DR_RESULT=$(detect_diminishing_returns "$STATE_FILE")
  case "$DR_RESULT" in
    strategy_change)
      echo "  Diminishing returns detected. Advancing to synthesize phase."
      jq '.phase = "synthesize"' "$STATE_FILE" > "$STATE_FILE.tmp" && mv "$STATE_FILE.tmp" "$STATE_FILE"
      ;;
    synthesize_now)
      echo "  Research plateaued. Running synthesis."
      run_synthesis_pass "converged"
      exit 0
      ;;
  esac

  # Check for convergence
  if check_convergence "$STATE_FILE"; then
    echo "  Research converged."
    run_synthesis_pass "converged"
    exit 0
  fi

  # Regenerate research-plan.md from current state
  generate_research_plan

  # Check for AI completion signal
  if echo "$OUTPUT" | grep -q "<oracle>COMPLETE</oracle>"; then
    echo ""
    echo "==============================================================="
    echo "  ORACLE RESEARCH COMPLETE!"
    echo "==============================================================="
    echo "Completed at iteration $i"
    run_synthesis_pass "converged"
    exit 0
  fi

  echo ""
  echo "Iteration $i complete. Continuing..."
  sleep 2
done

echo ""
echo "==============================================================="
echo "  ORACLE REACHED MAX ITERATIONS"
echo "==============================================================="
echo "Max iterations ($MAX_ITERATIONS) reached."
run_synthesis_pass "max_iterations"
exit 0
