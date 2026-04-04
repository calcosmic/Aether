<!-- Generated from .aether/commands/flag.yaml - DO NOT EDIT DIRECTLY -->
---
name: ant:flag
description: "🚩🐜📋🐜🚩 Create a project-specific flag (blocker, issue, or note)"
---

You are the **Queen**. Create a project-specific flag.

## Instructions

The flag is: `$ARGUMENTS`

### Step 1: Parse Arguments

Parse `$ARGUMENTS` for:
- `--type` or `-t`: blocker | issue | note (default: issue)
- `--phase` or `-p`: phase number (optional)
- Remaining text: the flag title/description

Examples:
- `/ant:flag "Build fails on auth module"` → issue type
- `/ant:flag --type blocker "API rate limit hit"` → blocker type
- `/ant:flag -t note -p 3 "Consider refactoring later"` → note for phase 3

If `$ARGUMENTS` is empty:
```
Usage: /ant:flag "<description>" [--type blocker|issue|note] [--phase N]

Types:
  blocker  Critical issue that blocks phase advancement
  issue    Warning that should be addressed (default)
  note     Informational for future reference

Examples:
  /ant:flag "Build fails on auth"
  /ant:flag --type blocker "Tests hanging"
  /ant:flag -t note "Consider refactoring"
```
Stop here.

### Step 2: Validate Colony

Read `.aether/data/COLONY_STATE.json`.
If file missing or `goal: null`:
```
No colony initialized. Run /ant:init first.
```
Stop here.

### Step 3: Create Flag

Run using the Bash tool with description "Raising colony flag...":
```bash
aether flag-add --severity "{type}" --title "{title}" --description "{description}" --source "manual" --phase {phase_or_null}
```

Parse the result for the flag ID.

### Step 4: Confirm

Output header based on flag type:

**For blocker:**
```

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
🚩🐜📋🐜🚩  B L O C K E R   F L A G   C R E A T E D
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━


```

**For issue:**
```

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
🚩🐜📋🐜🚩  I S S U E   F L A G   C R E A T E D
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━


```

**For note:**
```

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
🚩🐜📋🐜🚩  N O T E   F L A G   C R E A T E D
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━


```

Then output based on flag type:

**For blocker:**
```
🚫 BLOCKER flag created

   "{title}"

   ID: {flag_id}
   Phase: {phase or "all"}

⚠️  This will BLOCK phase advancement until resolved.
   Use /ant:flags to view all flags.
```

**For issue:**
```
⚠️  ISSUE flag created

   "{title}"

   ID: {flag_id}
   Phase: {phase or "all"}

   Use /ant:flags to view all flags.
```

**For note:**
```
📝 NOTE flag created

   "{title}"

   ID: {flag_id}
   Phase: {phase or "all"}

   Use /ant:flags to view all flags.
```


Generate the state-based Next Up block by running using the Bash tool with description "Generating Next Up suggestions...":
```bash
state=$(jq -r '.state // "IDLE"' .aether/data/COLONY_STATE.json)
current_phase=$(jq -r '.current_phase // 0' .aether/data/COLONY_STATE.json)
total_phases=$(jq -r '.plan.phases | length' .aether/data/COLONY_STATE.json)
aether print-next-up
```


---

## Flag Lifecycle

1. **Created** - Flag is active and visible
2. **Acknowledged** - Flag noted but continuing (issues/notes only)
3. **Resolved** - Flag closed with resolution message
4. **Auto-resolved** - Blockers auto-resolve on build_pass

Blockers CANNOT be acknowledged - they must be resolved before phase advancement.
