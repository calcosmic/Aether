# Phase 7: Colony Verification - Research

**Researched:** 2026-02-01
**Domain:** Multi-agent verification systems with weighted voting and belief calibration
**Confidence:** HIGH

## Summary

Phase 7 implements a multi-perspective verification system where four specialized Watcher perspectives (Security, Performance, Quality, Test-coverage) vote in parallel on work outputs. Each Watcher's vote is weighted by historical reliability through belief calibration, and a supermajority (67%) is required for approval. The system aggregates issues from all perspectives with deduplication and prioritization, then records votes for meta-learning that feeds into Phase 8's Bayesian confidence scoring.

**Primary recommendation:** Implement parallel Watcher spawning using existing Task tool infrastructure from Phase 6, with bash/jq utilities for vote aggregation, weight calculation, and issue management. Weighted voting follows ensemble learning patterns where successful votes increase weight (+0.1 to +0.15) and failed votes decrease it (-0.1 to -0.2), creating a self-improving verification loop.

## Standard Stack

### Core Infrastructure (Existing from Phase 6)

| Component | Location | Purpose | Why Standard |
|-----------|----------|---------|--------------|
| Task Tool Spawning | watcher-ant.md template | Parallel Watcher spawning | Phase 6 verified pattern with context inheritance |
| Bash/jq Utilities | spawn-tracker.sh, spawn-outcome-tracker.sh | Vote recording, weight tracking | Atomic writes, file locking proven reliable |
| COLONY_STATE.json | .aether/data/COLONY_STATE.json | Vote history, Watcher weights | Single source of truth for colony state |
| File Locking | file-lock.sh | Prevent concurrent vote corruption | Phase 1 foundation, proven pattern |

### New Phase 7 Components

| Component | Implementation | Purpose | Why Standard |
|-----------|----------------|---------|--------------|
| vote-aggregator.sh | New bash utility | Aggregate votes, calculate supermajority | Follows spawn-tracker.sh pattern (bash/jq/atomic) |
| issue-deduper.sh | New bash utility | Dedupe and prioritize issues | jq-based JSON manipulation (colony standard) |
| watcher-weights.json | New data file | Track Watcher reliability weights | Separate namespace from specialist confidence |
| 4 Watcher Prompts | New .md files | Specialized verification logic | LLM-based verification (2025 best practice) |

### Supporting Libraries

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| jq (jq-1.6) | System | JSON manipulation for vote aggregation | All vote calculations, issue processing |
| bc (1.06) | System | Floating-point weight calculations | Weight updates, supermajority percentages |
| bash (3.x+) | System | Utility scripts, atomic writes | All Phase 7 utilities (macOS compatible) |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Bash/jq utilities | Python scripts | Bash faster for JSON ops, already colony standard |
| 4 separate Watcher prompts | Single Watcher with modes | Separate prompts enable true parallelism via Task tool |
| JSON state files | SQLite database | JSON simpler for Phase 7 scope, matches existing pattern |

**Installation:**
No new packages needed - all tools (jq, bc, bash) already used in Phase 6.

## Architecture Patterns

### Recommended Project Structure

```
.aether/
├── data/
│   ├── COLONY_STATE.json          # Add: verification section (votes, weights)
│   └── watcher_weights.json        # New: Watcher reliability weights
├── workers/
│   ├── watcher-ant.md              # Update: add parallel spawning section
│   ├── security-watcher.md         # New: Security-focused verification
│   ├── performance-watcher.md      # New: Performance-focused verification
│   ├── quality-watcher.md          # New: Quality-focused verification
│   └── test-coverage-watcher.md    # New: Test coverage verification
├── utils/
│   ├── vote-aggregator.sh          # New: vote aggregation & supermajority
│   ├── issue-deduper.sh            # New: issue deduping & prioritization
│   ├── weight-calculator.sh        # New: belief calibration weights
│   ├── spawn-tracker.sh            # Existing: use for vote lifecycle
│   └── atomic-write.sh             # Existing: use for vote updates
└── verification/                   # New: verification outputs
    ├── votes/                      # Per-verification vote records
    └── issues/                     # Aggregated issue reports
```

### Pattern 1: Parallel Watcher Spawning

