#!/bin/bash
# Oracle Ant - In-Session Stop Hook
# Intercepts session exit when Oracle research is active.
# Re-feeds phase-aware prompts to continue the research loop.
#
# Modeled on the Ralph Loop Stop hook pattern:
# - Checks .aether/oracle/.loop-active for loop state
# - Session isolation via session_id
# - Outputs {"decision":"block","reason":"..."} to re-feed prompt
# - Outputs nothing (exit 0) to allow normal stop
#
# Completion criteria (any triggers synthesis pass):
# - Max iterations reached
# - overall_confidence >= target_confidence
# - <oracle>COMPLETE</oracle> in last assistant message
# - Convergence detected (composite score >= threshold + low novelty)

set -euo pipefail

# ---------------------------------------------------------------------------
# Configuration
# ---------------------------------------------------------------------------
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
AETHER_ROOT="$(cd "$SCRIPT_DIR/../../.." && pwd)"
ORACLE_DIR="$AETHER_ROOT/.aether/oracle"
MARKER_FILE="$ORACLE_DIR/.loop-active"
STATE_FILE="$ORACLE_DIR/state.json"
PLAN_FILE="$ORACLE_DIR/plan.json"
STOP_FILE="$ORACLE_DIR/.stop"
ORACLE_MD="$SCRIPT_DIR/oracle.md"

# Convergence thresholds (matching oracle.sh defaults)
CONV_THRESHOLD=${ORACLE_CONVERGENCE_THRESHOLD:-85}
DR_WINDOW=${ORACLE_DR_WINDOW:-3}

# ---------------------------------------------------------------------------
# Fast exit: no active loop
# ---------------------------------------------------------------------------
if [[ ! -f "$MARKER_FILE" ]]; then
  exit 0
fi

# ---------------------------------------------------------------------------
# Read hook input from stdin
# ---------------------------------------------------------------------------
HOOK_INPUT=$(cat)

INPUT_SESSION_ID=$(echo "$HOOK_INPUT" | jq -r '.session_id // ""')
TRANSCRIPT_PATH=$(echo "$HOOK_INPUT" | jq -r '.transcript_path // ""')

# ---------------------------------------------------------------------------
# Parse marker file frontmatter
# ---------------------------------------------------------------------------
FRONTMATTER=$(sed -n '/^---$/,/^---$/{ /^---$/d; p; }' "$MARKER_FILE")

ITERATION=$(echo "$FRONTMATTER" | grep '^iteration:' | sed 's/iteration: *//')
MAX_ITERATIONS=$(echo "$FRONTMATTER" | grep '^max_iterations:' | sed 's/max_iterations: *//')
MARKER_SESSION_ID=$(echo "$FRONTMATTER" | grep '^session_id:' | sed 's/session_id: *//' || true)
CURRENT_PHASE=$(echo "$FRONTMATTER" | grep '^phase:' | sed 's/phase: *//' || echo "survey")
TARGET_CONFIDENCE=$(echo "$FRONTMATTER" | grep '^target_confidence:' | sed 's/target_confidence: *//' || echo "95")
SYNTHESIS_DONE=$(echo "$FRONTMATTER" | grep '^synthesis_done:' | sed 's/synthesis_done: *//' || echo "false")

# ---------------------------------------------------------------------------
# Validate numeric fields
# ---------------------------------------------------------------------------
if [[ ! "$ITERATION" =~ ^[0-9]+$ ]]; then
  echo "Oracle hook: Invalid iteration in marker file. Removing." >&2
  rm -f "$MARKER_FILE"
  exit 0
fi

if [[ ! "$MAX_ITERATIONS" =~ ^[0-9]+$ ]]; then
  echo "Oracle hook: Invalid max_iterations in marker file. Removing." >&2
  rm -f "$MARKER_FILE"
  exit 0
fi

# ---------------------------------------------------------------------------
# Session isolation
# ---------------------------------------------------------------------------
if [[ -n "$MARKER_SESSION_ID" ]] && [[ "$MARKER_SESSION_ID" != "$INPUT_SESSION_ID" ]]; then
  # Different session started this loop -- don't interfere
  exit 0
fi

# ---------------------------------------------------------------------------
# Check for user-requested stop
# ---------------------------------------------------------------------------
if [[ -f "$STOP_FILE" ]]; then
  rm -f "$STOP_FILE"
  rm -f "$MARKER_FILE"
  echo "Oracle hook: Stop signal received. Ending research." >&2
  exit 0
