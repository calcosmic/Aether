#!/usr/bin/env bash
# test-adv.sh — Advanced Workers area requirements (ADV-01 through ADV-05)
# Proxy-verifies oracle, chaos, archaeology, dream, and interpret command files.
# Cross-copy parity check for SoT vs live command files.
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
echo "ADV Area: Advanced Workers Requirements"
echo "================================================================"

# ============================================================================
# Environment Setup
# ============================================================================

E2E_TMP_DIR=$(setup_e2e_env)
trap teardown_e2e_env EXIT

init_results

# ============================================================================
# Helper: parity check (SoT vs live copy line count within 20%)
# Returns 0 if parity ok, 1 if mismatch
# Usage: check_copy_parity "name" "sot_path" "live_path"
# ============================================================================

check_copy_parity() {
  local name="$1"
  local sot_path="$2"
  local live_path="$3"

  if [[ ! -f "$sot_path" ]]; then
    echo "  FAIL: SoT copy missing: $sot_path"
    return 1
  fi
  if [[ ! -f "$live_path" ]]; then
    echo "  FAIL: live copy missing: $live_path"
    return 1
  fi

  local sot_lines live_lines
  sot_lines=$(wc -l < "$sot_path" | tr -d ' ')
  live_lines=$(wc -l < "$live_path" | tr -d ' ')

  # Within 20% — calculate 80% of larger
  local larger smaller
  if [[ "$sot_lines" -ge "$live_lines" ]]; then
    larger="$sot_lines"
    smaller="$live_lines"
  else
    larger="$live_lines"
    smaller="$sot_lines"
  fi

  # 80% threshold: smaller >= larger * 0.8 => smaller * 10 >= larger * 8
  if [[ "$((smaller * 10))" -ge "$((larger * 8))" ]]; then
    echo "  PASS: $name parity ok (SoT: $sot_lines, live: $live_lines lines)"
    return 0
  else
    echo "  FAIL: $name parity mismatch (SoT: $sot_lines, live: $live_lines lines)"
    return 1
  fi
}

# ============================================================================
# ADV-01: oracle runs RALF research loop
# ============================================================================

echo ""
echo "--- ADV-01: oracle runs RALF research loop ---"

adv01_pass=true
adv01_notes=""

# Check 1: oracle.md exists
live_oracle="$PROJECT_ROOT/.claude/commands/ant/oracle.md"
if [[ ! -f "$live_oracle" ]]; then
  adv01_pass=false
  adv01_notes="$adv01_notes [FAIL: oracle.md not found]"
  echo "  FAIL: oracle.md missing"
else
  echo "  PASS: oracle.md exists"
fi

# Check 2: References RALF or research loop
if [[ -f "$live_oracle" ]]; then
  if grep -qi "RALF\|research\|loop\|iteration" "$live_oracle" 2>/dev/null; then
    echo "  PASS: oracle.md references RALF/research loop"
  else
    adv01_pass=false
    adv01_notes="$adv01_notes [FAIL: no RALF/research ref]"
    echo "  FAIL: oracle.md lacks RALF/research reference"
  fi
fi

# Check 3: References oracle subcommand calls or session tracking
if [[ -f "$live_oracle" ]]; then
  if grep -qi "oracle\|session\|research" "$live_oracle" 2>/dev/null; then
    echo "  PASS: oracle.md references oracle/session/research subcommand calls"
  else
    adv01_pass=false
    adv01_notes="$adv01_notes [FAIL: no oracle/session ref]"
    echo "  FAIL: oracle.md lacks oracle/session reference"
  fi
fi

# Check 4: SoT copy exists
sot_oracle="$PROJECT_ROOT/.aether/commands/claude/oracle.md"
if [[ -f "$sot_oracle" ]]; then
  echo "  PASS: SoT copy exists at .aether/commands/claude/oracle.md"
else
  adv01_pass=false
  adv01_notes="$adv01_notes [FAIL: SoT copy missing]"
  echo "  FAIL: SoT oracle.md missing"
fi

# Check 5: SoT/live parity check
if ! check_copy_parity "oracle" "$sot_oracle" "$live_oracle"; then
  adv01_pass=false
  adv01_notes="$adv01_notes [FAIL: parity mismatch]"
fi

# Check 6: oracle.sh exists at .aether/oracle/ (not utils/ — per oracle.md references)
oracle_sh="$PROJECT_ROOT/.aether/oracle/oracle.sh"
if [[ -f "$oracle_sh" ]]; then
  echo "  PASS: oracle.sh exists at .aether/oracle/oracle.sh"
else
  adv01_pass=false
  adv01_notes="$adv01_notes [FAIL: .aether/oracle/oracle.sh missing]"
  echo "  FAIL: oracle.sh not found at .aether/oracle/oracle.sh"
fi

if $adv01_pass; then
  record_result "ADV-01" "PASS" "oracle.md verified: RALF ref, oracle.sh exists, SoT copy present, parity ok"
else
  record_result "ADV-01" "FAIL" "$adv01_notes"
fi