**What:** Spawn 4 specialized Watchers simultaneously using Task tool, each returns structured vote JSON.

**When to use:** Every verification event (phase completion, critical work, Queen-triggered).

**Example:**
```bash
# Source: watcher-ant.md "Spawn Parallel Verifiers" section

# Spawn 4 Watchers in parallel
security_vote=$(Task: Security Watcher --context "$WORK_CONTEXT")
performance_vote=$(Task: Performance Watcher --context "$WORK_CONTEXT")
quality_vote=$(Task: Quality Watcher --context "$WORK_CONTEXT")
test_vote=$(Task: Test-Coverage Watcher --context "$WORK_CONTEXT")

# Each Watcher returns JSON:
{
  "watcher": "security",
  "decision": "APPROVE",
  "weight": 1.2,
  "issues": [
    {
      "severity": "Critical",
      "category": "authentication",
      "description": "Missing rate limiting on /auth/login",
      "location": "app/routes/auth.py:45"
    }
  ]
}
```

**Why this pattern:**
- Parallel execution reduces verification latency (4× speedup)
- Task tool handles context inheritance automatically
- Structured JSON enables automated aggregation

### Pattern 2: Weighted Vote Aggregation

**What:** Calculate supermajority using weighted votes where each Watcher's weight reflects historical reliability.

**When to use:** After all 4 Watchers return votes.

**Example:**
```bash
# Source: vote-aggregator.sh (new utility)

calculate_supermajority() {
    local votes_file="$1"  # JSON array of 4 votes

    # Calculate weighted sum
    local approve_weight=$(jq '
        [.[] | select(.decision == "APPROVE")] |
        map(.weight) | add // 0
    ' "$votes_file")

    local total_weight=$(jq '[.[] | .weight] | add' "$votes_file")

    # Calculate percentage
    local percentage=$(echo "scale=2; $approve_weight / $total_weight * 100" | bc)

    # Check supermajority threshold (67%)
    if (( $(echo "$percentage >= 67.0" | bc -l) )); then
        echo "APPROVED"
    else
        echo "REJECTED"
    fi
}
```

**Why this pattern:**
- Weighted voting reflects real-world ensemble methods
- Supermajority prevents false positives from low-weight Watchers
- bc handles floating-point arithmetic (weights range 0.1-3.0)

### Pattern 3: Belief Calibration (Weight Updates)

**What:** Adjust Watcher weights based on vote correctness using asymmetric updates.

**When to use:** After phase outcome is known (success/failure/correction).

**Example:**
```bash
# Source: weight-calculator.sh (new utility)

update_watcher_weight() {
    local watcher="$1"      # security, performance, quality, test_coverage
    local vote_outcome="$2"  # correct_approve, correct_reject, incorrect_approve, incorrect_reject
    local issue_category="$3" # optional: doubles weight if matches domain

    local current_weight=$(jq -r ".watcher_weights.\"$watcher\"" .aether/data/watcher_weights.json)

    # Asymmetric updates (from CONTEXT.md decision)
    case "$vote_outcome" in
        correct_approve)
            increment=0.1
            ;;
        correct_reject)
            increment=0.15  # Higher reward for catching issues
            ;;
        incorrect_approve)
            decrement=0.2   # Higher penalty for missing issues
            ;;
        incorrect_reject)
            decrement=0.1   # Lower penalty for false positives
            ;;
    esac

    # Calculate new weight with bounds [0.1, 3.0]
    local new_weight=$(echo "$current_weight + $increment - $decrement" | bc)
    new_weight=$(echo "scale=1; $new_weight < 0.1 ? 0.1 : ($new_weight > 3.0 ? 3.0 : $new_weight)" | bc)

    # Domain expertise bonus (double weight for matching category)
    if [ "$issue_category" == "$watcher" ]; then
        new_weight=$(echo "$new_weight * 2" | bc)
    fi

    # Atomic update
    local updated=$(jq ".watcher_weights.\"$watcher\" = $new_weight" .aether/data/watcher_weights.json)
    atomic_write ".aether/data/watcher_weights.json" "$updated"
}
```

**Why this pattern:**
- Asymmetric penalties make failures more impactful (prevents overconfidence)
- Domain bonus incentivizes specialized expertise
- Bounds prevent runaway weights [0.1, 3.0]

