# =============================================================================
# DEPRECATED — This script has been superseded by the Go binary (aether CLI).
# All functionality is now available via: aether <subcommand>
# Do NOT modify this file — it is retained for reference only.
# See: cmd/ (Go source) | Run: aether --help
# =============================================================================
#
#!/usr/bin/env bash
# Skills utility — frontmatter parsing, indexing, detection, matching, injection
# Provides: _skill_parse_frontmatter, _skill_build_index, _skill_read_index,
#           _skill_detect_codebase, _skill_match, _skill_inject, _skill_list,
#           _skill_manifest_read, _skill_diff, _skill_is_user_created
#
# These functions are sourced by aether-utils.sh at startup.
# All shared infrastructure (json_ok, json_err, DATA_DIR, SCRIPT_DIR) is available.
#
# Default skills directory: AETHER_SKILLS_DIR env var, falling back to $HOME/.aether/skills

# Parse YAML-like frontmatter from a SKILL.md file
# Usage: _skill_parse_frontmatter <path-to-SKILL.md>
# Returns JSON with all frontmatter fields via json_ok
_skill_parse_frontmatter() {
    local skill_file="$1"

    if [[ ! -f "$skill_file" ]]; then
        json_err "${E_FILE_NOT_FOUND:-SKILL_NOT_FOUND}" "Skill file not found: $skill_file"
        return 1
    fi

    local in_frontmatter=false
    local sp_name="" sp_description="" sp_type="" sp_priority="normal" sp_version="1.0"
    local sp_domains_raw="" sp_agent_roles_raw="" sp_detect_files_raw="" sp_detect_packages_raw=""

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
            # Trim whitespace from key
            key=$(echo "$key" | tr -d ' ')
            # Trim whitespace from value
            value=$(echo "$value" | sed 's/^[[:space:]]*//' | sed 's/[[:space:]]*$//')

            case "$key" in
                name)            sp_name="$value" ;;
                description)     sp_description="$value" ;;
                type)            sp_type="$value" ;;
                priority)        sp_priority="$value" ;;
                version)         sp_version=$(echo "$value" | tr -d '"'"'") ;;
                domains)         sp_domains_raw="$value" ;;
                agent_roles)     sp_agent_roles_raw="$value" ;;
                detect_files)    sp_detect_files_raw="$value" ;;
                detect_packages) sp_detect_packages_raw="$value" ;;
            esac
        fi
    done < "$skill_file"

    # Parse bracket arrays: [item1, item2] -> JSON array
    _sp_parse_bracket_array() {
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

    local sp_domains_json=$(_sp_parse_bracket_array "$sp_domains_raw")
    local sp_roles_json=$(_sp_parse_bracket_array "$sp_agent_roles_raw")
    local sp_detect_files_json=$(_sp_parse_bracket_array "$sp_detect_files_raw")
    local sp_detect_packages_json=$(_sp_parse_bracket_array "$sp_detect_packages_raw")

    # Use jq for safe JSON construction (handles special chars in description)
    local sp_result
    sp_result=$(jq -n \
        --arg name "$sp_name" \
        --arg description "$sp_description" \
        --arg type "$sp_type" \
        --argjson domains "$sp_domains_json" \
        --argjson agent_roles "$sp_roles_json" \
        --argjson detect_files "$sp_detect_files_json" \
        --argjson detect_packages "$sp_detect_packages_json" \
        --arg priority "$sp_priority" \
        --arg version "$sp_version" \
        --arg file_path "$skill_file" \
        '{
            name: $name,
            description: $description,
            type: $type,
            domains: $domains,
            agent_roles: $agent_roles,
            detect_files: $detect_files,
            detect_packages: $detect_packages,
            priority: $priority,
            version: $version,
            file_path: $file_path
        }')

    json_ok "$sp_result"
}

