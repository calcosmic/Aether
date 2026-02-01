---
name: ant:status
description: Show detailed AETHER system status - agent hierarchy, memory stats, error prevention
---

<objective>
Display comprehensive status of the AETHER autonomous agent system including active agent hierarchy, memory utilization, error prevention metrics, and recent execution history.
</objective>

<reference>
# `/ant:status` - Show AETHER System Status

## What It Shows

Displays a comprehensive overview of the entire AETHER system state:

```
/ant:status
```

## Output Sections

### 1. Agent Hierarchy

Shows the current agent tree structure:

```
🐜 AETHER Agent Hierarchy
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Goal-Agent (ACTIVE)
  ├── Orchestrator-Agent (IDLE)
  │   ├── Database-Specialist (ACTIVE)
  │   └── API-Specialist (COMPLETED)
  ├── Frontend-Specialist (ACTIVE)
  └── QA-Specialist (IDLE)

Total Agents: 6
Active: 3 | Idle: 2 | Completed: 1
Max Depth: 3 levels
```

### 2. Memory System Statistics

Shows memory utilization across all three layers:

```
🧠 Memory System Status
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

WORKING MEMORY (Current Session)
  Messages: 47 / 200
  Tokens: ~85,400 / 200,000 (42.7%)
  Items: 23 (contexts, facts, patterns)

SHORT-TERM MEMORY (Recent Sessions)
  Sessions: 3 / 10 compressed
  Total Items: 127
  Compression Ratio: 14.2x
  Oldest: 2 hours ago

LONG-TERM MEMORY (Persistent)
  Patterns: 34
  Best Practices: 18
  Anti-Patterns: 7
  Associative Links: 89

ASSOCIATIVE NETWORK
  Strong Links (>0.8): 12
  Medium Links (0.5-0.8): 45
  Weak Links (<0.5): 32
```

### 3. Error Prevention Metrics

Shows error learning and prevention statistics:

```
🛡️ Error Prevention System
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

ERROR LEDGER
  Total Errors Logged: 23
  By Severity:
    CRITICAL: 2
    HIGH: 5
    MEDIUM: 11
    LOW: 5

FLAGGED ISSUES (Auto-Flagged After 3 Occurrences)
  Active Flags: 3
    → [HIGH] database_connection_pool_exhaustion (7 occurrences)
    → [MEDIUM] async_timeout_not_handled (4 occurrences)
    → [LOW] missing_error_context (3 occurrences)

VALIDATION STATS
  Actions Validated: 342
  Actions Blocked: 18 (5.3% block rate)
  False Positives: 2 (1.1%)
  Critical Prevention: 3 would-be critical errors

GUARDRAILS
  Active Constraints: 12
  Validations Passed: 324
  Validations Failed: 18
```

### 4. Recent Goals Executed

Shows recent goal execution history:

```
🎯 Recent Goal Executions
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

1. [✅ SUCCESS] "Build authentication system"
   Time: 5 minutes ago
   Duration: 47 seconds
   Tasks: 5/5 completed
   Agents Spawned: 6
   Messages: 23

2. [✅ SUCCESS] "Create REST API with JWT"
   Time: 2 hours ago
   Duration: 2 minutes 14 seconds
   Tasks: 8/8 completed
   Agents Spawned: 9
   Messages: 41

3. [⚠️ PARTIAL] "Add real-time notifications"
   Time: 5 hours ago
   Duration: 1 minute 33 seconds
   Tasks: 4/6 completed
   Agents Spawned: 5
   Error: WebSocket connection timeout

Total Goals: 23
Success Rate: 91.3% (21/23)
Average Duration: 58 seconds
```

### 5. Spawn Event History

Shows recent agent spawning events:

```
🔄 Recent Spawn Events
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

1. Database-Specialist spawned by Orchestrator
   Reason: "Detected gap: PostgreSQL query optimization"
   Capabilities: ['sql', 'postgres', 'query_optimization']
   Time: 5 minutes ago
   Status: ACTIVE

2. API-Specialist spawned by Orchestrator
   Reason: "Detected gap: REST endpoint design"
   Capabilities: ['api_design', 'rest', 'openapi']
   Time: 5 minutes ago
   Status: COMPLETED

3. Security-Specialist spawned by API-Specialist
   Reason: "Detected gap: JWT token validation"
   Capabilities: ['security', 'jwt', 'authentication']
   Time: 5 minutes ago
   Status: COMPLETED

Total Spawns: 47
Spawn Success Rate: 97.9%
Average Agent Lifetime: 2 minutes 14 seconds
```