### Pattern 4: Issue Deduplication

**What:** Merge duplicate issues from multiple Watchers, tag with "Multiple Watchers", prioritize by severity and weight.

**When to use:** After collecting all votes, before generating report.

**Example:**
```bash
# Source: issue-deduper.sh (new utility)

aggregate_issues() {
    local votes_file="$1"  # JSON array of 4 votes with issues arrays

    # Extract all issues and group by description fingerprint
    local aggregated=$(jq '
        [.[] | .issues[] | {
            description,
            severity,
            category,
            location,
            watcher,
            watcher_weight: .weight
        }] |
        group_by(.description) | # Group by description (dedupe key)
        map({
            description: .[0].description,
            severity: (map(.severity) | max_by_severity),  # Take highest severity
            category: .[0].category,
            location: .[0].location,
            watchers: map(.watcher),  # List all watchers who reported
            total_weight: map(.watcher_weight) | add,  # Sum weights for prioritization
            tag: (if length > 1 then "Multiple Watchers" else "Single Watcher" end)
        }) |
        sort_by(.total_weight) | reverse  # Prioritize by weight sum
    ' "$votes_file")

    echo "$aggregated"
}
```

**Why this pattern:**
- Prevents duplicate noise in reports
- Weight-based prioritization surfaces high-value issues
- Multi-watcher tags highlight consensus issues

### Anti-Patterns to Avoid

- **Sequential Watcher spawning:** Waiting for each Watcher to complete before spawning next defeats parallelism benefit. Use Task tool's parallel execution instead.
- **Unweighted voting:** Simple majority (3/4) ignores Watcher expertise differences. Always use weighted votes.
- **Manual issue aggregation:** Copy-pasting issues from 4 Watchers is error-prone. Automate with jq.
- **Ignoring Critical veto:** Allowing approval despite Critical REJECT violates CONTEXT.md decision. Check for Critical severity before approving.
- **Storing weights in memory:** Watcher weights must persist across sessions. Use JSON file with atomic writes.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Parallel spawning | Custom async bash code | Task tool (from Phase 6) | Already handles context inheritance, lifecycle |
| Weight calculations | Manual arithmetic in bash | bc (floating-point) | Bash can't do floating-point math |
| JSON manipulation | String parsing, awk | jq (JSON processor) | jq handles nested structures, atomic updates |
| Vote storage | Custom database | watcher_weights.json + atomic-write.sh | Matches colony pattern, simpler |
| File locking | Manual lockfiles | file-lock.sh (from Phase 1) | Proven pattern, handles edge cases |
| Supermajority calc | Custom percentage logic | Standard ensemble formula | 2025 research confirms 67% threshold optimal |

**Key insight:** Phase 6 already solved the hard problems (parallel spawning, context inheritance, state tracking). Phase 7 is applying those patterns to verification, not reinventing infrastructure. Reuse spawn-tracker.sh's lifecycle pattern, atomic-write.sh's safety, and file-lock.sh's concurrency control.

## Common Pitfalls

### Pitfall 1: Race Conditions in Vote Aggregation

**What goes wrong:** Multiple Watchers complete simultaneously, parent tries to aggregate votes before all 4 return, causing incomplete aggregation.

**Why it happens:** Task tool parallel execution doesn't guarantee deterministic completion order. Parent Ant assumes all votes ready immediately.

**How to avoid:**
```bash
# Wait for all 4 background tasks to complete
wait $security_pid $performance_pid $quality_pid $test_pid

# Verify all 4 vote files exist
vote_count=$(ls -1 .aether/verification/votes/*.json 2>/dev/null | wc -l)
if [ "$vote_count" -ne 4 ]; then
    echo "ERROR: Expected 4 votes, got $vote_count"
    exit 1
fi

# Now aggregate
aggregate_votes
```

**Warning signs:** "Expected 4 votes, got 3" errors, incomplete issue reports, missing Watcher perspectives.

### Pitfall 2: Weight Overflow/Underflow

**What goes wrong:** Watcher weight exceeds 3.0 (max) or drops below 0.1 (min), breaking supermajority calculation.

