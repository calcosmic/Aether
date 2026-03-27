# Aether Skills Layer Implementation Plan

> **For agentic workers:** Use /ant:build to implement this plan phase by phase. Each task uses checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a skills system to Aether that smart-matches colony and domain skills to workers based on codebase detection, pheromone signals, and worker roles.

**Architecture:** Skills live at `~/.aether/skills/` (installed) and `.aether/skills/` (source). A new `skills.sh` utility handles indexing, detection, matching, and injection. Skills are injected into worker prompts as a separate section OUTSIDE colony-prime, with their own 12K char budget. Colony-prime remains unchanged.

**Tech Stack:** Bash (aether-utils.sh + skills.sh utility), Node.js (setupHub changes in cli.js), AVA + bash tests

**Spec:** `docs/specs/2026-03-22-aether-skills-layer-design.md`

---

## File Structure

### New Files

| File | Responsibility |
|------|---------------|
| `.aether/utils/skills.sh` | Core skill engine — index, detect, match, inject, cache |
| `.aether/skills/colony/*/SKILL.md` | 10 colony skill definitions |
| `.aether/skills/domain/*/SKILL.md` | 18 starter domain skill definitions |
| `.aether/skills/colony/.manifest.json` | Aether-owned colony skills list |
| `.aether/skills/domain/.manifest.json` | Aether-owned domain skills list |
| `.claude/commands/ant/skill-create.md` | Skill creation wizard command |
| `tests/bash/test-skills.sh` | Bash integration tests for skills engine |
| `tests/unit/skills.test.js` | AVA unit tests for skill subcommands |

### Modified Files

| File | Change |
|------|--------|
| `.aether/aether-utils.sh` (lines 26-32) | Add `source skills.sh` |
| `.aether/aether-utils.sh` (dispatch table ~line 980) | Add 8 skill subcommand cases |
| `.aether/aether-utils.sh` (help case ~line 988) | Add skill commands to help output |
| `.aether/docs/command-playbooks/build-context.md` | Add skill index + detect step |
| `.aether/docs/command-playbooks/build-wave.md` | Add per-worker skill matching + injection |
| `bin/cli.js` (setupHub) | Add skill directory sync to hub |
| `package.json` | Skills already covered by `.aether/` in files field |
| `CLAUDE.md` | Document skills system |

---

## Task 1: Create skills.sh utility — frontmatter parser and index cache

**Files:**
- Create: `.aether/utils/skills.sh`
- Test: `tests/bash/test-skills.sh`

This is the foundation. Build the frontmatter parser and index cache before anything else.

- [ ] **Step 1: Write test for frontmatter parsing**

Create `tests/bash/test-skills.sh` with test setup and first test:

```bash
#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/test-helpers.sh"

# Setup isolated test environment
setup_test_env() {
    TEST_DIR=$(mktemp -d)
    SKILLS_DIR="$TEST_DIR/skills"
    mkdir -p "$SKILLS_DIR/colony/test-skill"
    mkdir -p "$SKILLS_DIR/domain/test-domain"

    # Create test colony skill
    cat > "$SKILLS_DIR/colony/test-skill/SKILL.md" << 'SKILL'
---
name: test-skill
description: Use when testing the skills system
type: colony
domains: [testing, quality]
agent_roles: [builder, watcher]
priority: normal
version: "1.0"
---

Test skill content here.
SKILL

    # Create test domain skill
    cat > "$SKILLS_DIR/domain/test-domain/SKILL.md" << 'SKILL'
---
name: test-domain
description: Use when working with test frameworks
type: domain
domains: [testing, frontend]
agent_roles: [builder]
detect_files: ["*.test.js", "jest.config.*"]
detect_packages: ["jest", "vitest"]
priority: normal
version: "1.0"
---

Domain skill content for testing.
SKILL

    export AETHER_SKILLS_DIR="$SKILLS_DIR"
}

cleanup_test_env() {
    rm -rf "$TEST_DIR"
}

# Source the utility
AETHER_UTILS_SOURCE="$SCRIPT_DIR/../../.aether/aether-utils.sh"

test_parse_frontmatter() {
    test_start "parse-frontmatter returns valid JSON for colony skill"
    setup_test_env

    local output
    output=$(bash "$AETHER_UTILS_SOURCE" skill-parse-frontmatter "$SKILLS_DIR/colony/test-skill/SKILL.md" 2>&1)
    local exit_code=$?

    assert_exit_code $exit_code 0
    assert_json_valid "$output"
    assert_json_has_field "$output" "name"

    local name
    name=$(echo "$output" | jq -r '.result.name')
    [[ "$name" == "test-skill" ]] && test_pass || test_fail "Expected name 'test-skill', got '$name'"

    cleanup_test_env
}

test_parse_frontmatter_domains() {
    test_start "parse-frontmatter extracts domains array"
    setup_test_env

    local output
    output=$(bash "$AETHER_UTILS_SOURCE" skill-parse-frontmatter "$SKILLS_DIR/colony/test-skill/SKILL.md" 2>&1)

    local domains
    domains=$(echo "$output" | jq -r '.result.domains | length')
    [[ "$domains" == "2" ]] && test_pass || test_fail "Expected 2 domains, got '$domains'"

    cleanup_test_env
}

test_parse_frontmatter_detect() {
    test_start "parse-frontmatter extracts detect_files and detect_packages"
    setup_test_env

    local output
    output=$(bash "$AETHER_UTILS_SOURCE" skill-parse-frontmatter "$SKILLS_DIR/domain/test-domain/SKILL.md" 2>&1)

    local detect_files
    detect_files=$(echo "$output" | jq -r '.result.detect_files | length')
    local detect_packages
    detect_packages=$(echo "$output" | jq -r '.result.detect_packages | length')

    [[ "$detect_files" == "2" && "$detect_packages" == "2" ]] && test_pass || test_fail "Expected 2 files + 2 packages, got $detect_files + $detect_packages"

    cleanup_test_env
}

# Run tests
test_parse_frontmatter
test_parse_frontmatter_domains
test_parse_frontmatter_detect

test_summary
```

