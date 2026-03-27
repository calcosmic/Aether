#!/usr/bin/env bash
# Intelligence Sub-Scan Integration Tests
# Tests the three intelligence sub-scan functions from scan.sh:
# - _scan_colony_context (INTEL-01): prior colony summaries + existing charter
# - _scan_pheromone_suggestions (INTEL-02): deterministic pattern-to-signal mapping
# - _scan_governance (INTEL-03): prescriptive governance rules from config detection

set -euo pipefail

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
AETHER_UTILS_SOURCE="$PROJECT_ROOT/.aether/aether-utils.sh"
QUEEN_TEMPLATE="$PROJECT_ROOT/.aether/templates/QUEEN.md.template"

# Source test helpers
source "$SCRIPT_DIR/test-helpers.sh"

# Verify jq is available
require_jq

# Verify aether-utils.sh exists
if [[ ! -f "$AETHER_UTILS_SOURCE" ]]; then
    log_error "aether-utils.sh not found at: $AETHER_UTILS_SOURCE"
    exit 1
fi

if [[ ! -f "$QUEEN_TEMPLATE" ]]; then
    log_error "QUEEN.md.template not found at: $QUEEN_TEMPLATE"
    exit 1
fi

# ============================================================================
# Helper: Create isolated intelligence test environment
# Sets up a temp dir with aether-utils.sh, utils, exchange, QUEEN.md, COLONY_STATE
# ============================================================================
setup_intelligence_env() {
    local test_dir
    test_dir=$(mktemp -d)
    mkdir -p "$test_dir/.aether/data" "$test_dir/.aether/utils" "$test_dir/.aether/chambers"

    cp "$AETHER_UTILS_SOURCE" "$test_dir/.aether/aether-utils.sh"
    chmod +x "$test_dir/.aether/aether-utils.sh"

    local utils_source="$(dirname "$AETHER_UTILS_SOURCE")/utils"
    if [[ -d "$utils_source" ]]; then
        cp -r "$utils_source" "$test_dir/.aether/"
    fi

    local exchange_source="$(dirname "$AETHER_UTILS_SOURCE")/exchange"
    if [[ -d "$exchange_source" ]]; then
        cp -r "$exchange_source" "$test_dir/.aether/"
    fi

    # Copy QUEEN.md template
    cp "$QUEEN_TEMPLATE" "$test_dir/.aether/QUEEN.md"
    local ts
    ts=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
    sed -i.bak "s/{TIMESTAMP}/$ts/g" "$test_dir/.aether/QUEEN.md" && rm -f "${test_dir}/.aether/QUEEN.md.bak"

    # Write a minimal valid COLONY_STATE.json
    cat > "$test_dir/.aether/data/COLONY_STATE.json" << EOF
{
  "version": "3.0",
  "goal": "test goal",
  "state": "READY",
  "colony_name": null,
  "current_phase": 0,
  "session_id": "session_1234_test",
  "initialized_at": "2026-03-27T00:00:00Z",
  "plan": { "phases": [] },
  "memory": { "phase_learnings": [], "decisions": [], "instincts": [] },
  "errors": { "records": [], "flagged_patterns": [] },
  "events": [],
  "signals": [],
  "graveyards": []
}
EOF

    echo "$test_dir"
}

# Run a scan function against a temp directory
# Creates a minimal shim that defines required dependencies and sources scan.sh directly
# (sourcing aether-utils.sh triggers its main dispatch, so we bypass it)
run_scan_func() {
    local tmp_dir="$1"
    local func_name="$2"
    local target_dir="${3:-$tmp_dir}"

    AETHER_ROOT="$tmp_dir" DATA_DIR="$tmp_dir/.aether/data" \
        bash -c '
            set -uo pipefail
            SCRIPT_DIR="'"$tmp_dir"'/.aether"
            AETHER_ROOT="'"$tmp_dir"'"
            DATA_DIR="'"$tmp_dir"'/.aether/data"

            # Define minimal dependencies that scan.sh needs from aether-utils.sh
            json_ok() { printf "{\"ok\":true,\"result\":%s}\n" "$1"; }
            json_err() { printf "{\"ok\":false,\"error\":{\"code\":\"%s\",\"message\":\"%s\"}}\n" "$1" "$2" >&2; }
            E_FILE_NOT_FOUND="E_FILE_NOT_FOUND"
            E_VALIDATION_FAILED="E_VALIDATION_FAILED"

            # Source scan.sh which defines the functions
            source "$SCRIPT_DIR/utils/scan.sh"

            # Call the requested function
            '"$func_name"' "'"$target_dir"'"
        ' 2>/dev/null
}

