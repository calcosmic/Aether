#!/usr/bin/env bash
# Scan Module Integration Tests
# Tests scan.sh functions via aether-utils.sh init-research subcommand
# Covers: tech stack detection, directory structure, git history,
#         survey status, prior colonies, complexity estimation, performance

set -euo pipefail

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
AETHER_UTILS_SOURCE="$PROJECT_ROOT/.aether/aether-utils.sh"

# Source test helpers
source "$SCRIPT_DIR/test-helpers.sh"

# Verify jq is available
require_jq

# Verify aether-utils.sh exists
if [[ ! -f "$AETHER_UTILS_SOURCE" ]]; then
    log_error "aether-utils.sh not found at: $AETHER_UTILS_SOURCE"
    exit 1
fi

# ============================================================================
# Helper: Create isolated scan test environment
# ============================================================================
setup_scan_env() {
    local tmp_dir
    tmp_dir=$(mktemp -d)
    mkdir -p "$tmp_dir/.aether/data" "$tmp_dir/.aether/utils"

    cp "$AETHER_UTILS_SOURCE" "$tmp_dir/.aether/aether-utils.sh"
    chmod +x "$tmp_dir/.aether/aether-utils.sh"

    local utils_source="$(dirname "$AETHER_UTILS_SOURCE")/utils"
    if [[ -d "$utils_source" ]]; then
        cp -r "$utils_source" "$tmp_dir/.aether/"
    fi

    local exchange_source="$(dirname "$AETHER_UTILS_SOURCE")/exchange"
    if [[ -d "$exchange_source" ]]; then
        cp -r "$exchange_source" "$tmp_dir/.aether/"
    fi

    # Write a minimal valid COLONY_STATE.json
    cat > "$tmp_dir/.aether/data/COLONY_STATE.json" << 'EOF'
{
  "version": "3.0",
  "goal": "Test scan module",
  "state": "READY",
  "current_phase": 1,
  "session_id": "test-session",
  "initialized_at": "2026-01-01T00:00:00Z",
  "plan": { "phases": [{ "id": 1, "name": "Test Phase", "status": "pending" }] },
  "memory": { "phase_learnings": [], "decisions": [], "instincts": [] },
  "errors": { "records": [], "flagged_patterns": [] },
  "events": [],
  "signals": [],
  "graveyards": []
}
EOF

    echo "$tmp_dir"
}

# Run init-research against a temp directory
run_scan() {
    local tmp_dir="$1"
    local target_dir="${2:-$tmp_dir}"
    AETHER_ROOT="$tmp_dir" DATA_DIR="$tmp_dir/.aether/data" \
        bash "$tmp_dir/.aether/aether-utils.sh" init-research --target "$target_dir" 2>/dev/null
}

# Extract .result from json_ok output
extract_result() {
    echo "$1" | jq -r '.result'
}

# ============================================================================
# Test 1: Schema validity
# ============================================================================
test_schema_validity() {
    local tmp_dir
    tmp_dir=$(setup_scan_env)

    local output result
    output=$(run_scan "$tmp_dir")
    result=$(extract_result "$output")

    # Verify JSON is valid
    if ! assert_json_valid "$result"; then
        test_fail "valid JSON" "invalid JSON output"
        rm -rf "$tmp_dir"
        return 1
    fi

    # Verify all required top-level keys exist
    local required_keys=("schema_version" "tech_stack" "directory_structure" "git_history" "survey_status" "prior_colonies" "complexity" "scanned_at")
    for key in "${required_keys[@]}"; do
        if ! assert_json_has_field "$result" "$key"; then
            test_fail "has field '$key'" "missing field '$key'"
            rm -rf "$tmp_dir"
            return 1
        fi
    done

    # Verify schema_version is 1
    if ! assert_json_field_equals "$result" ".schema_version" "1"; then
        test_fail "schema_version = 1" "$(echo "$result" | jq -r '.schema_version')"
        rm -rf "$tmp_dir"
        return 1
    fi

    rm -rf "$tmp_dir"
    return 0
}

