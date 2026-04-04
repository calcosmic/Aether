<!-- Generated from .aether/commands/flags.yaml - DO NOT EDIT DIRECTLY -->
---
name: ant:flags
description: "📋🐜🚩🐜📋 List project flags (blockers, issues, notes)"
---

You are the **Queen**. Display project flags.

## Instructions

Arguments: `$ARGUMENTS`

### Step 1: Parse Arguments

Parse `$ARGUMENTS` for:
- `--all` or `-a`: Show resolved flags too
- `--type` or `-t`: Filter by type (blocker|issue|note)
- `--phase` or `-p`: Filter by phase number
- `--resolve` or `-r`: Resolve a specific flag ID
- `--ack` or `-k`: Acknowledge a specific flag ID

Examples:
- `/ant:flags` → Show active flags
- `/ant:flags --all` → Include resolved flags
- `/ant:flags -t blocker` → Show only blockers
- `/ant:flags --resolve flag_123 "Fixed by commit abc"` → Resolve a flag
- `/ant:flags --ack flag_456` → Acknowledge an issue

### Step 2: Handle Resolution/Acknowledgment


If `--resolve` was provided, run using the Bash tool with description "Resolving colony flag...":


```bash
aether flag-resolve --id "{flag_id}" --message "{resolution_message}"
```
Output:
```
✅ Flag resolved: {flag_id}

   Resolution: {message}
```
Stop here.


If `--ack` was provided, run using the Bash tool with description "Acknowledging colony flag...":


```bash
aether flag-acknowledge --id "{flag_id}"
```
Output:
```
👁️ Flag acknowledged: {flag_id}

   Flag noted. Continuing with work.
```
Stop here.

### Step 3: List Flags


Run using the Bash tool with description "Loading colony flags...":


```bash
aether flag-list {options}
```

Parse result for flags array.

### Step 4: Display

Output header:


```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
📋🐜🚩🐜📋  P R O J E C T   F L A G S
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━



If no flags:
```
       .-.
      (o o)  AETHER COLONY
      | O |  Flags
       `-"
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

✨ No active flags! Colony is clear.

{if --all was used: "No resolved flags either."}
```

If flags exist:
```
       .-.
      (o o)  AETHER COLONY
      | O |  Flags
       `-"
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

{for each flag, grouped by type:}

🚫 BLOCKERS ({count})
   {flag_id}: {title}
   Phase: {phase or "all"} | Created: {date}
   └─ {description preview}

⚠️  ISSUES ({count})
   {flag_id}: {title} {if acknowledged: "[ACK]"}
   Phase: {phase or "all"} | Created: {date}
   └─ {description preview}

📝 NOTES ({count})
   {flag_id}: {title}
   Phase: {phase or "all"} | Created: {date}
   └─ {description preview}

{if --all and resolved flags exist:}

✅ RESOLVED ({count})
   {flag_id}: {title}
   Resolved: {date} | {resolution}

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Summary: {blockers} blockers | {issues} issues | {notes} notes

{if blockers > 0:}
⚠️  Blockers must be resolved before /ant:continue

Commands:
  /ant:flags --resolve {id} "message"   Resolve a flag
  /ant:flags --ack {id}                 Acknowledge an issue
  /ant:flag "description"               Create new flag
```


Generate the state-based Next Up block by running using the Bash tool with description "Generating Next Up suggestions...":
```bash
state=$(jq -r '.state // "IDLE"' .aether/data/COLONY_STATE.json)
current_phase=$(jq -r '.current_phase // 0' .aether/data/COLONY_STATE.json)
total_phases=$(jq -r '.plan.phases | length' .aether/data/COLONY_STATE.json)
aether print-next-up
```


---

## Quick Actions

**Resolve a flag:**
```
/ant:flags --resolve flag_123456 "Fixed in commit abc123"
```

**Acknowledge an issue:**
```
/ant:flags --ack flag_789012
```

**Create a new flag:**
```
/ant:flag --type blocker "Critical issue here"
```
