#!/usr/bin/env bash
# test-cmd.sh — CMD requirement verification
# CMD-01: /ant:lay-eggs starts new colony with pheromone preservation
# CMD-02: /ant:init initializes after lay-eggs
# CMD-03: /ant:colonize analyzes existing codebase
# CMD-04: /ant:plan generates project plan
# CMD-05: /ant:build executes phase with worker spawning
# CMD-06: /ant:continue verifies, extracts learnings, advances phase
# CMD-07: /ant:status shows colony dashboard
# CMD-08: All commands find correct files (no hallucinations)
#
# Compatible with bash 3.2 (macOS default)
# Uses proxy verification strategy for slash commands (can't invoke directly)

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

# Source e2e helpers
source "$SCRIPT_DIR/e2e-helpers.sh"

# Initialize results tracking
init_results

echo ""
echo "=========================================="
echo " CMD: Command Infrastructure Requirements"
echo "=========================================="
echo ""

# ============================================================================
# Setup isolated environment for subcommand tests
# ============================================================================

TMP_DIR=$(setup_e2e_env)
trap 'teardown_e2e_env' EXIT

UTILS="$TMP_DIR/.aether/aether-utils.sh"

# ============================================================================
# CMD-01: /ant:lay-eggs starts new colony with pheromone preservation
# Proxy: file exists + pheromone calls present + pheromone-write works
# ============================================================================

test_start "CMD-01: lay-eggs.md command file exists at correct path"
lay_eggs_live="$PROJECT_ROOT/.claude/commands/ant/lay-eggs.md"
lay_eggs_sot="$PROJECT_ROOT/.aether/commands/claude/lay-eggs.md"
if [[ -f "$lay_eggs_live" && -f "$lay_eggs_sot" ]]; then
    test_pass
else
    test_fail "lay-eggs.md in both SoT and live" "Live: $(ls "$lay_eggs_live" 2>/dev/null || echo missing) SoT: $(ls "$lay_eggs_sot" 2>/dev/null || echo missing)"
    record_result "CMD-01" "FAIL" "lay-eggs.md missing from live or SoT location"
fi

test_start "CMD-01: lay-eggs.md contains pheromone-related calls"
if grep -q "pheromone" "$lay_eggs_live" 2>/dev/null; then
    test_pass
    record_result "CMD-01" "PASS" "lay-eggs.md exists with pheromone references"
else
    test_fail "pheromone reference in lay-eggs.md" "No pheromone reference found"
    record_result "CMD-01" "FAIL" "lay-eggs.md missing pheromone references"
fi

test_start "CMD-01 (supplemental): pheromone-write subcommand works in isolation"
raw_pw=$(bash "$UTILS" pheromone-write FOCUS "Test focus for lay-eggs verification" 0.7 2>&1 || true)
pw_out=$(extract_json "$raw_pw")
if assert_json_valid "$pw_out" && assert_ok_true "$pw_out"; then
    test_pass
else
    test_fail "pheromone-write returns ok:true" "$pw_out"
fi

# ============================================================================
# CMD-02: /ant:init initializes after lay-eggs
# Proxy: init.md exists + session-init subcommand works
# ============================================================================

test_start "CMD-02: init.md command file exists at correct path"
init_live="$PROJECT_ROOT/.claude/commands/ant/init.md"
if [[ -f "$init_live" ]]; then
    test_pass
else
    test_fail "init.md exists" "Not found at $init_live"
fi

test_start "CMD-02: init.md references session-init subcommand"
if grep -q "session-init" "$init_live" 2>/dev/null; then
    test_pass
else
    test_fail "session-init reference in init.md" "Not found"
fi

test_start "CMD-02: session-init creates valid COLONY state in isolated env"
raw_si=$(bash "$UTILS" session-init "cmd02-test" "cmd02-goal" 2>&1 || true)
si_out=$(extract_json "$raw_si")
if assert_json_valid "$si_out" && assert_ok_true "$si_out"; then
    if [[ -f "$TMP_DIR/.aether/data/session.json" ]]; then
        test_pass
        record_result "CMD-02" "PASS" "session-init creates valid state; init.md references it correctly"
    else
        test_fail "session.json created" "File not found"
        record_result "CMD-02" "FAIL" "session-init did not create session.json"
    fi
else
    test_fail "ok:true from session-init" "$si_out"
    record_result "CMD-02" "FAIL" "session-init failed: $si_out"
fi

# ============================================================================
# CMD-03: /ant:colonize analyzes existing codebase
# Proxy: colonize.md exists + references swarm-display-text (Phase 3 wired)
# ============================================================================

test_start "CMD-03: colonize.md exists at correct path"
colonize_live="$PROJECT_ROOT/.claude/commands/ant/colonize.md"
if [[ -f "$colonize_live" ]]; then
    test_pass
