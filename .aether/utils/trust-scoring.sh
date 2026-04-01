#!/bin/bash
# Trust Scoring utility functions — Aether Structural Learning Stack
# Provides: _trust_calculate, _trust_decay, _trust_tier
#
# These functions are sourced by aether-utils.sh at startup.
# All shared infrastructure (json_ok, json_err, json_warn, atomic_write,
# DATA_DIR, COLONY_DATA_DIR, SCRIPT_DIR, error constants) is available.
#
# This is a pure calculation module — no state is read or written.
# All functions accept --flag <value> arguments and return JSON via json_ok.

# ============================================================================
# _trust_calculate
# Calculate a weighted trust score from source, evidence, and activity inputs.
#
# Weights: source 40%, evidence 35%, activity 25%
# Activity uses a 60-day half-life decay from days_since_last_use.
# Floor: score is never below 0.2.
#
# Usage: trust-calculate --source <type> --evidence <type> --days-since <N>
#
# Source types and weights:
#   user_feedback   1.0
#   error_resolution 0.9
#   success_pattern 0.8
#   observation     0.6
#   heuristic       0.4
#
# Evidence types and weights:
#   test_verified   1.0
#   multi_phase     0.9
#   single_phase    0.7
#   anecdotal       0.4
#
# Output: {score, source_score, evidence_score, activity_score, tier}
# ============================================================================
_trust_calculate() {
    local source_type=""
    local evidence_type=""
    local days_since=""

    # Parse --flag value arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --source)
                source_type="${2:-}"
                shift 2
                ;;
            --evidence)
                evidence_type="${2:-}"
                shift 2
                ;;
            --days-since)
                days_since="${2:-}"
                shift 2
                ;;
            *)
                json_err "$E_VALIDATION_FAILED" "Usage: trust-calculate --source <type> --evidence <type> --days-since <N>"
                return
                ;;
        esac
    done

    [[ -z "$source_type" || -z "$evidence_type" || -z "$days_since" ]] && \
        json_err "$E_VALIDATION_FAILED" "Usage: trust-calculate --source <type> --evidence <type> --days-since <N>"

    # Map source type to score
    local source_score
    case "$source_type" in
        user_feedback)    source_score="1.0" ;;
        error_resolution) source_score="0.9" ;;
        success_pattern)  source_score="0.8" ;;
        observation)      source_score="0.6" ;;
        heuristic)        source_score="0.4" ;;
        *)
            json_err "$E_VALIDATION_FAILED" "Unknown source type: $source_type. Valid: user_feedback, error_resolution, success_pattern, observation, heuristic"
            return
            ;;
    esac

    # Map evidence type to score
    local evidence_score
    case "$evidence_type" in
        test_verified) evidence_score="1.0" ;;
        multi_phase)   evidence_score="0.9" ;;
        single_phase)  evidence_score="0.7" ;;
        anecdotal)     evidence_score="0.4" ;;
        *)
            json_err "$E_VALIDATION_FAILED" "Unknown evidence type: $evidence_type. Valid: test_verified, multi_phase, single_phase, anecdotal"
            return
            ;;
    esac

    # Validate days_since is a non-negative integer
    if ! [[ "$days_since" =~ ^[0-9]+$ ]]; then
        json_err "$E_VALIDATION_FAILED" "--days-since must be a non-negative integer, got: $days_since"
        return
    fi

    # Calculate activity score using 60-day half-life decay
    # activity = 0.5 ^ (days / 60)
    local activity_score
    activity_score=$(_trust_halflife_decay "1.0" "$days_since" "60")

    # Weighted formula: 0.4 * source + 0.35 * evidence + 0.25 * activity
    local raw_score
    if command -v bc &>/dev/null; then
        raw_score=$(echo "scale=6; 0.4 * $source_score + 0.35 * $evidence_score + 0.25 * $activity_score" | bc)
    else
        raw_score=$(awk "BEGIN{printf \"%.6f\", 0.4 * $source_score + 0.35 * $evidence_score + 0.25 * $activity_score}")
    fi

    # Apply floor of 0.2
    local score
    score=$(_trust_apply_floor "$raw_score" "0.2")

    # Derive tier from final score
    local tier
    tier=$(_trust_score_to_tier "$score")

    json_ok "$(jq -n \
        --argjson score "$score" \
        --argjson source_score "$source_score" \
        --argjson evidence_score "$evidence_score" \
        --argjson activity_score "$activity_score" \
        --arg tier "$tier" \
        '{
            score: $score,
            source_score: $source_score,
            evidence_score: $evidence_score,
            activity_score: $activity_score,
            tier: $tier
        }')"
}

# ============================================================================
# _trust_decay
# Apply time-based half-life decay to an existing trust score.
# The score never drops below the floor of 0.2.
#
# Formula: decayed = max(0.2, score * (0.5 ^ (days / 60)))
#
# Usage: trust-decay --score <float> --days <N>
# Output: {original, decayed, days, half_life: 60}
# ============================================================================
_trust_decay() {
    local score=""
    local days=""

    while [[ $# -gt 0 ]]; do
        case "$1" in
            --score)
                score="${2:-}"
                shift 2
                ;;
            --days)
                days="${2:-}"
                shift 2
                ;;
            *)
                json_err "$E_VALIDATION_FAILED" "Usage: trust-decay --score <float> --days <N>"
                return
                ;;
        esac
    done

    [[ -z "$score" || -z "$days" ]] && \
        json_err "$E_VALIDATION_FAILED" "Usage: trust-decay --score <float> --days <N>"

    # Validate score is a positive number
    if ! [[ "$score" =~ ^[0-9]+(\.[0-9]+)?$ ]]; then
        json_err "$E_VALIDATION_FAILED" "--score must be a non-negative number, got: $score"
        return
    fi

    # Validate days is a non-negative integer
    if ! [[ "$days" =~ ^[0-9]+$ ]]; then
        json_err "$E_VALIDATION_FAILED" "--days must be a non-negative integer, got: $days"
        return
    fi

    local decayed
    decayed=$(_trust_halflife_decay "$score" "$days" "60")
    local final
    final=$(_trust_apply_floor "$decayed" "0.2")

    json_ok "$(jq -n \
        --argjson original "$score" \
        --argjson decayed "$final" \
        --argjson days "$days" \
        --argjson half_life 60 \
        '{
            original: $original,
            decayed: $decayed,
            days: $days,
            half_life: $half_life
        }')"
}

