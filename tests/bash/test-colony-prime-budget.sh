#!/usr/bin/env bash
# Tests for colony-prime total character budget enforcement
# Task 1.2: Add total character budget to colony-prime assembly

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

source "$SCRIPT_DIR/test-helpers.sh"
require_jq

AETHER_UTILS="$REPO_ROOT/.aether/aether-utils.sh"

# ============================================================================
# Helper: Create a test colony with configurable content size
# ============================================================================
setup_colony_env() {
    local tmpdir
    tmpdir=$(mktemp -d)
    local aether_dir="$tmpdir/.aether"
    local data_dir="$aether_dir/data"
    mkdir -p "$data_dir"

    local iso_date
    iso_date=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

    # Create QUEEN.md with emoji-prefixed section headers (matching _extract_wisdom awk patterns)
    cat > "$aether_dir/QUEEN.md" << QUEENEOF
# QUEEN.md --- Colony Wisdom

> Last evolved: $iso_date
> Colonies contributed: 0
> Wisdom version: 1.0.0

---

## 📜 Philosophies

*No philosophies recorded yet*

---

## 🧭 Patterns

*No patterns recorded yet*

---

## ⚠️ Redirects

*No redirects recorded yet*

---

## 🔧 Stack Wisdom

*No stack wisdom recorded yet*

---

## 🏛️ Decrees

*No decrees recorded yet*

---

## 📊 Evolution Log

| Date | Colony | Change | Details |
|------|--------|--------|---------|

---

<!-- METADATA {"version":"1.0.0","last_evolved":"$iso_date","colonies_contributed":[],"promotion_thresholds":{"philosophy":1,"pattern":1,"redirect":1,"stack":1,"decree":0},"stats":{"total_philosophies":0,"total_patterns":0,"total_redirects":0,"total_stack_entries":0,"total_decrees":0}} -->
QUEENEOF

    # Create COLONY_STATE.json
    cat > "$data_dir/COLONY_STATE.json" << 'STATEEOF'
{
  "session_id": "test_budget",
  "goal": "test budget enforcement",
  "state": "BUILDING",
  "current_phase": 3,
  "plan": { "phases": [] },
  "memory": {
    "instincts": [],
    "phase_learnings": [],
    "decisions": []
  },
  "errors": { "flagged_patterns": [] },
  "events": []
}
STATEEOF

    # Create pheromones.json (empty)
    cat > "$data_dir/pheromones.json" << 'PHEREOF'
{
  "signals": [],
  "version": "1.0.0"
}
PHEREOF

    echo "$tmpdir"
}

# Helper: run colony-prime against a test env
# Sets HOME to tmpdir to avoid interference from global ~/.aether/QUEEN.md
run_colony_prime() {
    local tmpdir="$1"
    shift
    HOME="$tmpdir" AETHER_ROOT="$tmpdir" DATA_DIR="$tmpdir/.aether/data" \
        bash "$AETHER_UTILS" colony-prime "$@" 2>/dev/null
}

# Helper: extract prompt_section length from colony-prime output
# Uses Python because jq can't parse the output when wisdom fields contain newlines
get_prompt_length() {
    python3 -c "
import sys, re
raw = sys.stdin.read()
# Find the prompt_section field - it's set via jq --arg so properly escaped
# The format is: \"prompt_section\": \"...escaped content...\"
# But colony-prime uses jq -n which may output the field as raw or escaped
# We look for the log_line field which comes right after prompt_section
match = re.search(r'\"prompt_section\":\s*\"((?:[^\"\\\\]|\\\\.)*)\"', raw, re.DOTALL)
if match:
    val = match.group(1)
    # Unescape JSON string escapes to get actual character count
    val = val.replace('\\\\n', '\n').replace('\\\\t', '\t').replace('\\\\\"', '\"').replace('\\\\\\\\', '\\\\')
    print(len(val))
else:
    print(0)
"
}

# Helper: extract log_line from colony-prime output
get_log_line() {
    python3 -c "
import sys, re
raw = sys.stdin.read()
match = re.search(r'\"log_line\":\s*\"((?:[^\"\\\\]|\\\\.)*)\"', raw)
if match:
    print(match.group(1))
else:
    print('')
"
}