# ============================================================================
# ADV-02: chaos runs resilience tests
# ============================================================================

echo ""
echo "--- ADV-02: chaos runs resilience tests ---"

adv02_pass=true
adv02_notes=""

# Check 1: chaos.md exists
live_chaos="$PROJECT_ROOT/.claude/commands/ant/chaos.md"
if [[ ! -f "$live_chaos" ]]; then
  adv02_pass=false
  adv02_notes="$adv02_notes [FAIL: chaos.md not found]"
  echo "  FAIL: chaos.md missing"
else
  echo "  PASS: chaos.md exists"
fi

# Check 2: References chaos/resilience/edge case testing
if [[ -f "$live_chaos" ]]; then
  if grep -qi "chaos\|resilience\|edge\|boundary" "$live_chaos" 2>/dev/null; then
    echo "  PASS: chaos.md references chaos/resilience/edge case testing"
  else
    adv02_pass=false
    adv02_notes="$adv02_notes [FAIL: no resilience/edge ref]"
    echo "  FAIL: chaos.md lacks resilience/edge reference"
  fi
fi

# Check 3: References swarm-display-inline (per Phase 7 decision — Claude Code copies use inline)
if [[ -f "$live_chaos" ]]; then
  if grep -q "swarm-display-inline" "$live_chaos" 2>/dev/null; then
    echo "  PASS: chaos.md uses swarm-display-inline (Phase 7 decision)"
  else
    adv02_pass=false
    adv02_notes="$adv02_notes [FAIL: swarm-display-inline not found (Phase 7 requirement)]"
    echo "  FAIL: chaos.md does not use swarm-display-inline"
  fi
fi

# Check 4: SoT copy exists
sot_chaos="$PROJECT_ROOT/.aether/commands/claude/chaos.md"
if [[ -f "$sot_chaos" ]]; then
  echo "  PASS: SoT copy exists at .aether/commands/claude/chaos.md"
else
  adv02_pass=false
  adv02_notes="$adv02_notes [FAIL: SoT copy missing]"
  echo "  FAIL: SoT chaos.md missing"
fi

# Check 5: SoT/live parity check
if ! check_copy_parity "chaos" "$sot_chaos" "$live_chaos"; then
  adv02_pass=false
  adv02_notes="$adv02_notes [FAIL: parity mismatch]"
fi

if $adv02_pass; then
  record_result "ADV-02" "PASS" "chaos.md verified: resilience ref, swarm-display-inline, SoT copy, parity ok"
else
  record_result "ADV-02" "FAIL" "$adv02_notes"
fi

# ============================================================================
# ADV-03: archaeology analyzes git history
# ============================================================================

echo ""
echo "--- ADV-03: archaeology analyzes git history ---"

adv03_pass=true
adv03_notes=""

# Check 1: archaeology.md exists
live_arch="$PROJECT_ROOT/.claude/commands/ant/archaeology.md"
if [[ ! -f "$live_arch" ]]; then
  adv03_pass=false
  adv03_notes="$adv03_notes [FAIL: archaeology.md not found]"
  echo "  FAIL: archaeology.md missing"
else
  echo "  PASS: archaeology.md exists"
fi

# Check 2: References git history/blame/analysis
if [[ -f "$live_arch" ]]; then
  if grep -qi "git\|history\|archaeology\|excavat\|blame" "$live_arch" 2>/dev/null; then
    echo "  PASS: archaeology.md references git/history analysis"
  else
    adv03_pass=false
    adv03_notes="$adv03_notes [FAIL: no git/history ref]"
    echo "  FAIL: archaeology.md lacks git/history reference"
  fi
fi

# Check 3: References swarm-display-inline (Phase 7 sync decision)
if [[ -f "$live_arch" ]]; then
  if grep -q "swarm-display-inline" "$live_arch" 2>/dev/null; then
    echo "  PASS: archaeology.md uses swarm-display-inline (Phase 7 decision)"
  else
    adv03_pass=false
    adv03_notes="$adv03_notes [FAIL: swarm-display-inline not found]"
    echo "  FAIL: archaeology.md does not use swarm-display-inline"
  fi
fi

# Check 4: SoT copy exists
sot_arch="$PROJECT_ROOT/.aether/commands/claude/archaeology.md"
if [[ -f "$sot_arch" ]]; then
  echo "  PASS: SoT copy exists at .aether/commands/claude/archaeology.md"
else
  adv03_pass=false
  adv03_notes="$adv03_notes [FAIL: SoT copy missing]"
  echo "  FAIL: SoT archaeology.md missing"
fi

# Check 5: SoT/live parity check
if ! check_copy_parity "archaeology" "$sot_arch" "$live_arch"; then
  adv03_pass=false
  adv03_notes="$adv03_notes [FAIL: parity mismatch]"
fi

if $adv03_pass; then
  record_result "ADV-03" "PASS" "archaeology.md verified: git/history ref, swarm-display-inline, SoT copy, parity ok"
else
  record_result "ADV-03" "FAIL" "$adv03_notes"
fi

# ============================================================================
# ADV-04: dream writes wisdom to dream journal
# ============================================================================