# ============================================================================
# _trust_tier
# Map a trust score to a named tier.
#
# Tier table:
#   0.90–1.00  canonical   (index 0) — proven across multiple contexts
#   0.80–0.89  trusted     (index 1) — reliable with strong evidence
#   0.70–0.79  established (index 2) — consistent but limited evidence
#   0.60–0.69  emerging    (index 3) — promising, needs more validation
#   0.45–0.59  provisional (index 4) — early stage, minimal evidence
#   0.30–0.44  suspect     (index 5) — weak evidence or conflicting signals
#   0.20–0.29  dormant     (index 6) — inactive, near floor
#
# Usage: trust-tier --score <float>
# Output: {score, tier, tier_index}
# ============================================================================
_trust_tier() {
    local score=""

    while [[ $# -gt 0 ]]; do
        case "$1" in
            --score)
                score="${2:-}"
                shift 2
                ;;
            *)
                json_err "$E_VALIDATION_FAILED" "Usage: trust-tier --score <float>"
                return
                ;;
        esac
    done

    [[ -z "$score" ]] && \
        json_err "$E_VALIDATION_FAILED" "Usage: trust-tier --score <float>"

    # Validate score is a non-negative number
    if ! [[ "$score" =~ ^[0-9]+(\.[0-9]+)?$ ]]; then
        json_err "$E_VALIDATION_FAILED" "--score must be a non-negative number, got: $score"
        return
    fi

    local tier
    local tier_index
    tier=$(_trust_score_to_tier "$score")
    tier_index=$(_trust_tier_to_index "$tier")

    json_ok "$(jq -n \
        --argjson score "$score" \
        --arg tier "$tier" \
        --argjson tier_index "$tier_index" \
        '{
            score: $score,
            tier: $tier,
            tier_index: $tier_index
        }')"
}

# ============================================================================
# Internal helpers — not exposed as subcommands
# ============================================================================

# _trust_halflife_decay: compute score * (0.5 ^ (days / half_life))
# Usage: _trust_halflife_decay <score> <days> <half_life>
# Outputs the decayed value as a float string
_trust_halflife_decay() {
    local score="$1"
    local days="$2"
    local half_life="$3"

    if command -v bc &>/dev/null; then
        echo "scale=6; $score * e(l(0.5) * ($days / $half_life))" | bc -l
    else
        awk "BEGIN{printf \"%.6f\", $score * (0.5 ^ ($days / $half_life))}"
    fi
}

# _trust_apply_floor: return max(floor, value)
# Usage: _trust_apply_floor <value> <floor>
# Outputs the floored value as a float string
_trust_apply_floor() {
    local value="$1"
    local floor="$2"

    if command -v bc &>/dev/null; then
        local cmp
        cmp=$(echo "$value < $floor" | bc -l)
        if [[ "$cmp" == "1" ]]; then
            echo "$floor"
        else
            echo "$value"
        fi
    else
        awk "BEGIN{v=$value; f=$floor; printf \"%.6f\", (v < f ? f : v)}"
    fi
}

# _trust_score_to_tier: map a numeric score to a tier name
# Usage: _trust_score_to_tier <score>
# Outputs the tier name string
_trust_score_to_tier() {
    local score="$1"

    if command -v bc &>/dev/null; then
        if [[ $(echo "$score >= 0.90" | bc -l) == "1" ]]; then
            echo "canonical"
        elif [[ $(echo "$score >= 0.80" | bc -l) == "1" ]]; then
            echo "trusted"
        elif [[ $(echo "$score >= 0.70" | bc -l) == "1" ]]; then
            echo "established"
        elif [[ $(echo "$score >= 0.60" | bc -l) == "1" ]]; then
            echo "emerging"
        elif [[ $(echo "$score >= 0.45" | bc -l) == "1" ]]; then
            echo "provisional"
        elif [[ $(echo "$score >= 0.30" | bc -l) == "1" ]]; then
            echo "suspect"
        else
            echo "dormant"
        fi
    else
        awk "BEGIN{
            s = $score
            if (s >= 0.90)      print \"canonical\"
            else if (s >= 0.80) print \"trusted\"
            else if (s >= 0.70) print \"established\"
            else if (s >= 0.60) print \"emerging\"
            else if (s >= 0.45) print \"provisional\"
            else if (s >= 0.30) print \"suspect\"
            else                print \"dormant\"
        }"
    fi
}

# _trust_tier_to_index: map tier name to integer index
# Usage: _trust_tier_to_index <tier_name>
# Outputs the index as an integer string
_trust_tier_to_index() {
    local tier="$1"
    case "$tier" in
        canonical)    echo "0" ;;
        trusted)      echo "1" ;;
        established)  echo "2" ;;
        emerging)     echo "3" ;;
        provisional)  echo "4" ;;
        suspect)      echo "5" ;;
        dormant)      echo "6" ;;
        *)            echo "-1" ;;
    esac
}
