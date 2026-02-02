#!/bin/bash
# Aether Spawn Decision Utility
# Implements capability gap detection and specialist selection logic
#
# Usage:
#   source .aether/utils/spawn-decision.sh
#   analyze_task_requirements "task description"
#   compare_capabilities "caste" '["capability1","capability2"]'
#   detect_capability_gaps '["gaps"]' "task_type" 0
#   calculate_spawn_score 0.8 0.9 0.3 0.7 0.8
#   map_gap_to_specialist '["gaps"]' "task description"

# Source atomic-write for state operations
# We use absolute path from AETHER_ROOT or relative from this file's location
AETHER_ROOT="${AETHER_ROOT:-$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)}"
if [ -f "${AETHER_ROOT}/.aether/utils/atomic-write.sh" ]; then
    source "${AETHER_ROOT}/.aether/utils/atomic-write.sh"
else
    # Try relative path from current directory
    if [ -f ".aether/utils/atomic-write.sh" ]; then
        source ".aether/utils/atomic-write.sh"
    fi
fi

# Source Bayesian confidence library for meta-learning recommendations
if [ -f "${AETHER_ROOT}/.aether/utils/bayesian-confidence.sh" ]; then
    source "${AETHER_ROOT}/.aether/utils/bayesian-confidence.sh"
else
    # Try relative path from current directory
    if [ -f ".aether/utils/bayesian-confidence.sh" ]; then
        source ".aether/utils/bayesian-confidence.sh"
    fi
fi

# Colony state file (contains meta_learning.specialist_confidence)
COLONY_STATE_FILE="${AETHER_ROOT}/.aether/data/COLONY_STATE.json"
# Keep WORKER_ANTS_FILE for caste capabilities/mappings
WORKER_ANTS_FILE="${AETHER_ROOT}/.aether/data/worker_ants.json"

# Meta-learning configuration
MIN_CONFIDENCE_FOR_RECOMMENDATION=0.7  # 70% - minimum confidence to use meta-learning
MIN_SAMPLES_FOR_RECOMMENDATION=5      # Minimum spawns before trusting confidence
META_LEARNING_ENABLED=true             # Can be disabled to fall back to semantic-only

# Capability taxonomy
TECHNICAL_DOMAINS="database|frontend|backend|api|security|testing|performance|devops"
COMMON_FRAMEWORKS="react|vue|angular|django|fastapi|flask|express|spring|rails"
SKILL_TYPES="analysis|planning|implementation|validation|design|research"