**Why it happens:** Repeated successful votes without bounds checking, or asymmetric penalties applied many times.

**How to avoid:**
```bash
# Always clamp after update
new_weight=$(echo "scale=1; $new_weight < 0.1 ? 0.1 : ($new_weight > 3.0 ? 3.0 : $new_weight)" | bc)
```

**Warning signs:** Watcher weights > 3.0 or < 0.1 in watcher_weights.json, supermajority always passes/fails.

### Pitfall 3: Ignoring Critical Veto

**What goes wrong:** System calculates supermajority > 67% and approves, despite one Watcher casting Critical REJECT.

**Why it happens:** Code checks percentage first, bypasses Critical severity check (CONTEXT.md requires Critical veto power).

**How to avoid:**
```bash
# Check Critical veto BEFORE supermajority
has_critical_reject=$(jq '[.[] | select(.decision == "REJECT")] | any(.issues[]?; .severity == "Critical")' "$votes_file")

if [ "$has_critical_reject" == "true" ]; then
    echo "REJECTED (Critical veto)"
    exit 0
fi

# Now check supermajority
calculate_supermajority "$votes_file"
```

**Warning signs:** Critical issues in approved work, security vulnerabilities passing verification.

### Pitfall 4: Vote-Outcome Disconnect

**What goes wrong:** Watcher weight updates happen immediately after vote, but actual phase outcome unknown yet. Incorrect rewards/penalties applied.

**Why it happens:** Tight coupling between voting and learning. CONTEXT.md says votes update based on "successful phase outcome" or "fix addressing issues."

**How to avoid:**
```bash
# Phase 7: Record vote only (no weight update yet)
record_vote "$watcher" "$decision" "$issues"

# Phase 8 (after phase completes): Update weights based on outcome
if [ "$phase_outcome" == "success" ]; then
    update_watcher_weight "$watcher" "correct_approve"
elif [ "$phase_outcome" == "failed" ] && [ "$vote" == "REJECT" ]; then
    update_watcher_weight "$watcher" "correct_reject"
fi
```

**Warning signs:** Weights oscillating wildly, Watchers with high weight despite poor performance.

### Pitfall 5: Issue Deduplication False Positives

**What goes wrong:** Different issues with similar descriptions merged incorrectly, losing important context.

**Why it happens:** Simple string matching on description field. "Missing auth" and "Missing auth header" treated as duplicates.

**How to avoid:**
```bash
# Use fingerprint: description + category + location
fingerprint=$(echo "${issue_desc}${issue_cat}${issue_loc}" | sha256sum | cut -d' ' -f1)

# Group by fingerprint, not just description
group_by_fingerprint() {
    jq 'group_by(.fingerprint) | map(select(length == 1 or .[0].description == .[1].description))'
}
```

**Warning signs:** Issue reports missing details, Watchers complain their issues disappeared.

## Code Examples

### Spawning Four Watchers in Parallel

```bash
# Source: watcher-ant.md "Spawn Parallel Verifiers" section

spawn_verification_watchers() {
    local work_context="$1"

    # Record spawn event (reuse Phase 6 infrastructure)
    source .aether/utils/spawn-tracker.sh

    # Check resource constraints
    if ! can_spawn; then
        echo "Cannot spawn verification watchers: resource constraints"
        return 1
    fi

    # Spawn 4 Watchers in parallel using Task tool
    Task: Security Watcher <<EOF
Task: Security Watcher

## Inherited Context
Work Context: ${work_context}
Pheromones: $(cat .aether/data/pheromones.json | jq -r '.active_pheromones')
Working Memory: $(cat .aether/data/memory.json | jq -r '.working_memory[0:5]')

## Your Task
Perform security-focused verification on the provided work context.
Check for: OWASP Top 10, injection attacks, auth issues, input validation.
Return JSON: {decision: "APPROVE"|"REJECT", weight: <current_weight>, issues: [...]}

## Outcome Report
After verification, output JSON vote to .aether/verification/votes/security_<timestamp>.json
EOF

    # Repeat for Performance, Quality, Test-Coverage Watchers...

    # Wait for all to complete
    wait

    # Aggregate votes
    .aether/utils/vote-aggregator.sh aggregate
}
```

### Calculating Supermajority with Critical Veto

