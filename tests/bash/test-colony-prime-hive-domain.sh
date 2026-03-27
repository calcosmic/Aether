#!/usr/bin/env bash
# Tests for colony-prime domain-scoped hive retrieval (Task 3.1)
# colony-prime should use hive-read with domain tags from registry,
# with fallback to eternal memory.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

source "$SCRIPT_DIR/test-helpers.sh"
require_jq

AETHER_UTILS="$REPO_ROOT/.aether/aether-utils.sh"

# ============================================================================
# Helper: Create a test colony environment
# ============================================================================
setup_colony_env() {
    local tmpdir
    tmpdir=$(mktemp -d)
    local aether_dir="$tmpdir/.aether"
    local data_dir="$aether_dir/data"
    mkdir -p "$data_dir"

    local iso_date
    iso_date=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

    # Create QUEEN.md
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

    # Create COLONY_STATE.json
    cat > "$data_dir/COLONY_STATE.json" << 'STATEEOF'
{
  "session_id": "test_hive_domain",
  "goal": "test hive domain scoping",
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

# Helper: create hive wisdom file at ~/.aether/hive/wisdom.json
create_hive_wisdom() {
    local tmpdir="$1"
    local domain="${2:-}"   # comma-separated domain tags
    local count="${3:-3}"

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
        'text': f'Hive wisdom entry {i} for testing',
        'category': 'PATTERN',
        'confidence': round(0.5 + (i * 0.05), 2),
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

# Helper: create registry with domain tags for a repo
create_registry() {
    local tmpdir="$1"
    local repo_path="$2"
    local tags="$3"  # JSON array like ["web","api"]

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

# Helper: create eternal memory (legacy fallback)
create_eternal_memory() {
    local tmpdir="$1"
    local signal_count="${2:-3}"
    local eternal_dir="$tmpdir/.aether/eternal"
    mkdir -p "$eternal_dir"

    python3 -c "
import json
signals = []
for i in range(1, $signal_count + 1):
    signals.append({
        'content': f'Eternal wisdom signal {i}',
        'type': 'REDIRECT',
        'strength': round(0.5 + (i * 0.1), 1),
        'source_colony': f'colony-{i}',
        'promoted_at': '2026-03-01T00:00:00Z'
    })
with open('$eternal_dir/memory.json', 'w') as f:
    json.dump({
        'version': '1.0',
        'created_at': '2026-02-17T22:25:34Z',
        'colonies': [],
        'high_value_signals': signals,
        'cross_session_patterns': []
    }, f, indent=2)
"
}

# ============================================================================
# Tests
# ============================================================================

test_hive_read_used_when_hive_exists() {
    # When hive/wisdom.json exists, colony-prime should use hive-read output
    local tmpdir
    tmpdir=$(setup_colony_env)
    create_hive_wisdom "$tmpdir" "web" 3

    local result
    result=$(run_colony_prime "$tmpdir")

    if ! assert_contains "$result" '"ok":true'; then
        test_fail "Expected ok=true in output" ""
        rm -rf "$tmpdir"
        return 1
    fi

    local prompt
    prompt=$(echo "$result" | get_prompt_section)

    if ! assert_contains "$prompt" "HIVE WISDOM"; then
        test_fail "Expected HIVE WISDOM section in prompt" "$prompt"
        rm -rf "$tmpdir"
        return 1
    fi

    # Should contain hive-read formatted entries
    if ! assert_contains "$prompt" "Hive wisdom entry"; then
        test_fail "Expected hive-read entries in prompt" "$prompt"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

test_domain_label_in_header() {
    # When domain tags are found, header should show "Domain: web, api"
    local tmpdir
    tmpdir=$(setup_colony_env)
    create_hive_wisdom "$tmpdir" "web,api" 2
    create_registry "$tmpdir" "$tmpdir" '["web","api"]'

    local result
    result=$(run_colony_prime "$tmpdir")

    if ! assert_contains "$result" '"ok":true'; then
        test_fail "Expected ok=true" ""
        rm -rf "$tmpdir"
        return 1
    fi

    local prompt
    prompt=$(echo "$result" | get_prompt_section)

    if ! assert_contains "$prompt" "Domain:"; then
        test_fail "Expected 'Domain:' in HIVE WISDOM header" "$prompt"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

test_all_domains_when_no_registry() {
    # When no registry entry exists, header should show "All Domains"
    local tmpdir
    tmpdir=$(setup_colony_env)
    create_hive_wisdom "$tmpdir" "" 2

    # No registry file -> no domain filtering

    local result
    result=$(run_colony_prime "$tmpdir")

    if ! assert_contains "$result" '"ok":true'; then
        test_fail "Expected ok=true" ""
        rm -rf "$tmpdir"
        return 1
    fi

    local prompt
    prompt=$(echo "$result" | get_prompt_section)

    if ! assert_contains "$prompt" "All Domains"; then
        test_fail "Expected 'All Domains' in HIVE WISDOM header when no registry" "$prompt"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

test_eternal_fallback_when_no_hive() {
    # When hive/wisdom.json doesn't exist but eternal memory does, use eternal fallback
    local tmpdir
    tmpdir=$(setup_colony_env)
    # No hive wisdom, but eternal memory exists
    create_eternal_memory "$tmpdir" 3

    local result
    result=$(run_colony_prime "$tmpdir")

    if ! assert_contains "$result" '"ok":true'; then
        test_fail "Expected ok=true" ""
        rm -rf "$tmpdir"
        return 1
    fi

    local prompt
    prompt=$(echo "$result" | get_prompt_section)

    if ! assert_contains "$prompt" "HIVE WISDOM"; then
        test_fail "Expected HIVE WISDOM section via eternal fallback" "$prompt"
        rm -rf "$tmpdir"
        return 1
    fi

    # Should contain eternal signal content
    if ! assert_contains "$prompt" "Eternal wisdom signal"; then
        test_fail "Expected eternal signal content in fallback" "$prompt"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

test_no_hive_section_when_both_empty() {
    # When neither hive nor eternal exists, no HIVE WISDOM section
    local tmpdir
    tmpdir=$(setup_colony_env)

    local result
    result=$(run_colony_prime "$tmpdir")

    if ! assert_contains "$result" '"ok":true'; then
        test_fail "Expected ok=true" ""
        rm -rf "$tmpdir"
        return 1
    fi

    local prompt
    prompt=$(echo "$result" | get_prompt_section)

    if assert_contains "$prompt" "HIVE WISDOM"; then
        test_fail "Should NOT have HIVE WISDOM when both hive and eternal are absent" "$prompt"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

test_hive_preferred_over_eternal() {
    # When both hive and eternal exist, hive should be used (not eternal)
    local tmpdir
    tmpdir=$(setup_colony_env)
    create_hive_wisdom "$tmpdir" "" 2
    create_eternal_memory "$tmpdir" 3

    local result
    result=$(run_colony_prime "$tmpdir")

    if ! assert_contains "$result" '"ok":true'; then
        test_fail "Expected ok=true" ""
        rm -rf "$tmpdir"
        return 1
    fi

    local prompt
    prompt=$(echo "$result" | get_prompt_section)

    # Should have hive content, NOT eternal content
    if ! assert_contains "$prompt" "Hive wisdom entry"; then
        test_fail "Expected hive wisdom entries (preferred source)" "$prompt"
        rm -rf "$tmpdir"
        return 1
    fi

    if assert_contains "$prompt" "Eternal wisdom signal"; then
        test_fail "Should NOT have eternal signals when hive has entries" "$prompt"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

test_eternal_fallback_when_hive_empty() {
    # When hive exists but has 0 matching entries, fall back to eternal
    local tmpdir
    tmpdir=$(setup_colony_env)
    # Create hive with domain "mobile" but registry has "web" -> 0 matches
    create_hive_wisdom "$tmpdir" "mobile" 3
    create_registry "$tmpdir" "$tmpdir" '["web"]'
    create_eternal_memory "$tmpdir" 2

    local result
    result=$(run_colony_prime "$tmpdir")

    if ! assert_contains "$result" '"ok":true'; then
        test_fail "Expected ok=true" ""
        rm -rf "$tmpdir"
        return 1
    fi

    local prompt
    prompt=$(echo "$result" | get_prompt_section)

    # Should fall back to eternal since hive returned 0 domain-matched entries
    if ! assert_contains "$prompt" "Eternal wisdom signal"; then
        test_fail "Expected eternal fallback when hive has no matching domain entries" "$prompt"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

# ============================================================================
# Run tests
# ============================================================================

log_info "Running colony-prime domain-scoped hive retrieval tests"
log_info "Repo root: $REPO_ROOT"

run_test test_hive_read_used_when_hive_exists "Hive-read used when hive/wisdom.json exists"
run_test test_domain_label_in_header "Domain tags shown in HIVE WISDOM header"
run_test test_all_domains_when_no_registry "All Domains label when no registry entry"
run_test test_eternal_fallback_when_no_hive "Eternal fallback when no hive wisdom"
run_test test_no_hive_section_when_both_empty "No HIVE WISDOM when both sources empty"
run_test test_hive_preferred_over_eternal "Hive preferred over eternal when both exist"
run_test test_eternal_fallback_when_hive_empty "Eternal fallback when hive has no matching entries"

test_summary