# Analyze task requirements and extract capabilities
# Arguments: task_description
# Returns: JSON array of required capabilities
analyze_task_requirements() {
    local task_description="$1"

    # Convert to lowercase for matching
    local task_lower=$(echo "$task_description" | tr '[:upper:]' '[:lower:]')

    # Initialize capabilities array
    local capabilities=()

    # Extract technical domains
    if [[ "$task_lower" =~ (database|sql|nosql|mongo|postgres|mysql|schema|migration|query) ]]; then
        capabilities+=("database")
    fi
    if [[ "$task_lower" =~ (frontend|ui|ux|css|html|component|view|template) ]]; then
        capabilities+=("frontend")
    fi
    if [[ "$task_lower" =~ (backend|server|api|endpoint|route|controller|service) ]]; then
        capabilities+=("backend")
    fi
    if [[ "$task_lower" =~ (api|rest|graphql|endpoint|webhook|integration) ]]; then
        capabilities+=("api")
    fi
    if [[ "$task_lower" =~ (security|auth|authentication|authorization|encryption|csrf|xss) ]]; then
        capabilities+=("security")
    fi
    if [[ "$task_lower" =~ (test|testing|spec|validation|verify|quality|assert) ]]; then
        capabilities+=("testing")
    fi
    if [[ "$task_lower" =~ (performance|optimization|cache|speed|latency|scalability|efficient) ]]; then
        capabilities+=("performance")
    fi
    if [[ "$task_lower" =~ (devops|deploy|ci/cd|pipeline|docker|kubernetes|infrastructure) ]]; then
        capabilities+=("devops")
    fi

    # Extract frameworks
    if [[ "$task_lower" =~ react ]]; then
        capabilities+=("react")
    fi
    if [[ "$task_lower" =~ vue ]]; then
        capabilities+=("vue")
    fi
    if [[ "$task_lower" =~ angular ]]; then
        capabilities+=("angular")
    fi
    if [[ "$task_lower" =~ django ]]; then
        capabilities+=("django")
    fi
    if [[ "$task_lower" =~ fastapi ]]; then
        capabilities+=("fastapi")
    fi
    if [[ "$task_lower" =~ flask ]]; then
        capabilities+=("flask")
    fi
    if [[ "$task_lower" =~ express ]]; then
        capabilities+=("express")
    fi

    # Extract skills
    if [[ "$task_lower" =~ (analyze|analysis|research|investigate|explore) ]]; then
        capabilities+=("analysis")
    fi
    if [[ "$task_lower" =~ (plan|planning|design|architect|structure) ]]; then
        capabilities+=("planning")
    fi
    if [[ "$task_lower" =~ (implement|build|create|write|code|develop) ]]; then
        capabilities+=("implementation")
    fi
    if [[ "$task_lower" =~ (validat|verify|check|test|review|inspect) ]]; then
        capabilities+=("validation")
    fi

    # Remove duplicates and build JSON array
    local unique_capabilities=($(printf "%s\n" "${capabilities[@]}" | sort -u))

    # Build JSON array
    local json_array="["
    local first=true
    for cap in "${unique_capabilities[@]}"; do
        if [ "$first" = true ]; then
            json_array+="\"$cap\""
            first=false
        else
            json_array+=",\"$cap\""
        fi
    done
    json_array+="]"

    echo "$json_array"
}

