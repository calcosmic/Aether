#!/usr/bin/env bash
# Tests for colony-prime hive wisdom entry limits and end-to-end domain scoping
# Supplements test-colony-prime-hive-domain.sh with:
#   - Compact mode caps hive entries at 3
#   - Normal mode caps hive entries at 5
#   - Multi-domain filtering: only matching domains appear
#   - End-to-end: hive-init -> hive-store -> registry -> colony-prime

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

source "$SCRIPT_DIR/test-helpers.sh"
require_jq

AETHER_UTILS="$REPO_ROOT/.aether/aether-utils.sh"

# ============================================================================
# Helper: Create a test colony environment (reused pattern)
# ============================================================================
setup_colony_env() {
    local tmpdir
    tmpdir=$(mktemp -d)
    local aether_dir="$tmpdir/.aether"
    local data_dir="$aether_dir/data"
    mkdir -p "$data_dir"

    local iso_date
    iso_date=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

    cat > "$aether_dir/QUEEN.md" << QUEENEOF
# QUEEN.md --- Colony Wisdom

> Last evolved: $iso_date
> Colonies contributed: 0
> Wisdom version: 1.0.0

---

## Philosophies

*No philosophies recorded yet*

---

## Patterns

*No patterns recorded yet*

---

## Redirects

*No redirects recorded yet*

---

## Stack Wisdom

*No stack wisdom recorded yet*

---

## Decrees

*No decrees recorded yet*

---

## Evolution Log

| Date | Colony | Change | Details |
|------|--------|--------|---------|

---

<!-- METADATA {"version":"1.0.0","last_evolved":"$iso_date","colonies_contributed":[],"promotion_thresholds":{"philosophy":1,"pattern":1,"redirect":1,"stack":1,"decree":0},"stats":{"total_philosophies":0,"total_patterns":0,"total_redirects":0,"total_stack_entries":0,"total_decrees":0}} -->
QUEENEOF

    cat > "$data_dir/COLONY_STATE.json" << 'STATEEOF'
{
  "session_id": "test_hive_limits",
  "goal": "test hive limits",
  "state": "BUILDING",
  "current_phase": 1,
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

    cat > "$data_dir/pheromones.json" << 'PHEREOF'
{
  "signals": [],
  "version": "1.0.0"
}
PHEREOF

    echo "$tmpdir"
}

# Helper: run colony-prime against a test env
run_colony_prime() {
    local tmpdir="$1"
    shift
    HOME="$tmpdir" AETHER_ROOT="$tmpdir" DATA_DIR="$tmpdir/.aether/data" \
        bash "$AETHER_UTILS" colony-prime "$@" 2>/dev/null
}

# Helper: extract prompt_section content from colony-prime output
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

# Helper: create hive wisdom with N entries, each with specified domain
create_hive_wisdom_n() {
    local tmpdir="$1"
    local domain="$2"   # comma-separated
    local count="$3"

    local hive_dir="$tmpdir/.aether/hive"
    mkdir -p "$hive_dir"

    python3 -c "
import json
domain_tags = '$domain'.split(',') if '$domain' else []
domain_tags = [d.strip() for d in domain_tags if d.strip()]

entries = []
for i in range(1, $count + 1):
    entries.append({
        'id': f'hive-entry-{i}',
        'text': f'Wisdom number {i} about testing',
        'category': 'PATTERN',
        'confidence': round(0.5 + (i * 0.01), 2),
        'domain_tags': domain_tags,
        'source_repos': ['/test/repo'],
        'validated_count': i,
        'access_count': 0,
        'created_at': '2026-03-01T00:00:00Z',
        'last_accessed': None,
        'abstracted': True
    })

with open('$hive_dir/wisdom.json', 'w') as f:
    json.dump({
        'version': '1.0.0',
        'created_at': '2026-03-01T00:00:00Z',
        'last_updated': '2026-03-01T00:00:00Z',
        'entries': entries
    }, f, indent=2)
"
}

# Helper: create hive wisdom with entries in DIFFERENT domains
create_hive_mixed_domains() {
    local tmpdir="$1"

    local hive_dir="$tmpdir/.aether/hive"
    mkdir -p "$hive_dir"

    python3 -c "
import json

entries = [
    {
        'id': 'web-entry-1',
        'text': 'Web pattern: use semantic HTML',
        'category': 'PATTERN',
        'confidence': 0.9,
        'domain_tags': ['web'],
        'source_repos': ['/test/web-app'],
        'validated_count': 3,
        'access_count': 0,
        'created_at': '2026-03-01T00:00:00Z',
        'last_accessed': None,
        'abstracted': True
    },
    {
        'id': 'web-entry-2',
        'text': 'Web pattern: responsive breakpoints',
        'category': 'PATTERN',
        'confidence': 0.85,
        'domain_tags': ['web'],
        'source_repos': ['/test/web-app'],
        'validated_count': 2,
        'access_count': 0,
        'created_at': '2026-03-01T00:00:00Z',
        'last_accessed': None,
        'abstracted': True
    },
    {
        'id': 'api-entry-1',
        'text': 'API pattern: version endpoints',
        'category': 'PATTERN',
        'confidence': 0.88,
        'domain_tags': ['api'],
        'source_repos': ['/test/api-service'],
        'validated_count': 4,
        'access_count': 0,
        'created_at': '2026-03-01T00:00:00Z',
        'last_accessed': None,
        'abstracted': True
    },
    {
        'id': 'cli-entry-1',
        'text': 'CLI pattern: use structured output',
        'category': 'PATTERN',
        'confidence': 0.82,
        'domain_tags': ['cli'],
        'source_repos': ['/test/cli-tool'],
        'validated_count': 2,
        'access_count': 0,
        'created_at': '2026-03-01T00:00:00Z',
        'last_accessed': None,
        'abstracted': True
    }
]

with open('$hive_dir/wisdom.json', 'w') as f:
    json.dump({
        'version': '1.0.0',
        'created_at': '2026-03-01T00:00:00Z',
        'last_updated': '2026-03-01T00:00:00Z',
        'entries': entries
    }, f, indent=2)
"
}

# Helper: create registry with domain tags
create_registry() {
    local tmpdir="$1"
    local repo_path="$2"
    local tags="$3"  # JSON array

    mkdir -p "$tmpdir/.aether"

    cat > "$tmpdir/.aether/registry.json" << REGEOF
{
  "schema_version": 1,
  "repos": [
    {
      "path": "$repo_path",
      "version": "2.0.0",
      "registered_at": "2026-03-01T00:00:00Z",
      "updated_at": "2026-03-01T00:00:00Z",
      "domain_tags": $tags,
      "last_colony_goal": "test",
      "active_colony": true
    }
  ]
}
REGEOF
}

# Helper: run aether-utils subcommand with isolated HOME
run_aether_cmd() {
    local tmpdir="$1"
    local subcmd="$2"
    shift 2
    HOME="$tmpdir" bash "$AETHER_UTILS" "$subcmd" "$@" 2>/dev/null
}

# Helper: count occurrences of a pattern in text
count_occurrences() {
    local text="$1"
    local pattern="$2"
    echo "$text" | grep -c "$pattern" || echo "0"
}


# ============================================================================
# Test 1: Normal mode limits hive entries to 5
# ============================================================================
test_normal_mode_limits_to_5() {
    local tmpdir
    tmpdir=$(setup_colony_env)
    # Create 8 entries — more than the normal limit of 5
    create_hive_wisdom_n "$tmpdir" "web" 8

    local result
    result=$(run_colony_prime "$tmpdir")

    if ! assert_contains "$result" '"ok":true'; then
        test_fail "Expected ok=true" ""
        rm -rf "$tmpdir"
        return 1
    fi

    local prompt
    prompt=$(echo "$result" | get_prompt_section)

    # Count entries in HIVE WISDOM section (each entry starts with "- [")
    local entry_count
    # Extract just the hive section
    local hive_section
    hive_section=$(echo "$prompt" | sed -n '/HIVE WISDOM/,/END HIVE WISDOM/p')
    entry_count=$(echo "$hive_section" | grep -c '^\- ' || echo "0")

    if [[ "$entry_count" -gt 5 ]]; then
        test_fail "Normal mode should limit to 5 entries, got $entry_count" "$hive_section"
        rm -rf "$tmpdir"
        return 1
    fi

    if [[ "$entry_count" -lt 1 ]]; then
        test_fail "Expected at least 1 entry, got $entry_count" "$hive_section"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

# ============================================================================
# Test 2: Compact mode limits hive entries to 3
# ============================================================================
test_compact_mode_limits_to_3() {
    local tmpdir
    tmpdir=$(setup_colony_env)
    # Create 8 entries — more than the compact limit of 3
    create_hive_wisdom_n "$tmpdir" "web" 8

    local result
    result=$(run_colony_prime "$tmpdir" --compact)

    if ! assert_contains "$result" '"ok":true'; then
        test_fail "Expected ok=true with --compact" ""
        rm -rf "$tmpdir"
        return 1
    fi

    local prompt
    prompt=$(echo "$result" | get_prompt_section)

    local hive_section
    hive_section=$(echo "$prompt" | sed -n '/HIVE WISDOM/,/END HIVE WISDOM/p')
    local entry_count
    entry_count=$(echo "$hive_section" | grep -c '^\- ' || echo "0")

    if [[ "$entry_count" -gt 3 ]]; then
        test_fail "Compact mode should limit to 3 entries, got $entry_count" "$hive_section"
        rm -rf "$tmpdir"
        return 1
    fi

    if [[ "$entry_count" -lt 1 ]]; then
        test_fail "Expected at least 1 entry in compact mode, got $entry_count" "$hive_section"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

# ============================================================================
# Test 3: Multi-domain filtering — only matching domain entries shown
# ============================================================================
test_multi_domain_only_matching() {
    local tmpdir
    tmpdir=$(setup_colony_env)
    create_hive_mixed_domains "$tmpdir"
    # Registry tags: only "web" — should exclude api and cli entries
    create_registry "$tmpdir" "$tmpdir" '["web"]'

    local result
    result=$(run_colony_prime "$tmpdir")

    if ! assert_contains "$result" '"ok":true'; then
        test_fail "Expected ok=true" ""
        rm -rf "$tmpdir"
        return 1
    fi

    local prompt
    prompt=$(echo "$result" | get_prompt_section)

    # Should contain web entries
    if ! assert_contains "$prompt" "semantic HTML"; then
        test_fail "Expected web entry 'semantic HTML' to appear" "$prompt"
        rm -rf "$tmpdir"
        return 1
    fi

    # Should NOT contain api or cli entries
    if assert_contains "$prompt" "version endpoints"; then
        test_fail "Should NOT contain api entry 'version endpoints'" "$prompt"
        rm -rf "$tmpdir"
        return 1
    fi

    if assert_contains "$prompt" "structured output"; then
        test_fail "Should NOT contain cli entry 'structured output'" "$prompt"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

# ============================================================================
# Test 4: End-to-end: hive-init -> hive-store -> registry -> colony-prime
# ============================================================================
test_end_to_end_hive_flow() {
    local tmpdir
    tmpdir=$(setup_colony_env)

    # Step 1: Initialize hive
    local init_result
    init_result=$(run_aether_cmd "$tmpdir" hive-init)
    if ! assert_contains "$init_result" '"initialized":true'; then
        test_fail "hive-init should return initialized:true" "$init_result"
        rm -rf "$tmpdir"
        return 1
    fi

    # Step 2: Store entries with different domains via hive-store
    local store1
    store1=$(run_aether_cmd "$tmpdir" hive-store \
        --text "Always validate inputs" \
        --domain "web,api" \
        --source-repo "/test/project-a" \
        --confidence "0.85" \
        --category "PATTERN")
    if ! assert_contains "$store1" '"action":"stored"'; then
        test_fail "First hive-store should return stored" "$store1"
        rm -rf "$tmpdir"
        return 1
    fi

    local store2
    store2=$(run_aether_cmd "$tmpdir" hive-store \
        --text "Use database migrations" \
        --domain "api" \
        --source-repo "/test/project-b" \
        --confidence "0.90" \
        --category "PATTERN")
    if ! assert_contains "$store2" '"action":"stored"'; then
        test_fail "Second hive-store should return stored" "$store2"
        rm -rf "$tmpdir"
        return 1
    fi

    local store3
    store3=$(run_aether_cmd "$tmpdir" hive-store \
        --text "Cache static assets aggressively" \
        --domain "web" \
        --source-repo "/test/project-c" \
        --confidence "0.80" \
        --category "PATTERN")
    if ! assert_contains "$store3" '"action":"stored"'; then
        test_fail "Third hive-store should return stored" "$store3"
        rm -rf "$tmpdir"
        return 1
    fi

    # Step 3: Create registry with domain "web" for this repo
    create_registry "$tmpdir" "$tmpdir" '["web"]'

    # Step 4: Run colony-prime — should only see web-tagged entries
    local result
    result=$(run_colony_prime "$tmpdir")

    if ! assert_contains "$result" '"ok":true'; then
        test_fail "colony-prime should return ok:true" ""
        rm -rf "$tmpdir"
        return 1
    fi

    local prompt
    prompt=$(echo "$result" | get_prompt_section)

    # Should contain the web-domain entry
    if ! assert_contains "$prompt" "validate inputs"; then
        test_fail "Expected 'validate inputs' (web+api domain) in output" "$prompt"
        rm -rf "$tmpdir"
        return 1
    fi

    if ! assert_contains "$prompt" "Cache static assets"; then
        test_fail "Expected 'Cache static assets' (web domain) in output" "$prompt"
        rm -rf "$tmpdir"
        return 1
    fi

    # The api-only entry should NOT appear (registry has web, not api)
    if assert_contains "$prompt" "database migrations"; then
        test_fail "Should NOT contain api-only entry 'database migrations'" "$prompt"
        rm -rf "$tmpdir"
        return 1
    fi

    # Should show domain label
    if ! assert_contains "$prompt" "Domain:"; then
        test_fail "Expected 'Domain:' in HIVE WISDOM header" "$prompt"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

# ============================================================================
# Test 5: Normal mode with exactly 5 entries shows all 5
# ============================================================================
test_normal_mode_shows_exact_5() {
    local tmpdir
    tmpdir=$(setup_colony_env)
    create_hive_wisdom_n "$tmpdir" "web" 5

    local result
    result=$(run_colony_prime "$tmpdir")

    if ! assert_contains "$result" '"ok":true'; then
        test_fail "Expected ok=true" ""
        rm -rf "$tmpdir"
        return 1
    fi

    local prompt
    prompt=$(echo "$result" | get_prompt_section)

    local hive_section
    hive_section=$(echo "$prompt" | sed -n '/HIVE WISDOM/,/END HIVE WISDOM/p')
    local entry_count
    entry_count=$(echo "$hive_section" | grep -c '^\- ' || echo "0")

    if [[ "$entry_count" -ne 5 ]]; then
        test_fail "Normal mode with 5 entries should show all 5, got $entry_count" "$hive_section"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

# ============================================================================
# Test 6: Compact mode with exactly 3 entries shows all 3
# ============================================================================
test_compact_mode_shows_exact_3() {
    local tmpdir
    tmpdir=$(setup_colony_env)
    create_hive_wisdom_n "$tmpdir" "web" 3

    local result
    result=$(run_colony_prime "$tmpdir" --compact)

    if ! assert_contains "$result" '"ok":true'; then
        test_fail "Expected ok=true with --compact" ""
        rm -rf "$tmpdir"
        return 1
    fi

    local prompt
    prompt=$(echo "$result" | get_prompt_section)

    local hive_section
    hive_section=$(echo "$prompt" | sed -n '/HIVE WISDOM/,/END HIVE WISDOM/p')
    local entry_count
    entry_count=$(echo "$hive_section" | grep -c '^\- ' || echo "0")

    if [[ "$entry_count" -ne 3 ]]; then
        test_fail "Compact mode with 3 entries should show all 3, got $entry_count" "$hive_section"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

# ============================================================================
# Run tests
# ============================================================================

log_info "Running colony-prime hive limits and end-to-end tests"
log_info "Repo root: $REPO_ROOT"

run_test test_normal_mode_limits_to_5 "Normal mode limits hive entries to 5"
run_test test_compact_mode_limits_to_3 "Compact mode limits hive entries to 3"
run_test test_multi_domain_only_matching "Multi-domain filtering shows only matching entries"
run_test test_end_to_end_hive_flow "End-to-end: hive-init -> hive-store -> registry -> colony-prime"
run_test test_normal_mode_shows_exact_5 "Normal mode with exactly 5 entries shows all 5"
run_test test_compact_mode_shows_exact_3 "Compact mode with exactly 3 entries shows all 3"

test_summary