- [ ] **Step 2: Run test to verify it fails**

Run: `bash tests/bash/test-skills.sh`
Expected: FAIL — `skill-parse-frontmatter` subcommand doesn't exist yet

- [ ] **Step 3: Create skills.sh with frontmatter parser**

Create `.aether/utils/skills.sh`:

```bash
#!/usr/bin/env bash
# Skills utility — frontmatter parsing, indexing, detection, matching, injection
# Sourced by aether-utils.sh

# Parse YAML-like frontmatter from a SKILL.md file
# Returns JSON with all frontmatter fields
_skill_parse_frontmatter() {
    local skill_file="$1"

    if [[ ! -f "$skill_file" ]]; then
        json_err "SKILL_NOT_FOUND" "Skill file not found: $skill_file"
        return 1
    fi

    local in_frontmatter=false
    local name="" description="" type="" priority="normal" version="1.0"
    local domains_raw="" agent_roles_raw="" detect_files_raw="" detect_packages_raw=""

    while IFS= read -r line; do
        if [[ "$line" == "---" ]]; then
            if $in_frontmatter; then
                break  # End of frontmatter
            else
                in_frontmatter=true
                continue
            fi
        fi

        if $in_frontmatter; then
            local key="${line%%:*}"
            local value="${line#*: }"
            key=$(echo "$key" | tr -d ' ')
            value=$(echo "$value" | sed 's/^[[:space:]]*//' | sed 's/[[:space:]]*$//')

            case "$key" in
                name) name="$value" ;;
                description) description="$value" ;;
                type) type="$value" ;;
                priority) priority="$value" ;;
                version) version=$(echo "$value" | tr -d '"'"'" ) ;;
                domains) domains_raw="$value" ;;
                agent_roles) agent_roles_raw="$value" ;;
                detect_files) detect_files_raw="$value" ;;
                detect_packages) detect_packages_raw="$value" ;;
            esac
        fi
    done < "$skill_file"

    # Parse bracket arrays: [item1, item2] -> JSON array
    _parse_bracket_array() {
        local raw="$1"
        raw=$(echo "$raw" | sed 's/^\[//' | sed 's/\]$//' | sed 's/^[[:space:]]*//' | sed 's/[[:space:]]*$//')
        if [[ -z "$raw" ]]; then
            echo "[]"
            return
        fi
        local arr="["
        local first=true
        IFS=',' read -ra items <<< "$raw"
        for item in "${items[@]}"; do
            item=$(echo "$item" | sed 's/^[[:space:]]*//' | sed 's/[[:space:]]*$//' | tr -d '"'"'")
            if $first; then
                arr+="\"$item\""
                first=false
            else
                arr+=",\"$item\""
            fi
        done
        arr+="]"
        echo "$arr"
    }

    local domains_json=$(_parse_bracket_array "$domains_raw")
    local roles_json=$(_parse_bracket_array "$agent_roles_raw")
    local detect_files_json=$(_parse_bracket_array "$detect_files_raw")
    local detect_packages_json=$(_parse_bracket_array "$detect_packages_raw")

    local result
    result=$(cat <<ENDJSON
{
    "name": "$name",
    "description": "$description",
    "type": "$type",
    "domains": $domains_json,
    "agent_roles": $roles_json,
    "detect_files": $detect_files_json,
    "detect_packages": $detect_packages_json,
    "priority": "$priority",
    "version": "$version",
    "file_path": "$skill_file"
}
ENDJSON
)
    json_ok "$result"
}

# Build index from all SKILL.md files in a skills directory
_skill_build_index() {
    local skills_dir="${1:-${AETHER_SKILLS_DIR:-$HOME/.aether/skills}}"
    local cache_file="$skills_dir/.index.json"
    local entries="[]"
    local count=0

    for skill_dir in "$skills_dir"/colony/*/SKILL.md "$skills_dir"/domain/*/SKILL.md; do
        [[ -f "$skill_dir" ]] || continue
        local parsed
        parsed=$(_skill_parse_frontmatter "$skill_dir" 2>/dev/null)
        local ok
        ok=$(echo "$parsed" | jq -r '.ok // false' 2>/dev/null)
        if [[ "$ok" == "true" ]]; then
            local entry
            entry=$(echo "$parsed" | jq -r '.result' 2>/dev/null)
            entries=$(echo "$entries" | jq --argjson e "$entry" '. + [$e]' 2>/dev/null)
            count=$((count + 1))
        fi
    done

    # Write cache
    local index_json
    index_json=$(cat <<ENDJSON
{
    "version": "1.0",
    "built_at": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
    "skill_count": $count,
    "skills": $entries
}
ENDJSON
)
    echo "$index_json" > "$cache_file" 2>/dev/null || true
    json_ok "$index_json"
}

# Read cached index, rebuild if stale
_skill_read_index() {
    local skills_dir="${1:-${AETHER_SKILLS_DIR:-$HOME/.aether/skills}}"
    local cache_file="$skills_dir/.index.json"

    # Check if cache exists and is fresh
    if [[ -f "$cache_file" ]]; then
        local cache_mtime
        cache_mtime=$(stat -f %m "$cache_file" 2>/dev/null || stat -c %Y "$cache_file" 2>/dev/null || echo 0)
        local needs_rebuild=false

        # Check if any SKILL.md is newer than cache
        for skill_file in "$skills_dir"/colony/*/SKILL.md "$skills_dir"/domain/*/SKILL.md; do
            [[ -f "$skill_file" ]] || continue
            local file_mtime
            file_mtime=$(stat -f %m "$skill_file" 2>/dev/null || stat -c %Y "$skill_file" 2>/dev/null || echo 0)
            if [[ "$file_mtime" -gt "$cache_mtime" ]]; then
                needs_rebuild=true
                break
            fi
        done

        if ! $needs_rebuild; then
            json_ok "$(cat "$cache_file")"
            return
        fi
    fi

    # Rebuild
    _skill_build_index "$skills_dir"
}

# Detect which domain skills match the current codebase
_skill_detect_codebase() {
    local repo_dir="${1:-.}"
    local skills_dir="${2:-${AETHER_SKILLS_DIR:-$HOME/.aether/skills}}"

    # Read index
    local index_result
    index_result=$(_skill_read_index "$skills_dir" 2>/dev/null)
    local index
    index=$(echo "$index_result" | jq -r '.result' 2>/dev/null)

    local detections="[]"

    # For each domain skill with detect patterns (process substitution to avoid subshell)
    while IFS= read -r skill; do
        [[ -z "$skill" ]] && continue
        local skill_name
        skill_name=$(echo "$skill" | jq -r '.name')
        local score=0

        # Check detect_files (use find for recursive matching)
        local detect_files
        detect_files=$(echo "$skill" | jq -r '.detect_files[]' 2>/dev/null)
        while IFS= read -r pattern; do
            [[ -z "$pattern" ]] && continue
            if find "$repo_dir" -name "$pattern" -print -quit 2>/dev/null | grep -q .; then
                score=$((score + 30))
            fi
        done <<< "$detect_files"

        # Check detect_packages (package manifests)
        local detect_packages
        detect_packages=$(echo "$skill" | jq -r '.detect_packages[]' 2>/dev/null)
        while IFS= read -r pkg; do
            [[ -z "$pkg" ]] && continue
            # Check package.json
            if [[ -f "$repo_dir/package.json" ]] && jq -e --arg p "$pkg" '.dependencies[$p] // .devDependencies[$p]' "$repo_dir/package.json" > /dev/null 2>&1; then
                score=$((score + 40))
            fi
            # Check requirements.txt
            if [[ -f "$repo_dir/requirements.txt" ]] && grep -qi "^$pkg" "$repo_dir/requirements.txt" 2>/dev/null; then
                score=$((score + 40))
            fi
            # Check go.mod
            if [[ -f "$repo_dir/go.mod" ]] && grep -qi "$pkg" "$repo_dir/go.mod" 2>/dev/null; then
                score=$((score + 40))
            fi
            # Check Gemfile
            if [[ -f "$repo_dir/Gemfile" ]] && grep -qi "'$pkg'" "$repo_dir/Gemfile" 2>/dev/null; then
                score=$((score + 40))
            fi
        done <<< "$detect_packages"

        if [[ $score -gt 0 ]]; then
            detections=$(echo "$detections" | jq --arg n "$skill_name" --argjson s "$score" '. + [{"name": $n, "score": $s}]' 2>/dev/null)
        fi
    done < <(echo "$index" | jq -c '.skills[] | select(.type == "domain") | select((.detect_files | length > 0) or (.detect_packages | length > 0))' 2>/dev/null)

    json_ok "{\"detections\": $detections}"
}

# Match skills to a specific worker
_skill_match() {
    local worker_role="$1"
    local task_description="${2:-}"
    local skills_dir="${3:-${AETHER_SKILLS_DIR:-$HOME/.aether/skills}}"
    local max_colony=3
    local max_domain=3

    # Read index
    local index_result
    index_result=$(_skill_read_index "$skills_dir" 2>/dev/null)
    local skills
    skills=$(echo "$index_result" | jq -r '.result.skills' 2>/dev/null)

    # Read active pheromones for domain boosting
    local pheromone_domains=""
    if [[ -f "$DATA_DIR/pheromones.json" ]]; then
        pheromone_domains=$(jq -r '[.signals[]? | select(.active == true) | .content] | join(" ")' "$DATA_DIR/pheromones.json" 2>/dev/null || echo "")
    fi

    # Match colony skills — filter by role, score by domain relevance, take top 3
    local colony_matches
    colony_matches=$(echo "$skills" | jq -c --arg role "$worker_role" \
        '[.[] | select(.type == "colony") | select(.agent_roles | index($role))]' 2>/dev/null)

    # Score colony skills by pheromone domain overlap (process substitution to avoid subshell)
    local scored_colony="[]"
    while IFS= read -r skill; do
        local score=50  # Base score for colony skills
        local skill_domains
        skill_domains=$(echo "$skill" | jq -r '.domains[]' 2>/dev/null)
        while IFS= read -r domain; do
            [[ -z "$domain" ]] && continue
            if echo "$pheromone_domains $task_description" | grep -qi "$domain" 2>/dev/null; then
                score=$((score + 20))
            fi
        done <<< "$skill_domains"

        # Priority boost
        local priority
        priority=$(echo "$skill" | jq -r '.priority' 2>/dev/null)
        case "$priority" in
            high) score=$((score + 30)) ;;
            low) score=$((score - 10)) ;;
        esac

        scored_colony=$(echo "$scored_colony" | jq --argjson s "$skill" --argjson sc "$score" '. + [$s + {"match_score": $sc}]' 2>/dev/null)
    done < <(echo "$colony_matches" | jq -c '.[]' 2>/dev/null)

    # Sort and take top 3 colony
    local top_colony
    top_colony=$(echo "$scored_colony" | jq "[sort_by(-.match_score) | limit($max_colony; .[])]" 2>/dev/null)

    # Match domain skills — filter by role, use detection scores + pheromone boost, take top 3
    local domain_matches
    domain_matches=$(echo "$skills" | jq -c --arg role "$worker_role" \
        '[.[] | select(.type == "domain") | select(.agent_roles | index($role))]' 2>/dev/null)

    local scored_domain="[]"
    while IFS= read -r skill; do
        [[ -z "$skill" ]] && continue
        local score=0
        local skill_domains
        skill_domains=$(echo "$skill" | jq -r '.domains[]' 2>/dev/null)
        while IFS= read -r domain; do
            [[ -z "$domain" ]] && continue
            if echo "$pheromone_domains $task_description" | grep -qi "$domain" 2>/dev/null; then
                score=$((score + 15))
            fi
        done <<< "$skill_domains"

        scored_domain=$(echo "$scored_domain" | jq --argjson s "$skill" --argjson sc "$score" '. + [$s + {"match_score": $sc}]' 2>/dev/null)
    done < <(echo "$domain_matches" | jq -c '.[]' 2>/dev/null)

    local top_domain
    top_domain=$(echo "$scored_domain" | jq "[sort_by(-.match_score) | limit($max_domain; .[])]" 2>/dev/null)

    json_ok "{\"colony_skills\": ${top_colony:-[]}, \"domain_skills\": ${top_domain:-[]}}"
}

# Load full SKILL.md content for matched skills, format for injection
_skill_inject() {
    local match_json="$1"
    local budget=12000
    local output=""
    local total_chars=0

    # Inject domain skills first (trimmed first if over budget)
    local domain_skills
    domain_skills=$(echo "$match_json" | jq -c '.domain_skills[]?' 2>/dev/null)
    local domain_section=""
    while IFS= read -r skill; do
        [[ -z "$skill" ]] && continue
        local file_path
        file_path=$(echo "$skill" | jq -r '.file_path')
        local skill_name
        skill_name=$(echo "$skill" | jq -r '.name')

        if [[ -f "$file_path" ]]; then
            # Extract body (everything after second ---)
            local body
            body=$(awk '/^---$/{c++;next}c>=2' "$file_path")
            local body_len=${#body}

            if [[ $((total_chars + body_len)) -le $budget ]]; then
                domain_section+="### Domain Skill: $skill_name"$'\n'"$body"$'\n\n'
                total_chars=$((total_chars + body_len + 30))
            fi
        fi
    done <<< "$domain_skills"

    # Inject colony skills (trimmed last)
    local colony_skills
    colony_skills=$(echo "$match_json" | jq -c '.colony_skills[]?' 2>/dev/null)
    local colony_section=""
    while IFS= read -r skill; do
        [[ -z "$skill" ]] && continue
        local file_path
        file_path=$(echo "$skill" | jq -r '.file_path')
        local skill_name
        skill_name=$(echo "$skill" | jq -r '.name')

        if [[ -f "$file_path" ]]; then
            local body
            body=$(awk '/^---$/{c++;next}c>=2' "$file_path")
            local body_len=${#body}

            if [[ $((total_chars + body_len)) -le $budget ]]; then
                colony_section+="### Colony Skill: $skill_name"$'\n'"$body"$'\n\n'
                total_chars=$((total_chars + body_len + 30))
            fi
        fi
    done <<< "$colony_skills"

    # Assemble skill_section
    local skill_section=""
    if [[ -n "$colony_section" || -n "$domain_section" ]]; then
        skill_section="## MATCHED SKILLS"$'\n\n'
        [[ -n "$colony_section" ]] && skill_section+="$colony_section"
        [[ -n "$domain_section" ]] && skill_section+="$domain_section"
    fi

    local colony_count domain_count
    colony_count=$(echo "$match_json" | jq '[.colony_skills[]?] | length' 2>/dev/null || echo 0)
    domain_count=$(echo "$match_json" | jq '[.domain_skills[]?] | length' 2>/dev/null || echo 0)

    # Escape for JSON
    local escaped_section
    escaped_section=$(echo "$skill_section" | jq -Rs '.' 2>/dev/null)

    json_ok "{\"skill_section\": $escaped_section, \"colony_count\": $colony_count, \"domain_count\": $domain_count, \"total_chars\": $total_chars}"
}

# List all installed skills
_skill_list() {
    local skills_dir="${1:-${AETHER_SKILLS_DIR:-$HOME/.aether/skills}}"
    local index_result
    index_result=$(_skill_read_index "$skills_dir" 2>/dev/null)
    local index
    index=$(echo "$index_result" | jq -r '.result' 2>/dev/null)

    json_ok "$index"
}

# Read/check manifest for update safety
_skill_manifest_read() {
    local manifest_file="$1"
    if [[ -f "$manifest_file" ]]; then
        json_ok "$(cat "$manifest_file")"
    else
        json_ok '{"managed_by": "aether", "version": "0.0.0", "skills": []}'
    fi
}

# Compare user skill with Aether-shipped version
_skill_diff() {
    local skill_name="$1"
    local skills_dir="${2:-${AETHER_SKILLS_DIR:-$HOME/.aether/skills}}"
    local system_dir="${AETHER_SYSTEM_DIR:-$HOME/.aether/system/skills}"

    # Find user skill
    local user_file=""
    for category in colony domain; do
        if [[ -f "$skills_dir/$category/$skill_name/SKILL.md" ]]; then
            user_file="$skills_dir/$category/$skill_name/SKILL.md"
            break
        fi
    done

    # Find system (shipped) skill
    local system_file=""
    for category in colony domain; do
        if [[ -f "$system_dir/$category/$skill_name/SKILL.md" ]]; then
            system_file="$system_dir/$category/$skill_name/SKILL.md"
            break
        fi
    done

    if [[ -z "$user_file" ]]; then
        json_err "NOT_FOUND" "No user skill named '$skill_name' found"
        return 1
    fi

    if [[ -z "$system_file" ]]; then
        json_ok "{\"status\": \"user_only\", \"message\": \"Skill '$skill_name' is user-created with no Aether equivalent\"}"
        return
    fi

    # Compare
    if diff -q "$user_file" "$system_file" > /dev/null 2>&1; then
        json_ok "{\"status\": \"identical\", \"message\": \"User and Aether versions are identical\"}"
    else
        local diff_output
        diff_output=$(diff -u "$system_file" "$user_file" 2>/dev/null | head -50)
        local escaped_diff
        escaped_diff=$(echo "$diff_output" | jq -Rs '.' 2>/dev/null)
        json_ok "{\"status\": \"different\", \"message\": \"User and Aether versions differ\", \"diff\": $escaped_diff}"
    fi
}

# Check if a skill is user-created (not in manifest)
_skill_is_user_created() {
    local skill_name="$1"
    local manifest_file="$2"
    if [[ ! -f "$manifest_file" ]]; then
        echo "true"
        return
    fi
    local managed
    managed=$(jq -r --arg n "$skill_name" '.skills | index($n)' "$manifest_file" 2>/dev/null)
    if [[ "$managed" == "null" || -z "$managed" ]]; then
        echo "true"
    else
        echo "false"
    fi
}
```

