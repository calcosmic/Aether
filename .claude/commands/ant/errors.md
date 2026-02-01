---
name: ant:errors
description: View AETHER error ledger - logged errors, flagged issues, and prevention metrics
---

<objective>
Display the error prevention system state including all logged errors with root cause analysis, auto-flagged recurring issues, constraint validation status, and prevention metrics.
</objective>

<reference>
# `/ant:errors` - View Error Ledger and Flagged Issues

## What It Shows

Displays comprehensive error tracking and prevention information:

```
/ant:errors
```

## Output Sections

### 1. Error Ledger

Shows all logged errors with full details:

```
📋 ERROR LEDGER
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Total Errors Logged: 23
By Severity:
  CRITICAL: 2 | HIGH: 5 | MEDIUM: 11 | LOW: 5

RECENT ERRORS (Last 10)

[ERROR-23] MEDIUM - WebSocket Connection Timeout
  When: 5 hours ago
  Agent: Notification-Specialist (spawned by Goal-Agent)
  Symptom: "WebSocket connection timed out after 30 seconds"
  Root Cause: "No heartbeat mechanism implemented"
  Attempted Fix: "Increased timeout to 60 seconds"
  Fix Worked: No (error recurred)
  Prevention: "Implement ping/pong heartbeat every 15 seconds"
  Occurrences: 3

[ERROR-22] HIGH - Database Connection Pool Exhaustion
  When: 5 hours ago
  Agent: Database-Specialist (spawned by Orchestrator)
  Symptom: "Connection pool timeout - no available connections"
  Root Cause: "Agents not releasing connections after queries"
  Attempted Fix: "Added connection release in finally block"
  Fix Worked: Yes
  Prevention: "Use context manager for all connections"
  Occurrences: 7 ⚠️ FLAGGED

[ERROR-21] LOW - Missing Error Context in Logs
  When: 5 hours ago
  Agent: QA-Specialist (spawned by Goal-Agent)
  Symptom: "Error log missing stack trace and state"
  Root Cause: "Logging exception without context"
  Attempted Fix: "Added logger.exception() with full context"
  Fix Worked: Yes
  Prevention: "Always use structured error logging"
  Occurrences: 3 ⚠️ FLAGGED

... (20 more errors)
```

### 2. Flagged Issues

Shows auto-flagged recurring errors (3+ occurrences):

```
🚩 FLAGGED ISSUES (Auto-Flagged After 3 Occurrences)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Active Flags: 3

[FLAG-1] HIGH - database_connection_pool_exhaustion
  First Occurrence: 5 days ago
  Total Occurrences: 7
  Last Occurrence: 5 hours ago
  Trend: Increasing (was 1/week, now 2/day)

  Root Cause: "Agents not properly releasing database connections"

  Prevention Attempts:
    1. Added connection release in finally block (ERROR-15) - Failed
    2. Implemented context manager pattern (ERROR-18) - Partial
    3. Added connection timeout (ERROR-22) - Successful ✅

  Current Status: ⚠️ MONITORING
  Last Fix: "Use context manager for all connections" (worked for 2 occurrences)

  Recommended Actions:
    → Audit all database code for context manager usage
    → Add connection pool metrics
    → Implement circuit breaker for pool exhaustion

[FLAG-2] MEDIUM - async_timeout_not_handled
  First Occurrence: 2 days ago
  Total Occurrences: 4
  Last Occurrence: 5 hours ago
  Trend: Stable

  Root Cause: "Async functions not handling asyncio.TimeoutError"

  Prevention Attempts:
    1. Added timeout wrapper (ERROR-19) - Successful ✅

  Current Status: ✅ RESOLVED
  No new occurrences in last 5 hours

[FLAG-3] LOW - missing_error_context
  First Occurrence: 3 days ago
  Total Occurrences: 3
  Last Occurrence: 5 hours ago
  Trend: Decreasing

  Root Cause: "Inconsistent error logging across agents"

  Prevention Attempts:
    1. Created structured logging helper (ERROR-21) - Partial

  Current Status: ⚠️ MONITORING
  Recommended Actions:
    → Enforce structured logging in all agents
    → Add lint rule for error logging
```

### 3. Constraint Validation Status

Shows active constraints and validation results:

```
🔒 CONSTRAINT VALIDATION STATUS
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Active Constraints: 12

[CONSTRAINT-1] validate_sql_before_execution
  Type: PRE_EXECUTION
  Purpose: Prevent SQL injection via validation
  Status: ✅ ACTIVE

  Validations: 127
  Passed: 124 (97.6%)
  Blocked: 3 (2.4%)
    → [BLOCK-1] String concatenation in SQL query (ERROR-20)
    → [BLOCK-2] Unescaped user input in WHERE clause (ERROR-17)
    → [BLOCK-3] Non-parameterized query with user data (ERROR-22)

  False Positives: 0
  Critical Prevention: 2 potential SQL injections blocked

[CONSTRAINT-2] max_agent_spawn_depth
  Type: PRE_EXECUTION
  Purpose: Prevent infinite spawn loops
  Status: ✅ ACTIVE

  Validations: 89
  Passed: 87 (97.8%)
  Blocked: 2 (2.2%)
    → [BLOCK-4] Spawn depth would be 6 (max is 5) (ERROR-11)
    → [BLOCK-5] Circular spawn detected (ERROR-12)

  False Positives: 0
  Critical Prevention: 1 potential infinite loop blocked

[CONSTRAINT-3] require_error_context
  Type: POST_EXECUTION
  Purpose: Ensure all errors include context
  Status: ⚠️ PARTIAL

  Validations: 45
  Passed: 42 (93.3%)
  Failed: 3 (6.7%)
    → [FAIL-1] Missing stack trace in log (ERROR-21)
    → [FAIL-2] No state information in error (ERROR-23)
    → [FAIL-3] Missing input data in error message (ERROR-24)

  False Positives: 1 (ERROR-25 had context but in different format)

... (9 more constraints)
```

### 4. Prevention Metrics

Shows overall error prevention effectiveness:

```
🛡️ ERROR PREVENTION METRICS
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

OVERALL STATS
  Total Actions: 342
  Validations Performed: 342 (100% coverage)
  Actions Blocked: 18 (5.3%)
  Errors Logged: 23 (6.7% error rate)

BLOCK RATE BREAKDOWN
  By Severity:
    CRITICAL blocked: 3 (would be critical)
    HIGH blocked: 7 (would be severe)
    MEDIUM blocked: 5 (would be problematic)
    LOW blocked: 3 (would be minor)

  Block Rate Trend:
    Last hour: 8.1% (increasing)
    Last 6 hours: 5.3% (stable)
    Last 24 hours: 4.7% (decreasing)

FALSE POSITIVE RATE
  Total Blocks: 18
  False Positives: 2 (11.1%)
  True Positives: 16 (88.9%)

  False Positives:
    → [BLOCK-7] SQL block (was parameterized, constraint didn't detect)
    → [BLOCK-12] Spawn depth block (was safe circular reference)

CRITICAL PREVENTIONS
  Would-be critical errors prevented: 5
    1. SQL injection attack (CONSTRAINT-1, BLOCK-1)
    2. Infinite spawn loop (CONSTRAINT-2, BLOCK-4)
    3. Database connection leak (CONSTRAINT-4, BLOCK-8)
    4. Memory overflow (CONSTRAINT-5, BLOCK-9)
    5. Credential exposure (CONSTRAINT-6, BLOCK-11)

ERROR RATE BY AGENT TYPE
  Database-Specialist: 12.3% (9 errors, 73 actions)
  API-Specialist: 8.1% (6 errors, 74 actions)
  Frontend-Specialist: 5.4% (4 errors, 74 actions)
  QA-Specialist: 3.9% (3 errors, 77 actions)
  Orchestrator: 2.7% (1 error, 37 actions)

LEARNING RATE
  New errors per 100 actions:
    Last hour: 9.2 (learning phase)
    Last 6 hours: 6.7 (improving)
    Last 24 hours: 4.7 (stable)
    All time: 6.7%

  Trend: ✅ IMPROVING (error rate decreasing)
```

### 5. Prevention Recommendations

Shows actionable recommendations based on error patterns:

```
💡 PREVENTION RECOMMENDATIONS
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

HIGH PRIORITY (Based on error frequency and severity)

1. Implement database connection management
   Impact: Prevents 7 occurrences (30% of errors)
   Effort: Medium
   Action: Audit all database code, enforce context managers

2. Add comprehensive error logging
   Impact: Prevents 4 occurrences, improves debugging
   Effort: Low
   Action: Create logging helper, add to agent template

3. Implement async timeout handling
   Impact: Prevents 4 occurrences of timeout errors
   Effort: Low
   Action: Add timeout wrapper to all async calls

MEDIUM PRIORITY

4. Add heartbeat to WebSocket connections
   Impact: Prevents 3 occurrences of timeout errors
   Effort: Medium
   Action: Implement ping/pong every 15 seconds

5. Enforce SQL parameterization
   Impact: Prevents 3 potential SQL injections
   Effort: Low
   Action: Strengthen constraint validation

LOW PRIORITY (Nice to have)

6. Add spawn depth visualization
   Impact: Helps prevent spawn loops
   Effort: Low
   Action: Show depth in /ant:status

7. Implement error rate alerting
   Impact: Early warning of problems
   Effort: Medium
   Action: Alert when error rate > 10%
```

## Usage Examples

### View All Error Information

```bash
/ant:errors
```

Shows all five sections with full details.

### View Specific Section

```bash/ant:errors --section ledger
```

Shows only error ledger.

```bash/ant:errors --section flagged
```