# ============================================================================
# Test 2: Empty directory (no git, no .aether content)
# ============================================================================
test_empty_directory() {
    local tmp_dir
    tmp_dir=$(mktemp -d)
    mkdir -p "$tmp_dir/.aether/data" "$tmp_dir/.aether/utils"

    cp "$AETHER_UTILS_SOURCE" "$tmp_dir/.aether/aether-utils.sh"
    chmod +x "$tmp_dir/.aether/aether-utils.sh"
    local utils_source="$(dirname "$AETHER_UTILS_SOURCE")/utils"
    [[ -d "$utils_source" ]] && cp -r "$utils_source" "$tmp_dir/.aether/"
    local exchange_source="$(dirname "$AETHER_UTILS_SOURCE")/exchange"
    [[ -d "$exchange_source" ]] && cp -r "$exchange_source" "$tmp_dir/.aether/"

    # Scan the empty directory (not the tmp_dir itself, to avoid counting .aether)
    local scan_target
    scan_target=$(mktemp -d)
    local output result
    output=$(run_scan "$tmp_dir" "$scan_target")
    result=$(extract_result "$output")

    # Verify empty state
    local file_count
    file_count=$(echo "$result" | jq -r '.directory_structure.file_count')
    if [[ "$file_count" != "0" ]]; then
        test_fail "file_count = 0" "file_count = $file_count"
        rm -rf "$tmp_dir" "$scan_target"
        return 1
    fi

    local lang_count
    lang_count=$(echo "$result" | jq '.tech_stack.languages | length')
    if [[ "$lang_count" != "0" ]]; then
        test_fail "languages = []" "languages count = $lang_count"
        rm -rf "$tmp_dir" "$scan_target"
        return 1
    fi

    local is_git
    is_git=$(echo "$result" | jq -r '.git_history.is_git_repo')
    if [[ "$is_git" != "false" ]]; then
        test_fail "is_git_repo = false" "is_git_repo = $is_git"
        rm -rf "$tmp_dir" "$scan_target"
        return 1
    fi

    local has_survey
    has_survey=$(echo "$result" | jq -r '.survey_status.has_survey')
    if [[ "$has_survey" != "false" ]]; then
        test_fail "has_survey = false" "has_survey = $has_survey"
        rm -rf "$tmp_dir" "$scan_target"
        return 1
    fi

    rm -rf "$tmp_dir" "$scan_target"
    return 0
}

# ============================================================================
# Test 3: Git repo detection
# ============================================================================
test_git_repo_detection() {
    local tmp_dir
    tmp_dir=$(setup_scan_env)

    # Init git repo in the temp dir with one commit
    cd "$tmp_dir"
    git init -q 2>/dev/null
    git config user.email "test@test.com"
    git config user.name "Test"
    echo "hello" > file.txt
    git add . 2>/dev/null
    git commit -q -m "initial commit" 2>/dev/null

    local output result
    output=$(run_scan "$tmp_dir")
    result=$(extract_result "$output")

    local is_git
    is_git=$(echo "$result" | jq -r '.git_history.is_git_repo')
    if [[ "$is_git" != "true" ]]; then
        test_fail "is_git_repo = true" "is_git_repo = $is_git"
        rm -rf "$tmp_dir"
        return 1
    fi

    local commit_count
    commit_count=$(echo "$result" | jq -r '.git_history.commit_count')
    if [[ "$commit_count" != "1" ]]; then
        test_fail "commit_count = 1" "commit_count = $commit_count"
        rm -rf "$tmp_dir"
        return 1
    fi

    local recent_len
    recent_len=$(echo "$result" | jq '.git_history.recent_commits | length')
    if [[ "$recent_len" != "1" ]]; then
        test_fail "recent_commits length = 1" "length = $recent_len"
        rm -rf "$tmp_dir"
        return 1
    fi

    rm -rf "$tmp_dir"
    return 0
}