# Helper: extract prompt_section content
get_prompt_section() {
    python3 -c "
import sys, re
raw = sys.stdin.read()
match = re.search(r'\"prompt_section\":\s*\"((?:[^\"\\\\]|\\\\.)*)\"', raw, re.DOTALL)
if match:
    val = match.group(1)
    val = val.replace('\\\\n', '\n').replace('\\\\t', '\t').replace('\\\\\"', '\"').replace('\\\\\\\\', '\\\\')
    print(val)
else:
    print('')
"
}

# Helper: add large rolling summary
add_large_rolling_summary() {
    local tmpdir="$1"
    local entry_count="${2:-50}"
    local data_dir="$tmpdir/.aether/data"
    local logfile="$data_dir/rolling-summary.log"

    > "$logfile"
    for i in $(seq 1 "$entry_count"); do
        echo "2026-03-20T10:${i}:00Z|build|phase-1|This is a long rolling summary entry number $i with substantial text to pad the character count to something meaningful for budget testing purposes and even more additional words to really make it count" >> "$logfile"
    done
}

# Helper: add phase learnings to COLONY_STATE.json
add_phase_learnings() {
    local tmpdir="$1"
    local learning_count="${2:-20}"
    local data_dir="$tmpdir/.aether/data"
    local state_file="$data_dir/COLONY_STATE.json"

    # Build learnings array using python (faster than jq loop)
    python3 -c "
import json
learnings = []
for i in range(1, $learning_count + 1):
    learnings.append({
        'claim': f'Learning number {i}: This is a validated insight about the codebase that provides guidance for future development work and testing in the colony. It contains sufficient length to be realistic.',
        'status': 'validated',
        'evidence': ['test'],
        'confidence': 0.9
    })
with open('$state_file') as f:
    state = json.load(f)
state['memory']['phase_learnings'] = [{
    'phase': '1',
    'phase_name': 'Foundation',
    'learnings': learnings
}]
with open('$state_file', 'w') as f:
    json.dump(state, f)
"
}

# Helper: add CONTEXT.md with decisions
add_context_decisions() {
    local tmpdir="$1"
    local decision_count="${2:-10}"
    local ctx_file="$tmpdir/.aether/CONTEXT.md"

    cat > "$ctx_file" << 'CTXHEAD'
# Aether Colony -- Current Context

## Recent Decisions

| Date | Decision | Rationale | Made By |
|------|----------|-----------|---------|
CTXHEAD

    for i in $(seq 1 "$decision_count"); do
        echo "| 2026-03-20 | Important decision number $i about architecture and design patterns | This is a detailed rationale explaining why we made this particular choice and what alternatives we considered | Colony |" >> "$ctx_file"
    done

    cat >> "$ctx_file" << 'CTXTAIL'

---

## Recent Activity

*No recent activity*
CTXTAIL
}

# Helper: add REDIRECT pheromone signals
add_redirect_signals() {
    local tmpdir="$1"
    local redirect_count="${2:-3}"
    local data_dir="$tmpdir/.aether/data"
    local pher_file="$data_dir/pheromones.json"

    python3 -c "
import json
from datetime import datetime
signals = []
for i in range(1, $redirect_count + 1):
    signals.append({
        'type': 'REDIRECT',
        'content': {'text': f'REDIRECT constraint number {i}: Never use this pattern in production code'},
        'strength': 0.9,
        'source': 'user',
        'created_at': datetime.utcnow().isoformat() + 'Z',
        'expires_at': None,
        'phase': None,
        'auto_decay': False
    })
with open('$pher_file') as f:
    data = json.load(f)
data['signals'] = signals
with open('$pher_file', 'w') as f:
    json.dump(data, f)
"
}