```bash
# Source: vote-aggregator.sh (new utility)

calculate_supermajority() {
    local votes_dir=".aether/verification/votes"
    local votes_file="$votes_dir/aggregated_votes.json"

    # Combine all 4 vote files
    jq -s '.' "$votes_dir"/*.json > "$votes_file"

    # Check for Critical veto (CONTEXT.md requirement)
    local has_critical
    has_critical=$(jq '
        [.[] | select(.decision == "REJECT")] |
        any(.issues[]?; .severity == "Critical")
    ' "$votes_file")

    if [ "$has_critical" == "true" ]; then
        echo "REJECTED (Critical veto: One or more Critical severity REJECTs)"
        return 1
    fi

    # Calculate weighted supermajority
    local approve_weight total_weight percentage
    approve_weight=$(jq '[.[] | select(.decision == "APPROVE")] | map(.weight) | add // 0' "$votes_file")
    total_weight=$(jq '[.[] | .weight] | add' "$votes_file")

    percentage=$(echo "scale=2; $approve_weight / $total_weight * 100" | bc)

    echo "Weighted approval: $approve_weight / $total_weight = $percentage%"

    if (( $(echo "$percentage >= 67.0" | bc -l) )); then
        echo "APPROVED (Supermajority achieved: ${percentage}% >= 67%)"
        return 0
    else
        echo "REJECTED (Below supermajority: ${percentage}% < 67%)"
        return 1
    fi
}
```

### Issue Deduplication with Prioritization

```bash
# Source: issue-deduper.sh (new utility)

dedupe_and_prioritize() {
    local votes_file="$1"

    # Extract and dedupe issues
    local issues
    issues=$(jq '
        # Extract all issues with metadata
        [.[] | .issues[]? as $issue | {
            description: $issue.description,
            severity: $issue.severity,
            category: $issue.category,
            location: $issue.location,
            watcher: .watcher,
            watcher_weight: .weight
        }] |

        # Create fingerprint for deduping
        map(.fingerprint = (.description + .category + .location | @sha)) |

        # Group by fingerprint
        group_by(.fingerprint) |

        # Dedupe: take first occurrence, tag if multiple
        map({
            description: .[0].description,
            severity: (map(.severity) | max_by(["Critical", "High", "Medium", "Low"])),
            category: .[0].category,
            location: .[0].location,
            watchers: map(.watcher) | unique | join(", "),
            total_weight: map(.watcher_weight) | add,
            tag: (if length > 1 then "Multiple Watchers" else "Single Watcher" end)
        }) |

        # Sort by severity (Critical first) then weight (high first)
        sort_by(.severity, .total_weight) | reverse
    ' "$votes_file")

    echo "$issues"
}
```

### Recording Vote for Meta-Learning

```bash
# Source: vote-aggregator.sh (new utility)

record_vote_outcome() {
    local watcher="$1"           # security, performance, quality, test_coverage
    local decision="$2"          # APPROVE, REJECT
    local issues="$3"            # JSON array of issues
    local verification_id="$4"   # Unique ID for this verification event

    local timestamp=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

    # Record vote in COLONY_STATE.json (reuse Phase 6 meta_learning structure)
    local updated_state
    updated_state=$(jq "
        .verification.votes += [{
            \"id\": \"$verification_id\",
            \"watcher\": \"$watcher\",
            \"decision\": \"$decision\",
            \"issues\": $issues,
            \"timestamp\": \"$timestamp\",
            \"outcome\": \"pending\"  # Will be updated after phase completes
        }] |
        .verification.last_updated = \"$timestamp\"
    " .aether/data/COLONY_STATE.json)

    atomic_write ".aether/data/COLONY_STATE.json" "$updated_state"

    echo "Vote recorded: $watcher → $decision (verification: $verification_id)"
}
```

### Updating Watcher Weights After Phase Outcome

