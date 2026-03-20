# Phase 37: Changelog + Visibility - Research

**Researched:** 2026-02-21
**Domain:** Colony Memory Visibility, Changelog Management, Dashboard UX
**Confidence:** HIGH

## Summary

This phase implements continuous changelog updates and visible memory health metrics. The system already has rich data sources (QUEEN.md wisdom, learning observations, midden failures, colony state) but lacks unified visibility. This research identifies how to surface these insights through `/ant:resume` and `/ant:status` commands, plus automated CHANGELOG.md updates after each plan completes.

**Primary recommendation:** Build on existing data structures (QUEEN.md, learning-observations.json, midden.json, COLONY_STATE.json) with new utility functions for metrics aggregation and a standardized date-phase changelog format.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

**CHANGELOG.md Format:**
- Organized by date, then phase: `## 2026-02-21` with `### Phase 36` subsections under each date
- Not flat log, not purely hierarchical by phase

**Changelog Update Timing:**
- After each plan completes (not after each task, not after each phase)
- One entry per plan, aggregating all decisions/files from that plan

**Changelog Content (each entry includes):**
- Files changed (modified, added, deleted)
- Decisions made during the plan (why this approach)
- What worked/didn't (from Phase 36 midden/failures system)
- Requirements addressed by this plan

**Resume Dashboard Layout:**
- PRIMARY: "Where am I now" — current phase progress, last task, what's next
- SECONDARY: Learnings, failures, wisdom shown as counts only (not full items)
- Drill-down available if user wants details
- Recency from research: last 5 decisions, last 10 events

**Status Memory Health Display:**
- **Metrics to show (all four):**
  - Wisdom count (total in QUEEN.md)
  - Pending promotions (observations pending, deferred proposals)
  - Recent failures (from midden/)
  - Last activity (when QUEEN.md updated, last learning captured)
- **Format:** Table format (rows and columns, easy to scan in terminal)

### Claude's Discretion
- Exact table column widths and spacing
- Threshold for "recent" failures (last 5? last 24 hours?)
- Drill-down command for learnings details
- Changelog file location (root or .aether/)

### Deferred Ideas (OUT OF SCOPE)
- TRAILS/decision history — full structured decision archive
- BROOD/phase-index.json — quick-reference phase completion status
- Auto-NEST-update on significant events
- Cross-colony search in ROYAL-CHAMBER/
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| LOG-01 | Workers update CHANGELOG.md during work | Changelog update logic in plan.md completion handler; format defined as date-phase hierarchy |
| VIS-01 | /ant:resume shows learnings, failures, wisdom | Resume dashboard layout defined; data sources identified (QUEEN.md, midden/, learning-observations.json) |
| VIS-02 | /ant:status shows memory health | Four metrics defined; table format specified; data sources mapped to utility functions |
</phase_requirements>

## Standard Stack

### Core (Already Implemented)
| Component | Location | Purpose | Status |
|-----------|----------|---------|--------|
| QUEEN.md | `.aether/docs/QUEEN.md` | Colony wisdom storage | EXISTS - has philosophies, patterns, redirects, stack wisdom, decrees |
| learning-observations.json | `.aether/data/learning-observations.json` | Threshold-tracked observations | EXISTS - content hash deduplication, observation counts |
| learning-deferred.json | `.aether/data/learning-deferred.json` | Deferred promotion proposals | EXISTS - stores deferred wisdom proposals |
| midden.json | `.aether/data/midden/midden.json` | Archived pheromone signals + failures | EXISTS - contains archived signals and failure records |
| COLONY_STATE.json | `.aether/data/COLONY_STATE.json` | Colony state, events, decisions | EXISTS - events array, memory.decisions, memory.phase_learnings |
| aether-utils.sh | `.aether/aether-utils.sh` | Utility functions | EXISTS - queen-promote, learning-observe, learning-check-promotion already implemented |

### New Utilities Required
| Function | Purpose | Location |
|----------|---------|----------|
| `memory-metrics` | Aggregate all memory health metrics into JSON | aether-utils.sh |
| `changelog-append` | Add entry to CHANGELOG.md with proper formatting | aether-utils.sh |
| `resume-dashboard` | Generate resume dashboard data | aether-utils.sh |
| `midden-recent-failures` | Extract recent failures with threshold filtering | aether-utils.sh |