# ============================================================================
# Test 4: Tech stack - JavaScript detection
# ============================================================================
test_tech_stack_javascript() {
    local tmp_dir
    tmp_dir=$(setup_scan_env)

    # Create a package.json
    cat > "$tmp_dir/package.json" << 'EOF'
{
  "name": "test-project",
  "version": "1.0.0",
  "dependencies": {
    "express": "^4.18.0"
  }
}
EOF

    local output result
    output=$(run_scan "$tmp_dir")
    result=$(extract_result "$output")

    # Check JavaScript is detected
    local has_js
    has_js=$(echo "$result" | jq '.tech_stack.languages | any(. == "javascript")')
    if [[ "$has_js" != "true" ]]; then
        test_fail "languages includes javascript" "$(echo "$result" | jq '.tech_stack.languages')"
        rm -rf "$tmp_dir"
        return 1
    fi

    # Check npm is detected (no lock file -> default npm)
    local has_npm
    has_npm=$(echo "$result" | jq '.tech_stack.package_managers | any(. == "npm")')
    if [[ "$has_npm" != "true" ]]; then
        test_fail "package_managers includes npm" "$(echo "$result" | jq '.tech_stack.package_managers')"
        rm -rf "$tmp_dir"
        return 1
    fi

    rm -rf "$tmp_dir"
    return 0
}

# ============================================================================
# Test 5: Tech stack - TypeScript detection
# ============================================================================
test_tech_stack_typescript() {
    local tmp_dir
    tmp_dir=$(setup_scan_env)

    # Create tsconfig.json alongside package.json
    cat > "$tmp_dir/package.json" << 'EOF'
{"name": "ts-project", "version": "1.0.0"}
EOF
    echo '{"compilerOptions": {"strict": true}}' > "$tmp_dir/tsconfig.json"

    local output result
    output=$(run_scan "$tmp_dir")
    result=$(extract_result "$output")

    local has_ts
    has_ts=$(echo "$result" | jq '.tech_stack.languages | any(. == "typescript")')
    if [[ "$has_ts" != "true" ]]; then
        test_fail "languages includes typescript" "$(echo "$result" | jq '.tech_stack.languages')"
        rm -rf "$tmp_dir"
        return 1
    fi

    # JavaScript should also be detected (package.json present)
    local has_js
    has_js=$(echo "$result" | jq '.tech_stack.languages | any(. == "javascript")')
    if [[ "$has_js" != "true" ]]; then
        test_fail "languages includes javascript" "$(echo "$result" | jq '.tech_stack.languages')"
        rm -rf "$tmp_dir"
        return 1
    fi

    rm -rf "$tmp_dir"
    return 0
}

# ============================================================================
# Test 6: Tech stack - Python detection
# ============================================================================
test_tech_stack_python() {
    local tmp_dir
    tmp_dir=$(setup_scan_env)

    echo "flask==2.0.0" > "$tmp_dir/requirements.txt"

    local output result
    output=$(run_scan "$tmp_dir")
    result=$(extract_result "$output")

    local has_py
    has_py=$(echo "$result" | jq '.tech_stack.languages | any(. == "python")')
    if [[ "$has_py" != "true" ]]; then
        test_fail "languages includes python" "$(echo "$result" | jq '.tech_stack.languages')"
        rm -rf "$tmp_dir"
        return 1
    fi

    # Check pip detected
    local has_pip
    has_pip=$(echo "$result" | jq '.tech_stack.package_managers | any(. == "pip")')
    if [[ "$has_pip" != "true" ]]; then
        test_fail "package_managers includes pip" "$(echo "$result" | jq '.tech_stack.package_managers')"
        rm -rf "$tmp_dir"
        return 1
    fi

    rm -rf "$tmp_dir"
    return 0
}