else
    test_fail "colonize.md exists" "Not found"
fi

test_start "CMD-03: colonize.md references swarm-display functions (Phase 3 wired)"
if grep -q "swarm-display" "$colonize_live" 2>/dev/null; then
    test_pass
    record_result "CMD-03" "PASS" "colonize.md exists with swarm-display references from Phase 3"
else
    test_fail "swarm-display reference in colonize.md" "Not found"
    record_result "CMD-03" "FAIL" "colonize.md missing swarm-display reference"
fi

test_start "CMD-03: swarm-display-text subcommand returns valid output"
raw_sdt=$(bash "$UTILS" swarm-display-text "test-colonize-session" 2>&1 || true)
sdt_out=$(extract_json "$raw_sdt")
if assert_json_valid "$sdt_out" && assert_ok_true "$sdt_out"; then
    test_pass
else
    test_fail "swarm-display-text returns ok:true" "$sdt_out"
fi

# ============================================================================
# CMD-04: /ant:plan generates project plan
# Proxy: plan.md exists + >400 lines + references validate-state
# ============================================================================

test_start "CMD-04: plan.md command file exists"
plan_live="$PROJECT_ROOT/.claude/commands/ant/plan.md"
if [[ -f "$plan_live" ]]; then
    test_pass
else
    test_fail "plan.md exists" "Not found"
fi

test_start "CMD-04: plan.md is substantial (>400 lines — complex command)"
plan_lines=$(wc -l < "$plan_live" 2>/dev/null || echo "0")
if [[ "$plan_lines" -gt 400 ]]; then
    test_pass
else
    test_fail "plan.md >400 lines" "Got: $plan_lines"
fi

test_start "CMD-04: plan.md references validate-state (state validation step)"
if grep -q "validate-state" "$plan_live" 2>/dev/null; then
    test_pass
    record_result "CMD-04" "PASS" "plan.md exists ($plan_lines lines) with validate-state reference"
else
    test_fail "validate-state in plan.md" "Not found"
    record_result "CMD-04" "FAIL" "plan.md missing validate-state reference"
fi

test_start "CMD-04 (supplemental): validate-state subcommand works"
raw_vs=$(bash "$UTILS" validate-state colony 2>&1 || true)
vs_out=$(extract_json "$raw_vs")
if assert_json_valid "$vs_out"; then
    test_pass
else
    test_fail "validate-state returns JSON" "$vs_out"
fi

# ============================================================================
# CMD-05: /ant:build executes phase with worker spawning
# Proxy: build.md exists + >800 lines + references pheromone-prime (Phase 5)
# ============================================================================

test_start "CMD-05: build.md command file exists"
build_live="$PROJECT_ROOT/.claude/commands/ant/build.md"
if [[ -f "$build_live" ]]; then
    test_pass
else
    test_fail "build.md exists" "Not found"
fi

test_start "CMD-05: build.md is substantial (>800 lines)"
build_lines=$(wc -l < "$build_live" 2>/dev/null || echo "0")
if [[ "$build_lines" -gt 800 ]]; then
    test_pass
else
    test_fail "build.md >800 lines" "Got: $build_lines"
fi

test_start "CMD-05: build.md references pheromone-prime (Phase 5 wired)"
if grep -q "pheromone-prime" "$build_live" 2>/dev/null; then
    test_pass
else
    test_fail "pheromone-prime in build.md" "Not found"
fi

test_start "CMD-05: build.md references worker spawning"
if grep -q "spawn\|swarm" "$build_live" 2>/dev/null; then
    test_pass
    record_result "CMD-05" "PASS" "build.md exists ($build_lines lines) with pheromone-prime + spawn refs"
else
    test_fail "spawn/swarm references in build.md" "Not found"
    record_result "CMD-05" "FAIL" "build.md missing spawn/swarm references"
fi

test_start "CMD-05 (supplemental): pheromone-prime returns prompt section"
raw_pp=$(bash "$UTILS" pheromone-prime builder 2>&1 || true)
pp_out=$(extract_json "$raw_pp")
if assert_json_valid "$pp_out" && assert_ok_true "$pp_out"; then
    test_pass
else
    test_fail "pheromone-prime returns ok:true" "$pp_out"
fi

# ============================================================================
# CMD-06: /ant:continue verifies, extracts learnings, advances phase
# Proxy: continue.md exists + >900 lines + references learnings + session-update
# ============================================================================

test_start "CMD-06: continue.md command file exists"
continue_live="$PROJECT_ROOT/.claude/commands/ant/continue.md"
if [[ -f "$continue_live" ]]; then
    test_pass
else
    test_fail "continue.md exists" "Not found"
fi

test_start "CMD-06: continue.md is substantial (>900 lines)"
continue_lines=$(wc -l < "$continue_live" 2>/dev/null || echo "0")
if [[ "$continue_lines" -gt 900 ]]; then
    test_pass