- [ ] **Step 4: Source skills.sh from aether-utils.sh**

In `.aether/aether-utils.sh`, add after the midden.sh source line (~line 33):

```bash
[[ -f "$SCRIPT_DIR/utils/skills.sh" ]] && source "$SCRIPT_DIR/utils/skills.sh"
```

- [ ] **Step 5: Add dispatch cases for skill subcommands**

In `.aether/aether-utils.sh` dispatch table (~line 980), add:

```bash
skill-parse-frontmatter)
    _skill_parse_frontmatter "$2"
    ;;
skill-index)
    _skill_build_index "${2:-}"
    ;;
skill-index-read)
    _skill_read_index "${2:-}"
    ;;
skill-detect)
    _skill_detect_codebase "${2:-.}" "${3:-}"
    ;;
skill-match)
    _skill_match "$2" "${3:-}" "${4:-}"
    ;;
skill-inject)
    _skill_inject "$2"
    ;;
skill-list)
    _skill_list "${2:-}"
    ;;
skill-manifest-read)
    _skill_manifest_read "$2"
    ;;
skill-cache-rebuild)
    local skills_dir="${2:-${AETHER_SKILLS_DIR:-$HOME/.aether/skills}}"
    rm -f "$skills_dir/.index.json"
    _skill_build_index "$skills_dir"
    ;;
skill-diff)
    _skill_diff "$2" "${3:-}"
    ;;
```