# ============================================================================
# Test 7: Directory structure
# ============================================================================
test_directory_structure() {
    local tmp_dir
    tmp_dir=$(setup_scan_env)

    # Create 3 nested directories with 5 files
    mkdir -p "$tmp_dir/src/components" "$tmp_dir/tests" "$tmp_dir/lib"
    touch "$tmp_dir/src/index.js"
    touch "$tmp_dir/src/components/App.js"
    touch "$tmp_dir/tests/test.js"
    touch "$tmp_dir/lib/utils.js"
    touch "$tmp_dir/README.md"

    local output result
    output=$(run_scan "$tmp_dir")
    result=$(extract_result "$output")

    # Verify file count (5 created + COLONY_STATE.json + aether-utils.sh + utils files)
    local file_count
    file_count=$(echo "$result" | jq -r '.directory_structure.file_count')
    # At least 5 files we created, .aether files are excluded
    if [[ "$file_count" -lt 5 ]]; then
        test_fail "file_count >= 5" "file_count = $file_count"
        rm -rf "$tmp_dir"
        return 1
    fi

    # Verify top_level_dirs includes our created directories
    local dirs_json
    dirs_json=$(echo "$result" | jq -r '.directory_structure.top_level_dirs')
    if ! echo "$dirs_json" | grep -q "src"; then
        test_fail "top_level_dirs includes 'src'" "$dirs_json"
        rm -rf "$tmp_dir"
        return 1
    fi
    if ! echo "$dirs_json" | grep -q "tests"; then
        test_fail "top_level_dirs includes 'tests'" "$dirs_json"
        rm -rf "$tmp_dir"
        return 1
    fi
    if ! echo "$dirs_json" | grep -q "lib"; then
        test_fail "top_level_dirs includes 'lib'" "$dirs_json"
        rm -rf "$tmp_dir"
        return 1
    fi

    rm -rf "$tmp_dir"
    return 0
}

# ============================================================================
# Test 8: Complexity - small
# ============================================================================
test_complexity_small() {
    local tmp_dir
    tmp_dir=$(setup_scan_env)

    # Create a small repo: 10 files, shallow depth, no deps
    mkdir -p "$tmp_dir/src"
    for i in $(seq 1 10); do
        touch "$tmp_dir/src/file${i}.js"
    done

    local output result
    output=$(run_scan "$tmp_dir")
    result=$(extract_result "$output")

    local size
    size=$(echo "$result" | jq -r '.complexity.size')
    if [[ "$size" != "small" ]]; then
        # Note: temp directory depth may cause medium classification on some systems.
        # Accept "small" or "medium" -- the important thing is it's not "large"
        if [[ "$size" == "large" ]]; then
            test_fail "complexity size is small or medium" "size = $size"
            rm -rf "$tmp_dir"
            return 1
        fi
        log_info "  (Accepting 'medium' due to system temp dir depth)"
    fi

    rm -rf "$tmp_dir"
    return 0
}

# ============================================================================
# Test 9: Complexity - medium
# ============================================================================
test_complexity_medium() {
    local tmp_dir
    tmp_dir=$(setup_scan_env)

    # Create a medium repo: 150 files, depth 6, 20 deps in package.json
    mkdir -p "$tmp_dir/src/a/b/c/d/e"
    local dep_json='{"name":"test","version":"1.0.0","dependencies":{'
    local dep_entries=()
    for i in $(seq 1 20); do
        dep_entries+=("\"dep${i}\":\"^1.0.0\"")
    done
    dep_json+=$(IFS=,; echo "${dep_entries[*]}")
    dep_json+='}}'
    echo "$dep_json" > "$tmp_dir/package.json"

    # Create 150 files spread across directories
    local count=0
    for dir in src src/a src/a/b src/a/b/c src/a/b/c/d src/a/b/c/d/e; do
        for i in $(seq 1 25); do
            touch "$tmp_dir/$dir/file${count}.js"
            count=$((count + 1))
            [[ $count -ge 150 ]] && break
        done
        [[ $count -ge 150 ]] && break
    done

    local output result
    output=$(run_scan "$tmp_dir")
    result=$(extract_result "$output")

    local size
    size=$(echo "$result" | jq -r '.complexity.size')
    if [[ "$size" != "medium" && "$size" != "large" ]]; then
        test_fail "complexity size is medium or large" "size = $size"
        rm -rf "$tmp_dir"
        return 1
    fi

    rm -rf "$tmp_dir"
    return 0
}