fi

# ---------------------------------------------------------------------------
# Validate state files exist
# ---------------------------------------------------------------------------
if [[ ! -f "$STATE_FILE" ]]; then
  echo "Oracle hook: state.json missing. Ending research." >&2
  rm -f "$MARKER_FILE"
  exit 0
fi

# Validate state.json is valid JSON
if ! jq -e . "$STATE_FILE" >/dev/null 2>&1; then
  echo "Oracle hook: state.json is corrupt. Ending research." >&2
  rm -f "$MARKER_FILE"
  exit 0
fi

# ---------------------------------------------------------------------------
# Read state from state.json
# ---------------------------------------------------------------------------
OVERALL_CONFIDENCE=$(jq '.overall_confidence // 0' "$STATE_FILE" 2>/dev/null || echo "0")
STATE_TARGET=$(jq '.target_confidence // 95' "$STATE_FILE" 2>/dev/null || echo "95")
# Prefer marker's target_confidence, fall back to state.json
TARGET=${TARGET_CONFIDENCE:-$STATE_TARGET}

# ---------------------------------------------------------------------------
# Read last assistant message from transcript for COMPLETE tag
# ---------------------------------------------------------------------------
COMPLETE_TAG_FOUND=false
if [[ -n "$TRANSCRIPT_PATH" ]] && [[ -f "$TRANSCRIPT_PATH" ]]; then
  # Get last 100 assistant lines (bounded for performance)
  LAST_LINES=$(grep '"role":"assistant"' "$TRANSCRIPT_PATH" | tail -n 100 2>/dev/null || echo "")
  if [[ -n "$LAST_LINES" ]]; then
    LAST_OUTPUT=$(echo "$LAST_LINES" | jq -rs '
      map(.message.content[]? | select(.type == "text") | .text) | last // ""
    ' 2>/dev/null || echo "")
    if echo "$LAST_OUTPUT" | grep -q "<oracle>COMPLETE</oracle>"; then
      COMPLETE_TAG_FOUND=true
    fi
  fi
fi

# ---------------------------------------------------------------------------
# Convergence detection (ported from oracle.sh)
# ---------------------------------------------------------------------------

# Compute convergence metrics from plan.json
compute_convergence() {
  local plan_file="$1"
  local state_file="$2"

  [[ -f "$plan_file" ]] || { echo '{"gap_resolution_pct":0,"coverage_pct":0,"novelty_delta":0,"total_findings":0}'; return 0; }

  local total answered partial_high
  total=$(jq '[.questions[]] | length' "$plan_file" 2>/dev/null || echo "0")
  answered=$(jq '[.questions[] | select(.status == "answered")] | length' "$plan_file" 2>/dev/null || echo "0")
  partial_high=$(jq '[.questions[] | select(.status == "partial" and .confidence >= 70)] | length' "$plan_file" 2>/dev/null || echo "0")

  local gap_resolution=0
  if [[ "$total" -gt 0 ]]; then
    gap_resolution=$(( (answered + partial_high) * 100 / total ))
  fi

  local touched coverage=0
  touched=$(jq '[.questions[] | select((.iterations_touched // []) | length > 0)] | length' "$plan_file" 2>/dev/null || echo "0")
  if [[ "$total" -gt 0 ]]; then
    coverage=$(( touched * 100 / total ))
  fi

  local current_findings prev_findings novelty_delta
  current_findings=$(jq '[.questions[].key_findings | length] | add // 0' "$plan_file" 2>/dev/null || echo "0")
  prev_findings=$(jq '.convergence.prev_findings_count // 0' "$state_file" 2>/dev/null || echo "0")
  novelty_delta=$(( current_findings - prev_findings ))

  jq -n --argjson gap "$gap_resolution" --argjson cov "$coverage" \
        --argjson novelty "$novelty_delta" --argjson findings "$current_findings" \
    '{gap_resolution_pct: $gap, coverage_pct: $cov, novelty_delta: $novelty, total_findings: $findings}'
}

# Check if research has converged
check_convergence() {
  local state_file="$1"

  local composite_score
  composite_score=$(jq '.convergence.composite_score // 0' "$state_file" 2>/dev/null || echo "0")

  if [[ "$composite_score" -lt "$CONV_THRESHOLD" ]]; then
    return 1
  fi

  local history_len
  history_len=$(jq '(.convergence.history // []) | length' "$state_file" 2>/dev/null || echo "0")
  if [[ "$history_len" -lt 2 ]]; then
    return 1
  fi

  local low_novelty_count
  low_novelty_count=$(jq '[(.convergence.history // [])[-2:][] | select(.novelty_delta <= 1)] | length' "$state_file" 2>/dev/null || echo "0")

  [[ "$low_novelty_count" -ge 2 ]]
}

# Detect diminishing returns
detect_diminishing_returns() {
  local state_file="$1"

  local history_len
  history_len=$(jq '(.convergence.history // []) | length' "$state_file" 2>/dev/null || echo "0")

  if [[ "$history_len" -lt "$DR_WINDOW" ]]; then
    echo "continue"
    return 0
  fi

  local novelty_threshold
  case "$CURRENT_PHASE" in
    investigate) novelty_threshold=0 ;;
    *) novelty_threshold=1 ;;
  esac

  local low_change_count
  low_change_count=$(jq --argjson window "$DR_WINDOW" --argjson threshold "$novelty_threshold" \
    '[(.convergence.history // [])[-$window:][] | select(.novelty_delta <= $threshold)] | length' \
    "$state_file" 2>/dev/null || echo "0")

  if [[ "$low_change_count" -ge "$DR_WINDOW" ]]; then
    case "$CURRENT_PHASE" in
      survey|investigate) echo "strategy_change" ;;
      synthesize|verify) echo "synthesize_now" ;;
      *) echo "continue" ;;
    esac
  else
    echo "continue"
  fi
}