# Run init-research against a temp directory
run_init_research() {
    local tmp_dir="$1"
    local target_dir="${2:-$tmp_dir}"
    AETHER_ROOT="$tmp_dir" DATA_DIR="$tmp_dir/.aether/data" \
        bash "$tmp_dir/.aether/aether-utils.sh" init-research --target "$target_dir" 2>/dev/null
}

# Create a fake chamber with manifest.json and optionally CROWNED-ANTHILL.md
create_fake_chamber() {
    local chambers_dir="$1"
    local chamber_name="$2"
    local goal="${3:-Test goal}"
    local phases_completed="${4:-3}"
    local total_phases="${5:-5}"
    local milestone="${6:-Brood Stable}"
    local include_crowned="${7:-true}"

    local chamber_dir="$chambers_dir/$chamber_name"
    mkdir -p "$chamber_dir"

    cat > "$chamber_dir/manifest.json" << EOF
{
  "goal": "$goal",
  "phases_completed": $phases_completed,
  "total_phases": $total_phases,
  "milestone": "$milestone"
}
EOF

    if [[ "$include_crowned" == "true" ]]; then
        cat > "$chamber_dir/CROWNED-ANTHILL.md" << EOF
# Crowned Anthill

## The Work
Built the core API with authentication and database layer.
Implemented 50 tests covering all endpoints.

## Reflection
This was a solid colony run.
EOF
    fi
}

# ============================================================================
# Tests for _scan_colony_context (INTEL-01)
# ============================================================================

# Test 1: Colony context with chambers
test_colony_context_with_chambers() {
    local tmp_dir
    tmp_dir=$(setup_intelligence_env)
    local chambers_dir="$tmp_dir/.aether/chambers"

    # Create 2 fake chambers
    create_fake_chamber "$chambers_dir" "2026-03-10-build-api" "Build REST API" 3 5 "Brood Stable"
    create_fake_chamber "$chambers_dir" "2026-03-15-add-auth" "Add authentication" 4 4 "Crowned Anthill"

    local output
    output=$(run_scan_func "$tmp_dir" "_scan_colony_context" "$tmp_dir")

    # Assert: prior_colonies array has 2 entries
    local colony_count
    colony_count=$(echo "$output" | jq '.prior_colonies | length')
    if [[ "$colony_count" != "2" ]]; then
        test_fail "prior_colonies has 2 entries" "count = $colony_count"
        rm -rf "$tmp_dir"
        return 1
    fi

    # Assert: each entry has goal, phases, outcome, summary fields
    local first_goal first_phases first_outcome first_summary
    first_goal=$(echo "$output" | jq -r '.prior_colonies[0].goal')
    first_phases=$(echo "$output" | jq -r '.prior_colonies[0].phases')
    first_outcome=$(echo "$output" | jq -r '.prior_colonies[0].outcome')
    first_summary=$(echo "$output" | jq -r '.prior_colonies[0].summary')

    if [[ -z "$first_goal" || "$first_goal" == "null" ]]; then
        test_fail "first colony has goal" "goal = '$first_goal'"
        rm -rf "$tmp_dir"
        return 1
    fi

    if [[ -z "$first_phases" || "$first_phases" == "null" ]]; then
        test_fail "first colony has phases" "phases = '$first_phases'"
        rm -rf "$tmp_dir"
        return 1
    fi

    if [[ -z "$first_outcome" || "$first_outcome" == "null" ]]; then
        test_fail "first colony has outcome" "outcome = '$first_outcome'"
        rm -rf "$tmp_dir"
        return 1
    fi

    if [[ -z "$first_summary" || "$first_summary" == "null" ]]; then
        test_fail "first colony has summary" "summary = '$first_summary'"
        rm -rf "$tmp_dir"
        return 1
    fi

    rm -rf "$tmp_dir"
    return 0
}