- [ ] **Step 6: Add skill commands to help output**

In the help case (~line 988), add "Skills" section to the sections object and add subcommand names to the commands array.

- [ ] **Step 7: Run tests to verify they pass**

Run: `bash tests/bash/test-skills.sh`
Expected: PASS — all 3 frontmatter tests pass

- [ ] **Step 8: Commit**

```bash
git add .aether/utils/skills.sh .aether/aether-utils.sh tests/bash/test-skills.sh
git commit -m "feat(skills): add skills utility with frontmatter parser and index cache"
```

---

## Task 2: Create index, detect, match, and inject tests

**Files:**
- Modify: `tests/bash/test-skills.sh`

Build out comprehensive tests for all skill engine functions.

- [ ] **Step 1: Add index build test**

```bash
test_skill_index_build() {
    test_start "skill-index builds index from SKILL.md files"
    setup_test_env

    local output
    output=$(AETHER_SKILLS_DIR="$SKILLS_DIR" bash "$AETHER_UTILS_SOURCE" skill-index "$SKILLS_DIR" 2>&1)
    assert_exit_code $? 0
    assert_json_valid "$output"

    local count
    count=$(echo "$output" | jq -r '.result.skill_count')
    [[ "$count" == "2" ]] && test_pass || test_fail "Expected 2 skills, got $count"

    cleanup_test_env
}
```

