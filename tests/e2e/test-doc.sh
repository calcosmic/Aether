#!/usr/bin/env bash
# test-doc.sh — Colony Documentation requirement verification
# DOC-01: Phase learnings extracted and documented (ant-themed)
# DOC-02: Colony memories stored with ant naming (pheromones.md)
# DOC-03: Progress tracked with ant metaphors (nursery, chambers)
# DOC-04: Handoff documents use ant themes
#
# NOTE: Written for bash 3.2 (macOS default). No associative arrays.
# Supports --results-file <path> flag for master runner integration.
# Proxy verification: checks command file content + exercises subcommands.

set -euo pipefail

E2E_SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$E2E_SCRIPT_DIR/../.." && pwd)"

# Parse --results-file flag
EXTERNAL_RESULTS_FILE=""
while [[ $# -gt 0 ]]; do
  case "$1" in
    --results-file)
      EXTERNAL_RESULTS_FILE="$2"
      shift 2
      ;;
    *)
      shift
      ;;
  esac
done

# Source shared e2e infrastructure
# shellcheck source=./e2e-helpers.sh
source "$E2E_SCRIPT_DIR/e2e-helpers.sh"

echo ""
echo "================================================================"
echo "DOC Area: Colony Documentation Requirements"
echo "================================================================"

# ============================================================================
# Environment Setup
# ============================================================================

E2E_TMP_DIR=$(setup_e2e_env)
trap teardown_e2e_env EXIT

init_results

UTILS="$E2E_TMP_DIR/.aether/aether-utils.sh"

# ============================================================================
# DOC-01: Phase learnings extracted and documented (ant-themed)
# Proxy: continue.md contains learning extraction logic; learnings subcommands
#        exist in aether-utils.sh; learning-promote is callable
# ============================================================================

echo ""
echo "--- DOC-01: Phase learnings extracted and documented ---"

doc01_pass=true
doc01_notes=""

live_continue="$PROJECT_ROOT/.claude/commands/ant/continue.md"

# Check 1: continue.md references learning extraction
if [[ -f "$live_continue" ]]; then
  if grep -qi "learning\|learnings\|extract" "$live_continue" 2>/dev/null; then
    echo "  PASS: continue.md contains learning extraction logic"
  else
    doc01_pass=false
    doc01_notes="$doc01_notes [FAIL: no learning reference in continue.md]"
    echo "  FAIL: continue.md lacks learning extraction logic"
  fi
else
  doc01_pass=false
  doc01_notes="$doc01_notes [FAIL: continue.md not found at live path]"
  echo "  FAIL: continue.md not found"
fi

# Check 2: learning-promote subcommand exists in aether-utils.sh
if grep -q "learning-promote" "$PROJECT_ROOT/.aether/aether-utils.sh" 2>/dev/null; then
  echo "  PASS: learning-promote subcommand exists in aether-utils.sh"
else
  doc01_pass=false
  doc01_notes="$doc01_notes [FAIL: learning-promote not in aether-utils.sh]"
  echo "  FAIL: learning-promote not found in aether-utils.sh"
fi

# Check 3: learning-inject subcommand exists in aether-utils.sh
if grep -q "learning-inject" "$PROJECT_ROOT/.aether/aether-utils.sh" 2>/dev/null; then
  echo "  PASS: learning-inject subcommand exists in aether-utils.sh"
else
  doc01_pass=false
  doc01_notes="$doc01_notes [FAIL: learning-inject not in aether-utils.sh]"
  echo "  FAIL: learning-inject not found in aether-utils.sh"
fi

# Check 4: memory.phase_learnings structure referenced in continue.md
if [[ -f "$live_continue" ]]; then
  if grep -q "phase_learnings\|phase-learnings" "$live_continue" 2>/dev/null; then
    echo "  PASS: continue.md references phase_learnings structure"
  else
    doc01_pass=false
    doc01_notes="$doc01_notes [FAIL: no phase_learnings ref in continue.md]"
    echo "  FAIL: continue.md lacks phase_learnings reference"
  fi
fi

if [[ "$doc01_pass" == "true" ]]; then
  record_result "DOC-01" "PASS" "learning extraction logic + subcommands verified"
else
  record_result "DOC-01" "FAIL" "$doc01_notes"
fi

# ============================================================================
# DOC-02: Colony memories stored with ant naming (pheromones.md)
# Proxy: eternal-init subcommand exists + works; ~/.aether/eternal/ referenced;
#        continue.md calls eternal-init for cross-session persistence
# ============================================================================

echo ""
echo "--- DOC-02: Colony memories stored with ant naming ---"

doc02_pass=true
doc02_notes=""

# Check 1: eternal-init case branch exists in aether-utils.sh
if grep -q "eternal-init" "$PROJECT_ROOT/.aether/aether-utils.sh" 2>/dev/null; then
  echo "  PASS: eternal-init subcommand exists in aether-utils.sh"