# Test 2: Colony context with no chambers
test_colony_context_no_chambers() {
    local tmp_dir
    tmp_dir=$(setup_intelligence_env)

    # Remove the chambers directory
    rm -rf "$tmp_dir/.aether/chambers"

    local output
    output=$(run_scan_func "$tmp_dir" "_scan_colony_context" "$tmp_dir")

    # Assert: prior_colonies is empty array
    local colony_count
    colony_count=$(echo "$output" | jq '.prior_colonies | length')
    if [[ "$colony_count" != "0" ]]; then
        test_fail "prior_colonies is empty" "count = $colony_count"
        rm -rf "$tmp_dir"
        return 1
    fi

    # Assert: existing_charter fields are empty strings
    local intent vision governance
    intent=$(echo "$output" | jq -r '.existing_charter.intent')
    vision=$(echo "$output" | jq -r '.existing_charter.vision')
    governance=$(echo "$output" | jq -r '.existing_charter.governance')

    if [[ "$intent" != "" ]]; then
        test_fail "existing_charter.intent is empty" "intent = '$intent'"
        rm -rf "$tmp_dir"
        return 1
    fi

    if [[ "$vision" != "" ]]; then
        test_fail "existing_charter.vision is empty" "vision = '$vision'"
        rm -rf "$tmp_dir"
        return 1
    fi

    if [[ "$governance" != "" ]]; then
        test_fail "existing_charter.governance is empty" "governance = '$governance'"
        rm -rf "$tmp_dir"
        return 1
    fi

    rm -rf "$tmp_dir"
    return 0
}

# Test 3: Colony context with manifest only (no CROWNED-ANTHILL.md)
test_colony_context_manifest_only() {
    local tmp_dir
    tmp_dir=$(setup_intelligence_env)
    local chambers_dir="$tmp_dir/.aether/chambers"

    # Create chamber with manifest but NO CROWNED-ANTHILL.md
    create_fake_chamber "$chambers_dir" "2026-03-10-old-colony" "Old colony goal" 2 6 "Open Chambers" "false"

    local output
    output=$(run_scan_func "$tmp_dir" "_scan_colony_context" "$tmp_dir")

    # Assert: colony entry has goal and outcome from manifest
    local goal outcome summary
    goal=$(echo "$output" | jq -r '.prior_colonies[0].goal')
    outcome=$(echo "$output" | jq -r '.prior_colonies[0].outcome')
    summary=$(echo "$output" | jq -r '.prior_colonies[0].summary')

    if [[ "$goal" != "Old colony goal" ]]; then
        test_fail "goal from manifest" "goal = '$goal'"
        rm -rf "$tmp_dir"
        return 1
    fi

    if [[ "$outcome" != "Open Chambers" ]]; then
        test_fail "outcome from manifest" "outcome = '$outcome'"
        rm -rf "$tmp_dir"
        return 1
    fi

    # Assert: summary is empty string (no CROWNED-ANTHILL.md)
    if [[ "$summary" != "" ]]; then
        test_fail "summary is empty (no CROWNED-ANTHILL.md)" "summary = '$summary'"
        rm -rf "$tmp_dir"
        return 1
    fi

    rm -rf "$tmp_dir"
    return 0
}