# Build index from all SKILL.md files in a skills directory
# Usage: _skill_build_index [skills_dir]
# Scans colony/ and domain/ subdirectories for SKILL.md files
# Writes cache to .index.json and returns the index via json_ok
_skill_build_index() {
    local skills_dir="${1:-${AETHER_SKILLS_DIR:-$HOME/.aether/skills}}"
    local cache_file="$skills_dir/.index.json"
    local bi_entries="[]"
    local bi_count=0

    for bi_skill_file in "$skills_dir"/colony/*/SKILL.md "$skills_dir"/domain/*/SKILL.md; do
        [[ -f "$bi_skill_file" ]] || continue
        local bi_parsed
        bi_parsed=$(_skill_parse_frontmatter "$bi_skill_file" 2>/dev/null)
        local bi_ok
        bi_ok=$(echo "$bi_parsed" | jq -r '.ok // false' 2>/dev/null)
        if [[ "$bi_ok" == "true" ]]; then
            local bi_entry
            bi_entry=$(echo "$bi_parsed" | jq -r '.result' 2>/dev/null)
            bi_entries=$(echo "$bi_entries" | jq --argjson e "$bi_entry" '. + [$e]' 2>/dev/null)
            bi_count=$((bi_count + 1))
        fi
    done

    # Build index JSON
    local bi_index_json
    bi_index_json=$(jq -n \
        --arg version "1.0" \
        --arg built_at "$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
        --argjson skill_count "$bi_count" \
        --argjson skills "$bi_entries" \
        '{
            version: $version,
            built_at: $built_at,
            skill_count: $skill_count,
            skills: $skills
        }')

    # Write cache file
    echo "$bi_index_json" > "$cache_file" 2>/dev/null || true

    json_ok "$bi_index_json"
}

# Read cached index, rebuild if stale (mtime check)
# Usage: _skill_read_index [skills_dir]
# Returns cached index if fresh, rebuilds if any SKILL.md is newer than cache
_skill_read_index() {
    local skills_dir="${1:-${AETHER_SKILLS_DIR:-$HOME/.aether/skills}}"
    local cache_file="$skills_dir/.index.json"

    # Check if cache exists and is fresh
    if [[ -f "$cache_file" ]]; then
        local ri_cache_mtime
        ri_cache_mtime=$(stat -f %m "$cache_file" 2>/dev/null || stat -c %Y "$cache_file" 2>/dev/null || echo 0)
        local ri_needs_rebuild=false

        # Check if any SKILL.md is newer than cache
        for ri_skill_file in "$skills_dir"/colony/*/SKILL.md "$skills_dir"/domain/*/SKILL.md; do
            [[ -f "$ri_skill_file" ]] || continue
            local ri_file_mtime
            ri_file_mtime=$(stat -f %m "$ri_skill_file" 2>/dev/null || stat -c %Y "$ri_skill_file" 2>/dev/null || echo 0)
            if [[ "$ri_file_mtime" -gt "$ri_cache_mtime" ]]; then
                ri_needs_rebuild=true
                break
            fi
        done

        if ! $ri_needs_rebuild; then
            json_ok "$(cat "$cache_file")"
            return
        fi
    fi

    # Cache missing or stale -- rebuild
    _skill_build_index "$skills_dir"
}

# Detect which domain skills match the current codebase
# Usage: _skill_detect_codebase [repo_dir] [skills_dir]
# Checks detect_files (glob patterns via find) and detect_packages (package manifests)
# Returns JSON with detection scores via json_ok
_skill_detect_codebase() {
    local repo_dir="${1:-.}"
    local skills_dir="${2:-${AETHER_SKILLS_DIR:-$HOME/.aether/skills}}"

    # Read index
    local dc_index_result
    dc_index_result=$(_skill_read_index "$skills_dir" 2>/dev/null)
    local dc_index
    dc_index=$(echo "$dc_index_result" | jq -r '.result' 2>/dev/null)

    local dc_detections="[]"

    # For each domain skill with detect patterns (process substitution to avoid subshell)
    while IFS= read -r dc_skill; do
        [[ -z "$dc_skill" ]] && continue
        local dc_skill_name
        dc_skill_name=$(echo "$dc_skill" | jq -r '.name')
        local dc_score=0

        # Check detect_files (use find for recursive matching)
        while IFS= read -r dc_pattern; do
            [[ -z "$dc_pattern" ]] && continue
            if find "$repo_dir" -name "$dc_pattern" -print -quit 2>/dev/null | grep -q .; then
                dc_score=$((dc_score + 30))
            fi
        done < <(echo "$dc_skill" | jq -r '.detect_files[]' 2>/dev/null)

        # Check detect_packages (package manifests)
        while IFS= read -r dc_pkg; do
            [[ -z "$dc_pkg" ]] && continue
            # Check package.json
            if [[ -f "$repo_dir/package.json" ]] && jq -e --arg p "$dc_pkg" '.dependencies[$p] // .devDependencies[$p]' "$repo_dir/package.json" > /dev/null 2>&1; then
                dc_score=$((dc_score + 40))
            fi
            # Check requirements.txt
            if [[ -f "$repo_dir/requirements.txt" ]] && grep -qi "^$dc_pkg" "$repo_dir/requirements.txt" 2>/dev/null; then
                dc_score=$((dc_score + 40))
            fi
            # Check go.mod
            if [[ -f "$repo_dir/go.mod" ]] && grep -qi "$dc_pkg" "$repo_dir/go.mod" 2>/dev/null; then
                dc_score=$((dc_score + 40))
            fi
            # Check Gemfile
            if [[ -f "$repo_dir/Gemfile" ]] && grep -qi "'$dc_pkg'" "$repo_dir/Gemfile" 2>/dev/null; then
                dc_score=$((dc_score + 40))
            fi
        done < <(echo "$dc_skill" | jq -r '.detect_packages[]' 2>/dev/null)

        if [[ $dc_score -gt 0 ]]; then
            dc_detections=$(echo "$dc_detections" | jq --arg n "$dc_skill_name" --argjson s "$dc_score" '. + [{"name": $n, "score": $s}]' 2>/dev/null)
        fi
    done < <(echo "$dc_index" | jq -c '.skills[] | select(.type == "domain") | select((.detect_files | length > 0) or (.detect_packages | length > 0))' 2>/dev/null)

    json_ok "$(jq -n --argjson detections "$dc_detections" '{detections: $detections}')"
}

