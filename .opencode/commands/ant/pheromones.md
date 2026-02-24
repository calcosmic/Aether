---
name: ant:pheromones
description: "🎯🐜🚫🐜💬 View and manage active pheromone signals"
---

You are the **Queen**. Display and manage the colony's pheromone signals.

## Instructions

### Step -1: Normalize Arguments

Run: `normalized_args=$(bash .aether/aether-utils.sh normalize-args "$@")`

This ensures arguments work correctly in both Claude Code and OpenCode. Use `$normalized_args` throughout this command.

Parse `$normalized_args`:
- If contains `--no-visual`: set `visual_mode = false` (visual is ON by default)
- Otherwise: set `visual_mode = true`

Extract subcommand from `$normalized_args`:
- No argument or `all`: Show all active pheromones
- `focus`: Show only FOCUS signals
- `redirect`: Show only REDIRECT signals
- `feedback`: Show only FEEDBACK signals
- `clear`: Clear expired/inactive signals
- `expire <id>`: Expire a specific signal by ID

### Step 1: Read Colony State

Read `.aether/data/COLONY_STATE.json`.

If file missing or `goal: null`:
```
No colony initialized. Run /ant:init first.
```
Stop here.

### Step 2: Handle Subcommands

**If subcommand is `clear`:**

Run using the Bash tool:
```bash
# Count signals before
before_count=$(jq '[.signals[] | select(.active == true)] | length' .aether/data/pheromones.json 2>/dev/null || echo "0")

# Mark expired/inactive signals as inactive
now=$(date +%s)
jq --argjson now "$now" '
  def to_epoch(ts):
    if ts == null or ts == "" or ts == "phase_end" then null
    else
      (ts | split("T")) as $parts |
      ($parts[0] | split("-")) as $d |
      ($parts[1] | rtrimstr("Z") | split(":")) as $t |
      (($d[0] | tonumber) - 1970) * 365 * 86400 +
      (($d[1] | tonumber) - 1) * 30 * 86400 +
      (($d[2] | tonumber) - 1) * 86400 +
      ($t[0] | tonumber) * 3600 +
      ($t[1] | tonumber) * 60 +
      ($t[2] | rtrimstr("Z") | tonumber)
    end;

  def decay_days(t):
    if t == "FOCUS"    then 30
    elif t == "REDIRECT" then 60
    else 90
    end;

  .signals = [.signals[] |
    (to_epoch(.created_at)) as $created_epoch |
    (if $created_epoch != null then ($now - $created_epoch) / 86400 else 0 end) as $elapsed_days |
    (decay_days(.type)) as $dd |
    ((.strength // 0.8) * (1 - ($elapsed_days / $dd))) as $eff_raw |
    (if $eff_raw < 0 then 0 else $eff_raw end) as $eff |
    if $eff < 0.1 then . + {active: false} else . end
  ]
' .aether/data/pheromones.json > .aether/data/pheromones.json.tmp && mv .aether/data/pheromones.json.tmp .aether/data/pheromones.json

# Count signals after
after_count=$(jq '[.signals[] | select(.active == true)] | length' .aether/data/pheromones.json 2>/dev/null || echo "0")
cleared=$((before_count - after_count))

echo "before=$before_count after=$after_count cleared=$cleared"
```

Display:
```
🧹 Pheromone Cleanup

   Before: {before_count} active signals
   After:  {after_count} active signals
   Cleared: {cleared} expired signal(s)

Run /ant:pheromones to see remaining signals.
```
Stop here.

**If subcommand is `expire <id>`:**

Extract the signal ID from arguments.
Run using the Bash tool:
```bash
signal_id="{extracted_id}"
jq --arg id "$signal_id" '.signals = [.signals[] | if .id == $id then . + {active: false} else . end]' .aether/data/pheromones.json > .aether/data/pheromones.json.tmp && mv .aether/data/pheromones.json.tmp .aether/data/pheromones.json && echo "expired=$signal_id"
```

Display:
```
✓ Signal expired: {signal_id}

Run /ant:pheromones to see remaining signals.
```
Stop here.

### Step 3: Display Active Pheromones (default or filter)

Run using the Bash tool with description "Displaying pheromones...":
```bash
bash .aether/aether-utils.sh pheromone-display "{subcommand or 'all'}"
```

The output will be the formatted pheromone table.

### Step 4: Summary and Next Steps

Display guidance:
```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

💡 Managing Pheromones

   /ant:focus "area"      🎯 Guide attention
   /ant:redirect "avoid"  🚫 Set hard constraint
   /ant:feedback "note"   💬 Provide guidance
   /ant:pheromones clear  🧹 Clear expired signals

🐜 Signals decay over time: FOCUS 30d, REDIRECT 60d, FEEDBACK 90d
```

### Edge Cases

**No pheromones file:**
```
No pheromones active. Colony has no signals.

Inject signals with:
  /ant:focus "area"    - Guide attention
  /ant:redirect "avoid" - Set hard constraint
  /ant:feedback "note"  - Provide guidance
```

**No active signals of filtered type:**
```
No active {type} signals found.

Try: /ant:pheromones (to see all)
```

**Invalid subcommand:**
Display help showing valid subcommands.