# Test 4: Colony context max three
test_colony_context_max_three() {
    local tmp_dir
    tmp_dir=$(setup_intelligence_env)
    local chambers_dir="$tmp_dir/.aether/chambers"

    # Create 4 fake chambers with dated names
    create_fake_chamber "$chambers_dir" "2026-01-01-first" "First colony" 2 3 "Open Chambers"
    create_fake_chamber "$chambers_dir" "2026-02-01-second" "Second colony" 3 4 "Brood Stable"
    create_fake_chamber "$chambers_dir" "2026-03-01-third" "Third colony" 4 5 "Crowned Anthill"
    create_fake_chamber "$chambers_dir" "2026-03-15-fourth" "Fourth colony" 5 5 "Crowned Anthill"

    local output
    output=$(run_scan_func "$tmp_dir" "_scan_colony_context" "$tmp_dir")

    # Assert: prior_colonies array has exactly 3 entries (most recent 3)
    local colony_count
    colony_count=$(echo "$output" | jq '.prior_colonies | length')
    if [[ "$colony_count" != "3" ]]; then
        test_fail "prior_colonies capped at 3" "count = $colony_count"
        rm -rf "$tmp_dir"
        return 1
    fi

    # Assert: first entry is the most recent (2026-03-15)
    local first_goal
    first_goal=$(echo "$output" | jq -r '.prior_colonies[0].goal')
    if [[ "$first_goal" != "Fourth colony" ]]; then
        test_fail "most recent colony first" "first goal = '$first_goal'"
        rm -rf "$tmp_dir"
        return 1
    fi

    rm -rf "$tmp_dir"
    return 0
}

# Test 5: Colony context existing charter
test_colony_context_existing_charter() {
    local tmp_dir
    tmp_dir=$(setup_intelligence_env)
    local queen_file="$tmp_dir/.aether/QUEEN.md"

    # Write charter entries using charter-write subcommand (proven in test-init-smart-flow.sh)
    AETHER_ROOT="$tmp_dir" DATA_DIR="$tmp_dir/.aether/data" \
        bash "$tmp_dir/.aether/aether-utils.sh" charter-write \
        --intent "Build a REST API" \
        --vision "Scalable microservices" \
        --governance "Strict TDD always" \
        --goals "Ship v1 by March" >/dev/null 2>/dev/null

    local output
    output=$(run_scan_func "$tmp_dir" "_scan_colony_context" "$tmp_dir")

    # Assert: existing_charter.intent returns "Build a REST API" (stripped of Colony suffix)
    local intent
    intent=$(echo "$output" | jq -r '.existing_charter.intent')
    if [[ "$intent" != "Build a REST API" ]]; then
        test_fail "charter intent extracted (Colony suffix stripped)" "intent = '$intent'"
        rm -rf "$tmp_dir"
        return 1
    fi

    local vision
    vision=$(echo "$output" | jq -r '.existing_charter.vision')
    if [[ "$vision" != "Scalable microservices" ]]; then
        test_fail "charter vision extracted" "vision = '$vision'"
        rm -rf "$tmp_dir"
        return 1
    fi

    local governance
    governance=$(echo "$output" | jq -r '.existing_charter.governance')
    if [[ "$governance" != "Strict TDD always" ]]; then
        test_fail "charter governance extracted" "governance = '$governance'"
        rm -rf "$tmp_dir"
        return 1
    fi

    rm -rf "$tmp_dir"
    return 0
}

# ============================================================================
# Tests for _scan_pheromone_suggestions (INTEL-02)
# ============================================================================

# Test 6: Pheromone env files
test_pheromone_env_files() {
    local tmp_dir
    tmp_dir=$(setup_intelligence_env)

    # Create .env file in temp dir root
    echo "SECRET_KEY=abc123" > "$tmp_dir/.env"

    local output
    output=$(run_scan_func "$tmp_dir" "_scan_pheromone_suggestions" "$tmp_dir")

    # Assert: result contains at least one entry with type "REDIRECT"
    local redirect_count
    redirect_count=$(echo "$output" | jq '[.[] | select(.type == "REDIRECT")] | length')
    if [[ "$redirect_count" -lt 1 ]]; then
        test_fail "at least one REDIRECT for .env" "redirect_count = $redirect_count"
        rm -rf "$tmp_dir"
        return 1
    fi

    # Assert: content mentions secrets or .env
    local has_env_mention
    has_env_mention=$(echo "$output" | jq '[.[] | select(.type == "REDIRECT") | select(.content | test("secret|env"; "i"))] | length')
    if [[ "$has_env_mention" -lt 1 ]]; then
        test_fail "REDIRECT mentions secrets or .env" "no matching content"
        rm -rf "$tmp_dir"
        return 1
    fi

    rm -rf "$tmp_dir"
    return 0
}

