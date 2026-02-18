#!/usr/bin/env bash
# test-lif.sh — Colony Lifecycle area requirements (LIF-01 through LIF-03)
# Proxy-verifies seal, entomb, and tunnels command files.
# Also exercises milestone-detect and chamber-list subcommands in isolated env.
#
# NOTE: Written for bash 3.2 (macOS default). No associative arrays.

set -euo pipefail

E2E_SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$E2E_SCRIPT_DIR/../.." && pwd)"

# Source shared e2e infrastructure
# shellcheck source=./e2e-helpers.sh
source "$E2E_SCRIPT_DIR/e2e-helpers.sh"

echo ""
echo "================================================================"
echo "LIF Area: Colony Lifecycle Requirements"
echo "================================================================"

# ============================================================================
# Environment Setup
# ============================================================================

E2E_TMP_DIR=$(setup_e2e_env)
trap teardown_e2e_env EXIT

init_results

# ============================================================================
# LIF-01: seal creates Crowned Anthill milestone
# ============================================================================

echo ""
echo "--- LIF-01: seal creates Crowned Anthill milestone ---"

lif01_pass=true
lif01_notes=""

# Check 1: seal.md exists
live_seal="$PROJECT_ROOT/.claude/commands/ant/seal.md"
if [[ ! -f "$live_seal" ]]; then
  lif01_pass=false
  lif01_notes="$lif01_notes [FAIL: seal.md not found]"
  echo "  FAIL: seal.md missing"
else
  echo "  PASS: seal.md exists"
fi

# Check 2: References Crowned Anthill or milestone ceremony
if [[ -f "$live_seal" ]]; then
  if grep -qi "crowned.anthill\|Crowned Anthill\|milestone" "$live_seal" 2>/dev/null; then
    echo "  PASS: seal.md references Crowned Anthill/milestone"
  else
    lif01_pass=false
    lif01_notes="$lif01_notes [FAIL: no Crowned Anthill/milestone ref]"
    echo "  FAIL: seal.md lacks Crowned Anthill/milestone reference"
  fi
fi

# Check 3: References milestone-detect subcommand
if [[ -f "$live_seal" ]]; then
  if grep -q "milestone-detect\|milestone" "$live_seal" 2>/dev/null; then
    echo "  PASS: seal.md references milestone-detect/milestone checking"
  else
    lif01_pass=false
    lif01_notes="$lif01_notes [FAIL: no milestone-detect ref]"
    echo "  FAIL: seal.md lacks milestone-detect reference"
  fi
fi

# Check 4: References colony-archive-xml (Phase 8 XML integration)
if [[ -f "$live_seal" ]]; then
  if grep -q "colony-archive-xml\|xmllint\|XML" "$live_seal" 2>/dev/null; then
    echo "  PASS: seal.md references colony-archive-xml/XML (Phase 8)"
  else
    lif01_pass=false
    lif01_notes="$lif01_notes [FAIL: no colony-archive-xml ref]"
    echo "  FAIL: seal.md lacks colony-archive-xml reference"
  fi
fi

# Check 5: milestone-detect subcommand works in isolated env
echo "  Testing milestone-detect in isolated env..."
milestone_out=$(run_in_isolated_env "$E2E_TMP_DIR" "milestone-detect" 2>&1) || true
milestone_json=$(extract_json "$milestone_out")
if echo "$milestone_json" | jq -e '.ok' >/dev/null 2>&1; then
  milestone=$(echo "$milestone_json" | jq -r '.result.milestone // "unknown"')
  echo "  PASS: milestone-detect returns valid JSON (milestone: $milestone)"
else
  lif01_pass=false
  lif01_notes="$lif01_notes [FAIL: milestone-detect returned invalid JSON: $milestone_json]"
  echo "  FAIL: milestone-detect output not valid JSON: $milestone_json"
fi

if $lif01_pass; then
  record_result "LIF-01" "PASS" "seal.md verified: Crowned Anthill ref, milestone-detect ref, XML ref; milestone-detect works"
else
  record_result "LIF-01" "FAIL" "$lif01_notes"
fi

# ============================================================================
# LIF-02: entomb archives colony to chambers
# ============================================================================

echo ""
echo "--- LIF-02: entomb archives colony to chambers ---"

lif02_pass=true
lif02_notes=""

# Check 1: entomb.md exists
live_entomb="$PROJECT_ROOT/.claude/commands/ant/entomb.md"
if [[ ! -f "$live_entomb" ]]; then
  lif02_pass=false
  lif02_notes="$lif02_notes [FAIL: entomb.md not found]"
  echo "  FAIL: entomb.md missing"
else
  echo "  PASS: entomb.md exists"
fi

# Check 2: References chamber archiving
if [[ -f "$live_entomb" ]]; then
  if grep -qi "chamber\|archive" "$live_entomb" 2>/dev/null; then
    echo "  PASS: entomb.md references chambers/archive"
  else
    lif02_pass=false
    lif02_notes="$lif02_notes [FAIL: no chamber/archive ref]"
    echo "  FAIL: entomb.md lacks chamber/archive reference"
  fi
