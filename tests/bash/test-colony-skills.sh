#!/usr/bin/env bash
# Tests for the 10 colony SKILL.md files
# Validates: directory structure, frontmatter parsing, required fields

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

source "$SCRIPT_DIR/test-helpers.sh"
require_jq

AETHER_UTILS="$REPO_ROOT/.aether/aether-utils.sh"
COLONY_SKILLS_DIR="$REPO_ROOT/.aether/skills/colony"

# All 10 colony skill names
COLONY_SKILLS="colony-interaction colony-visuals pheromone-visibility build-discipline colony-lifecycle context-management state-safety error-presentation pheromone-protocol worker-priming"

# ============================================================================
# Tests
# ============================================================================

test_all_directories_exist() {
    for skill in $COLONY_SKILLS; do
        if ! assert_dir_exists "$COLONY_SKILLS_DIR/$skill"; then
            test_fail "Directory exists" "Missing: $COLONY_SKILLS_DIR/$skill"
            return 1
        fi
    done
    return 0
}

test_all_skill_files_exist() {
    for skill in $COLONY_SKILLS; do
        if ! assert_file_exists "$COLONY_SKILLS_DIR/$skill/SKILL.md"; then
            test_fail "SKILL.md exists" "Missing: $COLONY_SKILLS_DIR/$skill/SKILL.md"
            return 1
        fi
    done
    return 0
}

test_frontmatter_parses_ok() {
    for skill in $COLONY_SKILLS; do
        local result
        result=$(bash "$AETHER_UTILS" skill-parse-frontmatter "$COLONY_SKILLS_DIR/$skill/SKILL.md" 2>&1)
        if ! assert_ok_true "$result"; then
            test_fail "ok:true for $skill" "Got: $result"
            return 1
        fi
    done
    return 0
}

test_names_match_directories() {
    for skill in $COLONY_SKILLS; do
        local result name
        result=$(bash "$AETHER_UTILS" skill-parse-frontmatter "$COLONY_SKILLS_DIR/$skill/SKILL.md" 2>&1)
        name=$(echo "$result" | jq -r '.result.name')
        if [[ "$name" != "$skill" ]]; then
            test_fail "name=$skill" "Got name=$name"
            return 1
        fi
    done
    return 0
}

test_all_types_are_colony() {
    for skill in $COLONY_SKILLS; do
        local result skill_type
        result=$(bash "$AETHER_UTILS" skill-parse-frontmatter "$COLONY_SKILLS_DIR/$skill/SKILL.md" 2>&1)
        skill_type=$(echo "$result" | jq -r '.result.type')
        if [[ "$skill_type" != "colony" ]]; then
            test_fail "type=colony for $skill" "Got type=$skill_type"
            return 1
        fi
    done
    return 0
}

test_descriptions_start_with_use_when() {
    for skill in $COLONY_SKILLS; do
        local result desc
        result=$(bash "$AETHER_UTILS" skill-parse-frontmatter "$COLONY_SKILLS_DIR/$skill/SKILL.md" 2>&1)
        desc=$(echo "$result" | jq -r '.result.description')
        case "$desc" in
            "Use when"*) ;;
            *)
                test_fail "description starts with 'Use when' for $skill" "Got: $desc"
                return 1
                ;;
        esac
    done
    return 0
}

test_domains_are_arrays() {
    for skill in $COLONY_SKILLS; do
        local result domains_len
        result=$(bash "$AETHER_UTILS" skill-parse-frontmatter "$COLONY_SKILLS_DIR/$skill/SKILL.md" 2>&1)
        domains_len=$(echo "$result" | jq '.result.domains | length')
        if [[ "$domains_len" -lt 1 ]]; then
            test_fail "domains non-empty for $skill" "Got length=$domains_len"
            return 1
        fi
    done
    return 0
}

test_agent_roles_are_arrays() {
    for skill in $COLONY_SKILLS; do
        local result roles_len
        result=$(bash "$AETHER_UTILS" skill-parse-frontmatter "$COLONY_SKILLS_DIR/$skill/SKILL.md" 2>&1)
        roles_len=$(echo "$result" | jq '.result.agent_roles | length')
        if [[ "$roles_len" -lt 1 ]]; then
            test_fail "agent_roles non-empty for $skill" "Got length=$roles_len"
            return 1
        fi
    done
    return 0
}

test_versions_are_1_0() {
    for skill in $COLONY_SKILLS; do
        local result version
        result=$(bash "$AETHER_UTILS" skill-parse-frontmatter "$COLONY_SKILLS_DIR/$skill/SKILL.md" 2>&1)
        version=$(echo "$result" | jq -r '.result.version')
        if [[ "$version" != "1.0" ]]; then
            test_fail "version=1.0 for $skill" "Got version=$version"
            return 1
        fi
    done
    return 0
}

test_body_has_sufficient_content() {
    for skill in $COLONY_SKILLS; do
        local file="$COLONY_SKILLS_DIR/$skill/SKILL.md"
        # Count lines after the closing frontmatter ---
        local body_lines
        body_lines=$(awk '/^---$/{c++; if(c==2){found=1; next}} found{print}' "$file" | wc -l | tr -d ' ')
        # 200 words is roughly 20+ lines minimum
        if [[ "$body_lines" -lt 10 ]]; then
            test_fail "body >= 10 lines for $skill" "Got $body_lines lines"
            return 1
        fi
    done
    return 0
}

# ============================================================================
# Run all tests
# ============================================================================

log_info "Testing 10 colony SKILL.md files"
log ""

run_test test_all_directories_exist "All 10 colony skill directories exist"
run_test test_all_skill_files_exist "All 10 colony SKILL.md files exist"
run_test test_frontmatter_parses_ok "All frontmatter parses with ok:true"
run_test test_names_match_directories "Skill names match directory names"
run_test test_all_types_are_colony "All types are 'colony'"
run_test test_descriptions_start_with_use_when "All descriptions start with 'Use when'"
run_test test_domains_are_arrays "All domains are non-empty arrays"
run_test test_agent_roles_are_arrays "All agent_roles are non-empty arrays"
run_test test_versions_are_1_0 "All versions are '1.0'"
run_test test_body_has_sufficient_content "All bodies have sufficient content (>= 10 lines)"

test_summary
