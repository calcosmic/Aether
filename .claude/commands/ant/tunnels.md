---
name: ant:tunnels
description: "🕳️🐜🕳️ Explore tunnels (browse archived colonies)"
---

You are the **Queen**. Browse the colony history.

## Instructions

### Step 1: Check for Chambers Directory

Check if `.aether/chambers/` exists.

If not:
```
🕳️ ═══════════════════════════════════════════════════
   T U N N E L S   (Colony History)
══════════════════════════════════════════════════ 🕳️

No chambers found.

Archive colonies with /ant:entomb to build the tunnel network.
```
Stop here.

### Step 2: List All Chambers

Run: `bash .aether/aether-utils.sh chamber-list`

Parse JSON result into array of chambers.

If no chambers (empty array):
```
🕳️ ═══════════════════════════════════════════════════
   T U N N E L S   (Colony History)
══════════════════════════════════════════════════ 🕳️

Chambers: 0 colonies archived

The tunnel network is empty.
Archive colonies with /ant:entomb to preserve history.
```
Stop here.

### Step 3: Handle Detail View (if argument provided)

If `$ARGUMENTS` is not empty:
- Treat it as chamber name
- Check if `.aether/chambers/{arguments}/` exists
- If not found:
  ```
  Chamber not found: {arguments}

  Run /ant:tunnels to see available chambers.
  ```
  Stop here.

- If found, read manifest.json and display detailed view:
```
🕳️ ═══════════════════════════════════════════════════
   C H A M B E R   D E T A I L S
══════════════════════════════════════════════════ 🕳️

📦 {chamber_name}

👑 Goal:
   {goal}

🏆 Milestone: {milestone} ({version})
📍 Progress: {phases_completed} of {total_phases} phases
📅 Entombed: {entombed_at}

{If decisions exist:}
🧠 Decisions Preserved:
   {N} architectural decisions recorded
{End if}

{If learnings exist:}
💡 Learnings Preserved:
   {N} validated learnings recorded
{End if}

📁 Files:
   - COLONY_STATE.json (verified: {hash_status})
   - manifest.json

Run /ant:tunnels to return to chamber list.
```

To get the counts and hash status:
- Run `bash .aether/aether-utils.sh chamber-verify .aether/chambers/{chamber_name}`
- If verified: hash_status = "✅"
- If not verified: hash_status = "⚠️ hash mismatch"
- If error: hash_status = "⚠️ error"

Stop here.

### Step 4: Display Chamber List (default view)

```
🕳️ ═══════════════════════════════════════════════════
   T U N N E L S   (Colony History)
══════════════════════════════════════════════════ 🕳️

Chambers: {count} colonies archived

{For each chamber in sorted list:}
📦 {chamber_name}
   👑 {goal (truncated to 50 chars)}
   🏆 {milestone} ({version})
   📍 {phases_completed} phases | 📅 {date}

{End for}

Run /ant:tunnels <chamber_name> to view details
```

**Formatting details:**
- Sort by entombed_at descending (newest first) - already sorted by chamber-list
- Truncate goal to 50 characters with "..." if longer
- Format date as YYYY-MM-DD from ISO timestamp (extract first 10 chars of entombed_at)
- Show chamber count at top

**Edge cases:**
- Malformed manifest: show "⚠️  Invalid manifest" for that chamber and skip it
- Missing COLONY_STATE.json: show "⚠️  Incomplete chamber" for that chamber
- Very long chamber list: display all (no pagination for now)

## Implementation Notes

The `chamber-list` utility returns JSON in this format:
```json
{
  "ok": true,
  "result": [
    {
      "name": "add-user-auth-20260214-153022",
      "goal": "Add user authentication",
      "milestone": "Sealed Chambers",
      "phases_completed": 5,
      "entombed_at": "2026-02-14T15:30:22Z"
    }
  ]
}
```

Parse with jq: `jq -r '.result[] | "\(.name)|\(.goal)|\(.milestone)|\(.phases_completed)|\(.entombed_at)"'`

For detail view, read manifest.json directly:
```bash
jq -r '.goal, .milestone, .version, .phases_completed, .total_phases, .entombed_at, (.decisions | length), (.learnings | length)' .aether/chambers/{name}/manifest.json
```