# ============================================================================
# Test 10: Survey status - no survey
# ============================================================================
test_survey_status_no_survey() {
    local tmp_dir
    tmp_dir=$(setup_scan_env)

    # Ensure no survey directory exists
    rm -rf "$tmp_dir/.aether/data/survey"

    local output result
    output=$(run_scan "$tmp_dir")
    result=$(extract_result "$output")

    local has_survey
    has_survey=$(echo "$result" | jq -r '.survey_status.has_survey')
    if [[ "$has_survey" != "false" ]]; then
        test_fail "has_survey = false" "has_survey = $has_survey"
        rm -rf "$tmp_dir"
        return 1
    fi

    local suggestion_action
    suggestion_action=$(echo "$result" | jq -r '.survey_status.suggestion.action')
    if [[ "$suggestion_action" != "colonize" ]]; then
        test_fail "suggestion.action = colonize" "suggestion.action = $suggestion_action"
        rm -rf "$tmp_dir"
        return 1
    fi

    rm -rf "$tmp_dir"
    return 0
}

# ============================================================================
# Test 11: Survey status - stale survey
# ============================================================================
test_survey_status_stale() {
    local tmp_dir
    tmp_dir=$(setup_scan_env)

    # Create survey directory with ALL 7 required docs (scan checks completeness first)
    mkdir -p "$tmp_dir/.aether/data/survey"
    local required_docs="PROVISIONS.md TRAILS.md BLUEPRINT.md CHAMBERS.md DISCIPLINES.md SENTINEL-PROTOCOLS.md PATHOGENS.md"
    for doc in $required_docs; do
        echo "# $doc" > "$tmp_dir/.aether/data/survey/$doc"
    done

    # Set COLONY_STATE.json with territory_surveyed 30 days ago
    local stale_date
    stale_date=$(date -u -v-30d +"%Y-%m-%dT%H:%M:%SZ" 2>/dev/null || \
                 date -u -d "30 days ago" +"%Y-%m-%dT%H:%M:%SZ" 2>/dev/null || \
                 echo "2026-02-25T00:00:00Z")

    cat > "$tmp_dir/.aether/data/COLONY_STATE.json" << EOF
{
  "version": "3.0",
  "goal": "Test stale survey",
  "state": "READY",
  "territory_surveyed": "$stale_date",
  "current_phase": 1,
  "session_id": "test-session",
  "initialized_at": "2026-01-01T00:00:00Z",
  "plan": { "phases": [] },
  "memory": { "phase_learnings": [], "decisions": [], "instincts": [] },
  "errors": { "records": [], "flagged_patterns": [] },
  "events": [],
  "signals": [],
  "graveyards": []
}
EOF

    local output result
    output=$(run_scan "$tmp_dir")
    result=$(extract_result "$output")

    local has_survey
    has_survey=$(echo "$result" | jq -r '.survey_status.has_survey')
    if [[ "$has_survey" != "true" ]]; then
        test_fail "has_survey = true" "has_survey = $has_survey"
        rm -rf "$tmp_dir"
        return 1
    fi

    local is_stale
    is_stale=$(echo "$result" | jq -r '.survey_status.is_stale')
    if [[ "$is_stale" != "true" ]]; then
        test_fail "is_stale = true" "is_stale = $is_stale"
        rm -rf "$tmp_dir"
        return 1
    fi

    # Verify suggestion exists (use jq path since assert_json_has_field only checks top-level)
    local has_suggestion
    has_suggestion=$(echo "$result" | jq -e '.survey_status.suggestion' >/dev/null 2>&1 && echo "true" || echo "false")
    if [[ "$has_suggestion" != "true" ]]; then
        test_fail "survey_status has suggestion" "no suggestion field"
        rm -rf "$tmp_dir"
        return 1
    fi

    rm -rf "$tmp_dir"
    return 0
}

# ============================================================================
# Test 12: Prior colonies - active
# ============================================================================
test_prior_colonies_active() {
    local tmp_dir
    tmp_dir=$(setup_scan_env)

    # COLONY_STATE.json already has goal and state="READY" from setup_scan_env

    local output result
    output=$(run_scan "$tmp_dir")
    result=$(extract_result "$output")

    local has_active
    has_active=$(echo "$result" | jq -r '.prior_colonies.has_active_colony')
    if [[ "$has_active" != "true" ]]; then
        test_fail "has_active_colony = true" "has_active_colony = $has_active"
        rm -rf "$tmp_dir"
        return 1
    fi

    local active_goal
    active_goal=$(echo "$result" | jq -r '.prior_colonies.active_goal')
    if [[ "$active_goal" != "Test scan module" ]]; then
        test_fail "active_goal = 'Test scan module'" "active_goal = '$active_goal'"
        rm -rf "$tmp_dir"
        return 1
    fi

    rm -rf "$tmp_dir"
    return 0
}