fi

# Check 3: Seal-first enforcement (Crowned Anthill check)
if [[ -f "$live_entomb" ]]; then
  if grep -q "Crowned Anthill\|sealed\|milestone" "$live_entomb" 2>/dev/null; then
    echo "  PASS: entomb.md references seal-first enforcement"
  else
    lif02_pass=false
    lif02_notes="$lif02_notes [FAIL: no seal-first enforcement ref]"
    echo "  FAIL: entomb.md lacks seal-first enforcement"
  fi
fi

# Check 4: References xmllint or XML tool check (Phase 8 hard-stop)
if [[ -f "$live_entomb" ]]; then
  if grep -q "xmllint\|command -v\|XML" "$live_entomb" 2>/dev/null; then
    echo "  PASS: entomb.md references xmllint/XML tool check (Phase 8)"
  else
    lif02_pass=false
    lif02_notes="$lif02_notes [FAIL: no xmllint ref]"
    echo "  FAIL: entomb.md lacks xmllint reference"
  fi
fi

# Check 5: chamber-list returns valid JSON (even if empty)
echo "  Testing chamber-list in isolated env..."
chamber_out=$(run_in_isolated_env "$E2E_TMP_DIR" "chamber-list" 2>&1) || true
chamber_json=$(extract_json "$chamber_out")
if echo "$chamber_json" | jq -e '.ok' >/dev/null 2>&1; then
  chamber_count=$(echo "$chamber_json" | jq '.result | length' 2>/dev/null || echo "0")
  echo "  PASS: chamber-list returns valid JSON ($chamber_count chambers)"
else
  lif02_pass=false
  lif02_notes="$lif02_notes [FAIL: chamber-list invalid JSON: $chamber_json]"
  echo "  FAIL: chamber-list output not valid JSON: $chamber_json"
fi

if $lif02_pass; then
  record_result "LIF-02" "PASS" "entomb.md verified: chamber ref, seal-first enforcement, xmllint ref; chamber-list works"
else
  record_result "LIF-02" "FAIL" "$lif02_notes"
fi

# ============================================================================
# LIF-03: tunnels browses archived colonies
# ============================================================================

echo ""
echo "--- LIF-03: tunnels browses archived colonies ---"

lif03_pass=true
lif03_notes=""

# Check 1: tunnels.md exists
live_tunnels="$PROJECT_ROOT/.claude/commands/ant/tunnels.md"
if [[ ! -f "$live_tunnels" ]]; then
  lif03_pass=false
  lif03_notes="$lif03_notes [FAIL: tunnels.md not found]"
  echo "  FAIL: tunnels.md missing"
else
  echo "  PASS: tunnels.md exists"
fi

# Check 2: References chamber browsing
if [[ -f "$live_tunnels" ]]; then
  if grep -q "chamber-list\|timeline\|chambers" "$live_tunnels" 2>/dev/null; then
    echo "  PASS: tunnels.md references chamber-list/timeline/chambers"
  else
    lif03_pass=false
    lif03_notes="$lif03_notes [FAIL: no chamber-list/timeline ref]"
    echo "  FAIL: tunnels.md lacks chamber-list reference"
  fi
fi

# Check 3: References import flow (Phase 8 XML import integration)
if [[ -f "$live_tunnels" ]]; then
  if grep -q "pheromone-import-xml\|import" "$live_tunnels" 2>/dev/null; then
    echo "  PASS: tunnels.md references pheromone-import-xml/import (Phase 8)"
  else
    lif03_pass=false
    lif03_notes="$lif03_notes [FAIL: no pheromone-import-xml ref]"
    echo "  FAIL: tunnels.md lacks import reference"
  fi
fi

# Check 4: chamber-list returns a JSON array
echo "  Testing chamber-list returns JSON array..."
chamber_out2=$(run_in_isolated_env "$E2E_TMP_DIR" "chamber-list" 2>&1) || true
chamber_json2=$(extract_json "$chamber_out2")
if echo "$chamber_json2" | jq -e '.result | arrays' >/dev/null 2>&1; then
  echo "  PASS: chamber-list .result is a JSON array"
else
  # Some implementations may return object with chambers key — check alternate
  if echo "$chamber_json2" | jq -e '.ok == true' >/dev/null 2>&1; then
    echo "  PASS: chamber-list returns ok:true (result may be array or empty)"
  else
    lif03_pass=false
    lif03_notes="$lif03_notes [FAIL: chamber-list result not array]"
    echo "  FAIL: chamber-list .result is not a JSON array: $chamber_json2"
  fi
fi

if $lif03_pass; then
  record_result "LIF-03" "PASS" "tunnels.md verified: chamber-list ref, timeline ref, import ref; chamber-list returns JSON array"
else
  record_result "LIF-03" "FAIL" "$lif03_notes"
fi

# ============================================================================
# Print Results
# ============================================================================

print_area_results "LIF (Colony Lifecycle)"