# Test 7: Pheromone test config with tests
test_pheromone_test_config_with_tests() {
    local tmp_dir
    tmp_dir=$(setup_intelligence_env)

    # Create jest.config.js and a test file
    echo "module.exports = {};" > "$tmp_dir/jest.config.js"
    mkdir -p "$tmp_dir/tests"
    echo "test('hello', () => {});" > "$tmp_dir/tests/example.test.js"

    local output
    output=$(run_scan_func "$tmp_dir" "_scan_pheromone_suggestions" "$tmp_dir")

    # Assert: result contains FOCUS entry about testing
    local focus_testing
    focus_testing=$(echo "$output" | jq '[.[] | select(.type == "FOCUS") | select(.content | test("test|TDD"; "i"))] | length')
    if [[ "$focus_testing" -lt 1 ]]; then
        test_fail "FOCUS entry about testing" "no matching FOCUS entry"
        rm -rf "$tmp_dir"
        return 1
    fi

    rm -rf "$tmp_dir"
    return 0
}

# Test 8: Pheromone test config without tests (REDIRECT, not FOCUS)
test_pheromone_test_config_no_tests() {
    local tmp_dir
    tmp_dir=$(setup_intelligence_env)

    # Create jest.config.js but NO test files
    echo "module.exports = {};" > "$tmp_dir/jest.config.js"

    local output
    output=$(run_scan_func "$tmp_dir" "_scan_pheromone_suggestions" "$tmp_dir")

    # Assert: result contains REDIRECT entry about missing tests (not FOCUS about testing)
    local redirect_testing
    redirect_testing=$(echo "$output" | jq '[.[] | select(.type == "REDIRECT") | select(.content | test("test|no test"; "i"))] | length')
    if [[ "$redirect_testing" -lt 1 ]]; then
        test_fail "REDIRECT about missing tests" "no matching REDIRECT entry"
        rm -rf "$tmp_dir"
        return 1
    fi

    # Assert: NO FOCUS entry about testing (cross-reference validation)
    local focus_testing
    focus_testing=$(echo "$output" | jq '[.[] | select(.type == "FOCUS") | select(.content | test("test|TDD"; "i"))] | length')
    if [[ "$focus_testing" -gt 0 ]]; then
        test_fail "no FOCUS about testing (config without files)" "found $focus_testing FOCUS entries"
        rm -rf "$tmp_dir"
        return 1
    fi

    rm -rf "$tmp_dir"
    return 0
}

# Test 9: Pheromone empty repo
test_pheromone_empty_repo() {
    local tmp_dir
    tmp_dir=$(setup_intelligence_env)

    # Create a clean target with no config files
    local scan_target
    scan_target=$(mktemp -d)

    local output
    output=$(run_scan_func "$tmp_dir" "_scan_pheromone_suggestions" "$scan_target")

    # Assert: result is empty array
    local count
    count=$(echo "$output" | jq 'length')
    if [[ "$count" != "0" ]]; then
        test_fail "empty array for empty repo" "count = $count"
        rm -rf "$tmp_dir" "$scan_target"
        return 1
    fi

    rm -rf "$tmp_dir" "$scan_target"
    return 0
}

