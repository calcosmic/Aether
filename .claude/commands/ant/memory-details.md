---
name: memory-details
description: "Show detailed colony memory — wisdom, pending promotions, and recent failures"
symbol: brain
---

# /ant:memory-details — Colony Memory Details

Drill-down view of accumulated colony wisdom, pending promotions, and recent failures.

## Usage

```bash
/ant:memory-details
```

## Implementation

### Step 1: Load Memory Data

Run using the Bash tool with description "Loading colony memory...":
```bash
bash .aether/aether-utils.sh memory-metrics
```

### Step 2: Display Wisdom (from QUEEN.md)

Read .aether/docs/QUEEN.md and display entries by category:
- Philosophies
- Patterns
- Redirects
- Stack
- Decrees

Format:
```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
COLONY WISDOM (X entries)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

📜 Philosophies (N)
   - Entry 1...
   - Entry 2...

🔧 Patterns (N)
   - Entry 1...
```

### Step 3: Display Pending Promotions

Show observations meeting threshold but not yet promoted:
```
⏳ Pending Promotions (N)
   - [type] Content... (X observations)
```

Show deferred proposals:
```
💤 Deferred Proposals (N)
   - [type] Content... (deferred YYYY-MM-DD)
```

### Step 4: Display Recent Failures

Show last 5 failures from midden:
```
⚠️ Recent Failures (N)
   [YYYY-MM-DD HH:MM] Source: context
   Content...
```

### Step 5: Summary

Show counts summary and reminder command:
```
Run /ant:status for quick overview
```