# Smart-match skills to a worker by role, pheromones, and task description
# Usage: _skill_match <worker_role> [task_description] [skills_dir]
# Returns top 2 colony + top 2 domain skills (above minimum score) via json_ok
_skill_match() {
    local worker_role="$1"
    local task_description="${2:-}"
    local skills_dir="${3:-${AETHER_SKILLS_DIR:-$HOME/.aether/skills}}"
    local sm_max_colony=2
    local sm_max_domain=2
    local sm_min_score=20

    # Read index
    local sm_index_result
    sm_index_result=$(_skill_read_index "$skills_dir" 2>/dev/null)
    local sm_skills
    sm_skills=$(echo "$sm_index_result" | jq -r '.result.skills' 2>/dev/null)

    # Read active pheromones for domain boosting
    local sm_pheromone_domains=""
    if [[ -f "${DATA_DIR:-/dev/null}/pheromones.json" ]]; then
        sm_pheromone_domains=$(jq -r '[.signals[]? | select(.active == true) | .content] | join(" ")' "${DATA_DIR}/pheromones.json" 2>/dev/null || echo "")
    fi

    # Match colony skills -- filter by role
    local sm_colony_matches
    sm_colony_matches=$(echo "$sm_skills" | jq -c --arg role "$worker_role" \
        '[.[] | select(.type == "colony") | select(.agent_roles | index($role))]' 2>/dev/null)

    # Score colony skills by pheromone domain overlap (process substitution to avoid subshell)
    local sm_scored_colony="[]"
    while IFS= read -r sm_skill; do
        [[ -z "$sm_skill" ]] && continue
        local sm_score=50  # Base score for colony skills
        while IFS= read -r sm_domain; do
            [[ -z "$sm_domain" ]] && continue
            if echo "$sm_pheromone_domains $task_description" | grep -qi "$sm_domain" 2>/dev/null; then
                sm_score=$((sm_score + 20))
            fi
        done < <(echo "$sm_skill" | jq -r '.domains[]' 2>/dev/null)

        # Priority boost
        local sm_priority
        sm_priority=$(echo "$sm_skill" | jq -r '.priority' 2>/dev/null)
        case "$sm_priority" in
            high) sm_score=$((sm_score + 30)) ;;
            low)  sm_score=$((sm_score - 10)) ;;
        esac

        sm_scored_colony=$(echo "$sm_scored_colony" | jq --argjson s "$sm_skill" --argjson sc "$sm_score" '. + [$s + {"match_score": $sc}]' 2>/dev/null)
    done < <(echo "$sm_colony_matches" | jq -c '.[]' 2>/dev/null)

    # Filter by minimum score, sort, and take top N colony
    local sm_top_colony
    sm_top_colony=$(echo "$sm_scored_colony" | jq "[.[] | select(.match_score >= $sm_min_score)] | [sort_by(-.match_score) | limit($sm_max_colony; .[])]" 2>/dev/null)

    # Match domain skills -- filter by role
    local sm_domain_matches
    sm_domain_matches=$(echo "$sm_skills" | jq -c --arg role "$worker_role" \
        '[.[] | select(.type == "domain") | select(.agent_roles | index($role))]' 2>/dev/null)

    # Score domain skills (process substitution to avoid subshell)
    local sm_scored_domain="[]"
    while IFS= read -r sm_skill; do
        [[ -z "$sm_skill" ]] && continue
        local sm_score=0
        while IFS= read -r sm_domain; do
            [[ -z "$sm_domain" ]] && continue
            if echo "$sm_pheromone_domains $task_description" | grep -qi "$sm_domain" 2>/dev/null; then
                sm_score=$((sm_score + 15))
            fi
        done < <(echo "$sm_skill" | jq -r '.domains[]' 2>/dev/null)

        sm_scored_domain=$(echo "$sm_scored_domain" | jq --argjson s "$sm_skill" --argjson sc "$sm_score" '. + [$s + {"match_score": $sc}]' 2>/dev/null)
    done < <(echo "$sm_domain_matches" | jq -c '.[]' 2>/dev/null)

    # Filter by minimum score, sort, and take top N domain
    local sm_top_domain
    sm_top_domain=$(echo "$sm_scored_domain" | jq "[.[] | select(.match_score >= $sm_min_score)] | [sort_by(-.match_score) | limit($sm_max_domain; .[])]" 2>/dev/null)

    json_ok "$(jq -n --argjson colony "${sm_top_colony:-[]}" --argjson domain "${sm_top_domain:-[]}" \
      '{colony_skills: $colony, domain_skills: $domain}')"
}