# Test 10: Pheromone cap at five
test_pheromone_cap_five() {
    local tmp_dir
    tmp_dir=$(setup_intelligence_env)

    # Create enough config files to trigger more than 5 patterns:
    # 1. .env (priority 9, REDIRECT)
    echo "SECRET=value" > "$tmp_dir/.env"
    # 2. jest.config.js + test files (priority 8, FOCUS)
    echo "module.exports = {};" > "$tmp_dir/jest.config.js"
    mkdir -p "$tmp_dir/tests"
    echo "test('a', () => {});" > "$tmp_dir/tests/a.test.js"
    # 3. .eslintrc.json (priority 7, FOCUS)
    echo '{"rules":{}}' > "$tmp_dir/.eslintrc.json"
    # 4. .github/workflows/ci.yml (priority 7, FOCUS)
    mkdir -p "$tmp_dir/.github/workflows"
    echo "name: CI" > "$tmp_dir/.github/workflows/ci.yml"
    # 5. tsconfig.json with strict:true (priority 6, FOCUS)
    echo '{"compilerOptions":{"strict": true}}' > "$tmp_dir/tsconfig.json"
    # 6. CONTRIBUTING.md (priority 6, FOCUS)
    printf "# Contributing\n\nPlease follow these guidelines.\n" > "$tmp_dir/CONTRIBUTING.md"
    # 7. Dockerfile (priority 5, FOCUS)
    echo "FROM node:18" > "$tmp_dir/Dockerfile"
    # 8. package.json with helmet (priority 8, FOCUS - security)
    cat > "$tmp_dir/package.json" << 'EOF'
{
  "name": "test",
  "version": "1.0.0",
  "dependencies": {"helmet": "^7.0.0"}
}
EOF

    local output
    output=$(run_scan_func "$tmp_dir" "_scan_pheromone_suggestions" "$tmp_dir")

    # Assert: result array length is exactly 5
    local count
    count=$(echo "$output" | jq 'length')
    if [[ "$count" != "5" ]]; then
        test_fail "pheromone suggestions capped at 5" "count = $count"
        rm -rf "$tmp_dir"
        return 1
    fi

    rm -rf "$tmp_dir"
    return 0
}

# Test 11: Pheromone sorted by priority
test_pheromone_sorted_by_priority() {
    local tmp_dir
    tmp_dir=$(setup_intelligence_env)

    # Create files that trigger suggestions with different priorities:
    # .env -> priority 9 (REDIRECT)
    echo "SECRET=value" > "$tmp_dir/.env"
    # .eslintrc.json -> priority 7 (FOCUS)
    echo '{"rules":{}}' > "$tmp_dir/.eslintrc.json"
    # Dockerfile -> priority 5 (FOCUS)
    echo "FROM node:18" > "$tmp_dir/Dockerfile"

    local output
    output=$(run_scan_func "$tmp_dir" "_scan_pheromone_suggestions" "$tmp_dir")

    # Assert: at least 3 entries
    local count
    count=$(echo "$output" | jq 'length')
    if [[ "$count" -lt 3 ]]; then
        test_fail "at least 3 suggestions" "count = $count"
        rm -rf "$tmp_dir"
        return 1
    fi

    # Assert: entries are in descending priority order
    local first_priority second_priority
    first_priority=$(echo "$output" | jq '.[0].priority')
    second_priority=$(echo "$output" | jq '.[1].priority')

    if [[ "$first_priority" -lt "$second_priority" ]]; then
        test_fail "descending priority order" "first=$first_priority < second=$second_priority"
        rm -rf "$tmp_dir"
        return 1
    fi

    # Also verify the last entry has lowest priority
    local last_priority
    last_priority=$(echo "$output" | jq '.[-1].priority')
    if [[ "$first_priority" -lt "$last_priority" ]]; then
        test_fail "first priority >= last priority" "first=$first_priority < last=$last_priority"
        rm -rf "$tmp_dir"
        return 1
    fi

    rm -rf "$tmp_dir"
    return 0
}

# ============================================================================
# Tests for _scan_governance (INTEL-03)
# ============================================================================