# Update convergence metrics in state.json
update_convergence_metrics() {
  local state_file="$1"
  local plan_file="$2"
  local next_iteration="$3"
  local next_phase="$4"

  local metrics
  metrics=$(compute_convergence "$plan_file" "$state_file")

  local gap_pct coverage_pct novelty_delta total_findings
  gap_pct=$(echo "$metrics" | jq '.gap_resolution_pct')
  coverage_pct=$(echo "$metrics" | jq '.coverage_pct')
  novelty_delta=$(echo "$metrics" | jq '.novelty_delta')
  total_findings=$(echo "$metrics" | jq '.total_findings')

  local current_confidence prev_confidence confidence_delta
  current_confidence=$(jq '.overall_confidence // 0' "$state_file" 2>/dev/null || echo "0")
  prev_confidence=$(jq '.convergence.prev_overall_confidence // 0' "$state_file" 2>/dev/null || echo "0")
  confidence_delta=$(( current_confidence - prev_confidence ))

  # Composite score: gap*0.4 + coverage*0.3 + (novelty<=1?100:0)*0.3
  local novelty_component composite_score converged
  if [[ "$novelty_delta" -le 1 ]]; then
    novelty_component=100
  else
    novelty_component=0
  fi
  composite_score=$(( gap_pct * 40 / 100 + coverage_pct * 30 / 100 + novelty_component * 30 / 100 ))

  if [[ "$composite_score" -ge "$CONV_THRESHOLD" ]]; then
    converged="true"
  else
    converged="false"
  fi

  # Atomic write
  jq --argjson prev_findings "$total_findings" \
     --argjson prev_confidence "$current_confidence" \
     --argjson iteration "$next_iteration" \
     --argjson novelty "$novelty_delta" \
     --argjson conf_delta "$confidence_delta" \
     --argjson gap "$gap_pct" \
     --argjson cov "$coverage_pct" \
     --arg phase "$next_phase" \
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

# Compute trust scores from plan.json source tracking
compute_trust_scores() {
  local plan_file="$1"

  [[ -f "$plan_file" ]] || return 0

  local has_structured
  has_structured=$(jq '
    [.questions[].key_findings[] | type] | if length == 0 then false else any(. == "object") end
  ' "$plan_file" 2>/dev/null || echo "false")

  if [[ "$has_structured" != "true" ]]; then
    return 0
  fi

  local total_findings single_source multi_source
  total_findings=$(jq '[.questions[].key_findings[]] | length' "$plan_file" 2>/dev/null || echo "0")
  single_source=$(jq '[.questions[].key_findings[] | select(type == "object" and (.source_ids | length) == 1)] | length' "$plan_file" 2>/dev/null || echo "0")
  multi_source=$(jq '[.questions[].key_findings[] | select(type == "object" and (.source_ids | length) >= 2)] | length' "$plan_file" 2>/dev/null || echo "0")
  local no_source
  no_source=$(jq '[.questions[].key_findings[] | select(type == "object" and ((.source_ids // []) | length) == 0)] | length' "$plan_file" 2>/dev/null || echo "0")

  local trust_ratio=0
  if [[ "$total_findings" -gt 0 ]]; then
    trust_ratio=$(( multi_source * 100 / total_findings ))
  fi

  jq --argjson total "$total_findings" \
     --argjson single "$single_source" \
     --argjson multi "$multi_source" \
     --argjson nosrc "$no_source" \
     --argjson ratio "$trust_ratio" \
     '.trust_summary = {
       total_findings: $total,
       single_source: $single,
       multi_source: $multi,
       no_source: $nosrc,
       trust_ratio: $ratio
     }' "$plan_file" > "$plan_file.tmp" && mv "$plan_file.tmp" "$plan_file"
}

# ---------------------------------------------------------------------------
# Determine next phase (ported from oracle.sh)
# ---------------------------------------------------------------------------
determine_next_phase() {
  local state_file="$1"
  local plan_file="$2"

  [[ -f "$state_file" ]] || { echo "survey"; return 0; }
  [[ -f "$plan_file" ]] || { echo "survey"; return 0; }

  local total_questions touched_count avg_confidence below_50_count

  total_questions=$(jq '[.questions[]] | length' "$plan_file" 2>/dev/null || echo "0")
  if [[ "$total_questions" -eq 0 ]]; then
    echo "survey"
    return 0
  fi

  touched_count=$(jq '[.questions[] | select((.iterations_touched // []) | length > 0)] | length' "$plan_file" 2>/dev/null || echo "0")
  avg_confidence=$(jq '[.questions[].confidence] | if length > 0 then (add / length) else 0 end | floor' "$plan_file" 2>/dev/null || echo "0")
  below_50_count=$(jq '[.questions[] | select(.status != "answered" and .confidence < 50)] | length' "$plan_file" 2>/dev/null || echo "0")

  if [[ "$avg_confidence" -ge 80 ]]; then
    echo "verify"
    return 0
  fi

  if [[ "$avg_confidence" -ge 60 ]] || [[ "$below_50_count" -lt 2 ]]; then
    echo "synthesize"
    return 0
  fi

  if [[ "$touched_count" -ge "$total_questions" ]] || [[ "$avg_confidence" -ge 25 ]]; then
    echo "investigate"
    return 0
  fi

  echo "survey"
}

# ---------------------------------------------------------------------------
# Phase directive generators (copied verbatim from oracle.sh)
# ---------------------------------------------------------------------------
phase_directive_survey() {
  cat <<'DIRECTIVE'
## Current Phase: SURVEY

Cast a wide net -- get initial findings for every open question. Target untouched
questions first (those with empty iterations_touched arrays). Aim for 20-40%
confidence per question. List all discovered unknowns in gaps.md.

Do NOT go deep on any single question yet. Breadth over depth in this phase.
Your goal is to ensure every question has at least some initial findings before
the research moves to the investigation phase.

Source tracking is MANDATORY -- register sources and link every finding to source_ids.

---

DIRECTIVE
}

phase_directive_investigate() {
  cat <<'DIRECTIVE'
## Current Phase: INVESTIGATE

Target the lowest-confidence question and go DEEP. You MUST reference existing
findings in synthesis.md and ADD NEW information, not restate what is already there.
Aim to push confidence above 70% for your target question.

Update gaps.md with specific remaining unknowns. If you find contradictions with
existing findings, document them explicitly. One thoroughly investigated question
per iteration is better than shallow passes on many.

Source tracking is MANDATORY this iteration. Every new finding must have at least one source_id.

---

DIRECTIVE
}

phase_directive_synthesize() {
  cat <<'DIRECTIVE'
## Current Phase: SYNTHESIZE

Read ALL findings in synthesis.md before doing anything. Identify connections,
patterns, and contradictions ACROSS questions. Consolidate redundant findings.
Resolve contradictions with evidence. Push overall confidence toward the target.

Your job is NOT to find new information -- it is to make sense of what has already
been found. Cross-reference answers between questions. Strengthen weak claims
with evidence from other questions. Remove speculation that lacks support.

Verify source attribution is complete. Flag any findings missing source_ids.

---

DIRECTIVE
}

phase_directive_verify() {
  cat <<'DIRECTIVE'
## Current Phase: VERIFY

Focus on claims in gaps.md contradictions section. Cross-reference key findings
with additional sources. Confirm or correct confidence scores. Mark well-supported
questions as answered with 90%+ confidence.

Final gaps.md should contain only genuinely unresolvable unknowns. If a contradiction
cannot be resolved, document both positions with evidence quality assessment.
This is the final quality pass before research completion.

Cross-reference source coverage. Ensure all key findings have 2+ independent sources.

---

DIRECTIVE
}

phase_directive_for() {
  local phase="$1"
  case "$phase" in
    survey) phase_directive_survey ;;
    investigate) phase_directive_investigate ;;
    synthesize) phase_directive_synthesize ;;
    verify) phase_directive_verify ;;
    *) echo "## Current Phase: $phase"; echo ""; echo "---"; echo "" ;;
  esac
}

