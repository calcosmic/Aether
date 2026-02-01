---
name: ant:memory
description: View AETHER memory system - working, short-term, and long-term memory contents
---

<objective>
Display the contents and state of AETHER's triple-layer memory system including working memory (current session), short-term memory (compressed sessions), and long-term memory (persistent patterns).
</objective>

<reference>
# `/ant:memory` - View Memory System

## What It Shows

Displays the complete state of AETHER's three-layer memory architecture:

```
/ant:memory
```

## Memory Layers

### 1. Working Memory (Current Session)

Shows active session memory with recent messages, contexts, and facts:

```
💭 WORKING MEMORY (Current Session)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Capacity: 85,400 / 200,000 tokens (42.7%)
Age: 47 minutes
Messages: 47

RECENT MESSAGES (Last 10)
  [47] Goal-Agent → Orchestrator: "Task 4: Create login UI"
  [46] Orchestrator → Frontend-Specialist: "Spawning for UI design gap"
  [45] Frontend-Specialist → Goal-Agent: "UI component library selected: shadcn/ui"
  ...

ACTIVE CONTEXTS
  [project] Blog platform with markdown support
  [tech_stack] Python, FastAPI, React, PostgreSQL
  [goal] "Build authentication system"

FACTS
  [f1] PostgreSQL chosen for primary database
  [f2] JWT token expiration: 24 hours
  [f3] Password requirements: min 12 chars, must include special char

PATTERNS (Detected)
  [p1] Spawn occurs when capability gap detected
  [p2] API endpoints follow RESTful conventions
  [p3] Frontend components follow atomic design pattern
```

### 2. Short-Term Memory (Recent Sessions)

Shows compressed recent sessions:

```
🗂️ SHORT-TERM MEMORY (Recent Sessions)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Capacity: 3 / 10 sessions
Total Items: 127
Compression Ratio: 14.2x

SESSION 1 (47 minutes ago)
  Goal: "Build authentication system"
  Duration: 47 seconds
  Tasks: 5/5 completed
  Key Learnings:
    - PostgreSQL connection pooling reduces latency by 40%
    - JWT refresh token pattern prevents frequent re-login
  Errors Logged: 1 (MEDIUM)

SESSION 2 (2 hours ago)
  Goal: "Create REST API with JWT authentication"
  Duration: 2 minutes 14 seconds
  Tasks: 8/8 completed
  Key Learnings:
    - OpenAPI specification auto-generation reduces docs overhead
    - Rate limiting prevents brute force attacks
  Errors Logged: 2 (1 HIGH, 1 LOW)

SESSION 3 (5 hours ago)
  Goal: "Add real-time notifications"
  Duration: 1 minute 33 seconds
  Tasks: 4/6 completed (2 failed)
  Key Learnings:
    - WebSocket connections need heartbeat mechanism
    - Redis pub/sub scales better than direct connections
  Errors Logged: 3 (1 CRITICAL, 2 MEDIUM)
```

### 3. Long-Term Memory (Persistent)

Shows persistent patterns, best practices, and anti-patterns:

```
📚 LONG-TERM MEMORY (Persistent)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Total Patterns: 34
Best Practices: 18
Anti-Patterns: 7
Created: 23 days ago

PATTERNS (Database)
  [pdb-1] Connection pooling essential for multi-agent concurrency
  [pdb-2] Foreign key indexes improve query performance 3x
  [pdb-3] Transaction isolation level: READ_COMMITTED optimal

PATTERNS (API Design)
  [api-1] RESTful naming: plural nouns, lowercase with underscores
  [api-2] Version APIs via URL path (/v1/, /v2/)
  [api-3] HTTP status codes: 201 for created, 204 for no-content

PATTERNS (Agent Spawning)
  [as-1] Spawn when capability gap > 30%
  [as-2] Max depth 5 prevents spawn loops
  [as-3] Specialist agents terminate after task completion

BEST PRACTICES
  [bp-1] Always validate inputs before database queries
  [bp-2] Use parameterized queries to prevent SQL injection
  [bp-3] Implement circuit breakers for external service calls
  [bp-4] Log all errors with context (stack trace, state, inputs)
  [bp-5] Compress working memory when > 60% capacity

ANTI-PATTERNS (What to Avoid)
  [ap-1] Don't spawn agents for capabilities that exist
  [ap-2] Don't use string interpolation for SQL queries
  [ap-3] Don't hardcode configuration values
  [ap-4] Don't ignore error messages from spawned agents
  [ap-5] Don't let working memory exceed 80% (performance degrades)
```

### 4. Associative Links

Shows relationships between memory items:

```
🔗 ASSOCIATIVE NETWORK
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Total Links: 89

STRONG LINKS (Association > 0.8)
  [L-47] JWT authentication ←→ token refresh pattern (0.94)
  [L-45] PostgreSQL ←→ connection pooling (0.89)
  [L-38] WebSocket ←→ heartbeat mechanism (0.87)
  [L-32] Agent spawning ←→ capability detection (0.85)

MEDIUM LINKS (Association 0.5-0.8)
  [L-23] API design ←→ OpenAPI specification (0.76)
  [L-19] FastAPI ←→ automatic validation (0.72)
  [L-15] React ←→ component library (0.68)
  ... (41 more)

WEAK LINKS (Association < 0.5)
  [L-8] Markdown parsing ←→ frontend rendering (0.42)
  [L-3] Database indexing ←→ query optimization (0.38)
  ... (29 more)

RECENTLY ACTIVATED
  Last 5 minutes: JWT, PostgreSQL, agent spawning, WebSocket
  Last hour: +12 API design, +8 React, +5 database
```