# Test 12: Governance CONTRIBUTING.md
test_governance_contributing() {
    local tmp_dir
    tmp_dir=$(setup_intelligence_env)

    # Create CONTRIBUTING.md with content
    cat > "$tmp_dir/CONTRIBUTING.md" << 'EOF'
# Contributing

Please follow these coding guidelines when submitting PRs.
Use conventional commits for all commit messages.
EOF

    local output
    output=$(run_scan_func "$tmp_dir" "_scan_governance" "$tmp_dir")

    # Assert: rules array contains entry with rule matching "CONTRIBUTING.md"
    local contrib_rule_count
    contrib_rule_count=$(echo "$output" | jq '[.rules[] | select(.rule | test("CONTRIBUTING"; "i"))] | length')
    if [[ "$contrib_rule_count" -lt 1 ]]; then
        test_fail "rule about CONTRIBUTING.md" "no matching rule"
        rm -rf "$tmp_dir"
        return 1
    fi

    # Assert: source is "CONTRIBUTING.md"
    local source
    source=$(echo "$output" | jq -r '[.rules[] | select(.rule | test("CONTRIBUTING"; "i"))][0].source')
    if [[ "$source" != "CONTRIBUTING.md" ]]; then
        test_fail "source = CONTRIBUTING.md" "source = '$source'"
        rm -rf "$tmp_dir"
        return 1
    fi

    rm -rf "$tmp_dir"
    return 0
}

# Test 13: Governance test config with tests
test_governance_test_config_with_tests() {
    local tmp_dir
    tmp_dir=$(setup_intelligence_env)

    # Create jest.config.js AND test files
    echo "module.exports = {};" > "$tmp_dir/jest.config.js"
    mkdir -p "$tmp_dir/tests"
    echo "test('foo', () => {});" > "$tmp_dir/tests/foo.test.js"

    local output
    output=$(run_scan_func "$tmp_dir" "_scan_governance" "$tmp_dir")

    # Assert: rules contain "TDD required" entry
    local tdd_rule_count
    tdd_rule_count=$(echo "$output" | jq '[.rules[] | select(.rule | test("TDD"; "i"))] | length')
    if [[ "$tdd_rule_count" -lt 1 ]]; then
        test_fail "TDD required rule present" "no TDD rule found"
        rm -rf "$tmp_dir"
        return 1
    fi

    rm -rf "$tmp_dir"
    return 0
}

# Test 14: Governance test config without tests (no TDD rule)
test_governance_test_config_no_tests() {
    local tmp_dir
    tmp_dir=$(setup_intelligence_env)

    # Create jest.config.js but NO test files
    echo "module.exports = {};" > "$tmp_dir/jest.config.js"

    local output
    output=$(run_scan_func "$tmp_dir" "_scan_governance" "$tmp_dir")

    # Assert: rules do NOT contain TDD entry (cross-reference prevents false positive)
    local tdd_rule_count
    tdd_rule_count=$(echo "$output" | jq '[.rules[] | select(.rule | test("TDD"; "i"))] | length')
    if [[ "$tdd_rule_count" -gt 0 ]]; then
        test_fail "no TDD rule (config without test files)" "found $tdd_rule_count TDD rules"
        rm -rf "$tmp_dir"
        return 1
    fi

    rm -rf "$tmp_dir"
    return 0
}

# Test 15: Governance CI/CD
test_governance_cicd() {
    local tmp_dir
    tmp_dir=$(setup_intelligence_env)

    # Create .github/workflows/ directory with a dummy .yml file
    mkdir -p "$tmp_dir/.github/workflows"
    echo "name: CI" > "$tmp_dir/.github/workflows/ci.yml"

    local output
    output=$(run_scan_func "$tmp_dir" "_scan_governance" "$tmp_dir")

    # Assert: rules contain CI/CD entry
    local ci_rule_count
    ci_rule_count=$(echo "$output" | jq '[.rules[] | select(.rule | test("CI/CD|CI"; "i"))] | length')
    if [[ "$ci_rule_count" -lt 1 ]]; then
        test_fail "CI/CD rule present" "no CI rule found"
        rm -rf "$tmp_dir"
        return 1
    fi

    rm -rf "$tmp_dir"
    return 0
}