## Architecture Patterns

### Pattern 1: Date-Phase Changelog Hierarchy
**What:** Changelog organized by date first, then phase subsection
**When to use:** All changelog entries per user decision
**Format:**
```markdown
## 2026-02-21

### Phase 36
- **Files:** modified: `file1.md`, `file2.sh`; added: `new.md`
- **Decisions:** Used approach X because Y
- **What Worked:** Pattern Z succeeded
- **Requirements:** MEM-01, MEM-02 addressed
```

### Pattern 2: Counts-First Dashboard
**What:** Show counts as primary metrics, full details on drill-down
**When to use:** Resume dashboard to avoid overwhelming output
**Structure:**
```
📊 Memory Health
   Wisdom: 12 entries | Pending: 3 promotions | Recent Failures: 2

   Run /ant:memory-details for full breakdown
```

### Pattern 3: Table Format for Terminal
**What:** Row/column layout for easy scanning
**When to use:** Status command memory health display
**Format:**
```
┌─────────────────┬────────┬─────────────────────────────┐
│ Metric          │ Count  │ Last Updated                │
├─────────────────┼────────┼─────────────────────────────┤
│ Wisdom Entries  │ 12     │ 2026-02-20 14:30            │
│ Pending Promos  │ 3      │ 2 observations, 1 deferred  │
│ Recent Failures │ 2      │ Last: 2026-02-19 (builder)  │
│ Activity        │ —      │ QUEEN.md: 2026-02-20        │
└─────────────────┴────────┴─────────────────────────────┘
```

### Pattern 4: Recency-Weighted Pruning
**What:** Show only recent items by default, full history on request
**When to use:** Resume dashboard secondary section
**Thresholds:**
- Last 5 decisions from COLONY_STATE.json memory.decisions
- Last 10 events from COLONY_STATE.json events array
- Last 5 failures from midden (configurable)

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Wisdom parsing | Custom markdown parser | Read QUEEN.md frontmatter + grep section headers | Already standardized format |
| Observation counting | Custom counter | learning-observations.json existing structure | Content hash deduplication already implemented |
| Failure tracking | New failure database | midden.json existing archive | Phase 36 already logs failures to midden |
| Changelog formatting | Custom templating | Standard markdown with consistent hierarchy | User specified format |
| Date handling | Custom date math | ISO-8601 strings with bash date | Cross-platform compatible |

## Common Pitfalls

### Pitfall 1: Over-Displaying Information
**What goes wrong:** Showing full learning content, failure details, and wisdom entries by default overwhelms the user
**Why it happens:** All data is available so we show it all
**How to avoid:** Follow counts-first pattern; use drill-down commands for details
**Warning signs:** Resume output exceeds 50 lines

### Pitfall 2: Stale Data in Dashboard
**What goes wrong:** Showing "recent" failures that are weeks old
**Why it happens:** No threshold filtering on midden queries
**How to avoid:** Implement recency threshold (suggest: last 5 failures OR last 7 days)
**Warning signs:** User sees old failures as "recent"

### Pitfall 3: Changelog Duplication
**What goes wrong:** Multiple entries for same plan if plan command retries
**Why it happens:** Changelog update on every plan command run
**How to avoid:** Only update changelog when plan actually completes (new phases generated)
**Warning signs:** Duplicate date-phase entries in CHANGELOG.md

### Pitfall 4: Breaking Existing Changelog Format
**What goes wrong:** New date-phase format conflicts with existing Keep a Changelog format
**Why it happens:** CHANGELOG.md already exists with version-based format
**How to avoid:** Append new format after existing content; maintain both formats
**Warning signs:** Existing changelog parsers break

## Code Examples

### Reading QUEEN.md Wisdom Count
```bash
# Source: .aether/docs/QUEEN.md structure
# Count entries by section
philosophies=$(grep -c "^- \*\*" .aether/docs/QUEEN.md | head -20)
patterns=$(grep -c "^- \*\*" .aether/docs/QUEEN.md | head -20)
# Or parse from METADATA JSON at bottom
wisdom_count=$(jq '.stats.total_philosophies + .stats.total_patterns + .stats.total_redirects + .stats.total_stack_entries + .stats.total_decrees' .aether/docs/QUEEN.md 2>/dev/null || echo "0")
```