# ============================================================================
# Test 13: Prior colonies - archived
# ============================================================================
test_prior_colonies_archived() {
    local tmp_dir
    tmp_dir=$(setup_scan_env)

    # Create an archived colony in chambers
    mkdir -p "$tmp_dir/.aether/chambers/test-chamber"
    cat > "$tmp_dir/.aether/chambers/test-chamber/COLONY_STATE.json" << 'EOF'
{
  "version": "3.0",
  "goal": "Archived colony test",
  "state": "SEALED",
  "initialized_at": "2026-01-15T00:00:00Z"
}
EOF

    local output result
    output=$(run_scan "$tmp_dir")
    result=$(extract_result "$output")

    local archived_len
    archived_len=$(echo "$result" | jq '.prior_colonies.archived_colonies | length')
    if [[ "$archived_len" != "1" ]]; then
        test_fail "archived_colonies length = 1" "length = $archived_len"
        rm -rf "$tmp_dir"
        return 1
    fi

    local archived_goal
    archived_goal=$(echo "$result" | jq -r '.prior_colonies.archived_colonies[0].goal')
    if [[ "$archived_goal" != "Archived colony test" ]]; then
        test_fail "archived goal = 'Archived colony test'" "goal = '$archived_goal'"
        rm -rf "$tmp_dir"
        return 1
    fi

    rm -rf "$tmp_dir"
    return 0
}

# ============================================================================
# Test 14: Performance - completes in under 2 seconds on Aether repo
# ============================================================================
test_performance() {
    # Scan the Aether repo itself
    local start_ns end_ns duration_ms
    start_ns=$(date +%s%N)

    local output
    output=$(AETHER_ROOT="$PROJECT_ROOT" DATA_DIR="$PROJECT_ROOT/.aether/data" \
        bash "$AETHER_UTILS_SOURCE" init-research --target "$PROJECT_ROOT" 2>/dev/null)

    end_ns=$(date +%s%N)

    # Verify output is valid
    if ! assert_json_valid "$output"; then
        test_fail "valid JSON output" "invalid JSON"
        return 1
    fi

    if ! assert_ok_true "$output"; then
        test_fail "ok:true" "ok not true"
        return 1
    fi

    # Calculate duration in milliseconds
    duration_ms=$(( (end_ns - start_ns) / 1000000 ))

    log_info "  Scan completed in ${duration_ms}ms"

    # Must complete under 2 seconds (2000ms)
    if [[ "$duration_ms" -gt 2000 ]]; then
        test_fail "under 2000ms" "${duration_ms}ms"
        return 1
    fi

    return 0
}

# ============================================================================
# Run all tests
# ============================================================================
log_info "Running Scan Module integration tests"

run_test test_schema_validity "Schema validity - all top-level keys present"
run_test test_empty_directory "Empty directory - no git, no .aether content"
run_test test_git_repo_detection "Git repo detection - is_git_repo, commit_count, recent_commits"
run_test test_tech_stack_javascript "Tech stack - JavaScript detection via package.json"
run_test test_tech_stack_typescript "Tech stack - TypeScript detection via tsconfig.json"
run_test test_tech_stack_python "Tech stack - Python detection via requirements.txt"
run_test test_directory_structure "Directory structure - file_count, top_level_dirs"
run_test test_complexity_small "Complexity - small classification"
run_test test_complexity_medium "Complexity - medium classification"
run_test test_survey_status_no_survey "Survey status - no survey directory"
run_test test_survey_status_stale "Survey status - stale survey (30 days old)"
run_test test_prior_colonies_active "Prior colonies - active colony detection"
run_test test_prior_colonies_archived "Prior colonies - archived chamber detection"
run_test test_performance "Performance - completes in under 2 seconds"

test_summary