- [ ] **Step 2: Add index cache test**

```bash
test_skill_index_cache() {
    test_start "skill-index-read uses cache when fresh"
    setup_test_env

    # Build index
    AETHER_SKILLS_DIR="$SKILLS_DIR" bash "$AETHER_UTILS_SOURCE" skill-index "$SKILLS_DIR" > /dev/null 2>&1

    # Verify cache file exists
    [[ -f "$SKILLS_DIR/.index.json" ]] && test_pass || test_fail "Cache file not created"

    cleanup_test_env
}
```

- [ ] **Step 3: Add match test**

```bash
test_skill_match_by_role() {
    test_start "skill-match filters by agent role"
    setup_test_env

    # Build index first
    AETHER_SKILLS_DIR="$SKILLS_DIR" bash "$AETHER_UTILS_SOURCE" skill-index "$SKILLS_DIR" > /dev/null 2>&1

    local output
    output=$(AETHER_SKILLS_DIR="$SKILLS_DIR" bash "$AETHER_UTILS_SOURCE" skill-match "builder" "" "$SKILLS_DIR" 2>&1)
    assert_exit_code $? 0
    assert_json_valid "$output"

    # Builder should match both test-skill (colony) and test-domain (domain)
    local colony_count
    colony_count=$(echo "$output" | jq '[.result.colony_skills[]?] | length')
    [[ "$colony_count" -ge 1 ]] && test_pass || test_fail "Expected at least 1 colony match, got $colony_count"

    cleanup_test_env
}

test_skill_match_role_filter() {
    test_start "skill-match excludes skills for wrong role"
    setup_test_env

    AETHER_SKILLS_DIR="$SKILLS_DIR" bash "$AETHER_UTILS_SOURCE" skill-index "$SKILLS_DIR" > /dev/null 2>&1

    local output
    output=$(AETHER_SKILLS_DIR="$SKILLS_DIR" bash "$AETHER_UTILS_SOURCE" skill-match "chronicler" "" "$SKILLS_DIR" 2>&1)

    # Chronicler shouldn't match test-skill (only builder,watcher) or test-domain (only builder)
    local colony_count
    colony_count=$(echo "$output" | jq '[.result.colony_skills[]?] | length')
    [[ "$colony_count" == "0" ]] && test_pass || test_fail "Expected 0 colony matches for chronicler, got $colony_count"

    cleanup_test_env
}
```

