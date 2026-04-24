---
name: session-reporting
description: Use when a session needs a concise report of work completed, outcomes, and follow-up state
type: colony
domains: [reporting, session-management, analytics]
agent_roles: [chronicler, keeper]
workflow_triggers: [pause, seal]
task_keywords: [report, summary, session, what did we do, velocity]
priority: normal
version: "1.0"
---

# Session Reporting

## Purpose

Generate a session report with work summary, outcomes, and resource usage estimates. Produces a SESSION_REPORT.md that serves as both a historical record and a handoff artifact for the colony's tracking systems.

## When to Use

- A session is ending and you want to capture what happened
- User asks "what did we do today?", "summarize this session", or "session report"
- Before `session-handoff pause` to include a report with the handoff
- Periodic tracking of work velocity and patterns
- End-of-day or end-of-week review

## Instructions

### Generate Report

1. **Gather session data**:
   - Git log since session start: `git log --since="{session_start}" --oneline`
   - Files changed: `git diff --stat HEAD~{N}..HEAD`
   - Phase manifest changes: diff `.aether/data/phase-manifest.json` before/after
   - Todos created/resolved: diff `.aether/data/todos.jsonl` before/after
   - Seeds planted/germinated: diff `.aether/data/seeds.jsonl` before/after
   - Threads opened/appended: check `.aether/data/threads/` for recent changes

2. **Compute metrics**:
   - Duration: time from session start to now (estimate if unknown)
   - Commits: count and list
   - Lines changed: additions and deletions
   - Files touched: count and list
   - Phases progressed: any phase status changes
   - Todos: created vs resolved counts
   - Token usage estimate: based on conversation length and tool calls (rough proxy)

3. **Write SESSION_REPORT.md** to `.aether/data/session-reports/{date}-{N}.md`:
   ```markdown
   # Session Report

   **Date**: {YYYY-MM-DD}
   **Duration**: {estimated hours/minutes}
   **Phase**: {active phase number and name}

   ## Summary
   {2-3 sentence plain English summary of what was accomplished}

   ## Completed
   - {specific thing 1}
   - {specific thing 2}

   ## In Progress
   - {thing that was started but not finished}

   ## Blocked
   - {blocker if any}

   ## Metrics
   | Metric | Value |
   |--------|-------|
   | Commits | {N} |
   | Files changed | {N} |
   | Lines added | {N} |
   | Lines removed | {N} |
   | Todos created | {N} |
   | Todos resolved | {N} |
   | Seeds planted | {N} |

   ## Next Steps
   1. {what to pick up in the next session}
   2. {continuing item}

   ## Notes
   {any additional observations, decisions, or context}
   ```

4. Update `.aether/data/session-reports/index.json` with a summary entry for the session

### List Past Reports

1. Read `.aether/data/session-reports/index.json`
2. Display recent sessions in reverse chronological order:
   - Date, duration, phase, commits, summary line
3. Support `--last N` to show the last N sessions
4. Support `--since YYYY-MM-DD` for date range filtering

### Velocity Tracking

1. Aggregate data from recent session reports
2. Compute rolling averages:
   - Commits per session
   - Phases completed per week
   - Todos resolved per session
3. Present as a simple trend indicator: ` improving`, `-> steady`, ` declining`

## Key Patterns

- **Honest reporting**: Report blockers and incomplete work alongside successes
- **No inflation**: Don't count a reverted commit as progress
- **Linkage**: Reference specific phase numbers, todo IDs, and thread IDs in the report
- **Brevity**: The summary should be scannable in 30 seconds
- **Cumulative index**: The index.json enables cross-session analysis without reading every report file

## Output Format

```
Session Report -- 2026-04-22
  Duration: ~2.5 hours | Phase 3: Dashboard UI
  Commits: 4 | Files: 8 | +642/-128 lines
  Todos: 3 created, 1 resolved | Seeds: 1 planted

  Summary: Completed chart component and data binding layer.
  Started real-time update polling -- needs retry logic next session.

  Written to .aether/data/session-reports/2026-04-22-001.md
```

## Examples

```
# Generate report for current session
> session-reporter generate

# Generate with custom summary
> session-reporter generate --summary "Wrapped up API layer, started frontend integration"

# List recent sessions
> session-reporter list --last 5

# Show velocity trends
> session-reporter velocity
```