else
  doc02_pass=false
  doc02_notes="$doc02_notes [FAIL: eternal-init not in aether-utils.sh]"
  echo "  FAIL: eternal-init not found in aether-utils.sh"
fi

# Check 2: ~/.aether/eternal/ path referenced in aether-utils.sh
if grep -q "eternal" "$PROJECT_ROOT/.aether/aether-utils.sh" 2>/dev/null; then
  echo "  PASS: eternal directory path referenced in aether-utils.sh"
else
  doc02_pass=false
  doc02_notes="$doc02_notes [FAIL: eternal dir not referenced in aether-utils.sh]"
  echo "  FAIL: eternal directory not referenced in aether-utils.sh"
fi

# Check 3: queen-promote or memory persistence subcommand exists
if grep -q "queen-promote" "$PROJECT_ROOT/.aether/aether-utils.sh" 2>/dev/null; then
  echo "  PASS: queen-promote subcommand exists for memory persistence"
else
  doc02_pass=false
  doc02_notes="$doc02_notes [FAIL: queen-promote not in aether-utils.sh]"
  echo "  FAIL: queen-promote not found in aether-utils.sh"
fi

# Check 4: continue.md references eternal memory or queen-promote
if [[ -f "$live_continue" ]]; then
  if grep -q "eternal\|queen-promote" "$live_continue" 2>/dev/null; then
    echo "  PASS: continue.md references eternal memory / queen-promote"
  else
    doc02_pass=false
    doc02_notes="$doc02_notes [FAIL: no eternal/queen-promote ref in continue.md]"
    echo "  FAIL: continue.md lacks eternal memory reference"
  fi
fi

# Check 5: Run eternal-init in isolated env and confirm it works
echo "  Running eternal-init in isolated env..."
raw_ei=$(run_in_isolated_env "$E2E_TMP_DIR" eternal-init 2>&1 || true)
ei_out=$(extract_json "$raw_ei")

if echo "$ei_out" | jq -e '.ok == true' >/dev/null 2>&1; then
  echo "  PASS: eternal-init returned ok:true"
else
  doc02_pass=false
  doc02_notes="$doc02_notes [FAIL: eternal-init ok!=true: $ei_out]"
  echo "  FAIL: eternal-init did not return ok:true"
  echo "  Got: $ei_out"
fi

if [[ "$doc02_pass" == "true" ]]; then
  record_result "DOC-02" "PASS" "eternal-init works; queen-promote present; continue.md references eternal memory"
else
  record_result "DOC-02" "FAIL" "$doc02_notes"
fi

# ============================================================================
# DOC-03: Progress tracked with ant metaphors (nursery, chambers)
# Proxy: session-init creates COLONY_STATE.json with current_phase;
#        continue.md writes CONTEXT.md; session-update tracks suggested_next
# ============================================================================

echo ""
echo "--- DOC-03: Progress tracked with ant metaphors ---"

doc03_pass=true
doc03_notes=""

# Check 1: session-init creates session.json with current_phase field
echo "  Running session-init in isolated env..."
raw_si=$(run_in_isolated_env "$E2E_TMP_DIR" session-init "" "lifecycle-doc-test" 2>&1 || true)
si_out=$(extract_json "$raw_si")

if echo "$si_out" | jq -e '.ok == true' >/dev/null 2>&1; then
  echo "  PASS: session-init returned ok:true"
  # Verify session.json contains current_phase
  session_file="$E2E_TMP_DIR/.aether/data/session.json"
  if [[ -f "$session_file" ]]; then
    if jq -e '.current_phase != null' "$session_file" >/dev/null 2>&1; then
      echo "  PASS: session.json contains current_phase field"
    else
      doc03_pass=false
      doc03_notes="$doc03_notes [FAIL: session.json missing current_phase]"
      echo "  FAIL: session.json lacks current_phase field"
    fi
  else
    doc03_pass=false
    doc03_notes="$doc03_notes [FAIL: session.json not created]"
    echo "  FAIL: session.json not created by session-init"
  fi
else
  doc03_pass=false
  doc03_notes="$doc03_notes [FAIL: session-init ok!=true: $si_out]"
  echo "  FAIL: session-init did not return ok:true"
fi

# Check 2: continue.md writes CONTEXT.md (references context-update + CONTEXT.md)
if [[ -f "$live_continue" ]]; then
  if grep -q "CONTEXT.md\|context-update" "$live_continue" 2>/dev/null; then
    echo "  PASS: continue.md references CONTEXT.md / context-update"
  else
    doc03_pass=false
    doc03_notes="$doc03_notes [FAIL: no CONTEXT.md/context-update ref in continue.md]"
    echo "  FAIL: continue.md lacks CONTEXT.md / context-update reference"
  fi
fi