- [ ] **Step 4: Add inject test**

```bash
test_skill_inject() {
    test_start "skill-inject loads full skill content within budget"
    setup_test_env

    AETHER_SKILLS_DIR="$SKILLS_DIR" bash "$AETHER_UTILS_SOURCE" skill-index "$SKILLS_DIR" > /dev/null 2>&1

    local match_json='{"colony_skills":[{"name":"test-skill","file_path":"'$SKILLS_DIR'/colony/test-skill/SKILL.md"}],"domain_skills":[]}'

    local output
    output=$(bash "$AETHER_UTILS_SOURCE" skill-inject "$match_json" 2>&1)
    assert_exit_code $? 0

    local section
    section=$(echo "$output" | jq -r '.result.skill_section')
    [[ "$section" == *"Test skill content"* ]] && test_pass || test_fail "Skill content not injected"

    cleanup_test_env
}
```

- [ ] **Step 5: Add list test**

```bash
test_skill_list() {
    test_start "skill-list returns all installed skills"
    setup_test_env

    AETHER_SKILLS_DIR="$SKILLS_DIR" bash "$AETHER_UTILS_SOURCE" skill-index "$SKILLS_DIR" > /dev/null 2>&1

    local output
    output=$(AETHER_SKILLS_DIR="$SKILLS_DIR" bash "$AETHER_UTILS_SOURCE" skill-list "$SKILLS_DIR" 2>&1)
    assert_json_valid "$output"

    local count
    count=$(echo "$output" | jq '.result.skill_count')
    [[ "$count" == "2" ]] && test_pass || test_fail "Expected 2 skills, got $count"

    cleanup_test_env
}
```

- [ ] **Step 6: Add all new tests to the run section and run**

Run: `bash tests/bash/test-skills.sh`
Expected: All tests PASS

- [ ] **Step 7: Commit**

```bash
git add tests/bash/test-skills.sh
git commit -m "test(skills): add comprehensive tests for skill engine"
```

---

## Task 3: Create the 10 colony skill SKILL.md files

**Files:**
- Create: `.aether/skills/colony/{name}/SKILL.md` (10 files)
- Create: `.aether/skills/colony/.manifest.json`

Create all colony skills from the spec. Each SKILL.md has frontmatter + full process guide content.

- [ ] **Step 1: Create colony-interaction skill**
- [ ] **Step 2: Create colony-visuals skill**
- [ ] **Step 3: Create pheromone-visibility skill**
- [ ] **Step 4: Create build-discipline skill**
- [ ] **Step 5: Create colony-lifecycle skill**
- [ ] **Step 6: Create context-management skill**
- [ ] **Step 7: Create state-safety skill**
- [ ] **Step 8: Create error-presentation skill**
- [ ] **Step 9: Create pheromone-protocol skill**
- [ ] **Step 10: Create worker-priming skill**
- [ ] **Step 11: Create colony manifest**

Create `.aether/skills/colony/.manifest.json`:
```json
{
    "managed_by": "aether",
    "version": "2.1.0",
    "skills": [
        "colony-interaction",
        "colony-visuals",
        "pheromone-visibility",
        "build-discipline",
        "colony-lifecycle",
        "context-management",
        "state-safety",
        "error-presentation",
        "pheromone-protocol",
        "worker-priming"
    ]
}
```

- [ ] **Step 12: Verify all skills parse correctly**

Run: `for d in .aether/skills/colony/*/SKILL.md; do echo "--- $d ---"; bash .aether/aether-utils.sh skill-parse-frontmatter "$d" | jq '.ok'; done`
Expected: All return `true`

- [ ] **Step 13: Commit**

```bash
git add .aether/skills/colony/
git commit -m "feat(skills): add 10 colony skill definitions"
```

---

## Task 4: Create the 18 starter domain skill SKILL.md files

**Files:**
- Create: `.aether/skills/domain/{name}/SKILL.md` (18 files)
- Create: `.aether/skills/domain/.manifest.json`
- Create: `.aether/skills/domain/README.md`

- [ ] **Step 1: Create frontend skills (react, nextjs, vue, tailwind, html-css, svelte)**
- [ ] **Step 2: Create backend skills (nodejs, python, django, rails, golang)**
- [ ] **Step 3: Create data/API skills (postgresql, rest-api, graphql, prisma)**
- [ ] **Step 4: Create infrastructure skills (docker, typescript, testing)**
- [ ] **Step 5: Create domain manifest**

