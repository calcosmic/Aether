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

# Main loop
for i in $(seq 1 "$MAX_ITERATIONS"); do
  # Check for stop signal
  if [ -f "$STOP_FILE" ]; then
    rm -f "$STOP_FILE"
    echo ""
    echo "Oracle stopped by user at iteration $i"
    break
  fi

  echo ""
  echo "---------------------------------------------------------------"
  echo "  Iteration $i of $MAX_ITERATIONS"
  echo "---------------------------------------------------------------"

  # Run AI with phase-aware prompt (directive + oracle.md)
  OUTPUT=$(build_oracle_prompt "$STATE_FILE" "$SCRIPT_DIR/oracle.md" | $AI_CMD 2>&1 | tee /dev/stderr) || true

  # Basic jq validation after iteration (safety check -- Phase 8 adds recovery)
  if ! jq -e . "$STATE_FILE" >/dev/null 2>&1; then
    echo "WARNING: state.json is invalid JSON after iteration $i"
  fi
  if ! jq -e . "$PLAN_FILE" >/dev/null 2>&1; then
    echo "WARNING: plan.json is invalid JSON after iteration $i"
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

  # Regenerate research-plan.md from current state
  generate_research_plan

  # Check for completion signal
  if echo "$OUTPUT" | grep -q "<oracle>COMPLETE</oracle>"; then
    echo ""
    echo "==============================================================="
    echo "  ORACLE RESEARCH COMPLETE!"
    echo "==============================================================="
    echo "Completed at iteration $i"
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
echo "Max iterations ($MAX_ITERATIONS) reached without completion."
echo "Check $RESEARCH_PLAN_FILE for current research status."
exit 1