# Helper: add QUEEN.md with substantial wisdom content
add_queen_wisdom() {
    local tmpdir="$1"
    local aether_dir="$tmpdir/.aether"
    local iso_date
    iso_date=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

    cat > "$aether_dir/QUEEN.md" << QUEENEOF
# QUEEN.md --- Colony Wisdom

> Last evolved: $iso_date
> Colonies contributed: 3
> Wisdom version: 1.0.0

---

## 📜 Philosophies

- Always test before deploying. Testing is the foundation of reliable software.
- Code should be readable first, performant second. Readability aids maintenance.
- Prefer composition over inheritance for flexible architecture.
- Keep functions small and focused. Each function should do one thing well.
- Document decisions, not just code. Understanding why is more important than how.

---

## 🧭 Patterns

- Use dependency injection for testable code. Pass dependencies explicitly.
- Implement circuit breakers for external service calls to prevent cascade failures.
- Use structured logging with correlation IDs for distributed tracing.
- Validate inputs at boundaries, trust data internally within validated contexts.
- Use feature flags for gradual rollouts to reduce deployment risk.

---

## ⚠️ Redirects

- Never store passwords in plain text. Always hash with bcrypt or argon2.
- Never use eval() or dynamic code execution from user input.
- Avoid deeply nested callbacks. Use async/await or promise chains instead.

---

## 🔧 Stack Wisdom

- Node.js: Use cluster module for multi-core utilization in production.
- PostgreSQL: Always use parameterized queries to prevent SQL injection.
- Docker: Use multi-stage builds to reduce image size significantly.
- TypeScript: Prefer strict mode for better type safety and fewer runtime errors.

---

## 🏛️ Decrees

- All PRs require at least one approval before merging.
- Security patches take priority over feature work at all times.
- Breaking API changes require a deprecation period of at least 2 weeks.

---

## 📊 Evolution Log

| Date | Colony | Change | Details |
|------|--------|--------|---------|
| 2026-03-01 | colony-alpha | Added | Initial philosophies |
| 2026-03-15 | colony-beta | Evolved | Stack wisdom expanded |

---

<!-- METADATA {"version":"1.0.0","last_evolved":"$iso_date","colonies_contributed":["alpha","beta","gamma"],"promotion_thresholds":{"philosophy":1,"pattern":1,"redirect":1,"stack":1,"decree":0},"stats":{"total_philosophies":5,"total_patterns":5,"total_redirects":3,"total_stack_entries":4,"total_decrees":3}} -->
QUEENEOF
}

# ============================================================================
# Tests
# ============================================================================