Create `.aether/skills/domain/.manifest.json` with all 18 skill names.

- [ ] **Step 6: Create README for custom skills**

Create `.aether/skills/domain/README.md`:
```markdown
# Domain Skills

Drop a folder here with a SKILL.md to add custom domain skills.
Skills created with /ant:skill-create are placed here automatically.

## Format

See any existing skill for the SKILL.md format.
Your custom skills are never overwritten by aether update.
```

- [ ] **Step 7: Verify all domain skills parse**

Run: `bash .aether/aether-utils.sh skill-index .aether/skills | jq '.result.skill_count'`
Expected: `28`

- [ ] **Step 8: Commit**

```bash
git add .aether/skills/domain/
git commit -m "feat(skills): add 18 starter domain skill definitions"
```

---

## Task 5: Integrate skills into the build pipeline

**Files:**
- Modify: `.aether/docs/command-playbooks/build-context.md`
- Modify: `.aether/docs/command-playbooks/build-wave.md`

This is where skills actually start getting used — injected into worker prompts during builds.

- [ ] **Step 1: Add skill indexing to build-context.md**

After the colony-prime step (Step 4), add a new step:

```markdown
### Step 4.5: Skill Detection

Run skill detection to identify which domain skills match the codebase:

bash .aether/aether-utils.sh skill-index
bash .aether/aether-utils.sh skill-detect "$(pwd)"

Store the detection results as cross-stage state variable `skill_detections`.
Store the skills directory path as `skills_dir`.

Display to user:
"Skills: {count} indexed, {detection_count} matched to codebase"
```

- [ ] **Step 2: Add per-worker skill injection to build-wave.md**

In the worker spawn section of build-wave.md, AFTER the `{ prompt_section }` injection, add:

```markdown
### Skill Injection (per worker)

For each worker being spawned, run skill matching:

skill_match=$(bash .aether/aether-utils.sh skill-match "{worker_role}" "{task_description}")
skill_inject=$(bash .aether/aether-utils.sh skill-inject "$skill_match")
skill_section=$(echo "$skill_inject" | jq -r '.result.skill_section')

Append `skill_section` to the worker prompt after `prompt_section`.

Display to user:
"  Skills: {colony_count} colony + {domain_count} domain loaded for {worker_role}"
```

- [ ] **Step 3: Verify build-context and build-wave are syntactically correct**

Review both files to ensure markdown formatting and instruction flow are consistent with existing steps.

- [ ] **Step 4: Commit**

```bash
git add .aether/docs/command-playbooks/build-context.md .aether/docs/command-playbooks/build-wave.md
git commit -m "feat(skills): integrate skill detection and injection into build pipeline"
```

---

## Task 6: Add skills to hub distribution (setupHub)

**Files:**
- Modify: `bin/cli.js` (setupHub function)

- [ ] **Step 1: Add skills sync to setupHub**

In `bin/cli.js` setupHub function, after existing sync operations, add skills directory sync:

```javascript
// Sync skills to hub
const skillsSrc = path.join(aetherSrc, 'skills');
const HUB_SKILLS_DIR = path.join(HUB_DIR, 'skills');

if (fs.existsSync(skillsSrc)) {
    // Read manifests to know which skills we own
    for (const category of ['colony', 'domain']) {
        const srcCat = path.join(skillsSrc, category);
        const hubCat = path.join(HUB_SKILLS_DIR, category);

        if (!fs.existsSync(srcCat)) continue;
        fs.mkdirSync(hubCat, { recursive: true });

        // Copy manifest
        const manifestSrc = path.join(srcCat, '.manifest.json');
        if (fs.existsSync(manifestSrc)) {
            fs.copyFileSync(manifestSrc, path.join(hubCat, '.manifest.json'));
        }

        // Read manifest for owned skills
        let manifest = { skills: [] };
        try {
            manifest = JSON.parse(fs.readFileSync(manifestSrc, 'utf8'));
        } catch (e) { /* no manifest = no managed skills */ }

        // Sync only managed skills (skip user-created)
        const srcDirs = fs.readdirSync(srcCat, { withFileTypes: true })
            .filter(d => d.isDirectory());

        for (const dir of srcDirs) {
            if (manifest.skills.includes(dir.name)) {
                // Managed skill — overwrite
                const srcSkill = path.join(srcCat, dir.name);
                const hubSkill = path.join(hubCat, dir.name);
                syncDirWithCleanup(srcSkill, hubSkill);
            } else if (fs.existsSync(path.join(hubCat, dir.name))) {
                // User-created skill exists — skip and log
                console.log(`  Skipped skill '${dir.name}' — user version exists. Run 'aether skill-diff ${dir.name}' to compare.`);
            }
        }

        // Copy README if present
        const readmeSrc = path.join(srcCat, 'README.md');
        if (fs.existsSync(readmeSrc)) {
            fs.copyFileSync(readmeSrc, path.join(hubCat, 'README.md'));
        }
    }
}
```

- [ ] **Step 2: Test hub sync manually**

Run: `npm install -g .`
Verify: `ls ~/.aether/skills/colony/` shows 10 colony skills
Verify: `ls ~/.aether/skills/domain/` shows 18 domain skills

- [ ] **Step 3: Commit**

```bash
git add bin/cli.js
git commit -m "feat(skills): add skill distribution to setupHub"
```

---

## Task 7: Create /ant:skill-create command

**Files:**
- Create: `.claude/commands/ant/skill-create.md`

- [ ] **Step 1: Write the skill creation wizard command**