echo ""
echo "--- ADV-04: dream writes wisdom to dream journal ---"

adv04_pass=true
adv04_notes=""

# Check 1: dream.md exists
live_dream="$PROJECT_ROOT/.claude/commands/ant/dream.md"
if [[ ! -f "$live_dream" ]]; then
  adv04_pass=false
  adv04_notes="$adv04_notes [FAIL: dream.md not found]"
  echo "  FAIL: dream.md missing"
else
  echo "  PASS: dream.md exists"
fi

# Check 2: References dream/wisdom/philosophical observations
if [[ -f "$live_dream" ]]; then
  if grep -qi "dream\|wisdom\|philosophi\|observe\|wander" "$live_dream" 2>/dev/null; then
    echo "  PASS: dream.md references dream/wisdom/philosophical content"
  else
    adv04_pass=false
    adv04_notes="$adv04_notes [FAIL: no dream/wisdom ref]"
    echo "  FAIL: dream.md lacks dream/wisdom reference"
  fi
fi

# Check 3: References .aether/dreams/ path for output
if [[ -f "$live_dream" ]]; then
  if grep -q ".aether/dreams\|dreams/" "$live_dream" 2>/dev/null; then
    echo "  PASS: dream.md references .aether/dreams/ output path"
  else
    adv04_pass=false
    adv04_notes="$adv04_notes [FAIL: no .aether/dreams/ ref]"
    echo "  FAIL: dream.md lacks .aether/dreams/ path reference"
  fi
fi

# Check 4: SoT copy exists
sot_dream="$PROJECT_ROOT/.aether/commands/claude/dream.md"
if [[ -f "$sot_dream" ]]; then
  echo "  PASS: SoT copy exists at .aether/commands/claude/dream.md"
else
  adv04_pass=false
  adv04_notes="$adv04_notes [FAIL: SoT copy missing]"
  echo "  FAIL: SoT dream.md missing"
fi

# Check 5: SoT/live parity check
if ! check_copy_parity "dream" "$sot_dream" "$live_dream"; then
  adv04_pass=false
  adv04_notes="$adv04_notes [FAIL: parity mismatch]"
fi

if $adv04_pass; then
  record_result "ADV-04" "PASS" "dream.md verified: wisdom/dream ref, .aether/dreams/ path, SoT copy, parity ok"
else
  record_result "ADV-04" "FAIL" "$adv04_notes"
fi

# ============================================================================
# ADV-05: interpret validates dream entries
# ============================================================================

echo ""
echo "--- ADV-05: interpret validates dream entries ---"

adv05_pass=true
adv05_notes=""

# Check 1: interpret.md exists
live_interp="$PROJECT_ROOT/.claude/commands/ant/interpret.md"
if [[ ! -f "$live_interp" ]]; then
  adv05_pass=false
  adv05_notes="$adv05_notes [FAIL: interpret.md not found]"
  echo "  FAIL: interpret.md missing"
else
  echo "  PASS: interpret.md exists"
fi

# Check 2: References interpret/dream/validate
if [[ -f "$live_interp" ]]; then
  if grep -qi "interpret\|dream\|validate\|ground\|feasib" "$live_interp" 2>/dev/null; then
    echo "  PASS: interpret.md references interpret/dream/validation"
  else
    adv05_pass=false
    adv05_notes="$adv05_notes [FAIL: no interpret/dream/validate ref]"
    echo "  FAIL: interpret.md lacks interpret/dream/validate reference"
  fi
fi

# Check 3: SoT copy exists
sot_interp="$PROJECT_ROOT/.aether/commands/claude/interpret.md"
if [[ -f "$sot_interp" ]]; then
  echo "  PASS: SoT copy exists at .aether/commands/claude/interpret.md"
else
  adv05_pass=false
  adv05_notes="$adv05_notes [FAIL: SoT copy missing]"
  echo "  FAIL: SoT interpret.md missing"
fi

# Check 4: SoT/live parity check
if ! check_copy_parity "interpret" "$sot_interp" "$live_interp"; then
  adv05_pass=false
  adv05_notes="$adv05_notes [FAIL: parity mismatch]"
fi

# Check 5: OpenCode copy exists (Phase 7-03 added it)
oc_interp="$PROJECT_ROOT/.opencode/commands/ant/interpret.md"
if [[ -f "$oc_interp" ]]; then
  echo "  PASS: OpenCode copy exists at .opencode/commands/ant/interpret.md (Phase 7-03)"
else
  adv05_pass=false
  adv05_notes="$adv05_notes [FAIL: OpenCode interpret.md missing]"
  echo "  FAIL: OpenCode interpret.md missing at .opencode/commands/ant/"
fi

if $adv05_pass; then
  record_result "ADV-05" "PASS" "interpret.md verified: dream/validate ref, SoT copy, parity ok, OpenCode copy present"
else
  record_result "ADV-05" "FAIL" "$adv05_notes"
fi

# ============================================================================
# Print Results
# ============================================================================

print_area_results "ADV (Advanced Workers)"