# Test 16: Governance empty repo
test_governance_empty_repo() {
    local tmp_dir
    tmp_dir=$(setup_intelligence_env)

    # Create a clean target with no governance files
    local scan_target
    scan_target=$(mktemp -d)

    local output
    output=$(run_scan_func "$tmp_dir" "_scan_governance" "$scan_target")

    # Assert: rules is empty array
    local rule_count
    rule_count=$(echo "$output" | jq '.rules | length')
    if [[ "$rule_count" != "0" ]]; then
        test_fail "rules empty for empty repo" "count = $rule_count"
        rm -rf "$tmp_dir" "$scan_target"
        return 1
    fi

    rm -rf "$tmp_dir" "$scan_target"
    return 0
}

# ============================================================================
# Integration test
# ============================================================================

# Test 17: init-research includes intelligence fields
test_init_research_includes_intelligence() {
    local tmp_dir
    tmp_dir=$(setup_intelligence_env)
    local chambers_dir="$tmp_dir/.aether/chambers"

    # Add some chambers
    create_fake_chamber "$chambers_dir" "2026-03-10-test-colony" "Test colony" 3 5 "Brood Stable"

    # Add some config files
    echo '{"rules":{}}' > "$tmp_dir/.eslintrc.json"

    local output result
    output=$(run_init_research "$tmp_dir")
    result=$(echo "$output" | jq -r '.result')

    # Assert: result JSON has colony_context at the top level
    if ! echo "$result" | jq -e '.colony_context' >/dev/null 2>&1; then
        test_fail "result has colony_context" "field not found"
        rm -rf "$tmp_dir"
        return 1
    fi

    # Assert: result JSON has governance at the top level
    if ! echo "$result" | jq -e '.governance' >/dev/null 2>&1; then
        test_fail "result has governance" "field not found"
        rm -rf "$tmp_dir"
        return 1
    fi

    # Assert: result JSON has pheromone_suggestions at the top level
    if ! echo "$result" | jq -e '.pheromone_suggestions' >/dev/null 2>&1; then
        test_fail "result has pheromone_suggestions" "field not found"
        rm -rf "$tmp_dir"
        return 1
    fi

    rm -rf "$tmp_dir"
    return 0
}

# ============================================================================
# Run all tests
# ============================================================================
log_info "Running Intelligence Sub-Scan integration tests"

# _scan_colony_context tests (INTEL-01)
run_test test_colony_context_with_chambers "Colony context: chambers produce prior_colonies with goal/phases/outcome/summary"
run_test test_colony_context_no_chambers "Colony context: no chambers returns empty prior_colonies and empty charter"
run_test test_colony_context_manifest_only "Colony context: manifest without CROWNED-ANTHILL.md has empty summary"
run_test test_colony_context_max_three "Colony context: max 3 prior colonies (most recent first)"
run_test test_colony_context_existing_charter "Colony context: existing charter extracted from QUEEN.md with Colony suffix stripped"

# _scan_pheromone_suggestions tests (INTEL-02)
run_test test_pheromone_env_files "Pheromone suggestions: .env file triggers REDIRECT about secrets"
run_test test_pheromone_test_config_with_tests "Pheromone suggestions: jest config + tests triggers FOCUS about testing"
run_test test_pheromone_test_config_no_tests "Pheromone suggestions: jest config without tests triggers REDIRECT (not FOCUS)"
run_test test_pheromone_empty_repo "Pheromone suggestions: empty repo returns empty array"
run_test test_pheromone_cap_five "Pheromone suggestions: output capped at 5 maximum"
run_test test_pheromone_sorted_by_priority "Pheromone suggestions: sorted by priority descending"

# _scan_governance tests (INTEL-03)
run_test test_governance_contributing "Governance: CONTRIBUTING.md detected with correct source"
run_test test_governance_test_config_with_tests "Governance: jest config + tests produces TDD rule"
run_test test_governance_test_config_no_tests "Governance: jest config without tests skips TDD rule"
run_test test_governance_cicd "Governance: GitHub Actions detected as CI/CD rule"
run_test test_governance_empty_repo "Governance: empty repo returns empty rules"

# Integration test
run_test test_init_research_includes_intelligence "Integration: init-research includes colony_context, governance, and pheromone_suggestions"

test_summary