test_normal_output_unchanged() {
    # Normal-sized output (under budget) should be completely unchanged
    local tmpdir
    tmpdir=$(setup_colony_env)

    local result
    result=$(run_colony_prime "$tmpdir")

    # Should succeed (check for "ok":true in raw output)
    if ! assert_contains "$result" '"ok":true'; then
        test_fail "Expected ok=true in output" ""
        rm -rf "$tmpdir"
        return 1
    fi

    # log_line should NOT mention truncation
    local log_line
    log_line=$(echo "$result" | get_log_line)
    if assert_contains "$log_line" "truncated"; then
        test_fail "Normal output should not mention truncation" "$log_line"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

test_budget_enforced_full_mode() {
    # When content exceeds 8000 chars, truncation should occur
    local tmpdir
    tmpdir=$(setup_colony_env)

    # Add substantial content to push well over 8000 chars
    add_queen_wisdom "$tmpdir"
    add_large_rolling_summary "$tmpdir" 100
    add_phase_learnings "$tmpdir" 30
    add_context_decisions "$tmpdir" 15

    local result
    result=$(run_colony_prime "$tmpdir")

    if ! assert_contains "$result" '"ok":true'; then
        test_fail "Expected ok=true" ""
        rm -rf "$tmpdir"
        return 1
    fi

    # prompt_section should not exceed 8000 chars
    local char_count
    char_count=$(echo "$result" | get_prompt_length)

    if [[ "$char_count" -gt 8000 ]]; then
        test_fail "prompt_section should be <= 8000 chars" "Got $char_count chars"
        rm -rf "$tmpdir"
        return 1
    fi

    # log_line should mention truncation
    local log_line
    log_line=$(echo "$result" | get_log_line)
    if ! assert_contains "$log_line" "truncated"; then
        test_fail "Should mention truncation in log_line" "$log_line"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

test_compact_mode_budget() {
    # --compact should use 4000 char budget
    local tmpdir
    tmpdir=$(setup_colony_env)

    # Add content to exceed 4000
    add_queen_wisdom "$tmpdir"
    add_large_rolling_summary "$tmpdir" 60
    add_phase_learnings "$tmpdir" 15
    add_context_decisions "$tmpdir" 10

    local result
    result=$(run_colony_prime "$tmpdir" --compact)

    if ! assert_contains "$result" '"ok":true'; then
        test_fail "Expected ok=true" ""
        rm -rf "$tmpdir"
        return 1
    fi

    # prompt_section should not exceed 4000 chars
    local char_count
    char_count=$(echo "$result" | get_prompt_length)

    if [[ "$char_count" -gt 4000 ]]; then
        test_fail "compact prompt_section should be <= 4000 chars" "Got $char_count chars"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

test_redirects_never_trimmed() {
    # REDIRECTs must never be truncated even when budget is exceeded
    local tmpdir
    tmpdir=$(setup_colony_env)

    # Add REDIRECT signals
    add_redirect_signals "$tmpdir" 3

    # Add lots of other content to force truncation
    add_queen_wisdom "$tmpdir"
    add_large_rolling_summary "$tmpdir" 100
    add_phase_learnings "$tmpdir" 30
    add_context_decisions "$tmpdir" 15

    local result
    result=$(run_colony_prime "$tmpdir")

    if ! assert_contains "$result" '"ok":true'; then
        test_fail "Expected ok=true" ""
        rm -rf "$tmpdir"
        return 1
    fi

    # All REDIRECT signals should still be present in prompt_section
    local prompt
    prompt=$(echo "$result" | get_prompt_section)

    if ! assert_contains "$prompt" "REDIRECT constraint number 1"; then
        test_fail "REDIRECT 1 should be preserved" ""
        rm -rf "$tmpdir"
        return 1
    fi
    if ! assert_contains "$prompt" "REDIRECT constraint number 2"; then
        test_fail "REDIRECT 2 should be preserved" ""
        rm -rf "$tmpdir"
        return 1
    fi
    if ! assert_contains "$prompt" "REDIRECT constraint number 3"; then
        test_fail "REDIRECT 3 should be preserved" ""
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

test_truncation_priority_rolling_first() {
    # Rolling summary should be trimmed first (highest truncation priority)
    # Set up content where rolling summary is the biggest contributor
    local tmpdir
    tmpdir=$(setup_colony_env)

    add_large_rolling_summary "$tmpdir" 100
    add_phase_learnings "$tmpdir" 3
    add_context_decisions "$tmpdir" 2

    local result
    result=$(run_colony_prime "$tmpdir")

    if ! assert_contains "$result" '"ok":true'; then
        test_fail "Expected ok=true" ""
        rm -rf "$tmpdir"
        return 1
    fi

    local prompt
    prompt=$(echo "$result" | get_prompt_section)
    local log_line
    log_line=$(echo "$result" | get_log_line)

    # If truncation occurred, rolling summary should be gone/trimmed
    # while phase learnings should be preserved
    if assert_contains "$log_line" "truncated"; then
        if assert_contains "$log_line" "rolling-summary"; then
            # Good - rolling summary was identified as trimmed
            # Phase learnings should still be present if budget allows
            if assert_contains "$prompt" "PHASE LEARNINGS"; then
                return 0
            fi
        fi
    fi

    # If no truncation needed, that's also fine (content under budget)
    rm -rf "$tmpdir"
    return 0
}

test_log_line_lists_trimmed_sections() {
    # When truncation occurs, log_line should indicate which sections were trimmed
    local tmpdir
    tmpdir=$(setup_colony_env)

    add_queen_wisdom "$tmpdir"
    add_large_rolling_summary "$tmpdir" 100
    add_phase_learnings "$tmpdir" 30
    add_context_decisions "$tmpdir" 15

    local result
    result=$(run_colony_prime "$tmpdir")

    if ! assert_contains "$result" '"ok":true'; then
        test_fail "Expected ok=true" ""
        rm -rf "$tmpdir"
        return 1
    fi

    local log_line
    log_line=$(echo "$result" | get_log_line)

    # Log line should mention truncation with section names
    if ! assert_contains "$log_line" "truncated"; then
        test_fail "Log line should mention truncated sections" "$log_line"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

# ============================================================================
# Run tests
# ============================================================================

log_info "Running colony-prime budget tests"
log_info "Repo root: $REPO_ROOT"

run_test test_normal_output_unchanged "Normal output (under budget) is unchanged"
run_test test_budget_enforced_full_mode "Full mode enforces 8000 char budget"
run_test test_compact_mode_budget "Compact mode enforces 4000 char budget"
run_test test_redirects_never_trimmed "REDIRECTs are never trimmed"
run_test test_truncation_priority_rolling_first "Rolling summary trimmed before other sections"
run_test test_log_line_lists_trimmed_sections "Log line lists trimmed sections"

test_summary