# ---------------------------------------------------------------------------
# Strategy modifier (ported from oracle.sh)
# ---------------------------------------------------------------------------
strategy_modifier() {
  local state_file="$1"
  local strategy
  strategy=$(jq -r '.strategy // "adaptive"' "$state_file" 2>/dev/null || echo "adaptive")

  case "$strategy" in
    breadth-first)
      cat <<'STRATEGY'

STRATEGY NOTE: Breadth-first mode is active. Prioritize covering ALL questions
before going deep on any single one. Aim for broad coverage across the research
plan. When multiple questions are untouched, target the easiest-to-answer first
for maximum coverage.

STRATEGY
      ;;
    depth-first)
      cat <<'STRATEGY'

STRATEGY NOTE: Depth-first mode is active. Pick the single most important open
question and investigate it exhaustively. Push confidence to 80%+ before moving
to the next question. Prefer thorough, well-sourced answers over broad coverage.

STRATEGY
      ;;
    *) ;; # adaptive: no modifier
  esac
}

# ---------------------------------------------------------------------------
# Steering signals (ported from oracle.sh)
# ---------------------------------------------------------------------------
read_steering_signals() {
  local utils="$AETHER_ROOT/.aether/aether-utils.sh"

  [[ -f "$utils" ]] || return 0

  local signals
  signals=$(bash "$utils" pheromone-read 2>/dev/null || echo '{"signals":[]}')

  local signal_array
  signal_array=$(echo "$signals" | jq -c '.result.signals // .signals // []' 2>/dev/null || echo '[]')

  local count
  count=$(echo "$signal_array" | jq 'length' 2>/dev/null || echo "0")

  [[ "$count" -gt 0 ]] || return 0

  local directive=""

  # REDIRECT (max 2)
  local redirects
  redirects=$(echo "$signal_array" | jq -r '
    map(select(.type == "REDIRECT"))
    | sort_by(-.effective_strength)
    | .[:2]
    | .[] | "- [" + ((.effective_strength * 100 | floor | tostring)) + "%] \"" + (.content.text // "") + "\""
  ' 2>/dev/null || echo "")

  if [[ -n "$redirects" ]]; then
    directive+="**REDIRECT (Hard constraints -- MUST follow):**"$'\n'"$redirects"$'\n\n'
  fi

  # FOCUS (max 3)
  local focuses
  focuses=$(echo "$signal_array" | jq -r '
    map(select(.type == "FOCUS"))
    | sort_by(-.effective_strength)
    | .[:3]
    | .[] | "- [" + ((.effective_strength * 100 | floor | tostring)) + "%] \"" + (.content.text // "") + "\""
  ' 2>/dev/null || echo "")

  if [[ -n "$focuses" ]]; then
    directive+="**FOCUS (Prioritize these areas):**"$'\n'"$focuses"$'\n\n'
  fi

  # FEEDBACK (max 2)
  local feedbacks
  feedbacks=$(echo "$signal_array" | jq -r '
    map(select(.type == "FEEDBACK"))
    | sort_by(-.effective_strength)
    | .[:2]
    | .[] | "- [" + ((.effective_strength * 100 | floor | tostring)) + "%] \"" + (.content.text // "") + "\""
  ' 2>/dev/null || echo "")

  if [[ -n "$feedbacks" ]]; then
    directive+="**FEEDBACK (Adjust approach):**"$'\n'"$feedbacks"$'\n\n'
  fi

  if [[ -n "$directive" ]]; then
    echo "## Active Steering Signals"
    echo ""
    echo "$directive"
    echo "When selecting your target question, PRIORITIZE questions related to FOCUS areas."
    echo "REDIRECT signals are hard constraints -- adjust your approach to comply."
    echo "FEEDBACK signals are suggestions -- incorporate where appropriate."
    echo ""
    echo "---"
    echo ""
  fi
}

# ---------------------------------------------------------------------------
# Synthesis prompt builder (ported from oracle.sh)
# ---------------------------------------------------------------------------
build_synthesis_prompt() {
  local reason="$1"
  local template
  template=$(jq -r '.template // "custom"' "$STATE_FILE" 2>/dev/null || echo "custom")

  cat <<SYNTHESIS_DIRECTIVE
## SYNTHESIS PASS (Final Report)

This is the final pass. The oracle loop has ended (reason: $reason).
Produce the best possible research report from the current state.

Read ALL of these files:
- .aether/oracle/state.json -- session metadata
- .aether/oracle/plan.json -- questions, findings, confidence, AND sources registry
- .aether/oracle/synthesis.md -- accumulated findings
- .aether/oracle/gaps.md -- remaining unknowns

If any state file is unreadable, skip it and work with what you have.

Then REWRITE synthesis.md as a structured final report.

SYNTHESIS_DIRECTIVE

  # Template-specific sections
  case "$template" in
    tech-eval)
      cat <<'TEMPLATE'
### Required Sections:
1. **Executive Summary** -- 2-3 paragraphs: what was evaluated, key conclusion, recommendation
2. **Comparison Matrix** -- Table comparing the evaluated technology against alternatives on key dimensions
3. **Pros and Cons** -- Bullet lists with evidence citations
4. **Adoption Assessment** -- Community size, maintenance status, release cadence
5. **Migration/Integration Path** -- Steps to adopt, estimated effort, risks
6. **Recommendation** -- Clear recommendation with confidence level
7. **Open Questions** -- Remaining gaps
8. **Sources** -- All sources with inline citations [S1], [S2] format

TEMPLATE
      ;;
    architecture-review)
      cat <<'TEMPLATE'
### Required Sections:
1. **Executive Summary** -- 2-3 paragraphs: system overview, key findings, critical risks
2. **Component Map** -- List of major components with responsibilities
3. **Dependency Analysis** -- How components connect, coupling assessment
4. **Risk Assessment** -- Single points of failure, complexity hotspots
5. **Scalability Analysis** -- Current capacity, growth limitations
6. **Improvement Recommendations** -- Prioritized list
7. **Open Questions** -- Remaining gaps
8. **Sources** -- All sources with inline citations [S1], [S2] format

TEMPLATE
      ;;
    bug-investigation)
      cat <<'TEMPLATE'
### Required Sections:
1. **Executive Summary** -- 1-2 paragraphs: bug description, root cause, recommended fix
2. **Reproduction Steps** -- Exact steps to reproduce
3. **Root Cause Analysis** -- What causes the bug, code paths involved
4. **Impact Assessment** -- Who is affected, severity
5. **Fix Recommendations** -- Proposed fixes ranked by safety and effort
6. **Related Issues** -- Similar bugs, regression risk
7. **Open Questions** -- Remaining gaps
8. **Sources** -- All sources with inline citations [S1], [S2] format

TEMPLATE
      ;;
    best-practices)
      cat <<'TEMPLATE'
### Required Sections:
1. **Executive Summary** -- 2-3 paragraphs: domain overview, key recommendations
2. **Best Practice Benchmark** -- What industry consensus considers best practice
3. **Current State Assessment** -- How the subject compares
4. **Gap Analysis** -- Specific gaps prioritized by impact
5. **Action Plan** -- Ordered steps to close gaps
6. **Open Questions** -- Remaining gaps
7. **Sources** -- All sources with inline citations [S1], [S2] format

TEMPLATE
      ;;
    *)
      cat <<'TEMPLATE'
### Required Sections:
1. **Executive Summary** -- 2-3 paragraphs summarizing what was found
2. **Findings by Question** -- organized by sub-question, with confidence %
3. **Open Questions** -- remaining gaps
4. **Methodology Notes** -- how many iterations, which phases completed
5. **Sources** -- List ALL sources from plan.json sources registry

TEMPLATE
      ;;
  esac

  # Common directives
  cat <<'COMMON'

### Confidence Grouping:
Within each findings section, group findings by confidence level:
- **High confidence (80%+)** -- list first with full citations
- **Medium confidence (50-79%)** -- list with caveats
- **Low confidence (<50%)** -- list as tentative/unverified

Use inline citations [S1], [S2] linking findings to their sources.
Flag single-source findings with (single source) marker.

Also update state.json: set status to "complete" if reason is "converged",
or "stopped" otherwise.

COMMON

  # Append the base oracle.md for tool access and rules
  cat "$ORACLE_MD"
}

# ---------------------------------------------------------------------------
# Update marker file with new values
# ---------------------------------------------------------------------------
update_marker() {
  local new_iteration="$1"
  local new_phase="$2"
  local new_synthesis_done="$3"

  local oracle_md_body
  oracle_md_body=$(awk '/^---$/{i++; next} i>=2' "$MARKER_FILE")

  cat > "${MARKER_FILE}.tmp.$$" <<MARKER
---
iteration: $new_iteration
max_iterations: $MAX_ITERATIONS
session_id: $MARKER_SESSION_ID
phase: $new_phase
target_confidence: $TARGET_CONFIDENCE
synthesis_done: $new_synthesis_done
oracle_md_path: .aether/utils/oracle/oracle.md
---
$oracle_md_body
MARKER
  mv "${MARKER_FILE}.tmp.$$" "$MARKER_FILE"
}

# ---------------------------------------------------------------------------
# Update state.json iteration and phase
# ---------------------------------------------------------------------------
update_state_json() {
  local new_iteration="$1"
  local new_phase="$2"

  local ts
  ts=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

  jq --argjson iter "$new_iteration" \
     --arg phase "$new_phase" \
     --arg ts "$ts" \
     '.iteration = $iter | .phase = $phase | .last_updated = $ts' \
     "$STATE_FILE" > "$STATE_FILE.tmp" && mv "$STATE_FILE.tmp" "$STATE_FILE"
}

# ---------------------------------------------------------------------------
# MAIN DECISION LOGIC
# ---------------------------------------------------------------------------

NEXT_ITERATION=$((ITERATION + 1))

# Check completion conditions
IS_COMPLETE=false
COMPLETION_REASON=""

# 1. Max iterations reached
if [[ "$MAX_ITERATIONS" -gt 0 ]] && [[ "$NEXT_ITERATION" -gt "$MAX_ITERATIONS" ]]; then
  IS_COMPLETE=true
  COMPLETION_REASON="max_iterations"
fi

# 2. Confidence target met
if [[ "$OVERALL_CONFIDENCE" -ge "$TARGET" ]]; then
  IS_COMPLETE=true
  COMPLETION_REASON="${COMPLETION_REASON:-confidence_target}"
fi

# 3. AI signaled complete
if [[ "$COMPLETE_TAG_FOUND" == "true" ]]; then
  IS_COMPLETE=true
  COMPLETION_REASON="${COMPLETION_REASON:-ai_signal}"
fi

# 4. Convergence detected
if [[ -f "$PLAN_FILE" ]] && check_convergence "$STATE_FILE"; then
  IS_COMPLETE=true
  COMPLETION_REASON="${COMPLETION_REASON:-convergence}"
fi

# Handle synthesis pass
if [[ "$IS_COMPLETE" == "true" ]]; then
  if [[ "$SYNTHESIS_DONE" == "true" ]]; then
    # Research complete AND synthesis done -- allow stop
    rm -f "$MARKER_FILE"

    # Update state.json status
    local final_status
    case "$COMPLETION_REASON" in
      max_iterations) final_status="stopped" ;;
      convergence|confidence_target|ai_signal) final_status="complete" ;;
      *) final_status="stopped" ;;
    esac
    jq --arg status "$final_status" --arg reason "$COMPLETION_REASON" \
      '.status = $status | .stop_reason = $reason' \
      "$STATE_FILE" > "$STATE_FILE.tmp" && mv "$STATE_FILE.tmp" "$STATE_FILE"

    # Allow exit
    exit 0
  fi

  # Complete but no synthesis yet -- trigger synthesis pass
  update_marker "$NEXT_ITERATION" "synthesis-pass" "true"
  update_state_json "$NEXT_ITERATION" "synthesis-pass"

  # Build synthesis prompt
  SYNTHESIS_PROMPT=$(build_synthesis_prompt "$COMPLETION_REASON")

  SYSTEM_MSG="Oracle iteration $NEXT_ITERATION (SYNTHESIS PASS) -- Final report generation"

  jq -n \
    --arg prompt "$SYNTHESIS_PROMPT" \
    --arg msg "$SYSTEM_MSG" \
    '{
      "decision": "block",
      "reason": $prompt,
      "systemMessage": $msg
    }'

  exit 0
