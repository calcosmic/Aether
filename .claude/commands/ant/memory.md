---
name: ant:memory
description: View Queen Ant Colony memory - pheromone trails, learned preferences, patterns
---

<objective>
Display the colony's memory system including pheromone history, learned preferences from Queen's signals, detected patterns, and associative links.
</objective>

<reference>
# `/ant:memory` - View Colony Memory

## What It Shows

Displays the colony's learned patterns and preferences from pheromone signals:

```
/ant:memory
```

## Memory Sections

### 1. Learned Preferences

Shows what the colony has learned from Queen's pheromone patterns:

```
🧠 LEARNED PREFERENCES
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

FOCUS TOPICS (What Queen prioritizes)
  WebSocket security (3 occurrences)
  message reliability (2 occurrences)
  authentication (1 occurrence)
  test coverage (1 occurrence)

AVOID PATTERNS (What Queen redirects away from)
  string concatenation for SQL (2 occurrences) → ⚠️ One more becomes constraint
  callback patterns (1 occurrence)
  MongoDB for this project (1 occurrence)

FEEDBACK CATEGORIES
  Quality: 12 positive, 3 negative
  Speed: 5 "too slow", 8 "good pace"
  Direction: 2 "wrong approach" corrections
```

### 2. Pheromone History

Shows recent pheromone signals and their impact:

```
🌸 PHEROMONE HISTORY (Last 10)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

[1] INIT - "Build a real-time chat application"
  Time: 2 hours ago
  Strength: 1.0 (persistent)
  Impact: Colony mobilized, 5 phases created

[2] FOCUS - "WebSocket security"
  Time: 47 minutes ago
  Strength: 0.7 (decayed to 0.3)
  Impact: Executor prioritized security, Verifier increased scrutiny

[3] FOCUS - "message reliability"
  Time: 23 minutes ago
  Strength: 0.5 (decayed to 0.2)
  Impact: Message queue implemented, Redis configured

[4] FEEDBACK - "Great progress on WebSocket layer"
  Time: 15 minutes ago
  Strength: 0.5
  Category: quality (positive)
  Impact: Pattern recorded for reuse

[5] REDIRECT - "Don't use callbacks"
  Time: 10 minutes ago
  Strength: 0.7
  Impact: Planner adjusted to use async/await
```

### 3. Detected Patterns

Shows patterns the colony has extracted from work:

```
📊 DETECTED PATTERNS
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

PERFORMANCE PATTERNS
  → WebSocket pooling reduces connections by 40%
  → Redis pub/sub scales better than direct connections
  → Message queue prevents data loss during reconnects

ARCHITECTURE PATTERNS
  → Layered architecture enables parallel development
  → Event-driven patterns work best for real-time features

SECURITY PATTERNS
  → JWT token validation at entry point prevents bypass
  → Parameterized queries prevent SQL injection

QUALITY PATTERNS
  → Parallel testing reduces total test time by 60%
  → Load testing before deployment prevents production issues
```

### 4. Best Practices

Shows best practices learned from successful executions:

```
✅ BEST PRACTICES
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

SPAWNING
  [bp-1] Spawn specialists when capability gap > 30%
  [bp-2] Max subagent depth 3 prevents complexity explosion
  [bp-3] Terminating subagents after task completion frees resources

COMMUNICATION
  [bp-4] Peer-to-peer coordination reduces bottleneck
  [bp-5] Pheromone signals guide without commands
  [bp-6] Focus on areas, not specific implementations

EXECUTION
  [bp-7] Implement in priority order based on focus pheromones
  [bp-8] Test critical paths before edge cases
  [bp-9] Compress memory between phases
```

### 5. Anti-Patterns

Shows what to avoid (learned from redirects and errors):

```
❌ ANTI-PATTERNS
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

APPROACHES TO AVOID
  [ap-1] String concatenation for SQL (security risk)
  [ap-2] Callback hell (use async/await)
  [ap-3] Monolithic architecture (prevents parallel development)

BEHAVIORS TO AVOID
  [ap-4] Spawning without clear purpose (resource waste)
  [ap-5] Ignoring redirect pheromones (leads to constraints)
  [ap-6] Exceeding subagent depth limit (confusion)
```

### 6. Associative Network

Shows relationships between learned concepts:

```
🔗 ASSOCIATIVE NETWORK
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

STRONG ASSOCIATIONS (>0.8)
  WebSocket security ←→ JWT validation (0.94)
  message reliability ←→ Redis pub/sub (0.89)
  real-time features ←→ event-driven architecture (0.87)

MEDIUM ASSOCIATIONS (0.5-0.8)
  authentication ←→ session management (0.76)
  performance ←→ connection pooling (0.72)
  testing ←→ load testing (0.68)

WEAK ASSOCIATIONS (<0.5)
  markdown parsing ←→ frontend rendering (0.42)
  database indexing ←→ query optimization (0.38)

RECENTLY ACTIVATED
  Last 5 minutes: WebSocket, security, JWT
  Last hour: +authentication, +session, +Redis
```

## Usage Examples

### View All Memory

```bash
/ant:memory
```

Shows all sections.

### Search Memory

```bash
/ant:memory --search "WebSocket"
```

Search for "WebSocket" across all memory.

### View Specific Section

```bash
/ant:memory --section preferences
```

Shows only learned preferences.

```bash
/ant:memory --section patterns
```

Shows only detected patterns.

```bash
/ant:memory --section associations
```

Shows only associative network.

### View Associations

```bash
/ant:memory --associated "JWT"
```

Shows all concepts strongly associated with "JWT".

## Pheromone Learning

The colony learns from pheromone patterns:

### Focus Learning

```
After 3+ focuses on "WebSocket security":
  → Pattern: "Queen prioritizes WebSocket security"
  → Behavior: Executor always includes security in WebSocket work
  → Association: WebSocket ←→ security (strong link)
```

### Redirect Learning

```
After 1 redirect:
  → Logged in ERROR_LEDGER

After 2 redirects:
  → Pattern detected

After 3 redirects:
  → FLAGGED_ISSUE created
  → Constraint created: validate_approach_before_use
  → Blocks approach BEFORE execution
```

### Feedback Learning

```
Positive feedback ("Great work"):
  → Pattern recorded
  → Reused in similar contexts

Negative feedback ("Too many bugs"):
  → Verifier intensifies testing
  → Pattern: "Increase scrutiny when quality feedback"
```

## Memory Compression

Between phases, Synthesizer Ant compresses memory:

```
Phase Complete:
  → Extract key learnings
  → Identify new patterns
  → Compress to summary
  → Store in short-term memory

After 10 phases:
  → Promote to long-term memory
  → Persistent patterns available for all future work
```

## Research Foundation

Based on Phase 5 research:
- **Verification Feedback Loops**: Learning from feedback improves 39%
- **Explainable Verification**: Understanding why patterns work

Based on Phase 4 research:
- **Adaptive Personalization**: Systems learn user preferences
- **Anticipatory Context**: Predicting needs based on patterns

## Related Commands

```
/ant           # System overview
/ant:status    # Colony status with memory stats
/ant:errors    # Error ledger
/ant:focus     # Add focus pheromone (teaches preferences)
```

## Tips for Memory Management

### When Patterns Form

- **3+ focuses** on same topic → Preference learned
- **3+ redirects** on same pattern → Constraint created
- **5+ positive feedback** → Best practice established

### Forgetting Patterns

Patterns can be forgotten if:
- Never used again after 30 days
- Superseded by better approach
- Project-specific and project complete

### Strong Associations

Associations strengthen when:
- Concepts appear together frequently
- One causes the other consistently
- Queen focuses on both together
</reference>