### 6. System Health

Overall system health indicators:

```
🏥 System Health
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Overall Status: 🟢 HEALTHY

Performance Indicators:
  Agent Spawn Latency: 180ms (avg)
  Memory Access Time: 45ms (avg)
  Error Prevention Overhead: 23ms (avg)

Resource Utilization:
  Working Memory: 42.7% (optimal)
  Agent Count: 6 (within limits)
  Concurrent Operations: 3

Recent Issues:
  [WARN] Memory compression recommended (working > 40%)
  [INFO] 3 agents approaching idle timeout
```

## Usage Examples

### Basic Status Check

```bash
/ant:status
```

Shows all sections with default detail level.

### Filter Specific Section

```bash
/ant:status --section agents
```

Shows only agent hierarchy.

```bash
/ant:status --section memory
```

Shows only memory statistics.

```bash/ant:status --section errors
```
Shows only error prevention metrics.

### Verbose Output

```bash/ant:status --verbose
```

Shows detailed information for each agent including:
- Full capability list
- Current task queue
- Memory access patterns
- Spawn lineage

### JSON Output

```bash/ant:status --json
```

Outputs machine-readable JSON for programmatic access.

## Real-Time Monitoring

For continuous monitoring, use watch mode:

```bash
watch -n 5 "claude /ant:status"
```

Updates every 5 seconds.

## Understanding Agent States

| State | Meaning |
|-------|---------|
| **ACTIVE** | Agent is currently executing a task |
| **IDLE** | Agent is waiting for tasks or coordination |
| **COMPLETED** | Agent finished all assigned tasks |
| **BLOCKED** | Agent waiting on dependency or resource |
| **FAILED** | Agent encountered unrecoverable error |
| **TERMINATED** | Agent was terminated (completed or cancelled) |

## Memory Thresholds

| Memory Type | Warning | Critical | Action |
|-------------|---------|----------|--------|
| Working | 40% | 80% | Compress to short-term |
| Short-Term | 8 sessions | 10 sessions | Compress to long-term |
| Long-Term | N/A | N/A | Never expires (manual) |

## Error Prevention Metrics

**Block Rate**: Percentage of actions blocked by guardrails
- **< 5%**: Optimal - guardrails working, not too restrictive
- **5-10%**: Normal - acceptable protection level
- **> 10%**: Review - may be over-blocking

**False Positive Rate**: Blocked actions that would have succeeded
- **< 2%**: Excellent
- **2-5%**: Good
- **> 5%**: Needs tuning

**Critical Prevention**: Would-be critical errors that were prevented
- Higher is better - means the system is protecting you

## Related Commands

```
/ant                    # Show system overview
/ant:build <goal>     # Execute a new goal
/ant:memory            # View detailed memory contents
/ant:errors            # View error ledger and flagged issues
```

## Tips for Interpreting Status

### Healthy System Indicators

- ✅ Working memory 40-60%
- ✅ Block rate 3-7%
- ✅ Success rate > 85%
- ✅ Agent spawn latency < 500ms
- ✅ No active CRITICAL errors

### Warning Signs

- ⚠️ Working memory > 70% (needs compression)
- ⚠️ Block rate > 10% (may be over-blocking)
- ⚠️ Success rate < 75% (system struggling)
- ⚠️ Multiple FAILED agents (needs investigation)

### Critical Issues

- 🚨 Active CRITICAL flagged issues
- 🚨 Memory > 90% (immediate compression needed)
- 🚨 High failure rate cascade (agents spawning failing agents)

## System Optimization

Based on status output, you may want to:

1. **Compress memory** if working memory > 60%
2. **Review flagged issues** if many HIGH or CRITICAL
3. **Tune guardrails** if false positive rate > 5%
4. **Adjust spawn thresholds** if agent tree too deep
5. **Clear old sessions** if short-term memory near capacity
</reference>