## Usage Examples

### View All Memory Layers

```bash
/ant:memory
```

Shows all four sections (working, short-term, long-term, associative).

### View Specific Layer

```bash/ant:memory --layer working
```

Shows only working memory contents.

```bash/ant:memory --layer short-term
```

Shows only short-term memory (recent sessions).

```bash/ant:memory --layer long-term
```

Shows only long-term memory (persistent patterns).

```bash/ant:memory --layer associative
```

Shows only associative network links.

### Search Memory

```bash/ant:memory --search "authentication"
```

Search all memory layers for "authentication".

```bash/ant:memory --search "JWT" --layer long-term
```

Search only long-term memory for "JWT".

### Query Specific Pattern

```bash/ant:memory --pattern "pdb-1"
```

Show details of specific pattern by ID.

### Export Memory

```bash/ant:memory --export memory_backup.json
```

Export all memory to JSON file.

## Memory Operations

### Compress Working Memory

When working memory exceeds 60%, compress to short-term:

```bash/ant:memory --compress
```

This:
1. Compresses current session to key learnings
2. Stores in short-term memory
3. Clears working memory (keeps active context)
4. Frees up tokens for new work

### Promote to Long-Term

Move important patterns from short-term to long-term:

```bash/ant:memory --promote "connection pooling"
```

Promotes patterns matching "connection pooling" to persistent memory.

### Forget Specific Items

Remove specific memory items:

```bash/ant:memory --forget "pdb-1"
```

Removes pattern from memory (use with caution).

## Memory Capacity Planning

| Memory Type | Capacity | When to Act | Action |
|-------------|----------|-------------|--------|
| Working | 200k tokens | > 60% (120k) | Compress to short-term |
| Working | 200k tokens | > 80% (160k) | Compress immediately |
| Short-Term | 10 sessions | > 8 sessions | Review and promote |
| Short-Term | 10 sessions | = 10 sessions | Auto-compress to long-term |
| Long-Term | Unlimited | N/A | Manual curation only |

## Memory Compression Algorithm

When compressing working memory, AETHER:

1. **Extracts Key Learnings**: Identifies important patterns
2. **Removes Redundancy**: Deduplicates similar information
3. **Preserves Context**: Keeps project state and goals
4. **Creates Summary**: Generates condensed session summary
5. **Stores References**: Maintains links to important details

Result: 10-20x compression while preserving critical information.

## Association Scoring

Links are scored based on:

- **Co-occurrence**: How often items appear together (0-0.4)
- **Semantic similarity**: Related meaning (0-0.3)
- **Causal relationship**: One causes/influences other (0-0.3)

Total score: 0-1.0

**Strong links** (>0.8) are automatically activated when either item is accessed.

## Memory Query Syntax

### Basic Search

```bash
/ant:memory "authentication"
```

Finds all items containing "authentication".

### Boolean Operators

```bash/ant:memory "authentication AND JWT"
/ant:memory "database OR cache"
/ant:memory "authentication NOT OAuth"
```

### Wildcards

```bash/ant:memory "auth*"
```

Matches "authentication", "auth", "authorize", etc.

### Field Search

```bash/ant:memory "pattern:database"
/ant:memory "best_practice:validation"
/ant:memory "anti_pattern:hardcode"
```

### Association Query

```bash/ant:memory --associated "JWT"
```

Shows all items strongly associated with "JWT".

## Memory Health Indicators

### Healthy Memory

- ✅ Working memory 40-60%
- ✅ Short-term 5-7 sessions
- ✅ Strong associations forming
- ✅ Regular compression occurring

### Warning Signs

- ⚠️ Working memory > 70%
- ⚠️ Short-term > 8 sessions
- ⚠️ Few associative links
- ⚠️ No recent compression

### Critical Issues

- 🚨 Working memory > 85%
- 🚨 Short-term = 10 sessions
- 🚨 No associative links (isolated knowledge)

## Related Commands

```
/ant                    # Show system overview
/ant:status            # Show system status with memory stats
/ant:errors            # View error ledger
/ant:build <goal>     # Execute new goal
```

## Tips for Memory Management

### When to Compress

- After completing a major goal
- When working memory exceeds 60%
- Before starting a new complex task
- When system feels "slow" (memory bloat)

### What Gets Promoted to Long-Term

- Repeated patterns (occurs in 3+ sessions)
- Critical best practices (security, performance)
- Important anti-patterns (avoid mistakes)
- High-value associations (strong links)

### When to Forget

- Outdated patterns (superseded by new knowledge)
- Project-specific details (project completed)
- Incorrect information (was never true)
- Duplicate patterns (keep only best version)

## Memory System Architecture

```
┌─────────────────────────────────────────────────┐
│                 AETHER MEMORY                   │
└─────────────────────────────────────────────────┘
                        │
        ┌───────────────┼───────────────┐
        │               │               │
   ┌────▼────┐    ┌────▼────┐    ┌────▼────┐
   │ WORKING │    │ SHORT-  │    │  LONG-  │
   │  200k   │────▶ TERM    │────▶  TERM   │
   │ tokens  │    │ 10 sess │    │permanent │
   └────┬────┘    └────┬────┘    └────┬────┘
        │              │              │
        │         ┌────▼────┐        │
        └────────▶│ASSOCIA- │◀───────┘
                  │ TIVE    │
                  │ NETWORK │
                  └─────────┘
```

Flow: Working → compress → Short-term → promote → Long-term

All layers connected by associative network.
</reference>