# Compare required capabilities to caste capabilities
# Arguments: caste, required_capabilities (JSON array)
# Returns: JSON object with gaps array and coverage_percentage
compare_capabilities() {
    local caste="$1"
    local required_capabilities="$2"

    # Get caste capabilities from worker_ants.json
    local caste_capabilities=$(jq -r ".castes.$caste.capabilities // []" "$WORKER_ANTS_FILE")

    # Convert arrays to bash-compatible format
    local required_list=$(echo "$required_capabilities" | jq -r '.[]')
    local available_list=$(echo "$caste_capabilities" | jq -r '.[]')

    # Find gaps (required but not available)
    local gaps=()
    while IFS= read -r req; do
        # Check if requirement is in available capabilities
        local found=false
        while IFS= read -r avail; do
            # Direct match or substring match
            if [[ "$req" == "$avail" ]] || [[ "$avail" =~ $req ]] || [[ "$req" =~ $avail ]]; then
                found=true
                break
            fi
        done <<< "$available_list"

        if [ "$found" = false ]; then
            gaps+=("$req")
        fi
    done <<< "$required_list"

    # Calculate coverage percentage
    local required_count=$(echo "$required_capabilities" | jq '. | length')
    local gaps_count=${#gaps[@]}

    local coverage=0
    if [ "$required_count" -gt 0 ]; then
        coverage=$(awk "BEGIN {printf \"%.2f\", (($required_count - $gaps_count) / $required_count) * 100}")
    fi

    # Build gaps JSON array
    local gaps_json="["
    local first=true
    for gap in "${gaps[@]}"; do
        if [ "$first" = true ]; then
            gaps_json+="\"$gap\""
            first=false
        else
            gaps_json+=",\"$gap\""
        fi
    done
    gaps_json+="]"

    # Build result JSON
    local result=$(jq -n \
        --argjson gaps "$gaps_json" \
        --argjson coverage "$coverage" \
        '{gaps: $gaps, coverage_percentage: $coverage}')

    echo "$result"
}

# Detect capability gaps and decide whether to spawn
# Arguments: gaps (JSON array), task_type, failure_count
# Returns: JSON with decision and reason
detect_capability_gaps() {
    local gaps="$1"
    local task_type="$2"
    local failure_count="$3"

    local gaps_count=$(echo "$gaps" | jq '. | length')
    local decision="attempt"
    local reason="No significant capability gaps detected"

    # Explicit domain mismatch
    if [ "$gaps_count" -gt 0 ]; then
        decision="spawn"
        reason="Explicit domain mismatch: missing capabilities: $(echo "$gaps" | jq -r '. | join(", ")')"
    fi

    # Failure after attempts
    if [ "$failure_count" -gt 0 ]; then
        decision="spawn"
        reason="Failure after $failure_count attempts, spawning specialist for assistance"
    fi

    # Pattern recognition from meta_learning (placeholder for now)
    # This will be enhanced in later phases with actual meta-learning integration

    # Build result JSON
    local result=$(jq -n \
        --arg decision "$decision" \
        --arg reason "$reason" \
        '{decision: $decision, reason: $reason}')

    echo "$result"
}

# Calculate spawn score using multi-factor scoring
# Arguments: gap_score, priority, load, budget_remaining, resources
# Returns: float score (0.0 - 1.0)
calculate_spawn_score() {
    local gap_score="$1"
    local priority="$2"
    local load="$3"
    local budget_remaining="$4"
    local resources="$5"

    # Multi-factor scoring formula
    local spawn_score=$(awk "BEGIN {
        score = ($gap_score * 0.40) + \
                ($priority * 0.20) + \
                ($load * 0.15) + \
                ($budget_remaining * 0.15) + \
                ($resources * 0.10)
        printf \"%.2f\", score
    }")

    echo "$spawn_score"
}

# Recommend specialist by Bayesian confidence for a task type
# Arguments: task_type, min_confidence (default 0.7), min_samples (default 5)
# Returns: "specialist_caste|confidence" or "none|0.0" if no confident recommendation
recommend_specialist_by_confidence() {
    local task_type="$1"
    local min_confidence="${2:-$MIN_CONFIDENCE_FOR_RECOMMENDATION}"
    local min_samples="${3:-$MIN_SAMPLES_FOR_RECOMMENDATION}"

    # Validate meta-learning is enabled
    if [ "$META_LEARNING_ENABLED" != "true" ]; then
        echo "none|0.0"
        return 0
    fi

    # Validate COLONY_STATE_FILE exists
    if [ ! -f "$COLONY_STATE_FILE" ]; then
        echo "none|0.0"
        return 0
    fi

    # Find best specialist for this task type from COLONY_STATE.json
    local best=$(jq -r "
        .meta_learning.specialist_confidence |
        to_entries[] |
        select(.value | has(\"$task_type\")) |
        select(.value.\"$task_type\".total_spawns >= $min_samples) |
        select(.value.\"$task_type\".confidence >= $min_confidence) |
        \"\(.key)|\(.value.\"$task_type\".confidence)\"
    " "$COLONY_STATE_FILE" 2>/dev/null | sort -t'|' -k2 -nr | head -1)

    if [ -n "$best" ] && [[ "$best" != *"|"* ]]; then
        # Handle jq error case (no valid data)
        echo "none|0.0"
    elif [ -n "$best" ]; then
        echo "$best"
    else
        echo "none|0.0"
    fi
}

# Get all specialists with weighted confidence scores for a task type
# Arguments: task_type
# Returns: Ranked list of specialists with alpha/beta/confidence/weight/weighted_score
get_weighted_specialist_scores() {
    local task_type="$1"

    # Validate COLONY_STATE_FILE exists
    if [ ! -f "$COLONY_STATE_FILE" ]; then
        return 1
    fi

    # Get all specialists with this task type
    jq -r "
        .meta_learning.specialist_confidence |
        to_entries[] |
        select(.value | has(\"$task_type\")) |
        \"\(.key)|\(.value.\"$task_type\".alpha)|\(.value.\"$task_type\".beta)|\(.value.\"$task_type\".confidence)|\(.value.\"$task_type\".total_spawns)\"
    " "$COLONY_STATE_FILE" 2>/dev/null | while IFS='|' read -r specialist alpha beta confidence total_spawns; do
        # Skip if data missing
        [ -z "$specialist" ] && continue

        # Calculate sample size weight (min 10 samples for full weight)
        local weight=$(echo "scale=6; $total_spawns / 10" | bc)
        if (( $(echo "$weight > 1.0" | bc -l) )); then
            weight=1.0
        fi

        # Apply weighting: weighted = raw * (0.5 + 0.5 * weight)
        local weighted=$(echo "scale=6; $confidence * (0.5 + 0.5 * $weight)" | bc)

        echo "$specialist|$alpha|$beta|$confidence|$total_spawns|$weight|$weighted"
    done | sort -t'|' -k7 -nr  # Sort by weighted confidence descending
}

# Map capability gaps to appropriate specialist caste
# Arguments: gaps (JSON array), task_description
# Returns: JSON with caste name and specialization
map_gap_to_specialist() {
    local gaps="$1"
    local task_description="$2"

    # Convert to lowercase for matching
    local task_lower=$(echo "$task_description" | tr '[:upper:]' '[:lower:]')

    # Get specialist mappings from worker_ants.json
    local mappings=$(jq -r '.specialist_mappings.capability_to_caste // {}' "$WORKER_ANTS_FILE")

    # Try direct mapping first
    local specialist_caste=""
    local specialization=""

    # Check each gap for direct mapping
    while IFS= read -r gap; do
        # Try exact match in mappings
        local mapped_caste=$(echo "$mappings" | jq -r --arg gap "$gap" '.[$gap] // empty')

        if [ -n "$mapped_caste" ]; then
            specialist_caste="$mapped_caste"
            specialization="$gap specialist"
            break
        fi
    done <<< "$(echo "$gaps" | jq -r '.[]')"

    # Fallback to semantic analysis if no direct mapping
    if [ -z "$specialist_caste" ]; then
        # Semantic analysis based on task description
        if [[ "$task_lower" =~ (database|sql|nosql|mongo|postgres|schema|migration) ]]; then
            specialist_caste="scout"
            specialization="database expert"
        elif [[ "$task_lower" =~ (security|auth|authentication|authorization|encryption) ]]; then
            specialist_caste="watcher"
            specialization="security specialist"
        elif [[ "$task_lower" =~ (test|testing|validation|verification) ]]; then
            specialist_caste="watcher"
            specialization="testing specialist"
        elif [[ "$task_lower" =~ (api|endpoint|route|rest|graphql) ]]; then
            specialist_caste="route_setter"
            specialization="api design expert"
        elif [[ "$task_lower" =~ (react|vue|angular|frontend|ui|component) ]]; then
            specialist_caste="builder"
            specialization="frontend specialist"
        elif [[ "$task_lower" =~ (backend|server|service|controller) ]]; then
            specialist_caste="builder"
            specialization="backend specialist"
        elif [[ "$task_lower" =~ (performance|optimization|cache|scalability) ]]; then
            specialist_caste="architect"
            specialization="performance optimizer"
        elif [[ "$task_lower" =~ (document|documentation|readme|guide) ]]; then
            specialist_caste="scout"
            specialization="documentation expert"
        elif [[ "$task_lower" =~ (infrastructure|deploy|devops|ci/cd|docker) ]]; then
            specialist_caste="builder"
            specialization="infrastructure specialist"
        elif [[ "$task_lower" =~ (research|investigate|explore|analyze) ]]; then
            specialist_caste="scout"
            specialization="research specialist"
        elif [[ "$task_lower" =~ (plan|planning|design|architect) ]]; then
            specialist_caste="route_setter"
            specialization="planning specialist"
        elif [[ "$task_lower" =~ (compress|memory|pattern|synthesize) ]]; then
            specialist_caste="architect"
            specialization="memory specialist"
        else
            # Default fallback: scout for information gathering
            specialist_caste="scout"
            specialization="general specialist"
        fi
    fi

    # Build result JSON
    local result=$(jq -n \
        --arg caste "$specialist_caste" \
        --arg specialization "$specialization" \
        '{caste: $caste, specialization: $specialization}')

    echo "$result"
}

# Export functions
export -f analyze_task_requirements
export -f compare_capabilities
export -f detect_capability_gaps
export -f calculate_spawn_score
export -f map_gap_to_specialist
export -f recommend_specialist_by_confidence
export -f get_weighted_specialist_scores