# Load full SKILL.md content for matched skills, assemble within 8K char budget
# Usage: _skill_inject <match_json>
# match_json is the JSON result from skill-match (has colony_skills and domain_skills arrays)
# Injects domain skills first (trimmed first if over budget), then colony skills
# Logs trimmed skills to stderr
_skill_inject() {
    local match_json="$1"
    local si_budget=8000
    local si_total_chars=0

    # Inject domain skills first (these get trimmed first if over budget)
    local si_domain_section=""
    while IFS= read -r si_skill; do
        [[ -z "$si_skill" ]] && continue
        local si_file_path
        si_file_path=$(echo "$si_skill" | jq -r '.file_path')
        local si_skill_name
        si_skill_name=$(echo "$si_skill" | jq -r '.name')

        if [[ -f "$si_file_path" ]]; then
            # Extract body (everything after second ---)
            local si_body
            si_body=$(awk '/^---$/{c++;next}c>=2' "$si_file_path")
            local si_body_len=${#si_body}

            if [[ $((si_total_chars + si_body_len)) -le $si_budget ]]; then
                si_domain_section+="### Domain Skill: $si_skill_name"$'\n'"$si_body"$'\n\n'
                si_total_chars=$((si_total_chars + si_body_len + 30))
            else
                echo "[skills] trimmed: $si_skill_name (${si_body_len} chars)" >&2
            fi
        fi
    done < <(echo "$match_json" | jq -c '.domain_skills[]?' 2>/dev/null)

    # Inject colony skills (trimmed last -- higher priority)
    local si_colony_section=""
    while IFS= read -r si_skill; do
        [[ -z "$si_skill" ]] && continue
        local si_file_path
        si_file_path=$(echo "$si_skill" | jq -r '.file_path')
        local si_skill_name
        si_skill_name=$(echo "$si_skill" | jq -r '.name')

        if [[ -f "$si_file_path" ]]; then
            local si_body
            si_body=$(awk '/^---$/{c++;next}c>=2' "$si_file_path")
            local si_body_len=${#si_body}

            if [[ $((si_total_chars + si_body_len)) -le $si_budget ]]; then
                si_colony_section+="### Colony Skill: $si_skill_name"$'\n'"$si_body"$'\n\n'
                si_total_chars=$((si_total_chars + si_body_len + 30))
            else
                echo "[skills] trimmed: $si_skill_name (${si_body_len} chars)" >&2
            fi
        fi
    done < <(echo "$match_json" | jq -c '.colony_skills[]?' 2>/dev/null)

    # Assemble skill_section
    local si_skill_section=""
    if [[ -n "$si_colony_section" || -n "$si_domain_section" ]]; then
        si_skill_section="## MATCHED SKILLS"$'\n\n'
        [[ -n "$si_colony_section" ]] && si_skill_section+="$si_colony_section"
        [[ -n "$si_domain_section" ]] && si_skill_section+="$si_domain_section"
    fi

    local si_colony_count si_domain_count
    si_colony_count=$(echo "$match_json" | jq '[.colony_skills[]?] | length' 2>/dev/null || echo 0)
    si_domain_count=$(echo "$match_json" | jq '[.domain_skills[]?] | length' 2>/dev/null || echo 0)

    # Escape for JSON embedding
    local si_escaped_section
    si_escaped_section=$(printf '%s' "$si_skill_section" | jq -Rs '.' 2>/dev/null)

    json_ok "$(jq -n --argjson skill_section "$si_escaped_section" \
      --argjson colony_count "$si_colony_count" --argjson domain_count "$si_domain_count" \
      --argjson total_chars "$si_total_chars" \
      '{skill_section: $skill_section, colony_count: $colony_count, domain_count: $domain_count, total_chars: $total_chars}')"
}

# List all installed skills
# Usage: _skill_list [skills_dir]
# Returns the full index (from cache or rebuilt) via json_ok
_skill_list() {
    local skills_dir="${1:-${AETHER_SKILLS_DIR:-$HOME/.aether/skills}}"
    local sl_index_result
    sl_index_result=$(_skill_read_index "$skills_dir" 2>/dev/null)
    local sl_index
    sl_index=$(echo "$sl_index_result" | jq -r '.result' 2>/dev/null)

    json_ok "$sl_index"
}

# Read .manifest.json for update safety
# Usage: _skill_manifest_read <manifest_file>
# Returns manifest contents or default empty manifest
_skill_manifest_read() {
    local manifest_file="$1"
    if [[ -f "$manifest_file" ]]; then
        json_ok "$(cat "$manifest_file")"
    else
        json_ok '{"managed_by": "aether", "version": "0.0.0", "skills": []}'
    fi
}

# Compare user skill with shipped version
# Usage: _skill_diff <skill_name> [skills_dir]
# Looks for user version in skills_dir and system version in AETHER_SYSTEM_DIR
_skill_diff() {
    local skill_name="$1"
    local skills_dir="${2:-${AETHER_SKILLS_DIR:-$HOME/.aether/skills}}"
    local system_dir="${AETHER_SYSTEM_DIR:-$HOME/.aether/system/skills}"

    # Find user skill
    local sd_user_file=""
    for sd_category in colony domain; do
        if [[ -f "$skills_dir/$sd_category/$skill_name/SKILL.md" ]]; then
            sd_user_file="$skills_dir/$sd_category/$skill_name/SKILL.md"
            break
        fi
    done

    # Find system (shipped) skill
    local sd_system_file=""
    for sd_category in colony domain; do
        if [[ -f "$system_dir/$sd_category/$skill_name/SKILL.md" ]]; then
            sd_system_file="$system_dir/$sd_category/$skill_name/SKILL.md"
            break
        fi
    done

    if [[ -z "$sd_user_file" ]]; then
        json_err "${E_FILE_NOT_FOUND:-NOT_FOUND}" "No user skill named '$skill_name' found"
        return 1
    fi

    if [[ -z "$sd_system_file" ]]; then
        json_ok "$(jq -n --arg name "$skill_name" \
          '{status: "user_only", message: ("Skill \u0027" + $name + "\u0027 is user-created with no Aether equivalent")}')"
        return
    fi

    # Compare
    if diff -q "$sd_user_file" "$sd_system_file" > /dev/null 2>&1; then
        json_ok "{\"status\": \"identical\", \"message\": \"User and Aether versions are identical\"}"
    else
        local sd_diff_output
        sd_diff_output=$(diff -u "$sd_system_file" "$sd_user_file" 2>/dev/null | head -50)
        local sd_escaped_diff
        sd_escaped_diff=$(echo "$sd_diff_output" | jq -Rs '.' 2>/dev/null)
        json_ok "$(jq -n --argjson diff "$sd_escaped_diff" \
          '{status: "different", message: "User and Aether versions differ", diff: $diff}')"
    fi
}

# Check if a skill is user-created (not in manifest)
# Usage: _skill_is_user_created <skill_name> <manifest_file>
# Outputs "true" or "false" to stdout
_skill_is_user_created() {
    local skill_name="$1"
    local manifest_file="$2"
    if [[ ! -f "$manifest_file" ]]; then
        echo "true"
        return
    fi
    local suc_managed
    suc_managed=$(jq -r --arg n "$skill_name" '.skills | index($n)' "$manifest_file" 2>/dev/null)
    if [[ "$suc_managed" == "null" || -z "$suc_managed" ]]; then
        echo "true"
    else
        echo "false"
    fi
}
