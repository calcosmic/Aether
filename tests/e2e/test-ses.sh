#!/usr/bin/env bash
# test-ses.sh — Session Management area requirements (SES-01 through SES-03)
# Proxy-verifies pause-colony, resume-colony, and watch command files.
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
echo "SES Area: Session Management Requirements"
echo "================================================================"

# ============================================================================
# Environment Setup
# ============================================================================

E2E_TMP_DIR=$(setup_e2e_env)
trap teardown_e2e_env EXIT

init_results

# ============================================================================
# SES-01: pause-colony saves state
# ============================================================================

echo ""
echo "--- SES-01: pause-colony saves state ---"

ses01_pass=true
ses01_notes=""

# Check 1: pause-colony.md exists in live path
live_pause="$PROJECT_ROOT/.claude/commands/ant/pause-colony.md"
if [[ ! -f "$live_pause" ]]; then
  ses01_pass=false
  ses01_notes="$ses01_notes [FAIL: pause-colony.md not found at .claude/commands/ant/pause-colony.md]"
  echo "  FAIL: pause-colony.md missing"
else
  echo "  PASS: pause-colony.md exists"
fi

# Check 2: References state saving concepts
if [[ -f "$live_pause" ]]; then
  if grep -q "session\|COLONY_STATE\|state\|handoff" "$live_pause" 2>/dev/null; then
    echo "  PASS: pause-colony.md references state/session"
  else
    ses01_pass=false
    ses01_notes="$ses01_notes [FAIL: no session/state reference]"
    echo "  FAIL: pause-colony.md lacks session/state reference"
  fi
fi

# Check 3: Substantial content (>100 lines)
if [[ -f "$live_pause" ]]; then
  line_count=$(wc -l < "$live_pause" | tr -d ' ')
  if [[ "$line_count" -gt 100 ]]; then
    echo "  PASS: pause-colony.md has $line_count lines (>100)"
  else
    ses01_pass=false
    ses01_notes="$ses01_notes [FAIL: only $line_count lines, need >100]"
    echo "  FAIL: pause-colony.md has only $line_count lines"
  fi
fi

# Check 4: SoT copy exists
sot_pause="$PROJECT_ROOT/.aether/commands/claude/pause-colony.md"
if [[ -f "$sot_pause" ]]; then
  echo "  PASS: SoT copy at .aether/commands/claude/pause-colony.md exists"
else
  ses01_pass=false
  ses01_notes="$ses01_notes [FAIL: SoT copy missing]"
  echo "  FAIL: SoT pause-colony.md missing at .aether/commands/claude/"
fi

# Check 5: References HANDOFF.md write (state persistence mechanism)
if [[ -f "$live_pause" ]]; then
  if grep -q "HANDOFF\|COLONY_STATE" "$live_pause" 2>/dev/null; then
    echo "  PASS: pause-colony.md references HANDOFF/COLONY_STATE"
  else
    ses01_pass=false
    ses01_notes="$ses01_notes [FAIL: no HANDOFF or COLONY_STATE reference]"
    echo "  FAIL: pause-colony.md lacks HANDOFF/COLONY_STATE reference"
  fi
fi

if $ses01_pass; then
  record_result "SES-01" "PASS" "pause-colony.md verified: exists, references state, >100 lines, SoT copy present"
else
  record_result "SES-01" "FAIL" "$ses01_notes"
fi

# ============================================================================
# SES-02: resume-colony restores context
# ============================================================================

echo ""
echo "--- SES-02: resume-colony restores context ---"

ses02_pass=true
ses02_notes=""

# Check 1: resume-colony.md exists
live_resume="$PROJECT_ROOT/.claude/commands/ant/resume-colony.md"
if [[ ! -f "$live_resume" ]]; then
  ses02_pass=false
  ses02_notes="$ses02_notes [FAIL: resume-colony.md not found]"
  echo "  FAIL: resume-colony.md missing"
else
  echo "  PASS: resume-colony.md exists"
fi

# Check 2: References COLONY_STATE.json reading
if [[ -f "$live_resume" ]]; then
  if grep -q "COLONY_STATE" "$live_resume" 2>/dev/null; then
    echo "  PASS: resume-colony.md references COLONY_STATE"
  else
    ses02_pass=false
    ses02_notes="$ses02_notes [FAIL: no COLONY_STATE reference]"
    echo "  FAIL: resume-colony.md lacks COLONY_STATE reference"
  fi