### Counting Pending Promotions
```bash
# Source: learning-observations.json + learning-deferred.json
# Observations pending (meeting threshold but not yet promoted)
pending_observations=$(jq '[.observations[] | select(.observation_count >= 3)] | length' .aether/data/learning-observations.json 2>/dev/null || echo "0")
# Deferred proposals
deferred_count=$(jq '.deferred | length' .aether/data/learning-deferred.json 2>/dev/null || echo "0")
```

### Extracting Recent Failures from Midden
```bash
# Source: .aether/data/midden/midden.json
# Get last 5 failure entries
recent_failures=$(jq '[.signals[] | select(.type == "failure") | .created_at] | sort | reverse | .[0:5]' .aether/data/midden/midden.json 2>/dev/null || echo "[]")
```

### Changelog Append Pattern
```bash
# Source: User decision - date-phase hierarchy
changelog_entry() {
    local date_str="$1"
    local phase="$2"
    local files="$3"
    local decisions="$4"
    local worked="$5"
    local requirements="$6"

    # Check if date section exists
    if ! grep -q "^## ${date_str}$" CHANGELOG.md; then
        # Add new date section
        echo -e "\n## ${date_str}\n" >> CHANGELOG.md
    fi

    # Append phase subsection
    cat >> CHANGELOG.md << EOF
### ${phase}
- **Files:** ${files}
- **Decisions:** ${decisions}
- **What Worked:** ${worked}
- **Requirements:** ${requirements}

EOF
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Phase-only changelog entries | Date-phase hierarchy | Phase 37 (planned) | Better chronological context |
| Full detail display | Counts-first with drill-down | Phase 37 (planned) | Less overwhelming output |
| Manual changelog updates | Automatic after plan completion | Phase 37 (planned) | Consistent documentation |
| No memory health visibility | Table format metrics | Phase 37 (planned) | Quick health assessment |

## Open Questions

1. **Changelog Location Decision**
   - What we know: User deferred to Claude's discretion
   - What's unclear: Root CHANGELOG.md vs .aether/CHANGELOG.md
   - Recommendation: Use root CHANGELOG.md (standard convention), but check if existing file uses different format

2. **Recent Failures Threshold**
   - What we know: User deferred threshold to Claude's discretion
   - What's unclear: Last 5 failures vs last 24 hours vs last 7 days
   - Recommendation: Use "last 5 failures" - simple, predictable, matches decision recency pattern

3. **Drill-Down Command Name**
   - What we know: User wants drill-down for learnings details
   - What's unclear: Command name (/ant:memory-details? /ant:learnings?)
   - Recommendation: Use /ant:memory-details to avoid confusion with existing learning commands

4. **Existing Changelog Compatibility**
   - What we know: CHANGELOG.md exists with Keep a Changelog format
   - What's unclear: Whether to migrate, append, or create separate file
   - Recommendation: Append new format after existing content with clear separator

## Sources

### Primary (HIGH confidence)
- `.aether/docs/QUEEN.md` - Wisdom structure and metadata format
- `.aether/data/learning-observations.json` - Observation tracking schema
- `.aether/data/learning-deferred.json` - Deferred proposal schema
- `.aether/data/midden/midden.json` - Failure archive structure
- `.aether/aether-utils.sh` - Existing utility functions (queen-promote, learning-observe, learning-check-promotion)
- `.claude/commands/ant/continue.md` - Current changelog update logic (Step 2.3)
- `.claude/commands/ant/build.md` - Failure logging to midden (MEM-02)
- `.claude/commands/ant/resume.md` - Existing resume dashboard structure
- `.claude/commands/ant/status.md` - Existing status display patterns

### Secondary (MEDIUM confidence)
- `CHANGELOG.md` - Existing format analysis
- `tests/e2e/test-vis.sh` - VIS-01, VIS-02 requirements context

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - All components exist and are functional
- Architecture: HIGH - Patterns derived from existing codebase conventions
- Pitfalls: MEDIUM - Based on UX best practices and colony system complexity

**Research date:** 2026-02-21
**Valid until:** 2026-03-21 (30 days - stable domain)