Create `.claude/commands/ant/skill-create.md` with:
- Argument parsing for the skill topic
- Oracle mini-research (5-10 iterations) on the topic
- AskUserQuestion wizard: focus area, experience level, custom rules
- Generate SKILL.md from research + wizard answers
- Write to `~/.aether/skills/domain/{name}/SKILL.md`
- Show result to user and ask for approval
- Rebuild skill index cache after creation

- [ ] **Step 2: Commit**

```bash
git add .claude/commands/ant/skill-create.md
git commit -m "feat(skills): add /ant:skill-create wizard command"
```

---

## Task 8: Add AVA unit tests

**Files:**
- Create: `tests/unit/skills.test.js`

- [ ] **Step 1: Write AVA tests for skill subcommands**

```javascript
import test from 'ava';
import { execSync } from 'child_process';
import { mkdtempSync, mkdirSync, writeFileSync } from 'fs';
import { join } from 'path';
import { tmpdir } from 'os';

const UTILS = join(import.meta.dirname, '../../.aether/aether-utils.sh');

function runSkillCmd(cmd) {
    try {
        const out = execSync(`bash ${UTILS} ${cmd}`, {
            encoding: 'utf8',
            timeout: 10000,
            env: { ...process.env, PATH: process.env.PATH }
        });
        return JSON.parse(out.trim());
    } catch (e) {
        return { ok: false, error: e.message };
    }
}

function createTestSkillsDir() {
    const dir = mkdtempSync(join(tmpdir(), 'aether-skills-'));
    mkdirSync(join(dir, 'colony', 'test-skill'), { recursive: true });
    writeFileSync(join(dir, 'colony', 'test-skill', 'SKILL.md'),
        '---\nname: test-skill\ndescription: Test skill\ntype: colony\ndomains: [testing]\nagent_roles: [builder]\npriority: normal\nversion: "1.0"\n---\n\nTest content.\n');
    return dir;
}

test('skill-list returns valid JSON with skill_count', t => {
    const dir = createTestSkillsDir();
    runSkillCmd(`skill-index ${dir}`);
    const result = runSkillCmd(`skill-list ${dir}`);
    t.true(result.ok);
    t.is(typeof result.result.skill_count, 'number');
});

test('skill-parse-frontmatter extracts name field', t => {
    const dir = createTestSkillsDir();
    const skillFile = join(dir, 'colony', 'test-skill', 'SKILL.md');
    const result = runSkillCmd(`skill-parse-frontmatter ${skillFile}`);
    t.true(result.ok);
    t.is(result.result.name, 'test-skill');
});

test('skill-match returns colony and domain arrays', t => {
    const dir = createTestSkillsDir();
    runSkillCmd(`skill-index ${dir}`);
    const result = runSkillCmd(`skill-match builder "" ${dir}`);
    t.true(result.ok);
    t.true(Array.isArray(result.result.colony_skills));
    t.true(Array.isArray(result.result.domain_skills));
});
```

- [ ] **Step 2: Run tests**

Run: `npm run test:unit`
Expected: All pass

- [ ] **Step 3: Commit**

```bash
git add tests/unit/skills.test.js
git commit -m "test(skills): add AVA unit tests for skill engine"
```

---

## Task 9: Update documentation

**Files:**
- Modify: `CLAUDE.md`
- Modify: `.claude/rules/aether-colony.md`

- [ ] **Step 1: Add Skills section to CLAUDE.md**

Add after the Pheromone System section:
- Skills architecture overview
- Two categories (colony + domain)
- How matching works
- How to create custom skills
- Skill subcommand reference table

- [ ] **Step 2: Add skills info to aether-colony.md**

Add `/ant:skill-create` to the Available Commands table.
Add skill info to the Typical Workflow section.

- [ ] **Step 3: Update Quick Reference table in CLAUDE.md**

Update the count table to include skills.

- [ ] **Step 4: Commit**

```bash
git add CLAUDE.md .claude/rules/aether-colony.md
git commit -m "docs: add skills system documentation to CLAUDE.md"
```

---

## Task 10: Final integration test and validation

**Files:**
- No new files — validation only

- [ ] **Step 1: Run full test suite**

Run: `npm test`
Expected: All 542+ tests pass (plus new skills tests)

- [ ] **Step 2: Run linters**

Run: `npm run lint`
Expected: No new lint errors

- [ ] **Step 3: Validate package**

Run: `bash bin/validate-package.sh`
Expected: Package validates with skills included

- [ ] **Step 4: Test end-to-end skill flow**

```bash
# Index skills
bash .aether/aether-utils.sh skill-index ~/.aether/skills

# List skills
bash .aether/aether-utils.sh skill-list ~/.aether/skills | jq '.result.skill_count'

# Match skills for a builder
bash .aether/aether-utils.sh skill-match builder "implement login page" ~/.aether/skills

# Inject matched skills
bash .aether/aether-utils.sh skill-inject '{"colony_skills":[],"domain_skills":[]}'
```

- [ ] **Step 5: Commit any fixes**

```bash
git add -A
git commit -m "fix(skills): integration test fixes"
```

---

## Execution Order

Tasks must be executed in order — each builds on the previous:

```
Task 1 (skills.sh core)
    ↓
Task 2 (comprehensive tests)
    ↓
Task 3 (colony skills) ←→ Task 4 (domain skills)  [can run in parallel]
    ↓
Task 5 (build pipeline integration)
    ↓
Task 6 (hub distribution)
    ↓
Task 7 (skill-create command)
    ↓
Task 8 (AVA tests)
    ↓
Task 9 (documentation)
    ↓
Task 10 (validation)
```

Tasks 3 and 4 are independent and can run in parallel.
