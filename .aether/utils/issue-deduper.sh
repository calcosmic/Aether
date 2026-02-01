#!/bin/bash
# Aether Issue Deduplication Utility
# Implements issue fingerprinting, deduplication, and prioritization
#
# Usage:
#   source .aether/utils/issue-deduper.sh
#   fingerprint=$(create_fingerprint "Issue description" "security" "file.js:42")
#   deduped=$(dedupe_and_prioritize "$votes_file")

# Source required utilities
# Find Aether root: use git root or current directory
if git rev-parse --show-toplevel >/dev/null 2>&1; then
    AETHER_ROOT="$(git rev-parse --show-toplevel)"
else
    # Fallback: assume we're in the repo root
    AETHER_ROOT="$(pwd)"
fi

if [ -f "$AETHER_ROOT/.aether/utils/atomic-write.sh" ]; then
    source "$AETHER_ROOT/.aether/utils/atomic-write.sh"
else
    source ".aether/utils/atomic-write.sh"
fi

# Severity order for max_by comparison (Critical > High > Medium > Low)
# Critical severity has veto power in vote-aggregator.sh

# Create SHA256 fingerprint for issue deduplication
# Arguments: description, category, location
# Returns: hex string fingerprint
create_fingerprint() {
    local description="$1"
    local category="$2"
    local location="$3"

    # Combine fields and generate hash
    echo "${description}${category}${location}" | sha256sum | cut -d' ' -f1
}

# Severity value for sorting (Critical=4, High=3, Medium=2, Low=1)
get_severity_value() {
    local severity="$1"
    case "$severity" in
        "Critical") echo 4 ;;
        "High") echo 3 ;;
        "Medium") echo 2 ;;
        "Low") echo 1 ;;
        *) echo 0 ;;
    esac
}

# Get max severity from an array of severities
# Arguments: severity1, severity2, ...
# Returns: highest severity string
get_max_severity() {
    local max_sev="Low"
    local max_val=1

    for severity in "$@"; do
        local val=$(get_severity_value "$severity")
        if [ "$val" -gt "$max_val" ]; then
            max_val=$val
            max_sev="$severity"
        fi
    done

    echo "$max_sev"
}

# Deduplicate issues and prioritize by severity and weight
# Arguments: votes_file (JSON array of votes with issues)
# Returns: Aggregated issue array with fingerprints, deduping, and sorting
dedupe_and_prioritize() {
    local votes_file="$1"

    if [ ! -f "$votes_file" ]; then
        echo "Error: Votes file does not exist: $votes_file" >&2
        return 1
    fi

    # Extract all issues from all votes, create fingerprints, group by fingerprint
    # Takes highest severity among duplicates, tags as "Multiple Watchers" or "Single Watcher"
    # Sorts by severity (descending) then total_weight (descending)
    local deduped
    deduped=$(jq '
        # Extract all issues with their watcher and weight
        [.[] | .issues[]? as $issue | {
            description: $issue.description,
            severity: $issue.severity,
            category: $issue.category,
            location: $issue.location,
            watcher: .watcher,
            watcher_weight: .weight
        }] |

        # Create fingerprint for each issue
        map(.fingerprint = (.description + .category + .location | @sh)) |

        # Group by fingerprint
        group_by(.fingerprint) |

        # Aggregate each group
        map({
            description: .[0].description,
            severity: (map(.severity) | max_by({
                "Critical": 4,
                "High": 3,
                "Medium": 2,
                "Low": 1
            })),
            category: .[0].category,
            location: .[0].location,
            watchers: map(.watcher) | unique | join(", "),
            total_weight: map(.watcher_weight) | add,
            tag: (if length > 1 then "Multiple Watchers" else "Single Watcher" end)
        }) |

        # Sort by severity (descending) then total_weight (descending)
        sort_by(.severity, .total_weight) | reverse
    ' "$votes_file")

    echo "$deduped"
    return 0
}

# Sort issues by severity and return formatted list
# Arguments: issues_json (JSON array of issues)
# Returns: Sorted issues with severity prioritization
sort_by_severity() {
    local issues_json="$1"

    jq '
        sort_by(.severity) | reverse |
        map("\(.severity): \(.description) (\(.category)) @ \(.location)")
    ' <<< "$issues_json"
}

# Filter issues by severity
# Arguments: issues_json, min_severity (Critical, High, Medium, Low)
# Returns: Filtered issues at or above min_severity
filter_by_severity() {
    local issues_json="$1"
    local min_severity="$2"

    local min_val=$(get_severity_value "$min_severity")

    jq "
        map(select(
            (.severity == \"Critical\" and $min_val <= 4) or
            (.severity == \"High\" and $min_val <= 3) or
            (.severity == \"Medium\" and $min_val <= 2) or
            (.severity == \"Low\" and $min_val <= 1)
        ))
    " <<< "$issues_json"
}

# Filter issues by category
# Arguments: issues_json, category
# Returns: Filtered issues matching category
filter_by_category() {
    local issues_json="$1"
    local category="$2"

    jq "[.[] | select(.category == \"$category\")]" <<< "$issues_json"
}

# Get issue statistics
# Arguments: issues_json
# Returns: Statistics object with counts by severity and category
get_issue_stats() {
    local issues_json="$1"

    jq '{
        total: length,
        by_severity: {
            Critical: [.[] | select(.severity == "Critical")] | length,
            High: [.[] | select(.severity == "High")] | length,
            Medium: [.[] | select(.severity == "Medium")] | length,
            Low: [.[] | select(.severity == "Low")] | length
        },
        by_category: group_by(.category) | map({category: .[0].category, count: length}) | from_entries,
        multi_watcher_issues: [.[] | select(.tag == "Multiple Watchers")] | length
    }' <<< "$issues_json"
}

# Export functions
export -f create_fingerprint dedupe_and_prioritize sort_by_severity filter_by_severity filter_by_category get_issue_stats get_severity_value get_max_severity