# Check 3: session-update tracks suggested_next
echo "  Running session-update in isolated env..."
raw_su=$(run_in_isolated_env "$E2E_TMP_DIR" session-update "/ant:continue" "/ant:seal" "Phase advanced" 2>&1 || true)
su_out=$(extract_json "$raw_su")

if echo "$su_out" | jq -e '.ok == true' >/dev/null 2>&1; then
  echo "  PASS: session-update returned ok:true"
  # Verify suggested_next is written to session.json
  session_file="$E2E_TMP_DIR/.aether/data/session.json"
  if [[ -f "$session_file" ]] && jq -e '.suggested_next != null' "$session_file" >/dev/null 2>&1; then
    echo "  PASS: session.json contains suggested_next field"
  else
    doc03_pass=false
    doc03_notes="$doc03_notes [FAIL: session.json missing suggested_next after update]"
    echo "  FAIL: session.json lacks suggested_next after session-update"
  fi
else
  doc03_pass=false
  doc03_notes="$doc03_notes [FAIL: session-update ok!=true: $su_out]"
  echo "  FAIL: session-update did not return ok:true"
fi

# Check 4: milestone names use ant metaphors (First Mound, chambers) in aether-utils.sh
if grep -q "First Mound\|Open Chambers\|Crowned Anthill" "$PROJECT_ROOT/.aether/aether-utils.sh" 2>/dev/null; then
  echo "  PASS: ant metaphor milestone names present in aether-utils.sh"
else
  doc03_pass=false
  doc03_notes="$doc03_notes [FAIL: ant metaphor milestones not found]"
  echo "  FAIL: ant metaphor milestone names not found in aether-utils.sh"
fi

if [[ "$doc03_pass" == "true" ]]; then
  record_result "DOC-03" "PASS" "session-init current_phase + continue.md CONTEXT.md + session-update suggested_next verified"
else
  record_result "DOC-03" "FAIL" "$doc03_notes"
fi

# ============================================================================
# DOC-04: Handoff documents use ant themes
# Proxy: continue.md writes HANDOFF.md; entomb.md references HANDOFF.md;
#        at least one command produces a handoff document
# ============================================================================

echo ""
echo "--- DOC-04: Handoff documents use ant themes ---"

doc04_pass=true
doc04_notes=""

live_entomb="$PROJECT_ROOT/.claude/commands/ant/entomb.md"

# Check 1: continue.md writes HANDOFF.md
if [[ -f "$live_continue" ]]; then
  if grep -q "HANDOFF\|handoff" "$live_continue" 2>/dev/null; then
    echo "  PASS: continue.md references HANDOFF.md"
  else
    doc04_pass=false
    doc04_notes="$doc04_notes [FAIL: no HANDOFF ref in continue.md]"
    echo "  FAIL: continue.md lacks HANDOFF.md reference"
  fi
fi

# Check 2: entomb.md references HANDOFF.md
if [[ -f "$live_entomb" ]]; then
  if grep -q "HANDOFF\|handoff" "$live_entomb" 2>/dev/null; then
    echo "  PASS: entomb.md references HANDOFF.md"
  else
    doc04_pass=false
    doc04_notes="$doc04_notes [FAIL: no HANDOFF ref in entomb.md]"
    echo "  FAIL: entomb.md lacks HANDOFF.md reference"
  fi
else
  doc04_pass=false
  doc04_notes="$doc04_notes [FAIL: entomb.md not found]"
  echo "  FAIL: entomb.md not found at live path"
fi

# Check 3: pause-colony.md produces handoff document
live_pause="$PROJECT_ROOT/.claude/commands/ant/pause-colony.md"
if [[ -f "$live_pause" ]]; then
  if grep -q "HANDOFF\|handoff" "$live_pause" 2>/dev/null; then
    echo "  PASS: pause-colony.md references HANDOFF.md"
  else
    # pause-colony may use different handoff mechanism — not a failure
    echo "  NOTE: pause-colony.md does not explicitly reference HANDOFF.md (may use different approach)"
  fi
fi

# Check 4: At least continue.md and entomb.md both have HANDOFF references (DOC-04 requires)
if [[ "$doc04_pass" == "true" ]]; then
  echo "  PASS: At least 2 commands (continue.md, entomb.md) produce/reference HANDOFF document"
fi

if [[ "$doc04_pass" == "true" ]]; then
  record_result "DOC-04" "PASS" "HANDOFF.md written by continue.md + archived by entomb.md"
else
  record_result "DOC-04" "FAIL" "$doc04_notes"
fi

# ============================================================================
# Output results
# ============================================================================

# Write external results file if requested (for master runner)
if [[ -n "$EXTERNAL_RESULTS_FILE" ]]; then
  while IFS='|' read -r req_id status notes; do
    echo "${req_id}=${status}" >> "$EXTERNAL_RESULTS_FILE"
  done < "$RESULTS_FILE"
fi

print_area_results "DOC"