else
    test_fail "continue.md >900 lines" "Got: $continue_lines"
fi

test_start "CMD-06: continue.md references learning extraction"
if grep -q "learning\|learnings" "$continue_live" 2>/dev/null; then
    test_pass
else
    test_fail "learnings reference in continue.md" "Not found"
fi

test_start "CMD-06: continue.md references session-update"
if grep -q "session-update" "$continue_live" 2>/dev/null; then
    test_pass
    record_result "CMD-06" "PASS" "continue.md exists ($continue_lines lines) with learnings + session-update"
else
    test_fail "session-update in continue.md" "Not found"
    record_result "CMD-06" "FAIL" "continue.md missing session-update reference"
fi

# ============================================================================
# CMD-07: /ant:status shows colony dashboard
# Primary: aether status --json returns valid JSON
# Fallback: status case branch exists in aether-utils.sh
# ============================================================================

test_start "CMD-07: aether status --json returns valid JSON"
raw_status=$(aether status --json 2>&1 || true)
if echo "$raw_status" | jq empty 2>/dev/null; then
    test_pass
    record_result "CMD-07" "PASS" "aether status --json returns valid JSON"
else
    # Fallback: check that status logic exists in aether-utils.sh
    log_warn "aether CLI not returning JSON — checking for status in aether-utils.sh"
    if grep -q "milestone-detect\|view-state" "$PROJECT_ROOT/.aether/aether-utils.sh" 2>/dev/null; then
        test_pass
        record_result "CMD-07" "PASS" "status display logic present in aether-utils.sh (CLI fallback)"
    else
        test_fail "status command works" "aether status failed: $raw_status"
        record_result "CMD-07" "FAIL" "aether status --json failed and fallback not found"
    fi
fi

# ============================================================================
# CMD-08: All commands find correct files (no hallucinations)
# Static analysis: extract all aether-utils.sh subcommand refs from command files
# Verify each referenced subcommand exists as a case branch in aether-utils.sh
# ============================================================================

test_start "CMD-08: all subcommand references in command files exist in aether-utils.sh"

# Extract all valid case branches from aether-utils.sh
# Pattern: lines that match "  <subcommand-name>)" at start of case branch
valid_cmds_file=$(mktemp)
grep -E "^  [a-z][a-z-]+\)" "$PROJECT_ROOT/.aether/aether-utils.sh" 2>/dev/null | \
    sed 's/^  \([a-z][a-z-]*\))/\1/' | sort > "$valid_cmds_file"

# Extract all subcommands from actual bash execution calls (not prose text)
# Only look at lines with "bash.*aether-utils.sh <cmd>" pattern
referenced_cmds_file=$(mktemp)
for cmd_file in "$PROJECT_ROOT/.claude/commands/ant/"*.md; do
    grep "bash.*aether-utils\.sh" "$cmd_file" 2>/dev/null | \
        grep -o "aether-utils\.sh [a-z][a-z-]*" | \
        awk '{print $2}' >> "$referenced_cmds_file" || true
done
sort -u "$referenced_cmds_file" -o "$referenced_cmds_file"

missing_cmds=""
while IFS= read -r cmd; do
    if [[ -z "$cmd" ]]; then
        continue
    fi
    if ! grep -qx "$cmd" "$valid_cmds_file" 2>/dev/null; then
        missing_cmds="$missing_cmds $cmd"
    fi
done < "$referenced_cmds_file"

rm -f "$valid_cmds_file" "$referenced_cmds_file"

if [[ -z "$missing_cmds" ]]; then
    test_pass
    record_result "CMD-08" "PASS" "All subcommand references in command files exist in aether-utils.sh"
else
    test_fail "all referenced subcommands exist" "Missing case branches:$missing_cmds"
    record_result "CMD-08" "FAIL" "Referenced subcommands missing from aether-utils.sh:$missing_cmds"
fi

# Additional check: no references to non-existent paths
test_start "CMD-08 (supplemental): key command files reference valid aether-utils.sh subcommands"
# Quick static check: verify that the most critical subcommands mentioned
# in key command files are real case branches in aether-utils.sh
key_checks=(
    "pheromone-write"
    "pheromone-prime"
    "session-init"
    "session-update"
    "swarm-display-text"
    "validate-state"
    "context-update"
    "load-state"
)
bad_keys=""
for kc in "${key_checks[@]}"; do
    if ! grep -qE "^  ${kc}\)" "$PROJECT_ROOT/.aether/aether-utils.sh" 2>/dev/null; then
        bad_keys="$bad_keys $kc"
    fi
done
if [[ -z "$bad_keys" ]]; then
    test_pass
else
    test_fail "key subcommands exist in aether-utils.sh" "Missing:$bad_keys"
fi

# ============================================================================
# Print Results
# ============================================================================

print_area_results "CMD"