fi

# Check 3: References context restoration logic
if [[ -f "$live_resume" ]]; then
  if grep -q "session\|restore\|resume\|load-state" "$live_resume" 2>/dev/null; then
    echo "  PASS: resume-colony.md references session/restore/load-state"
  else
    ses02_pass=false
    ses02_notes="$ses02_notes [FAIL: no session/restore reference]"
    echo "  FAIL: resume-colony.md lacks session/restore reference"
  fi
fi

# Check 4: SoT copy exists
sot_resume="$PROJECT_ROOT/.aether/commands/claude/resume-colony.md"
if [[ -f "$sot_resume" ]]; then
  echo "  PASS: SoT copy at .aether/commands/claude/resume-colony.md exists"
else
  ses02_pass=false
  ses02_notes="$ses02_notes [FAIL: SoT copy missing]"
  echo "  FAIL: SoT resume-colony.md missing"
fi

# Check 5: References paused flag clearing (state restoration)
if [[ -f "$live_resume" ]]; then
  if grep -q "paused\|unload-state\|Clear\|clear\|Remove" "$live_resume" 2>/dev/null; then
    echo "  PASS: resume-colony.md references paused flag handling"
  else
    ses02_pass=false
    ses02_notes="$ses02_notes [FAIL: no paused/unload-state reference]"
    echo "  FAIL: resume-colony.md lacks paused/unload-state reference"
  fi
fi

if $ses02_pass; then
  record_result "SES-02" "PASS" "resume-colony.md verified: exists, COLONY_STATE ref, restoration logic, SoT copy"
else
  record_result "SES-02" "FAIL" "$ses02_notes"
fi

# ============================================================================
# SES-03: watch shows live colony visibility
# ============================================================================

echo ""
echo "--- SES-03: watch shows live colony visibility ---"

ses03_pass=true
ses03_notes=""

# Check 1: watch.md exists
live_watch="$PROJECT_ROOT/.claude/commands/ant/watch.md"
if [[ ! -f "$live_watch" ]]; then
  ses03_pass=false
  ses03_notes="$ses03_notes [FAIL: watch.md not found]"
  echo "  FAIL: watch.md missing"
else
  echo "  PASS: watch.md exists"
fi

# Check 2: References display functions
if [[ -f "$live_watch" ]]; then
  if grep -q "display\|watch-status\|activity\|tmux\|status" "$live_watch" 2>/dev/null; then
    echo "  PASS: watch.md references display/activity functions"
  else
    ses03_pass=false
    ses03_notes="$ses03_notes [FAIL: no display reference]"
    echo "  FAIL: watch.md lacks display reference"
  fi
fi

# Check 3: Substantial content (>100 lines)
if [[ -f "$live_watch" ]]; then
  line_count=$(wc -l < "$live_watch" | tr -d ' ')
  if [[ "$line_count" -gt 100 ]]; then
    echo "  PASS: watch.md has $line_count lines (>100)"
  else
    ses03_pass=false
    ses03_notes="$ses03_notes [FAIL: only $line_count lines, need >100]"
    echo "  FAIL: watch.md has only $line_count lines"
  fi
fi

# Check 4: SoT copy exists
sot_watch="$PROJECT_ROOT/.aether/commands/claude/watch.md"
if [[ -f "$sot_watch" ]]; then
  echo "  PASS: SoT copy at .aether/commands/claude/watch.md exists"
else
  # watch.md may not have a SoT copy — check
  ses03_pass=false
  ses03_notes="$ses03_notes [FAIL: SoT copy missing]"
  echo "  FAIL: SoT watch.md missing"
fi

# Check 5: References tmux session management (live visibility mechanism)
if [[ -f "$live_watch" ]]; then
  if grep -q "tmux\|aether-colony" "$live_watch" 2>/dev/null; then
    echo "  PASS: watch.md references tmux session management"
  else
    ses03_pass=false
    ses03_notes="$ses03_notes [FAIL: no tmux reference]"
    echo "  FAIL: watch.md lacks tmux reference"
  fi
fi

if $ses03_pass; then
  record_result "SES-03" "PASS" "watch.md verified: exists, display refs, >100 lines, SoT copy, tmux ref"
else
  record_result "SES-03" "FAIL" "$ses03_notes"
fi

# ============================================================================
# Print Results
# ============================================================================

print_area_results "SES (Session Management)"