fi

# ---------------------------------------------------------------------------
# NOT COMPLETE -- continue research loop
# ---------------------------------------------------------------------------

# Determine next phase from plan.json metrics
NEXT_PHASE=$(determine_next_phase "$STATE_FILE" "$PLAN_FILE")

# Check diminishing returns -- may force phase advancement
if [[ -f "$PLAN_FILE" ]]; then
  DR_RESULT=$(detect_diminishing_returns "$STATE_FILE")
  case "$DR_RESULT" in
    strategy_change)
      # Force synthesize phase
      NEXT_PHASE="synthesize"
      ;;
    synthesize_now)
      # Force early synthesis completion
      NEXT_PHASE="verify"
      ;;
  esac
fi

# Update convergence metrics (reads plan.json, writes state.json)
if [[ -f "$PLAN_FILE" ]]; then
  update_convergence_metrics "$STATE_FILE" "$PLAN_FILE" "$NEXT_ITERATION" "$NEXT_PHASE"
  compute_trust_scores "$PLAN_FILE"
fi

# Update state files
update_marker "$NEXT_ITERATION" "$NEXT_PHASE" "$SYNTHESIS_DONE"
update_state_json "$NEXT_ITERATION" "$NEXT_PHASE"

# Generate research-plan.md summary
if [[ -f "$PLAN_FILE" ]]; then
  {
    echo "# Research Plan"
    echo ""
    local_topic=$(jq -r '.topic // "unknown"' "$STATE_FILE" 2>/dev/null || echo "unknown")
    local_conf=$(jq '.overall_confidence // 0' "$STATE_FILE" 2>/dev/null || echo "0")
    echo "**Topic:** $local_topic"
    echo "**Status:** active | **Iteration:** $NEXT_ITERATION of $MAX_ITERATIONS"
    echo "**Overall Confidence:** ${local_conf}%"
    echo ""
    echo "## Questions"
    echo "| # | Question | Status | Confidence |"
    echo "|---|----------|--------|------------|"
    jq -r '.questions[] | "| \(.id) | \(.text) | \(.status) | \(.confidence)% |"' "$PLAN_FILE" 2>/dev/null || true
    echo ""
    echo "## Next Steps"
    local_next
    local_next=$(jq -r '[.questions[] | select(.status != "answered")] | sort_by(.confidence) | first | .text // "All questions answered"' "$PLAN_FILE" 2>/dev/null || echo "Continue research")
    echo "Next investigation: $local_next"
    echo ""
    echo "---"
    echo "*Generated from plan.json -- do not edit directly*"
  } > "$ORACLE_DIR/research-plan.md"
fi

# Build the full iteration prompt
FULL_PROMPT=""

# Phase directive
FULL_PROMPT+=$(phase_directive_for "$NEXT_PHASE")

# Strategy modifier
FULL_PROMPT+=$(strategy_modifier "$STATE_FILE")

# Steering signals
STEERING=$(read_steering_signals "$AETHER_ROOT")
if [[ -n "$STEERING" ]]; then
  FULL_PROMPT+="$STEERING"
fi

# Base oracle.md prompt
if [[ -f "$ORACLE_MD" ]]; then
  FULL_PROMPT+=$(cat "$ORACLE_MD")
fi

SYSTEM_MSG="Oracle iteration $NEXT_ITERATION ($NEXT_PHASE phase) | Confidence: ${OVERALL_CONFIDENCE}% / ${TARGET}% | Iterations remaining: $((MAX_ITERATIONS - NEXT_ITERATION))"

jq -n \
  --arg prompt "$FULL_PROMPT" \
  --arg msg "$SYSTEM_MSG" \
  '{
    "decision": "block",
    "reason": $prompt,
    "systemMessage": $msg
  }'

exit 0