```bash
# Source: weight-calculator.sh (new utility)

update_weights_from_outcome() {
    local phase_outcome="$1"      # success, failed, corrected
    local verification_id="$2"    # Link to votes

    # Get all votes for this verification
    local votes
    votes=$(jq -r ".verification.votes[] | select(.id == \"$verification_id\")" .aether/data/COLONY_STATE.json)

    # For each vote, determine correctness and update weight
    echo "$votes" | jq -c '.' | while read -r vote; do
        local watcher=$(echo "$vote" | jq -r '.watcher')
        local decision=$(echo "$vote" | jq -r '.decision')
        local has_critical=$(echo "$vote" | jq '.issues[]? | select(.severity == "Critical")' | wc -l)

        # Determine vote outcome (CONTEXT.md rules)
        local vote_outcome
        if [ "$phase_outcome" == "success" ] && [ "$decision" == "APPROVE" ]; then
            vote_outcome="correct_approve"
        elif [ "$phase_outcome" == "failed" ] && [ "$decision" == "REJECT" ]; then
            vote_outcome="correct_reject"
        elif [ "$phase_outcome" == "corrected" ] && [ "$decision" == "APPROVE" ]; then
            vote_outcome="incorrect_approve"  # Issues found after approval
        elif [ "$phase_outcome" == "success" ] && [ "$decision" == "REJECT" ]; then
            vote_outcome="incorrect_reject"  # False positive
        fi

        # Update weight
        update_watcher_weight "$watcher" "$vote_outcome"

        # Update vote outcome in state
        jq "
            (.verification.votes[] | select(.id == \"$verification_id\" and .watcher == \"$watcher\")) |=
            (.outcome = \"$vote_outcome\")
        " .aether/data/COLONY_STATE.json | sponge
    done
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Single verifier (human or LLM) | Multi-perspective ensemble (4 Watchers) | 2025 | Improved accuracy, reduced bias (see [DelphiAgent](https://www.sciencedirect.com/science/article/abs/pii/S0306457325001827)) |
| Unweighted voting (simple majority) | Weighted voting with belief calibration | 2025 | Reflects expertise, improves decision quality (see [Optimized Weighted Voting](https://www.emergentmind.com/topics/optimized-weighted-voting-framework)) |
| Manual issue aggregation | Automated deduping + prioritization | 2025 | Faster verification, cleaner reports (see [Steerable Multi-Agent Deep Research](https://arxiv.org/html/2510.17797v1)) |
| Static verifier selection | Dynamic specialist selection based on confidence | 2025 | Better specialist-task matching (see [Dynamic Model Selection](https://arxiv.org/html/2511.00126v1)) |

**Deprecated/outdated:**
- **Sequential verification:** Old pattern of one verifier after another. Replaced by parallel ensemble (4× faster).
- **Simple majority voting:** 50% + 1 threshold too permissive. Supermajority (67%) now standard (see [Supermajority Voting Rules](http://web.mit.edu/rholden/www/papers/Supermajority.pdf)).
- **Manual code review checklists:** Static lists can't adapt. Multi-agent LLM verification now dynamic (see [Multi-Agent AI Testing Guide 2025](https://zyrix.ai/blogs/multi-agent-ai-testing-guide-2025/)).

## Open Questions

1. **Watchers weight initialization:**
   - What we know: CONTEXT.md says "all Watchers start at equal weight (1.0)"
   - What's unclear: Should weights reset between colonies/projects, or persist globally?
   - Recommendation: Start with 1.0, persist in watcher_weights.json (global learning), add reset command for fresh starts

2. **Phase outcome detection:**
   - What we know: Weights update based on "phase outcome" (success/failed/corrected)
   - What's unclear: How does system detect outcome automatically? Queen input?
   - Recommendation: Phase 8's Bayesian system will track outcomes. Phase 7 records votes, Phase 8 updates weights.

3. **Domain expertise bonus scope:**
   - What we know: Watcher weight ×2 for matching domain (Security Watcher ×2 for security issues)
   - What's unclear: Does bonus apply during vote, or only during weight update?
   - Recommendation: Apply bonus during vote calculation (temporary multiplier), store base weight separately.

4. **Vote-Record latency:**
   - What we know: Votes recorded immediately, weights updated after phase outcome
   - What's unclear: How long can phase take before outcome? Timeout?
   - Recommendation: Add vote outcome "pending" state, timeout after 24 hours, default to "uncertain" (no weight change).

## Sources

### Primary (HIGH confidence)
- [CONTEXT.md](.planning/phases/07-colony-verification**---multi-perspective-verification-with-weighted-voting-and-belief-calibration/07-CONTEXT.md) - User decisions locked for implementation
- [COLONY_STATE.json](.aether/data/COLONY_STATE.json) - Existing meta_learning structure
- [spawn-tracker.sh](.aether/utils/spawn-tracker.sh) - Proven parallel spawning pattern
- [spawn-outcome-tracker.sh](.aether/utils/spawn-outcome-tracker.sh) - Confidence scoring implementation
- [watcher-ant.md](.aether/workers/watcher-ant.md) - Existing Watcher capabilities and spawning template

### Secondary (MEDIUM confidence)
- [Voting Classifier: A Comprehensive Guide for 2025](https://www.shadecoder.com/topics/voting-classifier-a-comprehensive-guide-for-2025) - Weight tuning, probability calibration
- [Belief-Calibrated Multi-Agent Consensus Seeking](https://arxiv.org/pdf/2510.06307) - BCCS method for consensus enhancement
- [DelphiAgent: Trustworthy Multi-Agent Verification](https://www.sciencedirect.com/science/article/abs/pii/S0306457325001827) - Multi-LLM verification workflows
- [Rethinking Verification for LLM Code Generation](https://openreview.net/forum?id=Gp2vgxWROE) - Multidimensional verification patterns
- [Supermajority Voting Rules - MIT](http://web.mit.edu/rholden/www/papers/Supermajority.pdf) - Optimal threshold analysis (67% standard)

### Tertiary (LOW confidence - marked for validation)
- [Multi-Agent Orchestration: Running 10+ Claude Instances](https://dev.to/bredmond1019/multi-agent-orchestration-running-10-claude-instances-in-parallel-part-3-29da) - Parallel execution patterns
- [Why Do Multi-Agent LLM Systems Fail?](https://neurips.cc/virtual/2025/poster/121528) - 14 failure modes (verification challenges)
- [BugPrioritizeAI for Multimodal Test Case Prioritisation](https://www.nature.com/articles/s41598-025-31851-z) - Priority scoring for issues
- [Agentic Workflows with Claude](https://medium.com/@reliabledataengineering/agentic-workflows-with-claude-architecture-patterns-design-principles-production-patterns-72bbe4f7e85a) - Sub-agent orchestration

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - All tools (jq, bc, bash) proven in Phase 6, Task tool pattern verified
- Architecture: HIGH - CONTEXT.md decisions locked, parallel spawning pattern from Phase 6
- Pitfalls: HIGH - Based on Phase 6 safeguard testing (25 tests passed), common bash issues documented
- Weight calculation: MEDIUM - Formula from CONTEXT.md, but outcome detection needs Phase 8 integration
- Issue deduping: MEDIUM - jq pattern standard, but fingerprint heuristic needs validation

**Research date:** 2026-02-01
**Valid until:** 2026-03-01 (30 days - stable domain, but LLM multi-agent research moving fast)

**Key assumptions:**
- Phase 6 Task tool spawning works reliably (verified: 8/8 must-haves)
- Queen provides phase outcome signal for weight updates (or Phase 8 auto-detects)
- 4 Watchers sufficient for Phase 7 scope (CONTEXT.md locked decision)
- watcher_weights.json can coexist with specialist_confidence in meta_learning

**Validation needs:**
- Test parallel Watcher spawning with Task tool (confirm 4× concurrent execution)
- Verify supermajority calculation with edge cases (0, 1, 2, 3, 4 APPROVE scenarios)
- Validate issue deduping fingerprint heuristic (test false positive rate)
- Confirm Critical veto logic (REJECT if ANY Critical severity issue)

**Next steps for planner:**
1. Create vote-aggregator.sh with calculate_supermajority(), record_vote_outcome()
2. Create issue-deduper.sh with dedupe_and_prioritize()
3. Create weight-calculator.sh with update_watcher_weight()
4. Create 4 Watcher prompts (security, performance, quality, test-coverage)
5. Update watcher-ant.md with parallel spawning section
6. Add verification section to COLONY_STATE.json schema
7. Create watcher_weights.json with initial weights (all 1.0)
8. Write test suite for voting logic (supermajority, veto, deduping)
