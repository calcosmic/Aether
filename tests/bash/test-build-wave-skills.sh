#!/usr/bin/env bash
# Tests that build-wave.md contains per-worker skill matching and injection
# Task 3.2: Verify skill-match/skill-inject integration in the builder spawn section

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

source "$SCRIPT_DIR/test-helpers.sh"

BUILD_WAVE="$REPO_ROOT/.aether/docs/command-playbooks/build-wave.md"

# ============================================================================
# Test 1: build-wave.md contains skill-match call for builder workers
# ============================================================================
test_contains_skill_match() {
    test_start "build-wave.md contains skill-match call for builder workers"

    if grep -q 'skill-match.*"builder"' "$BUILD_WAVE"; then
        test_pass
    else
        test_fail "Expected skill-match call with 'builder' role" "Not found in build-wave.md"
    fi
}

# ============================================================================
# Test 2: build-wave.md contains skill-inject call
# ============================================================================
test_contains_skill_inject() {
    test_start "build-wave.md contains skill-inject call"

    if grep -q 'skill-inject' "$BUILD_WAVE"; then
        test_pass
    else
        test_fail "Expected skill-inject call" "Not found in build-wave.md"
    fi
}

# ============================================================================
# Test 3: Builder Worker Prompt template includes { skill_section }
# ============================================================================
test_prompt_contains_skill_section() {
    test_start "Builder Worker Prompt includes skill_section placeholder"

    if grep -q '{ skill_section' "$BUILD_WAVE"; then
        test_pass
    else
        test_fail "Expected { skill_section } in Builder Worker Prompt" "Not found in build-wave.md"
    fi
}

# ============================================================================
# Test 4: { skill_section } appears AFTER { prompt_section } in the prompt
# ============================================================================
test_skill_section_after_prompt_section() {
    test_start "skill_section appears after prompt_section in Builder Worker Prompt"

    local prompt_line skill_line
    prompt_line=$(grep -n '{ prompt_section }' "$BUILD_WAVE" | head -1 | cut -d: -f1)
    skill_line=$(grep -n '{ skill_section' "$BUILD_WAVE" | head -1 | cut -d: -f1)

    if [[ -z "$prompt_line" || -z "$skill_line" ]]; then
        test_fail "Both placeholders must exist" "prompt_line=$prompt_line skill_line=$skill_line"
        return
    fi

    if [[ "$skill_line" -gt "$prompt_line" ]]; then
        test_pass
    else
        test_fail "skill_section (line $skill_line) should be after prompt_section (line $prompt_line)" "skill_section is before or at same line"
    fi
}

# ============================================================================
# Test 5: skill-match call appears BEFORE the Builder Worker Prompt template
# ============================================================================
test_skill_match_before_prompt_template() {
    test_start "skill-match call appears before Builder Worker Prompt template"

    local match_line prompt_template_line
    match_line=$(grep -n 'skill-match' "$BUILD_WAVE" | head -1 | cut -d: -f1)
    prompt_template_line=$(grep -n 'Builder Worker Prompt' "$BUILD_WAVE" | head -1 | cut -d: -f1)

    if [[ -z "$match_line" || -z "$prompt_template_line" ]]; then
        test_fail "Both skill-match and Builder Worker Prompt must exist" "match_line=$match_line prompt_template_line=$prompt_template_line"
        return
    fi

    if [[ "$match_line" -lt "$prompt_template_line" ]]; then
        test_pass
    else
        test_fail "skill-match (line $match_line) should be before Builder Worker Prompt (line $prompt_template_line)" "skill-match is after the prompt template"
    fi
}

# ============================================================================
# Test 6: Per-worker skill count display line exists
# ============================================================================
test_skill_count_display() {
    test_start "Per-worker skill count display line exists"

    if grep -q 'Skills:.*colony.*domain.*loaded' "$BUILD_WAVE"; then
        test_pass
    else
        test_fail "Expected per-worker skill count display line" "Not found in build-wave.md"
    fi
}

# ============================================================================
# Test 7: Skill injection is NON-BLOCKING (failure handling present)
# ============================================================================
test_skill_injection_non_blocking() {
    test_start "Skill injection is non-blocking (graceful failure handling)"

    if grep -q '2>/dev/null' "$BUILD_WAVE" && grep -q 'skill_section' "$BUILD_WAVE"; then
        # Check that there's error handling around skill calls
        if grep -q 'skill.*2>/dev/null' "$BUILD_WAVE"; then
            test_pass
        else
            test_fail "Expected 2>/dev/null on skill commands" "Not found"
        fi
    else
        test_fail "Expected non-blocking skill injection pattern" "Missing skill_section or error suppression"
    fi
}

# ============================================================================
# Run all tests
# ============================================================================

log_info "Testing build-wave.md skill injection integration"
log_info "Build wave file: $BUILD_WAVE"

test_contains_skill_match
test_contains_skill_inject
test_prompt_contains_skill_section
test_skill_section_after_prompt_section
test_skill_match_before_prompt_template
test_skill_count_display
test_skill_injection_non_blocking

test_summary