Shows only flagged issues.

```bash/ant:errors --section constraints
```

Shows only constraint validation status.

```bash/ant:errors --section metrics
```

Shows only prevention metrics.

```bash/ant:errors --section recommendations
```

Shows only recommendations.

### View Specific Error

```bash/ant:errors --error ERROR-22
```

Shows full details for specific error ID.

### View Specific Flag

```bash/ant:errors --flag FLAG-1
```

Shows full details for specific flagged issue.

### Filter by Severity

```bash/ant:errors --severity CRITICAL
```

Shows only CRITICAL errors.

```bash/ant:errors --severity HIGH,MEDIUM
```

Shows HIGH and MEDIUM errors.

### Filter by Time

```bash/ant:errors --since "1 hour ago"
```

Shows errors from last hour.

```bash/ant:errors --since yesterday
```

Shows errors since yesterday.

### Export Errors

```bash/ant:errors --export errors_report.json
```

Exports all errors to JSON file.

```bash/ant:errors --export errors.csv --format csv
```

Exports to CSV format.

## Error Resolution Workflow

When an error occurs:

1. **Log Error** - Symptom, context, agent, time
2. **Analyze Root Cause** - Why did it happen?
3. **Attempt Fix** - Try to fix immediately
4. **Log Result** - Did the fix work?
5. **Create Prevention** - How to prevent recurrence?
6. **Monitor** - Watch for similar errors

After 3 occurrences of similar errors:
- **Auto-Flag** - Created as flagged issue
- **High Priority** - Shows in /ant:errors
- **Track Trend** - Monitor frequency changes

When fix is validated (no occurrences for 24 hours):
- **Mark Resolved** - Flag marked as resolved
- **Create Pattern** - Best practice added to memory
- **Update Constraints** - Strengthen validation

## Understanding Error Severity

| Severity | Definition | Action Required |
|----------|------------|-----------------|
| **CRITICAL** | System-breaking, data loss, security breach | Immediate fix required |
| **HIGH** | Major feature broken, severe degradation | Fix within 1 hour |
| **MEDIUM** | Minor feature broken, workarounds available | Fix within 24 hours |
| **LOW** | Cosmetic issues, minor annoyances | Fix when convenient |

## Constraint Types

| Type | When It Runs | Purpose |
|------|--------------|---------|
| **PRE_EXECUTION** | Before action runs | Prevent errors before they happen |
| **POST_EXECUTION** | After action runs | Catch and log errors |
| **PERIODIC** | On schedule | Monitor for issues |

## Block Rate Interpretation

| Block Rate | Meaning | Action |
|------------|---------|--------|
| **< 3%** | Too permissive | Constraints may be too weak |
| **3-7%** | Optimal | Good protection, not too restrictive |
| **7-15%** | Acceptable | May need tuning |
| **> 15%** | Too restrictive | Constraints blocking valid work |

## Related Commands

```
/ant                    # Show system overview
/ant:status            # Show system status with error stats
/ant:memory            # View memory system
/ant:build <goal>     # Execute new goal
```

## Tips for Error Prevention

### Reduce Error Rate

1. **Review flagged issues** - Address recurring problems
2. **Strengthen constraints** - Add validation for common errors
3. **Learn from patterns** - Check long-term memory for best practices
4. **Test edge cases** - QA agents should find errors before production

### Improve Prevention Accuracy

1. **Reduce false positives** - Tune constraints to avoid blocking valid work
2. **Add context to blocks** - Explain why action was blocked
3. **Provide alternatives** - Suggest safe approaches when blocking

### Monitor Effectiveness

1. **Track error rate** - Should decrease over time
2. **Monitor block rate** - Should stay 3-7%
3. **Check flag trends** - New flags should decrease
4. **Validate critical preventions** - Ensure blocking real threats

## Error Ledger Format

Each error in the ledger includes:

```yaml
id: ERROR-23
severity: MEDIUM
timestamp: "2026-02-01T10:30:00Z"
agent: Notification-Specialist
agent_lineage: [Goal-Agent, Orchestrator, Notification-Specialist]

symptom: "WebSocket connection timed out after 30 seconds"
context:
  task: "Implement real-time notifications"
  state: "Connecting to WebSocket server"
  inputs: { url: "ws://localhost:8080", timeout: 30 }

root_cause: "No heartbeat mechanism implemented"
analysis: "WebSocket server expects ping every 15 seconds"

attempted_fix: "Increased timeout to 60 seconds"
fix_worked: false

prevention: "Implement ping/pong heartbeat every 15 seconds"
prevention_status: "pending"

occurrences: 3
first_occurrence: "2026-01-29T15:20:00Z"
last_occurrence: "2026-02-01T10:30:00Z"
trend: "stable"
```

This comprehensive format enables:
- Root cause analysis
- Pattern detection
- Prevention planning
- Trend monitoring
- Agent-specific debugging
</reference>
